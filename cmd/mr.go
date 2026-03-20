package cmd

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/PhilipKram/gitlab-cli/internal/api"
	"github.com/PhilipKram/gitlab-cli/internal/browser"
	"github.com/PhilipKram/gitlab-cli/internal/cmdutil"
	"github.com/PhilipKram/gitlab-cli/internal/errors"
	"github.com/PhilipKram/gitlab-cli/internal/formatter"
	gitutil "github.com/PhilipKram/gitlab-cli/internal/git"
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
	cmd.AddCommand(newMRSuggestCmd(f))
	cmd.AddCommand(newMRReplyCmd(f))
	cmd.AddCommand(newMRResolveCmd(f))
	cmd.AddCommand(newMRUnresolveCmd(f))
	cmd.AddCommand(newMREditCmd(f))
	cmd.AddCommand(newMRDiscussionsCmd(f))

	return cmd
}

func newMRCreateCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		title        string
		description  string
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

			mr, resp, err := client.MergeRequests.CreateMergeRequest(project, opts)
			if err != nil {
				statusCode := 0
				if resp != nil {
					statusCode = resp.StatusCode
				}
				url := api.APIURL(client.Host()) + "/projects/" + project + "/merge_requests"
				return errors.NewAPIError("POST", url, statusCode, "Failed to create merge request", err)
			}

			out := f.IOStreams.Out
			_, _ = fmt.Fprintf(out, "Created merge request !%d\n", mr.IID)
			_, _ = fmt.Fprintf(out, "%s\n", mr.WebURL)

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
		format    string
		web       bool
		stream    bool
	)

	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List merge requests",
		Long:    "List merge requests in the current project.",
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

			outputFormat, err := f.ResolveFormat(format, jsonFlag)
			if err != nil {
				return err
			}

			// Use streaming mode if --stream flag is set
			if stream {
				ctx := context.Background()
				fetchFunc := func(page int) ([]*gitlab.BasicMergeRequest, *gitlab.Response, error) {
					pageOpts := *opts
					pageOpts.Page = int64(page)
					if pageOpts.PerPage == 0 {
						pageOpts.PerPage = 100
					}
					return client.MergeRequests.ListProjectMergeRequests(project, &pageOpts)
				}

				paginateOpts := api.PaginateOptions{
					PerPage:    int(opts.PerPage),
					BufferSize: 100,
				}
				if limit > 0 && limit < 100 {
					paginateOpts.PerPage = limit
					paginateOpts.BufferSize = limit
				}

				results := api.PaginateToChannel(ctx, fetchFunc, paginateOpts)
				return cmdutil.FormatAndStream(f, results, outputFormat, limit, "merge requests")
			}

			// Non-streaming mode: fetch all at once
			mrs, resp, err := client.MergeRequests.ListProjectMergeRequests(project, opts)
			if err != nil {
				statusCode := 0
				if resp != nil {
					statusCode = resp.StatusCode
				}
				url := api.APIURL(client.Host()) + "/projects/" + project + "/merge_requests"
				return errors.NewAPIError("GET", url, statusCode, "Failed to list merge requests", err)
			}

			if len(mrs) == 0 {
				_, _ = fmt.Fprintln(f.IOStreams.ErrOut, "No merge requests match your search. Try adjusting filters (--state, --author, --label) or increase --limit.")
				return nil
			}

			return f.FormatAndPrint(mrs, string(outputFormat), false)
		},
	}

	cmd.Flags().StringVar(&state, "state", "opened", "Filter by state: opened, closed, merged, all")
	cmd.Flags().StringVar(&author, "author", "", "Filter by author username")
	cmd.Flags().StringVar(&assignee, "assignee", "", "Filter by assignee username")
	cmd.Flags().StringSliceVarP(&labels, "label", "l", nil, "Filter by labels")
	cmd.Flags().StringVarP(&milestone, "milestone", "m", "", "Filter by milestone")
	cmd.Flags().StringVar(&search, "search", "", "Search in title and description")
	cmd.Flags().IntVarP(&limit, "limit", "L", 30, "Maximum number of results")
	cmd.Flags().BoolVar(&jsonFlag, "json", false, "Output as JSON (deprecated: use --format=json)")
	cmd.Flags().StringVarP(&format, "format", "F", "table", "Output format: json, table, or plain")
	cmd.Flags().BoolVarP(&web, "web", "w", false, "Open in browser")
	cmd.Flags().BoolVar(&stream, "stream", false, "Enable streaming mode")

	return cmd
}

func newMRViewCmd(f *cmdutil.Factory) *cobra.Command {
	var web bool
	var format string
	var jsonFlag bool

	cmd := &cobra.Command{
		Use:   "view [<id>]",
		Short: "View a merge request",
		Long:  "Display the details of a merge request.",
		Example: `  $ glab mr view 123
  $ glab mr view 123 --web`,
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

			mr, resp, err := client.MergeRequests.GetMergeRequest(project, mrID, nil)
			if err != nil {
				statusCode := 0
				if resp != nil {
					statusCode = resp.StatusCode
				}
				url := fmt.Sprintf("%s/projects/%s/merge_requests/%d", api.APIURL(client.Host()), project, mrID)
				return errors.NewAPIError("GET", url, statusCode, fmt.Sprintf("Failed to get merge request !%d", mrID), err)
			}

			if web {
				return browser.Open(mr.WebURL)
			}

			// Backward compatibility: --json flag sets format to json
			if jsonFlag {
				_, _ = fmt.Fprintf(f.IOStreams.ErrOut, "Warning: --json is deprecated, use --format=json instead\n")
				format = "json"
			}

			if format != "" && format != "table" {
				return f.FormatAndPrint(mr, format, false)
			}

			// Default custom display
			out := f.IOStreams.Out
			_, _ = fmt.Fprintf(out, "!%d %s\n", mr.IID, mr.Title)
			_, _ = fmt.Fprintf(out, "State:   %s\n", mr.State)
			_, _ = fmt.Fprintf(out, "Author:  %s\n", mr.Author.Username)
			_, _ = fmt.Fprintf(out, "Branch:  %s -> %s\n", mr.SourceBranch, mr.TargetBranch)
			if mr.Assignee != nil {
				_, _ = fmt.Fprintf(out, "Assignee: %s\n", mr.Assignee.Username)
			}
			if len(mr.Reviewers) > 0 {
				var names []string
				for _, r := range mr.Reviewers {
					names = append(names, r.Username)
				}
				_, _ = fmt.Fprintf(out, "Reviewers: %s\n", strings.Join(names, ", "))
			}
			if len(mr.Labels) > 0 {
				_, _ = fmt.Fprintf(out, "Labels:  %s\n", strings.Join(mr.Labels, ", "))
			}
			if mr.Milestone != nil {
				_, _ = fmt.Fprintf(out, "Milestone: %s\n", mr.Milestone.Title)
			}
			_, _ = fmt.Fprintf(out, "Created: %s\n", timeAgo(mr.CreatedAt))
			_, _ = fmt.Fprintf(out, "URL:     %s\n", mr.WebURL)
			if mr.Description != "" {
				_, _ = fmt.Fprintf(out, "\n%s\n", mr.Description)
			}

			return nil
		},
	}

	cmd.Flags().BoolVarP(&web, "web", "w", false, "Open in browser")
	cmd.Flags().StringVarP(&format, "format", "F", "table", "Output format: json, table, or plain")
	cmd.Flags().BoolVar(&jsonFlag, "json", false, "Output as JSON (deprecated: use --format=json)")

	return cmd
}

func newMRMergeCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		squash       bool
		removeSource bool
		message      string
		whenPipeline bool
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
				Squash:                   &squash,
				ShouldRemoveSourceBranch: &removeSource,
			}

			if message != "" {
				opts.MergeCommitMessage = &message
			}

			if whenPipeline {
				autoMerge := true
				opts.AutoMerge = &autoMerge
			}

			mr, resp, err := client.MergeRequests.AcceptMergeRequest(project, mrID, opts)
			if err != nil {
				statusCode := 0
				if resp != nil {
					statusCode = resp.StatusCode
				}
				url := fmt.Sprintf("%s/projects/%s/merge_requests/%d/merge", api.APIURL(client.Host()), project, mrID)
				return errors.NewAPIError("PUT", url, statusCode, fmt.Sprintf("Failed to merge merge request !%d", mrID), err)
			}

			_, _ = fmt.Fprintf(f.IOStreams.Out, "Merged merge request !%d\n", mr.IID)
			return nil
		},
	}

	cmd.Flags().BoolVar(&squash, "squash", false, "Squash commits")
	cmd.Flags().BoolVar(&removeSource, "remove-source-branch", false, "Remove source branch")
	cmd.Flags().StringVar(&message, "message", "", "Custom merge commit message")
	cmd.Flags().BoolVar(&whenPipeline, "when-pipeline-succeeds", false, "Merge automatically when pipeline succeeds")

	return cmd
}

func newMRCloseCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "close [<id>]",
		Short:   "Close a merge request",
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

			mr, resp, err := client.MergeRequests.UpdateMergeRequest(project, mrID, opts)
			if err != nil {
				statusCode := 0
				if resp != nil {
					statusCode = resp.StatusCode
				}
				url := fmt.Sprintf("%s/projects/%s/merge_requests/%d", api.APIURL(client.Host()), project, mrID)
				return errors.NewAPIError("PUT", url, statusCode, fmt.Sprintf("Failed to close merge request !%d", mrID), err)
			}

			_, _ = fmt.Fprintf(f.IOStreams.Out, "Closed merge request !%d\n", mr.IID)
			return nil
		},
	}

	return cmd
}

func newMRReopenCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "reopen [<id>]",
		Short:   "Reopen a merge request",
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

			mr, resp, err := client.MergeRequests.UpdateMergeRequest(project, mrID, opts)
			if err != nil {
				statusCode := 0
				if resp != nil {
					statusCode = resp.StatusCode
				}
				url := fmt.Sprintf("%s/projects/%s/merge_requests/%d", api.APIURL(client.Host()), project, mrID)
				return errors.NewAPIError("PUT", url, statusCode, fmt.Sprintf("Failed to reopen merge request !%d", mrID), err)
			}

			_, _ = fmt.Fprintf(f.IOStreams.Out, "Reopened merge request !%d\n", mr.IID)
			return nil
		},
	}

	return cmd
}

func newMRApproveCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "approve [<id>]",
		Short:   "Approve a merge request",
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

			_, resp, err := client.MergeRequestApprovals.ApproveMergeRequest(project, mrID, nil)
			if err != nil {
				statusCode := 0
				if resp != nil {
					statusCode = resp.StatusCode
				}
				url := fmt.Sprintf("%s/projects/%s/merge_requests/%d/approve", api.APIURL(client.Host()), project, mrID)
				return errors.NewAPIError("POST", url, statusCode, fmt.Sprintf("Failed to approve merge request !%d", mrID), err)
			}

			_, _ = fmt.Fprintf(f.IOStreams.Out, "Approved merge request !%d\n", mrID)
			return nil
		},
	}

	return cmd
}

func newMRCheckoutCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "checkout [<id>]",
		Short:   "Check out a merge request branch locally",
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

			mr, resp, err := client.MergeRequests.GetMergeRequest(project, mrID, nil)
			if err != nil {
				statusCode := 0
				if resp != nil {
					statusCode = resp.StatusCode
				}
				url := fmt.Sprintf("%s/projects/%s/merge_requests/%d", api.APIURL(client.Host()), project, mrID)
				return errors.NewAPIError("GET", url, statusCode, fmt.Sprintf("Failed to get merge request !%d", mrID), err)
			}

			if err := gitutil.CheckoutBranch(mr.SourceBranch); err != nil {
				return fmt.Errorf("checking out branch %s: %w", mr.SourceBranch, err)
			}

			_, _ = fmt.Fprintf(f.IOStreams.Out, "Switched to branch '%s'\n", mr.SourceBranch)
			return nil
		},
	}

	return cmd
}

func newMRDiffCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "diff [<id>]",
		Short:   "View changes in a merge request",
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

			diffs, resp, err := client.MergeRequests.ListMergeRequestDiffs(project, mrID, nil)
			if err != nil {
				statusCode := 0
				if resp != nil {
					statusCode = resp.StatusCode
				}
				url := fmt.Sprintf("%s/projects/%s/merge_requests/%d/diffs", api.APIURL(client.Host()), project, mrID)
				return errors.NewAPIError("GET", url, statusCode, fmt.Sprintf("Failed to get merge request diffs for !%d", mrID), err)
			}

			out := f.IOStreams.Out
			for _, diff := range diffs {
				_, _ = fmt.Fprintf(out, "--- a/%s\n+++ b/%s\n", diff.OldPath, diff.NewPath)
				_, _ = fmt.Fprintln(out, diff.Diff)
			}

			return nil
		},
	}

	return cmd
}

func newMRCommentCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		body    string
		file    string
		line    int64
		oldLine int64
		commit  string
	)

	cmd := &cobra.Command{
		Use:   "comment [<id>]",
		Short: "Add a comment to a merge request",
		Long: `Add a comment to a merge request.

Without --file, adds a regular comment. With --file and --line, adds an
inline diff comment on the specified file and line.`,
		Example: `  $ glab mr comment 123 --body "Looks good!"
  $ glab mr comment 123 --body "Consider refactoring this" --file "cmd/mr.go" --line 42
  $ glab mr comment 123 --body "Good that this was removed" --file "cmd/mr.go" --old-line 10`,
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

			// Inline diff comment when --file is provided
			if cmd.Flags().Changed("file") {
				if !cmd.Flags().Changed("line") && !cmd.Flags().Changed("old-line") {
					return fmt.Errorf("--file requires at least one of --line or --old-line")
				}

				mr, resp, err := client.MergeRequests.GetMergeRequest(project, mrID, nil)
				if err != nil {
					statusCode := 0
					if resp != nil {
						statusCode = resp.StatusCode
					}
					url := fmt.Sprintf("%s/projects/%s/merge_requests/%d", api.APIURL(client.Host()), project, mrID)
					return errors.NewAPIError("GET", url, statusCode, fmt.Sprintf("Failed to get merge request !%d", mrID), err)
				}

				posType := "text"
				position := &gitlab.PositionOptions{
					BaseSHA:      &mr.DiffRefs.BaseSha,
					HeadSHA:      &mr.DiffRefs.HeadSha,
					StartSHA:     &mr.DiffRefs.StartSha,
					NewPath:      &file,
					OldPath:      &file,
					PositionType: &posType,
				}

				if cmd.Flags().Changed("line") {
					position.NewLine = &line
				}
				if cmd.Flags().Changed("old-line") {
					position.OldLine = &oldLine
				}

				opts := &gitlab.CreateMergeRequestDiscussionOptions{
					Body:     &body,
					Position: position,
				}

				if commit != "" {
					opts.CommitID = &commit
				}

				discussion, resp, err := client.Discussions.CreateMergeRequestDiscussion(project, mrID, opts)
				if err != nil {
					statusCode := 0
					if resp != nil {
						statusCode = resp.StatusCode
					}
					url := fmt.Sprintf("%s/projects/%s/merge_requests/%d/discussions", api.APIURL(client.Host()), project, mrID)
					return errors.NewAPIError("POST", url, statusCode, fmt.Sprintf("Failed to add inline comment to merge request !%d", mrID), err)
				}

				_, _ = fmt.Fprintf(f.IOStreams.Out, "Added inline comment to !%d on %s\n%s\n", mrID, file, discussion.Notes[0].Body)
				return nil
			}

			// Regular comment
			opts := &gitlab.CreateMergeRequestNoteOptions{
				Body: &body,
			}

			note, resp, err := client.Notes.CreateMergeRequestNote(project, mrID, opts)
			if err != nil {
				statusCode := 0
				if resp != nil {
					statusCode = resp.StatusCode
				}
				url := fmt.Sprintf("%s/projects/%s/merge_requests/%d/notes", api.APIURL(client.Host()), project, mrID)
				return errors.NewAPIError("POST", url, statusCode, fmt.Sprintf("Failed to add comment to merge request !%d", mrID), err)
			}

			_, _ = fmt.Fprintf(f.IOStreams.Out, "Added comment to !%d\n%s\n", mrID, note.Body)
			return nil
		},
	}

	cmd.Flags().StringVarP(&body, "body", "b", "", "Comment body (required)")
	cmd.Flags().StringVarP(&file, "file", "f", "", "File path in the diff for inline comment")
	cmd.Flags().Int64VarP(&line, "line", "l", 0, "Line number in the new version of the file")
	cmd.Flags().Int64Var(&oldLine, "old-line", 0, "Line number in the old version of the file")
	cmd.Flags().StringVar(&commit, "commit", "", "Specific commit SHA to comment on")
	_ = cmd.MarkFlagRequired("body")

	return cmd
}

func newMRSuggestCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		body string
		file string
		line int64
	)

	cmd := &cobra.Command{
		Use:   "suggest [<id>]",
		Short: "Add a suggestion comment to a merge request",
		Long: `Add a code suggestion comment to a merge request.

Creates an inline diff comment with a GitLab suggestion that can be applied
directly from the merge request interface.`,
		Example: `  $ glab mr suggest 123 --file "cmd/mr.go" --line 42 --body "newVariable"
  $ glab mr suggest --file "main.go" --line 10 --body "// Add comment"`,
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

			mr, resp, err := client.MergeRequests.GetMergeRequest(project, mrID, nil)
			if err != nil {
				statusCode := 0
				if resp != nil {
					statusCode = resp.StatusCode
				}
				url := fmt.Sprintf("%s/projects/%s/merge_requests/%d", api.APIURL(client.Host()), project, mrID)
				return errors.NewAPIError("GET", url, statusCode, fmt.Sprintf("Failed to get merge request !%d", mrID), err)
			}

			posType := "text"
			position := &gitlab.PositionOptions{
				BaseSHA:      &mr.DiffRefs.BaseSha,
				HeadSHA:      &mr.DiffRefs.HeadSha,
				StartSHA:     &mr.DiffRefs.StartSha,
				NewPath:      &file,
				OldPath:      &file,
				PositionType: &posType,
				NewLine:      &line,
			}

			// Format body as GitLab suggestion
			suggestionBody := fmt.Sprintf("```suggestion\n%s\n```", body)

			opts := &gitlab.CreateMergeRequestDiscussionOptions{
				Body:     &suggestionBody,
				Position: position,
			}

			discussion, resp, err := client.Discussions.CreateMergeRequestDiscussion(project, mrID, opts)
			if err != nil {
				statusCode := 0
				if resp != nil {
					statusCode = resp.StatusCode
				}
				url := fmt.Sprintf("%s/projects/%s/merge_requests/%d/discussions", api.APIURL(client.Host()), project, mrID)
				return errors.NewAPIError("POST", url, statusCode, fmt.Sprintf("Failed to add suggestion to merge request !%d", mrID), err)
			}

			_, _ = fmt.Fprintf(f.IOStreams.Out, "Added suggestion to !%d on %s:%d\n", mrID, file, line)
			_, _ = fmt.Fprintf(f.IOStreams.Out, "%s\n", discussion.Notes[0].Body)
			return nil
		},
	}

	cmd.Flags().StringVarP(&body, "body", "b", "", "Suggested code change (required)")
	cmd.Flags().StringVarP(&file, "file", "f", "", "File path in the diff (required)")
	cmd.Flags().Int64VarP(&line, "line", "l", 0, "Line number in the new version of the file (required)")
	_ = cmd.MarkFlagRequired("body")
	_ = cmd.MarkFlagRequired("file")
	_ = cmd.MarkFlagRequired("line")

	return cmd
}

func newMRReplyCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		body         string
		discussionID string
	)

	cmd := &cobra.Command{
		Use:   "reply [<id>]",
		Short: "Reply to a discussion thread on a merge request",
		Long:  "Reply to a specific discussion thread on a merge request.",
		Example: `  $ glab mr reply 123 --discussion abc123 --body "Thanks for the review!"
  $ glab mr reply 123 --discussion xyz789 --body "Fixed in the latest commit"`,
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

			opts := &gitlab.AddMergeRequestDiscussionNoteOptions{
				Body: &body,
			}

			note, resp, err := client.Discussions.AddMergeRequestDiscussionNote(project, mrID, discussionID, opts)
			if err != nil {
				statusCode := 0
				if resp != nil {
					statusCode = resp.StatusCode
				}
				url := fmt.Sprintf("%s/projects/%s/merge_requests/%d/discussions/%s/notes", api.APIURL(client.Host()), project, mrID, discussionID)
				return errors.NewAPIError("POST", url, statusCode, fmt.Sprintf("Failed to reply to discussion on merge request !%d", mrID), err)
			}

			_, _ = fmt.Fprintf(f.IOStreams.Out, "Replied to discussion on !%d\n%s\n", mrID, note.Body)
			return nil
		},
	}

	cmd.Flags().StringVarP(&body, "body", "b", "", "Reply body (required)")
	cmd.Flags().StringVarP(&discussionID, "discussion", "d", "", "Discussion ID to reply to (required)")
	_ = cmd.MarkFlagRequired("body")
	_ = cmd.MarkFlagRequired("discussion")

	return cmd
}

func newMRResolveCmd(f *cmdutil.Factory) *cobra.Command {
	var discussionID string

	cmd := &cobra.Command{
		Use:     "resolve [<id>]",
		Short:   "Resolve a discussion thread on a merge request",
		Long:    "Resolve a specific discussion thread on a merge request.",
		Example: `  $ glab mr resolve 123 --discussion abc123`,
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

			resolved := true
			opts := &gitlab.ResolveMergeRequestDiscussionOptions{
				Resolved: &resolved,
			}

			discussion, resp, err := client.Discussions.ResolveMergeRequestDiscussion(project, mrID, discussionID, opts)
			if err != nil {
				statusCode := 0
				if resp != nil {
					statusCode = resp.StatusCode
				}
				url := fmt.Sprintf("%s/projects/%s/merge_requests/%d/discussions/%s", api.APIURL(client.Host()), project, mrID, discussionID)
				return errors.NewAPIError("PUT", url, statusCode, fmt.Sprintf("Failed to resolve discussion on merge request !%d", mrID), err)
			}

			_, _ = fmt.Fprintf(f.IOStreams.Out, "Resolved discussion %s on !%d\n", discussion.ID, mrID)
			return nil
		},
	}

	cmd.Flags().StringVarP(&discussionID, "discussion", "d", "", "Discussion ID to resolve (required)")
	_ = cmd.MarkFlagRequired("discussion")

	return cmd
}

func newMRUnresolveCmd(f *cmdutil.Factory) *cobra.Command {
	var discussionID string

	cmd := &cobra.Command{
		Use:     "unresolve [<id>]",
		Short:   "Unresolve a discussion thread on a merge request",
		Long:    "Unresolve a specific discussion thread on a merge request.",
		Example: `  $ glab mr unresolve 123 --discussion abc123`,
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

			resolved := false
			opts := &gitlab.ResolveMergeRequestDiscussionOptions{
				Resolved: &resolved,
			}

			discussion, resp, err := client.Discussions.ResolveMergeRequestDiscussion(project, mrID, discussionID, opts)
			if err != nil {
				statusCode := 0
				if resp != nil {
					statusCode = resp.StatusCode
				}
				url := fmt.Sprintf("%s/projects/%s/merge_requests/%d/discussions/%s", api.APIURL(client.Host()), project, mrID, discussionID)
				return errors.NewAPIError("PUT", url, statusCode, fmt.Sprintf("Failed to unresolve discussion on merge request !%d", mrID), err)
			}

			_, _ = fmt.Fprintf(f.IOStreams.Out, "Unresolved discussion %s on !%d\n", discussion.ID, mrID)
			return nil
		},
	}

	cmd.Flags().StringVarP(&discussionID, "discussion", "d", "", "Discussion ID to unresolve (required)")
	_ = cmd.MarkFlagRequired("discussion")

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

			mr, resp, err := client.MergeRequests.UpdateMergeRequest(project, mrID, opts)
			if err != nil {
				statusCode := 0
				if resp != nil {
					statusCode = resp.StatusCode
				}
				url := fmt.Sprintf("%s/projects/%s/merge_requests/%d", api.APIURL(client.Host()), project, mrID)
				return errors.NewAPIError("PUT", url, statusCode, fmt.Sprintf("Failed to update merge request !%d", mrID), err)
			}

			_, _ = fmt.Fprintf(f.IOStreams.Out, "Updated merge request !%d\n%s\n", mr.IID, mr.WebURL)
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

func newMRDiscussionsCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		format   string
		jsonFlag bool
	)

	cmd := &cobra.Command{
		Use:   "discussions [<id>]",
		Short: "List discussion threads on a merge request",
		Long:  "List all discussion threads on a merge request, showing their status and content.",
		Example: `  $ glab mr discussions 123
  $ glab mr discussions 123 --format json
  $ glab mr discussions 123 --format plain`,
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

			discussions, resp, err := client.Discussions.ListMergeRequestDiscussions(project, mrID, nil)
			if err != nil {
				statusCode := 0
				if resp != nil {
					statusCode = resp.StatusCode
				}
				url := fmt.Sprintf("%s/projects/%s/merge_requests/%d/discussions", api.APIURL(client.Host()), project, mrID)
				return errors.NewAPIError("GET", url, statusCode, fmt.Sprintf("Failed to list discussions for merge request !%d", mrID), err)
			}

			if len(discussions) == 0 {
				_, _ = fmt.Fprintln(f.IOStreams.ErrOut, "No discussions found on this merge request")
				return nil
			}

			outputFormat, err := f.ResolveFormat(format, jsonFlag)
			if err != nil {
				return err
			}

			if outputFormat == formatter.JSONFormat {
				return f.FormatAndPrint(discussions, format, jsonFlag)
			}

			return printDiscussions(f.IOStreams.Out, discussions)
		},
	}

	cmd.Flags().StringVarP(&format, "format", "F", "table", "Output format: json, table, or plain")
	cmd.Flags().BoolVar(&jsonFlag, "json", false, "Output as JSON (deprecated: use --format=json)")

	return cmd
}

// printDiscussions renders discussions in a human-readable format showing
// inline comment context (file, line), author, resolution status, and threaded replies.
func printDiscussions(out io.Writer, discussions []*gitlab.Discussion) error {
	idx := 0
	for _, d := range discussions {
		if len(d.Notes) == 0 {
			continue
		}

		// Skip system-only discussions
		firstNote := d.Notes[0]
		if firstNote.System {
			continue
		}

		idx++
		if idx > 1 {
			_, _ = fmt.Fprintln(out)
		}

		// Header: #N (status) author [on file:line]
		var header strings.Builder
		_, _ = fmt.Fprintf(&header, "#%d", idx)
		if firstNote.Resolvable {
			if firstNote.Resolved {
				header.WriteString(" (resolved)")
			} else {
				header.WriteString(" (unresolved)")
			}
		}
		header.WriteString(" ")
		header.WriteString(firstNote.Author.Username)
		if pos := firstNote.Position; pos != nil {
			path := pos.NewPath
			line := pos.NewLine
			if path == "" {
				path = pos.OldPath
			}
			if line == 0 {
				line = pos.OldLine
			}
			if path != "" {
				_, _ = fmt.Fprintf(&header, " on %s", path)
				if line != 0 {
					_, _ = fmt.Fprintf(&header, ":%d", line)
				}
			}
		}
		_, _ = fmt.Fprintln(out, header.String())

		// Body of the first note
		for _, line := range strings.Split(strings.TrimRight(firstNote.Body, "\n"), "\n") {
			_, _ = fmt.Fprintf(out, "  %s\n", line)
		}

		// Threaded replies
		for _, reply := range d.Notes[1:] {
			if reply.System {
				continue
			}
			_, _ = fmt.Fprintf(out, "  └─ %s: ", reply.Author.Username)
			lines := strings.Split(strings.TrimRight(reply.Body, "\n"), "\n")
			_, _ = fmt.Fprintln(out, lines[0])
			for _, l := range lines[1:] {
				_, _ = fmt.Fprintf(out, "     %s\n", l)
			}
		}
	}
	return nil
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
