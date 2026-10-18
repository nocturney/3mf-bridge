# Third-party notices

## bl2std — pinned Git submodule
Project: https://github.com/BryanSmee/bl_to_std
Commit: 4252f5f0a880463fd2e6aeed3125ec3653de60ee
License: GNU GPL version 3.
Upstream license and source: third_party/bl2std/LICENSE and third_party/bl2std/.

3MF Bridge imports bl2std/pkg/converter and bl2std/pkg/printer. The U1
template and original source notices remain inside that submodule.

## Local-only vendor preset loading

For the three other targets, 3MF Bridge reads the user's already-installed
machine, print-process and filament JSON presets *at conversion time*:
- Flash Studio Desktop / Flashforge Creator 5
  https://github.com/FlashForge/Orca-Flashforge
- ELEGOO ElegooSlicer / Centauri Carbon 2
  https://github.com/elegooofficial/ElegooSlicer
- OrcaSlicer / QIDI X-Plus 5 "X 5 Series" profile
  https://github.com/OrcaSlicer/OrcaSlicer

Those vendor preset files (including machine scripts and profiles) are
NOT committed to this repository, baked into the Go engine, or shipped
with the portable Electron app. The project uses only local files the
user installed through their respective slicers. The converted output
3MF contains target machine settings derived locally from those presets.

Third-party profile terms and slicer installation rights remain applicable.
No source or generated code from the PolyForm Noncommercial projects
inspected during research is incorporated.
