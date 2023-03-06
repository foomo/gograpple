package actions

import (
	"github.com/foomo/gograpple/log"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.PersistentFlags().BoolVarP(&flagVerbose, "verbose", "v", false, "verbose mode")
	debugCmd.Flags().BoolVar(&flagAttach, "attach", false, "debug with attach")
	debugCmd.Flags().StringVarP(&flagConfigPath, "config", "c", "", "config path")
	rollbackCmd.Flags().StringVarP(&flagNamespace, "namespace", "n", "default", "namespace to work in")
	rootCmd.AddCommand(versionCmd, debugCmd, rollbackCmd)
}

var (
	flagVerbose bool
)

var (
	rootCmd = &cobra.Command{
		Use: "gograpple",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			if cmd.Name() == commandNameVersion {
				return
			}
			initLogger(flagVerbose, false)
		},
	}
)

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Logger().Fatal(err)
	}
}

func initLogger(verbose, jsonLog bool) {
	if jsonLog {
		log.Logger().SetFormatter(&logrus.JSONFormatter{
			DisableTimestamp: true,
		})
	}
	if verbose {
		log.Logger().SetLevel(logrus.TraceLevel)
	}
}
