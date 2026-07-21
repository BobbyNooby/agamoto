package ai

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestPing(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/models" {
			t.Errorf("expected /models, got %s", r.URL.Path)
		}
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-key" {
			t.Errorf("expected Bearer test-key, got %s", auth)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"data": []string{}})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", "test-model", 5*time.Second)
	if err := client.Ping(); err != nil {
		t.Fatalf("Ping: %v", err)
	}
}

func TestPingUnauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
	}))
	defer server.Close()

	client := NewClient(server.URL, "bad-key", "test-model", 5*time.Second)
	if err := client.Ping(); err == nil {
		t.Fatal("expected error for unauthorized")
	}
}

func TestChatStream(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/chat/completions" {
			t.Errorf("expected /chat/completions, got %s", r.URL.Path)
		}
		if r.Header.Get("Accept") != "text/event-stream" {
			t.Errorf("expected text/event-stream accept header, got %s", r.Header.Get("Accept"))
		}

		var req ChatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if len(req.Messages) != 2 {
			t.Fatalf("expected 2 messages, got %d", len(req.Messages))
		}
		if req.Messages[0].Role != "system" {
			t.Errorf("expected system role, got %s", req.Messages[0].Role)
		}
		if req.Messages[1].Role != "user" {
			t.Errorf("expected user role, got %s", req.Messages[1].Role)
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"Test \"}}]}\n\n"))
		w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"analysis\"}}]}\n\n"))
		w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", "test-model", 5*time.Second)
	var tokens []string
	response, err := client.ChatStream("system prompt", "user message", func(token string) {
		tokens = append(tokens, token)
	})
	if err != nil {
		t.Fatalf("ChatStream: %v", err)
	}

	if response != "Test analysis" {
		t.Errorf("expected 'Test analysis', got '%s'", response)
	}
	if len(tokens) != 2 || tokens[0] != "Test " || tokens[1] != "analysis" {
		t.Errorf("unexpected tokens: %v", tokens)
	}
}

func TestChatStreamError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("boom"))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", "test-model", 5*time.Second)
	_, err := client.ChatStream("system", "user", func(string) {})
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
}

func TestMarshalNoEscape(t *testing.T) {
	v := map[string]string{"content": "<tag> & value >"}
	b, err := marshalNoEscape(v)
	if err != nil {
		t.Fatalf("marshalNoEscape: %v", err)
	}
	if strings.Contains(string(b), `\u003c`) || strings.Contains(string(b), `\u003e`) || strings.Contains(string(b), `\u0026`) {
		t.Fatalf("expected no HTML unicode escapes, got %s", string(b))
	}
	if !strings.Contains(string(b), "<tag>") {
		t.Fatalf("expected literal <tag>, got %s", string(b))
	}
}
