package exec

import (
	"context"
	"fmt"
	"strings"
)

type DockerCmd struct {
	Cmd
}

func NewDockerCommand() *DockerCmd {
	return &DockerCmd{*NewCommand("docker")}
}

func (c DockerCmd) Build(workDir string, options ...string) *Cmd {
	return c.Args("build", workDir).Args(options...)
}

func (c DockerCmd) Push(image, tag string, options ...string) *Cmd {
	return c.Args("push", fmt.Sprintf("%v:%v", image, tag)).Args(options...)
}

func (c DockerCmd) Pull(image, tag string, options ...string) *Cmd {
	return c.Args("pull", fmt.Sprintf("%v:%v", image, tag)).Args(options...)
}

func (c DockerCmd) ImageInspect(options ...string) *Cmd {
	return c.Args("image", "inspect").Args(options...)
}

func (c DockerCmd) GetPlatform(ctx context.Context, image string) (*Platform, error) {
	out, err := c.ImageInspect("-f", "{{.Os}}/{{.Architecture}}", image).Run(ctx)
	if err != nil {
		return nil, err
	}
	return NewPlatform(strings.TrimRight(out, "\n"))
}

type Platform struct {
	OS   string
	Arch string
}

func NewPlatform(v string) (*Platform, error) {
	pieces := strings.Split(v, "/")
	if len(pieces) != 2 {
		return nil, fmt.Errorf("invalid platform format %q", v)
	}
	return &Platform{pieces[0], pieces[1]}, nil
}

func (p Platform) String() string {
	return fmt.Sprintf("%v/%v", p.OS, p.Arch)
}
