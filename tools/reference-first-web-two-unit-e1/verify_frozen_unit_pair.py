#!/usr/bin/env python3
"""Fail-closed, read-only E1 preflight for one Browser + frozen API pair.

This checks the exact published bytes, all available local diagnostic witnesses,
and Go build information in the executable. These are *not signed authorities*:
no Source Acceptance, Host permission, BuildArtifactLock, or production readiness
is granted by passing this diagnostic.
"""
import hashlib
import json
from pathlib import Path
import re
import subprocess
import sys

# Runtime preflight must never dirty a clean source checkout with __pycache__.
sys.dont_write_bytecode = True

from inspect_frozen_api_build import EXPECTED_TAGS, read_go_build_settings


SHA = re.compile(r"[0-9a-f]{64}\Z")
GIT = re.compile(r"[0-9a-f]{40}\Z")
SOURCE_SHA = re.compile(r"sha256:[0-9a-f]{64}\Z")
REQUIRED_FILES = {
    "measurement.json", "source-lineage.json",
    "browser/index.html", "browser/browser-command.js",
    "api/order-entry", "api/frozen-world.json",
}
KNOWN_DIAGNOSTIC_FILES = {"json-api-smoke.json"}
FLAGS_NOT_AUTHORIZED = (
    "sourceAcceptanceAuthorityVerified",
    "buildArtifactLockVerified",
    "runtimeModelAcceptanceAuthority",
    "productionAuthorizationVerified",
    "artifactAuthority",
    "fullE1BrowserAPIPackagingVerified",
)
BUILD_FLAGS_NOT_AUTHORIZED = (
    "builderArtifactAuthorityVerified",
    "buildArtifactLockVerified",
    "hermeticBuildVerified",
    "reproducibleBinaryVerified",
)


def object_without_duplicates(pairs):
    value = {}
    for key, entry in pairs:
        if key in value:
            raise ValueError("duplicate JSON key: " + key)
        value[key] = entry
    return value


def strict_json(path, limit):
    if path.stat().st_size > limit:
        raise ValueError("oversized JSON artifact: " + path.name)
    return json.loads(path.read_bytes(), object_pairs_hook=object_without_duplicates)


def sha256_file(path):
    sha = hashlib.sha256()
    size = 0
    with path.open("rb") as handle:
        for block in iter(lambda: handle.read(1024 * 1024), b""):
            size += len(block)
            sha.update(block)
    return sha.hexdigest(), size


def must_equal(actual, expected, label):
    if actual != expected:
        raise ValueError(label + " mismatch")


def require_false(doc, name):
    if doc.get(name) is not False:
        raise ValueError("unverified authority was asserted: " + name)


def preflight(root, build_info=None):
    root = Path(root).absolute()
    if root.is_symlink() or not root.is_dir():
        raise ValueError("artifact root must be a real directory")
    found = set()
    for entry in root.rglob("*"):
        if entry.is_symlink():
            raise ValueError("symlink in artifact tree")
        if not (entry.is_file() or entry.is_dir()):
            raise ValueError("unexpected artifact entry type")
        found.add(entry.relative_to(root).as_posix())
    directories = {"browser", "api"}
    allowed = REQUIRED_FILES | directories | KNOWN_DIAGNOSTIC_FILES
    if not REQUIRED_FILES.issubset(found) or not found.issubset(allowed):
        raise ValueError("missing or unexpected packaged unit member")
    if any(not (root / d).is_dir() for d in directories):
        raise ValueError("missing unit directory")
    m = strict_json(root / "measurement.json", 1024 * 1024)
    lineage = strict_json(root / "source-lineage.json", 1024 * 1024)
    frozen = strict_json(root / "api/frozen-world.json", 32 * 1024 * 1024)
    if not all(isinstance(x, dict) for x in (m, lineage, frozen)):
        raise ValueError("invalid observation envelope")
    must_equal(m.get("applicationId"), "order_entry", "Application")
    must_equal(m.get("browserTransport"), "same-origin-json", "transport")
    must_equal(m.get("apiWorldMode"), "digest-bound-frozen-world", "API mode")
    must_equal(m.get("apiBuildTags"),
               ["tei_order_only", "tei_api_only", "tei_frozen_world"], "API tags")
    if not isinstance(m.get("handoverBaselineId"), str) or not m["handoverBaselineId"]:
        raise ValueError("missing Handover B")
    source_revision = m.get("browserAndApiSourceRevision")
    if not isinstance(source_revision, str) or not GIT.fullmatch(source_revision):
        raise ValueError("invalid Source build revision")
    if not isinstance(m.get("baselineRepositoryRevision"), str) or not GIT.fullmatch(m["baselineRepositoryRevision"]):
        raise ValueError("invalid Handover R")
    for label, expected in (
        ("schema", "tei.reference.first-web-frozen-world/v1"),
        ("applicationId", "order_entry"),
        ("handoverBaselineId", m["handoverBaselineId"]),
        ("baselineRepositoryRevision", m["baselineRepositoryRevision"]),
        ("targetBuildRevision", source_revision),
    ):
        must_equal(frozen.get(label), expected, "frozen World " + label)
    if not isinstance(frozen.get("world"), dict):
        raise ValueError("missing frozen normalized World")
    for field in ("sourceModelRevision", "sourceWorldDerivedSHA256"):
        if not isinstance(m.get(field), str) or not SOURCE_SHA.fullmatch(m[field]):
            raise ValueError("invalid " + field)
    must_equal(m["sourceModelRevision"], lineage.get("sourceModelRevision"),
               "current Source revision")
    must_equal(m["sourceWorldDerivedSHA256"], lineage.get("derivedWorldSHA256"),
               "derived World")
    must_equal(m.get("currentAcceptedDecisionMatched"),
               lineage.get("currentAcceptedDecisionMatched"), "Decision lineage")
    if type(m["currentAcceptedDecisionMatched"]) is not bool:
        raise ValueError("invalid Decision match status")
    if not isinstance(lineage.get("currentAcceptedDecisionRecords"), list):
        raise ValueError("missing Decision record references")
    for field in FLAGS_NOT_AUTHORIZED:
        require_false(m, field)
    for field in ("sourceAcceptanceAuthorityVerified", "buildArtifactLockVerified",
                  "runtimeModelAcceptanceAuthority", "artifactAuthority"):
        require_false(lineage, field)
    checks = (
        ("browser/index.html", "browserSHA256", "browserBytes", 16 * 1024 * 1024),
        ("browser/browser-command.js", "browserScriptSHA256", "browserScriptBytes", 16 * 1024 * 1024),
        ("api/order-entry", "apiSHA256", "apiBytes", 256 * 1024 * 1024),
        ("api/frozen-world.json", "frozenWorldSHA256", "frozenWorldBytes", 32 * 1024 * 1024),
    )
    hashes = {}
    for member, key, size_key, max_bytes in checks:
        digest, size = sha256_file(root / member)
        if size <= 0 or size > max_bytes:
            raise ValueError("invalid packaged unit size: " + member)
        if not isinstance(m.get(key), str) or not SHA.fullmatch(m[key]):
            raise ValueError("invalid SHA-256 for " + member)
        must_equal(digest, m[key], member + " SHA-256")
        must_equal(size, m.get(size_key), member + " byte count")
        hashes[member] = digest
    must_equal(hashes["api/frozen-world.json"], lineage.get("frozenWorldSHA256"),
               "frozen World lineage SHA-256")
    build = m.get("apiBuildInputObservation")
    if not isinstance(build, dict):
        raise ValueError("missing API build observation")
    must_equal(build.get("kind"), "diagnostic-e1-api-build-input-observation", "build observation kind")
    must_equal(build.get("entrypoint"), "tei/tools/reference-first-web-target", "API entrypoint")
    must_equal(build.get("sourceCommit"), source_revision, "Go build Source commit")
    if not isinstance(build.get("sourceGitTree"), str) or not GIT.fullmatch(build["sourceGitTree"]):
        raise ValueError("invalid source Git tree")
    if not isinstance(build.get("goPackageClosureSHA256"), str) or not SHA.fullmatch(build["goPackageClosureSHA256"]):
        raise ValueError("missing Go package closure SHA-256")
    if type(build.get("goPackageCount")) is not int or build["goPackageCount"] <= 0:
        raise ValueError("invalid Go package closure count")
    must_equal(build.get("apiSHA256"), hashes["api/order-entry"], "built API SHA-256")
    must_equal(build.get("apiBytes"), m["apiBytes"], "built API byte count")
    for field in BUILD_FLAGS_NOT_AUTHORIZED:
        require_false(build, field)
    if build_info is None:
        result = subprocess.run(
            ["go", "version", "-m", str(root / "api/order-entry")],
            check=True, capture_output=True, text=True,
        )
        build_info = result.stdout
    actual_go = read_go_build_settings(build_info)
    must_equal(build.get("go"), actual_go, "actual embedded Go build options")
    must_equal(m.get("goVersion"), actual_go["goVersion"], "Browser/API Go toolchain")
    must_equal(set(actual_go["buildTags"]), EXPECTED_TAGS, "API build tags")
    if b'<form id="json-command" method="post" action="/scenario/submit"' not in (root / "browser/index.html").read_bytes():
        raise ValueError("missing JSON Browser transport form")
    return {
        "kind": "diagnostic-e1-two-unit-byte-and-build-preflight",
        "sourceRevision": source_revision,
        "sourceModelRevision": m["sourceModelRevision"],
        "browserSHA256": hashes["browser/index.html"],
        "apiSHA256": hashes["api/order-entry"],
        "frozenWorldSHA256": hashes["api/frozen-world.json"],
        "packageCount": build["goPackageCount"],
        "byteAndBuildConsistencyVerified": True,
        "sourceAcceptanceAuthorityVerified": False,
        "buildArtifactLockVerified": False,
        "productionAuthorizationVerified": False,
        "artifactAuthority": False,
    }


if __name__ == "__main__":
    if len(sys.argv) != 2:
        raise SystemExit("usage: python3 verify_frozen_unit_pair.py /absolute/artifact-root")
    print(json.dumps(preflight(sys.argv[1]), sort_keys=True))
