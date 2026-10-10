"""Prepare the fork's existing update-feed schema from the exact release files."""
import hashlib
import json
import re
import sys
from pathlib import Path

version, directory, notes_path = sys.argv[1:]
if not re.fullmatch(r"[0-9]+\.[0-9]+\.[0-9]+(?:-[A-Za-z0-9.-]+)?", version):
    raise SystemExit("invalid release version")
directory = Path(directory)
base = f"https://github.com/enderzcx/magpie/releases/download/v{version}"
files = sorted(p for p in directory.iterdir() if p.is_file() and p.name.startswith("magpie-"))
expected = {f"magpie-cli-{os}-{arch}{'.exe' if os == 'windows' else ''}" for os, arch in [
    ("darwin", "amd64"), ("darwin", "arm64"), ("linux", "amd64"), ("linux", "arm64"),
    ("windows", "amd64"), ("windows", "arm64"), ("android", "arm64")]}
expected |= {f"magpie-{os}-{arch}{'.zip' if os == 'darwin' else '.exe' if os == 'windows' else ''}" for os in ["darwin", "linux", "windows"] for arch in ["amd64", "arm64"]}
if {p.name for p in files} != expected:
    raise SystemExit("release asset set is incomplete or unexpected")
assets = {p.name: {"url": f"{base}/{p.name}", "size": p.stat().st_size, "sha256": hashlib.file_digest(p.open("rb"), "sha256").hexdigest()} for p in files}
notes = Path(notes_path).read_text()
release = {"version": version, "notes": notes, "url": f"https://github.com/enderzcx/magpie/releases/tag/v{version}", "assets": assets}
(directory / "update.json").write_text(json.dumps(release, indent=2) + "\n")
(directory / "notes.json").write_text(json.dumps({"releases": [{k: release[k] for k in ("version", "notes", "url")}]}, indent=2) + "\n")
(directory / "checksums.txt").write_text("".join(f"{a['sha256']}  {name}\n" for name, a in assets.items()))
