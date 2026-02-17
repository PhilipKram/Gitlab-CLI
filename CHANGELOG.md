# Changelog

## 0.2.0 — Full GitLab DevOps Coverage

42 commits bringing comprehensive GitLab platform coverage — from package registries to pipeline analytics to AI-powered workflows.

### ✨ New Features

- **📦 Package Registry Management** (#23) — list, view, delete, and download packages (npm, Maven, PyPI, NuGet, Conan, Composer, Helm, generic)
- **🐳 Container Registry Management** (#20) — list repositories, list/view/delete image tags, bulk tag deletion
- **🌍 Environment & Deployment Management** (#25) — list/view/stop/delete environments, list/view deployments
- **📊 Pipeline Analytics & Insights** (#26) — duration trends, success/failure rates, slowest jobs, flaky test detection
- **💬 Complete MR Review Workflow** (#15) — list discussions, reply to threads, inline code suggestions, resolve/unresolve threads
- **🤖 MCP Server with 39 Tools** (#4) — built-in Model Context Protocol server for AI integration
- **🧩 MCP Resources & Prompts** (#16) — 4 resources and 5 prompt templates for AI workflows
- **⚡ MCP install/uninstall/status** (#18) — one-command registration with Claude Code and Claude Desktop
- **🔧 Consistent JSON Output** (#24, #10) — every command supports `--format json` with consistent schemas
- **🔍 API Version Detection** (#21) — detect GitLab version and gracefully handle unsupported endpoints
- **📜 Virtual Scrolling** (#22) — windowed rendering for large collections
- **🔑 Complete Auth Lifecycle** (#12) — logout, status, token inspection, multi-instance switching, auto-refresh
- **⚙️ CI/CD Variable Management** (#7) — list, get, set, update, delete, export/import with masked/protected support
- **🚀 Enhanced Pipeline Commands** (#6) — job logs, retry/cancel individual jobs, download artifacts
- **💡 Actionable Error Messages** (#11) — structured errors with context and suggested resolutions
- **🌐 `--web` flag** (#27) for snippet and label commands

### 🔒 Security

- Mandatory checksum verification and URL validation during self-update (#28)
- HTTP client timeouts for OAuth, API, and asset downloads (#29)

### 🧪 Testing

- Integration & E2E test framework with mock GitLab API server (#19)
- Command-level test suite with 80%+ coverage (#9)
- Variable & MCP command test coverage (#17)

### 🐛 Bug Fixes

- `--branch` flag alias for pipeline run (#13)
- CLI compatibility improvements for flag aliases and API paths (#14)
- Resolved pagination deadlock and lint errors
- Fixed auth token format default

### 📚 Documentation

- Updated README with all new commands and examples
- Added CLAUDE.md project guide
- Updated docs site with new feature cards and command groups
- Updated MCP README with correct tool count (39)

---

### What's Changed

- docs: update README, CHANGELOG, docs site, MCP README, and add CLAUDE.md by @PhilipKramer in d8fbf2b
- fix: resolve lint errors and pagination deadlock by @PhilipKramer in ec170e2
- fix: remaining lint errors and auth token format default by @PhilipKramer in 608909b
- Add HTTP client timeouts to OAuth token exchange, raw API command, and asset downloads (#29) by @PhilipKramer in 219c3f2
- Pipeline Analytics & Insights (#26) by @PhilipKramer in 4958673
- Consistent JSON Output Across All Commands (#24) by @PhilipKramer in 9ac1370
- Package Registry Management (#23) by @PhilipKramer in a57172c
- Virtual Scrolling for Large Collections (#22) by @PhilipKramer in 9d97c1e
- API Version Detection & Graceful Degradation (#21) by @PhilipKramer in a33fe2a
- Container Registry Management (#20) by @PhilipKramer in 3b6bc49
- Integration & End-to-End Test Framework (#19) by @PhilipKramer in 661cffd
- Environment & Deployment Management (#25) by @PhilipKramer in f47c9d7
- Enforce mandatory checksum verification and URL validation during self-update (#28) by @PhilipKramer in c8c32c6
- Add --web flag to snippet and label commands (#27) by @PhilipKramer in 9f461e1
- Variable & MCP Command Test Coverage (#17) by @PhilipKramer in 6124ef5
- Complete Merge Request Review Workflow (#15) by @PhilipKramer in 70f7570
- MCP Server Resource & Prompt Expansion (#16) by @PhilipKramer in c2f82eb
- MCP install/uninstall/status subcommands (#18) by @PhilipKramer in 8694977
- feat: add MCP server exposing 33 GitLab tools (#4) by @PhilipKramer in 8a285db
- Enhanced Pipeline Commands (#6) by @PhilipKramer in 5183c52
- Command-Level Test Suite (#9) by @PhilipKramer in 5cc42d9
- Actionable Error Messages (#11) by @PhilipKramer in 19115aa
- Complete Auth Lifecycle Management (#12) by @PhilipKramer in 1cd00f1
- Universal JSON/Formatted Output (#10) by @PhilipKramer in b70b5ae
- fix: add --branch flag alias for pipeline run (#13) by @PhilipKramer in 1f775d2
- fix: CLI compatibility improvements for flag aliases and API paths (#14) by @PhilipKramer in 239f606
- CI/CD Variable Management (#7) by @PhilipKramer in 9a72742

### Thanks to all contributors

@PhilipKramer

---

## 0.1.0 — Initial Release

First public release of glab with core GitLab CLI functionality: authentication, merge requests, issues, repositories, pipelines, releases, snippets, labels, projects, configuration, shell completions, and self-update.

**Full Changelog**: https://github.com/PhilipKram/Gitlab-CLI/releases/tag/v0.1.0
