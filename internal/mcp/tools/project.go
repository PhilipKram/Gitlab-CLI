package tools

import (
	"context"
	"fmt"

	"github.com/PhilipKram/gitlab-cli/internal/cmdutil"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// RegisterProjectTools registers all project tools on the server.
func RegisterProjectTools(server *mcp.Server, f *cmdutil.Factory) {
	registerProjectList(server, f)
	registerProjectView(server, f)
	registerProjectMembers(server, f)
}

func registerProjectList(server *mcp.Server, f *cmdutil.Factory) {
	type Input struct {
		Group  string `json:"group,omitempty"  jsonschema:"filter by group path"`
		Search string `json:"search,omitempty" jsonschema:"search projects by name"`
		Limit  int64  `json:"limit,omitempty"  jsonschema:"maximum number of results (default 30)"`
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "project_list",
		Description: "List GitLab projects",
	}, func(_ context.Context, _ *mcp.CallToolRequest, in Input) (*mcp.CallToolResult, any, error) {
		client, err := f.Client()
		if err != nil {
			return nil, nil, err
		}

		if in.Group != "" {
			opts := &gitlab.ListGroupProjectsOptions{
				ListOptions: gitlab.ListOptions{PerPage: clampPerPage(in.Limit)},
			}
			if in.Search != "" {
				opts.Search = &in.Search
			}
			projects, _, err := client.Groups.ListGroupProjects(in.Group, opts)
			if err != nil {
				return nil, nil, fmt.Errorf("listing group projects: %w", err)
			}
			return textResult(projects)
		}

		trueVal := true
		opts := &gitlab.ListProjectsOptions{
			ListOptions: gitlab.ListOptions{PerPage: clampPerPage(in.Limit)},
			Membership:  &trueVal,
		}
		if in.Search != "" {
			opts.Search = &in.Search
		}
		projects, _, err := client.Projects.ListProjects(opts)
		if err != nil {
			return nil, nil, fmt.Errorf("listing projects: %w", err)
		}
		return textResult(projects)
	})
}

func registerProjectView(server *mcp.Server, f *cmdutil.Factory) {
	type Input struct {
		Repo string `json:"repo,omitempty" jsonschema:"project path in OWNER/REPO or HOST/OWNER/REPO format"`
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "project_view",
		Description: "View project details",
	}, func(_ context.Context, _ *mcp.CallToolRequest, in Input) (*mcp.CallToolResult, any, error) {
		client, project, err := resolveClientAndProject(f, in.Repo)
		if err != nil {
			return nil, nil, err
		}
		proj, _, err := client.Projects.GetProject(project, nil)
		if err != nil {
			return nil, nil, fmt.Errorf("getting project: %w", err)
		}
		return textResult(proj)
	})
}

func registerProjectMembers(server *mcp.Server, f *cmdutil.Factory) {
	type Input struct {
		Repo  string `json:"repo,omitempty"  jsonschema:"project path in OWNER/REPO or HOST/OWNER/REPO format"`
		Limit int64  `json:"limit,omitempty" jsonschema:"maximum number of results (default 30)"`
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "project_members",
		Description: "List project members",
	}, func(_ context.Context, _ *mcp.CallToolRequest, in Input) (*mcp.CallToolResult, any, error) {
		client, project, err := resolveClientAndProject(f, in.Repo)
		if err != nil {
			return nil, nil, err
		}
		opts := &gitlab.ListProjectMembersOptions{
			ListOptions: gitlab.ListOptions{PerPage: clampPerPage(in.Limit)},
		}
		members, _, err := client.ProjectMembers.ListAllProjectMembers(project, opts)
		if err != nil {
			return nil, nil, fmt.Errorf("listing project members: %w", err)
		}
		return textResult(members)
	})
}
