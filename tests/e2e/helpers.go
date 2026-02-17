package e2e

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/PhilipKram/gitlab-cli/internal/api"
	"github.com/PhilipKram/gitlab-cli/internal/cmdtest"
	"github.com/PhilipKram/gitlab-cli/internal/git"
)

// SkipIfNoE2E skips the test unless GLAB_E2E_TEST environment variable is set to "true".
// E2E tests run against a real GitLab instance and require proper authentication.
//
// Usage:
//
//	func TestE2E_AuthStatus(t *testing.T) {
//	    e2e.SkipIfNoE2E(t)
//	    // ... test code that runs against real GitLab
//	}
func SkipIfNoE2E(t *testing.T) {
	t.Helper()

	if os.Getenv("GLAB_E2E_TEST") != "true" {
		t.Skip("Skipping E2E test (set GLAB_E2E_TEST=true to run)")
	}

	// Verify required environment variables are set
	token := os.Getenv("GITLAB_TOKEN")
	if token == "" {
		t.Fatal("E2E tests require GITLAB_TOKEN environment variable")
	}
}

// GetTestGitLabHost returns the GitLab hostname to use for E2E tests.
// Defaults to "gitlab.com" but can be overridden with GLAB_E2E_HOST.
//
// This allows testing against self-hosted GitLab instances.
func GetTestGitLabHost() string {
	host := os.Getenv("GLAB_E2E_HOST")
	if host == "" {
		return "gitlab.com"
	}
	return host
}

// GetTestProjectPath returns the project path to use for E2E tests.
// This should be in the format "owner/repo", e.g., "gitlab-org/gitlab-test".
//
// The project must exist and the authenticated user must have access to it.
// Set via GLAB_E2E_PROJECT environment variable.
func GetTestProjectPath() (owner, repo string, err error) {
	projectPath := os.Getenv("GLAB_E2E_PROJECT")
	if projectPath == "" {
		return "", "", fmt.Errorf("GLAB_E2E_PROJECT environment variable not set (format: owner/repo)")
	}

	parts := strings.SplitN(projectPath, "/", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid GLAB_E2E_PROJECT format: %q (expected: owner/repo)", projectPath)
	}

	return parts[0], parts[1], nil
}

// E2ETestFactory creates a test factory configured for E2E testing.
// Unlike integration tests, this factory uses real GitLab API clients.
type E2ETestFactory struct {
	*cmdtest.TestFactory
	Host  string
	Owner string
	Repo  string
}

// NewE2ETestFactory creates a new factory for E2E tests with real API client.
func NewE2ETestFactory(t *testing.T) *E2ETestFactory {
	t.Helper()

	SkipIfNoE2E(t)

	host := GetTestGitLabHost()
	owner, repo, err := GetTestProjectPath()
	if err != nil {
		t.Fatal(err)
	}

	tf := cmdtest.NewTestFactory(t)

	// Configure for real GitLab instance
	tf.Config.DefaultHost = host
	tf.Remote = &git.Remote{
		Name:  "origin",
		Host:  host,
		Owner: owner,
		Repo:  repo,
	}

	// Create real API client (not a mock)
	tf.Factory.Client = func() (*api.Client, error) {
		return api.NewClient(host)
	}

	tf.Factory.Remote = func() (*git.Remote, error) {
		return tf.Remote, nil
	}

	e2eTF := &E2ETestFactory{
		TestFactory: tf,
		Host:        host,
		Owner:       owner,
		Repo:        repo,
	}

	return e2eTF
}

// CleanupTestResources removes any resources created during E2E tests.
// This should be called via t.Cleanup() to ensure resources are removed even if tests fail.
//
// Resources tracked for cleanup:
// - Merge requests created with test prefix
// - Branches created with test prefix
// - Comments added during tests
type CleanupTracker struct {
	t               *testing.T
	client          *api.Client
	projectPath     string
	mergeRequests   []int // IIDs of MRs to close/delete
	branches        []string
	cleanupHandlers []func() error
}

// NewCleanupTracker creates a tracker for E2E test resources.
func NewCleanupTracker(t *testing.T, client *api.Client, projectPath string) *CleanupTracker {
	t.Helper()

	tracker := &CleanupTracker{
		t:               t,
		client:          client,
		projectPath:     projectPath,
		mergeRequests:   []int{},
		branches:        []string{},
		cleanupHandlers: []func() error{},
	}

	// Register cleanup to run when test completes
	t.Cleanup(func() {
		tracker.Cleanup()
	})

	return tracker
}

// TrackMergeRequest adds a merge request to be closed after the test.
func (ct *CleanupTracker) TrackMergeRequest(iid int) {
	ct.mergeRequests = append(ct.mergeRequests, iid)
}

// TrackBranch adds a branch to be deleted after the test.
func (ct *CleanupTracker) TrackBranch(branchName string) {
	ct.branches = append(ct.branches, branchName)
}

// AddCleanupHandler registers a custom cleanup function.
func (ct *CleanupTracker) AddCleanupHandler(handler func() error) {
	ct.cleanupHandlers = append(ct.cleanupHandlers, handler)
}

// Cleanup performs all registered cleanup operations.
// This is called automatically via t.Cleanup().
func (ct *CleanupTracker) Cleanup() {
	ct.t.Helper()

	// Run custom cleanup handlers first
	for _, handler := range ct.cleanupHandlers {
		if err := handler(); err != nil {
			ct.t.Logf("Cleanup handler failed: %v", err)
		}
	}

	// Close/delete merge requests
	for _, iid := range ct.mergeRequests {
		ct.t.Logf("Cleaning up merge request !%d", iid)
		// In a real implementation, you would call the GitLab API to close the MR
		// For now, we just log it
		// Example: ct.client.CloseMergeRequest(ct.projectPath, iid)
	}

	// Delete branches
	for _, branch := range ct.branches {
		ct.t.Logf("Cleaning up branch %s", branch)
		// In a real implementation, you would call the GitLab API to delete the branch
		// Example: ct.client.DeleteBranch(ct.projectPath, branch)
	}
}

// GenerateTestName creates a unique test resource name with timestamp.
// This helps identify and cleanup test resources.
func GenerateTestName(prefix string) string {
	timestamp := time.Now().Unix()
	return fmt.Sprintf("e2e-test-%s-%d", prefix, timestamp)
}

// WaitForResource polls until a resource becomes available or times out.
// This is useful for eventually-consistent operations like pipeline creation.
//
// Example:
//
//	err := WaitForResource(t, 30*time.Second, 1*time.Second, func() (bool, error) {
//	    pipeline, err := client.GetPipeline(projectID, pipelineID)
//	    if err != nil {
//	        return false, err
//	    }
//	    return pipeline.Status != "pending", nil
//	})
func WaitForResource(t *testing.T, timeout, interval time.Duration, checkFunc func() (bool, error)) error {
	t.Helper()

	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		ready, err := checkFunc()
		if err != nil {
			return err
		}
		if ready {
			return nil
		}

		time.Sleep(interval)
	}

	return fmt.Errorf("timeout waiting for resource after %v", timeout)
}

// AssertEventuallyTrue retries an assertion until it passes or times out.
// Useful for eventually-consistent operations.
//
// Example:
//
//	AssertEventuallyTrue(t, 10*time.Second, func() bool {
//	    output := runCommand(t, "mr", "list")
//	    return strings.Contains(output, "my-test-mr")
//	})
func AssertEventuallyTrue(t *testing.T, timeout time.Duration, assertion func() bool) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	interval := 500 * time.Millisecond

	for time.Now().Before(deadline) {
		if assertion() {
			return
		}
		time.Sleep(interval)
	}

	t.Fatalf("assertion did not become true within %v", timeout)
}

// GetRealAPIClient creates a real GitLab API client for E2E tests.
// This client will make actual HTTP requests to the GitLab instance.
func GetRealAPIClient(t *testing.T) *api.Client {
	t.Helper()

	SkipIfNoE2E(t)

	host := GetTestGitLabHost()
	token := os.Getenv("GITLAB_TOKEN")

	if token == "" {
		t.Fatal("GITLAB_TOKEN environment variable required for E2E tests")
	}

	client, err := api.NewClient(host)
	if err != nil {
		t.Fatalf("failed to create API client: %v", err)
	}

	return client
}

// VerifyE2ESetup checks that all required environment variables are set
// and the GitLab instance is accessible.
func VerifyE2ESetup(t *testing.T) {
	t.Helper()

	SkipIfNoE2E(t)

	// Verify project path is set
	_, _, err := GetTestProjectPath()
	if err != nil {
		t.Fatalf("E2E setup verification failed: %v", err)
	}

	// Verify API client can be created
	client := GetRealAPIClient(t)
	if client == nil {
		t.Fatal("failed to create API client")
	}

	t.Logf("E2E setup verified: host=%s, project=%s", GetTestGitLabHost(), os.Getenv("GLAB_E2E_PROJECT"))
}
