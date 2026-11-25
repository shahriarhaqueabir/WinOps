package aiops

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	llama "github.com/goccy/go-llama"
)

// LlamaProvider wraps goccy/go-llama for in-process GGUF inference.
type LlamaProvider struct {
	mu                sync.RWMutex
	modelPath         string
	effectiveModel    string
	modelsDirOverride string

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
			modelPath = filepath.Join(home, ".winops", "models", "model.gguf")
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
		return filepath.Join(home, ".winops", "models")
	}
	return filepath.Join(".", "models")
}

func (p *LlamaProvider) modelsDir() string {
	if p != nil {
		p.mu.RLock()
		defer p.mu.RUnlock()
		if strings.TrimSpace(p.modelsDirOverride) != "" {
			return p.modelsDirOverride
		}
	}
	return modelsDir()
}

// SetModelsDir overrides the directory used to store GGUF models.
func (p *LlamaProvider) SetModelsDir(dir string) {
	if p == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.modelsDirOverride = strings.TrimSpace(dir)
}

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

func (p *LlamaProvider) PullModel(ctx context.Context, name string, onProgress func(ProgressResponse) error) error {
	// name may be an http(s) URL or a local filename already present.
	u, err := url.Parse(name)
	// SECURITY: HTTPS only, validated host
	if err == nil && u.Scheme == "https" && u.Host != "" {
		// Host whitelist (HuggingFace + trusted sources)
		allowedHosts := map[string]bool{
			"huggingface.co": true, "cdn-lfs.huggingface.co": true,
			"us.aws.cdn.hf.co": true, "github.com": true,
		}
		if !allowedHosts[u.Hostname()] {
			return fmt.Errorf("host not allowed: %s", u.Hostname())
		}
		if err := os.MkdirAll(p.modelsDir(), 0o700); err != nil {
			return err
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
		if err != nil {
			return err
		}
		// SECURITY: reject redirects
		client := &http.Client{
			Timeout: 30 * time.Minute,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return fmt.Errorf("redirects not allowed")
			},
		}
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("download failed: %s", resp.Status)
		}
		// Content-Type validation
		ct := resp.Header.Get("Content-Type")
		if strings.Contains(ct, "text/html") || strings.Contains(ct, "text/plain") {
			return fmt.Errorf("unexpected content type: %s", ct)
		}
		const maxModelDownloadBytes = int64(64 << 30)
		if resp.ContentLength > maxModelDownloadBytes {
			return fmt.Errorf("model download exceeds %d bytes", maxModelDownloadBytes)
		}
		total := resp.ContentLength
		filename := filepath.Base(u.Path)
		if filename == "" || filename == "." || filename == ".." || strings.ContainsAny(filename, `/\`) {
			return fmt.Errorf("invalid model filename")
		}
		// SECURITY: only .gguf files
		if !strings.HasSuffix(strings.ToLower(filename), ".gguf") {
			return fmt.Errorf("only .gguf files allowed")
		}
		dest := filepath.Join(p.modelsDir(), filename)
		tmpDest := dest + ".tmp"
		f, err := os.OpenFile(tmpDest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
		if err != nil {
			return err
		}
		buf := make([]byte, 32*1024)
		var downloaded int64
		for {
			n, rerr := resp.Body.Read(buf)
			if n > 0 {
				if _, werr := f.Write(buf[:n]); werr != nil {
					f.Close()
					os.Remove(tmpDest)
					return werr
				}
				downloaded += int64(n)
				if downloaded > maxModelDownloadBytes {
					f.Close()
					os.Remove(tmpDest)
					return fmt.Errorf("model download exceeds %d bytes", maxModelDownloadBytes)
				}
				if onProgress != nil {
					_ = onProgress(ProgressResponse{
						Status:    "downloading",
						Completed: downloaded,
						Total:     total,
					})
				}
			}
			if rerr == io.EOF {
				break
			}
			if rerr != nil {
				f.Close()
				os.Remove(tmpDest)
				return rerr
			}
		}
		f.Close()
		// GGUF magic bytes check
		gf, err := os.Open(tmpDest)
		if err == nil {
			magic := make([]byte, 4)
			_, _ = gf.Read(magic)
			gf.Close()
			if string(magic) != "GGUF" {
				os.Remove(tmpDest)
				return fmt.Errorf("downloaded file is not a valid GGUF model")
			}
		}
		// Atomic rename
		if err := os.Rename(tmpDest, dest); err != nil {
			os.Remove(tmpDest)
			return err
		}
		// File permission lockdown
		os.Chmod(dest, 0o600)
		if onProgress != nil {
			_ = onProgress(ProgressResponse{Status: "success"})
		}
		return nil
	}
	// Local path (no download); keep the caller's progress contract.
	if onProgress != nil {
		_ = onProgress(ProgressResponse{Status: "success"})
	}
	return nil
}

func (p *LlamaProvider) CreateModel(ctx context.Context, name, from, system string, parameters map[string]any, onProgress func(ProgressResponse) error) error {
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
		_ = os.WriteFile(sp, []byte(system), 0o600)
	}
	if onProgress != nil {
		_ = onProgress(ProgressResponse{Status: "success"})
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
