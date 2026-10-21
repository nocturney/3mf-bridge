"""Keep the repository root small and the moved documents reachable."""
from __future__ import annotations
import json
import re
import subprocess
import unittest
from pathlib import Path
from urllib.parse import unquote

ROOT = Path(__file__).resolve().parents[1]

# Only these files may live at the root of the repository.
ROOT_FILES = {
    ".gitignore", ".gitmodules", "CHANGELOG.md", "LICENSE", "README.md",
    "VERSION", "package-lock.json", "package.json",
}
# Public site entry point, served by GitHub Pages from main/docs.
PUBLIC_SITE = "docs/index.html"
# Files GitHub recognises in .github/ (community health and policy files).
GITHUB_POLICY = {"CODE_OF_CONDUCT.md", "CONTRIBUTING.md", "SECURITY.md", "SUPPORT.md"}
MOVED_DOCS = {
    "docs/legal": {"GO_LICENSE_AUDIT.md", "LICENSE_AUDIT.md", "LICENSING.md",
                   "PRIVACY.md", "THIRD_PARTY_NOTICES.md", "TRADEMARKS.md"},
    "docs/release": {"DISTRIBUTION_CHECKLIST.md", "RELEASE_NOTES_TEMPLATE.md",
                     "RELEASE_OPERATIONS.md", "RELEASE_PREFLIGHT.md", "VERSIONING.md"},
    "docs/help": {"FAQ.md"},
    "docs/community": {"BETA_TESTING.md"},
}
ELECTRON_FILES = {"main.js", "index.html", "preload.js", "slicer-profiles.js", "window-sizing.js"}

MD_LINK = re.compile(r"\]\(([^)\s]+)")
HTML_LINK = re.compile(r"""(?:href|src)=["']([^"']+)["']""", re.I)


def tracked_files() -> list[str]:
    result = subprocess.run(["git", "-C", str(ROOT), "ls-files"],
                            capture_output=True, text=True, check=True)
    return result.stdout.split()


class RepositoryLayout(unittest.TestCase):
    def setUp(self):
        self.tracked = tracked_files()

    def test_root_contains_only_required_files(self):
        root_files = {path for path in self.tracked if "/" not in path}
        self.assertEqual(root_files, ROOT_FILES)

    def test_electron_app_lives_under_desktop(self):
        package = json.loads((ROOT / "package.json").read_text(encoding="utf-8"))
        self.assertEqual(package["main"], "desktop/electron/main.js")
        for name in ELECTRON_FILES:
            self.assertTrue((ROOT / "desktop" / "electron" / name).is_file(), name)

    def test_public_site_is_docs_index(self):
        self.assertIn(PUBLIC_SITE, self.tracked)
        self.assertNotIn("index.html", self.tracked)

    def test_policy_files_are_in_github_folder(self):
        for name in GITHUB_POLICY:
            self.assertIn(".github/" + name, self.tracked, name)

    def test_auxiliary_docs_are_in_thematic_folders(self):
        for folder, names in MOVED_DOCS.items():
            for name in names:
                self.assertIn(folder + "/" + name, self.tracked, name)

    def test_relative_links_in_documents_resolve(self):
        broken = []
        documents = [p for p in self.tracked
                     if p.endswith((".md", ".html")) and not p.startswith("native/frontend/dist/")]
        for rel in documents:
            text = (ROOT / rel).read_text(encoding="utf-8", errors="replace")
            for target in MD_LINK.findall(text) + HTML_LINK.findall(text):
                if re.match(r"^[a-zA-Z][a-zA-Z0-9+.-]*:", target) or target.startswith(("#", "/", "<")):
                    continue
                path_part = unquote(target.split("#", 1)[0].split("?", 1)[0])
                if path_part and not (ROOT / rel).parent.joinpath(path_part).exists():
                    broken.append(f"{rel}: {target}")
        self.assertEqual(broken, [])


if __name__ == "__main__":
    unittest.main()
