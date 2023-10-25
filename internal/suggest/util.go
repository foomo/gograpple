package suggest

import (
	"fmt"
	"strings"

	"github.com/c-bata/go-prompt"
)

func Suggestions(items []string) []prompt.Suggest {
	if len(items) == 0 {
		return []prompt.Suggest{}
	}
	s := make([]prompt.Suggest, len(items))
	for i := range items {
		s[i] = prompt.Suggest{
			Text: fmt.Sprint(items[i]),
		}
	}
	return s
}

func MustList(fn func() ([]string, error)) []string {
	results, err := fn()
	if err != nil {
		panic(err)
	}
	return results
}

func Completer(d prompt.Document, items []string) []prompt.Suggest {
	return prompt.FilterContains(Suggestions(items), d.GetWordBeforeCursor(), true)
}

func ParseImage(s string) (repo, name, tag string, err error) {
	pieces := strings.Split(s, "/")
	switch true {
	case len(pieces) == 1 && pieces[0] == s:
		imageTag := strings.Split(s, ":")
		return "", imageTag[0], imageTag[1], nil
	case len(pieces) > 1:
		imageTag := strings.Split(pieces[len(pieces)-1], ":")
		return strings.Join(pieces[:len(pieces)-1], "/"), imageTag[0], imageTag[1], nil
	}
	return "", "", "", fmt.Errorf("invalid image value %q provided", s)
}
