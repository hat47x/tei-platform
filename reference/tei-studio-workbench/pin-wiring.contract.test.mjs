import test from "node:test";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";

const main = readFileSync(new URL("./main.ts", import.meta.url), "utf8");
const markup = readFileSync(new URL("./index.html", import.meta.url), "utf8");

function bodyBetween(start, end) {
  const first = main.indexOf(start);
  const last = main.indexOf(end, first + start.length);
  assert.ok(first >= 0 && last > first, `Workbench wiring not found: ${start}`);
  return main.slice(first + start.length, last)
    .split("\n")
    .filter(line => !line.trimStart().startsWith("//"))
    .join("\n");
}

test("PIN-WIRING-01: pinned reader uses the existing Host reader in its own response lane", () => {
  const source = bodyBetween("async function refreshPinnedSubject() {", "\nfunction renderList() {");
  assert.match(source, /responses\.run\("pinned-inspection"/);
  assert.match(source, /client!\.inspectSubject\(/);
  assert.match(source, /inspectEligibility\(catalog, request\.subject\)/);
  assert.match(source, /readInspection\(unwrap\(reply\), request\.subject, catalog\)/);
  assert.match(source, /acceptPinObservation\(/);
  assert.match(source, /pinRequest\(pinState\)/);
});

test("PIN-WIRING-02: background Pin cannot directly repaint the current selected view", () => {
  // A background reply also calls renderPin(); guard that indirect path.
  const source = [
    bodyBetween("function renderPin() {", "\nasync function refreshPinnedSubject() {"),
    bodyBetween("async function refreshPinnedSubject() {", "\nfunction renderList() {"),
  ].join("\n");
  const forbidden = [
    /\binspect\s*\(/,
    /\bselected\s*=/,
    /\beditor\s*\./,
    /\bgraph\s*\./,
    /\bnotice\s*\./,
    /\btitle\s*\./,
    /\bsetFindings\s*\(/,
    /\brenderList\s*\(/,
    /\bpanel\s*\./,
    /element(?:<[^>]+>)?\(["'](?:editor|graph|notice|subject-title|source-status|graph-note)["']\)/,
  ];
  for (const pattern of forbidden)
    assert.doesNotMatch(source, pattern, `Pinned read unexpectedly changes the selected view: ${pattern}`);
});

test("PIN-WIRING-03: explicit buttons are wired and have exactly one corresponding DOM id", () => {
  const pinUI = bodyBetween("function renderPin() {", "\nasync function refreshPinnedSubject() {");
  for (const id of [
    "pin-current", "unpin-current", "inspect-pinned",
    "pin-subject", "pin-status", "pin-version",
  ]) {
    assert.match(main, new RegExp(`["']${id}["']`));
    assert.equal(markup.split(`id="${id}"`).length - 1, 1, `missing or duplicate id: ${id}`);
  }
  assert.match(pinUI, /pinState\.observation\.state/);
  assert.match(main, /responses\.invalidate\("pinned-inspection"\)/);
  for (const id of ["pin-current", "unpin-current", "inspect-pinned"])
    assert.match(main, new RegExp(`element\\(["']${id}["']\\)\\.addEventListener\\(["']click["']`));
});

// These source-level wiring checks are regression guards, not DOM, Native Host,
// current disclosure, Windows WebView2, or end-to-end Harness evidence.
