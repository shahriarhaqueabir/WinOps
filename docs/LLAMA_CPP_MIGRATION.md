# Migration Plan: Ollama → llama.cpp (Winops)

## Status: PARTIALLY DONE — needs path update and model verification — Config pre-updated 2026-11-18

## Current State
Winops already has a multi-provider architecture:
- `internal/aiops/provider.go` — `AIProvider` interface with Ollama, Llama, and OpenAI variants
- `internal/aiops/llama.go` — in-process GGUF inference via goccy/go-llama
- `internal/aiops/llamaserver.go` — manages a llama-server subprocess
- `internal/aiops/ollama.go` — Ollama client
- `internal/aiops/openai.go` — OpenAI-compatible client (for llama-server)
- `internal/common/capability.go` — auto-detects available providers

**Auto-detection logic (DefaultProviderKind):**
1. If `AI_PROVIDER` env var is set → use that
2. If llama-server binary + GGUF model available → prefer OpenAI (llama-server)
3. Otherwise → fall back to Ollama

## Pre-Updated Config (2026-11-18)
- `UNIVERSALOPS_MODELS_DIR` set to `<models directory>` (user-level env var)
- `<models directory>\` contains Qwythos-9B Q4_K_M
- llama.cpp v1.0.0 on PATH at `<llama.cpp install directory>`
- No .env file in Winops (uses env vars + auto-detection)

Code already supports llama.cpp — main work is verification.

## What's Already Done
- ✅ `UNIVERSALOPS_MODELS_DIR` set to `<models directory>`
- ✅ llama.cpp v1.0.0 installed at `<llama.cpp install directory>` (on PATH)
- ✅ GGUF model at `<models directory>\Qwythos-9B-Claude-Mythos-5-1M-Q4_K_M.gguf`

## Migration Steps

### Phase 1: Verify Auto-Detection Works
1. Build Winops: `cd <repo root> && go build ./...`
2. Run: `go run .` or launch the GUI
3. Check logs — should auto-detect llama-server (since binary is on PATH and models dir is set)
4. Verify `ProviderOpenAI` is selected (not `ProviderOllama`)

### Phase 2: Update Default Models Directory (optional)
If you want `~/.winops/models/` to point to the new location:
```powershell
# Create symlink
New-Item -ItemType Junction -Path "$env:USERPROFILE\.winops\models" -Target "<models directory>"
```
Or update `llama.go` line 36 to default to `<models directory>`.

### Phase 3: Test Inference
1. Launch Winops GUI
2. Open AI Ops → Chat
3. Send a test message
4. Verify response comes from llama-server (check logs for "provider: openai")
5. Test streaming (ChatStream)
6. Test model listing (ListModels)

### Phase 4: Clean Up Ollama References (optional)
- `internal/aiops/ollama.go` and `ollama_test.go` can be kept for fallback
- `internal/aiops/modelfile.go` — only needed for Ollama model creation; can be deprecated
- Remove Ollama-specific UI elements if desired

### Phase 5: Update Documentation
- README.md: mention llama.cpp as primary backend
- USAGE.md: update setup instructions

## Files to Check/Modify
- `internal/aiops/llama.go` — line 36 (default model path), line 162 (env var override)
- `internal/common/llamaserver.go` — server lifecycle management
- `internal/common/capability.go` — auto-detection logic (already correct)
- `internal/aiops/provider.go` — provider selection (already correct)

## Dependencies
- llama.cpp v1.0.0 at `<llama.cpp install directory>` (on PATH)
- GGUF model at `<models directory>\`

## Estimated Effort
- Phase 1-3: 1-2 hours (mostly verification)
- Phase 4-5: 1 hour
- Total: ~2-3 hours (focused session)

## Notes
- Winops is the most migration-ready project — the architecture already supports llama.cpp
- The main work is verification, not code changes
- Consider adding a model picker UI that lists GGUF files from `<models directory>\`
