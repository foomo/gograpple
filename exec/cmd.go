package exec

import (
	"bytes"
	"io"
	"os"
	goexec "os/exec"
	"time"

	"github.com/sirupsen/logrus"
)

type Cmd struct {
	l *logrus.Entry
	// cmd           *exec.Cmd
	command       []string
	cwd           string
	env           []string
	stdin         io.Reader
	stdoutWriters []io.Writer
	stderrWriters []io.Writer
	wait          bool
	t             time.Duration
	preStartFunc  func() error
	postStartFunc func() error
	postEndFunc   func() error
}

func NewCommand(l *logrus.Entry, name string) *Cmd {
	return &Cmd{
		l:       l,
		command: []string{name},
		wait:    true,
		env:     os.Environ(),
	}
}

func (c Cmd) Base() *Cmd {
	c.command = []string{c.command[0]}
	return &c
}

func (c Cmd) Command() []string {
	return c.command
}

func (c *Cmd) Args(args ...string) *Cmd {
	c.command = append(c.command, args...)
	return c
}

func (c *Cmd) Cwd(path string) *Cmd {
	c.cwd = path
	return c
}

func (c *Cmd) Env(env ...string) *Cmd {
	c.env = append(c.env, env...)
	return c
}

func (c *Cmd) Stdin(r io.Reader) *Cmd {
	c.stdin = r
	return c
}

func (c *Cmd) Stdout(w io.Writer) *Cmd {
	if w == nil {
		w, _ = os.Open(os.DevNull)
	}
	c.stdoutWriters = append(c.stdoutWriters, w)
	return c
}

func (c *Cmd) Stderr(w io.Writer) *Cmd {
	if w == nil {
		w, _ = os.Open(os.DevNull)
	}
	c.stderrWriters = append(c.stderrWriters, w)
	return c
}

func (c *Cmd) Timeout(t time.Duration) *Cmd {
	c.t = t
	return c
}

func (c *Cmd) NoWait() *Cmd {
	c.wait = false
	return c
}

func (c *Cmd) PreStart(f func() error) *Cmd {
	c.preStartFunc = f
	return c
}

func (c *Cmd) PostStart(f func() error) *Cmd {
	c.postStartFunc = f
	return c
}

func (c *Cmd) PostEnd(f func() error) *Cmd {
	c.postEndFunc = f
	return c
}

func (c *Cmd) Run() (string, error) {
	cmd := goexec.Command(c.command[0], c.command[1:]...)
	cmd.Env = append(os.Environ(), c.env...)
	if c.cwd != "" {
		cmd.Dir = c.cwd
	}
	c.l.Tracef("executing %q", cmd.String())

	combinedBuf := new(bytes.Buffer)
	traceWriter := c.l.WriterLevel(logrus.TraceLevel)
	warnWriter := c.l.WriterLevel(logrus.WarnLevel)

	c.stdoutWriters = append(c.stdoutWriters, combinedBuf, traceWriter)
	c.stderrWriters = append(c.stderrWriters, combinedBuf, warnWriter)
	cmd.Stdout = io.MultiWriter(c.stdoutWriters...)
	cmd.Stderr = io.MultiWriter(c.stderrWriters...)

	if c.preStartFunc != nil {
		if err := c.preStartFunc(); err != nil {
			return "", err
		}
	}

	if err := cmd.Start(); err != nil {
		return "", err
	}

	if c.postStartFunc != nil {
		if err := c.postStartFunc(); err != nil {
			return "", err
		}
	}

	if c.wait {
		if c.t != 0 {
			timer := time.AfterFunc(c.t, func() {
				cmd.Process.Kill()
			})
			defer timer.Stop()
		}

		if err := cmd.Wait(); err != nil {
			if c.t == 0 {
				return "", err
			}
		}
		if c.postEndFunc != nil {
			if err := c.postEndFunc(); err != nil {
				return "", err
			}
		}
	}

	return combinedBuf.String(), nil
}
