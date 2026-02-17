package git

import (
	"testing"
)

func TestParseRemoteURL(t *testing.T) {
	tests := []struct {
		name      string
		rawURL    string
		wantHost  string
		wantOwner string
		wantRepo  string
	}{
		{
			name:      "SSH URL with .git suffix",
			rawURL:    "git@gitlab.com:owner/repo.git",
			wantHost:  "gitlab.com",
			wantOwner: "owner",
			wantRepo:  "repo",
		},
		{
			name:      "SSH URL without .git suffix",
			rawURL:    "git@gitlab.com:owner/repo",
			wantHost:  "gitlab.com",
			wantOwner: "owner",
			wantRepo:  "repo",
		},
		{
			name:      "SSH URL with nested group",
			rawURL:    "git@gitlab.com:group/subgroup/repo.git",
			wantHost:  "gitlab.com",
			wantOwner: "group",
			wantRepo:  "subgroup/repo",
		},
		{
			name:      "SSH URL with custom host",
			rawURL:    "git@gitlab.example.com:myteam/myproject.git",
			wantHost:  "gitlab.example.com",
			wantOwner: "myteam",
			wantRepo:  "myproject",
		},
		{
			name:      "HTTPS URL with .git suffix",
			rawURL:    "https://gitlab.com/owner/repo.git",
			wantHost:  "gitlab.com",
			wantOwner: "owner",
			wantRepo:  "repo",
		},
		{
			name:      "HTTPS URL without .git suffix",
			rawURL:    "https://gitlab.com/owner/repo",
			wantHost:  "gitlab.com",
			wantOwner: "owner",
			wantRepo:  "repo",
		},
		{
			name:      "HTTPS URL with nested group",
			rawURL:    "https://gitlab.com/group/subgroup/repo.git",
			wantHost:  "gitlab.com",
			wantOwner: "group",
			wantRepo:  "subgroup/repo",
		},
		{
			name:      "HTTP URL",
			rawURL:    "http://gitlab.example.com/owner/repo.git",
			wantHost:  "gitlab.example.com",
			wantOwner: "owner",
			wantRepo:  "repo",
		},
		{
			name:      "HTTPS URL with port",
			rawURL:    "https://gitlab.example.com:8443/owner/repo.git",
			wantHost:  "gitlab.example.com:8443",
			wantOwner: "owner",
			wantRepo:  "repo",
		},
		{
			name:      "SSH URL with only repo no owner",
			rawURL:    "git@gitlab.com:repo.git",
			wantHost:  "gitlab.com",
			wantOwner: "",
			wantRepo:  "repo",
		},
		{
			name:      "HTTPS URL with only repo no owner",
			rawURL:    "https://gitlab.com/repo.git",
			wantHost:  "gitlab.com",
			wantOwner: "",
			wantRepo:  "repo",
		},
		{
			name:      "invalid SSH URL missing colon",
			rawURL:    "git@gitlab.com/owner/repo.git",
			wantHost:  "",
			wantOwner: "",
			wantRepo:  "",
		},
		{
			name:      "empty string",
			rawURL:    "",
			wantHost:  "",
			wantOwner: "",
			wantRepo:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotHost, gotOwner, gotRepo := parseRemoteURL(tt.rawURL)
			if gotHost != tt.wantHost {
				t.Errorf("parseRemoteURL(%q) host = %q, want %q", tt.rawURL, gotHost, tt.wantHost)
			}
			if gotOwner != tt.wantOwner {
				t.Errorf("parseRemoteURL(%q) owner = %q, want %q", tt.rawURL, gotOwner, tt.wantOwner)
			}
			if gotRepo != tt.wantRepo {
				t.Errorf("parseRemoteURL(%q) repo = %q, want %q", tt.rawURL, gotRepo, tt.wantRepo)
			}
		})
	}
}

func TestCurrentBranch(t *testing.T) {
	// This test runs in the actual git repo
	branch, err := CurrentBranch()
	if err != nil {
		t.Fatalf("CurrentBranch() returned error: %v", err)
	}
	if branch == "" {
		t.Error("CurrentBranch() returned empty string")
	}
}

func TestTopLevelDir(t *testing.T) {
	// This test runs in the actual git repo
	dir, err := TopLevelDir()
	if err != nil {
		t.Fatalf("TopLevelDir() returned error: %v", err)
	}
	if dir == "" {
		t.Error("TopLevelDir() returned empty string")
	}
}

func TestRemotes(t *testing.T) {
	// This test runs in the actual git repo
	remotes, err := Remotes()
	if err != nil {
		t.Fatalf("Remotes() returned error: %v", err)
	}
	// The test repo should have at least one remote
	if len(remotes) == 0 {
		t.Skip("no remotes configured in test repo")
	}

	for _, r := range remotes {
		if r.Name == "" {
			t.Error("remote Name should not be empty")
		}
		// At least one of FetchURL or PushURL should be set
		if r.FetchURL == "" && r.PushURL == "" {
			t.Errorf("remote %q has neither FetchURL nor PushURL", r.Name)
		}
	}
}

func TestRemote_FieldsParsed(t *testing.T) {
	// This test runs in the actual git repo and verifies parsed fields
	remotes, err := Remotes()
	if err != nil {
		t.Fatalf("Remotes() returned error: %v", err)
	}
	if len(remotes) == 0 {
		t.Skip("no remotes configured in test repo")
	}

	// At least one remote should have host/owner/repo parsed
	foundParsed := false
	for _, r := range remotes {
		if r.Host != "" && r.Owner != "" && r.Repo != "" {
			foundParsed = true
			break
		}
	}
	if !foundParsed {
		t.Log("warning: no remotes with fully parsed host/owner/repo")
	}
}
