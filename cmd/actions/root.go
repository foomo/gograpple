package actions

import (
	"github.com/foomo/gograpple"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func init() {

	rootCmd.PersistentFlags().StringVarP(&flagDir, "dir", "d", ".", "Specifies working directory")
	rootCmd.PersistentFlags().StringVarP(&flagNamespace, "namespace", "n", "default", "namespace name")
	rootCmd.PersistentFlags().BoolVarP(&flagVerbose, "verbose", "v", false, "Specifies should command output be displayed")
	rootCmd.PersistentFlags().StringVarP(&flagPod, "pod", "p", "", "pod name (default most recent one)")
	rootCmd.PersistentFlags().StringVarP(&flagContainer, "container", "c", "", "container name (default deployment name)")
	rootCmd.PersistentFlags().BoolVarP(&flagDebug, "debug", "", false, "debug mode")
	patchCmd.Flags().StringVar(&flagImage, "image", "alpine:latest", "image to be used for patching (default alpine:latest)")
	patchCmd.Flags().StringArrayVarP(&flagMounts, "mount", "m", []string{}, "host path to be mounted (default none)")
	patchCmd.Flags().BoolVar(&flagRollback, "rollback", false, "rollback deployment to a previous state")
	delveCmd.Flags().StringVar(&flagSourcePath, "source", "", ".go file source path (default cwd)")
	delveCmd.Flags().Var(flagArgs, "args", "go file args")
	delveCmd.Flags().Var(flagListen, "listen", "delve host:port to listen on")
	delveCmd.Flags().BoolVar(&flagVscode, "vscode", false, "launch a debug configuration in vscode")
	delveCmd.Flags().BoolVar(&flagContinue, "continue", false, "start delve server execution without waiting for client connection")
	delveCmd.Flags().BoolVar(&flagJSONLog, "json-log", false, "log as json")
	configCmd.Flags().BoolVar(&flagAttach, "attach", false, "debug with attach")
	rootCmd.AddCommand(versionCmd, patchCmd, shellCmd, delveCmd, configCmd)
}

var (
	flagImage      string
	flagDir        string
	flagVerbose    bool
	flagNamespace  string
	flagPod        string
	flagContainer  string
	flagRepo       string
	flagMounts     []string
	flagSourcePath string
	flagArgs       = NewStringList(" ")
	flagRollback   bool
	flagListen     = NewHostPort("127.0.0.1", 0)
	flagVscode     bool
	flagContinue   bool
	flagJSONLog    bool
	flagDebug      bool
)

var (
	l       *logrus.Entry
	grapple *gograpple.Grapple

	rootCmd = &cobra.Command{
		Use: "gograpple",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if cmd.Name() == commandNameVersion || cmd.Name() == commandNameConfig {
				return nil
			}
			l = newLogger(flagVerbose, flagJSONLog)
			var err error
			err = gograpple.ValidatePath(".", &flagDir)
			if err != nil {
				return err
			}
			grapple, err = gograpple.NewGrapple(l, flagNamespace, args[0], flagDebug)
			if err != nil {
				return err
			}
			return gograpple.ValidatePath(flagDir, &flagSourcePath)
		},
	}
	patchCmd = &cobra.Command{
		Use:   "patch [DEPLOYMENT] -c {CONTAINER} -n {NAMESPACE} -i {IMAGE} -t {TAG} -m {MOUNT}",
		Short: "applies a development patch for a deployment",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if flagRollback {
				return grapple.Rollback()
			}
			mounts, err := gograpple.ValidateMounts(flagDir, flagMounts)
			if err != nil {
				return err
			}
			return grapple.Patch(flagImage, flagContainer, mounts)
		},
	}
	shellCmd = &cobra.Command{
		Use:   "shell [DEPLOYMENT] -n {NAMESPACE} -c {CONTAINER}",
		Short: "shell into the dev patched deployment",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return grapple.Shell(flagPod)
		},
	}
	delveCmd = &cobra.Command{
		Use:   "delve [DEPLOYMENT] --source {SRC} -n {NAMESPACE} -c {CONTAINER}",
		Short: "start a headless delve debug server for .go input on a patched deployment",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return grapple.Delve(flagPod, flagContainer, flagSourcePath, flagArgs.items, flagListen.Host, flagListen.Port, flagVscode, flagContinue)
		},
	}
)

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		l = logrus.NewEntry(logrus.New())
		l.Fatal(err)
	}
}

func newLogger(verbose, jsonLog bool) *logrus.Entry {
	logger := logrus.New()
	if jsonLog {
		logger.SetFormatter(&logrus.JSONFormatter{
			DisableTimestamp: true,
		})
	}
	if verbose {
		logger.SetLevel(logrus.TraceLevel)
	}
	return logrus.NewEntry(logger)
}
