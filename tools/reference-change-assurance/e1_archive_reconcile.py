#!/usr/bin/env python3
"""Read-only POC-12: reconcile the already-preserved synthetic E1 archive.

Reuse #1119's immutable archive verifier and POC-11's Workspace receipt
comparison. Do not create an Evidence Authority, retrieve GitHub credentials,
or claim the test-only E1 is a real Provider/DB observation.
"""
import argparse
import hashlib
import importlib.util
import io
import json
from pathlib import Path
import re
import stat
import subprocess
import sys
import zipfile


ROOT = Path(__file__).resolve().parents[2]
ARCHIVE_VERIFIER = ROOT / "tools/verify-critical-path-e1-archive.py"
GITHUB_DIGEST = re.compile(r"sha256:[0-9a-f]{64}\Z")


def canonical_archive():
    spec = importlib.util.spec_from_file_location("tei_existing_e1_archive", ARCHIVE_VERIFIER)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


def verify_supplied_zip(zip_path, expected_digest, archived):
    """Compare a caller-supplied historical GitHub ZIP byte for byte.

    The expected ZIP digest must be taken from the GitHub Actions artifact
    metadata independently by the caller; this function does not fetch it.
    It first reuses the original repository manifest/receipt verifier.
    """
    if not GITHUB_DIGEST.fullmatch(expected_digest or ""):
        raise ValueError("a sha256:<64hex> GitHub artifact digest is required with --artifact-zip")
    path = Path(zip_path)
    if not path.is_file() or path.is_symlink() or path.stat().st_size > 16 * 1024 * 1024:
        raise ValueError("artifact ZIP must be a bounded regular file")
    raw = path.read_bytes()
    actual = "sha256:" + hashlib.sha256(raw).hexdigest()
    if actual != expected_digest:
        raise ValueError("artifact ZIP bytes differ from supplied GitHub metadata digest")
    names = sorted([*archived.NAMES, "evidence-manifest.json"])
    with zipfile.ZipFile(io.BytesIO(raw)) as archive:
        infos = archive.infolist()
        if sorted(info.filename for info in infos) != names:
            raise ValueError("artifact ZIP has missing, duplicated or unexpected entries")
        for info in infos:
            mode = info.external_attr >> 16
            if info.is_dir() or stat.S_ISLNK(mode) or info.file_size > 32768 or info.compress_size > 1024 * 1024:
                raise ValueError("artifact ZIP contains an invalid, oversized or linked entry")
            body = archive.read(info)
            canonical = (archived.MANIFEST.read_bytes() if info.filename == "evidence-manifest.json"
                         else (archived.RECEIPTS / info.filename).read_bytes())
            if body != canonical:
                raise ValueError("artifact ZIP content differs from the archived original: " + info.filename)
    return {"status": "bytes-match", "sha256": actual, "receiptCount": len(archived.NAMES)}


def review_supplied_source_snapshots(before, after, archived, manifest, runner):
    """Use POC-11's existing Go read-only binding, not a second Source Index."""
    command = [
        "go", "run", "./tools/reference-change-assurance",
        "-before-workspace", str(Path(before).resolve()),
        "-after-workspace", str(Path(after).resolve()),
        "-scope", "order-entry-reference",
        "-consumer-kind", "application", "-consumer-id", "order_entry",
        "-archived-witness-dir", str(archived.RECEIPTS),
    ]
    result = runner(command, cwd=ROOT, check=True, capture_output=True, text=True)
    bound = json.loads(result.stdout)
    diff = bound["diff"]
    if (bound["integrationHead"] != manifest["exactCheckoutHead"] or
            diff["beforeSourceRevision"] != manifest["sourceBefore"] or
            diff["afterSourceRevision"] != manifest["sourceAfter"] or
            bound["linkStatus"] != "declared-links-consistent" or
            bound["qualification"] != "unresolved" or
            bound["executionAllowed"] is not False):
        raise ValueError("supplied Source snapshots do not match the archived E1 receipt claims")
    return {
        "status": "supplied-snapshot-revisions-match-archived-claims-only",
        "beforeRevision": diff["beforeSourceRevision"],
        "afterRevision": diff["afterSourceRevision"],
        "witnessClaimStatus": bound["linkStatus"],
        "independentlyAuthenticated": False,
    }


def inspect(artifact_zip=None, artifact_digest=None, before=None, after=None,
            *, runner=subprocess.run, verifier=None):
    if (artifact_zip is None) != (artifact_digest is None):
        raise ValueError("--artifact-zip and --artifact-digest must be supplied together")
    if (before is None) != (after is None):
        raise ValueError("--before-workspace and --after-workspace must be supplied together")
    archived = verifier if verifier is not None else canonical_archive()
    # The existing #1119 verifier checks a fixed manifest digest and the
    # hashes/claims of all eight persisted synthetic receipts.
    result = archived.verify_archive()
    manifest = archived.decode_unique(archived.MANIFEST.read_bytes(), archived.MANIFEST.name)
    output = {
        "schema": "tei.reference.e1-archive-reconciliation/v0",
        "archive": {
            "status": "repository-preserved-synthetic-receipts-verified",
            "receiptCount": result["receiptCount"],
            "workflowRunID": manifest["workflowRunID"],
            "workflowRunAttempt": manifest["workflowRunAttempt"],
            "checkout": manifest["exactCheckoutHead"],
            "manifestSHA256": archived.MANIFEST_SHA256,
        },
        "artifactZip": {"status": "not-supplied"},
        "sourceSnapshots": {"status": "not-supplied"},
        "qualification": "unresolved",
        "executionAllowed": False,
        "productGate": "UNQUALIFIED",
        "authorityLimits": [
            "archived-receipts-are-test-only-synthetic-provider-observations",
            "artifact-zip-digest-depends-on-caller-verified-github-metadata",
            "source-snapshots-are-not-inferred-from-receipts-or-current-main",
            "source-snapshot-matching-does-not-prove-atomicity-or-human-acceptance",
            "no-current-target-version-binding-host-authority-or-independent-db-read",
            "scenario-1180-and-e1-smoke-1163-are-not-one-execution",
        ],
    }
    if artifact_zip is not None:
        output["artifactZip"] = verify_supplied_zip(artifact_zip, artifact_digest, archived)
    if before is not None:
        output["sourceSnapshots"] = review_supplied_source_snapshots(before, after, archived, manifest, runner)
    return output


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--artifact-zip", help="previously downloaded original GitHub E1 artifact ZIP")
    parser.add_argument("--artifact-digest", help="sha256:... from GitHub artifact metadata, not a receipt")
    parser.add_argument("--before-workspace", help="actual archived test-only R0 Workspace root")
    parser.add_argument("--after-workspace", help="actual archived test-only R1 Workspace root")
    args = parser.parse_args()
    try:
        result = inspect(args.artifact_zip, args.artifact_digest,
                         args.before_workspace, args.after_workspace)
    except (ValueError, OSError, KeyError, json.JSONDecodeError,
            subprocess.CalledProcessError, zipfile.BadZipFile) as exc:
        print("archive reconciliation refused: " + str(exc), file=sys.stderr)
        return 1
    print(json.dumps(result, sort_keys=True, indent=2))
    return 0


if __name__ == "__main__":
    sys.exit(main())
