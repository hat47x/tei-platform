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

With --bundle-reader each stage drives the existing Node/esbuild child through
its TEI invoker, consuming that stage's just-generated Reader Part and recording
a separate JS Bundle artifact. It does not feed this new bundle to Wails yet.

With --stage-desktop-bundle, a separate worktree at the exact same Git HEAD
receives the new Bundle as the sole tracked Source difference for each Stage.
That staged checkout, not the original repository, supplies Wails go:embed.
Both desktop binaries must contain the new byte sequence and reject the
committed Bundle. This mode also requires changed tei.authoring/v1 Data.

With --studio-desktop each stage also drives a Go build of the Studio Wails
Desktop (the nested module cmd/tei-studio) with the same child:

  stage1  tei0 drives: Go build of cmd/tei-studio               -> studio1
  stage2  tei1 drives: Go build of cmd/tei-studio               -> studio2

and with --desktop-smoke the Stage2 Desktop is then started as a new process
and driven through the existing WebView2 smoke, which requires the generated
Inspector reader of the current contract to render real Inspect results.
Both are Windows-only: the Wails Desktop builds there with CGO disabled.

Re-entry is observed only when the Stage2 invoker is the Stage1 artifact, the
Stage2 part was produced by the generator Stage2 rebuilt, and the bytes agree.
Two launches of `go build` by this script would not satisfy any of these.

It claims no Seed Trust, acceptance or Runtime Authority. In the default
Desktop mode the Node/esbuild bundle remains the committed one. The separate
--bundle-reader mode makes a real new Bundle; only the explicit
--stage-desktop-bundle mode also builds a Desktop containing it. The
editor/source-write/meaning acceptance and formal Build Intent remain separate.
"""
from __future__ import annotations

import argparse
import hashlib
import json
import os
import re
import shutil
import subprocess
import sys
from pathlib import Path

SCHEMA = "tei.reference.studio-part-stage-reentry/v0"
WORKBENCH = Path("reference/tei-studio-workbench")
TRANSFORM = WORKBENCH / "builder-sources" / "inspection-reader-v1.transform.json"
CONTRACT = WORKBENCH / "inspection-reader.contract.json"
COMMITTED_PART = WORKBENCH / "inspection-reader.generated.ts"
STUDIO_MODULE = Path("cmd/tei-studio")
COMMITTED_BUNDLE = STUDIO_MODULE / "assets" / "workbench" / "main.js"
BUNDLE_CHILD = Path("tools/reference-studio-static-composition/build-generated-workbench.mjs")
SMOKE = STUDIO_MODULE / "test" / "desktop_webview_smoke.py"
EXE = ".exe" if os.name == "nt" else ""
CONTRACT_DIGEST = re.compile(rb'contractDigest: "(sha256:[0-9a-f]{64})"')


def digest_bytes(raw: bytes) -> str:
    return "sha256:" + hashlib.sha256(raw).hexdigest()


def digest_file(path: Path) -> str:
    return digest_bytes(path.read_bytes())


def authoring_source_digest(root: Path) -> str:
    """Fingerprint paths and exact bytes, not just the selected Data's text."""
    if not root.is_dir() or not (root / "tei.workspace.yaml").is_file():
        raise RuntimeError("authoring Workspace with tei.workspace.yaml is required")
    digest = hashlib.sha256()
    found = 0
    for path in sorted(root.rglob("*")):
        if path.is_symlink():
            raise RuntimeError("authoring Workspace symlinks are not accepted")
        if not path.is_file():
            continue
        found += 1
        digest.update(path.relative_to(root).as_posix().encode("utf-8"))
        digest.update(b"\0")
        digest.update(str(path.stat().st_size).encode("ascii"))
        digest.update(b"\0")
        digest.update(path.read_bytes())
        digest.update(b"\n")
    if found < 2:
        raise RuntimeError("authoring Workspace has no model Source")
    return "sha256:" + digest.hexdigest()


def git_head(root: Path) -> str:
    result = subprocess.run(["git", "-C", str(root), "rev-parse", "HEAD"],
                            capture_output=True, text=True, encoding="utf-8", check=True)
    return result.stdout.strip()


def drive(tei: Path, child: Path, child_args: list[str], artifact: Path) -> dict:
    """Start one TEI process and return what it recorded about its child."""
    completed = subprocess.run(
        [str(tei), "self-build", "-child", str(child), "-child-arg", json.dumps(child_args), "-artifact", str(artifact)],
        capture_output=True, text=True, encoding="utf-8", check=False,
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


def run_stage(tei: Path, go_child: Path, repo: Path, supply: Path, work: Path,
              studio_desktop: bool = False, node_child: Path | None = None,
              authoring_root: Path | None = None, authoring_data: str | None = None,
              stage_desktop_bundle: bool = False, expected_head: str | None = None) -> dict:
    work.mkdir(parents=True)

    def go_build(package: str, name: str, module: Path = repo) -> tuple[Path, dict]:
        out = work / (name + EXE)
        return out, drive(tei, go_child, [
            "-repo", str(module), "-package", package, "-supply", str(supply),
            "-artifact-out", str(out), "-state", str(work / (name + "-state")),
        ], out)

    tei_out, tei_build = go_build("./cmd/tei", "tei")
    generator, generator_build = go_build("./tools/reference-studio-part-generator", "generator")
    part = work / "inspection-reader.generated.ts"
    authoring_args = (["-authoring-root", str(authoring_root),
                       "-authoring-data", str(authoring_data)]
                      if authoring_root is not None else [])
    generation = (
        drive(tei, generator, [
            "-transform", str(repo / TRANSFORM), "-contract", str(repo / CONTRACT),
            "-artifact-out", str(part), "-state", str(work / "part-state"),
            *authoring_args,
        ], part)
        if generator.is_file() else {"failureCause": "the generator was not built"}
    )
    stage = {
        "invokerPath": str(tei),
        "invokerOnDisk": digest_file(tei),
        "teiBuild": tei_build,
        "generatorBuild": generator_build,
        "partGeneration": generation,
        "teiArtifactPath": str(tei_out),
        "partPath": str(part),
    }
    if authoring_root is not None:
        stage["authoringSourceDigest"] = authoring_source_digest(authoring_root)
        stage["authoringData"] = authoring_data
    if node_child is not None:
        bundle = work / "workbench.generated.js"
        # This is the existing N2 child; the TEI invoker, not Python, launches
        # Node and checks the new artifact's digest against the child receipt.
        stage["bundleGeneration"] = drive(tei, node_child, [
            str(repo / BUNDLE_CHILD), "--inspection-reader", str(part),
            "--artifact-out", str(bundle),
        ], bundle)
        stage["bundlePath"] = str(bundle)
        stage["bundleNodeDigest"] = digest_file(node_child)
        stage["bundleBuilderSourceDigest"] = digest_file(repo / BUNDLE_CHILD)
        stage["partContract"] = one_contract(part)
        stage["bundleContract"] = one_contract(bundle)
    if studio_desktop:
        if stage_desktop_bundle:
            # An isolated Git worktree preserves the exact HEAD and nested
            # go.mod relative replace, without writing to tracked workspace
            # assets. Only the generated main.js is changed in that worktree.
            checkout = work / "stage-desktop-source"
            subprocess.run(["git", "-C", str(repo), "worktree", "add", "--detach",
                            str(checkout), expected_head], capture_output=True,
                           text=True, encoding="utf-8", check=True)
            try:
                checkout_head = git_head(checkout)
                if checkout_head != expected_head:
                    raise RuntimeError("Stage Desktop checkout revision mismatch")
                source_bundle = checkout / COMMITTED_BUNDLE
                committed_digest = digest_file(repo / COMMITTED_BUNDLE)
                if digest_file(source_bundle) != committed_digest:
                    raise RuntimeError("Stage Desktop checkout has a different committed Bundle")
                bundle = Path(stage["bundlePath"])
                selected = digest_file(bundle)
                if selected == committed_digest or stage["partContract"] != stage["bundleContract"]:
                    raise RuntimeError("Stage Desktop must receive a changed, contract-matched Bundle")
                source_bundle.write_bytes(bundle.read_bytes())
                if digest_file(source_bundle) != selected:
                    raise RuntimeError("Stage Desktop staged Bundle differs from the Builder artifact")
                changed = subprocess.run(["git", "-C", str(checkout),
                                          "diff", "--name-only"], check=True,
                                         capture_output=True, text=True, encoding="utf-8").stdout.splitlines()
                if changed != [COMMITTED_BUNDLE.as_posix()]:
                    raise RuntimeError("Stage Desktop checkout changed other tracked Sources")
                studio, studio_build = go_build(
                    ".", "tei-studio-desktop-webview-smoke", checkout / STUDIO_MODULE)
                stage["desktopInputBundleDigest"] = selected
                stage["desktopCheckoutCommit"] = checkout_head
                stage["desktopSourceDifference"] = changed
            finally:
                subprocess.run(["git", "-C", str(repo), "worktree", "remove",
                                "--force", str(checkout)], check=True,
                               capture_output=True, text=True, encoding="utf-8")
        else:
            # Historical mode: both stages embed the committed asset.
            studio, studio_build = go_build(".", "tei-studio", repo / STUDIO_MODULE)
        stage["studioDesktopBuild"] = studio_build
        stage["studioDesktopPath"] = str(studio)
    return stage


def one_contract(path: Path) -> str | None:
    """Select exactly one generated Reader contract, never a substring match."""
    matches = CONTRACT_DIGEST.findall(path.read_bytes()) if path.is_file() else []
    return matches[0].decode() if len(matches) == 1 else None


def embedded_bundle(desktop: Path, bundle: bytes, part: Path) -> dict:
    """Inspect the Desktop bytes and the Reader contract independently.

    The caller supplies either the committed historical Bundle or the new
    Bundle that the TEI-driven Node child produced. The returned observation
    makes no assertion about who selected or accepted that input.
    """
    image = desktop.read_bytes() if desktop.is_file() else b""
    part_ids = CONTRACT_DIGEST.findall(part.read_bytes()) if part.is_file() else []
    bundle_ids = CONTRACT_DIGEST.findall(bundle)
    return {
        "desktopDigest": digest_bytes(image) if image else None,
        "committedBundleDigest": digest_bytes(bundle),
        "embedsCommittedBundle": bool(image) and image.count(bundle) == 1,
        "regeneratedPartContract": part_ids[0].decode() if len(part_ids) == 1 else None,
        "bundleContract": bundle_ids[0].decode() if len(bundle_ids) == 1 else None,
    }


def judge(stage1: dict, stage2: dict, committed_part: str, desktop: dict | None = None) -> dict:
    """Each judgement is separate; agreement of bytes alone is never re-entry."""
    bundle_mode = "bundleGeneration" in stage1 or "bundleGeneration" in stage2
    generated_desktop = (desktop is not None
                         and desktop.get("embedded", {}).get("inputMode") == "stage-generated")
    steps = ("teiBuild", "generatorBuild", "partGeneration") + (
        ("bundleGeneration",) if bundle_mode else ()) + (
        ("studioDesktopBuild",) if desktop is not None else ())

    def clean(stage: dict) -> bool:
        return all(
            stage.get(s, {}).get("invokedExitCode") == 0 and stage.get(s, {}).get("artifactCrossCheckMatched") is True
            and not stage.get(s, {}).get("failureCause")
            and stage.get(s, {}).get("artifactDigest") is not None
            and stage.get(s, {}).get("artifactDigest") == stage.get(s, {}).get("artifactOnDisk")
            for s in steps
        )

    def driven_by(stage: dict) -> bool:
        return all(stage.get(s, {}).get("invokerDigest") == stage["invokerOnDisk"] for s in steps)

    stage1_tei = stage1["teiBuild"].get("artifactDigest")
    judgement = {
        "everyStepCompletedAndCrossChecked": clean(stage1) and clean(stage2),
        "eachStageDrivenByItsOwnTei": driven_by(stage1) and driven_by(stage2),
        "stage1TeiArtifactIsStage2Invoker": stage1_tei is not None and stage2["invokerOnDisk"] == stage1_tei
        and all(stage2.get(s, {}).get("invokerDigest") == stage1_tei for s in steps),
        "stage2PartMadeByStage2RebuiltGenerator": stage2["generatorBuild"].get("artifactDigest") is not None
        and stage2["partGeneration"].get("invokedExecutableDigest") == stage2["generatorBuild"].get("artifactDigest"),
        "stage1PartMadeByStage1BuiltGenerator": stage1["generatorBuild"].get("artifactDigest") is not None
        and stage1["partGeneration"].get("invokedExecutableDigest") == stage1["generatorBuild"].get("artifactDigest"),
        "generatorsEqual": stage1["generatorBuild"].get("artifactDigest") == stage2["generatorBuild"].get("artifactDigest"),
        "teiArtifactsEqual": stage1_tei == stage2["teiBuild"].get("artifactDigest"),
        "partsEqual": stage1["partGeneration"].get("artifactDigest") is not None
        and stage1["partGeneration"].get("artifactDigest") == stage2["partGeneration"].get("artifactDigest"),
    }
    if generated_desktop:
        judgement.update({
            "stage2PartChangedFromCommittedPart": stage2["partGeneration"].get("artifactDigest") is not None
            and stage2["partGeneration"]["artifactDigest"] != committed_part,
            "authoringSourceStableAcrossStages": bool(stage1.get("authoringSourceDigest"))
            and stage1.get("authoringSourceDigest") == stage2.get("authoringSourceDigest")
            and stage1.get("authoringData") == stage2.get("authoringData")
            and stage1.get("authoringSourceDigest") == desktop.get("authoringSourceDigest")
            and desktop.get("authoringSourceDigest") == desktop.get("authoringSourceAfterDigest"),
        })
    else:
        judgement["stage2PartIsTheCommittedPart"] = (
            stage2["partGeneration"].get("artifactDigest") == committed_part)
    if bundle_mode:
        contract1, contract2 = stage1.get("partContract"), stage2.get("partContract")
        builder1 = stage1.get("bundleBuilderSourceDigest")
        judgement.update({
            "bundlesWereMadeByTheRecordedNodeChild": all(
                stage.get("bundleGeneration", {}).get("invokedExecutableDigest")
                == stage.get("bundleNodeDigest")
                and bool(stage.get("bundleNodeDigest"))
                for stage in (stage1, stage2)
            ),
            "bundlesUseSameBuilderSource": bool(builder1)
            and builder1 == stage2.get("bundleBuilderSourceDigest"),
            "eachBundleCarriesItsOwnGeneratedReader": all(
                stage.get("partContract") is not None
                and stage.get("partContract") == stage.get("bundleContract")
                for stage in (stage1, stage2)
            ),
            "bundleContractsEqualAcrossStages": contract1 is not None and contract1 == contract2,
            "bundlesEqualAcrossStages": stage1.get("bundleGeneration", {}).get("artifactDigest") is not None
            and stage1["bundleGeneration"]["artifactDigest"]
            == stage2.get("bundleGeneration", {}).get("artifactDigest"),
        })
    if desktop is not None:
        embedded = desktop.get("embedded", {})
        judgement["studioDesktopsEqual"] = (
            stage1["studioDesktopBuild"].get("artifactDigest") is not None
            and stage1["studioDesktopBuild"].get("artifactDigest")
            == stage2["studioDesktopBuild"].get("artifactDigest"))
        if generated_desktop:
            judgement.update({
                "eachDesktopUsedItsStageGeneratedBundle": all(
                    stage.get("desktopInputBundleDigest") is not None
                    and stage.get("desktopInputBundleDigest")
                    == stage.get("bundleGeneration", {}).get("artifactDigest")
                    and stage.get("desktopSourceDifference") == [COMMITTED_BUNDLE.as_posix()]
                    and stage.get("desktopCheckoutCommit") == embedded.get("sourceHead")
                    for stage in (stage1, stage2)
                ),
                "stage2DesktopEmbedsGeneratedBundle": embedded.get("embedsCommittedBundle") is True
                and embedded.get("desktopDigest") == stage2["studioDesktopBuild"].get("artifactDigest")
                and embedded.get("committedBundleDigest")
                == stage2.get("bundleGeneration", {}).get("artifactDigest"),
                "generatedDesktopExcludesCommittedBundle":
                embedded.get("committedBundleAbsent") is True,
                "generatedDesktopCarriesReaderContract":
                embedded.get("regeneratedPartContract") is not None
                and embedded.get("bundleContract") == embedded.get("regeneratedPartContract")
                and embedded.get("bundleContract") == stage2.get("partContract"),
            })
        else:
            judgement.update({
                "stage2DesktopEmbedsCommittedBundle": embedded.get("embedsCommittedBundle") is True
                and embedded.get("desktopDigest") == stage2["studioDesktopBuild"].get("artifactDigest"),
                "committedBundleCarriesRegeneratedPart": embedded.get("regeneratedPartContract") is not None
                and embedded.get("bundleContract") == embedded.get("regeneratedPartContract"),
            })
        if "smoke" in desktop:
            smoke = desktop["smoke"]
            judgement["stage2DesktopUsedGeneratedReaderInNewProcess"] = (
                smoke.get("exitCode") == 0
                and smoke.get("exeDigest") == stage2["studioDesktopBuild"].get("artifactDigest")
                and smoke.get("observedContract") is not None
                and smoke.get("observedContract") == embedded.get("regeneratedPartContract")
                and smoke.get("initialObservedContract") == embedded.get("regeneratedPartContract")
                and type(smoke.get("readerCalls")) is int
                and smoke["readerCalls"] >= 2
                and type(smoke.get("readerCallsAfterInitialInspect")) is int
                and 1 <= smoke["readerCallsAfterInitialInspect"] < smoke["readerCalls"])
    if desktop is not None and "visibleBeforeSourceDigest" in desktop:
        smoke = desktop.get("smoke", {})
        observed = smoke.get("visibleAuthoring") or {}
        before = observed.get("before") or {}
        after = observed.get("after") or {}

        def visible_fields(item: dict) -> dict | None:
            # Interpret the actual text read from CodeMirror by WebDriver.
            # The Reader's hidden contract marker is not the displayed result.
            try:
                canonical = json.loads(item.get("editorText", ""))
            except (TypeError, ValueError):
                return None
            if not isinstance(canonical, dict) or canonical.get("id") != "studio.inspection_response":
                return None
            attributes = canonical.get("attributes")
            if not isinstance(attributes, list) or len(attributes) != 2:
                return None
            fields = {}
            for attr in attributes:
                if not isinstance(attr, dict) or not isinstance(attr.get("symbol"), str):
                    return None
                if attr["symbol"] in fields:
                    return None
                fields[attr["symbol"]] = attr.get("required", False)
            return fields

        before_fields, after_fields = visible_fields(before), visible_fields(after)
        expected_reader = desktop.get("embedded", {}).get("regeneratedPartContract")
        judgement.update({
            "visibleBeforeSourceUnchanged": bool(desktop.get("visibleBeforeSourceDigest"))
            and desktop.get("visibleBeforeSourceDigest")
            == desktop.get("visibleBeforeSourceAfterDigest")
            and desktop.get("visibleBeforeSourceDigest") != desktop.get("authoringSourceDigest"),
            "sameNativeWebViewDisplaysChangedCanonical": (
                before.get("workspaceId") == "workspace-2"
                and after.get("workspaceId") == "workspace-3"
                and before.get("subjectTitle") == after.get("subjectTitle") == "studio.inspection_response"
                and before.get("editorEditable") == after.get("editorEditable") == "false"
                and before_fields == {"schemaVersion": True, "snapshotFingerprint": False}
                and after_fields == {"schemaVersion": True, "snapshotFingerprint": True}
                and isinstance(before.get("editorText"), str)
                and before.get("editorText") != after.get("editorText")
                and isinstance(expected_reader, str) and expected_reader.startswith("sha256:")
                and before.get("inspectionReader") == after.get("inspectionReader") == expected_reader
                and type(before.get("inspectionReaderCalls")) is int
                and type(after.get("inspectionReaderCalls")) is int
                and before["inspectionReaderCalls"] < after["inspectionReaderCalls"]
            ),
        })
    judgement["reentryObserved"] = all(judgement.values())
    return judgement


def smoke_observations(facts: dict) -> dict:
    """Read what the WebView2 smoke observed, without trusting its verdict.

    The values come from the page itself: the generated reader writes the
    contract digest and a call count on the document element only when it
    accepts a projection. The count after the Graph selection is the second
    Inspect, so it shows the part was used more than once in that process.
    """
    page = facts.get("page_state") or {}
    graph = page.get("graphSelection") or {}
    return {
        "exeDigest": ("sha256:" + facts["exe_sha256"]) if facts.get("exe_sha256") else None,
        "observedContract": graph.get("inspectionReader") or None,
        "initialObservedContract": page.get("inspectionReader") or None,
        "readerCalls": graph.get("inspectionReaderCalls"),
        "readerCallsAfterInitialInspect": page.get("inspectionReaderCalls"),
        "visibleAuthoring": facts.get("visible_authoring"),
    }


def desktop_smoke(repo: Path, desktop: Path, evidence: Path,
                  visible_before: Path | None = None, visible_after: Path | None = None) -> dict:
    """Start Stage2 Desktop through the same real WebView2 smoke."""
    extra = ([str(visible_before), str(visible_after)] if visible_before is not None else [])
    completed = subprocess.run([sys.executable, str(repo / SMOKE), str(desktop), str(evidence), *extra],
                               cwd=repo, capture_output=True, text=True, encoding="utf-8",
                               errors="replace", check=False)
    result_lines = [line for line in completed.stdout.splitlines() if line.startswith("SMOKE_RESULT=")]
    observed: dict = {}
    summary = evidence / "summary.json"
    if summary.is_file():
        observed = smoke_observations(json.loads(summary.read_text(encoding="utf-8-sig")))
    return {
        "exitCode": completed.returncode,
        "result": result_lines[-1] if result_lines else None,
        "evidenceDir": str(evidence),
        **observed,
    }


# The model-derived Reader is exercised by the real Go Generator and esbuild in
# reference/tei-studio-workbench/inspection-reader.model-derived.test.mjs.
# Do not simulate Generator output here: this test checks only Stage2 evidence.
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

    contract = "sha256:" + "c" * 64

    def desktop_stage(invoker: str) -> dict:
        built = stage(invoker)
        built["studioDesktopBuild"] = step(invoker, "studio")
        return built

    def desktop(embeds: bool = True, bundle: str = contract, used: str = contract,
                exe: str = "studio", calls: int = 3, initial_calls: int | None = 1,
                initial_contract: str = contract, exit_code: int = 0) -> dict:
        return {"embedded": {"desktopDigest": "studio", "embedsCommittedBundle": embeds,
                             "regeneratedPartContract": contract, "bundleContract": bundle},
                "smoke": {"exitCode": exit_code, "exeDigest": exe, "observedContract": used,
                          "initialObservedContract": initial_contract, "readerCalls": calls,
                          "readerCallsAfterInitialInspect": initial_calls}}

    assert judge(desktop_stage("tei0"), desktop_stage("tei1"), "part", desktop())["reentryObserved"]
    rejected = {
        "Desktop does not embed the committed bundle": desktop(embeds=False),
        "bundle was built from another contract": desktop(bundle="sha256:" + "d" * 64),
        "WebView ran a reader of another contract": desktop(used="sha256:" + "e" * 64),
        "smoke ran another executable": desktop(exe="ordinary-ci-build"),
        "only the first Inspect went through the reader": desktop(calls=1),
        "smoke failed": desktop(exit_code=1),
        "Graph selection did not increase Reader calls": desktop(calls=2, initial_calls=2),
        "initial Reader observation is absent": desktop(initial_calls=None),
        "initial Reader observation is boolean": desktop(initial_calls=True),
        "final Reader observation is boolean": desktop(calls=True),
        "initial Reader belonged to another contract": desktop(initial_contract="sha256:" + "e" * 64),
    }
    for name, observed in rejected.items():
        assert not judge(desktop_stage("tei0"), desktop_stage("tei1"), "part", observed)["reentryObserved"], name
    # A Stage2 Desktop built by the Stage0 TEI is not re-entry even if it runs.
    assert not judge(desktop_stage("tei0"), desktop_stage("tei0"), "part", desktop())["reentryObserved"]
    def bundle_stage(invoker: str) -> dict:
        built = stage(invoker)
        built.update({
            "bundleGeneration": step(invoker, "bundle", "node-child"),
            "bundleNodeDigest": "node-child",
            "bundleBuilderSourceDigest": "builder-source",
            "partContract": contract,
            "bundleContract": contract,
        })
        return built

    bundled = judge(bundle_stage("tei0"), bundle_stage("tei1"), "part")
    assert bundled["reentryObserved"] and bundled["eachBundleCarriesItsOwnGeneratedReader"], bundled
    bad_stage2 = bundle_stage("tei1")
    bad_stage2["bundleContract"] = "sha256:" + "d" * 64
    assert not judge(bundle_stage("tei0"), bad_stage2, "part")["reentryObserved"]
    bad_stage2 = bundle_stage("tei1")
    bad_stage2["bundleGeneration"]["invokedExecutableDigest"] = "other-node"
    assert not judge(bundle_stage("tei0"), bad_stage2, "part")["reentryObserved"]
    bad_stage2 = bundle_stage("tei1")
    bad_stage2["bundleGeneration"]["artifactOnDisk"] = "stale-bundle"
    assert not judge(bundle_stage("tei0"), bad_stage2, "part")["reentryObserved"]
    bad_stage2 = bundle_stage("tei1")
    bad_stage2["bundleBuilderSourceDigest"] = "other-builder"
    assert not judge(bundle_stage("tei0"), bad_stage2, "part")["reentryObserved"]
    bad_stage2 = bundle_stage("tei1")
    bad_stage2["bundleGeneration"]["artifactDigest"] = "old-bundle"
    bad_stage2["bundleGeneration"]["artifactOnDisk"] = "old-bundle"
    assert not judge(bundle_stage("tei0"), bad_stage2, "part")["reentryObserved"]
    # One side missing the bundle step must fail closed instead of KeyError.
    assert not judge(bundle_stage("tei0"), stage("tei1"), "part")["reentryObserved"]
    def changed_desktop_stage(invoker: str) -> dict:
        built = bundle_stage(invoker)
        built["partGeneration"] = step(invoker, "changed-part", "gen")
        built["studioDesktopBuild"] = step(invoker, "changed-studio")
        built["desktopInputBundleDigest"] = "bundle"
        built["desktopSourceDifference"] = [COMMITTED_BUNDLE.as_posix()]
        built["desktopCheckoutCommit"] = "pinned-head"
        built["authoringSourceDigest"] = "authoring-source"
        built["authoringData"] = "studio.inspection_response"
        return built

    def changed_desktop() -> dict:
        actual = desktop(exe="changed-studio")
        actual["embedded"].update({
            "inputMode": "stage-generated",
            "desktopDigest": "changed-studio",
            "committedBundleDigest": "bundle",
            "embedsCommittedBundle": True,
            "committedBundleAbsent": True,
            "sourceHead": "pinned-head",
        })
        actual["authoringSourceDigest"] = "authoring-source"
        actual["authoringSourceAfterDigest"] = "authoring-source"
        return actual

    changed = judge(changed_desktop_stage("tei0"), changed_desktop_stage("tei1"),
                    "part", changed_desktop())
    assert changed["reentryObserved"] and changed["stage2DesktopEmbedsGeneratedBundle"], changed

    invalid = changed_desktop()
    invalid["embedded"]["committedBundleAbsent"] = False
    assert not judge(changed_desktop_stage("tei0"), changed_desktop_stage("tei1"),
                     "part", invalid)["reentryObserved"]
    invalid = changed_desktop()
    invalid["embedded"]["embedsCommittedBundle"] = False
    assert not judge(changed_desktop_stage("tei0"), changed_desktop_stage("tei1"),
                     "part", invalid)["reentryObserved"]
    invalid = changed_desktop()
    invalid["authoringSourceAfterDigest"] = "source-changed"
    assert not judge(changed_desktop_stage("tei0"), changed_desktop_stage("tei1"),
                     "part", invalid)["reentryObserved"]
    invalid = changed_desktop()
    invalid["embedded"]["sourceHead"] = "other-head"
    assert not judge(changed_desktop_stage("tei0"), changed_desktop_stage("tei1"),
                     "part", invalid)["reentryObserved"]
    invalid = changed_desktop()
    invalid["embedded"]["committedBundleDigest"] = "old-bundle"
    assert not judge(changed_desktop_stage("tei0"), changed_desktop_stage("tei1"),
                     "part", invalid)["reentryObserved"]
    bad_stage = changed_desktop_stage("tei1")
    bad_stage["desktopInputBundleDigest"] = "another-bundle"
    assert not judge(changed_desktop_stage("tei0"), bad_stage,
                     "part", changed_desktop())["reentryObserved"]
    bad_stage = changed_desktop_stage("tei1")
    bad_stage["desktopSourceDifference"] = [COMMITTED_BUNDLE.as_posix(), "go.mod"]
    assert not judge(changed_desktop_stage("tei0"), bad_stage,
                     "part", changed_desktop())["reentryObserved"]
    bad_stage = changed_desktop_stage("tei1")
    bad_stage["authoringSourceDigest"] = "other-authoring"
    assert not judge(changed_desktop_stage("tei0"), bad_stage,
                     "part", changed_desktop())["reentryObserved"]
    bad_stage = changed_desktop_stage("tei1")
    bad_stage["partGeneration"] = step("tei1", "part", "gen")
    assert not judge(changed_desktop_stage("tei0"), bad_stage,
                     "part", changed_desktop())["reentryObserved"]
    def visible_desktop() -> dict:
        actual = changed_desktop()
        actual["visibleBeforeSourceDigest"] = "old-data-source"
        actual["visibleBeforeSourceAfterDigest"] = "old-data-source"
        actual["smoke"]["visibleAuthoring"] = {
            "before": {
                "workspaceId": "workspace-2", "subjectTitle": "studio.inspection_response",
                "editorEditable": "false",
                "editorText": json.dumps({"id": "studio.inspection_response", "attributes": [
                    {"symbol": "schemaVersion", "required": True},
                    {"symbol": "snapshotFingerprint", "required": False}]}),
                "inspectionReader": contract, "inspectionReaderCalls": 4,
            },
            "after": {
                "workspaceId": "workspace-3", "subjectTitle": "studio.inspection_response",
                "editorEditable": "false",
                "editorText": json.dumps({"id": "studio.inspection_response", "attributes": [
                    {"symbol": "schemaVersion", "required": True},
                    {"symbol": "snapshotFingerprint", "required": True}]}),
                "inspectionReader": contract, "inspectionReaderCalls": 5,
            },
        }
        return actual

    valid = judge(changed_desktop_stage("tei0"), changed_desktop_stage("tei1"),
                  "part", visible_desktop())
    assert valid["sameNativeWebViewDisplaysChangedCanonical"] and valid["reentryObserved"], valid

    for name in ("same-text", "wrong-reader", "same-source", "source-mutated-after-smoke",
                 "before-source-mutated-after-smoke", "stale-selection",
                 "editable-editor", "not-a-real-call", "wrong-required"):
        invalid = visible_desktop()
        before, after = invalid["smoke"]["visibleAuthoring"]["before"], invalid["smoke"]["visibleAuthoring"]["after"]
        if name == "same-text":
            after["editorText"] = before["editorText"]
        elif name == "wrong-reader":
            after["inspectionReader"] = "sha256:" + "0" * 64
        elif name == "same-source":
            invalid["visibleBeforeSourceDigest"] = "authoring-source"
        elif name == "source-mutated-after-smoke":
            invalid["authoringSourceAfterDigest"] = "unexpected-after-smoke"
        elif name == "before-source-mutated-after-smoke":
            invalid["visibleBeforeSourceAfterDigest"] = "unexpected-after-smoke"
        elif name == "stale-selection":
            after["workspaceId"] = "workspace-2"
        elif name == "editable-editor":
            after["editorEditable"] = "true"
        elif name == "not-a-real-call":
            after["inspectionReaderCalls"] = 4
        elif name == "wrong-required":
            after["editorText"] = json.dumps({"id": "studio.inspection_response", "attributes": [
                {"symbol": "schemaVersion", "required": True},
                {"symbol": "snapshotFingerprint", "required": False}]})
        assert not judge(changed_desktop_stage("tei0"), changed_desktop_stage("tei1"),
                         "part", invalid)["reentryObserved"], name
    print("self-test: 43 judgements behaved as specified")


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument("--self-test", action="store_true", help="check the judgement without building anything")
    parser.add_argument("--repo", type=Path, help="clean checkout to build from")
    parser.add_argument("--supply", type=Path, help="Go module supply (a GOPROXY file root, e.g. $GOMODCACHE/cache/download)")
    parser.add_argument("--work", type=Path, help="new directory for every stage's outputs; must not exist")
    parser.add_argument("--go", default="go", help="Stage0 seed Go executable")
    parser.add_argument("--studio-desktop", action="store_true",
                        help="also drive a Go build of the Studio Wails Desktop in each stage (Windows)")
    parser.add_argument("--desktop-smoke", action="store_true",
                        help="start the Stage2 Desktop as a new process through the WebView2 smoke "
                             "(Windows; implies --studio-desktop)")
    parser.add_argument("--bundle-reader", action="store_true",
                        help="have both TEI invokers drive the existing Node/esbuild child "
                             "using their generated Reader (does NOT embed it in a Desktop)")
    parser.add_argument("--node", default="node", help="Node executable for the existing esbuild child")
    parser.add_argument("--authoring-root", type=Path,
                        help="separate tei.authoring/v1 Workspace containing the changed Data Source")
    parser.add_argument("--authoring-data", help="canonical Data ID bound to Inspector Reader")
    parser.add_argument("--stage-desktop-bundle", action="store_true",
                        help="build staged Windows Desktop with the newly generated Bundle "
                             "and verify it in a fresh WebView2 process (requires authoring Data)")
    parser.add_argument("--visible-before-root", type=Path,
                        help="read before/after Data in one native WebView2; requires "
                             "--stage-desktop-bundle and uses --authoring-root as after")
    args = parser.parse_args()
    if args.desktop_smoke:
        args.studio_desktop = True
    if args.self_test:
        self_test()
        return 0
    if bool(args.authoring_root) != bool(args.authoring_data):
        parser.error("--authoring-root and --authoring-data must be supplied together")
    if args.stage_desktop_bundle:
        if os.name != "nt":
            parser.error("--stage-desktop-bundle requires a Windows Wails host")
        if not args.authoring_root or not args.desktop_smoke:
            parser.error("--stage-desktop-bundle requires changed --authoring-root, "
                         "--authoring-data, and --desktop-smoke")
        args.bundle_reader = True
        args.studio_desktop = True
    elif args.bundle_reader and args.studio_desktop:
        parser.error("separate Bundle and Desktop modes cannot be combined without "
                     "--stage-desktop-bundle")
    if args.authoring_root and not args.bundle_reader:
        parser.error("changed authoring Data requires a TEI-driven Reader Bundle")
    if args.visible_before_root and not args.stage_desktop_bundle:
        parser.error("--visible-before-root requires --stage-desktop-bundle")
    if not (args.repo and args.supply and args.work):
        parser.error("--repo, --supply and --work are required")
    repo, supply, work = args.repo.resolve(), args.supply.resolve(), args.work.resolve()
    expected_head = git_head(repo)
    authoring_root = args.authoring_root.resolve() if args.authoring_root else None
    authoring_before = authoring_source_digest(authoring_root) if authoring_root else None
    visible_before_root = args.visible_before_root.resolve() if args.visible_before_root else None
    visible_before_digest = authoring_source_digest(visible_before_root) if visible_before_root else None
    if visible_before_root is not None and (
            visible_before_root == authoring_root or visible_before_digest == authoring_before
            or args.authoring_data != "studio.inspection_response"):
        parser.error("visible before must be a different Source of studio.inspection_response")
    if work.exists():
        parser.error(f"--work already exists: {work}")
    go = shutil.which(args.go)
    if go is None:
        parser.error(f"Stage0 seed Go executable not found: {args.go}")
    node_child = None
    if args.bundle_reader:
        node_exe = shutil.which(args.node)
        if node_exe is None:
            parser.error(f"Node executable not found: {args.node}")
        node_child = Path(node_exe).resolve()
        if not (repo / WORKBENCH / "node_modules" / "esbuild").is_dir():
            parser.error("Workbench esbuild is not installed; run npm ci in reference/tei-studio-workbench")
    stage0 = work / "stage0"
    stage0.mkdir(parents=True)

    def seed_build(package: str, name: str) -> Path:
        out = stage0 / (name + EXE)
        subprocess.run([go, "build", "-trimpath", "-o", str(out), package], cwd=repo, check=True,
                       env={**os.environ, "CGO_ENABLED": "0", "GOFLAGS": "", "GOTOOLCHAIN": "local"})
        return out

    tei0 = seed_build("./cmd/tei", "tei")
    go_child = seed_build("./tools/reference-go-build-with-supply", "go-build-with-supply")

    stage1 = run_stage(
        tei0, go_child, repo, supply, work / "stage1", args.studio_desktop,
        node_child, authoring_root, args.authoring_data, args.stage_desktop_bundle,
        expected_head)
    tei1 = Path(stage1["teiArtifactPath"])
    if not tei1.is_file():
        raise RuntimeError(f"stage1 did not produce a TEI binary: {stage1['teiBuild'].get('failureCause')}")
    # The Stage2 driver is the Stage1 artifact itself, at the path Stage1 wrote it.
    stage2 = run_stage(
        tei1, go_child, repo, supply, work / "stage2", args.studio_desktop,
        node_child, authoring_root, args.authoring_data, args.stage_desktop_bundle,
        expected_head)
    authoring_after = authoring_source_digest(authoring_root) if authoring_root else None
    visible_before_after = authoring_source_digest(visible_before_root) if visible_before_root else None
    if visible_before_digest != visible_before_after:
        raise RuntimeError("visible before Source changed during the Stage1 -> Stage2 Attempt")
    if authoring_before != authoring_after:
        raise RuntimeError("authoring Source changed during the Stage1 -> Stage2 Attempt")
    if git_head(repo) != expected_head:
        raise RuntimeError("Git HEAD changed during the Stage1 -> Stage2 Attempt")
    desktop = None
    if args.studio_desktop:
        studio2 = Path(stage2["studioDesktopPath"])
        if args.stage_desktop_bundle:
            current_part = one_contract(Path(stage2["partPath"]))
            original_part = one_contract(repo / COMMITTED_PART)
            if current_part is None or original_part is None or current_part == original_part:
                raise RuntimeError("staged Desktop must have a different generated Reader contract")
            selected_bundle = Path(stage2["bundlePath"]).read_bytes()
            embedded = embedded_bundle(studio2, selected_bundle, Path(stage2["partPath"]))
            embedded.update({
                "inputMode": "stage-generated",
                "sourceHead": expected_head,
                "committedBundleAbsent": (repo / COMMITTED_BUNDLE).read_bytes()
                    not in (studio2.read_bytes() if studio2.is_file() else b""),
            })
            desktop = {"embedded": embedded, "authoringSourceDigest": authoring_before,
                       "authoringSourceAfterDigest": authoring_after}
        else:
            desktop = {"embedded": embedded_bundle(
                studio2, (repo / COMMITTED_BUNDLE).read_bytes(),
                Path(stage2["partPath"]))}
        if args.desktop_smoke and studio2.is_file():
            if visible_before_root is not None:
                desktop["visibleBeforeSourceDigest"] = visible_before_digest
                desktop["visibleBeforeSourceAfterDigest"] = visible_before_after
            desktop["smoke"] = desktop_smoke(
                repo, studio2, work / "stage2-desktop-smoke",
                visible_before_root, authoring_root if visible_before_root is not None else None)

    # Both model Sources were checked after Stage2 generation. Their
    # identities must also survive the *consumer* observation: a checksum
    # taken before WebView2 runs does not prove what it later read.
    if args.stage_desktop_bundle:
        final_authoring_digest = authoring_source_digest(authoring_root)
        if final_authoring_digest != authoring_before:
            raise RuntimeError("authoring Source changed before Stage2 WebView2 observation finished")
        authoring_after = final_authoring_digest
        if visible_before_root is not None:
            final_visible_digest = authoring_source_digest(visible_before_root)
            if final_visible_digest != visible_before_digest:
                raise RuntimeError("visible before Source changed before Stage2 WebView2 observation finished")
            visible_before_after = final_visible_digest
        if git_head(repo) != expected_head:
            raise RuntimeError("Git HEAD changed before Stage2 WebView2 observation finished")
        if desktop is not None:
            desktop["authoringSourceAfterDigest"] = authoring_after
            if visible_before_root is not None:
                desktop["visibleBeforeSourceAfterDigest"] = visible_before_after

    committed = digest_bytes((repo / COMMITTED_PART).read_bytes().replace(b"\r\n", b"\n"))
    result = {
        "schemaVersion": SCHEMA,
        "sources": {
            "repoHead": expected_head,
            "generatorSourceBytes": digest_file(repo / TRANSFORM),
            "contractSourceBytes": digest_file(repo / CONTRACT),
            "committedPart": committed,
            **({"authoringInputDigest": authoring_before,
                 "authoringInputDigestAfter": authoring_after,
                 "authoringData": args.authoring_data}
               if authoring_root is not None else {}),
            **({"visibleBeforeSourceDigest": visible_before_digest,
                 "visibleBeforeSourceAfterDigest": visible_before_after}
               if visible_before_root is not None else {}),
            **({"bundleBuilderSourceBytes": digest_file(repo / BUNDLE_CHILD)}
               if node_child is not None else {}),
        },
        "stage0": {
            "nature": "external Go seed; not a driven process",
            "goExecutable": go,
            "goExecutableDigest": digest_file(Path(go)),
            "goVersion": subprocess.run([go, "version"], capture_output=True, text=True, encoding="utf-8").stdout.strip(),
            "teiDigest": digest_file(tei0),
            "goBuildChildDigest": digest_file(go_child),
            **({"nodeExecutableDigest": digest_file(node_child)} if node_child is not None else {}),
        },
        "stage1": stage1,
        "stage2": stage2,
        "stage2Desktop": desktop,
        "judgement": judge(stage1, stage2, committed, desktop),
        "claimBoundary": "one host, one toolchain, one supply; the Go build child is the Stage0 seed's in both stages; "
                         + ("each TEI stage drove Node/esbuild and built a Wails Desktop from its generated Bundle in an isolated same-HEAD worktree; Studio Editor writing, Owner acceptance and formal Build Intent are NOT claimed; "
                            if args.stage_desktop_bundle else
                            "Node/esbuild is driven by each TEI invoker with its generated Reader, but this new bundle is NOT embedded in a Stage2 Desktop; "
                            if node_child is not None else
                            "the Node/esbuild bundle is the committed one, not rebuilt by TEI; ")
                         + "no Seed Trust, acceptance or Runtime Authority is claimed",
    }
    json.dump(result, sys.stdout, indent=2)
    print()
    return 0 if result["judgement"]["reentryObserved"] else 1


if __name__ == "__main__":
    sys.exit(main())
