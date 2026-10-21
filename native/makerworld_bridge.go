package main

import (
	"fmt"

	"github.com/nocturney/3mf-bridge/native/makerworld"
)

// DetectMakerWorldOrigins reads the operating system's existing download
// provenance. It does not scan the Downloads folder or modify any file.
func (a *App) DetectMakerWorldOrigins(paths []string) (map[string]makerworld.Source, error) {
	if len(paths) > maxBatchFiles {
		return nil, fmt.Errorf("too many paths")
	}
	results := make(map[string]makerworld.Source, len(paths))
	for _, p := range paths {
		source := makerworld.Detect(p)
		if source.Kind != "unknown" {
			results[p] = source
		}
	}
	return results, nil
}

// PreviewMakerWorldLink is intentionally non-executing; the browser integration
// must be explicitly enabled and tested before any protocol association or
// downloaded file is accepted.
func (a *App) PreviewMakerWorldLink(raw string) (makerworld.Link, error) {
	return makerworld.ParseLink(raw)
}
