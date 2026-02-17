package errors

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestAPIError_ErrorCode(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		want       string
	}{
		{"with status code", 404, "API_404"},
		{"with 500", 500, "API_500"},
		{"with zero status code", 0, "API_ERROR"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &APIError{StatusCode: tt.statusCode}
			got := err.ErrorCode()
			if got != tt.want {
				t.Errorf("ErrorCode() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestAPIError_ErrorDetails(t *testing.T) {
	t.Run("full details", func(t *testing.T) {
		underlying := errors.New("underlying")
		err := &APIError{
			Method:     "GET",
			URL:        "https://gitlab.com/api/v4/projects",
			StatusCode: 404,
			Suggestion: "Check the project path",
			Err:        underlying,
		}

		details := err.ErrorDetails()

		if details["method"] != "GET" {
			t.Errorf("method = %v, want GET", details["method"])
		}
		if details["url"] != "https://gitlab.com/api/v4/projects" {
			t.Errorf("url = %v, want expected URL", details["url"])
		}
		if details["status_code"] != 404 {
			t.Errorf("status_code = %v, want 404", details["status_code"])
		}
		if details["status_text"] != "Not Found" {
			t.Errorf("status_text = %v, want Not Found", details["status_text"])
		}
		if details["suggestion"] != "Check the project path" {
			t.Errorf("suggestion = %v, want expected suggestion", details["suggestion"])
		}
		if details["underlying_error"] != "underlying" {
			t.Errorf("underlying_error = %v, want 'underlying'", details["underlying_error"])
		}
	})

	t.Run("empty details", func(t *testing.T) {
		err := &APIError{}
		details := err.ErrorDetails()
		if len(details) != 0 {
			t.Errorf("expected empty details, got %v", details)
		}
	})
}

func TestAuthError_ErrorCode(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		want       string
	}{
		{"401", 401, "AUTH_UNAUTHENTICATED"},
		{"403", 403, "AUTH_FORBIDDEN"},
		{"other status", 500, "AUTH_500"},
		{"zero", 0, "AUTH_ERROR"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &AuthError{StatusCode: tt.statusCode}
			got := err.ErrorCode()
			if got != tt.want {
				t.Errorf("ErrorCode() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestAuthError_ErrorDetails(t *testing.T) {
	t.Run("full details", func(t *testing.T) {
		underlying := errors.New("token expired")
		err := &AuthError{
			Host:       "gitlab.com",
			Method:     "POST",
			URL:        "https://gitlab.com/api/v4/user",
			StatusCode: 401,
			Suggestion: "Re-authenticate",
			Err:        underlying,
		}

		details := err.ErrorDetails()

		if details["host"] != "gitlab.com" {
			t.Errorf("host = %v, want gitlab.com", details["host"])
		}
		if details["method"] != "POST" {
			t.Errorf("method = %v, want POST", details["method"])
		}
		if details["url"] != "https://gitlab.com/api/v4/user" {
			t.Errorf("url = %v", details["url"])
		}
		if details["status_code"] != 401 {
			t.Errorf("status_code = %v, want 401", details["status_code"])
		}
		if details["suggestion"] != "Re-authenticate" {
			t.Errorf("suggestion = %v", details["suggestion"])
		}
		if details["underlying_error"] != "token expired" {
			t.Errorf("underlying_error = %v", details["underlying_error"])
		}
	})

	t.Run("empty details", func(t *testing.T) {
		err := &AuthError{}
		details := err.ErrorDetails()
		if len(details) != 0 {
			t.Errorf("expected empty details, got %v", details)
		}
	})
}

func TestNetworkError_ErrorCode(t *testing.T) {
	err := &NetworkError{}
	got := err.ErrorCode()
	if got != "NETWORK_ERROR" {
		t.Errorf("ErrorCode() = %q, want NETWORK_ERROR", got)
	}
}

func TestNetworkError_ErrorDetails(t *testing.T) {
	t.Run("full details", func(t *testing.T) {
		underlying := errors.New("connection refused")
		err := &NetworkError{
			Host:       "gitlab.com",
			URL:        "https://gitlab.com/api/v4/projects",
			Suggestion: "Check connection",
			Err:        underlying,
		}

		details := err.ErrorDetails()

		if details["host"] != "gitlab.com" {
			t.Errorf("host = %v, want gitlab.com", details["host"])
		}
		if details["url"] != "https://gitlab.com/api/v4/projects" {
			t.Errorf("url = %v", details["url"])
		}
		if details["suggestion"] != "Check connection" {
			t.Errorf("suggestion = %v", details["suggestion"])
		}
		if details["underlying_error"] != "connection refused" {
			t.Errorf("underlying_error = %v", details["underlying_error"])
		}
	})

	t.Run("empty details", func(t *testing.T) {
		err := &NetworkError{}
		details := err.ErrorDetails()
		if len(details) != 0 {
			t.Errorf("expected empty details, got %v", details)
		}
	})
}

func TestVersionError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *VersionError
		contains []string
	}{
		{
			name: "full context",
			err: &VersionError{
				RequiredVersion: "15.0",
				ActualVersion:   "14.5",
				Feature:         "merge request approvals",
				Message:         "GitLab version too old",
				Suggestion:      "Upgrade to 15.0+",
				Err:             errors.New("version check failed"),
			},
			contains: []string{
				"GitLab version too old",
				"Feature: merge request approvals",
				"Required: GitLab 15.0 or higher",
				"Detected: GitLab 14.5",
				"version check failed",
				"Upgrade to 15.0+",
			},
		},
		{
			name: "minimal context",
			err:  &VersionError{},
			contains: []string{
				"GitLab version mismatch",
			},
		},
		{
			name: "no message uses default",
			err: &VersionError{
				RequiredVersion: "16.0",
				ActualVersion:   "15.0",
			},
			contains: []string{
				"GitLab version mismatch",
				"Required: GitLab 16.0 or higher",
				"Detected: GitLab 15.0",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.err.Error()
			for _, expected := range tt.contains {
				if !strings.Contains(result, expected) {
					t.Errorf("Error() missing %q\nGot: %s", expected, result)
				}
			}
		})
	}
}

func TestVersionError_Unwrap(t *testing.T) {
	underlying := errors.New("version mismatch")
	err := &VersionError{Err: underlying}
	if err.Unwrap() != underlying {
		t.Errorf("Unwrap() = %v, want %v", err.Unwrap(), underlying)
	}

	err2 := &VersionError{}
	if err2.Unwrap() != nil {
		t.Errorf("Unwrap() = %v, want nil", err2.Unwrap())
	}
}

func TestNewVersionError(t *testing.T) {
	underlying := errors.New("check failed")
	err := NewVersionError("15.0", "14.5", "merge approvals", "", underlying)

	if err.RequiredVersion != "15.0" {
		t.Errorf("RequiredVersion = %q, want 15.0", err.RequiredVersion)
	}
	if err.ActualVersion != "14.5" {
		t.Errorf("ActualVersion = %q, want 14.5", err.ActualVersion)
	}
	if err.Feature != "merge approvals" {
		t.Errorf("Feature = %q, want 'merge approvals'", err.Feature)
	}
	// Empty message should get auto-generated
	if !strings.Contains(err.Message, "15.0") || !strings.Contains(err.Message, "merge approvals") {
		t.Errorf("Message = %q, want auto-generated message with version and feature", err.Message)
	}
	if !strings.Contains(err.Suggestion, "15.0") || !strings.Contains(err.Suggestion, "14.5") {
		t.Errorf("Suggestion = %q, want auto-generated suggestion", err.Suggestion)
	}
	if err.Err != underlying {
		t.Errorf("Err = %v, want %v", err.Err, underlying)
	}
}

func TestNewVersionError_CustomMessage(t *testing.T) {
	err := NewVersionError("16.0", "15.0", "feature-x", "Custom message", nil)
	if err.Message != "Custom message" {
		t.Errorf("Message = %q, want 'Custom message'", err.Message)
	}
}

func TestIsVersionError(t *testing.T) {
	t.Run("direct VersionError", func(t *testing.T) {
		err := &VersionError{Message: "test"}
		if !IsVersionError(err) {
			t.Error("IsVersionError should return true for direct VersionError")
		}
	})

	t.Run("wrapped VersionError", func(t *testing.T) {
		verErr := &VersionError{Message: "test"}
		wrapped := fmt.Errorf("wrapped: %w", verErr)
		if !IsVersionError(wrapped) {
			t.Error("IsVersionError should return true for wrapped VersionError")
		}
	})

	t.Run("other error type", func(t *testing.T) {
		err := errors.New("generic error")
		if IsVersionError(err) {
			t.Error("IsVersionError should return false for non-VersionError")
		}
	})

	t.Run("nil error", func(t *testing.T) {
		if IsVersionError(nil) {
			t.Error("IsVersionError should return false for nil")
		}
	})
}

func TestAs_VersionError(t *testing.T) {
	t.Run("direct match", func(t *testing.T) {
		err := &VersionError{RequiredVersion: "15.0"}
		var target *VersionError
		if !As(err, &target) {
			t.Error("As should return true for direct VersionError")
		}
		if target.RequiredVersion != "15.0" {
			t.Errorf("target.RequiredVersion = %q, want 15.0", target.RequiredVersion)
		}
	})

	t.Run("in chain", func(t *testing.T) {
		verErr := &VersionError{RequiredVersion: "16.0"}
		apiErr := &APIError{Err: verErr}
		var target *VersionError
		if !As(apiErr, &target) {
			t.Error("As should find VersionError in chain")
		}
		if target != verErr {
			t.Error("As should set target to the VersionError in chain")
		}
	})
}

func TestVerboseMode(t *testing.T) {
	// Save original state
	original := verboseMode
	defer func() { verboseMode = original }()

	SetVerboseMode(false)
	if verboseMode {
		t.Error("SetVerboseMode(false) should set verboseMode to false")
	}

	SetVerboseMode(true)
	if !verboseMode {
		t.Error("SetVerboseMode(true) should set verboseMode to true")
	}

	if !IsVerboseMode() {
		t.Error("IsVerboseMode() should return true when verbose mode is enabled")
	}

	SetVerboseMode(false)
	// IsVerboseMode also checks GLAB_DEBUG env var, so result depends on env
}
