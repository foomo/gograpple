package gograpple

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/c-bata/go-prompt"
	"github.com/c-bata/go-prompt/completer"
	"github.com/foomo/gograpple/kubectl"
	"github.com/foomo/gograpple/suggest"
	"github.com/runz0rd/gencon"
	"gopkg.in/yaml.v3"
)

type Config struct {
	SourcePath    string `yaml:"source_path"`
	Cluster       string `yaml:"cluster"`
	Namespace     string `yaml:"namespace" depends:"Cluster"`
	Deployment    string `yaml:"deployment" depends:"Namespace"`
	Container     string `yaml:"container" depends:"Deployment"`
	AttachTo      string `yaml:"attach_to,omitempty" depends:"Container"`
	LaunchVscode  bool   `yaml:"launch_vscode" default:"false"`
	ListenAddr    string `yaml:"listen_addr,omitempty" default:":2345"`
	DelveContinue bool   `yaml:"delve_continue" default:"false"`
	Image         string `yaml:"image,omitempty" default:"alpine:latest"`
}

func (c Config) Addr() (host string, port int, err error) {
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

func (c Config) MarshalYAML() (interface{}, error) {
	// marshal relative paths into absolute
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

func (c Config) ClusterSuggest(d prompt.Document) []prompt.Suggest {
	return suggest.Completer(d, suggest.MustList(kubectl.ListContexts))
}

func (c Config) NamespaceSuggest(d prompt.Document) []prompt.Suggest {
	kubectl.SetContext(c.Cluster)
	return suggest.Completer(d, suggest.MustList(kubectl.ListNamespaces))
}

func (c Config) DeploymentSuggest(d prompt.Document) []prompt.Suggest {
	return suggest.Completer(d, suggest.MustList(func() ([]string, error) {
		return kubectl.ListDeployments(c.Namespace)
	}))
}

func (c Config) ContainerSuggest(d prompt.Document) []prompt.Suggest {
	return suggest.Completer(d, suggest.MustList(func() ([]string, error) {
		return kubectl.ListContainers(c.Namespace, c.Deployment)
	}))
}

func (c Config) AttachToSuggest(d prompt.Document) []prompt.Suggest {
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

func (c Config) LaunchVscodeSuggest(d prompt.Document) []prompt.Suggest {
	return []prompt.Suggest{{Text: "true"}, {Text: "false"}}
}

func (c Config) ListenAddrSuggest(d prompt.Document) []prompt.Suggest {
	return []prompt.Suggest{{Text: ":2345"}}
}

func (c Config) DelveContinueSuggest(d prompt.Document) []prompt.Suggest {
	return []prompt.Suggest{{Text: "true"}, {Text: "false"}}
}

func (c Config) ImageSuggest(d prompt.Document) []prompt.Suggest {
	suggestions := suggest.Completer(d, suggest.MustList(func() ([]string, error) {
		return kubectl.ListImages(c.Namespace, c.Deployment)
	}))
	return append(suggestions, prompt.Suggest{Text: defaultImage})
}

func LoadConfig(path string) (Config, error) {
	var c Config
	if _, err := os.Stat(path); err != nil {
		// needed due to panicking in ctrl+c binding (library limitation)
		defer handleExit()
		// if the config path doesnt exist
		// run configuration create with suggestions
		gencon.New(
			prompt.OptionShowCompletionAtStart(),
			prompt.OptionPrefixTextColor(prompt.Fuchsia),
			// since we have a file completer
			prompt.OptionCompletionWordSeparator("/"),
			// handle ctrl+c exit
			prompt.OptionAddKeyBind(prompt.KeyBind{
				Key: prompt.ControlC,
				Fn:  promptExit,
			}),
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

type Exit int

func promptExit(_ *prompt.Buffer) {
	panic(Exit(0))
}

func handleExit() {
	v := recover()
	switch v.(type) {
	case nil:
		return
	case Exit:
		vInt, _ := v.(int)
		os.Exit(vInt)
	default:
		fmt.Printf("%+v", v)
	}
}
