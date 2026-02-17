package git

import (
	"fmt"
	"net/url"
	"os/exec"
	"strings"
)

// Remote represents a git remote.
type Remote struct {
	Name     string
	FetchURL string
	PushURL  string
	Host     string
	Owner    string
	Repo     string
}

// Remotes returns all configured git remotes for the current repository.
func Remotes() ([]Remote, error) {
	output, err := runGit("remote", "-v")
	if err != nil {
		return nil, fmt.Errorf("listing remotes: %w", err)
	}

	seen := map[string]*Remote{}
	var remotes []Remote

	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 3 {
			continue
		}

		name := parts[0]
		rawURL := parts[1]
		urlType := strings.Trim(parts[2], "()")

		r, ok := seen[name]
		if !ok {
			r = &Remote{Name: name}
			seen[name] = r
		}

		switch urlType {
		case "fetch":
			r.FetchURL = rawURL
		case "push":
			r.PushURL = rawURL
		}

		host, owner, repo := parseRemoteURL(rawURL)
		if host != "" {
			r.Host = host
			r.Owner = owner
			r.Repo = repo
		}
	}

	for _, r := range seen {
		remotes = append(remotes, *r)
	}
	return remotes, nil
}

// CurrentBranch returns the currently checked-out branch name.
func CurrentBranch() (string, error) {
	output, err := runGit("rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", fmt.Errorf("determining current branch: %w", err)
	}
	return strings.TrimSpace(output), nil
}

// TopLevelDir returns the top-level directory of the current git repository.
func TopLevelDir() (string, error) {
	output, err := runGit("rev-parse", "--show-toplevel")
	if err != nil {
		return "", fmt.Errorf("determining repository root: %w", err)
	}
	return strings.TrimSpace(output), nil
}

// DefaultBranch returns the default branch of the repository (usually main or master).
func DefaultBranch(remote string) (string, error) {
	output, err := runGit("symbolic-ref", fmt.Sprintf("refs/remotes/%s/HEAD", remote))
	if err != nil {
		// Fallback: try common defaults
		for _, branch := range []string{"main", "master"} {
			if _, err := runGit("rev-parse", "--verify", fmt.Sprintf("refs/remotes/%s/%s", remote, branch)); err == nil {
				return branch, nil
			}
		}
		return "", fmt.Errorf("could not determine default branch for remote %s", remote)
	}

	ref := strings.TrimSpace(output)
	parts := strings.SplitN(ref, "/", 4)
	if len(parts) >= 4 {
		return parts[3], nil
	}
	return "", fmt.Errorf("unexpected ref format: %s", ref)
}

// CheckoutBranch checks out the given branch, creating it if necessary.
func CheckoutBranch(branch string) error {
	_, err := runGit("checkout", branch)
	if err != nil {
		_, err = runGit("checkout", "-b", branch)
	}
	return err
}

// parseRemoteURL extracts host, owner, and repo from a git remote URL.
func parseRemoteURL(rawURL string) (host, owner, repo string) {
	// Handle SSH URLs: git@gitlab.com:owner/repo.git
	if strings.HasPrefix(rawURL, "git@") {
		rawURL = strings.TrimPrefix(rawURL, "git@")
		parts := strings.SplitN(rawURL, ":", 2)
		if len(parts) != 2 {
			return "", "", ""
		}
		host = parts[0]
		path := strings.TrimSuffix(parts[1], ".git")
		pathParts := strings.SplitN(path, "/", 2)
		if len(pathParts) == 2 {
			return host, pathParts[0], pathParts[1]
		}
		return host, "", path
	}

	// Handle HTTPS URLs
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", "", ""
	}

	host = u.Host
	path := strings.TrimPrefix(u.Path, "/")
	path = strings.TrimSuffix(path, ".git")
	pathParts := strings.SplitN(path, "/", 2)
	if len(pathParts) == 2 {
		return host, pathParts[0], pathParts[1]
	}
	return host, "", path
}

// FindRemote finds a remote matching the given host.
func FindRemote(remoteName, host string) (*Remote, error) {
	remotes, err := Remotes()
	if err != nil {
		return nil, err
	}

	// First try to find by name
	if remoteName != "" {
		for _, r := range remotes {
			if r.Name == remoteName {
				return &r, nil
			}
		}
	}

	// Then try to find by host
	if host != "" {
		for _, r := range remotes {
			if r.Host == host {
				return &r, nil
			}
		}
	}

	// Fallback to "origin"
	for _, r := range remotes {
		if r.Name == "origin" {
			return &r, nil
		}
	}

	if len(remotes) > 0 {
		return &remotes[0], nil
	}

	return nil, fmt.Errorf("no git remotes found")
}

func runGit(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}
