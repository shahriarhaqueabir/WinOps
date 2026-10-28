package aiops

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// OpenAIProvider adapts an OpenAI-compatible HTTP endpoint (e.g. llama-server)
// to the AIProvider interface. It supports chat and streaming completions.
type OpenAIProvider struct {
	baseURL string
	model   string
	client  *http.Client
}

const defaultOpenAIBaseURL = "http://127.0.0.1:8080"

func getOpenAIBaseURL() string {
	if v := os.Getenv("LLAMA_SERVER_URL"); v != "" {
		return strings.TrimRight(v, "/")
	}
	if v := os.Getenv("OPENAI_BASE_URL"); v != "" {
		return strings.TrimRight(v, "/")
	}
	return defaultOpenAIBaseURL
}

func getOpenAIModel() string {
	if v := os.Getenv("LLAMA_SERVER_MODEL"); v != "" {
		return v
	}
	if v := os.Getenv("OPENAI_MODEL"); v != "" {
		return v
	}
	return "gpt-3.5-turbo"
}

// NewOpenAIProvider creates a provider targeting an OpenAI-compatible endpoint.
func NewOpenAIProvider(baseURL, model string) *OpenAIProvider {
	if baseURL == "" {
		baseURL = getOpenAIBaseURL()
	}
	if model == "" {
		model = getOpenAIModel()
	}
	return &OpenAIProvider{
		baseURL: strings.TrimRight(baseURL, "/"),
		model:   model,
		client:  &http.Client{Timeout: 120 * time.Second},
	}
}

func (p *OpenAIProvider) Name() string { return "openai" }

func (p *OpenAIProvider) GetModel() string { return p.model }

func (p *OpenAIProvider) SetModel(model string) {
	if model != "" {
		p.model = model
	}
}

func (p *OpenAIProvider) SetModelsDir(dir string) {}

// ── OpenAI request/response types ──────────────────────────────────────────

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIChatRequest struct {
	Model    string          `json:"model"`
	Messages []openAIMessage `json:"messages"`
	Stream   bool            `json:"stream"`
}

type openAIChatChoice struct {
	Message      openAIMessage `json:"message"`
	Delta        openAIMessage `json:"delta"`
	FinishReason string        `json:"finish_reason"`
}

type openAIChatResponse struct {
	Choices []openAIChatChoice `json:"choices"`
	Error   struct {
		Message string `json:"message"`
	} `json:"error"`
}

type openAIModel struct {
	ID string `json:"id"`
}

type openAIModelsResponse struct {
	Data []openAIModel `json:"data"`
}

// ── Status ─────────────────────────────────────────────────────────────────

func (p *OpenAIProvider) CheckStatus(ctx context.Context) (*OllamaStatus, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.baseURL+"/v1/models", nil)
	if err != nil {
		return &OllamaStatus{Available: false}, err
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return &OllamaStatus{Available: false}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return &OllamaStatus{Available: false}, fmt.Errorf("endpoint returned %d", resp.StatusCode)
	}
	var mr openAIModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&mr); err != nil {
		return &OllamaStatus{Available: false}, err
	}
	models := make([]string, 0, len(mr.Data))
	for _, m := range mr.Data {
		models = append(models, m.ID)
	}
	// Prefer the configured model if present, otherwise the first available.
	model := p.model
	if len(models) > 0 {
		found := false
		for _, m := range models {
			if m == p.model {
				found = true
				break
			}
		}
		if !found {
			model = models[0]
		}
	}
	return &OllamaStatus{
		Available:       true,
		Model:           model,
		Version:         "openai-compatible",
		AvailableModels: models,
	}, nil
}

func (p *OpenAIProvider) ListModels(ctx context.Context) ([]string, error) {
	st, err := p.CheckStatus(ctx)
	if err != nil {
		return nil, err
	}
	return st.AvailableModels, nil
}

// ── Chat ───────────────────────────────────────────────────────────────────

func (p *OpenAIProvider) Chat(ctx context.Context, messages []ChatMessage, modelOverride string) (string, error) {
	model := p.model
	if modelOverride != "" {
		model = modelOverride
	}
	msgs := make([]openAIMessage, len(messages))
	for i, m := range messages {
		msgs[i] = openAIMessage{Role: m.Role, Content: m.Content}
	}
	body, _ := json.Marshal(openAIChatRequest{Model: model, Messages: msgs, Stream: false})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := p.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("chat failed (%d): %s", resp.StatusCode, string(raw))
	}
	var cr openAIChatResponse
	if err := json.Unmarshal(raw, &cr); err != nil {
		return "", err
	}
	if cr.Error.Message != "" {
		return "", fmt.Errorf("chat error: %s", cr.Error.Message)
	}
	if len(cr.Choices) == 0 {
		return "", fmt.Errorf("no response from model")
	}
	return cr.Choices[0].Message.Content, nil
}

func (p *OpenAIProvider) ChatStream(ctx context.Context, messages []ChatMessage, modelOverride string, onToken func(string)) (string, error) {
	model := p.model
	if modelOverride != "" {
		model = modelOverride
	}
	msgs := make([]openAIMessage, len(messages))
	for i, m := range messages {
		msgs[i] = openAIMessage{Role: m.Role, Content: m.Content}
	}
	body, _ := json.Marshal(openAIChatRequest{Model: model, Messages: msgs, Stream: true})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := p.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("chat stream failed (%d): %s", resp.StatusCode, string(raw))
	}

	var full strings.Builder
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "[DONE]" {
			break
		}
		var cr openAIChatResponse
		if err := json.Unmarshal([]byte(data), &cr); err != nil {
			continue
		}
		if len(cr.Choices) > 0 {
			tok := cr.Choices[0].Delta.Content
			if tok != "" {
				full.WriteString(tok)
				if onToken != nil {
					onToken(tok)
				}
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return full.String(), err
	}
	return full.String(), nil
}

// ── Unsupported ops (server manages models) ────────────────────────────────

func (p *OpenAIProvider) PullModel(ctx context.Context, name string, onProgress func(ProgressResponse) error) error {
	return fmt.Errorf("pull not supported by openai-compatible endpoint")
}

func (p *OpenAIProvider) CreateModel(ctx context.Context, name, from, system string, parameters map[string]any, onProgress func(ProgressResponse) error) error {
	return fmt.Errorf("create not supported by openai-compatible endpoint")
}

func (p *OpenAIProvider) DeleteModel(ctx context.Context, name string) error {
	return fmt.Errorf("delete not supported by openai-compatible endpoint")
}