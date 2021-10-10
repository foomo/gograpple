package gograpple

import (
	"errors"
	"fmt"
	"strings"
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
	return g.cleanupDelve(pod, container)
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
