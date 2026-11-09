#!/usr/bin/env python3
"""Bounded OB-01/02 characterization of the *real* TEI CLI.

This exercises the compiled Go executable and the checked-in public negative
fixture. Expected operation IDs and gaps are independently frozen in the
pre-existing reference/standard-crud/README.md; this is NOT a business Oracle.
No Provider, Host grant, external Harness, or database effect is exercised.
"""
from __future__ import annotations

import argparse
import json
from pathlib import Path
import subprocess
import sys
import tempfile


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


def invoke(binary: Path, *args: str) -> subprocess.CompletedProcess[str]:
    result = subprocess.run(
        [str(binary), *args],
        capture_output=True,
        text=True,
        encoding="utf-8",
        timeout=60,
        check=False,
    )
    if result.returncode:
        raise AssertionError(
            f"CLI {args!r} exited {result.returncode}; "
            f"stdout={result.stdout[:800]!r}, stderr={result.stderr[:800]!r}"
        )
    return result


def require(condition: bool, message: str) -> None:
    if not condition:
        raise AssertionError(message)


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--binary", required=True, type=Path)
    parser.add_argument(
        "--workspace",
        default="reference/standard-crud/workspace",
        type=Path,
    )
    args = parser.parse_args()
    binary = args.binary.resolve(strict=True)
    workspace = args.workspace.resolve(strict=True)
    require(binary.is_file(), "actual built CLI binary missing")
    require((workspace / "tei.workspace.yaml").is_file(), "public workspace missing")

    discovery = json.loads(invoke(binary, "commands").stdout)
    require(
        discovery.get("scope") == "selected-read-only-commands",
        f"discovery scope changed: {discovery.get('scope')!r}",
    )
    named_commands = {item["name"] for item in discovery["commands"]}
    require("model catalog" in named_commands, "known read-only entry disappeared")
    require("operations" not in named_commands, "discovery coverage unexpectedly changed")
    require("artifacts" not in named_commands, "discovery coverage unexpectedly changed")

    cmd = ("operations", "-workspace", str(workspace), "-application", "ledger")
    first = invoke(binary, *cmd)
    second = invoke(binary, *cmd)
    require(first.stdout == second.stdout, "fixed-source stdout not deterministic")
    surface = json.loads(first.stdout)

    require(surface.get("schemaVersion") == "tei.operations.operation-surface/v0",
            "unexpected operation surface schema")
    require(surface.get("effect") == "read-only",
            "surface projection must be read-only")
    require(surface.get("application") == "ledger", "application filter lost")
    require(bool(surface.get("worldFingerprint")), "missing world fingerprint")
    require(bool(surface.get("modelRevision")), "missing model revision")
    rule = surface.get("rule", {})
    require(rule.get("id") == "tei.operation-surface/standard-crud",
            "wrong derivation rule")
    require(rule.get("version") == "v1" and bool(rule.get("fingerprint")),
            "rule version/fingerprint missing")
    require(bool(surface.get("authorityLimits")),
            "surface must explicitly deny being runtime authority")

    operations = surface["operations"]
    gaps = surface["gaps"]
    require(tuple(item["id"] for item in operations) == EXPECTED_OPERATIONS,
            f"operation expectation mismatch: {[item['id'] for item in operations]}")
    got_gaps = tuple(
        "/".join((item["target"], item["kind"], item["reason"]))
        for item in gaps
    )
    require(got_gaps == EXPECTED_GAPS, f"gap expectation mismatch: {got_gaps}")
    require(all(item.get("detail") for item in gaps), "gap explanation missing")
    require(surface["summary"]["operations"] == len(operations),
            "summary operation count mismatch")
    require(surface["summary"]["gaps"] == len(gaps),
            "summary gap count mismatch")

    by_id = {item["id"]: item for item in operations}
    for key in ("ledger.list.sales.order", "ledger.read.sales.order"):
        operation = by_id[key]
        require(operation["autonomy"] == "unattended" and not operation.get("effects"),
                f"read derivation changed: {key}")
    for kind, binding in (("create", "sales.order.register"),
                          ("update", "sales.order.cancel")):
        op = by_id[f"ledger.{kind}.sales.order"]
        require(op.get("autonomy") == "attended", f"{kind} is unattended")
        require(op.get("binding") == binding, f"{kind} binding changed")
        require(op.get("effects") == [f"{kind}:sales.order"],
                f"{kind} effect changed")
    require(by_id["ledger.update.sales.order"].get("concurrency") == "expected-state",
            "update lost concurrency constraint")

    # The same 'operations' command takes '-out' and writes an actual FILE.
    # Discovery of 'operations' as a read-only surface must not grant this
    # command variant a read-only OS-effect qualification.
    with tempfile.TemporaryDirectory(prefix="tei-action-surface-") as directory:
        out = Path(directory) / "operation-surface.json"
        require(not out.exists(), "output must not pre-exist")
        written = invoke(binary, *cmd, "-out", str(out))
        require(not written.stdout.strip(), "-out unexpectedly wrote surface to stdout")
        require(out.is_file(), "operations -out did not write the output file")
        require(json.loads(out.read_text(encoding="utf-8")) == surface,
                "file output differs from read-only stdout surface")
        require(len(tuple(Path(directory).iterdir())) == 1,
                "unexpected additional files in isolated output directory")

    print(json.dumps({
        "case": "OB-01/OB-02+operations-out-effect",
        "result": "actual-cli-observed",
        "operationCount": len(operations),
        "gapCount": len(gaps),
        "worldFingerprint": surface["worldFingerprint"],
        "modelRevision": surface["modelRevision"],
        "ruleFingerprint": rule["fingerprint"],
        "limits": [
            "one public synthetic negative fixture",
            "historical expected rather than independent business oracle",
            "no Host/Provider/DB/Harness/Owner acceptance",
        ],
    }, ensure_ascii=False, sort_keys=True))


if __name__ == "__main__":
    try:
        main()
    except (AssertionError, ValueError, KeyError, OSError,
            subprocess.SubprocessError) as exc:
        print(f"OB-01/02 qualification failed: {exc}", file=sys.stderr)
        sys.exit(1)
