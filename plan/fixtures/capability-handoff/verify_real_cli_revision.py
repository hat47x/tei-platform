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

from publication_path_readback import (verify_ledger_publication_paths,
                                       verify_pinned_ledger_publication_paths)

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
    # This reference is restricted to the public standard CRUD ledger fixture.
    # Its expected path layout is independent of the Writer's naming function.
    # Checking a Body multiset alone would accept two swapped table file bodies.
    require(application == "ledger",
            "path/member witness is only qualified for the ledger reference")
    readback = verify_pinned_ledger_publication_paths(
        baseline["artifacts"]["artifacts"], directory, manifest_path.read_bytes())
    return {"reader_validated_bytes": True,
            "manifest_file_sha256": hashlib.sha256(manifest_path.read_bytes()).hexdigest(),
            **readback}


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
        # Pin the R1 published manifest as actual bytes, not a reconstructed
        # digest or the Writer's expected in-memory object.
        r1_pin = (root / "published" / "manifest.json").read_bytes()

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

        # #2604 AP-06/09: A second *real CLI process* publishes a new
        # generation to the SAME directory. The independent Python reader
        # must reject the previously pinned R1 even when body bytes coincide.
        require(baseline["manifest_digest"] != changed["manifest_digest"],
                "two source revisions must not share a manifest generation")
        r2_manifest = run(binary, "artifacts", "-workspace", copied,
                          "-application", options.application,
                          "-out", root / "published")
        require(r2_manifest == changed["artifacts"]["manifest"],
                "R2 publisher stdout did not match its read-only manifest")
        r2_pin = (root / "published" / "manifest.json").read_bytes()
        require(r2_pin != r1_pin, "R2 republish did not change on-disk manifest")
        require(json.loads(r2_pin)["manifest"] == r2_manifest,
                "R2 manifest bytes differ from expected R2 generation")
        try:
            verify_pinned_ledger_publication_paths(
                baseline["artifacts"]["artifacts"], root / "published", r1_pin)
        except AssertionError as exc:
            require("pinned manifest changed before member read" in str(exc),
                    f"R1 pin refused for an unexpected reason: {exc}")
            r1_pin_rejected_after_republish = True
        else:
            raise AssertionError("stale R1 pin was accepted after R2 publication")
        r2_read = verify_pinned_ledger_publication_paths(
            changed["artifacts"]["artifacts"], root / "published", r2_pin)
        same_body_paths = sorted(
            path for path, digest in published["member_body_sha256_by_path"].items()
            if r2_read["member_body_sha256_by_path"].get(path) == digest)

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
            "republished": {
                "publisher": "same actual Go CLI, second separate invocation",
                "reader": "independent Python process; after each publication",
                "r1_pin_rejected_after_republish": r1_pin_rejected_after_republish,
                "r2_current_pin_readable": True,
                "r2_member_count": r2_read["file_count"],
                "unchanged_body_paths": same_body_paths,
                "current_manifest_sha256": hashlib.sha256(r2_pin).hexdigest(),
            },
            "after_relocation": {key: changed[key] for key in (
                "input_tree_sha256", "revision", "world_fingerprint",
                "gaps", "findings", "manifest_complete", "manifest_digest")},
            "old_operation_plus_new_artifact_qualified": cross_command_join_qualified,
            "limitations": ["no mid-command mutation or atomic-snapshot proof",
                "no Host/Provider/Skill/SHO/Owner acceptance",
                "workspace source copied to temporary directory",
                "ledger fixture path/byte witness only; not full Artifact envelope verification",
                "no atomic multi-file publication or independent recipient acceptance",
                "two sequential CLI publish invocations; no concurrent writer/reader race",
                "pre/post manifest pin check cannot prove retention, prevent ABA, or GC"],
        }, ensure_ascii=False, sort_keys=True, indent=2))


if __name__ == "__main__":
    main()
