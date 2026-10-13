#!/usr/bin/env python3
"""Verify ONE immutable historical E2a-1 real-PostgreSQL test receipt.

This guards recorded bytes and test-only scope. It does NOT rerun PostgreSQL,
verify Host Authority, or grant a TEI G2 / E2 Product Gate.
"""
import hashlib
import json
from pathlib import Path
import sys

REPO = Path(__file__).resolve().parents[1]
RECEIPT = REPO / "plan/evidence/TEI_E2A1_REAL_POSTGRES_RECEIPT_38311557882.json"
SOURCE_SHA256 = "2d766bb442ee009a56a8d457d2f9568d74936f5df78af1cc44a363eb2d378361"
GITHUB_MERGE_SHA = "a2679f0d8d52b0562543464cbd32f12c2ebd72c3"


def require(ok, label):
    if not ok:
        raise ValueError(label)


def decode_unique(raw):
    def unique(pairs):
        data = {}
        for key, val in pairs:
            require(key not in data, f"duplicate JSON key: {key}")
            data[key] = val
        return data
    return json.loads(raw, object_pairs_hook=unique)


def verify(path=RECEIPT):
    path = Path(path)
    require(path.is_file() and not path.is_symlink(), "receipt must be a regular file")
    raw = path.read_bytes()
    require(hashlib.sha256(raw).hexdigest() == SOURCE_SHA256,
            "original E2a-1 receipt byte SHA256 mismatch")
    record = decode_unique(raw)
    require(record["repositoryRevision"] == GITHUB_MERGE_SHA
            and record["evidenceNature"] == "real-postgres-ephemeral-fixture",
            "original exact integration revision/evidence nature differs")
    post = record["postRunRepository"]
    require(record["workingTreeClean"] is True
            and record["reproducibleRevision"] is True
            and post["checked"] is True
            and post["repositoryRevision"] == GITHUB_MERGE_SHA
            and post["workingTreeClean"] is True
            and post["revisionStable"] is True,
            "recorded checkout not clean/exact")
    scope = record["scope"]
    require(scope["productE2"] is False
            and scope["generatedRoundTrip"] is False
            and scope["handWrittenOverlay"] is True
            and scope["handWrittenInsertAndQuerySQL"] is True
            and scope["authority"] == "none"
            and scope["commandToPersistenceLinkage"] == "unverified"
            and scope["independenceKind"] == "connection-pool-separation-in-one-process",
            "test-only scope has been wrongly elevated")
    safe, after = record["fixtureDatabaseSafety"], record["fixtureCleanupSafety"]
    require(safe["guardPassed"] is True
            and safe["userRelationsBeforeFixture"] == 0
            and safe["eventTriggersBeforeFixture"] == 0
            and after["verified"] is True
            and after["userRelationsAfterCleanup"] == 0
            and after["eventTriggersAfterCleanup"] == 0,
            "temporary DB safety/cleanup incomplete")
    pos = record["positive"]
    require(pos["databaseProvider"] == "postgres"
            and pos["observationStatus"] == "observed"
            and pos["rowMatchesSubmission"] is True
            and pos["rowsRead"] == 1
            and pos["backendPidsDiffer"] is True
            and pos["backendPidCommandPool"] != pos["backendPidReadPoolBefore"]
            and pos["backendPidReadPoolBefore"] == pos["backendPidReadPoolAfter"]
            and pos["commandId"] == pos["observedCommandId"]
            and pos["readEvidenceNature"] == "real-postgres-ephemeral-fixture",
            "recorded positive real-DB/pool evidence differs")
    for name in ("n08MissingBinding", "s02AnalogueRelationAbsent", "s02AnalogueColumnMissing"):
        control = record[name]
        require(control["commandRefusedByGate"] is True and
                control["commandDispatchCount"] == 0,
                f"{name}: pre-dispatch rejection not recorded")
    for name in ("n11ClosedReadPool", "n11DroppedTable"):
        control = record[name]
        require(control["commandCommitted"] is True
                and control["observationStatus"] == "read-failed"
                and control["observationErrorSeen"] is True
                and control["neverReportedAsObserved"] is True,
                f"{name}: unobserved control not recorded")
    return {
        "receiptSHA256": SOURCE_SHA256,
        "exactVirtualMergeHead": GITHUB_MERGE_SHA,
        "evidenceNature": "real-postgres-ephemeral-fixture",
        "scope": "hand-written-one-process-two-pools-only",
        "productGate": "UNQUALIFIED",
    }


if __name__ == "__main__":
    path = Path(sys.argv[1]) if len(sys.argv) > 1 else RECEIPT
    if len(sys.argv) > 2:
        raise SystemExit("usage: verify-e2a1-postgres-archive.py [receipt.json]")
    print(json.dumps(verify(path), sort_keys=True))
