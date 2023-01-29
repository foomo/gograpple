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
			g, err := gograpple.NewGrapple(newLogger(flagVerbose, flagJSONLog), c.Namespace, c.Deployment)
			if err != nil {
				return err
			}
			if c.Attach == "" {
				addr := HostPort{}
				if err := addr.Set(c.ListenAddr); err != nil {
					return err
				}
				if err := kubectl.SetContext(c.Cluster); err != nil {
					return err
				}
				if err := g.Patch(c.Image, c.Container, nil); err != nil {
					return err
				}
				defer g.Rollback()
				// todo support binargs from config
				return g.Delve("", c.Container, c.SourcePath, nil, addr.Host, addr.Port, c.LaunchVscode, c.DelveContinue)
			}
			port, err := c.Port()
			if err != nil {
				return err
			}
			return g.Attach(c.Namespace, c.Deployment, c.Container, c.Attach, port)
		},
	}
)
