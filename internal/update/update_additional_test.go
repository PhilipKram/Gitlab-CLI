package update

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestCheckAndCache_Success(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("GLAB_CONFIG_DIR", tmpDir)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := ReleaseInfo{
			TagName: "v2.0.0",
			HTMLURL: "https://github.com/PhilipKram/Gitlab-CLI/releases/tag/v2.0.0",
			Assets:  []Asset{},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	t.Cleanup(srv.Close)

	origTransport := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = origTransport })
	http.DefaultTransport = &testRedirectTransport{target: srv.URL}

	// Run CheckAndCache - should write a state file
	CheckAndCache("1.0.0")

	// Verify state file was created
	state, err := LoadStateFile()
	if err != nil {
		t.Fatalf("LoadStateFile after CheckAndCache: %v", err)
	}
	if state.LatestVersion != "2.0.0" {
		t.Errorf("LatestVersion = %q, want %q", state.LatestVersion, "2.0.0")
	}
}

func TestCheckAndCache_SkipsWhenRecent(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("GLAB_CONFIG_DIR", tmpDir)

	// Write a recent state file
	state := &UpdateState{
		LastChecked:   time.Now(),
		LatestVersion: "1.5.0",
		ReleaseURL:    "https://example.com/release",
	}
	if err := SaveStateFile(state); err != nil {
		t.Fatal(err)
	}

	// The server should NOT be called since state is recent
	serverCalled := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serverCalled = true
		w.WriteHeader(500)
	}))
	t.Cleanup(srv.Close)

	origTransport := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = origTransport })
	http.DefaultTransport = &testRedirectTransport{target: srv.URL}

	CheckAndCache("1.0.0")

	if serverCalled {
		t.Error("expected server NOT to be called when state is recent")
	}
}

func TestCheckAndCache_SilentOnError(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("GLAB_CONFIG_DIR", tmpDir)

	// Server that always errors
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	t.Cleanup(srv.Close)

	origTransport := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = origTransport })
	http.DefaultTransport = &testRedirectTransport{target: srv.URL}

	// Should not panic even though server errors
	CheckAndCache("1.0.0")
}

func TestDetectInstallMethod(t *testing.T) {
	// We can't control the executable path in tests, but we can verify
	// the function doesn't panic and returns a valid InstallMethod
	method := DetectInstallMethod()
	switch method {
	case InstallBinary, InstallBrew, InstallDeb, InstallRPM, InstallGoBuild:
		// valid
	default:
		t.Errorf("DetectInstallMethod() returned unexpected value: %d", method)
	}
}

func TestExtractFromZip_Success(t *testing.T) {
	tmpDir := t.TempDir()
	archivePath := filepath.Join(tmpDir, "test.zip")

	binaryContent := []byte("#!/bin/bash\necho hello from zip")
	binaryName := "glab"
	if runtime.GOOS == "windows" {
		binaryName = "glab.exe"
	}

	// Create a zip file with glab binary
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	fw, err := zw.Create(binaryName)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fw.Write(binaryContent); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(archivePath, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}

	destDir := t.TempDir()
	binaryPath, err := ExtractBinary(archivePath, destDir)
	if err != nil {
		t.Fatalf("ExtractBinary from zip: %v", err)
	}

	extracted, err := os.ReadFile(binaryPath)
	if err != nil {
		t.Fatalf("reading extracted binary: %v", err)
	}
	if !bytes.Equal(extracted, binaryContent) {
		t.Error("extracted binary content does not match")
	}
}

func TestExtractFromZip_NoBinary(t *testing.T) {
	tmpDir := t.TempDir()
	archivePath := filepath.Join(tmpDir, "test.zip")

	// Create a zip without the glab binary
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	fw, err := zw.Create("README.md")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fw.Write([]byte("readme")); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(archivePath, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}

	destDir := t.TempDir()
	_, err = ExtractBinary(archivePath, destDir)
	if err == nil {
		t.Fatal("expected error when binary not found in zip")
	}
	if !strings.Contains(err.Error(), "not found in archive") {
		t.Errorf("expected 'not found in archive' error, got: %v", err)
	}
}

func TestExtractFromZip_InvalidZip(t *testing.T) {
	tmpDir := t.TempDir()
	archivePath := filepath.Join(tmpDir, "test.zip")

	if err := os.WriteFile(archivePath, []byte("not a real zip"), 0o644); err != nil {
		t.Fatal(err)
	}

	destDir := t.TempDir()
	_, err := extractFromZip(archivePath, destDir)
	if err == nil {
		t.Fatal("expected error for invalid zip file")
	}
}

func TestReplaceBinary_NonExistentNew(t *testing.T) {
	tmpDir := t.TempDir()

	// Test that ReplaceBinary handles errors when new binary path doesn't exist
	err := ReplaceBinary(filepath.Join(tmpDir, "nonexistent"))
	// The function may succeed or fail depending on os.Executable resolution
	// Just verify it doesn't panic
	_ = err
}

func TestExtractBinary_TarGz_NestedPath(t *testing.T) {
	// Create a tar.gz archive with binary in a subdirectory
	tmpDir := t.TempDir()
	archivePath := filepath.Join(tmpDir, "test.tar.gz")

	binaryContent := []byte("#!/bin/bash\necho nested binary")
	binaryName := "glab"
	if runtime.GOOS == "windows" {
		binaryName = "glab.exe"
	}

	var archiveBuf bytes.Buffer
	gzWriter, _ := newGzipWriter(&archiveBuf)
	tw := newTarWriter(gzWriter)

	// Write binary in a nested path
	writeTarEntry(t, tw, "glab_1.0.0/"+binaryName, binaryContent)
	tw.Close()
	gzWriter.Close()

	if err := os.WriteFile(archivePath, archiveBuf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}

	destDir := t.TempDir()
	binaryPath, err := ExtractBinary(archivePath, destDir)
	if err != nil {
		t.Fatalf("ExtractBinary with nested path: %v", err)
	}

	extracted, err := os.ReadFile(binaryPath)
	if err != nil {
		t.Fatalf("reading extracted binary: %v", err)
	}
	if !bytes.Equal(extracted, binaryContent) {
		t.Error("extracted binary content does not match")
	}
}

func TestExtractBinary_NonExistentFile(t *testing.T) {
	destDir := t.TempDir()
	_, err := ExtractBinary("/nonexistent/path/file.tar.gz", destDir)
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

func TestArchiveName_WithVPrefix(t *testing.T) {
	name := ArchiveName("v2.5.1")
	if strings.HasPrefix(name, "glab_v") {
		t.Errorf("ArchiveName should strip v prefix, got %q", name)
	}
	if !strings.HasPrefix(name, "glab_2.5.1_") {
		t.Errorf("ArchiveName = %q, expected to start with 'glab_2.5.1_'", name)
	}
}
