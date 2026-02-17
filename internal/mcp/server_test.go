package mcp

import (
	"context"
	"testing"

	"github.com/PhilipKram/gitlab-cli/internal/cmdtest"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestNewMCPServer(t *testing.T) {
	tf := cmdtest.NewTestFactory(t)

	server := NewMCPServer(tf.Factory)
	if server == nil {
		t.Fatal("NewMCPServer returned nil")
	}
}

func TestNewMCPServerRegistersToolsAndResources(t *testing.T) {
	tf := cmdtest.NewTestFactory(t)

	server := NewMCPServer(tf.Factory)
	if server == nil {
		t.Fatal("NewMCPServer returned nil")
	}

	// Connect a client to verify tools, resources, and prompts are registered.
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
	defer func() { _ = cs.Close() }()

	// List tools
	toolsResult, err := cs.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	if len(toolsResult.Tools) == 0 {
		t.Fatal("expected registered tools, got none")
	}

	// Verify expected tool names exist
	toolNames := make(map[string]bool)
	for _, tool := range toolsResult.Tools {
		toolNames[tool.Name] = true
	}

	expectedTools := []string{
		"mr_list", "mr_view", "mr_diff", "mr_comment", "mr_approve",
		"mr_merge", "mr_close", "mr_reopen", "mr_create", "mr_edit",
		"issue_list", "issue_view", "issue_create", "issue_close",
		"issue_reopen", "issue_comment", "issue_edit", "issue_delete",
		"pipeline_list", "pipeline_view", "pipeline_run", "pipeline_cancel",
		"pipeline_retry", "pipeline_delete", "pipeline_jobs", "pipeline_job_log",
		"repo_list", "repo_view",
		"release_list", "release_view", "release_create", "release_delete",
		"label_list", "label_create", "label_delete",
		"snippet_list", "snippet_view", "snippet_create", "snippet_delete",
		"branch_list", "branch_create", "branch_delete",
		"user_whoami",
	}

	for _, name := range expectedTools {
		if !toolNames[name] {
			t.Errorf("expected tool %q to be registered", name)
		}
	}

	// List prompts
	promptsResult, err := cs.ListPrompts(ctx, nil)
	if err != nil {
		t.Fatalf("ListPrompts: %v", err)
	}
	if len(promptsResult.Prompts) == 0 {
		t.Fatal("expected registered prompts, got none")
	}

	promptNames := make(map[string]bool)
	for _, p := range promptsResult.Prompts {
		promptNames[p.Name] = true
	}

	expectedPrompts := []string{
		"review_mr", "explain_pipeline_failure", "summarize_issues",
		"draft_mr_description", "create_release_notes",
	}
	for _, name := range expectedPrompts {
		if !promptNames[name] {
			t.Errorf("expected prompt %q to be registered", name)
		}
	}

	// List resource templates
	resourcesResult, err := cs.ListResourceTemplates(ctx, nil)
	if err != nil {
		t.Fatalf("ListResourceTemplates: %v", err)
	}
	if len(resourcesResult.ResourceTemplates) == 0 {
		t.Fatal("expected registered resource templates, got none")
	}

	templateNames := make(map[string]bool)
	for _, rt := range resourcesResult.ResourceTemplates {
		templateNames[rt.Name] = true
	}

	expectedTemplates := []string{"readme", "ci-config", "mr-diff", "pipeline-job-log"}
	for _, name := range expectedTemplates {
		if !templateNames[name] {
			t.Errorf("expected resource template %q to be registered", name)
		}
	}
}
