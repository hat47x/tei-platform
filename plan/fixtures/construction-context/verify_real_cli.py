#!/usr/bin/env python3
"""Read-only, actual-process TEI CLI handoff reference for #2516 W0a / #2605 S1.

The checked TEI binary is built outside the repository tree by the caller.
This test checks the observed CLI transport and provenance relationships.
It does not approve business meaning, physical DB reads, or disclosure.
"""
import argparse
import json
import subprocess
import sys
from pathlib import Path

from consumer_views import export_views, make_witness


def require(condition, message):
    if not condition:
        raise AssertionError(message)


def run_cli(binary, *args, expected_exit=0):
    completed = subprocess.run(
        [str(binary), *args], stdout=subprocess.PIPE, stderr=subprocess.PIPE,
        check=False, timeout=120, text=True, encoding="utf-8",
    )
    require(completed.returncode == expected_exit,
            f"{args[0]}: exit={completed.returncode}, expected={expected_exit}")
    require(not completed.stderr, f"{args[0]} emitted unexpected stderr")
    try:
        result = json.loads(completed.stdout)
    except (ValueError, TypeError) as exc:
        raise AssertionError(f"{args[0]} did not return JSON") from exc
    require(isinstance(result, dict), f"{args[0]} returned non-object JSON")
    return result


def positive(document, schema):
    require(document.get("schemaVersion") == schema, f"invalid envelope: {schema}")
    require(document.get("ok") is True and isinstance(document.get("result"), dict),
            f"{schema}: missing successful result")
    require(document.get("error") is None, f"{schema}: unexpected error field")
    return document["result"]


def same_projection(expected, observed, label):
    for field in ("worldFingerprint", "sourceIndexFingerprint"):
        value = expected.get(field)
        require(isinstance(value, str) and value, f"{label}: missing {field} in baseline")
        require(observed.get(field) == value, f"{label}: {field} changed")
    # Matching observed hashes proves correspondence of these projections
    # in this fixture, not that all inputs were atomically read.


def execute(binary, workspace, expected_head, output_dir=None):
    version = run_cli(binary, "version", "-json")
    require(version.get("schemaVersion") == "tei.cli.version/v0", "unexpected CLI version schema")
    require(version.get("source") == "build-info", "missing actual executable build identity")
    require(version.get("vcsRevision") == expected_head, "executed binary differs from checked HEAD")
    # cli_version.go uses omitempty: clean binaries omit vcsModified.
    require(version.get("vcsModified", False) is False,
            "binary was built with modified sources")

    discovery = run_cli(binary, "commands")
    require(discovery.get("schemaVersion") == "tei.cli.discovery/v0", "discovery schema changed")
    require(discovery.get("scope") == "selected-read-only-commands", "unexpected discovery scope")
    entries = discovery.get("commands", [])
    require(isinstance(entries, list), "discovery commands missing")
    named = {e.get("name"): e for e in entries if isinstance(e, dict)}
    for name in ("version", "model catalog", "inspect", "trace", "validate"):
        require(named.get(name, {}).get("effect") == "read-only",
                f"{name} was not declared read-only")

    root = str(workspace)
    catalog = positive(run_cli(binary, "model", "catalog", "-workspace", root),
                       "tei.cli.model/v0")
    require(catalog.get("schemaVersion") == "tei.operations.model-catalog/v0",
            "catalog result schema changed")
    require(catalog.get("effect") == "read-only", "catalog effect mismatch")
    require(catalog.get("modelRevision"), "catalog model revision missing")
    candidates = [
        e for e in catalog.get("entries", [])
        if isinstance(e, dict) and e.get("subject") ==
        {"kind": "semantic-relation", "id": "sales.order.customer"}
    ]
    require(len(candidates) == 1 and candidates[0].get("sourceStatus") == "indexed"
            and not candidates[0].get("ambiguous"), "subject not uniquely indexed in catalog")

    subject_args = ("-subject-kind", "semantic-relation",
                    "-subject-id", "sales.order.customer")
    inspected = positive(run_cli(binary, "inspect", "-workspace", root, *subject_args),
                         "tei.cli.inspect/v0")
    same_projection(catalog, inspected, "inspect/catalog")
    require(inspected.get("subject") == candidates[0]["subject"],
            "inspect has different semantic subject")
    require(inspected.get("sourceStatus") == "indexed", "inspect did not locate source")
    references = inspected.get("sources", {}).get("references")
    require(isinstance(references, list) and references, "inspect source evidence absent")

    trace_args = ("trace", "-workspace", root, "-scope", "reference-observation",
                  *subject_args, "-direction", "both")
    summary = positive(run_cli(binary, *trace_args, "-mode", "summary"),
                       "tei.cli.trace/v0")
    same_projection(catalog, summary, "trace/catalog")
    snapshot = summary.get("snapshotFingerprint")
    require(isinstance(snapshot, str) and snapshot, "trace continuation token missing")
    trace = summary.get("trace", {})
    require(trace.get("subject", {}).get("subject") == candidates[0]["subject"],
            "trace has different semantic subject")
    require(isinstance(trace.get("total"), int) and trace["total"] >= 2,
            "expected declared relation edges are missing")
    require(not trace.get("rows"), "summary unexpectedly returned full rows")
    source_refs = summary.get("source", {}).get("references", [])
    require(source_refs, "trace source references absent")
    require(set(r.get("artifactPath") for r in references).intersection(
            r.get("artifactPath") for r in source_refs),
            "trace and inspect source locations do not overlap")

    offset = 0
    observed_rows = 0
    seen_offsets = set()
    while True:
        require(offset not in seen_offsets and observed_rows <= trace["total"],
                "pagination loop or surplus rows")
        seen_offsets.add(offset)
        page = positive(run_cli(binary, *trace_args,
                                "-mode", "full", "-limit", "1",
                                "-offset", str(offset),
                                "-expected-fingerprint", snapshot),
                        "tei.cli.trace/v0")
        same_projection(catalog, page, "trace page/catalog")
        require(page.get("snapshotFingerprint") == snapshot,
                "trace page continuation token changed")
        detail = page.get("trace", {})
        require(detail.get("total") == trace["total"], "trace total changed between pages")
        require(detail.get("subject", {}).get("subject") == candidates[0]["subject"],
                "trace page subject changed")
        rows = detail.get("rows")
        require(isinstance(rows, list) and len(rows) == 1, "unexpected trace page size")
        observed_rows += len(rows)
        next_offset = detail.get("nextOffset")
        if next_offset is None:
            break
        require(isinstance(next_offset, int) and next_offset > offset,
                "trace continuation offset is invalid")
        offset = next_offset
    require(observed_rows == trace["total"], "page stitching lost or duplicated rows")

    stale = run_cli(binary, *trace_args, "-mode", "full", "-limit", "1",
                    "-offset", "1", "-expected-fingerprint", "sha256:stale",
                    expected_exit=1)
    require(stale.get("schemaVersion") == "tei.cli.trace/v0" and
            stale.get("ok") is False and
            stale.get("error", {}).get("code") == "STALE_SNAPSHOT",
            "stale trace was not rejected")

    validation = run_cli(binary, "validate", "-workspace", root, "-json")
    require(validation.get("schemaVersion") == "tei.cli.validate/v0",
            "validation schema changed")
    require(validation.get("ok") is True, "workspace validation failed")
    same_projection(catalog, validation, "validate/catalog")
    diagnostics = validation.get("diagnostics", [])
    require(isinstance(diagnostics, list), "validation diagnostics missing")

    # A single in-memory actual CLI observation is used for both view materials.
    witness = make_witness(
        subject=candidates[0]["subject"], binary_revision=version["vcsRevision"],
        catalog=catalog, inspect=inspected, trace=trace,
        rows=observed_rows, diagnostics=diagnostics,
    )
    if output_dir is not None:
        export_views(witness, output_dir)

    # Never log raw source, trace JSON, or private paths. This is a public
    # synthetic/reference Workspace only; an actual external disclosure
    # decision still belongs to the owner and OS/Host.
    print(json.dumps({
        "result": "synthetic-reference-cli-handoff-qualified",
        "observedProcessCalls": 7 + len(seen_offsets),
        "catalogEntries": len(catalog.get("entries", [])),
        "traceRows": observed_rows,
        "validationDiagnostics": len(diagnostics),
        "binaryRevisionMatchesHead": True,
        "humanAndAIViewParity": "verified" if output_dir is not None else "not-requested",
        "worldAndSourceIndexFingerprintsAligned": True,
        "scope": "reference/workspace only; no business/authorization acceptance",
    }, sort_keys=True))


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--binary", type=Path, required=True)
    parser.add_argument("--workspace", type=Path, required=True)
    parser.add_argument("--expected-head", required=True)
    parser.add_argument("--output-dir", type=Path)
    args = parser.parse_args()
    require(args.binary.is_file(), "actual TEI binary path missing")
    require((args.workspace / "tei.workspace.yaml").is_file(),
            "TEI workspace root missing")
    require(len(args.expected_head) == 40 and
            all(c in "0123456789abcdef" for c in args.expected_head),
            "expected HEAD is not a full SHA")
    execute(args.binary.resolve(), args.workspace.resolve(), args.expected_head,
            args.output_dir.resolve() if args.output_dir is not None else None)


if __name__ == "__main__":
    try:
        main()
    except (AssertionError, subprocess.TimeoutExpired, OSError) as exc:
        print(f"TEI CLI handoff check failed: {type(exc).__name__}: {exc}", file=sys.stderr)
        sys.exit(1)