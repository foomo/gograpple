package gograpple

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/foomo/gograpple/delve"
	"github.com/sirupsen/logrus"
)

const delveBin = "dlv"

func (g Grapple) Delve(pod, container, sourcePath string, binArgs []string, host string, port int, delveContinue, vscode bool) error {
	// validate k8s resources for delve session
	if err := g.kubeCmd.ValidatePod(g.deployment, &pod); err != nil {
		return err
	}
	if err := g.kubeCmd.ValidateContainer(g.deployment, &container); err != nil {
		return err
	}
	if !g.isPatched() {
		return fmt.Errorf("deployment not patched, stopping delve")
	}
	// populate bin args if empty
	if len(binArgs) == 0 {
		var err error
		d, err := g.kubeCmd.GetDeploymentFromConfigMap(g.DeploymentConfigMapName(), defaultConfigMapDeploymentKey)
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

	RunWithInterrupt(g.l, func(ctx context.Context) {
		// run pre-start cleanup
		clog := g.componentLog("cleanup")
		clog.Info("running pre-start cleanup")
		if err := g.cleanupPIDs(ctx, pod, container); err != nil {
			clog.Error(err)
			return
		}
		// deploy bin
		dlog := g.componentLog("deploy")
		dlog.Info("building and deploying bin")
		if err := g.deployBin(ctx, pod, container, goModPath, sourcePath); err != nil {
			dlog.Error(err)
			return
		}
		// start delve server
		dslog := g.componentLog("server")
		dslog.Infof("starting delve server on %v:%v", host, port)
		ds := delve.NewKubeDelveServer(dslog, g.deployment.Namespace, host, port)
		ds.StartNoWait(ctx, pod, container, g.binDestination(), delveContinue, binArgs)
		// port forward to pod with delve server
		dclog := g.componentLog("client")
		g.portForwardDelve(dclog, ctx, pod, host, port)
		// check server state with delve client
		if err := g.checkDelveConnection(dclog, ctx, 10, host, port); err != nil {
			dclog.WithError(err).Error("couldnt connect to delver server")
			return
		}
		// launch vscode
		if vscode {
			vlog := g.componentLog("vscode")
			if err := launchVSCode(ctx, vlog, goModPath, host, port, 5); err != nil {
				vlog.WithError(err).Error("couldnt launch vscode")
			}
		}
	})
	defer g.cleanupPIDs(context.Background(), pod, container)
	return nil
}
func (g Grapple) componentLog(name string) *logrus.Entry {
	return g.l.WithField("component", name)
}

func (g Grapple) binName() string {
	return g.deployment.Name
}

func (g Grapple) binDestination() string {
	return "/" + g.binName()
}

func (g Grapple) cleanupPIDs(ctx context.Context, pod, container string) error {
	// get pids of delve and app were debugging
	binPids, errBinPids := g.kubeCmd.GetPIDsOf(pod, container, g.binName())
	if errBinPids != nil {
		return errBinPids
	}
	delvePids, errDelvePids := g.kubeCmd.GetPIDsOf(pod, container, delveBin)
	if errDelvePids != nil {
		return errDelvePids
	}
	// kill pids directly on pod container
	maxTries := 10
	pids := append(binPids, delvePids...)
	return tryCallWithContext(ctx, maxTries, time.Millisecond*200, func(i int) error {
		killErrs := g.kubeCmd.KillPidsOnPod(pod, container, pids, true)
		if len(killErrs) == 0 {
			return nil
		}
		return fmt.Errorf("could not kill processes after %v attempts", maxTries)
	})
}

func (g Grapple) deployBin(ctx context.Context, pod, container, goModPath, sourcePath string) error {
	// build bin
	binSource := path.Join(os.TempDir(), g.binName())
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

	_, errBuild := g.goCmd.Build(goModPath, binSource, relInputs, `-gcflags="all=-N -l"`).Env("GOOS=linux").RunCtx(ctx)
	if errBuild != nil {
		return errBuild
	}
	// copy bin to pod
	_, errCopyToPod := g.kubeCmd.CopyToPod(pod, container, binSource, g.binDestination()).RunCtx(ctx)
	return errCopyToPod
}

func (g Grapple) portForwardDelve(l *logrus.Entry, ctx context.Context, pod, host string, port int) {
	l.Info("port-forwarding pod for delve server")
	cmd := g.kubeCmd.PortForwardPod(pod, host, port)
	go func() {
		_, err := cmd.RunCtx(ctx)
		if err != nil && err.Error() != "signal: killed" {
			l.WithError(err).Errorf("port-forwarding %v pod failed", pod)
		}
	}()
	<-cmd.Started()
}

func (g Grapple) checkDelveConnection(l *logrus.Entry, ctx context.Context, tries int, host string, port int) error {
	time.Sleep(1 * time.Second) // allow delve to become available
	err := tryCallWithContext(ctx, tries, 1*time.Second, func(i int) error {
		l.Infof("connecting to %v:%v (%d/%d)", host, port, i, tries)
		dc, err := delve.NewKubeDelveClient(ctx, host, port)
		if err != nil {
			l.WithError(err).Warn("couldnt connect to delve server")
			return err
		}
		defer func() {
			if err := dc.Close(); err != nil {
				l.WithError(err).Warn("couldnt close delve client")
			}
		}()
		if err := dc.ValidateState(); err != nil {
			l.WithError(err).Warn("couldnt get running state from delve server")
			return err
		}
		return nil
	})
	if err == nil {
		l.Infof("delve server connection and state ok")
	}
	return err
}
