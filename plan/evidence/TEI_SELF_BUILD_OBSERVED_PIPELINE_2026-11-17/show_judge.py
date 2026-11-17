"""Prints what the entry reported and what the observer's ledger says, side by side."""
import collections, json, re, sys

d = sys.argv[1]
r = json.load(open(d + "/report.json"))
a = r["artifactReproducibility"]
t = r["toolchainObservation"]
print("ENTRY: ok=%s l2b.claimed=%s reproducibility=%s stateSeparation.basis=%s stateObservation.kind=%s builderBinding=%s "
      "planHandover=%s toolchain.namedByPlan=%s matchesPlan=%s" % (
          r["ok"], r["l2b"]["claimed"], a["state"], (a.get("stateSeparation") or {}).get("basis"),
          (a.get("stateObservation") or {}).get("kind"), r["builderBinding"]["state"], r["planHandover"]["state"],
          t["namedByPlan"], t["matchesPlan"]))
l = json.load(open(d + "/ledger.json"))
o = json.load(open(d + "/observation.json"))
print("OBSERVER: boundedAbsence=%s" % l["boundedAbsence"]["state"])
for reason in l["boundedAbsence"].get("reasons") or []:
    print("  - " + reason)
print("  limits:", o["limits"])
print("  steps:", [(s["ordinal"], s["role"], s["execs"]) for s in l["steps"]])
print("  execs: total %d %s undeclared %d reportedOnly %d" % (
    l["execs"]["total"], l["execs"]["byRole"], l["execs"]["undeclaredTotal"], l["execs"]["reportedOnlyTotal"]))
for role, use in sorted(l["files"]["byRole"].items()):
    print("  files %-34s paths %6d read %6d write %6d remove %6d" % (role, use["paths"], use["read"], use["write"], use.get("remove", 0)))
f = l["files"]
print("  undeclared %d crossStep %d writeToReadOnly %d probesOutsideScope %d" % (
    f["undeclaredTotal"], f["crossStepTotal"], f["writeToReadOnlyTotal"], f["probesOutsideScope"]))
# every undeclared open, grouped, straight from the observation and the scope
sc = json.load(open(d + "/scope.json"))
roots = [p["root"] for p in sc["paths"]] + [x["root"] for x in sc["imageRoots"]]
def covered(p):
    if any(p == x or p.startswith(x.rstrip("/") + "/") for x in roots):
        return True
    return any(p == x or (x.endswith("/") and p.startswith(x)) for x in sc["osInterface"])
ordinal = {s["seq"]: s["ordinal"] for s in l["steps"]}
groups = collections.Counter()
for u in o["files"]:
    if u["read"] + u["write"] + u.get("remove", 0) and not covered(u["path"]):
        shape = re.sub(r"^(/tmp/tei-[a-z-]+-)\d+(/.*)?$", lambda m: m.group(1) + "N" + ("/**" if m.group(2) else ""), u["path"])
        shape = shape.replace(d, "<run>")
        groups[(shape, ordinal.get(u["step"], 0), u["image"].split("/")[-1], "w" if u["write"] else ("rm" if u.get("remove") else "r"))] += 1
for key, n in sorted(groups.items()):
    print("    undeclared x%d: %s" % (n, key))
