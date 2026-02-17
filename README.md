# glab - GitLab CLI

Work seamlessly with GitLab from the command line.

`glab` is a CLI tool for GitLab, written in Go. It follows the same interface patterns as the GitHub CLI (`gh`), providing a familiar experience for managing merge requests, issues, pipelines, repositories, and more.

**Documentation:** [philipkram.github.io/Gitlab-CLI](https://philipkram.github.io/Gitlab-CLI/)

## Installation

### Homebrew (macOS & Linux)

```bash
brew tap PhilipKram/glab https://github.com/PhilipKram/Gitlab-CLI
brew install glab
```

### Binary releases

Download pre-built binaries for Linux, macOS, and Windows (amd64/arm64) from the [releases page](https://github.com/PhilipKram/Gitlab-CLI/releases).

### Go install

```bash
go install github.com/PhilipKram/gitlab-cli@latest
```

### Build from source

```bash
git clone https://github.com/PhilipKram/Gitlab-CLI.git
cd Gitlab-CLI
make build
# Binary available at ./bin/glab
```

## Authentication

### Interactive login (recommended)

Just run `glab auth login` and follow the prompts — the same experience as `gh auth login`:

```
$ glab auth login
? What GitLab instance do you want to log into?
  [1] gitlab.com
  [2] GitLab Self-managed
  Choice: 1
? What is your preferred protocol for Git operations on this host?
  [1] HTTPS
  [2] SSH
  Choice: 1
? How would you like to authenticate glab?
  [1] Login with a web browser
  [2] Paste a token
  Choice: 1
? OAuth Application ID: <your-app-id>

! Opening gitlab.com in your browser...
- Waiting for authentication...
- glab config set -h gitlab.com git_protocol https
✓ Logged in to gitlab.com as username
```

### OAuth (browser-based login)

Authenticate via OAuth in the browser. You first need to create an OAuth application
in your GitLab instance under **Settings > Applications** with:
- Redirect URI: `http://127.0.0.1` (any port)
- Scopes: `api`, `read_user`, `read_repository`

```bash
# Interactive OAuth login
glab auth login --web --client-id <your-app-id>

# OAuth with a self-hosted instance
glab auth login --web --client-id <your-app-id> --hostname gitlab.example.com
```

The CLI starts a local server, opens your browser for authorization, and
automatically exchanges the code for a token using PKCE.

### Token-based login

```bash
# Login with a personal access token
glab auth login --token glpat-xxxxxxxxxxxxxxxxxxxx

# Login to a self-hosted GitLab instance
glab auth login --hostname gitlab.example.com --token glpat-xxxx

# Pipe a token from a file or secret manager
glab auth login --stdin < token.txt

# Check authentication status
glab auth status

# Environment variable authentication
export GITLAB_TOKEN="glpat-xxxxxxxxxxxxxxxxxxxx"
```

Required token scopes: `api`, `read_user`, `read_repository`

### Auth login flags

| Flag | Description |
|------|-------------|
| `--hostname, -h` | GitLab hostname (default: gitlab.com) |
| `--token, -t` | Personal access token |
| `--web, -w` | Authenticate via OAuth in the browser |
| `--client-id` | OAuth application ID (required with `--web`) |
| `--git-protocol, -p` | Preferred git protocol: `https` or `ssh` |
| `--stdin` | Read token from standard input |

## Commands

### Core Commands

| Command | Description |
|---------|-------------|
| `glab auth` | Authenticate glab and git with GitLab |
| `glab mr` | Manage merge requests |
| `glab issue` | Manage issues |
| `glab repo` | Manage repositories |

### CI/CD Commands

| Command | Description |
|---------|-------------|
| `glab pipeline` | Manage pipelines and CI/CD |
| `glab release` | Manage releases |

### Additional Commands

| Command | Description |
|---------|-------------|
| `glab snippet` | Manage snippets |
| `glab label` | Manage labels |
| `glab project` | Manage projects |

### Utility Commands

| Command | Description |
|---------|-------------|
| `glab api` | Make authenticated API requests |
| `glab browse` | Open project in browser |
| `glab config` | Manage configuration |
| `glab completion` | Generate shell completion scripts |

## Usage Examples

### Merge Requests

```bash
glab mr create --title "Add feature" --description "Details" --draft
glab mr list --state opened
glab mr view 123
glab mr merge 123 --squash
glab mr approve 123
glab mr checkout 123
glab mr diff 123
```

### Issues

```bash
glab issue create --title "Bug report" --label bug --assignee @user1
glab issue list --state opened --author johndoe
glab issue view 42
glab issue close 42
glab issue comment 42 --body "Fixed in !123"
```

### Pipelines

```bash
glab pipeline list
glab pipeline run --ref main
glab pipeline view 12345
glab pipeline jobs 12345
glab pipeline cancel 12345
```

### Repositories

```bash
glab repo clone owner/repo
glab repo create my-project --public --init
glab repo fork owner/repo --clone
glab repo view
glab repo list --owner my-group
```

### Configuration

```bash
glab config set protocol ssh
glab config set editor vim
glab config list
```

### Direct API Access

```bash
glab api projects
glab api users --method GET
glab api projects/:id/issues --method POST --body '{"title":"Bug"}'
```

## Configuration

Configuration is stored in `~/.config/glab/`. Override with `GLAB_CONFIG_DIR`.

| Key | Description | Default |
|-----|-------------|---------|
| `editor` | Preferred text editor | - |
| `pager` | Preferred pager | - |
| `browser` | Preferred web browser | - |
| `protocol` | Git protocol (https/ssh) | https |
| `git_remote` | Default git remote name | origin |

## Environment Variables

| Variable | Description |
|----------|-------------|
| `GITLAB_TOKEN` | Authentication token |
| `GLAB_TOKEN` | Authentication token (alternative) |
| `GITLAB_HOST` | Default GitLab hostname |
| `GLAB_CONFIG_DIR` | Configuration directory |

## Releasing

Releases are automated via [GoReleaser](https://goreleaser.com/) and GitHub Actions.

To create a release:

```bash
git tag v0.1.0
git push origin v0.1.0
```

This will:
1. Run tests
2. Build cross-platform binaries (linux/darwin/windows, amd64/arm64)
3. Create a GitHub Release with archives and checksums
4. Update the Homebrew formula in the `HomebrewFormula` directory
5. Publish deb/rpm packages

### Shell Completions

```bash
# Bash
source <(glab completion bash)

# Zsh
glab completion zsh > "${fpath[1]}/_glab"

# Fish
glab completion fish | source
```

## License

MIT
