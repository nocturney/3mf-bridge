#!/usr/bin/env python3
"""Cross-platform smoke for MakerWorld/Bambu source 3MF -> Prusa (and Orca, where it can run).

For each tests/fixtures source it:
  1. checks the source ZIP/package shell (always), and
  2. converts to a temporary directory with the Panda2Prusa engine when one is available, then
     runs the SliceBridge comparator (tests/panda/panda_qa.py) on the source/output pair.

Every combination that could not run is reported as UNTESTED with the reason. Nothing is
recorded as PASS without a real conversion and a clean comparison. Outputs go to a temp
directory (or --keep-output), never into the repository.

  python3 tests/makerworld/smoke_conversion.py [--keep-output DIR] [--strict]

Exit: 0 no FAIL (UNTESTED allowed unless --strict); 1 at least one FAIL; 2 --strict and UNTESTED.
"""
import argparse
import importlib.util
import json
import shutil
import sys
import tempfile
from pathlib import Path

HERE = Path(__file__).resolve().parent
sys.path.insert(0, str(HERE))
sys.path.insert(0, str(HERE.parent / "panda"))
import matrix  # noqa: E402
import panda_qa  # noqa: E402

DISCLAIMER = (
    "Structural and engine-contract evidence only. Nothing was sliced, no printer was used, and "
    "PASS does not mean the output prints or is safe to print. Physical QA on the real target "
    "printer is still required before anyone relies on these results."
)
BL2STD_MODULE = matrix.REPO / "third_party" / "bl2std" / "go.mod"


def prusa_result(fixture, workdir):
    """Return (status, detail) for one source converted to a PrusaSlicer-style 3MF."""
    engine = panda_qa.engine_command()
    if engine is None:
        return "UNTESTED", "no Panda2Prusa engine (set SLICEBRIDGE_PRUSA_ENGINE or vendor third_party/panda2prusa)"
    argv, description = engine
    if description == "vendored runner (python)" and importlib.util.find_spec("lxml") is None:
        return "UNTESTED", "vendored runner needs lxml in %s; pip install 'lxml>=5'" % sys.executable
    output = Path(workdir) / (fixture.stem + ".prusa.3mf")
    try:
        done = panda_qa.convert(argv, fixture, output, cwd=workdir)
    except OSError as exc:
        return "UNTESTED", "engine could not start (%s): %s" % (description, exc)
    try:
        reply = json.loads(done.stdout)
    except ValueError:
        reply = {}
    if reply.get("code") == "ENGINE_UNAVAILABLE":
        return "UNTESTED", "%s: %s" % (description, reply.get("error", "unavailable"))
    if done.returncode != 0 or reply.get("ok") is not True:
        tail = (done.stderr or done.stdout or "").strip().splitlines()[-1:] or [""]
        return "FAIL", "engine exit %s via %s: %s" % (done.returncode, description, tail[0][:200])
    shell = matrix.structure_problems(output)
    if shell:
        return "FAIL", "output package shell: %s" % "; ".join(shell)
    try:
        problems = panda_qa.compare(panda_qa.Package(fixture), panda_qa.Package(output))
        problems_stale = panda_qa.stale_gcode_problems(panda_qa.Package(output))
    except Exception as exc:  # the comparator must read what the engine wrote
        return "FAIL", "comparator could not read output: %r" % (exc,)
    if problems or problems_stale:
        names = sorted(problems) + (["stale_gcode"] if problems_stale else [])
        return "FAIL", "comparator mismatch: %s" % ", ".join(names)
    return "PASS", "engine via %s; comparator clean (geometry, transforms, paint, extruders, palette, no stale G-code)" % description


def orca_result():
    """Orca needs the Go adapter, which needs the bl2std submodule and a Go toolchain."""
    if shutil.which("go") is None:
        return "UNTESTED", "no Go toolchain on PATH; the Orca adapter is Go code (engine/bridge)"
    if not BL2STD_MODULE.is_file():
        return "UNTESTED", "third_party/bl2std is empty (submodule not checked out)"
    return "UNTESTED", "no Orca runner is wired into this smoke; Orca evidence comes only from native/engine go test in CI"


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__.split("\n")[0])
    parser.add_argument("--keep-output", metavar="DIR", help="copy converted outputs here")
    parser.add_argument("--strict", action="store_true", help="exit 2 if any combination is UNTESTED")
    args = parser.parse_args(argv)

    rows, failed, untested = [], 0, 0
    with tempfile.TemporaryDirectory(prefix="slicebridge-mw-") as workdir:
        for fixture in matrix.fixture_paths():
            shell = matrix.structure_problems(fixture)
            source = ("PASS", "ZIP/package shell sound") if not shell else ("FAIL", "; ".join(shell))
            rows.append((fixture.name, "source", *source))
            prusa = prusa_result(fixture, workdir)
            rows.append((fixture.name, "prusa", *prusa))
            rows.append((fixture.name, "orca", *orca_result()))
            if args.keep_output:
                keep = Path(args.keep_output)
                keep.mkdir(parents=True, exist_ok=True)
                out = Path(workdir) / (fixture.stem + ".prusa.3mf")
                if out.is_file():
                    shutil.copyfile(str(out), str(keep / out.name))
        for _, _, status, _ in rows:
            failed += status == "FAIL"
            untested += status == "UNTESTED"

    width = max(len(r[0]) for r in rows)
    for name, target, status, detail in rows:
        print("%-*s  %-6s  %-8s  %s" % (width, name, target, status, detail))
    print()
    print("Summary: %d FAIL, %d UNTESTED, %d PASS" % (
        failed, untested, sum(1 for r in rows if r[2] == "PASS")))
    print(DISCLAIMER)
    if failed:
        return 1
    if args.strict and untested:
        return 2
    return 0


if __name__ == "__main__":
    sys.exit(main())
