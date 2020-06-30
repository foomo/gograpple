package gograpple

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/foomo/squadron"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/apps/v1"
)

func RollbackDeployment(l *logrus.Entry, deployment, namespace string) *squadron.Cmd {
	cmd := []string{
		"kubectl", "-n", namespace,
		"rollout", "undo", fmt.Sprintf("deployment/%v", deployment),
	}
	return squadron.Command(l, cmd...)
}

func WaitForRollout(l *logrus.Entry, deployment, namespace, timeout string) *squadron.Cmd {
	cmd := []string{
		"kubectl", "-n", namespace,
		"rollout", "status", fmt.Sprintf("deployment/%v", deployment),
		"-w", "--timeout", timeout,
	}
	return squadron.Command(l, cmd...)
}

func GetMostRecentPodBySelectors(l *logrus.Entry, selectors map[string]string, namespace string) (string, error) {
	var selector []string
	for k, v := range selectors {
		selector = append(selector, fmt.Sprintf("%v=%v", k, v))
	}
	cmd := []string{
		"kubectl", "-n", namespace,
		"--selector", strings.Join(selector, ","),
		"get", "pods", "--sort-by=.status.startTime",
		"-o", "name",
	}
	out, err := squadron.Command(l, cmd...).Run()
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

func WaitForPodState(l *logrus.Entry, namepsace, pod, condition, timeout string) *squadron.Cmd {
	cmd := []string{
		"kubectl", "-n", namepsace,
		"wait", fmt.Sprintf("pod/%v", pod),
		fmt.Sprintf("--for=%v", condition),
		fmt.Sprintf("--timeout=%v", timeout),
	}
	return squadron.Command(l, cmd...)
}

func ExecShell(l *logrus.Entry, resource, path, namespace string) *squadron.Cmd {
	cmdArgs := []string{
		"kubectl", "-n", namespace,
		"exec", "-it", resource,
		"--", "/bin/sh", "-c",
		fmt.Sprintf("cd %v && /bin/sh", path),
	}

	return squadron.Command(l, cmdArgs...).Stdin(os.Stdin).Stdout(os.Stdout).Stderr(os.Stdout)
}

func PatchDeployment(l *logrus.Entry, patch, deployment, namespace string) *squadron.Cmd {
	cmd := []string{
		"kubectl", "-n", namespace,
		"patch", "deployment", deployment,
		"--patch", patch,
	}
	return squadron.Command(l, cmd...)
}

func CopyToPod(l *logrus.Entry, pod, container, namespace, source, destination string) *squadron.Cmd {
	cmd := []string{
		"kubectl", "-n", namespace,
		"cp", source, fmt.Sprintf("%v:%v", pod, destination),
		"-c", container,
	}
	return squadron.Command(l, cmd...)
}

func ExecPod(l *logrus.Entry, pod, container, namespace string, cmd []string) *squadron.Cmd {
	c := []string{
		"kubectl", "-n", namespace,
		"exec", pod,
		"-c", container,
		"--",
	}
	c = append(c, cmd...)
	return squadron.Command(l, c...)
}

func ExposePod(l *logrus.Entry, namespace, pod string, host string, port int) *squadron.Cmd {
	if host == "127.0.0.1" {
		host = ""
	}
	cmd := []string{
		"kubectl", "-n", namespace,
		"expose", "pod", pod,
		"--type=LoadBalancer",
		fmt.Sprintf("--port=%v", port),
		fmt.Sprintf("--external-ip=%v", host),
		// fmt.Sprintf("--name=%v-%v", pod, port),
	}
	return squadron.Command(l, cmd...)
}

func DeleteService(l *logrus.Entry, deployment v1.Deployment, service string) *squadron.Cmd {
	cmd := []string{
		"kubectl", "-n", deployment.Namespace,
		"delete", "service", service,
	}
	return squadron.Command(l, cmd...)
}

func GetDeployment(l *logrus.Entry, namespace, deployment string) (v1.Deployment, error) {
	cmd := []string{
		"kubectl", "-n", namespace,
		"get", "deployment", deployment,
		"-o", "json",
	}
	out, err := squadron.Command(l, cmd...).Run()
	if err != nil {
		return v1.Deployment{}, err
	}
	var d v1.Deployment
	if err := json.Unmarshal([]byte(out), &d); err != nil {
		return v1.Deployment{}, err
	}
	return d, nil
}

func GetNamespaces(l *logrus.Entry) ([]string, error) {
	cmd := []string{
		"kubectl",
		"get", "namespace",
		"-o", "name",
	}
	out, err := squadron.Command(l, cmd...).Run()
	if err != nil {
		return nil, err
	}

	return parseResources(out, "\n", "namespace/")
}

func GetDeployments(l *logrus.Entry, namespace string) ([]string, error) {
	cmd := []string{
		"kubectl", "-n", namespace,
		"get", "deployment",
		"-o", "name",
	}
	out, err := squadron.Command(l, cmd...).Run()
	if err != nil {
		return nil, err
	}

	return parseResources(out, "\n", "deployment.apps/")
}

func GetPods(l *logrus.Entry, namespace string, selectors map[string]string) ([]string, error) {
	var selector []string
	for k, v := range selectors {
		selector = append(selector, fmt.Sprintf("%v=%v", k, v))
	}
	cmd := []string{
		"kubectl", "-n", namespace,
		"--selector", strings.Join(selector, ","),
		"get", "pods", "--sort-by=.status.startTime",
		"-o", "name",
	}
	out, err := squadron.Command(l, cmd...).Run()
	if err != nil {
		return nil, err
	}

	return parseResources(out, "\n", "pod/")
}

func GetContainers(l *logrus.Entry, deployment v1.Deployment) []string {
	var containers []string
	for _, c := range deployment.Spec.Template.Spec.Containers {
		containers = append(containers, c.Name)
	}
	return containers
}

func GetPodsByLabels(l *logrus.Entry, labels []string) ([]string, error) {
	var selector []string
	for k, v := range labels {
		selector = append(selector, fmt.Sprintf("%v=%v", k, v))
	}
	cmd := []string{
		"kubectl", "get", "pods",
		"-l", strings.Join(labels, ","),
		"-o", "name", "-A",
	}
	out, err := squadron.Command(l, cmd...).Run()
	if err != nil {
		return nil, err
	}

	return parseResources(out, "\n", "pod/")
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

func RestartDeployment(l *logrus.Entry, deployment, namespace string) *squadron.Cmd {
	cmd := []string{
		"kubectl", "-n", namespace,
		"rollout", "restart", fmt.Sprintf("deployment/%v", deployment),
	}
	return squadron.Command(l, cmd...)
}

func CreateConfigMapFromFile(l *logrus.Entry, name, namespace, path string) (string, error) {
	cmd := []string{
		"kubectl", "-n", namespace,
		"create", "configmap", name,
		"--from-file", path,
	}
	return squadron.Command(l, cmd...).Run()
}

func CreateConfigMap(l *logrus.Entry, name, namespace string, keyMap map[string]string) (string, error) {
	cmd := []string{
		"kubectl", "-n", namespace,
		"create", "configmap", name,
	}
	for key, value := range keyMap {
		cmd = append(cmd, fmt.Sprintf("--from-literal=%v=%v", key, value))
	}
	return squadron.Command(l, cmd...).Run()
}

func DeleteConfigMap(l *logrus.Entry, name, namespace string) (string, error) {
	cmd := []string{
		"kubectl", "-n", namespace,
		"delete", "configmap", name,
	}
	return squadron.Command(l, cmd...).Run()
}

func GetConfigMapKey(l *logrus.Entry, name, namespace, key string) (string, error) {
	key = strings.ReplaceAll(key, ".", "\\.")
	// jsonpath map key is not very fond of dots
	cmd := []string{
		"kubectl", "-n", namespace,
		"get", "configmap", name,
		"-o", fmt.Sprintf("jsonpath={.data.%v}", key),
	}
	out, err := squadron.Command(l, cmd...).Run()
	if err != nil {
		return out, err
	}
	if out == "" {
		return out, fmt.Errorf("no key %q found in ConfigMap %q", key, name)
	}
	return out, nil
}
