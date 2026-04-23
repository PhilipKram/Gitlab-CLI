#!/bin/bash
# One-shot setup for `glab mcp serve` in Docker on macOS. Does three things:
#
#   1. Builds the Docker image (unless --skip-build).
#   2. Installs the Mac:443 -> <GITLAB_HOST>:443 passthrough as a LaunchDaemon
#      (needed because Docker containers cannot see the Mac's VPN — see README).
#      Skip with --no-proxy if your GitLab is on the public internet.
#   3. Starts the container via docker compose.
#
# Configuration: copy scripts/.env.example to scripts/.env and fill in
# GITLAB_HOST + GLAB_CLIENT_ID before running.
#
# Safe to re-run; steps are idempotent.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
ENV_FILE="$SCRIPT_DIR/.env"

SKIP_BUILD=false
NO_PROXY=false
IMAGE_TAG="glab-mcp:oauth"
while [[ $# -gt 0 ]]; do
    case "$1" in
        --skip-build) SKIP_BUILD=true; shift ;;
        --no-proxy)   NO_PROXY=true;   shift ;;
        --tag)        IMAGE_TAG="$2";  shift 2 ;;
        *) echo "Unknown flag: $1" >&2; exit 2 ;;
    esac
done

if [[ ! -f "$ENV_FILE" ]]; then
    echo "Missing $ENV_FILE — copy scripts/.env.example and fill in GITLAB_HOST + GLAB_CLIENT_ID" >&2
    exit 1
fi
# shellcheck disable=SC1090
set -a; source "$ENV_FILE"; set +a

if [[ -z "${GITLAB_HOST:-}" || -z "${GLAB_CLIENT_ID:-}" ]]; then
    echo "GITLAB_HOST and GLAB_CLIENT_ID must be set in $ENV_FILE" >&2
    exit 1
fi

if ! $SKIP_BUILD; then
    echo "==> Building $IMAGE_TAG"
    docker build -t "$IMAGE_TAG" "$REPO_ROOT"
fi

if ! $NO_PROXY; then
    echo "==> Installing Mac-side passthrough for $GITLAB_HOST (requires sudo, one time)"
    sudo UPSTREAM_HOST="$GITLAB_HOST" bash "$SCRIPT_DIR/install-passthrough.sh"
fi

echo "==> Starting container via docker compose"
docker compose --env-file "$ENV_FILE" -f "$SCRIPT_DIR/docker-compose.glab-mcp.yml" up -d

echo
echo "==> Status"
docker compose --env-file "$ENV_FILE" -f "$SCRIPT_DIR/docker-compose.glab-mcp.yml" ps
echo
if ! $NO_PROXY; then
    echo "   Proxy log: sudo tail -f /var/log/glab-mcp-passthrough.log"
fi
echo "   Container log: docker logs -f glab-mcp"
echo
echo "Register with Claude Code (one time):"
echo "   claude mcp add --scope user --transport http glab http://localhost:7171/mcp"
echo
echo "Then in Claude Code: /mcp -> glab -> Authenticate"
