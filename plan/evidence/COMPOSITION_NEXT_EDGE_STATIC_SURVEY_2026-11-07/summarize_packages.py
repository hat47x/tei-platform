#!/usr/bin/env python3
"""Summarize `go list -json -deps ./...` for the evidence note.

Reads the JSON stream (one package object per document) and writes the
package-level closure table quoted in section 2.2.

Usage:
    go list -json -deps ./... > deps.json
    summarize_packages.py deps.json package-summary.txt

The script only reads imports/closures; it never executes the packages.
"""
import json
import sys

TARGETS = [
    ("composition", "tei/internal/composition"),
    ("operations", "tei/internal/operations"),
    ("pluginRuntime", "tei/internal/plugin/runtime"),
    ("servicehost", "tei/internal/servicehost"),
    ("command", "tei/internal/command"),
]


def read_stream(path):
    records, buf = [], []
    with open(path, encoding="utf-8") as handle:
        for line in handle:
            buf.append(line)
            if line.rstrip() == "}":
                try:
                    records.append(json.loads("".join(buf)))
                except ValueError:
                    pass
                buf = []
    return [r for r in records if r.get("ImportPath", "").startswith("tei/")]


def main() -> int:
    records = read_stream(sys.argv[1])
    mains = [r for r in records if r.get("Name") == "main"]
    flags = {}
    for rec in records:
        deps = set(rec.get("Deps") or [])
        flags[rec["ImportPath"]] = {k: (p in deps) for k, p in TARGETS}

    lines = []
    lines.append("repository packages scanned: %d" % len(records))
    lines.append("main packages: %d" % len(mains))

    def show(title, predicate):
        rows = sorted(r["ImportPath"] for r in mains if predicate(flags[r["ImportPath"]]))
        lines.append(title + ": %d" % len(rows))
        for r in rows:
            lines.append("  " + r)

    show("main packages whose closure includes internal/composition",
         lambda f: f["composition"])
    show("main packages whose closure includes internal/operations",
         lambda f: f["operations"])
    show("main packages whose closure includes internal/plugin/runtime",
         lambda f: f["pluginRuntime"])
    show("main packages whose closure includes BOTH composition and operations",
         lambda f: f["composition"] and f["operations"])
    show("main packages whose closure includes BOTH composition and plugin/runtime",
         lambda f: f["composition"] and f["pluginRuntime"])
    rows = sorted(p for p, f in flags.items() if f["servicehost"])
    lines.append("packages whose closure includes internal/servicehost: %d" % len(rows))
    for r in rows:
        lines.append("  " + r)

    with open(sys.argv[2], "w", encoding="utf-8", newline="\n") as handle:
        handle.write("\n".join(lines) + "\n")
    print("wrote", sys.argv[2])
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
