package cli

import (
	"fmt"
	"os"

	"github.com/romerramos/previously_on/internal/ai"
	"github.com/romerramos/previously_on/internal/config"
	"github.com/romerramos/previously_on/internal/secrets"
	"github.com/spf13/cobra"
)

func disconnectCmd() *cobra.Command {
	var providerID string
	var all bool

	cmd := &cobra.Command{
		Use:   "disconnect",
		Short: "Remove stored AI provider credentials",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}

			store := secrets.KeyringStore{}
			if all {
				for _, provider := range ai.Providers() {
					if provider.RequiresKey {
						if err := store.Delete(provider.ID); err != nil {
							return err
						}
						fmt.Fprintf(cmd.OutOrStdout(), "Removed %s credentials from OS keychain.\n", provider.Name)
						warnEnv(cmd, provider)
					}
				}
				cfg.AI.Enabled = false
				if err := config.Save(cfg); err != nil {
					return err
				}
				fmt.Fprintln(cmd.OutOrStdout(), "AI disabled. Provider/model preferences were kept.")
				return nil
			}

			if providerID == "" {
				providerID = cfg.AI.Provider
			}
			provider, ok := ai.FindProvider(providerID)
			if !ok {
				return fmt.Errorf("unknown provider %q", providerID)
			}
			if provider.RequiresKey {
				if err := store.Delete(provider.ID); err != nil {
					return err
				}
				fmt.Fprintf(cmd.OutOrStdout(), "Removed %s credentials from OS keychain.\n", provider.Name)
				warnEnv(cmd, provider)
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "%s does not use stored API credentials.\n", provider.Name)
			}

			if cfg.AI.Provider == provider.ID {
				cfg.AI.Enabled = false
				if err := config.Save(cfg); err != nil {
					return err
				}
				fmt.Fprintf(cmd.OutOrStdout(), "AI disabled because %s was the selected provider.\n", provider.Name)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&providerID, "provider", "", "provider id to disconnect; defaults to the selected provider")
	cmd.Flags().BoolVar(&all, "all", false, "remove credentials for all providers and disable AI")
	return cmd
}

func warnEnv(cmd *cobra.Command, provider ai.ProviderInfo) {
	if provider.EnvVar != "" && os.Getenv(provider.EnvVar) != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "%s is still set, so %s may remain available through the environment.\n", provider.EnvVar, provider.Name)
	}
}
