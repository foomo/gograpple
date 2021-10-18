package exec

import (
	"fmt"
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
