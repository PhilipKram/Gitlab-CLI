package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/PhilipKram/gitlab-cli/internal/cmdutil"
	"github.com/PhilipKram/gitlab-cli/internal/tableprinter"
	"github.com/spf13/cobra"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// NewProjectCmd creates the project command group.
func NewProjectCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "project <command>",
		Short: "Manage projects",
		Long:  "List and view GitLab projects and groups.",
	}

	cmd.AddCommand(newProjectListCmd(f))
	cmd.AddCommand(newProjectViewCmd(f))
	cmd.AddCommand(newProjectMembersCmd(f))

	return cmd
}

func newProjectListCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		group    string
		limit    int
		jsonFlag bool
		search   string
	)

	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List projects",
		Aliases: []string{"ls"},
		Example: `  $ glab project list
  $ glab project list --group my-org
  $ glab project list --search "api"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := f.Client()
			if err != nil {
				return err
			}

			var projects []*gitlab.Project

			if group != "" {
				opts := &gitlab.ListGroupProjectsOptions{
					ListOptions: gitlab.ListOptions{PerPage: int64(limit)},
				}
				if search != "" {
					opts.Search = &search
				}
				projects, _, err = client.Groups.ListGroupProjects(group, opts)
			} else {
				trueVal := true
				opts := &gitlab.ListProjectsOptions{
					ListOptions: gitlab.ListOptions{PerPage: int64(limit)},
					Membership:  &trueVal,
				}
				if search != "" {
					opts.Search = &search
				}
				projects, _, err = client.Projects.ListProjects(opts)
			}

			if err != nil {
				return fmt.Errorf("listing projects: %w", err)
			}

			if len(projects) == 0 {
				fmt.Fprintln(f.IOStreams.ErrOut, "No projects found")
				return nil
			}

			if jsonFlag {
				data, err := json.MarshalIndent(projects, "", "  ")
				if err != nil {
					return err
				}
				fmt.Fprintln(f.IOStreams.Out, string(data))
				return nil
			}

			tp := tableprinter.New(f.IOStreams.Out)
			for _, p := range projects {
				tp.AddRow(
					p.PathWithNamespace,
					truncate(p.Description, 50),
					string(p.Visibility),
				)
			}
			return tp.Render()
		},
	}

	cmd.Flags().StringVar(&group, "group", "", "Filter by group")
	cmd.Flags().IntVarP(&limit, "limit", "L", 30, "Maximum number of results")
	cmd.Flags().BoolVar(&jsonFlag, "json", false, "Output as JSON")
	cmd.Flags().StringVar(&search, "search", "", "Search projects")

	return cmd
}

func newProjectViewCmd(f *cmdutil.Factory) *cobra.Command {
	var jsonFlag bool

	cmd := &cobra.Command{
		Use:   "view [<owner/repo>]",
		Short: "View project details",
		Example: `  $ glab project view
  $ glab project view my-group/my-project`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := f.Client()
			if err != nil {
				return err
			}

			var projectPath string
			if len(args) > 0 {
				projectPath = args[0]
			} else {
				projectPath, err = f.FullProjectPath()
				if err != nil {
					return err
				}
			}

			project, _, err := client.Projects.GetProject(projectPath, nil)
			if err != nil {
				return fmt.Errorf("getting project: %w", err)
			}

			if jsonFlag {
				data, err := json.MarshalIndent(project, "", "  ")
				if err != nil {
					return err
				}
				fmt.Fprintln(f.IOStreams.Out, string(data))
				return nil
			}

			out := f.IOStreams.Out
			fmt.Fprintf(out, "%s\n", project.PathWithNamespace)
			if project.Description != "" {
				fmt.Fprintf(out, "\n%s\n\n", project.Description)
			}
			fmt.Fprintf(out, "ID:             %d\n", project.ID)
			fmt.Fprintf(out, "Visibility:     %s\n", project.Visibility)
			fmt.Fprintf(out, "Default branch: %s\n", project.DefaultBranch)
			fmt.Fprintf(out, "Stars:          %d\n", project.StarCount)
			fmt.Fprintf(out, "Forks:          %d\n", project.ForksCount)
			fmt.Fprintf(out, "Open issues:    %d\n", project.OpenIssuesCount)
			fmt.Fprintf(out, "URL:            %s\n", project.WebURL)

			return nil
		},
	}

	cmd.Flags().BoolVar(&jsonFlag, "json", false, "Output as JSON")

	return cmd
}

func newProjectMembersCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		limit    int
		jsonFlag bool
	)

	cmd := &cobra.Command{
		Use:   "members [<owner/repo>]",
		Short: "List project members",
		Example: `  $ glab project members
  $ glab project members my-group/my-project`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := f.Client()
			if err != nil {
				return err
			}

			var projectPath string
			if len(args) > 0 {
				projectPath = args[0]
			} else {
				projectPath, err = f.FullProjectPath()
				if err != nil {
					return err
				}
			}

			opts := &gitlab.ListProjectMembersOptions{
				ListOptions: gitlab.ListOptions{PerPage: int64(limit)},
			}

			members, _, err := client.ProjectMembers.ListAllProjectMembers(projectPath, opts)
			if err != nil {
				return fmt.Errorf("listing members: %w", err)
			}

			if len(members) == 0 {
				fmt.Fprintln(f.IOStreams.ErrOut, "No members found")
				return nil
			}

			if jsonFlag {
				data, err := json.MarshalIndent(members, "", "  ")
				if err != nil {
					return err
				}
				fmt.Fprintln(f.IOStreams.Out, string(data))
				return nil
			}

			tp := tableprinter.New(f.IOStreams.Out)
			for _, m := range members {
				tp.AddRow(
					m.Username,
					m.Name,
					accessLevelName(m.AccessLevel),
				)
			}
			return tp.Render()
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "L", 30, "Maximum number of results")
	cmd.Flags().BoolVar(&jsonFlag, "json", false, "Output as JSON")

	return cmd
}

func accessLevelName(level gitlab.AccessLevelValue) string {
	switch level {
	case gitlab.NoPermissions:
		return "None"
	case gitlab.MinimalAccessPermissions:
		return "Minimal"
	case gitlab.GuestPermissions:
		return "Guest"
	case gitlab.ReporterPermissions:
		return "Reporter"
	case gitlab.DeveloperPermissions:
		return "Developer"
	case gitlab.MaintainerPermissions:
		return "Maintainer"
	case gitlab.OwnerPermissions:
		return "Owner"
	default:
		return fmt.Sprintf("Level %d", level)
	}
}
