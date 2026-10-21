package updates

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func fakeGitHub(t *testing.T, body string, code int, version, osName, arch string) (Result, error) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("wrong HTTP method %s", r.Method)
		}
		if r.Header.Get("Accept") != "application/vnd.github+json" {
			t.Error("missing GitHub Accept header")
		}
		w.WriteHeader(code)
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()
	return Check(context.Background(), srv.Client(), srv.URL, version, osName, arch)
}

func TestNewerPreReleaseSelectsPlatformArtifact(t *testing.T) {
	body := `[
  {"tag_name":"v0.4.0-beta.1","assets":[{"name":"3MF-Bridge-Windows-native.zip"}]},
  {"tag_name":"v0.4.0-beta.2","assets":[{"name":"3MF-Bridge-Windows-native.zip"},{"name":"3MF-Bridge-macOS-Apple-Silicon.zip"}]},
  {"tag_name":"v0.4.0-beta.10","assets":[{"name":"3MF-Bridge-Windows-native.zip"},{"name":"3MF-Bridge-macOS-Apple-Silicon.zip"}]}
 ]`
	for _, tc := range []struct{ os, arch, asset string }{
		{"windows", "amd64", "3MF-Bridge-Windows-native.zip"},
		{"darwin", "arm64", "3MF-Bridge-macOS-Apple-Silicon.zip"},
	} {
		found, err := fakeGitHub(t, body, 200, "0.4.0-beta.1", tc.os, tc.arch)
		if err != nil || found.Status != "update_available" || found.LatestVersion != "0.4.0-beta.10" || found.AssetName != tc.asset {
			t.Fatalf("%s/%s returned %+v: %v", tc.os, tc.arch, found, err)
		}
		url, err := ReleaseURL(found.ReleaseTag)
		if err != nil || url != ReleasesURL+"/tag/v0.4.0-beta.10" {
			t.Fatalf("invalid release URL: %s %v", url, err)
		}
	}
}
func TestDraftsDowngradesAndStableVersionOrdering(t *testing.T) {
	data := `[
  {"tag_name":"v0.4.0-beta.10","draft":true,"assets":[{"name":"3MF-Bridge-Windows-native.zip"}]},
  {"tag_name":"not-a-version","assets":[{"name":"3MF-Bridge-Windows-native.zip"}]},
  {"tag_name":"v0.4.0-beta.1","assets":[{"name":"3MF-Bridge-Windows-native.zip"}]}
 ]`
	found, err := fakeGitHub(t, data, 200, "0.4.0-beta.1", "windows", "amd64")
	if err != nil || found.Status != "up_to_date" {
		t.Fatalf("draft incorrectly offered: %+v %v", found, err)
	}
	stable, err := fakeGitHub(t, `[{"tag_name":"v0.4.0","assets":[{"name":"3MF-Bridge-Windows-native.zip"}]}]`, 200, "0.4.0-beta.9", "windows", "amd64")
	if err != nil || stable.Status != "update_available" {
		t.Fatalf("stable should beat prerelease: %+v %v", stable, err)
	}
}
func TestNoReleasesOrPlatformBuilds(t *testing.T) {
	empty, err := fakeGitHub(t, "[]", 200, "0.4.0-beta.1", "windows", "amd64")
	if err != nil || empty.Status != "no_releases" {
		t.Fatalf("empty response: %+v %v", empty, err)
	}
	other, err := fakeGitHub(t, `[{"tag_name":"v0.4.1","assets":[{"name":"3MF-Bridge-macOS-Apple-Silicon.zip"}]}]`, 200, "0.4.0-beta.1", "windows", "amd64")
	if err != nil || other.Status != "platform_unavailable" {
		t.Fatalf("wrong platform offered: %+v %v", other, err)
	}
	unsupported, err := fakeGitHub(t, "[]", 200, "0.4.0-beta.1", "linux", "arm64")
	if err != nil || unsupported.Status != "unsupported_platform" {
		t.Fatalf("unsupported accepted: %+v %v", unsupported, err)
	}
}

func TestAPIErrorConditionsAndBadTags(t *testing.T) {
	for _, tc := range []struct {
		status int
		body   string
	}{
		{403, "rate limited"}, {429, "slow down"}, {500, "server error"},
		{200, "invalid JSON"}, {200, "{}"},
	} {
		_, err := fakeGitHub(t, tc.body, tc.status, "0.4.0-beta.1", "windows", "amd64")
		if err == nil {
			t.Errorf("accepted invalid GitHub result with status %d", tc.status)
		}
	}
	if _, err := fakeGitHub(t, "[]", 200, "bad-local-version", "windows", "amd64"); err != ErrUnsupportedVersion {
		t.Fatalf("invalid local version accepted: %v", err)
	}
	for _, tag := range []string{"", "v1.2", "invalid-tag", "v1.2.3/extra", strings.Repeat("X", 120)} {
		if _, err := ReleaseURL(tag); err == nil {
			t.Errorf("unsafe tag accepted: %q", tag)
		}
	}
	oversize := "\"" + strings.Repeat("x", maxResponse+10) + "\""
	if _, err := fakeGitHub(t, oversize, 200, "0.4.0-beta.1", "windows", "amd64"); err == nil {
		t.Fatal("oversized HTTP body was accepted")
	}
}
func TestCompatibleOlderBetaNotHiddenByOtherOS(t *testing.T) {
	data := `[
  {"tag_name":"v0.5.0","assets":[{"name":"3MF-Bridge-macOS-Apple-Silicon.zip"}]},
  {"tag_name":"v0.4.0-beta.2","assets":[{"name":"3MF-Bridge-Windows-native.zip"}]}
 ]`
	found, err := fakeGitHub(t, data, 200, "0.4.0-beta.1", "windows", "amd64")
	if err != nil || found.Status != "update_available" || found.LatestVersion != "0.4.0-beta.2" {
		t.Fatalf("missed compatible beta: %+v %v", found, err)
	}
}

func TestAuthenticatedGitHubJSONReusesReleaseRules(t *testing.T) {
	raw := []byte(`[{"tag_name":"v0.4.0-beta.2","assets":[{"name":"3MF-Bridge-Windows-native.zip"}]}]`)
	result, err := evaluateCLIJSON(context.Background(), raw, "0.4.0-beta.1", "windows", "amd64")
	if err != nil || result.Status != "update_available" || result.ReleaseTag != "v0.4.0-beta.2" {
		t.Fatalf("authenticated result failed: %+v %v", result, err)
	}
	empty, err := evaluateCLIJSON(context.Background(), []byte("[]"), "0.4.0-beta.1", "darwin", "arm64")
	if err != nil || empty.Status != "no_releases" {
		t.Fatalf("no private releases: %+v %v", empty, err)
	}
}

func TestSliceBridgeReleaseArtifactsMatchAfterRename(t *testing.T) {
	body := `[{"tag_name":"v0.4.0-beta.4","assets":[
 {"name":"SliceBridge-Windows.zip"},
 {"name":"SliceBridge-macOS-Apple-Silicon.zip"}]}]`
	for _, tc := range []struct{ goos, arch, name string }{
		{"windows", "amd64", "SliceBridge-Windows.zip"},
		{"darwin", "arm64", "SliceBridge-macOS-Apple-Silicon.zip"},
	} {
		result, err := fakeGitHub(t, body, 200, "0.4.0-beta.3", tc.goos, tc.arch)
		if err != nil || result.Status != "update_available" || result.AssetName != tc.name {
			t.Fatalf("%s: wrong branding asset: %+v %v", tc.goos, result, err)
		}
	}
}
