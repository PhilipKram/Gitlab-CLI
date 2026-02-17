# glab - GitLab CLI

## Project Overview

A Go-based CLI tool for GitLab that follows GitHub CLI (`gh`) interface patterns. Binary name: `glab`.

## Tech Stack

- **Language:** Go 1.24
- **CLI framework:** [cobra](https://github.com/spf13/cobra)
- **GitLab API client:** [gitlab.com/gitlab-org/api/client-go](https://gitlab.com/gitlab-org/api/client-go)
- **MCP server:** [modelcontextprotocol/go-sdk](https://github.com/modelcontextprotocol/go-sdk)
- **OAuth:** golang.org/x/oauth2
- **Build:** Makefile + GoReleaser
- **Linter:** golangci-lint

## Project Structure

```
main.go                  # Entry point, passes version to root command
cmd/                     # All CLI commands (one file per command group)
  root.go                # Root command, registers all subcommands
  auth.go                # Authentication (login, logout, status, token)
  mr.go                  # Merge requests
  issue.go               # Issues
  repo.go                # Repositories
  pipeline.go            # Pipelines + job management
  pipeline_stats.go      # Pipeline analytics: stats
  pipeline_trends.go     # Pipeline analytics: duration trends
  pipeline_slowest_jobs.go # Pipeline analytics: slowest jobs
  pipeline_flaky.go      # Pipeline analytics: flaky job detection
  release.go             # Releases
  variable.go            # CI/CD variables
  package.go             # Package registries
  registry.go            # Container registries
  environment.go         # Environments
  deployment.go          # Deployments
  snippet.go             # Snippets
  label.go               # Labels
  project.go             # Projects
  api.go                 # Raw API requests
  browse.go              # Open in browser
  config.go              # Configuration management
  completion.go          # Shell completions
  mcp.go                 # MCP server command (serve, install, uninstall, status)
  upgrade.go             # Self-update
internal/
  api/                   # GitLab API client wrapper
  auth/                  # Token storage, OAuth flow, refresh
  browser/               # Open URLs in browser
  cmdtest/               # Test helpers
  cmdutil/               # Factory, IO streams, shared utilities
  config/                # Config file management (~/.config/glab/)
  errors/                # Structured error handling with actionable messages
  formatter/             # JSON/table/plain output formatting
  git/                   # Git remote detection
  mcp/                   # MCP server implementation
    server.go            # Server setup and registration
    tools/               # 39 MCP tools (issue, mr, pipeline, etc.)
    resources/           # 4 MCP resources (README, CI config, MR diff, job log)
    prompts/             # 5 MCP prompt templates
  prompt/                # Interactive prompts
  tableprinter/          # Table output
  update/                # Version check and upgrade logic
  version/               # API version detection
mcp/README.md            # MCP server documentation
docs/index.html          # Documentation website (GitHub Pages)
```

## Key Patterns

- **Command structure:** `glab <noun> <verb> [flags]` (e.g., `glab mr create --title "..."`)
- **Factory pattern:** `cmdutil.Factory` provides shared dependencies (API client, config, IO)
- **Output formats:** All commands support `--format json|table` and `--json` flags via the `formatter` package
- **Error handling:** Structured errors with actionable suggestions via `internal/errors`
- **Host resolution:** Git remote -> config default host -> first authenticated host
- **Auth:** OAuth with PKCE (default) or personal access tokens; auto-refresh on expiry

## Development

```bash
make build          # Build binary to bin/glab
make test           # Run unit tests
make test-coverage  # Tests with coverage report
make lint           # Run golangci-lint
make fmt            # Format code
make vet            # Go vet
make snapshot       # Cross-platform build (no publish)
```

## Testing

- Unit tests colocated with source (`cmd/*_test.go`)
- Test helpers in `internal/cmdtest/`
- Integration tests in `tests/integration/`
- E2E tests in `tests/e2e/`

## Guidelines

- Follow existing command patterns when adding new commands
- All commands must support `--format json` output
- Use `internal/errors` for user-facing error messages
- Run `make lint` before committing
- Do not commit or push without explicit user confirmation
