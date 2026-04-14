package tools

import (
	"context"
	"fmt"

	"github.com/PhilipKram/gitlab-cli/internal/cmdutil"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// RegisterRegistryTools registers all container registry tools on the server.
func RegisterRegistryTools(server *mcp.Server, f *cmdutil.Factory) {
	registerRegistryList(server, f)
	registerRegistryTags(server, f)
	registerRegistryView(server, f)
	registerRegistryDeleteTag(server, f)
}

func registerRegistryList(server *mcp.Server, f *cmdutil.Factory) {
	type Input struct {
		Repo  string `json:"repo,omitempty"  jsonschema:"repository in OWNER/REPO or HOST/OWNER/REPO format"`
		Limit int64  `json:"limit,omitempty" jsonschema:"maximum number of results (default 30)"`
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "registry_list",
		Description: "List container registry repositories for a project",
	}, func(_ context.Context, _ *mcp.CallToolRequest, in Input) (*mcp.CallToolResult, any, error) {
		client, project, err := resolveClientAndProject(f, in.Repo)
		if err != nil {
			return nil, nil, err
		}
		opts := &gitlab.ListProjectRegistryRepositoriesOptions{
			ListOptions: gitlab.ListOptions{PerPage: clampPerPage(in.Limit)},
		}
		repositories, _, err := client.ContainerRegistry.ListProjectRegistryRepositories(project, opts)
		if err != nil {
			return nil, nil, fmt.Errorf("listing container repositories: %w", err)
		}
		return textResult(repositories)
	})
}

func registerRegistryTags(server *mcp.Server, f *cmdutil.Factory) {
	type Input struct {
		RepositoryID int64  `json:"repository_id"   jsonschema:"container repository ID"`
		Repo         string `json:"repo,omitempty"   jsonschema:"repository in OWNER/REPO or HOST/OWNER/REPO format"`
		Limit        int64  `json:"limit,omitempty"  jsonschema:"maximum number of results (default 30)"`
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "registry_tags",
		Description: "List image tags for a container repository",
	}, func(_ context.Context, _ *mcp.CallToolRequest, in Input) (*mcp.CallToolResult, any, error) {
		if err := requireID(in.RepositoryID, "repository_id"); err != nil {
			return nil, nil, err
		}
		client, project, err := resolveClientAndProject(f, in.Repo)
		if err != nil {
			return nil, nil, err
		}
		opts := &gitlab.ListRegistryRepositoryTagsOptions{
			ListOptions: gitlab.ListOptions{PerPage: clampPerPage(in.Limit)},
		}
		tags, _, err := client.ContainerRegistry.ListRegistryRepositoryTags(project, in.RepositoryID, opts)
		if err != nil {
			return nil, nil, fmt.Errorf("listing repository tags: %w", err)
		}
		return textResult(tags)
	})
}

func registerRegistryView(server *mcp.Server, f *cmdutil.Factory) {
	type Input struct {
		RepositoryID int64  `json:"repository_id"   jsonschema:"container repository ID"`
		Tag          string `json:"tag,omitempty"    jsonschema:"view specific tag details"`
		Repo         string `json:"repo,omitempty"   jsonschema:"repository in OWNER/REPO or HOST/OWNER/REPO format"`
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "registry_view",
		Description: "View container repository or tag details",
	}, func(_ context.Context, _ *mcp.CallToolRequest, in Input) (*mcp.CallToolResult, any, error) {
		if err := requireID(in.RepositoryID, "repository_id"); err != nil {
			return nil, nil, err
		}

		if in.Tag != "" {
			client, project, err := resolveClientAndProject(f, in.Repo)
			if err != nil {
				return nil, nil, err
			}
			tagDetail, _, err := client.ContainerRegistry.GetRegistryRepositoryTagDetail(project, in.RepositoryID, in.Tag)
			if err != nil {
				return nil, nil, fmt.Errorf("getting tag details: %w", err)
			}
			return textResult(tagDetail)
		}

		// View repository details
		repoIDStr := fmt.Sprintf("%d", in.RepositoryID)
		trueVal := true
		opts := &gitlab.GetSingleRegistryRepositoryOptions{
			Tags:      &trueVal,
			TagsCount: &trueVal,
		}

		// GetSingleRegistryRepository uses the global registry API (no project needed)
		client, err := f.Client()
		if err != nil {
			return nil, nil, err
		}
		repository, _, err := client.ContainerRegistry.GetSingleRegistryRepository(repoIDStr, opts)
		if err != nil {
			return nil, nil, fmt.Errorf("getting container repository: %w", err)
		}
		return textResult(repository)
	})
}

func registerRegistryDeleteTag(server *mcp.Server, f *cmdutil.Factory) {
	type Input struct {
		RepositoryID int64  `json:"repository_id"  jsonschema:"container repository ID"`
		Tag          string `json:"tag"            jsonschema:"tag name to delete"`
		Repo         string `json:"repo,omitempty" jsonschema:"repository in OWNER/REPO or HOST/OWNER/REPO format"`
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "registry_delete_tag",
		Description: "Delete a container image tag",
	}, func(_ context.Context, _ *mcp.CallToolRequest, in Input) (*mcp.CallToolResult, any, error) {
		if err := requireID(in.RepositoryID, "repository_id"); err != nil {
			return nil, nil, err
		}
		if err := requireString(in.Tag, "tag"); err != nil {
			return nil, nil, err
		}
		client, project, err := resolveClientAndProject(f, in.Repo)
		if err != nil {
			return nil, nil, err
		}
		_, err = client.ContainerRegistry.DeleteRegistryRepositoryTag(project, in.RepositoryID, in.Tag)
		if err != nil {
			return nil, nil, fmt.Errorf("deleting tag: %w", err)
		}
		return plainResult(fmt.Sprintf("Deleted tag %q from repository %d", in.Tag, in.RepositoryID)), nil, nil
	})
}
