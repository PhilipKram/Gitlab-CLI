package auth

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/PhilipKram/gitlab-cli/internal/config"
)

var testConfigDir string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "auth-test-*")
	if err != nil {
		panic(err)
	}
	testConfigDir = dir
	os.Setenv("GLAB_CONFIG_DIR", dir)
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

// writeTestHosts writes a hosts.json to the shared test config dir.
func writeTestHosts(t *testing.T, hosts config.HostsConfig) {
	t.Helper()
	data, err := json.MarshalIndent(hosts, "", "  ")
	if err != nil {
		t.Fatalf("marshaling test hosts: %v", err)
	}
	if err := os.WriteFile(filepath.Join(testConfigDir, "hosts.json"), data, 0o600); err != nil {
		t.Fatalf("writing test hosts.json: %v", err)
	}
}

// clearTestHosts removes hosts.json from the test config dir.
func clearTestHosts(t *testing.T) {
	t.Helper()
	os.Remove(filepath.Join(testConfigDir, "hosts.json"))
}
