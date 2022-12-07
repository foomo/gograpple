package exec

type GoCmd struct {
	Cmd
}

func NewGoCommand() *GoCmd {
	return &GoCmd{*NewCommand("go")}
}

func (c GoCmd) Build(output string, inputs []string, flags ...string) *Cmd {
	return c.Args("build", "-o", output).Args(flags...).Args(inputs...)
}
