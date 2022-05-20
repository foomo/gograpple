package suggest

import (
	"fmt"
	"os"
	"strings"

	"github.com/bitfield/script"
	"github.com/life4/genesis/slices"
)

const (
	DefaultKubeConfig = "$HOME/.kube/config"
)

type KubeConfig string

func (c KubeConfig) Path() string {
	return string(c)
}

func (c KubeConfig) Exists() bool {
	if _, err := os.Stat(c.Path()); err != nil {
		return false
	}
	return true
}

func (c KubeConfig) SetContext(name string) error {
	// kubectl config use-context name
	_, err := script.Exec(fmt.Sprintf("kubectl config use-context %v", name)).String()
	return err
}

func (c KubeConfig) GetCurrentContext() (string, error) {
	// kubectl config current-context
	return script.Exec("kubectl configcurrent-context").String()
}

func (c KubeConfig) ContextExists(name string) bool {
	contexts, err := c.ListContexts()
	if err != nil {
		return false
	}
	if !slices.Contains(contexts, name) {
		return false
	}
	return true
}

func (c KubeConfig) ListContexts() ([]string, error) {
	results, err := script.Exec("kubectl config get-contexts -o name").Slice()
	if err != nil {
		return nil, fmt.Errorf(results[0])
	}
	return results, err
}

func (c KubeConfig) ListNamespaces() ([]string, error) {
	results, err := script.Exec("kubectl get namespaces -o name").FilterLine(func(s string) string {
		return strings.TrimPrefix(s, "namespace/")
	}).Slice()
	if err != nil {
		return nil, fmt.Errorf(results[0])
	}
	return results, err
}

func (c KubeConfig) ListDeployments(namespace string) ([]string, error) {
	results, err := script.Exec(fmt.Sprintf("kubectl get deployment -n %v -o name", namespace)).FilterLine(func(s string) string {
		return strings.TrimPrefix(s, "deployment.apps/")
	}).Slice()
	if err != nil {
		return nil, fmt.Errorf(results[0])
	}
	return results, err
}

func (c KubeConfig) ListPods(namespace, deployment string) ([]string, error) {
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

func (c KubeConfig) ListContainers(namespace, deployment string) ([]string, error) {
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

func (c KubeConfig) ListRepositories(namespace, deployment string) ([]string, error) {
	results, err := c.FilterImages(namespace, deployment, func(s string) string {
		repo, _, _, _ := ParseImage(s)
		return repo
	})
	return results, err
}

func (c KubeConfig) ListImages(namespace, deployment string) ([]string, error) {
	results, err := c.FilterImages(namespace, deployment, func(s string) string {
		return s
	})
	return results, err
}

func (c KubeConfig) FilterImages(namespace, deployment string, filter func(s string) string) ([]string, error) {
	results, err := script.Exec(
		fmt.Sprintf("kubectl -n %v get deployment %v -o jsonpath={.spec.template.spec.containers[*].image}", namespace, deployment)).
		Replace(" ", "\n").
		FilterLine(filter).Slice()
	if err != nil {
		return nil, fmt.Errorf(results[0])
	}
	return results, err
}

func (c KubeConfig) TempSwitchContext(context string, cb func() error) error {
	currentCtx, err := c.GetCurrentContext()
	if err != nil {
		return err
	}
	defer c.SetContext(currentCtx)
	if err := c.SetContext(context); err != nil {
		return err
	}
	return cb()
}
