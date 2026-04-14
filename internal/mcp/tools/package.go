package tools

import (
	"context"
	"fmt"

	"github.com/PhilipKram/gitlab-cli/internal/cmdutil"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// RegisterPackageTools registers all package registry tools on the server.
func RegisterPackageTools(server *mcp.Server, f *cmdutil.Factory) {
	registerPackageList(server, f)
	registerPackageView(server, f)
	registerPackageDelete(server, f)
}

func registerPackageList(server *mcp.Server, f *cmdutil.Factory) {
	type Input struct {
		Repo        string `json:"repo,omitempty"         jsonschema:"repository in OWNER/REPO or HOST/OWNER/REPO format"`
		Group       string `json:"group,omitempty"        jsonschema:"list packages for a specific group"`
		PackageType string `json:"package_type,omitempty" jsonschema:"filter by package type: npm, maven, pypi, nuget, conan, composer, helm, generic"`
		Limit       int64  `json:"limit,omitempty"        jsonschema:"maximum number of results (default 30)"`
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "package_list",
		Description: "List packages in a project or group registry",
	}, func(_ context.Context, _ *mcp.CallToolRequest, in Input) (*mcp.CallToolResult, any, error) {
		if in.Group != "" {
			client, err := f.Client()
			if err != nil {
				return nil, nil, err
			}
			opts := &gitlab.ListGroupPackagesOptions{
				ListOptions: gitlab.ListOptions{PerPage: clampPerPage(in.Limit)},
			}
			if in.PackageType != "" {
				opts.PackageType = &in.PackageType
			}
			packages, _, err := client.Packages.ListGroupPackages(in.Group, opts)
			if err != nil {
				return nil, nil, fmt.Errorf("listing group packages: %w", err)
			}
			return textResult(packages)
		}

		client, project, err := resolveClientAndProject(f, in.Repo)
		if err != nil {
			return nil, nil, err
		}
		opts := &gitlab.ListProjectPackagesOptions{
			ListOptions: gitlab.ListOptions{PerPage: clampPerPage(in.Limit)},
		}
		if in.PackageType != "" {
			opts.PackageType = &in.PackageType
		}
		packages, _, err := client.Packages.ListProjectPackages(project, opts)
		if err != nil {
			return nil, nil, fmt.Errorf("listing packages: %w", err)
		}
		return textResult(packages)
	})
}

func registerPackageView(server *mcp.Server, f *cmdutil.Factory) {
	type Input struct {
		Name  string `json:"name"            jsonschema:"package name"`
		Repo  string `json:"repo,omitempty"  jsonschema:"repository in OWNER/REPO or HOST/OWNER/REPO format"`
		Group string `json:"group,omitempty" jsonschema:"view package in a specific group"`
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "package_view",
		Description: "View package details and versions",
	}, func(_ context.Context, _ *mcp.CallToolRequest, in Input) (*mcp.CallToolResult, any, error) {
		if err := requireString(in.Name, "name"); err != nil {
			return nil, nil, err
		}

		if in.Group != "" {
			client, err := f.Client()
			if err != nil {
				return nil, nil, err
			}
			opts := &gitlab.ListGroupPackagesOptions{
				ListOptions: gitlab.ListOptions{PerPage: 100},
			}
			packages, _, err := client.Packages.ListGroupPackages(in.Group, opts)
			if err != nil {
				return nil, nil, fmt.Errorf("listing group packages: %w", err)
			}
			var matching []*gitlab.GroupPackage
			for _, pkg := range packages {
				if pkg.Name == in.Name {
					matching = append(matching, pkg)
				}
			}
			if len(matching) == 0 {
				return nil, nil, fmt.Errorf("package not found: %s", in.Name)
			}
			return textResult(matching)
		}

		client, project, err := resolveClientAndProject(f, in.Repo)
		if err != nil {
			return nil, nil, err
		}
		opts := &gitlab.ListProjectPackagesOptions{
			ListOptions: gitlab.ListOptions{PerPage: 100},
		}
		packages, _, err := client.Packages.ListProjectPackages(project, opts)
		if err != nil {
			return nil, nil, fmt.Errorf("listing packages: %w", err)
		}
		var matching []*gitlab.Package
		for _, pkg := range packages {
			if pkg.Name == in.Name {
				matching = append(matching, pkg)
			}
		}
		if len(matching) == 0 {
			return nil, nil, fmt.Errorf("package not found: %s", in.Name)
		}
		return textResult(matching)
	})
}

func registerPackageDelete(server *mcp.Server, f *cmdutil.Factory) {
	type Input struct {
		Name    string `json:"name"               jsonschema:"package name to delete"`
		Version string `json:"version,omitempty"   jsonschema:"delete a specific package version"`
		Repo    string `json:"repo,omitempty"      jsonschema:"repository in OWNER/REPO or HOST/OWNER/REPO format"`
		Group   string `json:"group,omitempty"     jsonschema:"delete package from a specific group"`
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "package_delete",
		Description: "Delete a package or specific version",
	}, func(_ context.Context, _ *mcp.CallToolRequest, in Input) (*mcp.CallToolResult, any, error) {
		if err := requireString(in.Name, "name"); err != nil {
			return nil, nil, err
		}

		if in.Group != "" {
			client, err := f.Client()
			if err != nil {
				return nil, nil, err
			}
			opts := &gitlab.ListGroupPackagesOptions{
				ListOptions: gitlab.ListOptions{PerPage: 100},
			}
			packages, _, err := client.Packages.ListGroupPackages(in.Group, opts)
			if err != nil {
				return nil, nil, fmt.Errorf("listing group packages: %w", err)
			}
			deleted := 0
			for _, pkg := range packages {
				if pkg.Name == in.Name && (in.Version == "" || pkg.Version == in.Version) {
					projectID := fmt.Sprintf("%d", pkg.ProjectID)
					_, err := client.Packages.DeleteProjectPackage(projectID, pkg.ID)
					if err != nil {
						return nil, nil, fmt.Errorf("deleting package %s (version %s): %w", pkg.Name, pkg.Version, err)
					}
					deleted++
				}
			}
			if deleted == 0 {
				return nil, nil, fmt.Errorf("package not found: %s", in.Name)
			}
			return plainResult(fmt.Sprintf("Deleted %d package(s) matching %q", deleted, in.Name)), nil, nil
		}

		client, project, err := resolveClientAndProject(f, in.Repo)
		if err != nil {
			return nil, nil, err
		}
		opts := &gitlab.ListProjectPackagesOptions{
			ListOptions: gitlab.ListOptions{PerPage: 100},
		}
		packages, _, err := client.Packages.ListProjectPackages(project, opts)
		if err != nil {
			return nil, nil, fmt.Errorf("listing packages: %w", err)
		}
		deleted := 0
		for _, pkg := range packages {
			if pkg.Name == in.Name && (in.Version == "" || pkg.Version == in.Version) {
				_, err := client.Packages.DeleteProjectPackage(project, pkg.ID)
				if err != nil {
					return nil, nil, fmt.Errorf("deleting package %s (version %s): %w", pkg.Name, pkg.Version, err)
				}
				deleted++
			}
		}
		if deleted == 0 {
			return nil, nil, fmt.Errorf("package not found: %s", in.Name)
		}
		return plainResult(fmt.Sprintf("Deleted %d package(s) matching %q", deleted, in.Name)), nil, nil
	})
}
