#!/usr/bin/env python3
"""Optional, narrowly scoped experiment for repository-root path references."""
import argparse
import json
import re
import sys
from datetime import date
from pathlib import Path

RULE = "repo-path-v2"
TOKEN = re.compile(r"`(?:(ref|planned):)?((?:docs|plan|tools|reference|requirements|product)/[A-Za-z0-9_./-]+)`")
DECLARATION = re.compile(r"`(ref|planned):([^`\n]*)`")


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
    # A declaration must not disappear merely because its name is outside the
    # limited grammar. Ordinary backticks remain outside the failure boundary.
    for match in DECLARATION.finditer(text):
        if TOKEN.fullmatch(match.group(0)):
            continue
        records.append({"line": text.count("\n", 0, match.start()) + 1, "name": match.group(2),
                        "declaration": match.group(1), "severity": "error", "result": "unsupported-declaration"})
    records.sort(key=lambda item: item["line"])
    return {"rule": RULE, "matching": "exact-filesystem-dependent-case", "document": document,
            "denominator": len(records), "records": records}


def apply_allowlist(report, data, today):
    """Apply exact document/name exceptions; report invalid or stale entries."""
    if not isinstance(data, dict) or set(data) != {"baseline_count", "entries"}:
        raise ValueError("allowlist must contain baseline_count and entries")
    entries, cap = data["entries"], data["baseline_count"]
    if not isinstance(entries, list) or type(cap) is not int or cap < 0 or len(entries) > cap:
        raise ValueError("allowlist entries exceed baseline_count or have invalid type")
    seen = set()
    issues = []
    for entry in entries:
        if not isinstance(entry, dict) or set(entry) - {"document", "name", "reason", "issue", "expires"}:
            raise ValueError("invalid allowlist entry fields")
        document, name, reason = (entry.get(k) for k in ("document", "name", "reason"))
        if not all(isinstance(v, str) and v.strip() and v == v.strip() for v in (document, name, reason)):
            raise ValueError("document, name and reason must be nonempty strings")
        if any(c in document + name for c in "*?[]\\") or not TOKEN.fullmatch("`ref:" + name + "`"):
            raise ValueError("allowlist paths must be exact supported names")
        if document.startswith("/") or any(p in ("", ".", "..") for p in document.split("/")):
            raise ValueError("allowlist document must be repository-relative")
        if any(p in (".", "..") for p in name.split("/")):
            raise ValueError("allowlist name must be repository-relative")
        issue, expires = entry.get("issue"), entry.get("expires")
        if issue is not None and (type(issue) is not int or issue <= 0):
            raise ValueError("issue must be a positive integer")
        if issue is None and expires is None:
            raise ValueError("allowlist entry needs issue or expires")
        if expires is not None:
            try:
                if not isinstance(expires, str) or date.fromisoformat(expires).isoformat() != expires:
                    raise ValueError()
                expired = date.fromisoformat(expires) < today
            except ValueError as exc:
                raise ValueError("expires must be an ISO date") from exc
        else:
            expired = False
        key = (document, name)
        if key in seen:
            raise ValueError("duplicate allowlist entry")
        seen.add(key)
        if document != report["document"]:
            continue
        matches = [r for r in report["records"] if r["name"] == name and r["declaration"] == "ref"]
        if expired:
            issues.append({"name": name, "result": "expired-allowlist", "severity": "error"})
        elif not matches or any(r["result"] != "unresolved" or r["severity"] != "error" for r in matches):
            issues.append({"name": name, "result": "stale-allowlist", "severity": "error"})
        else:
            for record in matches:
                if record["severity"] == "error":
                    record["severity"] = "warning"
                    record["result"] = "allowlisted-unresolved"
    report["allowlist"] = {"baseline_count": cap, "entry_count": len(entries), "issues": issues}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("document", help="repository-relative Markdown path")
    parser.add_argument("--root", type=Path, default=Path.cwd())
    parser.add_argument("--allowlist", type=Path, help="optional exact-entry JSON allowlist")
    args = parser.parse_args()
    root = args.root.resolve()
    doc = (root / args.document).resolve()
    if not doc.is_relative_to(root) or not doc.is_file() or doc.suffix != ".md":
        parser.error("document must be an existing Markdown file inside root")
    document = doc.relative_to(root).as_posix()
    report = check(root, document, doc.read_text(encoding="utf-8"))
    if args.allowlist:
        try:
            apply_allowlist(report, json.loads(args.allowlist.read_text(encoding="utf-8")), date.today())
        except (ValueError, OSError) as exc:
            parser.error(f"invalid allowlist: {exc}")
    print(json.dumps(report, ensure_ascii=False, indent=2))
    return int(any(item["severity"] == "error" for item in report["records"] + report.get("allowlist", {}).get("issues", [])))


if __name__ == "__main__":
    sys.exit(main())
