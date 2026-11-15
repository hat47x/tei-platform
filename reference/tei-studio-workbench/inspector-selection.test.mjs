import { test } from "node:test";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import {
  emitInspectorSelection, inspectorSelectionFromEvent, INSPECTOR_SELECTION_EVENT,
} from "./inspector-selection.ts";

const subject = { kind: "semantic-data", id: "sales.order" };
const scope = { workspaceId: "A", workspaceEpoch: 7, subject };

test("one explicit selection is a composed, bubbling, read-only UI request", () => {
  const target = new EventTarget();
  let count = 0;
  target.addEventListener(INSPECTOR_SELECTION_EVENT, (event) => {
    count++;
    assert.equal(event.bubbles, true);
    assert.equal(event.composed, true);
    assert.equal(event.cancelable, false);
    assert.deepEqual(inspectorSelectionFromEvent(event, "A", 7), subject);
    assert.deepEqual(Object.keys(event.detail).sort(),
      ["subject", "workspaceEpoch", "workspaceId"]);
  });
  emitInspectorSelection(target, { ...scope, effect: "commit", grant: true });
  assert.equal(count, 1);
});

test("A→B→A never accepts an old request solely because workspace ID returned", () => {
  const target = new EventTarget();
  let emitted;
  target.addEventListener(INSPECTOR_SELECTION_EVENT, (event) => { emitted = event; });
  emitInspectorSelection(target, scope);
  assert.equal(inspectorSelectionFromEvent(emitted, "B", 8), null);
  assert.equal(inspectorSelectionFromEvent(emitted, "A", 9), null);
  assert.deepEqual(inspectorSelectionFromEvent(emitted, "A", 7), subject);
});

test("malformed, ambiguous, or unscoped events never become Inspect subjects", () => {
  const bad = [
    null, {}, { ...scope, workspaceEpoch: -1 },
    { ...scope, workspaceEpoch: 2.5 },
    { ...scope, subject: { kind: "", id: "sales.order" } },
    { ...scope, subject: { kind: "semantic-data" } },
    { ...scope, workspaceId: "" },
    { ...scope, subject: null },
  ];
  for (const value of bad) {
    const event = new CustomEvent(INSPECTOR_SELECTION_EVENT, { detail: value });
    assert.equal(inspectorSelectionFromEvent(event, "A", 7), null);
  }
  assert.equal(inspectorSelectionFromEvent(new Event(INSPECTOR_SELECTION_EVENT), "A", 7), null);
  assert.equal(inspectorSelectionFromEvent(new CustomEvent("unrelated", { detail: scope }), "A", 7), null);
  assert.throws(() => emitInspectorSelection(new EventTarget(), { ...scope, subject: { kind: "", id: "invalid" } }),
    /INVALID_INSPECTOR_SELECTION_REQUEST/);
});

test("the event adapter rejects transport authority and does not bypass Inspect eligibility", () => {
  const main = readFileSync(new URL("./main.ts", import.meta.url), "utf8");
  assert.match(main, /inspectorSelectionFromEvent\(event, workspace\.value, workspaceEpoch\)/);
  assert.match(main, /emitInspectorSelection\(button,/);
  assert.match(main, /emitInspectorSelection\(element\("graph"\),/);
  assert.match(main, /inspectEligibility\(catalogSnapshot, subject\)/);
  assert.match(main, /responses\.run\("inspection",/);
  const target = new EventTarget();
  let selection;
  target.addEventListener(INSPECTOR_SELECTION_EVENT, (event) => {
    selection = inspectorSelectionFromEvent(event, "A", 7);
  });
  emitInspectorSelection(target, { ...scope, effect: "write", authority: ["fake-grant"] });
  assert.deepEqual(selection, subject);
  assert.equal(Object.hasOwn(selection, "effect"), false);
});
