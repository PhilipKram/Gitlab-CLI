#!/bin/bash
# One-time install of the Mac:443 -> <UPSTREAM_HOST>:443 passthrough as a
# LaunchDaemon. Needed when Docker containers (Rancher Desktop / Lima VM) must
# reach a private GitLab that is only reachable via the Mac's VPN (GlobalProtect,
# Tailscale with per-host routing, etc.).
#
# Run:   sudo UPSTREAM_HOST=gitlab.example.com bash install-passthrough.sh
#
# Uninstall:
#   sudo launchctl bootout system/com.glab.mcp-passthrough
#   sudo rm /Library/LaunchDaemons/com.glab.mcp-passthrough.plist

set -euo pipefail

if [[ $EUID -ne 0 ]]; then
    echo "Must run as root (sudo)" >&2
    exit 1
fi

if [[ -z "${UPSTREAM_HOST:-}" ]]; then
    echo "UPSTREAM_HOST must be set (e.g. UPSTREAM_HOST=gitlab.example.com)" >&2
    exit 1
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEMPLATE_SRC="$SCRIPT_DIR/com.glab.mcp-passthrough.plist.template"
SCRIPT_PATH="$SCRIPT_DIR/gitlab-passthrough.py"
PLIST_DST="/Library/LaunchDaemons/com.glab.mcp-passthrough.plist"
LABEL="com.glab.mcp-passthrough"

if [[ ! -f "$TEMPLATE_SRC" ]]; then
    echo "Missing $TEMPLATE_SRC" >&2
    exit 1
fi
if [[ ! -f "$SCRIPT_PATH" ]]; then
    echo "Missing $SCRIPT_PATH" >&2
    exit 1
fi

# Unload any previous instance before replacing.
if launchctl print "system/$LABEL" >/dev/null 2>&1; then
    launchctl bootout "system/$LABEL" || true
fi

# Render the plist template with the real script path and upstream host.
sed -e "s|@SCRIPT_PATH@|$SCRIPT_PATH|g" \
    -e "s|@UPSTREAM_HOST@|$UPSTREAM_HOST|g" \
    "$TEMPLATE_SRC" > "$PLIST_DST"
chown root:wheel "$PLIST_DST"
chmod 644 "$PLIST_DST"

launchctl bootstrap system "$PLIST_DST"

# Wait for the listener.
for i in $(seq 1 10); do
    if lsof -iTCP:443 -sTCP:LISTEN -P -n 2>/dev/null | grep -q python3; then
        echo "Passthrough is listening on :443 -> $UPSTREAM_HOST:443"
        lsof -iTCP:443 -sTCP:LISTEN -P -n | head -3
        exit 0
    fi
    sleep 1
done

echo "Proxy did not come up. Check /var/log/glab-mcp-passthrough.log" >&2
tail -n 20 /var/log/glab-mcp-passthrough.log 2>&1 || true
exit 1
