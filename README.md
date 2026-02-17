# glab - GitLab CLI

Work seamlessly with GitLab from the command line.

`glab` is a CLI tool for GitLab, written in Go. It follows the same interface patterns as the GitHub CLI (`gh`), providing a familiar experience for managing merge requests, issues, pipelines, repositories, and more.

**Documentation:** [philipkram.github.io/Gitlab-CLI](https://philipkram.github.io/Gitlab-CLI/)

## Installation

### Homebrew (macOS & Linux)

```bash
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

```bash
# Login with a personal access token
glab auth login --token glpat-xxxxxxxxxxxxxxxxxxxx

# Login to a self-hosted GitLab instance
glab auth login --hostname gitlab.example.com --token glpat-xxxx

# Check authentication status
glab auth status

# Environment variable authentication
export GITLAB_TOKEN="glpat-xxxxxxxxxxxxxxxxxxxx"
```

Required token scopes: `api`, `read_user`, `read_repository`

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
4. Update the Homebrew formula in `PhilipKram/homebrew-tap`
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
