package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func statusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show current repo and last-seen metadata",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := loadContext()
			if err != nil {
				return err
			}

			current, err := ctx.Store.Load(ctx.Repo.ID)
			if err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Repo: %s\n", ctx.Repo.Name)
			fmt.Fprintf(cmd.OutOrStdout(), "Root: %s\n", ctx.Repo.Root)
			fmt.Fprintf(cmd.OutOrStdout(), "Remote: %s\n", displayFallback(ctx.Repo.RemoteURL, "none"))
			fmt.Fprintf(cmd.OutOrStdout(), "Current branch: %s\n", displayFallback(ctx.Repo.CurrentBranch, "detached HEAD"))
			fmt.Fprintf(cmd.OutOrStdout(), "Current HEAD: %s\n", ctx.Repo.Head)

			if current == nil {
				fmt.Fprintln(cmd.OutOrStdout(), "Last seen: never")
				return nil
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Last seen: %s\n", current.LastSeenAt.Format(timeFormat))
			fmt.Fprintf(cmd.OutOrStdout(), "Last seen branch: %s\n", displayFallback(current.LastSeenBranch, "detached HEAD"))
			fmt.Fprintf(cmd.OutOrStdout(), "Last seen HEAD: %s\n", current.LastSeenHead)
			if !current.LastGeneratedSummaryAt.IsZero() {
				fmt.Fprintf(cmd.OutOrStdout(), "Last summary: %s\n", current.LastGeneratedSummaryAt.Format(timeFormat))
			}
			return nil
		},
	}
}

const timeFormat = "2006-01-02 15:04:05 MST"

func displayFallback(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
