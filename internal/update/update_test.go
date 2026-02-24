package update

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		current string
		latest  string
		want    bool
	}{
		{"0.0.11", "0.0.12", true},
		{"0.0.12", "0.0.12", false},
		{"0.0.13", "0.0.12", false},
		{"0.1.0", "0.2.0", true},
		{"1.0.0", "0.9.9", false},
		{"v0.0.11", "v0.0.12", true},
		{"v1.0.0", "1.0.0", false},
		{"0.0.1", "0.0.2", true},
		{"1.0.0", "2.0.0", true},
		{"1.2.3", "1.2.4", true},
		{"1.2.3", "1.3.0", true},
		{"1.2.3", "1.2.3", false},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s_vs_%s", tt.current, tt.latest), func(t *testing.T) {
			got := CompareVersions(tt.current, tt.latest)
			if got != tt.want {
				t.Errorf("CompareVersions(%q, %q) = %v, want %v", tt.current, tt.latest, got, tt.want)
			}
		})
	}
}

func TestArchiveName(t *testing.T) {
	name := ArchiveName("0.0.12")
	expected := fmt.Sprintf("glab_0.0.12_%s_%s", runtime.GOOS, runtime.GOARCH)
	if runtime.GOOS == "windows" {
		expected += ".zip"
	} else {
		expected += ".tar.gz"
	}
	if name != expected {
		t.Errorf("ArchiveName(\"0.0.12\") = %q, want %q", name, expected)
	}

	// With v prefix
	name = ArchiveName("v1.2.3")
	if got := name; got[:9] != "glab_1.2." {
		t.Errorf("ArchiveName should strip v prefix, got %q", got)
	}
}

func TestShouldCheckForUpdate(t *testing.T) {
	// nil state should trigger check
	if !ShouldCheckForUpdate(nil) {
		t.Error("ShouldCheckForUpdate(nil) should return true")
	}

	// Recent check should not trigger
	recent := &UpdateState{LastChecked: time.Now()}
	if ShouldCheckForUpdate(recent) {
		t.Error("ShouldCheckForUpdate with recent check should return false")
	}

	// Stale check should trigger
	stale := &UpdateState{LastChecked: time.Now().Add(-25 * time.Hour)}
	if !ShouldCheckForUpdate(stale) {
		t.Error("ShouldCheckForUpdate with stale check should return true")
	}
}

func TestPrintUpdateNotice(t *testing.T) {
	// Setup a temp config dir
	tmpDir := t.TempDir()
	t.Setenv("GLAB_CONFIG_DIR", tmpDir)

	// No state file — should not print
	var buf bytes.Buffer
	if PrintUpdateNotice(&buf, "0.0.11") {
		t.Error("PrintUpdateNotice should return false with no state file")
	}
	if buf.Len() != 0 {
		t.Error("PrintUpdateNotice should not write with no state file")
	}

	// Write state with newer version
	state := &UpdateState{
		LastChecked:   time.Now(),
		LatestVersion: "0.0.12",
		ReleaseURL:    "https://github.com/PhilipKram/Gitlab-CLI/releases/tag/v0.0.12",
	}
	if err := SaveStateFile(state); err != nil {
		t.Fatal(err)
	}

	buf.Reset()
	if !PrintUpdateNotice(&buf, "0.0.11") {
		t.Error("PrintUpdateNotice should return true when newer version exists")
	}
	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("0.0.12")) {
		t.Errorf("banner should contain new version, got: %s", output)
	}
	if !bytes.Contains([]byte(output), []byte("glab upgrade")) {
		t.Errorf("banner should mention 'glab upgrade', got: %s", output)
	}

	// Same version — should not print
	buf.Reset()
	if PrintUpdateNotice(&buf, "0.0.12") {
		t.Error("PrintUpdateNotice should return false when up-to-date")
	}
}

func TestStateFileRoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("GLAB_CONFIG_DIR", tmpDir)

	state := &UpdateState{
		LastChecked:   time.Now().Truncate(time.Second),
		LatestVersion: "1.2.3",
		ReleaseURL:    "https://example.com/release",
	}
	if err := SaveStateFile(state); err != nil {
		t.Fatal(err)
	}

	loaded, err := LoadStateFile()
	if err != nil {
		t.Fatal(err)
	}
	if loaded.LatestVersion != state.LatestVersion {
		t.Errorf("LatestVersion = %q, want %q", loaded.LatestVersion, state.LatestVersion)
	}
	if loaded.ReleaseURL != state.ReleaseURL {
		t.Errorf("ReleaseURL = %q, want %q", loaded.ReleaseURL, state.ReleaseURL)
	}
}

func TestVerifyChecksum(t *testing.T) {
	// Create a fake archive file
	tmpDir := t.TempDir()
	archiveContent := []byte("fake archive content for testing")
	archivePath := filepath.Join(tmpDir, "glab_1.0.0_linux_amd64.tar.gz")
	if err := os.WriteFile(archivePath, archiveContent, 0o644); err != nil {
		t.Fatal(err)
	}

	// Compute expected hash
	h := sha256.Sum256(archiveContent)
	expectedHash := fmt.Sprintf("%x", h)

	// Serve checksums.txt
	checksumContent := fmt.Sprintf("%s  glab_1.0.0_linux_amd64.tar.gz\nabcdef1234567890  glab_1.0.0_windows_amd64.zip\n", expectedHash)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, checksumContent)
	}))
	defer srv.Close()

	// Should pass with correct checksum
	if err := VerifyChecksum(archivePath, srv.URL+"/checksums.txt"); err != nil {
		t.Errorf("VerifyChecksum should pass: %v", err)
	}

	// Should fail with wrong content
	wrongPath := filepath.Join(tmpDir, "glab_1.0.0_linux_amd64.tar.gz")
	if err := os.WriteFile(wrongPath, []byte("tampered content"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := VerifyChecksum(wrongPath, srv.URL+"/checksums.txt"); err == nil {
		t.Error("VerifyChecksum should fail with tampered content")
	}

	// Empty checksum URL should be skipped
	if err := VerifyChecksum(archivePath, ""); err != nil {
		t.Errorf("VerifyChecksum with empty URL should skip: %v", err)
	}
}

func TestFindAssetURLs(t *testing.T) {
	release := &ReleaseInfo{
		TagName: "v1.0.0",
		HTMLURL: "https://github.com/PhilipKram/Gitlab-CLI/releases/tag/v1.0.0",
		Assets: []Asset{
			{Name: "glab_1.0.0_linux_amd64.tar.gz", BrowserDownloadURL: "https://example.com/glab_1.0.0_linux_amd64.tar.gz"},
			{Name: "glab_1.0.0_darwin_arm64.tar.gz", BrowserDownloadURL: "https://example.com/glab_1.0.0_darwin_arm64.tar.gz"},
			{Name: "checksums.txt", BrowserDownloadURL: "https://example.com/checksums.txt"},
		},
	}

	archiveURL, checksumURL, err := FindAssetURLs(release, "glab_1.0.0_linux_amd64.tar.gz")
	if err != nil {
		t.Fatal(err)
	}
	if archiveURL != "https://example.com/glab_1.0.0_linux_amd64.tar.gz" {
		t.Errorf("unexpected archive URL: %s", archiveURL)
	}
	if checksumURL != "https://example.com/checksums.txt" {
		t.Errorf("unexpected checksum URL: %s", checksumURL)
	}

	// Missing archive
	_, _, err = FindAssetURLs(release, "glab_1.0.0_freebsd_amd64.tar.gz")
	if err == nil {
		t.Error("FindAssetURLs should error for missing archive")
	}
}
