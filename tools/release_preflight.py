#!/usr/bin/env python3
"""Create nonpublishing release evidence from a pinned SliceBridge checkout.

Read-only source inspection; never builds, executes, signs or publishes applications.
"""
from __future__ import annotations
import argparse
from datetime import datetime, timezone
import hashlib
import json
from pathlib import Path, PurePosixPath
import re
import subprocess
import sys
import tempfile
import urllib.parse
import zipfile

MAX_ASSET_BYTES = 512 * 1024 * 1024
MAX_UNCOMPRESSED_BYTES = 1024 * 1024 * 1024
MAX_BINARY_BYTES = 100 * 1024 * 1024
KNOWN_SPDX = {"MIT", "ISC", "Apache-2.0", "BSD-2-Clause", "BSD-3-Clause",
              "GPL-3.0-only", "GPL-3.0-or-later", "MPL-2.0", "Unlicense", "CC0-1.0", "Zlib", "0BSD"}

def git(root: Path, *args: str, binary: bool = False):
    p = subprocess.run(["git", "-C", str(root), *args], capture_output=True)
    if p.returncode:
        raise RuntimeError("Git failed (" + " ".join(args) + "): " + p.stderr.decode("utf-8", "replace")[:500])
    return p.stdout if binary else p.stdout.decode("utf-8", "replace").strip()

def sha40(value: str) -> str:
    if not re.fullmatch(r"[0-9a-fA-F]{40}", value):
        raise ValueError("An immutable 40-character SHA is required")
    return value.lower()

def safe_path(name: str):
    p = PurePosixPath(name)
    if not name or name.startswith("/") or "\\" in name or "\x00" in name or ".." in p.parts:
        raise ValueError(f"Unsafe archive path: {name!r}")

def snapshot(root: Path, expected: str):
    """Use exact Git blob bytes, including initialized and pinned recursive submodules."""
    files, sources, visiting = {}, [], set()
    def walk(repo, sha, prefix):
        resolved = str(repo.resolve())
        if resolved in visiting:
            raise ValueError("Recursive submodule detected")
        visiting.add(resolved)
        if sha40(git(repo, "rev-parse", "HEAD")) != sha:
            raise ValueError("Source checkout does not match pinned commit: " + prefix)
        if git(repo, "status", "--porcelain", "--untracked-files=no"):
            raise ValueError("Dirty tracked source at: " + (prefix or "."))
        tree = git(repo, "ls-tree", "-r", "-z", sha, binary=True)
        for record in tree.split(b"\x00"):
            if not record:
                continue
            header, name_b = record.split(b"\t", 1)
            mode, kind, blobsha = header.decode("ascii").split(" ")
            rel = name_b.decode("utf-8", "surrogateescape")
            name = prefix + rel
            safe_path(name)
            if kind == "commit":
                sub = repo.joinpath(*PurePosixPath(rel).parts)
                if not (sub / ".git").exists():
                    raise ValueError("Submodule not initialized: " + name)
                walk(sub, sha40(blobsha), name + "/")
            elif kind == "blob":
                if name in files:
                    raise ValueError("Duplicate source path: " + name)
                files[name] = (git(repo, "cat-file", "blob", blobsha, binary=True), int(mode, 8))
            else:
                raise ValueError("Unknown tree object type: " + kind)
        sources.append({"path": prefix or ".", "commit": sha})
        visiting.remove(resolved)
    walk(root, expected, "")
    return files, sorted(sources, key=lambda s: s["path"])

def source_zip(files, dest, sha):
    with zipfile.ZipFile(dest, "w", compression=zipfile.ZIP_DEFLATED, compresslevel=9) as z:
        for name, (data, mode) in sorted(files.items()):
            arcname = f"slicebridge-source-{sha[:12]}/{name}"
            safe_path(arcname)
            info = zipfile.ZipInfo(arcname, date_time=(1980, 1, 1, 0, 0, 0))
            info.create_system, info.external_attr = 3, mode << 16
            info.compress_type = zipfile.ZIP_DEFLATED
            z.writestr(info, data, compress_type=zipfile.ZIP_DEFLATED, compresslevel=9)

def declared_dependencies(files):
    deps, manifests = {}, []
    for path, (raw, _) in sorted(files.items()):
        if path.endswith("package-lock.json"):
            lock = json.loads(raw.decode("utf-8"))
            manifests.append(path)
            for key, value in lock.get("packages", {}).items():
                if "node_modules/" not in key or not isinstance(value, dict):
                    continue
                name, ver = key.rsplit("node_modules/", 1)[-1], value.get("version")
                if isinstance(ver, str) and ver:
                    ident = ("npm", name, ver)
                    obj = deps.setdefault(ident, {"evidence": set(), "license": None})
                    obj["evidence"].add(path)
                    if value.get("license") in KNOWN_SPDX:
                        obj["license"] = value["license"]
        elif path.endswith("go.mod"):
            manifests.append(path)
            for line in raw.decode("utf-8", "replace").splitlines():
                line = line.split("//", 1)[0].strip()
                if line.startswith("require "):
                    line = line[len("require "):].strip()
                match = re.fullmatch(r"([^\s()]+)\s+(v[0-9][^\s]+)(?:\s+.*)?", line)
                if match and ("." in match.group(1) or "/" in match.group(1)):
                    ident = ("go", match.group(1), match.group(2))
                    deps.setdefault(ident, {"evidence": set(), "license": None})["evidence"].add(path)
    return deps, manifests

def binary_metadata(binary: bytes):
    """Read Go build info without executing the provided binary."""
    with tempfile.TemporaryDirectory(prefix="slicebridge-go-info-") as tmp:
        path = Path(tmp) / "binary"
        path.write_bytes(binary)
        try:
            r = subprocess.run(["go", "version", "-m", str(path)], capture_output=True,
                               text=True, timeout=30)
        except (FileNotFoundError, subprocess.TimeoutExpired):
            return None
        if r.returncode:
            return None
    modules = []
    for line in r.stdout.splitlines():
        match = re.match(r"\s*dep\s+([^\s]+)\s+(v[^\s]+)", line)
        if match:
            modules.append(("go", match.group(1), match.group(2)))
    commit = re.search(r"main\.buildCommit=([0-9a-fA-F]{40})", r.stdout)
    compiler = re.search(r"go1\.\d+(?:\.\d+)?", r.stdout)
    return {"modules": modules, "commit": commit.group(1).lower() if commit else None,
            "compiler": compiler.group(0) if compiler else None}

def draft_assets(asset_dir, source_sha):
    results, module_evidence, warnings = [], {}, []
    if asset_dir is None:
        return results, module_evidence, warnings
    if not asset_dir.is_dir():
        raise ValueError("Draft asset directory does not exist")
    paths = sorted(p for p in asset_dir.iterdir() if p.is_file() and p.suffix.lower() == ".zip")
    if len(paths) > 20:
        raise ValueError("Too many draft assets")
    for path in paths:
        if path.stat().st_size > MAX_ASSET_BYTES:
            raise ValueError("Oversized ZIP: " + path.name)
        result = {"name": path.name, "bytes": path.stat().st_size,
                  "sha256": hashlib.sha256(path.read_bytes()).hexdigest(),
                  "signingStatus": "NOT CHECKED (requires platform-specific trust checks)",
                  "goBuilds": []}
        with zipfile.ZipFile(path) as z:
            members = [m for m in z.infolist() if not m.is_dir()]
            if len(members) > 10000 or sum(m.file_size for m in members) > MAX_UNCOMPRESSED_BYTES:
                raise ValueError("Excessive archive expansion: " + path.name)
            seen = set()
            for m in members:
                safe_path(m.filename)
                if m.filename.casefold() in seen:
                    raise ValueError("Duplicate member name: " + m.filename)
                seen.add(m.filename.casefold())
            if z.testzip() is not None:
                raise ValueError("ZIP CRC failed: " + path.name)
            result["zipCrcPass"] = True
            result["entryCount"] = len(members)
            for m in members:
                low = m.filename.lower()
                if m.file_size > MAX_BINARY_BYTES or not (low.endswith(".exe") or "/contents/macos/" in low):
                    continue
                info = binary_metadata(z.read(m.filename))
                if info is not None:
                    result["goBuilds"].append({"path": m.filename, "compiler": info["compiler"],
                                                "buildCommit": info["commit"],
                                                "dependencyCount": len(info["modules"])})
                    if info["commit"] != source_sha:
                        warnings.append(path.name + ": embedded commit " + str(info["commit"])
                                        + " does not match requested " + source_sha)
                    for dep in info["modules"]:
                        module_evidence.setdefault(dep, set()).add(path.name)
        results.append(result)
    return results, module_evidence, warnings

def sbom(deps, binary_deps, sha, when):
    expanded = {k: {"evidence": set(v["evidence"]), "license": v["license"]} for k, v in deps.items()}
    for key, archives in binary_deps.items():
        expanded.setdefault(key, {"evidence": set(), "license": None})["evidence"].update(
            "Go binary build info: " + p for p in archives)
    components = []
    for (kind, name, ver), info in sorted(expanded.items()):
        ecosystem = "golang" if kind == "go" else "npm"
        purl = "pkg:" + ecosystem + "/" + urllib.parse.quote(name, safe="/@") + "@" + urllib.parse.quote(ver, safe=".+-_")
        component = {"type": "library", "name": name, "version": ver, "purl": purl,
                     "properties": [{"name": "slicebridge:evidence", "value": e}
                                    for e in sorted(info["evidence"])]}
        if info["license"] in KNOWN_SPDX:
            component["licenses"] = [{"license": {"id": info["license"]}}]
        components.append(component)
    return {
        "bomFormat": "CycloneDX", "specVersion": "1.5", "version": 1,
        "metadata": {"timestamp": when,
                     "component": {"type": "application", "name": "SliceBridge-source-evidence",
                                   "version": sha[:12]},
                     "properties": [
                         {"name": "slicebridge:source-sha", "value": sha},
                         {"name": "slicebridge:scope", "value": "Manifest and Go build-info evidence only; NOT a final packaged-binary SBOM"},
                         {"name": "slicebridge:release-approved", "value": "false"},
                     ]},
        "components": components,
    }

def write_json(path, obj):
    path.write_text(json.dumps(obj, indent=2, sort_keys=True, ensure_ascii=False) + "\n", encoding="utf-8")

def main():
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("--repo", type=Path, required=True)
    ap.add_argument("--out", type=Path, required=True)
    ap.add_argument("--source-sha", required=True)
    ap.add_argument("--assets-dir", type=Path)
    args = ap.parse_args()
    sha = sha40(args.source_sha)
    repo, out = args.repo.resolve(), args.out.resolve()
    if repo == out or repo in out.parents:
        raise ValueError("Output must be outside the source checkout")
    if out.exists() and any(out.iterdir()):
        raise ValueError("Output must be empty (no stale artifacts)")
    if sha40(git(repo, "rev-parse", "HEAD")) != sha:
        raise ValueError("Source checkout SHA does not match the requested immutable SHA")
    files, submodules = snapshot(repo, sha)
    out.mkdir(parents=True, exist_ok=True)
    archive_name = f"slicebridge-source-{sha[:12]}.zip"
    source_zip(files, out / archive_name, sha)
    with zipfile.ZipFile(out / archive_name) as archive:
        if archive.testzip() is not None:
            raise ValueError("Source ZIP CRC check failed")
    deps, manifests = declared_dependencies(files)
    draft, bin_deps, warnings = draft_assets(args.assets_dir, sha)
    timestamp = git(repo, "show", "-s", "--format=%cI", sha) or datetime.now(timezone.utc).isoformat()
    write_json(out / "manifest-sbom.cdx.json", sbom(deps, bin_deps, sha, timestamp))
    report = {
        "status": "PREFLIGHT EVIDENCE ONLY — NO RELEASE AUTHORIZED",
        "sourceCommit": sha, "sourceArchive": archive_name,
        "sourceFileCount": len(files), "pinnedSourceTrees": submodules,
        "manifests": manifests, "manifestDependencyCount": len(deps),
        "binaryEvidenceDependencyCount": len(bin_deps),
        "draftAssets": draft, "warnings": warnings,
        "releaseGates": {
            "approved": False, "rightsReviewed": False, "correspondingSourceVerified": False,
            "binarySBOMComplete": False, "signed": False, "notarized": False,
            "physicalPrinterTestsPassed": False, "published": False,
        },
    }
    write_json(out / "release-evidence.json", report)
    if draft:
        (out / "DRAFT_ASSET_SHA256SUMS.txt").write_text(
            "".join(f"{x['sha256']}  {x['name']}\n" for x in draft), encoding="utf-8")
    entries = sorted(p for p in out.iterdir() if p.is_file())
    (out / "SHA256SUMS.txt").write_text(
        "".join(f"{hashlib.sha256(p.read_bytes()).hexdigest()}  {p.name}\n" for p in entries),
        encoding="utf-8")
    print(json.dumps({"sourceCommit": sha, "sourceFiles": len(files),
                      "manifestDependencies": len(deps), "draftAssets": len(draft),
                      "warnings": warnings, "status": report["status"]}, ensure_ascii=False))
    return 0

if __name__ == "__main__":
    try:
        sys.exit(main())
    except (ValueError, RuntimeError, OSError, zipfile.BadZipFile, json.JSONDecodeError) as e:
        print("PREFLIGHT FAILED: " + str(e), file=sys.stderr)
        sys.exit(1)
