package update

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// testRedirectTransport rewrites all requests to point at the test server.
type testRedirectTransport struct {
	target string
}

func (t *testRedirectTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.URL.Scheme = "http"
	req.URL.Host = strings.TrimPrefix(t.target, "http://")
	transport := &http.Transport{}
	return transport.RoundTrip(req)
}

func newGzipWriter(w io.Writer) (*gzip.Writer, error) {
	return gzip.NewWriterLevel(w, gzip.DefaultCompression)
}

func newTarWriter(w io.Writer) *tar.Writer {
	return tar.NewWriter(w)
}

func writeTarEntry(t *testing.T, tw *tar.Writer, name string, content []byte) {
	t.Helper()
	hdr := &tar.Header{
		Name:     name,
		Mode:     0o755,
		Size:     int64(len(content)),
		Typeflag: tar.TypeReg,
	}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatalf("writing tar header: %v", err)
	}
	if _, err := tw.Write(content); err != nil {
		t.Fatalf("writing tar content: %v", err)
	}
}

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
		_, _ = fmt.Fprint(w, checksumContent)
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

	// Empty checksum URL should return error (mandatory verification)
	if err := VerifyChecksum(archivePath, ""); err == nil {
		t.Error("VerifyChecksum should fail when checksumURL is empty (mandatory verification)")
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

func TestValidateAssetURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		// Valid URLs
		{
			name:    "valid github.com URL",
			url:     "https://github.com/PhilipKram/Gitlab-CLI/releases/download/v1.0.0/glab_1.0.0_linux_amd64.tar.gz",
			wantErr: false,
		},
		{
			name:    "valid api.github.com URL",
			url:     "https://api.github.com/repos/PhilipKram/Gitlab-CLI/releases/assets/12345",
			wantErr: false,
		},
		{
			name:    "valid github.com subdomain releases.github.com",
			url:     "https://releases.github.com/PhilipKram/Gitlab-CLI/glab.tar.gz",
			wantErr: false,
		},
		{
			name:    "valid github.com with query params",
			url:     "https://github.com/PhilipKram/Gitlab-CLI/releases/download/v1.0.0/glab.tar.gz?raw=true",
			wantErr: false,
		},

		// Invalid URLs - empty or malformed
		{
			name:    "empty URL",
			url:     "",
			wantErr: true,
		},
		{
			name:    "malformed URL",
			url:     "not a url at all",
			wantErr: true,
		},
		{
			name:    "malformed URL with spaces",
			url:     "https://github com/file.tar.gz",
			wantErr: true,
		},

		// Invalid URLs - wrong scheme
		{
			name:    "http instead of https",
			url:     "http://github.com/PhilipKram/Gitlab-CLI/releases/download/v1.0.0/glab.tar.gz",
			wantErr: true,
		},
		{
			name:    "ftp scheme",
			url:     "ftp://github.com/file.tar.gz",
			wantErr: true,
		},
		{
			name:    "file scheme",
			url:     "file:///etc/passwd",
			wantErr: true,
		},

		// Invalid URLs - wrong domain
		{
			name:    "malicious domain",
			url:     "https://evil.com/glab_1.0.0_linux_amd64.tar.gz",
			wantErr: true,
		},
		{
			name:    "typosquatting githuh.com",
			url:     "https://githuh.com/PhilipKram/Gitlab-CLI/releases/download/v1.0.0/glab.tar.gz",
			wantErr: true,
		},
		{
			name:    "typosquatting github.co",
			url:     "https://github.co/PhilipKram/Gitlab-CLI/releases/download/v1.0.0/glab.tar.gz",
			wantErr: true,
		},
		{
			name:    "subdomain spoofing github.com.evil.com",
			url:     "https://github.com.evil.com/malware.tar.gz",
			wantErr: true,
		},
		{
			name:    "localhost",
			url:     "https://localhost:8080/glab.tar.gz",
			wantErr: true,
		},
		{
			name:    "IP address",
			url:     "https://192.168.1.1/glab.tar.gz",
			wantErr: true,
		},

		// Edge cases
		{
			name:    "URL with user info",
			url:     "https://user:pass@github.com/file.tar.gz",
			wantErr: false, // Valid HTTPS github.com URL with auth
		},
		{
			name:    "URL with fragment",
			url:     "https://github.com/PhilipKram/Gitlab-CLI/releases/download/v1.0.0/glab.tar.gz#section",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAssetURL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateAssetURL(%q) error = %v, wantErr %v", tt.url, err, tt.wantErr)
			}
		})
	}
}

func TestCheckLatestRelease_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{
			"tag_name": "v2.0.0",
			"html_url": "https://github.com/PhilipKram/Gitlab-CLI/releases/tag/v2.0.0",
			"assets": [
				{"name": "glab_2.0.0_linux_amd64.tar.gz", "browser_download_url": "https://example.com/glab.tar.gz"},
				{"name": "checksums.txt", "browser_download_url": "https://example.com/checksums.txt"}
			]
		}`)
	}))
	defer srv.Close()

	// Temporarily override releaseURL by replacing the HTTP client
	origTransport := http.DefaultTransport
	defer func() { http.DefaultTransport = origTransport }()
	http.DefaultTransport = &testRedirectTransport{target: srv.URL}

	result, err := CheckLatestRelease("1.0.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.LatestVersion != "2.0.0" {
		t.Errorf("LatestVersion = %q, want %q", result.LatestVersion, "2.0.0")
	}
	if !result.IsNewer {
		t.Error("expected IsNewer to be true")
	}
	if result.CurrentVersion != "1.0.0" {
		t.Errorf("CurrentVersion = %q, want %q", result.CurrentVersion, "1.0.0")
	}
	if result.Release == nil {
		t.Fatal("Release should not be nil")
	}
	if len(result.Release.Assets) != 2 {
		t.Errorf("expected 2 assets, got %d", len(result.Release.Assets))
	}
}

func TestCheckLatestRelease_RateLimit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(429)
	}))
	defer srv.Close()

	origTransport := http.DefaultTransport
	defer func() { http.DefaultTransport = origTransport }()
	http.DefaultTransport = &testRedirectTransport{target: srv.URL}

	_, err := CheckLatestRelease("1.0.0")
	if err == nil {
		t.Error("expected error for rate limited response")
	}
	if !strings.Contains(err.Error(), "rate limit") {
		t.Errorf("expected rate limit error, got: %v", err)
	}
}

func TestCheckLatestRelease_ForbiddenRateLimit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(403)
	}))
	defer srv.Close()

	origTransport := http.DefaultTransport
	defer func() { http.DefaultTransport = origTransport }()
	http.DefaultTransport = &testRedirectTransport{target: srv.URL}

	_, err := CheckLatestRelease("1.0.0")
	if err == nil {
		t.Error("expected error for 403 response")
	}
	if !strings.Contains(err.Error(), "rate limit") {
		t.Errorf("expected rate limit error, got: %v", err)
	}
}

func TestCheckLatestRelease_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer srv.Close()

	origTransport := http.DefaultTransport
	defer func() { http.DefaultTransport = origTransport }()
	http.DefaultTransport = &testRedirectTransport{target: srv.URL}

	_, err := CheckLatestRelease("1.0.0")
	if err == nil {
		t.Error("expected error for 500 response")
	}
	if !strings.Contains(err.Error(), "HTTP 500") {
		t.Errorf("expected HTTP 500 error, got: %v", err)
	}
}

func TestCheckLatestRelease_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, "not valid json")
	}))
	defer srv.Close()

	origTransport := http.DefaultTransport
	defer func() { http.DefaultTransport = origTransport }()
	http.DefaultTransport = &testRedirectTransport{target: srv.URL}

	_, err := CheckLatestRelease("1.0.0")
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
	if !strings.Contains(err.Error(), "parsing release info") {
		t.Errorf("expected parsing error, got: %v", err)
	}
}

func TestCheckLatestRelease_SameVersion(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"tag_name": "v1.0.0", "html_url": "https://example.com/release", "assets": []}`)
	}))
	defer srv.Close()

	origTransport := http.DefaultTransport
	defer func() { http.DefaultTransport = origTransport }()
	http.DefaultTransport = &testRedirectTransport{target: srv.URL}

	result, err := CheckLatestRelease("1.0.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsNewer {
		t.Error("expected IsNewer to be false for same version")
	}
}

func TestLoadStateFile_NoFile(t *testing.T) {
	// Remove any existing state file and restore after test
	path := stateFilePath()
	origData, origErr := os.ReadFile(path)
	_ = os.Remove(path)
	t.Cleanup(func() {
		if origErr == nil {
			_ = os.WriteFile(path, origData, 0o644)
		}
	})

	_, err := LoadStateFile()
	if err == nil {
		t.Error("expected error when state file does not exist")
	}
}

func TestLoadStateFile_InvalidJSON(t *testing.T) {
	// Backup existing state file and restore after test
	path := stateFilePath()
	origData, origErr := os.ReadFile(path)
	t.Cleanup(func() {
		if origErr == nil {
			_ = os.WriteFile(path, origData, 0o644)
		} else {
			_ = os.Remove(path)
		}
	})

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("not json"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadStateFile()
	if err == nil {
		t.Error("expected error for invalid JSON in state file")
	}
}

func TestPrintUpdateNotice_EmptyLatestVersion(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("GLAB_CONFIG_DIR", tmpDir)

	state := &UpdateState{
		LastChecked:   time.Now(),
		LatestVersion: "",
		ReleaseURL:    "https://example.com/release",
	}
	if err := SaveStateFile(state); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if PrintUpdateNotice(&buf, "1.0.0") {
		t.Error("PrintUpdateNotice should return false when LatestVersion is empty")
	}
	if buf.Len() != 0 {
		t.Error("PrintUpdateNotice should not write when LatestVersion is empty")
	}
}

func TestPrintUpdateNotice_OlderVersion(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("GLAB_CONFIG_DIR", tmpDir)

	state := &UpdateState{
		LastChecked:   time.Now(),
		LatestVersion: "0.9.0",
		ReleaseURL:    "https://example.com/release",
	}
	if err := SaveStateFile(state); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if PrintUpdateNotice(&buf, "1.0.0") {
		t.Error("PrintUpdateNotice should return false when latest is older than current")
	}
}

func TestFindAssetURLs_NoChecksum(t *testing.T) {
	release := &ReleaseInfo{
		TagName: "v1.0.0",
		HTMLURL: "https://example.com/release",
		Assets: []Asset{
			{Name: "glab_1.0.0_linux_amd64.tar.gz", BrowserDownloadURL: "https://example.com/archive.tar.gz"},
		},
	}

	archiveURL, checksumURL, err := FindAssetURLs(release, "glab_1.0.0_linux_amd64.tar.gz")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if archiveURL != "https://example.com/archive.tar.gz" {
		t.Errorf("unexpected archive URL: %s", archiveURL)
	}
	if checksumURL != "" {
		t.Errorf("expected empty checksum URL, got %q", checksumURL)
	}
}

func TestFindAssetURLs_EmptyAssets(t *testing.T) {
	release := &ReleaseInfo{
		TagName: "v1.0.0",
		HTMLURL: "https://example.com/release",
		Assets:  []Asset{},
	}

	_, _, err := FindAssetURLs(release, "glab_1.0.0_linux_amd64.tar.gz")
	if err == nil {
		t.Error("expected error for empty assets")
	}
}

func TestExtractBinary_TarGz(t *testing.T) {
	// Create a tar.gz archive with a glab binary
	tmpDir := t.TempDir()
	archivePath := filepath.Join(tmpDir, "test.tar.gz")

	binaryContent := []byte("#!/bin/bash\necho hello")

	// Build tar.gz in memory
	var archiveBuf bytes.Buffer
	gzWriter, _ := newGzipWriter(&archiveBuf)
	tw := newTarWriter(gzWriter)

	binaryName := "glab"
	if runtime.GOOS == "windows" {
		binaryName = "glab.exe"
	}

	writeTarEntry(t, tw, binaryName, binaryContent)
	tw.Close()
	gzWriter.Close()

	if err := os.WriteFile(archivePath, archiveBuf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}

	destDir := t.TempDir()
	binaryPath, err := ExtractBinary(archivePath, destDir)
	if err != nil {
		t.Fatalf("ExtractBinary failed: %v", err)
	}

	extracted, err := os.ReadFile(binaryPath)
	if err != nil {
		t.Fatalf("reading extracted binary: %v", err)
	}
	if !bytes.Equal(extracted, binaryContent) {
		t.Error("extracted binary content does not match")
	}
}

func TestExtractBinary_TarGz_NoBinary(t *testing.T) {
	tmpDir := t.TempDir()
	archivePath := filepath.Join(tmpDir, "test.tar.gz")

	var archiveBuf bytes.Buffer
	gzWriter, _ := newGzipWriter(&archiveBuf)
	tw := newTarWriter(gzWriter)

	// Write a file that is NOT the glab binary
	writeTarEntry(t, tw, "README.md", []byte("readme"))
	tw.Close()
	gzWriter.Close()

	if err := os.WriteFile(archivePath, archiveBuf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}

	destDir := t.TempDir()
	_, err := ExtractBinary(archivePath, destDir)
	if err == nil {
		t.Error("expected error when binary not found in archive")
	}
	if !strings.Contains(err.Error(), "not found in archive") {
		t.Errorf("expected 'not found in archive' error, got: %v", err)
	}
}

func TestExtractBinary_InvalidTarGz(t *testing.T) {
	tmpDir := t.TempDir()
	archivePath := filepath.Join(tmpDir, "test.tar.gz")

	if err := os.WriteFile(archivePath, []byte("not a real archive"), 0o644); err != nil {
		t.Fatal(err)
	}

	destDir := t.TempDir()
	_, err := ExtractBinary(archivePath, destDir)
	if err == nil {
		t.Error("expected error for invalid tar.gz")
	}
}

func TestVerifyChecksum_NoMatchingEntry(t *testing.T) {
	tmpDir := t.TempDir()
	archivePath := filepath.Join(tmpDir, "glab_1.0.0_linux_amd64.tar.gz")
	if err := os.WriteFile(archivePath, []byte("content"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Checksums file with no matching entry
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "abcdef1234567890  glab_1.0.0_windows_amd64.zip\n")
	}))
	defer srv.Close()

	err := VerifyChecksum(archivePath, srv.URL+"/checksums.txt")
	if err == nil {
		t.Error("expected error when no checksum entry matches")
	}
	if !strings.Contains(err.Error(), "no checksum found") {
		t.Errorf("expected 'no checksum found' error, got: %v", err)
	}
}

func TestVerifyChecksum_ServerError(t *testing.T) {
	tmpDir := t.TempDir()
	archivePath := filepath.Join(tmpDir, "glab_1.0.0_linux_amd64.tar.gz")
	if err := os.WriteFile(archivePath, []byte("content"), 0o644); err != nil {
		t.Fatal(err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer srv.Close()

	err := VerifyChecksum(archivePath, srv.URL+"/checksums.txt")
	if err == nil {
		t.Error("expected error for server error response")
	}
}

func TestCompareVersions_EdgeCases(t *testing.T) {
	tests := []struct {
		current string
		latest  string
		want    bool
	}{
		{"1.0", "1.0.1", true},     // missing patch treated as 0
		{"1", "2", true},           // single component
		{"", "", false},            // empty strings
		{"0", "1", true},           // single digit
		{"abc", "def", false},      // non-numeric (Atoi returns 0)
		{"1.2.3.4", "1.2.3", true}, // "3.4" parsed as 0 by Atoi, so 3 > 0
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

func TestDownloadAsset_Success(t *testing.T) {
	content := "downloaded binary content"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, content)
	}))
	defer srv.Close()

	tmpDir := t.TempDir()

	// We need a valid github.com URL, but redirect to our test server
	origTransport := http.DefaultTransport
	defer func() { http.DefaultTransport = origTransport }()
	http.DefaultTransport = &testRedirectTransport{target: srv.URL}

	path, err := DownloadAsset("https://github.com/test/repo/releases/download/v1.0.0/glab.tar.gz", tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading downloaded file: %v", err)
	}
	if string(data) != content {
		t.Errorf("downloaded content = %q, want %q", string(data), content)
	}
}

func TestDownloadAsset_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	}))
	defer srv.Close()

	tmpDir := t.TempDir()

	origTransport := http.DefaultTransport
	defer func() { http.DefaultTransport = origTransport }()
	http.DefaultTransport = &testRedirectTransport{target: srv.URL}

	_, err := DownloadAsset("https://github.com/test/repo/releases/download/v1.0.0/glab.tar.gz", tmpDir)
	if err == nil {
		t.Error("expected error for HTTP 404")
	}
	if !strings.Contains(err.Error(), "HTTP 404") {
		t.Errorf("expected HTTP 404 in error, got: %v", err)
	}
}

func TestDownloadAssetURLValidation(t *testing.T) {
	// Test that DownloadAsset rejects invalid URLs before attempting download
	tmpDir := t.TempDir()

	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{
			name:    "rejects empty URL",
			url:     "",
			wantErr: true,
		},
		{
			name:    "rejects http URL",
			url:     "http://github.com/file.tar.gz",
			wantErr: true,
		},
		{
			name:    "rejects malicious domain",
			url:     "https://evil.com/malware.tar.gz",
			wantErr: true,
		},
		{
			name:    "rejects typosquatting domain",
			url:     "https://githuh.com/file.tar.gz",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := DownloadAsset(tt.url, tmpDir)
			if (err != nil) != tt.wantErr {
				t.Errorf("DownloadAsset(%q) error = %v, wantErr %v", tt.url, err, tt.wantErr)
			}
			if err != nil && !bytes.Contains([]byte(err.Error()), []byte("URL validation failed")) {
				t.Errorf("DownloadAsset should return URL validation error, got: %v", err)
			}
		})
	}

	// Test that valid GitHub URL passes validation (but may fail network)
	// We use a URL that will fail network request but pass validation
	validURL := "https://github.com/nonexistent/nonexistent/releases/download/v0.0.0/nonexistent.tar.gz"
	_, err := DownloadAsset(validURL, tmpDir)
	// We expect a network error, not a validation error
	if err != nil && bytes.Contains([]byte(err.Error()), []byte("URL validation failed")) {
		t.Errorf("DownloadAsset should not return validation error for valid GitHub URL, got: %v", err)
	}
}
