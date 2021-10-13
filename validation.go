package gograpple

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	v1 "k8s.io/api/apps/v1"
)

func ValidateImage(d v1.Deployment, container string, image, tag *string) error {
	if *image == "" {
		for _, c := range d.Spec.Template.Spec.Containers {
			if container == c.Name {
				colonPieces := strings.Split(c.Image, ":")
				if len(colonPieces) < 2 {
					// invalid
					return fmt.Errorf("deployment image %q has invalid format", c.Image)
				} else if len(colonPieces) > 2 {
					// there might be a repo with a port in there
					*image = strings.Join(colonPieces[:len(colonPieces)-1], ":")
					*tag = colonPieces[len(colonPieces)-1]
				} else {
					// image:tag
					*image = colonPieces[0]
					*tag = colonPieces[1]
				}
				return nil
			}
		}
	}
	return nil
}

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
