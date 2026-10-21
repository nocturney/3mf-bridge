// Package updates implements an explicit, read-only GitHub Releases check.
// Never download or execute untrusted binaries from the desktop UI.
package updates

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"strings"
	"time"

	"golang.org/x/mod/semver"
)

const (
	Repository  = "nocturney/slicebridge"
	APIURL      = "https://api.github.com/repos/nocturney/slicebridge/releases?per_page=50"
	ReleasesURL = "https://github.com/nocturney/slicebridge/releases"
	BuildsURL   = "https://github.com/nocturney/slicebridge/actions/workflows/native.yml"
	maxResponse = 2 << 20
)

var ErrUnsupportedVersion = errors.New("invalid installed application version")
var ErrRepositoryPrivate = errors.New("GitHub repository is not publicly accessible")

// Result contains only publisher release metadata, never an executable URL.
// A human must follow the release link and install any update themselves.
type Result struct {
	Status           string `json:"status"`
	InstalledVersion string `json:"installedVersion"`
	LatestVersion    string `json:"latestVersion,omitempty"`
	ReleaseTag       string `json:"releaseTag,omitempty"`
	AssetName        string `json:"assetName,omitempty"`
	Platform         string `json:"platform"`
}
type release struct {
	TagName string `json:"tag_name"`
	Draft   bool   `json:"draft"`
	Assets  []struct {
		Name string `json:"name"`
	} `json:"assets"`
}

func normalizedVersion(s string) string {
	s = strings.TrimSpace(s)
	if s != "" && !strings.HasPrefix(s, "v") {
		s = "v" + s
	}
	core := strings.TrimPrefix(s, "v")
	core = strings.SplitN(core, "-", 2)[0]
	core = strings.SplitN(core, "+", 2)[0]
	if strings.Count(core, ".") != 2 || !semver.IsValid(s) {
		return ""
	}
	return semver.Canonical(s)
}
func safeReleaseTag(s string) bool {
	if len(s) == 0 || len(s) > 90 {
		return false
	}
	return normalizedVersion(s) != ""
}

// ReleaseURL constructs a link ONLY to our own GitHub Releases tag.
// No URL or hostname read from the GitHub response can be opened here.
func ReleaseURL(tag string) (string, error) {
	if !safeReleaseTag(tag) {
		return "", fmt.Errorf("invalid release tag")
	}
	return ReleasesURL + "/tag/" + tag, nil
}
func expectedAssets(goos, arch string) []string {
	switch goos + "/" + arch {
	case "windows/amd64":
		return []string{"slicebridge.exe", "slicebridge-windows.zip", "slicebridge-windows-native.zip", "3mf-bridge-windows-native.zip", "3mf-bridge-windows-x64.zip"}
	case "darwin/arm64":
		return []string{"slicebridge-macos-apple-silicon.zip", "slicebridge-macos-arm64.zip", "3mf-bridge-macos-apple-silicon.zip", "3mf-bridge-macos-arm64.zip"}
	case "darwin/amd64":
		return []string{"slicebridge-macos-intel.zip", "3mf-bridge-macos-intel.zip", "3mf-bridge-macos-amd64.zip"}
	}
	return nil
}
func matchingAsset(r release, names []string) string {
	for _, asset := range r.Assets {
		for _, want := range names {
			if strings.EqualFold(asset.Name, want) {
				return asset.Name
			}
		}
	}
	return ""
}
func NewClient() *http.Client {
	return &http.Client{Timeout: 12 * time.Second}
}

// Check queries the official public GitHub Releases API on user request.
// Includes published prereleases; ignores drafts and non-semver tags.
// If there are no releases, returns an honest no_releases status (not a
// misleading "up to date" claim). Network errors propagate to the caller.
func Check(ctx context.Context, client *http.Client, endpoint, installed, goos, arch string) (Result, error) {
	out := Result{InstalledVersion: installed, Platform: goos + "-" + arch}
	current := normalizedVersion(installed)
	if current == "" {
		return out, ErrUnsupportedVersion
	}
	if client == nil {
		client = NewClient()
	}
	if endpoint == "" {
		endpoint = APIURL
	}
	names := expectedAssets(goos, arch)
	if len(names) == 0 {
		out.Status = "unsupported_platform"
		return out, nil
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return out, err
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	request.Header.Set("User-Agent", "SliceBridge/"+strings.TrimPrefix(current, "v"))
	response, err := client.Do(request)
	if err != nil {
		return out, fmt.Errorf("GitHub Releases check: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusForbidden || response.StatusCode == http.StatusTooManyRequests {
		return out, fmt.Errorf("GitHub API access limited (%d)", response.StatusCode)
	}
	if response.StatusCode == http.StatusNotFound {
		return out, ErrRepositoryPrivate
	}
	if response.StatusCode != http.StatusOK {
		return out, fmt.Errorf("GitHub Releases returned HTTP %d", response.StatusCode)
	}
	raw, err := io.ReadAll(io.LimitReader(response.Body, maxResponse+1))
	if err != nil {
		return out, fmt.Errorf("reading GitHub Releases: %w", err)
	}
	if len(raw) > maxResponse {
		return out, fmt.Errorf("GitHub response is unexpectedly large")
	}
	var releases []release
	if err := json.Unmarshal(raw, &releases); err != nil {
		return out, fmt.Errorf("invalid GitHub Releases response: %w", err)
	}
	var latestAny, latestApplicable string
	var best release
	var name string
	for _, candidate := range releases {
		if candidate.Draft || !safeReleaseTag(candidate.TagName) {
			continue
		}
		version := normalizedVersion(candidate.TagName)
		if latestAny == "" || semver.Compare(version, latestAny) > 0 {
			latestAny = version
		}
		asset := matchingAsset(candidate, names)
		if asset == "" {
			continue
		}
		if latestApplicable == "" || semver.Compare(version, latestApplicable) > 0 {
			latestApplicable = version
			best = candidate
			name = asset
		}
	}
	if latestAny == "" {
		out.Status = "no_releases"
		return out, nil
	}
	if latestApplicable != "" && semver.Compare(latestApplicable, current) > 0 {
		out.Status = "update_available"
		out.LatestVersion = strings.TrimPrefix(latestApplicable, "v")
		out.ReleaseTag = best.TagName
		out.AssetName = name
		return out, nil
	}
	if semver.Compare(latestAny, current) > 0 {
		out.Status = "platform_unavailable"
		out.LatestVersion = strings.TrimPrefix(latestAny, "v")
		return out, nil
	}
	out.Status = "up_to_date"
	out.LatestVersion = strings.TrimPrefix(latestAny, "v")
	return out, nil
}

// CheckInstalled is the production entrypoint. Call only in response to a
// deliberate "Check for updates" action, never at startup or periodically.
func CheckInstalled(ctx context.Context, installed string) (Result, error) {
	return checkInstalledPrivate(ctx, installed, runtime.GOOS, runtime.GOARCH)
}
