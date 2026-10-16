import test from "node:test";
import assert from "node:assert/strict";
import { handleXeadPathKeydown } from "./keyboard.ts";

function event(key, { isComposing = false, keyCode = 13 } = {}) {
  let prevented = 0;
  return {
    value: { key, isComposing, keyCode, preventDefault: () => prevented++ },
    preventCount: () => prevented,
  };
}

test("ordinary Enter prevents default and observes once", () => {
  const keyEvent = event("Enter");
  let observed = 0;
  assert.equal(handleXeadPathKeydown(keyEvent.value, () => observed++), true);
  assert.equal(keyEvent.preventCount(), 1);
  assert.equal(observed, 1);
});

test("IME composition Enter does not prevent default or observe", () => {
  for (const options of [{ isComposing: true }, { keyCode: 229 }]) {
    const keyEvent = event("Enter", options);
    let observed = 0;
    assert.equal(handleXeadPathKeydown(keyEvent.value, () => observed++), false);
    assert.equal(keyEvent.preventCount(), 0);
    assert.equal(observed, 0);
  }
});

test("other keys do not prevent default or observe", () => {
  const keyEvent = event("Escape", { keyCode: 27 });
  let observed = 0;
  assert.equal(handleXeadPathKeydown(keyEvent.value, () => observed++), false);
  assert.equal(keyEvent.preventCount(), 0);
  assert.equal(observed, 0);
});
