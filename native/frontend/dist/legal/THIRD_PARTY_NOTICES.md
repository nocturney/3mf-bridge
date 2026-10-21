# Third-party notices

## bl2std — pinned Git submodule
Project: https://github.com/BryanSmee/bl_to_std
Commit: 4252f5f0a880463fd2e6aeed3125ec3653de60ee
License: GNU GPL version 3.
Upstream license and source: third_party/bl2std/LICENSE and third_party/bl2std/.

SliceBridge imports bl2std/pkg/converter and bl2std/pkg/printer. The U1
template and original source notices remain inside that submodule.

## Local-only vendor preset loading

For the three other targets, SliceBridge reads the user's already-installed
machine, print-process and filament JSON presets *at conversion time*:
- Flash Studio Desktop / Flashforge Creator 5
  https://github.com/FlashForge/Orca-Flashforge
- ELEGOO ElegooSlicer / Centauri Carbon 2
  https://github.com/elegooofficial/ElegooSlicer
- OrcaSlicer / QIDI X-Plus 5 "X 5 Series" profile
  https://github.com/OrcaSlicer/OrcaSlicer

Those vendor preset files (including machine scripts and profiles) are
NOT committed to this repository, baked into the Go engine, or shipped
with SliceBridge builds. The project uses only local files the
user installed through their respective slicers. The converted output
3MF contains target machine settings derived locally from those presets.

Third-party profile terms and slicer installation rights remain applicable.
No source or generated code from the PolyForm Noncommercial projects
inspected during research is incorporated.

## Wails v2 — native application framework
Project: https://github.com/wailsapp/wails
Version used: v2.17.0
License: MIT
Copyright (c) 2018-Present Lea Anthony

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
