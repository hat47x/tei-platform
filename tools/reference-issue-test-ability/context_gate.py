#!/usr/bin/env python3
"""Consumer-side, bounded Issue #3262 Design Context readiness probe.

Only reads JSON from existing TEI CLI commands; deliberately NOT a Source
parser, a second model/Scenario SSOT, a permission evaluator, or a product API.
The fixed obligations below are independent tests for public Reference only.
"""
from __future__ import annotations

import argparse
import copy
import hashlib
import json
from pathlib import Path

COMMANDS = ("commands", "version", "workspace-files", "catalog", "inspect",
            "source-index", "trace", "requirements", "validate")
ENVELOPES = {
    "workspace-files": "tei.cli.workspace-files/v0",
    "catalog": "tei.cli.model/v0",
    "inspect": "tei.cli.inspect/v0",
    "source-index": "tei.cli.source-index/v0",
    "trace": "tei.cli.trace/v0",
    "requirements": "tei.cli.requirements/v0",
    "validate": "tei.cli.validate/v0",
}
TARGET = {"kind": "semantic-data", "id": "sales.order"}
REQUIRED_SOURCE = "semantic/sales/order.tei"
# A test-only independent required set, NOT a claimed Workspace/Domain closure.
# Function/Business/Rule dependencies are deliberately still unqualified.
MISSING_CLAIMS = [
    "independent-required-closure: Function/Business/Rule/Procedure",
    "source-parser-role-and-accepted-meaning",
    "actual-function-query-to-provider-read-and-result-completeness",
    "consumer-disclosure-grant-and-external-harness-complete-delivery",
    "independent-domain-requirement-and-C1-NFR-coverage",
]


def _no_duplicate(pairs: list[tuple[str, object]]) -> dict:
    result = {}
    for key, value in pairs:
        if key in result:
            raise ValueError("duplicate JSON member: " + key)
        result[key] = value
    return result


def _no_nonfinite(constant: str) -> None:
    raise ValueError("non-finite JSON constant: " + constant)


def load(folder: Path) -> tuple[dict[str, dict], list[str], dict[str, dict]]:
    docs = {}
    failures = []
    byte_evidence = {}
    for name in COMMANDS:
        path = folder / (name + ".json")
        try:
            raw = path.read_bytes()
            byte_evidence[name] = {"bytes": len(raw), "sha256": hashlib.sha256(raw).hexdigest()}
            if not raw or not raw.endswith(b"\n"):
                raise ValueError("missing nonempty newline-terminated response")
            doc = json.loads(raw.decode("utf-8"), object_pairs_hook=_no_duplicate,
                             parse_constant=_no_nonfinite)
            if not isinstance(doc, dict):
                raise ValueError("JSON object expected")
            docs[name] = doc
        except (OSError, UnicodeError, ValueError) as exc:
            failures.append(f"{name}: {type(exc).__name__}")
    return docs, sorted(failures), byte_evidence


def assess(docs: dict[str, dict], failures: list[str] | None = None) -> dict:
    """Fail closed; 'partial' is a measured narrow slice, not full readiness."""
    blockers = list(failures or [])
    schema_ok = (
        docs.get("commands", {}).get("schemaVersion") == "tei.cli.discovery/v0"
        and docs.get("commands", {}).get("scope") == "selected-read-only-commands"
        and docs.get("version", {}).get("schemaVersion") == "tei.cli.version/v0"
    )
    for name, version in ENVELOPES.items():
        obj = docs.get(name, {})
        if obj.get("schemaVersion") != version or obj.get("ok") is not True:
            schema_ok = False
    if not schema_ok:
        blockers.append("schema-or-command-failure")
    if len(docs) != len(COMMANDS):
        blockers.append("missing-command-reply")

    managed = docs.get("workspace-files", {}).get("result", {}).get("managedFiles", [])
    if not isinstance(managed, list):
        managed = []
    matches = [e for e in managed if isinstance(e, dict) and e.get("path") == REQUIRED_SOURCE]
    if not matches:
        source_state = "missing"
    elif len(matches) != 1:
        source_state = "ambiguous"
    elif matches[0].get("readByLoader") is not True:
        source_state = "not-loaded"
    else:
        source_state = "read-by-loader"
    if source_state != "read-by-loader":
        blockers.append("required-source:" + source_state)

    target_entries = docs.get("catalog", {}).get("result", {}).get("entries", [])
    if not isinstance(target_entries, list):
        target_entries = []
    found = [e for e in target_entries if isinstance(e, dict) and e.get("subject") == TARGET]
    inspected = docs.get("inspect", {}).get("result", {})
    if not isinstance(inspected, dict):
        inspected = {}
    subject_state = "found"
    if not found or inspected.get("subject") != TARGET:
        subject_state = "missing-or-mismatched"
    elif len(found) != 1 or found[0].get("ambiguous") is True:
        subject_state = "ambiguous"
    if subject_state != "found":
        blockers.append("required-subject:" + subject_state)

    fingerprints = {}
    consistency = "consistent"
    for key in ("worldFingerprint", "sourceIndexFingerprint"):
        # Existing contract: these 4 commands expose common resolved fingerprints.
        values = [
            docs.get("catalog", {}).get("result", {}).get(key),
            inspected.get(key),
            docs.get("trace", {}).get("result", {}).get(key),
            docs.get("validate", {}).get(key),
        ]
        if not all(isinstance(x, str) and x for x in values):
            consistency = "missing"
        elif len(set(values)) != 1:
            consistency = "stale"
        if consistency != "consistent":
            break
        fingerprints[key] = values[0]
    if consistency != "consistent":
        blockers.append("fingerprint:" + consistency)

    # validate.ok is transport/evaluation success, not evidence that every
    # authored rule is accepted or that an actual Provider was exercised.
    val = docs.get("validate", {})
    root_findings = val.get("sourceRootFindings")
    workspace_findings = docs.get("workspace-files", {}).get("result", {}).get("sourceRootFindings")
    # These are two different projections of Source Root findings. A Consumer
    # must not infer "no missing roots" from only one successful command when
    # the other read-only Tool reply actually reports missing/unknown roots.
    if ((isinstance(root_findings, list) and root_findings) or
            (isinstance(workspace_findings, list) and workspace_findings)):
        blockers.append("source-root-findings")
    diagnostics = val.get("diagnostics")
    diagnostic_count = len(diagnostics) if isinstance(diagnostics, list) else None

    blockers = sorted(set(blockers))
    return {
        "schema": "tei.reference.design-context-readiness/v0",
        "issue": "hat47x/tei-platform#3262",
        "subject": TARGET,
        "requiredSource": REQUIRED_SOURCE,
        "observations": {
            "localReplies": f"{len(docs)}/{len(COMMANDS)}",
            "requiredSourceRead": source_state,
            "requiredSubject": subject_state,
            "fingerprintConsistency": consistency,
            "validationDiagnosticCount": diagnostic_count,
            "observedFingerprints": fingerprints if not blockers else {},
        },
        "boundedReference": "partial" if not blockers else "blocked",
        "testExecutionAdmission": "exploration-only" if not blockers else "blocked",
        # Deliberately impossible to auto-upgrade from this local CLI slice:
        "qualifiedTestGeneration": False,
        "productMeaningAccepted": False,
        "externalHarnessDeliveryComplete": None,
        "independentFullRequiredSetProven": False,
        "missingClaims": MISSING_CLAIMS,
        "blockers": blockers,
    }


def selftest(docs: dict[str, dict], baseline: dict) -> dict[str, str]:
    assert baseline["boundedReference"] == "partial", baseline["blockers"]
    assert baseline["testExecutionAdmission"] == "exploration-only"
    assert baseline["qualifiedTestGeneration"] is False

    controls = {}

    def check(name: str, variant: dict, missing: list[str] | None, fragment: str) -> None:
        report = assess(variant, missing)
        if report["boundedReference"] != "blocked" or report["testExecutionAdmission"] != "blocked":
            raise AssertionError(f"{name}: unsafe admission {report}")
        if not any(fragment in b for b in report["blockers"]):
            raise AssertionError(f"{name}: wrong rejection reason {report['blockers']}")
        if report["qualifiedTestGeneration"] is not False:
            raise AssertionError(f"{name}: false qualified")
        controls[name] = "blocked"

    # Change the response, NOT TEI's real Source or any business record.
    mutated = copy.deepcopy(docs)
    mutated["catalog"]["result"]["entries"] = [
        e for e in mutated["catalog"]["result"]["entries"] if e.get("subject") != TARGET
    ]
    check("missing-required-subject", mutated, None, "required-subject")

    mutated = copy.deepcopy(docs)
    files = mutated["workspace-files"]["result"]["managedFiles"]
    for e in files:
        if e.get("path") == REQUIRED_SOURCE:
            e["readByLoader"] = False
    check("inventory-not-consumed", mutated, None, "not-loaded")

    mutated = copy.deepcopy(docs)
    mutated["inspect"]["result"]["sourceIndexFingerprint"] = "synthetic-ABA-or-stale-fingerprint"
    check("mixed-revisions", mutated, None, "fingerprint:stale")

    mutated = copy.deepcopy(docs)
    mutated.pop("trace")
    check("missing-tool-reply", mutated, ["trace: missing"], "missing-command-reply")

    mutated = copy.deepcopy(docs)
    mutated["catalog"]["schemaVersion"] = "unexpected.contract/v77"
    check("schema-drift", mutated, None, "schema-or-command-failure")

    mutated = copy.deepcopy(docs)
    mutated["validate"]["sourceRootFindings"] = [{"synthetic": "unknown-root"}]
    check("missing-source-root", mutated, None, "source-root-findings")

    # The local response can be perfectly parsed but external receiving
    # authority, actual DB Read, Source Role, and Owner acceptance stay unknown.
    assert baseline["externalHarnessDeliveryComplete"] is None
    assert baseline["independentFullRequiredSetProven"] is False
    assert len(baseline["missingClaims"]) >= 5
    controls["unobserved-harness-and-provider"] = "qualification-withheld"
    return controls


def manifest_selftest(docs: dict[str, dict]) -> dict[str, str]:
    """Check Source inventory and root diagnosis from *both* CLI projections."""
    cases = {}
    mutant = copy.deepcopy(docs)
    mutant["workspace-files"]["result"]["sourceRootFindings"] = [
        {"synthetic": "workspace-files-only-unknown-root"}
    ]
    result = assess(mutant)
    if result["boundedReference"] != "blocked" or "source-root-findings" not in result["blockers"]:
        raise AssertionError("workspace-files Source root findings bypassed readiness")
    cases["workspace-files-root-only"] = "blocked"

    mutant = copy.deepcopy(docs)
    managed = mutant["workspace-files"]["result"]["managedFiles"]
    selected = next((x for x in managed if x.get("path") == REQUIRED_SOURCE), None)
    if selected is None:
        raise AssertionError("reference required Source disappeared")
    managed.append(copy.deepcopy(selected))
    result = assess(mutant)
    if result["boundedReference"] != "blocked" or "required-source:ambiguous" not in result["blockers"]:
        raise AssertionError("duplicate Required Source inventory bypassed readiness")
    cases["duplicate-required-source"] = "blocked"
    return cases


def transport_selftest(folder: Path) -> dict[str, str]:
    """Corrupt real received CLI JSON bytes, not just an in-memory decoded dict.

    A syntactically valid first page is *not* a qualified complete Tool reply.
    Keep this checker test-only; the actual Source/Runner/Workspace is untouched.
    """
    import tempfile

    cases = {}
    with tempfile.TemporaryDirectory(prefix="tei-context-transport-") as directory:
        scratch = Path(directory)
        for command in COMMANDS:
            (scratch / (command + ".json")).write_bytes(
                (folder / (command + ".json")).read_bytes())
        target = scratch / "catalog.json"
        original = target.read_bytes()
        corruptions = {
            "tool-reply-omitted": None,
            "tool-reply-truncated": original[:max(10, len(original) // 2)] + b"\\n",
            "tool-reply-not-terminated": original.rstrip(b"\\r\\n"),
            "json-duplicate-member":
                b'{"schemaVersion":"tei.cli.model/v0","schemaVersion":"tei.cli.model/v0","ok":true}\\n',
            "json-escaped-duplicate":
                b'{"id":1,"\\\\u0069d":2}\\n',
            "json-nonfinite":
                b'{"schemaVersion":"tei.cli.model/v0","ok":true,"value":NaN}\\n',
            "json-trailing-object": original + b'{"extra":true}\\n',
        }
        for name, data in corruptions.items():
            if data is None:
                target.unlink()
            else:
                target.write_bytes(data)
            docs, failures, _ = load(scratch)
            verdict = assess(docs, failures)
            if not any(entry.startswith("catalog:") for entry in failures):
                raise AssertionError(f"{name}: corrupt received byte reply was accepted")
            if (verdict["boundedReference"] != "blocked" or
                    verdict["testExecutionAdmission"] != "blocked" or
                    verdict["qualifiedTestGeneration"] is not False):
                raise AssertionError(f"{name}: incorrect qualification {verdict}")
            cases[name] = "rejected-before-readiness"
            target.write_bytes(original)
    return cases


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--in", dest="folder", type=Path, required=True)
    parser.add_argument("--out", type=Path, required=True)
    args = parser.parse_args()
    docs, failures, byte_evidence = load(args.folder)
    baseline = assess(docs, failures)
    controls = selftest(docs, baseline)
    report = dict(baseline)
    report["syntheticControls"] = controls
    report["additionalManifestControls"] = manifest_selftest(docs)
    report["transportByteControls"] = transport_selftest(args.folder)
    report["localReplyEvidence"] = byte_evidence
    args.out.parent.mkdir(parents=True, exist_ok=True)
    args.out.write_text(json.dumps(report, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    print(f"PASS: bounded Design Context {baseline['boundedReference']}; "
          f"{len(controls)} base controls + {len(report['additionalManifestControls'])} manifest "
          f"+ {len(report['transportByteControls'])} byte-reply rejection controls; "
          "test generation qualification withheld")


if __name__ == "__main__":
    main()
