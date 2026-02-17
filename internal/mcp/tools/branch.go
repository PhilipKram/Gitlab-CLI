package tools

import (
	"context"
	"fmt"

	"github.com/PhilipKram/gitlab-cli/internal/cmdutil"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// RegisterBranchTools registers all branch tools on the server.
func RegisterBranchTools(server *mcp.Server, f *cmdutil.Factory) {
	registerBranchList(server, f)
	registerBranchCreate(server, f)
	registerBranchDelete(server, f)
}

func registerBranchList(server *mcp.Server, f *cmdutil.Factory) {
	type Input struct {
		Repo   string `json:"repo,omitempty"   jsonschema:"repository in OWNER/REPO or HOST/OWNER/REPO format"`
		Search string `json:"search,omitempty" jsonschema:"search branches by name"`
		Limit  int64  `json:"limit,omitempty"  jsonschema:"maximum number of results (default 30)"`
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "branch_list",
		Description: "List branches for a GitLab project",
	}, func(_ context.Context, _ *mcp.CallToolRequest, in Input) (*mcp.CallToolResult, any, error) {
		client, project, err := resolveClientAndProject(f, in.Repo)
		if err != nil {
			return nil, nil, err
		}

		opts := &gitlab.ListBranchesOptions{
			ListOptions: gitlab.ListOptions{PerPage: clampPerPage(in.Limit)},
		}
		if in.Search != "" {
			opts.Search = &in.Search
		}

		branches, _, err := client.Branches.ListBranches(project, opts)
		if err != nil {
			return nil, nil, fmt.Errorf("listing branches: %w", err)
		}
		return textResult(branches)
	})
}

func registerBranchCreate(server *mcp.Server, f *cmdutil.Factory) {
	type Input struct {
		Repo string `json:"repo,omitempty" jsonschema:"repository in OWNER/REPO or HOST/OWNER/REPO format"`
		Name string `json:"name"          jsonschema:"name of the branch to create"`
		Ref  string `json:"ref"           jsonschema:"branch name or commit SHA to create branch from"`
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "branch_create",
		Description: "Create a new branch in a GitLab project",
	}, func(_ context.Context, _ *mcp.CallToolRequest, in Input) (*mcp.CallToolResult, any, error) {
		client, project, err := resolveClientAndProject(f, in.Repo)
		if err != nil {
			return nil, nil, err
		}

		opts := &gitlab.CreateBranchOptions{
			Branch: &in.Name,
			Ref:    &in.Ref,
		}

		branch, _, err := client.Branches.CreateBranch(project, opts)
		if err != nil {
			return nil, nil, fmt.Errorf("creating branch: %w", err)
		}
		return textResult(branch)
	})
}

func registerBranchDelete(server *mcp.Server, f *cmdutil.Factory) {
	type Input struct {
		Repo string `json:"repo,omitempty" jsonschema:"repository in OWNER/REPO or HOST/OWNER/REPO format"`
		Name string `json:"name"          jsonschema:"name of the branch to delete"`
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "branch_delete",
		Description: "Delete a branch from a GitLab project",
	}, func(_ context.Context, _ *mcp.CallToolRequest, in Input) (*mcp.CallToolResult, any, error) {
		client, project, err := resolveClientAndProject(f, in.Repo)
		if err != nil {
			return nil, nil, err
		}

		_, err = client.Branches.DeleteBranch(project, in.Name)
		if err != nil {
			return nil, nil, fmt.Errorf("deleting branch: %w", err)
		}
		return plainResult(fmt.Sprintf("Branch %q deleted successfully", in.Name)), nil, nil
	})
}
