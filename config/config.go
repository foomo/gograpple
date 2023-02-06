package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/bitfield/script"
	"github.com/c-bata/go-prompt"
	"github.com/runz0rd/gencon"
	"gopkg.in/yaml.v3"
)

const defaultImage = "alpine:latest"

func LoadConfig(base string, c interface{}) error {
	filePath := path.Join(base, "gograpple.yaml")
	defer func() {
		if err := saveConfig(filePath, c); err != nil {
			fmt.Println(err)
		}
		// needed due to panicking in ctrl+c binding (library limitation)
		handleConfigExit()
	}()
	configLoaded := false
	if _, err := os.Stat(filePath); err == nil {
		if err := LoadYaml(filePath, c); err != nil {
			// if the config path doesnt exist
			return err
		}
		configLoaded = true
	}
	var opts []gencon.Option
	if configLoaded {
		// skip filled when loaded from file
		opts = append(opts, gencon.OptionSkipFilled())
	}
	// run configuration create with suggestions
	w, err := gencon.New(opts...)
	if err != nil {
		return err
	}
	w.Prompt(c,
		prompt.OptionShowCompletionAtStart(),
		prompt.OptionPrefixTextColor(prompt.Fuchsia),
		// since we have a file completer
		prompt.OptionCompletionWordSeparator("/"),
		// handle ctrl+c exit
		prompt.OptionAddKeyBind(prompt.KeyBind{
			Key: prompt.ControlC,
			Fn:  promptExit,
		}))
	return nil
}

func saveConfig(path string, c interface{}) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(path, data, 0644)
}

type Exit int

func promptExit(_ *prompt.Buffer) {
	panic(Exit(0))
}

func handleConfigExit() {
	v := recover()
	switch v.(type) {
	case nil:
		return
	case Exit:
		fmt.Println("\nexiting")
		vInt, _ := v.(int)
		os.Exit(vInt)
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
