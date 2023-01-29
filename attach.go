package gograpple

import (
	"context"
	"fmt"
	"os"

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
	dlvDest := "/dlv"
	_, err = kubectl.ExecPod(pod, container, []string{"which", "dlv"}).String()
	if err != nil {
		// // if not install it
		// out, err := kubectl.ExecPod(pod, container, []string{"go", "get", "-u", "github.com/go-delve/delve/cmd/dlv"}).String()
		// if err != nil {
		// 	return errors.WithMessage(err, out)
		// }
		// build dlv for given arch
		// os.Setenv("GOBIN", "/tmp/")
		os.Setenv("CGO_ENABLED", "0")
		os.Setenv("GOOS", "linux")
		os.Setenv("GOARCH", "amd64")
		if out, err := script.Exec(
			`go install -ldflags "-s -w -extldflags '-static'" github.com/go-delve/delve/cmd/dlv@latest`).String(); err != nil {
			return errors.WithMessage(err, out)
		}

		dlvSrc := fmt.Sprintf("%v/go/bin/linux_amd64/dlv", os.Getenv("HOME"))
		// copy dlv to pod
		if err := kubectl.CopyToPod(pod, container, dlvSrc, dlvDest); err != nil {
			return err
		}
	}

	// find pid of bin by name
	pids, err := kubectl.GetPIDsOf(pod, container, bin)
	if err != nil {
		return err
	}
	if len(pids) != 1 {
		return fmt.Errorf("found none or more than one process named %q", bin)
	}
	g.portForwardDelve(g.l, context.Background(), pod, "", port)
	attachDelveOnPod_(pod, container, dlvDest, pids[0], port)
	// defer g.cleanup(pod, container)
	// launchVSCode(context.Background(), g.l, "./test/app", "", port, 3)
	// port forward to server
	// return kubectl.PortForwardPod(pod, port)
	return nil
}

func (g Grapple) attachDelveOnPod(pod, container, dlvPath, binPid string, port int) {
	g.l.Info("attaching delve server")
	cmd := g.kubeCmd.ExecPod(pod, container, []string{dlvPath, "attach", binPid,
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
func attachDelveOnPod_(pod, container, dlvPath, binPid string, port int) error {
	_, err := kubectl.ExecPod(pod, container, []string{dlvPath, "--headless", "attach", binPid,
		"--continue", "--api-version=2", "--accept-multiclient", "--log", fmt.Sprintf("--listen=:%v", port)}).Stdout()
	return err
}

func (g Grapple) cleanup(pod, container string) {
	pss := []string{"dlv", "kubectl"}
	for _, ps := range pss {
		kubectl.ExecPod(pod, container, []string{"pkill", ps}).Stdout()
		script.Exec(fmt.Sprintf("pkill %v", ps)).Stdout()
	}
}
