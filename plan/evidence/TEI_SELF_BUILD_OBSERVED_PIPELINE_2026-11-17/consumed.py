"""Declared inputs vs inputs the compiler actually opened, for one traced
self-build of ./cmd/tei (first build only)."""
import collections, json, os, re, sys

D = sys.argv[1]
o = json.load(open(D + "/observation.json"))
r = json.load(open(D + "/report.json"))
tree = r["childIsolation"]["root"]
steps = sorted({e["seq"] for e in o["execs"] if e["parentSeq"] == 0})
first = steps[0]
tool = lambda image: os.path.basename(image)

staged, by_tool = set(), collections.defaultdict(set)
dirs_opened = set()
for f in o["files"]:
    if f["step"] != first or not f["path"].startswith(tree + "/"):
        continue
    rel = f["path"][len(tree) + 1:]
    if not os.path.isfile(os.path.join(sys.argv[2], rel)):
        continue  # directories are opened for listing; only files are inputs here
    if tool(f["image"]) == "builder" and f["write"]:
        staged.add(rel)
    if f["read"] and tool(f["image"]) != "builder":
        by_tool[tool(f["image"])].add(rel)

compiled = by_tool["compile"] | by_tool["asm"] | by_tool["link"]
go_only = by_tool["go"] - compiled
never = staged - by_tool["go"] - compiled


def kind(rel):
    if rel.endswith("_test.go"):
        return "_test.go"
    if rel.endswith(".go"):
        return ".go"
    if rel in ("go.mod", "go.sum"):
        return "module"
    return "other:" + os.path.splitext(rel)[1]


print("staged files (declared, copied into the isolated tree):", len(staged))
print("opened by compile/asm/link:", len(compiled & staged), collections.Counter(kind(x) for x in compiled & staged))
print("opened only by `go` (listing, build constraints, embed):", len(go_only & staged),
      collections.Counter(kind(x) for x in go_only & staged))
print("opened by no toolchain program:", len(never), collections.Counter(kind(x) for x in never))
print("compile opened but not staged (directories etc.):", len(compiled - staged), sorted(compiled - staged)[:5])
size = lambda rels: sum(os.path.getsize(os.path.join(sys.argv[2], x)) for x in rels if os.path.isfile(os.path.join(sys.argv[2], x)))
print("bytes: staged %d, compiled %d, go-only %d, never %d" % (size(staged), size(compiled & staged), size(go_only & staged), size(never)))
json.dump({"compiled": sorted(compiled & staged), "goOnly": sorted(go_only & staged), "never": sorted(never)},
          open(D + "/consumed.json", "w"), indent=1)
print("sample never:", sorted(never)[:6])
print("sample go-only non-test:", [x for x in sorted(go_only & staged) if not x.endswith("_test.go")][:8])
