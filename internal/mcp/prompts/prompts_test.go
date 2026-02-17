package prompts

import (
	"context"
	"strings"
	"testing"

	"github.com/PhilipKram/gitlab-cli/internal/cmdtest"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func setupServer(t *testing.T) *mcp.ClientSession {
	t.Helper()

	tf := cmdtest.NewTestFactory(t)

	server := mcp.NewServer(&mcp.Implementation{
		Name:    "test-mcp",
		Version: "0.0.1",
	}, nil)

	RegisterPrompts(server, tf.Factory)

	st, ct := mcp.NewInMemoryTransports()
	ctx := context.Background()

	_, err := server.Connect(ctx, st, nil)
	if err != nil {
		t.Fatalf("server.Connect: %v", err)
	}

	client := mcp.NewClient(&mcp.Implementation{
		Name:    "test-client",
		Version: "0.0.1",
	}, nil)

	cs, err := client.Connect(ctx, ct, nil)
	if err != nil {
		t.Fatalf("client.Connect: %v", err)
	}
	t.Cleanup(func() { _ = cs.Close() })

	return cs
}

func getPrompt(t *testing.T, cs *mcp.ClientSession, name string, args map[string]string) (*mcp.GetPromptResult, error) {
	t.Helper()
	return cs.GetPrompt(context.Background(), &mcp.GetPromptParams{
		Name:      name,
		Arguments: args,
	})
}

func TestRegisterPrompts(t *testing.T) {
	cs := setupServer(t)

	result, err := cs.ListPrompts(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}

	names := make(map[string]bool)
	for _, p := range result.Prompts {
		names[p.Name] = true
	}

	expected := []string{
		"review_mr",
		"explain_pipeline_failure",
		"summarize_issues",
		"draft_mr_description",
		"create_release_notes",
	}

	for _, name := range expected {
		if !names[name] {
			t.Errorf("expected prompt %q to be registered", name)
		}
	}
}

func TestReviewMRPrompt(t *testing.T) {
	cs := setupServer(t)

	t.Run("basic", func(t *testing.T) {
		result, err := getPrompt(t, cs, "review_mr", map[string]string{
			"mr_id": "42",
		})
		if err != nil {
			t.Fatal(err)
		}
		if result.Description == "" {
			t.Error("expected non-empty description")
		}
		if !strings.Contains(result.Description, "!42") {
			t.Errorf("expected description to contain MR ID, got: %s", result.Description)
		}
		if len(result.Messages) == 0 {
			t.Fatal("expected at least one message")
		}
		tc, ok := result.Messages[0].Content.(*mcp.TextContent)
		if !ok {
			t.Fatalf("expected TextContent, got %T", result.Messages[0].Content)
		}
		if !strings.Contains(tc.Text, "!42") {
			t.Error("expected prompt text to contain MR ID")
		}
		if !strings.Contains(tc.Text, "Code Quality") {
			t.Error("expected prompt to contain 'Code Quality' section")
		}
	})

	t.Run("with repo", func(t *testing.T) {
		result, err := getPrompt(t, cs, "review_mr", map[string]string{
			"mr_id": "10",
			"repo":  "my-org/my-repo",
		})
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(result.Description, "my-org/my-repo") {
			t.Errorf("expected description to contain repo, got: %s", result.Description)
		}
	})

	t.Run("missing mr_id", func(t *testing.T) {
		_, err := getPrompt(t, cs, "review_mr", map[string]string{})
		if err == nil {
			t.Error("expected error for missing mr_id")
		}
	})
}

func TestExplainPipelineFailurePrompt(t *testing.T) {
	cs := setupServer(t)

	t.Run("basic", func(t *testing.T) {
		result, err := getPrompt(t, cs, "explain_pipeline_failure", map[string]string{
			"pipeline_id": "100",
		})
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(result.Description, "#100") {
			t.Errorf("expected description to contain pipeline ID, got: %s", result.Description)
		}
		tc := result.Messages[0].Content.(*mcp.TextContent)
		if !strings.Contains(tc.Text, "#100") {
			t.Error("expected prompt text to contain pipeline ID")
		}
		if !strings.Contains(tc.Text, "Root Cause") {
			t.Error("expected prompt to contain 'Root Cause' section")
		}
	})

	t.Run("with repo and job_id", func(t *testing.T) {
		result, err := getPrompt(t, cs, "explain_pipeline_failure", map[string]string{
			"pipeline_id": "100",
			"repo":        "my-org/my-repo",
			"job_id":       "500",
		})
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(result.Description, "my-org/my-repo") {
			t.Errorf("expected description to contain repo, got: %s", result.Description)
		}
		if !strings.Contains(result.Description, "job #500") {
			t.Errorf("expected description to contain job ID, got: %s", result.Description)
		}
	})

	t.Run("missing pipeline_id", func(t *testing.T) {
		_, err := getPrompt(t, cs, "explain_pipeline_failure", map[string]string{})
		if err == nil {
			t.Error("expected error for missing pipeline_id")
		}
	})
}

func TestSummarizeIssuesPrompt(t *testing.T) {
	cs := setupServer(t)

	t.Run("defaults", func(t *testing.T) {
		result, err := getPrompt(t, cs, "summarize_issues", map[string]string{})
		if err != nil {
			t.Fatal(err)
		}
		// Default state is "opened"
		if !strings.Contains(result.Description, "opened") {
			t.Errorf("expected description to contain default state 'opened', got: %s", result.Description)
		}
		tc := result.Messages[0].Content.(*mcp.TextContent)
		if !strings.Contains(tc.Text, "opened issues") {
			t.Error("expected prompt to reference opened issues")
		}
	})

	t.Run("with filters", func(t *testing.T) {
		result, err := getPrompt(t, cs, "summarize_issues", map[string]string{
			"repo":      "my-org/my-repo",
			"state":     "closed",
			"labels":    "bug,critical",
			"assignee":  "johndoe",
			"milestone": "v1.0",
		})
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(result.Description, "my-org/my-repo") {
			t.Errorf("expected repo in description, got: %s", result.Description)
		}
		if !strings.Contains(result.Description, "closed") {
			t.Errorf("expected state in description, got: %s", result.Description)
		}
		tc := result.Messages[0].Content.(*mcp.TextContent)
		if !strings.Contains(tc.Text, "labels: bug,critical") {
			t.Error("expected labels filter in prompt text")
		}
		if !strings.Contains(tc.Text, "assignee: johndoe") {
			t.Error("expected assignee filter in prompt text")
		}
		if !strings.Contains(tc.Text, "milestone: v1.0") {
			t.Error("expected milestone filter in prompt text")
		}
	})
}

func TestDraftMRDescriptionPrompt(t *testing.T) {
	cs := setupServer(t)

	t.Run("basic", func(t *testing.T) {
		result, err := getPrompt(t, cs, "draft_mr_description", map[string]string{})
		if err != nil {
			t.Fatal(err)
		}
		tc := result.Messages[0].Content.(*mcp.TextContent)
		if !strings.Contains(tc.Text, "Title") {
			t.Error("expected prompt to contain 'Title' section")
		}
		if !strings.Contains(tc.Text, "Summary") {
			t.Error("expected prompt to contain 'Summary' section")
		}
	})

	t.Run("with branches", func(t *testing.T) {
		result, err := getPrompt(t, cs, "draft_mr_description", map[string]string{
			"repo":          "my-org/my-repo",
			"source_branch": "feature-xyz",
			"target_branch": "main",
		})
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(result.Description, "my-org/my-repo") {
			t.Errorf("expected repo in description, got: %s", result.Description)
		}
		if !strings.Contains(result.Description, "feature-xyz") {
			t.Errorf("expected source branch in description, got: %s", result.Description)
		}
		if !strings.Contains(result.Description, "main") {
			t.Errorf("expected target branch in description, got: %s", result.Description)
		}
	})
}

func TestCreateReleaseNotesPrompt(t *testing.T) {
	cs := setupServer(t)

	t.Run("basic", func(t *testing.T) {
		result, err := getPrompt(t, cs, "create_release_notes", map[string]string{})
		if err != nil {
			t.Fatal(err)
		}
		tc := result.Messages[0].Content.(*mcp.TextContent)
		if !strings.Contains(tc.Text, "Highlights") {
			t.Error("expected prompt to contain 'Highlights' section")
		}
		if !strings.Contains(tc.Text, "Bug Fixes") {
			t.Error("expected prompt to contain 'Bug Fixes' section")
		}
	})

	t.Run("with milestone", func(t *testing.T) {
		result, err := getPrompt(t, cs, "create_release_notes", map[string]string{
			"milestone": "v2.0",
		})
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(result.Description, "milestone v2.0") {
			t.Errorf("expected milestone in description, got: %s", result.Description)
		}
	})

	t.Run("with from_tag and to_tag", func(t *testing.T) {
		result, err := getPrompt(t, cs, "create_release_notes", map[string]string{
			"from_tag": "v1.0.0",
			"to_tag":   "v2.0.0",
		})
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(result.Description, "v1.0.0") {
			t.Errorf("expected from_tag in description, got: %s", result.Description)
		}
		if !strings.Contains(result.Description, "v2.0.0") {
			t.Errorf("expected to_tag in description, got: %s", result.Description)
		}
	})

	t.Run("with from_tag only", func(t *testing.T) {
		result, err := getPrompt(t, cs, "create_release_notes", map[string]string{
			"from_tag": "v1.0.0",
		})
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(result.Description, "since v1.0.0") {
			t.Errorf("expected 'since' in description, got: %s", result.Description)
		}
	})

	t.Run("with to_tag only", func(t *testing.T) {
		result, err := getPrompt(t, cs, "create_release_notes", map[string]string{
			"to_tag": "v2.0.0",
		})
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(result.Description, "up to v2.0.0") {
			t.Errorf("expected 'up to' in description, got: %s", result.Description)
		}
	})

	t.Run("with repo", func(t *testing.T) {
		result, err := getPrompt(t, cs, "create_release_notes", map[string]string{
			"repo": "my-org/my-repo",
		})
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(result.Description, "my-org/my-repo") {
			t.Errorf("expected repo in description, got: %s", result.Description)
		}
	})
}

func TestPromptMessageRole(t *testing.T) {
	cs := setupServer(t)

	// All prompts should return messages with role "user"
	prompts := []struct {
		name string
		args map[string]string
	}{
		{"review_mr", map[string]string{"mr_id": "1"}},
		{"explain_pipeline_failure", map[string]string{"pipeline_id": "1"}},
		{"summarize_issues", map[string]string{}},
		{"draft_mr_description", map[string]string{}},
		{"create_release_notes", map[string]string{}},
	}

	for _, p := range prompts {
		result, err := getPrompt(t, cs, p.name, p.args)
		if err != nil {
			t.Errorf("%s: %v", p.name, err)
			continue
		}
		for i, msg := range result.Messages {
			if msg.Role != "user" {
				t.Errorf("%s message[%d]: role=%q, want 'user'", p.name, i, msg.Role)
			}
		}
	}
}
