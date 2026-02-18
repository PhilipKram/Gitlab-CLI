package tools

import (
	"context"
	"fmt"

	"github.com/PhilipKram/gitlab-cli/internal/cmdutil"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// RegisterRepoTools registers all repository tools on the server.
func RegisterRepoTools(server *mcp.Server, f *cmdutil.Factory) {
	registerRepoList(server, f)
	registerRepoView(server, f)
}

func registerRepoList(server *mcp.Server, f *cmdutil.Factory) {
	type Input struct {
		Group string `json:"group,omitempty" jsonschema:"group or namespace to list repos for"`
		Mine  bool   `json:"mine,omitempty"  jsonschema:"list only repositories you own or are a member of"`
		Limit int64  `json:"limit,omitempty" jsonschema:"maximum number of results (default 30)"`
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "repo_list",
		Description: "List GitLab repositories for a user or group",
	}, func(_ context.Context, _ *mcp.CallToolRequest, in Input) (*mcp.CallToolResult, any, error) {
		client, err := f.Client()
		if err != nil {
			return nil, nil, err
		}
		perPage := clampPerPage(in.Limit)

		if in.Group != "" {
			opts := &gitlab.ListGroupProjectsOptions{
				ListOptions: gitlab.ListOptions{PerPage: perPage},
			}
			projects, _, err := client.Groups.ListGroupProjects(in.Group, opts)
			if err != nil {
				return nil, nil, fmt.Errorf("listing group projects: %w", err)
			}
			return textResult(projects)
		}

		opts := &gitlab.ListProjectsOptions{
			ListOptions: gitlab.ListOptions{PerPage: perPage},
			Membership:  gitlab.Ptr(in.Mine),
		}
		projects, _, err := client.Projects.ListProjects(opts)
		if err != nil {
			return nil, nil, fmt.Errorf("listing projects: %w", err)
		}
		return textResult(projects)
	})
}

func registerRepoView(server *mcp.Server, f *cmdutil.Factory) {
	type Input struct {
		Repo string `json:"repo" jsonschema:"repository in OWNER/REPO or HOST/OWNER/REPO format"`
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "repo_view",
		Description: "View details of a GitLab repository",
	}, func(_ context.Context, _ *mcp.CallToolRequest, in Input) (*mcp.CallToolResult, any, error) {
		if in.Repo == "" {
			return nil, nil, fmt.Errorf("repo is required")
		}
		client, project, err := resolveClientAndProject(f, in.Repo)
		if err != nil {
			return nil, nil, err
		}
		p, _, err := client.Projects.GetProject(project, nil)
		if err != nil {
			return nil, nil, fmt.Errorf("getting project: %w", err)
		}
		return textResult(p)
	})
}
