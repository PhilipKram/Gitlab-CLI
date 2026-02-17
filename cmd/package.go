package cmd

import (
	"fmt"

	"github.com/PhilipKram/gitlab-cli/internal/api"
	"github.com/PhilipKram/gitlab-cli/internal/cmdutil"
	"github.com/PhilipKram/gitlab-cli/internal/errors"
	"github.com/spf13/cobra"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// NewPackageCmd creates the package command group.
func NewPackageCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "package <command>",
		Short: "Manage package registries",
		Long:  "List, view, delete, and download packages from GitLab package registries. Supports npm, Maven, PyPI, NuGet, Conan, Composer, Helm, and generic package types.",
	}

	cmd.AddCommand(newPackageListCmd(f))
	cmd.AddCommand(newPackageViewCmd(f))
	cmd.AddCommand(newPackageDeleteCmd(f))
	cmd.AddCommand(newPackageDownloadCmd(f))

	return cmd
}

// newPackageListCmd creates the package list command.
func newPackageListCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		limit      int
		format     string
		jsonFlag   bool
		packageType string
		groupPath  string
	)

	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List packages",
		Aliases: []string{"ls"},
		Long:    "List packages in a project or group package registry. When run in a project directory, lists packages for that project. When --group is specified, lists packages across all projects in the group.",
		Example: `  $ glab package list
  $ glab package list --type npm
  $ glab package list --group mygroup
  $ glab package list --format json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := f.Client()
			if err != nil {
				return err
			}

			// Determine if we're in group or project context
			if groupPath != "" {
				// Group context - use group-level API
				groupOpts := &gitlab.ListGroupPackagesOptions{
					ListOptions: gitlab.ListOptions{PerPage: int64(limit)},
				}

				// Apply package type filter if specified
				if packageType != "" {
					groupOpts.PackageType = &packageType
				}

				groupPackages, resp, err := client.Packages.ListGroupPackages(groupPath, groupOpts)
				if err != nil {
					statusCode := 0
					if resp != nil {
						statusCode = resp.StatusCode
					}
					url := api.APIURL(client.Host()) + "/groups/" + groupPath + "/packages"
					return errors.NewAPIError("GET", url, statusCode, "Failed to list group packages", err)
				}

				if len(groupPackages) == 0 {
					_, _ = fmt.Fprintln(f.IOStreams.ErrOut, "No packages found")
					return nil
				}

				return f.FormatAndPrint(groupPackages, format, jsonFlag)
			} else {
				// Project context - use project-level API
				project, err := f.FullProjectPath()
				if err != nil {
					return fmt.Errorf("must be run within a GitLab project or specify --group")
				}

				opts := &gitlab.ListProjectPackagesOptions{
					ListOptions: gitlab.ListOptions{PerPage: int64(limit)},
				}

				// Apply package type filter if specified
				if packageType != "" {
					opts.PackageType = &packageType
				}

				packages, resp, err := client.Packages.ListProjectPackages(project, opts)
				if err != nil {
					statusCode := 0
					if resp != nil {
						statusCode = resp.StatusCode
					}
					url := api.APIURL(client.Host()) + "/projects/" + project + "/packages"
					return errors.NewAPIError("GET", url, statusCode, "Failed to list packages", err)
				}

				if len(packages) == 0 {
					_, _ = fmt.Fprintln(f.IOStreams.ErrOut, "No packages found")
					return nil
				}

				return f.FormatAndPrint(packages, format, jsonFlag)
			}
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "L", 30, "Maximum number of results")
	cmd.Flags().StringVarP(&format, "format", "F", "table", "Output format: json, table, or plain")
	cmd.Flags().BoolVar(&jsonFlag, "json", false, "Output as JSON (deprecated: use --format=json)")
	cmd.Flags().StringVarP(&packageType, "type", "t", "", "Filter by package type: npm, maven, pypi, nuget, conan, composer, helm, generic")
	cmd.Flags().StringVarP(&groupPath, "group", "g", "", "List packages for a specific group")

	return cmd
}

// newPackageViewCmd creates the package view command.
func newPackageViewCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		format    string
		jsonFlag  bool
		groupPath string
	)

	cmd := &cobra.Command{
		Use:     "view <package-name>",
		Short:   "View package details",
		Long:    "View detailed information about a package including all published versions. Works with both project and group package registries.",
		Example: `  $ glab package view my-package
  $ glab package view @scope/package --format json
  $ glab package view my-package --group mygroup`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := f.Client()
			if err != nil {
				return err
			}

			packageName := args[0]

			// Determine if we're in group or project context
			if groupPath != "" {
				// Group context - use group-level API
				groupOpts := &gitlab.ListGroupPackagesOptions{
					ListOptions: gitlab.ListOptions{PerPage: 100},
				}

				groupPackages, resp, err := client.Packages.ListGroupPackages(groupPath, groupOpts)
				if err != nil {
					statusCode := 0
					if resp != nil {
						statusCode = resp.StatusCode
					}
					url := api.APIURL(client.Host()) + "/groups/" + groupPath + "/packages"
					return errors.NewAPIError("GET", url, statusCode, "Failed to list group packages", err)
				}

				// Find all versions of the package
				var matchingPackages []*gitlab.GroupPackage
				for _, pkg := range groupPackages {
					if pkg.Name == packageName {
						matchingPackages = append(matchingPackages, pkg)
					}
				}

				if len(matchingPackages) == 0 {
					return fmt.Errorf("package not found: %s", packageName)
				}

				// Validate format flag
				if format != "" && format != "table" {
					return f.FormatAndPrint(matchingPackages, format, jsonFlag)
				}

				// Default custom display for group packages
				out := f.IOStreams.Out
				firstPkg := matchingPackages[0]
				_, _ = fmt.Fprintf(out, "Name:         %s\n", firstPkg.Name)
				_, _ = fmt.Fprintf(out, "Package Type: %s\n", firstPkg.PackageType)
				_, _ = fmt.Fprintf(out, "Group:        %s\n", groupPath)
				_, _ = fmt.Fprintf(out, "\nVersions (%d):\n", len(matchingPackages))

				for _, pkg := range matchingPackages {
					_, _ = fmt.Fprintf(out, "  %s (created %s)\n", pkg.Version, timeAgo(pkg.CreatedAt))
				}

				return nil
			} else {
				// Project context - use project-level API
				project, err := f.FullProjectPath()
				if err != nil {
					return fmt.Errorf("must be run within a GitLab project or specify --group")
				}

				opts := &gitlab.ListProjectPackagesOptions{
					ListOptions: gitlab.ListOptions{PerPage: 100},
				}

				packages, resp, err := client.Packages.ListProjectPackages(project, opts)
				if err != nil {
					statusCode := 0
					if resp != nil {
						statusCode = resp.StatusCode
					}
					url := api.APIURL(client.Host()) + "/projects/" + project + "/packages"
					return errors.NewAPIError("GET", url, statusCode, "Failed to list packages", err)
				}

				// Find all versions of the package
				var matchingPackages []*gitlab.Package
				for _, pkg := range packages {
					if pkg.Name == packageName {
						matchingPackages = append(matchingPackages, pkg)
					}
				}

				if len(matchingPackages) == 0 {
					return fmt.Errorf("package not found: %s", packageName)
				}

				// Validate format flag
				if format != "" && format != "table" {
					return f.FormatAndPrint(matchingPackages, format, jsonFlag)
				}

				// Default custom display for project packages
				out := f.IOStreams.Out
				firstPkg := matchingPackages[0]
				_, _ = fmt.Fprintf(out, "Name:         %s\n", firstPkg.Name)
				_, _ = fmt.Fprintf(out, "Package Type: %s\n", firstPkg.PackageType)
				_, _ = fmt.Fprintf(out, "Project:      %s\n", project)
				_, _ = fmt.Fprintf(out, "\nVersions (%d):\n", len(matchingPackages))

				for _, pkg := range matchingPackages {
					_, _ = fmt.Fprintf(out, "  %s (created %s)\n", pkg.Version, timeAgo(pkg.CreatedAt))
				}

				return nil
			}
		},
	}

	cmd.Flags().StringVarP(&format, "format", "F", "table", "Output format: json, table, or plain")
	cmd.Flags().BoolVar(&jsonFlag, "json", false, "Output as JSON (deprecated: use --format=json)")
	cmd.Flags().StringVarP(&groupPath, "group", "g", "", "View package in a specific group")

	return cmd
}

// newPackageDeleteCmd creates the package delete command.
func newPackageDeleteCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		version   string
		groupPath string
	)

	cmd := &cobra.Command{
		Use:     "delete <package-name>",
		Short:   "Delete a package",
		Long:    "Delete a package or a specific package version from the registry. Works with both project and group package registries.",
		Example: `  $ glab package delete my-package
  $ glab package delete my-package --version 1.0.0
  $ glab package delete my-package --group mygroup`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := f.Client()
			if err != nil {
				return err
			}

			packageName := args[0]

			// Determine if we're in group or project context
			if groupPath != "" {
				// Group context - use group-level API to list packages
				groupOpts := &gitlab.ListGroupPackagesOptions{
					ListOptions: gitlab.ListOptions{PerPage: 100},
				}

				groupPackages, resp, err := client.Packages.ListGroupPackages(groupPath, groupOpts)
				if err != nil {
					statusCode := 0
					if resp != nil {
						statusCode = resp.StatusCode
					}
					url := api.APIURL(client.Host()) + "/groups/" + groupPath + "/packages"
					return errors.NewAPIError("GET", url, statusCode, "Failed to list group packages", err)
				}

				// Find matching package(s)
				var matchingPackages []*gitlab.GroupPackage
				for _, pkg := range groupPackages {
					if pkg.Name == packageName {
						if version == "" || pkg.Version == version {
							matchingPackages = append(matchingPackages, pkg)
						}
					}
				}

				if len(matchingPackages) == 0 {
					if version != "" {
						return fmt.Errorf("package not found: %s (version %s)", packageName, version)
					}
					return fmt.Errorf("package not found: %s", packageName)
				}

				// Delete matching package(s) - use project ID from group package
				for _, pkg := range matchingPackages {
					projectID := fmt.Sprintf("%d", pkg.ProjectID)
					resp, err := client.Packages.DeleteProjectPackage(projectID, pkg.ID)
					if err != nil {
						statusCode := 0
						if resp != nil {
							statusCode = resp.StatusCode
						}
						url := api.APIURL(client.Host()) + fmt.Sprintf("/projects/%s/packages/%d", projectID, pkg.ID)
						return errors.NewAPIError("DELETE", url, statusCode, "Failed to delete package", err)
					}

					if version != "" {
						_, _ = fmt.Fprintf(f.IOStreams.Out, "Deleted package %s (version %s)\n", packageName, version)
					} else {
						_, _ = fmt.Fprintf(f.IOStreams.Out, "Deleted package %s (version %s)\n", pkg.Name, pkg.Version)
					}
				}

				return nil
			} else {
				// Project context - use project-level API
				project, err := f.FullProjectPath()
				if err != nil {
					return fmt.Errorf("must be run within a GitLab project or specify --group")
				}

				opts := &gitlab.ListProjectPackagesOptions{
					ListOptions: gitlab.ListOptions{PerPage: 100},
				}

				packages, resp, err := client.Packages.ListProjectPackages(project, opts)
				if err != nil {
					statusCode := 0
					if resp != nil {
						statusCode = resp.StatusCode
					}
					url := api.APIURL(client.Host()) + "/projects/" + project + "/packages"
					return errors.NewAPIError("GET", url, statusCode, "Failed to list packages", err)
				}

				// Find matching package(s)
				var matchingPackages []*gitlab.Package
				for _, pkg := range packages {
					if pkg.Name == packageName {
						if version == "" || pkg.Version == version {
							matchingPackages = append(matchingPackages, pkg)
						}
					}
				}

				if len(matchingPackages) == 0 {
					if version != "" {
						return fmt.Errorf("package not found: %s (version %s)", packageName, version)
					}
					return fmt.Errorf("package not found: %s", packageName)
				}

				// Delete matching package(s)
				for _, pkg := range matchingPackages {
					resp, err := client.Packages.DeleteProjectPackage(project, pkg.ID)
					if err != nil {
						statusCode := 0
						if resp != nil {
							statusCode = resp.StatusCode
						}
						url := api.APIURL(client.Host()) + fmt.Sprintf("/projects/%s/packages/%d", project, pkg.ID)
						return errors.NewAPIError("DELETE", url, statusCode, "Failed to delete package", err)
					}

					if version != "" {
						_, _ = fmt.Fprintf(f.IOStreams.Out, "Deleted package %s (version %s)\n", packageName, version)
					} else {
						_, _ = fmt.Fprintf(f.IOStreams.Out, "Deleted package %s (version %s)\n", pkg.Name, pkg.Version)
					}
				}

				return nil
			}
		},
	}

	cmd.Flags().StringVar(&version, "version", "", "Delete a specific package version")
	cmd.Flags().StringVarP(&groupPath, "group", "g", "", "Delete package from a specific group")

	return cmd
}

// newPackageDownloadCmd creates the package download command.
func newPackageDownloadCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		version   string
		output    string
		groupPath string
	)

	cmd := &cobra.Command{
		Use:   "download <package-name>",
		Short: "Download a package",
		Long:  "List downloadable package files or download a specific package version. Works with both project and group package registries.",
		Example: `  $ glab package download my-package
  $ glab package download my-package --version 1.0.0
  $ glab package download my-package --version 1.0.0 --output ./downloads
  $ glab package download my-package --group mygroup`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := f.Client()
			if err != nil {
				return err
			}

			packageName := args[0]
			out := f.IOStreams.Out

			// Determine if we're in group or project context
			if groupPath != "" {
				// Group context - use group-level API to list packages
				groupOpts := &gitlab.ListGroupPackagesOptions{
					ListOptions: gitlab.ListOptions{PerPage: 100},
				}

				groupPackages, resp, err := client.Packages.ListGroupPackages(groupPath, groupOpts)
				if err != nil {
					statusCode := 0
					if resp != nil {
						statusCode = resp.StatusCode
					}
					url := api.APIURL(client.Host()) + "/groups/" + groupPath + "/packages"
					return errors.NewAPIError("GET", url, statusCode, "Failed to list group packages", err)
				}

				// Find matching package(s)
				var matchingPackages []*gitlab.GroupPackage
				for _, pkg := range groupPackages {
					if pkg.Name == packageName {
						if version != "" && pkg.Version == version {
							matchingPackages = append(matchingPackages, pkg)
							break
						} else if version == "" {
							matchingPackages = append(matchingPackages, pkg)
						}
					}
				}

				if len(matchingPackages) == 0 {
					if version != "" {
						return fmt.Errorf("package not found: %s (version %s)", packageName, version)
					}
					return fmt.Errorf("package not found: %s", packageName)
				}

				// Get package files for each matching package
				for _, pkg := range matchingPackages {
					projectID := fmt.Sprintf("%d", pkg.ProjectID)
					packageFiles, resp, err := client.Packages.ListPackageFiles(projectID, int64(pkg.ID), &gitlab.ListPackageFilesOptions{})
					if err != nil {
						statusCode := 0
						if resp != nil {
							statusCode = resp.StatusCode
						}
						url := api.APIURL(client.Host()) + fmt.Sprintf("/projects/%s/packages/%d/package_files", projectID, pkg.ID)
						return errors.NewAPIError("GET", url, statusCode, "Failed to get package files", err)
					}

					_, _ = fmt.Fprintf(out, "Package: %s\n", pkg.Name)
					_, _ = fmt.Fprintf(out, "Version: %s\n", pkg.Version)
					_, _ = fmt.Fprintf(out, "Type:    %s\n\n", pkg.PackageType)

					if len(packageFiles) > 0 {
						_, _ = fmt.Fprintln(out, "Package files:")
						for _, file := range packageFiles {
							_, _ = fmt.Fprintf(out, "  %s (%s)\n", file.FileName, byteCountSI(int64(file.Size)))
						}
					} else {
						_, _ = fmt.Fprintln(out, "No package files found")
					}

					if output != "" {
						_, _ = fmt.Fprintf(out, "\nNote: Automatic download to %s is not yet implemented.\n", output)
						_, _ = fmt.Fprintln(out, "Please use the GitLab package registry API or CLI tools for your package type.")
					}

					if len(matchingPackages) > 1 {
						_, _ = fmt.Fprintln(out, "")
					}
				}

				return nil
			} else {
				// Project context - use project-level API
				project, err := f.FullProjectPath()
				if err != nil {
					return fmt.Errorf("must be run within a GitLab project or specify --group")
				}

				opts := &gitlab.ListProjectPackagesOptions{
					ListOptions: gitlab.ListOptions{PerPage: 100},
				}

				packages, resp, err := client.Packages.ListProjectPackages(project, opts)
				if err != nil {
					statusCode := 0
					if resp != nil {
						statusCode = resp.StatusCode
					}
					url := api.APIURL(client.Host()) + "/projects/" + project + "/packages"
					return errors.NewAPIError("GET", url, statusCode, "Failed to list packages", err)
				}

				// Find matching package(s)
				var matchingPackages []*gitlab.Package
				for _, pkg := range packages {
					if pkg.Name == packageName {
						if version != "" && pkg.Version == version {
							matchingPackages = append(matchingPackages, pkg)
							break
						} else if version == "" {
							matchingPackages = append(matchingPackages, pkg)
						}
					}
				}

				if len(matchingPackages) == 0 {
					if version != "" {
						return fmt.Errorf("package not found: %s (version %s)", packageName, version)
					}
					return fmt.Errorf("package not found: %s", packageName)
				}

				// Get package files for each matching package
				for _, pkg := range matchingPackages {
					packageFiles, resp, err := client.Packages.ListPackageFiles(project, int64(pkg.ID), &gitlab.ListPackageFilesOptions{})
					if err != nil {
						statusCode := 0
						if resp != nil {
							statusCode = resp.StatusCode
						}
						url := api.APIURL(client.Host()) + fmt.Sprintf("/projects/%s/packages/%d/package_files", project, pkg.ID)
						return errors.NewAPIError("GET", url, statusCode, "Failed to get package files", err)
					}

					_, _ = fmt.Fprintf(out, "Package: %s\n", pkg.Name)
					_, _ = fmt.Fprintf(out, "Version: %s\n", pkg.Version)
					_, _ = fmt.Fprintf(out, "Type:    %s\n\n", pkg.PackageType)

					if len(packageFiles) > 0 {
						_, _ = fmt.Fprintln(out, "Package files:")
						for _, file := range packageFiles {
							_, _ = fmt.Fprintf(out, "  %s (%s)\n", file.FileName, byteCountSI(int64(file.Size)))
						}
					} else {
						_, _ = fmt.Fprintln(out, "No package files found")
					}

					if output != "" {
						_, _ = fmt.Fprintf(out, "\nNote: Automatic download to %s is not yet implemented.\n", output)
						_, _ = fmt.Fprintln(out, "Please use the GitLab package registry API or CLI tools for your package type.")
					}

					if len(matchingPackages) > 1 {
						_, _ = fmt.Fprintln(out, "")
					}
				}

				return nil
			}
		},
	}

	cmd.Flags().StringVar(&version, "version", "", "Download a specific package version")
	cmd.Flags().StringVarP(&output, "output", "o", "", "Output directory for downloaded files")
	cmd.Flags().StringVarP(&groupPath, "group", "g", "", "Download package from a specific group")

	return cmd
}

// byteCountSI converts bytes to human-readable format using SI units.
func byteCountSI(b int64) string {
	const unit = 1000
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "kMGTPE"[exp])
}
