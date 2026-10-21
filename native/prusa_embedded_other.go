//go:build !windows

package main

// On macOS the engine already lives inside SliceBridge.app/Contents/Resources.
// The Windows single-file overlay is intentionally not used on other systems.
func locateEmbeddedPrusaEngine(_ string) (string, bool, error) {
	return "", false, nil
}
