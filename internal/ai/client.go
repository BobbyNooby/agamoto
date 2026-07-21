package ai

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatRequest struct {
	Model            string        `json:"model"`
	Messages         []ChatMessage `json:"messages"`
	MaxTokens        int           `json:"max_tokens,omitempty"`
	Plugins          []Plugin      `json:"plugins,omitempty"`
	FrequencyPenalty float64       `json:"frequency_penalty,omitempty"`
	PresencePenalty  float64       `json:"presence_penalty,omitempty"`
}

type Plugin struct {
	ID         string `json:"id"`
	MaxResults int    `json:"max_results,omitempty"`
}

type StreamChunk struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
	} `json:"choices"`
}

type Client struct {
	APIBase             string
	APIKey              string
	Model               string
	HTTP                *http.Client
	WebSearchMaxResults int
	Debug               bool
}

func NewClient(apiBase, apiKey, model string, timeout time.Duration) *Client {
	return &Client{
		APIBase: apiBase,
		APIKey:  apiKey,
		Model:   model,
		HTTP:    &http.Client{Timeout: timeout},
	}
}

// marshalNoEscape marshals v to JSON without escaping HTML characters
// (<, >, &) so debug output and payloads remain readable.
func marshalNoEscape(v interface{}) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	b := buf.Bytes()
	if len(b) > 0 && b[len(b)-1] == '\n' {
		b = b[:len(b)-1]
	}
	return b, nil
}

// buildRequest creates the common ChatRequest used by both streaming and
// non-streaming calls. It applies penalties and web-search plugin settings.
func (c *Client) buildRequest(system, user string) ChatRequest {
	req := ChatRequest{
		Model: c.Model,
		Messages: []ChatMessage{
			{Role: "system", Content: system},
			{Role: "user", Content: user},
		},
		FrequencyPenalty: 0.2,
		PresencePenalty:  0.2,
	}
	if c.WebSearchMaxResults > 0 {
		req.Plugins = []Plugin{{ID: "web", MaxResults: c.WebSearchMaxResults}}
	}
	return req
}

// post performs the HTTP POST to the chat completions endpoint and returns
// the response body for the caller to handle.
func (c *Client) post(payload []byte, stream bool) (*http.Response, error) {
	httpReq, err := http.NewRequest("POST", c.APIBase+"/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("ai request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if stream {
		httpReq.Header.Set("Accept", "text/event-stream")
	}
	if c.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)
	}

	resp, err := c.HTTP.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("ai: %w", err)
	}
	return resp, nil
}

func (c *Client) ChatStream(system, user string, onToken func(string)) (string, error) {
	req := c.buildRequest(system, user)

	payloadMap := map[string]interface{}{
		"model":             req.Model,
		"messages":          req.Messages,
		"stream":            true,
		"frequency_penalty": req.FrequencyPenalty,
		"presence_penalty":  req.PresencePenalty,
	}
	if req.MaxTokens > 0 {
		payloadMap["max_tokens"] = req.MaxTokens
	}
	if len(req.Plugins) > 0 {
		payloadMap["plugins"] = req.Plugins
	}

	payload, err := marshalNoEscape(payloadMap)
	if err != nil {
		return "", fmt.Errorf("ai marshal: %w", err)
	}
	if c.Debug {
		fmt.Fprintf(os.Stderr, "[debug] AI stream request payload (%d bytes): %s\n", len(payload), string(payload))
	}

	resp, err := c.post(payload, true)
	if err != nil {
		return "", err
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
			continue
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
