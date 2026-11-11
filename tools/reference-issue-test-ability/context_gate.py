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
            "source-index", "source-index-all", "inspect-validation",
            "inspect-capability", "inspect-application", "trace", "requirements", "validate")
ENVELOPES = {
    "workspace-files": "tei.cli.workspace-files/v0",
    "catalog": "tei.cli.model/v0",
    "inspect": "tei.cli.inspect/v0",
    "source-index": "tei.cli.source-index/v0",
    "source-index-all": "tei.cli.source-index/v0",
    "inspect-validation": "tei.cli.inspect/v0",
    "inspect-capability": "tei.cli.inspect/v0",
    "inspect-application": "tei.cli.inspect/v0",
    "trace": "tei.cli.trace/v0",
    "requirements": "tei.cli.requirements/v0",
    "validate": "tei.cli.validate/v0",
}
TARGET = {"kind": "semantic-data", "id": "sales.order"}
REQUIRED_SOURCE = "semantic/sales/order.tei"
# A test-only independent required set, NOT a claimed Workspace/Domain closure.
# Function/Business/Rule dependencies are deliberately still unqualified.
# Fixed independent obligations for the *public, synthetic* Rule slice exercised
# by #3304. They are not inferred from Catalog or Scenario Expected, and do
# not purport to enumerate all Function/Business/Requirement dependencies.
REQUIRED_VALIDATION_SOURCE = "semantic/sales/validation.tei"
FOCUS_RULE = {"kind": "semantic-validation", "id": "sales.order_line.line_number_positive"}
FOCUS_RULE_CANONICAL = {
    "id": "sales.order_line.line_number_positive",
    "kind": "positive",
    "attribute": "sales.order_line.line_number",
}
FOCUS_RELATION = {"kind": "semantic-relation", "id": "sales.order.lines"}
# P-1b1 independent, fixed *declaration-level* Function Case. Do not infer
# actual business invocation, query binding, Host Grant or complete closure.
FUNCTION_SUBJECT = {"kind": "semantic-capability", "id": "sales.order.submit"}
APPLICATION_SUBJECT = {"kind": "application", "id": "order_entry"}
APPLICATION_SOURCE = "applications/order-entry/application.tei"
FUNCTION_INPUT = {"name": "order", "type": "sales.order"}
FUNCTION_OUTPUT = "sales.order"
# Independently fixed Data whose attribute types / Relation targets the Function
# slice depends on: the declared input/output Data and the required Relation
# target. One hop only; see source_currency_slice.
CLOSURE_DATA = ("sales.order", "sales.order_line")
FUNCTION_VALIDATIONS = (
    "sales.order_line.quantity_positive",
    "sales.order.lines.line_number_unique",
    "sales.order_line.line_number_positive",
)


REQUIRED_VALIDATIONS = (
    ("sales.order_line.quantity_positive", "positive"),
    ("sales.order.lines.line_number_unique", "relation-unique"),
    ("sales.order_line.line_number_positive", "positive"),
)

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



def required_validation_slice(docs: dict[str, dict]) -> tuple[dict, list[str]]:
    """Bound one *independently named* Rule set to existing Catalog and Index.

    This is a consumer of resolved TEI operations, not a duplicate DSL reader.
    Both replies may remain individually parseable and claim the same
    fingerprints while omitting a sibling required by this test obligation.
    """
    problems = []
    managed = docs.get("workspace-files", {}).get("result", {}).get("managedFiles", [])
    if not isinstance(managed, list):
        managed = []
    source_files = [x for x in managed if isinstance(x, dict)
                    and x.get("path") == REQUIRED_VALIDATION_SOURCE]
    source_state = ("missing" if not source_files else
                    "ambiguous" if len(source_files) != 1 else
                    "loaded" if source_files[0].get("readByLoader") is True else
                    "not-loaded")
    if source_state != "loaded":
        problems.append("required-rule-source:" + source_state)

    catalog = docs.get("catalog", {}).get("result", {}).get("entries", [])
    if not isinstance(catalog, list):
        catalog = []
    index_reply = docs.get("source-index-all", {})
    if index_reply.get("mode") != "index":
        problems.append("required-rule-index:wrong-mode")
    index = index_reply.get("result", {})
    if not isinstance(index, dict):
        index = {}
    refs = index.get("references", [])
    if not isinstance(refs, list):
        refs = []
    artifacts = index.get("artifacts", [])
    if not isinstance(artifacts, list):
        artifacts = []
    sources = [a for a in artifacts if isinstance(a, dict)
               and a.get("path") == REQUIRED_VALIDATION_SOURCE]
    if len(sources) != 1 or not isinstance(sources[0].get("fingerprint"), str) or not sources[0]["fingerprint"]:
        problems.append("required-rule-artifact:missing-or-ambiguous")

    rule_states = []
    for rule_id, kind in REQUIRED_VALIDATIONS:
        subject = {"kind": "semantic-validation", "id": rule_id}
        entries = [entry for entry in catalog if isinstance(entry, dict)
                   and entry.get("subject") == subject]
        if len(entries) != 1:
            catalog_state = "missing" if not entries else "ambiguous"
        elif entries[0].get("ambiguous") is True or entries[0].get("detail") != kind:
            catalog_state = "ambiguous-or-kind-mismatched"
        elif entries[0].get("sourceStatus") != "indexed":
            catalog_state = "not-indexed"
        else:
            catalog_state = "canonical-catalogued"
        if catalog_state != "canonical-catalogued":
            problems.append("required-rule-catalog:" + rule_id + ":" + catalog_state)
        # Definition Reference is an actual operation output, not proof of
        # Owner acceptance or semantic completeness of every other Rule.
        definitions = [ref for ref in refs if isinstance(ref, dict)
                       and ref.get("subject") == subject
                       and ref.get("role") == "definition"
                       and ref.get("artifactPath") == REQUIRED_VALIDATION_SOURCE]
        wrong = [ref for ref in refs if isinstance(ref, dict)
                 and ref.get("subject") == subject and ref not in definitions]
        index_state = ("definition-indexed" if len(definitions) == 1 and not wrong else
                       "missing-or-wrong-role-path" if not definitions else "ambiguous")
        if index_state != "definition-indexed":
            problems.append("required-rule-index:" + rule_id + ":" + index_state)
        rule_states.append({
            "subject": subject,
            "declaredKind": kind,
            "catalog": catalog_state,
            "definitionReference": index_state,
        })
    # A bounded, independent test obligation: *not* the full Workspace or
    # Function closure. No inference from absence in a generic partial Query.
    observation = {
        "scope": "fixed-public-reference-only",
        "source": REQUIRED_VALIDATION_SOURCE,
        "sourceRead": source_state,
        "requiredCount": len(REQUIRED_VALIDATIONS),
        "rules": rule_states,
        "status": "bounded-present" if not problems else "blocked",
        "productRuleMeaningAccepted": False,
        "functionalDependencyClosure": "unqualified",
    }
    return observation, sorted(set(problems))



def focused_structure_slice(docs: dict[str, dict]) -> tuple[dict, list[str]]:
    """Cross-check one real Rule's Canonical/Source and one related Relation.

    Assesses independently selected obligations using *existing* operations.
    The three-Rule indexed existence check stays in required_validation_slice.
    Neither is an independent model parser nor a general Closure resolver.
    """
    errors = []
    entries = docs.get("catalog", {}).get("result", {}).get("entries", [])
    if not isinstance(entries, list):
        entries = []
    relation = [e for e in entries if isinstance(e, dict)
                and e.get("subject") == FOCUS_RELATION]
    if not relation:
        relation_state = "missing"
    elif len(relation) != 1 or relation[0].get("ambiguous") is True:
        relation_state = "ambiguous"
    elif relation[0].get("parent") != TARGET:
        relation_state = "parent-mismatch"
    else:
        relation_state = "matched"
    if relation_state != "matched":
        errors.append("required-relation:" + relation_state)

    inspection = docs.get("inspect-validation", {}).get("result", {})
    if not isinstance(inspection, dict):
        inspection = {}
    canonical = inspection.get("canonical")
    if inspection.get("subject") != FOCUS_RULE:
        canonical_state = "subject-mismatch"
    elif not isinstance(canonical, dict):
        canonical_state = "missing"
    elif any(canonical.get(k) != v for k, v in FOCUS_RULE_CANONICAL.items()):
        canonical_state = "kind-or-attribute-mismatch"
    else:
        canonical_state = "matched"
    if canonical_state != "matched":
        errors.append("required-rule-canonical:" + canonical_state)

    sources = inspection.get("sources", {})
    if not isinstance(sources, dict):
        sources = {}
    refs = sources.get("references", [])
    artifacts = sources.get("artifacts", [])
    all_idx = docs.get("source-index-all", {}).get("result", {})
    if not isinstance(all_idx, dict):
        all_idx = {}
    all_refs = all_idx.get("references", [])
    all_artifacts = all_idx.get("artifacts", [])
    for obj in (refs, artifacts, all_refs, all_artifacts):
        if not isinstance(obj, list):
            errors.append("required-rule-source:invalid-index-shape")
    if not isinstance(refs, list):
        refs = []
    if not isinstance(artifacts, list):
        artifacts = []
    if not isinstance(all_refs, list):
        all_refs = []
    if not isinstance(all_artifacts, list):
        all_artifacts = []
    owned_refs = [r for r in refs if isinstance(r, dict)
                  and r.get("subject") == FOCUS_RULE]
    indexed_refs = [r for r in all_refs if isinstance(r, dict)
                    and r.get("subject") == FOCUS_RULE]
    indexed_artifacts = [a for a in all_artifacts if isinstance(a, dict)
                         and a.get("path") == REQUIRED_VALIDATION_SOURCE]
    current_artifacts = [a for a in artifacts if isinstance(a, dict)
                         and a.get("path") == REQUIRED_VALIDATION_SOURCE]
    if (inspection.get("sourceStatus") != "indexed" or
            len(owned_refs) != 1 or len(indexed_refs) != 1 or
            owned_refs[0] != indexed_refs[0] or
            owned_refs[0].get("artifactPath") != REQUIRED_VALIDATION_SOURCE or
            owned_refs[0].get("role") != "definition"):
        ref_state = "missing-or-inconsistent"
        errors.append("required-rule-source-reference:" + ref_state)
    else:
        ref_state = "same-definition-reference"

    if (len(current_artifacts) != 1 or len(indexed_artifacts) != 1 or
            current_artifacts[0] != indexed_artifacts[0] or
            not isinstance(current_artifacts[0].get("fingerprint"), str) or
            not current_artifacts[0]["fingerprint"].startswith("sha256:")):
        artifact_state = "missing-or-mismatched"
        errors.append("required-rule-source-artifact:" + artifact_state)
    else:
        artifact_state = "same-indexed-artifact"
    state = {
        "scope": "one independently required public Rule and Relation",
        "rule": FOCUS_RULE,
        "relation": FOCUS_RELATION,
        "canonical": canonical_state,
        "relationState": relation_state,
        "definitionReference": ref_state,
        "sourceArtifact": artifact_state,
        "status": "bounded-present" if not errors else "blocked",
        "allFunctionBusinessDependenciesClosed": False,
    }
    return state, sorted(set(errors))



def bounded_function_slice(docs: dict[str, dict]) -> tuple[dict, list[str]]:
    """P-1b1: one independent declaration obligation, from existing CLI only.

    The Application uses *namespace sales*, not a specific executable
    Capability. A matching definition cannot make the Function invocable.
    """
    errors: list[str] = []
    catalog = docs.get("catalog", {}).get("result", {})
    entries = catalog.get("entries", []) if isinstance(catalog, dict) else []
    if not isinstance(entries, list):
        entries = []
    managed = docs.get("workspace-files", {}).get("result", {}).get("managedFiles", [])
    if not isinstance(managed, list):
        managed = []
    allidx = docs.get("source-index-all", {}).get("result", {})
    allidx = allidx if isinstance(allidx, dict) else {}
    allrefs = allidx.get("references", [])
    allartifacts = allidx.get("artifacts", [])
    if not isinstance(allrefs, list):
        allrefs = []
        errors.append("function-index:references-not-observed")
    if not isinstance(allartifacts, list):
        allartifacts = []
        errors.append("function-index:artifacts-not-observed")
    states = {}
    for key, subject, required_path in (
            ("capability", FUNCTION_SUBJECT, REQUIRED_SOURCE),
            ("application", APPLICATION_SUBJECT, APPLICATION_SOURCE)):
        matches = [e for e in entries if isinstance(e, dict)
                   and e.get("subject") == subject]
        if not matches:
            subject_state = "missing"
        elif len(matches) != 1 or matches[0].get("ambiguous") is True:
            subject_state = "ambiguous"
        elif matches[0].get("sourceStatus") != "indexed":
            subject_state = "not-indexed"
        else:
            subject_state = "cataloged"
        if subject_state != "cataloged":
            errors.append(f"function-{key}-catalog:{subject_state}")

        owned = [e for e in managed if isinstance(e, dict)
                 and e.get("path") == required_path]
        if len(owned) != 1:
            source_state = "missing-or-ambiguous"
        elif owned[0].get("readByLoader") is not True:
            source_state = "not-loaded"
        else:
            source_state = "loaded"
        if source_state != "loaded":
            errors.append(f"function-{key}-source:{source_state}")

        result = docs.get("inspect-" + key, {}).get("result", {})
        result = result if isinstance(result, dict) else {}
        canonical = result.get("canonical")
        canonical = canonical if isinstance(canonical, dict) else {}
        source = result.get("sources")
        source = source if isinstance(source, dict) else {}
        ref = source.get("references", [])
        art = source.get("artifacts", [])
        if not isinstance(ref, list):
            ref = []
        if not isinstance(art, list):
            art = []
        indexed_refs = [x for x in allrefs if isinstance(x, dict)
                        and x.get("subject") == subject]
        inspected_refs = [x for x in ref if isinstance(x, dict)
                          and x.get("subject") == subject]
        matching_indexed_artifacts = [x for x in allartifacts if isinstance(x, dict)
                                     and x.get("path") == required_path]
        matching_inspected_artifacts = [x for x in art if isinstance(x, dict)
                                       and x.get("path") == required_path]
        if (result.get("subject") != subject or result.get("sourceStatus") != "indexed"
                or len(indexed_refs) != 1 or len(inspected_refs) != 1
                or indexed_refs[0] != inspected_refs[0]
                or indexed_refs[0].get("role") != "definition"
                or indexed_refs[0].get("artifactPath") != required_path
                or len(matching_indexed_artifacts) != 1
                or len(matching_inspected_artifacts) != 1
                or matching_indexed_artifacts[0] != matching_inspected_artifacts[0]
                or not isinstance(matching_indexed_artifacts[0].get("fingerprint"), str)
                or not matching_indexed_artifacts[0]["fingerprint"].startswith("sha256:")):
            anchor_state = "missing-or-conflicting"
            errors.append(f"function-{key}-definition:{anchor_state}")
        else:
            anchor_state = "same-indexed-definition"
        states[key] = {"catalog": subject_state, "source": source_state,
                       "definition": anchor_state, "canonical": canonical}

    capability = states["capability"]["canonical"]
    if (capability.get("id") != FUNCTION_SUBJECT["id"]
            or capability.get("inputs") != [FUNCTION_INPUT]
            or capability.get("output") != FUNCTION_OUTPUT):
        contract_state = "not-matched"
        errors.append("function-signature:" + contract_state)
    else:
        contract_state = "matched"

    app = states["application"]["canonical"]
    namespace_uses = app.get("semanticUses")
    declared_validations = app.get("validations")
    dbs = app.get("databases")
    if (app.get("id") != APPLICATION_SUBJECT["id"]
            or not isinstance(namespace_uses, list)
            or namespace_uses.count("sales") != 1
            or not isinstance(declared_validations, list)
            or any(declared_validations.count(v) != 1 for v in FUNCTION_VALIDATIONS)):
        use_state = "missing-or-conflicting"
        errors.append("function-application-uses:" + use_state)
    else:
        use_state = "matched"
    if not isinstance(dbs, list):
        dbs = []
    bound_dbs = [d for d in dbs if isinstance(d, dict)
                 and d.get("alias") == "sales"]
    if (len(bound_dbs) != 1
            or bound_dbs[0].get("target") != "sales_core"
            or bound_dbs[0].get("access") != "read-write"):
        binding_state = "missing-or-drifted"
        errors.append("function-application-database:" + binding_state)
    else:
        binding_state = "declared-only"

    return {
        "scope": "fixed-reference:sales.order.submit declaration and order_entry namespace use",
        "capability": FUNCTION_SUBJECT,
        "application": APPLICATION_SUBJECT,
        "sourcePaths": [REQUIRED_SOURCE, APPLICATION_SOURCE],
        "capabilitySource": {k: states["capability"][k] for k in ("catalog", "source", "definition")},
        "applicationSource": {k: states["application"][k] for k in ("catalog", "source", "definition")},
        "signature": contract_state,
        "namespaceAndValidationUses": use_state,
        "databaseRequirement": binding_state,
        "declarationStatus": "bounded-present" if not errors else "blocked",
        "functionInvocationBinding": "not-observed",
        "queryOrProviderRead": "not-observed",
        "hostGrantAndEffect": "not-observed",
        "functionQualifiedForTestGeneration": False,
    }, sorted(set(errors))


def source_currency_slice(docs: dict[str, dict]) -> tuple[dict, list[str], list[str]]:
    """Which Source revisions does the bounded Function slice actually rest on?

    worldFingerprint/sourceIndexFingerprint identify only the consumer World
    and the consumer-managed Source Index. A Subject the Catalog reports as
    'not-indexed' has its Source outside that Index. The 'prelude' flag marks
    the built-in set bound to the CLI build; any other not-indexed Subject is
    resolved from Source this Consumer cannot identify. The existing read-only
    CLI projects no Import, Provider Source occurrence or Provider Manifest
    identity, so such a dependency is 'unknown': never current, never absent,
    never zero. Returns (slice, blockers, unknowns). Only a contradiction
    between two received replies is a blocker.

    The closure is one fixed hop (Function input/output Data and the required
    Relation target: their attribute types and Relation targets). It is not a
    transitive, Workspace-wide or Provider-wide closure.
    """
    blockers: list[str] = []
    unknowns: list[str] = []
    entries = docs.get("catalog", {}).get("result", {}).get("entries")
    if not isinstance(entries, list):
        entries = []
        unknowns.append("currency-catalog:not-observed")
    by_id: dict[str, list[dict]] = {}
    for entry in entries:
        subject = entry.get("subject") if isinstance(entry, dict) else None
        if isinstance(subject, dict) and isinstance(subject.get("id"), str):
            by_id.setdefault(subject["id"], []).append(entry)

    referenced: dict[str, set[str]] = {}
    for data_id in CLOSURE_DATA:
        parent = {"kind": "semantic-data", "id": data_id}
        attributes = {}
        for entry in entries:
            if not isinstance(entry, dict) or entry.get("parent") != parent:
                continue
            subject = entry.get("subject", {})
            detail = entry.get("detail")
            if subject.get("kind") not in ("semantic-attribute", "semantic-relation"):
                continue
            if not isinstance(detail, str) or not detail:
                unknowns.append(f"currency-reference:{subject.get('id')}:target-not-observed")
                continue
            if subject.get("kind") == "semantic-attribute":
                attributes[subject.get("id")] = detail
            referenced.setdefault(detail, set()).add(
                "semantic-type" if subject.get("kind") == "semantic-attribute" else "semantic-data")
        if not attributes:
            # No attribute seen is a missing observation, not "no dependency".
            unknowns.append(f"currency-attributes:{data_id}:not-observed")
        if data_id == TARGET["id"]:
            canonical = docs.get("inspect", {}).get("result", {}).get("canonical")
            listed = canonical.get("attributes") if isinstance(canonical, dict) else None
            if not isinstance(listed, list) or not listed:
                unknowns.append(f"currency-attributes:{data_id}:canonical-not-observed")
            else:
                inspected = {x.get("id"): x.get("type") for x in listed if isinstance(x, dict)}
                if inspected != attributes:
                    blockers.append(f"currency-attributes:{data_id}:catalog-inspect-conflict")

    classes: dict[str, list[str]] = {"consumer-indexed": [], "prelude": [],
                                     "outside-consumer-source-index": [], "not-observed": []}
    for ref_id in sorted(referenced):
        found = [e for e in by_id.get(ref_id, [])
                 if e.get("subject", {}).get("kind") in referenced[ref_id]]
        if len(found) != 1 or found[0].get("ambiguous") is True:
            kind = "not-observed"
        elif found[0].get("sourceStatus") == "indexed":
            kind = "consumer-indexed"
        elif found[0].get("sourceStatus") == "not-indexed" and found[0].get("prelude") is True:
            kind = "prelude"
        else:
            kind = "outside-consumer-source-index"
        classes[kind].append(ref_id)
    unknowns.extend("currency-dependency:" + x + ":not-observed" for x in classes["not-observed"])
    unknowns.extend("currency-dependency:" + x + ":provider-source-occurrence-unknown"
                    for x in classes["outside-consumer-source-index"])
    unknowns = sorted(set(unknowns))
    provider = "not-required-by-bounded-closure" if not unknowns and not blockers else "unknown"
    return {
        "scope": "fixed-reference:sales.order.submit one-hop Data attribute-type and relation-target closure",
        "closureData": list(CLOSURE_DATA),
        "dependencies": classes,
        "importedProviderSourceOccurrence": provider,
        "importedProviderManifestIdentity": provider,
        # Limits that no reply of the existing read-only CLI can lift:
        "workspaceImportDeclarations": "not-observable-by-existing-cli",
        "transitiveClosureBeyondOneHop": "not-observed",
        "atomicMultiSourceSnapshot": "not-observed",
        "preludeRevision": "bound-to-cli-build-not-to-source-fingerprint",
    }, sorted(set(blockers)), unknowns


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
            docs.get("inspect-validation", {}).get("result", {}).get(key),
            docs.get("inspect-capability", {}).get("result", {}).get(key),
            docs.get("inspect-application", {}).get("result", {}).get(key),
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

    rule_slice, rule_blockers = required_validation_slice(docs)
    focus_slice, focus_blockers = focused_structure_slice(docs)
    function_slice, function_blockers = bounded_function_slice(docs)
    blockers.extend(function_blockers)
    blockers.extend(rule_blockers)
    blockers.extend(focus_blockers)
    currency_slice, currency_blockers, unknowns = source_currency_slice(docs)
    blockers.extend(currency_blockers)
    blockers = sorted(set(blockers))
    # 'unknown' is its own verdict: the required evidence was not observable.
    # It is neither 'blocked' (a contradicted/missing requirement) nor a
    # bounded pass, and it publishes no fingerprint a later Consumer could pin.
    verdict = "blocked" if blockers else "unknown" if unknowns else "partial"
    currency_slice["consumerManagedSources"] = (
        "bound-to-observed-source-index-fingerprint" if verdict == "partial" else "not-bound")
    currency_slice["currentnessClaim"] = {
        "partial": "consumer-revision-only; re-observe both fingerprints before reuse",
        "unknown": "unknown; consumer fingerprints do not cover the required Source",
        "blocked": "none",
    }[verdict]
    return {
        "schema": "tei.reference.design-context-readiness/v0",
        "issue": "hat47x/tei-platform#3262",
        "subject": TARGET,
        "requiredSource": REQUIRED_SOURCE,
        "requiredValidationSlice": rule_slice,
        "focusedStructureSlice": focus_slice,
        "boundedFunctionSlice": function_slice,
        "observations": {
            "localReplies": f"{len(docs)}/{len(COMMANDS)}",
            "requiredSourceRead": source_state,
            "requiredSubject": subject_state,
            "fingerprintConsistency": consistency,
            "validationDiagnosticCount": diagnostic_count,
            "observedFingerprints": fingerprints if verdict == "partial" else {},
        },
        "sourceCurrency": currency_slice,
        "boundedReference": verdict,
        "testExecutionAdmission": {"partial": "exploration-only", "unknown": "withheld-unknown",
                                   "blocked": "blocked"}[verdict],
        "unknowns": unknowns,
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



def required_validation_selftest(docs: dict[str, dict], baseline: dict) -> dict[str, str]:
    """TT-56: keep Reply parseable and fingerprints *unchanged*.

    Mutations are isolated in-memory copies of already received CLI responses.
    They neither modify Source nor claim an actual external Tool delivery test.
    """
    if baseline["requiredValidationSlice"]["status"] != "bounded-present":
        raise AssertionError("reference Rule slice not present in genuine CLI result")
    tests = {}
    required = {"kind": "semantic-validation",
                "id": "sales.order_line.line_number_positive"}

    def check(name: str, mutated: dict, needle: str) -> None:
        report = assess(mutated)
        if (report["boundedReference"] != "blocked"
                or report["testExecutionAdmission"] != "blocked"
                or report["qualifiedTestGeneration"] is not False
                or report["requiredValidationSlice"]["status"] != "blocked"):
            raise AssertionError(f"{name}: false qualified {report['blockers']}")
        if not any(needle in reason for reason in report["blockers"]):
            raise AssertionError(f"{name}: wrong blocker {report['blockers']}")
        # We intentionally did not touch the CLI's apparently matching
        # World/Index fingerprints; their equality never proves completeness.
        original = docs["catalog"]["result"]["worldFingerprint"]
        if mutated["catalog"]["result"]["worldFingerprint"] != original:
            raise AssertionError("negative must preserve declared fingerprint")
        tests[name] = "blocked"

    source = copy.deepcopy(docs)
    source["catalog"]["result"]["entries"] = [
        e for e in source["catalog"]["result"]["entries"] if e.get("subject") != required
    ]
    check("TT56-catalog-sibling-omitted", source, "required-rule-catalog")

    source = copy.deepcopy(docs)
    entry = next(x for x in source["catalog"]["result"]["entries"]
                 if x.get("subject") == required)
    entry["subject"]["kind"] = "semantic-data"
    check("TT56-kind-misclassified", source, "required-rule-catalog")

    source = copy.deepcopy(docs)
    source["source-index-all"]["result"]["references"] = [
        x for x in source["source-index-all"]["result"]["references"]
        if x.get("subject") != required
    ]
    check("TT56-index-definition-omitted", source, "required-rule-index")

    source = copy.deepcopy(docs)
    for ref in source["source-index-all"]["result"]["references"]:
        if ref.get("subject") == required:
            ref["role"] = "derived"
    check("TT56-wrong-source-role", source, "required-rule-index")

    source = copy.deepcopy(docs)
    for ref in source["source-index-all"]["result"]["references"]:
        if ref.get("subject") == required:
            ref["artifactPath"] = "semantic/sales/unrelated.tei"
    check("TT56-wrong-artifact-binding", source, "required-rule-index")

    source = copy.deepcopy(docs)
    duplicate = next(x for x in source["catalog"]["result"]["entries"]
                     if x.get("subject") == required)
    source["catalog"]["result"]["entries"].append(copy.deepcopy(duplicate))
    check("TT56-duplicate-required-rule", source, "required-rule-catalog")

    source = copy.deepcopy(docs)
    for entry in source["workspace-files"]["result"]["managedFiles"]:
        if entry.get("path") == REQUIRED_VALIDATION_SOURCE:
            entry["readByLoader"] = False
    check("TT56-rule-source-not-loaded", source, "required-rule-source")

    # Negative control for over-rejection: removing an *unrequired* Catalog
    # item must not invalidate this explicitly bounded three-Rule obligation.
    source = copy.deepcopy(docs)
    before = len(source["catalog"]["result"]["entries"])
    source["catalog"]["result"]["entries"] = [
        x for x in source["catalog"]["result"]["entries"]
        # std.binary: not an attribute type of the Function's one-hop closure
        # (std.money is, since the Source currency slice).
        if x.get("subject") != {"kind": "semantic-type", "id": "std.binary"}
    ]
    if len(source["catalog"]["result"]["entries"]) >= before:
        raise AssertionError("independent non-required subject not in fixture")
    if assess(source)["boundedReference"] != "partial":
        raise AssertionError("TT56 over-rejected unrelated subject omission")
    tests["TT56-outside-required-scope-remains-partial"] = "bounded-scope-maintained"
    return tests

def focused_structure_selftest(docs: dict[str, dict], baseline: dict) -> dict[str, str]:
    """New discrimination not already covered by the three-Rule/Index tests."""
    if baseline["focusedStructureSlice"]["status"] != "bounded-present":
        raise AssertionError("real read-only Rule/Relation structure was not matched")
    controls = {}

    def check(name: str, change, reason: str, structure_blocked: bool = True) -> None:
        mutant = copy.deepcopy(docs)
        change(mutant)
        report = assess(mutant)
        if (report["boundedReference"] != "blocked" or
                report["testExecutionAdmission"] != "blocked" or
                report["qualifiedTestGeneration"] is not False or
                not any(reason in x for x in report["blockers"])):
            raise AssertionError(f"{name}: false qualified {report['blockers']}")
        if structure_blocked and report["focusedStructureSlice"]["status"] != "blocked":
            raise AssertionError(f"{name}: structure erroneously accepted")
        # Mutation intentionally leaves all declared World/SourceIndex
        # revision strings alone except the explicit mixed-revision control.
        controls[name] = "rejected"

    def without_relation(d):
        e = d["catalog"]["result"]["entries"]
        d["catalog"]["result"]["entries"] = [
            v for v in e if v.get("subject") != FOCUS_RELATION
        ]

    check("TT56-related-relation-omitted", without_relation, "required-relation:missing")

    def duplicate_relation(d):
        e = d["catalog"]["result"]["entries"]
        target = next(v for v in e if v.get("subject") == FOCUS_RELATION)
        e.append(copy.deepcopy(target))

    check("TT56-related-relation-ambiguous", duplicate_relation, "required-relation:ambiguous")

    def unrelated_parent(d):
        v = next(x for x in d["catalog"]["result"]["entries"]
                 if x.get("subject") == FOCUS_RELATION)
        v["parent"] = {"kind": "semantic-data", "id": "sales.other"}

    check("TT56-related-relation-parent-changed", unrelated_parent, "required-relation:parent-mismatch")

    def wrong_attribute(d):
        d["inspect-validation"]["result"]["canonical"]["attribute"] = "sales.order_line.other"

    check("Rule-canonical-attribute-changed", wrong_attribute, "required-rule-canonical:")

    def wrong_kind(d):
        d["inspect-validation"]["result"]["canonical"]["kind"] = "relation-unique"

    check("Rule-canonical-kind-changed", wrong_kind, "required-rule-canonical:")

    def missing_inspected_ref(d):
        d["inspect-validation"]["result"]["sources"]["references"] = []

    check("Rule-inspection-definition-reference-omitted", missing_inspected_ref,
          "required-rule-source-reference:")

    def mismatched_artifact(d):
        for x in d["inspect-validation"]["result"]["sources"]["artifacts"]:
            if x.get("path") == REQUIRED_VALIDATION_SOURCE:
                x["fingerprint"] = "sha256:" + "0" * 64

    check("Rule-inspection-SourceIndex-artifact-differs", mismatched_artifact,
          "required-rule-source-artifact:")

    def mixed_inspect_revision(d):
        d["inspect-validation"]["result"]["sourceIndexFingerprint"] = "stale-index-only-rule"

    check("Rule-inspection-mixed-revision", mixed_inspect_revision,
          "fingerprint:stale", structure_blocked=False)
    return controls

def function_slice_selftest(docs: dict[str, dict], baseline: dict) -> dict[str, str]:
    """Same source fingerprint strings can accompany one missing Function facet."""
    if baseline["boundedFunctionSlice"]["declarationStatus"] != "bounded-present":
        raise AssertionError("public Function prerequisite was not observed by real CLI")
    controls: dict[str, str] = {}

    def reject(name: str, change, expected: str) -> None:
        mutant = copy.deepcopy(docs)
        original = {x: docs["catalog"]["result"][x] for x in
                    ("worldFingerprint", "sourceIndexFingerprint")}
        change(mutant)
        if any(mutant["catalog"]["result"].get(x) != val
               for x, val in original.items()):
            raise AssertionError(name + ": must preserve declared fingerprints")
        observed = assess(mutant)
        if (observed["boundedReference"] != "blocked"
                or observed["boundedFunctionSlice"]["declarationStatus"] != "blocked"
                or observed["qualifiedTestGeneration"] is not False
                or not any(expected in error for error in observed["blockers"])):
            raise AssertionError(f"{name}: missing rejection: {observed['blockers']}")
        controls[name] = "blocked"

    def omit_capability(d):
        e = d["catalog"]["result"]["entries"]
        d["catalog"]["result"]["entries"] = [
            x for x in e if x.get("subject") != FUNCTION_SUBJECT]

    reject("TT-function-capability-only-omitted", omit_capability, "function-capability-catalog:missing")

    def omit_application(d):
        e = d["catalog"]["result"]["entries"]
        d["catalog"]["result"]["entries"] = [
            x for x in e if x.get("subject") != APPLICATION_SUBJECT]

    reject("TT-function-application-only-omitted", omit_application, "function-application-catalog:missing")

    def mismatch_input(d):
        d["inspect-capability"]["result"]["canonical"]["inputs"][0]["type"] = "sales.other"

    reject("TT-function-input-drift", mismatch_input, "function-signature:not-matched")

    def mismatch_output(d):
        d["inspect-capability"]["result"]["canonical"]["output"] = "sales.other"

    reject("TT-function-output-drift", mismatch_output, "function-signature:not-matched")

    def omit_namespace_use(d):
        a = d["inspect-application"]["result"]["canonical"]
        a["semanticUses"] = [x for x in a["semanticUses"] if x != "sales"]

    reject("TT-function-namespace-use-omitted", omit_namespace_use, "function-application-uses:")

    def omit_validation_use(d):
        a = d["inspect-application"]["result"]["canonical"]
        a["validations"] = [x for x in a["validations"]
                            if x != "sales.order_line.line_number_positive"]

    reject("TT-function-rule-use-omitted", omit_validation_use, "function-application-uses:")

    def mismatch_db(d):
        a = d["inspect-application"]["result"]["canonical"]
        next(x for x in a["databases"] if x["alias"] == "sales")["target"] = "master_data"

    reject("TT-function-declared-database-changed", mismatch_db, "function-application-database:")

    def omit_definition(d):
        d["inspect-capability"]["result"]["sources"]["references"] = []

    reject("TT-function-definition-reference-omitted", omit_definition, "function-capability-definition:")

    def unread_application(d):
        for e in d["workspace-files"]["result"]["managedFiles"]:
            if e.get("path") == APPLICATION_SOURCE:
                e["readByLoader"] = False

    reject("TT-function-application-source-unread", unread_application, "function-application-source:not-loaded")

    # Only bounded Required is measured: unrelated sibling omission remains
    # partial, and the absence cannot qualify a global Function census.
    control = copy.deepcopy(docs)
    original = control["catalog"]["result"]["entries"]
    control["catalog"]["result"]["entries"] = [
        x for x in original if x.get("subject") !=
        {"kind": "semantic-type", "id": "std.binary"}]  # outside the one-hop closure
    if len(control["catalog"]["result"]["entries"]) == len(original):
        raise AssertionError("nonrequired prelude control unavailable")
    observed = assess(control)
    if (observed["boundedReference"] != "partial"
            or observed["boundedFunctionSlice"]["declarationStatus"] != "bounded-present"
            or observed["qualifiedTestGeneration"] is not False):
        raise AssertionError("unrelated subject erroneously blocked bounded Function")
    controls["TT-function-unrequired-prelude-omitted"] = "bounded-partial-only"
    return controls


def source_currency_selftest(docs: dict[str, dict], baseline: dict) -> dict[str, str]:
    """Unchanged consumer fingerprints must not hide an unidentified Source.

    Every variant keeps worldFingerprint/sourceIndexFingerprint and every
    reply 'ok'. The independent Expected is the verdict class, fixed here and
    not derived from assess().
    """
    current = baseline["sourceCurrency"]
    if (baseline["boundedReference"] != "partial" or baseline["unknowns"]
            or current["importedProviderSourceOccurrence"] != "not-required-by-bounded-closure"
            or current["dependencies"]["outside-consumer-source-index"]
            or current["dependencies"]["not-observed"]
            or current["workspaceImportDeclarations"] != "not-observable-by-existing-cli"):
        raise AssertionError("public Reference currency baseline was not observed by real CLI")
    controls: dict[str, str] = {}
    pinned = {x: docs["catalog"]["result"][x] for x in ("worldFingerprint", "sourceIndexFingerprint")}

    def entries(d):
        return d["catalog"]["result"]["entries"]

    def entry(d, kind, ident):
        found = [e for e in entries(d) if e.get("subject") == {"kind": kind, "id": ident}]
        if len(found) != 1:
            raise AssertionError(f"control prerequisite unavailable: {kind}:{ident}")
        return found[0]

    def check(name: str, change, want: str, fragment: str) -> None:
        mutant = copy.deepcopy(docs)
        change(mutant)
        if any(mutant["catalog"]["result"].get(k) != v for k, v in pinned.items()):
            raise AssertionError(name + ": control must keep consumer fingerprints")
        observed = assess(mutant)
        listed = observed["unknowns"] if want == "unknown" else observed["blockers"]
        if (observed["boundedReference"] != want
                or observed["testExecutionAdmission"] == "exploration-only"
                or observed["qualifiedTestGeneration"] is not False
                or observed["observations"]["observedFingerprints"]
                or observed["sourceCurrency"]["importedProviderSourceOccurrence"] != "unknown"
                or observed["sourceCurrency"]["importedProviderManifestIdentity"] != "unknown"
                or observed["sourceCurrency"]["consumerManagedSources"] != "not-bound"
                or (want == "unknown" and observed["blockers"])
                or not any(fragment in x for x in listed)):
            raise AssertionError(f"{name}: expected {want}/{fragment}, got "
                                 f"{observed['boundedReference']} {observed['blockers']} {observed['unknowns']}")
        controls[name] = want

    def imported_type(d):
        # Same shape as the real CLI reply for an imported Namespace type.
        entry(d, "semantic-attribute", "sales.order.external_id")["detail"] = "os.process_id"
        for attribute in d["inspect"]["result"]["canonical"]["attributes"]:
            if attribute["id"] == "sales.order.external_id":
                attribute["type"] = "os.process_id"
        entries(d).append({"subject": {"kind": "semantic-type", "id": "os.process_id"},
                           "parent": {"kind": "semantic", "id": "os"}, "symbol": "ProcessId",
                           "detail": "alias", "sourceStatus": "not-indexed"})

    check("TT-currency-imported-type-in-closure", imported_type, "unknown",
          "os.process_id:provider-source-occurrence-unknown")

    def relation_target_outside(d):
        entry(d, "semantic-data", "customer.customer")["sourceStatus"] = "not-indexed"

    check("TT-currency-relation-target-outside-index", relation_target_outside, "unknown",
          "customer.customer:provider-source-occurrence-unknown")

    def prelude_unmarked(d):
        entry(d, "semantic-type", "std.money").pop("prelude")

    check("TT-currency-not-indexed-without-prelude-mark", prelude_unmarked, "unknown",
          "std.money:provider-source-occurrence-unknown")

    def dependency_omitted(d):
        d["catalog"]["result"]["entries"] = [
            e for e in entries(d) if e.get("subject") != {"kind": "semantic-type", "id": "std.money"}]

    check("TT-currency-dependency-entry-omitted", dependency_omitted, "unknown",
          "std.money:not-observed")

    def attributes_omitted(d):
        parent = {"kind": "semantic-data", "id": "sales.order_line"}
        d["catalog"]["result"]["entries"] = [
            e for e in entries(d) if not (e.get("parent") == parent
                                          and e["subject"]["kind"] == "semantic-attribute")]

    # Zero listed attributes is an absent observation, not zero dependencies.
    check("TT-currency-attributes-not-listed", attributes_omitted, "unknown",
          "currency-attributes:sales.order_line:not-observed")

    def canonical_attributes_empty(d):
        d["inspect"]["result"]["canonical"]["attributes"] = []

    check("TT-currency-canonical-attributes-empty", canonical_attributes_empty, "unknown",
          "currency-attributes:sales.order:canonical-not-observed")

    def reference_target_dropped(d):
        entry(d, "semantic-attribute", "sales.order_line.unit_price").pop("detail")

    check("TT-currency-attribute-type-not-projected", reference_target_dropped, "unknown",
          "sales.order_line.unit_price:target-not-observed")

    def replies_disagree(d):
        entry(d, "semantic-attribute", "sales.order.total")["detail"] = "std.decimal"

    check("TT-currency-catalog-inspect-type-conflict", replies_disagree, "blocked",
          "currency-attributes:sales.order:catalog-inspect-conflict")
    return controls


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
    report["requiredValidationControls"] = required_validation_selftest(docs, baseline)
    report["focusedStructureControls"] = focused_structure_selftest(docs, baseline)
    report["boundedFunctionControls"] = function_slice_selftest(docs, baseline)
    report["sourceCurrencyControls"] = source_currency_selftest(docs, baseline)
    report["transportByteControls"] = transport_selftest(args.folder)
    report["localReplyEvidence"] = byte_evidence
    args.out.parent.mkdir(parents=True, exist_ok=True)
    args.out.write_text(json.dumps(report, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    print(f"PASS: bounded Design Context {baseline['boundedReference']}; "
          f"{len(controls)} base controls + {len(report['additionalManifestControls'])} manifest "
          f"+ {len(report['transportByteControls'])} byte-reply rejection controls "
          f"+ {len(report['requiredValidationControls'])} bounded Rule-slice checks "
          f"+ {len(report['focusedStructureControls'])} Rule/Relation Canonical checks "
          f"+ {len(report['boundedFunctionControls'])} bounded Function checks "
          f"+ {len(report['sourceCurrencyControls'])} Source currency checks; "
          "test generation qualification withheld")


if __name__ == "__main__":
    main()
