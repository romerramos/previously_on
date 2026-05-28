package cli

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/romerramos/previously_on/internal/hooks"
	"github.com/romerramos/previously_on/internal/nvim"
	"github.com/romerramos/previously_on/internal/nvimselect"
	"github.com/spf13/cobra"
)

func installCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install Previously on integrations",
	}
	cmd.AddCommand(installGitHookCmd())
	cmd.AddCommand(installNvimCmd())
	return cmd
}

func installGitHookCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "git-hook",
		Short: "Install Git hooks that show a summary after pull/merge/rebase",
		RunE: func(cmd *cobra.Command, args []string) error {
			repoRoot, err := gitRoot()
			if err != nil {
				return err
			}
			executable, err := os.Executable()
			if err != nil {
				return err
			}
			if resolved, err := filepath.EvalSymlinks(executable); err == nil {
				executable = resolved
			}

			results, err := hooks.InstallGitHooks(repoRoot, executable)
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Installed previously-on Git hook integration:")
			for _, result := range results {
				fmt.Fprintf(cmd.OutOrStdout(), "- %s: %s (%s)\n", result.HookName, result.Action, result.HookPath)
				if result.Warning != "" {
					fmt.Fprintf(cmd.OutOrStdout(), "  Warning: %s\n", result.Warning)
				}
			}
			fmt.Fprintln(cmd.OutOrStdout(), "The hooks run `previously-on summary --simple` after successful merge/rebase.")
			fmt.Fprintln(cmd.OutOrStdout(), "Set PREVIOUSLY_ON_SKIP_HOOK=1 to skip hook output temporarily.")
			return nil
		},
	}
}

func installNvimCmd() *cobra.Command {
	var strategy string
	cmd := &cobra.Command{
		Use:   "nvim",
		Short: "Install the Neovim plugin that opens a Markdown repo summary",
		RunE: func(cmd *cobra.Command, args []string) error {
			resolvedStrategy, err := resolveNvimStrategy(cmd, strategy)
			if err != nil {
				return err
			}

			executable, err := os.Executable()
			if err != nil {
				return err
			}
			if resolved, err := filepath.EvalSymlinks(executable); err == nil {
				executable = resolved
			}

			result, err := nvim.InstallAt(nvimDataHome(), nvimConfigHome(), executable, resolvedStrategy)
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Installed previously-on Neovim integration:")
			fmt.Fprintf(cmd.OutOrStdout(), "- strategy: %s\n", resolvedStrategy)
			fmt.Fprintf(cmd.OutOrStdout(), "- nvim plugin: %s (%s)\n", result.Action, result.PluginPath)
			if result.SpecPath != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "- lazy.nvim spec: installed (%s)\n", result.SpecPath)
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Open Neovim in a Git repo to show the Markdown summary, or run :PreviouslyOn.")
			return nil
		},
	}
	cmd.Flags().StringVar(&strategy, "strategy", "", "Neovim package strategy: native or lazy")
	return cmd
}

func resolveNvimStrategy(cmd *cobra.Command, value string) (nvim.Strategy, error) {
	value = strings.TrimSpace(strings.ToLower(value))
	if value != "" {
		return nvim.Strategy(value), nil
	}
	result, err := nvimselect.Run()
	if err != nil {
		return "", err
	}
	if !result.Selected {
		return "", fmt.Errorf("Neovim install cancelled")
	}
	return result.Strategy, nil
}

func nvimDataHome() string {
	if value := strings.TrimSpace(os.Getenv("XDG_DATA_HOME")); value != "" {
		return value
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".local", "share")
	}
	return "."
}

func nvimConfigHome() string {
	if value := strings.TrimSpace(os.Getenv("XDG_CONFIG_HOME")); value != "" {
		return value
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".config")
	}
	return "."
}

func gitRoot() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("not inside a Git repository: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return strings.TrimSpace(stdout.String()), nil
}
