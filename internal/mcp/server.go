package mcp

import (
	"github.com/PhilipKram/gitlab-cli/internal/cmdutil"
	"github.com/PhilipKram/gitlab-cli/internal/mcp/tools"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// NewMCPServer creates and configures the MCP server with all GitLab tools.
func NewMCPServer(f *cmdutil.Factory) *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "glab-mcp-server",
		Version: "0.1.0",
	}, nil)

	tools.RegisterMRTools(server, f)
	tools.RegisterIssueTools(server, f)
	tools.RegisterPipelineTools(server, f)
	tools.RegisterRepoTools(server, f)
	tools.RegisterReleaseTools(server, f)
	tools.RegisterLabelTools(server, f)
	tools.RegisterSnippetTools(server, f)

	return server
}
