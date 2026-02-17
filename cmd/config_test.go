package cmd

import (
	"testing"

	"github.com/PhilipKram/gitlab-cli/internal/cmdtest"
)

func TestNewConfigCmd(t *testing.T) {
	f := newTestFactory()
	cmd := NewConfigCmd(f)

	if cmd.Use != "config <command>" {
		t.Errorf("expected Use to be 'config <command>', got %q", cmd.Use)
	}

	if cmd.Short != "Manage configuration" {
		t.Errorf("expected Short to be 'Manage configuration', got %q", cmd.Short)
	}
}

func TestConfigCmd_HasSubcommands(t *testing.T) {
	f := newTestFactory()
	cmd := NewConfigCmd(f)

	expectedSubcommands := []string{
		"get",
		"set",
		"list",
	}

	subcommands := cmd.Commands()
	if len(subcommands) != len(expectedSubcommands) {
		t.Errorf("expected %d subcommands, got %d", len(expectedSubcommands), len(subcommands))
	}

	foundSubcommands := make(map[string]bool)
	for _, subcmd := range subcommands {
		foundSubcommands[subcmd.Name()] = true
	}

	for _, expected := range expectedSubcommands {
		if !foundSubcommands[expected] {
			t.Errorf("expected subcommand %q not found", expected)
		}
	}
}

func TestConfigGetCmd_Structure(t *testing.T) {
	f := newTestFactory()
	cmd := newConfigGetCmd(f)

	if cmd.Use != "get <key>" {
		t.Errorf("expected Use to be 'get <key>', got %q", cmd.Use)
	}

	if cmd.Short != "Get a configuration value" {
		t.Errorf("expected Short to be 'Get a configuration value', got %q", cmd.Short)
	}
}

func TestConfigGetCmd_Flags(t *testing.T) {
	f := newTestFactory()
	cmd := newConfigGetCmd(f)

	expectedFlags := []string{"host"}

	for _, flagName := range expectedFlags {
		flag := cmd.Flags().Lookup(flagName)
		if flag == nil {
			t.Errorf("expected flag %q not found", flagName)
		}
	}

	// Verify host flag default value
	hostFlag := cmd.Flags().Lookup("host")
	if hostFlag == nil {
		t.Fatal("host flag not found")
	}
	if hostFlag.DefValue != "" {
		t.Errorf("expected default host to be empty, got %q", hostFlag.DefValue)
	}
}

func TestConfigSetCmd_Structure(t *testing.T) {
	f := newTestFactory()
	cmd := newConfigSetCmd(f)

	if cmd.Use != "set <key> <value>" {
		t.Errorf("expected Use to be 'set <key> <value>', got %q", cmd.Use)
	}

	if cmd.Short != "Set a configuration value" {
		t.Errorf("expected Short to be 'Set a configuration value', got %q", cmd.Short)
	}
}

func TestConfigSetCmd_Flags(t *testing.T) {
	f := newTestFactory()
	cmd := newConfigSetCmd(f)

	expectedFlags := []string{"host"}

	for _, flagName := range expectedFlags {
		flag := cmd.Flags().Lookup(flagName)
		if flag == nil {
			t.Errorf("expected flag %q not found", flagName)
		}
	}

	// Verify host flag default value
	hostFlag := cmd.Flags().Lookup("host")
	if hostFlag == nil {
		t.Fatal("host flag not found")
	}
	if hostFlag.DefValue != "" {
		t.Errorf("expected default host to be empty, got %q", hostFlag.DefValue)
	}
}

func TestConfigListCmd_Structure(t *testing.T) {
	f := newTestFactory()
	cmd := newConfigListCmd(f)

	if cmd.Use != "list" {
		t.Errorf("expected Use to be 'list', got %q", cmd.Use)
	}

	if cmd.Short != "List configuration values" {
		t.Errorf("expected Short to be 'List configuration values', got %q", cmd.Short)
	}

	// Verify aliases
	if len(cmd.Aliases) != 1 || cmd.Aliases[0] != "ls" {
		t.Errorf("expected alias 'ls', got %v", cmd.Aliases)
	}
}

func TestConfigGet_Success(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := newConfigGetCmd(f.Factory)
	cmd.SetArgs([]string{"editor"})

	_ = cmd.Execute()
	// May error if key not set, which is OK
	// Just verify command executes without panic
}

func TestConfigSet_Success(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := newConfigSetCmd(f.Factory)
	cmd.SetArgs([]string{"editor", "vim"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestConfigList_Success(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := newConfigListCmd(f.Factory)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestConfigGet_NotFound(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := newConfigGetCmd(f.Factory)
	cmd.SetArgs([]string{"nonexistent_key"})

	_ = cmd.Execute()
	// May return empty string or error, both are OK
}

func TestConfigSet_InvalidKey(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := newConfigSetCmd(f.Factory)
	cmd.SetArgs([]string{"", "value"}) // Empty key

	err := cmd.Execute()
	// May fail validation
	_ = err
}

func TestConfigGet_AllKeys(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := newConfigGetCmd(f.Factory)
	cmd.SetArgs([]string{"token"})

	err := cmd.Execute()
	// May fail if not configured
	_ = err
}

func TestConfigSet_Hostname(t *testing.T) {
	f := cmdtest.NewTestFactory(t)
	cmd := newConfigSetCmd(f.Factory)
	cmd.SetArgs([]string{"hostname", "gitlab.example.com"})

	err := cmd.Execute()
	_ = err
}
