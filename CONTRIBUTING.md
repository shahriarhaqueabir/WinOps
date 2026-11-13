# Contributing to Winops

Thanks for your interest in contributing!

## Ways to Contribute
- **Report bugs**: Open an issue with steps to reproduce, OS version, and logs if possible.
- **Suggest features**: Describe the use case and why it fits a local-first ops tool.
- **Improve docs**: Fix typos, add screenshots, or clarify instructions.
- **Submit code**: Bug fixes, tests, and small focused improvements are welcome.

## Development Setup

1. Install prerequisites: Go 1.26+, Node 20+, Wails CLI v2, WebView2.
2. Clone the repo: `git clone https://github.com/shahriarhaqueabir/Winops.git`
3. Run dev: `wails dev`
4. Build: `wails build`

## Guidelines
- **Keep it local-first**: Avoid adding telemetry, external network calls, or cloud dependencies.
- **Security aware**: Don't log secrets/keys. Validate inputs for any command execution paths.
- **Focused PRs**: Keep changes small and explain the rationale.
- **Follow existing patterns**: Match Go/TypeScript style and module boundaries in the codebase.
- **Test where reasonable**: Add tests for pure logic when easy to do.

## Pull Requests
1. Fork the repo and create a feature branch.
2. Make your changes with clear commits.
3. Open a PR against `main` with a description of what/why.
4. Be responsive to review feedback.

By contributing, you agree that your contributions will be licensed under the MIT License.
