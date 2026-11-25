//go:build windows

package common

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// LlamaServerManager manages a bundled llama-server.exe child process so the
// app can get GPU-accelerated inference without requiring the user to start a
// separate application.
type LlamaServerManager struct {
	mu      sync.Mutex
	cmd     *exec.Cmd
	pid     int
	port    string
	model   string
	binary  string
	started bool
}

var (
	llamaServerInstance *LlamaServerManager
	llamaServerOnce     sync.Once
)

// GetLlamaServerManager returns the singleton manager.
func GetLlamaServerManager() *LlamaServerManager {
	llamaServerOnce.Do(func() {
		llamaServerInstance = &LlamaServerManager{port: "18020"}
	})
	return llamaServerInstance
}

// Configure sets the binary path, model path, and port before Start.
func (m *LlamaServerManager) Configure(binary, model, port string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if binary != "" {
		m.binary = binary
	}
	if model != "" {
		m.model = model
	}
	if port != "" {
		m.port = port
	}
}

// BinaryPath returns the configured or auto-detected binary path.
func (m *LlamaServerManager) BinaryPath() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.binary != "" {
		return m.binary
	}
	return findLlamaServerBinary()
}

// findLlamaServerBinary locates llama-server.exe.
func findLlamaServerBinary() string {
	// 1. Env override
	if v := os.Getenv("LLAMA_SERVER_PATH"); v != "" {
		if _, err := os.Stat(v); err == nil {
			return v
		}
	}
	// 2. Bundled folder next to the app
	base := GetBaseDir()
	candidates := []string{
		filepath.Join(base, "llama-b10952-bin-win-cuda-x64", "llama-server.exe"),
		filepath.Join(".", "llama-b10952-bin-win-cuda-x64", "llama-server.exe"),
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	// 3. PATH
	if p, err := exec.LookPath("llama-server.exe"); err == nil {
		return p
	}
	if p, err := exec.LookPath("llama-server"); err == nil {
		return p
	}
	return ""
}

// IsAvailable reports whether a usable binary was found.
func (m *LlamaServerManager) IsAvailable() bool {
	return m.BinaryPath() != ""
}

// Start launches llama-server and waits for readiness.
func (m *LlamaServerManager) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.started {
		return nil
	}
	bin := m.binary
	if bin == "" {
		bin = findLlamaServerBinary()
	}
	if bin == "" {
		return fmt.Errorf("llama-server binary not found")
	}
	if m.model == "" {
		m.model = os.Getenv("UNIVERSALOPS_MODEL_PATH")
	}
	if m.model == "" {
		// try first gguf in models dir
		dir := os.Getenv("UNIVERSALOPS_MODELS_DIR")
		if dir == "" {
			dir = filepath.Join(GetBaseDir(), "data", "models")
		}
		entries, _ := os.ReadDir(dir)
		for _, e := range entries {
			if strings.HasSuffix(strings.ToLower(e.Name()), ".gguf") {
				m.model = filepath.Join(dir, e.Name())
				break
			}
		}
	}
	if m.model == "" {
		// fall back to any gguf under data/
		dataDir := filepath.Join(GetBaseDir(), "data")
		entries, _ := os.ReadDir(dataDir)
		for _, e := range entries {
			if strings.HasSuffix(strings.ToLower(e.Name()), ".gguf") {
				m.model = filepath.Join(dataDir, e.Name())
				break
			}
		}
	}
	if m.model == "" {
		return fmt.Errorf("no GGUF model found")
	}
	if m.port == "" {
		m.port = "18020"
	}

	args := []string{
		"-m", m.model,
		"--host", "127.0.0.1",
		"--port", m.port,
		"-ngl", "99",
		"-c", "4096",
	}
	m.cmd = HiddenCommand(bin, args...)
	if err := m.cmd.Start(); err != nil {
		return fmt.Errorf("start llama-server: %w", err)
	}
	m.pid = m.cmd.Process.Pid
	RegisterProcess(m.cmd)

	// Wait for readiness (up to 60s)
	deadline := time.Now().Add(60 * time.Second)
	url := "http://127.0.0.1:" + m.port + "/health"
	for time.Now().Before(deadline) {
		resp, err := http.Get(url)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				m.started = true
				LogInfo("llama-server started on port %s (pid %d)", m.port, m.pid)
				return nil
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("llama-server did not become ready within 60s")
}

// Stop terminates the managed process.
func (m *LlamaServerManager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.started {
		return
	}
	if m.cmd != nil && m.cmd.Process != nil {
		_ = m.cmd.Process.Kill()
		UnregisterProcess(m.pid)
	}
	m.started = false
	m.cmd = nil
	LogInfo("llama-server stopped")
}

// URL returns the base URL of the managed server.
func (m *LlamaServerManager) URL() string {
	if m.port == "" {
		return "http://127.0.0.1:18020"
	}
	return "http://127.0.0.1:" + m.port
}

// IsStarted reports whether the server is running.
func (m *LlamaServerManager) IsStarted() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.started
}
