package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
}

type ChatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

type Client struct {
	APIBase string
	APIKey  string
	Model   string
	HTTP    *http.Client
}

func NewClient(apiBase, apiKey, model string, timeout time.Duration) *Client {
	return &Client{
		APIBase: apiBase,
		APIKey:  apiKey,
		Model:   model,
		HTTP:    &http.Client{Timeout: timeout},
	}
}

func (c *Client) Chat(system, user string) (string, error) {
	req := ChatRequest{
		Model: c.Model,
		Messages: []ChatMessage{
			{Role: "system", Content: system},
			{Role: "user", Content: user},
		},
	}

	payload, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("ai marshal: %w", err)
	}

	httpReq, err := http.NewRequest("POST", c.APIBase+"/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("ai request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if c.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)
	}

	resp, err := c.HTTP.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("ai: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("ai read: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ai: %s: %s", resp.Status, string(body))
	}

	var chatResp ChatResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return "", fmt.Errorf("ai decode: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("ai: no choices in response")
	}

	return chatResp.Choices[0].Message.Content, nil
}

func (c *Client) Ping() error {
	httpReq, err := http.NewRequest("GET", c.APIBase+"/models", nil)
	if err != nil {
		return fmt.Errorf("ping request: %w", err)
	}
	if c.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)
	}

	resp, err := c.HTTP.Do(httpReq)
	if err != nil {
		return fmt.Errorf("ping: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("ping: %s: %s", resp.Status, string(body))
	}

	return nil
}
