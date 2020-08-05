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
