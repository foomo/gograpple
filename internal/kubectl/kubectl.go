package kubectl

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/bitfield/script"
	"github.com/foomo/gograpple/internal/log"
	"github.com/foomo/gograpple/internal/suggest"
	"github.com/life4/genesis/slices"
	"github.com/pkg/errors"
	apps "k8s.io/api/apps/v1"
)

func Exists() bool {
	if _, err := os.Stat(os.Getenv("KUBECONFIG")); err != nil {
		return false
	}
	return true
}

func SetContext(name string) error {
	// kubectl config use-context name
	out, err := script.Exec(fmt.Sprintf("kubectl config use-context %v", name)).String()
	if err != nil {
		return errors.WithMessage(err, out)
	}
	return nil
}

func GetCurrentContext() (string, error) {
	// kubectl config current-context
	return script.Exec("kubectl configcurrent-context").String()
}

func ContextExists(name string) bool {
	contexts, err := ListContexts()
	if err != nil {
		return false
	}
	if !slices.Contains(contexts, name) {
		return false
	}
	return true
}

func ListContexts() ([]string, error) {
	results, err := script.Exec("kubectl config get-contexts -o name").Slice()
	if err != nil {
		return nil, fmt.Errorf(results[0])
	}
	return results, err
}

func ListNamespaces() ([]string, error) {
	results, err := script.Exec("kubectl get namespaces -o name").FilterLine(func(s string) string {
		return strings.TrimPrefix(s, "namespace/")
	}).Slice()
	if err != nil {
		return nil, fmt.Errorf(results[0])
	}
	return results, err
}

func ListDeployments(namespace string) ([]string, error) {
	results, err := script.Exec(fmt.Sprintf("kubectl get deployment -n %v -o name", namespace)).FilterLine(func(s string) string {
		return strings.TrimPrefix(s, "deployment.apps/")
	}).Slice()
	if err != nil {
		return nil, fmt.Errorf(results[0])
	}
	return results, err
}

func ListPods(namespace, deployment string) ([]string, error) {
	results, err := script.Exec(
		fmt.Sprintf("kubectl get pods -n %v -o name", namespace)).
		Match(deployment).
		FilterLine(func(s string) string {
			return strings.TrimPrefix(s, "pod/")
		}).Slice()
	if err != nil {
		return nil, fmt.Errorf(results[0])
	}
	return results, err
}

func ListContainers(namespace, deployment string) ([]string, error) {
	// kubectl get deployment %v -n %v -o jsonpath={.spec.template.spec.containers[*].name}
	results, err := script.Exec(
		fmt.Sprintf("kubectl -n %v get deployment %v -o jsonpath={.spec.template.spec.containers[*].name}", namespace, deployment)).
		Replace(" ", "\n").
		FilterLine(func(s string) string {
			return strings.TrimPrefix(s, "pod/")
		}).Slice()
	if err != nil {
		return nil, fmt.Errorf(results[0])
	}
	return results, err
}

func ListRepositories(namespace, deployment string) ([]string, error) {
	results, err := FilterImages(namespace, deployment, func(s string) string {
		repo, _, _, _ := suggest.ParseImage(s)
		return repo
	})
	return results, err
}

func ListImages(namespace, deployment string) ([]string, error) {
	results, err := FilterImages(namespace, deployment, func(s string) string {
		return s
	})
	return results, err
}

func FilterImages(namespace, deployment string, filter func(s string) string) ([]string, error) {
	results, err := script.Exec(
		fmt.Sprintf("kubectl -n %v get deployment %v -o jsonpath={.spec.template.spec.containers[*].image}", namespace, deployment)).
		Replace(" ", "\n").
		FilterLine(filter).Slice()
	if err != nil {
		return nil, fmt.Errorf(results[0])
	}
	return results, err
}

func ExecPod(namespace, pod, container string, cmd []string) *script.Pipe {
	return script.Exec(fmt.Sprintf(
		"kubectl -n %v exec %v -c %v -- %v", namespace, pod, container, strings.Join(cmd, " ")))
}

func GetDeployment(namespace, deployment string) (*apps.Deployment, error) {
	out, err := script.Exec(fmt.Sprintf(
		"kubectl -n %v get deployment %v -o json", namespace, deployment)).String()
	if err != nil {
		return nil, err
	}
	var d apps.Deployment
	if err := json.Unmarshal([]byte(out), &d); err != nil {
		return nil, err
	}
	return &d, nil
}

func GetMostRecentRunningPodBySelectors(namespace string, selectors map[string]string) (string, error) {
	var selector []string
	for k, v := range selectors {
		selector = append(selector, fmt.Sprintf("%v=%v", k, v))
	}
	cmd := fmt.Sprintf(
		"kubectl -n %v --selector %v get pods --field-selector=status.phase=Running --sort-by=.status.startTime -o name",
		namespace, strings.Join(selector, ","))
	pods, err := script.Exec(cmd).FilterLine(func(s string) string {
		return strings.TrimLeft(s, "pod/")
	}).Slice()
	if err != nil {
		return "", err
	}
	if len(pods) > 0 {
		return pods[len(pods)-1], nil
	}
	return "", fmt.Errorf("no pods found")
}

func TempSwitchContext(context string, cb func() error) error {
	currentCtx, err := GetCurrentContext()
	if err != nil {
		return err
	}
	defer SetContext(currentCtx)
	if err := SetContext(context); err != nil {
		return err
	}
	return cb()
}

func PortForwardPod(namespace, pod string, port int) error {
	cmd := fmt.Sprintf("kubectl -n %v port-forward pods/%v %v:%v", namespace, pod, port, port)
	_, err := script.Exec(cmd).WithStdout(log.Writer("kubectl")).Stdout()
	// if err != nil {
	// 	return errors.WithMessage(err, out)
	// }
	return err
}

func GetPIDsOf(namespace, pod, container, process string) (pids []string, err error) {
	return ExecPod(namespace, pod, container, []string{"pidof", process}).Replace(" ", "\n").Slice()
}

func KillPidsOnPod(namespace, pod, container string, pids []string, murder bool) []error {
	var errs []error
	for _, pid := range pids {
		cmd := []string{"kill"}
		if murder {
			cmd = append(cmd, "-s", "9")
		}
		cmd = append(cmd, pid)
		_, err := ExecPod(namespace, pod, container, cmd).Stdout()
		if err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}

func CopyToPod(namespace, pod, container, source, destination string) error {
	out, err := script.Exec(fmt.Sprintf("kubectl -n %v cp %v %v:%v -c %v", namespace, source, pod, destination, container)).String()
	if err != nil {
		return errors.WithMessage(err, out)
	}
	return nil
}
