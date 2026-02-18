# glab MCP Server

The MCP server is built into `glab` itself — no separate runtime or installation needed.
Start it with:

```bash
glab mcp serve
```

It exposes 19 GitLab tools over stdio using the [Model Context Protocol](https://modelcontextprotocol.io),
built with the official [`modelcontextprotocol/go-sdk`](https://github.com/modelcontextprotocol/go-sdk).

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

## Connect to Claude Desktop

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

## Connect to Claude Code

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

## Security

The server calls the GitLab API directly via the official Go client — no shell execution,
no argument interpolation, no injection surface.
