package cmd

import (
	"context"
	"fmt"

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

	return cmd
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
			fmt.Fprintln(f.IOStreams.ErrOut, "glab MCP server running on stdio")
			return server.Run(context.Background(), &mcp.StdioTransport{})
		},
	}
}
