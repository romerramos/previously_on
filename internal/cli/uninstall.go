package cli

import (
	"fmt"

	"github.com/romerramos/previously_on/internal/hooks"
	"github.com/romerramos/previously_on/internal/nvim"
	"github.com/spf13/cobra"
)

func uninstallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Uninstall Previously on integrations",
	}
	cmd.AddCommand(uninstallGitHookCmd())
	cmd.AddCommand(uninstallNvimCmd())
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

func uninstallNvimCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "nvim",
		Short: "Remove the Neovim plugin installed by Previously on",
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := nvim.Uninstall()
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Uninstalled previously-on Neovim integration:")
			fmt.Fprintf(cmd.OutOrStdout(), "- nvim plugin: %s (%s)\n", result.Action, result.PluginPath)
			if result.SpecPath != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "- lazy.nvim spec: removed if managed (%s)\n", result.SpecPath)
			}
			return nil
		},
	}
}
