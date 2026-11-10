#!/usr/bin/env python3
"""Read the pinned TEI K1 files through the real SHO main G1b CLI.

This tests actual local file delivery to one product Reader, not OKF v0.2
YAML conformance, a common TEI/SHO plugin, a writer, or a source grant.
"""
from __future__ import annotations

import argparse
import hashlib
import json
import subprocess
from pathlib import Path

SHO_COMMIT = "fc0127400733bac161b3fc0d556f451991d0a510"
FILES = ("index.md", "data/order.md", "functions/submit.md", "business/submit.md")
LINK_COUNTS = {"index.md": 3, "data/order.md": 0, "functions/submit.md": 1,
               "business/submit.md": 1}


def require(condition: bool, message: str) -> None:
    if not condition:
        raise AssertionError(message)


def inspect(sho_root: Path, file: Path):
    proc = subprocess.run(
        ["node", str(sho_root / "bin/sho-inspect.cjs"), "inspect", str(file)],
        cwd=sho_root, text=True, capture_output=True, check=False, timeout=30
    )
    require(proc.returncode == 0, f"SHO CLI failed for {file.name}: {proc.stderr[:250]}")
    require(not proc.stderr.strip(), f"SHO CLI unexpected stderr: {file.name}")
    return json.loads(proc.stdout)


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--bundle", type=Path, required=True)
    parser.add_argument("--expected", type=Path, required=True)
    parser.add_argument("--sho", type=Path, required=True)
    opts = parser.parse_args()
    root = opts.bundle.resolve(strict=True)
    sho = opts.sho.resolve(strict=True)
    expected = json.loads(opts.expected.read_text(encoding="utf-8"))
    require(expected["fixture"] == "okf-k1-v02", "Wrong K1 fixture")

    actual_head = subprocess.check_output(
        ["git", "-C", str(sho), "rev-parse", "HEAD"], text=True
    ).strip()
    require(actual_head == SHO_COMMIT, f"Unexpected SHO checkout {actual_head}")
    require((sho / "bin/sho-inspect.cjs").is_file(), "Real SHO CLI missing")
    require(sorted(expected["required_files"]) == sorted(FILES), "Scope is not pinned")

    results = []
    for rel in FILES:
        original = root / rel
        require(original.is_file() and not original.is_symlink(),
                f"Source file missing or symlinked: {rel}")
        byte_content = original.read_bytes()
        digest = hashlib.sha256(byte_content).hexdigest()
        require(digest == expected["raw_sha256_utf8"][rel],
                f"Current physical fixture bytes changed: {rel}")
        actual = inspect(sho, original.resolve(strict=True))
        source = actual.get("source") or {}
        coverage = actual.get("coverage") or {}
        profile = actual.get("parser") or {}
        require(actual.get("status") == "indexed",
                f"SHO CLI did not index {rel}")
        require(source.get("sha256") == digest and
                source.get("byteLength") == len(byte_content),
                f"SHO CLI actual-read byte identity differs: {rel}")
        require(source.get("revision") == "unknown",
                f"SHO Reader unexpectedly qualified Source revision: {rel}")
        require(profile.get("exactVersion") == "14.3.3",
                f"SHO Parser version mismatch: {rel}")
        require(profile.get("grammarScope") ==
                "markdown-it commonmark link_open tokens only",
                f"SHO grammar scope changed: {rel}")
        require(actual.get("linkOccurrenceCount") == LINK_COUNTS[rel],
                f"Scoped link token count differs: {rel}")
        require(coverage.get("parsedMarkdownLinkTokens") is True and
                coverage.get("allDocumentReferencesEnumerated") is False and
                coverage.get("externalSourcesAcquired") is False and
                coverage.get("sourceWriterInvoked") is False,
                f"Reader scope or side effects changed: {rel}")
        results.append({
            "file": rel, "raw_sha256": digest,
            "byte_length": len(byte_content),
            "markdown_link_open_count": actual["linkOccurrenceCount"],
            "whole_reference_completeness": "UNKNOWN",
            "actual_local_sho_reader": "PASS",
        })

    absent = root / "business" / "not-found-synthetic-k1.md"
    require(not absent.exists(), "Missing-file negative is not missing")
    neg = subprocess.run(
        ["node", str(sho / "bin/sho-inspect.cjs"), "inspect", str(absent)],
        cwd=sho, text=True, capture_output=True, check=False, timeout=30
    )
    require(neg.returncode == 2, "Missing file must not return the zero-link success")
    require(json.loads(neg.stderr).get("error") == "not-found",
            "Missing file was not differentiated from an empty result")

    print(json.dumps({
        "kind": "okf-k1-sho-real-local-file-reader/v1",
        "tei_fixture": expected["fixture"],
        "sho_exact_git_head": actual_head,
        "observed": results,
        "missing_file_negative": "PASS",
        "claim": "SHO main G1b CLI read actual TEI K1 physical Markdown files",
        "not_qualified": [
            "OKF YAML parser or F0 specification conformance",
            "same selected OKF provider invoked by TEI and SHO",
            "real SUI producer to SHO same-file H1",
            "source material fetch, meaning acceptance or disclosure",
            "SHO G2 disk writer or OKF publisher",
        ],
    }, ensure_ascii=False, sort_keys=True, indent=2))


if __name__ == "__main__":
    main()
