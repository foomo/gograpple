package grapple

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/foomo/gograpple/internal/exec"
	"github.com/foomo/gograpple/util"
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

func newLaunchArgs(host string, port int, workspaceFolder string) *launchArgs {
	return &launchArgs{
		Host:       host,
		Name:       fmt.Sprintf("delve-%v", time.Now().Unix()),
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

func launchVSCode(ctx context.Context, l *logrus.Entry, goModDir, host string, port, tries int) error {
	openFile := goModDir
	workspaceFolder := "${workspaceFolder}"
	// is there a workspace in that dir
	files, errReadDir := os.ReadDir(goModDir)
	if errReadDir == nil {
		for _, file := range files {
			if !file.IsDir() && strings.HasSuffix(file.Name(), ".code-workspace") {
				openFile = filepath.Join(goModDir, file.Name())
				workspaceFolder = goModDir
				break
			}
		}
	}

	exec.NewCommand("code").Logger(l).Args(openFile).PostEnd(func() error {
		return tryCallWithContext(ctx, tries, 200*time.Millisecond, func(i int) error {
			l.Infof("waiting for vscode status (%v/%v)", i, tries)
			_, err := exec.NewCommand("code").Logger(l).Args("-s").Run(ctx)
			return err
		})
	}).Run(ctx)

	l.Infof("opening debug configuration")
	la, err := newLaunchArgs(host, port, workspaceFolder).toJson()
	if err != nil {
		return err
	}
	_, err = util.Open(l, ctx, `vscode://fabiospampinato.vscode-debug-launcher/launch?args=`+url.QueryEscape(la))
	if err != nil {
		return err
	}
	return nil
}
