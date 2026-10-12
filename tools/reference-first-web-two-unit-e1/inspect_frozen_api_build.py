#!/usr/bin/env python3
"""Observe exact Go API build inputs; never issue a BuildArtifactLock.

This diagnostic probes the *built executable* with the Go tool's build-info
reader and the actual build dependency listing. All values describe a local
build, not Source Acceptance, Builder trust or deployment authorization.
"""
import argparse
import hashlib
import json
from pathlib import Path
import re
import subprocess


EXPECTED_TAGS = frozenset({"tei_order_only", "tei_api_only", "tei_frozen_world"})
EXPECTED_PACKAGE = "tei/tools/reference-first-web-target"
HEX40 = re.compile(r"[0-9a-f]{40}\Z")


def read_go_build_settings(output):
    lines = output.splitlines()
    if not lines:
        raise ValueError("missing Go build information")
    first = lines[0].split()
    if len(first) < 2 or not re.fullmatch(r"go[0-9]+\.[0-9]+(?:\.[0-9]+)?", first[-1]):
        raise ValueError("unknown Go binary version")
    settings = {}
    for line in lines[1:]:
        parts = line.strip().split("\t")
        if len(parts) < 2 or parts[0] != "build":
            continue
        for field in parts[1:]:
            if "=" not in field:
                raise ValueError("invalid Go build setting")
            key, value = field.split("=", 1)
            if key in settings:
                raise ValueError(f"duplicate Go build setting {key}")
            settings[key] = value
    # A failed/disabled build-info extraction must not silently turn into
    # a claim that the toolchain options were correct.
    expected = {
        "GOOS": "linux",
        "GOARCH": "amd64",
        "GOAMD64": "v1",
        "CGO_ENABLED": "0",
        "-trimpath": "true",
    }
    for key, want in expected.items():
        if settings.get(key) != want:
            raise ValueError(f"Go binary build setting mismatch: {key}")
    tags = settings.get("-tags", "")
    actual_tags = tags.split(",")
    if len(actual_tags) != len(EXPECTED_TAGS) or set(actual_tags) != EXPECTED_TAGS:
        raise ValueError("Go binary build tags differ from frozen API selection")
    return {
        "goVersion": first[-1],
        "goos": settings["GOOS"],
        "goarch": settings["GOARCH"],
        "goamd64": settings["GOAMD64"],
        "cgoEnabled": False,
        "trimpath": True,
        "buildTags": sorted(EXPECTED_TAGS),
    }


def observe(binary_path, package_path, git_tree, git_revision, build_info):
    if not HEX40.fullmatch(git_tree) or not HEX40.fullmatch(git_revision):
        raise ValueError("Git source tree and revision must be exact hex40 IDs")
    binary = Path(binary_path)
    if binary.is_symlink() or not binary.is_file():
        raise ValueError("API executable must be a real file")
    packages = Path(package_path).read_bytes()
    names = packages.decode("utf-8").splitlines()
    if not names or names[-1] != EXPECTED_PACKAGE:
        raise ValueError("API Go dependency closure is incomplete")
    if any(name != name.strip() or not name or name.startswith("#") for name in names):
        raise ValueError("invalid dependency package name")
    if len(set(names)) != len(names):
        raise ValueError("duplicate dependency package")
    required = {"tei/internal/ir", "tei/internal/driver", EXPECTED_PACKAGE}
    if not required.issubset(names):
        raise ValueError("missing Go runtime implementation package")
    forbidden = {"tei/internal/workspace", "tei/internal/syntax"}
    if forbidden.intersection(names):
        raise ValueError("build-only TEI source packages reached API binary")
    if packages != ("\n".join(names) + "\n").encode():
        raise ValueError("dependency listing is not canonical newline text")
    binary_bytes = binary.read_bytes()
    if not binary_bytes:
        raise ValueError("missing binary bytes")
    settings = read_go_build_settings(build_info)
    return {
        "kind": "diagnostic-e1-api-build-input-observation",
        "sourceCommit": git_revision,
        "sourceGitTree": git_tree,
        "entrypoint": EXPECTED_PACKAGE,
        "go": settings,
        "goPackageClosureSHA256": hashlib.sha256(packages).hexdigest(),
        "goPackageCount": len(names),
        "apiSHA256": hashlib.sha256(binary_bytes).hexdigest(),
        "apiBytes": len(binary_bytes),
        "builderArtifactAuthorityVerified": False,
        "buildArtifactLockVerified": False,
        "hermeticBuildVerified": False,
        "reproducibleBinaryVerified": False,
    }


def main():
    args = argparse.ArgumentParser()
    args.add_argument("--binary", required=True)
    args.add_argument("--packages", required=True)
    args.add_argument("--git-tree", required=True)
    args.add_argument("--git-revision", required=True)
    args.add_argument("--out", required=True)
    opts = args.parse_args()
    result = subprocess.run(
        ["go", "version", "-m", opts.binary],
        check=True, capture_output=True, text=True,
    )
    observation = observe(
        opts.binary, opts.packages, opts.git_tree, opts.git_revision, result.stdout,
    )
    Path(opts.out).write_text(
        json.dumps(observation, sort_keys=True, indent=2) + "\n", encoding="utf-8",
    )


if __name__ == "__main__":
    main()
