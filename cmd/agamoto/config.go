package main

import (
	"fmt"
	"os"

	"github.com/BobbyNooby/agamoto/internal/config"
	"github.com/spf13/cobra"
)

var (
	cfgAPIKey         string
	cfgAPIBase        string
	cfgModel          string
	cfgNVDAPIKey      string
	cfgMaxResearchPasses int
	cfgMaxURLsPerQuery   int
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage agamoto configuration",
	Long: `View or update agamoto configuration stored in ~/.config/agamoto/config.json.

Configuration precedence (low to high):
  defaults < config file < environment variables < flags

Available settings:
  --api-key             OpenAI-compatible API key
  --api-base            OpenAI-compatible base URL
  --model               Model name (default: deepseek/deepseek-v4-flash)
  --nvd-api-key         NVD API key (optional; higher rate limits)
  --max-research-passes Maximum deep-research passes (default: 3)
  --max-urls-per-query  Maximum URLs to fetch per DDG query (default: 5)`,
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
			MaxResearchPasses: cfgMaxResearchPasses,
			MaxURLsPerQuery:   cfgMaxURLsPerQuery,
		}

		// Apply precedence: defaults < file < env < flags
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
	configCmd.Flags().StringVar(&cfgNVDAPIKey, "nvd-api-key", "", "NVD API key (optional; removes rate limits)")
	configCmd.Flags().IntVar(&cfgMaxResearchPasses, "max-research-passes", 0, "Maximum deep-research passes")
	configCmd.Flags().IntVar(&cfgMaxURLsPerQuery, "max-urls-per-query", 0, "Maximum URLs to fetch per DDG query")
	rootCmd.AddCommand(configCmd)
}
