package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/bitfield/script"
	"github.com/c-bata/go-prompt"
	"github.com/runz0rd/gencon"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

var log *logrus.Entry

func init() {
	log = logrus.NewEntry(logrus.StandardLogger())
}

const defaultImage = "alpine:latest"

// load the already existing config or enter interactive mode to generate one
func Interact(filePath string, config interface{}) error {
	defer handleConfigExit()
	var opts []gencon.Option
	if filePath != "" {
		configLoaded := false
		if _, err := os.Stat(filePath); err == nil {
			if err := loadYaml(filePath, config); err != nil {
				// if the config path doesnt exist
				return err
			}
			configLoaded = true
		}
		if configLoaded {
			// skip filled when loaded from file
			opts = append(opts, gencon.OptionSkipFilled())
		}
	}
	// run configuration create with suggestions
	w, err := gencon.New(opts...)
	if err != nil {
		return err
	}
	if err := w.Prompt(config,
		prompt.OptionShowCompletionAtStart(),
		prompt.OptionPrefixTextColor(prompt.Fuchsia),
		// since we have a file completer
		prompt.OptionCompletionWordSeparator("/"),
		// handle ctrl+c exit
		prompt.OptionAddKeyBind(prompt.KeyBind{
			Key: prompt.ControlC,
			Fn:  promptExit,
		})); err != nil {
		return err
	}
	return save(filePath, config)
}

func save(path string, c interface{}) error {
	log.Infof("saving %q", path)
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

type PromptExit int

func promptExit(_ *prompt.Buffer) {
	fmt.Println()
	panic(PromptExit(0))
}

func handleConfigExit() {
	v := recover()
	switch v.(type) {
	case PromptExit:
		log.Info("exiting")
		vInt, _ := v.(int)
		os.Exit(vInt)
		return
	case nil:
		return
	default:
		fmt.Printf("%+v", v)
	}
}

func loadYaml(path string, data interface{}) error {
	bs, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(bs, data)
}

func find(args ...string) ([]string, error) {
	return script.Exec(fmt.Sprintf("find %v", strings.Join(args, " "))).Slice()
}

func findContaining(v string, args ...string) ([]string, error) {
	return script.Exec(fmt.Sprintf("find %v -exec grep -lr %q {} +", strings.Join(args, " "), v)).Slice()
}
