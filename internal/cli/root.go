package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "previously-on",
	Short: "Brief yourself on what changed in a Git repo since you last opened it",
	Long: `Previously on... generates a concise repository briefing from local Git history.

It tracks the last seen Git state for each repository under ~/.previously-on and
can produce a Markdown summary for humans or editor integrations.`,
}

func Execute() error {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return err
	}
	return nil
}

func init() {
	rootCmd.AddCommand(statusCmd())
	rootCmd.AddCommand(summaryCmd())
	rootCmd.AddCommand(markSeenCmd())
	rootCmd.AddCommand(resetCmd())
	rootCmd.AddCommand(configCmd())
	rootCmd.AddCommand(connectCmd())
	rootCmd.AddCommand(disconnectCmd())
	rootCmd.AddCommand(modelsCmd())
	rootCmd.AddCommand(installCmd())
	rootCmd.AddCommand(uninstallCmd())
}
