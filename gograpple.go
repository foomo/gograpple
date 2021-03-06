package gograpple

import (
	"github.com/foomo/gograpple/exec"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/apps/v1"
)

const (
	devDeploymentPatchFile        = "deployment-patch.yaml"
	defaultWaitTimeout            = "30s"
	conditionContainersReady      = "condition=ContainersReady"
	defaultPatchedLabel           = "dev-mode-patched"
	defaultPatchImage             = "gograpple-patch:latest"
	defaultConfigMapMount         = "/etc/config/mounted"
	defaultConfigMapDeploymentKey = "deployment.json"
)

type Grapple struct {
	l          *logrus.Entry
	deployment v1.Deployment
	kubeCmd    *exec.KubectlCmd
	dockerCmd  *exec.DockerCmd
	goCmd      *exec.GoCmd
}

func NewGrapple(l *logrus.Entry, namespace, deployment string) (*Grapple, error) {
	g := &Grapple{l: l}
	g.kubeCmd = exec.NewKubectlCommand(l)
	g.dockerCmd = exec.NewDockerCommand(l)
	g.goCmd = exec.NewGoCommand(l)
	g.kubeCmd.Args("-n", namespace)

	if err := g.validateNamespace(namespace); err != nil {
		return nil, err
	}
	if err := g.validateDeployment(namespace, deployment); err != nil {
		return nil, err
	}

	d, err := g.kubeCmd.GetDeployment(deployment)
	if err != nil {
		return nil, err
	}
	g.deployment = *d

	return g, nil
}
