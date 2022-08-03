package delve

import (
	"context"
	"fmt"
	"os"

	"github.com/foomo/gograpple/exec"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type KubeDelveServer struct {
	host    string
	port    int
	kubeCmd *exec.KubectlCmd
	process *os.Process
}

func (kds KubeDelveServer) Host() string {
	return kds.host
}

func (kds KubeDelveServer) Port() int {
	return kds.port
}

func NewKubeDelveServer(l *logrus.Entry, namespace, host string, port int) *KubeDelveServer {
	kubectl := exec.NewKubectlCommand()
	kubectl.Logger(l).Args("-n", namespace)
	return &KubeDelveServer{host, port, kubectl, nil}
}

func (kds *KubeDelveServer) StartNoWait(ctx context.Context, pod, container string,
	binDest string, binArgs []string, doContinue bool) {
	cmd := kds.kubeCmd.ExecPod(pod, container, kds.getRunCmd(binDest, binArgs, doContinue))
	cmd.PostStart(
		func(p *os.Process) error {
			kds.process = p
			return nil
		}).NoWait().Run(ctx)
	<-cmd.Started()
}

func (kds *KubeDelveServer) Start(ctx context.Context, pod, container string,
	binDest string, binArgs []string, doContinue bool) error {
	cmd := kds.kubeCmd.ExecPod(pod, container, kds.getRunCmd(binDest, binArgs, doContinue))
	// execute command to run dlv on container
	out, err := cmd.PostStart(
		func(p *os.Process) error {
			kds.process = p
			return nil
		}).Run(ctx)
	return errors.WithMessage(err, out)
}

// doContinue will start the execution without waiting for a client connection
func (kds KubeDelveServer) getRunCmd(binDest string, binArgs []string, doContinue bool) []string {
	cmd := []string{
		"dlv", "exec", binDest, "--headless", "--accept-multiclient",
		fmt.Sprintf("--listen=:%v", kds.port),
	}
	if doContinue {
		cmd = append(cmd, "--continue")
	}
	// cmd = append(cmd, "--log")
	if len(binArgs) > 0 {
		cmd = append(cmd, "--")
		cmd = append(cmd, binArgs...)
	}
	return cmd
}

func (kds *KubeDelveServer) Stop() error {
	if kds.process == nil {
		return fmt.Errorf("no process found, run Start first")
	}
	if err := kds.process.Release(); err != nil {
		return err
	}
	return kds.process.Kill()
}
