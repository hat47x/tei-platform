#!/usr/bin/env python3
"""Read-only E1 linkage to the existing Shared Acceptance Decision Index.

Input decisions MUST be produced by reference-shared-acceptance decisions,
which rechecks content-addressed Decision records against the current
Workspace. This probe never issues Acceptance, BuildArtifactLock or Host
authority and never treats a Decision's Origin as its approving actor.
"""
import argparse
import hashlib
import json
from pathlib import Path
import re


SHA = re.compile(r"^sha256:[0-9a-f]{64}$")
HEX = re.compile(r"^[0-9a-f]{64}$")


def unique_object(pairs):
    result = {}
    for key, value in pairs:
        if key in result:
            raise ValueError(f"duplicate JSON member: {key}")
        result[key] = value
    return result


def load_json(path):
    return json.loads(Path(path).read_text("utf-8"), object_pairs_hook=unique_object)


def observe(frozen_path, metadata_path, decisions_path):
    frozen_bytes = Path(frozen_path).read_bytes()
    frozen = load_json(frozen_path)
    metadata = load_json(metadata_path)
    decisions = load_json(decisions_path)
    actual_frozen_digest = hashlib.sha256(frozen_bytes).hexdigest()
    frozen_digest = metadata.get("frozenWorldSha256")
    derived = metadata.get("derivedWorldSha256")
    if not isinstance(frozen_digest, str) or not HEX.fullmatch(frozen_digest):
        raise ValueError("missing frozen World digest")
    if actual_frozen_digest != frozen_digest:
        raise ValueError("frozen World differs from generator observation")
    if not isinstance(derived, str) or not SHA.fullmatch(derived):
        raise ValueError("invalid derived World SHA-256")
    if frozen.get("schema") != "tei.reference.first-web-frozen-world/v1" or frozen.get("applicationId") != "order_entry":
        raise ValueError("unexpected fixed-world schema/application")
    if not isinstance(frozen.get("world"), dict):
        raise ValueError("missing normalized World")
    if decisions.get("schema") != "tei.reference.decision-index/v0":
        raise ValueError("unexpected Shared Acceptance Decision Index schema")
    revision = decisions.get("modelRevision")
    if not isinstance(revision, str) or not SHA.fullmatch(revision):
        raise ValueError("invalid current Source Index model revision")
    records = decisions.get("records")
    if not isinstance(records, list):
        raise ValueError("missing Decision Index records")
    current_accepted = []
    for record in records:
        if not isinstance(record, dict):
            raise ValueError("invalid Decision record")
        status = record.get("status")
        if status not in ("current", "superseded"):
            raise ValueError("invalid Decision status")
        if status != "current":
            continue
        decision = record.get("decision")
        if not isinstance(decision, dict) or decision.get("decision") != "accepted":
            continue
        if decision.get("sourceChanged") is not True:
            raise ValueError("current accepted Decision lacks recorded source change")
        if decision.get("modelRevisionAfter") != revision:
            raise ValueError("current accepted Decision revision mismatch")
        if record.get("sourceEdits") != "intact":
            raise ValueError("current accepted Decision has diverged source edits")
        if record.get("reviewBindingStatus") == "invalid" or record.get("evidenceStatus") == "diverged":
            raise ValueError("current accepted Decision has invalid review/source evidence")
        if decision.get("derivedWorldSha256") != derived:
            raise ValueError("current accepted Decision does not describe the generated World")
        path = record.get("path")
        if not isinstance(path, str) or not re.fullmatch(r"\.tei/decisions/[0-9a-f]{64}\.json", path):
            raise ValueError("invalid content-addressed Decision path")
        current_accepted.append(path)
    return {
        "kind": "diagnostic-first-web-frozen-world-lineage",
        "sourceModelRevision": revision,
        "derivedWorldSHA256": derived,
        "frozenWorldSHA256": frozen_digest,
        "currentAcceptedDecisionRecords": sorted(current_accepted),
        "currentAcceptedDecisionMatched": bool(current_accepted),
        # The current Decision ledger marks the actor and the decision
        # authority as unresolved; neither a fingerprint nor a claimed
        # Origin authorizes deployment of the API or any plugin.
        "sourceAcceptanceAuthorityVerified": False,
        "buildArtifactLockVerified": False,
        "runtimeModelAcceptanceAuthority": False,
        "artifactAuthority": False,
    }


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--frozen-world", required=True)
    parser.add_argument("--generator-observation", required=True)
    parser.add_argument("--decision-index", required=True)
    parser.add_argument("--out", required=True)
    args = parser.parse_args()
    report = observe(args.frozen_world, args.generator_observation, args.decision_index)
    Path(args.out).write_text(json.dumps(report, sort_keys=True, indent=2) + "\n", encoding="utf-8")


if __name__ == "__main__":
    main()
