# Flags & Review Notes — WinOps (WinOps)

## Changes applied (README playbook rewrite)

- Rewrote README.md using the playbook template.
- Added CI badge + created `.github/workflows/ci.yml` (Go build/vet/test).
- Added Screenshots section placeholder.
- Added TL;DR block.
- License set to MIT.

## Open flags

- [ ] CI workflow is basic (build/vet/test) — consider adding frontend build step and lint.
- [ ] "Go 1.26+" — verify Go version in go.mod.
- [ ] Screenshots: create `docs/screenshots/*.png` and replace placeholders.
- [ ] Add `LICENSE` file (MIT) if missing.
- [ ] Verify Wails v2 and React 19 versions still current.
- [ ] Test coverage is minimal — flag for improvement.
- [ ] Ollama model name (`winops`) — verify it exists or is a placeholder.
