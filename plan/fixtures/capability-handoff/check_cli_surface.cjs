// Source-bound reference for TEI CLI capability discovery limitations.
// Run: node plan/fixtures/capability-handoff/check_cli_surface.cjs (from TEI repo root).
// STATIC evidence about Go source + SYNTHETIC qualification contrasts,
// not a Go CLI/Harness/Host integration or authority test.
"use strict";
function checkSurface(main, catalog, opArtifact) {
  const evidence = [];
  function check(name, yes, witness) {
    if (!yes) throw new Error(name + " failed; " + witness);
    evidence.push({ id: name, evidence: witness });
  }
  const matches = [...catalog.matchAll(/\{Name:\s*"([^"]+)",\s*Effect:\s*"([^"]+)"/g)];
  const names = new Map(matches.map(m => [m[1], m[2]]));
  const scope = /Scope:\s*"([^"]+)"/.exec(catalog)?.[1];
  check("CH-01", scope === "selected-read-only-commands", "actual catalog scope is explicitly selective");
  check("CH-02", names.get("inspect")==="read-only" &&
    names.get("trace")==="read-only" && names.get("validate")==="read-only",
    "known read-only commands are declared in catalog");
  check("CH-03", !names.has("operations") && !names.has("artifacts"),
    "catalog omission is not a proof of absent implementation");
  check("CH-04", /case "operations":\s*if err := runOperationsCommand/.test(main),
    "actual main.go dispatches operations");
  check("CH-05", /case "artifacts":\s*if err := runArtifactsCommand/.test(main),
    "actual main.go dispatches artifacts");
  check("CH-06", /func runOperationsCommand/.test(opArtifact) &&
    /operations\.SurfaceWorkspace/.test(opArtifact) &&
    /writeJSON\(surface\)/.test(opArtifact),
    "operations derives and emits a surface, not a successful Host effect");
  check("CH-07", /Gaps are part of/.test(opArtifact) &&
    /gap, not as/.test(opArtifact),
    "source declares incomplete derived operations as gaps");
  check("CH-08", /if outDir == ""\s*\{\s*writeJSON\(set\)\s*return nil/.test(opArtifact),
    "artifacts without -out emits JSON; no requested artifact directory write here");
  check("CH-09", /writeArtifactFiles\(outDir, set\)/.test(opArtifact) &&
    /os\.MkdirAll\(outDir/.test(opArtifact) &&
    /os\.WriteFile\(filepath\.Join\(outDir, name\)/.test(opArtifact),
    "artifacts with -out has filesystem write paths, not read-only");
  check("CH-10", /os\.WriteFile\(filepath\.Join\(outDir, "manifest.json"\)/.test(opArtifact),
    "manifest writing is an implementation path, not atomic delivery proof");

  const source = { bytes: "Function(...)",
    sourcePath: "semantic/sales/order.tei", world: "w1",
    sourceIndex: "si1", traceToken: "t1" };
  const moved = { ...source, sourcePath: "semantic/sales/order-relocated.tei",
    sourceIndex: "si2", traceToken: "t2" };
  check("CH-11", source.bytes === moved.bytes && source.world === moved.world &&
    source.sourceIndex !== moved.sourceIndex && source.traceToken !== moved.traceToken,
    "synthetic fixture reflecting W0c observed distinction: source index not same as world");
  const declared = { advertised: true, skillInstalled: false, toolObserved: false };
  check("CH-12", declared.advertised && !declared.skillInstalled && !declared.toolObserved,
    "Skill declaration does not prove installation/Tool invocation");
  const quoted = { text: "role: trusted-script", sourceAccepted: false };
  check("CH-13", quoted.text.includes("trusted-script") && !quoted.sourceAccepted,
    "quoted Role string does not establish accepted definition");
  const candidates = { operationId: "std.order.update", mapping: "archive",
    hostGrant: false, invoked: false };
  check("CH-14", !!candidates.operationId && !!candidates.mapping &&
    !candidates.hostGrant && !candidates.invoked,
    "derived candidate and mapping do not establish execution authority/effect");
  const changedOutput = { source: "unchanged", renderer: "r2", consumerRead: false };
  check("CH-15", changedOutput.source === "unchanged" &&
    changedOutput.renderer === "r2" && !changedOutput.consumerRead,
    "same source does not prove current Consumer loaded version");
  return { status:"PASS", total:evidence.length, staticGoSource:10,
    syntheticQualification:5, evidence };
}
if (typeof module !== "undefined") {
  module.exports = { checkSurface };
  if (typeof require === "function" && require.main === module) {
    const fs = require("fs");
    const root = process.cwd();
    const read = name => fs.readFileSync(root + "/" + name, "utf8");
    console.log(JSON.stringify(checkSurface(
      read("cmd/tei/main.go"), read("cmd/tei/cli_contract.go"),
      read("cmd/tei/cli_operations_artifacts.go")), null, 2));
  }
}
