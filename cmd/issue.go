package cmd

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/PhilipKram/gitlab-cli/internal/api"
	"github.com/PhilipKram/gitlab-cli/internal/browser"
	"github.com/PhilipKram/gitlab-cli/internal/cmdutil"
	"github.com/PhilipKram/gitlab-cli/internal/tableprinter"
	"github.com/spf13/cobra"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// NewIssueCmd creates the issue command group.
func NewIssueCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "issue <command>",
		Short: "Manage issues",
		Long:  "Create, view, and manage GitLab issues.",
	}

	cmd.AddCommand(newIssueCreateCmd(f))
	cmd.AddCommand(newIssueListCmd(f))
	cmd.AddCommand(newIssueViewCmd(f))
	cmd.AddCommand(newIssueCloseCmd(f))
	cmd.AddCommand(newIssueReopenCmd(f))
	cmd.AddCommand(newIssueCommentCmd(f))
	cmd.AddCommand(newIssueEditCmd(f))
	cmd.AddCommand(newIssueDeleteCmd(f))

	return cmd
}

func newIssueCreateCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		title        string
		description  string
		assignees    []string
		labels       []string
		milestone    string
		confidential bool
		weight       int64
		web          bool
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create an issue",
		Example: `  $ glab issue create --title "Bug report" --description "Steps to reproduce..."
  $ glab issue create --title "Feature request" --label enhancement --assignee @user1
  $ glab issue create --title "Secret issue" --confidential`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := f.Client()
			if err != nil {
				return err
			}

			project, err := f.FullProjectPath()
			if err != nil {
				return err
			}

			opts := &gitlab.CreateIssueOptions{
				Title:        &title,
				Description:  &description,
				Confidential: &confidential,
			}

			if len(assignees) > 0 {
				ids, err := resolveUserIDs(client, assignees)
				if err != nil {
					return fmt.Errorf("resolving assignees: %w", err)
				}
				opts.AssigneeIDs = &ids
			}

			if len(labels) > 0 {
				labelOpts := gitlab.LabelOptions(labels)
				opts.Labels = &labelOpts
			}

			if milestone != "" {
				mid, err := strconv.ParseInt(milestone, 10, 64)
				if err != nil {
					return fmt.Errorf("invalid milestone ID: %s", milestone)
				}
				opts.MilestoneID = &mid
			}

			if cmd.Flags().Changed("weight") {
				opts.Weight = &weight
			}

			issue, _, err := client.Issues.CreateIssue(project, opts)
			if err != nil {
				return fmt.Errorf("creating issue: %w", err)
			}

			out := f.IOStreams.Out
			fmt.Fprintf(out, "Created issue #%d\n", issue.IID)
			fmt.Fprintf(out, "%s\n", issue.WebURL)

			if web {
				_ = browser.Open(issue.WebURL)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&title, "title", "t", "", "Issue title (required)")
	cmd.Flags().StringVarP(&description, "description", "d", "", "Issue description")
	cmd.Flags().StringSliceVarP(&assignees, "assignee", "a", nil, "Assign users by username")
	cmd.Flags().StringSliceVarP(&labels, "label", "l", nil, "Add labels")
	cmd.Flags().StringVarP(&milestone, "milestone", "m", "", "Milestone ID")
	cmd.Flags().BoolVar(&confidential, "confidential", false, "Mark as confidential")
	cmd.Flags().Int64Var(&weight, "weight", 0, "Issue weight")
	cmd.Flags().BoolVarP(&web, "web", "w", false, "Open in browser after creation")
	_ = cmd.MarkFlagRequired("title")

	return cmd
}

func newIssueListCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		state     string
		author    string
		assignee  string
		labels    []string
		milestone string
		search    string
		limit     int
		jsonFlag  bool
		web       bool
	)

	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List issues",
		Aliases: []string{"ls"},
		Example: `  $ glab issue list
  $ glab issue list --state closed --author johndoe
  $ glab issue list --label bug,critical --limit 50`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := f.Client()
			if err != nil {
				return err
			}

			project, err := f.FullProjectPath()
			if err != nil {
				return err
			}

			if web {
				remote, _ := f.Remote()
				host := "gitlab.com"
				if remote != nil {
					host = remote.Host
				}
				return browser.Open(api.WebURL(host, project+"/-/issues"))
			}

			opts := &gitlab.ListProjectIssuesOptions{
				ListOptions: gitlab.ListOptions{PerPage: int64(limit)},
			}

			if state != "" {
				opts.State = &state
			}
			if author != "" {
				opts.AuthorUsername = &author
			}
			if assignee != "" {
				opts.AssigneeUsername = &assignee
			}
			if len(labels) > 0 {
				labelOpts := gitlab.LabelOptions(labels)
				opts.Labels = &labelOpts
			}
			if milestone != "" {
				opts.Milestone = &milestone
			}
			if search != "" {
				opts.Search = &search
			}

			issues, _, err := client.Issues.ListProjectIssues(project, opts)
			if err != nil {
				return fmt.Errorf("listing issues: %w", err)
			}

			if len(issues) == 0 {
				fmt.Fprintln(f.IOStreams.ErrOut, "No issues match your search")
				return nil
			}

			if jsonFlag {
				data, err := json.MarshalIndent(issues, "", "  ")
				if err != nil {
					return err
				}
				fmt.Fprintln(f.IOStreams.Out, string(data))
				return nil
			}

			tp := tableprinter.New(f.IOStreams.Out)
			for _, issue := range issues {
				assigneeStr := ""
				if len(issue.Assignees) > 0 {
					assigneeStr = issue.Assignees[0].Username
				}
				tp.AddRow(
					fmt.Sprintf("#%d", issue.IID),
					issue.Title,
					issue.State,
					assigneeStr,
					timeAgo(issue.CreatedAt),
				)
			}
			return tp.Render()
		},
	}

	cmd.Flags().StringVar(&state, "state", "opened", "Filter by state: opened, closed, all")
	cmd.Flags().StringVar(&author, "author", "", "Filter by author username")
	cmd.Flags().StringVar(&assignee, "assignee", "", "Filter by assignee username")
	cmd.Flags().StringSliceVarP(&labels, "label", "l", nil, "Filter by labels")
	cmd.Flags().StringVarP(&milestone, "milestone", "m", "", "Filter by milestone")
	cmd.Flags().StringVar(&search, "search", "", "Search in title and description")
	cmd.Flags().IntVarP(&limit, "limit", "L", 30, "Maximum number of results")
	cmd.Flags().BoolVar(&jsonFlag, "json", false, "Output as JSON")
	cmd.Flags().BoolVarP(&web, "web", "w", false, "Open in browser")

	return cmd
}

func newIssueViewCmd(f *cmdutil.Factory) *cobra.Command {
	var web bool
	var jsonFlag bool

	cmd := &cobra.Command{
		Use:   "view [<id>]",
		Short: "View an issue",
		Example: `  $ glab issue view 42
  $ glab issue view 42 --web`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := f.Client()
			if err != nil {
				return err
			}

			project, err := f.FullProjectPath()
			if err != nil {
				return err
			}

			issueID, err := parseIssueArg(args)
			if err != nil {
				return err
			}

			issue, _, err := client.Issues.GetIssue(project, issueID)
			if err != nil {
				return fmt.Errorf("getting issue: %w", err)
			}

			if web {
				return browser.Open(issue.WebURL)
			}

			if jsonFlag {
				data, err := json.MarshalIndent(issue, "", "  ")
				if err != nil {
					return err
				}
				fmt.Fprintln(f.IOStreams.Out, string(data))
				return nil
			}

			out := f.IOStreams.Out
			fmt.Fprintf(out, "#%d %s\n", issue.IID, issue.Title)
			fmt.Fprintf(out, "State:   %s\n", issue.State)
			fmt.Fprintf(out, "Author:  %s\n", issue.Author.Username)
			if len(issue.Assignees) > 0 {
				var names []string
				for _, a := range issue.Assignees {
					names = append(names, a.Username)
				}
				fmt.Fprintf(out, "Assignee: %s\n", strings.Join(names, ", "))
			}
			if len(issue.Labels) > 0 {
				fmt.Fprintf(out, "Labels:  %s\n", strings.Join(issue.Labels, ", "))
			}
			if issue.Milestone != nil {
				fmt.Fprintf(out, "Milestone: %s\n", issue.Milestone.Title)
			}
			if issue.Weight != 0 {
				fmt.Fprintf(out, "Weight:  %d\n", issue.Weight)
			}
			fmt.Fprintf(out, "Created: %s\n", timeAgo(issue.CreatedAt))
			fmt.Fprintf(out, "URL:     %s\n", issue.WebURL)
			if issue.Description != "" {
				fmt.Fprintf(out, "\n%s\n", issue.Description)
			}

			return nil
		},
	}

	cmd.Flags().BoolVarP(&web, "web", "w", false, "Open in browser")
	cmd.Flags().BoolVar(&jsonFlag, "json", false, "Output as JSON")

	return cmd
}

func newIssueCloseCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "close [<id>]",
		Short:   "Close an issue",
		Example: `  $ glab issue close 42`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := f.Client()
			if err != nil {
				return err
			}

			project, err := f.FullProjectPath()
			if err != nil {
				return err
			}

			issueID, err := parseIssueArg(args)
			if err != nil {
				return err
			}

			closed := "close"
			opts := &gitlab.UpdateIssueOptions{
				StateEvent: &closed,
			}

			issue, _, err := client.Issues.UpdateIssue(project, issueID, opts)
			if err != nil {
				return fmt.Errorf("closing issue: %w", err)
			}

			fmt.Fprintf(f.IOStreams.Out, "Closed issue #%d\n", issue.IID)
			return nil
		},
	}

	return cmd
}

func newIssueReopenCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "reopen [<id>]",
		Short:   "Reopen an issue",
		Example: `  $ glab issue reopen 42`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := f.Client()
			if err != nil {
				return err
			}

			project, err := f.FullProjectPath()
			if err != nil {
				return err
			}

			issueID, err := parseIssueArg(args)
			if err != nil {
				return err
			}

			reopen := "reopen"
			opts := &gitlab.UpdateIssueOptions{
				StateEvent: &reopen,
			}

			issue, _, err := client.Issues.UpdateIssue(project, issueID, opts)
			if err != nil {
				return fmt.Errorf("reopening issue: %w", err)
			}

			fmt.Fprintf(f.IOStreams.Out, "Reopened issue #%d\n", issue.IID)
			return nil
		},
	}

	return cmd
}

func newIssueCommentCmd(f *cmdutil.Factory) *cobra.Command {
	var body string

	cmd := &cobra.Command{
		Use:     "comment [<id>]",
		Short:   "Add a comment to an issue",
		Example: `  $ glab issue comment 42 --body "This is a comment"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := f.Client()
			if err != nil {
				return err
			}

			project, err := f.FullProjectPath()
			if err != nil {
				return err
			}

			issueID, err := parseIssueArg(args)
			if err != nil {
				return err
			}

			opts := &gitlab.CreateIssueNoteOptions{
				Body: &body,
			}

			note, _, err := client.Notes.CreateIssueNote(project, issueID, opts)
			if err != nil {
				return fmt.Errorf("adding comment: %w", err)
			}

			fmt.Fprintf(f.IOStreams.Out, "Added comment to #%d\n%s\n", issueID, note.Body)
			return nil
		},
	}

	cmd.Flags().StringVarP(&body, "body", "b", "", "Comment body (required)")
	_ = cmd.MarkFlagRequired("body")

	return cmd
}

func newIssueEditCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		title        string
		description  string
		assignees    []string
		labels       []string
		milestone    string
		confidential bool
		weight       int64
	)

	cmd := &cobra.Command{
		Use:   "edit [<id>]",
		Short: "Edit an issue",
		Example: `  $ glab issue edit 42 --title "Updated title"
  $ glab issue edit 42 --assignee user1 --label bug`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := f.Client()
			if err != nil {
				return err
			}

			project, err := f.FullProjectPath()
			if err != nil {
				return err
			}

			issueID, err := parseIssueArg(args)
			if err != nil {
				return err
			}

			opts := &gitlab.UpdateIssueOptions{}

			if cmd.Flags().Changed("title") {
				opts.Title = &title
			}
			if cmd.Flags().Changed("description") {
				opts.Description = &description
			}
			if cmd.Flags().Changed("assignee") {
				ids, err := resolveUserIDs(client, assignees)
				if err != nil {
					return err
				}
				opts.AssigneeIDs = &ids
			}
			if cmd.Flags().Changed("label") {
				labelOpts := gitlab.LabelOptions(labels)
				opts.Labels = &labelOpts
			}
			if cmd.Flags().Changed("milestone") {
				mid, err := strconv.ParseInt(milestone, 10, 64)
				if err != nil {
					return fmt.Errorf("invalid milestone ID: %s", milestone)
				}
				opts.MilestoneID = &mid
			}
			if cmd.Flags().Changed("confidential") {
				opts.Confidential = &confidential
			}
			if cmd.Flags().Changed("weight") {
				opts.Weight = &weight
			}

			issue, _, err := client.Issues.UpdateIssue(project, issueID, opts)
			if err != nil {
				return fmt.Errorf("updating issue: %w", err)
			}

			fmt.Fprintf(f.IOStreams.Out, "Updated issue #%d\n%s\n", issue.IID, issue.WebURL)
			return nil
		},
	}

	cmd.Flags().StringVarP(&title, "title", "t", "", "New title")
	cmd.Flags().StringVarP(&description, "description", "d", "", "New description")
	cmd.Flags().StringSliceVarP(&assignees, "assignee", "a", nil, "Assignees")
	cmd.Flags().StringSliceVarP(&labels, "label", "l", nil, "Labels")
	cmd.Flags().StringVarP(&milestone, "milestone", "m", "", "Milestone ID")
	cmd.Flags().BoolVar(&confidential, "confidential", false, "Mark as confidential")
	cmd.Flags().Int64Var(&weight, "weight", 0, "Issue weight")

	return cmd
}

func newIssueDeleteCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "delete [<id>]",
		Short:   "Delete an issue",
		Example: `  $ glab issue delete 42`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := f.Client()
			if err != nil {
				return err
			}

			project, err := f.FullProjectPath()
			if err != nil {
				return err
			}

			issueID, err := parseIssueArg(args)
			if err != nil {
				return err
			}

			_, err = client.Issues.DeleteIssue(project, issueID)
			if err != nil {
				return fmt.Errorf("deleting issue: %w", err)
			}

			fmt.Fprintf(f.IOStreams.Out, "Deleted issue #%d\n", issueID)
			return nil
		},
	}

	return cmd
}

func parseIssueArg(args []string) (int64, error) {
	if len(args) == 0 {
		return 0, fmt.Errorf("issue ID required")
	}
	id := strings.TrimPrefix(args[0], "#")
	n, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid issue ID: %s", args[0])
	}
	return n, nil
}
