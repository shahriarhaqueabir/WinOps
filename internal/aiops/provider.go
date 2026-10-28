package aiops

import (
	"context"
	"os"
	"strings"

	"github.com/shahriarhaqueabir/UniversalOps/internal/common"
)

// AIProvider abstracts a local LLM backend (Ollama or llama.cpp).
type AIProvider interface {
	Name() string
	CheckStatus(ctx context.Context) (*OllamaStatus, error)
	SetModel(model string)
	GetModel() string
	SetModelsDir(dir string)
	Chat(ctx context.Context, messages []ChatMessage, modelOverride string) (string, error)
	ChatStream(ctx context.Context, messages []ChatMessage, modelOverride string, onToken func(string)) (string, error)
	PullModel(ctx context.Context, name string, onProgress func(ProgressResponse) error) error
	CreateModel(ctx context.Context, name, from, system string, parameters map[string]any, onProgress func(ProgressResponse) error) error
	DeleteModel(ctx context.Context, name string) error
	ListModels(ctx context.Context) ([]string, error)
}

// NewProvider creates a provider by kind.
func NewProvider(kind ProviderKind, modelPath string) AIProvider {
	switch kind {
	case ProviderLlama:
		return NewLlamaProvider(modelPath)
	case ProviderOpenAI:
		url := common.GetLlamaServerManager().URL()
		return NewOpenAIProvider(url, "")
	default:
		return NewOllamaProvider()
	}
}

// DefaultProviderKind returns the configured provider kind.
func DefaultProviderKind() ProviderKind {
	if v := os.Getenv("AI_PROVIDER"); v != "" {
		v = strings.ToLower(v)
		if v == "llama" {
			return ProviderLlama
		}
		if v == "openai" || v == "llama-server" || v == "llama.cpp" || v == "server" {
			return ProviderOpenAI
		}
	}
	// Auto-detect: if a GPU llama-server binary and a GGUF model are available,
	// prefer the in-process-managed server for fast GPU inference with no
	// separate app to launch.
	if common.GetLlamaServerManager().IsAvailable() {
		return ProviderOpenAI
	}
	return ProviderOllama
}

// ProviderKind selects the backend implementation.
type ProviderKind string

const (
	ProviderOllama ProviderKind = "ollama"
	ProviderLlama  ProviderKind = "llama"
	ProviderOpenAI ProviderKind = "openai"
)

// OllamaProvider adapts *OllamaClient to AIProvider.
type OllamaProvider struct {
	Client *OllamaClient
}

func NewOllamaProvider() *OllamaProvider { return &OllamaProvider{Client: NewOllamaClient()} }

func (p *OllamaProvider) Name() string { return "ollama" }
func (p *OllamaProvider) CheckStatus(ctx context.Context) (*OllamaStatus, error) {
	return p.Client.CheckStatus(ctx)
}
func (p *OllamaProvider) SetModel(model string)   { p.Client.SetModel(model) }
func (p *OllamaProvider) SetModelsDir(dir string) {}
func (p *OllamaProvider) GetModel() string        { return p.Client.GetModel() }
func (p *OllamaProvider) Chat(ctx context.Context, messages []ChatMessage, modelOverride string) (string, error) {
	return p.Client.Chat(ctx, messages, modelOverride)
}
func (p *OllamaProvider) ChatStream(ctx context.Context, messages []ChatMessage, modelOverride string, onToken func(string)) (string, error) {
	return p.Client.ChatStream(ctx, messages, modelOverride, onToken)
}
func (p *OllamaProvider) PullModel(ctx context.Context, name string, onProgress func(ProgressResponse) error) error {
	return p.Client.PullModel(ctx, name, onProgress)
}
func (p *OllamaProvider) CreateModel(ctx context.Context, name, from, system string, parameters map[string]any, onProgress func(ProgressResponse) error) error {
	return p.Client.CreateModel(ctx, name, from, system, parameters, onProgress)
}
func (p *OllamaProvider) DeleteModel(ctx context.Context, name string) error {
	return p.Client.DeleteModel(ctx, name)
}
func (p *OllamaProvider) ListModels(ctx context.Context) ([]string, error) {
	st, err := p.Client.CheckStatus(ctx)
	if err != nil {
		return nil, err
	}
	return st.AvailableModels, nil
}
