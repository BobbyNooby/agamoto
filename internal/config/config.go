package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const (
	DefaultAPIBase = "https://openrouter.ai/api/v1"
	DefaultModel   = "deepseek/deepseek-v4-flash"
)

type Config struct {
	APIKey  string `json:"api_key"`
	APIBase string `json:"api_base"`
	Model   string `json:"model"`
}

func Defaults() Config {
	return Config{
		APIBase: DefaultAPIBase,
		Model:   DefaultModel,
	}
}

func Path() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("home dir: %w", err)
	}
	return filepath.Join(home, ".config", "agamoto", "config.json"), nil
}

func Load() (Config, error) {
	cfg := Defaults()

	path, err := Path()
	if err != nil {
		return cfg, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, fmt.Errorf("read config: %w", err)
	}

	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("parse config: %w", err)
	}

	return cfg, nil
}

func (c Config) Save() error {
	path, err := Path()
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	return nil
}

func FromEnv() Config {
	return Config{
		APIKey:  os.Getenv("OPENAI_API_KEY"),
		APIBase: os.Getenv("OPENAI_BASE_URL"),
		Model:   os.Getenv("AI_MODEL"),
	}
}

func Merge(base, overlay Config) Config {
	if overlay.APIKey != "" {
		base.APIKey = overlay.APIKey
	}
	if overlay.APIBase != "" {
		base.APIBase = overlay.APIBase
	}
	if overlay.Model != "" {
		base.Model = overlay.Model
	}
	return base
}

func (c Config) String() string {
	keyDisplay := "(not set)"
	if c.APIKey != "" {
		keyDisplay = "(set)"
	}
	return fmt.Sprintf("api_key:  %s\napi_base: %s\nmodel:    %s\n", keyDisplay, c.APIBase, c.Model)
}
