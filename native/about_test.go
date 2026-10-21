package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAboutLegalDocumentsAreEmbeddedAndMatchRepository(t *testing.T) {
	app := &App{}
	for _, item := range []struct {
		kind     string
		path     string
		expected string
	}{
		{"license", filepath.Join("..", "LICENSE"), "GNU GENERAL PUBLIC LICENSE"},
		{"credits", filepath.Join("..", "docs", "legal", "THIRD_PARTY_NOTICES.md"), "bl2std"},
	} {
		got, err := app.GetAboutLegalText(item.kind)
		if err != nil {
			t.Fatalf("%s missing embedded document: %v", item.kind, err)
		}
		source, err := os.ReadFile(item.path)
		if err != nil {
			t.Fatal(err)
		}
		if got != string(source) {
			t.Fatalf("%s embedded bytes drifted from %s", item.kind, item.path)
		}
		if !strings.Contains(got, item.expected) {
			t.Fatalf("%s missing required project notice", item.kind)
		}
	}
	for _, invalid := range []string{"", "unknown", "../LICENSE", "https://evil.test"} {
		if _, err := app.GetAboutLegalText(invalid); err == nil {
			t.Fatalf("untrusted legal document selector accepted: %q", invalid)
		}
	}
}

func TestAboutLinksAllowlistedAndSourcePinnedToInstalledCommit(t *testing.T) {
	const commit = "f38712059ee944ae655da9a5f84dc0ade6464858"
	got, err := aboutPageURL("source", commit)
	if err != nil || got != aboutRepositoryURL+"/tree/"+commit {
		t.Fatalf("source URL = %q, %v", got, err)
	}
	if got, err := aboutPageURL("source", "development"); err != nil || got != aboutRepositoryURL {
		t.Fatalf("unstamped source must fall back to repo, got %q %v", got, err)
	}
	if got, err := aboutPageURL("releases", commit); err != nil || got != aboutRepositoryURL+"/releases" {
		t.Fatalf("release URL = %q, %v", got, err)
	}
	for _, invalid := range []string{"", "license", "https://bad.example", "../", "source?u=bad"} {
		if _, err := aboutPageURL(invalid, commit); err == nil {
			t.Fatalf("unsafe about URL kind accepted: %q", invalid)
		}
	}
}
