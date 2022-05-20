package exec

import (
	"bytes"
	"context"
	"io"
	"os"
	goexec "os/exec"

	"github.com/sirupsen/logrus"
)

type Cmd struct {
	l *logrus.Entry
	// actual        *exec.Cmd
	command       []string
	cwd           string
	env           []string
	stdin         io.Reader
	stdoutWriters []io.Writer
	stderrWriters []io.Writer
	wait          bool
	// t             time.Duration
	preStartFunc  func() error
	postStartFunc func(p *os.Process) error
	postEndFunc   func() error
	chanStarted   chan struct{}
	chanDone      chan struct{}
}

func NewCommand(name string) *Cmd {
	return &Cmd{
		command:     []string{name},
		wait:        true,
		env:         os.Environ(),
		chanStarted: make(chan struct{}),
		chanDone:    make(chan struct{}),
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

func (c *Cmd) NoWait() *Cmd {
	c.wait = false
	return c
}

func (c *Cmd) PreStart(f func() error) *Cmd {
	c.preStartFunc = f
	return c
}

func (c *Cmd) PostStart(f func(p *os.Process) error) *Cmd {
	c.postStartFunc = f
	return c
}

func (c *Cmd) PostEnd(f func() error) *Cmd {
	c.postEndFunc = f
	return c
}

func (c *Cmd) Logger(l *logrus.Entry) *Cmd {
	c.l = l
	return c
}

func (c *Cmd) Run(ctx context.Context) (string, error) {
	return c.run(goexec.CommandContext(ctx, c.command[0], c.command[1:]...))
}

func (c *Cmd) Started() <-chan struct{} {
	return c.chanStarted
}

func (c *Cmd) Done() <-chan struct{} {
	return c.chanDone
}

func (c *Cmd) run(cmd *goexec.Cmd) (string, error) {
	cmd.Env = append(os.Environ(), c.env...)
	if c.cwd != "" {
		cmd.Dir = c.cwd
	}

	combinedBuf := new(bytes.Buffer)
	c.stdoutWriters = append(c.stdoutWriters, combinedBuf)
	c.stderrWriters = append(c.stderrWriters, combinedBuf)
	if c.l != nil {
		c.l.Tracef("executing %q", cmd.String())
		c.stdoutWriters = append(c.stdoutWriters, c.l.WriterLevel(logrus.TraceLevel))
		c.stderrWriters = append(c.stderrWriters, c.l.WriterLevel(logrus.WarnLevel))
	}
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
		if err := c.postStartFunc(cmd.Process); err != nil {
			return "", err
		}
	}

	go func() {
		c.chanStarted <- struct{}{}
	}()

	if c.wait {
		if err := cmd.Wait(); err != nil {
			return "", err
		}
		if c.postEndFunc != nil {
			if err := c.postEndFunc(); err != nil {
				return "", err
			}
		}
	}

	go func() {
		c.chanDone <- struct{}{}
	}()

	return combinedBuf.String(), nil
}
