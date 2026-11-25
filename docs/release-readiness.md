# Release Readiness / Desired State

- **Repo:** `private/WinOps`
- **Owner:** `TBD`
- **Status:** `draft`
- **Last verified:** `2026-11-25` (recon only; no runtime verification)
- **Target platforms:** `TBD`
- **Distribution modes:** `source checkout | release build` (decide per repo)

## 0) Recon snapshot (auto-scaffolded)

> Groundwork only. Verify before treating as fact.

- **Path:** `E:\Projects\repos\private\WinOps`
- **Branch:** `main`
- **Dirty files:** `3` (do not include accidental WIP in release snapshot)
- **Remote:** `https://github.com/shahriarhaqueabir/WinOps.git`
- **Detected markers:** `go.mod`
- **Workflows:** `ci.yml`
- **License files:** `LICENSE`
- **Has docs/tests/src:** `yes/no/no`
- **Top dirs:** `.github,.memory,.ruff_cache,cmd,data,docs,internal,llama-b10952-bin-win-cuda-x64,logs,private`
- **Package scripts (if any):** `(none)`
- **README lead:** `# WinOps - Local-First Windows Operations Dashboard / <p align="center"> /   <img src="WinOps.jpg" alt="WinOps" width="400"> / </p> / > **SysOps, NetOps, SecOps, DevOps, and AIOps in one native Windows desktop / > app** - with Ollama-powered local AI. All analysis runs on your machine. No`
- **Recon note:** Large binaries/models present; define thin source release vs fat portable/release artifact.

## 1) User promise

- **Source checkout:** `go build` / Wails dev flow - verify from README/build docs; README shows build from source and portable zip recipe.
- **Release build:** README mentions download/install recommended and portable zip recipe; CI exists but release workflow not detected by name.
- **First-run experience:** `TBD - detect -> diff -> provision missing/drifted -> verify -> handoff`
- **Non-goals:** `TBD - what this release does not provision/support`

## 2) Desired-state manifest

| Component | Required | Version/Constraint | Source of truth | Install/Provision (idempotent) | Verify | Notes/Rollback |
|---|---:|---|---|---|---|---|
| OS/runtime | TBD | TBD | README/docs/script | Detect; install or guide; never silently destructive | command/check |  |
| Toolchain/package manager | TBD | TBD | go.mod | idempotent install | `TBD` |  |
| App dependencies | TBD | lockfile/manifest | repo manifest | install only when missing/drifted | `TBD` |  |
| External services/CLIs | TBD | TBD | docs/scripts | preserve user config; install fallback only if empty | health check |  |
| Models/assets/binaries | TBD | hash/size/version | manifest/LFS/release asset | download/verify only missing/drifted | hash + smoke |  |
| Config/secrets | yes | no secrets committed | env/config docs | create example/default only when missing; never overwrite | secret scan |  |
| Network/ports/permissions | TBD | TBD | docs/code | document; request minimal permissions | smoke |  |

**Recon provisioning hints:** Go toolchain for source build, WebView2/runtime deps, optional local AI model/runtime (llama binaries/models), config/data dirs.

## 3) Provision flow (XaaS-style)

**Flow:** `detect -> diff -> provision missing/drifted -> verify -> handoff`

- **Detect:** `TBD - script/check that inspects current machine/repo state`
- **Diff:** missing, wrong version, unhealthy, or drifted from desired state.
- **Provision:** only missing/drifted items; safe to rerun; user-visible actions.
- **Verify:** required checks pass before user handoff.
- **Handoff:** user-driven "Test all systems / Go" or explicit ready state.

### Verification checklist

| Check | Required | Command/Method | Pass criteria | Status |
|---|---:|---|---|---|
| Bootstrap/install | yes | `TBD` | completes cleanly on fresh env | TBD |
| App starts | yes | `TBD` | process/window/service healthy | TBD |
| Core smoke flow | yes | `TBD` | primary user flow works | TBD |
| Existing tests/build | if present | `go test ./...` (UNVERIFIED), build smoke, app launch smoke on Windows. | exit 0 / expected pass count | TBD |
| Cleanup/stop/cancel | if applicable | `TBD` | no orphan process/resource leak | TBD |

## 4) Clean snapshot / release contents

### Include (candidate)
- `README*`, `LICENSE*`, user docs, required source/runtime assets, supported install/run scripts, tests/verification scripts you support.

### Exclude (candidate)
- `.memory/`, `private/`, `logs/`, `data/` if user-specific, large model/runtime caches not needed for source-only snapshot, `.git/`.
- Add repo-specific excludes after review.

### Packaging notes
- **Source checkout artifact:** `TBD`
- **Release artifact:** `TBD`
- **Integrity:** `TBD - SHA256/signature/provenance`
- **Size/bandwidth:** `TBD - call out large runtimes/models/assets`

## 5) Security / config hygiene

- [ ] No credentials/tokens/PII/personal paths in tracked release files.
- [ ] Existing user config preserved; defaults installed only when missing.
- [ ] Download URLs, hashes, and permissions explicit.
- [ ] Telemetry/network behavior documented (or explicitly none).
- [ ] Logs do not leak secrets.

## 6) Licensing / redistribution

| Component | License/Terms | Bundled/Downloaded | Redistribution OK | Action |
|---|---|---|---:|---|
| Repo code | LICENSE | n/a | TBD | verify |
| Third-party deps/runtime | TBD | TBD | TBD | inventory |
| Models/assets/fonts/binaries | TBD | TBD | TBD | verify before release |

**Recon note:** LICENSE present - verify bundled llama/model/runtime redistribution terms.

## 7) Open decisions / blockers

- [ ] Confirm source-checkout vs release-build user promise.
- [ ] Define desired-state manifest and idempotent provision checks.
- [ ] Run verification commands and record evidence.
- [ ] Confirm clean snapshot include/exclude list.
- [ ] Complete licensing/redistribution review.
