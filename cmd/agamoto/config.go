package main

import (
	"fmt"
	"os"

	"github.com/BobbyNooby/agamoto/internal/config"
	"github.com/spf13/cobra"
)

var (
	cfgAPIKey             string
	cfgAPIBase            string
	cfgModel              string
	cfgNVDAPIKey          string
	cfgWebSearchMaxResults int
)

var configCmd = &cobra.Command{
	Use:     "config",
	Aliases: []string{"c"},
	Short:   "Manage agamoto configuration",
	Long: `View or update agamoto configuration stored in ~/.config/agamoto/config.json.

Configuration precedence (low to high):
  defaults < config file < environment variables < flags

Available settings:
  --api-key                  OpenAI-compatible API key
  --api-base                 OpenAI-compatible base URL
  --model                    Model name (default: deepseek/deepseek-v4-flash)
  --nvd-api-key              NVD API key (optional; higher rate limits)
  --web-search-max-results   Web search results per request (default: 5)`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fileCfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		envCfg := config.FromEnv()
		flagCfg := config.Config{
			APIKey:            cfgAPIKey,
			APIBase:           cfgAPIBase,
			Model:             cfgModel,
			NVDAPIKey:         cfgNVDAPIKey,
			WebSearchMaxResults: cfgWebSearchMaxResults,
		}

		cfg := config.Merge(config.Merge(fileCfg, envCfg), flagCfg)

		if cmd.Flags().NFlag() == 0 {
			fmt.Print(cfg)
			return nil
		}

		if err := cfg.Save(); err != nil {
			return fmt.Errorf("save config: %w", err)
		}

		fmt.Fprintln(os.Stderr, "Configuration saved.")
		fmt.Print(cfg)
		return nil
	},
}

func init() {
	configCmd.Flags().StringVar(&cfgAPIKey, "api-key", "", "OpenAI-compatible API key")
	configCmd.Flags().StringVar(&cfgAPIBase, "api-base", "", "OpenAI-compatible base URL")
	configCmd.Flags().StringVar(&cfgModel, "model", "", "Model name")
	configCmd.Flags().StringVar(&cfgNVDAPIKey, "nvd-api-key", "", "NVD API key (optional; higher rate limits)")
	configCmd.Flags().IntVar(&cfgWebSearchMaxResults, "web-search-max-results", 0, "Web search results per request (1-10)")
	rootCmd.AddCommand(configCmd)
}
