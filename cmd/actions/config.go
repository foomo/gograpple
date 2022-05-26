package actions

import (
	"fmt"

	"github.com/foomo/gograpple"
	"github.com/foomo/gograpple/suggest"
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
			if err := suggest.KubeConfig(suggest.DefaultKubeConfig).SetContext(c.Cluster); err != nil {
				return err
			}
			g, err := gograpple.NewGrapple(newLogger(flagVerbose, flagJSONLog), c.Namespace, c.Deployment)
			if err != nil {
				return err
			}
			if err := g.Patch(c.Repository, c.Dockerfile, c.Container, nil); err != nil {
				return err
			}
			defer g.Rollback()
			switch c.Launch {
			//TODO implement goland launch support
			case "":
			case "vscode":
				flagVscode = true
			default:
				return fmt.Errorf("unsupported launch option %q", c.Launch)
			}
			addr := HostPort{}
			if err := addr.Set(c.ListenAddr); err != nil {
				return err
			}
			return g.Delve("", c.Container, c.SourcePath, nil, addr.Host, addr.Port, flagVscode)
		},
	}
)
