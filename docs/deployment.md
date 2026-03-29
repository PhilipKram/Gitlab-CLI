# glab Remote MCP Server — Deployment Guide

This document covers everything the infrastructure team needs to deploy and operate the glab remote MCP server. The server exposes GitLab tools over HTTP using the [Model Context Protocol](https://modelcontextprotocol.io), allowing AI assistants (Claude Code, Copilot, etc.) to interact with GitLab remotely.

---

## What it is

A single static Go binary (`glab`) that acts as an HTTP proxy between AI clients and the GitLab API. It does not store data, run CI jobs, or perform heavy computation. It translates MCP tool calls into GitLab REST/GraphQL API requests and streams results back via Server-Sent Events (SSE).

## Resource requirements

| Resource | Small team (<20 users) | Medium (20-100 users) | Large (100+ users) |
|----------|----------------------|----------------------|-------------------|
| **CPU** | 0.25 vCPU (250m) | 0.5 vCPU (500m) | 1 vCPU |
| **Memory** | 128 MB (256 MB limit) | 256 MB (512 MB limit) | 512 MB (1 GB limit) |
| **Disk** | ~20 MB (binary only) | ~20 MB | ~20 MB |
| **Replicas** | 1 | 2 | 2+ behind LB |

The server is stateless when using `--stateless` mode or per-user OAuth, so horizontal scaling is straightforward.

## Network requirements

| Direction | Target | Port | Protocol | Purpose |
|-----------|--------|------|----------|---------|
| **Inbound** | MCP server | 8080 (configurable) | HTTP | AI client connections (SSE streams) |
| **Outbound** | GitLab instance | 443 | HTTPS | GitLab API calls |
| **Outbound** | GitLab instance | 443 | HTTPS | OAuth token exchange (if using per-user OAuth) |

No other network access is required. The server does not phone home, collect telemetry, or connect to external services.

## Prerequisites

- Docker (or a Linux/macOS host to run the binary directly)
- Network access to your GitLab instance (gitlab.com or self-hosted)
- A TLS-terminating reverse proxy for production (nginx, Caddy, AWS ALB, etc.)
- **For shared-token mode:** a GitLab personal access token (PAT) with `api` scope
- **For per-user OAuth mode:** a GitLab OAuth application (see [OAuth setup](#per-user-oauth-setup))

---

## Authentication modes

Choose one of the two modes based on your security requirements:

### Option A: Shared token (simple)

A single GitLab personal access token is used for all API calls. A bearer token protects the HTTP endpoint.

- **Pros:** Simple setup, no OAuth app needed
- **Cons:** All API calls use one identity; no per-user audit trail in GitLab
- **Best for:** Small teams, internal/dev environments, service accounts

**Required secrets:**
| Secret | Purpose |
|--------|---------|
| `GITLAB_TOKEN` | GitLab PAT with `api` scope — used for all API calls |
| `--token <value>` | Bearer token clients use to authenticate to the MCP server |

### Option B: Per-user OAuth (recommended for production)

Each user authenticates with their own GitLab account via OAuth 2.1 (PKCE). No shared GitLab token is needed — the server uses each user's individual OAuth token.

- **Pros:** Per-user audit trail, least-privilege access, no shared secrets
- **Cons:** Requires a GitLab OAuth application, slightly more complex setup
- **Best for:** Production, teams with compliance requirements

**Required configuration:**
| Parameter | Purpose |
|-----------|---------|
| `--client-id` | GitLab OAuth application ID |
| `--gitlab-host` | GitLab hostname (e.g., `gitlab.example.com`) |
| `--external-url` | Public URL of the MCP server (for OAuth callbacks) |

---

## Deployment options

### 1. Docker (recommended)

```bash
# Build
docker build -t glab-mcp .

# --- Shared token mode ---
docker run -d --name glab-mcp \
  -p 8080:8080 \
  -v glab-mcp-data:/config \
  -e GITLAB_TOKEN=glpat-xxxxxxxxxxxx \
  --restart unless-stopped \
  glab-mcp --token <bearer-secret>

# --- Per-user OAuth mode ---
docker run -d --name glab-mcp \
  -p 8080:8080 \
  -v glab-mcp-data:/config \
  --restart unless-stopped \
  glab-mcp \
    --client-id <gitlab-oauth-app-id> \
    --gitlab-host gitlab.example.com \
    --external-url https://mcp.example.com
```

The `-v glab-mcp-data:/config` mount persists bearer tokens and OAuth sessions across container restarts.
The image is ~20 MB (scratch base, static binary, TLS certs only).

### 2. Docker Compose

```yaml
version: "3.8"
services:
  glab-mcp:
    build: .
    ports:
      - "8080:8080"
    volumes:
      - glab-mcp-data:/config
    environment:
      GITLAB_TOKEN: glpat-xxxxxxxxxxxx    # shared token mode only
    command:
      - "--client-id=<gitlab-oauth-app-id>"
      - "--gitlab-host=gitlab.example.com"
      - "--external-url=https://mcp.example.com"
    restart: unless-stopped
    deploy:
      resources:
        limits:
          cpus: "0.5"
          memory: 256M
volumes:
  glab-mcp-data:
```

### 3. Kubernetes

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: glab-mcp
  labels:
    app: glab-mcp
spec:
  replicas: 2
  selector:
    matchLabels:
      app: glab-mcp
  template:
    metadata:
      labels:
        app: glab-mcp
    spec:
      containers:
        - name: glab-mcp
          image: glab-mcp:latest
          ports:
            - containerPort: 8080
          args:
            - "--client-id=$(OAUTH_CLIENT_ID)"
            - "--gitlab-host=gitlab.example.com"
            - "--external-url=https://mcp.example.com"
          env:
            - name: OAUTH_CLIENT_ID
              valueFrom:
                secretKeyRef:
                  name: glab-mcp-secrets
                  key: client-id
          resources:
            requests:
              cpu: 250m
              memory: 128Mi
            limits:
              cpu: 500m
              memory: 256Mi
          livenessProbe:
            httpGet:
              path: /mcp
              port: 8080
            initialDelaySeconds: 5
            periodSeconds: 30
          readinessProbe:
            httpGet:
              path: /mcp
              port: 8080
            initialDelaySeconds: 3
            periodSeconds: 10
---
apiVersion: v1
kind: Service
metadata:
  name: glab-mcp
spec:
  selector:
    app: glab-mcp
  ports:
    - port: 80
      targetPort: 8080
  type: ClusterIP
---
apiVersion: v1
kind: Secret
metadata:
  name: glab-mcp-secrets
type: Opaque
stringData:
  client-id: "<gitlab-oauth-app-id>"
  # For shared token mode, also add:
  # gitlab-token: "glpat-xxxxxxxxxxxx"
  # bearer-token: "<bearer-secret>"
```

### 4. Bare binary (systemd)

```ini
# /etc/systemd/system/glab-mcp.service
[Unit]
Description=glab Remote MCP Server
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=glab-mcp
ExecStart=/usr/local/bin/glab mcp serve --transport http --host 0.0.0.0 --port 8080 \
  --client-id <app-id> --gitlab-host gitlab.example.com \
  --external-url https://mcp.example.com
Environment=GLAB_CONFIG_DIR=/etc/glab
Restart=on-failure
RestartSec=5
LimitNOFILE=65536

[Install]
WantedBy=multi-user.target
```

```bash
sudo systemctl enable --now glab-mcp
```

---

## Reverse proxy (TLS termination)

The MCP server serves plain HTTP. Place a reverse proxy in front for TLS.

### nginx

```nginx
upstream glab-mcp {
    server 127.0.0.1:8080;
}

server {
    listen 443 ssl http2;
    server_name mcp.example.com;

    ssl_certificate     /etc/ssl/certs/mcp.example.com.crt;
    ssl_certificate_key /etc/ssl/private/mcp.example.com.key;

    # MCP endpoint (SSE streams)
    location /mcp {
        proxy_pass http://glab-mcp/mcp;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_buffering off;
        proxy_cache off;
        proxy_read_timeout 86400s;
    }

    # OAuth endpoints (per-user OAuth mode)
    location /oauth {
        proxy_pass http://glab-mcp/oauth;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
    location /auth {
        proxy_pass http://glab-mcp/auth;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
    location /.well-known {
        proxy_pass http://glab-mcp/.well-known;
        proxy_set_header Host $host;
    }
}
```

### Caddy (auto TLS)

```
mcp.example.com {
    reverse_proxy /mcp/* 127.0.0.1:8080 {
        flush_interval -1
    }
    reverse_proxy /oauth/* 127.0.0.1:8080
    reverse_proxy /auth/* 127.0.0.1:8080
    reverse_proxy /.well-known/* 127.0.0.1:8080
}
```

### AWS ALB / GCP Load Balancer

- Target group: port 8080, HTTP health check on `GET /mcp`
- Enable sticky sessions if not using `--stateless` mode
- Set idle timeout to 3600s+ (SSE connections are long-lived)

---

## Per-user OAuth setup

1. **Create a GitLab OAuth application:**
   - Go to GitLab > Admin > Applications (instance-wide) or User Settings > Applications (user-scoped)
   - Name: `glab MCP Server` (or any name)
   - Redirect URI: `https://mcp.example.com/auth/redirect`
   - Scopes: `api`, `read_user`
   - Confidential: No (public client, PKCE is used)
   - Save the **Application ID** — this is your `--client-id`

2. **Start the server with OAuth:**
   ```bash
   glab mcp serve --transport http --host 0.0.0.0 \
     --client-id <application-id> \
     --gitlab-host gitlab.example.com \
     --external-url https://mcp.example.com
   ```

3. **User onboarding:** Users connect their AI client by running:
   ```bash
   glab mcp install --transport http --host mcp.example.com --port 443
   ```
   The MCP client handles the OAuth flow automatically.

---

## Health checks and monitoring

| Check | Endpoint | Expected |
|-------|----------|----------|
| **Liveness** | `GET /mcp` | HTTP 200 or 401 (server is up) |
| **Process** | Check PID or container status | Running |
| **GitLab connectivity** | Server logs (stderr) | No connection errors |

The server logs to stderr. In Docker, use `docker logs glab-mcp`. For systemd, use `journalctl -u glab-mcp`.

### Signals

| Signal | Behavior |
|--------|----------|
| `SIGINT` / `SIGTERM` | Graceful shutdown (drains active connections, 5s timeout) |

---

## Security considerations

- The server **does not execute shell commands** — all GitLab interaction goes through the official Go API client
- No data is stored on disk (except optional token persistence in `$GLAB_CONFIG_DIR/mcp_token`)
- In per-user OAuth mode, no shared GitLab credentials are needed
- Restrict network ingress to known AI client IPs where possible
- Use TLS in production (via reverse proxy)
- The `--no-auth` flag disables all authentication — **never use in production**

---

## All server flags

| Flag | Default | Description |
|------|---------|-------------|
| `--transport` | `stdio` | Transport: `stdio` or `http` |
| `--host` | `localhost` | HTTP listen address |
| `--port` | `8080` | HTTP listen port |
| `--token` | (auto-generated) | Bearer token for shared-token auth |
| `--no-auth` | `false` | Disable authentication entirely |
| `--stateless` | `false` | Stateless mode (no session tracking) |
| `--base-path` | `/mcp` | HTTP endpoint path |
| `--client-id` | | GitLab OAuth application ID (enables per-user OAuth) |
| `--gitlab-host` | from config | GitLab hostname for OAuth |
| `--external-url` | | Public base URL for OAuth callbacks |

## Environment variables

| Variable | Purpose |
|----------|---------|
| `GITLAB_TOKEN` | GitLab PAT — used in shared-token mode for API calls |
| `GLAB_TOKEN` | Alternative to `GITLAB_TOKEN` |
| `GITLAB_HOST` | Default GitLab hostname |
| `GLAB_CONFIG_DIR` | Config directory (default: `~/.config/glab`) |
