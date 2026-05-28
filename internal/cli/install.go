package cli

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/romerramos/previously_on/internal/hooks"
	"github.com/spf13/cobra"
)

func installCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install Previously on integrations",
	}
	cmd.AddCommand(installGitHookCmd())
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
