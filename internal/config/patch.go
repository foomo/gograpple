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

type PatchConfig struct {
	SourcePath string `yaml:"source_path"`
	Cluster    string `yaml:"cluster"`
	Namespace  string `yaml:"namespace" depends:"Cluster"`
	Deployment string `yaml:"deployment" depends:"Namespace"`
	Container  string `yaml:"container" depends:"Deployment"`
	ListenAddr string `yaml:"listen_addr,omitempty" default:"127.0.0.1:2345"`

	Image         string `yaml:"image,omitempty" default:"alpine:latest"`
	DelveContinue bool   `yaml:"delve_continue" default:"false"`
	LaunchVscode  bool   `yaml:"launch_vscode" default:"false"`
}

func (c PatchConfig) Addr() (host string, port int, err error) {
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

func (c PatchConfig) MarshalYAML() (interface{}, error) {
	// marshal relative paths into absolute
	if !path.IsAbs(c.SourcePath) && c.SourcePath != "" {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		c.SourcePath = path.Join(cwd, c.SourcePath)
	}
	type alias PatchConfig
	node := yaml.Node{}
	err := node.Encode(alias(c))
	if err != nil {
		return nil, err
	}
	return node, nil
}

func (c PatchConfig) SourcePathSuggest(d prompt.Document) []prompt.Suggest {
	return suggest.Completer(d, suggest.MustList(func() ([]string, error) {
		return findContaining("package main", ".", "-type", "f", "-name", "*.go")
	}))
}

func (c PatchConfig) ClusterSuggest(d prompt.Document) []prompt.Suggest {
	return suggest.Completer(d, suggest.MustList(kubectl.ListContexts))
}

func (c PatchConfig) NamespaceSuggest(d prompt.Document) []prompt.Suggest {
	kubectl.SetContext(c.Cluster)
	return suggest.Completer(d, suggest.MustList(kubectl.ListNamespaces))
}

func (c PatchConfig) DeploymentSuggest(d prompt.Document) []prompt.Suggest {
	return suggest.Completer(d, suggest.MustList(func() ([]string, error) {
		return kubectl.ListDeployments(c.Namespace)
	}))
}

func (c PatchConfig) ContainerSuggest(d prompt.Document) []prompt.Suggest {
	return suggest.Completer(d, suggest.MustList(func() ([]string, error) {
		return kubectl.ListContainers(c.Namespace, c.Deployment)
	}))
}

func (c PatchConfig) ListenAddrSuggest(d prompt.Document) []prompt.Suggest {
	return []prompt.Suggest{{Text: ":2345"}}
}

func (c PatchConfig) ImageSuggest(d prompt.Document) []prompt.Suggest {
	suggestions := suggest.Completer(d, suggest.MustList(func() ([]string, error) {
		return kubectl.ListImages(c.Namespace, c.Deployment)
	}))
	return append(suggestions, prompt.Suggest{Text: defaultImage})
}

func (c PatchConfig) DelveContinueSuggest(d prompt.Document) []prompt.Suggest {
	return []prompt.Suggest{{Text: "true"}, {Text: "false"}}
}

func (c PatchConfig) LaunchVscodeSuggest(d prompt.Document) []prompt.Suggest {
	return []prompt.Suggest{{Text: "true"}, {Text: "false"}}
}
