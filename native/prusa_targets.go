package main

import (
	"fmt"

	"github.com/nocturney/slicebridge/native/presets"
)

// Installed PrusaSlicer printers. A user printer is listed when its preset
// resolves completely (see presets/prusa.go). Listing is not readiness.
// SliceBridge does not yet attach a validated native printer, print and filament
// to the 3MF, so every Prusa target is reported as not ready. Conversion of a
// Prusa ID fails with an explicit code instead of writing an unverified project.
// The generic prusa-project export is unaffected.

// prusaExportVerified flips to true only after a PrusaSlicer-recognized
// per-printer export has been verified end to end. Tests flip it to exercise
// the gate.
var prusaExportVerified = false

const prusaExportNotVerified = "PrusaSlicer per-printer export is not verified yet. Use the generic PrusaSlicer project."

// Injectable for tests. Production reads only locally installed PrusaSlicer
// presets, through the same resolver conversion uses.
var discoverInstalledPrusa = presets.DiscoverPrusa
var resolveInstalledPrusa = presets.ResolvePrusaTarget

// PrusaTargetView is one installed Prusa target as the frontend sees it.
type PrusaTargetView struct {
	presets.PrusaTarget
	Ready  bool   `json:"ready"`
	Reason string `json:"reason,omitempty"`
}

// InstalledPrusaReply mirrors the Orca discovery shape: targets and rejected.
type InstalledPrusaReply struct {
	Targets  []PrusaTargetView        `json:"targets"`
	Rejected []presets.PrusaRejection `json:"rejected"`
}

// prusaReadiness reports whether a Prusa target can be converted, and why not.
func prusaReadiness() (bool, string) {
	if !prusaExportVerified {
		return false, prusaExportNotVerified
	}
	return true, ""
}

func (a *App) InstalledPrusaTargets() InstalledPrusaReply {
	return installedPrusaReply(discoverInstalledPrusa())
}

func installedPrusaReply(discovery presets.PrusaDiscovery) InstalledPrusaReply {
	reply := InstalledPrusaReply{Targets: []PrusaTargetView{}, Rejected: discovery.Rejected}
	ready, reason := prusaReadiness()
	for _, target := range discovery.Targets {
		reply.Targets = append(reply.Targets, PrusaTargetView{PrusaTarget: target, Ready: ready, Reason: reason})
	}
	return reply
}

// validatePrusaSelection accepts a saved Prusa printer only when it re-resolves
// completely and is ready. The same gate applies to conversion.
func validatePrusaSelection(id string) error {
	if _, err := resolveInstalledPrusa(id); err != nil {
		return fmt.Errorf("installed printer profile is not ready")
	}
	if ready, reason := prusaReadiness(); !ready {
		return fmt.Errorf("%s", reason)
	}
	return nil
}

// prusaInstalledConvert re-resolves the ID from disk before anything else. It
// never writes output while the per-printer export is unverified.
func prusaInstalledConvert(target string) ConvertResult {
	result := ConvertResult{Target: target}
	if _, err := resolveInstalledPrusa(target); err != nil {
		result.Code = "TARGET_PROFILE_INVALID"
		result.Error = err.Error()
		return result
	}
	if ready, reason := prusaReadiness(); !ready {
		result.Code = "PRUSA_TARGET_NOT_READY"
		result.Error = reason
		return result
	}
	// Still refuses explicitly: the verified export path is not implemented yet.
	result.Code = "PRUSA_EXPORT_NOT_IMPLEMENTED"
	result.Error = "per-printer PrusaSlicer export is not implemented"
	return result
}
