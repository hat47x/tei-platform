"""Independent check of the three-stage run. Reads only files on disk and the
observer's record; it does not trust any `ok` the stages printed except where
it says so."""
import collections, hashlib, json, os, sys

L = sys.argv[1]
GOROOT = sys.argv[2]
SUBJECTS = ["tei", "builder", "author"]


def digest(path):
    h = hashlib.sha256()
    with open(path, "rb") as f:
        for chunk in iter(lambda: f.read(1 << 20), b""):
            h.update(chunk)
    return "sha256:" + h.hexdigest()


d0 = {s: digest(f"{L}/stage0/{s}") for s in SUBJECTS}
d1 = {s: digest(f"{L}/stage1/{s}/artifact") for s in SUBJECTS}
d2 = {s: digest(f"{L}/stage2/{s}/artifact") for s in SUBJECTS}
print("subject  stage0            stage1            stage2            1==2  0==1")
for s in SUBJECTS:
    print(f"{s:8} {d0[s][7:23]}  {d1[s][7:23]}  {d2[s][7:23]}  {d1[s]==d2[s]!s:5} {d0[s]==d1[s]}")

label = {}
for s in SUBJECTS:
    label[d0[s]] = "STAGE0:" + s
for s in SUBJECTS:
    label[d1[s]] = "stage1:" + s  # wins if a digest is shared with stage 0

for stage in ("stage1", "stage2"):
    o = json.load(open(f"{L}/{stage}.observation.json"))
    by = collections.Counter()
    glue = collections.Counter()
    for e in o["execs"]:
        if e["imageDigest"] in label:
            by[label[e["imageDigest"]]] += 1
        elif e["image"].startswith(GOROOT + "/"):
            by["toolchain"] += 1
        else:
            by["glue"] += 1
            glue[e["image"]] += 1
    print(f"\n{stage}: {len(o['execs'])} execs, exit {o['exitCode']}")
    for k, v in sorted(by.items()):
        print(f"  {k:18} {v}")
    print("  glue images:", dict(glue))
    # who started each builder, and who started each toolchain driver
    seq = {e["seq"]: e for e in o["execs"]}
    parents = collections.Counter()
    for e in o["execs"]:
        me = label.get(e["imageDigest"])
        if me and me.endswith("builder"):
            p = seq[e["parentSeq"]]
            parents[(me, label.get(p["imageDigest"], os.path.basename(p["image"])))] += 1
    print("  builder <- started by:", dict(parents))
    if o.get("filesTraced"):
        touched = collections.Counter()
        for f in o["files"]:
            if f["read"] + f["write"] == 0:
                continue
            p = f["path"]
            for area in ("stage0", "stage1"):
                if p.startswith(f"{L}/{area}/"):
                    rest = p[len(L) + 1:].split("/")
                    key = "/".join(rest[:3]) if area == "stage1" else "/".join(rest[:2])
                    touched[(key, "w" if f["write"] else "r")] += 1
        print("  opens of earlier stages:", dict(touched))

print()
for stage in ("stage1", "stage2"):
    for s in SUBJECTS:
        r = json.load(open(f"{L}/{stage}/{s}/report.json"))
        h = r.get("planHandover") or {}
        l2b = r.get("l2b") or {}
        rep = (r.get("artifactReproducibility") or {}).get("state")
        inv = (r.get("childRun") or {}).get("invokerDigest", "")
        print(f"{stage} {s:8} ok={r.get('ok')} l2b={l2b.get('claimed')} handover={h.get('state')} "
              f"reproduced={rep} invoker={label.get(inv, inv[7:19])} builderBinding={(r.get('builderBinding') or {}).get('state')}")
