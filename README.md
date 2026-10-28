# UniversalOps

**Local-first Windows operations dashboard for SysOps, NetOps, SecOps, DevOps, and AIOps — with Ollama-powered local AI.**

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Platform: Windows](https://img.shields.io/badge/Platform-Windows-0078D6.svg)](https://www.microsoft.com/windows)
[![Go](https://img.shields.io/badge/Go-1.26+-00ADD8.svg)](https://go.dev)
[![Wails](https://img.shields.io/badge/Wails-v2-E01563.svg)](https://wails.io)
[![React](https://img.shields.io/badge/React-19-61DAFB.svg)](https://react.dev)
[![Ollama](https://img.shields.io/badge/Ollama-Local%20AI-black.svg)](https://ollama.com)

> *All analysis runs on your machine. No telemetry. No cloud accounts required.*

## Screenshot

![UniversalOps Dashboard](docs/assets/dashboard.png)

*Drop your screenshots/GIFs into `docs/assets/` as you capture them. Recommended: `dashboard.png`, `sysops.png`, `netops.png`, `secops.png`, `aiops.png`, and `demo.gif`.* 

## Why UniversalOps?

Most "ops" toolkits either ship as a web app that phones home, require complex agent setups, or lock you into cloud dashboards. **UniversalOps** is built to run entirely offline on Windows: inspect hardware and processes, diagnose networks, audit security posture, manage containers/Kubernetes, and get grounded AI assistance — all from a single native desktop app backed by a local SQLite database.

## Download & Install (Recommended)

1. **Download** the latest Windows build from [Releases](https://github.com/shahriarhaqueabir/UniversalOps/releases).
2. **Install prerequisites**: [Microsoft Edge WebView2 Runtime](https://developer.microsoft.com/microsoft-edge/webview2/) (usually preinstalled on modern Windows).
3. **Run** `UniversalOps.exe`. No installer required.

> Some features (e.g. firewall rules, process termination, service management, Defender queries) require elevated privileges. Run as Administrator when performing remediation actions.

### Optional: Local AI with Ollama

For AIOps features, install [Ollama](https://ollama.com) and pull a model:

```bash
ollama pull universalops
# or use any compatible model (e.g. llama-3.1, qwen2.5, mistral)
```

Set `OLLAMA_HOST`/`OLLAMA_MODEL` if your Ollama instance runs on a non-default host. UniversalOps also supports an in-process `llama.cpp` backend via `AI_PROVIDER=llama` with GGUF models placed in your local models directory.

## Features

| Module | What it does |
| --- | --- |
| **Dashboard** | Live system overview, alerts, health signals, and quick actions in one place. |
| **SysOps** | CPU/memory/disk/GPU/battery telemetry, processes/services, startup items, logs, and performance diagnostics. |
| **NetOps** | Interface inventory, routing/DNS, ARP, Wi-Fi, port scanning, connectivity tests (ping/traceroute), topology mapping, and network health checks. |
| **SecOps** | Security posture scoring, firewall inspection/controls, Defender status, listening ports, hardening checks, identity/password policy, forensics capture/export, and incident response actions. |
| **DevOps** | Docker/container management, Kubernetes inspection/actions, packages/services, scripts/workflows, and environment diagnostics. |
| **AIOps** | Local AI assistant powered by Ollama/llama.cpp with system context, persona/custom model creation, chat with tool-aware workflows, and AI-generated reports. |
| **Workflows & Reports** | Automate multi-step checks, schedule tasks, and generate shareable operational reports. |

## Privacy & Security

- **Local-first**: All data stays on disk in a local SQLite database. Nothing is sent to third-party services.
- **No telemetry**: No analytics, crash reporting, or phone-home calls.
- **Least privilege aware**: Destructive/remediation actions are clearly labeled in the UI and require explicit user intent.
- **Auditable**: Forensics snapshots and logs are stored locally under your data directory.

## Build From Source

### Prerequisites

- [Go 1.26+](https://go.dev/dl/)
- [Node.js 20+](https://nodejs.org/)
- [Wails CLI v2](https://wails.io/docs/gettingstarted/installation): `go install github.com/wailsapp/wails/v2/cmd/wails@latest`
- [Microsoft Edge WebView2 Runtime](https://developer.microsoft.com/microsoft-edge/webview2/)

### Development

```bash
# Clone
git clone https://github.com/shahriarhaqueabir/UniversalOps.git
cd UniversalOps

# Run in dev mode (hot reload)
wails dev
```

### Production Build

```bash
wails build
# Executable: build/bin/UniversalOps.exe
```

## Contributing

Contributions are welcome! If you spot a bug or have an idea:

1. Open an [issue](https://github.com/shahriarhaqueabir/UniversalOps/issues) with details/repro steps.
2. Submit a PR with focused changes and a clear description.

## License

[MIT](LICENSE) © UniversalOps
