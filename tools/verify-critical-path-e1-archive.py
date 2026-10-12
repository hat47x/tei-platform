#!/usr/bin/env python3
"""Verify retained *synthetic* E1 evidence against its original CI manifest.

This verifies archived bytes and recorded internal consistency, not GitHub's
runtime authority, independent DB effects, or human acceptance.
"""
import hashlib
import json
from pathlib import Path
import re
import stat

ROOT = Path(__file__).resolve().parents[1]
MANIFEST = ROOT / "plan/evidence/TEI_E1_SYNTHETIC_MANIFEST_20261012_38267656696.json"
RECEIPTS = ROOT / "plan/evidence/tei-e1-synthetic-38267656696"
MANIFEST_SHA256 = "60c2babe223fa89383bea24cd718407dbc8e037f8d29aa448858c52997a70584"
INTEGRATION_HEAD = "466b1d10080460390d46b3b342d8bc9c19e09208"
NAMES = frozenset({
    "workspace-r1.json", "target-launch.json", "web-positive.json",
    "target-negative-launch.json", "web-negative-unit-price.json",
    "target-n11-launch.json", "n11-unobserved-control.json",
    "stale-source-control.json",
})
SHA256 = re.compile(r"[0-9a-f]{64}\Z")


def require(condition, message):
    if not condition:
        raise ValueError(message)


def decode_unique(raw, name):
    def no_duplicates(pairs):
        obj = {}
        for key, value in pairs:
            require(key not in obj, f"{name}: duplicate JSON key {key!r}")
            obj[key] = value
        return obj

    return json.loads(raw, object_pairs_hook=no_duplicates)


def verify_archive(manifest_path=MANIFEST, receipts_dir=RECEIPTS):
    """Reject missing, substituted, contradictory or extra witness files.

    A fixed manifest digest prevents silently replacing the reference digest
    together with a changed witness. It cannot authenticate the original CI
    runner or prove that an actual Provider was used.
    """
    manifest_path = Path(manifest_path)
    receipts_dir = Path(receipts_dir)
    manifest_bytes = manifest_path.read_bytes()
    require(hashlib.sha256(manifest_bytes).hexdigest() == MANIFEST_SHA256,
            "original evidence manifest digest mismatch")
    manifest = decode_unique(manifest_bytes, manifest_path.name)
    require(manifest["schema"] == "tei.reference.e1-synthetic-evidence-manifest/v0",
            "wrong evidence schema")
    require(manifest["workflowRunID"] == "38267656696" and
            manifest["workflowRunAttempt"] == "1" and
            manifest["exactCheckoutHead"] == INTEGRATION_HEAD,
            "run, attempt or checkout mismatch")
    require(manifest["evidenceMode"] == "E1_A_B_LIVE_HTTP_SYNTHETIC_PROVIDER_ONLY" and
            manifest["provider"] == "synthetic", "not the recorded synthetic mode")
    require(manifest["productGate"] == "UNQUALIFIED" and
            manifest["runtimeVersionBindingVerified"] is False and
            manifest["staleSourceRejectedByTarget"] is False and
            manifest["semanticEquivalentR2RejectedByTarget"] is False and
            manifest["independentDatabaseRead"] is False and
            manifest["humanAcceptance"] is False,
            "false product qualification or runtime/DB authority claim")
    require(set(manifest["evidenceSHA256"]) == NAMES, "incomplete manifest inventory")
    require({p.name for p in receipts_dir.iterdir()} == NAMES,
            "missing/extra evidence file")
    receipts = {}
    for name in sorted(NAMES):
        path = receipts_dir / name
        require(stat.S_ISREG(path.lstat().st_mode), f"{name}: not a regular file")
        raw = path.read_bytes()
        require(0 < len(raw) <= 32768, f"{name}: oversized/empty")
        digest = manifest["evidenceSHA256"][name]
        require(isinstance(digest, str) and SHA256.fullmatch(digest),
                f"{name}: invalid digest format")
        require(hashlib.sha256(raw).hexdigest() == digest,
                f"{name}: evidence bytes do not match original digest")
        receipt = decode_unique(raw, name)
        require(receipt["integrationHead"] == INTEGRATION_HEAD,
                f"{name}: integration head differs")
        require("synthetic" in receipt["evidenceNature"] or
                "test-only" in receipt["evidenceNature"],
                f"{name}: source was not marked synthetic/test-only")
        receipts[name] = receipt

    src = receipts["workspace-r1.json"]
    pos = receipts["web-positive.json"]
    neg = receipts["web-negative-unit-price.json"]
    n11 = receipts["n11-unobserved-control.json"]
    stale = receipts["stale-source-control.json"]
    launch = receipts["target-launch.json"]
    nlaunch = receipts["target-negative-launch.json"]
    l11 = receipts["target-n11-launch.json"]
    r2 = launch["equivalentSourceControl"]
    r0, r1 = manifest["sourceBefore"], manifest["sourceAfter"]
    require(r0 != r1 and src["sourceBefore"] == r0 and
            src["sourceAfter"] == src["decisionModelRevisionAfter"] == r1,
            "accepted R0 -> R1 provenance mismatch")
    require(src["humanApproval"] is False and
            src["decision"] == "automated-scratch-test-only",
            "test-only approval was changed")
    require(launch["workspaceSourceRevisionBeforeLaunch"] == r1 and
            pos["acceptedSourceRevisionClaim"] == r1 and
            neg["acceptedSourceRevisionClaim"] == r1 and
            n11["acceptedSourceRevisionClaim"] == r1, "Web does not claim R1")
    require(pos["httpStatus"] == 200 and pos["commandObserved"] is True and
            pos["observationNature"] == "synthetic" and
            len(pos["selectedValidationIDs"]) == 4, "positive synthetic Web mismatch")
    require(neg["selectedValidation"] == "sales.order_line.unit_price_positive" and
            neg["commandDispatched"] is False and
            neg["observationExecuted"] is False and
            neg["targetBinarySHA256"] == nlaunch["targetBinarySHA256"] != launch["targetBinarySHA256"],
            "negative execution/binary contrast mismatch")
    require(n11["commandCompleted"] is True and
            n11["observationResultRecorded"] is False and
            n11["httpStatus"] == 422 and
            n11["targetBinarySHA256"] == l11["targetBinarySHA256"],
            "N11 wrongly observed or linked")
    require(stale["acceptedSourceRevisionClaim"] == r1 and
            stale["actuallyLoadedWorkspaceSourceRevision"] == r0 and
            stale["mismatchDetectedByExternalTestHarness"] is True and
            stale["webServerRejectsMismatchedLabels"] is False and
            stale["targetBinarySHA256"] == launch["targetBinarySHA256"],
            "R0 mismatch was not a harness-only negative")
    require(r2["acceptedR1SourceRevision"] == r1 and
            r2["externallyCheckedChangedSourceRevision"] != r1 and
            r2["sourceBeforeSHA256"] != r2["sourceAfterSHA256"] and
            r2["derivedWorldSHA256"] == r2["acceptedWorldSHA256"] and
            r2["manifestSemanticSHA256"] == r2["acceptedManifestSemanticSHA256"] == pos["manifestSemanticSHA256"] and
            r2["mismatchDetectedBySourceRevisionCheck"] is True and
            r2["currentRuntimeVersionBindingEnforced"] is False,
            "R2 source-only difference or lack of runtime binding was obscured")
    require(all(r["independentDatabaseRead"] is False for n, r in receipts.items()
                if "independentDatabaseRead" in r),
            "archive cannot prove an independent DB read")
    return {"receiptCount": len(receipts), "evidenceMode": manifest["evidenceMode"],
            "checkout": INTEGRATION_HEAD, "productGate": "UNQUALIFIED"}


if __name__ == "__main__":
    print(json.dumps(verify_archive(), sort_keys=True))
