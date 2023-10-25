package config

import (
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/c-bata/go-prompt"
	"github.com/foomo/gograpple/internal/kubectl"
	"github.com/foomo/gograpple/internal/suggest"
	"gopkg.in/yaml.v3"
)

type AttachConfig struct {
	SourcePath string `yaml:"source_path"`
	Cluster    string `yaml:"cluster"`
	Namespace  string `yaml:"namespace" depends:"Cluster"`
	Deployment string `yaml:"deployment" depends:"Namespace"`
	Container  string `yaml:"container" depends:"Deployment"`
	ListenAddr string `yaml:"listen_addr,omitempty" default:"127.0.0.1:2345"`

	AttachTo string `yaml:"attach_to" depends:"Container"`
	Arch     string `yaml:"arch" default:"amd64"`
}

func (c AttachConfig) Addr() (host string, port int, err error) {
	pieces := strings.Split(c.ListenAddr, ":")
	if len(pieces) != 2 {
		return host, port, fmt.Errorf("unable to parse addr from %q", c.ListenAddr)
	}
	host = pieces[0]
	if host == "" {
		host = "127.0.0.1"
	}
	if port, err = strconv.Atoi(pieces[1]); err != nil {
		return host, port, err
	}
	return host, port, err
}

func (c AttachConfig) MarshalYAML() (interface{}, error) {
	// marshal relative paths into absolute
	if !path.IsAbs(c.SourcePath) && c.SourcePath != "" {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		c.SourcePath = path.Join(cwd, c.SourcePath)
	}
	type alias AttachConfig
	node := yaml.Node{}
	err := node.Encode(alias(c))
	if err != nil {
		return nil, err
	}
	return node, nil
}

func (c AttachConfig) SourcePathSuggest(d prompt.Document) []prompt.Suggest {
	return suggest.Completer(d, suggest.MustList(func() ([]string, error) {
		return findContaining("package main", ".", "-type", "f", "-name", "*.go")
	}))
}

func (c AttachConfig) ClusterSuggest(d prompt.Document) []prompt.Suggest {
	return suggest.Completer(d, suggest.MustList(kubectl.ListContexts))
}

func (c AttachConfig) NamespaceSuggest(d prompt.Document) []prompt.Suggest {
	kubectl.SetContext(c.Cluster)
	return suggest.Completer(d, suggest.MustList(kubectl.ListNamespaces))
}

func (c AttachConfig) DeploymentSuggest(d prompt.Document) []prompt.Suggest {
	return suggest.Completer(d, suggest.MustList(func() ([]string, error) {
		return kubectl.ListDeployments(c.Namespace)
	}))
}

func (c AttachConfig) ContainerSuggest(d prompt.Document) []prompt.Suggest {
	return suggest.Completer(d, suggest.MustList(func() ([]string, error) {
		return kubectl.ListContainers(c.Namespace, c.Deployment)
	}))
}

func (c AttachConfig) ListenAddrSuggest(d prompt.Document) []prompt.Suggest {
	return []prompt.Suggest{{Text: ":2345"}}
}

func (c AttachConfig) AttachToSuggest(d prompt.Document) []prompt.Suggest {
	return suggest.Completer(d, suggest.MustList(func() ([]string, error) {
		d, err := kubectl.GetDeployment(c.Namespace, c.Deployment)
		if err != nil {
			return nil, err
		}
		pod, err := kubectl.GetMostRecentRunningPodBySelectors(c.Namespace, d.Spec.Selector.MatchLabels)
		if err != nil {
			return nil, err
		}
		ps, err := kubectl.ExecPod(c.Namespace, pod, c.Container, []string{"ps", "-o", "comm"}).Replace("COMMAND", "").String()
		if err != nil {
			return nil, err
		}
		return strings.Split(strings.Trim(ps, "\n"), "\n"), nil
	}))
}

func (c AttachConfig) ArchSuggest(d prompt.Document) []prompt.Suggest {
	return []prompt.Suggest{{Text: "amd64"}, {Text: "arm64"}}
}
