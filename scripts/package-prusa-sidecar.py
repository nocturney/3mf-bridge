#!/usr/bin/env python3
"""Build, stage and verify the bundled Panda2Prusa sidecar (slicebridge-prusa-engine).

Developer and CI entry point. Stdlib plus PyInstaller (pinned in
scripts/prusa-sidecar-requirements.txt). No conversion service, no API, no network calls
beyond the pip install you run yourself.

  python3 scripts/package-prusa-sidecar.py build
  python3 scripts/package-prusa-sidecar.py pack-windows --bin-dir native/build/bin
  python3 scripts/package-prusa-sidecar.py verify --platform macos --bin-dir native/build/bin
  python3 scripts/package-prusa-sidecar.py print-ci-steps

Subcommands:
  build     needs Python 3.11+, lxml>=5, the pinned PyInstaller and a vendored
            third_party/panda2prusa/{panda2prusa/, LICENSE, slicebridge_runner.py}.
            Writes dist/prusa-sidecar/{slicebridge-prusa-engine[.exe], panda2prusa-LICENSE.txt,
            manifest.json} after a smoke conversion of tests/fixtures/bambu-five-colors.3mf.
  pack-windows  embeds the sidecar and MIT license inside one Windows SliceBridge.exe.
  stage     places the sidecar in SliceBridge.app/Contents/Resources on macOS
            and re-seals the app (Windows must use pack-windows).
  verify    checks Windows embedded payload or macOS staged files, upstream
            license, and sidecar hash against the build manifest.
  print-ci-steps  prints the workflow steps the CI jobs need.

Relative paths resolve against the repository root, so commands work from any directory.
Exit codes: 0 success, 1 failed check, 2 missing prerequisite or usage error.
"""
from __future__ import annotations

import argparse
import hashlib
import io
import json
import os
import re
import shutil
import subprocess
import struct
import sys
import tempfile
import zipfile
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
SIDECAR = "slicebridge-prusa-engine"
VENDOR_DIR = ROOT / "third_party" / "panda2prusa"
VENDOR_RUNNER = "slicebridge_runner.py"
VENDOR_ITEMS = (VENDOR_RUNNER, "panda2prusa", "LICENSE")
REQUIREMENTS = ROOT / "scripts" / "prusa-sidecar-requirements.txt"
DIST_DIR = ROOT / "dist" / "prusa-sidecar"
LICENSE_NAME = "panda2prusa-LICENSE.txt"
MANIFEST_NAME = "manifest.json"
MIT_MARKERS = ("MIT License", "Permission is hereby granted, free of charge")
SMOKE_FIXTURE = ROOT / "tests" / "fixtures" / "bambu-five-colors.3mf"
SMOKE_TIMEOUT = 300
MIN_PYTHON = (3, 11)
WIN_APP_EXE = "SliceBridge.exe"
MAC_APP = "SliceBridge.app"
PLATFORMS = ("windows", "macos")
# Appended to the Windows PE after Wails builds it. The PE loader ignores the
# overlay; SliceBridge validates and extracts this exact payload when needed.
PACK_MAGIC = b"SLICEBRIDGE_PRUSA_V1"
PACK_TRAILER_SIZE = len(PACK_MAGIC) + 8 + 32
PACK_MAX_BYTES = 128 << 20
PACK_NAMES = ("slicebridge-prusa-engine.exe", LICENSE_NAME)


class Failure(Exception):
    """A prerequisite is missing or a check failed. The message is printed for the user."""

    exit_code = 2


class CheckFailed(Failure):
    exit_code = 1


def host_platform():
    if sys.platform == "win32":
        return "windows"
    if sys.platform == "darwin":
        return "macos"
    raise Failure("Panda2Prusa sidecar packaging runs on Windows or macOS hosts only")


def exe_name(platform):
    return SIDECAR + (".exe" if platform == "windows" else "")


def resolve(path):
    """Relative paths are taken from the repository root, not the caller's directory."""
    candidate = Path(path)
    return candidate if candidate.is_absolute() else ROOT / candidate


def sha256(path):
    digest = hashlib.sha256()
    with open(path, "rb") as handle:
        for chunk in iter(lambda: handle.read(1 << 20), b""):
            digest.update(chunk)
    return digest.hexdigest()


def check_mit(license_text, origin):
    if not all(marker in license_text for marker in MIT_MARKERS):
        raise CheckFailed(f"{origin} is not the MIT license text the attribution policy requires")


def pinned_pyinstaller():
    text = REQUIREMENTS.read_text(encoding="utf-8")
    match = re.search(r"^pyinstaller==([0-9][0-9A-Za-z.+-]*)\s*$", text, re.M)
    if not match:
        raise Failure(f"{REQUIREMENTS} must pin pyinstaller with '=='")
    return match.group(1)


def major(version):
    match = re.match(r"\d+", version)
    return int(match.group()) if match else -1


def check_build_prerequisites(pin):
    if sys.version_info < MIN_PYTHON:
        raise Failure(
            f"Python {MIN_PYTHON[0]}.{MIN_PYTHON[1]}+ is required to build the sidecar; "
            f"this interpreter is {sys.version.split()[0]} ({sys.executable})")
    from importlib import metadata
    try:
        lxml_version = metadata.version("lxml")
        pyinstaller_version = metadata.version("pyinstaller")
    except metadata.PackageNotFoundError as error:
        raise Failure(f"{error.args[0]} is not installed. Run: "
                      f"{sys.executable} -m pip install -r {REQUIREMENTS.relative_to(ROOT)}")
    if major(lxml_version) < 5:
        raise Failure(f"lxml>=5 is required; found {lxml_version}")
    if pyinstaller_version != pin:
        raise Failure(f"pyinstaller=={pin} is required; found {pyinstaller_version}")
    return lxml_version, pyinstaller_version


def check_vendor(vendor):
    missing = [item for item in VENDOR_ITEMS if not (vendor / item).exists()]
    if missing:
        raise Failure(
            f"Panda2Prusa is not vendored under {vendor}: missing {', '.join(missing)}. "
            "Expected third_party/panda2prusa/{panda2prusa/, LICENSE, slicebridge_runner.py}.")
    check_mit((vendor / "LICENSE").read_text(encoding="utf-8", errors="replace"),
              str(vendor / "LICENSE"))


def run(command, **kwargs):
    proc = subprocess.run(command, capture_output=True, text=True, **kwargs)
    if proc.returncode != 0:
        tail = (proc.stdout + proc.stderr).strip()[-3000:]
        raise Failure(f"{Path(command[0]).name} exited {proc.returncode}:\n{tail}")
    return proc


def vendor_commit(vendor):
    """Upstream commit of a vendored checkout. Only its own repository counts, not SliceBridge's."""
    if not (vendor / ".git").exists():
        return None
    try:
        proc = subprocess.run(["git", "-C", str(vendor), "rev-parse", "HEAD"],
                              capture_output=True, text=True, timeout=30)
    except (OSError, subprocess.TimeoutExpired):
        return None
    return proc.stdout.strip() if proc.returncode == 0 else None


def smoke_convert(exe, workdir):
    """Run the sidecar on the repo fixture and check the runner contract: exit 0, JSON stdout, ZIP output."""
    if not SMOKE_FIXTURE.is_file():
        raise Failure(f"smoke fixture missing: {SMOKE_FIXTURE}")
    source_before = sha256(SMOKE_FIXTURE)
    output = Path(workdir) / "smoke-output.3mf"
    proc = subprocess.run(
        [str(exe), "--source", str(SMOKE_FIXTURE), "--output", str(output), "--json"],
        capture_output=True, text=True, timeout=SMOKE_TIMEOUT, cwd=str(workdir))
    if proc.returncode != 0:
        raise CheckFailed(f"sidecar smoke conversion exited {proc.returncode}: "
                          f"{(proc.stderr or proc.stdout).strip()[-2000:]}")
    try:
        json.loads(proc.stdout)
    except ValueError:
        raise CheckFailed("sidecar --json output is not a JSON document on stdout")
    if not zipfile.is_zipfile(output):
        raise CheckFailed("sidecar smoke conversion did not write a 3MF ZIP")
    if sha256(SMOKE_FIXTURE) != source_before:
        raise CheckFailed("sidecar smoke conversion modified its source file")
    return output


def refuse_symlinks(*paths):
    for path in paths:
        if path.is_symlink():
            raise CheckFailed(f"refusing to use symlinked path {path}")


def install_file(source, dest, mode):
    """Copy via a temp file in the destination folder, then replace. Never follows a symlink."""
    refuse_symlinks(dest)
    if dest.is_dir():
        raise CheckFailed(f"destination is a directory: {dest}")
    temp = dest.with_name(dest.name + ".staging")
    refuse_symlinks(temp)
    if temp.exists():
        temp.unlink()
    shutil.copyfile(str(source), str(temp))
    os.chmod(str(temp), mode)
    os.replace(str(temp), str(dest))
    return dest


def codesign_adhoc(app, sidecar):
    if shutil.which("codesign") is None:
        raise Failure("codesign is required on macOS to re-seal the app after staging")
    run(["codesign", "--force", "--sign", "-", str(sidecar)])
    run(["codesign", "--force", "--sign", "-", str(app)])
    run(["codesign", "--verify", "--deep", "--strict", str(app)])


def dist_artifacts(dist, platform):
    sidecar = dist / exe_name(platform)
    license_file = dist / LICENSE_NAME
    manifest_file = dist / MANIFEST_NAME
    for path in (sidecar, license_file, manifest_file):
        if not path.is_file():
            raise Failure(f"missing {path}; run the build subcommand first")
    manifest = json.loads(manifest_file.read_text(encoding="utf-8"))
    if manifest.get("platform") != platform:
        raise CheckFailed(f"sidecar in {dist} was built for {manifest.get('platform')}, not {platform}")
    if manifest.get("sha256") != sha256(sidecar):
        raise CheckFailed(f"{sidecar} does not match its manifest hash")
    check_mit(license_file.read_text(encoding="utf-8", errors="replace"), str(license_file))
    return sidecar, license_file, manifest


def staged_targets(bin_dir, platform, create=False):
    """Return (directory that receives the sidecar and license, app path or None)."""
    if platform == "windows":
        app_exe = bin_dir / WIN_APP_EXE
        refuse_symlinks(bin_dir, app_exe)
        if not app_exe.is_file():
            raise Failure(f"{app_exe} is missing; build the Wails app before staging")
        return bin_dir, None
    app = bin_dir / MAC_APP
    refuse_symlinks(bin_dir, app)
    if not app.is_dir():
        raise Failure(f"{app} is missing; build the Wails app before staging")
    contents = app / "Contents"
    resources = contents / "Resources"
    refuse_symlinks(contents, resources)
    if create:
        resources.mkdir(parents=True, exist_ok=True)
    elif not resources.is_dir():
        raise CheckFailed(f"{resources} is missing; stage the sidecar first")
    return resources, app


def choose_platform(value):
    return host_platform() if value in (None, "auto") else value


def cmd_build(args):
    pin = pinned_pyinstaller()
    lxml_version, pyinstaller_version = check_build_prerequisites(pin)
    platform = host_platform()
    vendor = resolve(args.vendor_dir)
    check_vendor(vendor)
    output = resolve(args.output_dir)
    workdir = Path(tempfile.mkdtemp(prefix="prusa-sidecar-"))
    try:
        dist_tmp = workdir / "dist"
        command = [
            sys.executable, "-m", "PyInstaller", "--onefile", "--clean", "--noconfirm",
            "--name", SIDECAR,
            "--distpath", str(dist_tmp), "--workpath", str(workdir / "build"),
            "--specpath", str(workdir / "spec"),
            "--paths", str(vendor),
            "--collect-submodules", "lxml",
            "--collect-submodules", "panda2prusa",
            str(vendor / VENDOR_RUNNER),
        ]
        print("building", exe_name(platform), "with PyInstaller", pyinstaller_version)
        # PyInstaller refuses *any* working directory beneath Windows TEMP.
        # The clone may itself be in TEMP during developer smoke tests. Run
        # from the host home folder while all generated paths and entrypoint
        # are absolute and isolated under workdir.
        run(command, cwd=str(Path.home()))
        built = dist_tmp / exe_name(platform)
        if not built.is_file():
            raise Failure(f"PyInstaller did not produce {built}")
        if not args.skip_smoke:
            smoke_convert(built, workdir)
            print("smoke conversion of", SMOKE_FIXTURE.name, "passed")
        output.mkdir(parents=True, exist_ok=True)
        sidecar = install_file(built, output / exe_name(platform), 0o755)
        license_file = install_file(vendor / "LICENSE", output / LICENSE_NAME, 0o644)
        manifest = {
            "name": SIDECAR,
            "platform": platform,
            "file": exe_name(platform),
            "sha256": sha256(sidecar),
            "bytes": sidecar.stat().st_size,
            "license": LICENSE_NAME,
            "pyinstaller": pyinstaller_version,
            "lxml": lxml_version,
            "python": sys.version.split()[0],
            "vendor_commit": vendor_commit(vendor),
        }
        (output / MANIFEST_NAME).write_text(json.dumps(manifest, indent=2) + "\n", encoding="utf-8")
        print("built", sidecar)
        print("license", license_file)
        print("sha256", manifest["sha256"])
    finally:
        if args.keep_work:
            print("kept work directory", workdir)
        else:
            shutil.rmtree(str(workdir), ignore_errors=True)
    return 0


def make_windows_bundle(sidecar, license_file):
    output = io.BytesIO()
    with zipfile.ZipFile(output, "w", compression=zipfile.ZIP_STORED) as archive:
        for name, source in ((SIDECAR + ".exe", sidecar), (LICENSE_NAME, license_file)):
            info = zipfile.ZipInfo(name, date_time=(1980, 1, 1, 0, 0, 0))
            info.compress_type = zipfile.ZIP_STORED
            archive.writestr(info, source.read_bytes())
    result = output.getvalue()
    if not result or len(result) > PACK_MAX_BYTES:
        raise CheckFailed("Windows Prusa bundle is missing or oversized")
    return result


def inspect_windows_bundle(app, expected_engine_sha=None):
    """Verify exact packed bytes, overlay SHA, ZIP CRC, and the upstream MIT license."""
    with app.open("rb") as handle:
        if handle.read(2) != b"MZ":
            raise CheckFailed("Windows portable app is not a PE executable")
        handle.seek(0, os.SEEK_END)
        size = handle.tell()
        if size < PACK_TRAILER_SIZE:
            raise CheckFailed("Windows app has no embedded Prusa bundle")
        handle.seek(size - PACK_TRAILER_SIZE)
        trailer = handle.read(PACK_TRAILER_SIZE)
        if not trailer.startswith(PACK_MAGIC):
            raise CheckFailed("Windows app has no embedded Prusa bundle")
        offset = len(PACK_MAGIC)
        length = struct.unpack("<Q", trailer[offset:offset+8])[0]
        if not 0 < length <= PACK_MAX_BYTES or length > size - PACK_TRAILER_SIZE:
            raise CheckFailed("Windows embedded engine length is invalid")
        handle.seek(size - PACK_TRAILER_SIZE - length)
        payload = handle.read(length)
    digest = hashlib.sha256(payload).digest()
    if digest != trailer[-32:]:
        raise CheckFailed("Windows embedded engine SHA-256 mismatch")
    try:
        with zipfile.ZipFile(io.BytesIO(payload)) as archive:
            if sorted(archive.namelist()) != sorted(PACK_NAMES):
                raise CheckFailed("Windows embedded archive must contain only engine and license")
            if archive.testzip() is not None:
                raise CheckFailed("Windows embedded engine ZIP CRC failed")
            engine = archive.read(SIDECAR + ".exe")
            license_text = archive.read(LICENSE_NAME).decode("utf-8", "replace")
    except zipfile.BadZipFile as error:
        raise CheckFailed("Windows embedded archive is invalid: " + str(error))
    if not engine.startswith(b"MZ"):
        raise CheckFailed("embedded Prusa sidecar is not a PE executable")
    check_mit(license_text, "Windows embedded MIT license")
    engine_digest = hashlib.sha256(engine).hexdigest()
    if expected_engine_sha and engine_digest != expected_engine_sha:
        raise CheckFailed("Windows embedded engine differs from its build manifest")
    return {"payloadSha256": digest.hex(), "engineSha256": engine_digest,
            "payloadBytes": len(payload), "exeBytes": app.stat().st_size, "engine": engine}


def cmd_pack_windows(args):
    if host_platform() != "windows":
        raise Failure("Windows PE packing must run on the Windows build host")
    sidecar, license_file, manifest = dist_artifacts(resolve(args.dist_dir), "windows")
    app = resolve(args.bin_dir) / WIN_APP_EXE
    refuse_symlinks(app)
    if not app.is_file():
        raise Failure("build the Windows Wails app before embedding the Prusa engine")
    with app.open("rb") as handle:
        if handle.read(2) != b"MZ":
            raise CheckFailed("the Wails Windows executable is not a PE file")
        if app.stat().st_size >= PACK_TRAILER_SIZE:
            handle.seek(-PACK_TRAILER_SIZE, os.SEEK_END)
            if handle.read(len(PACK_MAGIC)) == PACK_MAGIC:
                raise CheckFailed("Windows app already has a Prusa payload; never append twice")
    payload = make_windows_bundle(sidecar, license_file)
    tmp = app.with_name(app.name + ".packing")
    refuse_symlinks(tmp)
    if tmp.exists():
        raise CheckFailed("stale Windows packing staging file exists")
    try:
        with app.open("rb") as inp, tmp.open("xb") as out:
            shutil.copyfileobj(inp, out, length=1 << 20)
            out.write(payload)
            out.write(PACK_MAGIC)
            out.write(struct.pack("<Q", len(payload)))
            out.write(hashlib.sha256(payload).digest())
        os.chmod(tmp, app.stat().st_mode)
        os.replace(tmp, app)
    finally:
        if tmp.exists():
            tmp.unlink()
    info = inspect_windows_bundle(app, manifest["sha256"])
    print(json.dumps({k:v for k,v in info.items() if k!="engine"}, indent=2))
    return 0


def cmd_stage(args):
    platform = choose_platform(args.platform)
    if platform not in PLATFORMS:
        raise Failure(f"unknown platform {platform}")
    bin_dir = resolve(args.bin_dir)
    sidecar, license_file, _ = dist_artifacts(resolve(args.dist_dir), platform)
    if platform == "windows":
        raise Failure("Windows users need one EXE. Use pack-windows instead of stage.")
    target, app = staged_targets(bin_dir, platform, create=True)
    staged_sidecar = install_file(sidecar, target / exe_name(platform), 0o755)
    staged_license = install_file(license_file, target / LICENSE_NAME, 0o644)
    if platform == "macos" and not args.no_codesign:
        codesign_adhoc(app, staged_sidecar)
    print("staged", staged_sidecar)
    print("staged", staged_license)
    if app is not None:
        print("app", app, "(ad-hoc signed)" if not args.no_codesign else "(not re-signed)")
    return 0


def cmd_verify(args):
    platform = choose_platform(args.platform)
    if platform not in PLATFORMS:
        raise Failure(f"unknown platform {platform}")
    bin_dir = resolve(args.bin_dir)
    dist = resolve(args.dist_dir)
    sidecar_built, _, manifest = dist_artifacts(dist, platform)
    if platform == "windows":
        packed_app = bin_dir / WIN_APP_EXE
        info = inspect_windows_bundle(packed_app, manifest["sha256"])
        if args.smoke:
            if host_platform() != platform:
                raise Failure("--smoke must run on Windows for a Windows bundle")
            with tempfile.TemporaryDirectory(prefix="slicebridge-embedded-prusa-smoke-") as folder:
                extracted = Path(folder) / (SIDECAR + ".exe")
                extracted.write_bytes(info["engine"])
                smoke_convert(extracted, folder)
        print(json.dumps({k:v for k,v in info.items() if k!="engine"}, indent=2))
        return 0
    target, app = staged_targets(bin_dir, platform)
    staged_sidecar = target / exe_name(platform)
    staged_license = target / LICENSE_NAME
    for path in (staged_sidecar, staged_license):
        if not path.is_file() or path.is_symlink():
            raise CheckFailed(f"staged file missing or not a regular file: {path}")
    if sha256(staged_sidecar) != manifest["sha256"]:
        raise CheckFailed(f"staged {staged_sidecar} does not match the built sidecar")
    check_mit(staged_license.read_text(encoding="utf-8", errors="replace"), str(staged_license))
    if platform == "macos":
        if not os.access(str(staged_sidecar), os.X_OK):
            raise CheckFailed(f"staged sidecar is not executable: {staged_sidecar}")
        if not args.no_codesign:
            if shutil.which("codesign") is None:
                raise Failure("codesign is required to verify the macOS app seal")
            run(["codesign", "--verify", "--deep", "--strict", str(app)])
    if args.smoke:
        if host_platform() != platform:
            raise Failure("--smoke runs the staged sidecar, so it needs a host of the same platform")
        with tempfile.TemporaryDirectory(prefix="prusa-sidecar-verify-") as workdir:
            smoke_convert(staged_sidecar, workdir)
    print(json.dumps({"platform": platform, "sidecar": str(staged_sidecar),
                      "license": str(staged_license), "sha256": manifest["sha256"],
                      "smoke": bool(args.smoke)}, indent=2))
    return 0


CI_STEPS = """\
# Windows job (native.yml, windows-native). Run from the repository root.
# Add these steps after "Build native Windows single exe" and before artifact upload.
- name: Set up Python for the Panda2Prusa sidecar
  uses: actions/setup-python@v6
  with:
    python-version: '3.12'
- name: Build Panda2Prusa sidecar
  working-directory: .
  shell: pwsh
  run: |
    python -m pip install --no-cache-dir -r scripts/prusa-sidecar-requirements.txt
    if ($LASTEXITCODE -ne 0) { throw 'pip install failed' }
    python scripts/package-prusa-sidecar.py build
    if ($LASTEXITCODE -ne 0) { throw 'Panda2Prusa sidecar build failed' }
- name: Embed, verify and smoke-test the Prusa engine in SliceBridge.exe
  working-directory: .
  shell: pwsh
  run: |
    python scripts/package-prusa-sidecar.py pack-windows --bin-dir native/build/bin
    if ($LASTEXITCODE -ne 0) { throw 'Windows one-file packing failed' }
    python scripts/package-prusa-sidecar.py verify --platform windows --bin-dir native/build/bin --smoke
    if ($LASTEXITCODE -ne 0) { throw 'Embedded engine smoke failed' }
    & native/build/bin/SliceBridge.exe '--verify-bundled-prusa'
    if ($LASTEXITCODE -ne 0) { throw 'The actual EXE cannot load its embedded engine' }
# Upload native/build/bin/SliceBridge.exe directly; do not create a ZIP of multiple executables.

# macOS job (native.yml, macos-native). Run from the repository root.
# Add these steps after "Build macOS app using system WebKit" and before "Check native app size and package".
- name: Set up Python for the Panda2Prusa sidecar
  uses: actions/setup-python@v6
  with:
    python-version: '3.12'
- name: Build, stage and verify Panda2Prusa sidecar
  working-directory: .
  run: |
    python3 -m pip install --no-cache-dir -r scripts/prusa-sidecar-requirements.txt
    python3 scripts/package-prusa-sidecar.py build
    python3 scripts/package-prusa-sidecar.py stage --platform macos --bin-dir native/build/bin
    python3 scripts/package-prusa-sidecar.py verify --platform macos --bin-dir native/build/bin --smoke
# The existing ditto step already packs the whole .app, so Contents/Resources ships unchanged.
"""


def cmd_print_ci_steps(_args):
    print(CI_STEPS, end="")
    return 0


def build_parser():
    parser = argparse.ArgumentParser(description="Build, stage and verify the Panda2Prusa sidecar.")
    commands = parser.add_subparsers(dest="command", required=True)

    build = commands.add_parser("build", help="PyInstaller onefile build plus smoke conversion")
    build.add_argument("--vendor-dir", default=str(VENDOR_DIR))
    build.add_argument("--output-dir", default=str(DIST_DIR))
    build.add_argument("--skip-smoke", action="store_true", help="skip the fixture conversion")
    build.add_argument("--keep-work", action="store_true", help="keep the PyInstaller work directory")
    build.set_defaults(func=cmd_build)

    pack = commands.add_parser("pack-windows", help="Embed Python Prusa engine inside one Windows SliceBridge.exe")
    pack.add_argument("--bin-dir", required=True, help="native/build/bin for the Wails output")
    pack.add_argument("--dist-dir", default=str(DIST_DIR))
    pack.set_defaults(func=cmd_pack_windows)

    for name, func in (("stage", cmd_stage), ("verify", cmd_verify)):
        sub = commands.add_parser(name)
        sub.add_argument("--platform", choices=("auto",) + PLATFORMS, default="auto")
        sub.add_argument("--bin-dir", required=True, help="native/build/bin for the Wails output")
        sub.add_argument("--dist-dir", default=str(DIST_DIR))
        sub.add_argument("--no-codesign", action="store_true",
                         help="macOS: skip the ad-hoc re-seal (stage) or the seal check (verify)")
        if name == "verify":
            sub.add_argument("--smoke", action="store_true", help="run the staged sidecar on the fixture")
        sub.set_defaults(func=func)

    ci = commands.add_parser("print-ci-steps", help="print the workflow steps for CI")
    ci.set_defaults(func=cmd_print_ci_steps)
    return parser


def main(argv=None):
    args = build_parser().parse_args(argv)
    try:
        return args.func(args)
    except Failure as error:
        print("ERROR:", error, file=sys.stderr)
        return error.exit_code


if __name__ == "__main__":
    sys.exit(main())
