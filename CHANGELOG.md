# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.6.4] - 2026-10-26

### Added
- Local-first Windows operations dashboard with SysOps, NetOps, SecOps, DevOps, and AIOps modules
- Ollama integration with direct HTTP client and llama.cpp fallback support
- Workflow engine, reports, forensics, and security posture scoring
- Native Wails v2 desktop UI with React + TypeScript

### Security
- Removed Ollama SDK dependency; hardened subprocess execution and file handling
- Added security scanning and dependency hygiene improvements
