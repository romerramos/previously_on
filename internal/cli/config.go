package cli

import (
	"encoding/json"
	"fmt"

	"github.com/romerramos/previously_on/internal/config"
	"github.com/spf13/cobra"
)

func configCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "config",
		Short: "Show effective configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}

			data, err := json.MarshalIndent(cfg, "", "  ")
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), string(data))
			return nil
		},
	}
}
