package cmd

import (
	"fmt"

	"github.com/PhilipKram/gitlab-cli/internal/api"
	"github.com/PhilipKram/gitlab-cli/internal/browser"
	"github.com/PhilipKram/gitlab-cli/internal/cmdutil"
	"github.com/spf13/cobra"
)

// NewBrowseCmd creates the browse command.
func NewBrowseCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		branch   string
		settings bool
		members  bool
		issues   bool
		mrs      bool
		pipeline bool
	)

	cmd := &cobra.Command{
		Use:   "browse [path]",
		Short: "Open project in browser",
		Long:  "Open the GitLab project page in your default web browser.",
		Example: `  $ glab browse
  $ glab browse --settings
  $ glab browse --issues
  $ glab browse --mrs
  $ glab browse --pipeline
  $ glab browse src/main.go
  $ glab browse src/main.go --branch develop`,
		RunE: func(cmd *cobra.Command, args []string) error {
			remote, err := f.Remote()
			if err != nil {
				return err
			}

			project, err := f.FullProjectPath()
			if err != nil {
				return err
			}

			baseURL := api.WebURL(remote.Host, project)

			var url string
			switch {
			case settings:
				url = baseURL + "/-/edit"
			case members:
				url = baseURL + "/-/project_members"
			case issues:
				url = baseURL + "/-/issues"
			case mrs:
				url = baseURL + "/-/merge_requests"
			case pipeline:
				url = baseURL + "/-/pipelines"
			case len(args) > 0:
				path := args[0]
				if branch == "" {
					branch = "HEAD"
				}
				url = fmt.Sprintf("%s/-/blob/%s/%s", baseURL, branch, path)
			default:
				url = baseURL
			}

			fmt.Fprintln(f.IOStreams.Out, url)
			return browser.Open(url)
		},
	}

	cmd.Flags().StringVarP(&branch, "branch", "b", "", "Branch to browse")
	cmd.Flags().BoolVarP(&settings, "settings", "s", false, "Open settings page")
	cmd.Flags().BoolVar(&members, "members", false, "Open members page")
	cmd.Flags().BoolVar(&issues, "issues", false, "Open issues page")
	cmd.Flags().BoolVar(&mrs, "mrs", false, "Open merge requests page")
	cmd.Flags().BoolVarP(&pipeline, "pipeline", "p", false, "Open pipelines page")

	return cmd
}
