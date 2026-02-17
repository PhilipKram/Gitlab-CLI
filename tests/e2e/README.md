# End-to-End (E2E) Tests

This directory contains end-to-end tests that verify `glab` commands against a **real GitLab instance**. Unlike integration tests which use mock API servers, E2E tests exercise the complete system including actual GitLab API interactions.

## Overview

E2E tests provide the highest level of confidence that `glab` works correctly with real GitLab instances. These tests:

- Run against a real GitLab instance (gitlab.com or self-hosted)
- Require valid GitLab authentication
- Create and cleanup real resources (MRs, branches, etc.)
- Are **skipped by default** and only run when explicitly enabled
- Are slower than unit and integration tests

**Key characteristics:**
- Tests run against real GitLab API (not mocks)
- Require `GLAB_E2E_TEST=true` to enable
- Require `GITLAB_TOKEN` for authentication
- Require `GLAB_E2E_PROJECT` pointing to a test project
- Slower execution (seconds per test)
- May have quota/rate limiting considerations

## When to Use E2E Tests vs Integration Tests

| Aspect | Integration Tests | E2E Tests |
|--------|------------------|-----------|
| **API** | Mock HTTP server | Real GitLab instance |
| **Speed** | Fast (milliseconds) | Slow (seconds) |
| **Dependencies** | None | GitLab instance, auth, test project |
| **CI** | Run on every PR | Run nightly or on-demand |
| **Purpose** | Verify command logic | Verify real API compatibility |
| **Cost** | Free | May hit rate limits |

**General guidance:**
- Write **integration tests** for most command testing
- Write **E2E tests** for critical flows and API compatibility verification
- E2E tests should be a small subset of your test suite

## Setup

### Required Environment Variables

E2E tests require the following environment variables:

```bash
# Enable E2E tests (required)
export GLAB_E2E_TEST=true

# GitLab personal access token with API scope (required)
export GITLAB_TOKEN=glpat-your-token-here

# Test project in format "owner/repo" (required)
export GLAB_E2E_PROJECT=your-username/test-project

# GitLab hostname (optional, defaults to gitlab.com)
export GLAB_E2E_HOST=gitlab.com
```

### Creating a Test Project

E2E tests need a GitLab project to run against. This project should:

1. **Be dedicated to testing** - don't use a production project
2. **Have minimal value** - tests may create/delete branches and MRs
3. **Be accessible** - your token must have appropriate permissions

**Create a test project:**

```bash
# Using glab (if already set up)
glab repo create glab-e2e-test --public

# Or via GitLab web UI:
# 1. Go to https://gitlab.com/projects/new
# 2. Create project "glab-e2e-test"
# 3. Note the project path (e.g., "yourusername/glab-e2e-test")
```

**Set the project path:**

```bash
export GLAB_E2E_PROJECT=yourusername/glab-e2e-test
```

### Getting a GitLab Token

E2E tests require a GitLab personal access token:

1. Go to https://gitlab.com/-/profile/personal_access_tokens
2. Click "Add new token"
3. Name: "glab-e2e-tests"
4. Scopes: Select `api` (full API access)
5. Expiration: Set a reasonable date
6. Click "Create personal access token"
7. Copy the token immediately (it won't be shown again)

```bash
export GITLAB_TOKEN=glpat-your-token-here
```

**Security note:** Never commit tokens to version control. Use environment variables or a secure credential manager.

## Running E2E Tests

### Run All E2E Tests

```bash
# With environment variables set
GLAB_E2E_TEST=true \
GITLAB_TOKEN=glpat-your-token \
GLAB_E2E_PROJECT=yourusername/glab-e2e-test \
go test -v ./tests/e2e/...
```

### Run Specific E2E Test

```bash
GLAB_E2E_TEST=true \
GITLAB_TOKEN=glpat-your-token \
GLAB_E2E_PROJECT=yourusername/glab-e2e-test \
go test -v ./tests/e2e -run TestE2E_AuthStatus
```

### Skip E2E Tests (Default Behavior)

By default, E2E tests are skipped:

```bash
# E2E tests will be skipped with message:
# "Skipping E2E test (set GLAB_E2E_TEST=true to run)"
go test -v ./tests/e2e/...
```

## Writing E2E Tests

### Basic E2E Test Structure

```go
package e2e_test

import (
	"strings"
	"testing"

	"github.com/PhilipKram/gitlab-cli/cmd"
	"github.com/PhilipKram/gitlab-cli/tests/e2e"
)

func TestE2E_AuthStatus(t *testing.T) {
	// 1. Skip if E2E not enabled
	e2e.SkipIfNoE2E(t)

	// 2. Create E2E test factory (real API client)
	tf := e2e.NewE2ETestFactory(t)

	// 3. Execute command
	authCmd := cmd.NewAuthCmd(tf.Factory)
	authCmd.SetArgs([]string{"status"})
	authCmd.SetOut(tf.IO.Out)
	authCmd.SetErr(tf.IO.ErrOut)

	err := authCmd.Execute()
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}

	// 4. Assert output from real GitLab
	output := tf.IO.String()
	if !strings.Contains(output, tf.Host) {
		t.Errorf("expected output to contain host %q", tf.Host)
	}
}
```

### E2E Test with Resource Cleanup

```go
func TestE2E_MRWorkflow(t *testing.T) {
	e2e.SkipIfNoE2E(t)

	tf := e2e.NewE2ETestFactory(t)
	client := e2e.GetRealAPIClient(t)

	// Track resources for cleanup
	cleanup := e2e.NewCleanupTracker(t, client, tf.Owner+"/"+tf.Repo)

	// Generate unique test branch name
	branchName := e2e.GenerateTestName("mr-test")
	cleanup.TrackBranch(branchName)

	// Create test branch (in real test, use git or API)
	// ...

	// Create MR
	mrCmd := cmd.NewMRCmd(tf.Factory)
	mrCmd.SetArgs([]string{
		"create",
		"--title", "E2E Test MR",
		"--source-branch", branchName,
		"--target-branch", "main",
	})

	err := mrCmd.Execute()
	if err != nil {
		t.Fatalf("failed to create MR: %v", err)
	}

	// Parse MR IID from output and track for cleanup
	output := tf.IO.String()
	// ... parse MR IID ...
	// cleanup.TrackMergeRequest(mrIID)

	// Verify MR was created
	if !strings.Contains(output, "E2E Test MR") {
		t.Error("MR title not found in output")
	}

	// Cleanup happens automatically via t.Cleanup()
}
```

### Testing Eventually-Consistent Operations

Some GitLab operations are eventually consistent (e.g., pipelines starting). Use polling helpers:

```go
func TestE2E_PipelineRun(t *testing.T) {
	e2e.SkipIfNoE2E(t)

	tf := e2e.NewE2ETestFactory(t)

	// Run pipeline
	pipelineCmd := cmd.NewPipelineCmd(tf.Factory)
	pipelineCmd.SetArgs([]string{"run"})

	err := pipelineCmd.Execute()
	if err != nil {
		t.Fatalf("failed to run pipeline: %v", err)
	}

	// Wait for pipeline to start (with timeout)
	e2e.AssertEventuallyTrue(t, 30*time.Second, func() bool {
		listCmd := cmd.NewPipelineCmd(tf.Factory)
		listCmd.SetArgs([]string{"list", "--limit", "1"})

		tf.IO.Out.Reset()
		err := listCmd.Execute()
		if err != nil {
			return false
		}

		output := tf.IO.String()
		return strings.Contains(output, "running") || strings.Contains(output, "success")
	})
}
```

### Custom Cleanup Handlers

For complex cleanup scenarios:

```go
func TestE2E_CustomCleanup(t *testing.T) {
	e2e.SkipIfNoE2E(t)

	cleanup := e2e.NewCleanupTracker(t, client, projectPath)

	// Add custom cleanup logic
	cleanup.AddCleanupHandler(func() error {
		// Custom cleanup code
		// Return error if cleanup fails (will be logged)
		return nil
	})

	// ... test code ...
}
```

## E2E Test Helpers Reference

### Core Functions

| Function | Purpose |
|----------|---------|
| `SkipIfNoE2E(t)` | Skip test unless `GLAB_E2E_TEST=true` |
| `GetTestGitLabHost()` | Get GitLab hostname for tests (default: gitlab.com) |
| `GetTestProjectPath()` | Get test project owner/repo from `GLAB_E2E_PROJECT` |
| `NewE2ETestFactory(t)` | Create factory with real API client |
| `GetRealAPIClient(t)` | Create real GitLab API client |
| `VerifyE2ESetup(t)` | Verify all E2E requirements are met |

### Resource Management

| Function | Purpose |
|----------|---------|
| `NewCleanupTracker(t, client, project)` | Track resources for cleanup |
| `TrackMergeRequest(iid)` | Mark MR for cleanup |
| `TrackBranch(name)` | Mark branch for cleanup |
| `AddCleanupHandler(func)` | Add custom cleanup function |
| `GenerateTestName(prefix)` | Create unique timestamped name |

### Polling & Assertions

| Function | Purpose |
|----------|---------|
| `WaitForResource(t, timeout, interval, checkFunc)` | Poll until resource ready |
| `AssertEventuallyTrue(t, timeout, assertion)` | Retry assertion until success |

## Best Practices

### 1. Always Use SkipIfNoE2E

**Always** start E2E tests with `e2e.SkipIfNoE2E(t)`:

```go
func TestE2E_Something(t *testing.T) {
	e2e.SkipIfNoE2E(t)
	// ... rest of test
}
```

This ensures tests are skipped unless explicitly enabled.

### 2. Clean Up Resources

Use `CleanupTracker` to ensure resources are deleted even if tests fail:

```go
cleanup := e2e.NewCleanupTracker(t, client, projectPath)
cleanup.TrackMergeRequest(mrIID)
cleanup.TrackBranch(branchName)
// Cleanup happens automatically
```

### 3. Use Unique Names

Generate unique names to avoid conflicts with concurrent tests:

```go
branchName := e2e.GenerateTestName("feature")
// e.g., "e2e-test-feature-1709571234"
```

### 4. Handle Rate Limits

Be mindful of GitLab rate limits:

```go
// Add delays between API-heavy operations if needed
time.Sleep(1 * time.Second)

// Or use exponential backoff for retries
```

### 5. Verify Setup Early

Use `VerifyE2ESetup` at the start of test suites:

```go
func TestMain(m *testing.M) {
	// This will fail fast if setup is incorrect
	// when E2E tests are enabled
	code := m.Run()
	os.Exit(code)
}
```

### 6. Keep E2E Tests Minimal

Don't duplicate integration test coverage. Focus on:

- Critical user workflows
- Real API compatibility verification
- Features that are hard to mock accurately

### 7. Document Prerequisites

If tests require specific project setup:

```go
func TestE2E_ProtectedBranch(t *testing.T) {
	e2e.SkipIfNoE2E(t)

	// Document requirements in test or skip message
	t.Log("This test requires 'main' branch to be protected")

	// ... test code ...
}
```

## Testing Against Self-Hosted GitLab

E2E tests support self-hosted GitLab instances:

```bash
export GLAB_E2E_HOST=gitlab.example.com
export GITLAB_TOKEN=your-self-hosted-token
export GLAB_E2E_PROJECT=team/project
export GLAB_E2E_TEST=true

go test -v ./tests/e2e/...
```

**Note:** Ensure your self-hosted instance:
- Is accessible from your test environment
- Has API enabled
- Has a test project available
- Your token has appropriate scopes

## Troubleshooting

### Tests Skip Even With GLAB_E2E_TEST=true

Check that `GLAB_E2E_TEST` is exactly `"true"`:

```bash
echo $GLAB_E2E_TEST  # Should print: true
```

### "GITLAB_TOKEN environment variable required"

Ensure token is exported:

```bash
echo $GITLAB_TOKEN  # Should print your token
```

### "GLAB_E2E_PROJECT environment variable not set"

Set the test project path:

```bash
export GLAB_E2E_PROJECT=yourusername/your-test-project
```

### "invalid GLAB_E2E_PROJECT format"

Project path must be in `owner/repo` format:

```bash
# ✅ Correct
export GLAB_E2E_PROJECT=myusername/my-repo

# ❌ Wrong
export GLAB_E2E_PROJECT=my-repo
export GLAB_E2E_PROJECT=https://gitlab.com/myusername/my-repo
```

### Rate Limiting Errors

If you hit rate limits:

1. Add delays between tests: `time.Sleep(time.Second)`
2. Reduce number of concurrent E2E tests
3. Use a GitLab instance with higher rate limits
4. Implement exponential backoff in test helpers

## CI/CD Integration

### GitHub Actions Example

```yaml
name: E2E Tests (Nightly)

on:
  schedule:
    - cron: '0 2 * * *'  # Run at 2 AM daily
  workflow_dispatch:  # Allow manual trigger

jobs:
  e2e:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'

      - name: Run E2E Tests
        env:
          GLAB_E2E_TEST: true
          GITLAB_TOKEN: ${{ secrets.GITLAB_E2E_TOKEN }}
          GLAB_E2E_PROJECT: glab-org/e2e-test-project
        run: go test -v ./tests/e2e/...
```

### GitLab CI Example

```yaml
e2e-tests:
  stage: test
  only:
    - schedules  # Run on scheduled pipelines only
  script:
    - export GLAB_E2E_TEST=true
    - export GITLAB_TOKEN=$E2E_GITLAB_TOKEN
    - export GLAB_E2E_PROJECT=glab-org/e2e-test-project
    - go test -v ./tests/e2e/...
  when: manual  # Or remove for automatic runs
```

## Example Test Scenarios

### 1. Auth Flow

```go
func TestE2E_AuthStatus(t *testing.T) {
	e2e.SkipIfNoE2E(t)
	tf := e2e.NewE2ETestFactory(t)

	// Verify auth status against real GitLab
	// ... test implementation ...
}
```

### 2. MR Workflow

```go
func TestE2E_MRListAndView(t *testing.T) {
	e2e.SkipIfNoE2E(t)
	tf := e2e.NewE2ETestFactory(t)

	// List real MRs in test project
	// View a specific MR
	// Verify output format
}
```

### 3. Pipeline Workflow

```go
func TestE2E_PipelineList(t *testing.T) {
	e2e.SkipIfNoE2E(t)
	tf := e2e.NewE2ETestFactory(t)

	// List real pipelines
	// Verify output contains expected data
}
```

## Summary

- **E2E tests** run against real GitLab instances
- **Skipped by default** - require `GLAB_E2E_TEST=true`
- **Use sparingly** - complement integration tests, don't replace them
- **Clean up resources** - use `CleanupTracker`
- **Run in CI** - on nightly schedules or manually

For most testing, prefer **integration tests** (`tests/integration/`). Use E2E tests for critical flows and API compatibility verification.
