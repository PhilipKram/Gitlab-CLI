package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/PhilipKram/gitlab-cli/internal/cmdutil"
	glabmcp "github.com/PhilipKram/gitlab-cli/internal/mcp"
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
		scope  string
		client string
	)

	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install glab as an MCP server in an AI client",
		Long: `Register glab as a Model Context Protocol server in the specified AI client.

Supported clients:
  claude-code     - Claude Code CLI (default)
  claude-desktop  - Claude Desktop app`,
		Example: `  # Install for Claude Code (default)
  $ glab mcp install

  # Install for Claude Desktop
  $ glab mcp install --client claude-desktop

  # Install with project scope for Claude Code
  $ glab mcp install --scope project`,
		RunE: func(cmd *cobra.Command, args []string) error {
			out := f.IOStreams.Out

			glabPath := glabBinaryPath()
			configJSON, err := mcpConfigJSON(glabPath)
			if err != nil {
				return err
			}

			switch client {
			case "claude-code":
				return installClaudeCode(out, scope, configJSON)
			case "claude-desktop":
				return installClaudeDesktop(out, glabPath)
			default:
				return fmt.Errorf("unsupported client: %s (supported: claude-code, claude-desktop)", client)
			}
		},
	}

	cmd.Flags().StringVar(&scope, "scope", "user", "Installation scope: user, local, or project")
	cmd.Flags().StringVar(&client, "client", "claude-code", "AI client: claude-code or claude-desktop")

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
	return &cobra.Command{
		Use:   "serve",
		Short: "Start the MCP server over stdio",
		Long: `Start a Model Context Protocol server over stdio, exposing 33 GitLab tools
to AI assistants such as Claude, GitHub Copilot, and any MCP-compatible client.

Authentication uses the credentials already stored by 'glab auth login'.
Run from inside a git repo to auto-detect the project, or pass --repo to
specify it explicitly.`,
		Example: `  # Start from inside a git repo (project auto-detected)
  $ glab mcp serve

  # Start with an explicit project
  $ glab -R gitlab.example.com/owner/repo mcp serve`,
		RunE: func(cmd *cobra.Command, args []string) error {
			server := glabmcp.NewMCPServer(f)
			_, _ = fmt.Fprintln(f.IOStreams.ErrOut, "glab MCP server running on stdio")
			return server.Run(context.Background(), &mcp.StdioTransport{})
		},
	}
}
