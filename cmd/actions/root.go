package actions

import (
	"github.com/foomo/gograpple"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	v1 "k8s.io/api/apps/v1"
)

func init() {

	rootCmd.PersistentFlags().StringVarP(&flagTag, "tag", "t", "latest", "Specifies the image tag")
	rootCmd.PersistentFlags().StringVarP(&flagDir, "dir", "d", ".", "Specifies working directory")
	rootCmd.PersistentFlags().StringVarP(&flagNamespace, "namespace", "n", "default", "namespace name")
	rootCmd.PersistentFlags().BoolVarP(&flagVerbose, "verbose", "v", false, "Specifies should command output be displayed")

	rootCmd.PersistentFlags().StringVarP(&flagPod, "pod", "p", "", "pod name (default most recent one)")
	rootCmd.PersistentFlags().StringVarP(&flagContainer, "container", "c", "", "container name (default deployment name)")
	devPatchCmd.Flags().StringVarP(&flagImage, "image", "i", "", "image to be used for patching (default deployment image)")
	devPatchCmd.Flags().StringArrayVarP(&flagMounts, "mount", "m", []string{}, "host path to be mounted (default none)")
	devPatchCmd.Flags().BoolVar(&flagRollback, "rollback", false, "rollback deployment to a previous state")
	devDelveCmd.Flags().StringVar(&flagInput, "input", "", "go file input (default cwd)")
	devDelveCmd.Flags().BoolVar(&flagCleanup, "cleanup", false, "cleanup delve debug session")
	devDelveCmd.Flags().BoolVar(&flagContinue, "continue", false, "delve --continue option")
	devDelveCmd.Flags().Var(flagArgs, "args", "go file args")
	devDelveCmd.Flags().Var(flagListen, "listen", "delve host:port to listen on")
	devDelveCmd.Flags().BoolVar(&flagVscode, "vscode", false, "launch a debug configuration in vscode")
	rootCmd.AddCommand(versionCmd, devPatchCmd, devShellCmd, devDelveCmd)
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
)

var (
	log        = logrus.New()
	l          *logrus.Entry
	deployment *v1.Deployment

	rootCmd = &cobra.Command{
		Use: "gograpple",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			l = newLogger(flagVerbose)
			var err error
			err = gograpple.ValidatePath(".", &flagDir)
			if err != nil {
				return err
			}
			err = gograpple.ValidateNamespace(l, flagNamespace)
			if err != nil {
				return err
			}
			err = gograpple.ValidateDeployment(l, flagNamespace, args[0])
			if err != nil {
				return err
			}
			deployment, err = gograpple.GetDeployment(l, flagNamespace, args[0])
			if err != nil {
				return err
			}
			err = gograpple.ValidatePod(l, deployment, &flagPod)
			if err != nil {
				return err
			}
			err = gograpple.ValidateContainer(l, deployment, &flagContainer)
			if err != nil {
				return err
			}
			err = gograpple.ValidateImage(l, deployment, flagContainer, &flagImage, &flagTag)
			if err != nil {
				return err
			}
			return gograpple.ValidatePath(flagDir, &flagInput)
		},
	}
	devPatchCmd = &cobra.Command{
		Use:   "patch [DEPLOYMENT] -c {CONTAINER} -n {NAMESPACE} -i {IMAGE} -t {TAG} -m {MOUNT}",
		Short: "applies a development patch for a deployment",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			mounts, err := gograpple.ValidateMounts(flagDir, flagMounts)
			if err != nil {
				return err
			}

			_, err = patch(l, flagNamespace, deployment, flagPod, flagContainer, flagImage, flagTag, mounts, flagRollback)
			return err
		},
	}
	devShellCmd = &cobra.Command{
		Use:   "shell [DEPLOYMENT] -n {NAMESPACE} -c {CONTAINER}",
		Short: "shell into the dev patched deployment",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			_, err := shell(l, deployment, flagPod)
			if err != nil {
				log.WithError(err).Fatalf("shelling into dev mode failed")
			}
		},
	}
	devDelveCmd = &cobra.Command{
		Use:   "delve [DEPLOYMENT] -input {INPUT} -n {NAMESPACE} -c {CONTAINER}",
		Short: "start a headless delve debug server for .go input on a patched deployment",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			_, err := delve(l, deployment, flagPod, flagContainer, flagInput, flagArgs.items, flagListen.Host, flagListen.Port, flagCleanup, flagContinue, flagVscode)
			if err != nil {
				log.WithError(err).Fatalf("debug in dev mode failed")
			}
		},
	}
)

func patch(l *logrus.Entry, namespace string, deployment *v1.Deployment, pod, container, image, tag string, mounts []gograpple.Mount, rollback bool) (string, error) {
	if rollback {
		return gograpple.Rollback(l, deployment)
	}
	return gograpple.Patch(l, deployment, container, image, tag, mounts)
}

func shell(l *logrus.Entry, deployment *v1.Deployment, pod string) (string, error) {
	return gograpple.Shell(l, deployment, pod)
}

func delve(l *logrus.Entry, deployment *v1.Deployment, pod, container, input string, args []string, host string, port int, cleanup, dlvContinue, vscode bool) (string, error) {
	if cleanup {
		return gograpple.DelveCleanup(l, deployment, pod, container)
	}
	return gograpple.Delve(l, deployment, pod, container, input, args, dlvContinue, host, port, vscode)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}

}

func newLogger(verbose bool) *logrus.Entry {
	logger := logrus.New()
	if verbose {
		logger.SetLevel(logrus.TraceLevel)
	}
	return logrus.NewEntry(logger)
}
