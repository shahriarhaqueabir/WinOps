package aiops

import (
	"context"
	"os"

	"github.com/ollama/ollama/api"
)

// AIProvider abstracts a local LLM backend (Ollama or llama.cpp).
type AIProvider interface {
	Name() string
	CheckStatus(ctx context.Context) (*OllamaStatus, error)
	SetModel(model string)
	GetModel() string
	Chat(ctx context.Context, messages []ChatMessage, modelOverride string) (string, error)
	ChatStream(ctx context.Context, messages []ChatMessage, modelOverride string, onToken func(string)) (string, error)
	PullModel(ctx context.Context, name string, onProgress func(any) error) error
	CreateModel(ctx context.Context, name, from, system string, parameters map[string]any, onProgress func(any) error) error
	DeleteModel(ctx context.Context, name string) error
	ListModels(ctx context.Context) ([]string, error)
}

// NewProvider creates a provider by kind.
func NewProvider(kind ProviderKind, modelPath string) AIProvider {
	switch kind {
	case ProviderLlama:
		return NewLlamaProvider(modelPath)
	default:
		return NewOllamaProvider()
	}
}

// DefaultProviderKind returns the configured provider kind.
func DefaultProviderKind() ProviderKind {
	if v := os.Getenv("AI_PROVIDER"); v == "llama" {
		return ProviderLlama
	}
	return ProviderOllama
}

// ProviderKind selects the backend implementation.
type ProviderKind string

const (
	ProviderOllama ProviderKind = "ollama"
	ProviderLlama  ProviderKind = "llama"
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
func (p *OllamaProvider) SetModel(model string) { p.Client.SetModel(model) }
func (p *OllamaProvider) GetModel() string      { return p.Client.GetModel() }
func (p *OllamaProvider) Chat(ctx context.Context, messages []ChatMessage, modelOverride string) (string, error) {
	return p.Client.Chat(ctx, messages, modelOverride)
}
func (p *OllamaProvider) ChatStream(ctx context.Context, messages []ChatMessage, modelOverride string, onToken func(string)) (string, error) {
	return p.Client.ChatStream(ctx, messages, modelOverride, onToken)
}
func (p *OllamaProvider) PullModel(ctx context.Context, name string, onProgress func(any) error) error {
	return p.Client.PullModel(ctx, name, func(r api.ProgressResponse) error {
		if onProgress == nil {
			return nil
		}
		return onProgress(r)
	})
}
func (p *OllamaProvider) CreateModel(ctx context.Context, name, from, system string, parameters map[string]any, onProgress func(any) error) error {
	return p.Client.CreateModel(ctx, name, from, system, parameters, func(r api.ProgressResponse) error {
		if onProgress == nil {
			return nil
		}
		return onProgress(r)
	})
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
