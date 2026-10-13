#!/usr/bin/env python3
"""Fail-closed *historical evidence join* for the two archived #1119 receipts.

Do not treat a successful archive integrity check as execution/authority proof.
The E1 synthetic Web and E2a-1 real Postgres receipts are NOT one workflow.
"""
import argparse
import importlib.util
import json
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
TOOLS = ROOT / "tools"


def load_verifier(filename, module_name):
    spec = importlib.util.spec_from_file_location(module_name, TOOLS / filename)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


E1 = load_verifier("verify-critical-path-e1-archive.py", "tei_e1_archive_join")
E2 = load_verifier("verify-e2a1-postgres-archive.py", "tei_e2_archive_join")

# These are *historical evidence* obligations. A different E2 artifact will
# require a new verifier/review, not a changed acceptance threshold here.
OBLIGATIONS = (
    "SAME_ACCEPTED_SOURCE_REVISION",
    "SAME_APPLICATION_AND_BUSINESS_SUBJECT",
    "REAL_WEB_COMMAND_LINKED_TO_POSTGRES_WRITE",
    "RUNTIME_REJECTS_R0_AND_R2_BEFORE_DISPATCH",
    "CURRENT_HOST_WRITE_AUTHORITY",
    "INDEPENDENT_READ_INVOCATION",
    "HUMAN_ACCEPTANCE_AUTHORITY",
    "DATA_FIRST_GENERATED_EXECUTION",
)


def audit_pair(e1_manifest, e1_launch, e2_receipt):
    """Report missing links, never silently synthesize a joined success."""
    claims = {
        "SAME_ACCEPTED_SOURCE_REVISION": (
            isinstance(e2_receipt.get("acceptedSourceRevision"), str)
            and e2_receipt["acceptedSourceRevision"] == e1_manifest["sourceAfter"]
            and e2_receipt.get("acceptedSourceVerification") == "runtime-verified"
        ),
        "SAME_APPLICATION_AND_BUSINESS_SUBJECT": (
            e1_launch.get("application") == "order_entry"
            and e2_receipt.get("application") == "order_entry"
            and e2_receipt.get("semanticData") == "sales.order"
        ),
        "REAL_WEB_COMMAND_LINKED_TO_POSTGRES_WRITE": (
            e1_manifest["provider"] != "synthetic"
            and e2_receipt.get("webCommandLinkage") == "verified"
            and e2_receipt["positive"].get("commandId") == e1_launch.get("commandId")
            and e2_receipt["scope"].get("commandToPersistenceLinkage") == "verified"
        ),
        "RUNTIME_REJECTS_R0_AND_R2_BEFORE_DISPATCH": (
            e1_manifest["runtimeVersionBindingVerified"] is True
            and e1_manifest["staleSourceRejectedByTarget"] is True
            and e1_manifest["semanticEquivalentR2RejectedByTarget"] is True
        ),
        "CURRENT_HOST_WRITE_AUTHORITY": (
            e2_receipt["scope"].get("authority") == "runtime-host-verified"
            and e2_receipt.get("authorizationAtCommit") is True
        ),
        "INDEPENDENT_READ_INVOCATION": (
            e2_receipt["scope"].get("independenceKind") == "separate-invocation"
            and e2_receipt["positive"]["observationStatus"] == "observed"
            and e2_receipt["positive"]["rowMatchesSubmission"] is True
        ),
        "HUMAN_ACCEPTANCE_AUTHORITY": (
            e1_manifest["humanAcceptance"] is True
            and e2_receipt.get("humanAcceptedSourceAuthority") is True
        ),
        "DATA_FIRST_GENERATED_EXECUTION": (
            e2_receipt["scope"].get("generatedRoundTrip") is True
            and e2_receipt["scope"].get("handWrittenOverlay") is False
            and e2_receipt["scope"].get("handWrittenInsertAndQuerySQL") is False
        ),
    }
    blocked = [code for code in OBLIGATIONS if claims[code] is not True]
    # A theoretical future all-true tuple is NOT enough: a separate trusted
    # product gate must qualify source, authority, runtime and DB evidence.
    return {
        "schema": "tei.reference.critical-path-historical-evidence-join/v0",
        "historicalEvidenceOnly": True,
        "e1": {
            "checkout": e1_manifest["exactCheckoutHead"],
            "runID": e1_manifest["workflowRunID"],
            "acceptedSourceRevisionClaim": e1_manifest["sourceAfter"],
            "application": e1_launch["application"],
            "evidenceMode": e1_manifest["evidenceMode"],
            "provider": e1_manifest["provider"],
        },
        "e2a1": {
            "checkout": e2_receipt["repositoryRevision"],
            "applicationFromBinding": e2_receipt["positive"]["bindingIdentity"].split("/")[0],
            "databaseProvider": e2_receipt["positive"]["databaseProvider"],
            "readScope": e2_receipt["scope"]["independenceKind"],
            "productE2Claim": e2_receipt["scope"]["productE2"],
        },
        "crossEvidence": {
            "distinctGitCheckout": e1_manifest["exactCheckoutHead"] != e2_receipt["repositoryRevision"],
            "differentApplication": e1_launch["application"] != e2_receipt["positive"]["bindingIdentity"].split("/")[0],
            "sameAcceptedRuntimeSourceVerified": claims["SAME_ACCEPTED_SOURCE_REVISION"],
        },
        "blockedObligations": blocked,
        "status": "UNJOINABLE_HISTORICAL_EVIDENCE",
        "productGate": "UNQUALIFIED",
        "canSatisfyG2": False,
        "reason": "Synthetic E1 and handwritten PostgreSQL E2a-1 are separate historical tests, not one authorized accepted-Source Web-to-DB invocation.",
    }


def verify_join(e1_manifest_path=None, e1_receipts_dir=None, e2_receipt_path=None):
    # Validate original *bytes* and all existing cross-file checks first.
    # A caller cannot use a forged decoded JSON object in production mode.
    E1.verify_archive(e1_manifest_path or E1.MANIFEST,
                      e1_receipts_dir or E1.RECEIPTS)
    E2.verify(e2_receipt_path or E2.RECEIPT)
    manifest = E1.decode_unique(Path(e1_manifest_path or E1.MANIFEST).read_bytes(), "E1 manifest")
    launch = E1.decode_unique(Path(e1_receipts_dir or E1.RECEIPTS).joinpath("target-launch.json").read_bytes(), "E1 target")
    receipt = E2.decode_unique(Path(e2_receipt_path or E2.RECEIPT).read_bytes())
    report = audit_pair(manifest, launch, receipt)
    if (report["blockedObligations"] != list(OBLIGATIONS)
            or report["e1"]["application"] != "order_entry"
            or report["e2a1"]["applicationFromBinding"] != "request_log"
            or report["crossEvidence"]["distinctGitCheckout"] is not True):
        raise ValueError("archived cross-evidence qualification boundary unexpectedly changed")
    return report


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--require-g2", action="store_true",
                        help="Fail unless evidence qualifies product G2 (these historical archives cannot).")
    args = parser.parse_args()
    report = verify_join()
    print(json.dumps(report, ensure_ascii=False, indent=2, sort_keys=True))
    return 3 if args.require_g2 and not report["canSatisfyG2"] else 0


if __name__ == "__main__":
    raise SystemExit(main())
