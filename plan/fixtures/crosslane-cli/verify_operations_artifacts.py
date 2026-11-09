#!/usr/bin/env python3
"""Actual-process, read-only TEI CLI operations->artifact handoff.

Only public reference workspaces. No provider execution, business acceptance,
read-cohort proof, client disclosure, or runtime authority is tested here.
"""
import argparse
import copy
import json
import subprocess
import sys
import tempfile
from collections import Counter
from pathlib import Path


def require(condition, message):
    if not condition:
        raise AssertionError(message)


def cli(binary, *args):
    result = subprocess.run(
        [str(binary), *args], capture_output=True, text=True,
        encoding="utf-8", timeout=120, check=False,
    )
    require(result.returncode == 0,
            f"{args[0]} failed: exit={result.returncode}; stderr={result.stderr[:300]}")
    require(not result.stderr, f"{args[0]}: unexpected stderr")
    try:
        value = json.loads(result.stdout)
    except ValueError as exc:
        raise AssertionError(f"{args[0]} returned invalid JSON") from exc
    require(isinstance(value, dict), f"{args[0]} did not return a JSON object")
    return value


def artifact_surface(set_doc):
    artifacts = set_doc.get("artifacts")
    require(isinstance(artifacts, list), "artifact list missing")
    surfaces = [x for x in artifacts if isinstance(x, dict)
                and x.get("kind") == "operation-surface"]
    require(len(surfaces) == 1, "expected exactly one operation-surface artifact")
    return surfaces[0]


def compare_handoff(operations, set_doc, app):
    """Compare only fields explicitly carried by both published projections."""
    require(operations.get("effect") == "read-only",
            "operations must remain a read-only derivation")
    require(operations.get("application") == app, "wrong application on operation surface")
    revision = operations.get("modelRevision")
    require(isinstance(revision, str) and revision, "operations model revision missing")
    require(set_doc.get("inputRevision") == revision,
            "actual operations and artifacts derived from different model revisions")
    require(set_doc.get("findings") == [],
            "artifact consistency findings: not a qualified artifact set")
    manifest = set_doc.get("manifest")
    require(isinstance(manifest, dict), "manifest missing")
    require(manifest.get("inputRevision") == revision, "manifest revision drift")
    entries = manifest.get("entries")
    require(isinstance(entries, list), "manifest entries missing")
    artifacts = set_doc.get("artifacts")
    expected = Counter((a["kind"], a["digest"], len(a.get("unresolved") or []))
                       for a in artifacts)
    observed = Counter((e["kind"], e["digest"], e["unresolved"])
                       for e in entries)
    require(expected == observed,
            "manifest membership/digest/unresolved differs from actual artifact set")
    require(bool(manifest.get("complete")) ==
            all(not (a.get("unresolved") or []) for a in artifacts),
            "manifest completeness differs from listed unresolved counts")

    surface_artifact = artifact_surface(set_doc)
    require(surface_artifact.get("inputRevision") == revision,
            "surface artifact revision differs")
    try:
        projected = json.loads(surface_artifact["content"])
    except (KeyError, ValueError) as exc:
        raise AssertionError("operation surface artifact content is not valid JSON") from exc
    shared = ("schemaVersion", "application", "rule",
              "operations", "gaps", "summary")
    for field in shared:
        require(projected.get(field) == operations.get(field),
                f"real CLI operation surface / artifact field differs: {field}")
    require(len(surface_artifact.get("unresolved") or []) == len(operations["gaps"]),
            "gaps were not carried into artifact unresolved list")
    expected_gaps = Counter((g["target"], g["reason"],
                             f"{g['kind']}: {g['detail']}") for g in operations["gaps"])
    actual_gaps = Counter((u["subject"], u["reason"], u["detail"])
                          for u in (surface_artifact.get("unresolved") or []))
    require(expected_gaps == actual_gaps,
            "unresolved artifact reasons differ from actual CLI gaps")

    # The content format is intentionally a *subset*: it does not carry
    # worldFingerprint, modelRevision, authorityLimits, or conflicts.
    # Never call byte correspondence of this subset full consumer equivalence.
    require("authorityLimits" in operations and operations["authorityLimits"],
            "actual operations did not declare their authority limitations")
    require("authorityLimits" not in projected,
            "artifact projection changed: re-qualify this bounded comparison")
    require("worldFingerprint" not in projected,
            "artifact projection changed: re-qualify this bounded comparison")
    return revision, surface_artifact


def verify_export(set_doc, exported, folder):
    """Read bytes actually written by tei artifacts -out; not just memory."""
    manifest_path = folder / "manifest.json"
    surface_path = folder / "operation-surface.json"
    require(manifest_path.is_file() and surface_path.is_file(),
            "actual artifact files missing")
    on_disk = json.loads(manifest_path.read_text(encoding="utf-8"))
    require(on_disk == {"manifest": set_doc["manifest"],
                        "findings": set_doc["findings"]},
            "manifest file differs from actual artifact set")
    require(exported == set_doc["manifest"],
            "-out stdout manifest differs from in-memory artifact set")
    expected_body = artifact_surface(set_doc)["content"].encode("utf-8")
    actual_body = surface_path.read_bytes()
    require(actual_body == expected_body,
            "operation-surface.json bytes differ from actual produced artifact")


def observe(binary, workspace, app, expect_negative):
    workspace = workspace.resolve(strict=True)
    args = ("-workspace", str(workspace), "-application", app)
    ops = cli(binary, "operations", *args)
    set_doc = cli(binary, "artifacts", *args)
    revision, surface = compare_handoff(ops, set_doc, app)
    require(cli(binary, "operations", *args) == ops,
            "independent operations process changed without source modification")
    require(cli(binary, "artifacts", *args) == set_doc,
            "independent artifacts process changed without source modification")
    if expect_negative:
        require(bool(ops["gaps"]), "negative fixture unexpectedly has no gaps")
        require(set_doc["manifest"]["complete"] is False,
                "negative fixture was silently called complete")
    else:
        # Independent observed counterexample at this reference revision:
        # all standard operations derived does not mean every document did.
        require(len(ops["gaps"]) == 0,
                "order_entry operation gaps changed: re-qualify the fixture")
        require(set_doc["manifest"]["complete"] is False,
                "order_entry manifest completeness changed: re-qualify the fixture")
        other_unresolved = sum(len(item.get("unresolved") or [])
                               for item in set_doc["artifacts"]
                               if item.get("kind") != "operation-surface")
        require(other_unresolved > 0,
                "manifest incompleteness must originate outside operation-surface")
        require(not (artifact_surface(set_doc).get("unresolved") or []),
                "surface must have no unresolved when operation gaps are empty")
        require(any(x.get("id") == "order_entry.update.sales.order"
                    for x in ops["operations"]),
                "expected accepted-model standard update is not listed")

    with tempfile.TemporaryDirectory(prefix="tei-handoff-") as path:
        folder = Path(path)
        out = cli(binary, "artifacts", *args, "-out", str(folder))
        verify_export(set_doc, out, folder)
        surface_file = folder / "operation-surface.json"
        surface_file.write_bytes(surface_file.read_bytes() + b"\n")
        try:
            verify_export(set_doc, out, folder)
        except AssertionError as exc:
            require("bytes differ" in str(exc),
                    "tampered file failed for an unrelated reason")
        else:
            raise AssertionError("mutated consumer bytes were not detected")

    return {
        "application": app,
        "revisionPresent": bool(revision),
        "operations": len(ops["operations"]),
        "gaps": len(ops["gaps"]),
        "artifactCount": len(set_doc["artifacts"]),
        "manifestComplete": set_doc["manifest"]["complete"],
        "crossProcessFieldCorrespondence": "observed-limited-subset",
        "artifactFileBytesChecked": True,
        "tamperedFileRejected": True,
        "actualExecutionAuthority": "not-tested",
        "realProviderReadOrWrite": "not-tested",
    }


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--binary", type=Path, required=True)
    parser.add_argument("--reference-workspace", type=Path, required=True)
    parser.add_argument("--negative-workspace", type=Path, required=True)
    parser.add_argument("--expected-head", required=True)
    args = parser.parse_args()
    require(args.binary.is_file(), "actual TEI binary missing")
    require(len(args.expected_head) == 40 and
            all(ch in "0123456789abcdef" for ch in args.expected_head),
            "expected HEAD must be exact lowercase SHA")
    version = cli(args.binary, "version", "-json")
    require(version.get("schemaVersion") == "tei.cli.version/v0",
            "unexpected actual executable version schema")
    require(version.get("vcsRevision") == args.expected_head,
            "actual CLI binary is not the candidate HEAD")
    require(version.get("vcsModified", False) is False,
            "the executed binary was built from a modified checkout")
    observed = [
        observe(args.binary.resolve(), args.negative_workspace, "ledger", True),
        observe(args.binary.resolve(), args.reference_workspace,
                "order_entry", False),
    ]
    # These are two actual CLI process chains, not accepted Function Context
    # or end-to-end Data Operations/Provider/Business Result.
    print(json.dumps({
        "result": "real-tei-cli-operations-artifacts-projection-observed",
        "binaryMatchesExactHead": True,
        "cases": observed,
        "caveat": "artifact surface is a limited projection, not full CLI equivalence",
    }, ensure_ascii=True, sort_keys=True))


if __name__ == "__main__":
    try:
        main()
    except (AssertionError, OSError, subprocess.TimeoutExpired, KeyError,
            TypeError, ValueError) as exc:
        print(f"real CLI handoff check failed: {type(exc).__name__}: {exc}",
              file=sys.stderr)
        sys.exit(1)
