import test from "node:test";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";

const main = readFileSync(new URL("./main.ts", import.meta.url), "utf8");
const markup = readFileSync(new URL("./index.html", import.meta.url), "utf8");
const productReader = readFileSync(new URL("./pin-reader.ts", import.meta.url), "utf8");

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
  assert.match(source, /readPinnedSubject\(client!, request, catalog\)/);
  assert.match(productReader, /client\.inspectSubject\(/);
  assert.match(productReader, /inspectEligibility\(catalog, request\.subject\)/);
  assert.match(productReader, /readInspection\(reply\.value, request\.subject, catalog\)/);
  assert.match(source, /acceptPinObservation\(/);
  assert.match(source, /pinRequest\(pinState\)/);
});

test("PIN-WIRING-02: background Pin cannot directly repaint the current selected view", () => {
  // A background reply also calls renderPin(); guard that indirect path.
  const source = [
    bodyBetween("function renderPin() {", "\nasync function refreshPinnedSubject() {"),
    bodyBetween("async function refreshPinnedSubject() {", "\nfunction renderList() {"),
  ].join("\n");
  assert.doesNotMatch(productReader, /\b(?:document|window)\./);
  assert.doesNotMatch(productReader, /\belement\s*\(/);
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
    "pin-current", "unpin-current", "inspect-pinned", "open-pinned",
    "pin-subject", "pin-status", "pin-version",
  ]) {
    assert.match(main, new RegExp(`["']${id}["']`));
    assert.equal(markup.split(`id="${id}"`).length - 1, 1, `missing or duplicate id: ${id}`);
  }
  assert.match(pinUI, /pinState\.observation\.state/);
  assert.match(main, /responses\.invalidate\("pinned-inspection"\)/);
  for (const id of ["pin-current", "unpin-current", "inspect-pinned", "open-pinned"])
    assert.match(main, new RegExp(`element\\(["']${id}["']\\)\\.addEventListener\\(["']click["']`));
});

// These source-level wiring checks are regression guards, not DOM, Native Host,
// current disclosure, Windows WebView2, or end-to-end Harness evidence.


test("PIN-WIRING-04: workspace return permits explicit rebind, not automatic old observation acceptance", () => {
  const render = bodyBetween("function renderPin() {", "\nasync function refreshPinnedSubject() {");
  const click = bodyBetween('element("inspect-pinned").addEventListener("click", () => {',
    '\nrenderPin();\nsearch.addEventListener("input", renderList);');
  const workspaceLoad = bodyBetween("async function loadWorkspace() {",
    "\nconst panel = mountCommandPanel(");
  assert.match(render, /intent\.workspaceId !== workspace\.value/);
  assert.match(render, /!pinnedEligible/);
  assert.match(render, /intent\.workspaceEpoch !== workspaceEpoch/);
  assert.match(click, /inspectEligibility\(catalogSnapshot, intent\.subject\)\.ok/);
  assert.match(click, /pinSubject\(pinState, workspace\.value, workspaceEpoch, intent\.subject\)/);
  assert.match(click, /refreshPinnedSubject\(\)/);
  assert.match(workspaceLoad, /pinState = invalidatePin\(pinState\)/);
  assert.doesNotMatch(workspaceLoad, /pinSubject\(pinState/);
});

test("PIN-WIRING-05: a changed Catalog cannot silently rebind an old Subject Pin", () => {
  const render = bodyBetween("function renderPin() {", "\nasync function refreshPinnedSubject() {");
  const refresh = bodyBetween("async function refreshPinnedSubject() {", "\nfunction renderList() {");
  const pinClick = bodyBetween('element("pin-current").addEventListener("click", () => {',
    '\nelement("unpin-current").addEventListener("click", () => {');
  const unpinClick = bodyBetween('element("unpin-current").addEventListener("click", () => {',
    '\nelement("inspect-pinned").addEventListener("click", () => {');
  const recheck = bodyBetween('element("inspect-pinned").addEventListener("click", () => {',
    '\nrenderPin();\nsearch.addEventListener("input", renderList);');
  assert.match(main, /import \{ readPinnedSubject, catalogIdentity \} from "\.\/pin-reader\.ts"/);
  assert.match(main, /pinnedCatalogIdentity === catalogIdentity\(catalogSnapshot\)/);
  assert.match(render, /pinnedCatalogMatches\(\)/);
  assert.match(render, /モデルの版が未取得か、固定後に変わりました/);
  assert.match(refresh, /!pinnedCatalogMatches\(\)/);
  assert.match(pinClick, /pinnedCatalogIdentity = catalogIdentity\(catalogSnapshot\)/);
  assert.match(unpinClick, /pinnedCatalogIdentity = null/);
  assert.match(recheck, /!pinnedCatalogMatches\(\)/);
});

test("PIN-WIRING-06: reply acceptance checks drift and distinguishes stale projections from Host denial", () => {
  const reader = bodyBetween("async function refreshPinnedSubject() {", "\nfunction renderList() {");
  assert.match(reader, /catalogSnapshot !== catalog \|\| !pinnedCatalogMatches\(\)/);
  assert.match(reader, /"STALE_MODEL_PROJECTION"/);
  assert.match(reader, /"INVALID_INSPECTION_RESPONSE"/);
  assert.match(reader, /"SUBJECT_AMBIGUOUS"/);
  assert.match(reader, /"SUBJECT_NOT_FOUND"/);
  assert.match(reader, /const state = mustRequalify \? "needs-requalification" : "unavailable"/);
  assert.match(reader, /acceptPinObservation\(pinState, request, \{ state \}\)/);
});

test("PIN-WIRING-07: unknown Catalog identity disables Pin and never revives stale observation", () => {
  const render = bodyBetween("function renderPin() {", "\nasync function refreshPinnedSubject() {");
  const click = bodyBetween('element("pin-current").addEventListener("click", () => {',
    '\nelement("unpin-current").addEventListener("click", () => {');
  const background = bodyBetween("async function refreshPinnedSubject() {", "\nfunction renderList() {");
  assert.match(productReader, /typeof value === "string" && value\.trim\(\)\.length > 0/);
  assert.match(productReader, /CATALOG_IDENTITY_UNAVAILABLE/);
  assert.match(render, /catalogIdentity\(catalogSnapshot\) !== null/);
  assert.match(render, /モデルの版が未取得か/);
  assert.match(click, /catalogIdentity\(catalogSnapshot\) === null/);
  assert.match(background, /"CATALOG_IDENTITY_UNAVAILABLE"/);
});

test("PIN-WIRING-08: current Pin handlers are unique and the obsolete controller is absent", () => {
  for (const id of ["pin-current", "unpin-current", "inspect-pinned", "open-pinned"]) {
    const handler = `element("${id}").addEventListener("click"`;
    assert.equal(main.split(handler).length - 1, 1, `duplicate or missing handler: ${id}`);
  }
  assert.doesNotMatch(main, /\bpinReader\b/);
  assert.doesNotMatch(main, /renderPin\(pinReader\.current\(\)\)/);
  assert.doesNotMatch(main, /element\(["']pin-(?:check|release)["']\)/);
  assert.doesNotMatch(markup, /id="pin-(?:check|release)"/);
});

test("PIN-WIRING-09: only explicit Open may navigate from Pin to foreground Inspect", () => {
  const render = bodyBetween("function renderPin() {", "\nasync function refreshPinnedSubject() {");
  const open = bodyBetween('element("open-pinned").addEventListener("click", () => {',
    '\nelement("unpin-current").addEventListener("click", () => {');
  assert.match(render, /element<.*>\("open-pinned"\)\.disabled/);
  assert.match(render, /intent\.workspaceEpoch !== workspaceEpoch/);
  assert.match(render, /pinState\.observation\.state !== "inspected"/);
  assert.match(open, /intent\.workspaceId !== workspace\.value/);
  assert.match(open, /intent\.workspaceEpoch !== workspaceEpoch/);
  assert.match(open, /!pinnedCatalogMatches\(\)/);
  assert.match(open, /pinState\.observation\.state !== "inspected"/);
  assert.match(open, /!inspectEligibility\(catalogSnapshot, intent\.subject\)\.ok/);
  assert.match(open, /void inspect\(intent\.subject\)\.catch\(showFailure\)/);
  assert.doesNotMatch(open, /refreshPinnedSubject\(\)/);
  assert.match(markup, /id="open-pinned"[^>]*disabled>固定対象を開く<\/button>/);
});
