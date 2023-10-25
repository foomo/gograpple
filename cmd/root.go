package cmd

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.PersistentFlags().BoolVarP(&flagDebug, "debug", "", false, "debug mode")
}

var (
	// flagImage      string
	// flagDir        string
	flagDebug bool
	// flagNamespace  string
	// flagPod        string
	// flagContainer  string
	// flagRepo       string
	// flagMounts     []string
	// flagSourcePath string
	// flagArgs       = NewStringList(" ")
	// flagRollback   bool
	// flagListen     = NewHostPort("127.0.0.1", 0)
	// flagVscode     bool
	// flagContinue   bool
	// flagJSONLog    bool
	// flagDebug bool
)

var (
	rootCmd = &cobra.Command{
		Use: "gograpple",
	}
)

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		le := newLogEntry(flagDebug)
		le.Fatal(err)
	}
}

func newLogEntry(debug bool) *logrus.Entry {
	logger := logrus.New()
	if debug {
		logger.SetLevel(logrus.TraceLevel)
	}
	return logrus.NewEntry(logger)
}
