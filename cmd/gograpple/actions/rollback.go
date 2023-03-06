package actions

import (
	"github.com/foomo/gograpple"
	"github.com/spf13/cobra"
)

var (
	flagNamespace  string
	rollbackCmd    = &cobra.Command{
		Use:   "rollback [DEPLOYMENT] -n {NAMESPACE} [FLAGS]",
		Short: "rollback deployment",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			g, err := gograpple.NewGrapple(flagNamespace, args[0])
			if err != nil {
				return err
			}
			return g.Rollback()
		},
	}
)
