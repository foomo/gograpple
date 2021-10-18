package exec

type GoCmd struct {
	Cmd
}

func NewGoCommand() *GoCmd {
	return &GoCmd{*NewCommand("go")}
}

func (c GoCmd) Build(workDir, output string, inputs []string, flags ...string) *Cmd {
	return c.Args("build", "-o", output).Cwd(workDir).Args(flags...).Args(inputs...)
}
