# UniversalOps

**Local-first Windows operations dashboard for SysOps, NetOps, SecOps, DevOps, and AIOps — with Ollama-powered local AI.**

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Platform: Windows](https://img.shields.io/badge/Platform-Windows-0078D6.svg)](https://www.microsoft.com/windows)
[![Go](https://img.shields.io/badge/Go-1.26+-00ADD8.svg)](https://go.dev)
[![Wails](https://img.shields.io/badge/Wails-v2-E01563.svg)](https://wails.io)
[![React](https://img.shields.io/badge/React-19-61DAFB.svg)](https://react.dev)
[![Ollama](https://img.shields.io/badge/Ollama-Local%20AI-black.svg)](https://ollama.com)

> *All analysis runs on your machine. No telemetry. No cloud accounts required.*

---

## The Problem

Modern IT operations tooling is fragmented and cloud-dependent. A typical workflow means juggling Task Manager, PowerShell, `netsh`, Wireshark, Docker Desktop, kubectl, Windows Defender consoles, and a browser tab full of SaaS dashboards — each phoning home, each with its own auth, each a separate window. When something breaks at 2 AM on an air-gapped or restricted network, you need answers *now*, not a login screen.

**UniversalOps** collapses that stack into a single native Windows desktop app that runs entirely offline: hardware and process telemetry, network diagnostics, security posture, container/Kubernetes management, and a locally-hosted AI analyst — all backed by a local SQLite database. Nothing leaves the machine.

## Who It's For

- **SysAdmins / IT generalists** who want one pane of glass for a Windows workstation or server.
- **Security analysts** triaging a host: posture score, firewall rules, listening ports, hardening gaps, forensics capture.
- **Network engineers** mapping a LAN, scanning ports, tracing routes, inspecting DNS/ARP.
- **DevOps engineers** managing Docker/Kubernetes locally without installing three separate GUIs.
- **Privacy-conscious teams** who cannot send telemetry to a vendor.

## Use Cases

1. **Morning health check** — open the Dashboard, see CPU/RAM/disk/network at a glance, review overnight alerts, run a scheduled workflow.
2. **Incident triage** — SecOps module scores posture, lists listening ports, captures a forensic snapshot, and the AI assistant correlates telemetry to suggest a root cause.
3. **Network mapping** — NetOps discovers LAN devices, draws a topology, runs health checks, exports a report.
4. **Hardening audit** — run all hardening checks, see pass/fail, apply remediations with explicit confirmation.
5. **AI-assisted ops** — ask "why is CPU spiking?" and get an answer grounded in live metrics, logs, and events from *your* machine.

## How It Works

```
┌─────────────────────────────────────────────────────────────┐
│  Wails v2 (WebView2)  ←→  Go backend (bound methods)        │
│  React 19 + TS frontend      │                              │
│                              │                              │
│  ┌──────────┐  ┌──────────┐  │  ┌──────────────────────┐    │
│  │Dashboard │  │ SysOps   │  │  │ Collectors (registry)│    │
│  │          │  │ NetOps   │  │  │  → DataPipeline      │    │
│  │          │  │ SecOps   │  │  │  → AlertEngine       │    │
│  │          │  │ DevOps   │  │  │  → EventBus          │    │
│  │          │  │ AIOps    │  │  │  → EngineLoop (2s)   │    │
│  └──────────┘  └──────────┘  │  └──────────┬───────────┘    │
│                              │             │                 │
│                              │  ┌──────────▼───────────┐    │
│                              │  │ SQLite (modernc)     │    │
│                              │  │ metrics, logs, events│    │
│                              │  │ alerts, reports, …   │    │
│                              │  └──────────────────────┘    │
│                              │                              │
│  AIOps ──► Ollama / llama.cpp (local LLM, optional)         │
└─────────────────────────────────────────────────────────────┘
```

- **Collectors** (`internal/common` collector registry) probe the OS on intervals and push `MetricSample`s into the `DataPipeline`.
- **EngineLoop** (2 s tick) emits live metrics to the frontend, evaluates `AlertEngine` rules, persists events, and triggers reports.
- **AlertEngine** holds default + user-defined rules; fires alerts persisted to SQLite; supports flap-count debouncing.
- **AIOps** builds a grounded prompt from the knowledge snapshot + metric history + recent events, streams tokens from a local LLM, and can emit `<action_request>` tags that require explicit user confirmation via the safety handshake.
- **Sandbox** (`internal/common/sandbox*.go`) runs OS commands with restricted tokens, dropped privileges, and low integrity on Windows.

### Code snippet — safe command execution with validation

```go
// internal/sysops/actions.go
func RunSystemAction(action SystemAction) (*ActionResult, error) {
    // ...
    switch action {
    case ActionReboot:
        cmd = common.HiddenCommand("shutdown", "/r", "/t", "0")
    case ActionFlushDNS:
        cmd = common.HiddenCommand("ipconfig", "/flushdns")
    // ...
    }
    output, err := cmd.CombinedOutput()
    // ...
}
```

### Code snippet — alert rule persistence with full sync

```go
// internal/common/alerts.go
func (ae *AlertEngine) PersistRules() {
    ae.mu.RLock()
    rules := make([]AlertRule, len(ae.rules))
    copy(rules, ae.rules)
    ae.mu.RUnlock()

    s := GetStorage()
    if s == nil { return }

    tx, err := s.Begin()
    if err != nil { return }
    defer tx.Rollback()

    // Sync: clear then re-insert so removed rules don't linger.
    if _, err := tx.Exec(`DELETE FROM alert_rules`); err != nil { return }
    for _, r := range rules {
        _ = s.InsertAlertRule(AlertRuleRecord{...}, tx)
    }
    _ = tx.Commit()
}
```

### Code snippet — frontend data fetching (React Query)

```tsx
// cmd/opsforall-gui/frontend/src/pages/SysOps/HardwareTab.tsx
const { data: hw, isLoading, isError, refetch } = useQuery<HardwareInfo>({
  queryKey: ['sysops-hardware-info'],
  queryFn: async () => (await call('SysOps.GetHardwareInfo')) as HardwareInfo,
  refetchInterval: 60000,
})
```

## Tech Stack & Key Decisions

| Layer | Choice | Why |
|---|---|---|
| Desktop shell | **Wails v2** (WebView2) | Native window, Go bindings, no Electron bloat, small binary (~36 MB). |
| Backend | **Go 1.26** | Single static binary, great concurrency for collectors, strong stdlib. |
| Frontend | **React 19 + TypeScript + Vite** | Modern, fast HMR, type-safe Wails bindings. |
| Styling | **Tailwind CSS 4** | Utility-first, dark-mode friendly, small runtime. |
| State/data | **TanStack Query + Zustand** | Server-state caching + lightweight client state. |
| Charts | **Recharts** | Declarative, composable. |
| Database | **modernc.org/sqlite** (pure Go) | No CGO, portable, WAL mode for concurrent reads. |
| System metrics | **gopsutil v4** | Cross-API coverage without cgo. |
| AI | **Ollama HTTP client** (or in-process `go-llama`) | Local-only LLM; no SDK dependency (removed for security/hygiene). |
| DNS | **miekg/dns** | Full-featured resolver for advanced queries. |
| Sandbox | Windows restricted tokens + integrity levels | Least-privilege command execution. |

**Notable decisions**
- **No telemetry, ever.** No analytics, crash reporting, or phone-home. The only outbound calls are the optional GitHub update check and on-demand LHM download — both user-initiated.
- **Pure-Go SQLite** (`modernc.org/sqlite`) to avoid CGO and keep `wails build` reproducible.
- **Removed the Ollama SDK** in favor of a direct HTTP client — smaller attack surface, fewer transitive deps.
- **Windows-only target.** Dropped Linux/macOS code paths to keep the codebase focused and auditable.
- **Safety handshake** for all destructive actions: AI-suggested remediations require explicit user confirmation and are audited in `decisions_audit`.

## Privacy & Security

- **Local-first**: all data in `data/universalops.db` (SQLite, WAL mode). Nothing sent to third parties.
- **No telemetry**: no analytics, crash reporting, or phone-home.
- **Least privilege**: destructive actions are labeled and gated behind explicit confirmation.
- **Auditable**: forensics snapshots, decisions audit, and logs stored locally.
- **Sandboxed execution**: OS commands run with restricted tokens and low integrity levels.
- **Input validation**: PIDs, IPs, usernames, service/interface/k8s names validated before reaching `exec`.
- **Secret scrubbing**: `ScrubSecrets` redacts passwords/keys/tokens from persisted events.

## Modules

| Module | Highlights |
|---|---|
| **Dashboard** | Live system overview, alerts, health signals, quick actions. |
| **SysOps** | CPU/mem/disk/GPU/battery telemetry, processes, services, startup items, logs, diagnostics. |
| **NetOps** | Interfaces, routing, DNS, ARP, Wi-Fi, port scan, ping/traceroute, topology, health checks. |
| **SecOps** | Posture score, firewall, Defender, listening ports, hardening, identity/password policy, forensics, IR. |
| **DevOps** | Docker, Kubernetes, packages, services, scripts/workflows, environment diagnostics. |
| **AIOps** | Local AI chat with live system context, persona/model management, reports, tool-aware workflows. |
| **Workflows & Reports** | Multi-step checks, scheduling, shareable operational reports. |

## Data Stored Locally

`metrics`, `logs`, `events`, `alerts` + `alert_rules`, `conversations`, `settings`, `forensics`, `reports`, `baselines`, `health_scores`, `custom_workflows`, `incidents`, `report_rules`, `decisions_audit`, `slo_definitions`, `topology_devices`/`topology_connections`. All in `data/universalops.db`. Retention is configurable (default 7 days, pruned daily).

## Build From Source

### Prerequisites
- [Go 1.26+](https://go.dev/dl/)
- [Node.js 20+](https://nodejs.org/)
- [Wails CLI v2](https://wails.io/docs/gettingstarted/installation): `go install github.com/wailsapp/wails/v2/cmd/wails@latest`
- [Microsoft Edge WebView2 Runtime](https://developer.microsoft.com/microsoft-edge/webview2/)

### Development
```bash
git clone https://github.com/shahriarhaqueabir/UniversalOps.git
cd UniversalOps
wails dev   # hot reload
```

### Production Build
```bash
wails build
# Executable: build/bin/UniversalOps.exe
```

### Tests
```bash
go build ./... && go vet ./... && go test ./...
cd cmd/opsforall-gui/frontend && npm run build
```

## Download & Install (Recommended)

1. Download the latest Windows build from [Releases](https://github.com/shahriarhaqueabir/UniversalOps/releases).
2. Install the [WebView2 Runtime](https://developer.microsoft.com/microsoft-edge/webview2/) (preinstalled on modern Windows).
3. Run `UniversalOps.exe`. No installer required.

> Some features require Administrator privileges. Run as Administrator for remediation actions.

### Optional: Local AI with Ollama
```bash
ollama pull universalops   # or llama3, qwen2.5, mistral, ...
```
Set `OLLAMA_HOST`/`OLLAMA_MODEL` for non-default hosts. Or use `AI_PROVIDER=llama` with a GGUF model in your models directory.

## Contributing & Collaboration

UniversalOps is open source (MIT) and built for the community. Contributions welcome:

- **Bug reports / feature ideas**: open an [issue](https://github.com/shahriarhaqueabir/UniversalOps/issues) with repro steps.
- **Pull requests**: focused changes, clear descriptions, tests where possible.
- **Areas needing help**: test coverage (currently minimal), cross-platform hardening, new collectors, UI/UX polish, documentation, translations.
- **Security reports**: please open a private issue or contact the maintainer directly rather than public disclosure.

If you find this project useful, please ⭐ star the repo — it helps others discover it.

## License

[MIT](LICENSE) © UniversalOps

---

*Built with care for operators who value privacy, speed, and control over their own machines.*