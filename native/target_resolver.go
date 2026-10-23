package main

import (
	"fmt"

	"github.com/nocturney/slicebridge/engine/bridge"
	"github.com/nocturney/slicebridge/native/presets"
)

// conversionTarget is the native half of the unified target contract. One
// physical printer (a stable built-in alias or an installed Orca target ID)
// resolves to the bridge request, the slicer that owns its profiles and the
// output prefix. Nozzle, engine and profile are never UI inputs: the nozzle is
// either fixed by the installed machine or is the user's saved physical nozzle.
type conversionTarget struct {
	ID        string
	Prefix    string
	Slicer    string
	Request   bridge.TargetRequest
	Overrides map[string]string
	cleanup   func()
}

func (t *conversionTarget) release() {
	if t != nil && t.cleanup != nil {
		t.cleanup()
	}
}

// targetResolveError carries the stable conversion code for a refusal.
type targetResolveError struct {
	Code string
	Err  error
}

func (e *targetResolveError) Error() string { return e.Err.Error() }

func resolveFail(code string, err error) *targetResolveError {
	return &targetResolveError{Code: code, Err: err}
}

// targetLoaders are the installed-profile sources. Production uses the real
// installed-slicer readers; tests inject synthetic ones.
type targetLoaders struct {
	vendor func(target, nozzle, slicer string, overrides map[string]string) (*presets.Bundle, error)
	orca   func(id string) (*presets.OrcaBundle, error)
}

var installedLoaders = targetLoaders{vendor: presets.LoadConfigured, orca: presets.ResolveOrcaBundle}

func explicitNozzle(preference PrinterSettings) string {
	if preference.Nozzle == "auto" {
		return ""
	}
	return preference.Nozzle
}

// resolveConversionTarget is the single native resolver for every Bambu-based
// destination. It returns either a fully prepared target or a fail-closed
// refusal; it never falls back to a different nozzle, slicer or profile than
// the one selected. The caller owns release().
//
// The source project's nozzle is consulted ONLY for a built-in alias whose saved
// nozzle is "auto" (the historical, field-validated behavior), and the bridge
// reports that case as target_nozzle_origin=source_matched. A saved physical
// nozzle (for example Flashforge Creator 5 at 0.25 mm) always wins over the
// source, and installed Orca profiles ignore the source nozzle entirely.
//
// TODO(P0-B): require a saved physical nozzle for built-in aliases instead of
// source matching, once the printer picker stores one for every selection.
func resolveConversionTarget(target, sourceNozzle string, preference PrinterSettings, loaders targetLoaders) (*conversionTarget, *targetResolveError) {
	if presets.IsOrcaTargetID(target) {
		return resolveInstalledOrcaTarget(target, preference, loaders)
	}
	return resolveBuiltinAlias(target, sourceNozzle, preference, loaders)
}

func resolveInstalledOrcaTarget(target string, preference PrinterSettings, loaders targetLoaders) (*conversionTarget, *targetResolveError) {
	bundle, err := loaders.orca(target)
	if err != nil {
		return nil, resolveFail("TARGET_PROFILE_INVALID", err)
	}
	if bundle.Target != target {
		return nil, resolveFail("TARGET_PROFILE_INVALID",
			fmt.Errorf("installed profile %s does not belong to requested target %s", bundle.Target, target))
	}
	file, cleanup, err := tempOrcaBundle(bundle)
	if err != nil {
		return nil, resolveFail("TARGET_PROFILE_INVALID", err)
	}
	return &conversionTarget{
		ID: target, Prefix: presets.OrcaPrefix(target), Slicer: bundle.Slicer,
		// A stale saved nozzle that disagrees with the installed machine is
		// refused by the bridge (TARGET_NOZZLE_MISMATCH), never reconciled.
		Request:   bridge.TargetRequest{Target: target, ProfileFile: file, Nozzle: explicitNozzle(preference)},
		Overrides: map[string]string{}, cleanup: cleanup,
	}, nil
}

func resolveBuiltinAlias(target, sourceNozzle string, preference PrinterSettings, loaders targetLoaders) (*conversionTarget, *targetResolveError) {
	preferred := explicitNozzle(preference)
	nativeNozzle := sourceNozzle
	if preferred != "" {
		nativeNozzle = preferred
		// Variant() carries the owner-approved 0.2 -> 0.25 source remap for the
		// Creator 5. That remap must never apply to an explicit selection.
		if spec, known := presets.Specs[target]; known && !containsString(spec.Nozzles, preferred) {
			return nil, resolveFail("TARGET_NOZZLE_UNAVAILABLE",
				fmt.Errorf("%s has no %s mm nozzle profile", spec.Name, preferred))
		}
	}
	nozzle, _, _, _, err := presets.Variant(target, nativeNozzle)
	if err != nil {
		return nil, resolveFail("TARGET_NOZZLE_UNAVAILABLE", err)
	}
	useBuiltin, bundle := selectSnapmakerProfile(target, nozzle, preference, loaders.vendor)
	resolved := &conversionTarget{
		ID: target, Prefix: outputs[target], Slicer: "Built-in Snapmaker U1 template",
		Request:   bridge.TargetRequest{Target: target, Nozzle: preferred},
		Overrides: preference.Filaments,
	}
	if useBuiltin {
		return resolved, nil
	}
	if bundle == nil {
		var loadErr error
		if bundle, loadErr = loaders.vendor(target, nozzle, preference.Slicer, preference.Filaments); loadErr != nil {
			return nil, resolveFail("TARGET_PROFILE_INVALID", loadErr)
		}
	}
	file, cleanup, err := tempVendorBundle(bundle)
	if err != nil {
		return nil, resolveFail("TARGET_PROFILE_INVALID", err)
	}
	resolved.Slicer = bundle.Slicer
	resolved.Request.ProfileFile = file
	resolved.cleanup = cleanup
	return resolved, nil
}

func containsString(list []string, value string) bool {
	for _, item := range list {
		if item == value {
			return true
		}
	}
	return false
}
