package main

import (
    "errors"
    "testing"

    "github.com/nocturney/slicebridge/native/presets"
)

func TestSnapmakerAutoPrefersValidatedInstalledVendorProfile(t *testing.T) {
    pref := PrinterSettings{Nozzle: "auto", Slicer: "auto", Filaments: map[string]string{}}
    validated := &presets.Bundle{Slicer: "Snapmaker Orca"}
    calls := 0
    loader := func(target, nozzle, slicer string, overrides map[string]string) (*presets.Bundle, error) {
        calls++
        if target != "u1" || nozzle != "0.4" || slicer != "auto" || len(overrides) != 0 {
            t.Fatalf("unexpected manufacturer probe: target=%q nozzle=%q slicer=%q override=%+v", target, nozzle, slicer, overrides)
        }
        return validated, nil
    }
    builtIn, bundle := selectSnapmakerProfile("u1", "0.4", pref, loader)
    if builtIn || bundle != validated || calls != 1 {
        t.Fatalf("validated vendor profile should win: template=%v bundle=%+v calls=%d", builtIn, bundle, calls)
    }

    fallback := func(string, string, string, map[string]string) (*presets.Bundle, error) {
        return nil, errors.New("incomplete installed profile")
    }
    builtIn, bundle = selectSnapmakerProfile("u1", "0.4", pref, fallback)
    if !builtIn || bundle != nil {
        t.Fatalf("offline default 0.4 must retain built-in template: template=%v bundle=%+v", builtIn, bundle)
    }
}

func TestSnapmakerExplicitPreferencesAreNeverSilentlyReplaced(t *testing.T) {
    noCalls := func(string, string, string, map[string]string) (*presets.Bundle, error) {
        t.Fatal("an explicit or unrelated target must not probe a replacement")
        return nil, nil
    }
    cases := []struct {
        name string
        target, nozzle string
        preference PrinterSettings
    }{
        {"explicit slicer", "u1", "0.4", PrinterSettings{Nozzle: "auto", Slicer: "snapmaker-orca"}},
        {"explicit filament", "u1", "0.4", PrinterSettings{Nozzle: "auto", Slicer: "auto", Filaments: map[string]string{"PLA": "custom"}}},
        {"other nozzle", "u1", "0.6", PrinterSettings{Nozzle: "0.6", Slicer: "auto"}},
        {"other model", "qidi", "0.4", PrinterSettings{Nozzle: "auto", Slicer: "auto"}},
    }
    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            builtIn, bundle := selectSnapmakerProfile(tc.target, tc.nozzle, tc.preference, noCalls)
            if builtIn || bundle != nil {
                t.Fatalf("explicit preference silently redirected: builtin=%v bundle=%+v", builtIn, bundle)
            }
        })
    }
}

