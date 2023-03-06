package gograpple

import (
	"context"

	"github.com/foomo/gograpple/exec"
	"github.com/foomo/gograpple/log"
	v1 "k8s.io/api/apps/v1"
)

const (
	devDeploymentPatchFile           = "deployment-patch.yaml"
	defaultWaitTimeout               = "30s"
	conditionContainersReady         = "condition=ContainersReady"
	defaultPatchImageSuffix          = "-patch"
	defaultConfigMapMount            = "/etc/config/mounted"
	defaultConfigMapDeploymentKey    = "deployment.json"
	defaultConfigMapDeploymentSuffix = "-patch"
	defaultTag                       = "latest"
	patchImageName                   = "patch-image"
	defaultPatchChangeCause          = "gograpple patch"
	changeCauseAnnotation            = "kubernetes.io/change-cause"
	defaultPatchCreator              = "gograpple"
	createdByAnnotation              = "app.kubernetes.io/created-by"
)

type Grapple struct {
	deployment v1.Deployment
	kubeCmd    *exec.KubectlCmd
	dockerCmd  *exec.DockerCmd
	goCmd      *exec.GoCmd
}

func NewGrapple(namespace, deployment string) (*Grapple, error) {
	le := log.Entry("")
	g := &Grapple{}
	g.kubeCmd = exec.NewKubectlCommand()
	g.dockerCmd = exec.NewDockerCommand()
	g.goCmd = exec.NewGoCommand()
	g.kubeCmd.Logger(le)
	g.dockerCmd.Logger(le)
	g.goCmd.Logger(le)
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
