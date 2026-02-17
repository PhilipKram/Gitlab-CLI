# glab MCP Server

The MCP server is built into `glab` itself — no separate runtime or installation needed.
Start it with:

```bash
glab mcp serve
```

It exposes 39 GitLab tools, 4 resource types, and 5 prompt templates over stdio using the [Model Context Protocol](https://modelcontextprotocol.io),
built with the official [`modelcontextprotocol/go-sdk`](https://github.com/modelcontextprotocol/go-sdk).

## Quick Start

Register `glab` as an MCP server in Claude Code with a single command:

```bash
glab mcp install
```

That's it. The command auto-detects the `glab` binary path and registers it with Claude Code at user scope. To verify:

```bash
glab mcp status
```

To remove the registration:

```bash
glab mcp uninstall
```

### Command Reference

#### `glab mcp install`

Registers `glab` as an MCP server with your AI client.

```bash
# Install for Claude Code (default, user scope)
glab mcp install

# Install for a specific project only
glab mcp install --scope project

# Install for Claude Desktop instead
glab mcp install --client claude-desktop
```

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--scope` | `user` | Where to store the config: `user` (global), `local` (workspace), or `project` (`.claude/mcp.json`) |
| `--client` | `claude-code` | Target AI client: `claude-code` or `claude-desktop` |

#### `glab mcp uninstall`

Removes the `glab` MCP server registration.

```bash
# Uninstall from Claude Code (default, user scope)
glab mcp uninstall

# Uninstall from a specific scope
glab mcp uninstall --scope project

# Uninstall from Claude Desktop
glab mcp uninstall --client claude-desktop
```

**Flags:** Same as `install` (`--scope`, `--client`).

#### `glab mcp status`

Checks whether `glab` is currently registered as an MCP server.

```bash
glab mcp status

# Check Claude Code specifically
glab mcp status --client claude-code
```

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--client` | `claude-code` | Target AI client to check: `claude-code` or `claude-desktop` |

## Tools

| Category | Tools |
|----------|-------|
| **Merge Requests** | `mr_list`, `mr_view`, `mr_diff`, `mr_comment`, `mr_approve`, `mr_merge`, `mr_close`, `mr_reopen`, `mr_create`, `mr_edit` |
| **Issues** | `issue_list`, `issue_view`, `issue_create`, `issue_close`, `issue_reopen`, `issue_comment`, `issue_edit`, `issue_delete` |
| **Pipelines** | `pipeline_list`, `pipeline_view`, `pipeline_run`, `pipeline_cancel`, `pipeline_retry`, `pipeline_delete`, `pipeline_jobs`, `pipeline_job_log` |
| **Repositories** | `repo_list`, `repo_view` |
| **Releases** | `release_list`, `release_view`, `release_create`, `release_delete` |
| **Labels** | `label_list`, `label_create`, `label_delete` |
| **Snippets** | `snippet_list`, `snippet_view`, `snippet_create`, `snippet_delete` |

## Resources

MCP resources provide read-only access to GitLab project files and data. AI clients can read these resources
to gather context for answering questions or generating content.

| Resource | URI Template | Description |
|----------|--------------|-------------|
| **Project README** | `gitlab:///{repo}/README.md` | Read the project's README file |
| **CI Configuration** | `gitlab:///{repo}/.gitlab-ci.yml` | Read the project's GitLab CI/CD configuration |
| **MR Diff** | `gitlab:///{repo}/mr/{mr}/diff` | Read the diff for a specific merge request |
| **Pipeline Job Log** | `gitlab:///{repo}/pipeline/{pipeline}/job/{job}/log` | Read logs from a specific pipeline job |

**Example URIs:**
- `gitlab:///gitlab-org/gitlab/README.md`
- `gitlab:///gitlab-org/gitlab/.gitlab-ci.yml`
- `gitlab:///gitlab-org/gitlab/mr/123/diff`
- `gitlab:///gitlab-org/gitlab/pipeline/456/job/789/log`

## Prompts

MCP prompts provide pre-built templates for common GitLab workflows. AI clients can use these prompts
to guide structured interactions with GitLab data.

| Prompt | Arguments | Description |
|--------|-----------|-------------|
| **review_mr** | `repo` (optional), `mr_id` (required) | Structured MR review guidance covering code quality, tests, security, and performance |
| **explain_pipeline_failure** | `repo` (optional), `pipeline_id` (required), `job_id` (optional) | Analyze pipeline failures with log analysis and root cause investigation |
| **summarize_issues** | `repo` (optional), `state` (optional), `labels` (optional) | Summarize issues with pattern identification and priority breakdown |
| **draft_mr_description** | `repo` (optional), `source_branch` (required), `target_branch` (optional) | Generate MR descriptions from commit history and code changes |
| **create_release_notes** | `repo` (optional), `from_tag` (required), `to_tag` (optional) | Generate release notes from MRs, issues, and commits |

## Prerequisites

`glab` installed and authenticated — no Node.js, no extra dependencies.

## Authentication

Authenticate once before using the MCP server. OAuth (recommended) or a personal access token both work.

**OAuth (recommended):**
```bash
glab auth login --web --client-id <your-client-id> --hostname gitlab.example.com
```

**Personal access token:**
```bash
glab auth login --hostname gitlab.example.com
```

Verify:
```bash
glab auth status
```

`GITLAB_TOKEN` environment variable is **optional** — only needed if not using `glab auth login`.

## Project detection

Run `glab mcp serve` from inside a git repository to auto-detect the project from the remote.
Pass `--repo` to specify it explicitly for any tool call:

```bash
glab -R gitlab.example.com/owner/repo mcp serve
```

Each tool also accepts a `repo` parameter in `OWNER/REPO` or `HOST/OWNER/REPO` format.

---

## Connect to GitHub Copilot CLI

Add to `~/.copilot/mcp-config.json`:

```json
{
  "mcpServers": {
    "glab": {
      "command": "glab",
      "args": ["mcp", "serve"]
    }
  }
}
```

Or pass inline for a single session:

```bash
copilot --additional-mcp-config '{"mcpServers":{"glab":{"command":"glab","args":["mcp","serve"]}}}'
```

---

## Connect to Claude Code

The recommended way is the automated install command:

```bash
glab mcp install
```

See [Quick Start](#quick-start) for details and options.

### Manual Configuration

Add to your project's `.claude/mcp.json`:

```json
{
  "mcpServers": {
    "glab": {
      "command": "glab",
      "args": ["mcp", "serve"]
    }
  }
}
```

Or register globally via the Claude Code CLI:

```bash
claude mcp add glab -- glab mcp serve
```

---

## Connect to Claude Desktop

The recommended way is:

```bash
glab mcp install --client claude-desktop
```

### Manual Configuration

Add to `~/Library/Application Support/Claude/claude_desktop_config.json` (macOS)
or `%APPDATA%\Claude\claude_desktop_config.json` (Windows):

```json
{
  "mcpServers": {
    "glab": {
      "command": "glab",
      "args": ["mcp", "serve"]
    }
  }
}
```

Restart Claude Desktop after editing the config.

---

## Security

The server calls the GitLab API directly via the official Go client — no shell execution,
no argument interpolation, no injection surface.
