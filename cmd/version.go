package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(versionCmd)
}

// set on build
var version = ""

var (
	versionCmd = &cobra.Command{
		Use:   "version",
		Short: "prints cli version",
		Long:  "prints the current installed cli version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(getVersion())
		},
	}
)

func getVersion() string {
	if version == "" {
		return "latest"
	}
	return version
}
