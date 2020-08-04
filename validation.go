package gograpple

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
)

func ValidateMounts(wd string, ms []string) ([]Mount, error) {
	var mounts []Mount
	for _, m := range ms {
		pieces := strings.Split(m, ":")
		if len(pieces) != 2 {
			return nil, fmt.Errorf("bad format for mount %q, should be %q separated", m, ":")
		}
		hostPath := pieces[0]
		mountPath := pieces[1]
		if err := ValidatePath(wd, &hostPath); err != nil {
			return nil, fmt.Errorf("bad format for mount %q, host path bad: %s", m, err)
		}
		if !path.IsAbs(mountPath) {
			return nil, fmt.Errorf("bad format for mount %q, mount path should be absolute", m)
		}
		mounts = append(mounts, Mount{hostPath, mountPath})
	}
	return mounts, nil

}

func validateResource(resourceType, resource, suffix string, available []string) error {
	if !stringIsInSlice(resource, available) {
		return fmt.Errorf("%v %q not found %v, available: %v", resourceType, resource, suffix, strings.Join(available, ", "))
	}
	return nil
}

func ValidatePath(wd string, p *string) error {
	if !filepath.IsAbs(*p) {
		*p = path.Join(wd, *p)
	}
	absPath, err := filepath.Abs(*p)
	if err != nil {
		return err
	}
	_, err = os.Stat(absPath)
	if err != nil {
		return err
	}
	*p = absPath
	return nil
}

func (g Grapple) validateNamespace(namespace string) error {
	available, err := g.kubeCmd.GetNamespaces()
	if err != nil {
		return err
	}
	return validateResource("namespace", namespace, "", available)
}

func (g Grapple) validateDeployment(namespace, deployment string) error {
	available, err := g.kubeCmd.GetDeployments()
	if err != nil {
		return err
	}
	return validateResource("deployment", deployment, fmt.Sprintf("for namespace %q", namespace), available)
}

func (g Grapple) validatePod(pod *string) error {
	if *pod == "" {
		var err error
		*pod, err = g.kubeCmd.GetMostRecentPodBySelectors(g.deployment.Spec.Selector.MatchLabels)
		if err != nil || *pod == "" {
			return err
		}
		return nil
	}
	available, err := g.kubeCmd.GetPods(g.deployment.Spec.Selector.MatchLabels)
	if err != nil {
		return err
	}
	return validateResource("pod", *pod, fmt.Sprintf("for deployment %q", g.deployment.Name), available)
}

func (g Grapple) validateContainer(container *string) error {
	if *container == "" {
		*container = g.deployment.Name
	}
	available := g.kubeCmd.GetContainers(g.deployment)
	return validateResource("container", *container, fmt.Sprintf("for deployment %q", g.deployment.Name), available)
}

func (g Grapple) validateImage(container string, image, tag *string) error {
	if *image == "" {
		for _, c := range g.deployment.Spec.Template.Spec.Containers {
			if container == c.Name {
				pieces := strings.Split(c.Image, ":")
				if len(pieces) != 2 {
					return fmt.Errorf("deployment image %q has invalid format", c.Image)
				}
				*image = pieces[0]
				*tag = pieces[1]
				return nil
			}
		}
	}
	return nil
}
