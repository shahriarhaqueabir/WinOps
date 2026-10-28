package aiops

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func testOllamaClient(t *testing.T, srv *httptest.Server) *OllamaClient {
	t.Helper()
	t.Setenv("OLLAMA_HOST", srv.URL)
	t.Setenv("OLLAMA_MODEL", "universalops")
	return NewOllamaClient()
}

func TestNormalizeBaseURL(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"", defaultOllamaURL},
		{"127.0.0.1:11434", "http://127.0.0.1:11434"},
		{"http://127.0.0.1:11434/", "http://127.0.0.1:11434"},
		{"https://example.test/ollama/", "https://example.test/ollama"},
		{"0.0.0.0:11434", "http://127.0.0.1:11434"},
		{"http://[::]:11434", "http://127.0.0.1:11434"},
		{"ftp://example.test", defaultOllamaURL},
		{"://bad", defaultOllamaURL},
	}
	for _, tt := range tests {
		if got := normalizeBaseURL(tt.in); got != tt.want {
			t.Errorf("normalizeBaseURL(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestOllamaChat(t *testing.T) {
	var got ollamaChatRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/chat" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Errorf("decode request: %v", err)
		}
		_ = json.NewEncoder(w).Encode(ollamaChatResponse{
			Message: ollamaChatMessage{Role: "assistant", Content: "hello"},
			Done:    true,
		})
	}))
	defer srv.Close()

	client := testOllamaClient(t, srv)
	out, err := client.Chat(context.Background(), []ChatMessage{{Role: "user", Content: "hi"}}, "")
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if out != "hello" {
		t.Fatalf("Chat = %q, want hello", out)
	}
	if got.Model != "universalops" || got.Stream {
		t.Errorf("request model/stream = %q/%v, want universalops/false", got.Model, got.Stream)
	}
	if len(got.Messages) != 1 || got.Messages[0].Content != "hi" {
		t.Errorf("request messages = %#v", got.Messages)
	}
}

func TestOllamaChatStream(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/chat" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/x-ndjson")
		enc := json.NewEncoder(w)
		_ = enc.Encode(ollamaChatResponse{Message: ollamaChatMessage{Role: "assistant", Content: "Hel"}})
		_ = enc.Encode(ollamaChatResponse{Message: ollamaChatMessage{Role: "assistant", Content: "lo"}})
		_ = enc.Encode(ollamaChatResponse{Done: true})
	}))
	defer srv.Close()

	client := testOllamaClient(t, srv)
	var tokens []string
	out, err := client.ChatStream(context.Background(), []ChatMessage{{Role: "user", Content: "hi"}}, "", func(tok string) {
		tokens = append(tokens, tok)
	})
	if err != nil {
		t.Fatalf("ChatStream: %v", err)
	}
	if out != "Hello" || strings.Join(tokens, "") != "Hello" {
		t.Fatalf("ChatStream = %q tokens=%#v, want Hello", out, tokens)
	}
}

func TestOllamaPullModelProgress(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/pull" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		_ = json.NewDecoder(r.Body).Decode(&got)
		w.Header().Set("Content-Type", "application/x-ndjson")
		enc := json.NewEncoder(w)
		_ = enc.Encode(ProgressResponse{Status: "pulling manifest"})
		_ = enc.Encode(ProgressResponse{Status: "downloading", Total: 100, Completed: 50})
		_ = enc.Encode(ProgressResponse{Status: "success"})
	}))
	defer srv.Close()

	client := testOllamaClient(t, srv)
	var progress []ProgressResponse
	if err := client.PullModel(context.Background(), "llama3", func(p ProgressResponse) error {
		progress = append(progress, p)
		return nil
	}); err != nil {
		t.Fatalf("PullModel: %v", err)
	}
	if len(progress) != 3 || progress[0].Status != "pulling manifest" || progress[2].Status != "success" {
		t.Fatalf("unexpected progress: %#v", progress)
	}
	if got["model"] != "llama3" || got["stream"] != true {
		t.Errorf("request body = %#v", got)
	}
}

func TestOllamaPullModelError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-ndjson")
		_ = json.NewEncoder(w).Encode(ProgressResponse{Error: "pull failed"})
	}))
	defer srv.Close()

	client := testOllamaClient(t, srv)
	err := client.PullModel(context.Background(), "llama3", nil)
	if err == nil || !strings.Contains(err.Error(), "pull failed") {
		t.Fatalf("PullModel error = %v, want pull failed", err)
	}
}

func TestOllamaCreateModel(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/create" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		_ = json.NewDecoder(r.Body).Decode(&got)
		w.Header().Set("Content-Type", "application/x-ndjson")
		_ = json.NewEncoder(w).Encode(ProgressResponse{Status: "success"})
	}))
	defer srv.Close()

	client := testOllamaClient(t, srv)
	if err := client.CreateModel(context.Background(), "persona", "base:latest", "system prompt", map[string]any{"temperature": 0.2}, nil); err != nil {
		t.Fatalf("CreateModel: %v", err)
	}
	if got["model"] != "persona" || got["from"] != "base:latest" || got["system"] != "system prompt" {
		t.Errorf("request body = %#v", got)
	}
	params, ok := got["parameters"].(map[string]any)
	if !ok || params["temperature"] != 0.2 {
		t.Errorf("parameters = %#v", got["parameters"])
	}
}

func TestOllamaDeleteModel(t *testing.T) {
	var got map[string]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/api/delete" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		_ = json.NewDecoder(r.Body).Decode(&got)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := testOllamaClient(t, srv)
	if err := client.DeleteModel(context.Background(), "old-model"); err != nil {
		t.Fatalf("DeleteModel: %v", err)
	}
	if got["model"] != "old-model" {
		t.Errorf("request body = %#v", got)
	}
}

func TestOllamaCheckStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/tags":
			_ = json.NewEncoder(w).Encode(ollamaListResponse{Models: []struct {
				Name string `json:"name"`
			}{
				{Name: "qwen2.5:7b"},
				{Name: "universalops:latest"},
			}})
		case "/api/version":
			_ = json.NewEncoder(w).Encode(ollamaVersionResponse{Version: "0.40.11"})
		default:
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	client := testOllamaClient(t, srv)
	status, err := client.CheckStatus(context.Background())
	if err != nil {
		t.Fatalf("CheckStatus: %v", err)
	}
	if !status.Available || status.Version != "0.40.11" {
		t.Fatalf("status = %#v", status)
	}
	if status.Model != "universalops:latest" {
		t.Errorf("model = %q, want universalops:latest", status.Model)
	}
	if len(status.AvailableModels) != 2 {
		t.Errorf("models = %#v", status.AvailableModels)
	}
}

func TestOllamaHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "boom"})
	}))
	defer srv.Close()

	client := testOllamaClient(t, srv)
	_, err := client.Chat(context.Background(), []ChatMessage{{Role: "user", Content: "hi"}}, "")
	if err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("Chat error = %v, want boom", err)
	}
}

func TestTruncateHistory(t *testing.T) {
	messages := []ChatMessage{
		{Role: "user", Content: "one two"},
		{Role: "assistant", Content: "three four"},
		{Role: "user", Content: "five six"},
	}
	got := TruncateHistory(messages, 4)
	if len(got) != 2 || got[0].Content != "three four" || got[1].Content != "five six" {
		t.Fatalf("TruncateHistory = %#v", got)
	}
	if len(TruncateHistory(messages, 0)) != 3 {
		t.Fatalf("default limit should keep all messages")
	}
}
