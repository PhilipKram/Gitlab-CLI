package version

import (
	"strings"
	"testing"
)

func TestParseVersion(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    Version
		wantErr bool
	}{
		{
			name:  "GitLab EE version",
			input: "16.7.0-ee",
			want:  Version{Major: 16, Minor: 7, Patch: 0, Original: "16.7.0-ee"},
		},
		{
			name:  "GitLab CE version",
			input: "15.11.3-ce",
			want:  Version{Major: 15, Minor: 11, Patch: 3, Original: "15.11.3-ce"},
		},
		{
			name:  "Simple version",
			input: "17.0.0",
			want:  Version{Major: 17, Minor: 0, Patch: 0, Original: "17.0.0"},
		},
		{
			name:  "Version with v prefix",
			input: "v16.7.0",
			want:  Version{Major: 16, Minor: 7, Patch: 0, Original: "v16.7.0"},
		},
		{
			name:  "Version with whitespace",
			input: "  16.7.0-ee  ",
			want:  Version{Major: 16, Minor: 7, Patch: 0, Original: "16.7.0-ee"},
		},
		{
			name:    "Empty version",
			input:   "",
			wantErr: true,
		},
		{
			name:    "Whitespace only",
			input:   "   ",
			wantErr: true,
		},
		{
			name:    "Invalid format",
			input:   "invalid",
			wantErr: true,
		},
		{
			name:    "Incomplete version",
			input:   "16.7",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseVersion(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseVersion() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got.Major != tt.want.Major || got.Minor != tt.want.Minor || got.Patch != tt.want.Patch {
					t.Errorf("ParseVersion() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		name    string
		current Version
		other   Version
		want    int
	}{
		{
			name:    "Current greater (major)",
			current: Version{Major: 16, Minor: 0, Patch: 0},
			other:   Version{Major: 15, Minor: 11, Patch: 3},
			want:    1,
		},
		{
			name:    "Current greater (minor)",
			current: Version{Major: 16, Minor: 7, Patch: 0},
			other:   Version{Major: 16, Minor: 6, Patch: 0},
			want:    1,
		},
		{
			name:    "Current greater (patch)",
			current: Version{Major: 16, Minor: 7, Patch: 1},
			other:   Version{Major: 16, Minor: 7, Patch: 0},
			want:    1,
		},
		{
			name:    "Versions equal",
			current: Version{Major: 16, Minor: 7, Patch: 0},
			other:   Version{Major: 16, Minor: 7, Patch: 0},
			want:    0,
		},
		{
			name:    "Current less (major)",
			current: Version{Major: 15, Minor: 11, Patch: 3},
			other:   Version{Major: 16, Minor: 0, Patch: 0},
			want:    -1,
		},
		{
			name:    "Current less (minor)",
			current: Version{Major: 16, Minor: 6, Patch: 0},
			other:   Version{Major: 16, Minor: 7, Patch: 0},
			want:    -1,
		},
		{
			name:    "Current less (patch)",
			current: Version{Major: 16, Minor: 7, Patch: 0},
			other:   Version{Major: 16, Minor: 7, Patch: 1},
			want:    -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CompareVersions(tt.current, tt.other)
			if got != tt.want {
				t.Errorf("CompareVersions() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsVersionSupported(t *testing.T) {
	tests := []struct {
		name           string
		currentVersion string
		minVersion     string
		want           bool
	}{
		{
			name:           "Version supported",
			currentVersion: "16.7.0-ee",
			minVersion:     "16.0.0",
			want:           true,
		},
		{
			name:           "Version equal",
			currentVersion: "16.0.0",
			minVersion:     "16.0.0",
			want:           true,
		},
		{
			name:           "Version not supported",
			currentVersion: "15.11.3-ce",
			minVersion:     "16.0.0",
			want:           false,
		},
		{
			name:           "Empty current version (graceful degradation)",
			currentVersion: "",
			minVersion:     "16.0.0",
			want:           true,
		},
		{
			name:           "Empty min version (graceful degradation)",
			currentVersion: "16.7.0",
			minVersion:     "",
			want:           true,
		},
		{
			name:           "Invalid current version (graceful degradation)",
			currentVersion: "invalid",
			minVersion:     "16.0.0",
			want:           true,
		},
		{
			name:           "Invalid min version (graceful degradation)",
			currentVersion: "16.7.0",
			minVersion:     "invalid",
			want:           true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsVersionSupported(tt.currentVersion, tt.minVersion)
			if got != tt.want {
				t.Errorf("IsVersionSupported() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCheckVersionRequirement(t *testing.T) {
	tests := []struct {
		name           string
		currentVersion string
		minVersion     string
		wantErr        bool
		errorContains  string
	}{
		{
			name:           "Version supported",
			currentVersion: "16.7.0-ee",
			minVersion:     "16.0.0",
			wantErr:        false,
		},
		{
			name:           "Version not supported",
			currentVersion: "15.11.3-ce",
			minVersion:     "16.0.0",
			wantErr:        true,
			errorContains:  "requires GitLab 16.0.0 or later",
		},
		{
			name:           "Empty current version (graceful degradation)",
			currentVersion: "",
			minVersion:     "16.0.0",
			wantErr:        false,
		},
		{
			name:           "Invalid current version (graceful degradation)",
			currentVersion: "invalid",
			minVersion:     "16.0.0",
			wantErr:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckVersionRequirement(tt.currentVersion, tt.minVersion)
			if (err != nil) != tt.wantErr {
				t.Errorf("CheckVersionRequirement() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("CheckVersionRequirement() error = %q, want to contain %q", err.Error(), tt.errorContains)
				}

				// Verify it returns a VersionError type
				if _, ok := err.(*VersionError); !ok {
					t.Errorf("CheckVersionRequirement() error type = %T, want *VersionError", err)
				}
			}
		})
	}
}

func TestVersionError(t *testing.T) {
	err := NewVersionError("15.11.3", "16.0.0", "merge request approvals", "gitlab.example.com")
	if err == nil {
		t.Fatal("NewVersionError() returned nil")
	}

	if err.CurrentVersion != "15.11.3" {
		t.Errorf("CurrentVersion = %q, want %q", err.CurrentVersion, "15.11.3")
	}
	if err.RequiredVersion != "16.0.0" {
		t.Errorf("RequiredVersion = %q, want %q", err.RequiredVersion, "16.0.0")
	}
	if err.Feature != "merge request approvals" {
		t.Errorf("Feature = %q, want %q", err.Feature, "merge request approvals")
	}
	if err.Host != "gitlab.example.com" {
		t.Errorf("Host = %q, want %q", err.Host, "gitlab.example.com")
	}

	errStr := err.Error()
	if errStr == "" {
		t.Error("VersionError.Error() returned empty string")
	}

	// Check that error message contains key information
	expectedParts := []string{"15.11.3", "16.0.0", "merge request approvals", "gitlab.example.com"}
	for _, part := range expectedParts {
		if !strings.Contains(errStr, part) {
			t.Errorf("Error message should contain %q, got:\n%s", part, errStr)
		}
	}

	// Check for actionable suggestion
	if !strings.Contains(errStr, "Contact your GitLab administrator") {
		t.Error("Error message should contain suggestion to contact administrator for custom instance")
	}
}

func TestVersionError_GitLabComHost(t *testing.T) {
	err := NewVersionError("15.11.3", "16.0.0", "feature", "gitlab.com")

	errStr := err.Error()

	// gitlab.com should suggest upgrading instead of contacting administrator
	if !strings.Contains(errStr, "Upgrade your GitLab instance") {
		t.Error("Error message for gitlab.com should suggest upgrading")
	}
	if strings.Contains(errStr, "Contact your GitLab administrator") {
		t.Error("Error message for gitlab.com should not mention contacting administrator")
	}
}

func TestVersionError_WithoutCustomMessage(t *testing.T) {
	err := &VersionError{
		RequiredVersion: "16.0.0",
	}

	errStr := err.Error()

	if !strings.Contains(errStr, "GitLab version 16.0.0 is required") {
		t.Error("Error message should contain default message when Message is empty")
	}
}

func TestVersionString(t *testing.T) {
	tests := []struct {
		name    string
		version Version
		want    string
	}{
		{
			name:    "Version with original",
			version: Version{Major: 16, Minor: 7, Patch: 0, Original: "16.7.0-ee"},
			want:    "16.7.0-ee",
		},
		{
			name:    "Version without original",
			version: Version{Major: 16, Minor: 7, Patch: 0},
			want:    "16.7.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.version.String()
			if got != tt.want {
				t.Errorf("Version.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

