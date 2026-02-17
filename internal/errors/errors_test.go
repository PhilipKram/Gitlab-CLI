package errors

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

// TestAPIError_Error tests the Error() method of APIError
func TestAPIError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *APIError
		contains []string
	}{
		{
			name: "full context",
			err: &APIError{
				Method:     "GET",
				URL:        "https://gitlab.com/api/v4/projects",
				StatusCode: 404,
				Message:    "Project not found",
				Suggestion: "Verify the project path is correct.",
				Err:        errors.New("underlying error"),
			},
			contains: []string{
				"Project not found",
				"GET https://gitlab.com/api/v4/projects",
				"404 Not Found",
				"underlying error",
				"→ Verify the project path is correct.",
			},
		},
		{
			name: "minimal context",
			err: &APIError{
				StatusCode: 500,
			},
			contains: []string{
				"API request failed",
				"500 Internal Server Error",
			},
		},
		{
			name: "with suggestion no underlying error",
			err: &APIError{
				Method:     "POST",
				URL:        "https://gitlab.com/api/v4/projects/1/issues",
				StatusCode: 422,
				Message:    "Validation failed",
				Suggestion: "Check your input parameters.",
			},
			contains: []string{
				"Validation failed",
				"POST https://gitlab.com/api/v4/projects/1/issues",
				"422 Unprocessable Entity",
				"→ Check your input parameters.",
			},
		},
		{
			name: "no suggestion",
			err: &APIError{
				Method:     "DELETE",
				URL:        "https://gitlab.com/api/v4/projects/1",
				StatusCode: 403,
				Message:    "Forbidden",
			},
			contains: []string{
				"Forbidden",
				"DELETE https://gitlab.com/api/v4/projects/1",
				"403 Forbidden",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.err.Error()
			for _, expected := range tt.contains {
				if !strings.Contains(result, expected) {
					t.Errorf("Error() output missing expected string %q\nGot: %s", expected, result)
				}
			}
		})
	}
}

// TestAPIError_Unwrap tests the Unwrap() method of APIError
func TestAPIError_Unwrap(t *testing.T) {
	underlyingErr := errors.New("connection timeout")
	apiErr := &APIError{
		Message: "Request failed",
		Err:     underlyingErr,
	}

	if unwrapped := apiErr.Unwrap(); unwrapped != underlyingErr {
		t.Errorf("Unwrap() = %v, want %v", unwrapped, underlyingErr)
	}

	apiErrNoUnderlying := &APIError{
		Message: "Request failed",
	}
	if unwrapped := apiErrNoUnderlying.Unwrap(); unwrapped != nil {
		t.Errorf("Unwrap() = %v, want nil", unwrapped)
	}
}

// TestAuthError_Error tests the Error() method of AuthError
func TestAuthError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *AuthError
		contains []string
	}{
		{
			name: "401 with full context",
			err: &AuthError{
				Host:       "gitlab.com",
				Method:     "GET",
				URL:        "https://gitlab.com/api/v4/user",
				StatusCode: 401,
				Message:    "Invalid token",
				Suggestion: "Run 'glab auth login' to re-authenticate.",
				Err:        errors.New("token expired"),
			},
			contains: []string{
				"Invalid token",
				"GET https://gitlab.com/api/v4/user",
				"401 Unauthorized",
				"gitlab.com",
				"token expired",
				"→ Run 'glab auth login' to re-authenticate.",
			},
		},
		{
			name: "403 with default message",
			err: &AuthError{
				Host:       "gitlab.example.com",
				Method:     "POST",
				URL:        "https://gitlab.example.com/api/v4/projects",
				StatusCode: 403,
			},
			contains: []string{
				"Permission denied",
				"POST https://gitlab.example.com/api/v4/projects",
				"403 Forbidden",
				"gitlab.example.com",
			},
		},
		{
			name: "401 with default message",
			err: &AuthError{
				StatusCode: 401,
			},
			contains: []string{
				"Authentication failed",
				"401 Unauthorized",
			},
		},
		{
			name: "other status code with default message",
			err: &AuthError{
				StatusCode: 500,
			},
			contains: []string{
				"Authorization error",
				"500 Internal Server Error",
			},
		},
		{
			name: "custom message overrides default",
			err: &AuthError{
				StatusCode: 401,
				Message:    "Custom auth error",
			},
			contains: []string{
				"Custom auth error",
				"401 Unauthorized",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.err.Error()
			for _, expected := range tt.contains {
				if !strings.Contains(result, expected) {
					t.Errorf("Error() output missing expected string %q\nGot: %s", expected, result)
				}
			}
		})
	}
}

// TestAuthError_Unwrap tests the Unwrap() method of AuthError
func TestAuthError_Unwrap(t *testing.T) {
	underlyingErr := errors.New("token validation failed")
	authErr := &AuthError{
		Message: "Auth failed",
		Err:     underlyingErr,
	}

	if unwrapped := authErr.Unwrap(); unwrapped != underlyingErr {
		t.Errorf("Unwrap() = %v, want %v", unwrapped, underlyingErr)
	}

	authErrNoUnderlying := &AuthError{
		Message: "Auth failed",
	}
	if unwrapped := authErrNoUnderlying.Unwrap(); unwrapped != nil {
		t.Errorf("Unwrap() = %v, want nil", unwrapped)
	}
}

// TestNetworkError_Error tests the Error() method of NetworkError
func TestNetworkError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *NetworkError
		contains []string
	}{
		{
			name: "full context with URL",
			err: &NetworkError{
				Host:       "gitlab.com",
				URL:        "https://gitlab.com/api/v4/projects",
				Message:    "Connection timeout",
				Suggestion: "Check your internet connection.",
				Err:        errors.New("dial tcp: timeout"),
			},
			contains: []string{
				"Connection timeout",
				"https://gitlab.com/api/v4/projects",
				"dial tcp: timeout",
				"→ Check your internet connection.",
			},
		},
		{
			name: "host only no URL",
			err: &NetworkError{
				Host:       "gitlab.example.com",
				Message:    "Connection refused",
				Suggestion: "Check firewall settings.",
			},
			contains: []string{
				"Connection refused",
				"gitlab.example.com",
				"→ Check firewall settings.",
			},
		},
		{
			name: "minimal context",
			err: &NetworkError{
				Host: "gitlab.com",
			},
			contains: []string{
				"Network error",
				"gitlab.com",
			},
		},
		{
			name: "URL takes precedence over host",
			err: &NetworkError{
				Host:    "gitlab.com",
				URL:     "https://gitlab.com/api/v4/user",
				Message: "DNS resolution failed",
			},
			contains: []string{
				"DNS resolution failed",
				"https://gitlab.com/api/v4/user",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.err.Error()
			for _, expected := range tt.contains {
				if !strings.Contains(result, expected) {
					t.Errorf("Error() output missing expected string %q\nGot: %s", expected, result)
				}
			}
		})
	}
}

// TestNetworkError_Unwrap tests the Unwrap() method of NetworkError
func TestNetworkError_Unwrap(t *testing.T) {
	underlyingErr := errors.New("connection refused")
	netErr := &NetworkError{
		Message: "Network failed",
		Err:     underlyingErr,
	}

	if unwrapped := netErr.Unwrap(); unwrapped != underlyingErr {
		t.Errorf("Unwrap() = %v, want %v", unwrapped, underlyingErr)
	}

	netErrNoUnderlying := &NetworkError{
		Message: "Network failed",
	}
	if unwrapped := netErrNoUnderlying.Unwrap(); unwrapped != nil {
		t.Errorf("Unwrap() = %v, want nil", unwrapped)
	}
}

// TestSuggestForStatusCode tests the SuggestForStatusCode function
func TestSuggestForStatusCode(t *testing.T) {
	tests := []struct {
		statusCode int
		contains   string
	}{
		{401, "glab auth login"},
		{403, "Permission denied"},
		{404, "Resource not found"},
		{422, "Validation failed"},
		{429, "Rate limit exceeded"},
		{500, "GitLab server error"},
		{502, "GitLab server error"},
		{503, "GitLab server error"},
		{504, "GitLab server error"},
		{400, "Request failed"},
		{418, "Request failed"}, // any 4xx
		{501, "Server error"},   // any other 5xx
		{200, ""},               // 2xx should return empty
		{300, ""},               // 3xx should return empty
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("status_%d", tt.statusCode), func(t *testing.T) {
			result := SuggestForStatusCode(tt.statusCode)
			if tt.contains == "" {
				if result != "" {
					t.Errorf("SuggestForStatusCode(%d) = %q, want empty string", tt.statusCode, result)
				}
			} else {
				if !strings.Contains(result, tt.contains) {
					t.Errorf("SuggestForStatusCode(%d) = %q, want to contain %q", tt.statusCode, result, tt.contains)
				}
			}
		})
	}
}

// TestSuggestForAuth tests the SuggestForAuth function
func TestSuggestForAuth(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		host       string
		contains   []string
	}{
		{
			name:       "401 with host",
			statusCode: 401,
			host:       "gitlab.example.com",
			contains:   []string{"glab auth login", "gitlab.example.com"},
		},
		{
			name:       "403 with host",
			statusCode: 403,
			host:       "gitlab.com",
			contains:   []string{"Permission denied", "gitlab.com", "api", "read_user", "write_repository"},
		},
		{
			name:       "other status code falls back",
			statusCode: 404,
			host:       "gitlab.com",
			contains:   []string{"Resource not found"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SuggestForAuth(tt.statusCode, tt.host)
			for _, expected := range tt.contains {
				if !strings.Contains(result, expected) {
					t.Errorf("SuggestForAuth(%d, %s) = %q, want to contain %q", tt.statusCode, tt.host, result, expected)
				}
			}
		})
	}
}

// TestSuggestForNetwork tests the SuggestForNetwork function
func TestSuggestForNetwork(t *testing.T) {
	result := SuggestForNetwork("gitlab.example.com")
	expected := []string{
		"gitlab.example.com",
		"internet connection",
		"proxy",
		"firewall",
	}

	for _, exp := range expected {
		if !strings.Contains(result, exp) {
			t.Errorf("SuggestForNetwork() = %q, want to contain %q", result, exp)
		}
	}
}

// TestNewAPIError tests the NewAPIError constructor
func TestNewAPIError(t *testing.T) {
	underlyingErr := errors.New("underlying error")
	apiErr := NewAPIError("GET", "https://gitlab.com/api/v4/projects", 404, "Not found", underlyingErr)

	if apiErr.Method != "GET" {
		t.Errorf("Method = %q, want %q", apiErr.Method, "GET")
	}
	if apiErr.URL != "https://gitlab.com/api/v4/projects" {
		t.Errorf("URL = %q, want %q", apiErr.URL, "https://gitlab.com/api/v4/projects")
	}
	if apiErr.StatusCode != 404 {
		t.Errorf("StatusCode = %d, want %d", apiErr.StatusCode, 404)
	}
	if apiErr.Message != "Not found" {
		t.Errorf("Message = %q, want %q", apiErr.Message, "Not found")
	}
	if apiErr.Err != underlyingErr {
		t.Errorf("Err = %v, want %v", apiErr.Err, underlyingErr)
	}
	if !strings.Contains(apiErr.Suggestion, "Resource not found") {
		t.Errorf("Suggestion = %q, want to contain %q", apiErr.Suggestion, "Resource not found")
	}
}

// TestNewAuthError tests the NewAuthError constructor
func TestNewAuthError(t *testing.T) {
	underlyingErr := errors.New("token expired")
	authErr := NewAuthError("gitlab.com", "GET", "https://gitlab.com/api/v4/user", 401, "Unauthorized", underlyingErr)

	if authErr.Host != "gitlab.com" {
		t.Errorf("Host = %q, want %q", authErr.Host, "gitlab.com")
	}
	if authErr.Method != "GET" {
		t.Errorf("Method = %q, want %q", authErr.Method, "GET")
	}
	if authErr.URL != "https://gitlab.com/api/v4/user" {
		t.Errorf("URL = %q, want %q", authErr.URL, "https://gitlab.com/api/v4/user")
	}
	if authErr.StatusCode != 401 {
		t.Errorf("StatusCode = %d, want %d", authErr.StatusCode, 401)
	}
	if authErr.Message != "Unauthorized" {
		t.Errorf("Message = %q, want %q", authErr.Message, "Unauthorized")
	}
	if authErr.Err != underlyingErr {
		t.Errorf("Err = %v, want %v", authErr.Err, underlyingErr)
	}
	if !strings.Contains(authErr.Suggestion, "glab auth login") {
		t.Errorf("Suggestion = %q, want to contain %q", authErr.Suggestion, "glab auth login")
	}
}

// TestNewNetworkError tests the NewNetworkError constructor
func TestNewNetworkError(t *testing.T) {
	underlyingErr := errors.New("connection refused")
	netErr := NewNetworkError("gitlab.com", "https://gitlab.com/api/v4/projects", "Connection failed", underlyingErr)

	if netErr.Host != "gitlab.com" {
		t.Errorf("Host = %q, want %q", netErr.Host, "gitlab.com")
	}
	if netErr.URL != "https://gitlab.com/api/v4/projects" {
		t.Errorf("URL = %q, want %q", netErr.URL, "https://gitlab.com/api/v4/projects")
	}
	if netErr.Message != "Connection failed" {
		t.Errorf("Message = %q, want %q", netErr.Message, "Connection failed")
	}
	if netErr.Err != underlyingErr {
		t.Errorf("Err = %v, want %v", netErr.Err, underlyingErr)
	}
	if !strings.Contains(netErr.Suggestion, "gitlab.com") {
		t.Errorf("Suggestion = %q, want to contain %q", netErr.Suggestion, "gitlab.com")
	}
}

// TestWrapAPIError tests the WrapAPIError function
func TestWrapAPIError(t *testing.T) {
	t.Run("nil error returns nil", func(t *testing.T) {
		result := WrapAPIError(nil, "GET", "https://gitlab.com/api/v4/projects", 404)
		if result != nil {
			t.Errorf("WrapAPIError(nil, ...) = %v, want nil", result)
		}
	})

	t.Run("existing APIError returned unchanged", func(t *testing.T) {
		originalErr := &APIError{
			Method:     "POST",
			URL:        "https://gitlab.com/api/v4/issues",
			StatusCode: 422,
			Message:    "Validation failed",
		}
		result := WrapAPIError(originalErr, "GET", "https://different.com", 500)
		if result != originalErr {
			t.Errorf("WrapAPIError(APIError, ...) should return original error unchanged")
		}
	})

	t.Run("existing AuthError returned unchanged", func(t *testing.T) {
		originalErr := &AuthError{
			Host:       "gitlab.com",
			StatusCode: 401,
			Message:    "Unauthorized",
		}
		result := WrapAPIError(originalErr, "GET", "https://gitlab.com/api/v4/user", 401)
		if result != originalErr {
			t.Errorf("WrapAPIError(AuthError, ...) should return original error unchanged")
		}
	})

	t.Run("generic error wrapped as APIError", func(t *testing.T) {
		genericErr := errors.New("something went wrong")
		result := WrapAPIError(genericErr, "DELETE", "https://gitlab.com/api/v4/projects/1", 500)

		var apiErr *APIError
		if !As(result, &apiErr) {
			t.Fatalf("WrapAPIError should return APIError")
		}

		if apiErr.Method != "DELETE" {
			t.Errorf("Method = %q, want %q", apiErr.Method, "DELETE")
		}
		if apiErr.URL != "https://gitlab.com/api/v4/projects/1" {
			t.Errorf("URL = %q, want %q", apiErr.URL, "https://gitlab.com/api/v4/projects/1")
		}
		if apiErr.StatusCode != 500 {
			t.Errorf("StatusCode = %d, want %d", apiErr.StatusCode, 500)
		}
		if apiErr.Err != genericErr {
			t.Errorf("Err = %v, want %v", apiErr.Err, genericErr)
		}
	})
}

// TestIsAPIError tests the IsAPIError function
func TestIsAPIError(t *testing.T) {
	t.Run("direct APIError", func(t *testing.T) {
		err := &APIError{Message: "test"}
		if !IsAPIError(err) {
			t.Error("IsAPIError should return true for direct APIError")
		}
	})

	t.Run("wrapped APIError", func(t *testing.T) {
		apiErr := &APIError{Message: "test"}
		wrapped := fmt.Errorf("wrapped: %w", apiErr)
		if !IsAPIError(wrapped) {
			t.Error("IsAPIError should return true for wrapped APIError")
		}
	})

	t.Run("other error type", func(t *testing.T) {
		err := errors.New("generic error")
		if IsAPIError(err) {
			t.Error("IsAPIError should return false for non-APIError")
		}
	})

	t.Run("nil error", func(t *testing.T) {
		if IsAPIError(nil) {
			t.Error("IsAPIError should return false for nil")
		}
	})
}

// TestIsAuthError tests the IsAuthError function
func TestIsAuthError(t *testing.T) {
	t.Run("direct AuthError", func(t *testing.T) {
		err := &AuthError{Message: "test"}
		if !IsAuthError(err) {
			t.Error("IsAuthError should return true for direct AuthError")
		}
	})

	t.Run("wrapped AuthError", func(t *testing.T) {
		authErr := &AuthError{Message: "test"}
		wrapped := fmt.Errorf("wrapped: %w", authErr)
		if !IsAuthError(wrapped) {
			t.Error("IsAuthError should return true for wrapped AuthError")
		}
	})

	t.Run("other error type", func(t *testing.T) {
		err := errors.New("generic error")
		if IsAuthError(err) {
			t.Error("IsAuthError should return false for non-AuthError")
		}
	})

	t.Run("nil error", func(t *testing.T) {
		if IsAuthError(nil) {
			t.Error("IsAuthError should return false for nil")
		}
	})
}

// TestIsNetworkError tests the IsNetworkError function
func TestIsNetworkError(t *testing.T) {
	t.Run("direct NetworkError", func(t *testing.T) {
		err := &NetworkError{Message: "test"}
		if !IsNetworkError(err) {
			t.Error("IsNetworkError should return true for direct NetworkError")
		}
	})

	t.Run("wrapped NetworkError", func(t *testing.T) {
		netErr := &NetworkError{Message: "test"}
		wrapped := fmt.Errorf("wrapped: %w", netErr)
		if !IsNetworkError(wrapped) {
			t.Error("IsNetworkError should return true for wrapped NetworkError")
		}
	})

	t.Run("other error type", func(t *testing.T) {
		err := errors.New("generic error")
		if IsNetworkError(err) {
			t.Error("IsNetworkError should return false for non-NetworkError")
		}
	})

	t.Run("nil error", func(t *testing.T) {
		if IsNetworkError(nil) {
			t.Error("IsNetworkError should return false for nil")
		}
	})
}

// TestAs tests the As function
func TestAs(t *testing.T) {
	t.Run("direct APIError match", func(t *testing.T) {
		err := &APIError{Message: "test"}
		var target *APIError
		if !As(err, &target) {
			t.Error("As should return true for direct match")
		}
		if target != err {
			t.Error("As should set target to the error")
		}
	})

	t.Run("wrapped APIError match", func(t *testing.T) {
		apiErr := &APIError{Message: "test", StatusCode: 404}
		wrappedErr := &NetworkError{Err: apiErr}
		var target *APIError
		if !As(wrappedErr, &target) {
			t.Error("As should return true for wrapped match")
		}
		if target != apiErr {
			t.Error("As should set target to the wrapped error")
		}
	})

	t.Run("no match", func(t *testing.T) {
		err := errors.New("generic error")
		var target *APIError
		if As(err, &target) {
			t.Error("As should return false when no match")
		}
	})

	t.Run("nil error", func(t *testing.T) {
		var target *APIError
		if As(nil, &target) {
			t.Error("As should return false for nil error")
		}
	})

	t.Run("AuthError in chain", func(t *testing.T) {
		authErr := &AuthError{StatusCode: 401}
		apiErr := &APIError{Err: authErr}
		var target *AuthError
		if !As(apiErr, &target) {
			t.Error("As should find AuthError in chain")
		}
		if target != authErr {
			t.Error("As should set target to the AuthError in chain")
		}
	})

	t.Run("NetworkError in chain", func(t *testing.T) {
		netErr := &NetworkError{Host: "gitlab.com"}
		authErr := &AuthError{Err: netErr}
		var target *NetworkError
		if !As(authErr, &target) {
			t.Error("As should find NetworkError in chain")
		}
		if target != netErr {
			t.Error("As should set target to the NetworkError in chain")
		}
	})
}
