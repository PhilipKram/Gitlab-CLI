package cmd

import (
	"testing"
	"time"
)

func TestPipelineWatchCmd_Structure(t *testing.T) {
	f := newTestFactory()
	cmd := newPipelineWatchCmd(f)

	if cmd.Use != "watch <pipeline-id>" {
		t.Errorf("expected Use to be 'watch <pipeline-id>', got %q", cmd.Use)
	}

	if cmd.Short != "Watch a pipeline in real-time" {
		t.Errorf("expected Short to be 'Watch a pipeline in real-time', got %q", cmd.Short)
	}

	if cmd.Long == "" {
		t.Error("expected Long to be non-empty")
	}

	if cmd.Example == "" {
		t.Error("expected Example to be non-empty")
	}
}

func TestPipelineWatchCmd_Args(t *testing.T) {
	f := newTestFactory()
	cmd := newPipelineWatchCmd(f)

	// Should require exactly 1 argument
	if cmd.Args == nil {
		t.Fatal("expected Args validator to be set")
	}

	// Validate that 0 args fails
	err := cmd.Args(cmd, []string{})
	if err == nil {
		t.Error("expected error with 0 args")
	}

	// Validate that 1 arg succeeds
	err = cmd.Args(cmd, []string{"12345"})
	if err != nil {
		t.Errorf("expected no error with 1 arg, got %v", err)
	}

	// Validate that 2 args fails
	err = cmd.Args(cmd, []string{"12345", "extra"})
	if err == nil {
		t.Error("expected error with 2 args")
	}
}

func TestPipelineWatchCmd_Flags(t *testing.T) {
	f := newTestFactory()
	cmd := newPipelineWatchCmd(f)

	// Check interval flag exists
	intervalFlag := cmd.Flags().Lookup("interval")
	if intervalFlag == nil {
		t.Fatal("expected 'interval' flag to be defined")
	}

	if intervalFlag.Shorthand != "i" {
		t.Errorf("expected interval shorthand to be 'i', got %q", intervalFlag.Shorthand)
	}

	if intervalFlag.DefValue != "5s" {
		t.Errorf("expected default interval to be '5s', got %q", intervalFlag.DefValue)
	}
}

func TestStatusColor(t *testing.T) {
	tests := []struct {
		status   string
		contains string
	}{
		{"success", "\033[32m"},
		{"failed", "\033[31m"},
		{"running", "\033[33m"},
		{"pending", "\033[33m"},
		{"canceled", "\033[90m"},
		{"unknown", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			result := statusColor(tt.status)
			if len(result) == 0 {
				t.Error("expected non-empty result")
			}
			if tt.status != "unknown" {
				// Colored statuses should contain the ANSI code and the status text
				if !containsStr(result, tt.contains) {
					t.Errorf("expected result to contain %q, got %q", tt.contains, result)
				}
				if !containsStr(result, tt.status) {
					t.Errorf("expected result to contain status %q, got %q", tt.status, result)
				}
				if !containsStr(result, "\033[0m") {
					t.Errorf("expected result to contain reset code, got %q", result)
				}
			} else {
				if result != tt.status {
					t.Errorf("expected result to be %q, got %q", tt.status, result)
				}
			}
		})
	}
}

func TestIsTerminalStatus(t *testing.T) {
	terminal := []string{"success", "failed", "canceled", "skipped"}
	nonTerminal := []string{"running", "pending", "created", "manual"}

	for _, s := range terminal {
		if !isTerminalStatus(s) {
			t.Errorf("expected %q to be terminal", s)
		}
	}

	for _, s := range nonTerminal {
		if isTerminalStatus(s) {
			t.Errorf("expected %q to be non-terminal", s)
		}
	}
}

func TestTruncateWatch(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"short", 10, "short"},
		{"exactly10!", 10, "exactly10!"},
		{"this is a long string", 10, "this is..."},
		{"ab", 1, "a"},
	}

	for _, tt := range tests {
		result := truncateWatch(tt.input, tt.maxLen)
		if result != tt.expected {
			t.Errorf("truncateWatch(%q, %d) = %q, want %q", tt.input, tt.maxLen, result, tt.expected)
		}
	}
}

func TestPipelineWatchCmd_IntervalParsing(t *testing.T) {
	f := newTestFactory()
	cmd := newPipelineWatchCmd(f)

	// Set interval flag to a custom value
	err := cmd.Flags().Set("interval", "10s")
	if err != nil {
		t.Fatalf("failed to set interval flag: %v", err)
	}

	val, err := cmd.Flags().GetDuration("interval")
	if err != nil {
		t.Fatalf("failed to get interval flag: %v", err)
	}
	if val != 10*time.Second {
		t.Errorf("expected interval to be 10s, got %v", val)
	}
}

// containsStr checks if s contains substr.
func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && searchStr(s, substr)
}

func searchStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
