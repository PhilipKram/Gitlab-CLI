package cmd

import (
	"testing"
)

func TestPipelineTrendsCmd(t *testing.T) {
	f := newTestFactory()
	cmd := newPipelineTrendsCmd(f)

	if cmd.Use != "trends" {
		t.Errorf("expected Use to be 'trends', got %q", cmd.Use)
	}

	if cmd.Short != "Show pipeline duration trends" {
		t.Errorf("expected Short to be 'Show pipeline duration trends', got %q", cmd.Short)
	}

	expectedLong := "Analyze pipeline duration trends over time, showing whether pipeline durations are increasing, decreasing, or stable."
	if cmd.Long != expectedLong {
		t.Errorf("expected Long to be %q, got %q", expectedLong, cmd.Long)
	}

	if cmd.Example == "" {
		t.Error("expected Example to be set, got empty string")
	}
}

func TestPipelineTrendsCmd_Flags(t *testing.T) {
	f := newTestFactory()
	cmd := newPipelineTrendsCmd(f)

	expectedFlags := []string{
		"branch",
		"days",
		"bucket-size",
		"format",
		"json",
	}

	for _, flagName := range expectedFlags {
		flag := cmd.Flags().Lookup(flagName)
		if flag == nil {
			t.Errorf("expected flag %q not found", flagName)
		}
	}

	// Verify default days is 30
	daysFlag := cmd.Flags().Lookup("days")
	if daysFlag == nil {
		t.Fatal("days flag not found")
	}
	if daysFlag.DefValue != "30" {
		t.Errorf("expected default days to be 30, got %q", daysFlag.DefValue)
	}

	// Verify default bucket-size is 7
	bucketSizeFlag := cmd.Flags().Lookup("bucket-size")
	if bucketSizeFlag == nil {
		t.Fatal("bucket-size flag not found")
	}
	if bucketSizeFlag.DefValue != "7" {
		t.Errorf("expected default bucket-size to be 7, got %q", bucketSizeFlag.DefValue)
	}

	// Verify default format is "table"
	formatFlag := cmd.Flags().Lookup("format")
	if formatFlag == nil {
		t.Fatal("format flag not found")
	}
	if formatFlag.DefValue != "table" {
		t.Errorf("expected default format to be 'table', got %q", formatFlag.DefValue)
	}

	// Verify json flag default is false
	jsonFlag := cmd.Flags().Lookup("json")
	if jsonFlag == nil {
		t.Fatal("json flag not found")
	}
	if jsonFlag.DefValue != "false" {
		t.Errorf("expected default json to be false, got %q", jsonFlag.DefValue)
	}
}

func TestPipelineTrendsCmd_FlagShorthand(t *testing.T) {
	f := newTestFactory()
	cmd := newPipelineTrendsCmd(f)

	// Verify branch flag has shorthand -b
	branchFlag := cmd.Flags().Lookup("branch")
	if branchFlag == nil {
		t.Fatal("branch flag not found")
	}
	if branchFlag.Shorthand != "b" {
		t.Errorf("expected branch flag shorthand to be 'b', got %q", branchFlag.Shorthand)
	}

	// Verify days flag has shorthand -d
	daysFlag := cmd.Flags().Lookup("days")
	if daysFlag == nil {
		t.Fatal("days flag not found")
	}
	if daysFlag.Shorthand != "d" {
		t.Errorf("expected days flag shorthand to be 'd', got %q", daysFlag.Shorthand)
	}

	// Verify format flag has shorthand -F
	formatFlag := cmd.Flags().Lookup("format")
	if formatFlag == nil {
		t.Fatal("format flag not found")
	}
	if formatFlag.Shorthand != "F" {
		t.Errorf("expected format flag shorthand to be 'F', got %q", formatFlag.Shorthand)
	}
}
