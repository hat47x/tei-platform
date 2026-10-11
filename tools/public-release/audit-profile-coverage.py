#!/usr/bin/env python3
"""Offline, source-pinned audit of a proposed public source path manifest.

Static coverage is never an assurance of runnable tests, disclosure clearance,
copyright permission, public-readiness or even complete non-Go fixture closure.
"""
import argparse
import pathlib
import re
import subprocess
import sys

SHA = re.compile(r"[0-9a-f]{40}\Z")
SAFE = re.compile(r"[A-Za-z0-9_./-]+\Z")
PROVENANCE = "# Based on exact private main revision: "
FIXTURE_PARTS = frozenset(("testdata", "fixture", "fixtures", "golden"))


def fail(message: str) -> None:
    raise SystemExit("FAIL: " + message)


def git(repo: pathlib.Path, *args: str) -> bytes:
    return subprocess.check_output(
        ["git", "-C", str(repo), *args], stderr=subprocess.PIPE
    )


def valid_path(path: str) -> bool:
    return (
        bool(SAFE.fullmatch(path))
        and not path.startswith("/")
        and "//" not in path
        and all(part not in ("", ".", "..", ".git") for part in path.split("/"))
    )


def load_manifest(path: pathlib.Path, sha: str) -> set[str]:
    entries: set[str] = set()
    found_sha: list[str] = []
    for raw in path.read_text(encoding="utf-8").splitlines():
        if raw.startswith(PROVENANCE):
            found_sha.append(raw[len(PROVENANCE) :])
            continue
        if raw.startswith("#") or not raw.strip():
            continue
        if raw != raw.strip() or not valid_path(raw):
            fail("invalid source path in manifest")
        if raw in entries:
            fail("duplicate source path in manifest")
        entries.add(raw)
    if found_sha != [sha]:
        fail("manifest provenance does not match the exact source commit")
    if not entries:
        fail("manifest is empty")
    return entries


def snapshot_blobs(repo: pathlib.Path, sha: str) -> dict[str, str]:
    result: dict[str, str] = {}
    raw = git(repo, "ls-tree", "-r", "-z", "--full-tree", sha)
    for entry in raw.split(b"\0"):
        if not entry:
            continue
        metadata, raw_path = entry.split(b"\t", 1)
        mode, kind, _oid = metadata.decode("ascii").split()
        path = raw_path.decode("utf-8")
        if kind == "blob" and mode in ("100644", "100755"):
            result[path] = mode
    return result


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("repository", type=pathlib.Path)
    parser.add_argument("source_sha")
    parser.add_argument("manifest", type=pathlib.Path)
    parser.add_argument("--profile", choices=("development", "distribution"), required=True)
    args = parser.parse_args()

    if not SHA.fullmatch(args.source_sha):
        fail("expected exact 40-character lowercase SHA")
    repo = pathlib.Path(
        git(args.repository.resolve(strict=True), "rev-parse", "--show-toplevel").decode().strip()
    ).resolve()
    actual = git(repo, "rev-parse", "--verify", args.source_sha + "^{commit}").decode().strip()
    if actual != args.source_sha:
        fail("source commit does not exist")

    reviewed = load_manifest(args.manifest.resolve(strict=True), args.source_sha)
    source = snapshot_blobs(repo, args.source_sha)
    absent = sorted(reviewed.difference(source))
    if absent:
        fail(f"manifest contains {len(absent)} paths absent from source, first: {absent[0]}")
    missing_basics = set(("README.md", "LICENSE", "go.mod", "go.sum", "cmd/tei/main.go")).difference(reviewed)
    if missing_basics:
        fail("essential source paths missing: " + ", ".join(sorted(missing_basics)))
    for path in reviewed:
        if source[path] not in ("100644", "100755"):
            fail("source path is not an ordinary regular file")
    readme = git(repo, "show", args.source_sha + ":README.md").decode("utf-8").splitlines()[:15]
    preamble = "\n".join(readme)
    if not ("生成AI" in preamble and "人的レビュー" in preamble):
        fail("AI assistance and lack-of-human-review notice absent from README preamble")

    # Refuse case-fold collisions in an explicit source selection, to avoid
    # losing tracked files during Windows/macOS source candidate extraction.
    cases: set[str] = set()
    for path in reviewed:
        folded = path.casefold()
        if folded in cases:
            fail("selected paths collide on a case-insensitive filesystem")
        cases.add(folded)

    go_paths = {p for p in source if p.endswith(".go")}
    test_paths = {p for p in go_paths if p.endswith("_test.go")}
    fixtures = {
        p for p in source
        if any(seg in FIXTURE_PARTS for seg in p.split("/"))
    }
    missing_go = go_paths.difference(reviewed)
    missing_tests = test_paths.difference(reviewed)
    missing_fixtures = fixtures.difference(reviewed)
    print(f"source={args.source_sha} profile={args.profile} tracked_regular={len(source)}")
    print(f"selected={len(reviewed)} excluded={len(source)-len(reviewed)}")
    print(f"go_sources_selected={len(go_paths)-len(missing_go)}/{len(go_paths)}")
    print(f"go_tests_selected={len(test_paths)-len(missing_tests)}/{len(test_paths)}")
    print(f"conventional_fixtures_selected={len(fixtures)-len(missing_fixtures)}/{len(fixtures)}")
    if args.profile == "development" and (missing_go or missing_tests or missing_fixtures):
        fail("development profile omits Go source/tests or conventional fixtures")
    if args.profile == "distribution" and missing_tests:
        print("WARNING: Distribution is not equivalent to full original Go test coverage.")
    print("STATIC_FILE_COVERAGE_ONLY: PASS")
    print("NOT_A_PUBLICATION_APPROVAL: tests, build, license, secrets and link closure not verified.")


if __name__ == "__main__":
    main()
