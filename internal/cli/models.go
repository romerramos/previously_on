package cli

import (
	"fmt"
	"strings"

	"github.com/romerramos/previously_on/internal/ai"
	"github.com/romerramos/previously_on/internal/config"
	"github.com/romerramos/previously_on/internal/modelselect"
	"github.com/spf13/cobra"
)

func modelsCmd() *cobra.Command {
	var providerID string
	var setModel string

	cmd := &cobra.Command{
		Use:   "models",
		Short: "Show or set the model name for an AI provider",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			if providerID == "" {
				providerID = cfg.AI.Provider
			}
			provider, ok := ai.FindProvider(providerID)
			if !ok {
				return fmt.Errorf("unknown provider %q", providerID)
			}

			current := currentModel(cfg, provider)
			if strings.TrimSpace(setModel) == "" {
				result, err := modelselect.Run(provider, current)
				if err != nil {
					return err
				}
				if !result.Saved {
					return nil
				}
				setModel = result.Model
			}

			setModel = strings.TrimSpace(setModel)
			if setModel == "" {
				return fmt.Errorf("model name cannot be empty")
			}
			if cfg.AI.Models == nil {
				cfg.AI.Models = map[string]string{}
			}
			cfg.AI.Models[provider.ID] = setModel
			if cfg.AI.Provider == provider.ID {
				cfg.AI.Model = setModel
			}
			if err := config.Save(cfg); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Saved model %q for %s.\n", setModel, provider.Name)
			fmt.Fprintln(cmd.OutOrStdout(), "Double-check that this model exists for your account/provider.")
			return nil
		},
	}

	cmd.Flags().StringVar(&providerID, "provider", "", "provider id; defaults to the selected provider")
	cmd.Flags().StringVar(&setModel, "set", "", "set model name without opening the interactive prompt")
	return cmd
}

func currentModel(cfg config.Config, provider ai.ProviderInfo) string {
	if cfg.AI.Models != nil && cfg.AI.Models[provider.ID] != "" {
		return cfg.AI.Models[provider.ID]
	}
	if cfg.AI.Provider == provider.ID && cfg.AI.Model != "" {
		return cfg.AI.Model
	}
	return provider.DefaultModel
}
