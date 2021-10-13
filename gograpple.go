package gograpple

import (
	"github.com/foomo/gograpple/exec"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/apps/v1"
)

const (
	devDeploymentPatchFile           = "deployment-patch.yaml"
	defaultWaitTimeout               = "30s"
	conditionContainersReady         = "condition=ContainersReady"
	defaultPatchedLabel              = "dev-mode-patched"
	defaultPatchImageSuffix          = "-patch"
	defaultConfigMapMount            = "/etc/config/mounted"
	defaultConfigMapDeploymentKey    = "deployment.json"
	defaultConfigMapDeploymentSuffix = "-patch"
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

	if err := g.kubeCmd.ValidateNamespace(namespace); err != nil {
		return nil, err
	}
	if err := g.kubeCmd.ValidateDeployment(namespace, deployment); err != nil {
		return nil, err
	}

	d, err := g.kubeCmd.GetDeployment(deployment)
	if err != nil {
		return nil, err
	}
	g.deployment = *d

	return g, nil
}
