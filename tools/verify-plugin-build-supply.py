#!/usr/bin/env python3
"""Run the plugin-build-supply tests (sample 7, stages 1-4) against this
checkout and write a receipt.

Run it from a clean, committed checkout (for example a shallow clone of the
working branch). Stage 3 runs the existing self-build runner, which refuses a
working tree with changes or untracked files, and the tests treat that refusal
as part of the result; a dirty tree is therefore refused here as well.

Every test of the package must PASS and none may SKIP. No module or Go
toolchain is downloaded: GOPROXY is off and the exact Go executable comes from
the local GOROOT. Nothing here is evidence of a production builder, of
OS-level isolation of the builds, of a signed or trusted builder, or of a
remote artifact store.
"""
import argparse
import json
import os
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile
import time

REPO = Path(__file__).resolve().parents[1]
PACKAGE = "./tools/reference-plugin-build-supply/"
EXPECTED_TESTS = (
    "TestResolveBuilderStopsOnAmbiguityWhateverTheOrder",
    "TestLockIsStableAndRevocationSeparatesReproductionFromUse",
    "TestAttemptHistoryRefusesFailuresAndDoubleCounting",
    "TestStoreRefusesCorruptionWithoutRepair",
    "TestVerifyThroughNeedsEveryStageAndOneIdentity",
    "TestThroughPluginBuildSupply",
    "TestPJOB01ConcurrentAttemptsKeepDistinctOccurrences",
    "TestPJOB01FailuresAndMismatchesDoNotYieldReproducibility",
    "TestPGOV01ResolutionStopsAndRevocationRefusesUse",
    "TestClosureThatCannotBeFixedStops",
    "TestThroughDetectsSwappedIdentityAndMissingStage",
)


def run(cmd, **kw):
    return subprocess.run(cmd, check=True, capture_output=True, text=True, **kw).stdout.strip()


def git_status():
    try:
        head = run(["git", "rev-parse", "HEAD"], cwd=REPO)
        status = run(["git", "status", "--porcelain=v1", "--untracked-files=all"], cwd=REPO)
    except (OSError, subprocess.CalledProcessError) as exc:
        raise SystemExit(f"git state unavailable: {exc}")
    return head, status == ""


def exact_go(candidate):
    # GOPROXY=off: a toolchain switch may use a local toolchain but never
    # downloads one.
    env = dict(os.environ, GOPROXY="off")
    root = run([candidate, "env", "GOROOT"], env=env)
    go = Path(root) / "bin" / "go"
    if not go.is_file():
        raise SystemExit(f"no Go executable under {root}")
    return str(go.resolve()), root


def summarize(events):
    results = {}
    for ev in events:
        test = ev.get("Test")
        if test and "/" not in test and ev.get("Action") in ("pass", "fail", "skip"):
            results[test] = ev["Action"]
    return results


def main():
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("--out", help="receipt directory (default: new temp dir outside the repository)")
    ap.add_argument("--go", default=shutil.which("go") or os.path.expanduser("~/sdk/go/bin/go"))
    args = ap.parse_args()

    head, clean = git_status()
    if not clean:
        raise SystemExit("working tree has changes or untracked files; run from a clean clone of the commit")
    out = Path(args.out) if args.out else Path(tempfile.mkdtemp(prefix="tei-pbs-receipt-"))
    out = out.resolve()
    if out == REPO or REPO in out.parents:
        raise SystemExit("output directory must be outside the repository")
    out.mkdir(parents=True, exist_ok=True)
    (out / "observed.jsonl").write_text("")

    go, goroot = exact_go(args.go)
    receipt = {
        "schema": "tei.reference.plugin-build-supply-receipt/v0",
        "repositoryRevision": head,
        "workingTreeClean": clean,
        "steps": [1, 2, 3, 4],
        "evidenceNature": "real-go-builds-and-the-existing-self-build-runner-on-one-machine-synthetic-provider-catalog-and-trust-store",
        "notEvidenceFor": [
            "a production builder resolver, provider conformance evidence or a signed trust root",
            "OS-level isolation of the builds (Go state is isolated by environment and directories only)",
            "a host-supplied build attempt type (the sample's host issues occurrence IDs itself)",
            "a remote or shared artifact store",
            "the identity of the engine image actually linked into the host",
            "build plans of the package, native and transform builders derived from stage 1 (the runner still assembles them)",
        ],
        "goExecutable": go,
        "goRoot": goroot,
        "goVersion": run([go, "version"], env=dict(os.environ, GOTOOLCHAIN="local")),
        "goRootFromModuleCacheToolchain": "/pkg/mod/golang.org/toolchain@" in goroot,
        "cpuCount": os.cpu_count(),
    }
    env = dict(os.environ)
    env.update({
        "TEI_PBS_REPO": str(REPO),
        "TEI_PBS_GO": go,
        "TEI_PBS_OUT_DIR": str(out),
        "GOPROXY": "off",
        "GOFLAGS": "-mod=readonly",
        "GOTOOLCHAIN": "local",
        "GOWORK": "off",
    })
    cmd = [go, "test", "-json", "-count=1", "-p", "1", "-timeout", "60m", PACKAGE]
    receipt["command"] = " ".join(cmd[1:])
    started = time.monotonic()
    proc = subprocess.run(cmd, cwd=REPO, env=env, capture_output=True, text=True)
    receipt["testSeconds"] = round(time.monotonic() - started, 1)
    (out / "go-test.jsonl").write_text(proc.stdout)
    events = [json.loads(line) for line in proc.stdout.splitlines() if line.startswith("{")]
    results = summarize(events)
    counts = {a: sum(1 for v in results.values() if v == a) for a in ("pass", "fail", "skip")}
    receipt["tests"] = results
    receipt["counts"] = counts
    missing = [t for t in EXPECTED_TESTS if results.get(t) != "pass"]
    receipt["observed"] = [json.loads(l) for l in (out / "observed.jsonl").read_text().splitlines() if l]
    chain = out / "chain.json"
    if chain.exists():
        doc = json.loads(chain.read_text())
        records = doc.get("chain", {}).get("records", [])
        receipt["chain"] = {"records": len(records), "stages": [r["stage"] for r in records],
                            "head": records[-1]["digest"] if records else None, "through": doc.get("through")}
    _, clean_after = git_status()
    receipt["workingTreeCleanAfter"] = clean_after
    receipt["passed"] = (proc.returncode == 0 and counts["fail"] == 0 and counts["skip"] == 0 and not missing
                         and chain.exists() and bool(receipt.get("chain", {}).get("through", {}).get("complete"))
                         and clean_after)
    if missing:
        receipt["missingOrNotPassed"] = missing
    if proc.stderr.strip():
        receipt["stderrTail"] = proc.stderr.strip()[-2000:]
    (out / "receipt.json").write_text(json.dumps(receipt, indent=2, ensure_ascii=False) + "\n")
    print(json.dumps({k: receipt.get(k) for k in ("repositoryRevision", "workingTreeClean", "goVersion", "counts", "passed",
                                                   "testSeconds", "workingTreeCleanAfter")}, ensure_ascii=False))
    print(f"receipt: {out / 'receipt.json'}")
    return 0 if receipt.get("passed") else 1


if __name__ == "__main__":
    sys.exit(main())
