#!/usr/bin/env python3
"""Fail when a public TEI checkout contains non-public resource markers.

The markers are assembled from fragments so this checker does not match its own
source. This is a regression guard, not a replacement for GitHub secret scanning.
"""

from __future__ import annotations

import argparse
import pathlib
import subprocess
import sys

# (kind, value). Values are split so the literal marker never appears here.
MARKERS = (
    ("非公開リポジトリ名", "tei-platform-" + "private" + "-resources"),
    ("アーカイブ内パス", "archives/" + "x-tea-2026-10-14"),
    ("調査記録のパス", "research/" + "low-code-management" + "-tools"),
)
SKIP = {
    "tools/check-public-boundary/check.py",
    "tools/check-public-boundary/check_test.py",
}


def scan_text(text: str) -> list[str]:
    return [kind for kind, value in MARKERS if value in text]


def tracked_files(root: pathlib.Path) -> list[str]:
    output = subprocess.check_output(["git", "-C", str(root), "ls-files", "-z"])
    return [item for item in output.decode("utf-8", "surrogateescape").split("\0") if item]


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--root", default=".", help="repository root (default: .)")
    args = parser.parse_args()

    root = pathlib.Path(args.root).resolve()
    failures = 0
    for relative in tracked_files(root):
        if relative in SKIP:
            continue
        try:
            text = (root / relative).read_text(encoding="utf-8", errors="ignore")
        except OSError:
            continue
        hits = scan_text(text)
        if not hits:
            continue
        failures += 1
        kinds = "、".join(sorted(set(hits)))
        print(f"{relative}: 非公開のマーカー（{kinds}）", file=sys.stderr)

    if failures:
        print(f"public boundary: 非公開のマーカーを含むファイル {failures} 件", file=sys.stderr)
        return 1
    print("public boundary: ok")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
