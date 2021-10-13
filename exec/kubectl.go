package exec

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
)

type KubectlCmd struct {
	Cmd
}

func NewKubectlCommand(l *logrus.Entry) *KubectlCmd {
	return &KubectlCmd{*NewCommand(l, "kubectl")}
}

func (c KubectlCmd) RollbackDeployment(deployment string) *Cmd {
	return c.Args("rollout", "undo", fmt.Sprintf("deployment/%v", deployment))
}

func (c KubectlCmd) WaitForRollout(deployment, timeout string) *Cmd {
	return c.Args("rollout", "status", fmt.Sprintf("deployment/%v", deployment),
		"-w", "--timeout", timeout)
}

func (c KubectlCmd) GetMostRecentPodBySelectors(selectors map[string]string) (string, error) {
	var selector []string
	for k, v := range selectors {
		selector = append(selector, fmt.Sprintf("%v=%v", k, v))
	}
	out, err := c.Args("--selector", strings.Join(selector, ","),
		"get", "pods", "--sort-by=.status.startTime", "-o", "name").Run()
	if err != nil {
		return "", err
	}

	pods, err := parseResources(out, "\n", "pod/")
	if err != nil {
		return "", err
	}
	if len(pods) > 0 {
		return pods[len(pods)-1], nil
	}
	return "", fmt.Errorf("no pods found")
}

func (c KubectlCmd) WaitForPodState(pod, condition, timeout string) *Cmd {
	return c.Args("wait", fmt.Sprintf("pod/%v", pod),
		fmt.Sprintf("--for=%v", condition),
		fmt.Sprintf("--timeout=%v", timeout))
}

func (c KubectlCmd) ExecShell(resource, path string) *Cmd {
	return c.Args("exec", "-it", resource,
		"--", "/bin/sh", "-c",
		fmt.Sprintf("cd %v && /bin/sh", path),
	).Stdin(os.Stdin).Stdout(os.Stdout).Stderr(os.Stdout)
}

func (c KubectlCmd) PatchDeployment(patch, deployment string) *Cmd {
	return c.Args("patch", "deployment", deployment, "--patch", patch)
}

func (c KubectlCmd) CopyToPod(pod, container, source, destination string) *Cmd {
	return c.Args("cp", source, fmt.Sprintf("%v:%v", pod, destination), "-c", container)
}

func (c KubectlCmd) ExecPod(pod, container string, cmd []string) *Cmd {
	return c.Args("exec", pod, "-c", container, "--").Args(cmd...)
}

func (c KubectlCmd) ExposePod(pod string, host string, port int) *Cmd {
	if host == "127.0.0.1" {
		host = ""
	}
	return c.Args("expose", "pod", pod, "--type=LoadBalancer",
		fmt.Sprintf("--port=%v", port), fmt.Sprintf("--external-ip=%v", host))
}

func (c KubectlCmd) PortForwardPod(pod string, host string, port int) *Cmd {
	return c.Args("port-forward", "pods/"+pod, strconv.Itoa(port)+":"+strconv.Itoa(port))
}

func (c KubectlCmd) DeleteService(service string) *Cmd {
	return c.Args("delete", "service", service)
}

func (c KubectlCmd) GetDeployment(deployment string) (*apps.Deployment, error) {
	out, err := c.Args("get", "deployment", deployment, "-o", "json").Run()
	if err != nil {
		return nil, err
	}
	var d apps.Deployment
	if err := json.Unmarshal([]byte(out), &d); err != nil {
		return nil, err
	}
	return &d, nil
}

func (c KubectlCmd) GetNamespaces() ([]string, error) {
	out, err := c.Args("get", "namespace", "-o", "name").Run()
	if err != nil {
		return nil, err
	}
	return parseResources(out, "\n", "namespace/")
}

func (c KubectlCmd) GetDeployments() ([]string, error) {
	out, err := c.Args("get", "deployment", "-o", "name").Run()
	if err != nil {
		return nil, err
	}
	return parseResources(out, "\n", "deployment.apps/")
}

func (c KubectlCmd) GetPods(selectors map[string]string) ([]string, error) {
	var selector []string
	for k, v := range selectors {
		selector = append(selector, fmt.Sprintf("%v=%v", k, v))
	}
	out, err := c.Args("--selector", strings.Join(selector, ","),
		"get", "pods", "--sort-by=.status.startTime",
		"-o", "name").Run()
	if err != nil {
		return nil, err
	}
	return parseResources(out, "\n", "pod/")
}

func (c KubectlCmd) GetContainers(deployment apps.Deployment) []string {
	var containers []string
	for _, c := range deployment.Spec.Template.Spec.Containers {
		containers = append(containers, c.Name)
	}
	return containers
}

func (c KubectlCmd) GetPodsByLabels(labels []string) ([]string, error) {
	out, err := c.Args("get", "pods", "-l", strings.Join(labels, ","), "-o", "name", "-A").Run()
	if err != nil {
		return nil, err
	}
	return parseResources(out, "\n", "pod/")
}

func (c KubectlCmd) RestartDeployment(deployment string) *Cmd {
	return c.Args("rollout", "restart", fmt.Sprintf("deployment/%v", deployment))
}

func (c KubectlCmd) CreateConfigMapFromFile(name, path string) (string, error) {
	return c.Args("create", "configmap", name, "--from-file", path).Run()
}

func (c KubectlCmd) CreateConfigMap(name string, keyMap map[string]string) (string, error) {
	c.Args("create", "configmap", name)
	for key, value := range keyMap {
		c.Args(fmt.Sprintf("--from-literal=%v=%v", key, value))
	}
	return c.Run()
}

func (c KubectlCmd) DeleteConfigMap(name string) (string, error) {
	return c.Args("delete", "configmap", name).Run()
}

func (c KubectlCmd) GetConfigMapKey(name, key string) (string, error) {
	key = strings.ReplaceAll(key, ".", "\\.")
	// jsonpath map key is not very fond of dots
	out, err := c.Args("get", "configmap", name, "-o",
		fmt.Sprintf("jsonpath={.data.%v}", key)).Run()
	if err != nil {
		return out, err
	}
	if out == "" {
		return out, fmt.Errorf("no key %q found in ConfigMap %q", key, name)
	}
	return out, nil
}

func parseResources(out, delimiter, prefix string) ([]string, error) {
	var res []string
	if out == "" {
		return res, nil
	}
	lines := strings.Split(out, delimiter)
	if len(lines) == 1 && lines[0] == "" {
		return nil, fmt.Errorf("delimiter %q not found in %q", delimiter, out)
	}
	for _, line := range lines {
		if line == "" {
			continue
		}
		unprefixed := strings.TrimPrefix(line, prefix)
		if unprefixed == line {
			return nil, fmt.Errorf("prefix %q not found in %q", prefix, line)
		}
		res = append(res, strings.TrimPrefix(line, prefix))
	}
	return res, nil
}

func (c KubectlCmd) KillPidsOnPod(pod, container string, pids []string, murder bool) []error {
	var errs []error
	for _, pid := range pids {
		cmd := []string{"kill"}
		if murder {
			cmd = append(cmd, "-s", "9")
		}
		cmd = append(cmd, pid)
		_, errKill := c.ExecPod(pod, container, cmd).Run()
		if errKill != nil {
			errs = append(errs, errKill)
		}
	}
	return errs
}

func (c KubectlCmd) GetDeploymentFromConfigMap(deployment, configMapKey string) (*apps.Deployment, error) {
	out, err := c.GetConfigMapKey(deployment, configMapKey)
	if err != nil {
		return nil, err
	}
	var d apps.Deployment
	if err := json.Unmarshal([]byte(out), &d); err != nil {
		return nil, err
	}
	return &d, nil

}

func (_ KubectlCmd) GetContainerFromDeployment(container string, d *apps.Deployment) (*core.Container, error) {
	for _, c := range d.Spec.Template.Spec.Containers {
		if c.Name == container {
			return &c, nil
		}
	}
	return nil, fmt.Errorf("no container %q found in deployment %q", container, d.Name)
}

func (c KubectlCmd) GetPIDsOf(pod, container, process string) (pids []string, err error) {
	rawPids, errExec := c.ExecPod(pod, container, []string{"pidof", process}).Run()
	if errExec != nil {
		if errExec.Error() == "exit status 1" {
			return []string{}, nil
		}
		return nil, errors.New("could not get pid of process: " + errExec.Error())
	}
	stripped := []string{}
	for _, rawPid := range strings.Split(rawPids, " ") {
		stripped = append(stripped, strings.Trim(rawPid, "\n"))
	}
	return stripped, nil
}

func (c KubectlCmd) ValidateNamespace(namespace string) error {
	available, err := c.GetNamespaces()
	if err != nil {
		return err
	}
	return validateResource("namespace", namespace, "", available)
}

func (c KubectlCmd) ValidateDeployment(namespace, deployment string) error {
	available, err := c.GetDeployments()
	if err != nil {
		return err
	}
	return validateResource("deployment", deployment, fmt.Sprintf("for namespace %q", namespace), available)
}

func (c KubectlCmd) ValidatePod(d apps.Deployment, pod *string) error {
	if *pod == "" {
		var err error
		*pod, err = c.GetMostRecentPodBySelectors(d.Spec.Selector.MatchLabels)
		if err != nil || *pod == "" {
			return err
		}
		return nil
	}
	available, err := c.GetPods(d.Spec.Selector.MatchLabels)
	if err != nil {
		return err
	}
	return validateResource("pod", *pod, fmt.Sprintf("for deployment %q", d.Name), available)
}

func (c KubectlCmd) ValidateContainer(d apps.Deployment, container *string) error {
	if *container == "" {
		*container = d.Name
	}
	return validateResource("container", *container, fmt.Sprintf("for deployment %q", d.Name), c.GetContainers(d))
}

func validateResource(resourceType, resource, suffix string, available []string) error {
	if !stringIsInSlice(resource, available) {
		return fmt.Errorf("%v %q not found %v, available: %v", resourceType, resource, suffix, strings.Join(available, ", "))
	}
	return nil
}

func stringIsInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}
