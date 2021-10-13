package gograpple

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"text/template"
	"time"

	"github.com/foomo/gograpple/exec"
	"github.com/sirupsen/logrus"
)

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

func Open(l *logrus.Entry, ctx context.Context, path string) (string, error) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "linux":
		cmd = exec.NewCommand(l, "xdg-open").Args(path)
	case "windows":
		cmd = exec.NewCommand(l, "rundll32").Args("url.dll,FileProtocolHandler", path)
	case "darwin":
		cmd = exec.NewCommand(l, "open").Args(path)
	default:
		return "", fmt.Errorf("unsupported platform")
	}
	return cmd.RunCtx(ctx)
}

func TryCall(tries int, waitBetweenAttempts time.Duration, f func(i int) error) error {
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

func tryCallWithContext(ctx context.Context, tries int, waitBetweenAttempts time.Duration, f func(i int) error) error {
	var err error
	for i := 1; i < tries+1; i++ {
		select {
		case <-ctx.Done():
			return context.Canceled
		default:
			err = f(i)
			if err == nil {
				return nil
			}
			time.Sleep(waitBetweenAttempts)
		}
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

func stringIsInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}
