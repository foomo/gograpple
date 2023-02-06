package actions

import (
	"github.com/foomo/gograpple"
	"github.com/foomo/gograpple/kubectl"
	"github.com/spf13/cobra"
)

const commandNameConfig = "config"

var (
	configCmd = &cobra.Command{
		Use:   "config [PATH]",
		Short: "load/create config and run patch and delve",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := gograpple.LoadConfig(args[0])
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
			if c.AttachTo == "" {
				if err := g.Patch(c.Image, c.Container, nil); err != nil {
					return err
				}
				defer g.Rollback()
				// todo support binargs from config
				return g.Delve("", c.Container, c.SourcePath, nil, host, port, c.LaunchVscode, c.DelveContinue)
			}
			return g.Attach(c.Namespace, c.Deployment, c.Container, c.AttachTo, c.Arch, host, port)
		},
	}
)
