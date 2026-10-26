package aiops

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	llama "github.com/goccy/go-llama"
)

// LlamaProvider wraps goccy/go-llama for in-process GGUF inference.
type LlamaProvider struct {
	mu            sync.RWMutex
	modelPath     string
	effectiveModel string

	inst  *llama.Llama
	model *llama.Model
}

// NewLlamaProvider creates a provider bound to the given GGUF model path.
func NewLlamaProvider(modelPath string) *LlamaProvider {
	if modelPath == "" {
		modelPath = os.Getenv("UNIVERSALOPS_MODEL_PATH")
	}
	if modelPath == "" {
		if home, err := os.UserHomeDir(); err == nil {
			modelPath = filepath.Join(home, ".universalops", "models", "model.gguf")
		}
	}
	return &LlamaProvider{modelPath: modelPath, effectiveModel: filepath.Base(modelPath)}
}

func (p *LlamaProvider) Name() string { return "llama" }

func (p *LlamaProvider) GetModel() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.effectiveModel
}

func (p *LlamaProvider) SetModel(model string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.modelPath = model
	p.effectiveModel = filepath.Base(model)
	// drop loaded model so it reloads
	if p.model != nil {
		_ = p.model.Close()
		p.model = nil
	}
	if p.inst != nil {
		_ = p.inst.Close()
		p.inst = nil
	}
}

func (p *LlamaProvider) ensureLoaded() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.model != nil {
		return nil
	}
	if _, err := os.Stat(p.modelPath); err != nil {
		return fmt.Errorf("llama model not found: %s", p.modelPath)
	}
	inst, err := llama.New()
	if err != nil {
		return fmt.Errorf("llama instance: %w", err)
	}
	model, err := inst.LoadModel(p.modelPath)
	if err != nil {
		_ = inst.Close()
		return fmt.Errorf("llama load model: %w", err)
	}
	p.inst = inst
	p.model = model
	return nil
}

func (p *LlamaProvider) CheckStatus(ctx context.Context) (*OllamaStatus, error) {
	if err := p.ensureLoaded(); err != nil {
		return &OllamaStatus{Available: false}, err
	}
	p.mu.RLock()
	defer p.mu.RUnlock()
	info, _ := p.model.Info()
	return &OllamaStatus{
		Available:       true,
		Model:           p.effectiveModel,
		Version:         info.Desc,
		AvailableModels: []string{p.effectiveModel},
	}, nil
}

func (p *LlamaProvider) buildPrompt(messages []ChatMessage) (string, error) {
	// Prepend stored system prompt if present.
	if data, err := os.ReadFile(p.modelPath + ".system.txt"); err == nil && len(data) > 0 {
		messages = append([]ChatMessage{{Role: "system", Content: string(data)}}, messages...)
	}
	p.mu.RLock()
	model := p.model
	p.mu.RUnlock()
	if model == nil {
		return "", fmt.Errorf("llama model not loaded")
	}
	llamaMsgs := make([]llama.Message, len(messages))
	for i, m := range messages {
		llamaMsgs[i] = llama.Message{Role: m.Role, Content: m.Content}
	}
	prompt, err := model.ApplyChatTemplate(llamaMsgs, "", true)
	if err != nil {
		// fallback: concatenate
		var b strings.Builder
		for _, m := range messages {
			b.WriteString(m.Role)
			b.WriteString(": ")
			b.WriteString(m.Content)
			b.WriteString("\n")
		}
		return b.String(), nil
	}
	return prompt, nil
}

func (p *LlamaProvider) Chat(ctx context.Context, messages []ChatMessage, modelOverride string) (string, error) {
	if err := p.ensureLoaded(); err != nil {
		return "", err
	}
	prompt, err := p.buildPrompt(messages)
	if err != nil {
		return "", err
	}
	p.mu.RLock()
	model := p.model
	p.mu.RUnlock()
	ctxObj, err := model.NewContext(llama.ContextParams{NCtx: 4096})
	if err != nil {
		return "", fmt.Errorf("llama context: %w", err)
	}
	defer ctxObj.Close()
	res, err := ctxObj.Generate(prompt, llama.Params{NPredict: 512, Temperature: 0.7, TopP: 0.9})
	if err != nil {
		return "", fmt.Errorf("llama generate: %w", err)
	}
	if res.Text == "" {
		return "", fmt.Errorf("llama returned empty response")
	}
	return res.Text, nil
}

// modelsDir returns the directory used to store GGUF models.
func modelsDir() string {
	if d := os.Getenv("UNIVERSALOPS_MODELS_DIR"); d != "" {
		return d
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".universalops", "models")
	}
	return filepath.Join(".", "models")
}

func (p *LlamaProvider) modelsDir() string { return modelsDir() }

func (p *LlamaProvider) ListModels(ctx context.Context) ([]string, error) {
	dir := p.modelsDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if strings.HasSuffix(strings.ToLower(e.Name()), ".gguf") {
			out = append(out, e.Name())
		}
	}
	return out, nil
}

func (p *LlamaProvider) PullModel(ctx context.Context, name string, onProgress func(any) error) error {
	// name may be an http(s) URL or a local filename already present.
	if strings.HasPrefix(name, "http://") || strings.HasPrefix(name, "https://") {
		if err := os.MkdirAll(p.modelsDir(), 0o755); err != nil {
			return err
		}
		resp, err := http.Get(name)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("download failed: %s", resp.Status)
		}
		total := resp.ContentLength
		filename := filepath.Base(name)
		if idx := strings.Index(filename, "?"); idx >= 0 {
			filename = filename[:idx]
		}
		dest := filepath.Join(p.modelsDir(), filename)
		f, err := os.Create(dest)
		if err != nil {
			return err
		}
		defer f.Close()
		buf := make([]byte, 32*1024)
		var downloaded int64
		for {
			n, rerr := resp.Body.Read(buf)
			if n > 0 {
				if _, werr := f.Write(buf[:n]); werr != nil {
					return werr
				}
				downloaded += int64(n)
				if onProgress != nil {
					_ = onProgress(map[string]any{
						"status":     "downloading",
						"completed":  downloaded,
						"total":      total,
						"filename":   filename,
					})
				}
			}
			if rerr == io.EOF {
				break
			}
			if rerr != nil {
				return rerr
			}
		}
		if onProgress != nil {
			_ = onProgress(map[string]any{"status": "success", "filename": filename})
		}
		return nil
	}
	// local: nothing to do
	if onProgress != nil {
		_ = onProgress(map[string]any{"status": "success", "filename": name})
	}
	return nil
}

func (p *LlamaProvider) CreateModel(ctx context.Context, name, from, system string, parameters map[string]any, onProgress func(any) error) error {
	// Ensure base model is available (download if URL).
	if from != "" {
		if err := p.PullModel(ctx, from, onProgress); err != nil {
			return err
		}
	}
	// Resolve model path.
	modelPath := from
	if !strings.HasPrefix(from, "http") {
		// treat as filename in models dir
		if !filepath.IsAbs(from) {
			modelPath = filepath.Join(p.modelsDir(), from)
		}
	} else {
		modelPath = filepath.Join(p.modelsDir(), filepath.Base(from))
	}
	p.SetModel(modelPath)
	// Store system prompt alongside model.
	if system != "" {
		sp := modelPath + ".system.txt"
		_ = os.WriteFile(sp, []byte(system), 0o644)
	}
	if onProgress != nil {
		_ = onProgress(map[string]any{"status": "success"})
	}
	return nil
}

func (p *LlamaProvider) DeleteModel(ctx context.Context, name string) error {
	path := name
	if !filepath.IsAbs(name) {
		path = filepath.Join(p.modelsDir(), name)
	}
	_ = os.Remove(path)
	_ = os.Remove(path + ".system.txt")
	if p.GetModel() == filepath.Base(name) {
		p.SetModel("")
	}
	return nil
}

func (p *LlamaProvider) ChatStream(ctx context.Context, messages []ChatMessage, modelOverride string, onToken func(string)) (string, error) {
	if err := p.ensureLoaded(); err != nil {
		return "", err
	}
	prompt, err := p.buildPrompt(messages)
	if err != nil {
		return "", err
	}
	p.mu.RLock()
	model := p.model
	p.mu.RUnlock()
	ctxObj, err := model.NewContext(llama.ContextParams{NCtx: 4096})
	if err != nil {
		return "", fmt.Errorf("llama context: %w", err)
	}
	defer ctxObj.Close()
	res, err := ctxObj.Stream(prompt, llama.Params{NPredict: 512, Temperature: 0.7, TopP: 0.9}, onToken)
	if err != nil {
		return "", fmt.Errorf("llama stream: %w", err)
	}
	if res.Text == "" {
		return "", fmt.Errorf("llama returned empty response")
	}
	return res.Text, nil
}
