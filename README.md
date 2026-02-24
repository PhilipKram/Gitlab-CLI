# glab - GitLab CLI

Work seamlessly with GitLab from the command line.

`glab` is a CLI tool for GitLab, written in Go. It follows the same interface patterns as the GitHub CLI (`gh`), providing a familiar experience for managing merge requests, issues, pipelines, repositories, and more.

**Documentation:** [philipkram.github.io/Gitlab-CLI](https://philipkram.github.io/Gitlab-CLI/)

## Installation

### Homebrew (macOS & Linux)

```bash
brew tap PhilipKram/tap https://github.com/PhilipKram/Gitlab-CLI
brew install PhilipKram/tap/glab
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

### OAuth login (default)

`glab` authenticates via OAuth by default. Just run `glab auth login` and it opens
your browser — no flags needed.

**First run** — prompts for host, protocol, and OAuth application ID, then stores everything:

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
? OAuth Application ID: <your-app-id>

! Opening gitlab.com in your browser...
- Waiting for authentication...
✓ Logged in to gitlab.com as username
```

**Subsequent runs** — all settings are remembered, goes straight to the browser:

```
$ glab auth login
! Opening gitlab.com in your browser...
- Waiting for authentication...
✓ Logged in to gitlab.com as username
```

### OAuth setup

Before using OAuth, create an OAuth application in your GitLab instance
under **Settings > Applications** with:
- Redirect URI: `http://localhost:7171/auth/redirect`
- Scopes: `api`, `read_user`, `write_repository`, `openid`, `profile`

The CLI starts a local server on port 7171, opens your browser for authorization, and
automatically exchanges the code for a token using PKCE — no client secret needed.

### Token-based login

If you prefer personal access tokens over OAuth, use `--token` or `--stdin`:

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

Required token scopes: `api`, `read_user`, `write_repository`

### Auth login flags

| Flag | Description |
|------|-------------|
| `--hostname` | GitLab hostname (default: gitlab.com) |
| `--token, -t` | Personal access token (skips OAuth, uses PAT) |
| `--client-id` | OAuth application ID |
| `--git-protocol, -p` | Preferred git protocol: `https` or `ssh` |
| `--stdin` | Read token from standard input (skips OAuth, uses PAT) |

### Per-host configuration

Store OAuth settings per host so you don't need to pass them every time:

```bash
# Store OAuth client ID for a self-hosted instance
glab config set client_id <app-id> --host gitlab.example.com

# Store custom redirect URI (default: http://localhost:7171/auth/redirect)
glab config set redirect_uri http://localhost:8080/callback --host gitlab.example.com

# Store custom OAuth scopes
glab config set oauth_scopes "api read_user write_repository" --host gitlab.example.com
```

These values are also automatically saved during your first interactive `glab auth login`.

## Global Flags

| Flag | Description |
|------|-------------|
| `--repo, -R` | Select a GitLab repository using `HOST/OWNER/REPO` format |

The `--repo` flag lets you target any project without being in its git repository:

```bash
glab issue list -R gitlab.example.com/owner/repo
glab mr list --state opened -R gitlab.example.com/group/project
```

When no `--repo` is specified, glab resolves the host from the git remote. If the remote isn't a GitLab host, it falls back to the default host, then to the first authenticated host.

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
glab mr comment 123 --body "Looks good!"
glab mr comment 123 --body "Consider refactoring this" --file "cmd/mr.go" --line 42
glab mr comment 123 --body "Good removal" --file "cmd/mr.go" --old-line 10
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

# Per-host config
glab config set client_id <app-id> --host gitlab.example.com
glab config get client_id --host gitlab.example.com
```

### Direct API Access

```bash
glab api projects
glab api users --method GET
glab api projects/:id/issues --method POST --body '{"title":"Bug"}'

# Target a specific host
glab api '/projects?membership=true' --hostname gitlab.example.com
```

## Configuration

Configuration is stored in `~/.config/glab/`. Override with `GLAB_CONFIG_DIR`.

### Global keys

| Key | Description | Default |
|-----|-------------|---------|
| `editor` | Preferred text editor | - |
| `pager` | Preferred pager | - |
| `browser` | Preferred web browser | - |
| `protocol` | Git protocol (https/ssh) | https |
| `git_remote` | Default git remote name | origin |

### Per-host keys (use with `--host`)

| Key | Description | Default |
|-----|-------------|---------|
| `client_id` | OAuth application ID | - |
| `redirect_uri` | OAuth redirect URI | `http://localhost:7171/auth/redirect` |
| `oauth_scopes` | OAuth scopes | `openid profile api read_user write_repository` |
| `protocol` | Git protocol for this host | - |
| `api_host` | API hostname override | - |

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
