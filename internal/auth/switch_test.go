package auth

import (
	"bytes"
	"strings"
	"testing"

	"github.com/PhilipKram/gitlab-cli/internal/config"
)

func TestSwitch(t *testing.T) {
	t.Setenv("GITLAB_TOKEN", "")
	t.Setenv("GLAB_TOKEN", "")

	writeTestHosts(t, config.HostsConfig{
		"gitlab.com": &config.HostConfig{
			Token:      "pat-token-abcdefgh",
			User:       "bob",
			AuthMethod: "pat",
		},
		"gitlab.corp.com": &config.HostConfig{
			Token:      "oauth-token-xyz",
			User:       "alice",
			AuthMethod: "oauth",
		},
	})
	t.Cleanup(func() { clearTestHosts(t) })

	// Simulate user selecting the second option (gitlab.corp.com)
	in := strings.NewReader("2\n")
	out := &bytes.Buffer{}

	selectedHost, err := Switch(in, out)
	if err != nil {
		t.Fatalf("Switch: %v", err)
	}

	// Verify the selected host was returned (note: order may vary due to map iteration)
	if selectedHost != "gitlab.com" && selectedHost != "gitlab.corp.com" {
		t.Errorf("Switch returned unexpected host: %s", selectedHost)
	}

	// Verify default_host was updated in config
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("loading config: %v", err)
	}

	if cfg.DefaultHost == "" {
		t.Error("default_host should be set after Switch")
	}

	if cfg.DefaultHost != selectedHost {
		t.Errorf("default_host = %q, want %q", cfg.DefaultHost, selectedHost)
	}
}

func TestSwitch_NoAuthenticatedHosts(t *testing.T) {
	t.Setenv("GITLAB_TOKEN", "")
	t.Setenv("GLAB_TOKEN", "")

	writeTestHosts(t, config.HostsConfig{})
	t.Cleanup(func() { clearTestHosts(t) })

	in := strings.NewReader("")
	out := &bytes.Buffer{}

	_, err := Switch(in, out)
	if err == nil {
		t.Fatal("expected Switch to fail with no authenticated hosts, but it succeeded")
	}

	expectedMsg := "no authenticated hosts"
	if !strings.Contains(err.Error(), expectedMsg) {
		t.Errorf("Switch error = %q, want to contain %q", err.Error(), expectedMsg)
	}
}

func TestSwitch_OnlyOneHost(t *testing.T) {
	t.Setenv("GITLAB_TOKEN", "")
	t.Setenv("GLAB_TOKEN", "")

	writeTestHosts(t, config.HostsConfig{
		"gitlab.com": &config.HostConfig{
			Token:      "pat-token-abcdefgh",
			User:       "bob",
			AuthMethod: "pat",
		},
	})
	t.Cleanup(func() { clearTestHosts(t) })

	in := strings.NewReader("")
	out := &bytes.Buffer{}

	_, err := Switch(in, out)
	if err == nil {
		t.Fatal("expected Switch to fail with only one host, but it succeeded")
	}

	expectedMsg := "only one authenticated host"
	if !strings.Contains(err.Error(), expectedMsg) {
		t.Errorf("Switch error = %q, want to contain %q", err.Error(), expectedMsg)
	}
}

func TestSwitch_InvalidSelection(t *testing.T) {
	t.Setenv("GITLAB_TOKEN", "")
	t.Setenv("GLAB_TOKEN", "")

	writeTestHosts(t, config.HostsConfig{
		"gitlab.com": &config.HostConfig{
			Token:      "pat-token-abcdefgh",
			User:       "bob",
			AuthMethod: "pat",
		},
		"gitlab.corp.com": &config.HostConfig{
			Token:      "oauth-token-xyz",
			User:       "alice",
			AuthMethod: "oauth",
		},
	})
	t.Cleanup(func() { clearTestHosts(t) })

	// Simulate user entering invalid selection
	in := strings.NewReader("99\n")
	out := &bytes.Buffer{}

	_, err := Switch(in, out)
	if err == nil {
		t.Fatal("expected Switch to fail with invalid selection, but it succeeded")
	}

	expectedMsg := "selecting host"
	if !strings.Contains(err.Error(), expectedMsg) {
		t.Errorf("Switch error = %q, want to contain %q", err.Error(), expectedMsg)
	}
}

func TestSwitch_InvalidSelection_Zero(t *testing.T) {
	t.Setenv("GITLAB_TOKEN", "")
	t.Setenv("GLAB_TOKEN", "")

	writeTestHosts(t, config.HostsConfig{
		"gitlab.com": &config.HostConfig{
			Token:      "pat-token-abcdefgh",
			User:       "bob",
			AuthMethod: "pat",
		},
		"gitlab.corp.com": &config.HostConfig{
			Token:      "oauth-token-xyz",
			User:       "alice",
			AuthMethod: "oauth",
		},
	})
	t.Cleanup(func() { clearTestHosts(t) })

	// Simulate user entering zero (invalid)
	in := strings.NewReader("0\n")
	out := &bytes.Buffer{}

	_, err := Switch(in, out)
	if err == nil {
		t.Fatal("expected Switch to fail with zero selection, but it succeeded")
	}

	expectedMsg := "selecting host"
	if !strings.Contains(err.Error(), expectedMsg) {
		t.Errorf("Switch error = %q, want to contain %q", err.Error(), expectedMsg)
	}
}

func TestSwitch_InvalidSelection_NonNumeric(t *testing.T) {
	t.Setenv("GITLAB_TOKEN", "")
	t.Setenv("GLAB_TOKEN", "")

	writeTestHosts(t, config.HostsConfig{
		"gitlab.com": &config.HostConfig{
			Token:      "pat-token-abcdefgh",
			User:       "bob",
			AuthMethod: "pat",
		},
		"gitlab.corp.com": &config.HostConfig{
			Token:      "oauth-token-xyz",
			User:       "alice",
			AuthMethod: "oauth",
		},
	})
	t.Cleanup(func() { clearTestHosts(t) })

	// Simulate user entering non-numeric input
	in := strings.NewReader("abc\n")
	out := &bytes.Buffer{}

	_, err := Switch(in, out)
	if err == nil {
		t.Fatal("expected Switch to fail with non-numeric selection, but it succeeded")
	}

	expectedMsg := "selecting host"
	if !strings.Contains(err.Error(), expectedMsg) {
		t.Errorf("Switch error = %q, want to contain %q", err.Error(), expectedMsg)
	}
}

func TestSwitch_FirstOption(t *testing.T) {
	t.Setenv("GITLAB_TOKEN", "")
	t.Setenv("GLAB_TOKEN", "")

	writeTestHosts(t, config.HostsConfig{
		"gitlab.com": &config.HostConfig{
			Token:      "pat-token-abcdefgh",
			User:       "bob",
			AuthMethod: "pat",
		},
		"gitlab.corp.com": &config.HostConfig{
			Token:      "oauth-token-xyz",
			User:       "alice",
			AuthMethod: "oauth",
		},
	})
	t.Cleanup(func() { clearTestHosts(t) })

	// Simulate user selecting the first option
	in := strings.NewReader("1\n")
	out := &bytes.Buffer{}

	selectedHost, err := Switch(in, out)
	if err != nil {
		t.Fatalf("Switch: %v", err)
	}

	// Verify a host was selected
	if selectedHost == "" {
		t.Error("expected Switch to return a host, got empty string")
	}

	// Verify default_host was updated
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("loading config: %v", err)
	}

	if cfg.DefaultHost != selectedHost {
		t.Errorf("default_host = %q, want %q", cfg.DefaultHost, selectedHost)
	}
}

func TestSwitch_DisplaysPrompt(t *testing.T) {
	t.Setenv("GITLAB_TOKEN", "")
	t.Setenv("GLAB_TOKEN", "")

	writeTestHosts(t, config.HostsConfig{
		"gitlab.com": &config.HostConfig{
			Token:      "pat-token-abcdefgh",
			User:       "bob",
			AuthMethod: "pat",
		},
		"gitlab.corp.com": &config.HostConfig{
			Token:      "oauth-token-xyz",
			User:       "alice",
			AuthMethod: "oauth",
		},
	})
	t.Cleanup(func() { clearTestHosts(t) })

	in := strings.NewReader("1\n")
	out := &bytes.Buffer{}

	_, err := Switch(in, out)
	if err != nil {
		t.Fatalf("Switch: %v", err)
	}

	output := out.String()

	// Verify the prompt was displayed
	if !strings.Contains(output, "Select a GitLab instance:") {
		t.Errorf("expected prompt in output, got %q", output)
	}

	// Verify both hosts appear in the output
	if !strings.Contains(output, "gitlab.com") {
		t.Errorf("expected gitlab.com in output, got %q", output)
	}

	if !strings.Contains(output, "gitlab.corp.com") {
		t.Errorf("expected gitlab.corp.com in output, got %q", output)
	}
}
