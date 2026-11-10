"""Bounded, fixture-owned path/byte readback for the standard CRUD ledger.

The map is an independent expected layout for this one public reference
fixture. It is NOT a universal artifact naming rule or a publication manifest.
Artifact envelope fingerprints are intentionally not treated as content SHA256.
"""

import hashlib
from pathlib import Path


LEDGER_EXPECTED_LAYOUT = {
    ("table-definition", "table public.notes"): "table-public-notes.md",
    ("table-definition", "table public.orders"): "table-public-orders.md",
    ("table-definition", "table public.order_archive"): "table-public-order_archive.md",
    ("function-definition", "application ledger"): "function-ledger.md",
    ("operation-surface", None): "operation-surface.json",
}


def verify_ledger_publication_paths(members, directory):
    """Check exact logical member->relative path->disk bytes in a *fresh* dir.

    The caller already compared the real CLI's full ArtifactSet/Manifest and
    must not reinterpret this as an atomic publish, independent Source closure,
    envelope verification from disk, or business acceptance.
    """
    directory = Path(directory)
    expected_by_path = {}
    for member in members:
        if not isinstance(member, dict):
            raise AssertionError("artifact member is not an object")
        content = member.get("content")
        if not isinstance(content, str):
            raise AssertionError("artifact member has no content")
        kind = member.get("kind")
        header = None if kind == "operation-surface" else content.split("\n", 1)[0].strip()
        identity = (kind, header)
        if identity not in LEDGER_EXPECTED_LAYOUT:
            raise AssertionError(f"unsupported ledger member role: {identity!r}")
        path = LEDGER_EXPECTED_LAYOUT[identity]
        if path in expected_by_path:
            raise AssertionError(f"duplicate ledger member for {path}")
        expected_by_path[path] = content.encode("utf-8")

    if set(expected_by_path) != set(LEDGER_EXPECTED_LAYOUT.values()):
        raise AssertionError("ledger reference required-member set is incomplete")

    entries = list(directory.iterdir())
    if any(entry.is_symlink() or not entry.is_file() for entry in entries):
        raise AssertionError("unexpected directory or symlink in published output")
    actual_paths = {entry.name for entry in entries if entry.name != "manifest.json"}
    if actual_paths != set(expected_by_path):
        raise AssertionError(
            f"published ledger path set mismatch: missing={sorted(set(expected_by_path)-actual_paths)}, "
            f"unexpected={sorted(actual_paths-set(expected_by_path))}")

    body_digests = {}
    for path, expected in sorted(expected_by_path.items()):
        actual = (directory / path).read_bytes()
        if actual != expected:
            raise AssertionError(f"published member path-to-byte mismatch: {path}")
        body_digests[path] = hashlib.sha256(actual).hexdigest()
    return {"file_count": len(body_digests),
            "member_binding": "ledger-fixture-path-to-bytes-only",
            "member_body_sha256_by_path": body_digests}


def verify_pinned_ledger_publication_paths(members, directory, pinned_manifest_bytes):
    """Read the *pinned* ledger generation, refusing a changed manifest.

    This detects a manifest change before/during member read in a bounded
    sequential reference. Two reads do NOT establish filesystem atomicity,
    prevent manifest ABA, retain older body files, or establish publisher
    authority. Byte identities still come from the fixed ledger layout.
    """
    if not isinstance(pinned_manifest_bytes, bytes) or not pinned_manifest_bytes:
        raise AssertionError("pinned manifest bytes are required")
    directory = Path(directory)
    manifest_path = directory / "manifest.json"
    if manifest_path.is_symlink() or not manifest_path.is_file():
        raise AssertionError("pinned manifest is not a regular file")
    if manifest_path.read_bytes() != pinned_manifest_bytes:
        raise AssertionError("pinned manifest changed before member read")
    evidence = verify_ledger_publication_paths(members, directory)
    if manifest_path.read_bytes() != pinned_manifest_bytes:
        raise AssertionError("pinned manifest changed during member read")
    return evidence
