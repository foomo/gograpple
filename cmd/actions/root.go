package actions

import (
	"github.com/foomo/gograpple"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func init() {

	rootCmd.PersistentFlags().StringVarP(&flagTag, "tag", "t", "latest", "Specifies the image tag")
	rootCmd.PersistentFlags().StringVarP(&flagDir, "dir", "d", ".", "Specifies working directory")
	rootCmd.PersistentFlags().StringVarP(&flagNamespace, "namespace", "n", "default", "namespace name")
	rootCmd.PersistentFlags().BoolVarP(&flagVerbose, "verbose", "v", false, "Specifies should command output be displayed")
	rootCmd.PersistentFlags().StringVarP(&flagPod, "pod", "p", "", "pod name (default most recent one)")
	rootCmd.PersistentFlags().StringVarP(&flagContainer, "container", "c", "", "container name (default deployment name)")
	patchCmd.Flags().StringVarP(&flagImage, "image", "i", "", "image to be used for patching (default deployment image)")
	patchCmd.Flags().StringArrayVarP(&flagMounts, "mount", "m", []string{}, "host path to be mounted (default none)")
	patchCmd.Flags().BoolVar(&flagRollback, "rollback", false, "rollback deployment to a previous state")
	delveCmd.Flags().StringVar(&flagInput, "input", "", "go file input (default cwd)")
	delveCmd.Flags().BoolVar(&flagCleanup, "cleanup", false, "cleanup delve debug session")
	delveCmd.Flags().BoolVar(&flagContinue, "continue", false, "delve --continue option")
	delveCmd.Flags().Var(flagArgs, "args", "go file args")
	delveCmd.Flags().Var(flagListen, "listen", "delve host:port to listen on")
	delveCmd.Flags().BoolVar(&flagVscode, "vscode", false, "launch a debug configuration in vscode")
	delveCmd.Flags().BoolVar(&flagJSONLog, "json-log", false, "log as json")
	rootCmd.AddCommand(versionCmd, patchCmd, shellCmd, delveCmd)
}

var (
	flagTag       string
	flagDir       string
	flagVerbose   bool
	flagNamespace string
	flagPod       string
	flagContainer string
	flagImage     string
	flagMounts    []string
	flagInput     string
	flagArgs      = newStringList(" ")
	flagCleanup   bool
	flagRollback  bool
	flagContinue  bool
	flagListen    = newHostPort("127.0.0.1", 0)
	flagVscode    bool
	flagJSONLog   bool
)

var (
	l       *logrus.Entry
	grapple *gograpple.Grapple

	rootCmd = &cobra.Command{
		Use: "gograpple",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if cmd.Name() == commandNameVersion {
				return nil
			}
			l = newLogger(flagVerbose, flagJSONLog)
			var err error
			err = gograpple.ValidatePath(".", &flagDir)
			if err != nil {
				return err
			}
			grapple, err = gograpple.NewGrapple(l, flagNamespace, args[0])
			if err != nil {
				return err
			}
			return gograpple.ValidatePath(flagDir, &flagInput)
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
			return grapple.Patch(flagImage, flagTag, flagContainer, mounts)
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
		Use:   "delve [DEPLOYMENT] --input {INPUT} -n {NAMESPACE} -c {CONTAINER}",
		Short: "start a headless delve debug server for .go input on a patched deployment",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if flagCleanup {
				return grapple.Cleanup(flagPod, flagContainer)
			}
			return grapple.Delve(flagPod, flagContainer, flagInput, flagArgs.items, flagListen.Host, flagListen.Port, flagContinue, flagVscode)
		},
	}
)

func Execute() {
	if err := rootCmd.Execute(); err != nil {
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
