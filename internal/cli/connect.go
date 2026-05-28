package cli

import (
	"fmt"

	"github.com/romerramos/previously_on/internal/ai"
	"github.com/romerramos/previously_on/internal/config"
	"github.com/romerramos/previously_on/internal/connect"
	"github.com/romerramos/previously_on/internal/secrets"
	"github.com/spf13/cobra"
)

func connectCmd() *cobra.Command {
	var providerID string
	var list bool

	cmd := &cobra.Command{
		Use:   "connect",
		Short: "Connect an AI provider and store its API key securely",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}

			if list {
				for _, provider := range ai.Providers() {
					mark := " "
					if cfg.AI.Enabled && provider.ID == cfg.AI.Provider {
						mark = "✓"
					}
					model := provider.DefaultModel
					if cfg.AI.Models != nil && cfg.AI.Models[provider.ID] != "" {
						model = cfg.AI.Models[provider.ID]
					}
					fmt.Fprintf(cmd.OutOrStdout(), "%s %-10s %s (model: %s)\n", mark, provider.ID, provider.Description, model)
				}
				return nil
			}
			if providerID != "" {
				if _, ok := ai.FindProvider(providerID); !ok {
					return fmt.Errorf("unknown provider %q", providerID)
				}
			}

			result, err := connect.Run(cfg, secrets.KeyringStore{}, providerID)
			if err != nil {
				return err
			}
			if result.Saved {
				if result.Provider.RequiresKey {
					fmt.Fprintf(cmd.OutOrStdout(), "Connected %s using model %s. API key stored in OS keychain.\n", result.Provider.Name, result.Model)
				} else {
					fmt.Fprintf(cmd.OutOrStdout(), "Connected %s using model %s.\n", result.Provider.Name, result.Model)
				}
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&providerID, "provider", "", "preselect provider by id")
	cmd.Flags().BoolVar(&list, "list", false, "list supported providers")
	return cmd
}
