package actions

import (
	"path"

	"github.com/foomo/gograpple"
	"github.com/foomo/gograpple/config"
	"github.com/foomo/gograpple/kubectl"
	"github.com/spf13/cobra"
)

var (
	flagAttach     bool
	flagConfigPath string
	flagGenerate   bool
	debugCmd       = &cobra.Command{
		Use:   "debug [FLAGS]",
		Short: "run patch and delve",
		Args:  cobra.MinimumNArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			if flagAttach {
				return attachDebug(flagConfigPath)
			}
			return patchDebug(flagConfigPath)
		},
	}
)

// if filePath != "" {
// 	defer func() {
// 		if err := save(filePath, config); err != nil {
// 			log.Error(err)
// 		}
// 	}()
// 	configLoaded := false
// 	if _, err := os.Stat(filePath); err == nil {
// 		if err := LoadYaml(filePath, config); err != nil {
// 			// if the config path doesnt exist
// 			return err
// 		}
// 		configLoaded = true
// 	}
// 	if configLoaded {
// 		// skip filled when loaded from file
// 		opts = append(opts, gencon.OptionSkipFilled())
// 	}
// }

func attachDebug(configPath string) error {
	if configPath == "" {
		fp = path.Join(configPath, "./gograpple-attach.yaml")
	}
	c := config.AttachConfig{}
	err := config.Generate(&c)
	if err != nil {
		return err
	}
	g, err := gograpple.NewGrapple(c.Namespace, c.Deployment)
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
	g, err := gograpple.NewGrapple(c.Namespace, c.Deployment)
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
