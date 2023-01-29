package gograpple

import (
	"fmt"
	"os"
	"runtime"

	"github.com/bitfield/script"
	"github.com/foomo/gograpple/kubectl"
	"github.com/foomo/gograpple/log"
	"github.com/pkg/errors"
)

func (g Grapple) Attach(namespace, deployment, container, bin, host string, port int) error {
	pod, err := kubectl.GetMostRecentRunningPodBySelectors(g.deployment.Spec.Selector.MatchLabels)
	if err != nil {
		return err
	}
	cleanup(pod, container)
	// check if delve is available
	dlvDest := "/dlv"
	_, err = kubectl.ExecPod(pod, container, []string{"which", "dlv"}).String()
	if err != nil {
		if err := copyDelve(pod, container, dlvDest); err != nil {
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
	go attachDelveOnPod(pod, container, dlvDest, pids[0], host, port, g.debug)
	// defer g.cleanup(pod, container)
	// launchVSCode(context.Background(), g.l, "./test/app", "", port, 3)
	return kubectl.PortForwardPod(pod, port)
}

func attachCmd(dlvPath, binPid, host string, port int, debug bool) []string {
	cmd := []string{dlvPath, "--headless", "attach", binPid,
		"--continue", "--api-version=2", "--accept-multiclient",
		fmt.Sprintf("--listen=%v:%v", host, port)}
	if debug {
		cmd = append(cmd, "--log", "--log-output=rpc,dap,debugger")
	}
	return cmd
}

func dapCmd(dlvPath string, port int, debug bool) []string {
	cmd := []string{dlvPath, "dap", "--listen",
		fmt.Sprintf("127.0.0.1:%v", port)}
	if debug {
		cmd = append(cmd, "--log", "--log-output=rpc,dap,debugger")
	}
	return cmd
}

func attachDelveOnPod(pod, container, dlvPath, binPid, host string, port int, debug bool) error {
	_, err := kubectl.ExecPod(pod, container, attachCmd(dlvPath, binPid, host, port, debug)).WithStdout(log.Writer("dlv")).Stdout()
	return err
}

func cleanup(pod, container string) {
	pss := []string{"dlv"}
	for _, ps := range pss {
		kubectl.ExecPod(pod, container, []string{"pkill", ps}).WithStdout(log.Writer("cleanup")).Stdout()
		// script.Exec(fmt.Sprintf("pkill %v", ps)).Stdout()
	}
}

func copyDelve(pod, container, dlvDest string) error {
	// build dlv for given arch
	dlvSrc := fmt.Sprintf("%v/go/bin/linux_amd64/dlv", os.Getenv("HOME"))
	if runtime.GOOS == "linux" && runtime.GOARCH == "amd64" {
		// if its the same os and arch use a different location
		os.Setenv("GOBIN", "/tmp/")
		dlvSrc = "/tmp/dlv"
	}
	os.Setenv("CGO_ENABLED", "0")
	os.Setenv("GOOS", "linux")
	os.Setenv("GOARCH", "amd64")
	if out, err := script.Exec(
		`go install -ldflags "-s -w -extldflags '-static'" github.com/go-delve/delve/cmd/dlv@latest`).String(); err != nil {
		return errors.WithMessage(err, out)
	}
	// copy dlv to pod
	return kubectl.CopyToPod(pod, container, dlvSrc, dlvDest)
}
