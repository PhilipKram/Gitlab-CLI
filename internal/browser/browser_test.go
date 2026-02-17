package browser

import (
	"strings"
	"testing"
)

func TestOpen_URLValidation(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "rejects empty string",
			url:     "",
			wantErr: true,
			errMsg:  "refusing to open non-HTTP URL",
		},
		{
			name:    "rejects ftp URL",
			url:     "ftp://example.com/file",
			wantErr: true,
			errMsg:  "refusing to open non-HTTP URL",
		},
		{
			name:    "rejects file URL",
			url:     "file:///etc/passwd",
			wantErr: true,
			errMsg:  "refusing to open non-HTTP URL",
		},
		{
			name:    "rejects ssh URL",
			url:     "ssh://user@host",
			wantErr: true,
			errMsg:  "refusing to open non-HTTP URL",
		},
		{
			name:    "rejects javascript URL",
			url:     "javascript:alert(1)",
			wantErr: true,
			errMsg:  "refusing to open non-HTTP URL",
		},
		{
			name:    "rejects data URL",
			url:     "data:text/html,<h1>test</h1>",
			wantErr: true,
			errMsg:  "refusing to open non-HTTP URL",
		},
		{
			name:    "rejects plain text",
			url:     "not-a-url",
			wantErr: true,
			errMsg:  "refusing to open non-HTTP URL",
		},
		{
			name:    "rejects URL with http in middle",
			url:     "nothttp://example.com",
			wantErr: true,
			errMsg:  "refusing to open non-HTTP URL",
		},
		{
			name:    "accepts http URL",
			url:     "http://example.com",
			wantErr: false,
		},
		{
			name:    "accepts https URL",
			url:     "https://gitlab.com/user/repo",
			wantErr: false,
		},
		{
			name:    "accepts http with port",
			url:     "http://localhost:8080/path",
			wantErr: false,
		},
		{
			name:    "accepts https with query params",
			url:     "https://gitlab.com/api/v4/projects?per_page=100",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Open(tt.url)
			if tt.wantErr {
				if err == nil {
					t.Errorf("Open(%q) = nil, want error", tt.url)
					return
				}
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Open(%q) error = %q, want to contain %q", tt.url, err.Error(), tt.errMsg)
				}
			}
			// For valid URLs, we don't check err == nil because the browser
			// command may fail in a test environment (no display, etc.).
			// We only verify that the URL validation itself passed by checking
			// that the error is NOT "refusing to open non-HTTP URL".
			if !tt.wantErr && err != nil {
				if strings.Contains(err.Error(), "refusing to open non-HTTP URL") {
					t.Errorf("Open(%q) rejected valid URL: %v", tt.url, err)
				}
			}
		})
	}
}
