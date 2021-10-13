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
	kubectl := exec.NewKubectlCommand(l)
	kubectl.Args("-n", namespace)
	return &KubeDelveServer{host, port, kubectl, nil}
}

func (kds *KubeDelveServer) StartNoWait(ctx context.Context, pod, container string, binDest string,
	useContinue bool, binArgs []string) {
	cmd := kds.kubeCmd.ExecPod(pod, container, kds.buildCommand(binDest, useContinue, binArgs))
	cmd.PostStart(
		func(p *os.Process) error {
			kds.process = p
			return nil
		}).NoWait().RunCtx(ctx)
	<-cmd.Started()
}

func (kds *KubeDelveServer) Start(ctx context.Context, pod, container string, binDest string,
	useContinue bool, binArgs []string) error {
	cmd := kds.kubeCmd.ExecPod(pod, container, kds.buildCommand(binDest, useContinue, binArgs))
	// execute command to run dlv on container
	out, err := cmd.PostStart(
		func(p *os.Process) error {
			kds.process = p
			return nil
		}).RunCtx(ctx)
	return errors.WithMessage(err, out)
}

func (kds KubeDelveServer) buildCommand(binDest string, useContinue bool, binArgs []string) []string {
	cmd := []string{
		"dlv", "exec", binDest, "--api-version=2", "--headless",
		fmt.Sprintf("--listen=:%v", kds.port), "--accept-multiclient",
	}
	if useContinue {
		cmd = append(cmd, "--continue")
	}
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
