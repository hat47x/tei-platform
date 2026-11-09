#!/usr/bin/env python3
"""Run the order-assist-provider steps 7 and 8 tests (offline quotes on sales
terminals with synchronisation, discount approval and an external credit
check, and the composite scenario C5) against a real PostgreSQL instance
started in Docker, and write a receipt.

The script creates one container named tei-oapo-<random>-pg and removes
exactly that container when it ends. It never touches any other container.
Every test of the package must PASS and none may SKIP: a skipped test is not
evidence. The quote realizations need Go and Node.js on PATH (generation 1
is plugins/reference-order-quote and plugins/reference-order-quote-js,
generation 2 is the sample's quotev2/; each Go source is built as a native
executable and a WASI module).

The terminals, the order server, the credit provider process and the saga
worker processes all run on the same machine as the database. The receipt
says so: nothing here is evidence of real sales terminals, a real credit
provider, real approvers, or a production deployment.
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

REPO = Path(__file__).resolve().parents[1]
PACKAGE = "./tools/reference-order-assist-provider-offline/"
PREFIX = "tei-oapo-"
DATABASE = "tei_oapo"
EXPECTED_TESTS = (
    "TestPMIX01RuleDistributionLeavesOneSiteBehind",
    "TestPostgresPMIX02OfflineSyncConflictAndReprice",
    "TestPostgresPMIX03OldGenerationTerminal",
    "TestPostgresPWF01ApprovalPinnedToQuoteDigest",
    "TestPostgresPWF02PolicyRevisionAndMigration",
    "TestPostgresPTX02CreditUnknownOutcome",
    "TestPostgresPTX03CompensationConverges",
    "TestPostgresC5ThroughOfflineToCredit",
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
        dirty = run(["git", "status", "--porcelain", "--untracked-files=no"], cwd=REPO)
    except (OSError, subprocess.CalledProcessError) as exc:
        raise SystemExit(f"git state unavailable: {exc}")
    return head, dirty == ""


def psql(name, sql):
    return run(["docker", "exec", name, "psql", "-U", "postgres", "-h", "127.0.0.1", "-tAc", sql])


def wait_ready(name, deadline=90):
    end = time.monotonic() + deadline
    while time.monotonic() < end:
        # The entrypoint's init server listens on the socket only; TCP success
        # means the final server is up.
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


def main():
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("--out", help="receipt directory (default: new temp dir outside the repository)")
    ap.add_argument("--image", default="postgres:16-alpine")
    ap.add_argument("--go", default=shutil.which("go") or os.path.expanduser("~/sdk/go/bin/go"))
    ap.add_argument("--allow-dirty", action="store_true", help="record a dirty tree instead of refusing")
    args = ap.parse_args()

    head, clean = git_state()
    if not clean and not args.allow_dirty:
        raise SystemExit("working tree has uncommitted changes; commit first or pass --allow-dirty")
    node = shutil.which("node")
    if not node:
        raise SystemExit("node is not on PATH; the quote realizations cannot run (this is not a pass)")
    out = Path(args.out) if args.out else Path(tempfile.mkdtemp(prefix="tei-oapo-receipt-"))
    out = out.resolve()
    if out == REPO or REPO in out.parents:
        raise SystemExit("output directory must be outside the repository")
    out.mkdir(parents=True, exist_ok=True)
    (out / "observed.jsonl").write_text("")
    chain = out / "chain.json"
    if chain.exists():
        chain.unlink()

    name = f"{PREFIX}{secrets.token_hex(4)}-pg"
    port = free_port()
    receipt = {
        "schema": "tei.reference.order-assist-provider-offline-receipt/v0",
        "repositoryRevision": head,
        "workingTreeClean": clean,
        "steps": [7, 8],
        "compositeScenario": "C5",
        "evidenceNature": "real-postgres-one-local-instance-simulated-terminals-and-credit-provider-same-machine",
        "notEvidenceFor": ["real sales terminals or their storage", "a real external credit provider or its API",
                           "real approvers, an identity provider or approval authority", "the order confirmation service",
                           "networks between machines", "an accepted quote contract, policy or migration table",
                           "production deployment"],
        "image": args.image,
        "container": name,
    }
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
        receipt["nodeVersion"] = run([node, "--version"])
        env = dict(os.environ)
        env["PATH"] = str(Path(args.go).parent) + os.pathsep + env.get("PATH", "")
        env["TEI_OAPO_ADMIN_DSN"] = f"postgres://postgres@127.0.0.1:{port}/{DATABASE}?sslmode=disable"
        env["TEI_OAPO_CONFIRM_ISOLATED"] = DATABASE
        env["TEI_OAPO_OUT_DIR"] = str(out)
        cmd = [args.go, "test", "-json", "-count=1", "-p", "1", "-timeout", "30m", PACKAGE]
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
        if chain.exists():
            records = json.loads(chain.read_text()).get("records", [])
            receipt["chain"] = {"records": len(records), "head": records[-1]["digest"] if records else None,
                                "steps": [r["step"] for r in records],
                                "carriedQuoteDigest": next((r["quoteDigest"] for r in records if r["step"] == "revalidated-final"), None)}
        receipt["passed"] = (proc.returncode == 0 and counts["fail"] == 0 and counts["skip"] == 0 and not missing
                             and chain.exists())
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
                                                   "serverVersion", "nodeVersion", "testSeconds", "containersRemoved")},
                     ensure_ascii=False))
    print(f"receipt: {out / 'receipt.json'}")
    return 0 if receipt.get("passed") else 1


if __name__ == "__main__":
    sys.exit(main())
