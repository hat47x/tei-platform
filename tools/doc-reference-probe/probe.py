#!/usr/bin/env python3
"""Optional, narrowly scoped experiment for repository-root path references."""
import argparse
import json
import re
import sys
from pathlib import Path

RULE = "repo-path-v1"
TOKEN = re.compile(r"`(?:(ref|planned):)?((?:docs|plan|tools|reference|requirements|product)/[A-Za-z0-9_./-]+)`")


def check(root, document, text):
    root = root.resolve()
    planned_doc = document.startswith("plan/")
    records = []
    for match in TOKEN.finditer(text):
        kind, name = match.groups()
        valid = not any(part in ("..", ".") for part in name.split("/"))
        target = (root / name).resolve()
        valid = valid and target.is_relative_to(root)
        exists = valid and (target.is_dir() if name.endswith("/") else target.exists())
        if not valid:
            severity, result = ("error" if kind else "warning"), "invalid-path"
        elif kind == "planned" and not planned_doc:
            severity, result = "error", "planned-outside-plan"
        elif kind == "planned" and exists:
            severity, result = "warning", "planned-now-exists"
        elif kind == "planned":
            severity, result = "info", "planned"
        elif exists:
            severity, result = "info", "resolved"
        elif kind == "ref" and not planned_doc:
            severity, result = "error", "unresolved"
        else:
            severity, result = "warning", "unresolved"
        records.append({"line": text.count("\n", 0, match.start()) + 1, "name": name,
                        "declaration": kind or "implicit", "severity": severity, "result": result})
    return {"rule": RULE, "matching": "exact-filesystem-dependent-case", "document": document,
            "denominator": len(records), "records": records}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("document", help="repository-relative Markdown path")
    parser.add_argument("--root", type=Path, default=Path.cwd())
    args = parser.parse_args()
    root = args.root.resolve()
    doc = (root / args.document).resolve()
    if not doc.is_relative_to(root) or not doc.is_file() or doc.suffix != ".md":
        parser.error("document must be an existing Markdown file inside root")
    document = doc.relative_to(root).as_posix()
    report = check(root, document, doc.read_text(encoding="utf-8"))
    print(json.dumps(report, ensure_ascii=False, indent=2))
    return int(any(item["severity"] == "error" for item in report["records"]))


if __name__ == "__main__":
    sys.exit(main())
