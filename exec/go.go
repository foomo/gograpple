package exec

import (
	"github.com/sirupsen/logrus"
)

type GoCmd struct {
	Cmd
}

func NewGoCommand(l *logrus.Entry) *GoCmd {
	return &GoCmd{*NewCommand(l, "go")}
}

func (c GoCmd) Build(workDir, output string, inputs []string, flags ...string) *Cmd {
	return c.Args("build", "-o", output).Cwd(workDir).Args(flags...).Args(inputs...)
}
