package gograpple

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
)

var (
	//go:embed the-hook
	bindata embed.FS
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

func (g Grapple) newPatchValues(deployment, container, image string, mounts []Mount) *patchValues {
	return &patchValues{
		Label:          defaultPatchedLabel,
		Deployment:     deployment,
		Container:      container,
		ConfigMapMount: defaultConfigMapMount,
		Mounts:         mounts,
		Image:          image,
	}
}

func (g Grapple) Patch(repo, dockerfile, container string, mounts []Mount) error {
	ctx := context.Background()
	if g.isPatched() {
		g.l.Warn("deployment already patched, rolling back first")
		if err := g.rollback(ctx); err != nil {
			return err
		}
	}
	if err := g.kubeCmd.ValidateContainer(g.deployment, &container); err != nil {
		return err
	}

	g.l.Infof("creating a configmap with deployment data")
	bs, err := json.Marshal(g.deployment)
	if err != nil {
		return err
	}
	_, _ = g.kubeCmd.DeleteConfigMap(g.DeploymentConfigMapName()).Run(ctx)
	data := map[string]string{defaultConfigMapDeploymentKey: string(bs)}
	_, err = g.kubeCmd.CreateConfigMap(g.DeploymentConfigMapName(), data).Run(ctx)
	if err != nil {
		return err
	}

	g.l.Infof("waiting for deployment to get ready")
	_, err = g.kubeCmd.WaitForRollout(g.deployment.Name, defaultWaitTimeout).Run(ctx)
	if err != nil {
		return err
	}

	g.l.Infof("extracting patch files")

	const (
		patchFolder    = "the-hook"
		dockerfileName = "Dockerfile"
		patchFileName  = "deployment-patch.yaml"
		perm           = 0700
	)

	patchDockerfile, err := bindata.ReadFile(filepath.Join(patchFolder, dockerfileName))
	if err != nil {
		return err
	}
	deploymentPatch, err := bindata.ReadFile(filepath.Join(patchFolder, patchFileName))
	if err != nil {
		return err
	}

	theHookPath := path.Join(os.TempDir(), patchFolder)
	_ = os.Mkdir(theHookPath, perm)
	err = os.WriteFile(filepath.Join(theHookPath, dockerfileName), patchDockerfile, perm)
	if err != nil {
		return err
	}
	err = os.WriteFile(filepath.Join(theHookPath, patchFileName), deploymentPatch, perm)
	if err != nil {
		return err
	}

	image := defaultImage
	if dockerfile != "" {
		g.l.Infof("building dockerfile %v", dockerfile)
		_, err = g.dockerCmd.Build(
			filepath.Dir(dockerfile),
			"-f", dockerfile, "-t", patchImageName,
			"--platform", "linux/amd64").Run(ctx)
		if err != nil {
			return err
		}
		image = patchImageName
	}

	patchedImage := g.patchedImageName(repo)
	completePatchedImage := fmt.Sprintf("%v:%v", patchedImage, defaultTag)
	g.l.Infof("building patch image with %v", completePatchedImage)
	_, err = g.dockerCmd.Build(theHookPath, "--build-arg",
		fmt.Sprintf("IMAGE=%v", image), "-t", completePatchedImage,
		"--platform", "linux/amd64").Run(ctx)
	if err != nil {
		return err
	}

	if repo != "" {
		//contains a repo, push the built image
		g.l.Infof("pushing patch image with %v", completePatchedImage)
		_, err = g.dockerCmd.Push(patchedImage, defaultTag).Run(ctx)
		if err != nil {
			return err
		}
	}

	g.l.Infof("rendering deployment patch template")
	patch, err := renderTemplate(
		path.Join(theHookPath, devDeploymentPatchFile),
		g.newPatchValues(g.deployment.Name, container, completePatchedImage, mounts),
	)
	if err != nil {
		return err
	}

	g.l.Infof("patching deployment for development %s", g.deployment.Name)
	_, err = g.kubeCmd.PatchDeployment(patch, g.deployment.Name).Run(ctx)
	if err != nil {
		return err
	}

	g.l.Infof("waiting for deployment to get ready")
	_, err = g.kubeCmd.WaitForRollout(g.deployment.Name, defaultWaitTimeout).Run(ctx)
	return err
}

func (g *Grapple) Rollback() error {
	g.l.Info("rolling back")
	if !g.isPatched() {
		return fmt.Errorf("deployment not patched, stopping rollback")
	}
	return g.rollback(context.Background())
}

func (g Grapple) isPatched() bool {
	d, err := g.kubeCmd.GetDeployment(context.Background(), g.deployment.Name)
	if err != nil {
		return false
	}
	_, ok := d.Spec.Template.ObjectMeta.Labels[defaultPatchedLabel]
	return ok
}

func (g Grapple) rollback(ctx context.Context) error {
	revision, err := g.helmCmd.GetLatestRevision(ctx, g.deployment.Name)
	if err != nil {
		return err
	}
	for i := revision - 1; i >= 0; i-- {
		g.l.Infof("removing configmap %v", g.DeploymentConfigMapName())
		_, err := g.kubeCmd.DeleteConfigMap(g.DeploymentConfigMapName()).Run(ctx)
		if err != nil {
			// may not exist
			g.l.Warn("invalid patch state! label present but no configmap found")
		}
		g.l.Infof("rolling back deployment %v to revision %v", g.deployment.Name, i)
		if _, err = g.helmCmd.Rollback(g.deployment.Name, i).Run(ctx); err != nil {
			return err
		}
		if !g.isPatched() {
			// if the deployment is unpatched, exit
			return nil
		}
	}
	return fmt.Errorf("couldnt rollback deployment %v into unpatched state", g.deployment.Name)
}

func (g Grapple) DeploymentConfigMapName() string {
	return g.deployment.Name + defaultConfigMapDeploymentSuffix
}

func (g Grapple) patchedImageName(repo string) string {
	if repo != "" {
		return path.Join(repo, g.deployment.Name) + defaultPatchImageSuffix
	}
	return g.deployment.Name + defaultPatchImageSuffix
}

func (g *Grapple) updateDeployment() error {
	d, err := g.kubeCmd.GetDeployment(context.Background(), g.deployment.Name)
	if err != nil {
		return err
	}
	g.deployment = *d
	return nil
}
