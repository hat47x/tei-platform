#!/usr/bin/env python3
"""One-revision cross-command witness for the *real* TEI executable.

Requires a built ./cmd/tei binary. Uses only public reference workspace copied
into a temporary directory; mutation is a file relocation in that copy.
This is NOT a transaction snapshot/Host/DB/Skill/SHO/Owner acceptance test.

Example:
 python3 -B plan/fixtures/capability-handoff/verify_real_cli_revision.py \
    --binary /tmp/tei --workspace reference/standard-crud/workspace \
    --application ledger --expected-head <40-hex-head>
"""
import argparse
from collections import Counter
import hashlib
import json
from pathlib import Path
import shutil
import subprocess
import tempfile

SOURCE = Path("semantic/sales/order.tei")
MOVED = Path("semantic/sales/order-relocated.tei")


def require(condition, explanation):
    if not condition:
        raise AssertionError(explanation)


def run(binary, *args):
    result = subprocess.run(
        [str(binary), *map(str, args)], capture_output=True,
        text=True, encoding="utf-8", timeout=90, check=False)
    require(result.returncode == 0,
            f"TEI command {args[0]} failed with code {result.returncode}: "
            f"{result.stderr[:1200]} / {result.stdout[:1000]}")
    try:
        data = json.loads(result.stdout)
    except json.JSONDecodeError as exc:
        raise AssertionError(f"invalid JSON from {args[0]}: {result.stdout[:1000]}") from exc
    require(isinstance(data, dict), f"expected JSON object from {args[0]}")
    return data


def source_tree_identity(workspace):
    """Local source snapshot witness, NOT an atomic concurrent read guarantee."""
    h = hashlib.sha256()
    for path in sorted(workspace.rglob("*")):
        require(not path.is_symlink(), "fixture unexpectedly contains a symlink")
        if not path.is_file():
            continue
        relative = path.relative_to(workspace).as_posix()
        data = path.read_bytes()
        h.update(len(relative.encode("utf-8")).to_bytes(8, "big"))
        h.update(relative.encode("utf-8"))
        h.update(len(data).to_bytes(8, "big"))
        h.update(data)
    return h.hexdigest()


def observe(binary, workspace, application):
    inputs_before = source_tree_identity(workspace)
    checked = run(binary, "validate", "-workspace", workspace, "-json")
    require(checked.get("schemaVersion") == "tei.cli.validate/v0"
            and checked.get("ok") is True, "actual workspace validation failed")
    operations = run(binary, "operations", "-workspace", workspace,
                     "-application", application)
    require(operations.get("schemaVersion") == "tei.operations.operation-surface/v0"
            and isinstance(operations.get("gaps"), list)
            and operations.get("application") == application,
            "derived operation surface did not match scoped application")
    artifacts = run(binary, "artifacts", "-workspace", workspace,
                    "-application", application)
    manifest = artifacts.get("manifest")
    require(isinstance(manifest, dict) and isinstance(artifacts.get("artifacts"), list),
            "actual artifacts did not include a manifest and members")
    revision = operations.get("modelRevision")
    require(isinstance(revision, str) and revision
            and revision == artifacts.get("inputRevision")
            and revision == manifest.get("inputRevision"),
            "operations / artifact set / manifest are NOT same model revision")
    require(all(member.get("inputRevision") == revision
                for member in artifacts["artifacts"]),
            "artifact member revision mismatch")
    require(Counter(item["digest"] for item in artifacts["artifacts"])
            == Counter(item["digest"] for item in manifest["entries"]),
            "manifest entries do not cover declared artifact digests")

    inputs_after = source_tree_identity(workspace)
    require(inputs_before == inputs_after,
            "reference workspace changed during multi-command observation")
    return {"input_tree_sha256": inputs_before, "revision": revision,
            "world_fingerprint": operations.get("worldFingerprint"),
            "gaps": len(operations["gaps"]),
            "findings": len(artifacts.get("findings", [])),
            "manifest_complete": manifest.get("complete"),
            "manifest_digest": manifest.get("digest"),
            "artifacts": artifacts, "operations": operations}


def verify_reader(binary, workspace, application, baseline, directory):
    """Observe real file bytes via independent Python read, never infer from a digest."""
    require(not directory.exists(), "expected a fresh output directory")
    written = run(binary, "artifacts", "-workspace", workspace,
                  "-application", application, "-out", directory)
    require(written == baseline["artifacts"]["manifest"],
            "stdout published manifest differs from prior read-only result")
    manifest_path = directory / "manifest.json"
    require(manifest_path.is_file(), "manifest file is missing")
    stored = json.loads(manifest_path.read_bytes())
    require(stored["manifest"] == baseline["artifacts"]["manifest"],
            "real manifest file content differs from expected stdout")
    require(stored["findings"] == baseline["artifacts"].get("findings", []),
            "real manifest file findings differ")
    physical = [p for p in directory.iterdir() if p.is_file() and p.name != "manifest.json"]
    require(all(p.parent == directory and not p.is_symlink() for p in physical),
            "unexpected artifact paths")
    expected_bodies = Counter(
        member["content"].encode("utf-8") for member in baseline["artifacts"]["artifacts"])
    actual_bodies = Counter(p.read_bytes() for p in physical)
    require(expected_bodies == actual_bodies,
            "published file bytes do not cover the exact ArtifactSet content")
    return {"file_count": len(physical), "reader_validated_bytes": True,
            "manifest_file_sha256": hashlib.sha256(manifest_path.read_bytes()).hexdigest()}


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--binary", type=Path, required=True)
    parser.add_argument("--workspace", type=Path, required=True)
    parser.add_argument("--application", default="ledger")
    parser.add_argument("--expected-head", required=True)
    options = parser.parse_args()
    binary = options.binary.resolve(strict=True)
    original = options.workspace.resolve(strict=True)
    require(len(options.expected_head) == 40
            and all(c in "0123456789abcdef" for c in options.expected_head),
            "expected-head must be a full lower-case commit SHA")
    version = run(binary, "version", "-json")
    require(version.get("source") == "build-info"
            and version.get("vcsRevision") == options.expected_head
            and version.get("vcsModified", False) is False,
            "executable is not the unmodified exact checkout commit")
    catalog = run(binary, "commands")
    require(catalog.get("scope") == "selected-read-only-commands",
            "command catalog unexpectedly changed its declared scope")
    names = {c.get("name") for c in catalog.get("commands", [])}
    require("operations" not in names and "artifacts" not in names,
            "this reference explicitly distinguishes out-of-catalog commands")
    require({"inspect", "trace", "validate"} <= names,
            "expected limited read-only command catalog absent")

    require((original / SOURCE).is_file(), "expected standard CRUD input source absent")
    original_tree = source_tree_identity(original)
    with tempfile.TemporaryDirectory(prefix="tei-one-revision-") as temp:
        root = Path(temp)
        copied = root / "workspace"
        shutil.copytree(original, copied)
        baseline = observe(binary, copied, options.application)
        published = verify_reader(
            binary, copied, options.application, baseline, root / "published")

        moved = copied / SOURCE
        replacement = copied / MOVED
        original_bytes = moved.read_bytes()
        moved.rename(replacement)
        require(replacement.read_bytes() == original_bytes and not moved.exists(),
                "relocation must preserve Source bytes")
        changed = observe(binary, copied, options.application)
        require(baseline["input_tree_sha256"] != changed["input_tree_sha256"],
                "file relocation did not change the input Source path witness")
        require(baseline["revision"] != changed["revision"],
                "Source relocation was not visible in model revision")
        require(baseline["world_fingerprint"] == changed["world_fingerprint"],
                "World projection changed in this source-only relocation test")

        # The two independent results are each coherent at their own revisions,
        # but they cannot be joined to claim a single model revision.
        cross_command_join_qualified = (
            baseline["operations"]["modelRevision"]
            == changed["artifacts"]["manifest"]["inputRevision"])
        require(not cross_command_join_qualified,
                "old operation surface was incorrectly qualified with new artifact set")
        require(source_tree_identity(original) == original_tree,
                "original reference workspace was altered")
        print(json.dumps({
            "status": "PASS", "scope": "real TEI CLI; separate command invocations",
            "binary_revision": version["vcsRevision"],
            "catalog_scope": catalog["scope"],
            "before": {key: baseline[key] for key in (
                "input_tree_sha256", "revision", "world_fingerprint",
                "gaps", "findings", "manifest_complete", "manifest_digest")},
            "published": published,
            "after_relocation": {key: changed[key] for key in (
                "input_tree_sha256", "revision", "world_fingerprint",
                "gaps", "findings", "manifest_complete", "manifest_digest")},
            "old_operation_plus_new_artifact_qualified": cross_command_join_qualified,
            "limitations": ["no mid-command mutation or atomic-snapshot proof",
                "no Host/Provider/Skill/SHO/Owner acceptance",
                "workspace source copied to temporary directory"],
        }, ensure_ascii=False, sort_keys=True, indent=2))


if __name__ == "__main__":
    main()
