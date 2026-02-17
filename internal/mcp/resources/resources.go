package resources

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/url"
	"strings"

	"github.com/PhilipKram/gitlab-cli/internal/api"
	"github.com/PhilipKram/gitlab-cli/internal/cmdutil"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// RegisterResources registers all GitLab resource templates on the server.
func RegisterResources(server *mcp.Server, f *cmdutil.Factory) {
	registerReadmeResource(server, f)
	registerCIConfigResource(server, f)
	registerMRDiffResource(server, f)
	registerPipelineJobLogResource(server, f)
}

func registerReadmeResource(server *mcp.Server, f *cmdutil.Factory) {
	server.AddResourceTemplate(&mcp.ResourceTemplate{
		Name:        "readme",
		URITemplate: "gitlab:///{repo}/README.md",
		Description: "Read the README.md file from a GitLab repository",
		MIMEType:    "text/markdown",
	}, func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		repo, err := extractRepoFromURI(req.Params.URI)
		if err != nil {
			return nil, err
		}

		client, project, err := resolveClientAndProject(f, repo)
		if err != nil {
			return nil, err
		}

		// Get the README file from the repository
		file, _, err := client.RepositoryFiles.GetFile(
			project,
			"README.md",
			&gitlab.GetFileOptions{Ref: gitlab.Ptr("HEAD")},
		)
		if err != nil {
			return nil, mcp.ResourceNotFoundError(req.Params.URI)
		}

		// Decode base64-encoded content
		decoded, err := base64.StdEncoding.DecodeString(file.Content)
		if err != nil {
			return nil, fmt.Errorf("decoding README content: %w", err)
		}
		content := string(decoded)

		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{
				{
					URI:      req.Params.URI,
					MIMEType: "text/markdown",
					Text:     content,
				},
			},
		}, nil
	})
}

func registerCIConfigResource(server *mcp.Server, f *cmdutil.Factory) {
	server.AddResourceTemplate(&mcp.ResourceTemplate{
		Name:        "ci-config",
		URITemplate: "gitlab:///{repo}/.gitlab-ci.yml",
		Description: "Read the .gitlab-ci.yml configuration file from a GitLab repository",
		MIMEType:    "text/yaml",
	}, func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		repo, err := extractRepoFromURI(req.Params.URI)
		if err != nil {
			return nil, err
		}

		client, project, err := resolveClientAndProject(f, repo)
		if err != nil {
			return nil, err
		}

		// Get the .gitlab-ci.yml file from the repository
		file, _, err := client.RepositoryFiles.GetFile(
			project,
			".gitlab-ci.yml",
			&gitlab.GetFileOptions{Ref: gitlab.Ptr("HEAD")},
		)
		if err != nil {
			return nil, mcp.ResourceNotFoundError(req.Params.URI)
		}

		// Decode base64-encoded content
		decoded, err := base64.StdEncoding.DecodeString(file.Content)
		if err != nil {
			return nil, fmt.Errorf("decoding CI config content: %w", err)
		}
		content := string(decoded)

		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{
				{
					URI:      req.Params.URI,
					MIMEType: "text/yaml",
					Text:     content,
				},
			},
		}, nil
	})
}

func registerMRDiffResource(server *mcp.Server, f *cmdutil.Factory) {
	server.AddResourceTemplate(&mcp.ResourceTemplate{
		Name:        "mr-diff",
		URITemplate: "gitlab:///{repo}/mr/{mr}/diff",
		Description: "Read the diff of a merge request",
		MIMEType:    "text/plain",
	}, func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		repo, mr, err := extractRepoAndMRFromURI(req.Params.URI)
		if err != nil {
			return nil, err
		}

		client, project, err := resolveClientAndProject(f, repo)
		if err != nil {
			return nil, err
		}

		// Get the merge request diffs
		diffs, _, err := client.MergeRequests.ListMergeRequestDiffs(project, mr, nil)
		if err != nil {
			return nil, mcp.ResourceNotFoundError(req.Params.URI)
		}

		// Format the diff similar to mr_diff tool
		var sb strings.Builder
		for _, d := range diffs {
			fmt.Fprintf(&sb, "--- a/%s\n+++ b/%s\n%s\n", d.OldPath, d.NewPath, d.Diff)
		}

		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{
				{
					URI:      req.Params.URI,
					MIMEType: "text/plain",
					Text:     sb.String(),
				},
			},
		}, nil
	})
}

func registerPipelineJobLogResource(server *mcp.Server, f *cmdutil.Factory) {
	server.AddResourceTemplate(&mcp.ResourceTemplate{
		Name:        "pipeline-job-log",
		URITemplate: "gitlab:///{repo}/pipeline/{pipeline}/job/{job}/log",
		Description: "Read the log output of a pipeline job",
		MIMEType:    "text/plain",
	}, func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		repo, job, err := extractRepoAndJobFromURI(req.Params.URI)
		if err != nil {
			return nil, err
		}

		client, project, err := resolveClientAndProject(f, repo)
		if err != nil {
			return nil, err
		}

		// Get the job log
		reader, _, err := client.Jobs.GetTraceFile(project, job)
		if err != nil {
			return nil, mcp.ResourceNotFoundError(req.Params.URI)
		}

		// Read the log content
		log, err := readLog(reader)
		if err != nil {
			return nil, fmt.Errorf("reading job log: %w", err)
		}

		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{
				{
					URI:      req.Params.URI,
					MIMEType: "text/plain",
					Text:     log,
				},
			},
		}, nil
	})
}

// extractRepoFromURI extracts the repository path from a gitlab:// URI.
// URI format: gitlab:///{repo}/path
// Returns the repo portion (e.g., "owner/project")
func extractRepoFromURI(uri string) (string, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return "", fmt.Errorf("parsing URI: %w", err)
	}

	if u.Scheme != "gitlab" {
		return "", fmt.Errorf("invalid URI scheme: expected 'gitlab', got '%s'", u.Scheme)
	}

	// Path format: /{repo}/file
	// Split and extract repo (first non-empty segment)
	parts := strings.Split(strings.TrimPrefix(u.Path, "/"), "/")
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid URI path: expected /{repo}/file format")
	}

	// Repo is typically owner/project, so join first two parts
	repo := parts[0] + "/" + parts[1]
	return repo, nil
}

// extractRepoAndMRFromURI extracts the repository path and MR IID from a gitlab:// URI.
// URI format: gitlab:///{repo}/mr/{mr}/...
// Returns the repo portion (e.g., "owner/project") and the MR IID
func extractRepoAndMRFromURI(uri string) (string, int64, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return "", 0, fmt.Errorf("parsing URI: %w", err)
	}

	if u.Scheme != "gitlab" {
		return "", 0, fmt.Errorf("invalid URI scheme: expected 'gitlab', got '%s'", u.Scheme)
	}

	// Path format: /{owner}/{project}/mr/{mr}/...
	// Split and extract repo and MR IID
	parts := strings.Split(strings.TrimPrefix(u.Path, "/"), "/")
	if len(parts) < 4 {
		return "", 0, fmt.Errorf("invalid URI path: expected /{repo}/mr/{mr} format")
	}

	// Repo is owner/project
	repo := parts[0] + "/" + parts[1]

	// Check that "mr" is in the right position
	if parts[2] != "mr" {
		return "", 0, fmt.Errorf("invalid URI path: expected /{repo}/mr/{mr} format")
	}

	// Parse MR IID
	var mr int64
	_, err = fmt.Sscanf(parts[3], "%d", &mr)
	if err != nil {
		return "", 0, fmt.Errorf("invalid MR IID in URI: %w", err)
	}

	return repo, mr, nil
}

// extractRepoAndJobFromURI extracts the repository path and job ID from a gitlab:// URI.
// URI format: gitlab:///{repo}/pipeline/{pipeline}/job/{job}/log
// Returns the repo portion (e.g., "owner/project") and the job ID
func extractRepoAndJobFromURI(uri string) (string, int64, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return "", 0, fmt.Errorf("parsing URI: %w", err)
	}

	if u.Scheme != "gitlab" {
		return "", 0, fmt.Errorf("invalid URI scheme: expected 'gitlab', got '%s'", u.Scheme)
	}

	// Path format: /{owner}/{project}/pipeline/{pipeline}/job/{job}/log
	// Split and extract repo, pipeline, and job ID
	parts := strings.Split(strings.TrimPrefix(u.Path, "/"), "/")
	if len(parts) < 7 {
		return "", 0, fmt.Errorf("invalid URI path: expected /{repo}/pipeline/{pipeline}/job/{job}/log format")
	}

	// Repo is owner/project
	repo := parts[0] + "/" + parts[1]

	// Check that "pipeline" and "job" are in the right positions
	if parts[2] != "pipeline" || parts[4] != "job" {
		return "", 0, fmt.Errorf("invalid URI path: expected /{repo}/pipeline/{pipeline}/job/{job}/log format")
	}

	// Parse job ID (we only need the job ID for GetTraceFile)
	var job int64
	_, err = fmt.Sscanf(parts[5], "%d", &job)
	if err != nil {
		return "", 0, fmt.Errorf("invalid job ID in URI: %w", err)
	}

	return repo, job, nil
}

// readLog reads log content from a reader with a 1 MiB size limit.
func readLog(r io.Reader) (string, error) {
	const maxLogBytes = 1024 * 1024 // 1 MiB
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

// resolveClientAndProject returns an authenticated API client and the project path.
// repo may be empty (falls back to git remote), OWNER/REPO, or HOST/OWNER/REPO.
func resolveClientAndProject(f *cmdutil.Factory, repo string) (*api.Client, string, error) {
	if repo == "" {
		client, err := f.Client()
		if err != nil {
			return nil, "", err
		}
		project, err := f.FullProjectPath()
		if err != nil {
			return nil, "", fmt.Errorf("%w\nProvide the repo in the URI as gitlab:///{OWNER/REPO}/file", err)
		}
		return client, project, nil
	}

	// Note: This is simplified - the actual implementation in tools/helpers.go
	// supports HOST/OWNER/REPO format. For resources, we expect OWNER/REPO
	// in the URI template.
	client, err := f.Client()
	if err != nil {
		return nil, "", err
	}
	return client, repo, nil
}
