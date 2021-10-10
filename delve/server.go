package delve

import (
	"fmt"
	"os"

	"github.com/foomo/gograpple/exec"
	"github.com/sirupsen/logrus"
)

type KubeDelveServer struct {
	l       *logrus.Entry
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

func NewKubeDelveServer(l *logrus.Entry, host string, port int) *KubeDelveServer {
	return &KubeDelveServer{l, host, port, exec.NewKubectlCommand(l), nil}
}

func (kds *KubeDelveServer) Start(pod, container string, binDest string, useContinue bool, binArgs []string) error {
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

	// execute command to run dlv on container
	// this will block until is killed or fails
	_, err := kds.kubeCmd.ExecPod(pod, container, cmd).PostStart(
		func(p *os.Process) error {
			kds.process = p
			// after starting
			// port-forward from localhost to the pod
			kds.l.Infof("port-forwarding %v pod for delve server", pod)
			_, pfErr := kds.kubeCmd.PortForwardPod(pod, kds.host, kds.port).Run()
			return pfErr
		},
	).Run()
	return err
}

func (kds *KubeDelveServer) Stop() error {
	if kds.process == nil {
		return fmt.Errorf("no process found, run start successfully first")
	}
	if err := kds.process.Release(); err != nil {
		return err
	}
	return kds.process.Kill()
}
