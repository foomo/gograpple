package gograpple

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/bitfield/script"
	"github.com/foomo/gograpple/kubectl"
	"github.com/pkg/errors"
)

func (g Grapple) Attach(namespace, deployment, container, bin string, port int) error {
	pod, err := kubectl.GetMostRecentRunningPodBySelectors(g.deployment.Spec.Selector.MatchLabels)
	if err != nil {
		return err
	}
	g.cleanup(pod, container)
	// check if delve is available
	dlvBin := "/dlv"
	// _, err = kubectl.ExecPod(pod, container, []string{"which", "dlv"}).String()
	// if err != nil {
	// 	// // if not install it
	// 	// out, err := kubectl.ExecPod(pod, container, []string{"go", "get", "-u", "github.com/go-delve/delve/cmd/dlv"}).String()
	// 	// if err != nil {
	// 	// 	return errors.WithMessage(err, out)
	// 	// }
	// 	// build dlv for given arch
	// 	os.Setenv("GOBIN", "/tmp/")
	// 	os.Setenv("CGO_ENABLED", "0")
	// 	os.Setenv("GOOS", "linux")
	// 	os.Setenv("GOARCH", "amd64")
	// 	if out, err := script.Exec(
	// 		`go install -ldflags "-s -w -extldflags '-static'" github.com/go-delve/delve/cmd/dlv@latest`).String(); err != nil {
	// 		return errors.WithMessage(err, out)
	// 	}

	// 	// copy dlv to pod
	// 	if err := kubectl.CopyToPod(pod, container, "/tmp/dlv", "/dlv"); err != nil {
	// 		return err
	// 	}
	// 	dlvBin = "/dlv"
	// }

	// find pid of bin by name
	pids, err := kubectl.GetPIDsOf(pod, container, bin)
	if err != nil {
		return err
	}
	if len(pids) != 1 {
		return fmt.Errorf("found none or more than one process named %q", bin)
	}
	g.portForwardDelve(g.l, context.Background(), pod, "", port)
	attachDelveOnPod_(pod, container, dlvBin, pids[0], port)
	// defer g.cleanup(pod, container)
	// launchVSCode(context.Background(), g.l, "./test/app", "", port, 3)
	// port forward to server
	// return kubectl.PortForwardPod(pod, port)
	return nil
}

func (g Grapple) attachDelveOnPod(pod, container, dlvBin, binPid string, port int) {
	g.l.Info("attaching delve server")
	cmd := g.kubeCmd.ExecPod(pod, container, []string{dlvBin, "attach", binPid,
		"--headless", "--continue", "--api-version=2", "--accept-multiclient", "--log",
		fmt.Sprintf("--listen=:%v", port)})
	cmd.Stdout(os.Stdout)
	go func() {
		_, err := cmd.Run(context.Background())
		if err != nil && err.Error() != "signal: killed" {
			g.l.WithError(err).Errorf("dlv attach for %v pod failed", pod)
		}
	}()
	<-cmd.Started()
}
func attachDelveOnPod_(pod, container, dlvBin, binPid string, port int) error {
	_, err := kubectl.ExecPod(pod, container, []string{dlvBin, "--headless", "attach", binPid,
		"--continue", "--api-version=2", "--accept-multiclient", "--log", fmt.Sprintf("--listen=:%v", port)}).Stdout()
	return err
}

func (g Grapple) cleanup(pod, container string) {
	// cleanup pod dlv process
	remotePids, err := kubectl.GetPIDsOf(pod, container, "dlv")
	if err != nil {
		g.l.Error(errors.WithMessage(err, "remote get dlv pid failed"))
	}
	if len(remotePids) > 0 {
		if errs := kubectl.KillPidsOnPod(pod, container, remotePids, true); len(errs) > 0 {
			for _, err := range errs {
				g.l.Error(errors.WithMessage(err, "remote kill dlv pid failed"))
			}
		}
	}
	// // cleanup local kubectl
	// kubectlPids, err := kubectl.GetPIDsOf(pod, container, "kubectl")
	// if err != nil {
	// 	return
	// }
	// cleanup local
	localDlvPids, err := script.Exec("pidof dlv").Slice()
	if err != nil {
		g.l.Error(errors.WithMessage(err, "local get dlv pid failed"))
	}
	localKubectlPids, err := script.Exec("pidof kubectl").Slice()
	if err != nil {
		g.l.Error(errors.WithMessage(err, "local get kubectl pid failed"))
	}
	localPids := append(localDlvPids, localKubectlPids...)
	if len(localPids) > 0 {
		script.Exec(fmt.Sprintf("kill -s 9 %v", strings.Join(localPids, " "))).Stdout()
	}
}
