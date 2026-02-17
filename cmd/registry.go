package cmd

import (
	"fmt"
	"strconv"
	"time"

	"github.com/PhilipKram/gitlab-cli/internal/api"
	"github.com/PhilipKram/gitlab-cli/internal/cmdutil"
	"github.com/PhilipKram/gitlab-cli/internal/errors"
	"github.com/spf13/cobra"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// NewRegistryCmd creates the registry command group.
func NewRegistryCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "registry <command>",
		Short: "Manage container registries",
		Long:  "List, view, and manage GitLab container registries and image tags.",
	}

	cmd.AddCommand(newRegistryListCmd(f))
	cmd.AddCommand(newRegistryTagsCmd(f))
	cmd.AddCommand(newRegistryViewCmd(f))
	cmd.AddCommand(newRegistryDeleteCmd(f))

	return cmd
}

func newRegistryListCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		limit    int
		format   string
		jsonFlag bool
		project  string
	)

	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List container repositories",
		Aliases: []string{"ls"},
		Example: `  $ glab registry list
  $ glab registry list --project my-group/my-project`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := f.Client()
			if err != nil {
				return err
			}

			var projectPath string
			if project != "" {
				projectPath = project
			} else {
				projectPath, err = f.FullProjectPath()
				if err != nil {
					return err
				}
			}

			opts := &gitlab.ListProjectRegistryRepositoriesOptions{
				ListOptions: gitlab.ListOptions{PerPage: int64(limit)},
			}

			repositories, resp, err := client.ContainerRegistry.ListProjectRegistryRepositories(projectPath, opts)
			if err != nil {
				statusCode := 0
				if resp != nil {
					statusCode = resp.StatusCode
				}
				url := api.APIURL(client.Host()) + "/projects/" + projectPath + "/registry/repositories"
				return errors.NewAPIError("GET", url, statusCode, "Failed to list container repositories", err)
			}

			if len(repositories) == 0 {
				_, _ = fmt.Fprintln(f.IOStreams.ErrOut, "No container repositories found")
				return nil
			}

			return f.FormatAndPrint(repositories, format, jsonFlag)
		},
	}

	cmd.Flags().StringVar(&project, "project", "", "Project to list repositories from (uses current project if not specified)")
	cmd.Flags().IntVarP(&limit, "limit", "L", 30, "Maximum number of results")
	cmd.Flags().StringVarP(&format, "format", "F", "table", "Output format: json, table, or plain")
	cmd.Flags().BoolVar(&jsonFlag, "json", false, "Output as JSON (deprecated: use --format=json)")

	return cmd
}

func newRegistryTagsCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		limit    int
		format   string
		jsonFlag bool
		project  string
	)

	cmd := &cobra.Command{
		Use:   "tags <repository-id>",
		Short: "List image tags",
		Example: `  $ glab registry tags 123
  $ glab registry tags 456 --project my-group/my-project`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := f.Client()
			if err != nil {
				return err
			}

			var projectPath string
			if project != "" {
				projectPath = project
			} else {
				projectPath, err = f.FullProjectPath()
				if err != nil {
					return err
				}
			}

			repositoryIDStr := args[0]
			repositoryID, err := strconv.ParseInt(repositoryIDStr, 10, 64)
			if err != nil {
				return fmt.Errorf("invalid repository ID: %s", repositoryIDStr)
			}

			opts := &gitlab.ListRegistryRepositoryTagsOptions{
				ListOptions: gitlab.ListOptions{PerPage: int64(limit)},
			}

			tags, resp, err := client.ContainerRegistry.ListRegistryRepositoryTags(projectPath, repositoryID, opts)
			if err != nil {
				statusCode := 0
				if resp != nil {
					statusCode = resp.StatusCode
				}
				url := api.APIURL(client.Host()) + "/projects/" + projectPath + "/registry/repositories/" + repositoryIDStr + "/tags"
				return errors.NewAPIError("GET", url, statusCode, "Failed to list repository tags", err)
			}

			if len(tags) == 0 {
				_, _ = fmt.Fprintln(f.IOStreams.ErrOut, "No tags found")
				return nil
			}

			return f.FormatAndPrint(tags, format, jsonFlag)
		},
	}

	cmd.Flags().StringVar(&project, "project", "", "Project to list tags from (uses current project if not specified)")
	cmd.Flags().IntVarP(&limit, "limit", "L", 30, "Maximum number of results")
	cmd.Flags().StringVarP(&format, "format", "F", "table", "Output format: json, table, or plain")
	cmd.Flags().BoolVar(&jsonFlag, "json", false, "Output as JSON (deprecated: use --format=json)")

	return cmd
}

func newRegistryViewCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		format   string
		jsonFlag bool
		tag      string
		project  string
	)

	cmd := &cobra.Command{
		Use:   "view <repository-id>",
		Short: "View repository details",
		Args:  cobra.ExactArgs(1),
		Example: `  $ glab registry view 123
  $ glab registry view 456
  $ glab registry view 123 --tag latest
  $ glab registry view 456 --tag v1.0.0 --project my-group/my-project`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := f.Client()
			if err != nil {
				return err
			}

			repositoryIDStr := args[0]

			// If --tag flag is provided, fetch tag details instead
			if tag != "" {
				var projectPath string
				if project != "" {
					projectPath = project
				} else {
					projectPath, err = f.FullProjectPath()
					if err != nil {
						return err
					}
				}

				repositoryID, err := strconv.ParseInt(repositoryIDStr, 10, 64)
				if err != nil {
					return fmt.Errorf("invalid repository ID: %s", repositoryIDStr)
				}

				tagDetail, resp, err := client.ContainerRegistry.GetRegistryRepositoryTagDetail(projectPath, repositoryID, tag)
				if err != nil {
					statusCode := 0
					if resp != nil {
						statusCode = resp.StatusCode
					}
					url := api.APIURL(client.Host()) + "/projects/" + projectPath + "/registry/repositories/" + repositoryIDStr + "/tags/" + tag
					return errors.NewAPIError("GET", url, statusCode, "Failed to get tag details", err)
				}

				// Backward compatibility: --json flag sets format to json
				if jsonFlag {
					format = "json"
				}

				// Validate format flag
				if format != "" && format != "table" {
					return f.FormatAndPrint(tagDetail, format, false)
				}

				// Default custom display for tag
				out := f.IOStreams.Out
				_, _ = fmt.Fprintf(out, "%s\n", tagDetail.Name)
				if tagDetail.Location != "" {
					_, _ = fmt.Fprintf(out, "\n%s\n\n", tagDetail.Location)
				}
				if tagDetail.CreatedAt != nil {
					_, _ = fmt.Fprintf(out, "Created:        %s\n", tagDetail.CreatedAt.String())
				}
				if tagDetail.Digest != "" {
					_, _ = fmt.Fprintf(out, "Digest:         %s\n", tagDetail.Digest)
				}
				if tagDetail.TotalSize != 0 {
					_, _ = fmt.Fprintf(out, "Total Size:     %d\n", tagDetail.TotalSize)
				}

				return nil
			}

			// Default behavior: view repository details
			repositoryID := repositoryIDStr

			// Options to include additional details
			trueVal := true
			opts := &gitlab.GetSingleRegistryRepositoryOptions{
				Tags:      &trueVal,
				TagsCount: &trueVal,
			}

			repository, resp, err := client.ContainerRegistry.GetSingleRegistryRepository(repositoryID, opts)
			if err != nil {
				statusCode := 0
				if resp != nil {
					statusCode = resp.StatusCode
				}
				url := api.APIURL(client.Host()) + "/registry/repositories/" + repositoryID
				return errors.NewAPIError("GET", url, statusCode, "Failed to get container repository", err)
			}

			// Backward compatibility: --json flag sets format to json
			if jsonFlag {
				format = "json"
			}

			// Validate format flag
			if format != "" && format != "table" {
				return f.FormatAndPrint(repository, format, false)
			}

			// Default custom display
			out := f.IOStreams.Out
			_, _ = fmt.Fprintf(out, "%s\n", repository.Path)
			if repository.Location != "" {
				_, _ = fmt.Fprintf(out, "\n%s\n\n", repository.Location)
			}
			_, _ = fmt.Fprintf(out, "ID:             %d\n", repository.ID)
			_, _ = fmt.Fprintf(out, "Project ID:     %d\n", repository.ProjectID)
			_, _ = fmt.Fprintf(out, "Name:           %s\n", repository.Name)
			if repository.CreatedAt != nil {
				_, _ = fmt.Fprintf(out, "Created:        %s\n", repository.CreatedAt.String())
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&tag, "tag", "", "View specific tag details")
	cmd.Flags().StringVar(&project, "project", "", "Project to get tag from (uses current project if not specified, required with --tag)")
	cmd.Flags().StringVarP(&format, "format", "F", "table", "Output format: json, table, or plain")
	cmd.Flags().BoolVar(&jsonFlag, "json", false, "Output as JSON (deprecated: use --format=json)")

	return cmd
}

func newRegistryDeleteCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		tag       string
		yes       bool
		project   string
		olderThan string
	)

	cmd := &cobra.Command{
		Use:   "delete <repository-id>",
		Short: "Delete image tags",
		Example: `  $ glab registry delete 123 --tag v1.0.0
  $ glab registry delete 456 --tag latest --yes
  $ glab registry delete 789 --tag dev --project my-group/my-project
  $ glab registry delete 123 --older-than 30d --yes`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := f.Client()
			if err != nil {
				return err
			}

			var projectPath string
			if project != "" {
				projectPath = project
			} else {
				projectPath, err = f.FullProjectPath()
				if err != nil {
					return err
				}
			}

			repositoryIDStr := args[0]
			repositoryID, err := strconv.ParseInt(repositoryIDStr, 10, 64)
			if err != nil {
				return fmt.Errorf("invalid repository ID: %s", repositoryIDStr)
			}

			// Handle bulk deletion with --older-than
			if olderThan != "" {
				// Parse duration string (e.g., "30d", "7d", "24h")
				duration, err := parseDuration(olderThan)
				if err != nil {
					return fmt.Errorf("invalid duration format: %s (use format like '30d', '7d', '24h')", olderThan)
				}

				// Calculate cutoff time
				cutoffTime := time.Now().Add(-duration)

				// Fetch all tags
				opts := &gitlab.ListRegistryRepositoryTagsOptions{
					ListOptions: gitlab.ListOptions{PerPage: 100},
				}

				tags, resp, err := client.ContainerRegistry.ListRegistryRepositoryTags(projectPath, repositoryID, opts)
				if err != nil {
					statusCode := 0
					if resp != nil {
						statusCode = resp.StatusCode
					}
					url := api.APIURL(client.Host()) + "/projects/" + projectPath + "/registry/repositories/" + repositoryIDStr + "/tags"
					return errors.NewAPIError("GET", url, statusCode, "Failed to list repository tags", err)
				}

				// Filter tags by age
				var tagsToDelete []string
				for _, t := range tags {
					if t.CreatedAt != nil && t.CreatedAt.Before(cutoffTime) {
						tagsToDelete = append(tagsToDelete, t.Name)
					}
				}

				if len(tagsToDelete) == 0 {
					_, _ = fmt.Fprintf(f.IOStreams.Out, "No tags older than %s found\n", olderThan)
					return nil
				}

				// Prompt for confirmation unless --yes is provided
				if !yes {
					out := f.IOStreams.Out
					_, _ = fmt.Fprintf(out, "Found %d tag(s) older than %s:\n", len(tagsToDelete), olderThan)
					for _, tagName := range tagsToDelete {
						_, _ = fmt.Fprintf(out, "  - %s\n", tagName)
					}
					_, _ = fmt.Fprintf(out, "\nAre you sure you want to delete these tags? [y/N] ")

					var response string
					_, err := fmt.Scanln(&response)
					if err != nil && err.Error() != "unexpected newline" {
						return err
					}

					if response != "y" && response != "Y" && response != "yes" && response != "Yes" {
						_, _ = fmt.Fprintln(f.IOStreams.ErrOut, "Deletion cancelled")
						return nil
					}
				}

				// Delete each tag
				deletedCount := 0
				failedCount := 0
				for _, tagName := range tagsToDelete {
					resp, err := client.ContainerRegistry.DeleteRegistryRepositoryTag(projectPath, repositoryID, tagName)
					if err != nil {
						statusCode := 0
						if resp != nil {
							statusCode = resp.StatusCode
						}
						url := api.APIURL(client.Host()) + "/projects/" + projectPath + "/registry/repositories/" + repositoryIDStr + "/tags/" + tagName
						_, _ = fmt.Fprintf(f.IOStreams.ErrOut, "Failed to delete tag '%s': %v\n", tagName, err)
						_ = errors.NewAPIError("DELETE", url, statusCode, "Failed to delete tag", err)
						failedCount++
						continue
					}
					deletedCount++
					_, _ = fmt.Fprintf(f.IOStreams.Out, "Deleted tag '%s'\n", tagName)
				}

				_, _ = fmt.Fprintf(f.IOStreams.Out, "\nDeleted %d of %d tag(s)\n", deletedCount, len(tagsToDelete))
				if failedCount > 0 {
					return fmt.Errorf("failed to delete %d of %d tag(s)", failedCount, len(tagsToDelete))
				}
				return nil
			}

			// Require --tag flag for single tag deletion
			if tag == "" {
				return fmt.Errorf("--tag flag is required for tag deletion")
			}

			// Prompt for confirmation unless --yes is provided
			if !yes {
				out := f.IOStreams.Out
				_, _ = fmt.Fprintf(out, "Are you sure you want to delete tag '%s' from repository %s? [y/N] ", tag, repositoryIDStr)

				var response string
				_, err := fmt.Scanln(&response)
				if err != nil && err.Error() != "unexpected newline" {
					return err
				}

				if response != "y" && response != "Y" && response != "yes" && response != "Yes" {
					_, _ = fmt.Fprintln(f.IOStreams.ErrOut, "Deletion cancelled")
					return nil
				}
			}

			// Delete the tag
			resp, err := client.ContainerRegistry.DeleteRegistryRepositoryTag(projectPath, repositoryID, tag)
			if err != nil {
				statusCode := 0
				if resp != nil {
					statusCode = resp.StatusCode
				}
				url := api.APIURL(client.Host()) + "/projects/" + projectPath + "/registry/repositories/" + repositoryIDStr + "/tags/" + tag
				return errors.NewAPIError("DELETE", url, statusCode, "Failed to delete tag", err)
			}

			out := f.IOStreams.Out
			_, _ = fmt.Fprintf(out, "Deleted tag '%s' from repository %s\n", tag, repositoryIDStr)

			return nil
		},
	}

	cmd.Flags().StringVar(&tag, "tag", "", "Tag name to delete (required for single tag deletion)")
	cmd.Flags().StringVar(&olderThan, "older-than", "", "Delete tags older than specified duration (e.g., '30d', '7d', '24h')")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation prompt")
	cmd.Flags().StringVar(&project, "project", "", "Project to delete tag from (uses current project if not specified)")

	return cmd
}

// parseDuration parses a duration string like "30d", "7d", "24h" into a time.Duration
func parseDuration(s string) (time.Duration, error) {
	if len(s) < 2 {
		return 0, fmt.Errorf("invalid duration format")
	}

	// Extract number and unit
	numStr := s[:len(s)-1]
	unit := s[len(s)-1:]

	// Parse the number
	num, err := strconv.ParseInt(numStr, 10, 64)
	if err != nil {
		return 0, err
	}

	// Convert based on unit
	switch unit {
	case "d":
		return time.Duration(num) * 24 * time.Hour, nil
	case "h":
		return time.Duration(num) * time.Hour, nil
	case "m":
		return time.Duration(num) * time.Minute, nil
	case "s":
		return time.Duration(num) * time.Second, nil
	default:
		return 0, fmt.Errorf("unsupported unit: %s (use d, h, m, or s)", unit)
	}
}
