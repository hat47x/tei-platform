#!/usr/bin/env python3
"""W0c RC-01: one actual TEI CLI reference-Source relocation counterexample.

The *only* modified files are copied under TemporaryDirectory. Neither TEI
source, accepted meaning, the checkout, nor the consumer policy is edited.
"""
import argparse
import hashlib
import json
import shutil
import sys
import tempfile
from pathlib import Path

from consumer_views import export_views, make_witness
from verify_real_cli import positive, require, run_cli, same_projection

SUBJECT = {"kind": "semantic-relation", "id": "sales.order.customer"}
OLD = "semantic/sales/order.tei"
NEW = "semantic/sales/order-relocated.tei"


def observe(binary, root, revision):
    version = run_cli(binary, "version", "-json")
    require(version.get("source") == "build-info"
            and version.get("vcsRevision") == revision
            and version.get("vcsModified", False) is False,
            "actual Go executable identity is not the checked HEAD")

    catalog = positive(run_cli(binary, "model", "catalog", "-workspace", root),
                       "tei.cli.model/v0")
    entry = [item for item in catalog.get("entries", [])
             if item.get("subject") == SUBJECT]
    require(len(entry) == 1 and entry[0].get("sourceStatus") == "indexed"
            and not entry[0].get("ambiguous"),
            "the expected Canonical Subject is not uniquely indexed")
    inspected = positive(run_cli(binary, "inspect", "-workspace", root,
                                 "-subject-kind", SUBJECT["kind"],
                                 "-subject-id", SUBJECT["id"]),
                         "tei.cli.inspect/v0")
    same_projection(catalog, inspected, "inspect/catalog")
    require(inspected["subject"] == SUBJECT, "inspect selected a different subject")
    refs = inspected["sources"]["references"]
    require(refs and all(isinstance(x.get("artifactPath"), str) for x in refs),
            "source references are missing")

    trace = positive(run_cli(binary, "trace", "-workspace", root,
                             "-scope", "reference-observation",
                             "-subject-kind", SUBJECT["kind"],
                             "-subject-id", SUBJECT["id"],
                             "-mode", "summary"),
                     "tei.cli.trace/v0")
    same_projection(catalog, trace, "trace/catalog")
    require(isinstance(trace.get("snapshotFingerprint"), str)
            and trace["snapshotFingerprint"], "missing continuation snapshot")
    detail = trace["trace"]
    require(detail.get("total", 0) >= 2 and detail.get("subject", {}).get("subject") == SUBJECT,
            "reference relation is not the requested trace subject")

    validation = run_cli(binary, "validate", "-workspace", root, "-json")
    require(validation.get("schemaVersion") == "tei.cli.validate/v0"
            and validation.get("ok") is True, "reference validation failed")
    same_projection(catalog, validation, "validate/catalog")
    return {
        "catalog": catalog, "inspect": inspected, "trace": trace,
        "diagnostics": validation["diagnostics"],
        "source_paths": sorted({x["artifactPath"] for x in refs}),
        "witness": make_witness(
            subject=SUBJECT, binary_revision=version["vcsRevision"],
            catalog=catalog, inspect=inspected, trace=detail,
            rows=detail["total"], diagnostics=validation["diagnostics"],
        ),
    }


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--binary", type=Path, required=True)
    parser.add_argument("--workspace", type=Path, required=True)
    parser.add_argument("--expected-head", required=True)
    args = parser.parse_args()
    binary = args.binary.resolve(strict=True)
    source_workspace = args.workspace.resolve(strict=True)
    require(len(args.expected_head) == 40
            and all(ch in "0123456789abcdef" for ch in args.expected_head),
            "expected HEAD must be a full lowercase SHA")
    require((source_workspace / OLD).is_file(), "public reference Source is missing")
    immutable_original = hashlib.sha256((source_workspace / OLD).read_bytes()).hexdigest()

    with tempfile.TemporaryDirectory(prefix="tei-rc01-") as temp:
        tmp = Path(temp)
        copied = tmp / "workspace"
        shutil.copytree(source_workspace, copied, symlinks=False)
        baseline = observe(binary, str(copied), args.expected_head)
        require(OLD in baseline["source_paths"],
                "baseline inspect does not point to the expected Source")
        baseline_view = tmp / "baseline-view"
        baseline_view.mkdir()
        export_views(baseline["witness"], baseline_view)

        moved = copied / OLD
        renamed = copied / NEW
        original_bytes = moved.read_bytes()
        moved.rename(renamed)
        require(not moved.exists() and renamed.read_bytes() == original_bytes,
                "the relocation changed the Source's bytes")

        after = observe(binary, str(copied), args.expected_head)
        require(NEW in after["source_paths"] and OLD not in after["source_paths"],
                "inspect did not resolve the relocated Source occurrence")
        changed_view = tmp / "changed-view"
        changed_view.mkdir()
        export_views(after["witness"], changed_view)

        require(baseline["inspect"]["canonical"] == after["inspect"]["canonical"],
                "a path-only edit unexpectedly changed the canonical relation")
        require(baseline["catalog"]["worldFingerprint"]
                == after["catalog"]["worldFingerprint"],
                "path-only relocation changed the normalized World")
        require(baseline["catalog"]["sourceIndexFingerprint"]
                != after["catalog"]["sourceIndexFingerprint"],
                "Source Index did not detect the relocated occurrence")
        require(baseline["trace"]["snapshotFingerprint"]
                != after["trace"]["snapshotFingerprint"],
                "trace continuation token did not respond to Source move")
        require(baseline["witness"]["sourceArtifacts"]
                != after["witness"]["sourceArtifacts"],
                "the consumer-visible source citation was not invalidated")
        require(baseline["witness"]["subjectId"] == after["witness"]["subjectId"]
                and baseline["witness"]["binaryRevision"]
                == after["witness"]["binaryRevision"], "subject or binary changed")
        require((source_workspace / OLD).is_file()
                and hashlib.sha256((source_workspace / OLD).read_bytes()).hexdigest()
                == immutable_original,
                "original repository Reference was changed")

        print(json.dumps({
            "status": "reference-source-relocation-observed",
            "binaryMatchesHead": True,
            "subjectIdentityUnchanged": True,
            "canonicalValueUnchanged": True,
            "worldFingerprintUnchanged": True,
            "sourceIndexFingerprintChanged": True,
            "traceContinuationTokenChanged": True,
            "sourceCitationChanged": True,
            "twoViewsReadBack": True,
            "originalReferenceUnchanged": True,
            "proofScope": "temporary copy of public reference/workspace only",
            "businessMeaningAndCurrentDisclosure": "not-established",
        }, sort_keys=True))


if __name__ == "__main__":
    try:
        main()
    except (AssertionError, ValueError, OSError, KeyError) as exc:
        print("W0c RC-01 failed: " + type(exc).__name__ + ": " + str(exc),
              file=sys.stderr)
        sys.exit(1)
