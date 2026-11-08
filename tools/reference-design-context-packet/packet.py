#!/usr/bin/env python3
"""Reference *consumer* projection of existing TEI CLI evidence, not a new reader.

The separate obligations are a bounded test contract.  This does not parse TEI
Source, infer Owner meaning, authorize disclosure, or supply AI test oracles.
Failure to receive a complete reply, evidence disagreement, or missing required
obligations is represented explicitly; never by an empty-success default.
"""
from __future__ import annotations

import argparse
import copy
import hashlib
import json
from pathlib import Path
from typing import Any

NAMES = ("commands", "version", "workspace-files", "catalog", "inspect",
         "source-index", "trace", "requirements", "validate")
SCHEMAS = {
    "commands": "tei.cli.discovery/v0",
    "version": "tei.cli.version/v0",
    "workspace-files": "tei.cli.workspace-files/v0",
    "catalog": "tei.cli.model/v0",
    "inspect": "tei.cli.inspect/v0",
    "source-index": "tei.cli.source-index/v0",
    "trace": "tei.cli.trace/v0",
    "requirements": "tei.cli.requirements/v0",
    "validate": "tei.cli.validate/v0",
}
REQUIRED_FACETS = ("data", "function", "business", "rule", "dependency",
                   "parser-and-role", "accepted-meaning", "disclosure",
                   "external-delivery")


def sha256(raw: bytes) -> str:
    return hashlib.sha256(raw).hexdigest()


def read_complete(folder: Path) -> tuple[dict[str, dict], dict[str, dict]]:
    docs: dict[str, dict] = {}
    evidence: dict[str, dict] = {}
    for name in NAMES:
        path = folder / f"{name}.json"
        if not path.is_file():
            raise ValueError(f"missing Tool reply {name}")
        raw = path.read_bytes()
        if not raw or not raw.endswith(b"\n"):
            raise ValueError(f"Tool reply incomplete/unterminated {name}")
        try:
            item = json.loads(raw)
        except (UnicodeError, json.JSONDecodeError) as exc:
            raise ValueError(f"Tool reply is not complete JSON {name}") from exc
        if not isinstance(item, dict):
            raise ValueError(f"Tool reply not an object {name}")
        if item.get("schemaVersion") != SCHEMAS[name]:
            raise ValueError(f"Tool reply schema mismatch {name}")
        if name not in ("commands", "version") and item.get("ok") is not True:
            raise ValueError(f"Tool reply failure {name}")
        docs[name] = item
        evidence[name] = {"bytes": len(raw), "sha256": sha256(raw), "received": "complete-local-file"}
    if docs["commands"].get("scope") != "selected-read-only-commands":
        raise ValueError("Discovery scope is not the selected read-only subset")
    return docs, evidence


def read_obligations(path: Path) -> dict:
    obj = json.loads(path.read_bytes())
    if not isinstance(obj, dict) or obj.get("schema") != "tei.reference.design-context-obligations/v0":
        raise ValueError("unknown requirements contract")
    target = obj.get("target")
    sources = obj.get("requiredSources")
    if not isinstance(target, dict) or set(target) != {"kind", "id"} or not all(
        isinstance(v, str) and v for v in target.values()
    ):
        raise ValueError("missing exact required target")
    if (not isinstance(sources, list) or not sources or
            any(not isinstance(s, str) or not s or s.startswith("/") or ".." in s.split("/") for s in sources) or
            len(set(sources)) != len(sources)):
        raise ValueError("missing, duplicate, or unsafe required Source set")
    if obj.get("requiredFacets") != list(REQUIRED_FACETS):
        raise ValueError("the independently required facets were changed")
    return obj


def build_packet(docs: dict[str, dict], evidence: dict[str, dict], obligations: dict) -> dict:
    target = obligations["target"]
    required_sources = obligations["requiredSources"]
    errors: list[str] = []

    entries = docs["catalog"].get("result", {}).get("entries")
    if not isinstance(entries, list):
        errors.append("catalog.entries:not-observed")
        entries = []
    selected = [x for x in entries if isinstance(x, dict) and x.get("subject") == target]
    subject_ok = len(selected) == 1 and selected[0].get("ambiguous") is not True
    inspected = docs["inspect"].get("result")
    if not isinstance(inspected, dict) or inspected.get("subject") != target or not subject_ok:
        errors.append("required-subject:unresolved")
    managed = docs["workspace-files"].get("result", {}).get("managedFiles")
    if not isinstance(managed, list):
        managed = []
        errors.append("managed-inventory:not-observed")
    source_ledger = []
    for required in required_sources:
        matches = [x for x in managed if isinstance(x, dict) and x.get("path") == required]
        if len(matches) != 1:
            state = "missing-or-ambiguous"
        else:
            state = "read-by-loader" if matches[0].get("readByLoader") is True else "not-read-by-loader"
        source_ledger.append({"path": required, "loaderObservation": state,
                              "parsedAndRole": "not-observed", "acceptedMeaning": "not-observed"})
        if state != "read-by-loader":
            errors.append(f"required-source:{required}:{state}")

    fingerprints: dict[str, dict[str, Any]] = {}
    for key in ("worldFingerprint", "sourceIndexFingerprint"):
        observed = {}
        for name in ("catalog", "inspect", "trace", "validate"):
            body = docs[name].get("result") if name != "validate" else docs[name]
            observed[name] = body.get(key) if isinstance(body, dict) else None
        vals = list(observed.values())
        if not all(isinstance(v, str) and v for v in vals):
            errors.append(f"{key}:unobserved")
            state = "incomplete"
        elif len(set(vals)) != 1:
            errors.append(f"{key}:conflict")
            state = "stale"
        else:
            state = "same-observed-fingerprint"
        fingerprints[key] = {"state": state, "values": observed}
    source_roots = docs["workspace-files"]["result"].get("sourceRootFindings", [])
    if not isinstance(source_roots, list):
        errors.append("source-root-findings:not-observed")
        source_roots = []
    if source_roots:
        errors.append("source-root-findings:nonempty")
    validate = docs["validate"]
    diagnostic_count = len(validate.get("diagnostics", [])) if isinstance(validate.get("diagnostics"), list) else None
    if diagnostic_count is None:
        errors.append("validation-diagnostics:not-observed")

    states = {x: "not-observed" for x in REQUIRED_FACETS}
    states["data"] = "observed-subject-only" if subject_ok else "incomplete"
    if errors:
        status = "stale" if any(e.endswith(":conflict") for e in errors) else "incomplete"
    else:
        status = "partial"
    # Every required facet is independent. Never promote observed Data to
    # Function/Business/Rule/parser/meaning/external-delivery qualification.
    return {
        "schema": "tei.reference.design-context-packet/v0",
        "obligations": {"issue": obligations["issue"], "target": target,
                        "requiredSources": required_sources, "requiredFacets": list(REQUIRED_FACETS)},
        "state": status,
        "claimCeiling": "bounded-local-observation; no complete design or test-generation qualification",
        "complete": False,
        "candidateTestGenerationQualified": False,
        "data": {"target": target, "subjectObserved": subject_ok},
        "sourceLedger": source_ledger,
        "facets": states,
        "versionAgreement": fingerprints,
        "validation": {"diagnosticsCount": diagnostic_count,
                       "sourceRootFindingCount": len(source_roots),
                       "ownerMeaningAccepted": None},
        "delivery": {"channel": "local-workflow-file",
                     "externalHarnessReplyComplete": None, "disclosureGranted": None},
        "unresolved": errors,
        "cliEvidence": evidence,
    }


def negative_controls(docs: dict, evidence: dict, obligations: dict) -> dict[str, str]:
    positive = build_packet(docs, evidence, obligations)
    if positive["state"] != "partial" or not positive["data"]["subjectObserved"]:
        raise AssertionError(f"reference source no longer matches bounded obligations: {positive['unresolved']}")
    if not all(e["loaderObservation"] == "read-by-loader" for e in positive["sourceLedger"]):
        raise AssertionError("required Source must be actually read in base fixture")
    controls: dict[str, str] = {}
    cases = ("missing-subject", "unloaded-source", "missing-source",
             "stale-inspect-revision", "missing-world-fingerprint", "source-root-finding")
    for case in cases:
        mutated = copy.deepcopy(docs)
        target = obligations["target"]
        source = obligations["requiredSources"][0]
        records = mutated["workspace-files"]["result"]["managedFiles"]
        if case == "missing-subject":
            mutated["catalog"]["result"]["entries"] = [
                e for e in mutated["catalog"]["result"]["entries"] if e.get("subject") != target]
        elif case == "unloaded-source":
            for x in records:
                if x.get("path") == source:
                    x["readByLoader"] = False
        elif case == "missing-source":
            mutated["workspace-files"]["result"]["managedFiles"] = [
                x for x in records if x.get("path") != source]
        elif case == "stale-inspect-revision":
            mutated["inspect"]["result"]["sourceIndexFingerprint"] = "synthetic-stale"
        elif case == "missing-world-fingerprint":
            mutated["catalog"]["result"].pop("worldFingerprint", None)
        elif case == "source-root-finding":
            mutated["workspace-files"]["result"]["sourceRootFindings"] = [{"synthetic": True}]
        observed = build_packet(mutated, evidence, obligations)
        want = "stale" if case == "stale-inspect-revision" else "incomplete"
        if observed["state"] != want or observed["complete"] or observed["candidateTestGenerationQualified"]:
            raise AssertionError(f"{case} bypassed fail-closed packet qualification: {observed}")
        controls[case] = observed["state"]
    return controls


def main() -> None:
    p = argparse.ArgumentParser()
    p.add_argument("--input-dir", required=True, type=Path)
    p.add_argument("--obligations", required=True, type=Path)
    p.add_argument("--output", required=True, type=Path)
    args = p.parse_args()
    obligations = read_obligations(args.obligations)
    docs, evidence = read_complete(args.input_dir)
    pkt = build_packet(docs, evidence, obligations)
    pkt["syntheticControls"] = negative_controls(docs, evidence, obligations)
    # Re-run read errors with actual temporary reply files: these must not be
    # normalized to [] or falsely treated as success.
    import tempfile
    with tempfile.TemporaryDirectory() as temp:
        fixture = Path(temp)
        for name in NAMES:
            (fixture / f"{name}.json").write_bytes((args.input_dir / f"{name}.json").read_bytes())
        for name, raw in (
            ("omitted-reply", None),
            ("truncated-reply", b'{"schemaVersion":"tei.cli.model/v0","ok":true'),
            ("wrong-schema", b'{"schemaVersion":"tei.cli.model/v999","ok":true}\n'),
        ):
            path = fixture / "catalog.json"
            old = path.read_bytes()
            if raw is None:
                path.unlink()
            else:
                path.write_bytes(raw)
            try:
                read_complete(fixture)
            except ValueError:
                pkt["syntheticControls"][name] = "rejected-before-packet"
            else:
                raise AssertionError(f"{name}: incomplete Tool reply accepted")
            path.write_bytes(old)
    args.output.parent.mkdir(parents=True, exist_ok=True)
    args.output.write_text(json.dumps(pkt, indent=2, ensure_ascii=False, sort_keys=True) + "\n",
                           encoding="utf-8")
    assert pkt["state"] == "partial" and len(pkt["syntheticControls"]) == 9
    print("PASS: bounded Design Context Packet / 9 negative controls; completeness remains PARTIAL")
    print("Subject:", obligations["target"], "Sources:", pkt["sourceLedger"])
    print("Full context, Parser, Meaning, Rule/Business/Function, external delivery: NOT QUALIFIED")


if __name__ == "__main__":
    main()
