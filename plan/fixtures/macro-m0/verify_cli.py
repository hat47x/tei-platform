#!/usr/bin/env python3
"""実TEI CLIの操作面・成果物を同一Source版で照合する読取り専用Reference。

公開用 reference/standard-crud の限定ケースのみ。意味受理、実Host権限、
PostgreSQL更新、成果物の実Reader/配布はこの検査に含まれない。
"""
import argparse
import json
import subprocess
import sys
from pathlib import Path


class CheckError(AssertionError):
    pass


EXPECTED_OPERATIONS = (
    "ledger.list.sales.note",
    "ledger.list.sales.order",
    "ledger.read.sales.order",
    "ledger.create.sales.order",
    "ledger.update.sales.order",
)
EXPECTED_GAPS = (
    "sales.note/create/missing-semantics",
    "sales.note/read/missing-semantics",
    "sales.order/list/conflict",
    "sales.order/retire/missing-input",
)
EXPECTED_BINDINGS = {
    "ledger.create.sales.order": "sales.order.register",
    "ledger.update.sales.order": "sales.order.cancel",
}


def check(condition, explanation):
    if not condition:
        raise CheckError(explanation)


def nonempty_string(value, name):
    check(isinstance(value, str) and bool(value), f"{name}: nonempty string required")
    return value


def json_object(binary, *args):
    # No command executed here writes to the Workspace or issues an operation.
    check("-out" not in args, "M0 must not write artifact files")
    process = subprocess.run(
        [str(binary), *args],
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        timeout=120,
        text=True,
        encoding="utf-8",
        check=False,
    )
    # Do not print stderr or the returned JSON (Source paths/diagnostics may leak).
    check(process.returncode == 0, f"{args[0]}: nonzero exit {process.returncode}")
    try:
        payload = json.loads(process.stdout)
    except (ValueError, TypeError) as exc:
        raise CheckError(f"{args[0]}: invalid JSON") from exc
    check(isinstance(payload, dict), f"{args[0]}: JSON object required")
    return payload


def inspect_observation(version, discovery, validation, surface, artifact_set, expected_head):
    """Pure contract checker. Caller owns actual subprocess and Source provenance."""
    check(version.get("schemaVersion") == "tei.cli.version/v0", "CLI version schema")
    check(version.get("source") == "build-info", "CLI binary lacks build identity")
    check(version.get("vcsRevision") == expected_head, "binary differs from checked HEAD")
    check(version.get("vcsModified", False) is False, "checked binary is dirty")
    check(discovery.get("schemaVersion") == "tei.cli.discovery/v0", "CLI discovery schema")
    check(discovery.get("scope") == "selected-read-only-commands", "discovery scope")
    commands = discovery.get("commands")
    check(isinstance(commands, list), "discovery commands type")
    names = {}
    for command in commands:
        check(isinstance(command, dict) and isinstance(command.get("name"), str),
              "discovery entry shape")
        name = command["name"]
        check(name not in names, "duplicate command discovery name")
        names[name] = command
    for name in ("version", "model catalog", "inspect", "trace", "validate"):
        check(names.get(name, {}).get("effect") == "read-only",
              "required read-only command is missing or mislabeled")

    check(validation.get("schemaVersion") == "tei.cli.validate/v0", "validation schema")
    check(validation.get("ok") is True, "validation failed")
    diagnostics = validation.get("diagnostics")
    check(isinstance(diagnostics, list), "validation diagnostics type")
    world = nonempty_string(validation.get("worldFingerprint"), "validation.worldFingerprint")
    nonempty_string(validation.get("sourceIndexFingerprint"),
                    "validation.sourceIndexFingerprint")
    check(surface.get("schemaVersion") == "tei.operations.operation-surface/v0",
          "operation-surface schema")
    check(surface.get("effect") == "read-only", "operation surface is not read-only")
    check(surface.get("application") == "ledger", "operation application changed")
    check(surface.get("worldFingerprint") == world,
          "operation World differs from validated World")
    revision = nonempty_string(surface.get("modelRevision"), "surface.modelRevision")
    rule = surface.get("rule")
    check(isinstance(rule, dict), "derivation rule missing")
    for field in ("id", "version", "fingerprint"):
        nonempty_string(rule.get(field), f"surface.rule.{field}")
    check(isinstance(surface.get("authorityLimits"), list),
          "authority limits must be reported")

    operations = surface.get("operations")
    gaps = surface.get("gaps")
    conflicts = surface.get("conflicts")
    check(isinstance(operations, list) and isinstance(gaps, list)
          and isinstance(conflicts, list), "operation/gap/conflict lists missing")
    actual_ops = [op.get("id") for op in operations if isinstance(op, dict)]
    check(len(actual_ops) == len(operations), "malformed operation")
    check(tuple(actual_ops) == EXPECTED_OPERATIONS, "standard-crud operation regression")
    actual_gaps = []
    for gap in gaps:
        check(isinstance(gap, dict), "malformed gap")
        nonempty_string(gap.get("detail"), "gap.detail")
        actual_gaps.append("/".join(str(gap.get(key, "")) for key in
                                    ("target", "kind", "reason")))
    check(tuple(actual_gaps) == EXPECTED_GAPS, "standard-crud gap regression")
    indexed = {op["id"]: op for op in operations}
    for operation_id, binding in EXPECTED_BINDINGS.items():
        check(indexed[operation_id].get("binding") == binding,
              "derived write binding changed")
    # A binding is a declared capability, never current Host authority.

    check(artifact_set.get("inputRevision") == revision,
          "artifact input revision differs from operation revision")
    artifacts = artifact_set.get("artifacts")
    manifest = artifact_set.get("manifest")
    findings = artifact_set.get("findings")
    check(isinstance(artifacts, list) and bool(artifacts), "no artifact set")
    check(isinstance(manifest, dict) and isinstance(findings, list),
          "artifact manifest/findings missing")
    check(manifest.get("inputRevision") == revision,
          "manifest input revision differs from operation revision")
    check(isinstance(manifest.get("entries"), list) and bool(manifest["entries"]),
          "manifest lacks entries")
    nonempty_string(manifest.get("digest"), "manifest.digest")
    check(type(manifest.get("complete")) is bool, "manifest complete must be boolean")
    # Complete/empty findings concern internal consistency, not consumer-specific
    # necessary members, actual published bytes, or an external reader.
    return {
        "status": "actual-cli-read-only-observed",
        "fixedApplication": "ledger",
        "binaryRevisionMatchesExpected": True,
        "discoveryScope": discovery["scope"],
        "operationsAdvertised": "operations" in names,
        "artifactsAdvertised": "artifacts" in names,
        "operationCount": len(operations),
        "gapCount": len(gaps),
        "conflictCount": len(conflicts),
        "validationDiagnosticCount": len(diagnostics),
        "manifestComplete": manifest["complete"],
        "artifactCount": len(artifacts),
        "artifactFindingCount": len(findings),
        "worldProjectionAligned": True,
        "operationArtifactInputRevisionAligned": True,
        "actualHostAuthority": "NOT_RUN",
        "providerEffect": "NOT_RUN",
        "atomicMultiProcessSnapshot": "NOT_ESTABLISHED",
        "consumerArtifactDelivery": "NOT_RUN",
        "businessMeaningAcceptance": "NOT_RUN",
    }


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--binary", required=True, type=Path)
    parser.add_argument("--workspace", required=True, type=Path)
    parser.add_argument("--expected-head", required=True)
    args = parser.parse_args()
    check(args.binary.is_file(), "TEI binary missing")
    check((args.workspace / "tei.workspace.yaml").is_file(), "workspace manifest missing")
    check(len(args.expected_head) == 40 and
          all(c in "0123456789abcdef" for c in args.expected_head),
          "exact 40-digit HEAD required")
    binary = args.binary.resolve()
    root = str(args.workspace.resolve())
    result = inspect_observation(
        json_object(binary, "version", "-json"),
        json_object(binary, "commands"),
        json_object(binary, "validate", "-workspace", root, "-json"),
        json_object(binary, "operations", "-workspace", root, "-application", "ledger"),
        json_object(binary, "artifacts", "-workspace", root, "-application", "ledger"),
        args.expected_head,
    )
    # Summary contains no raw Source, diagnostics, executable path or credentials.
    print(json.dumps(result, ensure_ascii=False, sort_keys=True))


if __name__ == "__main__":
    try:
        main()
    except (CheckError, OSError, subprocess.TimeoutExpired) as exc:
        print(f"macro-m0 reference failed: {type(exc).__name__}: {exc}", file=sys.stderr)
        sys.exit(1)
