package actions

import (
	"path"

	"github.com/foomo/gograpple"
	"github.com/foomo/gograpple/config"
	"github.com/foomo/gograpple/kubectl"
	"github.com/spf13/cobra"
)

const commandNameConfig = "config"

var (
	flagAttach  bool
	flagSaveDir string
	configCmd   = &cobra.Command{
		Use:   "config [FLAGS]",
		Short: "load/create config and run patch and delve",
		Args:  cobra.MinimumNArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			if flagAttach {
				return attachDebug(flagSaveDir)
			}
			return patchDebug(flagSaveDir)
		},
	}
)

func attachDebug(baseDir string) error {
	fp := ""
	if baseDir != "" {
		fp = path.Join(baseDir, "gograpple-attach.yaml")
	}
	c := config.AttachConfig{}
	err := config.Init(fp, &c)
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

func patchDebug(baseDir string) error {
	fp := ""
	if baseDir != "" {
		fp = path.Join(baseDir, "gograpple-patch.yaml")
	}
	c := config.PatchConfig{}
	err := config.Init(fp, &c)
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
