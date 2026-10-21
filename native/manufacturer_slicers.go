package main

import "github.com/nocturney/slicebridge/native/presets"

// ManufacturerSlicers reports popular slicers installed on this computer. It
// is detection only: it converts nothing, claims no printer-model compatibility,
// and takes no path from the frontend.
func (a *App) ManufacturerSlicers() []presets.ManufacturerSlicer {
	return presets.DetectManufacturerSlicers()
}
