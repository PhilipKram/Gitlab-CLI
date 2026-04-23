package tools

import (
	"context"
	"fmt"

	"github.com/PhilipKram/gitlab-cli/internal/cmdutil"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// RegisterEnvironmentTools registers all environment tools on the server.
func RegisterEnvironmentTools(server *mcp.Server, f *cmdutil.Factory) {
	registerEnvironmentList(server, f)
	registerEnvironmentView(server, f)
	registerEnvironmentStop(server, f)
	registerEnvironmentDelete(server, f)
}

func registerEnvironmentList(server *mcp.Server, f *cmdutil.Factory) {
	type Input struct {
		Repo  string `json:"repo,omitempty"  jsonschema:"repository in OWNER/REPO or HOST/OWNER/REPO format"`
		State string `json:"state,omitempty" jsonschema:"filter by state: available or stopped"`
		Limit int64  `json:"limit,omitempty" jsonschema:"maximum number of results (default 30)"`
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "environment_list",
		Description: "List environments for a GitLab project",
	}, func(_ context.Context, _ *mcp.CallToolRequest, in Input) (*mcp.CallToolResult, any, error) {
		client, project, err := resolveClientAndProject(f, in.Repo)
		if err != nil {
			return nil, nil, err
		}
		opts := &gitlab.ListEnvironmentsOptions{
			ListOptions: gitlab.ListOptions{PerPage: clampPerPage(in.Limit)},
		}
		if in.State != "" {
			opts.States = &in.State
		}
		environments, _, err := client.Environments.ListEnvironments(project, opts)
		if err != nil {
			return nil, nil, fmt.Errorf("listing environments: %w", err)
		}
		return textResult(environments)
	})
}

func registerEnvironmentView(server *mcp.Server, f *cmdutil.Factory) {
	type Input struct {
		Environment int64  `json:"environment" jsonschema:"environment ID"`
		Repo string `json:"repo,omitempty" jsonschema:"repository in OWNER/REPO or HOST/OWNER/REPO format"`
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "environment_view",
		Description: "View details of a specific environment",
	}, func(_ context.Context, _ *mcp.CallToolRequest, in Input) (*mcp.CallToolResult, any, error) {
		if err := requireID(in.Environment, "environment"); err != nil {
			return nil, nil, err
		}
		client, project, err := resolveClientAndProject(f, in.Repo)
		if err != nil {
			return nil, nil, err
		}
		environment, _, err := client.Environments.GetEnvironment(project, in.Environment)
		if err != nil {
			return nil, nil, fmt.Errorf("getting environment: %w", err)
		}
		return textResult(environment)
	})
}

func registerEnvironmentStop(server *mcp.Server, f *cmdutil.Factory) {
	type Input struct {
		Environment int64  `json:"environment" jsonschema:"environment ID"`
		Repo string `json:"repo,omitempty" jsonschema:"repository in OWNER/REPO or HOST/OWNER/REPO format"`
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "environment_stop",
		Description: "Stop an environment",
	}, func(_ context.Context, _ *mcp.CallToolRequest, in Input) (*mcp.CallToolResult, any, error) {
		if err := requireID(in.Environment, "environment"); err != nil {
			return nil, nil, err
		}
		client, project, err := resolveClientAndProject(f, in.Repo)
		if err != nil {
			return nil, nil, err
		}
		_, _, err = client.Environments.StopEnvironment(project, in.Environment, nil)
		if err != nil {
			return nil, nil, fmt.Errorf("stopping environment: %w", err)
		}
		return plainResult(fmt.Sprintf("Stopped environment #%d", in.Environment)), nil, nil
	})
}

func registerEnvironmentDelete(server *mcp.Server, f *cmdutil.Factory) {
	type Input struct {
		Environment int64  `json:"environment" jsonschema:"environment ID"`
		Repo string `json:"repo,omitempty" jsonschema:"repository in OWNER/REPO or HOST/OWNER/REPO format"`
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "environment_delete",
		Description: "Delete an environment",
	}, func(_ context.Context, _ *mcp.CallToolRequest, in Input) (*mcp.CallToolResult, any, error) {
		if err := requireID(in.Environment, "environment"); err != nil {
			return nil, nil, err
		}
		client, project, err := resolveClientAndProject(f, in.Repo)
		if err != nil {
			return nil, nil, err
		}
		_, err = client.Environments.DeleteEnvironment(project, in.Environment)
		if err != nil {
			return nil, nil, fmt.Errorf("deleting environment: %w", err)
		}
		return plainResult(fmt.Sprintf("Deleted environment #%d", in.Environment)), nil, nil
	})
}
