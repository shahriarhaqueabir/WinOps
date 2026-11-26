# WinOps Cleanup — Design Spec

Date: 2026-10-28
Scope: Four sub-projects executed in order: (1) commit hygiene, (2) populate partial features, (3) remove LibreHardwareMonitor, (4) remove cross-platform leftovers.

## Sub-project 1 — Commit hygiene (by concern)

Goal: turn the dirty working tree (39 modified tracked files + untracked source/docs) into a clean, logically-grouped history on top of the single "Initial commit".

Approach: stage only source/config/doc files by concern; never stage runtime artifacts (`data/`, `logs/`, `build/`, `node_modules/`, `*.db`, `*.gguf`). Add `*.gguf` to `.gitignore`. Commit groups:

1. `chore: project docs, changelog, contributing`
2. `chore: dependency updates (go.mod/go.sum)`
3. `feat: onboarding flow`
4. `feat: common infrastructure (sandbox, storage, platform, capability, events)`
5. `feat: sysops module`
6. `feat: netops module`
7. `feat: secops module`
8. `feat: devops module`
9. `feat: aiops module`
10. `feat: app layer (App, AIOps, DevOps, SecOps, Types)`
11. `test: ollama client`
12. `chore: gitignore gguf models`

Each commit must build (`go build ./...`) and frontend must build where touched. No behavioral changes in this sub-project — purely history organization.

## Sub-project 2 — Populate partial features

Scope (user-selected): alert rules, settings persistence, conversations (chat history), network topology.

- **Alert rules**: Wire `AlertRuleRecord` CRUD (already in storage) to `AlertEngine` so custom rules are loaded at startup, evaluated each tick, and editable from UI (Settings/Alerts). Add default-rule seeding only when DB empty.
- **Settings persistence**: Replace in-memory/default settings with `storage.GetSetting`/`UpsertSetting` for `autoEcoMode`, `retention_days`, storage paths, model dir, log level. Load at startup in `PipelineAPI.LoadSettings` + `App.Startup`.
- **Conversations**: Persist AIOps chat via `InsertMessage`/`GetMessages`/`ListSessions`/`DeleteSession`; load last session on chat open; add session switcher in UI.
- **Topology**: Persist devices/connections via `topology_devices`/`topology_connections`; wire NetOps topology tab to CRUD; auto-populate from discovery where possible.

Each feature gets: storage calls, app facade methods, frontend tab updates, and at least one smoke test where feasible.

## Sub-project 3 — Remove LibreHardwareMonitor

- Delete `internal/common/lhm.go`.
- Remove LHM manager calls in `App.Startup`/`Shutdown` (`GetLHMManager().Stop()`), hardware tab bindings, and `CapLHM` capability (remove from capability registry + frontend).
- Remove docs references (USAGE/FAQ if any).
- Hardware sensors fall back to gopsutil/WMI only; UI shows "sensors unavailable" instead of LHM prompt.
- Remove `lhmDownloadURL`/`lhmLicenseURL` and any download code.

## Sub-project 4 — Remove cross-platform leftovers

- Delete Linux/macOS-only code paths and commands: `sudo`, `systemctl`, `apt-get`, `dmesg`, `journalctl`, `crontab`, `iwlist`, `resolvectl`, `ip`, `dhclient`, `fstrim`, `rm -rf`, `sh -c`.
- Remove `IsLinux`/`IsMacOS` usage where only Windows remains; keep `IsWindows` checks but simplify. Keep platform.go minimal (Windows + generic).
- Remove darwin/linux build-tagged files if any (none found besides lhm which is windows-tagged).
- Keep `IsAdmin` Windows path; remove Unix branch.
- Update `GetBaseDir`/`ConfigDir` to Windows-standard (AppData) only.
- Remove dangerous `rm -rf /tmp/*` entirely.
- Ensure `go build` (windows) and `go vet` pass; no `//go:build linux/darwin` files remain (except none).

## Constraints

- Windows-only target (amd64).
- No new external dependencies unless necessary.
- Preserve local-first/no-telemetry north star.
- Each sub-project must leave the build green.

## Execution order

1 → 2 → 3 → 4, each with its own spec/plan/implementation cycle and verification (build + vet + tests + frontend build).