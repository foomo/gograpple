package suggest

import (
	"fmt"

	"github.com/bitfield/script"
)

func ListRepos() ([]string, error) {
	results, err := script.Exec("docker images --format '{{.Repository}}'").Slice()
	if err != nil {
		return nil, fmt.Errorf(results[0])
	}
	return results, err
}

func ListTag(repository string) ([]string, error) {
	results, err := script.Exec(
		fmt.Sprintf("docker images --format '{{.Tag}}' | grep -i %v", repository)).Slice()
	if err != nil {
		return nil, fmt.Errorf(results[0])
	}
	return results, err
}
