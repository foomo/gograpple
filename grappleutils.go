package gograpple

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	v1 "k8s.io/api/apps/v1"
)

func (g Grapple) Cleanup(pod, container string) error {
	if !g.isPatched() {
		return fmt.Errorf("deployment not patched, stopping delve")
	}
	if err := g.validatePod(&pod); err != nil {
		return err
	}
	if err := g.validateContainer(&container); err != nil {
		return err
	}
	return g.dlvCleanup(g.l, pod, container)
}

func (g Grapple) getPIDsOf(pod, container, name string) (pids []string, err error) {
	rawPids, errExec := g.kubeCmd.ExecPod(pod, container, []string{"pidof", name}).Run()
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

func (g *Grapple) updateDeployment() error {
	d, err := g.kubeCmd.GetDeployment(g.deployment.Name)
	if err != nil {
		return err
	}
	g.deployment = *d
	return nil
}

func (g Grapple) getArgsFromConfigMap(configMap, container string) ([]string, error) {
	out, err := g.kubeCmd.GetConfigMapKey(configMap, defaultConfigMapDeploymentKey)
	if err != nil {
		return nil, err
	}
	var d v1.Deployment
	if err := json.Unmarshal([]byte(out), &d); err != nil {
		return nil, err
	}
	for _, c := range d.Spec.Template.Spec.Containers {
		if c.Name == container {
			return c.Args, nil
		}
	}
	return nil, fmt.Errorf("no args found for container %q", container)
}
