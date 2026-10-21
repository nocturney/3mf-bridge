package main

import (
	"fmt"
	"regexp"
	"strings"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

const aboutRepositoryURL = "https://github.com/nocturney/slicebridge"

var stampedCommit = regexp.MustCompile(`^[a-fA-F0-9]{40}$`)

// GetAboutLegalText returns the exact license and credit files embedded in the
// installed app. It never fetches GitHub, so it also works fully offline.
func (a *App) GetAboutLegalText(kind string) (string, error) {
	var filename string
	switch kind {
	case "license":
		filename = "frontend/dist/legal/LICENSE.txt"
	case "credits":
		filename = "frontend/dist/legal/THIRD_PARTY_NOTICES.md"
	default:
		return "", fmt.Errorf("unknown legal document")
	}
	content, err := assets.ReadFile(filename)
	if err != nil {
		return "", fmt.Errorf("could not read bundled %s: %w", kind, err)
	}
	return string(content), nil
}

// aboutPageURL is deliberately allowlisted: the webview cannot supply arbitrary URLs.
func aboutPageURL(kind, commit string) (string, error) {
	switch kind {
	case "source":
		if stampedCommit.MatchString(commit) {
			return aboutRepositoryURL + "/tree/" + strings.ToLower(commit), nil
		}
		return aboutRepositoryURL, nil
	case "releases":
		return aboutRepositoryURL + "/releases", nil
	default:
		return "", fmt.Errorf("unknown About link")
	}
}

func (a *App) OpenAboutPage(kind string) error {
	url, err := aboutPageURL(kind, buildCommit)
	if err != nil {
		return err
	}
	if a.ctx == nil {
		return fmt.Errorf("native desktop window is not ready")
	}
	wailsruntime.BrowserOpenURL(a.ctx, url)
	return nil
}
