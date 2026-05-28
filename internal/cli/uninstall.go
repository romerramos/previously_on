package cli

import (
	"fmt"

	"github.com/romerramos/previously_on/internal/hooks"
	"github.com/spf13/cobra"
)

func uninstallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Uninstall Previously on integrations",
	}
	cmd.AddCommand(uninstallGitHookCmd())
	return cmd
}

func uninstallGitHookCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "git-hook",
		Short: "Remove Git hook blocks installed by Previously on",
		RunE: func(cmd *cobra.Command, args []string) error {
			repoRoot, err := gitRoot()
			if err != nil {
				return err
			}
			results, err := hooks.UninstallGitHooks(repoRoot)
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Uninstalled previously-on Git hook integration:")
			for _, result := range results {
				fmt.Fprintf(cmd.OutOrStdout(), "- %s: %s (%s)\n", result.HookName, result.Action, result.HookPath)
			}
			return nil
		},
	}
}
