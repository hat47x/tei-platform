#!/usr/bin/env python3
"""Run the external-adoption-authority chain (sample 9, stages 1-4) against
this checkout and write a receipt.

Run it from a clean, committed checkout (for example a shallow clone of the
working branch). The tests build the two existing asset CLIs
(tools/reference-sui-adoption and tools/reference-semantic-extension-catalog)
unchanged with the exact Go executable of the local GOROOT, copy the fixed
inputs into a temporary directory and never write to the repository.

Every test of the package must PASS and none may SKIP: a skipped through test
is not evidence. The chain record (chain-record.json) must also say passed.
No module or Go toolchain is downloaded (GOPROXY=off) and nothing is fetched
from the SUI repository: the external side is a copy of the pinned table.
Trust grants, signer keys, trust points and the acceptance Host are synthetic
and in the test process; time is logical. The receipt says so; it is not
evidence of conformance to the current SUI application, of a real publisher
or trust store, or of product acceptance.
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
PACKAGE = "./tools/reference-external-adoption-authority/"
EXPECTED_TESTS = (
    "TestPinnedTableHasOneRevisionAndFourBlobs",
    "TestCheckExternalPinStopsOnAnyBlobButNotOnRevision",
    "TestVerifyThroughNeedsEveryStageOneIdentityAndLosslessFalse",
    "TestSignerLookupJudgesOnlyThePointItIsAskedAbout",
    "TestThroughExternalAdoptionAuthority",
)


def run(cmd, **kw):
    return subprocess.run(cmd, check=True, capture_output=True, text=True, **kw).stdout.strip()


def git_state():
    try:
        head = run(["git", "rev-parse", "HEAD"], cwd=REPO)
        status = run(["git", "status", "--porcelain=v1", "--untracked-files=all"], cwd=REPO)
    except (OSError, subprocess.CalledProcessError) as exc:
        return None, False, str(exc)
    return head, status == "", ""


def exact_go(candidate):
    # GOPROXY=off: a toolchain switch may use a local toolchain but never
    # downloads one.
    env = dict(os.environ, GOPROXY="off")
    root = run([candidate, "env", "GOROOT"], env=env, cwd=REPO)
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


def chain_summary(record):
    chain = record.get("chain", {})
    return {
        "passed": record.get("passed"),
        "through": record.get("through"),
        "stages": [{"stage": r["stage"], "name": r["name"], "digest": r["digest"]} for r in chain.get("records", [])],
        "carried": (chain.get("records") or [{}])[-1].get("carried"),
        "faultInjections": {f["id"]: f["passed"] for f in record.get("faultInjections", [])},
        "completionConditions": {c["id"]: c["passed"] for c in record.get("completionConditions", [])},
        "trustPoints": [{"seq": p["seq"], "label": p["label"], "now": p["now"]}
                        for p in record.get("trustPoints", {}).get("points", [])],
        "platformGaps": len(record.get("platformGaps") or []),
        "limits": record.get("limits"),
    }


def main():
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("--out", help="receipt directory (default: new temp dir outside the repository)")
    ap.add_argument("--go", default=shutil.which("go") or os.path.expanduser("~/sdk/go/bin/go"))
    ap.add_argument("--allow-dirty", action="store_true", help="record a dirty or unknown tree instead of refusing")
    args = ap.parse_args()

    head, clean, git_error = git_state()
    if not clean and not args.allow_dirty:
        raise SystemExit("working tree has changes or untracked files (or git is unavailable); "
                         "run from a clean clone of the commit or pass --allow-dirty")
    out = Path(args.out) if args.out else Path(tempfile.mkdtemp(prefix="tei-eaa-receipt-"))
    out = out.resolve()
    if out == REPO or REPO in out.parents:
        raise SystemExit("output directory must be outside the repository")
    out.mkdir(parents=True, exist_ok=True)
    record_path = out / "chain-record.json"
    if record_path.exists():
        record_path.unlink()

    go, goroot = exact_go(args.go)
    receipt = {
        "schema": "tei.reference.external-adoption-authority-receipt/v0",
        "repositoryRevision": head,
        "workingTreeClean": clean,
        "steps": [1, 2, 3, 4],
        "evidenceNature": "existing-asset-clis-and-libraries-unchanged-on-copied-fixed-inputs-"
                          "simulated-external-side-synthetic-trust-signer-keys-and-host-logical-time",
        "notEvidenceFor": [
            "conformance to the current SUI application (only the pinned blobs, compared with a fixed copy)",
            "a real publisher, signer or Host trust store (grants, keys and trust points are synthetic)",
            "product acceptance (synthetic Host; the existing acceptance check covers path imports only)",
            "revocation propagation across servers or processes",
            "a production distribution, signature or BuildOutputKind format",
            "a lossless mapping (parent presence stays unresolved, lossless=false)",
        ],
        "goExecutable": go,
        "goRoot": goroot,
        "goVersion": run([go, "version"], env=dict(os.environ, GOTOOLCHAIN="local")),
        "goRootFromModuleCacheToolchain": "/pkg/mod/golang.org/toolchain@" in goroot,
    }
    if git_error:
        receipt["gitError"] = git_error
    env = dict(os.environ)
    env.update({
        "TEI_EAA_REPO": str(REPO),
        "TEI_EAA_GO": go,
        "TEI_EAA_OUT_DIR": str(out),
        "GOPROXY": "off",
        "GOFLAGS": "-mod=readonly",
        "GOTOOLCHAIN": "local",
        "GOWORK": "off",
    })
    cmd = [go, "test", "-json", "-count=1", "-timeout", "20m", PACKAGE]
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
    record = json.loads(record_path.read_text()) if record_path.exists() else {}
    receipt["chain"] = chain_summary(record)
    _, clean_after, _ = git_state()
    receipt["workingTreeCleanAfter"] = clean_after
    receipt["passed"] = (proc.returncode == 0 and counts["fail"] == 0 and counts["skip"] == 0
                         and not missing and record.get("passed") is True
                         and (clean_after or args.allow_dirty))
    if missing:
        receipt["missingOrNotPassed"] = missing
    if proc.stderr.strip():
        receipt["stderrTail"] = proc.stderr.strip()[-2000:]
    (out / "receipt.json").write_text(json.dumps(receipt, indent=2, ensure_ascii=False) + "\n")
    print(json.dumps({k: receipt.get(k) for k in ("repositoryRevision", "workingTreeClean", "goVersion", "counts",
                                                   "passed", "testSeconds", "workingTreeCleanAfter")},
                     ensure_ascii=False))
    print(f"receipt: {out / 'receipt.json'}")
    return 0 if receipt.get("passed") else 1


if __name__ == "__main__":
    sys.exit(main())
