package cli

import (
	"fmt"

	"github.com/romerramos/previously_on/internal/state"
	"github.com/spf13/cobra"
)

func markSeenCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "mark-seen",
		Short: "Update this repo's last-seen state without generating a summary",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := loadContext()
			if err != nil {
				return err
			}

			snapshot, err := state.FromRepo(ctx.Repo)
			if err != nil {
				return err
			}
			if err := ctx.Store.Save(snapshot); err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Marked %s as seen at %s\n", ctx.Repo.Name, snapshot.LastSeenAt.Format(timeFormat))
			return nil
		},
	}
}
