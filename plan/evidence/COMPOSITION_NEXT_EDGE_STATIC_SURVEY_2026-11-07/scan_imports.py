#!/usr/bin/env python3
"""Repository-wide import scan used by the 2026-11-07 evidence note.

Reads every *.go file, extracts tei/internal/* import strings, and reports:
  * files that import both internal/operations and internal/composition
  * files that import internal/servicehost (and whether any is non-test)
  * files that put internal/composition and internal/plugin/runtime together
It does NOT prove runtime composition; it only records the import graph shape.

Usage: scan_imports.py <repo-root> <output.json>
"""
import json
import os
import re
import sys

TARGETS = [
    "tei/internal/composition",
    "tei/internal/operations",
    "tei/internal/servicehost",
    "tei/internal/plugin/runtime",
    "tei/internal/driver",
    "tei/internal/command",
]

IMPORT_RE = re.compile(r'"(tei/internal/[^"]+)"')


def main() -> int:
    root, out = sys.argv[1], sys.argv[2]
    files = []
    for dirpath, dirnames, filenames in os.walk(root):
        dirnames[:] = [d for d in dirnames if d not in {".git", "node_modules"}]
        for name in filenames:
            if name.endswith(".go"):
                files.append(os.path.join(dirpath, name))
    rows = []
    scanned = 0
    for path in sorted(files):
        scanned += 1
        with open(path, encoding="utf-8", errors="replace") as handle:
            text = handle.read()
        imports = set(IMPORT_RE.findall(text))
        if not imports.intersection(TARGETS):
            continue
        relative = os.path.relpath(path, root).replace(os.sep, "/")
        rows.append({
            "path": relative,
            "isTest": relative.endswith("_test.go"),
            "has": {t: (t in imports) for t in TARGETS},
        })
    payload = {"root": root, "fileCount": scanned, "targets": TARGETS, "files": rows}
    with open(out, "w", encoding="utf-8") as handle:
        json.dump(payload, handle, ensure_ascii=False, indent=1)
    print("scanned", scanned, "files;", len(rows), "import a target ->", out)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
