package tools

import (
	"context"
	"fmt"

	"github.com/PhilipKram/gitlab-cli/internal/cmdutil"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// RegisterDeploymentTools registers all deployment tools on the server.
func RegisterDeploymentTools(server *mcp.Server, f *cmdutil.Factory) {
	registerDeploymentList(server, f)
	registerDeploymentView(server, f)
}

func registerDeploymentList(server *mcp.Server, f *cmdutil.Factory) {
	type Input struct {
		Repo        string `json:"repo,omitempty"        jsonschema:"repository in OWNER/REPO or HOST/OWNER/REPO format"`
		Environment string `json:"environment,omitempty" jsonschema:"filter by environment name"`
		Status      string `json:"status,omitempty"      jsonschema:"filter by status: created, running, success, failed, canceled"`
		OrderBy     string `json:"order_by,omitempty"    jsonschema:"order by: id, iid, created_at, updated_at, ref"`
		Sort        string `json:"sort,omitempty"        jsonschema:"sort order: asc or desc"`
		Limit       int64  `json:"limit,omitempty"       jsonschema:"maximum number of results (default 30)"`
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "deployment_list",
		Description: "List deployments for a GitLab project",
	}, func(_ context.Context, _ *mcp.CallToolRequest, in Input) (*mcp.CallToolResult, any, error) {
		client, project, err := resolveClientAndProject(f, in.Repo)
		if err != nil {
			return nil, nil, err
		}
		opts := &gitlab.ListProjectDeploymentsOptions{
			ListOptions: gitlab.ListOptions{PerPage: clampPerPage(in.Limit)},
		}
		if in.Environment != "" {
			opts.Environment = &in.Environment
		}
		if in.Status != "" {
			opts.Status = &in.Status
		}
		if in.OrderBy != "" {
			opts.OrderBy = &in.OrderBy
		}
		if in.Sort != "" {
			opts.Sort = &in.Sort
		}
		deployments, _, err := client.Deployments.ListProjectDeployments(project, opts)
		if err != nil {
			return nil, nil, fmt.Errorf("listing deployments: %w", err)
		}
		return textResult(deployments)
	})
}

func registerDeploymentView(server *mcp.Server, f *cmdutil.Factory) {
	type Input struct {
		Deployment int64  `json:"deployment"     jsonschema:"deployment ID"`
		Repo       string `json:"repo,omitempty" jsonschema:"repository in OWNER/REPO or HOST/OWNER/REPO format"`
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "deployment_view",
		Description: "View details of a specific deployment",
	}, func(_ context.Context, _ *mcp.CallToolRequest, in Input) (*mcp.CallToolResult, any, error) {
		if err := requireID(in.Deployment, "deployment"); err != nil {
			return nil, nil, err
		}
		client, project, err := resolveClientAndProject(f, in.Repo)
		if err != nil {
			return nil, nil, err
		}
		deployment, _, err := client.Deployments.GetProjectDeployment(project, in.Deployment)
		if err != nil {
			return nil, nil, fmt.Errorf("getting deployment: %w", err)
		}
		return textResult(deployment)
	})
}
