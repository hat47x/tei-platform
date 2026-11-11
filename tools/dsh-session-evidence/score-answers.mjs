#!/usr/bin/env node
"use strict";
/*
 * Mechanical pre-scoring for the G3-20261111 comparison answers
 * (TEI #2509 / TAI #168). This checks only whether each required Expected
 * fact appears in an answer text; a human Reviewer still judges claims,
 * unknowns, and false-success items against EXPECTED-and-scoring.md.
 *
 * Usage: node score-answers.mjs <evidenceDir> [--label h2-a ...]
 */
import fs from "node:fs";
import path from "node:path";

const CHECKS = [
  { id: "Q1-meaning-order", pattern: /(注文|受注|Order)/ },
  { id: "Q1-six-attributes", pattern: /(属性は6|6つの属性|6件の属性|attributes.{0,12}6)/ },
  { id: "Q2-definition-file", pattern: /semantic\/sales\/order\.tei/ },
  { id: "Q2-source-fingerprint", pattern: /f2063418/ },
  { id: "Q3-total-identifier", pattern: /sales\.order\.total/ },
  { id: "Q3-total-type", pattern: /std\.money/ },
  { id: "Q3-definition-line", pattern: /(12行|line 12|326)/ },
  { id: "Q4-validate-ok", pattern: /("ok"\s*:\s*true|ok.{0,4}true|警告.{0,3}3|warning.{0,3}3)/i },
  { id: "Q4-warning-codes", pattern: /APPLICATION_RELEASE_CONTRACT_NOT_CURRENT/ },
  { id: "Q4-warning-db", pattern: /DATABASE_RELEASE_CONTRACT_NOT_CURRENT/ },
  { id: "Q5-unknown-explicit", pattern: /未確認/ },
  { id: "Q6-cli-usage", pattern: /(tei\.exe|bin\\tei|commands|version|catalog|inspect|validate)/i },
  { id: "Q6-version-pin", pattern: /50ebf801/ },
];

function scoreFile(file) {
  const text = fs.readFileSync(file, "utf8");
  const rows = CHECKS.map((c) => ({ id: c.id, present: c.pattern.test(text) }));
  const missing = rows.filter((r) => !r.present).map((r) => r.id);
  return {
    file: path.basename(file),
    bytes: Buffer.byteLength(text, "utf8"),
    checks: rows,
    missingRequired: missing,
  };
}

function main() {
  const argv = process.argv.slice(2);
  const dir = argv[0];
  const labels = argv.slice(1).filter((a) => !a.startsWith("--"));
  const out = [];
  for (const label of labels.length > 0 ? labels : discoverLabels(dir)) {
    const file = path.join(dir, label, `${label}.final.txt`);
    if (!fs.existsSync(file)) { out.push({ label, missing: true }); continue; }
    out.push({ label, ...scoreFile(file) });
  }
  console.log(JSON.stringify(out, null, 2));
}

function discoverLabels(dir) {
  return fs.readdirSync(dir, { withFileTypes: true })
    .filter((e) => e.isDirectory())
    .map((e) => e.name)
    .filter((name) => fs.existsSync(path.join(dir, name, `${name}.final.txt`)))
    .sort();
}

main();
