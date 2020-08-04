package gograpple

import (
	"errors"
	"fmt"
	"net"
	"os"
	"os/signal"
	"path"
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
		g.kubeCmd.ExecPod(pod, container, cmd).PostStart(
			g.dlvWatch(host, pod, port, iteration, lockingCleanup, vscode, goModDir, chanExitRun),
		).Run()
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
			errExpose := g.dlvExpose(pod, host, port)
			if errExpose != nil {
				return errExpose
			}
			go run(chanExitRun)
		case err := <-chanErr:
			lockingCleanup("an error occurred while listeing for keyboard commands:" + err.Error())
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
		client, errTryDelveServer := dlvTryServer(g.l, host, port, 5, 1*time.Second)
		if errTryDelveServer != nil {
			return errTryDelveServer
		}
		go func() {
			i := 0
			for {
				select {
				case <-chanExitRun:
					return
				case <-time.After(time.Millisecond * 500):
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
						g.l.WithField("pid", client.ProcessPid()).Info("dlv is up - process is not running - is it a zombie ?!")
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
	errTryCall := tryCall(tries, sleep, func(i int) error {
		l.Infof("checking delve connection on %v:%v (%v/%v)", host, port, i, tries)
		newClient, _, errCheck := dlvCheckServer(l, host, port, 1*time.Second, nil)
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
	var conn net.Conn

	if client == nil {
		// get a tcp connection for the rpc dlv rpc client
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("%v:%v", host, port), timeout)
		if err != nil {
			return nil, nil, err
		}
		timer := time.AfterFunc(timeout, func() {
			errClose := conn.Close()
			if errClose != nil {
				l.WithError(errClose).Error("could not close stale connection")
			}
			l.Warn("stale connection timeout")
		})
		// rpc2.NewClientFromConn(conn) will implicitly call setAPIVersion on the dlv server
		// while there might be a connection to a socket, that was created with
		// kubctl expose it still might happen, that dlv will not answer
		// for this reason we are creating this hack
		client = rpc2.NewClientFromConn(conn)
		if !timer.Stop() {
			// we ripped out the underlying connection
			return nil, nil, errors.New("stale connection to dlv, aborting after timeout")
		}
	}
	st, errState := client.GetStateNonBlocking()
	if errState == nil && conn != nil {
		conn.SetDeadline(time.Now().Add(time.Second * 3600))
	}
	return client, st, errState
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
	durReset := time.Millisecond * 5000
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
	_, errBuild := g.goCmd.Build(goModDir, binPath, input, `-gcflags="all=-N -l"`).Env("GOOS=linux").Run()
	if errBuild != nil {
		return "", errBuild
	}

	g.l.Infof("copying binary to pod %v", pod)
	tempDest = binDest + "-build"
	_, errCopyToPod := g.kubeCmd.CopyToPod(pod, container, binPath, tempDest).Run()
	return tempDest, errCopyToPod
}
