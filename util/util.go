package util

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"time"

	"github.com/foomo/gograpple/internal/exec"
	"github.com/sirupsen/logrus"
)

func RunWithInterrupt(l *logrus.Entry, callback func(ctx context.Context)) {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	durReload := 3 * time.Second
	for {
		ctx, cancelCtx := context.WithCancel(context.Background())
		// do stuff
		go callback(ctx)
		select {
		case <-signalChan: // first signal
			l.Info("-")
			l.Infof("interrupt received, trigger one more within %v to terminate", durReload)
			cancelCtx()
			select {
			case <-time.After(durReload): // reloads durReload after first signal
				l.Info("-")
				l.Info("reloading")
			case <-signalChan: // second signal, hard exit
				l.Info("-")
				l.Info("terminating")
				signal.Stop(signalChan)
				// exit loop
				return
			}
		}
	}
}

func Open(l *logrus.Entry, ctx context.Context, path string) (string, error) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "linux":
		cmd = exec.NewCommand("xdg-open").Logger(l).Args(path)
	case "windows":
		cmd = exec.NewCommand("rundll32").Logger(l).Args("url.dll,FileProtocolHandler", path)
	case "darwin":
		cmd = exec.NewCommand("open").Logger(l).Args(path)
	default:
		return "", fmt.Errorf("unsupported platform")
	}
	return cmd.Run(ctx)
}

func GetPlatformInfo(platform string) (os, arch string, err error) {
	pieces := strings.Split(platform, "/")
	if len(pieces) != 2 {
		return os, arch, fmt.Errorf("invalid format for platform %q", platform)
	}
	return pieces[0], pieces[1], nil
}

func ParseImage(s string) (repo, name, tag string, err error) {
	pieces := strings.Split(s, "/")
	switch true {
	case len(pieces) == 1 && pieces[0] == s:
		imageTag := strings.Split(s, ":")
		return "", imageTag[0], imageTag[1], nil
	case len(pieces) > 1:
		imageTag := strings.Split(pieces[len(pieces)-1], ":")
		return strings.Join(pieces[:len(pieces)-1], "/"), imageTag[0], imageTag[1], nil
	}
	return "", "", "", fmt.Errorf("invalid image value %q provided", s)
}
