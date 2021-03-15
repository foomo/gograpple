package gograpple

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/foomo/gograpple/exec"
	"github.com/sirupsen/logrus"
)

type launchArgs struct {
	Name       string `json:"name,omitempty"`
	Request    string `json:"request,omitempty"`
	Type       string `json:"type,omitempty"`
	Mode       string `json:"mode,omitempty"`
	RemotePath string `json:"remotePath,omitempty"`
	Port       int    `json:"port,omitempty"`
	Host       string `json:"host,omitempty"`
	Trace      string `json:"trace,omitempty"`
	LogOutput  string `json:"logOutput,omitempty"`
	ShowLog    bool   `json:"showLog,omitempty"`
}

func newLaunchArgs(pod, host string, port, iteration int, workspaceFolder string) *launchArgs {
	return &launchArgs{
		Host:       host,
		Name:       fmt.Sprintf("delve-%v-run-%v", pod, iteration),
		Port:       port,
		Request:    "attach",
		Type:       "go",
		Mode:       "remote",
		RemotePath: workspaceFolder,

		// Trace:      "verbose",
		// LogOutput: "rpc",
		// ShowLog:   true,
	}
}

func (la *launchArgs) toJson() (string, error) {
	bytes, err := json.Marshal(la)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func launchVscode(l *logrus.Entry, goModDir, pod, host string, port, tries, iteration int, sleep time.Duration) error {

	openFile := goModDir
	workspaceFolder := "${workspaceFolder}"
	// is there a workspace in that dir
	files, errReadDir := ioutil.ReadDir(goModDir)
	if errReadDir == nil {
		for _, file := range files {
			if !file.IsDir() && strings.HasSuffix(file.Name(), ".code-workspace") {
				openFile = filepath.Join(goModDir, file.Name())
				workspaceFolder = goModDir
				break
			}
		}
	}

	exec.NewCommand(l, "code").Args(openFile).PostEnd(func() error {
		return tryCall(tries, time.Millisecond*200, func(i int) error {
			l.Infof("waiting for vscode status (%v/%v)", i, tries)
			_, err := exec.NewCommand(l, "code").Args("-s").Run()
			return err
		})
	}).Run()

	l.Infof("opening debug configuration")
	la, err := newLaunchArgs(pod, host, port, iteration, workspaceFolder).toJson()
	if err != nil {
		return err
	}
	_, err = runOpen(l, `vscode://fabiospampinato.vscode-debug-launcher/launch?args=`+url.QueryEscape(la))
	if err != nil {
		return err
	}
	return nil
}
