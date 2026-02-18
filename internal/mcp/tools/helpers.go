package tools

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/PhilipKram/gitlab-cli/internal/api"
	"github.com/PhilipKram/gitlab-cli/internal/cmdutil"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	// maxPerPage is the maximum number of results any list tool will request.
	// The GitLab API caps per_page at 100 server-side; we enforce it here so
	// AI-supplied values never silently exceed intent.
	maxPerPage = int64(100)

	// maxLogBytes is the maximum job log size read into memory at once.
	maxLogBytes = int64(1 << 20) // 1 MiB
)

// clampPerPage returns perPage clamped to [1, maxPerPage], defaulting to 30.
func clampPerPage(perPage int64) int64 {
	if perPage <= 0 {
		return 30
	}
	if perPage > maxPerPage {
		return maxPerPage
	}
	return perPage
}

// readLog reads at most maxLogBytes from r, appending a truncation notice if needed.
func readLog(r io.Reader) (string, error) {
	limited := io.LimitReader(r, maxLogBytes+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return "", err
	}
	if int64(len(data)) > maxLogBytes {
		data = data[:maxLogBytes]
		return string(data) + "\n[log truncated at 1 MiB]", nil
	}
	return string(data), nil
}

// resolveClientAndProject returns an authenticated API client and the OWNER/REPO
// path. repo may be empty (falls back to git remote), OWNER/REPO, or HOST/OWNER/REPO.
func resolveClientAndProject(f *cmdutil.Factory, repo string) (*api.Client, string, error) {
	if repo == "" {
		client, err := f.Client()
		if err != nil {
			return nil, "", err
		}
		project, err := f.FullProjectPath()
		if err != nil {
			return nil, "", fmt.Errorf("%w\nProvide the 'repo' parameter as HOST/OWNER/REPO or OWNER/REPO", err)
		}
		return client, project, nil
	}

	parts := strings.SplitN(repo, "/", 3)
	if len(parts) == 3 {
		host := parts[0]
		project := parts[1] + "/" + parts[2]
		client, err := api.NewClient(host)
		if err != nil {
			return nil, "", err
		}
		return client, project, nil
	}

	// OWNER/REPO — use factory client
	client, err := f.Client()
	if err != nil {
		return nil, "", err
	}
	return client, repo, nil
}

// textResult marshals v as indented JSON and wraps it in a CallToolResult.
func textResult(v any) (*mcp.CallToolResult, any, error) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, nil, fmt.Errorf("marshaling result: %w", err)
	}
	return plainResult(string(data)), nil, nil
}

// plainResult wraps a plain string in a CallToolResult.
func plainResult(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: text}},
	}
}
