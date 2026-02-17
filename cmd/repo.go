package cmd

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/PhilipKram/gitlab-cli/internal/api"
	"github.com/PhilipKram/gitlab-cli/internal/browser"
	"github.com/PhilipKram/gitlab-cli/internal/cmdutil"
	"github.com/PhilipKram/gitlab-cli/internal/config"
	"github.com/PhilipKram/gitlab-cli/internal/tableprinter"
	"github.com/spf13/cobra"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// NewRepoCmd creates the repo command group.
func NewRepoCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "repo <command>",
		Short:   "Manage repositories",
		Long:    "Create, clone, fork, and manage GitLab repositories.",
		Aliases: []string{"project"},
	}

	cmd.AddCommand(newRepoCloneCmd(f))
	cmd.AddCommand(newRepoCreateCmd(f))
	cmd.AddCommand(newRepoForkCmd(f))
	cmd.AddCommand(newRepoViewCmd(f))
	cmd.AddCommand(newRepoListCmd(f))
	cmd.AddCommand(newRepoArchiveCmd(f))
	cmd.AddCommand(newRepoDeleteCmd(f))

	return cmd
}

func newRepoCloneCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "clone <owner/repo>",
		Short: "Clone a repository",
		Example: `  $ glab repo clone owner/repo
  $ glab repo clone owner/repo -- --depth 1`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repoPath := args[0]
			host := config.DefaultHost()

			// Build clone URL
			cfg, _ := f.Config()
			protocol := "https"
			if cfg != nil && cfg.Protocol != "" {
				protocol = cfg.Protocol
			}

			var cloneURL string
			if protocol == "ssh" {
				cloneURL = fmt.Sprintf("git@%s:%s.git", host, repoPath)
			} else {
				cloneURL = fmt.Sprintf("https://%s/%s.git", host, repoPath)
			}

			gitArgs := []string{"clone", cloneURL}
			if len(args) > 1 {
				gitArgs = append(gitArgs, args[1:]...)
			}

			gitCmd := exec.Command("git", gitArgs...)
			gitCmd.Stdout = f.IOStreams.Out
			gitCmd.Stderr = f.IOStreams.ErrOut
			gitCmd.Stdin = f.IOStreams.In

			if err := gitCmd.Run(); err != nil {
				return fmt.Errorf("cloning repository: %w", err)
			}

			return nil
		},
	}

	return cmd
}

func newRepoCreateCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		name         string
		description  string
		visibility   string
		internal     bool
		private      bool
		public       bool
		initReadme   bool
		defaultBranch string
		groupID      int64
		web          bool
	)

	cmd := &cobra.Command{
		Use:   "create [<name>]",
		Short: "Create a new repository",
		Example: `  $ glab repo create my-project
  $ glab repo create my-project --description "A new project" --private
  $ glab repo create my-project --group-id 123 --public`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				name = args[0]
			}
			if name == "" {
				return fmt.Errorf("repository name is required")
			}

			client, err := f.Client()
			if err != nil {
				// Fall back to default host
				client, err = api.NewClient(config.DefaultHost())
				if err != nil {
					return err
				}
			}

			vis := gitlab.PrivateVisibility
			if public {
				vis = gitlab.PublicVisibility
			} else if internal {
				vis = gitlab.InternalVisibility
			} else if visibility != "" {
				switch visibility {
				case "public":
					vis = gitlab.PublicVisibility
				case "internal":
					vis = gitlab.InternalVisibility
				case "private":
					vis = gitlab.PrivateVisibility
				default:
					return fmt.Errorf("invalid visibility: %s (use public, internal, or private)", visibility)
				}
			}

			opts := &gitlab.CreateProjectOptions{
				Name:                 &name,
				Description:          &description,
				Visibility:           &vis,
				InitializeWithReadme: &initReadme,
			}

			if defaultBranch != "" {
				opts.DefaultBranch = &defaultBranch
			}

			if groupID > 0 {
				opts.NamespaceID = &groupID
			}

			project, _, err := client.Projects.CreateProject(opts)
			if err != nil {
				return fmt.Errorf("creating repository: %w", err)
			}

			out := f.IOStreams.Out
			fmt.Fprintf(out, "Created repository %s\n", project.PathWithNamespace)
			fmt.Fprintf(out, "%s\n", project.WebURL)

			if web {
				_ = browser.Open(project.WebURL)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&description, "description", "d", "", "Repository description")
	cmd.Flags().StringVar(&visibility, "visibility", "", "Visibility: public, internal, private")
	cmd.Flags().BoolVar(&public, "public", false, "Make repository public")
	cmd.Flags().BoolVar(&private, "private", false, "Make repository private (default)")
	cmd.Flags().BoolVar(&internal, "internal", false, "Make repository internal")
	cmd.Flags().BoolVar(&initReadme, "init", false, "Initialize with README")
	cmd.Flags().StringVar(&defaultBranch, "default-branch", "", "Default branch name")
	cmd.Flags().Int64Var(&groupID, "group-id", 0, "Group/namespace ID")
	cmd.Flags().BoolVarP(&web, "web", "w", false, "Open in browser after creation")

	return cmd
}

func newRepoForkCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		targetNamespace string
		targetName      string
		cloneAfter      bool
	)

	cmd := &cobra.Command{
		Use:   "fork [<owner/repo>]",
		Short: "Fork a repository",
		Example: `  $ glab repo fork
  $ glab repo fork owner/repo
  $ glab repo fork owner/repo --namespace my-group --clone`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := f.Client()
			if err != nil {
				return err
			}

			var project string
			if len(args) > 0 {
				project = args[0]
			} else {
				project, err = f.FullProjectPath()
				if err != nil {
					return err
				}
			}

			opts := &gitlab.ForkProjectOptions{}
			if targetNamespace != "" {
				opts.NamespacePath = &targetNamespace
			}
			if targetName != "" {
				opts.Name = &targetName
			}

			forked, _, err := client.Projects.ForkProject(project, opts)
			if err != nil {
				return fmt.Errorf("forking repository: %w", err)
			}

			out := f.IOStreams.Out
			fmt.Fprintf(out, "Forked %s to %s\n", project, forked.PathWithNamespace)
			fmt.Fprintf(out, "%s\n", forked.WebURL)

			if cloneAfter {
				gitCmd := exec.Command("git", "clone", forked.HTTPURLToRepo)
				gitCmd.Stdout = f.IOStreams.Out
				gitCmd.Stderr = f.IOStreams.ErrOut
				if err := gitCmd.Run(); err != nil {
					return fmt.Errorf("cloning forked repository: %w", err)
				}
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&targetNamespace, "namespace", "", "Target namespace for the fork")
	cmd.Flags().StringVar(&targetName, "name", "", "Name for the forked repository")
	cmd.Flags().BoolVar(&cloneAfter, "clone", false, "Clone the fork after creation")

	return cmd
}

func newRepoViewCmd(f *cmdutil.Factory) *cobra.Command {
	var web bool
	var jsonFlag bool

	cmd := &cobra.Command{
		Use:   "view [<owner/repo>]",
		Short: "View a repository",
		Example: `  $ glab repo view
  $ glab repo view owner/repo
  $ glab repo view --web`,
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

			if web {
				return browser.Open(project.WebURL)
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
				fmt.Fprintf(out, "\n%s\n", project.Description)
			}
			fmt.Fprintln(out)
			fmt.Fprintf(out, "Visibility:     %s\n", project.Visibility)
			fmt.Fprintf(out, "Default branch: %s\n", project.DefaultBranch)
			fmt.Fprintf(out, "Stars:          %d\n", project.StarCount)
			fmt.Fprintf(out, "Forks:          %d\n", project.ForksCount)
			if project.ForkedFromProject != nil {
				fmt.Fprintf(out, "Forked from:    %s\n", project.ForkedFromProject.PathWithNamespace)
			}
			fmt.Fprintf(out, "URL:            %s\n", project.WebURL)
			fmt.Fprintf(out, "SSH URL:        %s\n", project.SSHURLToRepo)
			fmt.Fprintf(out, "HTTP URL:       %s\n", project.HTTPURLToRepo)

			return nil
		},
	}

	cmd.Flags().BoolVarP(&web, "web", "w", false, "Open in browser")
	cmd.Flags().BoolVar(&jsonFlag, "json", false, "Output as JSON")

	return cmd
}

func newRepoListCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		owner    string
		limit    int
		jsonFlag bool
		archived bool
		search   string
	)

	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List repositories",
		Aliases: []string{"ls"},
		Example: `  $ glab repo list
  $ glab repo list --owner my-group --limit 50
  $ glab repo list --archived --search "web"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			host := config.DefaultHost()
			client, err := api.NewClient(host)
			if err != nil {
				return err
			}

			var projects []*gitlab.Project

			if owner != "" {
				// List group projects
				opts := &gitlab.ListGroupProjectsOptions{
					ListOptions: gitlab.ListOptions{PerPage: int64(limit)},
				}
				if cmd.Flags().Changed("archived") {
					opts.Archived = &archived
				}
				if search != "" {
					opts.Search = &search
				}
				projects, _, err = client.Groups.ListGroupProjects(owner, opts)
			} else {
				// List authenticated user's projects
				trueVal := true
				opts := &gitlab.ListProjectsOptions{
					ListOptions: gitlab.ListOptions{PerPage: int64(limit)},
					Membership:  &trueVal,
				}
				if cmd.Flags().Changed("archived") {
					opts.Archived = &archived
				}
				if search != "" {
					opts.Search = &search
				}
				projects, _, err = client.Projects.ListProjects(opts)
			}

			if err != nil {
				return fmt.Errorf("listing repositories: %w", err)
			}

			if len(projects) == 0 {
				fmt.Fprintln(f.IOStreams.ErrOut, "No repositories found")
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
				vis := string(p.Visibility)
				tp.AddRow(
					p.PathWithNamespace,
					truncate(p.Description, 50),
					vis,
					timeAgo(p.LastActivityAt),
				)
			}
			return tp.Render()
		},
	}

	cmd.Flags().StringVarP(&owner, "owner", "o", "", "Filter by group/user")
	cmd.Flags().IntVarP(&limit, "limit", "L", 30, "Maximum number of results")
	cmd.Flags().BoolVar(&jsonFlag, "json", false, "Output as JSON")
	cmd.Flags().BoolVar(&archived, "archived", false, "Include archived repositories")
	cmd.Flags().StringVar(&search, "search", "", "Search repositories")

	return cmd
}

func newRepoArchiveCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "archive [<owner/repo>]",
		Short: "Archive a repository",
		Example: `  $ glab repo archive
  $ glab repo archive owner/repo`,
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

			project, _, err := client.Projects.ArchiveProject(projectPath)
			if err != nil {
				return fmt.Errorf("archiving repository: %w", err)
			}

			fmt.Fprintf(f.IOStreams.Out, "Archived %s\n", project.PathWithNamespace)
			return nil
		},
	}

	return cmd
}

func newRepoDeleteCmd(f *cmdutil.Factory) *cobra.Command {
	var confirm bool

	cmd := &cobra.Command{
		Use:   "delete <owner/repo>",
		Short: "Delete a repository",
		Example: `  $ glab repo delete owner/repo --confirm`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !confirm {
				return fmt.Errorf("use --confirm to delete repository %s", args[0])
			}

			client, err := f.Client()
			if err != nil {
				return err
			}

			_, err = client.Projects.DeleteProject(args[0], nil)
			if err != nil {
				return fmt.Errorf("deleting repository: %w", err)
			}

			fmt.Fprintf(f.IOStreams.Out, "Deleted repository %s\n", args[0])
			return nil
		},
	}

	cmd.Flags().BoolVar(&confirm, "confirm", false, "Confirm deletion")

	return cmd
}

func truncate(s string, maxLen int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
