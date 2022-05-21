package actions

import (
	"path"

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
			g, err := gograpple.NewGrapple(newLogger(flagVerbose, flagJSONLog), c.Namespace, c.Deployment)
			if err != nil {
				return err
			}
			repo, image, tag, err := suggest.ParseImage(c.Image)
			if err != nil {
				return err
			}
			if repo != "" {
				image = path.Join(repo, image)
			}
			if err := g.Patch(c.Repository, image, tag, c.Container, nil); err != nil {
				return err
			}
			defer g.Rollback()
			switch c.Launch {
			case "vscode":
				flagVscode = true
			case "goland":
				//TODO implement goland launch support
			}
			return g.Delve("", c.Container, c.SourcePath, nil, flagListen.Host, flagListen.Port, flagVscode)
		},
	}
)
