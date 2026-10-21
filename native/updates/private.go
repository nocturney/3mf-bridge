package updates

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"
)

// cachedReleaseReply allows the exact same validated release-selection
// routine to consume an authenticated GitHub CLI JSON response. Credentials
// are managed exclusively by gh, not by the application or its renderer.
type cachedReleaseReply struct{ data []byte }

func (r cachedReleaseReply) RoundTrip(req *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body:       io.NopCloser(bytes.NewReader(r.data)),
		Request:    req,
	}, nil
}
func evaluateCLIJSON(ctx context.Context, raw []byte, installed, goos, arch string) (Result, error) {
	client := &http.Client{Transport: cachedReleaseReply{data: raw}, Timeout: 3 * time.Second}
	return Check(ctx, client, APIURL, installed, goos, arch)
}

func locateGitHubCLI() string {
	// Executable names come from known OS install locations or the user's
	// configured PATH; never from untrusted Github API data.
	names := []string{}
	if runtime.GOOS == "windows" {
		if pf := os.Getenv("ProgramFiles"); pf != "" {
			names = append(names, filepath.Join(pf, "GitHub CLI", "gh.exe"))
		}
		if local := os.Getenv("LOCALAPPDATA"); local != "" {
			names = append(names, filepath.Join(local, "Programs", "GitHub CLI", "gh.exe"))
		}
	} else if runtime.GOOS == "darwin" {
		if home, err := os.UserHomeDir(); err == nil {
			names = append(names, filepath.Join(home, ".local", "bin", "gh"))
		}
		names = append(names, "/opt/homebrew/bin/gh", "/usr/local/bin/gh")
	}
	for _, name := range names {
		if info, err := os.Stat(name); err == nil && !info.IsDir() {
			return name
		}
	}
	executable, err := exec.LookPath("gh")
	if err == nil {
		return executable
	}
	return ""
}

// authenticatedReleases queries the same hardcoded GitHub repository through
// the OS user's existing GitHub CLI session (system credential store).
// It never asks for, reads, prints, embeds, or persists a GitHub token.
func authenticatedReleases(ctx context.Context) ([]byte, error) {
	binary := locateGitHubCLI()
	if binary == "" {
		return nil, errors.New("GitHub CLI is not installed")
	}
	deadline, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()
	command := exec.CommandContext(deadline, binary,
		"api", "repos/nocturney/3mf-bridge/releases?per_page=50",
		"--method", "GET")
	command.Stdin = nil
	data, err := command.Output()
	if err != nil {
		return nil, errors.New("GitHub CLI cannot access the private repository")
	}
	if len(data) > maxResponse {
		return nil, fmt.Errorf("GitHub CLI response too large")
	}
	return data, nil
}
func checkInstalledPrivate(ctx context.Context, installed, goos, arch string) (Result, error) {
	if normalizedVersion(installed) == "" {
		return Result{InstalledVersion: installed, Platform: goos + "-" + arch}, ErrUnsupportedVersion
	}
	if len(expectedAssets(goos, arch)) == 0 {
		return Result{InstalledVersion: installed, Platform: goos + "-" + arch, Status: "unsupported_platform"}, nil
	}
	// This repo is currently private. Prefer the user's authenticated gh
	// session to avoid misleading unauthenticated 404 / no-update results.
	raw, cliErr := authenticatedReleases(ctx)
	if cliErr == nil {
		return evaluateCLIJSON(ctx, raw, installed, goos, arch)
	}
	// If releases are later made public, clients without gh still work.
	result, err := Check(ctx, NewClient(), APIURL, installed, goos, arch)
	if errors.Is(err, ErrRepositoryPrivate) {
		return Result{
			InstalledVersion: installed,
			Platform:         goos + "-" + arch,
			Status:           "authentication_required",
		}, nil
	}
	return result, err
}
