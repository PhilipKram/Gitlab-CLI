package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/PhilipKram/gitlab-cli/internal/cmdutil"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// RegisterIssueTools registers all issue tools on the server.
func RegisterIssueTools(server *mcp.Server, f *cmdutil.Factory) {
	registerIssueList(server, f)
	registerIssueView(server, f)
	registerIssueCreate(server, f)
	registerIssueClose(server, f)
	registerIssueReopen(server, f)
	registerIssueComment(server, f)
	registerIssueEdit(server, f)
	registerIssueDelete(server, f)
}

func registerIssueList(server *mcp.Server, f *cmdutil.Factory) {
	type Input struct {
		Repo      string `json:"repo,omitempty"      jsonschema:"repository in OWNER/REPO or HOST/OWNER/REPO format"`
		State     string `json:"state,omitempty"     jsonschema:"filter by state: opened, closed, all"`
		Author    string `json:"author,omitempty"    jsonschema:"filter by author username"`
		Assignee  string `json:"assignee,omitempty"  jsonschema:"filter by assignee username"`
		Label     string `json:"label,omitempty"     jsonschema:"filter by label name"`
		Milestone string `json:"milestone,omitempty" jsonschema:"filter by milestone title"`
		Limit     int64  `json:"limit,omitempty"     jsonschema:"maximum number of results (default 30)"`
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "issue_list",
		Description: "List issues for a GitLab project",
	}, func(_ context.Context, _ *mcp.CallToolRequest, in Input) (*mcp.CallToolResult, any, error) {
		client, project, err := resolveClientAndProject(f, in.Repo)
		if err != nil {
			return nil, nil, err
		}
		opts := &gitlab.ListProjectIssuesOptions{
			ListOptions: gitlab.ListOptions{PerPage: clampPerPage(in.Limit)},
		}
		if in.State != "" {
			opts.State = &in.State
		}
		if in.Author != "" {
			opts.AuthorUsername = &in.Author
		}
		if in.Assignee != "" {
			opts.AssigneeUsername = &in.Assignee
		}
		if in.Label != "" {
			labels := gitlab.LabelOptions(strings.Split(in.Label, ","))
			opts.Labels = &labels
		}
		if in.Milestone != "" {
			opts.Milestone = &in.Milestone
		}
		issues, _, err := client.Issues.ListProjectIssues(project, opts)
		if err != nil {
			return nil, nil, fmt.Errorf("listing issues: %w", err)
		}
		return textResult(issues)
	})
}

func registerIssueView(server *mcp.Server, f *cmdutil.Factory) {
	type Input struct {
		Issue int64  `json:"issue"           jsonschema:"issue IID"`
		Repo  string `json:"repo,omitempty"  jsonschema:"repository in OWNER/REPO or HOST/OWNER/REPO format"`
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "issue_view",
		Description: "View details of a specific issue",
	}, func(_ context.Context, _ *mcp.CallToolRequest, in Input) (*mcp.CallToolResult, any, error) {
		if in.Issue <= 0 {
			return nil, nil, fmt.Errorf("issue is required")
		}
		client, project, err := resolveClientAndProject(f, in.Repo)
		if err != nil {
			return nil, nil, err
		}
		issue, _, err := client.Issues.GetIssue(project, in.Issue)
		if err != nil {
			return nil, nil, fmt.Errorf("getting issue: %w", err)
		}
		return textResult(issue)
	})
}

func registerIssueCreate(server *mcp.Server, f *cmdutil.Factory) {
	type Input struct {
		Title       string `json:"title"                 jsonschema:"issue title"`
		Repo        string `json:"repo,omitempty"        jsonschema:"repository in OWNER/REPO or HOST/OWNER/REPO format"`
		Description string `json:"description,omitempty" jsonschema:"issue description"`
		Label       string `json:"label,omitempty"       jsonschema:"labels to apply (comma-separated)"`
		Milestone   string `json:"milestone,omitempty"   jsonschema:"milestone ID to assign"`
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "issue_create",
		Description: "Create a new issue",
	}, func(_ context.Context, _ *mcp.CallToolRequest, in Input) (*mcp.CallToolResult, any, error) {
		if in.Title == "" {
			return nil, nil, fmt.Errorf("title is required")
		}
		client, project, err := resolveClientAndProject(f, in.Repo)
		if err != nil {
			return nil, nil, err
		}
		opts := &gitlab.CreateIssueOptions{
			Title: &in.Title,
		}
		if in.Description != "" {
			opts.Description = &in.Description
		}
		if in.Label != "" {
			labels := gitlab.LabelOptions(strings.Split(in.Label, ","))
			opts.Labels = &labels
		}
		issue, _, err := client.Issues.CreateIssue(project, opts)
		if err != nil {
			return nil, nil, fmt.Errorf("creating issue: %w", err)
		}
		return plainResult(fmt.Sprintf("Created issue #%d\n%s", issue.IID, issue.WebURL)), nil, nil
	})
}

func registerIssueClose(server *mcp.Server, f *cmdutil.Factory) {
	type Input struct {
		Issue int64  `json:"issue"           jsonschema:"issue IID"`
		Repo  string `json:"repo,omitempty"  jsonschema:"repository in OWNER/REPO or HOST/OWNER/REPO format"`
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "issue_close",
		Description: "Close an issue",
	}, func(_ context.Context, _ *mcp.CallToolRequest, in Input) (*mcp.CallToolResult, any, error) {
		if in.Issue <= 0 {
			return nil, nil, fmt.Errorf("issue is required")
		}
		client, project, err := resolveClientAndProject(f, in.Repo)
		if err != nil {
			return nil, nil, err
		}
		event := "close"
		issue, _, err := client.Issues.UpdateIssue(project, in.Issue, &gitlab.UpdateIssueOptions{
			StateEvent: &event,
		})
		if err != nil {
			return nil, nil, fmt.Errorf("closing issue: %w", err)
		}
		return plainResult(fmt.Sprintf("Closed issue #%d", issue.IID)), nil, nil
	})
}

func registerIssueReopen(server *mcp.Server, f *cmdutil.Factory) {
	type Input struct {
		Issue int64  `json:"issue"           jsonschema:"issue IID"`
		Repo  string `json:"repo,omitempty"  jsonschema:"repository in OWNER/REPO or HOST/OWNER/REPO format"`
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "issue_reopen",
		Description: "Reopen a closed issue",
	}, func(_ context.Context, _ *mcp.CallToolRequest, in Input) (*mcp.CallToolResult, any, error) {
		if in.Issue <= 0 {
			return nil, nil, fmt.Errorf("issue is required")
		}
		client, project, err := resolveClientAndProject(f, in.Repo)
		if err != nil {
			return nil, nil, err
		}
		event := "reopen"
		issue, _, err := client.Issues.UpdateIssue(project, in.Issue, &gitlab.UpdateIssueOptions{
			StateEvent: &event,
		})
		if err != nil {
			return nil, nil, fmt.Errorf("reopening issue: %w", err)
		}
		return plainResult(fmt.Sprintf("Reopened issue #%d", issue.IID)), nil, nil
	})
}

func registerIssueComment(server *mcp.Server, f *cmdutil.Factory) {
	type Input struct {
		Issue   int64  `json:"issue"           jsonschema:"issue IID"`
		Message string `json:"message"         jsonschema:"comment body text"`
		Repo    string `json:"repo,omitempty"  jsonschema:"repository in OWNER/REPO or HOST/OWNER/REPO format"`
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "issue_comment",
		Description: "Add a comment to an issue",
	}, func(_ context.Context, _ *mcp.CallToolRequest, in Input) (*mcp.CallToolResult, any, error) {
		if in.Issue <= 0 {
			return nil, nil, fmt.Errorf("issue is required")
		}
		if in.Message == "" {
			return nil, nil, fmt.Errorf("message is required")
		}
		client, project, err := resolveClientAndProject(f, in.Repo)
		if err != nil {
			return nil, nil, err
		}
		note, _, err := client.Notes.CreateIssueNote(project, in.Issue, &gitlab.CreateIssueNoteOptions{
			Body: &in.Message,
		})
		if err != nil {
			return nil, nil, fmt.Errorf("adding comment: %w", err)
		}
		return plainResult(fmt.Sprintf("Added comment to #%d (note ID %d)", in.Issue, note.ID)), nil, nil
	})
}

func registerIssueEdit(server *mcp.Server, f *cmdutil.Factory) {
	type Input struct {
		Issue       int64  `json:"issue"                 jsonschema:"issue IID"`
		Repo        string `json:"repo,omitempty"        jsonschema:"repository in OWNER/REPO or HOST/OWNER/REPO format"`
		Title       string `json:"title,omitempty"       jsonschema:"new title"`
		Description string `json:"description,omitempty" jsonschema:"new description"`
		Label       string `json:"label,omitempty"       jsonschema:"labels to set (comma-separated)"`
		Milestone   string `json:"milestone,omitempty"   jsonschema:"milestone ID to assign"`
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "issue_edit",
		Description: "Edit an existing issue",
	}, func(_ context.Context, _ *mcp.CallToolRequest, in Input) (*mcp.CallToolResult, any, error) {
		if in.Issue <= 0 {
			return nil, nil, fmt.Errorf("issue is required")
		}
		client, project, err := resolveClientAndProject(f, in.Repo)
		if err != nil {
			return nil, nil, err
		}
		opts := &gitlab.UpdateIssueOptions{}
		if in.Title != "" {
			opts.Title = &in.Title
		}
		if in.Description != "" {
			opts.Description = &in.Description
		}
		if in.Label != "" {
			labels := gitlab.LabelOptions(strings.Split(in.Label, ","))
			opts.Labels = &labels
		}
		issue, _, err := client.Issues.UpdateIssue(project, in.Issue, opts)
		if err != nil {
			return nil, nil, fmt.Errorf("updating issue: %w", err)
		}
		return plainResult(fmt.Sprintf("Updated issue #%d\n%s", issue.IID, issue.WebURL)), nil, nil
	})
}

func registerIssueDelete(server *mcp.Server, f *cmdutil.Factory) {
	type Input struct {
		Issue int64  `json:"issue"           jsonschema:"issue IID"`
		Repo  string `json:"repo,omitempty"  jsonschema:"repository in OWNER/REPO or HOST/OWNER/REPO format"`
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "issue_delete",
		Description: "Delete an issue (requires owner/maintainer permissions)",
	}, func(_ context.Context, _ *mcp.CallToolRequest, in Input) (*mcp.CallToolResult, any, error) {
		if in.Issue <= 0 {
			return nil, nil, fmt.Errorf("issue is required")
		}
		client, project, err := resolveClientAndProject(f, in.Repo)
		if err != nil {
			return nil, nil, err
		}
		_, err = client.Issues.DeleteIssue(project, in.Issue)
		if err != nil {
			return nil, nil, fmt.Errorf("deleting issue: %w", err)
		}
		return plainResult(fmt.Sprintf("Deleted issue #%d", in.Issue)), nil, nil
	})
}
