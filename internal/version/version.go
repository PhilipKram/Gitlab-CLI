package version

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Version represents a semantic version with major, minor, and patch components.
type Version struct {
	Major int
	Minor int
	Patch int
	// Original stores the original version string (e.g., "16.7.0-ee")
	Original string
}

// String returns the string representation of the version.
func (v Version) String() string {
	if v.Original != "" {
		return v.Original
	}
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

// versionRegex matches GitLab version strings like "16.7.0-ee", "15.11.3-ce", "17.0.0"
var versionRegex = regexp.MustCompile(`^v?(\d+)\.(\d+)\.(\d+)(?:-[a-z]+)?`)

// ParseVersion extracts semantic version from GitLab version strings.
// Examples:
//   - "16.7.0-ee" -> Version{16, 7, 0, "16.7.0-ee"}
//   - "15.11.3-ce" -> Version{15, 11, 3, "15.11.3-ce"}
//   - "17.0.0" -> Version{17, 0, 0, "17.0.0"}
//   - "v16.7.0" -> Version{16, 7, 0, "v16.7.0"}
//
// Returns an error if the version string is malformed or empty.
func ParseVersion(versionStr string) (Version, error) {
	versionStr = strings.TrimSpace(versionStr)
	if versionStr == "" {
		return Version{}, fmt.Errorf("version string is empty")
	}

	matches := versionRegex.FindStringSubmatch(versionStr)
	if len(matches) < 4 {
		return Version{}, fmt.Errorf("invalid version format: %q (expected format: X.Y.Z or X.Y.Z-suffix)", versionStr)
	}

	major, err := strconv.Atoi(matches[1])
	if err != nil {
		return Version{}, fmt.Errorf("invalid major version: %v", err)
	}

	minor, err := strconv.Atoi(matches[2])
	if err != nil {
		return Version{}, fmt.Errorf("invalid minor version: %v", err)
	}

	patch, err := strconv.Atoi(matches[3])
	if err != nil {
		return Version{}, fmt.Errorf("invalid patch version: %v", err)
	}

	return Version{
		Major:    major,
		Minor:    minor,
		Patch:    patch,
		Original: versionStr,
	}, nil
}

// CompareVersions compares two versions and returns:
//   - 1 if current > other
//   - 0 if current == other
//   - -1 if current < other
func CompareVersions(current, other Version) int {
	if current.Major != other.Major {
		if current.Major > other.Major {
			return 1
		}
		return -1
	}

	if current.Minor != other.Minor {
		if current.Minor > other.Minor {
			return 1
		}
		return -1
	}

	if current.Patch != other.Patch {
		if current.Patch > other.Patch {
			return 1
		}
		return -1
	}

	return 0
}

// IsVersionSupported checks if the current version meets the minimum version requirement.
// Returns true if current >= minimum, false otherwise.
// If either version string is empty or invalid, returns true for graceful degradation.
func IsVersionSupported(currentVersion, minVersion string) bool {
	// Graceful degradation: if we don't know the version, assume it's supported
	if currentVersion == "" || minVersion == "" {
		return true
	}

	current, err := ParseVersion(currentVersion)
	if err != nil {
		// Graceful degradation: if we can't parse, assume it's supported
		return true
	}

	minimum, err := ParseVersion(minVersion)
	if err != nil {
		// Graceful degradation: if we can't parse minimum, assume it's supported
		return true
	}

	return CompareVersions(current, minimum) >= 0
}

// CheckVersionRequirement checks if the current version meets the minimum requirement.
// Returns nil if the requirement is met, or a VersionError if not.
// If currentVersion is empty (unknown), returns nil for graceful degradation.
func CheckVersionRequirement(currentVersion, minVersion string) error {
	// Graceful degradation: if version is unknown, allow the operation
	if currentVersion == "" {
		return nil
	}

	if !IsVersionSupported(currentVersion, minVersion) {
		current, _ := ParseVersion(currentVersion)
		minimum, _ := ParseVersion(minVersion)
		return &VersionError{
			CurrentVersion:  current.String(),
			RequiredVersion: minimum.String(),
			Message:         fmt.Sprintf("This feature requires GitLab %s or later, but the current instance is running %s", minimum.String(), current.String()),
		}
	}

	return nil
}

// VersionError represents a version compatibility error.
type VersionError struct {
	// CurrentVersion is the detected GitLab version
	CurrentVersion string
	// RequiredVersion is the minimum required GitLab version
	RequiredVersion string
	// Feature is the feature or command that requires the minimum version
	Feature string
	// Host is the GitLab host
	Host string
	// Message is a descriptive error message
	Message string
}

// Error implements the error interface.
func (e *VersionError) Error() string {
	var b strings.Builder

	// Primary error message
	if e.Message != "" {
		b.WriteString(e.Message)
	} else {
		fmt.Fprintf(&b, "GitLab version %s is required", e.RequiredVersion)
	}

	// Version details
	if e.CurrentVersion != "" {
		fmt.Fprintf(&b, "\n  Current version: %s", e.CurrentVersion)
	}
	if e.RequiredVersion != "" {
		fmt.Fprintf(&b, "\n  Required version: %s or later", e.RequiredVersion)
	}
	if e.Host != "" {
		fmt.Fprintf(&b, "\n  Host: %s", e.Host)
	}
	if e.Feature != "" {
		fmt.Fprintf(&b, "\n  Feature: %s", e.Feature)
	}

	// Actionable suggestion
	suggestion := "Upgrade your GitLab instance to use this feature."
	if e.Host != "" && e.Host != "gitlab.com" {
		suggestion = fmt.Sprintf("Contact your GitLab administrator to upgrade %s, or use this feature on a newer GitLab instance.", e.Host)
	}
	fmt.Fprintf(&b, "\n\n→ %s", suggestion)

	return b.String()
}

// NewVersionError creates a new VersionError with the given details.
func NewVersionError(currentVersion, requiredVersion, feature, host string) *VersionError {
	message := fmt.Sprintf("This feature requires GitLab %s or later, but the current instance is running %s", requiredVersion, currentVersion)
	return &VersionError{
		CurrentVersion:  currentVersion,
		RequiredVersion: requiredVersion,
		Feature:         feature,
		Host:            host,
		Message:         message,
	}
}
