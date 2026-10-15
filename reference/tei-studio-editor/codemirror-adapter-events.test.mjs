import test from "node:test";
import assert from "node:assert/strict";
import { registerHooks } from "node:module";

// Run in a separate Node test process (Node >=22.16). The real adapter is
// imported below; only @codemirror/* is stubbed. This is NOT a DOM/browser
// keyboard test and does NOT certify actual CodeMirror key handling.
const mocks = {
  "@codemirror/state": 