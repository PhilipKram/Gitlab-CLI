package tools

import (
	"context"
	"fmt"

	"github.com/PhilipKram/gitlab-cli/internal/cmdutil"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// RegisterTagTools registers all tag tools on the server.
func RegisterTagTools(server *mcp.Server, f *cmdutil.Factory) {
	registerTagList(server, f)
	registerTagCreate(server, f)
	registerTagDelete(server, f)
}

func registerTagList(server *mcp.Server, f *cmdutil.Factory) {
	type Input struct {
		Repo  string `json:"repo,omitempty"  jsonschema:"repository in OWNER/REPO or HOST/OWNER/REPO format"`
		Limit int64  `json:"limit,omitempty" jsonschema:"maximum number of results (default 30)"`
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "tag_list",
		Description: "List repository tags",
	}, func(_ context.Context, _ *mcp.CallToolRequest, in Input) (*mcp.CallToolResult, any, error) {
		client, project, err := resolveClientAndProject(f, in.Repo)
		if err != nil {
			return nil, nil, err
		}
		opts := &gitlab.ListTagsOptions{
			ListOptions: gitlab.ListOptions{PerPage: clampPerPage(in.Limit)},
		}
		tags, _, err := client.Tags.ListTags(project, opts)
		if err != nil {
			return nil, nil, fmt.Errorf("listing tags: %w", err)
		}
		return textResult(tags)
	})
}

func registerTagCreate(server *mcp.Server, f *cmdutil.Factory) {
	type Input struct {
		Name    string `json:"name"                jsonschema:"tag name"`
		Ref     string `json:"ref,omitempty"       jsonschema:"source branch or commit SHA (default main)"`
		Message string `json:"message,omitempty"   jsonschema:"tag message (creates annotated tag)"`
		Repo    string `json:"repo,omitempty"      jsonschema:"repository in OWNER/REPO or HOST/OWNER/REPO format"`
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "tag_create",
		Description: "Create a repository tag",
	}, func(_ context.Context, _ *mcp.CallToolRequest, in Input) (*mcp.CallToolResult, any, error) {
		if err := requireString(in.Name, "name"); err != nil {
			return nil, nil, err
		}
		client, project, err := resolveClientAndProject(f, in.Repo)
		if err != nil {
			return nil, nil, err
		}
		ref := in.Ref
		if ref == "" {
			ref = "main"
		}
		opts := &gitlab.CreateTagOptions{
			TagName: &in.Name,
			Ref:     &ref,
		}
		if in.Message != "" {
			opts.Message = &in.Message
		}
		tag, _, err := client.Tags.CreateTag(project, opts)
		if err != nil {
			return nil, nil, fmt.Errorf("creating tag: %w", err)
		}
		return plainResult(fmt.Sprintf("Created tag %q from %q", tag.Name, ref)), nil, nil
	})
}

func registerTagDelete(server *mcp.Server, f *cmdutil.Factory) {
	type Input struct {
		Name string `json:"name"            jsonschema:"tag name to delete"`
		Repo string `json:"repo,omitempty"  jsonschema:"repository in OWNER/REPO or HOST/OWNER/REPO format"`
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "tag_delete",
		Description: "Delete a repository tag",
	}, func(_ context.Context, _ *mcp.CallToolRequest, in Input) (*mcp.CallToolResult, any, error) {
		if err := requireString(in.Name, "name"); err != nil {
			return nil, nil, err
		}
		client, project, err := resolveClientAndProject(f, in.Repo)
		if err != nil {
			return nil, nil, err
		}
		_, err = client.Tags.DeleteTag(project, in.Name)
		if err != nil {
			return nil, nil, fmt.Errorf("deleting tag: %w", err)
		}
		return plainResult(fmt.Sprintf("Deleted tag %q", in.Name)), nil, nil
	})
}
