"""Writes one build declaration: a Go command and the module-local packages
`go list` says it compiles, plus go.mod and go.sum. Only locators are written;
digests are derived when the declaration is resolved."""
import json, sys

repo, d, subject, entry, ident, toolchain = sys.argv[1:7]
dirs = sorted({l.strip() for l in open("%s/%s.golist" % (d, subject)) if l.strip().startswith(repo)})
sources, definition = [], ""
for i, p in enumerate(dirs):
    ref = p[len(repo) + 1:]
    sources.append({"kind": "go-package", "id": "pkg%d" % i, "ref": ref})
    if ref == entry[2:]:
        definition = "pkg%d" % i
sources += [{"kind": "go-module", "id": "go.mod", "ref": "go.mod"},
            {"kind": "go-module", "id": "go.sum", "ref": "go.sum"}]
body = {"formatVersion": 1,
        "subject": {"kind": "plugin", "id": ident, "version": "0.0.0", "definitionSource": definition},
        "sources": sources,
        "builder": {"contract": "tei.plugin-build", "compatibleMajor": 1},
        "recipe": {"kind": "go-command", "entrypoint": entry}}
if toolchain:
    body["parameters"] = [{"name": "go.toolchain.closureDigest", "value": toolchain}]
body.update({"outputKind": "runtime-artifact", "target": "host/linux/amd64", "profile": "release"})
open("%s/%s.intent.json" % (d, subject), "w").write(json.dumps(body, indent=2) + "\n")
