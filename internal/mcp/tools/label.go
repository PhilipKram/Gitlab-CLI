package tools

import (
	"context"
	"fmt"

	"github.com/PhilipKram/gitlab-cli/internal/cmdutil"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// RegisterLabelTools registers all label tools on the server.
func RegisterLabelTools(server *mcp.Server, f *cmdutil.Factory) {
	registerLabelList(server, f)
	registerLabelCreate(server, f)
	registerLabelDelete(server, f)
}

func registerLabelList(server *mcp.Server, f *cmdutil.Factory) {
	type Input struct {
		Repo   string `json:"repo,omitempty"   jsonschema:"repository in OWNER/REPO or HOST/OWNER/REPO format"`
		Search string `json:"search,omitempty" jsonschema:"search labels by name"`
		Limit  int64  `json:"limit,omitempty"  jsonschema:"maximum number of results (default 30)"`
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "label_list",
		Description: "List labels for a GitLab project",
	}, func(_ context.Context, _ *mcp.CallToolRequest, in Input) (*mcp.CallToolResult, any, error) {
		client, project, err := resolveClientAndProject(f, in.Repo)
		if err != nil {
			return nil, nil, err
		}
		opts := &gitlab.ListLabelsOptions{
			ListOptions: gitlab.ListOptions{PerPage: clampPerPage(in.Limit)},
		}
		if in.Search != "" {
			opts.Search = &in.Search
		}
		labels, _, err := client.Labels.ListLabels(project, opts)
		if err != nil {
			return nil, nil, fmt.Errorf("listing labels: %w", err)
		}
		return textResult(labels)
	})
}

func registerLabelCreate(server *mcp.Server, f *cmdutil.Factory) {
	type Input struct {
		Name        string `json:"name"                  jsonschema:"label name"`
		Color       string `json:"color"                 jsonschema:"label color in hex format (e.g. #FF0000)"`
		Repo        string `json:"repo,omitempty"        jsonschema:"repository in OWNER/REPO or HOST/OWNER/REPO format"`
		Description string `json:"description,omitempty" jsonschema:"label description"`
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "label_create",
		Description: "Create a new label",
	}, func(_ context.Context, _ *mcp.CallToolRequest, in Input) (*mcp.CallToolResult, any, error) {
		if in.Name == "" {
			return nil, nil, fmt.Errorf("name is required")
		}
		if in.Color == "" {
			return nil, nil, fmt.Errorf("color is required")
		}
		client, project, err := resolveClientAndProject(f, in.Repo)
		if err != nil {
			return nil, nil, err
		}
		createOpts := &gitlab.CreateLabelOptions{
			Name:  &in.Name,
			Color: &in.Color,
		}
		if in.Description != "" {
			createOpts.Description = &in.Description
		}
		label, _, err := client.Labels.CreateLabel(project, createOpts)
		if err != nil {
			return nil, nil, fmt.Errorf("creating label: %w", err)
		}
		return plainResult(fmt.Sprintf("Created label %q (%s)", label.Name, label.Color)), nil, nil
	})
}

func registerLabelDelete(server *mcp.Server, f *cmdutil.Factory) {
	type Input struct {
		Name string `json:"name"            jsonschema:"label name to delete"`
		Repo string `json:"repo,omitempty"  jsonschema:"repository in OWNER/REPO or HOST/OWNER/REPO format"`
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "label_delete",
		Description: "Delete a label",
	}, func(_ context.Context, _ *mcp.CallToolRequest, in Input) (*mcp.CallToolResult, any, error) {
		if in.Name == "" {
			return nil, nil, fmt.Errorf("name is required")
		}
		client, project, err := resolveClientAndProject(f, in.Repo)
		if err != nil {
			return nil, nil, err
		}
		_, err = client.Labels.DeleteLabel(project, in.Name, nil)
		if err != nil {
			return nil, nil, fmt.Errorf("deleting label: %w", err)
		}
		return plainResult(fmt.Sprintf("Deleted label %q", in.Name)), nil, nil
	})
}
