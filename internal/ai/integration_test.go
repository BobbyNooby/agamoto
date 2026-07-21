package ai

import (
	"testing"
	"time"

	"github.com/BobbyNooby/agamoto/internal/config"
)

func resolveTestConfig() (apiBase, apiKey, model string) {
	base := config.Defaults()
	if cfg, err := config.Load(); err == nil {
		base = config.Merge(base, cfg)
	}
	merged := config.Merge(base, config.FromEnv())
	return merged.APIBase, merged.APIKey, merged.Model
}

func TestRealPingOpenRouter(t *testing.T) {
	apiBase, apiKey, model := resolveTestConfig()
	if apiKey == "" {
		t.Skip("no API key found in env or config file, skipping real API test")
	}

	t.Logf("Pinging %s with model %s", apiBase, model)
	client := NewClient(apiBase, apiKey, model, 10*time.Second)
	if err := client.Ping(); err != nil {
		t.Fatalf("Real ping failed: %v", err)
	}
}

func TestRealChatStreamOpenRouter(t *testing.T) {
	apiBase, apiKey, model := resolveTestConfig()
	if apiKey == "" {
		t.Skip("no API key found in env or config file, skipping real API test")
	}

	t.Logf("ChatStream with %s using model %s", apiBase, model)
	client := NewClient(apiBase, apiKey, model, 15*time.Second)
	resp, err := client.ChatStream("Respond with only the word hello", "Say hello", func(token string) {
		_ = token
	})
	if err != nil {
		t.Fatalf("Real chat failed: %v", err)
	}
	t.Logf("Response: %s", resp)
}
