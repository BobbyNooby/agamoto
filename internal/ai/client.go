package ai

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
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

type StreamChunk struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
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

func (c *Client) ChatStream(system, user string, onToken func(string)) (string, error) {
	req := ChatRequest{
		Model: c.Model,
		Messages: []ChatMessage{
			{Role: "system", Content: system},
			{Role: "user", Content: user},
		},
		MaxTokens: 0,
	}

	// Add stream flag via custom JSON marshaling or set after marshal
	// We'll marshal to map and add stream field
	payloadMap := map[string]interface{}{
		"model":    req.Model,
		"messages": req.Messages,
		"stream":   true,
	}
	if req.MaxTokens > 0 {
		payloadMap["max_tokens"] = req.MaxTokens
	}

	payload, err := json.Marshal(payloadMap)
	if err != nil {
		return "", fmt.Errorf("ai marshal: %w", err)
	}

	httpReq, err := http.NewRequest("POST", c.APIBase+"/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("ai request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")
	if c.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)
	}

	resp, err := c.HTTP.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("ai: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("ai: %s: %s", resp.Status, string(body))
	}

	var full strings.Builder
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		var chunk StreamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue // skip malformed chunks
		}
		if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
			token := chunk.Choices[0].Delta.Content
			full.WriteString(token)
			onToken(token)
		}
	}

	if err := scanner.Err(); err != nil {
		return full.String(), fmt.Errorf("ai stream read: %w", err)
	}

	return full.String(), nil
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
