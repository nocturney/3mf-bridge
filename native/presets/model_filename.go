package presets

import (
	"crypto/sha256"
	"fmt"
	"strings"
)

// ModelFilenamePrefix creates a stable filesystem-safe prefix from the actual
// installed target printer_model, never the slicer's brand or profile ID.
// It intentionally drops only the model's leading manufacturer name; model
// modifiers such as "Pro", "Max", nozzle-independent variants remain intact.
// Unknown/non-ASCII model labels receive a stable Printer<hash> prefix rather
// than masquerading as a different manufacturer.
func ModelFilenamePrefix(model, vendor string) string {
	original := strings.TrimSpace(model)
	label := original
	manufacturers := []string{strings.TrimSpace(vendor)}
	switch strings.ToLower(strings.TrimSpace(vendor)) {
	case "bbl":
		manufacturers = append(manufacturers, "Bambu Lab")
	case "x 5 series", "qiditechnology":
		manufacturers = append(manufacturers, "QIDI", "QIDI Tech")
	}
	for _, maker := range manufacturers {
		if maker == "" || len(label) <= len(maker) || !strings.EqualFold(label[:len(maker)], maker) {
			continue
		}
		separator := label[len(maker)]
		if separator != ' ' && separator != '-' && separator != '_' {
			continue
		}
		rest := strings.TrimLeft(label[len(maker):], " -_")
		if rest != "" {
			label = rest
			break
		}
	}
	var out strings.Builder
	for _, char := range label {
		if ('A' <= char && char <= 'Z') || ('a' <= char && char <= 'z') ||
			('0' <= char && char <= '9') {
			if out.Len() >= 40 {
				break
			}
			out.WriteRune(char)
		}
	}
	if out.Len() > 0 {
		return out.String()
	}
	hash := sha256.Sum256([]byte(original + "\x00" + vendor))
	return fmt.Sprintf("Printer%x", hash[:5])
}

// InstalledOrcaOutputPrefix starts with the real target printer model, then
// encodes the physical nozzle and the stable opaque installed-target identity.
// Distinct nozzle presets or different installed slicers must NEVER race for
// identical output names and misleadingly swap the generated '-2' file.
// The "SB" marker also allows guarded folder-import filtering without guessing
// that every filename beginning with a familiar printer model is our output.
func InstalledOrcaOutputPrefix(id, model, vendor, nozzle string) string {
	prefix := ModelFilenamePrefix(model, vendor)
	if !IsOrcaTargetID(id) {
		// Fail closed for malformed identities: callers already require a
		// successfully resolved target, so this branch is defensive only.
		sum := sha256.Sum256([]byte(id + "\x00" + model + "\x00" + vendor))
		return fmt.Sprintf("%s-Unknown-SB%x", prefix, sum[:5])
	}
	switch nozzle {
	case "0.2", "0.25", "0.4", "0.6", "0.8":
		return prefix + "-" + nozzle + "-SB" + id[len(id)-10:]
	default:
		// An unsupported physical nozzle is never silently re-labelled.
		return prefix + "-Unknown-SB" + id[len(id)-10:]
	}
}
