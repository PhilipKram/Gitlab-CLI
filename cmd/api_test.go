package cmd

import (
	"net/http"
	"strings"
	"testing"

	"github.com/PhilipKram/gitlab-cli/internal/cmdtest"
)

func TestNewAPICmd(t *testing.T) {
	f := newTestFactory()
	cmd := NewAPICmd(f)

	if cmd.Use != "api <endpoint>" {
		t.Errorf("expected Use to be 'api <endpoint>', got %q", cmd.Use)
	}

	if cmd.Short != "Make authenticated API requests" {
		t.Errorf("expected Short to be 'Make authenticated API requests', got %q", cmd.Short)
	}
}

func TestAPICmd_Flags(t *testing.T) {
	f := newTestFactory()
	cmd := NewAPICmd(f)

	expectedFlags := []string{
		"method",
		"body",
		"field",
		"header",
		"hostname",
	}

	for _, flagName := range expectedFlags {
		flag := cmd.Flags().Lookup(flagName)
		if flag == nil {
			t.Errorf("expected flag %q not found", flagName)
		}
	}

	// Verify method flag default value is GET
	methodFlag := cmd.Flags().Lookup("method")
	if methodFlag == nil {
		t.Fatal("method flag not found")
	}
	if methodFlag.DefValue != "GET" {
		t.Errorf("expected default method to be 'GET', got %q", methodFlag.DefValue)
	}

	// Verify method has shorthand -X
	if methodFlag.Shorthand != "X" {
		t.Errorf("expected method shorthand to be 'X', got %q", methodFlag.Shorthand)
	}

	// Verify field flag has shorthand -f
	fieldFlag := cmd.Flags().Lookup("field")
	if fieldFlag == nil {
		t.Fatal("field flag not found")
	}
	if fieldFlag.Shorthand != "f" {
		t.Errorf("expected field shorthand to be 'f', got %q", fieldFlag.Shorthand)
	}

	// Verify header flag has shorthand -H
	headerFlag := cmd.Flags().Lookup("header")
	if headerFlag == nil {
		t.Fatal("header flag not found")
	}
	if headerFlag.Shorthand != "H" {
		t.Errorf("expected header shorthand to be 'H', got %q", headerFlag.Shorthand)
	}

	// Verify body flag default value is empty
	bodyFlag := cmd.Flags().Lookup("body")
	if bodyFlag == nil {
		t.Fatal("body flag not found")
	}
	if bodyFlag.DefValue != "" {
		t.Errorf("expected default body to be empty, got %q", bodyFlag.DefValue)
	}

	// Verify hostname flag default value is empty
	hostnameFlag := cmd.Flags().Lookup("hostname")
	if hostnameFlag == nil {
		t.Fatal("hostname flag not found")
	}
	if hostnameFlag.DefValue != "" {
		t.Errorf("expected default hostname to be empty, got %q", hostnameFlag.DefValue)
	}
}

func TestAPIExecute_GETSuccess(t *testing.T) {
	_ = cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && strings.Contains(r.URL.Path, "/api/v4/projects") {
			cmdtest.JSONResponse(w, 200, map[string]interface{}{
				"id":   1,
				"name": "test-repo",
			})
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := NewAPICmd(f.Factory)
	cmd.SetArgs([]string{"/projects/1"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := f.IO.String()
	if !strings.Contains(output, "test-repo") {
		t.Errorf("expected output to contain project name, got: %s", output)
	}
}

func TestAPIExecute_POSTSuccess(t *testing.T) {
	_ = cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			cmdtest.JSONResponse(w, 201, map[string]interface{}{
				"id":      2,
				"created": true,
			})
			return
		}
		cmdtest.ErrorResponse(w, 404, "not found")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := NewAPICmd(f.Factory)
	cmd.SetArgs([]string{"-X", "POST", "/projects"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAPIExecute_Unauthorized(t *testing.T) {
	_ = cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.ErrorResponse(w, 401, "401 Unauthorized")
	})

	f := cmdtest.NewTestFactory(t)
	cmd := NewAPICmd(f.Factory)
	cmd.SetArgs([]string{"/projects/1"})

	// API command doesn't return errors for HTTP error codes,
	// it just outputs the response body
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := f.IO.String()
	if !strings.Contains(output, "401") {
		t.Errorf("expected output to contain error message, got: %s", output)
	}
}

func TestAPI_POSTRequest(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			cmdtest.JSONResponse(w, 201, map[string]interface{}{"created": true})
		} else {
			cmdtest.ErrorResponse(w, 405, "Method not allowed")
		}
	})

	f := cmdtest.NewTestFactory(t)
	cmd := NewAPICmd(f.Factory)
	cmd.SetArgs([]string{"-X", "POST", "/projects"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAPI_WithData(t *testing.T) {
	cmdtest.MockGitLabServer(t, "gitlab.com", func(w http.ResponseWriter, r *http.Request) {
		cmdtest.JSONResponse(w, 200, map[string]interface{}{"success": true})
	})

	f := cmdtest.NewTestFactory(t)
	cmd := NewAPICmd(f.Factory)
	cmd.SetArgs([]string{"-X", "POST", "--body", `{"name":"test"}`, "/projects"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
