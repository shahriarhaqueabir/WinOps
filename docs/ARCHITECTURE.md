# Winops — Architecture

This document describes the internal architecture for contributors and reviewers.

## High-Level Overview

Winops is a Wails v2 desktop app. A Go backend exposes bound methods to a React/TypeScript frontend rendered in WebView2. All state is local (SQLite). A collector-based engine loop drives telemetry, alerting, and events at a 2-second cadence.

## Package Layout

```
Winops/
├── main.go / main_windows.go       # Wails entry, prereq checks
├── internal/
│   ├── app/                        # Wails-bound facades (App, SysOps, NetOps, SecOps, DevOps, AIOps, Dashboard, …)
│   ├── common/                     # Collectors, pipeline, alert engine, storage, sandbox, capability registry, events, knowledge
│   ├── sysops/                     # System operations (CPU, memory, disk, GPU, processes, services, logs, actions)
│   ├── netops/                     # Network operations (interfaces, DNS, ARP, Wi-Fi, scan, topology, health)
│   ├── secops/                     # Security operations (posture, firewall, defender, hardening, identity, forensics, IR)
│   ├── devops/                     # Docker, Kubernetes, packages, shell, services
│   └── aiops/                      # Ollama/llama client, provider, reporting, analyst prompt, workflows, state query
├── cmd/winops-gui/frontend/     # React + TS + Vite frontend
└── data/                           # Runtime data (gitignored): SQLite DB, models, baseline
```

## Core Subsystems

### Collector Registry & Scheduler
`internal/common/collector.go` + `registry.go` + `scheduler.go`. Collectors implement `Collector` interface with `Info()` and `Collect(ctx)`. The scheduler runs enabled collectors at configured intervals, pushes `MetricSample`s into the `DataPipeline`, and records status.

### DataPipeline
`internal/common/engine.go` + `pipeline.go`. Holds time-series per metric, computes trends/forecasts/window stats. Configurable tick interval, capacity, ping count, DNS timeout. Persisted via `PipelineAPI.PersistSettings`/`LoadSettings`.

### AlertEngine
`internal/common/alerts.go`. Rules (metric, condition gt/lt, threshold, flap count, severity, message). Default rules seeded at startup. Supports `AddRule`/`RemoveRule`/`GetRules` and persists full set to `alert_rules` (sync). Restores from DB on startup. Evaluation runs in the engine loop.

### EventBus & Timeline
`internal/common/events.go`. In-memory pub/sub (buffered 1000). Subscribers persist events to SQLite and (optionally) emit to frontend. `NewEvent`/`NewEventWithMeta` with `ScrubSecrets`.

### Storage (SQLite)
`internal/common/storage.go`. modernc.org/sqlite, WAL mode, 2 connections, batched async writes (metrics 512, logs 256), daily prune (configurable retention), weekly VACUUM if fragmented. Schema versioned (v1–v5). Tables: metrics, logs, events, alerts, alert_rules, conversations, settings, forensics, reports, baselines, health_scores, custom_workflows, incidents, report_rules, decisions_audit, slo_definitions, topology_devices/connections.

### EngineLoop
`internal/common/engine.go` (EngineLoop). 2-second tick: collects metrics, emits `EventLiveMetrics` to frontend, evaluates alerts, persists events, triggers reports via `ReportTrigger`. Decoupled via callbacks (`OnMetricsEmit`, `OnAlertsEmit`).

### Sandbox
`internal/common/sandbox*.go`. `SandboxConfig` (deny network, read-only FS, deny process spawn, drop privileges). Platform implementation (Windows): restricted tokens, disabled dangerous privileges, low integrity level, Job Objects. `DefaultSandbox`, `SystemQuerySandbox`, `RemediationSandbox` presets.

### Capability Registry
`internal/common/capability.go`. Probes PATH for tools (nmap, docker, kubectl, ollama, powershell, etc.) with user overrides, background refresh, version detection. Used by DevOps to gate features.

### Knowledge Layer
`internal/common/knowledge.go`. Maintains a system state snapshot (security grade, alert count, connections, uptime, interfaces) consumed by AIOps for grounded prompts.

### AIOps
`internal/app/AIOps.go` + `internal/aiops/`. Provider abstraction (Ollama HTTP client or in-process llama.cpp). Builds analyst prompt from knowledge + metric history + events + anomalies. Streams tokens. Supports tool-aware workflows via `<action_request>` tags requiring explicit confirmation (safety handshake). Persists conversations per session. Model/persona management via Ollama create/delete.

### Safety Handshake
`internal/common/handshake.go` + `App.ConfirmAction`. AI-suggested actions (kill process, block IP, isolate host, hardening, service restart, system actions, workflows) are registered as pending and require explicit user confirmation. Approved actions are audited in `decisions_audit`.

## Data Flow

```
Collectors → MetricSample → DataPipeline (time-series)
                ↓
            EngineLoop (2s)
                ├──► OnMetricsEmit → frontend (live metrics) + Knowledge.UpdateSystemState
                ├──► AlertEngine.Evaluate → fired alerts → OnAlertsEmit → frontend + persist
                └──► EventBus → persist events → (optional) frontend
```

## Security Model

- **Local-only**: no outbound network except optional update check and on-demand LHM download.
- **Sandboxed commands**: restricted tokens, low integrity, Job Objects.
- **Input validation**: all untrusted values validated before `exec`.
- **Parameterized SQL**: no string interpolation of values.
- **Secret scrubbing**: `ScrubSecrets` redacts credentials in persisted events.
- **Audit trail**: `decisions_audit` logs all approved remediations.

## Frontend Architecture

- React 19 + TypeScript, Vite build, Tailwind CSS 4.
- Wails-generated bindings in `wailsjs/go/` (regenerated on build).
- TanStack Query for server-state caching with refetch intervals.
- Zustand for client state (config, metrics, ollama, settings).
- Pages under `src/pages/` per module; tabs under `src/pages/{module}/`.

## Build & Release

- `wails build` produces `build/bin/Winops.exe` (~36 MB).
- Frontend embedded via `//go:embed all:cmd/winops-gui/frontend/dist`.
- No CGO (pure-Go SQLite) for reproducible builds.

## Extension Points

- **New collector**: implement `Collector`, register in `RegisterCollectors`.
- **New module**: add facade in `internal/app/`, bind in `main.go`.
- **New alert rule**: `AlertEngine.AddRule` + `AlertAPI.AddRule`.
- **New workflow**: register in `WorkflowEngine`.
- **New AI action**: add case in `App.ConfirmAction` and tool definition in aiops.