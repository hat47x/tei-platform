#!/usr/bin/env python3
"""Summarize import-graph.json into the numbers quoted by the evidence note.

Usage: summarize_imports.py <import-graph.json> <summary.txt>
"""
import json
import sys

TARGETS = [
    "tei/internal/composition",
    "tei/internal/operations",
    "tei/internal/servicehost",
    "tei/internal/plugin/runtime",
]


def main() -> int:
    data = json.load(open(sys.argv[1], encoding="utf-8"))
    out = open(sys.argv[2], "w", encoding="utf-8", newline="\n")
    files = data["files"]

    def count(target):
        return sum(1 for f in files if f["has"][target])

    def list_for(predicate):
        return [f["path"] for f in files if predicate(f)]

    out.write("files scanned: %d\n" % data["fileCount"])
    for t in TARGETS:
        out.write("files importing %s: %d\n" % (t, count(t)))
    out.write("\n")

    both_ops_comp = list_for(lambda f: f["has"][TARGETS[1]] and f["has"][TARGETS[0]])
    out.write("files importing BOTH internal/operations and internal/composition: %d\n"
              % len(both_ops_comp))
    for p in both_ops_comp:
        out.write("  %s\n" % p)
    out.write("\n")

    svc = list_for(lambda f: f["has"][TARGETS[2]])
    out.write("files importing internal/servicehost: %d\n" % len(svc))
    for p in svc:
        out.write("  %s%s\n" % (p, "  (test)" if p.endswith("_test.go") else "  (NON-TEST)"))
    out.write("non-test files importing internal/servicehost: %d\n"
              % sum(1 for p in svc if not p.endswith("_test.go")))
    out.write("\n")

    comp_runtime = list_for(lambda f: f["has"][TARGETS[0]] and f["has"][TARGETS[3]])
    out.write("files importing BOTH internal/composition and internal/plugin/runtime: %d\n"
              % len(comp_runtime))
    for p in comp_runtime:
        out.write("  %s%s\n" % (p, "  (test)" if p.endswith("_test.go") else "  (NON-TEST)"))
    out.write("non-test files importing both: %d\n"
              % sum(1 for p in comp_runtime if not p.endswith("_test.go")))
    out.close()
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
