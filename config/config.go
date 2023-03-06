package config

import (
	"fmt"
	"io/ioutil"
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

func Generate(config interface{}) error {
	defer handleConfigExit()
	var opts []gencon.Option
	w, err := gencon.New(opts...)
	if err != nil {
		return err
	}
	return w.Prompt(config,
		prompt.OptionShowCompletionAtStart(),
		prompt.OptionPrefixTextColor(prompt.Fuchsia),
		// since we have a file completer
		prompt.OptionCompletionWordSeparator("/"),
		// handle ctrl+c exit
		prompt.OptionAddKeyBind(prompt.KeyBind{
			Key: prompt.ControlC,
			Fn:  promptExit,
		}))
}

func Load(path string, c interface{}) error {
	log.Infof("loading %q", path)
	if _, err := os.Stat(filePath); err == nil {
		if err := LoadYaml(filePath, config); err != nil {
			// if the config path doesnt exist
			return err
		}
	}
}

func Save(path string, c interface{}) error {
	log.Infof("saving %q", path)
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(path, data, 0644)
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

func LoadYaml(path string, data interface{}) error {
	bs, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(bs, data)
}

func Find(args ...string) ([]string, error) {
	return script.Exec(fmt.Sprintf("find %v", strings.Join(args, " "))).Slice()
}
