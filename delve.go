package gograpple

import (
	"context"
	"fmt"
	"os"
	"path"
	"time"

	"github.com/foomo/gograpple/delve"
	"github.com/foomo/gograpple/exec"
	"github.com/foomo/gograpple/log"
	"github.com/sirupsen/logrus"
)

const delveBin = "dlv"

func (g Grapple) Delve(pod, container, sourcePath string, binArgs []string, host string,
	port int, vscode, delveContinue bool) error {
	ctx := context.Background()
	if !g.isPatched() {
		return fmt.Errorf("deployment not patched, stopping delve")
	}

	// populate bin args if empty
	if len(binArgs) == 0 {
		var err error
		d, err := g.kubeCmd.GetDeploymentFromConfigMap(ctx, g.DeploymentConfigMapName(),
			defaultConfigMapDeploymentKey)
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

	RunWithInterrupt(func(ctx context.Context) {
		log.Logger().Infof("waiting for deployment to get ready")
		_, err := g.kubeCmd.WaitForRollout(g.deployment.Name, defaultWaitTimeout).Run(ctx)
		if err != nil {
			log.Logger().Error(err)
			return
		}
		// validate and get k8s resources for delve session
		if err := g.kubeCmd.ValidatePod(context.Background(), g.deployment, &pod); err != nil {
			log.Logger().Error(err)
			return
		}
		if err := g.kubeCmd.ValidateContainer(g.deployment, &container); err != nil {
			log.Logger().Error(err)
			return
		}

		// run pre-start cleanup
		cl := log.Entry("cleanup")
		cl.Info("running pre-start cleanup")
		if err := g.cleanupPIDs(ctx, pod, container); err != nil {
			cl.Error(err)
			return
		}

		// deploy bin
		dl := log.Entry("deploy")
		dl.Info("building and deploying bin")
		// get image used in the deployment so we can get platform
		deploymentImage, err := g.kubeCmd.GetImage(ctx, g.deployment, container)
		if err != nil {
			dl.Error(err)
			return
		}
		// get platform from deployment image
		deploymentPlatform, err := g.dockerCmd.GetPlatform(ctx, deploymentImage)
		if err != nil {
			dl.Error(err)
			return
		}
		if err := g.deployBin(ctx, pod, container, goModPath, sourcePath, deploymentPlatform); err != nil {
			dl.Error(err)
			return
		}
		// start delve server
		dsl := log.Entry("server")
		dsl.Infof("starting delve server on %v:%v", host, port)
		ds := delve.NewKubeDelveServer(dsl, g.deployment.Namespace, host, port)
		ds.StartNoWait(ctx, pod, container, g.binDestination(), binArgs, delveContinue)
		// port forward to pod with delve server
		dcl := log.Entry("client")
		g.portForwardDelve(dcl, ctx, pod, host, port)
		// check server state with delve client
		if err := g.checkDelveConnection(dcl, ctx, 10, host, port); err != nil {
			dcl.WithError(err).Error("couldnt connect to delver server")
			return
		}
		// launch vscode
		if vscode {
			vl := log.Entry("vscode")
			if err := launchVSCode(ctx, vl, goModPath, host, port, 5); err != nil {
				vl.WithError(err).Error("couldnt launch vscode")
			}
		}
	})
	defer g.cleanupPIDs(context.Background(), pod, container)
	return nil
}

func (g Grapple) binName() string {
	return g.deployment.Name
}

func (g Grapple) binDestination() string {
	return "/" + g.binName()
}

func (g Grapple) cleanupPIDs(ctx context.Context, pod, container string) error {
	// get pids of delve and app were debugging
	binPids, errBinPids := g.kubeCmd.GetPIDsOf(ctx, pod, container, g.binName())
	if errBinPids != nil {
		return errBinPids
	}
	delvePids, errDelvePids := g.kubeCmd.GetPIDsOf(ctx, pod, container, delveBin)
	if errDelvePids != nil {
		return errDelvePids
	}
	// kill pids directly on pod container
	maxTries := 10
	pids := append(binPids, delvePids...)
	return tryCallWithContext(ctx, maxTries, time.Millisecond*200, func(i int) error {
		killErrs := g.kubeCmd.KillPidsOnPod(ctx, pod, container, pids, true)
		if len(killErrs) == 0 {
			return nil
		}
		return fmt.Errorf("could not kill processes after %v attempts", maxTries)
	})
}

func (g Grapple) deployBin(ctx context.Context, pod, container, goModPath, sourcePath string, p *exec.Platform) error {
	// build bin
	binSource := path.Join(os.TempDir(), g.binName())
	_, err := g.goCmd.Build(binSource, []string{sourcePath}, "-gcflags", "-N -l").
		Env(fmt.Sprintf("GOOS=%v", p.OS), fmt.Sprintf("GOARCH=%v", p.Arch), fmt.Sprintf("CGO_ENABLED=%v", 0)).Run(ctx)
	if err != nil {
		return err
	}
	// copy bin to pod
	_, errCopyToPod := g.kubeCmd.CopyToPod(pod, container, binSource, g.binDestination()).Run(ctx)
	return errCopyToPod
}

func (g Grapple) portForwardDelve(l *logrus.Entry, ctx context.Context, pod, host string, port int) {
	l.Info("port-forwarding pod for delve server")
	cmd := g.kubeCmd.PortForwardPod(pod, host, port).Logger(l)
	go func() {
		_, err := cmd.Run(ctx)
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
