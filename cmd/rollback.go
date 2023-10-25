package cmd

import (
	"github.com/foomo/gograpple/internal/grapple"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(rollbackCmd)
}

var (
	rollbackCmd = &cobra.Command{
		Use:   "rollback [namespace] [deployment]",
		Short: "rollback the patched deployment",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			g, err := grapple.NewGrapple(newLogEntry(flagDebug), args[0], args[1])
			if err != nil {
				return err
			}
			return g.Rollback()
		},
	}
)
