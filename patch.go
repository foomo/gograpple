package gograpple

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/pkg/errors"
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
	ChangeCause    string
	CreatedBy      string
	Deployment     string
	Container      string
	ConfigMapMount string
	Mounts         []Mount
	Image          string
}

func (g Grapple) newPatchValues(deployment, container, image string, mounts []Mount) *patchValues {
	return &patchValues{
		ChangeCause:    defaultPatchChangeCause,
		CreatedBy:      defaultPatchCreator,
		Deployment:     deployment,
		Container:      container,
		ConfigMapMount: defaultConfigMapMount,
		Mounts:         mounts,
		Image:          image,
	}
}

func (g Grapple) Patch(image, container string, mounts []Mount) error {
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
	_, _ = g.kubeCmd.DeleteConfigMap(g.DeploymentConfigMapName()).Quiet().Run(ctx)
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

	// get image used in the deployment so we can and platform
	deploymentImage, err := g.kubeCmd.GetImage(ctx, g.deployment, container)
	if err != nil {
		return err
	}
	// get repo from deployment image
	imageRepo, _, _, err := ParseImage(deploymentImage)
	if err != nil {
		return err
	}
	// get platform from deployment image
	deploymentPlatform, err := g.dockerCmd.GetPlatform(ctx, deploymentImage)
	if err != nil {
		return err
	}

	pathedImageName := g.patchedImageName(imageRepo)
	g.l.Infof("building patch image %v:%v", pathedImageName, defaultTag)
	out, err := g.dockerCmd.Build(theHookPath, "--build-arg",
		fmt.Sprintf("IMAGE=%v", image), "-t", fmt.Sprintf("%v:%v", pathedImageName, defaultTag),
		"--platform", deploymentPlatform.String()).Quiet().Run(ctx)
	if err != nil {
		return errors.WithMessage(err, out)
	}

	if imageRepo != "" {
		//contains a repo, push the built image
		g.l.Infof("pushing patch image %v:%v", pathedImageName, defaultTag)
		_, err = g.dockerCmd.Push(pathedImageName, defaultTag).Run(ctx)
		if err != nil {
			return err
		}
	}

	g.l.Infof("rendering deployment patch template")
	patch, err := renderTemplate(
		path.Join(theHookPath, devDeploymentPatchFile),
		g.newPatchValues(g.deployment.Name, container, fmt.Sprintf("%v:%v", pathedImageName, defaultTag), mounts),
	)
	if err != nil {
		return err
	}

	g.l.Infof("patching deployment %s for development with patch", g.deployment.Name)
	_, err = g.kubeCmd.PatchDeployment(patch, g.deployment.Name).Run(ctx)
	if err != nil {
		return err
	}
	return nil
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
	createdBy, ok := d.Spec.Template.ObjectMeta.Annotations[createdByAnnotation]
	return ok && createdBy == defaultPatchCreator
}

func (g Grapple) rollback(ctx context.Context) error {
	revision, err := g.kubeCmd.GetLatestRevision(ctx, g.deployment.Name)
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
		if _, err = g.kubeCmd.RolloutUndo(g.deployment.Name, i).Run(ctx); err != nil {
			return err
		}
		if !g.isPatched() {
			// annotate rollback
			if _, err = g.kubeCmd.UpdateChangeCause(g.deployment.Name, fmt.Sprintf("rollback to %v", i)).Run(ctx); err != nil {
				return err
			}
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
