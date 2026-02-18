package tools

import (
	"context"
	"fmt"

	"github.com/PhilipKram/gitlab-cli/internal/cmdutil"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// RegisterReleaseTools registers all release tools on the server.
func RegisterReleaseTools(server *mcp.Server, f *cmdutil.Factory) {
	registerReleaseList(server, f)
	registerReleaseView(server, f)
	registerReleaseCreate(server, f)
	registerReleaseDelete(server, f)
}

func registerReleaseList(server *mcp.Server, f *cmdutil.Factory) {
	type Input struct {
		Repo  string `json:"repo,omitempty"  jsonschema:"repository in OWNER/REPO or HOST/OWNER/REPO format"`
		Limit int64  `json:"limit,omitempty" jsonschema:"maximum number of results (default 30)"`
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "release_list",
		Description: "List releases for a GitLab project",
	}, func(_ context.Context, _ *mcp.CallToolRequest, in Input) (*mcp.CallToolResult, any, error) {
		client, project, err := resolveClientAndProject(f, in.Repo)
		if err != nil {
			return nil, nil, err
		}
		releases, _, err := client.Releases.ListReleases(project, &gitlab.ListReleasesOptions{
			ListOptions: gitlab.ListOptions{PerPage: clampPerPage(in.Limit)},
		})
		if err != nil {
			return nil, nil, fmt.Errorf("listing releases: %w", err)
		}
		return textResult(releases)
	})
}

func registerReleaseView(server *mcp.Server, f *cmdutil.Factory) {
	type Input struct {
		Tag  string `json:"tag"             jsonschema:"release tag name (e.g. v1.0.0)"`
		Repo string `json:"repo,omitempty"  jsonschema:"repository in OWNER/REPO or HOST/OWNER/REPO format"`
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "release_view",
		Description: "View details of a specific release",
	}, func(_ context.Context, _ *mcp.CallToolRequest, in Input) (*mcp.CallToolResult, any, error) {
		if in.Tag == "" {
			return nil, nil, fmt.Errorf("tag is required")
		}
		client, project, err := resolveClientAndProject(f, in.Repo)
		if err != nil {
			return nil, nil, err
		}
		release, _, err := client.Releases.GetRelease(project, in.Tag)
		if err != nil {
			return nil, nil, fmt.Errorf("getting release: %w", err)
		}
		return textResult(release)
	})
}

func registerReleaseCreate(server *mcp.Server, f *cmdutil.Factory) {
	type Input struct {
		Tag         string `json:"tag"                   jsonschema:"tag name for the release (e.g. v1.0.0)"`
		Repo        string `json:"repo,omitempty"        jsonschema:"repository in OWNER/REPO or HOST/OWNER/REPO format"`
		Name        string `json:"name,omitempty"        jsonschema:"release name"`
		Description string `json:"description,omitempty" jsonschema:"release description / changelog"`
		Ref         string `json:"ref,omitempty"         jsonschema:"branch or commit SHA to create the tag from"`
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "release_create",
		Description: "Create a new release",
	}, func(_ context.Context, _ *mcp.CallToolRequest, in Input) (*mcp.CallToolResult, any, error) {
		if in.Tag == "" {
			return nil, nil, fmt.Errorf("tag is required")
		}
		client, project, err := resolveClientAndProject(f, in.Repo)
		if err != nil {
			return nil, nil, err
		}
		opts := &gitlab.CreateReleaseOptions{
			TagName: &in.Tag,
		}
		if in.Name != "" {
			opts.Name = &in.Name
		}
		if in.Description != "" {
			opts.Description = &in.Description
		}
		if in.Ref != "" {
			opts.Ref = &in.Ref
		}
		release, _, err := client.Releases.CreateRelease(project, opts)
		if err != nil {
			return nil, nil, fmt.Errorf("creating release: %w", err)
		}
		return plainResult(fmt.Sprintf("Created release %s", release.TagName)), nil, nil
	})
}

func registerReleaseDelete(server *mcp.Server, f *cmdutil.Factory) {
	type Input struct {
		Tag  string `json:"tag"             jsonschema:"release tag name to delete"`
		Repo string `json:"repo,omitempty"  jsonschema:"repository in OWNER/REPO or HOST/OWNER/REPO format"`
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "release_delete",
		Description: "Delete a release",
	}, func(_ context.Context, _ *mcp.CallToolRequest, in Input) (*mcp.CallToolResult, any, error) {
		if in.Tag == "" {
			return nil, nil, fmt.Errorf("tag is required")
		}
		client, project, err := resolveClientAndProject(f, in.Repo)
		if err != nil {
			return nil, nil, err
		}
		_, _, err = client.Releases.DeleteRelease(project, in.Tag)
		if err != nil {
			return nil, nil, fmt.Errorf("deleting release: %w", err)
		}
		return plainResult(fmt.Sprintf("Deleted release %s", in.Tag)), nil, nil
	})
}
