#!/usr/bin/env python3
"""Observer prototype used for the 2026-11-12 state observation record (not a product tool).

Drives one child builder twice, as the entry does, but judges state separation
from what THIS process observes, never from the child's stdout `stateDir`.

  observer.py --child <exe> --child-arg-json '[...]' --work <dir> --boundary none|ns

Verdict vocabulary: PASS-LIMITED / FAIL / UNKNOWN. Nothing here is G4.
"""
import argparse, hashlib, json, os, subprocess, sys, time


def sha(path):
    h = hashlib.sha256()
    with open(path, "rb") as f:
        for chunk in iter(lambda: f.read(1 << 20), b""):
            h.update(chunk)
    return "sha256:" + h.hexdigest()


def tree(root):
    """(file count, total bytes, digest over relative path + size)."""
    n = size = 0
    h = hashlib.sha256()
    for base, dirs, files in os.walk(root):
        dirs.sort()
        for name in sorted(files):
            p = os.path.join(base, name)
            try:
                st = os.lstat(p)
            except OSError:
                continue
            n += 1
            size += st.st_size
            h.update(os.path.relpath(p, root).encode() + b"\0" + str(st.st_size).encode() + b"\n")
    return n, size, h.hexdigest()[:16]


NS_LAUNCH = r'''
set -u
A="$1"; O="$2"; shift 2
MASKS=()
while [ "$1" != "--" ]; do MASKS+=("$1"); shift; done
shift
mount --bind "$A" "$A" || exit 97
mount --bind "$O" "$O" || exit 97
mount -t tmpfs tmpfs /tmp || exit 97
for m in "${MASKS[@]}"; do mount -t tmpfs tmpfs "$m" || exit 97; done
# Every other mount becomes read-only. Failures are reported, not hidden.
findmnt -rno TARGET | sort -r | while read -r t; do
  case "$t" in "$A"|"$O"|/tmp|/proc*|/sys*|/dev/pts*) continue;; esac
  skip=0; for m in "${MASKS[@]}"; do [ "$t" = "$m" ] && skip=1; done; [ $skip = 1 ] && continue
  mount -o remount,bind,ro "$t" 2>/dev/null || echo "RO_FAILED $t" >&2
done
# Boundary self-check: which candidate escape locations are still writable?
for probe in $PROBES; do
  if ( : > "$probe/.task6-probe" ) 2>/dev/null; then rm -f "$probe/.task6-probe"; echo "WRITABLE $probe" >&2; fi
done
exec "$@"
'''


def run_attempt(args, i, state, out_dir, masks, probes):
    artifact = os.path.join(out_dir, "artifact")
    argv = [args.child] + json.loads(args.child_arg_json) + ["-artifact-out", artifact, "-state", state]
    pre = tree(state)
    st = os.stat(state)
    if args.boundary == "ns":
        env = dict(os.environ, PROBES=" ".join(probes))
        argv = ["unshare", "--user", "--map-root-user", "--mount", "bash", "-c", NS_LAUNCH, "ns",
                state, out_dir] + masks + ["--"] + argv
    else:
        env = dict(os.environ)
    child_digest_before = sha(args.child)
    t0 = time.time()
    p = subprocess.Popen(argv, stdout=subprocess.PIPE, stderr=subprocess.PIPE, env=env)
    try:
        exe = os.readlink("/proc/%d/exe" % p.pid)
    except OSError as e:
        exe = "unreadable: %s" % e
    stdout, stderr = p.communicate()
    rec = {
        "attempt": i, "observerPid": os.getpid(), "childPid": p.pid, "procExeAtStart": exe,
        "exit": p.returncode, "seconds": round(time.time() - t0, 2),
        "childDigestBefore": child_digest_before, "childDigestAfter": sha(args.child),
        "statePassed": state, "stateDevIno": [st.st_dev, st.st_ino],
        "statePre": pre, "statePost": tree(state),
        "boundaryNotes": [l for l in stderr.decode(errors="replace").splitlines() if l.startswith(("RO_FAILED", "WRITABLE"))],
    }
    try:
        rep = json.loads(stdout)
    except Exception:
        rep = {}
    rec["reportedStateDir"] = rep.get("stateDir")          # self-report, recorded only
    rec["reportedArtifactDigest"] = rep.get("artifactDigest")
    rec["buildError"] = rep.get("buildError") or (stderr.decode(errors="replace").strip().splitlines() or [""])[-1][:200] if p.returncode or rep.get("buildError") else None
    rec["artifactOnDisk"] = sha(artifact) if os.path.isfile(artifact) else None
    return rec


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--child", required=True)
    ap.add_argument("--child-arg-json", required=True)
    ap.add_argument("--work", required=True)
    ap.add_argument("--boundary", choices=["none", "ns"], default="none")
    ap.add_argument("--stateful", choices=["yes", "no"], default="yes",
                    help="does the adapter contract say the builder writes into -state?")
    ap.add_argument("--probe", action="append", default=[])
    args = ap.parse_args()

    recs = []
    for i in (1, 2):
        state = os.path.join(args.work, "attempt%d" % i, "state")
        out_dir = os.path.join(args.work, "attempt%d" % i, "out")
        os.makedirs(state); os.makedirs(out_dir)          # the observer creates both, empty
        masks = [os.path.join(args.work, "attempt%d" % j) for j in (1, 2) if j < i]
        recs.append(run_attempt(args, i, state, out_dir, masks, args.probe))

    a, b = recs
    reasons, unknown = [], []
    if a["exit"] != 0 or b["exit"] != 0 or not a["artifactOnDisk"] or not b["artifactOnDisk"]:
        reasons.append("an attempt did not produce an artifact (exit %s/%s)" % (a["exit"], b["exit"]))
    else:
        if a["artifactOnDisk"] != b["artifactOnDisk"]:
            reasons.append("artifact bytes differ")
        for r in recs:
            if r["reportedArtifactDigest"] != r["artifactOnDisk"]:
                reasons.append("attempt %d reported a digest that is not the bytes on disk" % r["attempt"])
            if r["childDigestBefore"] != r["childDigestAfter"]:
                reasons.append("attempt %d: the child file changed during the run" % r["attempt"])
    if a["stateDevIno"] == b["stateDevIno"]:
        reasons.append("the two passed state directories are the same object")
    if not reasons:
        for r in recs:
            if r["statePre"][0] != 0:
                reasons.append("attempt %d state was not empty before the run" % r["attempt"])
            elif r["statePost"][0] == 0:
                if args.stateful == "yes":
                    reasons.append("attempt %d: the build succeeded but wrote nothing into the state it was passed" % r["attempt"])
                else:
                    unknown.append("attempt %d: no state effect observed; separation is vacuous for this builder" % r["attempt"])
    if args.boundary == "none":
        unknown.append("no OS boundary: writes and reads outside the passed state were not observed")
    else:
        for r in recs:
            for note in r["boundaryNotes"]:
                unknown.append("attempt %d boundary: %s" % (r["attempt"], note))
        unknown.append("ns boundary: pre-existing host files stayed readable (ambient input is not excluded)")

    verdict = "FAIL" if reasons else "PASS-LIMITED"
    print(json.dumps({
        "boundary": args.boundary, "verdict": verdict, "failReasons": reasons, "notObserved": unknown,
        "selfReportedStateDirs": [a["reportedStateDir"], b["reportedStateDir"]],
        "selfReportMatchesPassed": [a["reportedStateDir"] == a["statePassed"], b["reportedStateDir"] == b["statePassed"]],
        "artifactBytesEqual": bool(a["artifactOnDisk"]) and a["artifactOnDisk"] == b["artifactOnDisk"],
        "attempts": recs,
    }, indent=1))


if __name__ == "__main__":
    main()
