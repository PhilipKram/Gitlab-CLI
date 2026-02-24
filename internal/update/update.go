package update

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/PhilipKram/gitlab-cli/internal/config"
)

const (
	releaseURL    = "https://api.github.com/repos/PhilipKram/Gitlab-CLI/releases/latest"
	stateFileName = "update-check.json"
	checkInterval = 24 * time.Hour
)

// InstallMethod represents how glab was installed.
type InstallMethod int

const (
	InstallBinary  InstallMethod = iota // direct binary download
	InstallBrew                         // Homebrew
	InstallDeb                          // Debian package
	InstallRPM                          // RPM package
	InstallGoBuild                      // go install / dev build
)

// ReleaseInfo holds GitHub release API response fields.
type ReleaseInfo struct {
	TagName string  `json:"tag_name"`
	Assets  []Asset `json:"assets"`
	HTMLURL string  `json:"html_url"`
}

// Asset holds GitHub release asset fields.
type Asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

// CheckResult holds the result of a version check.
type CheckResult struct {
	CurrentVersion string
	LatestVersion  string
	IsNewer        bool
	ReleaseURL     string
	Release        *ReleaseInfo
}

// UpdateState is persisted to disk to cache version check results.
type UpdateState struct {
	LastChecked   time.Time `json:"last_checked"`
	LatestVersion string    `json:"latest_version"`
	ReleaseURL    string    `json:"release_url"`
}

// CompareVersions returns true if latest is newer than current.
// Both versions may optionally have a "v" prefix.
func CompareVersions(current, latest string) bool {
	current = strings.TrimPrefix(current, "v")
	latest = strings.TrimPrefix(latest, "v")

	curParts := strings.SplitN(current, ".", 3)
	latParts := strings.SplitN(latest, ".", 3)

	for i := 0; i < 3; i++ {
		var c, l int
		if i < len(curParts) {
			c, _ = strconv.Atoi(curParts[i])
		}
		if i < len(latParts) {
			l, _ = strconv.Atoi(latParts[i])
		}
		if l > c {
			return true
		}
		if l < c {
			return false
		}
	}
	return false
}

// stateFilePath returns the full path to the update state file.
func stateFilePath() string {
	return filepath.Join(config.ConfigDir(), stateFileName)
}

// LoadStateFile reads the cached update state from disk.
func LoadStateFile() (*UpdateState, error) {
	data, err := os.ReadFile(stateFilePath())
	if err != nil {
		return nil, err
	}
	var state UpdateState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}
	return &state, nil
}

// SaveStateFile writes the update state to disk.
func SaveStateFile(state *UpdateState) error {
	dir := config.ConfigDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(stateFilePath(), data, 0o644)
}

// ShouldCheckForUpdate returns true if the cache is stale or missing.
func ShouldCheckForUpdate(state *UpdateState) bool {
	if state == nil {
		return true
	}
	return time.Since(state.LastChecked) > checkInterval
}

// CheckLatestRelease queries GitHub for the latest release and compares versions.
func CheckLatestRelease(currentVersion string) (*CheckResult, error) {
	req, err := http.NewRequest("GET", releaseURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("checking for updates: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 403 || resp.StatusCode == 429 {
		return nil, fmt.Errorf("GitHub API rate limit exceeded (HTTP %d), try again later", resp.StatusCode)
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("GitHub API returned HTTP %d", resp.StatusCode)
	}

	var release ReleaseInfo
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("parsing release info: %w", err)
	}

	latestVersion := strings.TrimPrefix(release.TagName, "v")
	result := &CheckResult{
		CurrentVersion: currentVersion,
		LatestVersion:  latestVersion,
		IsNewer:        CompareVersions(currentVersion, latestVersion),
		ReleaseURL:     release.HTMLURL,
		Release:        &release,
	}
	return result, nil
}

// CheckAndCache checks for updates and writes the result to the state file.
// Intended to run as a goroutine; silently ignores all errors.
func CheckAndCache(currentVersion string) {
	state, _ := LoadStateFile()
	if !ShouldCheckForUpdate(state) {
		return
	}

	result, err := CheckLatestRelease(currentVersion)
	if err != nil {
		return
	}

	newState := &UpdateState{
		LastChecked:   time.Now(),
		LatestVersion: result.LatestVersion,
		ReleaseURL:    result.ReleaseURL,
	}
	_ = SaveStateFile(newState)
}

// PrintUpdateNotice reads the cached state and prints an update banner if
// a newer version is available. Returns true if a banner was printed.
func PrintUpdateNotice(out io.Writer, currentVersion string) bool {
	state, err := LoadStateFile()
	if err != nil || state == nil {
		return false
	}
	if state.LatestVersion == "" {
		return false
	}
	if !CompareVersions(currentVersion, state.LatestVersion) {
		return false
	}

	fmt.Fprintf(out, "\nA new version of glab is available: v%s → v%s\n",
		strings.TrimPrefix(currentVersion, "v"),
		strings.TrimPrefix(state.LatestVersion, "v"))
	fmt.Fprintf(out, "Run `glab upgrade` to update, or download from:\n")
	fmt.Fprintf(out, "%s\n\n", state.ReleaseURL)
	return true
}

// DetectInstallMethod determines how glab was installed.
func DetectInstallMethod() InstallMethod {
	exe, err := os.Executable()
	if err != nil {
		return InstallBinary
	}
	resolved, err := filepath.EvalSymlinks(exe)
	if err != nil {
		resolved = exe
	}

	// Homebrew detection
	if strings.Contains(resolved, "/Cellar/") || strings.Contains(resolved, "/homebrew/") {
		return InstallBrew
	}
	if prefix := os.Getenv("HOMEBREW_PREFIX"); prefix != "" && strings.HasPrefix(resolved, prefix) {
		return InstallBrew
	}

	// System package detection
	if strings.HasPrefix(resolved, "/usr/bin/") {
		// Check for dpkg
		if _, err := os.Stat("/var/lib/dpkg/info/glab.list"); err == nil {
			return InstallDeb
		}
		// Check for rpm
		if _, err := os.Stat("/var/lib/rpm"); err == nil {
			return InstallRPM
		}
	}

	return InstallBinary
}

// ArchiveName builds the expected archive filename for the current platform.
func ArchiveName(version string) string {
	version = strings.TrimPrefix(version, "v")
	ext := "tar.gz"
	if runtime.GOOS == "windows" {
		ext = "zip"
	}
	return fmt.Sprintf("glab_%s_%s_%s.%s", version, runtime.GOOS, runtime.GOARCH, ext)
}

// FindAssetURLs locates the archive and checksums URLs from a release's assets.
func FindAssetURLs(release *ReleaseInfo, archiveName string) (archiveURL, checksumURL string, err error) {
	for _, a := range release.Assets {
		if a.Name == archiveName {
			archiveURL = a.BrowserDownloadURL
		}
		if a.Name == "checksums.txt" {
			checksumURL = a.BrowserDownloadURL
		}
	}
	if archiveURL == "" {
		return "", "", fmt.Errorf("no release asset found for %s (OS=%s, Arch=%s)\nCheck available assets at: %s",
			archiveName, runtime.GOOS, runtime.GOARCH, release.HTMLURL)
	}
	return archiveURL, checksumURL, nil
}

// DownloadAsset downloads a URL to a file in destDir and returns the file path.
func DownloadAsset(url, destDir string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("downloading %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("downloading %s: HTTP %d", url, resp.StatusCode)
	}

	name := filepath.Base(url)
	destPath := filepath.Join(destDir, name)
	f, err := os.Create(destPath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	if _, err := io.Copy(f, resp.Body); err != nil {
		return "", fmt.Errorf("writing %s: %w", destPath, err)
	}
	return destPath, nil
}

// VerifyChecksum downloads checksums.txt, computes SHA256 of the archive,
// and verifies the hash matches.
func VerifyChecksum(archivePath, checksumURL string) error {
	if checksumURL == "" {
		return nil // no checksum file available, skip verification
	}

	// Download checksums.txt
	resp, err := http.Get(checksumURL)
	if err != nil {
		return fmt.Errorf("downloading checksums: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("downloading checksums: HTTP %d", resp.StatusCode)
	}
	checksumData, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading checksums: %w", err)
	}

	// Find expected hash for our archive
	archiveName := filepath.Base(archivePath)
	var expectedHash string
	for _, line := range strings.Split(string(checksumData), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[1] == archiveName {
			expectedHash = fields[0]
			break
		}
	}
	if expectedHash == "" {
		return fmt.Errorf("no checksum found for %s in checksums.txt", archiveName)
	}

	// Compute actual hash
	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return fmt.Errorf("computing checksum: %w", err)
	}
	actualHash := hex.EncodeToString(h.Sum(nil))

	if actualHash != expectedHash {
		return fmt.Errorf("checksum mismatch for %s\n  expected: %s\n  actual:   %s", archiveName, expectedHash, actualHash)
	}
	return nil
}

// ExtractBinary extracts the glab binary from a tar.gz or zip archive.
func ExtractBinary(archivePath, destDir string) (string, error) {
	if strings.HasSuffix(archivePath, ".zip") {
		return extractFromZip(archivePath, destDir)
	}
	return extractFromTarGz(archivePath, destDir)
}

func extractFromTarGz(archivePath, destDir string) (string, error) {
	f, err := os.Open(archivePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return "", fmt.Errorf("opening gzip: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	binaryName := "glab"
	if runtime.GOOS == "windows" {
		binaryName = "glab.exe"
	}

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("reading tar: %w", err)
		}

		if filepath.Base(hdr.Name) == binaryName && hdr.Typeflag == tar.TypeReg {
			destPath := filepath.Join(destDir, binaryName)
			out, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
			if err != nil {
				return "", err
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return "", fmt.Errorf("extracting %s: %w", binaryName, err)
			}
			out.Close()
			return destPath, nil
		}
	}
	return "", fmt.Errorf("binary %s not found in archive", binaryName)
}

func extractFromZip(archivePath, destDir string) (string, error) {
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return "", fmt.Errorf("opening zip: %w", err)
	}
	defer r.Close()

	binaryName := "glab"
	if runtime.GOOS == "windows" {
		binaryName = "glab.exe"
	}

	for _, zf := range r.File {
		if filepath.Base(zf.Name) != binaryName {
			continue
		}
		rc, err := zf.Open()
		if err != nil {
			return "", err
		}
		destPath := filepath.Join(destDir, binaryName)
		out, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
		if err != nil {
			rc.Close()
			return "", err
		}
		if _, err := io.Copy(out, rc); err != nil {
			out.Close()
			rc.Close()
			return "", fmt.Errorf("extracting %s: %w", binaryName, err)
		}
		out.Close()
		rc.Close()
		return destPath, nil
	}
	return "", fmt.Errorf("binary %s not found in archive", binaryName)
}

// ReplaceBinary replaces the currently running binary with a new one.
func ReplaceBinary(newBinaryPath string) error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("locating current binary: %w", err)
	}
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return fmt.Errorf("resolving binary path: %w", err)
	}

	oldPath := exe + ".old"

	// Rename current binary out of the way
	if err := os.Rename(exe, oldPath); err != nil {
		return fmt.Errorf("backing up current binary: %w\nYou may need to run with elevated permissions (e.g. sudo)", err)
	}

	// Move new binary into place
	if err := os.Rename(newBinaryPath, exe); err != nil {
		// Try to restore old binary
		_ = os.Rename(oldPath, exe)
		return fmt.Errorf("installing new binary: %w\nYou may need to run with elevated permissions (e.g. sudo)", err)
	}

	// Clean up old binary
	_ = os.Remove(oldPath)
	return nil
}
