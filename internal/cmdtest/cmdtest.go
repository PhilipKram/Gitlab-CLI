package cmdtest

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/PhilipKram/gitlab-cli/internal/api"
	"github.com/PhilipKram/gitlab-cli/internal/cmdutil"
	"github.com/PhilipKram/gitlab-cli/internal/config"
	"github.com/PhilipKram/gitlab-cli/internal/git"
	"github.com/PhilipKram/gitlab-cli/pkg/iostreams"
	"github.com/spf13/cobra"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// TestIO captures stdout, stderr, and provides stdin for command testing.
type TestIO struct {
	In     *bytes.Buffer
	Out    *bytes.Buffer
	ErrOut *bytes.Buffer
}

// NewTestIO creates a new TestIO with empty buffers.
func NewTestIO() *TestIO {
	return &TestIO{
		In:     &bytes.Buffer{},
		Out:    &bytes.Buffer{},
		ErrOut: &bytes.Buffer{},
	}
}

// IOStreams returns an iostreams.IOStreams backed by the test buffers.
func (tio *TestIO) IOStreams() *iostreams.IOStreams {
	return &iostreams.IOStreams{
		In:     tio.In,
		Out:    tio.Out,
		ErrOut: tio.ErrOut,
	}
}

// String returns the captured stdout as a string.
func (tio *TestIO) String() string {
	return tio.Out.String()
}

// ErrString returns the captured stderr as a string.
func (tio *TestIO) ErrString() string {
	return tio.ErrOut.String()
}

// TestFactory creates a test Factory with controllable dependencies.
type TestFactory struct {
	*cmdutil.Factory
	IO     *TestIO
	Config *config.Config
	Client *api.Client
	Remote *git.Remote
}

// NewTestFactory creates a Factory suitable for testing with captured IO.
func NewTestFactory(t *testing.T) *TestFactory {
	t.Helper()

	// Isolate config directory so tests don't read/write real config
	t.Setenv("GLAB_CONFIG_DIR", t.TempDir())

	// Set a test token for authentication
	// This will be used by config.TokenForHost() which checks GITLAB_TOKEN env var first
	t.Setenv("GITLAB_TOKEN", "test-token-12345")

	tf := &TestFactory{
		Factory: &cmdutil.Factory{},
		IO:      NewTestIO(),
		Config: &config.Config{
			GitRemote: "origin",
		},
		Remote: &git.Remote{
			Name:  "origin",
			Host:  "gitlab.com",
			Owner: "test-owner",
			Repo:  "test-repo",
		},
	}

	tf.IOStreams = tf.IO.IOStreams()

	tf.Factory.Config = func() (*config.Config, error) {
		return tf.Config, nil
	}

	tf.Factory.Client = func() (*api.Client, error) {
		if tf.Client != nil {
			return tf.Client, nil
		}
		// Return a client with retries disabled so error tests run fast
		return api.NewClientWithToken("gitlab.com", "test-token-12345", gitlab.WithCustomRetryMax(0))
	}

	tf.Factory.Remote = func() (*git.Remote, error) {
		return tf.Remote, nil
	}

	tf.Version = "test-version"

	return tf
}

// RunCommand executes a cobra command with the given arguments and returns the output.
// It uses the TestFactory's captured IO for stdout and stderr.
func RunCommand(t *testing.T, tf *TestFactory, cmd *cobra.Command, args ...string) (stdout, stderr string, err error) {
	t.Helper()

	cmd.SetArgs(args)
	cmd.SetIn(tf.IO.In)
	cmd.SetOut(tf.IO.Out)
	cmd.SetErr(tf.IO.ErrOut)

	err = cmd.Execute()
	stdout = tf.IO.Out.String()
	stderr = tf.IO.ErrOut.String()

	return stdout, stderr, err
}

// StubInput sets the stdin content for testing interactive commands.
func StubInput(t *testing.T, tf *TestFactory, input string) {
	t.Helper()
	tf.IO.In = bytes.NewBufferString(input)
	tf.IOStreams.In = tf.IO.In
}

// AssertContains fails the test if the string does not contain the substring.
func AssertContains(t *testing.T, str, substr string) {
	t.Helper()
	if !strings.Contains(str, substr) {
		t.Errorf("expected output to contain %q, got:\n%s", substr, str)
	}
}

// AssertNotContains fails the test if the string contains the substring.
func AssertNotContains(t *testing.T, str, substr string) {
	t.Helper()
	if strings.Contains(str, substr) {
		t.Errorf("expected output NOT to contain %q, got:\n%s", substr, str)
	}
}

// AssertEqual fails the test if the values are not equal.
func AssertEqual(t *testing.T, got, want interface{}) {
	t.Helper()
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

// CopyReader copies an io.Reader to a new bytes.Buffer and returns both the buffer and the original content.
func CopyReader(r io.Reader) (*bytes.Buffer, error) {
	buf := &bytes.Buffer{}
	_, err := io.Copy(buf, r)
	return buf, err
}
