package gograpple

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/foomo/gograpple/delve"
)

const delveBin = "dlv"

func (g Grapple) Delve(pod, container, sourcePath string, binArgs []string, host string, port int, delveContinue, vscode bool) error {
	// validate k8s resources for delve session
	if err := g.validatePod(&pod); err != nil {
		return err
	}
	if err := g.validateContainer(&container); err != nil {
		return err
	}
	if !g.isPatched() {
		return fmt.Errorf("deployment not patched, stopping delve")
	}
	// populate bin args if empty
	if len(binArgs) == 0 {
		var err error
		d, err := g.kubeCmd.GetDeploymentFromConfigMap(g.deployment.Name, defaultConfigMapDeploymentKey)
		if err != nil {
			return err
		}
		c, err := g.kubeCmd.GetContainerFromDeployment(container, d)
		if err != nil {
			return err
		}
		binArgs = c.Args
	}
	// validate sourcePath
	goModPath, err := findGoProjectRoot(sourcePath)
	if err != nil {
		return fmt.Errorf("couldnt find go.mod path for source %q", sourcePath)
	}

	delveServer := delve.NewKubeDelveServer(g.l, host, port)
	return g.registerInterrupt(2*time.Second).wait(
		func() error {
			g.onExit(delveServer, pod, container)
			return nil
		},
		func() error {
			// on(Re)Load
			// run pre-start cleanup
			if err := g.cleanupDelve(pod, container); err != nil {
				return err
			}
			// deploy bin
			if err := g.deployBin(pod, container, goModPath, sourcePath); err != nil {
				return err
			}
			// exec delve server and port-forward to pod
			go delveServer.Start(pod, container, g.binDestination(), delveContinue, binArgs)
			// check server state with delve client
			go g.checkDelveConnection(host, port)
			// start vscode
			if vscode {
				if err := launchVSCode(g.l, goModPath, host, port, 5); err != nil {
					return err
				}
			}
			defer g.onExit(delveServer, pod, container)
			return nil
		},
	)
}

func (g Grapple) onExit(ds *delve.KubeDelveServer, pod, container string) {
	// onExit
	// try stopping the delve server regularly
	if err := ds.Stop(); err != nil {
		g.l.WithError(err).Warn("could not stop delve regularly")
	}
	// kill the remaining pids
	if err := g.cleanupDelve(pod, container); err != nil {
		g.l.WithError(err).Warn("could not cleanup delve")
	}
}

func (g Grapple) binName() string {
	return g.deployment.Name
}

func (g Grapple) binDestination() string {
	return "/" + g.binName()
}

func (g Grapple) cleanupDelve(pod, container string) error {
	// get pids of delve and app were debugging
	g.l.Info("killing debug processes")
	binPids, errBinPids := g.getPIDsOf(pod, container, g.binName())
	if errBinPids != nil {
		return errBinPids
	}
	delvePids, errDelvePids := g.getPIDsOf(pod, container, delveBin)
	if errDelvePids != nil {
		return errDelvePids
	}
	// kill pids directly on pod container
	maxTries := 10
	pids := append(binPids, delvePids...)
	return tryCall(g.l, maxTries, time.Millisecond*200, func(i int) error {
		killErrs := g.kubeCmd.KillPidsOnPod(pod, container, pids, true)
		if len(killErrs) == 0 {
			return nil
		}
		return fmt.Errorf("could not kill processes after %v attempts", maxTries)
	})
}

func (g Grapple) deployBin(pod, container, goModPath, sourcePath string) error {
	binSource := path.Join(os.TempDir(), g.binName())
	g.l.Infof("building %q for debug", sourcePath)

	var relInputs []string
	inputInfo, errInputInfo := os.Stat(sourcePath)
	if errInputInfo != nil {
		return errInputInfo
	}
	if inputInfo.IsDir() {
		if files, err := os.ReadDir(sourcePath); err != nil {
			return err
		} else {
			for _, file := range files {
				if path.Ext(file.Name()) == ".go" {
					relInputs = append(relInputs, strings.TrimPrefix(path.Join(sourcePath, file.Name()), goModPath+string(filepath.Separator)))
				}
			}
		}
	} else {
		relInputs = append(relInputs, strings.TrimPrefix(sourcePath, goModPath+string(filepath.Separator)))
	}

	_, errBuild := g.goCmd.Build(goModPath, binSource, relInputs, `-gcflags="all=-N -l"`).Env("GOOS=linux").Run()
	if errBuild != nil {
		return errBuild
	}

	g.l.Infof("copying binary to pod %v", pod)
	_, errCopyToPod := g.kubeCmd.CopyToPod(pod, container, binSource, g.binDestination()).Run()
	return errCopyToPod
}

func (g Grapple) checkDelveConnection(host string, port int) error {
	return tryCall(g.l, 50, 200*time.Millisecond, func(i int) error {
		delveClient, err := delve.NewKubeDelveClient(host, port, 3*time.Second)
		if err != nil {
			g.l.WithError(err).Warn("couldnt connect to delve server")
			return err
		}
		if err := delveClient.ValidateState(); err != nil {
			g.l.WithError(err).Warn("couldnt get running state from delve server")
			return err
		}
		return nil
	})
}
