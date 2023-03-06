package gograpple

import (
	"context"
	"fmt"

	"github.com/foomo/gograpple/log"
)

func (g Grapple) Shell(pod string) error {
	ctx := context.Background()
	if !g.isPatched() {
		return fmt.Errorf("deployment not patched, stopping shell")
	}
	if err := g.kubeCmd.ValidatePod(context.Background(), g.deployment, &pod); err != nil {
		return err
	}
	log.Logger().Infof("waiting for pod %v with %q", pod, conditionContainersReady)
	_, err := g.kubeCmd.WaitForPodState(pod, conditionContainersReady, defaultWaitTimeout).Run(ctx)
	if err != nil {
		return err
	}

	log.Logger().Infof("running interactive shell for patched deployment %v", g.deployment.Name)
	_, err = g.kubeCmd.ExecShell(fmt.Sprintf("pod/%v", pod), "/").Run(ctx)
	return err
}
