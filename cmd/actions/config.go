package actions

import (
	"github.com/foomo/gograpple"
	"github.com/foomo/gograpple/config"
	"github.com/foomo/gograpple/kubectl"
	"github.com/spf13/cobra"
)

const commandNameConfig = "config"

var (
	flagAttach = false
	configCmd  = &cobra.Command{
		Use:   "config [DIR] [FLAGS]",
		Short: "load/create config and run patch and delve",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if flagAttach {
				return attachDebug(args[0])
			}
			return patchDebug(args[0])
		},
	}
)

func attachDebug(base string) error {
	c, err := config.NewAttachConfig(base)
	if err != nil {
		return err
	}
	g, err := gograpple.NewGrapple(newLogger(flagVerbose, flagJSONLog), c.Namespace, c.Deployment, flagDebug)
	if err != nil {
		return err
	}
	host, port, err := c.Addr()
	if err != nil {
		return err
	}
	if err := kubectl.SetContext(c.Cluster); err != nil {
		return err
	}
	return g.Attach(c.Namespace, c.Deployment, c.Container, c.AttachTo, c.Arch, host, port)
}

func patchDebug(base string) error {
	c, err := config.NewPatchConfig(base)
	if err != nil {
		return err
	}
	g, err := gograpple.NewGrapple(newLogger(flagVerbose, flagJSONLog), c.Namespace, c.Deployment, flagDebug)
	if err != nil {
		return err
	}
	host, port, err := c.Addr()
	if err != nil {
		return err
	}
	if err := kubectl.SetContext(c.Cluster); err != nil {
		return err
	}
	if err := g.Patch(c.Image, c.Container, nil); err != nil {
		return err
	}
	defer g.Rollback()
	return g.Delve("", c.Container, c.SourcePath, nil, host, port, c.LaunchVscode, c.DelveContinue)
}
