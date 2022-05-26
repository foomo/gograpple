package gograpple

import (
	"context"

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
	defaultTag                       = "latest"
	defaultImage                     = "alpine:latest"
	patchImageName                   = "patch-image"
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
	g.kubeCmd = exec.NewKubectlCommand()
	g.dockerCmd = exec.NewDockerCommand()
	g.goCmd = exec.NewGoCommand()
	g.kubeCmd.Logger(l)
	g.dockerCmd.Logger(l)
	g.goCmd.Logger(l)
	g.kubeCmd.Args("-n", namespace)

	validateCtx := context.Background()
	if err := g.kubeCmd.ValidateNamespace(validateCtx, namespace); err != nil {
		return nil, err
	}
	if err := g.kubeCmd.ValidateDeployment(validateCtx, namespace, deployment); err != nil {
		return nil, err
	}

	d, err := g.kubeCmd.GetDeployment(validateCtx, deployment)
	if err != nil {
		return nil, err
	}
	g.deployment = *d

	return g, nil
}
