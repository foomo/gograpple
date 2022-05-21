package gograpple

import (
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/c-bata/go-prompt"
	"github.com/c-bata/go-prompt/completer"
	"github.com/foomo/gograpple/suggest"
	"github.com/runz0rd/gencon"
	"gopkg.in/yaml.v3"
)

type Config struct {
	SourcePath string `yaml:"source_path"`
	Image      string `yaml:"image,omitempty" depends:"Deployment"`
	Cluster    string `yaml:"cluster"`
	Namespace  string `yaml:"namespace" depends:"Cluster"`
	Deployment string `yaml:"deployment" depends:"Namespace"`
	Container  string `yaml:"container,omitempty" depends:"Deployment"`
	Repository string `yaml:"repository,omitempty" depends:"Deployment"`
	Launch     string `yaml:"launch,omitempty"`
}

func (c Config) MarshalYAML() (interface{}, error) {
	// marshal SourcePath into absolute path
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	c.SourcePath = path.Join(cwd, c.SourcePath)
	type alias Config
	node := yaml.Node{}
	err = node.Encode(alias(c))
	if err != nil {
		return nil, err
	}
	return node, nil
}

func (c Config) SourcePathSuggest(d prompt.Document) []prompt.Suggest {
	completer := completer.FilePathCompleter{
		IgnoreCase: true,
		Filter: func(fi os.FileInfo) bool {
			return fi.IsDir() || strings.HasSuffix(fi.Name(), ".go")
		},
	}
	return completer.Complete(d)
}

func (c Config) ImageSuggest(d prompt.Document) []prompt.Suggest {
	kc := suggest.KubeConfig(suggest.DefaultKubeConfig)
	suggestions := suggest.Completer(d, suggest.MustList(func() ([]string, error) {
		return kc.ListImages(c.Namespace, c.Deployment)
	}))
	return append(suggestions, prompt.Suggest{Text: "golang:latest"})
}

func (c Config) ClusterSuggest(d prompt.Document) []prompt.Suggest {
	kc := suggest.KubeConfig(suggest.DefaultKubeConfig)
	return suggest.Completer(d, suggest.MustList(kc.ListContexts))
}

func (c Config) NamespaceSuggest(d prompt.Document) []prompt.Suggest {
	kc := suggest.KubeConfig(suggest.DefaultKubeConfig)
	kc.SetContext(c.Cluster)
	return suggest.Completer(d, suggest.MustList(kc.ListNamespaces))
}

func (c Config) DeploymentSuggest(d prompt.Document) []prompt.Suggest {
	kc := suggest.KubeConfig(suggest.DefaultKubeConfig)
	return suggest.Completer(d, suggest.MustList(func() ([]string, error) {
		return kc.ListDeployments(c.Namespace)
	}))
}

func (c Config) ContainerSuggest(d prompt.Document) []prompt.Suggest {
	kc := suggest.KubeConfig(suggest.DefaultKubeConfig)
	return suggest.Completer(d, suggest.MustList(func() ([]string, error) {
		return kc.ListContainers(c.Namespace, c.Deployment)
	}))
}

func (c Config) RepositorySuggest(d prompt.Document) []prompt.Suggest {
	kc := suggest.KubeConfig(suggest.DefaultKubeConfig)
	return suggest.Completer(d, suggest.MustList(func() ([]string, error) {
		return kc.ListRepositories(c.Namespace, c.Deployment)
	}))
}

func (c Config) LaunchSuggest(d prompt.Document) []prompt.Suggest {
	return []prompt.Suggest{{Text: "vscode"}, {Text: "goland"}}
}

func LoadConfig(path string) (Config, error) {
	var c Config
	if _, err := os.Stat(path); err != nil {
		// if the config path doesnt exist
		// run configuration create with suggestions
		gencon.New(
			prompt.OptionShowCompletionAtStart(),
			prompt.OptionPrefixTextColor(prompt.Fuchsia),
			// since we have a file completer
			prompt.OptionCompletionWordSeparator("/"),
		).Run(&c)
		// save yaml file
		data, err := yaml.Marshal(c)
		if err != nil {
			return c, err
		}
		err = ioutil.WriteFile(path, data, 0644)
		if err != nil {
			return c, err
		}
	}
	err := LoadYaml(path, &c)
	return c, err
}
