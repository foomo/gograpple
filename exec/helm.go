package exec

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

type HelmCmd struct {
	Cmd
}

func NewHelmCommand() *HelmCmd {
	return &HelmCmd{*NewCommand("helm")}
}

func (c HelmCmd) Rollback(deployment string, revision int) *Cmd {
	return c.Args("rollback", deployment, fmt.Sprint(revision), "--force")
}

func (c HelmCmd) GetLatestRevision(ctx context.Context, deployment string) (int, error) {
	command := "helm history example | tail -1 | cut -d ' ' -f1"
	// since were piping well be using bash
	out, err := NewCommand("bash").Args("-c", command).Run(ctx)
	if err != nil {
		return 0, err
	}
	revision, err := strconv.Atoi(strings.Trim(out, "\n"))
	if err != nil {
		return 0, err
	}
	return revision, nil
}
