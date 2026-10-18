#!/usr/bin/env python3
"""List every Go fuzz target in the repository as a JSON array.

Emits one entry per `func FuzzXxx(f *testing.F)` in a *_test.go file:

    [{"module": ".", "pkg": "./internal/pricing", "target": "FuzzComputePrice"}]

`module` is the directory of the nearest ancestor go.mod (the repository
contains a second module under tools/print-agent), `pkg` is the package path
relative to that module, and `target` is the fuzz function name. Used by
.github/workflows/fuzz.yml to fan out one parallel job per target.
"""

import json
import os
import re
import sys

# Directories that never contain first-party Go sources worth fuzzing.
SKIP_DIRS = {
    ".git",
    ".opencode",
    "node_modules",
    "vendor",
    "dist",
    "build",
    "bin",
}

FUZZ_FUNC = re.compile(r"^func (Fuzz[A-Za-z0-9_]*)\(", re.MULTILINE)


def module_root(path: str) -> str:
    """Return the directory of the nearest ancestor go.mod, or '.'."""
    probe = os.path.dirname(os.path.abspath(path))
    while True:
        if os.path.isfile(os.path.join(probe, "go.mod")):
            return probe
        parent = os.path.dirname(probe)
        if parent == probe:
            return os.getcwd()
        probe = parent


def collect(root: str) -> list[dict[str, str]]:
    repo = os.path.abspath(root)
    targets: list[dict[str, str]] = []
    for dirpath, dirnames, filenames in os.walk(repo):
        dirnames[:] = sorted(d for d in dirnames if d not in SKIP_DIRS)
        for name in sorted(filenames):
            if not name.endswith("_test.go"):
                continue
            path = os.path.join(dirpath, name)
            try:
                with open(path, encoding="utf-8") as handle:
                    text = handle.read()
            except OSError as err:  # pragma: no cover - unreadable file
                print(f"warning: skipping {path}: {err}", file=sys.stderr)
                continue
            names = FUZZ_FUNC.findall(text)
            if not names:
                continue
            module = os.path.relpath(module_root(path), repo)
            rel = os.path.relpath(dirpath, os.path.join(repo, module))
            targets.extend(
                {
                    "module": module,
                    "pkg": "." if rel == "." else "./" + rel.replace(os.sep, "/"),
                    "target": target,
                }
                for target in names
            )
    targets.sort(key=lambda t: (t["module"], t["pkg"], t["target"]))
    return targets


def main() -> int:
    targets = collect(sys.argv[1] if len(sys.argv) > 1 else ".")
    json.dump(targets, sys.stdout, separators=(",", ":"))
    sys.stdout.write("\n")
    return 0


if __name__ == "__main__":
    sys.exit(main())
