package tools

import (
	"context"
	"fmt"

	"github.com/PhilipKram/gitlab-cli/internal/cmdutil"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// RegisterSnippetTools registers all snippet tools on the server.
func RegisterSnippetTools(server *mcp.Server, f *cmdutil.Factory) {
	registerSnippetList(server, f)
	registerSnippetView(server, f)
	registerSnippetCreate(server, f)
	registerSnippetDelete(server, f)
}

func registerSnippetList(server *mcp.Server, f *cmdutil.Factory) {
	type Input struct {
		Limit int64 `json:"limit,omitempty" jsonschema:"maximum number of results (default 30)"`
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "snippet_list",
		Description: "List your personal GitLab snippets",
	}, func(_ context.Context, _ *mcp.CallToolRequest, in Input) (*mcp.CallToolResult, any, error) {
		client, err := f.Client()
		if err != nil {
			return nil, nil, err
		}
		snippets, _, err := client.Snippets.ListSnippets(&gitlab.ListSnippetsOptions{
			ListOptions: gitlab.ListOptions{PerPage: clampPerPage(in.Limit)},
		})
		if err != nil {
			return nil, nil, fmt.Errorf("listing snippets: %w", err)
		}
		return textResult(snippets)
	})
}

func registerSnippetView(server *mcp.Server, f *cmdutil.Factory) {
	type Input struct {
		Snippet int64 `json:"snippet" jsonschema:"snippet ID"`
		Raw     bool  `json:"raw,omitempty" jsonschema:"return raw file content instead of metadata"`
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "snippet_view",
		Description: "View a snippet's metadata or raw content",
	}, func(_ context.Context, _ *mcp.CallToolRequest, in Input) (*mcp.CallToolResult, any, error) {
		if in.Snippet <= 0 {
			return nil, nil, fmt.Errorf("snippet is required")
		}
		client, err := f.Client()
		if err != nil {
			return nil, nil, err
		}
		if in.Raw {
			content, _, err := client.Snippets.SnippetContent(in.Snippet)
			if err != nil {
				return nil, nil, fmt.Errorf("getting snippet content: %w", err)
			}
			return plainResult(string(content)), nil, nil
		}
		snippet, _, err := client.Snippets.GetSnippet(in.Snippet)
		if err != nil {
			return nil, nil, fmt.Errorf("getting snippet: %w", err)
		}
		return textResult(snippet)
	})
}

func registerSnippetCreate(server *mcp.Server, f *cmdutil.Factory) {
	type Input struct {
		Title      string `json:"title"                jsonschema:"snippet title"`
		Filename   string `json:"filename"             jsonschema:"filename for the snippet content (e.g. main.go)"`
		Content    string `json:"content"              jsonschema:"snippet file content"`
		Visibility string `json:"visibility,omitempty" jsonschema:"visibility: public, internal, or private (default: private)"`
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "snippet_create",
		Description: "Create a new personal snippet",
	}, func(_ context.Context, _ *mcp.CallToolRequest, in Input) (*mcp.CallToolResult, any, error) {
		if in.Title == "" {
			return nil, nil, fmt.Errorf("title is required")
		}
		if in.Filename == "" {
			return nil, nil, fmt.Errorf("filename is required")
		}
		if in.Content == "" {
			return nil, nil, fmt.Errorf("content is required")
		}
		client, err := f.Client()
		if err != nil {
			return nil, nil, err
		}
		vis := gitlab.PrivateVisibility
		switch in.Visibility {
		case "public":
			vis = gitlab.PublicVisibility
		case "internal":
			vis = gitlab.InternalVisibility
		}
		snippet, _, err := client.Snippets.CreateSnippet(&gitlab.CreateSnippetOptions{
			Title:      &in.Title,
			Visibility: &vis,
			Files: &[]*gitlab.CreateSnippetFileOptions{
				{FilePath: &in.Filename, Content: &in.Content},
			},
		})
		if err != nil {
			return nil, nil, fmt.Errorf("creating snippet: %w", err)
		}
		return plainResult(fmt.Sprintf("Created snippet #%d\n%s", snippet.ID, snippet.WebURL)), nil, nil
	})
}

func registerSnippetDelete(server *mcp.Server, f *cmdutil.Factory) {
	type Input struct {
		Snippet int64 `json:"snippet" jsonschema:"snippet ID"`
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "snippet_delete",
		Description: "Delete a snippet",
	}, func(_ context.Context, _ *mcp.CallToolRequest, in Input) (*mcp.CallToolResult, any, error) {
		if in.Snippet <= 0 {
			return nil, nil, fmt.Errorf("snippet is required")
		}
		client, err := f.Client()
		if err != nil {
			return nil, nil, err
		}
		_, err = client.Snippets.DeleteSnippet(in.Snippet)
		if err != nil {
			return nil, nil, fmt.Errorf("deleting snippet: %w", err)
		}
		return plainResult(fmt.Sprintf("Deleted snippet #%d", in.Snippet)), nil, nil
	})
}
