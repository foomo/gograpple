package kubectl

import (
	"fmt"
	"os"
	"strings"

	"github.com/bitfield/script"
	"github.com/foomo/gograpple/suggest"
	"github.com/life4/genesis/slices"
	"github.com/pkg/errors"
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
