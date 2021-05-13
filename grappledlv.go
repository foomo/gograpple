package gograpple

import (
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/go-delve/delve/service/api"
	"github.com/go-delve/delve/service/rpc2"
	"github.com/sirupsen/logrus"
)

func (g Grapple) Delve(pod, container, input string, args []string, host string, port int, delveContinue, vscode bool) error {
	// input validation
	if err := g.validatePod(&pod); err != nil {
		return err
	}
	if err := g.validateContainer(&container); err != nil {
		return err
	}
	goModDir, err := findGoProjectRoot(input)
	if err != nil {
		return fmt.Errorf("couldnt find go.mod dir for input %q", input)
	}

	// are patched?
	if !g.isPatched() {
		return fmt.Errorf("deployment not patched, stopping delve")
	}

	binDest := "/" + g.deployment.Name
	chanExitRun, lockingCleanup := g.dlvGetLockingCleanup(pod, container)
	cmd, errCmd := g.dlvGetCommand(container, binDest, port, delveContinue, args)
	if errCmd != nil {
		return errCmd
	}

	chanExit, chanReload, chanErr, errListenForInterrupts := listenForInterrupts(g.l.WithField("task", "interrupt-listener"))
	if errListenForInterrupts != nil {
		lockingCleanup("failed to listen interrupt signals: " + errListenForInterrupts.Error())
		return errListenForInterrupts
	}

	chanRunErr := make(chan error)
	iteration := 0
	run := func(chanExitRun chan struct{}) {
		iteration++
		g.l.Infof("executing delve command on pod %v", pod)
		_, errRun := g.kubeCmd.ExecPod(pod, container, cmd).PostStart(
			g.dlvWatch(host, pod, port, iteration, lockingCleanup, vscode, goModDir, chanExitRun),
		).Run()
		if errRun != nil && errRun.Error() != "signal: interrupt" {
			chanRunErr <- errRun
		}
	}
	g.l.Info("running initial cleanup, just in case ...")
	g.dlvCleanup(g.l, pod, container)
	go func() {
		chanReload <- "initial load"
	}()
	for {
		select {
		case errRun := <-chanRunErr:
			lockingCleanup(errRun.Error())
			return errRun
		case <-chanExitRun:

		case <-chanExit:
			lockingCleanup("termination")
			return nil
		case <-chanReload:
			var binTemp string
			var errRebuild error
			wgReload := sync.WaitGroup{}
			wgReload.Add(2)
			go func() {
				lockingCleanup("reload it baby")
				wgReload.Done()
			}()
			go func() {
				binTemp, errRebuild = g.rebuildAndUpload(goModDir, pod, container, input, binDest)
				wgReload.Done()
			}()
			wgReload.Wait()
			if errRebuild != nil {
				return errRebuild
			}
			errMove := g.dlvMoveBinary(pod, container, binTemp, binDest)
			if errMove != nil {
				return errMove
			}
			go func() {
				errExpose := g.dlvExpose(pod, host, port)
				if errExpose != nil {
					log.Println(errExpose)
				}
			}()

			go run(chanExitRun)
		case err := <-chanErr:
			lockingCleanup("an error occurred while listening for keyboard commands:" + err.Error())
			return err
		}
	}
}

func (g Grapple) dlvGetCommand(container, binDest string, port int, delveContinue bool, args []string) (cmd []string, err error) {
	cmd = []string{
		"dlv", "exec", binDest,
		"--api-version=2", "--headless",
		fmt.Sprintf("--listen=:%v", port),
		"--accept-multiclient",
	}
	if delveContinue {
		cmd = append(cmd, "--continue")
	}
	if len(args) == 0 {
		args, err = g.getArgsFromConfigMap(g.deployment.Name, container)
		if err != nil {
			return nil, err
		}
	}
	if len(args) > 0 {
		cmd = append(cmd, "--")
		cmd = append(cmd, args...)
	}
	return cmd, nil
}

func (g Grapple) dlvMoveBinary(pod, container, binTemp, binDest string) error {
	cmd := []string{"mv", binTemp, binDest}
	_, errRun := g.kubeCmd.ExecPod(pod, container, cmd).Run()
	return errRun

}

func (g Grapple) dlvWatch(host string, pod string, port, iteration int, cleanup func(reason string) error, vscode bool, goModDir string, chanExitRun chanCommand) func() error {
	return func() error {
		client, errTryDelveServer := dlvTryServer(g.l, host, port, 50, 200*time.Millisecond)
		if errTryDelveServer != nil {
			return errors.New("could not get dlv client: " + errTryDelveServer.Error())
		}
		go func() {
			i := 0
			for {
				select {
				case <-chanExitRun:
					return
				case <-time.After(1000 * time.Millisecond):
					_, state, errState := dlvCheckServer(g.l.WithField("task", "watch-dlv"), host, port, 3*time.Second, client)
					if errState != nil {
						g.l.WithError(errState).Error("dlv seems to be down on", host, ":", port)
						cleanup("dlv is down")
						os.Exit(1)
					}
					if i%20 == 0 {
						g.l.WithField("pid", client.ProcessPid()).Info("dlv is up")
					}
					if state.Exited {
						cleanup("dlv state.Exited == true")
						os.Exit(1)
					} else if !state.Running {
						// there still is the case, when you are in a breakpoint on a zombie process
						// dlv will not handle that gracefully
						g.l.WithField("pid", client.ProcessPid()).Info("dlv is up - process is not running - is it a zombie, or has it been halted by a breakpoint ?")
					}
					i++
				}
			}
		}()
		if vscode {
			if err := launchVscode(g.l, goModDir, pod, host, port, 5, iteration, 1*time.Second); err != nil {
				return err
			}
		}
		return nil
	}
}

func (g Grapple) dlvGetLockingCleanup(pod, container string) (chanExitRun chanCommand, cleanup func(reason string) error) {
	cleanupLock := sync.Mutex{}
	cleanupStarted := false
	chanExitRun = make(chanCommand)
	return chanExitRun, func(reason string) error {
		cleanupLock.Lock()
		defer cleanupLock.Unlock()
		cl := g.l.WithField("reason", reason)
		if cleanupStarted {
			cl.Warning("aborting cleanup already started")
			return nil
		}
		cleanupStarted = true
		defer func() {
			cleanupStarted = false
		}()
		cl.Info("cleaning up")
		go func() { chanExitRun <- struct{}{} }()
		cl.Info("informed running tasks")
		errDelveCleanup := g.dlvCleanup(cl, pod, container)
		if errDelveCleanup != nil {
			cl.WithError(errDelveCleanup).Error("could not clean up")
			return errDelveCleanup
		}
		return nil
	}
}

func (g Grapple) dlvExpose(pod, host string, port int) error {
	g.l.Infof("exposing deployment %v for delve", g.deployment.Name)
	_, err := g.kubeCmd.ForwardPod(pod, host, port).Run()
	if err != nil {
		return err
	}
	return nil
}

func (g Grapple) _dlvExpose(pod, host string, port int) error {
	g.l.Infof("exposing deployment %v for delve", g.deployment.Name)
	_, err := g.kubeCmd.ExposePod(pod, host, port).Run()
	if err != nil {
		return err
	}
	return nil
}

func (g Grapple) dlvCleanup(l *logrus.Entry, pod, container string) error {
	l.Infof("removing delve service")
	outDeleteService, errDeleteService := g.kubeCmd.DeleteService(pod).Run()
	if errDeleteService != nil {
		l.WithError(errDeleteService).Warn("could not delete exposing service: " + outDeleteService)
	}
	l.Info("cleaning up debug processes")
	const nameDLV = "dlv"
	nameProgram := g.deployment.Name
	pidsOfProgram, errGetPIDsOfProgram := g.getPIDsOf(pod, container, nameProgram)
	if errGetPIDsOfProgram != nil {
		return errGetPIDsOfProgram
	}
	pidsOfDLV, errGetPIDsOfDelve := g.getPIDsOf(pod, container, nameDLV)
	if errGetPIDsOfDelve != nil {
		return errGetPIDsOfDelve
	}
	kill := func(name string, pids []string, murder bool) (leftToKill []string) {
		leftToKill = []string{}
		for _, pid := range pids {
			remainingPIDs, errGetPIDs := g.getPIDsOf(pod, container, name)
			killed := true
			if errGetPIDs == nil {
				for _, remainingPID := range remainingPIDs {
					if remainingPID == pid {
						killed = false
						break
					}
				}
			}
			if killed {
				continue
			}
			cmd := []string{"kill"}
			if murder {
				cmd = append(cmd, "-s", "9")
			}
			cmd = append(cmd, pid)
			outKill, errKill := g.kubeCmd.ExecPod(pod, container, cmd).Run()
			if errKill != nil {
				l.WithError(errKill).Warn("could not kill process", outKill)
			}
			leftToKill = append(leftToKill, pid)

		}
		return leftToKill
	}
	const maxAttempts = 10
	for i := 0; i < maxAttempts; i++ {
		pidsOfProgram = kill(nameProgram, pidsOfProgram, i > 0)
		pidsOfDLV = kill(nameDLV, pidsOfDLV, i > 0)
		if len(pidsOfDLV) == 0 && len(pidsOfProgram) == 0 {
			return nil
		}
		time.Sleep(time.Millisecond * 200)
	}
	return fmt.Errorf("could not kill processes after max attempts %v", maxAttempts)
}

func dlvTryServer(l *logrus.Entry, host string, port, tries int, sleep time.Duration) (client *rpc2.RPCClient, err error) {
	errTryCall := tryCall(l, tries, sleep, func(i int) error {
		l.Infof("checking delve connection on %v:%v (%v/%v)", host, port, i, tries)
		newClient, _, errCheck := dlvCheckServer(l, host, port, 200*time.Millisecond, client)
		client = newClient
		return errCheck
	})
	if errTryCall != nil {
		return nil, errTryCall
	}
	l.Infof("delve server listening on %v:%v", host, port)
	return client, nil
}

func dlvCheckServer(
	l *logrus.Entry, host string, port int, timeout time.Duration,
	client *rpc2.RPCClient,
) (
	*rpc2.RPCClient, *api.DebuggerState, error,
) {
	if client == nil {
		// connection timeouts suck with k8s, because the port is open and you can connect, but ...
		// early on the connection is a dead and the timeout does not kick in, despite the fact
		// that the connection is not "really" establised
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("%v:%v", host, port), time.Hour)
		if err != nil {
			return nil, nil, err
		}
		chanClient := make(chan struct{})
		go func() {
			// the next level of suck is the rpc client
			// it does not return an error for its constructor,
			// even though it actually does make a call and that fails
			// and the cient will not do any other calls,
			// because it has shutdown internally, without telling us
			// at least that is what I understand here
			client = rpc2.NewClientFromConn(conn)
			chanClient <- struct{}{}
		}()
		select {
		case <-time.After(timeout):
			// this is the actual timeout, because the connetion timeout does not work,
			// because k8s ...
			conn.Close()
			// l.Warn("dlv server check timeout", timeout)
			return nil, nil, errors.New("stale connection to dlv, aborting after timeout")
		case <-chanClient:
			// we are good to go
			// l.Info("hey, we got a client")
		}
	}
	st, errState := client.GetState()
	if errState != nil {
		// l.Info("could not get state from server using client", errState)
		return nil, nil, errState
	}
	return client, st, nil
}

type chanCommand chan struct{}

func listenForInterrupts(l *logrus.Entry) (chanExit chanCommand, chanReload chan string, chanErr chan error, err error) {
	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan)
	chanExit = make(chanCommand)
	chanReload = make(chan string)
	chanErr = make(chan error)
	i := 0
	exiting := false
	readyToReset := false
	durReset := 2 * time.Second
	go func() {
		for {
			select {
			case <-time.After(durReset):
				if readyToReset {
					l.Info("resetting termination timer")
					readyToReset = false
				}
				i = 0
			case sig := <-sigchan:
				switch sig {
				case os.Interrupt:
					l.Info("received interrupt signal, tigger one more interrupt within ", durReset, " to terminate")
					readyToReset = true
					if exiting {
						l.Warn("already exiting - ignoring interupt")
						continue
					}
					if i == 0 {
						l.Info("triggering reload")
						chanReload <- "interrupt => reload"
					} else {
						l.Info("triggering exit")
						exiting = true
						chanExit <- struct{}{}
					}
					i++
				}
			}
		}
	}()
	return chanExit, chanReload, chanErr, nil
}

func (g Grapple) rebuildAndUpload(goModDir, pod, container, input, binDest string) (tempDest string, err error) {
	binPath := path.Join(os.TempDir(), g.deployment.Name)
	g.l.Infof("building %q for debug", input)

	var relInputs []string
	inputInfo, errInputInfo := os.Stat(input)
	if errInputInfo != nil {
		return "", err
	}
	if inputInfo.IsDir() {
		if files, err := os.ReadDir(input); err != nil {
			return "", err
		} else {
			for _, file := range files {
				if path.Ext(file.Name()) == ".go" {
					relInputs = append(relInputs, strings.TrimPrefix(path.Join(input, file.Name()), goModDir+string(filepath.Separator)))
				}
			}
		}
	} else {
		relInputs = append(relInputs, strings.TrimPrefix(input, goModDir+string(filepath.Separator)))
	}

	_, errBuild := g.goCmd.Build(goModDir, binPath, relInputs, `-gcflags="all=-N -l"`).Env("GOOS=linux").Run()
	if errBuild != nil {
		return "", errBuild
	}

	g.l.Infof("copying binary to pod %v", pod)
	tempDest = binDest + "-build"
	_, errCopyToPod := g.kubeCmd.CopyToPod(pod, container, binPath, tempDest).Run()
	return tempDest, errCopyToPod
}
