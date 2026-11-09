#!/usr/bin/env python3
"""Run the order-multilingual-operation chain (sample 8, stages 1-6) against a
real PostgreSQL instance started in Docker, and write a receipt.

Run it from a clean, committed checkout (for example a shallow clone of the
working branch). The script creates one container named tei-omo-<random>-pg
and removes exactly that container when it ends. It never touches any other
container. Every test of the package must PASS and none may SKIP: a skipped
real-DB test is not evidence. The chain record (chain-record.json) must also
say passed.

The chain copies reference/workspace into a temporary directory and never
writes to it. SQLite runs through the standard sqlite3 module of the Python
that runs this script (TEI_OMO_PYTHON); stage 6 runs Node (TEI_OMO_NODE) on the
unchanged probe of reference/multilingual-format. A missing sqlite3 module or
Node is a failure, not a skip. The translator is a mock: nothing is sent to an
external service. Acceptance and grants are a synthetic host in the test
process. The receipt says so; it is not evidence of product acceptance, a
translation service or a production database.
"""
import argparse
import json
import os
from pathlib import Path
import secrets
import shutil
import socket
import subprocess
import sys
import tempfile
import time

try:
    import sqlite3
except ImportError as exc:  # pragma: no cover - the check itself
    raise SystemExit(f"stage 5 needs the sqlite3 module of Python: {exc}")

REPO = Path(__file__).resolve().parents[1]
PACKAGE = "./tools/reference-order-multilingual-operation/"
PREFIX = "tei-omo-"
DATABASE = "tei_omo"
EXPECTED_TESTS = (
    "TestStagesOneToFourWithoutDatabase",
    "TestCheckBundleAgainstSourcesListsOnlyRevisedDisplays",
    "TestPageDefectsCountsLossAndDuplicates",
    "TestRenderGoSelectsPluralAndGroups",
    "TestVerifyThroughNeedsEveryStageAndOneIdentity",
    "TestThroughOrderMultilingualOperation",
)


def run(cmd, **kw):
    return subprocess.run(cmd, check=True, capture_output=True, text=True, **kw).stdout.strip()


def free_port():
    with socket.socket() as s:
        s.bind(("127.0.0.1", 0))
        return s.getsockname()[1]


def git_state():
    try:
        head = run(["git", "rev-parse", "HEAD"], cwd=REPO)
        status = run(["git", "status", "--porcelain=v1", "--untracked-files=all"], cwd=REPO)
    except (OSError, subprocess.CalledProcessError) as exc:
        return None, False, str(exc)
    return head, status == "", ""


def psql(name, sql):
    return run(["docker", "exec", name, "psql", "-U", "postgres", "-h", "127.0.0.1", "-tAc", sql])


def wait_ready(name, attempts=90):
    # Count attempts rather than wall-clock time: the WSL2 clock can jump.
    for _ in range(attempts):
        if subprocess.run(["docker", "exec", name, "psql", "-U", "postgres", "-h", "127.0.0.1",
                           "-tAc", "SELECT 1"], capture_output=True).returncode == 0:
            return
        time.sleep(1)
    raise SystemExit(f"{name} did not become ready")


def summarize(events):
    results = {}
    for ev in events:
        test = ev.get("Test")
        if test and "/" not in test and ev.get("Action") in ("pass", "fail", "skip"):
            results[test] = ev["Action"]
    return results


def chain_summary(record):
    return {
        "passed": record.get("passed"),
        "through": record.get("through"),
        "stages": [{"stage": s["stage"], "name": s["name"], "conditional": s.get("conditional", False),
                    "lockFingerprint": s.get("lockFingerprint"), "release": s.get("release"), "digest": s["digest"]}
                   for s in record.get("stages", [])],
        "faultInjections": {f["id"]: f["passed"] for f in record.get("faultInjections", [])},
        "completionConditions": {c["id"]: c["passed"] for c in record.get("completionConditions", [])},
        "conditionalConditions": {c["id"]: c["passed"] for c in record.get("conditionalConditions", [])},
        "platformGaps": len(record.get("platformGaps") or []),
        "limits": record.get("limits"),
    }


def main():
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("--out", help="receipt directory (default: new temp dir outside the repository)")
    ap.add_argument("--image", default="postgres:16-alpine")
    ap.add_argument("--go", default=shutil.which("go") or os.path.expanduser("~/sdk/go/bin/go"))
    ap.add_argument("--node", default=shutil.which("node") or "node")
    ap.add_argument("--allow-dirty", action="store_true", help="record a dirty or unknown tree instead of refusing")
    args = ap.parse_args()

    head, clean, git_error = git_state()
    if not clean and not args.allow_dirty:
        raise SystemExit("working tree has changes or untracked files (or git is unavailable); "
                         "run from a clean clone of the commit or pass --allow-dirty")
    out = Path(args.out) if args.out else Path(tempfile.mkdtemp(prefix="tei-omo-receipt-"))
    out = out.resolve()
    if out == REPO or REPO in out.parents:
        raise SystemExit("output directory must be outside the repository")
    out.mkdir(parents=True, exist_ok=True)
    record_path = out / "chain-record.json"
    if record_path.exists():
        record_path.unlink()

    name = f"{PREFIX}{secrets.token_hex(4)}-pg"
    port = free_port()
    receipt = {
        "schema": "tei.reference.order-multilingual-operation-receipt/v0",
        "repositoryRevision": head,
        "workingTreeClean": clean,
        "steps": [1, 2, 3, 4, 5, 6],
        "evidenceNature": "copied-reference-workspace-unchanged-multilingual-assets-synthetic-host-acceptance-"
                          "mock-translator-one-disposable-postgres-sqlite-file-via-python-node-probe",
        "notEvidenceFor": [
            "product acceptance, grants or authorization (synthetic host in the test process)",
            "a translation service or generative AI (mock translator, nothing sent externally)",
            "linguistic equivalence of collation (only the declared binary order is compared)",
            "a signed or authenticated distribution of the locale bundle",
            "stage 5 and 6 as completion (their preconditions are not established in the workspace)",
            "production databases or servers on separate machines",
        ],
        "image": args.image,
        "container": name,
    }
    if git_error:
        receipt["gitError"] = git_error
    try:
        if subprocess.run(["docker", "container", "inspect", name], capture_output=True).returncode == 0:
            raise SystemExit(f"container {name} already exists; refusing to reuse it")
        run(["docker", "run", "-d", "--name", name, "-e", "POSTGRES_HOST_AUTH_METHOD=trust",
             "-p", f"127.0.0.1:{port}:5432", args.image])
        wait_ready(name)
        psql(name, f"CREATE DATABASE {DATABASE}")
        receipt["serverVersion"] = psql(name, "SHOW server_version")
        receipt["imageId"] = run(["docker", "image", "inspect", "--format", "{{.Id}}", args.image])
        receipt["goVersion"] = run([args.go, "version"])
        receipt["nodeVersion"] = run([args.node, "--version"])
        receipt["python"] = sys.version.split()[0]
        receipt["sqliteVersion"] = sqlite3.sqlite_version
        env = dict(os.environ)
        env.update({
            "TEI_OMO_DSN": f"postgres://postgres@127.0.0.1:{port}/{DATABASE}?sslmode=disable",
            "TEI_OMO_CONFIRM_ISOLATED": DATABASE,
            "TEI_OMO_OUT_DIR": str(out),
            "TEI_OMO_PYTHON": sys.executable,
            "TEI_OMO_NODE": args.node,
        })
        cmd = [args.go, "test", "-json", "-count=1", "-timeout", "20m", PACKAGE]
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
    finally:
        removed = []
        if subprocess.run(["docker", "rm", "-f", "-v", name], capture_output=True).returncode == 0:
            removed.append(name)
        receipt["containersRemoved"] = removed
    (out / "receipt.json").write_text(json.dumps(receipt, indent=2, ensure_ascii=False) + "\n")
    print(json.dumps({k: receipt.get(k) for k in ("repositoryRevision", "workingTreeClean", "counts", "passed",
                                                   "serverVersion", "sqliteVersion", "nodeVersion", "testSeconds",
                                                   "workingTreeCleanAfter", "containersRemoved")}, ensure_ascii=False))
    print(f"receipt: {out / 'receipt.json'}")
    return 0 if receipt.get("passed") else 1


if __name__ == "__main__":
    sys.exit(main())
