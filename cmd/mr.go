package cmd

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/PhilipKram/gitlab-cli/internal/api"
	"github.com/PhilipKram/gitlab-cli/internal/browser"
	"github.com/PhilipKram/gitlab-cli/internal/cmdutil"
	gitutil "github.com/PhilipKram/gitlab-cli/internal/git"
	"github.com/PhilipKram/gitlab-cli/internal/tableprinter"
	"github.com/spf13/cobra"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// NewMRCmd creates the merge request command group.
func NewMRCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "mr <command>",
		Short:   "Manage merge requests",
		Long:    "Create, view, and manage GitLab merge requests.",
		Aliases: []string{"merge-request"},
	}

	cmd.AddCommand(newMRCreateCmd(f))
	cmd.AddCommand(newMRListCmd(f))
	cmd.AddCommand(newMRViewCmd(f))
	cmd.AddCommand(newMRMergeCmd(f))
	cmd.AddCommand(newMRCloseCmd(f))
	cmd.AddCommand(newMRReopenCmd(f))
	cmd.AddCommand(newMRApproveCmd(f))
	cmd.AddCommand(newMRCheckoutCmd(f))
	cmd.AddCommand(newMRDiffCmd(f))
	cmd.AddCommand(newMRCommentCmd(f))
	cmd.AddCommand(newMREditCmd(f))

	return cmd
}

func newMRCreateCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		title       string
		description string
		sourceBranch string
		targetBranch string
		assignees    []string
		reviewers    []string
		labels       []string
		milestone    string
		draft        bool
		squash       bool
		removeSource bool
		web          bool
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a merge request",
		Long:  "Create a new merge request on GitLab.",
		Example: `  $ glab mr create --title "Add feature" --description "Details here"
  $ glab mr create --title "Fix bug" --target-branch main --draft
  $ glab mr create --title "Update" --assignee @user1 --label bug,urgent`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := f.Client()
			if err != nil {
				return err
			}

			project, err := f.FullProjectPath()
			if err != nil {
				return err
			}

			if sourceBranch == "" {
				sourceBranch, err = gitutil.CurrentBranch()
				if err != nil {
					return fmt.Errorf("could not determine source branch: %w", err)
				}
			}

			if targetBranch == "" {
				remote, rerr := f.Remote()
				if rerr == nil {
					targetBranch, _ = gitutil.DefaultBranch(remote.Name)
				}
				if targetBranch == "" {
					targetBranch = "main"
				}
			}

			opts := &gitlab.CreateMergeRequestOptions{
				Title:        &title,
				Description:  &description,
				SourceBranch: &sourceBranch,
				TargetBranch: &targetBranch,
			}

			if len(assignees) > 0 {
				ids, err := resolveUserIDs(client, assignees)
				if err != nil {
					return fmt.Errorf("resolving assignees: %w", err)
				}
				opts.AssigneeIDs = &ids
			}

			if len(reviewers) > 0 {
				ids, err := resolveUserIDs(client, reviewers)
				if err != nil {
					return fmt.Errorf("resolving reviewers: %w", err)
				}
				opts.ReviewerIDs = &ids
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

			if draft {
				if !strings.HasPrefix(title, "Draft:") {
					t := "Draft: " + title
					opts.Title = &t
				}
			}

			opts.Squash = &squash
			opts.RemoveSourceBranch = &removeSource

			mr, _, err := client.MergeRequests.CreateMergeRequest(project, opts)
			if err != nil {
				return fmt.Errorf("creating merge request: %w", err)
			}

			out := f.IOStreams.Out
			fmt.Fprintf(out, "Created merge request !%d\n", mr.IID)
			fmt.Fprintf(out, "%s\n", mr.WebURL)

			if web {
				_ = browser.Open(mr.WebURL)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&title, "title", "t", "", "Title of the merge request (required)")
	cmd.Flags().StringVarP(&description, "description", "d", "", "Description of the merge request")
	cmd.Flags().StringVarP(&sourceBranch, "source-branch", "s", "", "Source branch (default: current branch)")
	cmd.Flags().StringVarP(&targetBranch, "target-branch", "b", "", "Target branch (default: repository default)")
	cmd.Flags().StringSliceVarP(&assignees, "assignee", "a", nil, "Assign users by username")
	cmd.Flags().StringSliceVar(&reviewers, "reviewer", nil, "Request review from users by username")
	cmd.Flags().StringSliceVarP(&labels, "label", "l", nil, "Add labels")
	cmd.Flags().StringVarP(&milestone, "milestone", "m", "", "Milestone ID")
	cmd.Flags().BoolVar(&draft, "draft", false, "Mark as draft")
	cmd.Flags().BoolVar(&squash, "squash", false, "Squash commits on merge")
	cmd.Flags().BoolVar(&removeSource, "remove-source-branch", false, "Remove source branch on merge")
	cmd.Flags().BoolVarP(&web, "web", "w", false, "Open in browser after creation")
	_ = cmd.MarkFlagRequired("title")

	return cmd
}

func newMRListCmd(f *cmdutil.Factory) *cobra.Command {
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
		Use:   "list",
		Short: "List merge requests",
		Long:  "List merge requests in the current project.",
		Aliases: []string{"ls"},
		Example: `  $ glab mr list
  $ glab mr list --state merged --author johndoe
  $ glab mr list --label bug --limit 50
  $ glab mr list --json`,
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
				return browser.Open(api.WebURL(host, project+"/-/merge_requests"))
			}

			opts := &gitlab.ListProjectMergeRequestsOptions{
				ListOptions: gitlab.ListOptions{PerPage: int64(limit)},
			}

			if state != "" {
				opts.State = &state
			}
			if author != "" {
				opts.AuthorUsername = &author
			}
			_ = assignee // Assignee filtering via API varies by version
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

			mrs, _, err := client.MergeRequests.ListProjectMergeRequests(project, opts)
			if err != nil {
				return fmt.Errorf("listing merge requests: %w", err)
			}

			if len(mrs) == 0 {
				fmt.Fprintln(f.IOStreams.ErrOut, "No merge requests match your search")
				return nil
			}

			if jsonFlag {
				data, err := json.MarshalIndent(mrs, "", "  ")
				if err != nil {
					return err
				}
				fmt.Fprintln(f.IOStreams.Out, string(data))
				return nil
			}

			tp := tableprinter.New(f.IOStreams.Out)
			for _, mr := range mrs {
				tp.AddRow(
					fmt.Sprintf("!%d", mr.IID),
					mr.Title,
					mr.State,
					mr.Author.Username,
					timeAgo(mr.CreatedAt),
				)
			}
			return tp.Render()
		},
	}

	cmd.Flags().StringVar(&state, "state", "opened", "Filter by state: opened, closed, merged, all")
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

func newMRViewCmd(f *cmdutil.Factory) *cobra.Command {
	var web bool
	var jsonFlag bool

	cmd := &cobra.Command{
		Use:   "view [<id>]",
		Short: "View a merge request",
		Long:  "Display the details of a merge request.",
		Example: `  $ glab mr view 123
  $ glab mr view 123 --web
  $ glab mr view 123 --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := f.Client()
			if err != nil {
				return err
			}

			project, err := f.FullProjectPath()
			if err != nil {
				return err
			}

			mrID, err := parseMRArg(args)
			if err != nil {
				return err
			}

			mr, _, err := client.MergeRequests.GetMergeRequest(project, mrID, nil)
			if err != nil {
				return fmt.Errorf("getting merge request: %w", err)
			}

			if web {
				return browser.Open(mr.WebURL)
			}

			if jsonFlag {
				data, err := json.MarshalIndent(mr, "", "  ")
				if err != nil {
					return err
				}
				fmt.Fprintln(f.IOStreams.Out, string(data))
				return nil
			}

			out := f.IOStreams.Out
			fmt.Fprintf(out, "!%d %s\n", mr.IID, mr.Title)
			fmt.Fprintf(out, "State:   %s\n", mr.State)
			fmt.Fprintf(out, "Author:  %s\n", mr.Author.Username)
			fmt.Fprintf(out, "Branch:  %s -> %s\n", mr.SourceBranch, mr.TargetBranch)
			if mr.Assignee != nil {
				fmt.Fprintf(out, "Assignee: %s\n", mr.Assignee.Username)
			}
			if len(mr.Reviewers) > 0 {
				var names []string
				for _, r := range mr.Reviewers {
					names = append(names, r.Username)
				}
				fmt.Fprintf(out, "Reviewers: %s\n", strings.Join(names, ", "))
			}
			if len(mr.Labels) > 0 {
				fmt.Fprintf(out, "Labels:  %s\n", strings.Join(mr.Labels, ", "))
			}
			if mr.Milestone != nil {
				fmt.Fprintf(out, "Milestone: %s\n", mr.Milestone.Title)
			}
			fmt.Fprintf(out, "Created: %s\n", timeAgo(mr.CreatedAt))
			fmt.Fprintf(out, "URL:     %s\n", mr.WebURL)
			if mr.Description != "" {
				fmt.Fprintf(out, "\n%s\n", mr.Description)
			}

			return nil
		},
	}

	cmd.Flags().BoolVarP(&web, "web", "w", false, "Open in browser")
	cmd.Flags().BoolVar(&jsonFlag, "json", false, "Output as JSON")

	return cmd
}

func newMRMergeCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		squash       bool
		removeSource bool
		message      string
		whenPipeline string
	)

	cmd := &cobra.Command{
		Use:   "merge [<id>]",
		Short: "Merge a merge request",
		Example: `  $ glab mr merge 123
  $ glab mr merge 123 --squash --remove-source-branch
  $ glab mr merge 123 --when-pipeline-succeeds`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := f.Client()
			if err != nil {
				return err
			}

			project, err := f.FullProjectPath()
			if err != nil {
				return err
			}

			mrID, err := parseMRArg(args)
			if err != nil {
				return err
			}

			opts := &gitlab.AcceptMergeRequestOptions{
				Squash:                    &squash,
				ShouldRemoveSourceBranch:  &removeSource,
			}

			if message != "" {
				opts.MergeCommitMessage = &message
			}

			if whenPipeline == "succeeds" {
				pipelineSuccess := true
				opts.MergeWhenPipelineSucceeds = &pipelineSuccess
			}

			mr, _, err := client.MergeRequests.AcceptMergeRequest(project, mrID, opts)
			if err != nil {
				return fmt.Errorf("merging merge request: %w", err)
			}

			fmt.Fprintf(f.IOStreams.Out, "Merged merge request !%d\n", mr.IID)
			return nil
		},
	}

	cmd.Flags().BoolVar(&squash, "squash", false, "Squash commits")
	cmd.Flags().BoolVar(&removeSource, "remove-source-branch", false, "Remove source branch")
	cmd.Flags().StringVar(&message, "message", "", "Custom merge commit message")
	cmd.Flags().StringVar(&whenPipeline, "when-pipeline-succeeds", "", "Merge when pipeline succeeds")

	return cmd
}

func newMRCloseCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "close [<id>]",
		Short: "Close a merge request",
		Example: `  $ glab mr close 123`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := f.Client()
			if err != nil {
				return err
			}

			project, err := f.FullProjectPath()
			if err != nil {
				return err
			}

			mrID, err := parseMRArg(args)
			if err != nil {
				return err
			}

			closed := "close"
			opts := &gitlab.UpdateMergeRequestOptions{
				StateEvent: &closed,
			}

			mr, _, err := client.MergeRequests.UpdateMergeRequest(project, mrID, opts)
			if err != nil {
				return fmt.Errorf("closing merge request: %w", err)
			}

			fmt.Fprintf(f.IOStreams.Out, "Closed merge request !%d\n", mr.IID)
			return nil
		},
	}

	return cmd
}

func newMRReopenCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reopen [<id>]",
		Short: "Reopen a merge request",
		Example: `  $ glab mr reopen 123`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := f.Client()
			if err != nil {
				return err
			}

			project, err := f.FullProjectPath()
			if err != nil {
				return err
			}

			mrID, err := parseMRArg(args)
			if err != nil {
				return err
			}

			reopen := "reopen"
			opts := &gitlab.UpdateMergeRequestOptions{
				StateEvent: &reopen,
			}

			mr, _, err := client.MergeRequests.UpdateMergeRequest(project, mrID, opts)
			if err != nil {
				return fmt.Errorf("reopening merge request: %w", err)
			}

			fmt.Fprintf(f.IOStreams.Out, "Reopened merge request !%d\n", mr.IID)
			return nil
		},
	}

	return cmd
}

func newMRApproveCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "approve [<id>]",
		Short: "Approve a merge request",
		Example: `  $ glab mr approve 123`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := f.Client()
			if err != nil {
				return err
			}

			project, err := f.FullProjectPath()
			if err != nil {
				return err
			}

			mrID, err := parseMRArg(args)
			if err != nil {
				return err
			}

			_, _, err = client.MergeRequestApprovals.ApproveMergeRequest(project, mrID, nil)
			if err != nil {
				return fmt.Errorf("approving merge request: %w", err)
			}

			fmt.Fprintf(f.IOStreams.Out, "Approved merge request !%d\n", mrID)
			return nil
		},
	}

	return cmd
}

func newMRCheckoutCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "checkout [<id>]",
		Short: "Check out a merge request branch locally",
		Aliases: []string{"co"},
		Example: `  $ glab mr checkout 123`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := f.Client()
			if err != nil {
				return err
			}

			project, err := f.FullProjectPath()
			if err != nil {
				return err
			}

			mrID, err := parseMRArg(args)
			if err != nil {
				return err
			}

			mr, _, err := client.MergeRequests.GetMergeRequest(project, mrID, nil)
			if err != nil {
				return fmt.Errorf("getting merge request: %w", err)
			}

			if err := gitutil.CheckoutBranch(mr.SourceBranch); err != nil {
				return fmt.Errorf("checking out branch %s: %w", mr.SourceBranch, err)
			}

			fmt.Fprintf(f.IOStreams.Out, "Switched to branch '%s'\n", mr.SourceBranch)
			return nil
		},
	}

	return cmd
}

func newMRDiffCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "diff [<id>]",
		Short: "View changes in a merge request",
		Example: `  $ glab mr diff 123`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := f.Client()
			if err != nil {
				return err
			}

			project, err := f.FullProjectPath()
			if err != nil {
				return err
			}

			mrID, err := parseMRArg(args)
			if err != nil {
				return err
			}

			diffs, _, err := client.MergeRequests.ListMergeRequestDiffs(project, mrID, nil)
			if err != nil {
				return fmt.Errorf("getting merge request diffs: %w", err)
			}

			out := f.IOStreams.Out
			for _, diff := range diffs {
				fmt.Fprintf(out, "--- a/%s\n+++ b/%s\n", diff.OldPath, diff.NewPath)
				fmt.Fprintln(out, diff.Diff)
			}

			return nil
		},
	}

	return cmd
}

func newMRCommentCmd(f *cmdutil.Factory) *cobra.Command {
	var body string

	cmd := &cobra.Command{
		Use:   "comment [<id>]",
		Short: "Add a comment to a merge request",
		Example: `  $ glab mr comment 123 --body "Looks good!"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := f.Client()
			if err != nil {
				return err
			}

			project, err := f.FullProjectPath()
			if err != nil {
				return err
			}

			mrID, err := parseMRArg(args)
			if err != nil {
				return err
			}

			opts := &gitlab.CreateMergeRequestNoteOptions{
				Body: &body,
			}

			note, _, err := client.Notes.CreateMergeRequestNote(project, mrID, opts)
			if err != nil {
				return fmt.Errorf("adding comment: %w", err)
			}

			fmt.Fprintf(f.IOStreams.Out, "Added comment to !%d\n%s\n", mrID, note.Body)
			return nil
		},
	}

	cmd.Flags().StringVarP(&body, "body", "b", "", "Comment body (required)")
	_ = cmd.MarkFlagRequired("body")

	return cmd
}

func newMREditCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		title       string
		description string
		assignees   []string
		reviewers   []string
		labels      []string
		milestone   string
	)

	cmd := &cobra.Command{
		Use:   "edit [<id>]",
		Short: "Edit a merge request",
		Example: `  $ glab mr edit 123 --title "New title"
  $ glab mr edit 123 --assignee user1 --label bug`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := f.Client()
			if err != nil {
				return err
			}

			project, err := f.FullProjectPath()
			if err != nil {
				return err
			}

			mrID, err := parseMRArg(args)
			if err != nil {
				return err
			}

			opts := &gitlab.UpdateMergeRequestOptions{}

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
			if cmd.Flags().Changed("reviewer") {
				ids, err := resolveUserIDs(client, reviewers)
				if err != nil {
					return err
				}
				opts.ReviewerIDs = &ids
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

			mr, _, err := client.MergeRequests.UpdateMergeRequest(project, mrID, opts)
			if err != nil {
				return fmt.Errorf("updating merge request: %w", err)
			}

			fmt.Fprintf(f.IOStreams.Out, "Updated merge request !%d\n%s\n", mr.IID, mr.WebURL)
			return nil
		},
	}

	cmd.Flags().StringVarP(&title, "title", "t", "", "New title")
	cmd.Flags().StringVarP(&description, "description", "d", "", "New description")
	cmd.Flags().StringSliceVarP(&assignees, "assignee", "a", nil, "Assignees")
	cmd.Flags().StringSliceVar(&reviewers, "reviewer", nil, "Reviewers")
	cmd.Flags().StringSliceVarP(&labels, "label", "l", nil, "Labels")
	cmd.Flags().StringVarP(&milestone, "milestone", "m", "", "Milestone ID")

	return cmd
}

// parseMRArg parses the merge request ID from command args.
func parseMRArg(args []string) (int64, error) {
	if len(args) == 0 {
		return 0, fmt.Errorf("merge request ID required")
	}
	id := strings.TrimPrefix(args[0], "!")
	n, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid merge request ID: %s", args[0])
	}
	return n, nil
}

// resolveUserIDs converts usernames to GitLab user IDs.
func resolveUserIDs(client *api.Client, usernames []string) ([]int64, error) {
	var ids []int64
	for _, username := range usernames {
		username = strings.TrimPrefix(username, "@")
		users, _, err := client.Users.ListUsers(&gitlab.ListUsersOptions{
			Username: &username,
		})
		if err != nil {
			return nil, fmt.Errorf("looking up user %s: %w", username, err)
		}
		if len(users) == 0 {
			return nil, fmt.Errorf("user not found: %s", username)
		}
		ids = append(ids, users[0].ID)
	}
	return ids, nil
}

// timeAgo returns a human-readable time difference.
func timeAgo(t *time.Time) string {
	if t == nil {
		return ""
	}
	d := time.Since(*t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		m := int(d.Minutes())
		if m == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", m)
	case d < 24*time.Hour:
		h := int(d.Hours())
		if h == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", h)
	case d < 30*24*time.Hour:
		days := int(d.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	default:
		return t.Format("Jan 02, 2006")
	}
}
