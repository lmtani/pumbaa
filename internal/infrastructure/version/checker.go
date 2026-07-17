// Package version provides functionality for checking pumbaa version updates.
package version

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/lmtani/pumbaa/internal/application/ports"
)

// VersionInfo contains version comparison results. It is an alias of the
// port type so GitHubChecker satisfies ports.UpdateChecker directly.
type VersionInfo = ports.VersionInfo

// Compile-time check: GitHubChecker implements the update-check port.
var _ ports.UpdateChecker = (*GitHubChecker)(nil)

// GitHubChecker checks for updates using GitHub releases API.
type GitHubChecker struct {
	repo       string
	releaseURL string
	timeout    time.Duration
}

// NewGitHubChecker creates a new GitHub-based version checker.
func NewGitHubChecker(repo string) *GitHubChecker {
	return &GitHubChecker{
		repo:       repo,
		releaseURL: fmt.Sprintf("https://github.com/%s/releases/tag", repo),
		timeout:    3 * time.Second,
	}
}

// Check starts an async check for the latest version.
// Returns a channel that will receive the result (or nil on error/timeout).
func (c *GitHubChecker) Check(currentVersion string) <-chan *VersionInfo {
	ch := make(chan *VersionInfo, 1)

	go func() {
		defer close(ch)

		// Skip check for dev versions
		if currentVersion == "dev" || currentVersion == "" {
			return
		}

		latest, err := c.fetchLatestVersion()
		if err != nil {
			return // Silent failure
		}

		info := &VersionInfo{
			Current:         currentVersion,
			Latest:          latest,
			UpdateAvailable: isNewerVersion(latest, currentVersion),
			ReleaseURL:      fmt.Sprintf("%s/v%s", c.releaseURL, latest),
		}
		ch <- info
	}()

	return ch
}

// githubRelease represents the GitHub API response for a release.
type githubRelease struct {
	TagName string `json:"tag_name"`
}

// fetchLatestVersion fetches the latest release tag from GitHub.
func (c *GitHubChecker) fetchLatestVersion() (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", c.repo)

	client := &http.Client{Timeout: c.timeout}
	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("github API returned status %d", resp.StatusCode)
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", err
	}

	// Strip 'v' prefix if present
	return strings.TrimPrefix(release.TagName, "v"), nil
}

// isNewerVersion returns true if latest is newer than current.
// Uses simple string comparison after normalizing versions.
func isNewerVersion(latest, current string) bool {
	// Normalize versions
	latest = strings.TrimPrefix(latest, "v")
	current = strings.TrimPrefix(current, "v")

	// Parse as semver-like (major.minor.patch)
	latestParts := parseVersion(latest)
	currentParts := parseVersion(current)

	// Compare each part
	for i := 0; i < 3; i++ {
		if latestParts[i] > currentParts[i] {
			return true
		}
		if latestParts[i] < currentParts[i] {
			return false
		}
	}
	return false
}

// parseVersion parses a version string into [major, minor, patch].
func parseVersion(v string) [3]int {
	parts := strings.Split(v, ".")
	result := [3]int{0, 0, 0}
	for i := 0; i < len(parts) && i < 3; i++ {
		var num int
		_, _ = fmt.Sscanf(parts[i], "%d", &num)
		result[i] = num
	}
	return result
}
