package gograpple

import (
	"fmt"
	"strings"
)

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
