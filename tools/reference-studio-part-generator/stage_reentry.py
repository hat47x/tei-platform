#!/usr/bin/env python3
"""Observe Stage1 -> Stage2 re-entry for one generated TEI Studio part.

Every build and every generation below is started by a TEI binary through the
existing `tei self-build -child ... -artifact ...` entry, which records the
invoker and the invoked process separately. This script only prepares the
Stage0 seed, starts those TEI processes, and compares what they recorded.

  stage0  external Go seed builds cmd/tei                      -> tei0 (not driven)
  stage1  tei0 drives: Go build of cmd/tei                      -> tei1
          tei0 drives: Go build of the part generator           -> gen1
          tei0 drives: gen1 over the contract Source            -> part1
  stage2  tei1 drives: Go build of cmd/tei                      -> tei2
          tei1 drives: Go build of the part generator           -> gen2
          tei1 drives: gen2 over the same contract Source       -> part2

Re-entry is observed only when the Stage2 invoker is the Stage1 artifact, the
Stage2 part was produced by the generator Stage2 rebuilt, and the bytes agree.
Two launches of `go build` by this script would not satisfy any of these.

It claims no Seed Trust, acceptance, Runtime Authority, or Desktop use. The
Node/esbuild bundling and the Wails build are not driven here.
"""
from __future__ import annotations

import argparse
import hashlib
import json
import os
import shutil
import subprocess
import sys
from pathlib import Path

SCHEMA = "tei.reference.studio-part-stage-reentry/v0"
WORKBENCH = Path("reference/tei-studio-workbench")
TRANSFORM = WORKBENCH / "builder-sources" / "inspection-reader-v1.transform.json"
CONTRACT = WORKBENCH / "inspection-reader.contract.json"
COMMITTED_PART = WORKBENCH / "inspection-reader.generated.ts"


def digest_bytes(raw: bytes) -> str:
    return "sha256:" + hashlib.sha256(raw).hexdigest()


def digest_file(path: Path) -> str:
    return digest_bytes(path.read_bytes())


def drive(tei: Path, child: Path, child_args: list[str], artifact: Path) -> dict:
    """Start one TEI process and return what it recorded about its child."""
    completed = subprocess.run(
        [str(tei), "self-build", "-child", str(child), "-child-arg", json.dumps(child_args), "-artifact", str(artifact)],
        capture_output=True, text=True, check=False,
    )
    if completed.returncode != 0:
        raise RuntimeError(f"{tei.name} self-build did not complete the observation: {completed.stderr.strip()}")
    evidence = json.loads(completed.stdout)
    return {
        "invokerDigest": evidence.get("invokerDigest"),
        "invokedExecutableDigest": evidence.get("invoked", {}).get("executableDigest"),
        "invokedExitCode": evidence.get("invoked", {}).get("exitCode"),
        "artifactDigest": evidence.get("artifactCrossCheckedDigest"),
        "artifactCrossCheckMatched": evidence.get("artifactCrossCheckMatched"),
        "failureCause": evidence.get("failureCause"),
        # What actually sits on disk, read by this script and not by TEI.
        "artifactOnDisk": digest_file(artifact) if artifact.is_file() else None,
    }


def run_stage(tei: Path, go_child: Path, repo: Path, supply: Path, work: Path) -> dict:
    work.mkdir(parents=True)

    def go_build(package: str, name: str) -> tuple[Path, dict]:
        out = work / name
        return out, drive(tei, go_child, [
            "-repo", str(repo), "-package", package, "-supply", str(supply),
            "-artifact-out", str(out), "-state", str(work / (name + "-state")),
        ], out)

    tei_out, tei_build = go_build("./cmd/tei", "tei")
    generator, generator_build = go_build("./tools/reference-studio-part-generator", "generator")
    part = work / "inspection-reader.generated.ts"
    generation = (
        drive(tei, generator, [
            "-transform", str(repo / TRANSFORM), "-contract", str(repo / CONTRACT),
            "-artifact-out", str(part), "-state", str(work / "part-state"),
        ], part)
        if generator.is_file() else {"failureCause": "the generator was not built"}
    )
    return {
        "invokerPath": str(tei),
        "invokerOnDisk": digest_file(tei),
        "teiBuild": tei_build,
        "generatorBuild": generator_build,
        "partGeneration": generation,
        "teiArtifactPath": str(tei_out),
    }


def judge(stage1: dict, stage2: dict, committed_part: str) -> dict:
    """Each judgement is separate; agreement of bytes alone is never re-entry."""
    steps = ("teiBuild", "generatorBuild", "partGeneration")

    def clean(stage: dict) -> bool:
        return all(
            stage[s].get("invokedExitCode") == 0 and stage[s].get("artifactCrossCheckMatched") is True
            and not stage[s].get("failureCause") and stage[s].get("artifactDigest") == stage[s].get("artifactOnDisk")
            for s in steps
        )

    def driven_by(stage: dict) -> bool:
        return all(stage[s].get("invokerDigest") == stage["invokerOnDisk"] for s in steps)

    stage1_tei = stage1["teiBuild"].get("artifactDigest")
    judgement = {
        "everyStepCompletedAndCrossChecked": clean(stage1) and clean(stage2),
        "eachStageDrivenByItsOwnTei": driven_by(stage1) and driven_by(stage2),
        "stage1TeiArtifactIsStage2Invoker": stage1_tei is not None and stage2["invokerOnDisk"] == stage1_tei
        and all(stage2[s].get("invokerDigest") == stage1_tei for s in steps),
        "stage2PartMadeByStage2RebuiltGenerator": stage2["generatorBuild"].get("artifactDigest") is not None
        and stage2["partGeneration"].get("invokedExecutableDigest") == stage2["generatorBuild"].get("artifactDigest"),
        "stage1PartMadeByStage1BuiltGenerator": stage1["generatorBuild"].get("artifactDigest") is not None
        and stage1["partGeneration"].get("invokedExecutableDigest") == stage1["generatorBuild"].get("artifactDigest"),
        "generatorsEqual": stage1["generatorBuild"].get("artifactDigest") == stage2["generatorBuild"].get("artifactDigest"),
        "teiArtifactsEqual": stage1_tei == stage2["teiBuild"].get("artifactDigest"),
        "partsEqual": stage1["partGeneration"].get("artifactDigest") is not None
        and stage1["partGeneration"].get("artifactDigest") == stage2["partGeneration"].get("artifactDigest"),
        "stage2PartIsTheCommittedPart": stage2["partGeneration"].get("artifactDigest") == committed_part,
    }
    judgement["reentryObserved"] = all(judgement.values())
    return judgement


def self_test() -> None:
    def step(invoker: str, artifact: str, invoked: str = "child") -> dict:
        return {"invokerDigest": invoker, "invokedExecutableDigest": invoked, "invokedExitCode": 0,
                "artifactDigest": artifact, "artifactCrossCheckMatched": True, "failureCause": None,
                "artifactOnDisk": artifact}

    def stage(invoker: str) -> dict:
        return {"invokerOnDisk": invoker, "teiBuild": step(invoker, "tei1"),
                "generatorBuild": step(invoker, "gen"), "partGeneration": step(invoker, "part", "gen")}

    good = judge(stage("tei0"), stage("tei1"), "part")
    assert good["reentryObserved"], good
    # Same bytes produced twice by the Stage0 TEI: equal artifacts, no re-entry.
    twice = judge(stage("tei0"), stage("tei0"), "part")
    assert twice["partsEqual"] and not twice["stage1TeiArtifactIsStage2Invoker"] and not twice["reentryObserved"], twice
    # Stage2 part made by a generator Stage2 did not rebuild.
    foreign = stage("tei1")
    foreign["partGeneration"] = step("tei1", "part", "some-other-generator")
    other = judge(stage("tei0"), foreign, "part")
    assert not other["stage2PartMadeByStage2RebuiltGenerator"] and not other["reentryObserved"], other
    # Stage2 regenerated something else than the part the Workbench is built from.
    stale = judge(stage("tei0"), stage("tei1"), "another-part")
    assert not stale["stage2PartIsTheCommittedPart"] and not stale["reentryObserved"], stale
    # A child that reports a digest the disk does not hold.
    lying = stage("tei1")
    lying["partGeneration"]["artifactOnDisk"] = "different"
    assert not judge(stage("tei0"), lying, "part")["reentryObserved"]
    print("self-test: 5 judgements behaved as specified")


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument("--self-test", action="store_true", help="check the judgement without building anything")
    parser.add_argument("--repo", type=Path, help="clean checkout to build from")
    parser.add_argument("--supply", type=Path, help="Go module supply (a GOPROXY file root, e.g. $GOMODCACHE/cache/download)")
    parser.add_argument("--work", type=Path, help="new directory for every stage's outputs; must not exist")
    parser.add_argument("--go", default="go", help="Stage0 seed Go executable")
    args = parser.parse_args()
    if args.self_test:
        self_test()
        return 0
    if not (args.repo and args.supply and args.work):
        parser.error("--repo, --supply and --work are required")
    repo, supply, work = args.repo.resolve(), args.supply.resolve(), args.work.resolve()
    if work.exists():
        parser.error(f"--work already exists: {work}")
    go = shutil.which(args.go)
    if go is None:
        parser.error(f"Stage0 seed Go executable not found: {args.go}")
    stage0 = work / "stage0"
    stage0.mkdir(parents=True)

    def seed_build(package: str, name: str) -> Path:
        out = stage0 / name
        subprocess.run([go, "build", "-trimpath", "-o", str(out), package], cwd=repo, check=True,
                       env={**os.environ, "CGO_ENABLED": "0", "GOFLAGS": "", "GOTOOLCHAIN": "local"})
        return out

    tei0 = seed_build("./cmd/tei", "tei")
    go_child = seed_build("./tools/reference-go-build-with-supply", "go-build-with-supply")

    stage1 = run_stage(tei0, go_child, repo, supply, work / "stage1")
    tei1 = Path(stage1["teiArtifactPath"])
    if not tei1.is_file():
        raise RuntimeError(f"stage1 did not produce a TEI binary: {stage1['teiBuild'].get('failureCause')}")
    # The Stage2 driver is the Stage1 artifact itself, at the path Stage1 wrote it.
    stage2 = run_stage(tei1, go_child, repo, supply, work / "stage2")

    committed = digest_bytes((repo / COMMITTED_PART).read_bytes().replace(b"\r\n", b"\n"))
    result = {
        "schemaVersion": SCHEMA,
        "sources": {
            "repoHead": subprocess.run(["git", "rev-parse", "HEAD"], cwd=repo, capture_output=True, text=True).stdout.strip() or None,
            "generatorSourceBytes": digest_file(repo / TRANSFORM),
            "contractSourceBytes": digest_file(repo / CONTRACT),
            "committedPart": committed,
        },
        "stage0": {
            "nature": "external Go seed; not a driven process",
            "goExecutable": go,
            "goExecutableDigest": digest_file(Path(go)),
            "goVersion": subprocess.run([go, "version"], capture_output=True, text=True).stdout.strip(),
            "teiDigest": digest_file(tei0),
            "goBuildChildDigest": digest_file(go_child),
        },
        "stage1": stage1,
        "stage2": stage2,
        "judgement": judge(stage1, stage2, committed),
        "claimBoundary": "one host, one toolchain, one supply; the Go build child is the Stage0 seed's in both stages; "
                         "no Seed Trust, acceptance, Runtime Authority, bundling, Wails build or Desktop use is claimed",
    }
    json.dump(result, sys.stdout, indent=2)
    print()
    return 0 if result["judgement"]["reentryObserved"] else 1


if __name__ == "__main__":
    sys.exit(main())
