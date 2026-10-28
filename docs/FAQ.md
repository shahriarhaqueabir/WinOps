# FAQ

## General

**Is UniversalOps safe to run?**
Yes. It's local-first with no telemetry. Actions that modify system state are explicit and labeled in the UI.

**Does it work offline?**
Mostly yes. All SysOps/NetOps/SecOps/DevOps inspection works offline. AIOps requires a local LLM backend (Ollama or llama.cpp running locally). If no backend is available, those features will show status/unavailable.

**Which OS is supported?**
Windows (amd64). WebView2 is required.

## Installation

**Do I need Go/Node/Wails to run it?**
No. Download `UniversalOps.exe` from [Releases](https://github.com/shahriarhaqueabir/UniversalOps/releases).

**WebView2 is missing — what do I do?**
Install the [Evergreen WebView2 Runtime](https://developer.microsoft.com/microsoft-edge/webview2/) from Microsoft.

## AI / Ollama

**Ollama isn't detected.**
- Ensure `ollama serve` is running.
- Check `OLLAMA_HOST` (default `http://127.0.0.1:11434`).
- Pull a model: `ollama pull universalops` (or any model you want to use).

**Can I use llama.cpp without Ollama?**
Yes. Set `AI_PROVIDER=llama` and place a GGUF model in your local models directory. See the README for details.

## Troubleshooting

**Some actions fail with access denied.**
Run UniversalOps as Administrator. Certain operations (firewall, services, processes, Defender) require elevated privileges.

**The app won't start.**
- Verify WebView2 is installed.
- Check Windows Defender/SmartScreen (if downloaded from releases, you may need to allow it).
- Ensure you extracted the EXE (don't run from inside a locked archive in some cases).
