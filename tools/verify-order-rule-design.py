#!/usr/bin/env python3
"""Run the order-rule-design chain (sample 2, stages 1-6) against this
checkout and write a receipt.

Run it from a clean, committed checkout (for example a shallow clone of the
working branch): stage 4 calls tools/reference-tool-neutral-development/run.sh
unchanged, and that script refuses a checkout with changes. The tests build
the existing asset commands unchanged with the exact Go executable of the
local GOROOT, copy the shared inputs into a temporary directory and never
write to the repository, to reference/workspace or to .tei/decisions.

Every test of the package must PASS and none may SKIP: a skipped through test
is not evidence. The chain record (chain-record.json) must also say passed.
Node.js and OpenSSH (ssh-keygen with verify-time) are required; Docker is not
used. No module or Go toolchain is downloaded (GOPROXY=off). Time is logical.

The sample submits the candidate, its provenance and the signed decision to
the shared acceptance for preview and review only. It never accepts. The
receipt says so; it is not evidence of an acceptance, of a real signer
authority, or of conformance to a real legacy design document.

--repeat N runs the whole chain N times. Some existing tools are not
byte-deterministic, so the runs are compared by their judgments and
diagnostics (stableSummaryDigest), not by raw bytes.
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
PACKAGE = "./tools/reference-order-rule-design/"
EXPECTED_TESTS = (
    "TestThroughCompletesOnlyWhenEveryIdentityIsOneValue",
    "TestMatrixRefusesNoFindingOnlyAndMissingScope",
    "TestImpactRangeRefusesOverclaims",
    "TestSourceVariantsGiveTheThreeStates",
    "TestThroughOrderRuleDesign",
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
    last = (chain.get("records") or [{}])[-1].get("carried", {})
    return {
        "passed": record.get("passed"),
        "through": record.get("through"),
        "stages": [{"stage": r["stage"], "name": r["name"], "digest": r["digest"]} for r in chain.get("records", [])],
        "provenanceState": last.get("provenanceState"),
        "unresolved": last.get("unresolved"),
        "faultInjections": {f["id"]: f["passed"] for f in record.get("faultInjections", [])},
        "completionConditions": {c["id"]: c["passed"] for c in record.get("completionConditions", [])},
        "notVerified": record.get("notVerified"),
        "platformGaps": len(record.get("platformGaps") or []),
        "stableSummaryDigest": record.get("stableSummaryDigest"),
        "limits": record.get("limits"),
    }


def one_run(go, goroot, out, args):
    out.mkdir(parents=True, exist_ok=True)
    record_path = out / "chain-record.json"
    if record_path.exists():
        record_path.unlink()
    head, clean, git_error = git_state()
    ssh = subprocess.run(["ssh", "-V"], capture_output=True, text=True)
    receipt = {
        "schema": "tei.reference.order-rule-design-receipt/v0",
        "repositoryRevision": head,
        "workingTreeClean": clean,
        "steps": [1, 2, 3, 4, 5, 6],
        "evidenceNature": "existing-asset-commands-and-libraries-unchanged-on-copied-shared-inputs-"
                          "synthetic-legacy-material-synthetic-derivations-and-in-flight-cases-"
                          "test-signing-key-logical-time-submission-to-shared-acceptance-preview-only",
        "notEvidenceFor": [
            "an acceptance (the shared acceptance is called for preview and review only; order-workflow is not on main)",
            "a real legacy design document (the material is a synthetic minimal XML and the contract is the synthetic example)",
            "decider authority (the signing key and allowed signer are test values)",
            "generated execution targets or a runtime that returns running cases (both are declared inputs)",
            "a qualification receipt (qualify.sh needs the full repository health check and is not called)",
        ],
        "goExecutable": go,
        "goRoot": goroot,
        "goVersion": run([go, "version"], env=dict(os.environ, GOTOOLCHAIN="local")),
        "nodeVersion": run(["node", "--version"]),
        "sshVersion": (ssh.stderr or ssh.stdout).strip(),
    }
    if git_error:
        receipt["gitError"] = git_error
    env = dict(os.environ)
    env.update({
        "TEI_ORD_REPO": str(REPO),
        "TEI_ORD_GO": go,
        "TEI_ORD_OUT_DIR": str(out),
        "GOPROXY": "off",
        "GOFLAGS": "-mod=readonly",
        "GOTOOLCHAIN": "local",
        "GOWORK": "off",
    })
    cmd = [go, "test", "-json", "-count=1", "-timeout", "25m", PACKAGE]
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
    return receipt


def main():
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("--out", help="receipt directory (default: new temp dir outside the repository)")
    ap.add_argument("--go", default=shutil.which("go") or os.path.expanduser("~/sdk/go/bin/go"))
    ap.add_argument("--allow-dirty", action="store_true", help="record a dirty or unknown tree instead of refusing")
    ap.add_argument("--repeat", type=int, default=1, help="run the chain this many times and compare the judgments")
    args = ap.parse_args()

    _, clean, _ = git_state()
    if not clean and not args.allow_dirty:
        raise SystemExit("working tree has changes or untracked files (or git is unavailable); "
                         "run from a clean clone of the commit or pass --allow-dirty")
    for tool in ("node", "ssh-keygen", "git", "bash"):
        if shutil.which(tool) is None:
            raise SystemExit(f"{tool} is required; a missing tool is not a pass")
    base = Path(args.out) if args.out else Path(tempfile.mkdtemp(prefix="tei-ord-receipt-"))
    base = base.resolve()
    if base == REPO or REPO in base.parents:
        raise SystemExit("output directory must be outside the repository")
    go, goroot = exact_go(args.go)
    receipts = []
    for i in range(args.repeat):
        out = base if args.repeat == 1 else base / f"run-{i + 1}"
        receipts.append(one_run(go, goroot, out, args))
    ok = all(r["passed"] for r in receipts)
    if args.repeat > 1:
        stable = {r["chain"].get("stableSummaryDigest") for r in receipts}
        summary = {"runs": len(receipts), "passedRuns": sum(1 for r in receipts if r["passed"]),
                   "stableSummaryDigests": sorted(x for x in stable if x),
                   "judgmentsStable": len(stable) == 1 and None not in stable}
        (base / "repeat-summary.json").write_text(json.dumps(summary, indent=2) + "\n")
        print(json.dumps(summary))
        ok = ok and summary["judgmentsStable"]
    return 0 if ok else 1


if __name__ == "__main__":
    sys.exit(main())
