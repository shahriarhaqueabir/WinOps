package aiops

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// ChatMessage represents a message in a chat conversation.
type ChatMessage struct {
	Role    string `json:"role"`    // "user", "assistant", "system"
	Content string `json:"content"` // message content
}

// ProgressResponse is a single update emitted by long-running Ollama
// operations (pull/create). It mirrors the relevant fields of Ollama's NDJSON
// progress stream.
type ProgressResponse struct {
	Status    string `json:"status"`
	Digest    string `json:"digest,omitempty"`
	Total     int64  `json:"total,omitempty"`
	Completed int64  `json:"completed,omitempty"`
	Error     string `json:"error,omitempty"`
}

// OllamaStatus represents the availability and configuration of Ollama.
type OllamaStatus struct {
	Available       bool
	Model           string
	Version         string
	AvailableModels []string
}

const (
	defaultOllamaURL   = "http://127.0.0.1:11434"
	defaultOllamaModel = "universalops"
	defaultHttpTimeout = 120 * time.Second

	// longRunningHttpTimeout is used for model create/pull operations, which
	// can take many minutes when Ollama must first download a large base model
	// (e.g. Qwythos-9B ~6.85 GiB). The 120s chat timeout is far too short for
	// these streaming requests and caused "context deadline exceeded" failures
	// during persona deployment.
	//
	// A generous-but-finite bound is used (rather than 0 = no timeout) so that
	// a stalled download or unresponsive daemon cannot hang the Wails binding
	// call forever. 45 minutes comfortably covers even a slow multi-GiB pull
	// while still guaranteeing the operation eventually returns.
	longRunningHttpTimeout = 45 * time.Minute

	// maxJSONResponseBytes bounds non-streaming JSON responses so a malicious
	// or malfunctioning local daemon cannot exhaust memory.
	maxJSONResponseBytes = 8 << 20

	// maxErrorResponseBytes bounds error bodies read from Ollama.
	maxErrorResponseBytes = 1 << 20

	// maxStreamBytes bounds the total NDJSON progress/chat stream size.
	maxStreamBytes = 64 << 20
)

func getOllamaURL() string {
	if host := os.Getenv("OLLAMA_HOST"); host != "" {
		return host
	}
	return defaultOllamaURL
}

func getOllamaModel() string {
	if model := os.Getenv("OLLAMA_MODEL"); model != "" {
		return model
	}
	return defaultOllamaModel
}

func newHTTPClient(timeout time.Duration) *http.Client {
	return &http.Client{Timeout: timeout}
}

// normalizeBaseURL accepts either a full URL or Ollama's host:port form and
// returns a URL suitable for appending API paths. Wildcard listen addresses are
// rewritten to loopback so client calls always target the local daemon.
func normalizeBaseURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return defaultOllamaURL
	}
	if !strings.Contains(raw, "://") {
		raw = "http://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return defaultOllamaURL
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return defaultOllamaURL
	}
	if host := u.Hostname(); host == "0.0.0.0" || host == "::" {
		loopback := "127.0.0.1"
		if port := u.Port(); port != "" {
			loopback = net.JoinHostPort(loopback, port)
		}
		u.Host = loopback
	}
	u.Path = strings.TrimRight(u.Path, "/")
	return u.String()
}

type ollamaListResponse struct {
	Models []struct {
		Name string `json:"name"`
	} `json:"models"`
	Error string `json:"error,omitempty"`
}

type ollamaVersionResponse struct {
	Version string `json:"version"`
	Error   string `json:"error,omitempty"`
}

type ollamaChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ollamaChatRequest struct {
	Model    string              `json:"model"`
	Messages []ollamaChatMessage `json:"messages"`
	Stream   bool                `json:"stream"`
}

type ollamaChatResponse struct {
	Message ollamaChatMessage `json:"message"`
	Done    bool              `json:"done"`
	Error   string            `json:"error,omitempty"`
}

// ── Global Compatibility Wrappers ───────────────────────────────────────────

var (
	defaultClientMu sync.Mutex
	defaultClient   *OllamaClient
)

func getDefaultClient() *OllamaClient {
	defaultClientMu.Lock()
	defer defaultClientMu.Unlock()

	if defaultClient == nil {
		defaultClient = NewOllamaClient()
	}
	return defaultClient
}

// ResetDefaultClient forces the global client to be re-initialized from environment.
// Primarily used for testing.
func ResetDefaultClient() {
	defaultClientMu.Lock()
	defer defaultClientMu.Unlock()
	defaultClient = nil
}

// Chat sends a chat request using the default client and context.Background().
func Chat(messages []ChatMessage) (string, error) {
	return ChatWithContext(context.Background(), messages)
}

// ChatWithContext sends a chat request using the default client and the provided context.
func ChatWithContext(ctx context.Context, messages []ChatMessage) (string, error) {
	return getDefaultClient().Chat(ctx, messages, "")
}

// ChatWithModel sends a chat request using a specific model override.
func ChatWithModel(messages []ChatMessage, modelOverride string) (string, error) {
	return ChatWithModelAndContext(context.Background(), messages, modelOverride)
}

// ChatWithModelAndContext sends a chat request using a specific model override and context.
func ChatWithModelAndContext(ctx context.Context, messages []ChatMessage, modelOverride string) (string, error) {
	return getDefaultClient().Chat(ctx, messages, modelOverride)
}

// ChatStreamWithModelAndContext sends a streaming chat request using a specific model
// override and context. Each token is delivered via onToken.
func ChatStreamWithModelAndContext(ctx context.Context, messages []ChatMessage, modelOverride string, onToken func(string)) (string, error) {
	return getDefaultClient().ChatStream(ctx, messages, modelOverride, onToken)
}

// SetModel updates the model for the default client.
func SetModel(model string) {
	getDefaultClient().SetModel(model)
}

// CheckOllama checks the status using the default client.
func CheckOllama() (*OllamaStatus, error) {
	return CheckOllamaWithContext(context.Background())
}

// CheckOllamaWithContext checks the status using the default client and context.
func CheckOllamaWithContext(ctx context.Context) (*OllamaStatus, error) {
	return getDefaultClient().CheckStatus(ctx)
}

// PullModel downloads a model using the default client.
func PullModel(name string, onProgress func(ProgressResponse) error) error {
	return PullModelWithContext(context.Background(), name, onProgress)
}

// PullModelWithContext downloads a model using the default client and context.
func PullModelWithContext(ctx context.Context, name string, onProgress func(ProgressResponse) error) error {
	return getDefaultClient().PullModel(ctx, name, onProgress)
}

// CreateModel creates a model using the default client.
func CreateModel(name, from, system string, parameters map[string]any) error {
	return CreateModelWithContext(context.Background(), name, from, system, parameters, nil)
}

// CreateModelWithContext creates a model using the default client and context.
// onProgress, if non-nil, is invoked with each progress update from Ollama
// (useful for surfacing base-model pull progress during persona creation).
func CreateModelWithContext(ctx context.Context, name, from, system string, parameters map[string]any, onProgress func(ProgressResponse) error) error {
	return getDefaultClient().CreateModel(ctx, name, from, system, parameters, onProgress)
}

// DeleteModel removes a model using the default client.
func DeleteModel(name string) error {
	return DeleteModelWithContext(context.Background(), name)
}

// DeleteModelWithContext removes a model using the default client and context.
func DeleteModelWithContext(ctx context.Context, name string) error {
	return getDefaultClient().DeleteModel(ctx, name)
}

// ── OllamaClient Implementation ─────────────────────────────────────────────

// OllamaClient is a minimal Ollama HTTP API client. It intentionally implements
// only the endpoints UniversalOps uses, avoiding a dependency on Ollama's full
// server module and its transitive packages.
type OllamaClient struct {
	httpClient     *http.Client
	longHTTPClient *http.Client
	baseURL        string
	modelMu        sync.RWMutex
	effectiveModel string
}

func NewOllamaClient() *OllamaClient {
	return &OllamaClient{
		httpClient:     newHTTPClient(defaultHttpTimeout),
		longHTTPClient: newHTTPClient(longRunningHttpTimeout),
		baseURL:        normalizeBaseURL(getOllamaURL()),
		effectiveModel: getOllamaModel(),
	}
}

func (c *OllamaClient) SetModel(model string) {
	c.modelMu.Lock()
	defer c.modelMu.Unlock()
	c.effectiveModel = model
}

func (c *OllamaClient) Name() string { return "ollama" }

func (c *OllamaClient) GetModel() string {
	c.modelMu.RLock()
	defer c.modelMu.RUnlock()
	return c.effectiveModel
}

func (c *OllamaClient) endpoint(path string) string {
	return strings.TrimRight(c.baseURL, "/") + path
}

func (c *OllamaClient) doJSON(ctx context.Context, client *http.Client, method, path string, body, out any) error {
	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("ollama request encode failed: %w", err)
		}
		reader = bytes.NewReader(payload)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.endpoint(path), reader)
	if err != nil {
		return fmt.Errorf("ollama request failed: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("ollama request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return decodeOllamaError(resp)
	}
	if out == nil {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, maxErrorResponseBytes))
		return nil
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxJSONResponseBytes)).Decode(out); err != nil {
		return fmt.Errorf("ollama response decode failed: %w", err)
	}
	return nil
}

func (c *OllamaClient) doStream(ctx context.Context, client *http.Client, path string, body any, onItem func(json.RawMessage) error) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("ollama request encode failed: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint(path), bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("ollama request failed: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("ollama request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return decodeOllamaError(resp)
	}

	decoder := json.NewDecoder(io.LimitReader(resp.Body, maxStreamBytes))
	for {
		var item json.RawMessage
		if err := decoder.Decode(&item); err != nil {
			if err == io.EOF {
				return nil
			}
			return fmt.Errorf("ollama stream decode failed: %w", err)
		}
		if onItem != nil {
			if err := onItem(item); err != nil {
				return err
			}
		}
	}
}

func decodeOllamaError(resp *http.Response) error {
	body, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrorResponseBytes))
	var payload struct {
		Error string `json:"error"`
	}
	if json.Unmarshal(body, &payload) == nil && strings.TrimSpace(payload.Error) != "" {
		return fmt.Errorf("ollama: %s", strings.TrimSpace(payload.Error))
	}
	if text := strings.TrimSpace(string(body)); text != "" {
		return fmt.Errorf("ollama returned %s: %s", resp.Status, text)
	}
	return fmt.Errorf("ollama returned %s", resp.Status)
}

// CheckOllamaBinary returns true if the ollama binary is found in the system PATH.
func CheckOllamaBinary() bool {
	_, err := exec.LookPath("ollama")
	return err == nil
}

// PullModel downloads a model from the Ollama library.
// Uses a long-running client because pulling a large base model can take
// minutes and must not be cut off by the 120s chat timeout.
func (c *OllamaClient) PullModel(ctx context.Context, name string, onProgress func(ProgressResponse) error) error {
	body := map[string]any{"model": name, "stream": true}
	return c.doStream(ctx, c.longHTTPClient, "/api/pull", body, func(raw json.RawMessage) error {
		var progress ProgressResponse
		if err := json.Unmarshal(raw, &progress); err != nil {
			return fmt.Errorf("ollama pull progress decode failed: %w", err)
		}
		if progress.Error != "" {
			return fmt.Errorf("ollama: %s", progress.Error)
		}
		if onProgress != nil {
			return onProgress(progress)
		}
		return nil
	})
}

// DeleteModel removes a local model.
func (c *OllamaClient) DeleteModel(ctx context.Context, name string) error {
	return c.doJSON(ctx, c.httpClient, http.MethodDelete, "/api/delete", map[string]string{"model": name}, nil)
}

// CreateModel creates a new model from structured parameters.
// Uses a long-running client because Ollama may need to pull the base model
// first (e.g. Qwythos-9B ~6.85 GiB), which exceeds the 120s chat timeout.
// onProgress, when non-nil, receives each progress update from the create
// stream so the frontend can show live download/build progress.
func (c *OllamaClient) CreateModel(ctx context.Context, name string, from string, system string, parameters map[string]any, onProgress func(ProgressResponse) error) error {
	body := map[string]any{
		"model":      name,
		"from":       from,
		"system":     system,
		"parameters": parameters,
		"stream":     true,
	}
	return c.doStream(ctx, c.longHTTPClient, "/api/create", body, func(raw json.RawMessage) error {
		var progress ProgressResponse
		if err := json.Unmarshal(raw, &progress); err != nil {
			return fmt.Errorf("ollama create progress decode failed: %w", err)
		}
		if progress.Error != "" {
			return fmt.Errorf("ollama: %s", progress.Error)
		}
		if onProgress != nil {
			return onProgress(progress)
		}
		return nil
	})
}

// CheckStatus checks if the Ollama service is available.
func (c *OllamaClient) CheckStatus(ctx context.Context) (*OllamaStatus, error) {
	var listResp ollamaListResponse
	if err := c.doJSON(ctx, c.httpClient, http.MethodGet, "/api/tags", nil, &listResp); err != nil {
		return &OllamaStatus{Available: false}, fmt.Errorf("ollama list failed: %w", err)
	}
	if listResp.Error != "" {
		return &OllamaStatus{Available: false}, fmt.Errorf("ollama list failed: %s", listResp.Error)
	}

	currentModel := c.GetModel()
	availableModels := make([]string, 0, len(listResp.Models))
	found := false
	for _, m := range listResp.Models {
		availableModels = append(availableModels, m.Name)
		if m.Name == currentModel || strings.HasPrefix(m.Name, currentModel+":") {
			found = true
			currentModel = m.Name
		}
	}

	// When the configured model isn't found in Ollama, do NOT silently
	// overwrite effectiveModel with the first available model (e.g., qwen2.5:7b).
	// Keeping the configured value lets the frontend detect the persona is
	// missing and show the "Initialize Persona" action button.
	if found {
		c.SetModel(currentModel)
	}

	version := "detected"
	var versionResp ollamaVersionResponse
	if err := c.doJSON(ctx, c.httpClient, http.MethodGet, "/api/version", nil, &versionResp); err == nil && versionResp.Version != "" {
		version = versionResp.Version
	}

	return &OllamaStatus{
		Available:       true,
		Model:           currentModel,
		Version:         version,
		AvailableModels: availableModels,
	}, nil
}

// Chat sends a chat request to the Ollama API (non-streaming).
func (c *OllamaClient) Chat(ctx context.Context, messages []ChatMessage, modelOverride string) (string, error) {
	model := c.GetModel()
	if modelOverride != "" {
		model = modelOverride
	}

	reqBody := ollamaChatRequest{
		Model:    model,
		Messages: toOllamaMessages(messages),
		Stream:   false,
	}
	var resp ollamaChatResponse
	if err := c.doJSON(ctx, c.httpClient, http.MethodPost, "/api/chat", reqBody, &resp); err != nil {
		return "", fmt.Errorf("ollama chat failed: %w", err)
	}
	if resp.Error != "" {
		return "", fmt.Errorf("ollama chat failed: %s", resp.Error)
	}

	responseText := resp.Message.Content
	if responseText == "" {
		return "", fmt.Errorf("ollama returned empty response")
	}
	return responseText, nil
}

// ChatStream sends a streaming chat request to the Ollama API. Each content
// token is delivered via the onToken callback. The final accumulated response
// is returned along with any error.
func (c *OllamaClient) ChatStream(ctx context.Context, messages []ChatMessage, modelOverride string, onToken func(string)) (string, error) {
	model := c.GetModel()
	if modelOverride != "" {
		model = modelOverride
	}

	reqBody := ollamaChatRequest{
		Model:    model,
		Messages: toOllamaMessages(messages),
		Stream:   true,
	}

	var responseText string
	err := c.doStream(ctx, c.httpClient, "/api/chat", reqBody, func(raw json.RawMessage) error {
		var resp ollamaChatResponse
		if err := json.Unmarshal(raw, &resp); err != nil {
			return fmt.Errorf("ollama chat stream decode failed: %w", err)
		}
		if resp.Error != "" {
			return fmt.Errorf("ollama: %s", resp.Error)
		}
		if resp.Message.Content != "" {
			responseText += resp.Message.Content
			if onToken != nil {
				onToken(resp.Message.Content)
			}
		}
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("ollama chat stream failed: %w", err)
	}
	if responseText == "" {
		return "", fmt.Errorf("ollama returned empty response")
	}
	return responseText, nil
}

func toOllamaMessages(messages []ChatMessage) []ollamaChatMessage {
	out := make([]ollamaChatMessage, len(messages))
	for i, m := range messages {
		out[i] = ollamaChatMessage(m)
	}
	return out
}

// TruncateHistory ensures the chat history doesn't exceed a safe context window.
// Uses a simple word-count heuristic (approx 0.75 words per token).
// Limit is roughly 24k words for a 32k context.
func TruncateHistory(messages []ChatMessage, wordLimit int) []ChatMessage {
	if wordLimit <= 0 {
		wordLimit = 24000
	}

	totalWords := 0
	// Count from newest to oldest
	var keptMessages []ChatMessage
	for i := len(messages) - 1; i >= 0; i-- {
		msg := messages[i]
		words := len(strings.Fields(msg.Content))
		if totalWords+words > wordLimit {
			break
		}
		totalWords += words
		keptMessages = append([]ChatMessage{msg}, keptMessages...)
	}

	return keptMessages
}
