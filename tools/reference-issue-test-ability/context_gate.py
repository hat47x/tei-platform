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
            "inspect-validation", "source-index", "trace", "requirements", "validate")
ENVELOPES = {
    "workspace-files": "tei.cli.workspace-files/v0",
    "catalog": "tei.cli.model/v0",
    "inspect": "tei.cli.inspect/v0",
    "inspect-validation": "tei.cli.inspect/v0",
    "source-index": "tei.cli.source-index/v0",
    "trace": "tei.cli.trace/v0",
    "requirements": "tei.cli.requirements/v0",
    "validate": "tei.cli.validate/v0",
}
TARGET = {"kind": "semantic-data", "id": "sales.order"}
REQUIRED_SOURCE = "semantic/sales/order.tei"
# Independently chosen *one-Rule* obligations from the public #3273 / #3304
# case. This is a bounded case prerequisite, NOT complete Function/Business
# or organizational/permission closure.
REQUIRED_RULE = {"kind": "semantic-validation", "id": "sales.order_line.line_number_positive"}
REQUIRED_RULE_SOURCE = "semantic/sales/validation.tei"
REQUIRED_RULE_CANONICAL = {
    "id": "sales.order_line.line_number_positive",
    "kind": "positive",
    "attribute": "sales.order_line.line_number",
}
REQUIRED_RELATION = {"kind": "semantic-relation", "id": "sales.order.lines"}

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

    # A source being present and loaded is separate from the fact that the
    # canonical Rule was actually parsed, cataloged and linked to an anchor.
    rule_files = [e for e in managed if isinstance(e, dict)
                  and e.get("path") == REQUIRED_RULE_SOURCE]
    if not rule_files:
        rule_source_state = "missing"
    elif len(rule_files) != 1:
        rule_source_state = "ambiguous"
    elif rule_files[0].get("readByLoader") is not True:
        rule_source_state = "not-loaded"
    else:
        rule_source_state = "read-by-loader"
    if rule_source_state != "read-by-loader":
        blockers.append("required-rule-source:" + rule_source_state)

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

    # TT-56: JSON can be syntactically complete with the same asserted
    # fingerprints, yet omit one required Rule/Relation. Never infer
    # enumeration completeness from healthy envelopes or matching digests.
    rule_matches = [e for e in target_entries if isinstance(e, dict)
                    and e.get("subject") == REQUIRED_RULE]
    if not rule_matches:
        rule_state = "missing"
    elif len(rule_matches) != 1 or rule_matches[0].get("ambiguous") is True:
        rule_state = "ambiguous"
    elif (rule_matches[0].get("detail") != "positive" or
          rule_matches[0].get("sourceStatus") != "indexed"):
        rule_state = "not-source-backed"
    else:
        rule_state = "cataloged"
    if rule_state != "cataloged":
        blockers.append("required-rule:" + rule_state)

    relation_matches = [e for e in target_entries if isinstance(e, dict)
                        and e.get("subject") == REQUIRED_RELATION]
    if (len(relation_matches) != 1 or
            relation_matches[0].get("ambiguous") is True or
            relation_matches[0].get("parent") != TARGET):
        relation_state = "missing-or-ambiguous"
        blockers.append("required-relation:" + relation_state)
    else:
        relation_state = "cataloged"

    rule_inspection = docs.get("inspect-validation", {}).get("result", {})
    if not isinstance(rule_inspection, dict):
        rule_inspection = {}
    if rule_inspection.get("subject") != REQUIRED_RULE:
        rule_canonical_state = "subject-mismatch"
    elif not isinstance(rule_inspection.get("canonical"), dict):
        rule_canonical_state = "missing-canonical"
    elif any(rule_inspection["canonical"].get(k) != v
             for k, v in REQUIRED_RULE_CANONICAL.items()):
        rule_canonical_state = "rule-contract-drift"
    else:
        rule_canonical_state = "matched"
    if rule_canonical_state != "matched":
        blockers.append("required-rule-canonical:" + rule_canonical_state)

    source_refs = rule_inspection.get("sources", {}).get("references", [])
    artifacts = rule_inspection.get("sources", {}).get("artifacts", [])
    # Require exactly one *definition* reference to the independently
    # required file and an exact indexed artifact, not just a path guess.
    if not isinstance(source_refs, list):
        source_refs = []
    if not isinstance(artifacts, list):
        artifacts = []
    matching_refs = [r for r in source_refs if isinstance(r, dict)
                     and r.get("subject") == REQUIRED_RULE]
    matching_artifacts = [a for a in artifacts if isinstance(a, dict)
                          and a.get("path") == REQUIRED_RULE_SOURCE]
    if (rule_inspection.get("sourceStatus") != "indexed" or
            len(matching_refs) != 1 or len(matching_artifacts) != 1 or
            matching_refs[0].get("artifactPath") != REQUIRED_RULE_SOURCE or
            matching_refs[0].get("role") != "definition" or
            not isinstance(matching_artifacts[0].get("fingerprint"), str) or
            not matching_artifacts[0]["fingerprint"].startswith("sha256:")):
        rule_anchor_state = "missing-or-unqualified"
        blockers.append("required-rule-anchor:" + rule_anchor_state)
    else:
        rule_anchor_state = "definition-source-indexed"

    fingerprints = {}
    consistency = "consistent"
    for key in ("worldFingerprint", "sourceIndexFingerprint"):
        # Existing contract: these 4 commands expose common resolved fingerprints.
        values = [
            docs.get("catalog", {}).get("result", {}).get(key),
            inspected.get(key),
            rule_inspection.get(key),
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
            "requiredRuleSourceRead": rule_source_state,
            "requiredSubject": subject_state,
            "requiredRule": rule_state,
            "requiredRelation": relation_state,
            "requiredRuleCanonical": rule_canonical_state,
            "requiredRuleAnchor": rule_anchor_state,
            "fingerprintConsistency": consistency,
            "validationDiagnosticCount": diagnostic_count,
            "observedFingerprints": fingerprints if not blockers else {},
        },
        "boundedReference": "partial" if not blockers else "blocked",
        "independentlyRequiredCaseScope": "one validation Rule, its typed attribute, one relation, and two Source files",
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



def required_rule_selftest(docs: dict[str, dict]) -> dict[str, str]:
    """TT-56 and related prerequisite failures, tested on actual CLI JSON copies.

    All mutants keep the original world/index fingerprints. These are local
    consumer discrimination tests, not live provider/Harness transport faults.
    """
    results = {}

    def block(name: str, mutant: dict, expected_fragment: str) -> None:
        verdict = assess(mutant)
        if (verdict["boundedReference"] != "blocked"
                or verdict["testExecutionAdmission"] != "blocked"
                or verdict["qualifiedTestGeneration"] is not False
                or not any(expected_fragment in x for x in verdict["blockers"])):
            raise AssertionError(f"{name} falsely admitted: {verdict['blockers']}")
        results[name] = "rejected-with-identical-claimed-revisions"

    mutant = copy.deepcopy(docs)
    entries = mutant["catalog"]["result"]["entries"]
    mutant["catalog"]["result"]["entries"] = [
        e for e in entries if e.get("subject") != REQUIRED_RULE
    ]
    block("tt56-parseable-omitted-rule", mutant, "required-rule:missing")

    mutant = copy.deepcopy(docs)
    entries = mutant["catalog"]["result"]["entries"]
    mutant["catalog"]["result"]["entries"] = [
        e for e in entries if e.get("subject") != REQUIRED_RELATION
    ]
    block("tt56-parseable-omitted-relation", mutant, "required-relation:")

    mutant = copy.deepcopy(docs)
    entries = mutant["catalog"]["result"]["entries"]
    selected = next(e for e in entries if e.get("subject") == REQUIRED_RULE)
    entries.append(copy.deepcopy(selected))
    block("tt56-duplicate-rule-subject", mutant, "required-rule:ambiguous")

    mutant = copy.deepcopy(docs)
    mutant["inspect-validation"]["result"]["canonical"]["attribute"] = "std.incorrect"
    block("canonical-rule-attribute-drift", mutant, "required-rule-canonical:rule-contract-drift")

    mutant = copy.deepcopy(docs)
    mutant["inspect-validation"]["result"]["sources"]["references"] = []
    block("source-definition-reference-omitted", mutant, "required-rule-anchor:")

    mutant = copy.deepcopy(docs)
    for r in mutant["inspect-validation"]["result"]["sources"]["references"]:
        if r.get("subject") == REQUIRED_RULE:
            r["role"] = "derived"
    block("source-definition-role-substituted", mutant, "required-rule-anchor:")

    mutant = copy.deepcopy(docs)
    for entry in mutant["workspace-files"]["result"]["managedFiles"]:
        if entry.get("path") == REQUIRED_RULE_SOURCE:
            entry["readByLoader"] = False
    block("required-rule-source-not-consumed", mutant, "required-rule-source:not-loaded")

    mutant = copy.deepcopy(docs)
    mutant["inspect-validation"]["result"]["sourceIndexFingerprint"] = "synthetic-stale-rule-index"
    block("source-index-revision-mixed-for-rule", mutant, "fingerprint:stale")
    return results


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
            "tool-reply-truncated": original[:max(10, len(original) // 2)] + b"\n",
            "tool-reply-not-terminated": original.rstrip(b"\r\n"),
            "json-duplicate-member":
                b'{"schemaVersion":"tei.cli.model/v0","schemaVersion":"tei.cli.model/v0","ok":true}\n',
            "json-escaped-duplicate":
                b'{"id":1,"\\u0069d":2}\n',
            "json-nonfinite":
                b'{"schemaVersion":"tei.cli.model/v0","ok":true,"value":NaN}\n',
            "json-trailing-object": original + b'{"extra":true}\n',
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
    report["additionalRequiredRuleControls"] = required_rule_selftest(docs)
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
