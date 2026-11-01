#!/usr/bin/env python3
"""Fail when a tracked file names the private materials store.

docs/documentation-quality.md §11.1の規則（非公開資料の名前・所在・集計を
公開文書に書かない）のうち、名前と所在に当たる語を機械で検査する。
禁止語はリポジトリ名・アーカイブ内パス・調査記録のパスの3種である。
パターンは断片を組み立てて作り、この検査器自身のソースへ全文が現れない
ようにする。自分自身のファイルも走査対象から除く。
これは公開境界の回帰を防ぐ補助であり、GitHubのsecret scanningや参照モデルの
利用条件の確認（#941）の代わりではない（#2342、#2436）。
"""

from __future__ import annotations

import argparse
import pathlib
import subprocess
import sys

FORBIDDEN = (
    ("repository-name", "tei-platform-" + "private-resources"),
    ("archive-path", "archives/" + "x-tea-2026-10-14"),
    ("research-note-path", "research/low-code-management-tools/" + "xtea/N-open-questions.md"),
)
SKIP = {
    "tools/check-public-boundary/check.py",
    "tools/check-public-boundary/check_test.py",
}


def scan_text(text: str) -> list[str]:
    return [kind for kind, pattern in FORBIDDEN if pattern in text]


def tracked_files(root: pathlib.Path) -> list[str]:
    output = subprocess.check_output(
        ["git", "-C", str(root), "ls-files", "-z"], stderr=subprocess.STDOUT
    )
    return [item for item in output.decode("utf-8").split("\0") if item]


def scan_repository(root: pathlib.Path) -> list[tuple[str, str]]:
    findings: list[tuple[str, str]] = []
    for relative in tracked_files(root):
        if relative in SKIP:
            continue
        path = root / relative
        if not path.is_file():
            continue
        try:
            text = path.read_text(encoding="utf-8")
        except (UnicodeDecodeError, OSError):
            continue
        for kind in scan_text(text):
            findings.append((relative, kind))
    return findings


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--root", default=".")
    args = parser.parse_args()
    root = pathlib.Path(args.root).resolve()
    findings = scan_repository(root)
    for relative, kind in findings:
        print(f"{relative}: forbidden marker kind={kind}", file=sys.stderr)
    if findings:
        return 1
    print("public boundary: ok")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
