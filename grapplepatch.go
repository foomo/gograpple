package gograpple

import (
	"encoding/json"
	"fmt"
	"os"
	"path"

	"github.com/foomo/gograpple/bindata"
)

type Mount struct {
	HostPath  string
	MountPath string
}

type patchValues struct {
	Label          string
	Deployment     string
	Container      string
	ConfigMapMount string
	Mounts         []Mount
	Image          string
}

func newPatchValues(deployment, container string, mounts []Mount) *patchValues {
	return &patchValues{
		Label:          defaultPatchedLabel,
		Deployment:     deployment,
		Container:      container,
		ConfigMapMount: defaultConfigMapMount,
		Mounts:         mounts,
		Image:          defaultPatchImage,
	}
}

func (g Grapple) Patch(image, tag, container string, mounts []Mount) error {
	if g.isPatched() {
		g.l.Warn("deployment already patched, rolling back first")
		if err := g.rollbackUntilUnpatched(); err != nil {
			return err
		}
	}
	if err := g.validateContainer(&container); err != nil {
		return err
	}
	if err := g.validateImage(container, &image, &tag); err != nil {
		return err
	}

	g.l.Infof("creating a ConfigMap with deployment data")
	bs, err := json.Marshal(g.deployment)
	if err != nil {
		return err
	}
	data := map[string]string{defaultConfigMapDeploymentKey: string(bs)}
	_, err = g.kubeCmd.CreateConfigMap(g.deployment.Name, data)
	if err != nil {
		return err
	}

	g.l.Infof("waiting for deployment to get ready")
	_, err = g.kubeCmd.WaitForRollout(g.deployment.Name, defaultWaitTimeout).Run()
	if err != nil {
		return err
	}

	g.l.Infof("extracting patch files")
	const patchFolder = "the-hook"
	if err := bindata.RestoreAssets(os.TempDir(), patchFolder); err != nil {
		return err
	}
	theHookPath := path.Join(os.TempDir(), patchFolder)

	g.l.Infof("building patch image with %v:%v", image, tag)
	_, err = g.dockerCmd.Build(theHookPath, "--build-arg",
		fmt.Sprintf("IMAGE=%v:%v", image, tag), "-t", defaultPatchImage).Run()
	if err != nil {
		return err
	}

	g.l.Infof("rendering deployment patch template")
	patch, err := renderTemplate(
		path.Join(theHookPath, devDeploymentPatchFile),
		newPatchValues(g.deployment.Name, container, mounts),
	)
	if err != nil {
		return err
	}

	g.l.Infof("patching deployment for development %s with patch %s", g.deployment.Name, patch)
	_, err = g.kubeCmd.PatchDeployment(patch, g.deployment.Name).Run()
	return err
}

func (g *Grapple) Rollback() error {
	if !g.isPatched() {
		return fmt.Errorf("deployment not patched, stopping rollback")
	}
	return g.rollbackUntilUnpatched()
}

func (g Grapple) isPatched() bool {
	_, ok := g.deployment.Spec.Template.ObjectMeta.Labels[defaultPatchedLabel]
	return ok
}

func (g *Grapple) rollbackUntilUnpatched() error {
	if !g.isPatched() {
		return nil
	}
	if err := g.rollback(); err != nil {
		return err
	}
	if err := g.updateDeployment(); err != nil {
		return err
	}
	return g.rollbackUntilUnpatched()
}

func (g Grapple) rollback() error {
	g.l.Infof("removing ConfigMap %v", g.deployment.Name)
	_, err := g.kubeCmd.DeleteConfigMap(g.deployment.Name)
	if err != nil {
		// may not exist
		g.l.Warn(err)
	}

	g.l.Infof("waiting for deployment to get ready")
	_, err = g.kubeCmd.WaitForRollout(g.deployment.Name, defaultWaitTimeout).Run()
	if err != nil {
		return err
	}

	g.l.Infof("rolling back deployment %v", g.deployment.Name)
	_, err = g.kubeCmd.RollbackDeployment(g.deployment.Name).Run()
	if err != nil {
		return err
	}
	return nil
}
