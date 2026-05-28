package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func resetCmd() *cobra.Command {
	var all bool

	cmd := &cobra.Command{
		Use:   "reset",
		Short: "Delete stored state for the current repo",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := loadContext()
			if err != nil {
				return err
			}

			if all {
				if err := ctx.Store.ResetAll(); err != nil {
					return err
				}
				fmt.Fprintln(cmd.OutOrStdout(), "Deleted all previously-on state")
				return nil
			}

			if err := ctx.Store.Reset(ctx.Repo.ID); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Deleted stored state for %s\n", ctx.Repo.Name)
			return nil
		},
	}

	cmd.Flags().BoolVar(&all, "all", false, "delete all stored repository state")
	return cmd
}
