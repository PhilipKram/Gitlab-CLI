package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/PhilipKram/gitlab-cli/internal/cmdutil"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// RegisterMRTools registers all merge request tools on the server.
func RegisterMRTools(server *mcp.Server, f *cmdutil.Factory) {
	registerMRList(server, f)
	registerMRView(server, f)
	registerMRDiff(server, f)
	registerMRComment(server, f)
	registerMRApprove(server, f)
	registerMRMerge(server, f)
	registerMRClose(server, f)
	registerMRReopen(server, f)
	registerMRCreate(server, f)
	registerMREdit(server, f)
}

func registerMRList(server *mcp.Server, f *cmdutil.Factory) {
	type Input struct {
		Repo      string `json:"repo,omitempty"      jsonschema:"repository in OWNER/REPO or HOST/OWNER/REPO format"`
		State     string `json:"state,omitempty"     jsonschema:"filter by state: opened, closed, merged, all"`
		Author    string `json:"author,omitempty"    jsonschema:"filter by author username"`
		Assignee  string `json:"assignee,omitempty"  jsonschema:"filter by assignee username"`
		Label     string `json:"label,omitempty"     jsonschema:"filter by label name"`
		Milestone string `json:"milestone,omitempty" jsonschema:"filter by milestone title"`
		Limit     int64  `json:"limit,omitempty"     jsonschema:"maximum number of results (default 30)"`
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "mr_list",
		Description: "List merge requests for a GitLab project",
	}, func(_ context.Context, _ *mcp.CallToolRequest, in Input) (*mcp.CallToolResult, any, error) {
		client, project, err := resolveClientAndProject(f, in.Repo)
		if err != nil {
			return nil, nil, err
		}

		opts := &gitlab.ListProjectMergeRequestsOptions{
			ListOptions: gitlab.ListOptions{PerPage: clampPerPage(in.Limit)},
		}
		if in.State != "" {
			opts.State = &in.State
		}
		if in.Author != "" {
			opts.AuthorUsername = &in.Author
		}
		if in.Assignee != "" {
			assigneeID := gitlab.AssigneeID(in.Assignee)
			opts.AssigneeID = assigneeID
		}
		if in.Label != "" {
			labels := gitlab.LabelOptions(strings.Split(in.Label, ","))
			opts.Labels = &labels
		}
		if in.Milestone != "" {
			opts.Milestone = &in.Milestone
		}

		mrs, _, err := client.MergeRequests.ListProjectMergeRequests(project, opts)
		if err != nil {
			return nil, nil, fmt.Errorf("listing merge requests: %w", err)
		}
		return textResult(mrs)
	})
}

func registerMRView(server *mcp.Server, f *cmdutil.Factory) {
	type Input struct {
		MR   int64  `json:"mr"              jsonschema:"merge request IID"`
		Repo string `json:"repo,omitempty"  jsonschema:"repository in OWNER/REPO or HOST/OWNER/REPO format"`
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "mr_view",
		Description: "View details of a specific merge request",
	}, func(_ context.Context, _ *mcp.CallToolRequest, in Input) (*mcp.CallToolResult, any, error) {
		if in.MR <= 0 {
			return nil, nil, fmt.Errorf("mr is required")
		}
		client, project, err := resolveClientAndProject(f, in.Repo)
		if err != nil {
			return nil, nil, err
		}
		mr, _, err := client.MergeRequests.GetMergeRequest(project, in.MR, nil)
		if err != nil {
			return nil, nil, fmt.Errorf("getting merge request: %w", err)
		}
		return textResult(mr)
	})
}

func registerMRDiff(server *mcp.Server, f *cmdutil.Factory) {
	type Input struct {
		MR   int64  `json:"mr"              jsonschema:"merge request IID"`
		Repo string `json:"repo,omitempty"  jsonschema:"repository in OWNER/REPO or HOST/OWNER/REPO format"`
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "mr_diff",
		Description: "Show the diff of a merge request",
	}, func(_ context.Context, _ *mcp.CallToolRequest, in Input) (*mcp.CallToolResult, any, error) {
		if in.MR <= 0 {
			return nil, nil, fmt.Errorf("mr is required")
		}
		client, project, err := resolveClientAndProject(f, in.Repo)
		if err != nil {
			return nil, nil, err
		}
		diffs, _, err := client.MergeRequests.ListMergeRequestDiffs(project, in.MR, nil)
		if err != nil {
			return nil, nil, fmt.Errorf("getting merge request diff: %w", err)
		}
		var sb strings.Builder
		for _, d := range diffs {
			fmt.Fprintf(&sb, "--- a/%s\n+++ b/%s\n%s\n", d.OldPath, d.NewPath, d.Diff)
		}
		return plainResult(sb.String()), nil, nil
	})
}

func registerMRComment(server *mcp.Server, f *cmdutil.Factory) {
	type Input struct {
		MR      int64  `json:"mr"              jsonschema:"merge request IID"`
		Message string `json:"message"         jsonschema:"comment body text"`
		Repo    string `json:"repo,omitempty"  jsonschema:"repository in OWNER/REPO or HOST/OWNER/REPO format"`
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "mr_comment",
		Description: "Add a comment to a merge request",
	}, func(_ context.Context, _ *mcp.CallToolRequest, in Input) (*mcp.CallToolResult, any, error) {
		if in.MR <= 0 {
			return nil, nil, fmt.Errorf("mr is required")
		}
		if in.Message == "" {
			return nil, nil, fmt.Errorf("message is required")
		}
		client, project, err := resolveClientAndProject(f, in.Repo)
		if err != nil {
			return nil, nil, err
		}
		note, _, err := client.Notes.CreateMergeRequestNote(project, in.MR, &gitlab.CreateMergeRequestNoteOptions{
			Body: &in.Message,
		})
		if err != nil {
			return nil, nil, fmt.Errorf("adding comment: %w", err)
		}
		return plainResult(fmt.Sprintf("Added comment to !%d (note ID %d)", in.MR, note.ID)), nil, nil
	})
}

func registerMRApprove(server *mcp.Server, f *cmdutil.Factory) {
	type Input struct {
		MR   int64  `json:"mr"              jsonschema:"merge request IID"`
		Repo string `json:"repo,omitempty"  jsonschema:"repository in OWNER/REPO or HOST/OWNER/REPO format"`
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "mr_approve",
		Description: "Approve a merge request",
	}, func(_ context.Context, _ *mcp.CallToolRequest, in Input) (*mcp.CallToolResult, any, error) {
		if in.MR <= 0 {
			return nil, nil, fmt.Errorf("mr is required")
		}
		client, project, err := resolveClientAndProject(f, in.Repo)
		if err != nil {
			return nil, nil, err
		}
		_, _, err = client.MergeRequestApprovals.ApproveMergeRequest(project, in.MR, nil)
		if err != nil {
			return nil, nil, fmt.Errorf("approving merge request: %w", err)
		}
		return plainResult(fmt.Sprintf("Approved merge request !%d", in.MR)), nil, nil
	})
}

func registerMRMerge(server *mcp.Server, f *cmdutil.Factory) {
	type Input struct {
		MR                 int64  `json:"mr"                            jsonschema:"merge request IID"`
		Repo               string `json:"repo,omitempty"                jsonschema:"repository in OWNER/REPO or HOST/OWNER/REPO format"`
		Squash             bool   `json:"squash,omitempty"              jsonschema:"squash commits on merge"`
		RemoveSourceBranch bool   `json:"remove_source_branch,omitempty" jsonschema:"remove source branch after merge"`
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "mr_merge",
		Description: "Merge a merge request",
	}, func(_ context.Context, _ *mcp.CallToolRequest, in Input) (*mcp.CallToolResult, any, error) {
		if in.MR <= 0 {
			return nil, nil, fmt.Errorf("mr is required")
		}
		client, project, err := resolveClientAndProject(f, in.Repo)
		if err != nil {
			return nil, nil, err
		}
		opts := &gitlab.AcceptMergeRequestOptions{
			Squash:                   &in.Squash,
			ShouldRemoveSourceBranch: &in.RemoveSourceBranch,
		}
		mr, _, err := client.MergeRequests.AcceptMergeRequest(project, in.MR, opts)
		if err != nil {
			return nil, nil, fmt.Errorf("merging merge request: %w", err)
		}
		return plainResult(fmt.Sprintf("Merged merge request !%d", mr.IID)), nil, nil
	})
}

func registerMRClose(server *mcp.Server, f *cmdutil.Factory) {
	type Input struct {
		MR   int64  `json:"mr"              jsonschema:"merge request IID"`
		Repo string `json:"repo,omitempty"  jsonschema:"repository in OWNER/REPO or HOST/OWNER/REPO format"`
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "mr_close",
		Description: "Close a merge request",
	}, func(_ context.Context, _ *mcp.CallToolRequest, in Input) (*mcp.CallToolResult, any, error) {
		if in.MR <= 0 {
			return nil, nil, fmt.Errorf("mr is required")
		}
		client, project, err := resolveClientAndProject(f, in.Repo)
		if err != nil {
			return nil, nil, err
		}
		event := "close"
		mr, _, err := client.MergeRequests.UpdateMergeRequest(project, in.MR, &gitlab.UpdateMergeRequestOptions{
			StateEvent: &event,
		})
		if err != nil {
			return nil, nil, fmt.Errorf("closing merge request: %w", err)
		}
		return plainResult(fmt.Sprintf("Closed merge request !%d", mr.IID)), nil, nil
	})
}

func registerMRReopen(server *mcp.Server, f *cmdutil.Factory) {
	type Input struct {
		MR   int64  `json:"mr"              jsonschema:"merge request IID"`
		Repo string `json:"repo,omitempty"  jsonschema:"repository in OWNER/REPO or HOST/OWNER/REPO format"`
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "mr_reopen",
		Description: "Reopen a closed merge request",
	}, func(_ context.Context, _ *mcp.CallToolRequest, in Input) (*mcp.CallToolResult, any, error) {
		if in.MR <= 0 {
			return nil, nil, fmt.Errorf("mr is required")
		}
		client, project, err := resolveClientAndProject(f, in.Repo)
		if err != nil {
			return nil, nil, err
		}
		event := "reopen"
		mr, _, err := client.MergeRequests.UpdateMergeRequest(project, in.MR, &gitlab.UpdateMergeRequestOptions{
			StateEvent: &event,
		})
		if err != nil {
			return nil, nil, fmt.Errorf("reopening merge request: %w", err)
		}
		return plainResult(fmt.Sprintf("Reopened merge request !%d", mr.IID)), nil, nil
	})
}

func registerMRCreate(server *mcp.Server, f *cmdutil.Factory) {
	type Input struct {
		Title        string `json:"title"                   jsonschema:"merge request title"`
		Repo         string `json:"repo,omitempty"          jsonschema:"repository in OWNER/REPO or HOST/OWNER/REPO format"`
		Description  string `json:"description,omitempty"   jsonschema:"merge request description"`
		SourceBranch string `json:"source_branch"           jsonschema:"source branch (required)"`
		TargetBranch string `json:"target_branch,omitempty" jsonschema:"target branch (defaults to default branch)"`
		Draft        bool   `json:"draft,omitempty"         jsonschema:"create as draft"`
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "mr_create",
		Description: "Create a new merge request",
	}, func(_ context.Context, _ *mcp.CallToolRequest, in Input) (*mcp.CallToolResult, any, error) {
		if in.Title == "" {
			return nil, nil, fmt.Errorf("title is required")
		}
		if in.SourceBranch == "" {
			return nil, nil, fmt.Errorf("source_branch is required")
		}
		client, project, err := resolveClientAndProject(f, in.Repo)
		if err != nil {
			return nil, nil, err
		}
		title := in.Title
		if in.Draft && !strings.HasPrefix(title, "Draft:") {
			title = "Draft: " + title
		}
		opts := &gitlab.CreateMergeRequestOptions{
			Title:        &title,
			SourceBranch: &in.SourceBranch,
		}
		if in.Description != "" {
			opts.Description = &in.Description
		}
		if in.TargetBranch != "" {
			opts.TargetBranch = &in.TargetBranch
		}
		mr, _, err := client.MergeRequests.CreateMergeRequest(project, opts)
		if err != nil {
			return nil, nil, fmt.Errorf("creating merge request: %w", err)
		}
		return plainResult(fmt.Sprintf("Created merge request !%d\n%s", mr.IID, mr.WebURL)), nil, nil
	})
}

func registerMREdit(server *mcp.Server, f *cmdutil.Factory) {
	type Input struct {
		MR           int64  `json:"mr"                      jsonschema:"merge request IID"`
		Repo         string `json:"repo,omitempty"          jsonschema:"repository in OWNER/REPO or HOST/OWNER/REPO format"`
		Title        string `json:"title,omitempty"         jsonschema:"new title"`
		Description  string `json:"description,omitempty"   jsonschema:"new description"`
		TargetBranch string `json:"target_branch,omitempty" jsonschema:"new target branch"`
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "mr_edit",
		Description: "Edit an existing merge request",
	}, func(_ context.Context, _ *mcp.CallToolRequest, in Input) (*mcp.CallToolResult, any, error) {
		if in.MR <= 0 {
			return nil, nil, fmt.Errorf("mr is required")
		}
		client, project, err := resolveClientAndProject(f, in.Repo)
		if err != nil {
			return nil, nil, err
		}
		opts := &gitlab.UpdateMergeRequestOptions{}
		if in.Title != "" {
			opts.Title = &in.Title
		}
		if in.Description != "" {
			opts.Description = &in.Description
		}
		if in.TargetBranch != "" {
			opts.TargetBranch = &in.TargetBranch
		}
		mr, _, err := client.MergeRequests.UpdateMergeRequest(project, in.MR, opts)
		if err != nil {
			return nil, nil, fmt.Errorf("updating merge request: %w", err)
		}
		return plainResult(fmt.Sprintf("Updated merge request !%d\n%s", mr.IID, mr.WebURL)), nil, nil
	})
}
