package tools

import (
	"context"
	"fmt"

	"github.com/PhilipKram/gitlab-cli/internal/cmdutil"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// RegisterUserTools registers all user tools on the server.
func RegisterUserTools(server *mcp.Server, f *cmdutil.Factory) {
	registerUserWhoami(server, f)
}

func registerUserWhoami(server *mcp.Server, f *cmdutil.Factory) {
	type Input struct{}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "user_whoami",
		Description: "Get the currently authenticated GitLab user",
	}, func(_ context.Context, _ *mcp.CallToolRequest, in Input) (*mcp.CallToolResult, any, error) {
		client, err := f.Client()
		if err != nil {
			return nil, nil, err
		}

		user, _, err := client.Users.CurrentUser()
		if err != nil {
			return nil, nil, fmt.Errorf("getting current user: %w", err)
		}
		return textResult(user)
	})
}
