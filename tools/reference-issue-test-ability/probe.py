#!/usr/bin/env python3
"""Bounded test-ability probe for the existing TEI Driver Scenario CLI.

This is a test-only, independent property checker, NOT a second Scenario runner
or Domain Authority. Positive/negative controls use the same real TEI binary.
Only synthetic reference fixtures are copied/mutated in a temporary directory.
"""
from __future__ import annotations

import argparse
import copy
from decimal import Decimal
import hashlib
import json
from pathlib import Path
import re
import shutil
import subprocess
import tempfile

SCENARIO = Path("reference/driver-scenarios/order-entry-pattern-smoke.json")
WORKSPACE = "reference/workspace"
STEPS = {
    "browse-pattern", "record-pattern", "browse-synthetic",
    "validate-positive", "validate-denied", "observe-synthetic",
}
# Independent test-only expectation for the public six-step Reference.
# Never infer the expected Operation from the tested Scenario or CLI receipt.
# This is not an Owner-accepted business meaning or a new Scenario SSOT.
STEP_OPERATIONS = {
    "browse-pattern": "resolve-pattern",
    "record-pattern": "resolve-pattern",
    "browse-synthetic": "execute-browse-query",
    "validate-positive": "evaluate-validations",
    "validate-denied": "evaluate-validations",
    "observe-synthetic": "observe-synthetic-state",
}
QUANTITY = "sales.order_line.quantity_positive"
LINE_POSITIVE = "sales.order_line.line_number_positive"
LINE_UNIQUE = "sales.order.lines.line_number_unique"
RULES = {QUANTITY, LINE_POSITIVE, LINE_UNIQUE}


def assert_independent_step_operations(observed: dict[str, str]) -> None:
    """Reject a co-corrupted Scenario/receipt pair against a fixed Reference."""
    if observed != STEP_OPERATIONS:
        raise AssertionError("Scenario Step->Operation differs from independent obligation")


def digest(data: bytes) -> str:
    return hashlib.sha256(data).hexdigest()


def run(cli: Path, root: Path, *argv: str, success: bool = True) -> subprocess.CompletedProcess[str]:
    proc = subprocess.run(
        [str(cli), *map(str, argv)], cwd=root, capture_output=True,
        text=True, encoding="utf-8", errors="replace", timeout=90, check=False,
    )
    if success and (proc.returncode != 0 or proc.stderr.strip()):
        raise AssertionError(
            f"TEI {' '.join(map(str, argv[:3]))}: expected clean success, "
            f"exit={proc.returncode}, stderr={proc.stderr[-500:]}"
        )
    if not success and proc.returncode == 0:
        raise AssertionError(f"TEI {' '.join(map(str, argv[:3]))}: mutant survived (exit 0)")
    return proc


def as_json(proc: subprocess.CompletedProcess[str]) -> dict:
    obj = json.loads(proc.stdout)
    if not isinstance(obj, dict):
        raise AssertionError("TEI output is not a JSON object")
    return obj


def independent_rule_properties(fixture: dict) -> dict[str, str]:
    """Independent known-case rule properties, not read from Scenario expected."""
    numbers = fixture["numbers"]
    relation = fixture["relationKeys"]
    if len(relation) != 1 or relation[0]["relation"] != "sales.order.lines":
        raise AssertionError("unsupported test-only relation fixture")
    keys = relation[0]["keys"]
    return {
        QUANTITY: "satisfied" if Decimal(numbers["sales.order_line.quantity"]) > 0 else "not-satisfied",
        LINE_POSITIVE: "satisfied" if Decimal(numbers["sales.order_line.line_number"]) > 0 else "not-satisfied",
        LINE_UNIQUE: "satisfied" if len(set(keys)) == len(keys) else "not-satisfied",
    }


def expect_real_evidence(result: dict, step: str) -> dict:
    if result.get("executionMode") != "generated-pinned" or result.get("selectedStepCount") != 1:
        raise AssertionError(f"{step}: not a pinned, selected real-process call")
    evidence = result.get("evidence")
    if not isinstance(evidence, list) or len(evidence) != 1 or evidence[0].get("stepId") != step:
        raise AssertionError(f"{step}: wrong/missing evidence")
    if evidence[0].get("verified") is not True:
        raise AssertionError(f"{step}: actual step not verified")
    return evidence[0]


def negative(proc: subprocess.CompletedProcess[str], *diagnostic_fragments: str) -> None:
    if proc.returncode == 0:
        raise AssertionError("wrongly accepted negative test")
    diagnostic = proc.stderr + proc.stdout
    for fragment in diagnostic_fragments:
        if fragment not in diagnostic:
            raise AssertionError(
                f"negative test rejected for wrong reason; missing {fragment!r}: {diagnostic[-600:]}"
            )



def scoped_step_coverage(required: set[str], received: list[dict],
                         scenario_sha: str, actual_source_sha: str,
                         end_received: bool, expected_operations: dict[str, str]) -> dict:
    """Read-only scope projection over independently checked real CLI receipts.

    Absence in a partial or malformed inventory is never proof of an untested
    obligation. Presence in a complete pinned *Scenario* is still not proof
    of all Product requirements, semantic Rule coverage, or current Grant.
    Caller supplies the independent required step set and actual Source digest.
    """
    if scenario_sha != actual_source_sha:
        return {
            "inventory": "source-mismatch",
            "localObserved": [],
            "notObserved": sorted(required),
            "untestedConfirmed": [],
            "productCoverage": "unknown",
            "claimCeiling": "unqualified",
        }
    indexed: dict[str, bool] = {}
    for row in received:
        if not isinstance(row, dict) or not isinstance(row.get("stepId"), str):
            return {
                "inventory": "invalid-receipt",
                "localObserved": [],
                "notObserved": sorted(required),
                "untestedConfirmed": [],
                "productCoverage": "unknown",
                "claimCeiling": "unqualified",
            }
        key = row["stepId"]
        if key not in required or key in indexed:
            return {
                "inventory": "unexpected-or-duplicate-step",
                "localObserved": [],
                "notObserved": sorted(required),
                "untestedConfirmed": [],
                "productCoverage": "unknown",
                "claimCeiling": "unqualified",
            }
        if (expected_operations.get(key) is None
                or row.get("operation") != expected_operations[key]):
            return {
                "inventory": "operation-mismatch",
                "localObserved": [],
                "notObserved": sorted(required),
                "untestedConfirmed": [],
                "productCoverage": "unknown",
                "claimCeiling": "unqualified",
            }
        indexed[key] = row.get("verified") is True
    verified = {key for key, is_verified in indexed.items() if is_verified}
    complete = (end_received is True and set(indexed) == required
                and len(indexed) == len(required) and all(indexed.values()))
    return {
        "inventory": "complete-for-one-pinned-scenario" if complete else "partial",
        "localObserved": sorted(verified),
        "notObserved": sorted(required - verified),
        # No negative test-absence verdict: the Scenario step listing is not
        # a verified complete inventory of all possible tests for a Rule.
        "untestedConfirmed": [],
        "productCoverage": "unknown",
        "claimCeiling": "six-pinned-scenario-steps-only" if complete else "partial-only",
    }


def scoped_step_selftest(required: set[str], actual: dict[str, dict],
                         scenario_sha: str, observed_source_sha: str,
                         expected_operations: dict[str, str]
                         ) -> tuple[dict, dict[str, str]]:
    """TT-105/106 using validated real CLI observations plus copy-only controls."""
    # Carry the actual CLI evidence fields, never fabricate verified=True.
    receipts = [
        {"stepId": actual[key].get("stepId"),
         "verified": actual[key].get("verified"),
         "operation": actual[key].get("operation")}
        for key in sorted(actual)
    ]
    # Re-read Source after real CLI Step execution. This detects a persistent
    # local change between observations, not A->B->A or actual Runtime input.
    baseline = scoped_step_coverage(required, receipts, scenario_sha,
                                    observed_source_sha, True, expected_operations)
    if not (baseline["inventory"] == "complete-for-one-pinned-scenario"
            and baseline["localObserved"] == sorted(required)
            and baseline["productCoverage"] == "unknown"
            and not baseline["untestedConfirmed"]):
        raise AssertionError("actual six-step bounded observation not retained")
    controls: dict[str, str] = {}

    # TT-105: this synthetic *partial receipt* omits an independently
    # required step. It is NOT proof that the step was never tested.
    dropped = [entry for entry in receipts if entry["stepId"] != "validate-denied"]
    missing = scoped_step_coverage(required, dropped, scenario_sha, scenario_sha, False, expected_operations)
    if not (missing["inventory"] == "partial"
            and "validate-denied" in missing["notObserved"]
            and "validate-positive" in missing["localObserved"]
            and missing["untestedConfirmed"] == []):
        raise AssertionError("TT-105: partial inventory falsely established untested")
    controls["TT-105-partial-step-inventory"] = "absence-unqualified"

    # TT-106: a narrow positive witness survives lost unrelated inventory,
    # while all-requirement coverage remains unknown.
    if missing["productCoverage"] != "unknown":
        raise AssertionError("TT-106: local coverage leaked into whole Product")
    controls["TT-106-positive-local-but-global-unknown"] = "bounded-observation-maintained"

    # Receiving every step without an end witness does not close the report.
    incomplete = scoped_step_coverage(required, receipts, scenario_sha,
                                      scenario_sha, False, expected_operations)
    if incomplete["inventory"] != "partial" or incomplete["productCoverage"] != "unknown":
        raise AssertionError("complete-looking inventory missing end marker")
    controls["TT-105-complete-looking-without-end"] = "incomplete"

    # A fake end marker with a missing step is not a closed inventory.
    partial_with_end = scoped_step_coverage(required, dropped, scenario_sha,
                                             scenario_sha, True, expected_operations)
    if partial_with_end["inventory"] != "partial" or partial_with_end["untestedConfirmed"]:
        raise AssertionError("missing receipt was misclassified as complete")
    controls["TT-105-false-end-claim"] = "incomplete"

    changed = scoped_step_coverage(required, receipts, scenario_sha,
                                   "sha256:different-source", True, expected_operations)
    if changed["inventory"] != "source-mismatch" or changed["localObserved"]:
        raise AssertionError("foreign Source digest qualified an old receipt")
    controls["stale-source-receipts"] = "unqualified"

    duplicate = scoped_step_coverage(required, receipts + [receipts[0]],
                                     scenario_sha, scenario_sha, True, expected_operations)
    if duplicate["inventory"] != "unexpected-or-duplicate-step":
        raise AssertionError("duplicate Step receipt was accepted")
    controls["duplicate-receipt"] = "unqualified"

    unverified = [dict(x, verified=False if x["stepId"] == "validate-positive"
                       else x["verified"]) for x in receipts]
    verdict = scoped_step_coverage(required, unverified, scenario_sha,
                                   scenario_sha, True, expected_operations)
    if (verdict["inventory"] != "partial"
            or "validate-positive" in verdict["localObserved"]
            or verdict["untestedConfirmed"]):
        raise AssertionError("unverified receipt became qualified positive")
    controls["unverified-positive-receipt"] = "partial"
    # A different Operation with the same Step ID is not the requested target.
    wrong = [dict(entry) for entry in receipts]
    wrong_target = next(x for x in wrong if x["stepId"] == "validate-positive")
    wrong_target["operation"] = "resolve-pattern"
    wrong_verdict = scoped_step_coverage(required, wrong, scenario_sha,
                                         scenario_sha, True, expected_operations)
    if (wrong_verdict["inventory"] != "operation-mismatch"
            or wrong_verdict["localObserved"]
            or wrong_verdict["untestedConfirmed"]
            or wrong_verdict["productCoverage"] != "unknown"):
        raise AssertionError("same step id with wrong Operation was qualified")
    controls["wrong-operation-for-step-id"] = "unqualified"

    # TT-154: simultaneous corruption of the Scenario-derived expectation
    # and the received Operation can fool a self-consistency projection.
    # The *independent* fixed obligation detects it. This control does not
    # alter the real Scenario or claim an actual Product code mutation.
    coupled_receipts = [dict(entry) for entry in receipts]
    coupled = next(x for x in coupled_receipts if x["stepId"] == "validate-positive")
    coupled["operation"] = "resolve-pattern"
    coupled_expected = dict(expected_operations)
    coupled_expected["validate-positive"] = "resolve-pattern"
    self_consistent = scoped_step_coverage(
        required, coupled_receipts, scenario_sha, scenario_sha, True,
        coupled_expected,
    )
    if self_consistent["inventory"] != "complete-for-one-pinned-scenario":
        raise AssertionError("coupled-error control did not exercise self-consistency gap")
    try:
        assert_independent_step_operations(coupled_expected)
    except AssertionError as exc:
        if "independent obligation" not in str(exc):
            raise
    else:
        raise AssertionError("independent expectation accepted co-corrupted Operation")
    controls["TT-154-coupled-operation-corruption"] = "independent-obligation-rejected"
    return baseline, controls


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--root", type=Path, default=Path("."))
    parser.add_argument("--cli", type=Path, required=True)
    parser.add_argument("--out", type=Path, required=True)
    parser.add_argument("--coverage", type=Path, required=True)
    parser.add_argument("--context", type=Path, required=True)
    args = parser.parse_args()
    root = args.root.resolve()
    cli = args.cli.resolve()
    assert cli.is_file(), cli
    # The Design Context is only a bounded read-only observation, not an
    # Accepted Model or a consumer Grant. Any missing/mixed Source blocks even
    # this exploratory pinned Scenario test; full test generation remains off.
    design = json.loads(args.context.read_text(encoding="utf-8"))
    if (design.get("schema") != "tei.reference.design-context-readiness/v0"
            or design.get("testExecutionAdmission") != "exploration-only"
            or design.get("boundedReference") != "partial"
            or design.get("qualifiedTestGeneration") is not False
            or design.get("productMeaningAccepted") is not False
            or design.get("externalHarnessDeliveryComplete") is not None
            or design.get("blockers")
            or design.get("observations", {}).get("fingerprintConsistency") != "consistent"):
        raise AssertionError("unqualified Design Context; refuse bounded Scenario qualification")
    args.out.parent.mkdir(parents=True, exist_ok=True)

    original_scenario = root / SCENARIO
    source = original_scenario.read_bytes()
    scenario = json.loads(source)
    scenario_steps = {s["id"]: s for s in scenario["steps"]}
    if set(scenario_steps) != STEPS or len(scenario["steps"]) != len(STEPS):
        raise AssertionError("required Scenario step obligation absent, ambiguous, or extra")
    # The Step ID and requested Operation must independently match the public
    # Reference obligation. Otherwise Scenario and CLI can agree on a wrong
    # Operation and still appear to constitute a covered test.
    assert_independent_step_operations(
        {key: item.get("operation") for key, item in scenario_steps.items()}
    )

    fixture_dir = original_scenario.parent / "fixtures"
    valid = json.loads((fixture_dir / "order_valid.json").read_text(encoding="utf-8"))
    denied = json.loads((fixture_dir / "order_invalid.json").read_text(encoding="utf-8"))
    # These expectations are fixed by the test obligation, NOT copied from
    # the stored Scenario. They also catch an accidentally changed input set.
    positive_expected = {QUANTITY: "satisfied", LINE_POSITIVE: "satisfied", LINE_UNIQUE: "satisfied"}
    denied_expected = {QUANTITY: "not-satisfied", LINE_POSITIVE: "satisfied", LINE_UNIQUE: "satisfied"}
    assert independent_rule_properties(valid) == positive_expected
    assert independent_rule_properties(denied) == denied_expected
    for step, expected in (("validate-positive", positive_expected), ("validate-denied", denied_expected)):
        if scenario_steps[step]["expectValidations"] != expected:
            raise AssertionError(f"{step}: Scenario expected differs from independent fixed rule obligation")

    static = as_json(run(cli, root, "scenario", "verify", "-scenario", str(SCENARIO),
                         "-workspace", WORKSPACE))
    assert static["executionMode"] == "static-verification"
    assert static["selectedStepCount"] == len(STEPS)
    generated = run(cli, root, "scenario", "generate", "-scenario", str(SCENARIO),
                    "-workspace", WORKSPACE)
    plan = as_json(generated)
    commands = plan["commands"]
    assert isinstance(commands, list) and len(commands) == len(STEPS)
    assert {entry["stepId"] for entry in commands} == STEPS
    with tempfile.TemporaryDirectory(prefix="tei-test-ability-") as temporary:
        temp = Path(temporary)
        plan_path = temp / "plan.json"
        plan_path.write_text(generated.stdout, encoding="utf-8")
        checked = as_json(run(cli, root, "scenario", "check", "-scenario", str(SCENARIO),
                              "-workspace", WORKSPACE, "-plan", str(plan_path)))
        assert checked.get("planMatches") is True
        measured = {}
        for command in commands:
            step = command["stepId"]
            argv = command["argv"]
            if not isinstance(argv, list) or len(argv) < 4 or argv[:2] != ["tei", "scenario"]:
                raise AssertionError(f"{step}: missing real generated CLI argv")
            payload = as_json(run(cli, root, *argv[1:]))
            ev = expect_real_evidence(payload, step)
            if ev.get("operation") != STEP_OPERATIONS[step]:
                raise AssertionError(f"{step}: actual CLI Operation differs from independent obligation")
            if step == "validate-positive" and ev.get("validationResults") != positive_expected:
                raise AssertionError("positive property mismatch in actual CLI")
            if step == "validate-denied" and ev.get("validationResults") != denied_expected:
                raise AssertionError("business rejection lost or mislabeled as execution error")
            if step == "browse-synthetic" and (
                ev.get("rowCount") != 1 or ev.get("evidenceNature") != "synthetic-browse-query-executed"
            ):
                raise AssertionError("synthetic query execution not observed")
            if step == "observe-synthetic" and (
                ev.get("observationResult") != "pending"
                or ev.get("evidenceNature") != "synthetic-independent-observation-read"
            ):
                raise AssertionError("synthetic Observation not independently read")
            measured[step] = {
                "stepId": ev["stepId"],
                "operation": ev["operation"],
                "verified": ev["verified"],
                "nature": ev["evidenceNature"],
            }

        # Negative N1: actual canonical relation-unique evaluator must reject
        # duplicated keys while the unchanged Scenario expects unique keys.
        copied = temp / "mutation-duplicate"
        shutil.copytree(original_scenario.parent, copied)
        mutated_fixture = copied / "fixtures/order_valid.json"
        data = json.loads(mutated_fixture.read_text(encoding="utf-8"))
        data["relationKeys"][0]["keys"] = ["1", "1"]
        assert independent_rule_properties(data)[LINE_UNIQUE] == "not-satisfied"
        mutated_fixture.write_text(json.dumps(data), encoding="utf-8")
        bad_keys = run(cli, root, "scenario", "run", "-scenario",
                       str(copied / original_scenario.name), "-workspace", WORKSPACE,
                       "-step", "validate-positive", success=False)
        negative(bad_keys, LINE_UNIQUE, "result=not-satisfied")

        # Negative N2: a lying Scenario expected result cannot override
        # the real evaluator of the original zero-quantity fixture.
        copied = temp / "mutation-expected"
        shutil.copytree(original_scenario.parent, copied)
        candidate = copied / original_scenario.name
        data = json.loads(candidate.read_text(encoding="utf-8"))
        for step in data["steps"]:
            if step["id"] == "validate-denied":
                step["expectValidations"][QUANTITY] = "satisfied"
        candidate.write_text(json.dumps(data), encoding="utf-8")
        bad_expected = run(cli, root, "scenario", "run", "-scenario",
                           str(candidate), "-workspace", WORKSPACE,
                           "-step", "validate-denied", success=False)
        negative(bad_expected, QUANTITY, "result=not-satisfied")

        # Negative N3: a previously generated command must reject a changed
        # Fixture before running, independent of the exploratory N1 result.
        copied = temp / "mutation-stale-pin"
        shutil.copytree(original_scenario.parent, copied)
        saved_scenario = copied / original_scenario.name
        original_plan = as_json(run(cli, root, "scenario", "generate", "-scenario",
                                    str(saved_scenario), "-workspace", WORKSPACE))
        original_command = next(c["argv"] for c in original_plan["commands"]
                                if c["stepId"] == "validate-positive")
        p = copied / "fixtures/order_valid.json"
        f = json.loads(p.read_text(encoding="utf-8"))
        f["relationKeys"][0]["keys"] = ["1", "1"]
        p.write_text(json.dumps(f), encoding="utf-8")
        stale = run(cli, root, *original_command[1:], success=False)
        negative(stale, "Fixture fingerprint differs")

        # Negative N4: modified generated argv cannot be blessed by check.
        changed_plan = copy.deepcopy(plan)
        changed_plan["commands"][0]["argv"][0] = "fabricated-cli"
        tampered_path = temp / "tampered-plan.json"
        tampered_path.write_text(json.dumps(changed_plan), encoding="utf-8")
        changed = run(cli, root, "scenario", "check", "-scenario", str(SCENARIO),
                      "-workspace", WORKSPACE, "-plan", str(tampered_path), success=False)
        negative(changed, "stale or modified CLI plan")

        # Negative N7: change a real Canonical Source file *after* a plan was
        # generated, within an isolated Workspace copy, not an in-memory CLI
        # reply. A stale pinned World must not execute against this new model.
        isolated = temp / "mutation-world-source"
        shutil.copytree(root / WORKSPACE, isolated)
        src = isolated / "semantic/sales/order.tei"
        original_src = src.read_bytes()
        before_world = as_json(run(cli, root, "scenario", "generate", "-scenario",
                                   str(SCENARIO), "-workspace", str(isolated)))
        source_run = next(c["argv"] for c in before_world["commands"]
                          if c["stepId"] == "validate-positive")
        before_run = as_json(run(cli, root, *source_run[1:]))
        expect_real_evidence(before_run, "validate-positive")
        anchor = b"attribute externalId type std.text"
        if original_src.count(anchor) != 1:
            raise AssertionError("Source mutant anchor absent or ambiguous")
        changed_src = original_src.replace(anchor, b"attribute externalId type std.code")
        if changed_src == original_src:
            raise AssertionError("Source mutation was ineffective")
        src.write_bytes(changed_src)
        after_world = as_json(run(cli, root, "scenario", "generate", "-scenario",
                                  str(SCENARIO), "-workspace", str(isolated)))
        if before_world["worldSha256"] == after_world["worldSha256"]:
            raise AssertionError("real Source mutation changed no resolved World revision")
        stale_source = run(cli, root, *source_run[1:], success=False)
        negative(stale_source, "Canonical World fingerprint differs")
        src.write_bytes(original_src)
        restored = as_json(run(cli, root, *source_run[1:]))
        expect_real_evidence(restored, "validate-positive")
        # A -> B -> A is accepted by the content pin after restoration. This
        # proves current-content equality only, NOT history-free qualification,
        # Source observation identity or a stable Provider/Authority revision.
        if digest(src.read_bytes()) != digest(original_src):
            raise AssertionError("Source not restored after A-B-A test")

        # Negative N8: alter only the Plugin Lock *byte content* between
        # generation and execution. The Scenario pin catches stale Lock bytes,
        # but says nothing about actual Plugin Consumer load or authorization.
        isolated = temp / "mutation-plugin-lock"
        shutil.copytree(root / WORKSPACE, isolated)
        lock = isolated / ".tei/plugin-lock.json"
        original_lock = lock.read_bytes()
        before_lock = as_json(run(cli, root, "scenario", "generate", "-scenario",
                                  str(SCENARIO), "-workspace", str(isolated)))
        lock_run = next(c["argv"] for c in before_lock["commands"]
                        if c["stepId"] == "validate-positive")
        original_fragment = b'"formatVersion": 2,'
        if original_lock.count(original_fragment) != 1:
            raise AssertionError("Plugin Lock mutation anchor absent or ambiguous")
        # JSON-equivalent whitespace is enough to change a byte-level Lock pin.
        lock.write_bytes(original_lock.replace(original_fragment, b'"formatVersion": 2 ,'))
        after_lock = as_json(run(cli, root, "scenario", "generate", "-scenario",
                                 str(SCENARIO), "-workspace", str(isolated)))
        if before_lock["pluginLockSha256"] == after_lock["pluginLockSha256"]:
            raise AssertionError("Plugin Lock byte mutation escaped the pin")
        stale_lock = run(cli, root, *lock_run[1:], success=False)
        negative(stale_lock, "Plugin Lock fingerprint differs")
        lock.write_bytes(original_lock)

        # Negative N5: a shortened Scenario may still be executable, but
        # cannot satisfy the independent six-step testing obligation.
        reduced = copy.deepcopy(scenario)
        reduced["steps"] = [x for x in reduced["steps"] if x["id"] != "validate-denied"]
        observed_ids = {x["id"] for x in reduced["steps"]}
        assert observed_ids != STEPS and "validate-denied" in STEPS - observed_ids

        # Negative N6: even if an attester claims 'satisfied' for everything,
        # the independent fixed property catches the actual rejected input.
        always_satisfied = {rule: "satisfied" for rule in RULES}
        assert always_satisfied != independent_rule_properties(denied)

    scoped_inventory, scoped_controls = scoped_step_selftest(
        STEPS, measured, digest(source), digest(original_scenario.read_bytes()),
        {step: record["operation"] for step, record in scenario_steps.items()},
    )
    coverage = subprocess.run(["go", "tool", "cover", "-func", str(args.coverage.resolve())],
                              cwd=root, capture_output=True, text=True, check=True)
    last = coverage.stdout.strip().splitlines()[-1]
    if not re.fullmatch(r"total:\s*\(statements\)\s+[0-9]+(?:\.[0-9]+)?%", last):
        raise AssertionError(f"unrecognized Go statement coverage result: {last!r}")
    summary = {
        "schema": "tei.reference.issue-test-ability/v0",
        "designContext": {"admission": design["testExecutionAdmission"],
                          "fullTestGenerationQualified": False,
                          "requiredSourceRead": design["observations"]["requiredSourceRead"],
                          "sourceIndexFingerprint": design["observations"].get("observedFingerprints", {}).get("sourceIndexFingerprint")},
        "execution": "real built CLI, actual generated pinned Scenario commands",
        "scenarioInventory": scoped_inventory,
        "scenarioInventoryControls": scoped_controls,
        "tested": sorted(measured),
        "observed": measured,
        "independentRuleObligation": {
            "positive": positive_expected,
            "zero-quantity": denied_expected,
            "duplicate-keys": {**positive_expected, LINE_UNIQUE: "not-satisfied"},
        },
        "mutantsKilled": {
            "duplicate-relation-keys": "real-cli-rejected",
            "false-Scenario-expected": "real-cli-rejected",
            "stale-fixture-pin": "real-cli-rejected",
            "tampered-generated-plan": "real-cli-rejected",
            "changed-real-canonical-source": "real-cli-stale-world-rejected",
            "changed-real-plugin-lock": "real-cli-stale-lock-rejected",
            "omitted-required-step": "independent-obligation-rejected",
            "always-satisfied-grader": "independent-property-rejected",
        },
        "discriminationCounts": {
            "realCliFixtureExpectedPlanSourceAndLock": 6,
            "independentObligationAndOracle": 2,
            "actualProductEvaluatorCodeMutant": "not-run-in-this-probe",
        },
        "productTestEvidence": "synthetic Canonical Validation and Browse/Observation; no real DB",
        "sourceReversionObservation": "A-B-A returns to identical Source bytes and old pin succeeds; intervening history/Authority not proven",
        "pluginLockQualification": "byte pin only; actual plugin Consumer Load/Provider Behavior not measured",
        "coverage": {
            "requiredScenarioSteps": len(STEPS),
            "observedPinnedSteps": len(measured),
            "goC0StatementCoverageForCmdTei": last,
            "goC1BranchCoverage": "unmeasured",
            "domainRequirementsOverall": "not-qualified",
            "nonFunctionalRequirements": "not-measured",
        },
        "source": {"scenario": str(SCENARIO), "sha256": digest(source), "binarySha256": digest(cli.read_bytes())},
        "claimCeiling": "bounded-test-ability; not Scenario/Owner/Host/provider-wide",
    }
    args.out.write_text(json.dumps(summary, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    print("PASS: 6 real pinned Scenario steps; "
          "6 actual CLI input/pin/plan rejection controls plus "
          "2 independent obligation/oracle controls; no injected Product code mutant")
    print("TT-105/106: " + scoped_inventory["inventory"] +
          "; observed local steps without claiming complete Product coverage")
    print("C0:", last, " / C1: unmeasured; overall requirements/NFR: not-qualified")
    print("Scenario SHA256:", digest(source), " / binary SHA256:", digest(cli.read_bytes()))


if __name__ == "__main__":
    main()
