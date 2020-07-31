package gograpple

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"net"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/foomo/gograpple/bindata"
	"github.com/foomo/squadron/util"
	"github.com/go-delve/delve/service/api"
	"github.com/go-delve/delve/service/rpc2"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/apps/v1"
)

const (
	devDeploymentPatchFile        = "deployment-patch.yaml"
	defaultWaitTimeout            = "30s"
	conditionContainersReady      = "condition=ContainersReady"
	defaultPatchedLabel           = "dev-mode-patched"
	defaultPatchImage             = "gograpple-patch:latest"
	defaultConfigMapMount         = "/etc/config/mounted"
	defaultConfigMapDeploymentKey = "deployment.json"
)

type Mount struct {
	HostPath  string
	MountPath string
}

type patchValues struct {
	Label          string
	Deployment     string
	Container      string
	ConfigMapMount string
	Mounts         []Mount
	Image          string
}

func newPatchValues(deployment, container string, mounts []Mount) *patchValues {
	return &patchValues{
		Label:          defaultPatchedLabel,
		Deployment:     deployment,
		Container:      container,
		ConfigMapMount: defaultConfigMapMount,
		Mounts:         mounts,
		Image:          defaultPatchImage,
	}
}

type launchArgs struct {
	Name       string `json:"name,omitempty"`
	Request    string `json:"request,omitempty"`
	Type       string `json:"type,omitempty"`
	Mode       string `json:"mode,omitempty"`
	RemotePath string `json:"remotePath,omitempty"`
	Port       int    `json:"port,omitempty"`
	Host       string `json:"host,omitempty"`
	Trace      string `json:"trace,omitempty"`
	LogOutput  string `json:"logOutput,omitempty"`
	ShowLog    bool   `json:"showLog,omitempty"`
}

func newLaunchArgs(pod, host string, port int) *launchArgs {
	return &launchArgs{
		Host:       host,
		Name:       fmt.Sprintf("delve-%v", pod),
		Port:       port,
		Request:    "attach",
		Type:       "go",
		Mode:       "remote",
		RemotePath: "${workspaceFolder}",
		// Trace:      "verbose",
		// LogOutput: "rpc",
		// ShowLog:   true,
	}
}

func (la *launchArgs) toJson() (string, error) {
	bytes, err := json.Marshal(la)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

type Grapple struct {
	l          *logrus.Entry
	deployment v1.Deployment
	kubeCmd    *util.KubeCmd
	dockerCmd  *util.DockerCmd
	goCmd      *util.GoCmd
}

func NewGrapple(l *logrus.Entry, namespace, deployment string) (*Grapple, error) {
	g := &Grapple{l: l}
	g.kubeCmd = util.NewKubeCommand(l)
	g.dockerCmd = util.NewDockerCommand(l)
	g.goCmd = util.NewGoCommand(l)
	g.kubeCmd.Args("-n", namespace)

	if err := g.validateNamespace(namespace); err != nil {
		return nil, err
	}
	if err := g.validateDeployment(namespace, deployment); err != nil {
		return nil, err
	}

	d, err := g.kubeCmd.GetDeployment(deployment)
	if err != nil {
		return nil, err
	}
	g.deployment = *d

	return g, nil
}

func (g *Grapple) Rollback() error {
	if !g.isPatched() {
		return fmt.Errorf("deployment not patched, stopping rollback")
	}
	return g.rollbackUntilUnpatched()
}

func (g Grapple) Patch(image, tag, container string, mounts []Mount) error {
	if g.isPatched() {
		g.l.Warn("deployment already patched, rolling back first")
		if err := g.rollbackUntilUnpatched(); err != nil {
			return err
		}
	}
	if err := g.validateContainer(&container); err != nil {
		return err
	}
	if err := g.validateImage(container, &image, &tag); err != nil {
		return err
	}

	g.l.Infof("creating a ConfigMap with deployment data")
	bs, err := json.Marshal(g.deployment)
	if err != nil {
		return err
	}
	data := map[string]string{defaultConfigMapDeploymentKey: string(bs)}
	_, err = g.kubeCmd.CreateConfigMap(g.deployment.Name, data)
	if err != nil {
		return err
	}

	g.l.Infof("waiting for deployment to get ready")
	_, err = g.kubeCmd.WaitForRollout(g.deployment.Name, defaultWaitTimeout).Run()
	if err != nil {
		return err
	}

	g.l.Infof("extracting patch files")
	const patchFolder = "the-hook"
	if err := bindata.RestoreAssets(os.TempDir(), patchFolder); err != nil {
		return err
	}
	theHookPath := path.Join(os.TempDir(), patchFolder)

	g.l.Infof("building patch image with %v:%v", image, tag)
	_, err = g.dockerCmd.Build(theHookPath, "--build-arg",
		fmt.Sprintf("IMAGE=%v:%v", image, tag), "-t", defaultPatchImage).Run()
	if err != nil {
		return err
	}

	g.l.Infof("rendering deployment patch template")
	patch, err := renderTemplate(
		path.Join(theHookPath, devDeploymentPatchFile),
		newPatchValues(g.deployment.Name, container, mounts),
	)
	if err != nil {
		return err
	}

	g.l.Infof("patching deployment for development")
	_, err = g.kubeCmd.PatchDeployment(patch, g.deployment.Name).Run()
	return err
}

func (g Grapple) Shell(pod string) error {
	if !g.isPatched() {
		return fmt.Errorf("deployment not patched, stopping shell")
	}
	if err := g.validatePod(&pod); err != nil {
		return err
	}
	g.l.Infof("waiting for pod %v with %q", pod, conditionContainersReady)
	_, err := g.kubeCmd.WaitForPodState(pod, conditionContainersReady, defaultWaitTimeout).Run()
	if err != nil {
		return err
	}

	g.l.Infof("running interactive shell for patched deployment %v", g.deployment.Name)
	_, err = g.kubeCmd.ExecShell(fmt.Sprintf("pod/%v", pod), "/").Run()
	return err
}

func (g Grapple) Cleanup(pod, container string) error {
	if !g.isPatched() {
		return fmt.Errorf("deployment not patched, stopping delve")
	}
	if err := g.validatePod(&pod); err != nil {
		return err
	}
	if err := g.validateContainer(&container); err != nil {
		return err
	}
	return g.delveCleanup(g.l, pod, container)
}

func (g Grapple) delveCleanup(l *logrus.Entry, pod, container string) error {
	l.Infof("removing delve service")
	g.kubeCmd.DeleteService(pod).Run()

	g.l.Infof("cleaning up debug processes")
	g.kubeCmd.ExecPod(pod, container, []string{"pkill", "-9", "dlv"}).Run()
	g.kubeCmd.ExecPod(pod, container, []string{"pkill", "-9", g.deployment.Name}).Run()

	return nil
}

func (g Grapple) Delve(pod, container, input string, args []string, host string, port int, delveContinue, vscode bool) error {
	if !g.isPatched() {
		return fmt.Errorf("deployment not patched, stopping delve")
	}
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

	binPath := path.Join(os.TempDir(), g.deployment.Name)
	g.l.Infof("building %q for debug", input)
	g.goCmd.Build(goModDir, binPath, input, `-gcflags="all=-N -l"`).Env("GOOS=linux").Run()
	if err != nil {
		return err
	}

	g.l.Infof("copying binary to pod %v", pod)
	binDest := fmt.Sprintf("/%v", g.deployment.Name)
	_, err = g.kubeCmd.CopyToPod(pod, container, binPath, binDest).Run()
	if err != nil {
		return err
	}

	// one locked point / helper to clean up
	cleanupLock := sync.Mutex{}
	cleanupStarted := false
	cleanup := func(reason string) {
		cleanupLock.Lock()
		defer cleanupLock.Unlock()
		cl := g.l.WithField("reason", reason)
		if cleanupStarted {
			cl.Warning("aborting cleanup already started")
			return
		}
		cleanupStarted = true
		cl.Info("cleaning up")
		errDelveCleanup := g.delveCleanup(cl, pod, container)
		if errDelveCleanup != nil {
			// cl.WithError(errDelveCleanup).Error(out)
		} else {
			// cl.Info(out)
		}
	}
	defer cleanup("termination")

	signalCapture(g.l)

	g.l.Infof("exposing deployment %v for delve", g.deployment.Name)
	_, err = g.kubeCmd.ExposePod(pod, host, port).Run()
	if err != nil {
		return err

	}

	g.l.Infof("executing delve command on pod %v", pod)
	cmd := []string{
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
			return err
		}
	}
	if len(args) > 0 {
		cmd = append(cmd, "--")
		cmd = append(cmd, args...)
	}

	g.kubeCmd.ExecPod(pod, container, cmd).PostStart(
		func() error {
			client, errTryDelveServer := tryDelveServer(g.l, host, port, 5, 1*time.Second)
			if errTryDelveServer != nil {
				return errTryDelveServer
			}
			go func() {
				// TODO add cancelable context
				i := 0
				for {
					time.Sleep(time.Millisecond * 500)
					_, state, errState := checkDelveServer(g.l.WithField("task", "watch-dlv"), host, port, 3*time.Second, client)

					if errState != nil {
						g.l.WithError(errState).Error("dlv seems to be down on", host, ":", port)
						cleanup("dlv is down")
						os.Exit(1)
					} else {
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
					}
					i++
				}
			}()

			if vscode {
				if err := launchVscode(g.l, goModDir, pod, host, port, 5, 1*time.Second); err != nil {
					return err
				}
			}
			return nil
		},
	).Run()
	return nil
}

func (g Grapple) validateNamespace(namespace string) error {
	available, err := g.kubeCmd.GetNamespaces()
	if err != nil {
		return err
	}
	return validateResource("namespace", namespace, "", available)
}

func (g Grapple) validateDeployment(namespace, deployment string) error {
	available, err := g.kubeCmd.GetDeployments()
	if err != nil {
		return err
	}
	return validateResource("deployment", deployment, fmt.Sprintf("for namespace %q", namespace), available)
}

func (g Grapple) validatePod(pod *string) error {
	if *pod == "" {
		var err error
		*pod, err = g.kubeCmd.GetMostRecentPodBySelectors(g.deployment.Spec.Selector.MatchLabels)
		if err != nil || *pod == "" {
			return err
		}
		return nil
	}
	available, err := g.kubeCmd.GetPods(g.deployment.Spec.Selector.MatchLabels)
	if err != nil {
		return err
	}
	return validateResource("pod", *pod, fmt.Sprintf("for deployment %q", g.deployment.Name), available)
}

func (g Grapple) validateContainer(container *string) error {
	if *container == "" {
		*container = g.deployment.Name
	}
	available := g.kubeCmd.GetContainers(g.deployment)
	return validateResource("container", *container, fmt.Sprintf("for deployment %q", g.deployment.Name), available)
}

func (g Grapple) validateImage(container string, image, tag *string) error {
	if *image == "" {
		for _, c := range g.deployment.Spec.Template.Spec.Containers {
			if container == c.Name {
				pieces := strings.Split(c.Image, ":")
				if len(pieces) != 2 {
					return fmt.Errorf("deployment image %q has invalid format", c.Image)
				}
				*image = pieces[0]
				*tag = pieces[1]
				return nil
			}
		}
	}
	return nil
}

func (g Grapple) isPatched() bool {
	_, ok := g.deployment.Spec.Template.ObjectMeta.Labels[defaultPatchedLabel]
	return ok
}

func (g *Grapple) updateDeployment() error {
	d, err := g.kubeCmd.GetDeployment(g.deployment.Name)
	if err != nil {
		return err
	}
	g.deployment = *d
	return nil
}

func (g *Grapple) rollbackUntilUnpatched() error {
	if !g.isPatched() {
		return nil
	}
	if err := g.rollback(); err != nil {
		return err
	}
	if err := g.updateDeployment(); err != nil {
		return err
	}
	return g.rollbackUntilUnpatched()
}

func (g Grapple) rollback() error {
	g.l.Infof("removing ConfigMap %v", g.deployment.Name)
	_, err := g.kubeCmd.DeleteConfigMap(g.deployment.Name)
	if err != nil {
		// may not exist
		g.l.Warn(err)
	}

	g.l.Infof("waiting for deployment to get ready")
	_, err = g.kubeCmd.WaitForRollout(g.deployment.Name, defaultWaitTimeout).Run()
	if err != nil {
		return err
	}

	g.l.Infof("rolling back deployment %v", g.deployment.Name)
	_, err = g.kubeCmd.RollbackDeployment(g.deployment.Name).Run()
	if err != nil {
		return err
	}
	return nil
}

func FindFreePort(host string) (int, error) {
	tcpAddr, err := CheckTCPConnection(host, 0)
	if err != nil {
		return 0, err
	}
	return tcpAddr.Port, nil
}

func CheckTCPConnection(host string, port int) (*net.TCPAddr, error) {
	addr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%v:%v", host, port))
	if err != nil {
		return nil, err
	}
	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return nil, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr), nil
}

func DeploymentIsPatched(l *logrus.Entry, deployment v1.Deployment) bool {
	_, ok := deployment.Spec.Template.ObjectMeta.Labels[defaultPatchedLabel]
	return ok
}

func validateResource(resourceType, resource, suffix string, available []string) error {
	if !stringInSlice(resource, available) {
		return fmt.Errorf("%v %q not found %v, available: %v", resourceType, resource, suffix, strings.Join(available, ", "))
	}
	return nil
}

func ValidatePath(wd string, p *string) error {
	if !filepath.IsAbs(*p) {
		*p = path.Join(wd, *p)
	}
	absPath, err := filepath.Abs(*p)
	if err != nil {
		return err
	}
	_, err = os.Stat(absPath)
	if err != nil {
		return err
	}
	*p = absPath
	return nil
}

func ValidateMounts(wd string, ms []string) ([]Mount, error) {
	var mounts []Mount
	for _, m := range ms {
		pieces := strings.Split(m, ":")
		if len(pieces) != 2 {
			return nil, fmt.Errorf("bad format for mount %q, should be %q separated", m, ":")
		}
		hostPath := pieces[0]
		mountPath := pieces[1]
		if err := ValidatePath(wd, &hostPath); err != nil {
			return nil, fmt.Errorf("bad format for mount %q, host path bad: %s", m, err)
		}
		if !path.IsAbs(mountPath) {
			return nil, fmt.Errorf("bad format for mount %q, mount path should be absolute", m)
		}
		mounts = append(mounts, Mount{hostPath, mountPath})
	}
	return mounts, nil

}

func renderTemplate(path string, values interface{}) (string, error) {
	tpl, err := template.ParseFiles(path)
	if err != nil {
		return "", err
	}
	buf := new(bytes.Buffer)
	err = tpl.Execute(buf, values)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func (g Grapple) getArgsFromConfigMap(configMap, container string) ([]string, error) {
	out, err := g.kubeCmd.GetConfigMapKey(configMap, defaultConfigMapDeploymentKey)
	if err != nil {
		return nil, err
	}
	var d v1.Deployment
	if err := json.Unmarshal([]byte(out), &d); err != nil {
		return nil, err
	}
	for _, c := range d.Spec.Template.Spec.Containers {
		if c.Name == container {
			return c.Args, nil
		}
	}
	return nil, fmt.Errorf("no args found for container %q", container)
}

func signalCapture(l *logrus.Entry) {
	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, os.Interrupt)
	go func() {
		l.Warnf("signal %s recieved", <-sigchan)
	}()
}

func checkDelveServer(
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

func runOpen(l *logrus.Entry, path string) (string, error) {
	var cmd *util.Cmd
	switch runtime.GOOS {
	case "linux":
		cmd = util.NewCommand(l, "xdg-open").Args(path)
	case "windows":
		cmd = util.NewCommand(l, "rundll32").Args("url.dll,FileProtocolHandler", path)
	case "darwin":
		cmd = util.NewCommand(l, "open").Args(path)
	default:
		return "", fmt.Errorf("unsupported platform")
	}
	return cmd.Run()
}

func tryDelveServer(l *logrus.Entry, host string, port, tries int, sleep time.Duration) (client *rpc2.RPCClient, err error) {
	errTryCall := tryCall(tries, sleep, func(i int) error {
		l.Infof("checking delve connection on %v:%v (%v/%v)", host, port, i, tries)
		newClient, _, errCheck := checkDelveServer(l, host, port, 1*time.Second, nil)
		client = newClient
		return errCheck
	})
	if errTryCall != nil {
		return nil, errTryCall
	}
	l.Infof("delve server listening on %v:%v", host, port)
	return client, nil
}

func launchVscode(l *logrus.Entry, goModDir, pod, host string, port, tries int, sleep time.Duration) error {
	util.NewCommand(l, "code").Args(goModDir).PostEnd(func() error {
		return tryCall(tries, time.Millisecond*200, func(i int) error {
			l.Infof("waiting for vscode status (%v/%v)", i, tries)
			_, err := util.NewCommand(l, "code").Args("-s").Run()
			return err
		})
	}).Run()

	l.Infof("opening debug configuration")
	la, err := newLaunchArgs(pod, host, port).toJson()
	if err != nil {
		return err
	}
	_, err = runOpen(l, `vscode://fabiospampinato.vscode-debug-launcher/launch?args=`+la)
	if err != nil {
		return err
	}
	return nil
}

func tryCall(tries int, waitBetweenAttempts time.Duration, f func(i int) error) error {
	var err error
	for i := 1; i < tries+1; i++ {
		err = f(i)
		if err == nil {
			return nil
		}
		time.Sleep(waitBetweenAttempts)
	}
	return err
}

func findGoProjectRoot(path string) (string, error) {
	abs, errAbs := filepath.Abs(path)
	if errAbs != nil {
		return "", errAbs
	}
	dir := filepath.Dir(abs)
	statDir, errStatDir := os.Stat(dir)
	if errStatDir != nil {
		return "", errStatDir
	}
	if !statDir.IsDir() {
		return "", fmt.Errorf("%q is not a dir", dir)
	}
	modFile := filepath.Join(dir, "go.mod")
	stat, errStat := os.Stat(modFile)
	if errStat == nil {
		if stat.IsDir() {
			return "", fmt.Errorf("go.mod is a directory")
		}
		return dir, nil
	}
	if dir == "/" {
		return "", fmt.Errorf("reached / without finding go.mod")
	}
	return findGoProjectRoot(dir)
}
