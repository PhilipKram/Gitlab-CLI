package cmd

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/PhilipKram/gitlab-cli/internal/api"
	"github.com/PhilipKram/gitlab-cli/internal/auth"
	"github.com/PhilipKram/gitlab-cli/internal/cmdutil"
	"github.com/PhilipKram/gitlab-cli/internal/config"
	"github.com/PhilipKram/gitlab-cli/internal/git"
	glabmcp "github.com/PhilipKram/gitlab-cli/internal/mcp"
	"github.com/PhilipKram/gitlab-cli/pkg/iostreams"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"
)

// NewMCPCmd creates the mcp command group.
func NewMCPCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mcp <command>",
		Short: "Model Context Protocol server",
		Long:  "Start and manage the GitLab MCP server for AI assistant integration.",
	}

	cmd.AddCommand(newMCPServeCmd(f))
	cmd.AddCommand(newMCPInstallCmd(f))
	cmd.AddCommand(newMCPUninstallCmd(f))
	cmd.AddCommand(newMCPStatusCmd(f))

	return cmd
}

// glabBinaryPath returns the absolute path to the running glab binary.
// Falls back to "glab" if the executable path cannot be determined.
func glabBinaryPath() string {
	exe, err := os.Executable()
	if err != nil {
		return "glab"
	}
	resolved, err := filepath.EvalSymlinks(exe)
	if err != nil {
		return exe
	}
	return resolved
}

// mcpConfigJSON builds the MCP server configuration as a JSON string
// suitable for passing to `claude mcp add-json`.
func mcpConfigJSON(glabPath string) (string, error) {
	config := map[string]interface{}{
		"command": glabPath,
		"args":    []string{"mcp", "serve"},
	}
	data, err := json.Marshal(config)
	if err != nil {
		return "", fmt.Errorf("marshalling MCP config: %w", err)
	}
	return string(data), nil
}

// mcpRemoteConfigJSON builds a remote MCP server configuration as a JSON string.
func mcpRemoteConfigJSON(host string, port int, basePath, token string) (string, error) {
	url := fmt.Sprintf("http://%s:%d%s", host, port, basePath)
	config := map[string]interface{}{
		"url": url,
	}
	if token != "" {
		config["headers"] = map[string]string{
			"Authorization": "Bearer " + token,
		}
	}
	data, err := json.Marshal(config)
	if err != nil {
		return "", fmt.Errorf("marshalling remote MCP config: %w", err)
	}
	return string(data), nil
}

// claudeDesktopConfigPath returns the platform-specific path to the
// Claude Desktop configuration file.
func claudeDesktopConfigPath() (string, error) {
	switch runtime.GOOS {
	case "darwin":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("determining home directory: %w", err)
		}
		return filepath.Join(home, "Library", "Application Support", "Claude", "claude_desktop_config.json"), nil
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData == "" {
			return "", fmt.Errorf("APPDATA environment variable is not set")
		}
		return filepath.Join(appData, "Claude", "claude_desktop_config.json"), nil
	case "linux":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("determining home directory: %w", err)
		}
		return filepath.Join(home, ".config", "Claude", "claude_desktop_config.json"), nil
	default:
		return "", fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

// findClaude locates the 'claude' CLI binary in PATH.
// Returns the path to the binary, or an error with a helpful message if not found.
func findClaude() (string, error) {
	path, err := exec.LookPath("claude")
	if err != nil {
		return "", fmt.Errorf("'claude' CLI not found in PATH\nInstall Claude Code: https://docs.anthropic.com/en/docs/claude-code")
	}
	return path, nil
}

func newMCPInstallCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		scope     string
		client    string
		transport string
		host      string
		port      int
		basePath  string
		token     string
	)

	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install glab as an MCP server in an AI client",
		Long: `Register glab as a Model Context Protocol server in the specified AI client.

Supported clients:
  claude-code     - Claude Code CLI (default)
  claude-desktop  - Claude Desktop app

Supports two transport modes:
  stdio  - Local subprocess (default)
  http   - Remote HTTP server`,
		Example: `  # Install for Claude Code (default, stdio)
  $ glab mcp install

  # Install for Claude Desktop
  $ glab mcp install --client claude-desktop

  # Install with project scope for Claude Code
  $ glab mcp install --scope project

  # Install a remote HTTP MCP server
  $ glab mcp install --transport http --host myserver.example.com --port 8080 --token my-secret`,
		RunE: func(cmd *cobra.Command, args []string) error {
			out := f.IOStreams.Out

			var configJSON string
			var err error

			switch transport {
			case "stdio":
				glabPath := glabBinaryPath()
				configJSON, err = mcpConfigJSON(glabPath)
				if err != nil {
					return err
				}
			case "http":
				// Use persisted token if none provided explicitly
				if token == "" {
					token = loadMCPToken()
				}
				configJSON, err = mcpRemoteConfigJSON(host, port, basePath, token)
				if err != nil {
					return err
				}
			default:
				return fmt.Errorf("unsupported transport: %s (supported: stdio, http)", transport)
			}

			switch client {
			case "claude-code":
				return installClaudeCode(out, scope, configJSON)
			case "claude-desktop":
				if transport == "http" {
					return installClaudeDesktopRemote(out, host, port, basePath, token)
				}
				glabPath := glabBinaryPath()
				return installClaudeDesktop(out, glabPath)
			default:
				return fmt.Errorf("unsupported client: %s (supported: claude-code, claude-desktop)", client)
			}
		},
	}

	cmd.Flags().StringVar(&scope, "scope", "user", "Installation scope: user, local, or project")
	cmd.Flags().StringVar(&client, "client", "claude-code", "AI client: claude-code or claude-desktop")
	cmd.Flags().StringVar(&transport, "transport", "stdio", "Transport: stdio or http")
	cmd.Flags().StringVar(&host, "host", "localhost", "Remote MCP server host")
	cmd.Flags().IntVar(&port, "port", 8080, "Remote MCP server port")
	cmd.Flags().StringVar(&basePath, "base-path", "/mcp", "Remote MCP server endpoint path")
	cmd.Flags().StringVar(&token, "token", "", "Bearer token for remote MCP server")

	return cmd
}

// installClaudeCode registers glab as an MCP server in Claude Code using the claude CLI.
func installClaudeCode(out io.Writer, scope, configJSON string) error {
	claudePath, err := findClaude()
	if err != nil {
		return err
	}

	//nolint:gosec // arguments are validated
	c := exec.Command(claudePath, "mcp", "add-json", "--scope", scope, "glab", configJSON)
	c.Stdout = out
	c.Stderr = out

	if err := c.Run(); err != nil {
		return fmt.Errorf("failed to register MCP server with Claude Code: %w", err)
	}

	_, _ = fmt.Fprintln(out, "glab MCP server registered with Claude Code.")
	return nil
}

// installClaudeDesktop registers glab as an MCP server in Claude Desktop
// by merging configuration into the desktop config JSON file.
func installClaudeDesktop(out io.Writer, glabPath string) error {
	configPath, err := claudeDesktopConfigPath()
	if err != nil {
		return err
	}
	return installClaudeDesktopToPath(out, glabPath, configPath)
}

// installClaudeDesktopToPath registers glab as an MCP server by merging
// configuration into the given config JSON file path.
func installClaudeDesktopToPath(out io.Writer, glabPath, configPath string) error {
	// Read existing config or start fresh
	var config map[string]interface{}
	data, err := os.ReadFile(configPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("reading Claude Desktop config: %w", err)
		}
		config = make(map[string]interface{})
	} else {
		if err := json.Unmarshal(data, &config); err != nil {
			return fmt.Errorf("parsing Claude Desktop config: %w", err)
		}
	}

	// Ensure mcpServers key exists
	servers, ok := config["mcpServers"].(map[string]interface{})
	if !ok {
		servers = make(map[string]interface{})
	}

	// Add glab entry
	servers["glab"] = map[string]interface{}{
		"command": glabPath,
		"args":    []string{"mcp", "serve"},
	}
	config["mcpServers"] = servers

	// Write back
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	out2, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("marshalling config: %w", err)
	}

	if err := os.WriteFile(configPath, out2, 0o644); err != nil {
		return fmt.Errorf("writing Claude Desktop config: %w", err)
	}

	_, _ = fmt.Fprintf(out, "glab MCP server registered in %s\n", configPath)
	_, _ = fmt.Fprintln(out, "Restart Claude Desktop to activate.")
	return nil
}

// installClaudeDesktopRemote registers a remote glab MCP server in Claude Desktop.
func installClaudeDesktopRemote(out io.Writer, host string, port int, basePath, token string) error {
	configPath, err := claudeDesktopConfigPath()
	if err != nil {
		return err
	}
	return installClaudeDesktopRemoteToPath(out, host, port, basePath, token, configPath)
}

// installClaudeDesktopRemoteToPath registers a remote glab MCP server in the given config file.
func installClaudeDesktopRemoteToPath(out io.Writer, host string, port int, basePath, token, configPath string) error {
	var config map[string]interface{}
	data, err := os.ReadFile(configPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("reading Claude Desktop config: %w", err)
		}
		config = make(map[string]interface{})
	} else {
		if err := json.Unmarshal(data, &config); err != nil {
			return fmt.Errorf("parsing Claude Desktop config: %w", err)
		}
	}

	servers, ok := config["mcpServers"].(map[string]interface{})
	if !ok {
		servers = make(map[string]interface{})
	}

	url := fmt.Sprintf("http://%s:%d%s", host, port, basePath)
	entry := map[string]interface{}{
		"url": url,
	}
	if token != "" {
		entry["headers"] = map[string]string{
			"Authorization": "Bearer " + token,
		}
	}
	servers["glab"] = entry
	config["mcpServers"] = servers

	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	out2, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("marshalling config: %w", err)
	}

	if err := os.WriteFile(configPath, out2, 0o644); err != nil {
		return fmt.Errorf("writing Claude Desktop config: %w", err)
	}

	_, _ = fmt.Fprintf(out, "glab remote MCP server registered in %s\n", configPath)
	_, _ = fmt.Fprintln(out, "Restart Claude Desktop to activate.")
	return nil
}

func newMCPUninstallCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		scope  string
		client string
	)

	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Uninstall glab as an MCP server from an AI client",
		Long: `Remove glab as a Model Context Protocol server from the specified AI client.

Supported clients:
  claude-code     - Claude Code CLI (default)
  claude-desktop  - Claude Desktop app`,
		Example: `  # Uninstall from Claude Code (default)
  $ glab mcp uninstall

  # Uninstall from Claude Desktop
  $ glab mcp uninstall --client claude-desktop

  # Uninstall from project scope for Claude Code
  $ glab mcp uninstall --scope project`,
		RunE: func(cmd *cobra.Command, args []string) error {
			out := f.IOStreams.Out

			switch client {
			case "claude-code":
				return uninstallClaudeCode(out, scope)
			case "claude-desktop":
				return uninstallClaudeDesktop(out)
			default:
				return fmt.Errorf("unsupported client: %s (supported: claude-code, claude-desktop)", client)
			}
		},
	}

	cmd.Flags().StringVar(&scope, "scope", "user", "Installation scope: user, local, or project")
	cmd.Flags().StringVar(&client, "client", "claude-code", "AI client: claude-code or claude-desktop")

	return cmd
}

// uninstallClaudeCode removes glab as an MCP server from Claude Code using the claude CLI.
func uninstallClaudeCode(out io.Writer, scope string) error {
	claudePath, err := findClaude()
	if err != nil {
		return err
	}

	//nolint:gosec // arguments are validated
	c := exec.Command(claudePath, "mcp", "remove", "glab", "--scope", scope)
	c.Stdout = out
	c.Stderr = out

	if err := c.Run(); err != nil {
		// Exit cleanly if not registered — the remove command may fail if glab isn't registered
		_, _ = fmt.Fprintln(out, "glab MCP server is not registered with Claude Code (nothing to remove).")
		return nil
	}

	_, _ = fmt.Fprintln(out, "glab MCP server removed from Claude Code.")
	return nil
}

// uninstallClaudeDesktop removes glab as an MCP server from Claude Desktop
// by removing the 'glab' key from the desktop config JSON file.
func uninstallClaudeDesktop(out io.Writer) error {
	configPath, err := claudeDesktopConfigPath()
	if err != nil {
		return err
	}
	return uninstallClaudeDesktopFromPath(out, configPath)
}

// uninstallClaudeDesktopFromPath removes glab from the given config file path.
func uninstallClaudeDesktopFromPath(out io.Writer, configPath string) error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			_, _ = fmt.Fprintln(out, "glab MCP server is not registered with Claude Desktop (nothing to remove).")
			return nil
		}
		return fmt.Errorf("reading Claude Desktop config: %w", err)
	}

	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("parsing Claude Desktop config: %w", err)
	}

	servers, ok := config["mcpServers"].(map[string]interface{})
	if !ok {
		_, _ = fmt.Fprintln(out, "glab MCP server is not registered with Claude Desktop (nothing to remove).")
		return nil
	}

	if _, exists := servers["glab"]; !exists {
		_, _ = fmt.Fprintln(out, "glab MCP server is not registered with Claude Desktop (nothing to remove).")
		return nil
	}

	delete(servers, "glab")
	config["mcpServers"] = servers

	out2, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("marshalling config: %w", err)
	}

	if err := os.WriteFile(configPath, out2, 0o644); err != nil {
		return fmt.Errorf("writing Claude Desktop config: %w", err)
	}

	_, _ = fmt.Fprintf(out, "glab MCP server removed from %s\n", configPath)
	_, _ = fmt.Fprintln(out, "Restart Claude Desktop to apply changes.")
	return nil
}

func newMCPStatusCmd(f *cmdutil.Factory) *cobra.Command {
	var client string

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Check if glab is registered as an MCP server",
		Long: `Check whether glab is registered as a Model Context Protocol server in the specified AI client.

Supported clients:
  claude-code     - Claude Code CLI (default)`,
		Example: `  # Check status in Claude Code (default)
  $ glab mcp status

  # Check status in a specific client
  $ glab mcp status --client claude-code`,
		RunE: func(cmd *cobra.Command, args []string) error {
			out := f.IOStreams.Out

			switch client {
			case "claude-code":
				return statusClaudeCode(out)
			default:
				return fmt.Errorf("unsupported client: %s (supported: claude-code)", client)
			}
		},
	}

	cmd.Flags().StringVar(&client, "client", "claude-code", "AI client: claude-code")

	return cmd
}

// statusClaudeCode checks if glab is registered as an MCP server in Claude Code.
func statusClaudeCode(out io.Writer) error {
	claudePath, err := findClaude()
	if err != nil {
		return err
	}

	//nolint:gosec // arguments are validated
	c := exec.Command(claudePath, "mcp", "get", "glab")
	output, err := c.Output()
	if err != nil {
		_, _ = fmt.Fprintln(out, "glab is not registered in Claude Code.")
		_, _ = fmt.Fprintln(out, "Run `glab mcp install` to register it.")
		return fmt.Errorf("glab MCP server is not registered with Claude Code")
	}

	_, _ = fmt.Fprintln(out, "glab is registered in Claude Code.")
	if len(output) > 0 {
		_, _ = fmt.Fprintf(out, "\nConfiguration:\n%s", string(output))
	}
	return nil
}

func newMCPServeCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		transport   string
		port        int
		host        string
		token       string
		noAuth      bool
		stateless   bool
		basePath    string
		clientID    string
		gitlabHost  string
		externalURL string
	)

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the MCP server",
		Long: `Start a Model Context Protocol server, exposing GitLab tools to AI assistants
such as Claude, GitHub Copilot, and any MCP-compatible client.

Supports two transports:
  stdio  - Standard I/O (default), for use as a subprocess
  http   - HTTP with Server-Sent Events, for remote/networked access

HTTP authentication modes:
  --token / auto-generated  Single shared bearer token (default)
  --client-id               Per-user OAuth via MCP protocol authorization

When --client-id is set, the server implements the MCP authorization spec
(RFC 8414 metadata, dynamic client registration, PKCE). MCP clients like
Claude Code handle the OAuth flow automatically — users just configure the
server URL with no token needed.`,
		Example: `  # Start over stdio (default)
  $ glab mcp serve

  # Start as HTTP server with a shared token
  $ glab mcp serve --transport http

  # Start with per-user OAuth authentication
  $ glab mcp serve --transport http --client-id my-oauth-app-id --gitlab-host gitlab.example.com

  # Start in stateless mode without authentication
  $ glab mcp serve --transport http --stateless --no-auth

  # Start with an explicit project
  $ glab -R gitlab.example.com/owner/repo mcp serve --transport http`,
		RunE: func(cmd *cobra.Command, args []string) error {
			switch transport {
			case "stdio":
				server := glabmcp.NewMCPServer(f)
				_, _ = fmt.Fprintln(f.IOStreams.ErrOut, "glab MCP server running on stdio")
				return server.Run(context.Background(), &mcp.StdioTransport{})
			case "http":
				// Fall back to the client_id stored in host config when --client-id is not set.
				// Users can run: `glab config set client_id <app-id> --host gitlab.example.com`
				// then just `glab mcp serve --transport http` to get per-user OAuth.
				if clientID == "" {
					h := gitlabHost
					if h == "" {
						h = config.DefaultHost()
					}
					if h != "" {
						clientID = config.ClientIDForHost(h)
					}
				}
				if clientID != "" {
					return serveHTTPOAuth(f, host, port, clientID, gitlabHost, stateless, basePath, externalURL)
				}
				server := glabmcp.NewMCPServer(f)
				return serveHTTP(f, server, host, port, token, noAuth, stateless, basePath)
			default:
				return fmt.Errorf("unsupported transport: %s (supported: stdio, http)", transport)
			}
		},
	}

	cmd.Flags().StringVar(&transport, "transport", "stdio", "Transport: stdio or http")
	cmd.Flags().IntVar(&port, "port", 8080, "HTTP listen port")
	cmd.Flags().StringVar(&host, "host", "localhost", "HTTP listen address")
	cmd.Flags().StringVar(&token, "token", "", "Bearer token for HTTP auth (auto-generated if empty)")
	cmd.Flags().BoolVar(&noAuth, "no-auth", false, "Disable authentication for HTTP transport")
	cmd.Flags().BoolVar(&stateless, "stateless", false, "Stateless HTTP mode (no session tracking)")
	cmd.Flags().StringVar(&basePath, "base-path", "/mcp", "HTTP endpoint path")
	cmd.Flags().StringVar(&clientID, "client-id", "", "GitLab OAuth application ID (enables per-user OAuth)")
	cmd.Flags().StringVar(&gitlabHost, "gitlab-host", "", "GitLab hostname for OAuth (default: from config)")
	cmd.Flags().StringVar(&externalURL, "external-url", "", "Public base URL for OAuth callbacks (e.g. https://mcp.example.com)")

	return cmd
}

// serveHTTP starts the MCP server over HTTP with optional bearer token authentication.
func serveHTTP(f *cmdutil.Factory, server *mcp.Server, host string, port int, token string, noAuth, stateless bool, basePath string) error {
	errOut := f.IOStreams.ErrOut

	// Handle authentication token — reuse a persisted token when possible
	// so that MCP clients (e.g. Claude Code) stay authenticated across restarts.
	if !noAuth && token == "" {
		if saved := loadMCPToken(); saved != "" {
			token = saved
		} else {
			var err error
			token, err = generateToken()
			if err != nil {
				return fmt.Errorf("generating auth token: %w", err)
			}
			if err := saveMCPToken(token); err != nil {
				_, _ = fmt.Fprintf(errOut, "Warning: could not persist token: %v\n", err)
			}
		}
	}

	if noAuth {
		_, _ = fmt.Fprintln(errOut, "WARNING: running without authentication — anyone with network access can use this server")
	}

	handler := mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		return server
	}, &mcp.StreamableHTTPOptions{
		Stateless: stateless,
	})

	mux := http.NewServeMux()
	var h http.Handler = handler
	if !noAuth {
		h = bearerAuthMiddleware(handler, token)
	}
	mux.Handle(basePath, h)

	addr := fmt.Sprintf("%s:%d", host, port)
	httpServer := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	// Print server info
	url := fmt.Sprintf("http://%s%s", addr, basePath)
	_, _ = fmt.Fprintf(errOut, "glab MCP server running on %s\n", url)
	if !noAuth {
		_, _ = fmt.Fprintf(errOut, "Auth token: %s\n", token)
	}

	// Graceful shutdown on SIGINT/SIGTERM
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		errCh <- httpServer.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if err != nil && err != http.ErrServerClosed {
			return fmt.Errorf("HTTP server error: %w", err)
		}
	case <-ctx.Done():
		_, _ = fmt.Fprintln(errOut, "\nShutting down...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown error: %w", err)
		}
	}

	return nil
}

// bearerAuthMiddleware wraps an http.Handler to require a valid Bearer token.
func bearerAuthMiddleware(next http.Handler, token string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		provided := strings.TrimPrefix(auth, "Bearer ")
		if subtle.ConstantTimeCompare([]byte(provided), []byte(token)) != 1 {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// generateToken returns a cryptographically random 64-character hex string.
func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// mcpTokenPath returns the path to the persisted MCP bearer token file.
func mcpTokenPath() string {
	return filepath.Join(config.ConfigDir(), "mcp_token")
}

// loadMCPToken reads a previously saved MCP bearer token from disk.
// Returns empty string if no token file exists.
func loadMCPToken() string {
	data, err := os.ReadFile(mcpTokenPath())
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// saveMCPToken persists the MCP bearer token to disk so it survives restarts.
func saveMCPToken(token string) error {
	dir := config.ConfigDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}
	return os.WriteFile(mcpTokenPath(), []byte(token+"\n"), 0o600)
}

// --- Per-user OAuth session management (MCP spec-compliant) ---
//
// When --client-id is set, the MCP server acts as an OAuth 2.1 authorization
// server (RFC 8414) that proxies to GitLab. MCP clients like Claude Code
// discover the OAuth endpoints via /.well-known/oauth-authorization-server
// and handle the entire flow automatically. Users just configure:
//
//   {"url": "http://server:8090/mcp"}
//
// Flow:
//  1. Client POSTs to /mcp → 401
//  2. Client fetches /.well-known/oauth-authorization-server
//  3. Client registers via /oauth/register (dynamic client registration)
//  4. Client opens browser to /oauth/authorize
//  5. Server redirects to GitLab with its own client_id + PKCE
//  6. User authorizes on GitLab
//  7. GitLab redirects to /oauth/callback (server-side)
//  8. Server exchanges code with GitLab, generates its own auth code
//  9. Server redirects to client's redirect_uri with the auth code
// 10. Client exchanges code at /oauth/token
// 11. Server returns a session access token
// 12. Client uses session token for /mcp requests

// defaultTokenExpiry is the lifetime (in seconds) advertised to MCP clients
// via the expires_in field. A long expiry minimises re-authentication prompts
// across Claude Code sessions. The server still refreshes the underlying
// GitLab token proactively.
const defaultTokenExpiry = 30 * 24 * 3600 // 30 days

// Background-refresh cadence: every `interval`, rotate any session whose GitLab
// token expires within `window`. Shorter than typical access-token TTL so
// refresh tokens rotate even with no MCP traffic. Overridable via env vars for
// operators with non-default GitLab token TTLs.
const (
	defaultBackgroundRefreshInterval = 5 * time.Minute
	defaultBackgroundRefreshWindow   = 15 * time.Minute
)

// backgroundRefreshTuning reads the interval and window from env vars, falling
// back to defaults. GLAB_MCP_REFRESH_INTERVAL_SECONDS and ..._WINDOW_SECONDS.
func backgroundRefreshTuning() (interval, window time.Duration) {
	interval = defaultBackgroundRefreshInterval
	window = defaultBackgroundRefreshWindow
	if v := os.Getenv("GLAB_MCP_REFRESH_INTERVAL_SECONDS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			interval = time.Duration(n) * time.Second
		}
	}
	if v := os.Getenv("GLAB_MCP_REFRESH_WINDOW_SECONDS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			window = time.Duration(n) * time.Second
		}
	}
	return
}

// errRefreshAuthFailed marks a refresh failure that's terminal: GitLab rejected
// the refresh token (4xx). Callers should drop the session and force re-auth.
// Anything else (network error, 5xx) is transient and the session should be kept.
var errRefreshAuthFailed = errors.New("gitlab refresh token rejected")

// mcpSession represents an authenticated user's session.
type mcpSession struct {
	BearerToken       string `json:"bearer_token"`
	GitLabHost        string `json:"gitlab_host"`
	AccessToken       string `json:"access_token"`
	RefreshToken      string `json:"refresh_token"`
	MCPRefreshToken   string `json:"mcp_refresh_token,omitempty"` // MCP session refresh token
	TokenExpiresAt    int64  `json:"token_expires_at"`
	ClientID          string `json:"client_id"`
	GitLabClientID    string `json:"gitlab_client_id,omitempty"` // GitLab OAuth app ID (for token refresh)
	GitLabRedirectURI string `json:"gitlab_redirect_uri,omitempty"`
	Username          string `json:"username,omitempty"`

	// refreshMu serialises token refreshes for this session so a request-path
	// refresh and a background-tick refresh can't both consume the same
	// single-use GitLab refresh token.
	refreshMu sync.Mutex `json:"-"`
}

// oauthRegisteredClient represents a dynamically registered OAuth client.
type oauthRegisteredClient struct {
	ClientID     string   `json:"client_id"`
	ClientName   string   `json:"client_name,omitempty"`
	RedirectURIs []string `json:"redirect_uris"`
}

// oauthAuthRequest tracks an in-progress authorization across the proxy.
type oauthAuthRequest struct {
	// From the MCP client's authorize request
	RegisteredClientID string
	ClientRedirectURI  string
	ClientState        string
	CodeChallenge      string
	CodeChallengeMethod string

	// For GitLab OAuth
	GitLabState    string
	GitLabVerifier string // PKCE verifier for GitLab

	CreatedAt time.Time
}

// oauthAuthCode is a generated auth code ready for the client to exchange.
type oauthAuthCode struct {
	Code            string
	ClientID        string
	RedirectURI     string
	CodeChallenge   string
	CodeChallengeMethod string
	GitLabAccessToken  string
	GitLabRefreshToken string
	GitLabExpiresAt    int64
	CreatedAt       time.Time
}

// mcpSessionStore manages OAuth clients, auth requests, codes, and sessions.
// Sessions and registered clients are persisted to disk so they survive restarts.
type mcpSessionStore struct {
	mu        sync.RWMutex
	sessions  map[string]*mcpSession           // bearer token → session
	clients   map[string]*oauthRegisteredClient // client_id → client
	pending   map[string]*oauthAuthRequest      // gitlab state → auth request (ephemeral)
	codes     map[string]*oauthAuthCode         // auth code → code info (ephemeral)

	// tokenEndpoint builds the GitLab /oauth/token URL for a given host.
	// Overridable for tests that point at an httptest server.
	tokenEndpoint func(host string) string
	// httpClient is used for GitLab refresh calls. Overridable for tests.
	httpClient *http.Client
}

// mcpSessionsFile is the on-disk format for persisted session store data.
type mcpSessionsFile struct {
	Sessions []*mcpSession            `json:"sessions"`
	Clients  []*oauthRegisteredClient `json:"clients,omitempty"`
}

func mcpSessionsPath() string {
	return filepath.Join(config.ConfigDir(), "mcp_sessions.json")
}

func newSessionStore() *mcpSessionStore {
	s := &mcpSessionStore{
		sessions:      make(map[string]*mcpSession),
		clients:       make(map[string]*oauthRegisteredClient),
		pending:       make(map[string]*oauthAuthRequest),
		codes:         make(map[string]*oauthAuthCode),
		tokenEndpoint: defaultTokenEndpoint,
		httpClient:    &http.Client{Timeout: 30 * time.Second},
	}
	s.loadFromDisk()
	return s
}

func defaultTokenEndpoint(host string) string {
	return fmt.Sprintf("https://%s/oauth/token", host)
}

// loadFromDisk restores persisted sessions and clients.
func (s *mcpSessionStore) loadFromDisk() {
	data, err := os.ReadFile(mcpSessionsPath())
	if err != nil {
		return
	}
	var file mcpSessionsFile
	if err := json.Unmarshal(data, &file); err != nil {
		return
	}
	for _, sess := range file.Sessions {
		if sess.BearerToken != "" {
			s.sessions[sess.BearerToken] = sess
		}
	}
	for _, c := range file.Clients {
		if c.ClientID != "" {
			s.clients[c.ClientID] = c
		}
	}
}

// saveToDisk persists sessions and clients. Must be called with mu held.
// Writes are atomic (temp file + rename) so a crash mid-write never leaves a
// truncated sessions file — which would otherwise lose newly-rotated refresh
// tokens and force users to re-authenticate.
func (s *mcpSessionStore) saveToDisk() {
	file := mcpSessionsFile{}
	for _, sess := range s.sessions {
		file.Sessions = append(file.Sessions, sess)
	}
	for _, c := range s.clients {
		file.Clients = append(file.Clients, c)
	}
	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return
	}
	dir := config.ConfigDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return
	}
	tmp, err := os.CreateTemp(dir, "mcp_sessions-*.tmp")
	if err != nil {
		return
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return
	}
	_ = os.Chmod(tmpPath, 0o600)
	if err := os.Rename(tmpPath, mcpSessionsPath()); err != nil {
		_ = os.Remove(tmpPath)
	}
}

func (s *mcpSessionStore) getSession(bearerToken string) *mcpSession {
	s.mu.RLock()
	sess := s.sessions[bearerToken]
	s.mu.RUnlock()
	if sess == nil {
		return nil
	}
	// Auto-refresh if the GitLab token expires within 5 minutes
	if sess.TokenExpiresAt > 0 && time.Now().Unix() > sess.TokenExpiresAt-300 {
		if err := s.refreshSession(sess); err != nil {
			if errors.Is(err, errRefreshAuthFailed) {
				// GitLab rejected the refresh token. Drop the session.
				s.deleteSession(sess.BearerToken)
				return nil
			}
			// Transient (network/5xx). If the access token is already expired
			// the session is unusable; otherwise keep serving the current token.
			if time.Now().Unix() >= sess.TokenExpiresAt {
				return nil
			}
		}
	}
	return sess
}

// startBackgroundRefresh proactively refreshes GitLab access tokens before they
// expire, so sessions stay alive even when MCP clients are idle for long periods.
// Each tick it finds sessions whose tokens expire within `window` and refreshes them.
// It runs until ctx is cancelled.
func (s *mcpSessionStore) startBackgroundRefresh(ctx context.Context, interval, window time.Duration, errOut io.Writer) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.refreshExpiringSessions(window, errOut)
		}
	}
}

// refreshExpiringSessions refreshes every session whose GitLab token expires
// within the window. Called by the background loop.
func (s *mcpSessionStore) refreshExpiringSessions(window time.Duration, errOut io.Writer) {
	cutoff := time.Now().Unix() + int64(window.Seconds())

	s.mu.RLock()
	var toRefresh []*mcpSession
	for _, sess := range s.sessions {
		if sess.TokenExpiresAt > 0 && sess.TokenExpiresAt <= cutoff && sess.RefreshToken != "" && sess.GitLabClientID != "" {
			toRefresh = append(toRefresh, sess)
		}
	}
	s.mu.RUnlock()

	for _, sess := range toRefresh {
		err := s.refreshSession(sess)
		if err == nil {
			continue
		}
		if errors.Is(err, errRefreshAuthFailed) {
			// GitLab rejected the refresh token. Drop the session so we stop
			// retrying a dead token every tick.
			s.deleteSession(sess.BearerToken)
			if errOut != nil {
				_, _ = fmt.Fprintf(errOut, "dropped session for client %s: %v\n", sess.ClientID, err)
			}
			continue
		}
		// Transient (VPN/passthrough down, GitLab 5xx). Keep the session; we'll
		// try again on the next tick.
		if errOut != nil {
			_, _ = fmt.Fprintf(errOut, "background refresh for client %s transient failure: %v\n", sess.ClientID, err)
		}
	}
}

// refreshSession uses the stored refresh token to obtain a new GitLab access
// token. Errors are classified:
//   - nil: refresh succeeded (or was unnecessary because another goroutine just
//     rotated the tokens)
//   - wraps errRefreshAuthFailed: GitLab returned 4xx. The refresh token is dead;
//     callers should delete the session.
//   - any other error: transient (network, 5xx). The session is still usable;
//     don't delete it — retry later.
func (s *mcpSessionStore) refreshSession(sess *mcpSession) error {
	// Serialise refreshes per session so a request-path auto-refresh and a
	// background-tick refresh can't race to consume the same single-use
	// GitLab refresh token.
	sess.refreshMu.Lock()
	defer sess.refreshMu.Unlock()

	// Another goroutine may have just refreshed while we were waiting for the
	// lock. If the session is no longer near expiry, skip — burning a refresh
	// token for nothing is how we lose sessions on flaky networks.
	if sess.TokenExpiresAt > 0 && time.Now().Unix() < sess.TokenExpiresAt-300 {
		return nil
	}

	if sess.RefreshToken == "" || sess.GitLabClientID == "" {
		return fmt.Errorf("no refresh token or client ID available")
	}

	endpoint := s.tokenEndpoint
	if endpoint == nil {
		endpoint = defaultTokenEndpoint
	}
	tokenURL := endpoint(sess.GitLabHost)
	data := url.Values{
		"client_id":     {sess.GitLabClientID},
		"refresh_token": {sess.RefreshToken},
		"grant_type":    {"refresh_token"},
	}
	if sess.GitLabRedirectURI != "" {
		data.Set("redirect_uri", sess.GitLabRedirectURI)
	}

	client := s.httpClient
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	resp, err := client.PostForm(tokenURL, data)
	if err != nil {
		// Transient: didn't reach GitLab (connection refused, TLS timeout) or
		// the response was truncated (EOF). Leave the refresh token alone.
		return fmt.Errorf("requesting token refresh: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading token refresh response: %w", err)
	}

	if resp.StatusCode >= 400 && resp.StatusCode < 500 {
		// GitLab rejected us. Refresh token is dead — caller should drop the session.
		return fmt.Errorf("%w (HTTP %d): %s", errRefreshAuthFailed, resp.StatusCode, string(body))
	}
	if resp.StatusCode != http.StatusOK {
		// 5xx: GitLab is sick, try again later.
		return fmt.Errorf("token refresh failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var tokenResp auth.OAuthTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return fmt.Errorf("parsing token refresh response: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	sess.AccessToken = tokenResp.AccessToken
	if tokenResp.RefreshToken != "" {
		sess.RefreshToken = tokenResp.RefreshToken
	}
	createdAt := tokenResp.CreatedAt
	if createdAt == 0 {
		createdAt = time.Now().Unix()
	}
	if tokenResp.ExpiresIn > 0 {
		sess.TokenExpiresAt = createdAt + int64(tokenResp.ExpiresIn)
	}

	s.saveToDisk()
	return nil
}

func (s *mcpSessionStore) addSession(sess *mcpSession) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[sess.BearerToken] = sess
	s.saveToDisk()
}

// deleteSession removes a session and persists the change. Called when GitLab
// has definitively rejected the session's refresh token; further attempts to
// use it would just burn cycles and pollute logs.
func (s *mcpSessionStore) deleteSession(bearerToken string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.sessions[bearerToken]; !ok {
		return
	}
	delete(s.sessions, bearerToken)
	s.saveToDisk()
}

func (s *mcpSessionStore) addClient(c *oauthRegisteredClient) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.clients[c.ClientID] = c
	s.saveToDisk()
}

func (s *mcpSessionStore) getClient(clientID string) *oauthRegisteredClient {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.clients[clientID]
}

func (s *mcpSessionStore) addPending(gitlabState string, req *oauthAuthRequest) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pending[gitlabState] = req
}

func (s *mcpSessionStore) takePending(gitlabState string) *oauthAuthRequest {
	s.mu.Lock()
	defer s.mu.Unlock()
	req := s.pending[gitlabState]
	delete(s.pending, gitlabState)
	return req
}

func (s *mcpSessionStore) addCode(ac *oauthAuthCode) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.codes[ac.Code] = ac
}

func (s *mcpSessionStore) takeCode(code string) *oauthAuthCode {
	s.mu.Lock()
	defer s.mu.Unlock()
	ac := s.codes[code]
	delete(s.codes, code)
	return ac
}

// sessionBearerAuth wraps an http.Handler to require a valid session bearer token.
// Returns 401 for unauthenticated requests so MCP clients trigger the OAuth flow.
func (s *mcpSessionStore) sessionBearerAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hdr := r.Header.Get("Authorization")
		if !strings.HasPrefix(hdr, "Bearer ") {
			w.Header().Set("WWW-Authenticate", "Bearer")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		token := strings.TrimPrefix(hdr, "Bearer ")
		if s.getSession(token) == nil {
			w.Header().Set("WWW-Authenticate", "Bearer error=\"invalid_token\"")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// newUserFactory creates a Factory scoped to a specific user session.
func newUserFactory(sess *mcpSession, parentFactory *cmdutil.Factory) *cmdutil.Factory {
	return &cmdutil.Factory{
		IOStreams: &iostreams.IOStreams{
			In:     strings.NewReader(""),
			Out:    io.Discard,
			ErrOut: io.Discard,
		},
		Config: func() (*config.Config, error) {
			return config.Load()
		},
		Client: func() (*api.Client, error) {
			return api.NewOAuthClient(sess.GitLabHost, sess.AccessToken)
		},
		Remote: func() (*git.Remote, error) {
			if parentFactory.Remote != nil {
				return parentFactory.Remote()
			}
			return nil, fmt.Errorf("no git remote available on remote MCP server")
		},
	}
}

// serveHTTPOAuth starts the MCP server with MCP spec-compliant OAuth proxy.
func serveHTTPOAuth(f *cmdutil.Factory, host string, port int, clientID, gitlabHost string, stateless bool, basePath, externalURL string) error {
	errOut := f.IOStreams.ErrOut

	if gitlabHost == "" {
		gitlabHost = config.DefaultHost()
	}

	addr := fmt.Sprintf("%s:%d", host, port)
	baseURL := fmt.Sprintf("http://%s", addr)
	if externalURL != "" {
		baseURL = strings.TrimRight(externalURL, "/")
	}
	gitlabCallbackURI := baseURL + "/auth/redirect"

	store := newSessionStore()

	// MCP handler with per-user server creation
	mcpHandler := mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		hdr := r.Header.Get("Authorization")
		token := strings.TrimPrefix(hdr, "Bearer ")
		sess := store.getSession(token)
		if sess == nil {
			return nil
		}
		userFactory := newUserFactory(sess, f)
		return glabmcp.NewMCPServer(userFactory)
	}, &mcp.StreamableHTTPOptions{
		Stateless: stateless,
	})

	mux := http.NewServeMux()

	// MCP endpoint with session auth
	mux.Handle(basePath, store.sessionBearerAuth(mcpHandler))

	// OAuth Authorization Server Metadata (RFC 8414)
	mux.HandleFunc("/.well-known/oauth-authorization-server", oauthMetadataHandler(baseURL))

	// Dynamic Client Registration (RFC 7591)
	mux.HandleFunc("/oauth/register", oauthRegisterHandler(store))

	// Authorization endpoint — proxies to GitLab
	mux.HandleFunc("/oauth/authorize", oauthAuthorizeHandler(store, gitlabHost, clientID, gitlabCallbackURI))

	// GitLab callback — exchanges code, redirects to client
	mux.HandleFunc("/auth/redirect", oauthGitLabCallbackHandler(store, gitlabHost, clientID, gitlabCallbackURI, errOut))

	// Token endpoint — exchanges our auth code for a session token
	mux.HandleFunc("/oauth/token", oauthTokenHandler(store, gitlabHost, clientID, gitlabCallbackURI, errOut))

	httpServer := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	mcpURL := baseURL + basePath
	_, _ = fmt.Fprintf(errOut, "glab MCP server running on %s\n", mcpURL)
	_, _ = fmt.Fprintf(errOut, "OAuth mode: per-user (GitLab: %s)\n", gitlabHost)
	_, _ = fmt.Fprintf(errOut, "Users configure: {\"url\": \"%s\"}\n", mcpURL)

	// Graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Background refresh: keeps GitLab tokens alive when MCP clients are idle.
	interval, window := backgroundRefreshTuning()
	_, _ = fmt.Fprintf(errOut, "Background refresh: every %s, window %s\n", interval, window)
	go store.startBackgroundRefresh(ctx, interval, window, errOut)

	errCh := make(chan error, 1)
	go func() {
		errCh <- httpServer.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if err != nil && err != http.ErrServerClosed {
			return fmt.Errorf("HTTP server error: %w", err)
		}
	case <-ctx.Done():
		_, _ = fmt.Fprintln(errOut, "\nShutting down...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown error: %w", err)
		}
	}

	return nil
}

// oauthMetadataHandler serves RFC 8414 OAuth Authorization Server Metadata.
func oauthMetadataHandler(baseURL string) http.HandlerFunc {
	metadata := map[string]interface{}{
		"issuer":                                baseURL,
		"authorization_endpoint":                baseURL + "/oauth/authorize",
		"token_endpoint":                        baseURL + "/oauth/token",
		"registration_endpoint":                 baseURL + "/oauth/register",
		"response_types_supported":              []string{"code"},
		"grant_types_supported":                 []string{"authorization_code", "refresh_token"},
		"code_challenge_methods_supported":       []string{"S256"},
		"token_endpoint_auth_methods_supported": []string{"none"},
	}
	body, _ := json.Marshal(metadata)

	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	}
}

// oauthRegisterHandler handles dynamic client registration (RFC 7591).
func oauthRegisterHandler(store *mcpSessionStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			ClientName   string   `json:"client_name"`
			RedirectURIs []string `json:"redirect_uris"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		if len(req.RedirectURIs) == 0 {
			http.Error(w, "redirect_uris is required", http.StatusBadRequest)
			return
		}

		regClientID, err := generateToken()
		if err != nil {
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}

		client := &oauthRegisteredClient{
			ClientID:     regClientID,
			ClientName:   req.ClientName,
			RedirectURIs: req.RedirectURIs,
		}
		store.addClient(client)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"client_id":                    regClientID,
			"client_name":                  req.ClientName,
			"redirect_uris":               req.RedirectURIs,
			"grant_types":                 []string{"authorization_code"},
			"response_types":              []string{"code"},
			"token_endpoint_auth_method":  "none",
		})
	}
}

// oauthAuthorizeHandler is the authorization endpoint. It proxies to GitLab.
func oauthAuthorizeHandler(store *mcpSessionStore, gitlabHost, gitlabClientID, gitlabCallbackURI string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		clientID := q.Get("client_id")
		redirectURI := q.Get("redirect_uri")
		clientState := q.Get("state")
		codeChallenge := q.Get("code_challenge")
		codeChallengeMethod := q.Get("code_challenge_method")

		// Validate registered client
		client := store.getClient(clientID)
		if client == nil {
			http.Error(w, "Unknown client_id", http.StatusBadRequest)
			return
		}

		// Validate redirect_uri
		validRedirect := false
		for _, uri := range client.RedirectURIs {
			if uri == redirectURI {
				validRedirect = true
				break
			}
		}
		if !validRedirect {
			http.Error(w, "Invalid redirect_uri", http.StatusBadRequest)
			return
		}

		// Generate PKCE for GitLab
		gitlabVerifier, err := auth.GenerateCodeVerifier()
		if err != nil {
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}
		gitlabChallenge := auth.GenerateCodeChallenge(gitlabVerifier)

		gitlabState, err := auth.GenerateState()
		if err != nil {
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}

		// Store the auth request linking client params to GitLab params
		store.addPending(gitlabState, &oauthAuthRequest{
			RegisteredClientID:  clientID,
			ClientRedirectURI:   redirectURI,
			ClientState:         clientState,
			CodeChallenge:       codeChallenge,
			CodeChallengeMethod: codeChallengeMethod,
			GitLabState:         gitlabState,
			GitLabVerifier:      gitlabVerifier,
			CreatedAt:           time.Now(),
		})

		// Redirect to GitLab
		authURL := auth.BuildAuthURL(gitlabHost, gitlabClientID, gitlabCallbackURI, gitlabState, gitlabChallenge, auth.DefaultScopes())
		http.Redirect(w, r, authURL, http.StatusFound)
	}
}

// oauthGitLabCallbackHandler handles GitLab's OAuth callback,
// exchanges the code with GitLab, and redirects back to the MCP client.
func oauthGitLabCallbackHandler(store *mcpSessionStore, gitlabHost, gitlabClientID, gitlabCallbackURI string, errOut io.Writer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		gitlabState := r.URL.Query().Get("state")
		if gitlabState == "" {
			http.Error(w, "Missing state", http.StatusBadRequest)
			return
		}

		pending := store.takePending(gitlabState)
		if pending == nil {
			http.Error(w, "Invalid or expired state", http.StatusBadRequest)
			return
		}

		// Check for OAuth errors from GitLab
		if errMsg := r.URL.Query().Get("error"); errMsg != "" {
			desc := r.URL.Query().Get("error_description")
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = fmt.Fprintf(w, `<html><body><h1>Authorization Failed</h1><p>%s: %s</p></body></html>`,
				html.EscapeString(errMsg), html.EscapeString(desc))
			return
		}

		gitlabCode := r.URL.Query().Get("code")
		if gitlabCode == "" {
			http.Error(w, "No authorization code received", http.StatusBadRequest)
			return
		}

		// Exchange code with GitLab
		tokenResp, err := auth.ExchangeCode(gitlabHost, gitlabClientID, gitlabCode, gitlabCallbackURI, pending.GitLabVerifier)
		if err != nil {
			_, _ = fmt.Fprintf(errOut, "GitLab token exchange failed: %v\n", err)
			http.Error(w, "Token exchange failed", http.StatusBadGateway)
			return
		}

		createdAt := tokenResp.CreatedAt
		if createdAt == 0 {
			createdAt = time.Now().Unix()
		}
		var expiresAt int64
		if tokenResp.ExpiresIn > 0 {
			expiresAt = createdAt + int64(tokenResp.ExpiresIn)
		}

		// Generate our own auth code for the MCP client
		ourCode, err := generateToken()
		if err != nil {
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}

		store.addCode(&oauthAuthCode{
			Code:               ourCode,
			ClientID:           pending.RegisteredClientID,
			RedirectURI:        pending.ClientRedirectURI,
			CodeChallenge:      pending.CodeChallenge,
			CodeChallengeMethod: pending.CodeChallengeMethod,
			GitLabAccessToken:  tokenResp.AccessToken,
			GitLabRefreshToken: tokenResp.RefreshToken,
			GitLabExpiresAt:    expiresAt,
			CreatedAt:          time.Now(),
		})

		_, _ = fmt.Fprintf(errOut, "GitLab OAuth successful, redirecting to client\n")

		// Redirect to the MCP client with our auth code
		redirectURL := pending.ClientRedirectURI + "?code=" + ourCode
		if pending.ClientState != "" {
			redirectURL += "&state=" + pending.ClientState
		}
		http.Redirect(w, r, redirectURL, http.StatusFound)
	}
}

// oauthTokenHandler exchanges our auth code for a session access token,
// or refreshes an existing session via refresh_token grant.
func oauthTokenHandler(store *mcpSessionStore, gitlabHost, gitlabClientID, gitlabCallbackURI string, errOut io.Writer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		if err := r.ParseForm(); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		switch r.FormValue("grant_type") {
		case "authorization_code":
			handleAuthCodeGrant(w, r, store, gitlabHost, gitlabClientID, gitlabCallbackURI, errOut)
		case "refresh_token":
			handleRefreshGrant(w, r, store, errOut)
		default:
			jsonError(w, "unsupported_grant_type", "Supported: authorization_code, refresh_token", http.StatusBadRequest)
		}
	}
}

// handleAuthCodeGrant handles the authorization_code grant type.
func handleAuthCodeGrant(w http.ResponseWriter, r *http.Request, store *mcpSessionStore, gitlabHost, gitlabClientID, gitlabCallbackURI string, errOut io.Writer) {
	code := r.FormValue("code")
	clientID := r.FormValue("client_id")
	redirectURI := r.FormValue("redirect_uri")
	codeVerifier := r.FormValue("code_verifier")

	// Look up and consume the auth code
	ac := store.takeCode(code)
	if ac == nil {
		jsonError(w, "invalid_grant", "Invalid or expired authorization code", http.StatusBadRequest)
		return
	}

	// Validate client_id and redirect_uri
	if ac.ClientID != clientID {
		jsonError(w, "invalid_grant", "client_id mismatch", http.StatusBadRequest)
		return
	}
	if ac.RedirectURI != redirectURI {
		jsonError(w, "invalid_grant", "redirect_uri mismatch", http.StatusBadRequest)
		return
	}

	// Validate PKCE
	if ac.CodeChallenge != "" {
		expectedChallenge := auth.GenerateCodeChallenge(codeVerifier)
		if subtle.ConstantTimeCompare([]byte(expectedChallenge), []byte(ac.CodeChallenge)) != 1 {
			jsonError(w, "invalid_grant", "PKCE verification failed", http.StatusBadRequest)
			return
		}
	}

	// Generate session token and MCP refresh token
	sessionToken, err := generateToken()
	if err != nil {
		jsonError(w, "server_error", "Failed to generate token", http.StatusInternalServerError)
		return
	}
	mcpRefreshToken, err := generateToken()
	if err != nil {
		jsonError(w, "server_error", "Failed to generate refresh token", http.StatusInternalServerError)
		return
	}

	store.addSession(&mcpSession{
		BearerToken:       sessionToken,
		GitLabHost:        gitlabHost,
		AccessToken:       ac.GitLabAccessToken,
		RefreshToken:      ac.GitLabRefreshToken,
		MCPRefreshToken:   mcpRefreshToken,
		TokenExpiresAt:    ac.GitLabExpiresAt,
		ClientID:          clientID,
		GitLabClientID:    gitlabClientID,
		GitLabRedirectURI: gitlabCallbackURI,
	})

	_, _ = fmt.Fprintf(errOut, "New OAuth session created for client %s\n", clientID)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"access_token":  sessionToken,
		"token_type":    "bearer",
		"expires_in":    defaultTokenExpiry,
		"refresh_token": mcpRefreshToken,
	})
}

// handleRefreshGrant handles the refresh_token grant type.
// It refreshes the underlying GitLab token and issues a new MCP session token.
func handleRefreshGrant(w http.ResponseWriter, r *http.Request, store *mcpSessionStore, errOut io.Writer) {
	mcpRefreshToken := r.FormValue("refresh_token")
	if mcpRefreshToken == "" {
		jsonError(w, "invalid_request", "refresh_token is required", http.StatusBadRequest)
		return
	}

	// Find session by MCP refresh token
	var sess *mcpSession
	store.mu.RLock()
	for _, s := range store.sessions {
		if s.MCPRefreshToken == mcpRefreshToken {
			sess = s
			break
		}
	}
	store.mu.RUnlock()

	if sess == nil {
		jsonError(w, "invalid_grant", "Invalid refresh token", http.StatusBadRequest)
		return
	}

	// Refresh the GitLab token
	if err := store.refreshSession(sess); err != nil {
		_, _ = fmt.Fprintf(errOut, "GitLab token refresh failed during MCP refresh: %v\n", err)
		if errors.Is(err, errRefreshAuthFailed) {
			// Terminal: GitLab rejected the refresh token. Drop the session
			// and tell the client to re-authorize.
			store.deleteSession(sess.BearerToken)
			jsonError(w, "invalid_grant", "GitLab token refresh failed — re-authorization required", http.StatusBadRequest)
			return
		}
		// Transient (VPN/passthrough blip, GitLab 5xx). Keep the session so the
		// client can retry without losing its authorization.
		jsonError(w, "temporarily_unavailable", "GitLab unreachable — try again", http.StatusServiceUnavailable)
		return
	}

	// Generate new MCP session bearer and refresh tokens
	newBearer, err := generateToken()
	if err != nil {
		jsonError(w, "server_error", "Failed to generate token", http.StatusInternalServerError)
		return
	}
	newMCPRefresh, err := generateToken()
	if err != nil {
		jsonError(w, "server_error", "Failed to generate refresh token", http.StatusInternalServerError)
		return
	}

	// Remove old session, create new one
	store.mu.Lock()
	delete(store.sessions, sess.BearerToken)
	newSess := &mcpSession{
		BearerToken:       newBearer,
		GitLabHost:        sess.GitLabHost,
		AccessToken:       sess.AccessToken,
		RefreshToken:      sess.RefreshToken,
		MCPRefreshToken:   newMCPRefresh,
		TokenExpiresAt:    sess.TokenExpiresAt,
		ClientID:          sess.ClientID,
		GitLabClientID:    sess.GitLabClientID,
		GitLabRedirectURI: sess.GitLabRedirectURI,
		Username:          sess.Username,
	}
	store.sessions[newBearer] = newSess
	store.saveToDisk()
	store.mu.Unlock()

	_, _ = fmt.Fprintf(errOut, "MCP session refreshed for client %s\n", sess.ClientID)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"access_token":  newBearer,
		"token_type":    "bearer",
		"expires_in":    defaultTokenExpiry,
		"refresh_token": newMCPRefresh,
	})
}

// jsonError writes an OAuth-compliant JSON error response.
func jsonError(w http.ResponseWriter, errCode, description string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error":             errCode,
		"error_description": description,
	})
}
