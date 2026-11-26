# WinOps Workflows

This document catalogs the operational workflows shipped with WinOps, plus how
the workflow engine behaves. Workflows are multi-step, read-mostly operational
sequences you can run from **Workflows & Reports** in the UI (or via
`WorkflowAPI`). Each step is executed in order; the workflow **aborts on the
first failing step** (transactional by default).

## How a workflow runs

- `WorkflowAPI.ExecuteWorkflow(id)` looks up the definition, then iterates its
  steps.
- For each step:
  - If `step.Action` is set, it is invoked with a `context.Context`.
  - Else if `step.Type == powershell` and `step.Command` is set, the command
    runs via `devops.RunPowerShell`.
  - Otherwise the step errors out ("no action or command").
- On success, the step result is wrapped in a standardized envelope
  (`NewSuccessStepResult`): status, summary, duration, item count, timestamp,
  and raw data.
- On failure, the step is marked with the error, wrapped in
  `NewErrorStepResult`, and the workflow aborts with
  `workflow aborted at step "<id>": <err>`.
- All results are returned to the frontend for display in the Workflow Center.

## Step types

| Type | Meaning |
|------|---------|
| `query` | Read-only data collection (default for most steps). |
| `powershell` | Runs a PowerShell command (`step.Command`). |
| `action` | A remediation/operation action (gated by the safety handshake where applicable). |

## Categories

- `system` — OS, hardware, processes, services, storage.
- `security` — firewall, users, listeners, Defender, threats, compliance.
- `network` — connectivity, DNS, ports, traces, adapters.
- `devops` — developer tooling and environment checks.
- `intelligence` — AI-assisted cross-layer analysis.

## Built-in workflows (15)

| # | ID | Name | Category | Steps | Description |
|---|----|------|----------|-------|-------------|
| 1 | `diag-slow-pc` | Diagnose Slow PC | system | 3 | Identify resource bottlenecks and rogue processes. |
| 2 | `sec-sweep` | Quick Security Sweep | security | 2 | Verify basic system hardening and perimeter safety. |
| 3 | `core-maintenance` | Core Maintenance | system | 5 | Comprehensive system health sweep: resources, alerts, storage, processes, and services. |
| 4 | `net-diag` | Full Network Diagnostic | network | 5 | Multi-layered network sanity check (Ping, DNS, Ports, Trace, Conns). |
| 5 | `sec-audit` | Comprehensive Security Audit | security | 5 | Deep inspection of firewall, users, listeners, defender, and events. |
| 6 | `deep-health` | Deep System Health Check | system | 5 | Detailed hardware and OS health inspection. |
| 7 | `ai-audit` | AI System Auditor | intelligence | 2 | Multi-layered audit with AI-powered risk assessment. |
| 8 | `devops-audit` | DevOps Environment Audit | devops | 4 | Verify development tools, services, and process environment. |
| 9 | `ps-daily-ops` | Daily Operations Snapshot | system | 1 | Quick overview of OS, disk volumes, performance counters, and top processes. |
| 10 | `ps-system-review` | System Review | system | 1 | Comprehensive system information audit — OS, CPU, Memory, Disks, Volumes, BIOS, GPU, and uptime. |
| 11 | `ps-security-audit` | Security Surface Audit | security | 1 | Enumerate admin groups and verify Windows Defender protection status. |
| 12 | `ps-network-diagnostics` | Network Diagnostics | network | 1 | Detailed adapter configuration and active TCP connections snapshot. |
| 13 | `ps-threat-hunt` | Threat Hunting Primer | security | 1 | Surface suspicious listening sockets and recently modified system executables. |
| 14 | `ps-change-audit` | System Change Audit | system | 1 | Review recently installed programs from the Windows registry. |
| 15 | `ps-compliance-check` | Compliance Check | security | 1 | Audit password policy and account lockout settings. |

## Workflow details

### 1. Diagnose Slow PC (`diag-slow-pc`)
- **Why:** Context switching between Task Manager, Resource Monitor, and terminal is inefficient. This correlates CPU, RAM, and Disk I/O instantly.
- **Risks:** Minimal — collection has negligible overhead.
- **Typical values:** CPU < 70%, RAM < 85%, Disk < 90%.
- **Steps:**
  1. `check-cpu` — CPU Pressure Audit: collect per-core utilization and load averages.
  2. `check-mem` — Memory Saturation: measure RAM and swap usage.
  3. `top-procs` — Impactful Processes: filter top 10 processes by resource consumption.

### 2. Quick Security Sweep (`sec-sweep`)
- **Why:** Manually checking firewall, listeners, and defender status is error-prone.
- **Risks:** None — read-only operation.
- **Typical values:** Firewall ON, no 0.0.0.0 listeners without reason.
- **Steps:**
  1. `firewall-status` — Firewall State: check Domain, Private, and Public profile states.
  2. `check-listeners` — External Listeners: identify processes listening on all interfaces (0.0.0.0).

### 3. Core Maintenance (`core-maintenance`)
- **Why:** Routine maintenance prevents drift. Captures a full-system snapshot in one pass.
- **Risks:** None — read-only, zero side effects.
- **Typical values:** CPU < 70%, RAM < 85%, Disk < 90%, 0 critical alerts.
- **Steps:**
  1. `health-snapshot` — capture CPU, RAM, and Disk utilization at this moment.
  2. `alert-review` — enumerate all active security and system alerts.
  3. `storage-integrity` — verify disk partition health and free-space margins.
  4. `process-audit` — identify the top 15 processes by CPU and memory consumption.
  5. `service-health` — enumerate all running and stopped Windows services.

### 4. Full Network Diagnostic (`net-diag`)
- **Why:** Network issues are often multi-faceted. Correlates connectivity, resolution, and exposure in one pass.
- **Risks:** Minimal — port scan restricted to common local ports.
- **Typical values:** DNS resolves < 100 ms, Ping loss 0%.
- **Steps:**
  1. `ping-check` — ping default gateway and public endpoint (8.8.8.8).
  2. `dns-check` — resolve google.com via system DNS.
  3. `port-scan` — scan common ports on localhost for exposed services.
  4. `trace-route` — trace the route to 8.8.8.8.
  5. `conn-audit` — enumerate active TCP connections.

### 5. Comprehensive Security Audit (`sec-audit`)
- **Why:** Compliance and safety require a broad view of system posture.
- **Risks:** None — read-only enumeration.
- **Typical values:** Zero unauthorized listeners, Defender ACTIVE.
- **Steps:**
  1. `firewall-rules` — enumerate all firewall rules with direction and action.
  2. `user-audit` — list all local user accounts and admin group membership.
  3. `listeners` — identify processes with open network listeners.
  4. `defender` — verify Windows Defender real-time protection and signature age.
  5. `sec-events` — review recent security log events.

### 6. Deep System Health Check (`deep-health`)
- **Why:** Correlating CPU, Memory, Disk, and Processes in a single report identifies cross-resource contention.
- **Risks:** None.
- **Typical values:** CPU < 70%, RAM < 85%, Disk < 90%.
- **Steps:**
  1. `cpu-stats` — per-core utilization and load averages.
  2. `mem-stats` — RAM usage, available bytes, and swap analysis.
  3. `disk-usage` — partition utilization and free space.
  4. `system-info` — OS details, uptime, and process count.
  5. `top-procs` — identify top 10 processes by resource consumption.

### 7. AI System Auditor (`ai-audit`)
- **Why:** Traditional reports require human interpretation. Uses local AI to synthesize system, security, and network data into actionable findings.
- **Risks:** None — read-only, high CPU during AI processing.
- **Typical values:** AI Confidence > 80%.
- **Steps:**
  1. `collect-data` — gather system health, security posture, and network diagnostics.
  2. `ai-analyze` — send aggregated data to the local AI for cross-layer risk correlation.

### 8. DevOps Environment Audit (`devops-audit`)
- **Why:** Development environments can drift. Ensures Go, Git, and critical services are available and healthy.
- **Risks:** None.
- **Typical values:** Go/Git installed, core services RUNNING.
- **Steps:**
  1. `check-git` — verify Git CLI availability.
  2. `check-docker-k8s` — verify Docker daemon and kubectl cluster connectivity.
  3. `check-sdk` — verify Node.js and Go runtimes.
  4. `check-services` — enumerate running background services.

### 9. Daily Operations Snapshot (`ps-daily-ops`)
- **Why:** A single command that replaces four separate checks — ideal for a morning health check.
- **Risks:** None — read-only, lightweight counters.
- **Typical values:** CPU < 70%, Disk > 15% free.
- **Steps:**
  1. `run-daily-ops` — collect OS info, volume usage, performance counters, and top 5 processes.

### 10. System Review (`ps-system-review`)
- **Why:** One-shot inventory of every hardware and OS detail. Use before capacity planning or handoff.
- **Risks:** None — read-only CIM queries.
- **Typical values:** All components reported, no errors.
- **Steps:**
  1. `run-system-review` — execute full system info audit across all hardware and OS layers.

### 11. Security Surface Audit (`ps-security-audit`)
- **Why:** Quickly confirm who has admin access and whether real-time protection is active.
- **Risks:** None — read-only enumeration.
- **Typical values:** Defender ON, limited admin group membership.
- **Steps:**
  1. `run-sec-audit` — check administrator group membership and antivirus status.

### 12. Network Diagnostics (`ps-network-diagnostics`)
- **Why:** Correlate IP, DNS, gateway, and live connections to pinpoint network misconfigurations.
- **Risks:** Minimal — read-only netstat snapshot.
- **Typical values:** All adapters have valid IP, no unexpected listeners.
- **Steps:**
  1. `run-net-diag` — enumerate adapter details and capture top 20 TCP connections.

### 13. Threat Hunting Primer (`ps-threat-hunt`)
- **Why:** First-pass triage for compromised systems — unusual sockets and new binaries are strong indicators.
- **Risks:** Minimal — read-only enumeration.
- **Typical values:** No unexpected LISTENING sockets, System32 binaries unchanged.
- **Steps:**
  1. `run-threat-hunt` — scan for suspicious sockets and recently modified executables in System32.

### 14. System Change Audit (`ps-change-audit`)
- **Why:** Detect unauthorized or unexpected software installations.
- **Risks:** None — read-only registry query.
- **Typical values:** Only known software in install history.
- **Steps:**
  1. `run-change-audit` — query the registry for the 10 most recently installed programs.

### 15. Compliance Check (`ps-compliance-check`)
- **Why:** Verify that domain or local password policies meet security baselines.
- **Risks:** None — read-only policy query.
- **Typical values:** Password complexity enforced, lockout after 5 attempts.
- **Steps:**
  1. `run-compliance` — retrieve current password and lockout policy settings.

## Custom workflows

WinOps also supports user-defined workflows:

- `WorkflowAPI.SaveCustomWorkflow(def)` persists a `WorkflowDefinition` to
  SQLite (`UpsertCustomWorkflow`).
- `WorkflowAPI.ListWorkflows()` merges built-in definitions with any custom
  ones loaded from storage.
- `WorkflowAPI.DeleteCustomWorkflow(id)` removes a custom workflow.
- Custom workflows use the same step model and execution engine, so they
  benefit from the same standardized result envelopes and abort-on-error
  policy.

## Related files

- Engine and types: `internal/common/workflow.go`
- Registration and execution: `internal/app/Workflow.go`
- Frontend view: `cmd/winops-gui/frontend/src/pages/WorkflowCenter.tsx`
- Reports integration: `internal/app/Reports.go` / `ReportsAPI`
