#!/usr/bin/env python3
"""Execute a *proposed* new line-number-zero Scenario via the actual TEI CLI.

A candidate is written only to a temporary copy of the existing #1180 Scenario.
This is NOT a second Scenario SSOT, nor an automatic Owner acceptance.
The proposal is the already-known #3273 task; this test does NOT prove
blind AI discovery, external Skill/Harness delivery, C1, or real DB effects.
"""
from __future__ import annotations

import argparse
from copy import deepcopy
from decimal import Decimal
import hashlib
import json
from pathlib import Path
import shutil
import subprocess
import tempfile

BASE_SCENARIO = Path("reference/driver-scenarios/order-entry-pattern-smoke.json")
VALIDATION_SOURCE = Path("reference/workspace/semantic/sales/validation.tei")
WORKSPACE = Path("reference/workspace")
TARGET_RULE = "sales.order_line.line_number_positive"
QUANTITY_RULE = "sales.order_line.quantity_positive"
UNIQUE_RULE = "sales.order.lines.line_number_unique"
TARGET_STEP = "validate-line-number-zero-proposal"
CONTEXT_SCHEMA = "tei.reference.design-context-readiness/v0"


def digest(data: bytes) -> str:
    return hashlib.sha256(data).hexdigest()


def real_cli(cli: Path, root: Path, *args: str, pass_expected: bool = True) -> subprocess.CompletedProcess[str]:
    result = subprocess.run([str(cli), *map(str, args)], cwd=root, text=True,
                            encoding="utf-8", errors="replace",
                            capture_output=True, timeout=90, check=False)
    if pass_expected and (result.returncode != 0 or result.stderr.strip()):
        raise AssertionError(
            f"TEI command failed: {args[:3]} status={result.returncode}, stderr={result.stderr[-550:]}"
        )
    if not pass_expected and result.returncode == 0:
        raise AssertionError(f"TEI wrongly passed a negative: {args[:3]}")
    return result


def as_object(result: subprocess.CompletedProcess[str]) -> dict:
    value = json.loads(result.stdout)
    if not isinstance(value, dict):
        raise AssertionError("object JSON expected from actual CLI")
    return value


def assert_independent_rule(case: dict, expected: str) -> None:
    """Fixed elementary known-case property; not read from Candidate Expected."""
    number = Decimal(case["numbers"]["sales.order_line.line_number"])
    observed = "satisfied" if number > Decimal("0") else "not-satisfied"
    if observed != expected:
        raise AssertionError(f"independent strict-positive property contradicted: {number}")


def wrong_result(result: subprocess.CompletedProcess[str], reason: str) -> None:
    if result.returncode == 0:
        raise AssertionError("negative run succeeded")
    if reason not in result.stderr or TARGET_RULE not in result.stderr:
        raise AssertionError(f"wrong negative diagnosis: {result.stderr[-550:]}")


def make_candidate(base: dict, fixture_name: str) -> dict:
    candidate = deepcopy(base)
    if candidate.get("schemaVersion") != "tei.driver-scenario/v0":
        raise AssertionError("unrecognized Scenario source contract")
    if len(candidate["steps"]) != 6 or TARGET_STEP in {x["id"] for x in candidate["steps"]}:
        raise AssertionError("baseline Scenario requirement/step set changed")
    candidate["id"] = "order-entry-line-number-zero-t1-proposal"
    candidate["steps"].append({
        "id": TARGET_STEP,
        "operation": "evaluate-validations",
        "fixture": "fixtures/" + fixture_name,
        "expectValidations": {
            QUANTITY_RULE: "satisfied",
            UNIQUE_RULE: "satisfied",
            TARGET_RULE: "not-satisfied",
        },
    })
    return candidate


def main() -> None:
    ap = argparse.ArgumentParser()
    ap.add_argument("--root", required=True, type=Path)
    ap.add_argument("--cli", required=True, type=Path)
    ap.add_argument("--context", required=True, type=Path)
    ap.add_argument("--out", required=True, type=Path)
    args = ap.parse_args()
    root, cli = args.root.resolve(), args.cli.resolve()
    context = json.loads(args.context.read_text(encoding="utf-8"))
    if not (
        context.get("schema") == CONTEXT_SCHEMA
        and context.get("boundedReference") == "partial"
        and context.get("testExecutionAdmission") == "exploration-only"
        and context.get("qualifiedTestGeneration") is False
        and context.get("independentFullRequiredSetProven") is False
        and context.get("externalHarnessDeliveryComplete") is None
        and not context.get("blockers")
        and context.get("observations", {}).get("fingerprintConsistency") == "consistent"
    ):
        raise AssertionError("cannot use incomplete/stale Design Context for local Candidate experiment")
    source = (root / VALIDATION_SOURCE).read_bytes()
    declaration = b"validation sales.order_line.line_number_positive positive {"
    if source.count(declaration) != 1:
        raise AssertionError("Source has no unique reference for the target Validation")
    original = (root / BASE_SCENARIO).read_bytes()
    base = json.loads(original)

    with tempfile.TemporaryDirectory(prefix="tei-t1-candidate-") as directory:
        scratch = Path(directory) / "scenario"
        # A temporary Candidate, never an accepted Scenario or another SSOT.
        shutil.copytree(root / BASE_SCENARIO.parent, scratch)
        proposal = scratch / "line-number-zero-proposal.json"
        fixture_name = "line_number_zero_proposal.json"
        fixture_path = scratch / "fixtures" / fixture_name

        # Distinguish the input of the new case from the two existing ones:
        # only line_number is zero, quantity remains positive, keys stay unique.
        valid_fixture = json.loads((root / BASE_SCENARIO.parent / "fixtures/order_valid.json").read_text())
        new_fixture = deepcopy(valid_fixture)
        new_fixture["numbers"]["sales.order_line.line_number"] = "0"
        assert_independent_rule(new_fixture, "not-satisfied")
        assert_independent_rule(valid_fixture, "satisfied")
        if new_fixture["numbers"]["sales.order_line.quantity"] != valid_fixture["numbers"]["sales.order_line.quantity"]:
            raise AssertionError("candidate unexpectedly changed quantity")
        if new_fixture["relationKeys"] != valid_fixture["relationKeys"]:
            raise AssertionError("candidate unexpectedly changed relation uniqueness")
        fixture_path.write_text(json.dumps(new_fixture, indent=2) + "\n", encoding="utf-8")

        candidate = make_candidate(base, fixture_name)
        proposal.write_text(json.dumps(candidate, indent=2) + "\n", encoding="utf-8")
        verification = as_object(real_cli(cli, root, "scenario", "verify",
                                          "-scenario", str(proposal),
                                          "-workspace", str(WORKSPACE)))
        if verification.get("selectedStepCount") != 7:
            raise AssertionError("Candidate step missing from actual verification")
        evs = verification.get("evidence", [])
        proposed_verification = next((e for e in evs if e.get("stepId") == TARGET_STEP), None)
        if (proposed_verification is None or proposed_verification.get("verified") is not False
                or "not-evaluated" not in proposed_verification.get("evidenceNature", "")):
            raise AssertionError("static verify incorrectly claimed to have executed new Candidate")

        generated = real_cli(cli, root, "scenario", "generate", "-scenario",
                             str(proposal), "-workspace", str(WORKSPACE))
        plan = as_object(generated)
        commands = plan.get("commands")
        if not isinstance(commands, list) or len(commands) != 7:
            raise AssertionError("new Candidate not in generated command plan")
        selected = [e for e in commands if e.get("stepId") == TARGET_STEP]
        if len(selected) != 1 or selected[0].get("argv", [])[:2] != ["tei", "scenario"]:
            raise AssertionError("Candidate argv missing or ambiguous")
        argv = selected[0]["argv"]
        if not all(token in argv for token in (
                "-scenario-sha256", "-world-sha256", "-plugin-lock-sha256", "-fixture-sha256")):
            raise AssertionError("new CLI plan does not pin Scenario/World/Lock/Fixture")
        plan_file = Path(directory) / "proposal-plan.json"
        plan_file.write_text(generated.stdout, encoding="utf-8")
        verified_plan = as_object(real_cli(cli, root, "scenario", "check", "-scenario",
                                           str(proposal), "-workspace", str(WORKSPACE),
                                           "-plan", str(plan_file)))
        if verified_plan.get("planMatches") is not True:
            raise AssertionError("candidate plan does not match its generated revision")

        result = as_object(real_cli(cli, root, *argv[1:]))
        if result.get("executionMode") != "generated-pinned" or result.get("selectedStepCount") != 1:
            raise AssertionError("candidate not actually run in pinned mode")
        evidence = result.get("evidence", [])
        expected = candidate["steps"][-1]["expectValidations"]
        if (len(evidence) != 1 or evidence[0].get("stepId") != TARGET_STEP
                or evidence[0].get("verified") is not True
                or evidence[0].get("validationResults") != expected
                or evidence[0].get("evidenceNature") != "canonical-validations-synthetic-input-evaluated"):
            raise AssertionError("actual CLI did not evaluate Candidate's required boundary")

        # Independent controls: invalidating either side of the test must
        # change the verdict. "not-satisfied" is a business result, NOT an error.
        candidate["steps"][-1]["expectValidations"][TARGET_RULE] = "satisfied"
        proposal.write_text(json.dumps(candidate, indent=2) + "\n", encoding="utf-8")
        lying_expected = real_cli(cli, root, "scenario", "run", "-scenario", str(proposal),
                                  "-workspace", str(WORKSPACE), "-step", TARGET_STEP,
                                  pass_expected=False)
        wrong_result(lying_expected, "result=not-satisfied")

        candidate["steps"][-1]["expectValidations"][TARGET_RULE] = "not-satisfied"
        proposal.write_text(json.dumps(candidate, indent=2) + "\n", encoding="utf-8")
        new_fixture["numbers"]["sales.order_line.line_number"] = "1"
        assert_independent_rule(new_fixture, "satisfied")
        fixture_path.write_text(json.dumps(new_fixture, indent=2) + "\n", encoding="utf-8")
        lying_fixture = real_cli(cli, root, "scenario", "run", "-scenario", str(proposal),
                                 "-workspace", str(WORKSPACE), "-step", TARGET_STEP,
                                 pass_expected=False)
        wrong_result(lying_fixture, "result=satisfied")
        stale = real_cli(cli, root, *argv[1:], pass_expected=False)
        if "Fixture fingerprint differs" not in stale.stderr:
            raise AssertionError("candidate stale pin rejected for wrong reason")
        result_report = {
            "schema": "tei.reference.t1-line-number-candidate/v0",
            "owner": "#3273; proposed-only; not adopted in #1180",
            "candidateOrigin": "AI-authored from previously specified #3273 rule boundary; not blind external Harness generation",
            "claimCeiling": "bounded-proposed-case-executed; unknown coverage denominator and Owner meaning",
            "source": {"path": str(VALIDATION_SOURCE), "sha256": digest(source)},
            "baseScenario": {"path": str(BASE_SCENARIO), "sha256": digest(original)},
            "case": {"stepId": TARGET_STEP, "input": "line number 0 with positive quantity and unique relation keys",
                     "observed": evidence[0]["validationResults"], "executionMode": result["executionMode"],
                     "nature": evidence[0]["evidenceNature"]},
            "generated": {"scenarioSHA256": plan.get("scenarioSha256"),
                          "worldSHA256": plan.get("worldSha256"),
                          "pluginLockSHA256": plan.get("pluginLockSha256"),
                          "candidateFixtureSHA256": plan.get("fixtureSha256", {}).get(TARGET_STEP)},
            "negativeControls": {
                "lying-expected": "actual-evaluator-rejected",
                "lying-fixture": "actual-evaluator-rejected",
                "changed-fixture-stale-pin": "actual-cli-rejected",
            },
            "context": {"boundedReference": "partial", "qualifiedTestGeneration": False},
            "coverage": {
                "newProposedValidationBoundary": 1,
                "globalRequirementCoverage": "unknown",
                "goC0": "not-measured-in-this-probe",
                "goC1": "not-measured",
            },
            "externalHarnessSkillLoadedHere": False,
            "independentBlindHoldoutApplied": False,
            "ownerMeaningAccepted": False,
            "scenarioSSOTChanged": False,
            "realProviderOrHostEffect": False,
            "binarySha256": digest(cli.read_bytes()),
        }
        args.out.parent.mkdir(parents=True, exist_ok=True)
        args.out.write_text(json.dumps(result_report, ensure_ascii=False, indent=2) + "\n",
                            encoding="utf-8")
        print("PASS: one proposed new boundary in real pinned CLI; "
              "false expected + false fixture + stale pin rejected")
        print("LIMIT: not blind AI generation, not Scenario SSOT adoption, "
              "not full test generation qualification")


if __name__ == "__main__":
    main()
