#!/usr/bin/env python3
"""Run the order-migration-dual-db slice-1 tests against three real PostgreSQL
instances started in Docker, and write a receipt.

The script creates three containers named tei-omdd-<random>-{legacy,primary,
inventory}, and removes exactly those containers when it ends. It never
touches any other container. Every test of the package must PASS and none may
SKIP: a skipped real-DB test is not evidence.

The three instances are the same product (PostgreSQL). The receipt says so;
it is not evidence for requirements that need different database products
(P-DB-03, P-DB-04).
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
PACKAGE = "./tools/reference-order-migration-dual-db/"
ROLES = ("legacy", "primary", "inventory")
PREFIX = "tei-omdd-"
EXPECTED_DB_TESTS = (
    "TestPostgresPMIG01ResumeAfterKillsAndDetectOneYen",
    "TestPostgresPTX01KillAroundDelivery",
    "TestPostgresPTX03ShortageConvergesBothOrCompensated",
    "TestPostgresPDB02InventoryStopBoundedAndResolved",
    "TestPostgresThroughSlice1",
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


def start_container(name, image, port):
    if subprocess.run(["docker", "container", "inspect", name], capture_output=True).returncode == 0:
        raise SystemExit(f"container {name} already exists; refusing to reuse it")
    run(["docker", "run", "-d", "--name", name, "-e", "POSTGRES_HOST_AUTH_METHOD=trust",
         "-p", f"127.0.0.1:{port}:5432", image])


def wait_ready(name, deadline=90):
    end = time.time() + deadline
    while time.time() < end:
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
    out = Path(args.out) if args.out else Path(tempfile.mkdtemp(prefix="tei-omdd-receipt-"))
    out = out.resolve()
    if out == REPO or REPO in out.parents:
        raise SystemExit("output directory must be outside the repository")
    out.mkdir(parents=True, exist_ok=True)
    (out / "observed.jsonl").write_text("")

    token = secrets.token_hex(4)
    names = {role: f"{PREFIX}{token}-{role}" for role in ROLES}
    ports = {role: free_port() for role in ROLES}
    receipt = {
        "schema": "tei.reference.order-migration-dual-db-receipt/v0",
        "repositoryRevision": head,
        "workingTreeClean": clean,
        "evidenceNature": "real-postgres-three-local-instances-same-product",
        "heterogeneousDatabases": False,
        "notEvidenceFor": ["P-DB-03", "P-DB-04", "real legacy system", "other database products"],
        "image": args.image,
    }
    try:
        for role in ROLES:
            start_container(names[role], args.image, ports[role])
        for role in ROLES:
            wait_ready(names[role])
            run(["docker", "exec", names[role], "psql", "-U", "postgres", "-h", "127.0.0.1",
                 "-c", f"CREATE DATABASE tei_omdd_{role}"])
        receipt["serverVersions"] = {role: run(["docker", "exec", names[role], "psql", "-U", "postgres", "-h", "127.0.0.1",
                                                "-tAc", "SHOW server_version"]) for role in ROLES}
        receipt["imageId"] = run(["docker", "image", "inspect", "--format", "{{.Id}}", args.image])
        receipt["goVersion"] = run([args.go, "version"])
        env = dict(os.environ)
        for role in ROLES:
            env[f"TEI_OMDD_{role.upper()}_DSN"] = f"postgres://postgres@127.0.0.1:{ports[role]}/tei_omdd_{role}?sslmode=disable"
        env["TEI_OMDD_CONFIRM_ISOLATED"] = "tei_omdd"
        env["TEI_OMDD_INVENTORY_CONTAINER"] = names["inventory"]
        env["TEI_OMDD_OUT_DIR"] = str(out)
        cmd = [args.go, "test", "-json", "-count=1", "-timeout", "20m", PACKAGE]
        receipt["command"] = " ".join(cmd[1:])
        started = time.time()
        proc = subprocess.run(cmd, cwd=REPO, env=env, capture_output=True, text=True)
        receipt["testSeconds"] = round(time.time() - started, 1)
        (out / "go-test.jsonl").write_text(proc.stdout)
        events = [json.loads(line) for line in proc.stdout.splitlines() if line.startswith("{")]
        results = summarize(events)
        counts = {a: sum(1 for v in results.values() if v == a) for a in ("pass", "fail", "skip")}
        receipt["tests"] = results
        receipt["counts"] = counts
        missing = [t for t in EXPECTED_DB_TESTS if results.get(t) != "pass"]
        receipt["observed"] = [json.loads(l) for l in (out / "observed.jsonl").read_text().splitlines() if l]
        receipt["passed"] = proc.returncode == 0 and counts["fail"] == 0 and counts["skip"] == 0 and not missing
        if missing:
            receipt["missingOrNotPassed"] = missing
        if proc.stderr.strip():
            receipt["stderrTail"] = proc.stderr.strip()[-2000:]
    finally:
        removed = []
        for name in names.values():
            if subprocess.run(["docker", "rm", "-f", "-v", name], capture_output=True).returncode == 0:
                removed.append(name)
        receipt["containersRemoved"] = removed
    (out / "receipt.json").write_text(json.dumps(receipt, indent=2, ensure_ascii=False) + "\n")
    print(json.dumps({k: receipt.get(k) for k in ("repositoryRevision", "workingTreeClean", "counts", "passed",
                                                   "serverVersions", "testSeconds", "containersRemoved")}, ensure_ascii=False))
    print(f"receipt: {out / 'receipt.json'}")
    return 0 if receipt.get("passed") else 1


if __name__ == "__main__":
    sys.exit(main())
