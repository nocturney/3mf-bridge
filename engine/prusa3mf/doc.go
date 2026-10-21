// Package prusa3mf converts Bambu/Orca 3MF project files into PrusaSlicer 3MF
// packages using only the Go standard library.
//
// Supported: single-mesh objects, build-item transforms, per-triangle paint
// (TriangleSelector trees using channels 1-18, copied unchanged), object and volume extruder
// assignments, and the complete numbered filament table, including duplicate
// and sparse channels, which are preserved rather than compacted. Anything
// outside that scope fails with a stable error code (see ErrorCode).
//
// Reference: Panda2Prusa (MIT License, https://github.com/CupsOhJoe/Panda2Prusa)
// is the reference project for Bambu-to-Prusa conversion conventions. No code
// is copied from it or from PrusaSlicer (AGPL-3.0). See README.md for the
// verification status of each convention.
package prusa3mf
