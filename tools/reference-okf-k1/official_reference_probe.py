#!/usr/bin/env python3
"""Bounded K1 witness using the pinned upstream OKF reference document parser.

This is test-only. It is NOT a TEI/SHO production reader, an OKF-wide
conformance certification, an attester, or evidence of source/disclosure grants.
All mutations are in memory. No source file is edited or written.
"""
from __future__ import annotations

import argparse
import hashlib
import importlib.util
import json
import subprocess
import sys
from pathlib import Path

UPSTREAM_COMMIT = "ad30107c31c06aec8a7d5636e0d1058118604e6f"
PY_YAML_VERSION = "6.0.3"
FILE_SET = ("index.md", "data/order.md", "functions/submit.md", "business/submit.md")


def git_blob_sha(raw: bytes) -> str:
    return hashlib.sha1(b"blob " + str(len(raw)).encode("ascii") + b"\x00" + raw).hexdigest()


def require(condition: bool, message: str) -> None:
    if not condition:
        raise AssertionError(message)


def reference_reader(root: Path, expected: dict):
    import yaml

    require(yaml.__version__ == PY_YAML_VERSION,
            f"PyYAML version changed: {yaml.__version__}")
    ref = root / "src/reference_agent/bundle/document.py"
    spec = root / "SPEC.md"
    require(ref.is_file() and spec.is_file(), "Pinned upstream files are missing")
    head = subprocess.check_output(
        ["git", "-C", str(root), "rev-parse", "HEAD"], text=True
    ).strip()
    require(head == UPSTREAM_COMMIT, f"unexpected upstream commit {head}")
    doc_raw = ref.read_bytes()
    require(git_blob_sha(doc_raw) == expected["okf"]["reference_document_git_blob"],
            "Upstream parser blob changed")
    require(hashlib.sha256(spec.read_bytes()).hexdigest() ==
            expected["okf"]["spec_sha256_utf8"], "Upstream SPEC raw bytes changed")
    # Direct module loading avoids importing the reference agent and its ADK,
    # network, BigQuery or other unrelated producer dependencies.
    spec_obj = importlib.util.spec_from_file_location("okf_pinned_document", ref)
    require(spec_obj is not None and spec_obj.loader is not None,
            "Cannot import pinned reference document parser")
    mod = importlib.util.module_from_spec(spec_obj)
    sys.modules[spec_obj.name] = mod
    spec_obj.loader.exec_module(mod)
    return mod, {"upstream_commit": head, "reference_document_blob": git_blob_sha(doc_raw),
                 "spec_sha256": hashlib.sha256(spec.read_bytes()).hexdigest(),
                 "pyyaml": yaml.__version__}


def validate_reference(mod, documents: dict[str, str]) -> tuple[bool, str]:
    try:
        for name, value in documents.items():
            if name.endswith("/index.md") or name == "index.md" or name.endswith("/log.md"):
                continue
            mod.OKFDocument.parse(value).validate()
        return True, "OK"
    except (mod.OKFDocumentError, ValueError) as exc:
        return False, type(exc).__name__


def validate_required_string_type(mod, documents: dict[str, str]) -> bool:
    # Specification-derived check separate from the permissive upstream
    # sample: type must be a short string, not just any truthy YAML value.
    try:
        for name, value in documents.items():
            if name == "index.md" or name.endswith("/index.md") or name.endswith("/log.md"):
                continue
            ty = mod.OKFDocument.parse(value).frontmatter.get("type")
            if not isinstance(ty, str) or not ty.strip():
                return False
        return True
    except (mod.OKFDocumentError, ValueError):
        return False


def mutate(name: str, base: dict[str, str]) -> dict[str, str]:
    out = dict(base)
    if name == "delete type from data/order.md":
        out["data/order.md"] = out["data/order.md"].replace("type: Data Entity\n", "", 1)
    elif name == "make business/submit.md YAML unparseable":
        out["business/submit.md"] = out["business/submit.md"].replace(
            "type: Unclassified Draft Item", "type: [unclosed", 1)
    elif name == "add extra x-unknown frontmatter key to data/order.md":
        out["data/order.md"] = out["data/order.md"].replace(
            "type: Data Entity\n", "type: Data Entity\nx-future-field: allowed\n", 1)
    elif name == "delete index.md":
        del out["index.md"]
    elif name == "change business/submit.md type to Future Experimental Role":
        out["business/submit.md"] = out["business/submit.md"].replace(
            "type: Unclassified Draft Item", "type: Future Experimental Role", 1)
    elif name == "append broken link to temporary copy of business/submit.md":
        out["business/submit.md"] += "\n[unresolved](/business/missing.md)\n"
    elif name == "append OKF bundle-root absolute link /data/order.md to temporary copy of functions/submit.md":
        out["functions/submit.md"] += "\n[bundle-root](/data/order.md)\n"
    elif name == "change data/order.md type to a nonempty YAML list":
        out["data/order.md"] = out["data/order.md"].replace(
            "type: Data Entity", "type: [Data Entity]", 1)
    else:
        raise AssertionError(f"Unregistered mutation: {name}")
    require(out != base, f"Mutation did not change its input: {name}")
    return out


def run(bundle_root: Path, expected_path: Path, upstream_root: Path) -> dict:
    expected = json.loads(expected_path.read_text(encoding="utf-8"))
    require(expected["fixture"] == "okf-k1-v02", "Unexpected fixture identity")
    require(expected["okf"]["spec_commit"] == UPSTREAM_COMMIT, "SPEC pin drift")
    require(expected["expected_runtime_status"] == "NOT RUN", "Runtime claim drift")
    require(expected["validation_status"]["owner_review"] == "NOT RUN",
            "Owner approval cannot be inferred from raw source tests")

    ref, pinned = reference_reader(upstream_root, expected)
    require(sorted(expected["required_files"]) == sorted(FILE_SET), "Required files drift")
    observed = sorted(
        p.relative_to(bundle_root).as_posix()
        for p in bundle_root.rglob("*.md")
    )
    require(observed == sorted(FILE_SET),
            f"Bundle inventory differs from independent required set: {observed}")
    documents = {}
    digests = {}
    for rel in FILE_SET:
        file = bundle_root / rel
        require(file.is_file() and not file.is_symlink(), f"Invalid file: {rel}")
        raw = file.read_bytes()
        require(raw.endswith(b"\n") and b"\r" not in raw, f"Raw EOL drift: {rel}")
        text = raw.decode("utf-8", errors="strict")
        digest = hashlib.sha256(raw).hexdigest()
        require(digest == expected["raw_sha256_utf8"][rel],
                f"Fixture byte mismatch: {rel}")
        documents[rel] = text
        digests[rel] = digest

    index = ref.OKFDocument.parse(documents["index.md"])
    require(str(index.frontmatter.get("okf_version")) == "0.2",
            "Bundle root OKF version pin drift")
    actual_concepts = {key: val for key, val in documents.items() if key != "index.md"}
    baseline, error = validate_reference(ref, actual_concepts)
    require(baseline and validate_required_string_type(ref, actual_concepts),
            f"Pinned upstream reader baseline rejected fixture: {error}")
    function_doc = ref.OKFDocument.parse(documents["functions/submit.md"])
    verified = ref.normalize_verified(function_doc.frontmatter)
    require(len(verified) == 1, "Singleton verified must normalize to 1 event")
    require(ref.trust_tier(function_doc.frontmatter) == "machine-confirmed",
            "Synthetic process verified must not be human approval")
    data_doc = ref.OKFDocument.parse(documents["data/order.md"])
    require(data_doc.frontmatter["sources"][0]["resource"] ==
            "all generated order rows in the K1 synthetic sample",
            "Scope descriptor silently changed")

    seen_mutants = []
    for mutant in expected["negative_mutants"]:
        name = mutant["name"]
        changed = mutate(name, documents)
        # The mutation is only in memory, not a second source repository.
        ref_actual, _ = validate_reference(ref, changed)
        spec_subset_actual = validate_required_string_type(ref, changed)
        want = mutant["expected_format_conformant"]
        require(spec_subset_actual is want, f"SPEC subset mutant mismatch: {name}")
        if name == "change data/order.md type to a nonempty YAML list":
            # Known upstream sample limitation: .validate() only tests
            # truthiness, while the OKF SPEC requires a string type.
            require(ref_actual is True and want is False,
                    "Expected reference implementation gap has changed")
            outcome = "KNOWN_REFERENCE_GAP"
        else:
            require(ref_actual is want, f"Reference reader mutant mismatch: {name}")
            outcome = "MATCH"
        seen_mutants.append({"name": name, "reference_accepts": ref_actual,
                             "required_string_type_accepts": spec_subset_actual,
                             "expected_spec_subset": want, "result": outcome})
    require(len(seen_mutants) == 8 and len(set(m["name"] for m in seen_mutants)) == 8,
            "Expected eight distinct one-factor mutations")

    return {
        "kind": "okf-k1-test-only-reference-witness/v1",
        "upstream": pinned, "fixture": expected["fixture"],
        "observed_file_sha256": digests,
        "baseline_reference_parse_validate": "PASS",
        "verified_events_synthetic_process": len(verified),
        "mutations": seen_mutants,
        "scope": "pinned upstream Python reference reader and small SPEC-derived type check",
        "not_qualified": [
            "full OKF v0.2 conformance or normative parser completeness",
            "TEI or SHO product reader and plugin invocation",
            "real external source, permission, publisher or owner meaning acceptance",
            "K2 real two-consumer delivery, K4 computation, K5 disk writer",
        ],
    }


def main() -> None:
    p = argparse.ArgumentParser()
    p.add_argument("--bundle", type=Path, required=True)
    p.add_argument("--expected", type=Path, required=True)
    p.add_argument("--upstream", type=Path, required=True)
    args = p.parse_args()
    report = run(args.bundle.resolve(strict=True), args.expected.resolve(strict=True),
                 args.upstream.resolve(strict=True))
    print(json.dumps(report, sort_keys=True, ensure_ascii=False, indent=2))


if __name__ == "__main__":
    main()
