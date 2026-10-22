# Manufacturer profile priority

SliceBridge converts MakerWorld/Bambu-origin 3MF to **other printers**. Manufacturer slicers
with verified local machine + process + PLA profile chains are the first choice.
OrcaSlicer and PrusaSlicer are broad fallback libraries, not equally preferred defaults.

## Deterministic policy

1. Exact manufacturer-owned application with a **fully resolved** target (for example:
   Snapmaker Orca, Flash Studio, ELEGOO Slicer or QIDIStudio). A ready built-in
   Flashforge/ELEGOO/QIDI adapter uses its validated manufacturer profile.
2. Snapmaker U1's built-in 0.4 mm template is a fallback **only when an installed
   Snapmaker Orca profile cannot be validated**, never ahead of an exact installed
   Snapmaker profile. The U1 template does not claim to originate from an installation.
3. Generic OrcaSlicer with an installed, validated machine/process/PLA chain.
4. Other fully resolved Orca-family forks without an exact reviewed manufacturer match.
5. Generic PrusaSlicer, if a matching ready target actually exists. The separate
   prusa-project export is **not** an automatically resolved printer profile.
6. Catalogue-only identifiers: selectable inventory, not ready for conversion.

The default within the best available source tier is 0.4 mm if present, otherwise
a consistent nozzle order. This default affects only newly checked cards; **existing
saved IDs, nozzle and slicer overrides are never rewritten or silently migrated**.
PrusaSlicer is itself the manufacturer app for Prusa Research targets when an exact
validated installed target is available. Manufacturer/slicer pairs are allowlisted,
never assumed from an arbitrary app name or just from a vendor folder.

## Native selection and fallback

Windows/macOS native source discovery looks for the manufacturer's slicer across
both system and per-user install bases before trying universal OrcaSlicer, even if
the universal app is installed in a normally higher-priority filesystem base.
Windows QIDI auto now tries QIDIStudio before OrcaSlicer.

An auto selection must load a **complete** machine, process and PLA profile chain.
If a manufacturer directory exists but its presets are incomplete or corrupt,
the next trusted source is tried. Explicit saved slicer and filament selections
fail closed rather than switching to another app. The U1 historical 0.4 mm
template is retained as a final offline fallback when explicitly using the
old auto route and no validated Snapmaker Orca profile is available.

Bambu is a source format, never a conversion target. No Cura, ideaMaker or
manufacturer-specific project-format engine is added.

## Verification requirements

- Node UI tests: Flash Studio, ELEGOO, QIDIStudio, Snapmaker Orca and Prusa
  precedence; an unrelated slicer cannot impersonate a manufacturer profile;
  saved IDs unchanged; U1 template fallback only.
- Go: trusted per-platform source order, existing overrides, malformed target
  rejection, complete-profile fallback when vendor directory is broken, and
  historical U1 default behaviour.
- Real rendered RTL/LTR picker geometry in Windows and macOS.
- GitHub native CI Windows WebView2 and macOS WebKit must be green before merge
  or any release. Physical prints, automatic Prusa per-printer profile mapping
  and slicer GUI round-trip are **not** claimed.
