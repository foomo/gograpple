package grapple

import (
	"fmt"
	"os"
	"os/signal"
	"runtime"

	"github.com/bitfield/script"
	"github.com/foomo/gograpple/internal/kubectl"
	"github.com/foomo/gograpple/internal/log"
	"github.com/pkg/errors"
)

func (g Grapple) Attach(namespace, deployment, container, bin, arch, host string, port int, debug bool) error {
	pod, err := kubectl.GetMostRecentRunningPodBySelectors(namespace, g.deployment.Spec.Selector.MatchLabels)
	if err != nil {
		return err
	}
	go handleExit(namespace, pod, container)
	// check if delve is available
	dlvDest := "dlv"
	_, err = kubectl.ExecPod(namespace, pod, container, []string{"which", "dlv"}).String()
	if err != nil {
		if err := copyDelve(namespace, pod, container, arch, dlvDest); err != nil {
			return err
		}
		dlvDest = "/dlv"
	}
	// find pid of bin by name
	pids, err := kubectl.GetPIDsOf(namespace, pod, container, bin)
	if err != nil {
		return err
	}
	if len(pids) != 1 {
		return fmt.Errorf("found none or more than one process named %q", bin)
	}
	go attachDelveOnPod(namespace, pod, container, dlvDest, pids[0], host, port, debug)
	// launchVSCode(context.Background(), g.l, "./test/app", "", port, 3)
	return kubectl.PortForwardPod(namespace, pod, port)
}

func attachCmd(dlvPath, binPid, host string, port int, debug bool) []string {
	cmd := []string{dlvPath, "--headless", "attach", binPid, "--api-version=2",
		"--continue", "--accept-multiclient", fmt.Sprintf("--listen=%v:%v", host, port)}
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

func attachDelveOnPod(namespace, pod, container, dlvPath, binPid, host string, port int, debug bool) error {
	_, err := kubectl.ExecPod(namespace, pod, container, attachCmd(dlvPath, binPid, host, port, debug)).WithStdout(log.Writer("dlv")).Stdout()
	return err
}

func cleanup(namespace, pod, container string) {
	pss := []string{"dlv"}
	for _, ps := range pss {
		kubectl.ExecPod(namespace, pod, container, []string{"pkill", ps}).WithStdout(log.Writer("cleanup")).Stdout()
		// script.Exec(fmt.Sprintf("pkill %v", ps)).Stdout()
	}
}

func copyDelve(namespace, pod, container, arch, dlvDest string) error {
	// build dlv for given arch
	dlvSrc := fmt.Sprintf("%v/go/bin/linux_%v/dlv", os.Getenv("HOME"), arch)
	if runtime.GOOS == "linux" && arch == runtime.GOARCH {
		// if its the same os and arch use a different location
		os.Setenv("GOBIN", "/tmp/")
		dlvSrc = "/tmp/dlv"
	}
	os.Setenv("CGO_ENABLED", "0")
	os.Setenv("GOOS", "linux")
	os.Setenv("GOARCH", arch)
	if out, err := script.Exec(
		`go install -ldflags "-s -w -extldflags '-static'" github.com/go-delve/delve/cmd/dlv@latest`).String(); err != nil {
		return errors.WithMessage(err, out)
	}
	// copy dlv to pod
	return kubectl.CopyToPod(namespace, pod, container, dlvSrc, dlvDest)
}

func handleExit(namespace, pod, container string) {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	<-signalChan
	log.Entry("cleanup").Info("exiting")
	cleanup(namespace, pod, container)
	os.Exit(0)
}
