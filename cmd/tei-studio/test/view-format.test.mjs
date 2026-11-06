import test from "node:test";
import assert from "node:assert/strict";
import {summarizeView} from "../assets/view-format.mjs";

test("record view is summarized into items, commands, and unused attributes", () => {
  const sections = summarizeView({
    pattern: "order_entry.order_record",
    kind: "record",
    data: "sales.order",
    items: [{attribute: "sales.order.order_id", label: "受注番号", type: "std.identifier", input: "text", required: true}],
    commands: [{capability: "sales.order.submit", label: "submit", inputs: [{name: "order"}], outcomes: [{name: "submitted"}], effect: {kind: "update", data: "sales.order"}}],
    hooks: [{phase: "before-update", reads: ["sales.order.status"], failure: "reject"}],
    unused: [{attribute: "sales.order.status", label: "状態", type: "std.code"}],
  });
  assert.deepEqual(sections.map((section) => section.title), ["基本", "項目", "コマンド", "フック", "未使用の属性"]);
  assert.equal(sections[1].rows[0][0], "受注番号");
  assert.equal(sections[2].rows[0][1], "sales.order.submit");
  assert.equal(sections[3].rows[0][0], "before-update");
  assert.equal(sections[4].rows[0][0], "sales.order.status");
});

test("browse view is summarized into columns, filters, and sorts", () => {
  const sections = summarizeView({
    pattern: "order_entry.recent_orders",
    data: "sales.order",
    query: "order_entry.recent_orders",
    pageSize: 20,
    columns: [{attribute: "sales.order.order_id", label: "受注番号", type: "std.identifier"}],
    filters: [{attribute: "sales.order.status", label: "状態", operators: ["eq"], parameter: "status"}],
    sorts: [{attribute: "sales.order.status", label: "状態", direction: "asc"}],
  });
  assert.deepEqual(sections.map((section) => section.title), ["一覧", "列", "絞り込み", "並び順"]);
  assert.equal(sections[2].rows[0][2], "status");
  assert.equal(sections[3].rows[0][1], "asc");
});

test("application view is summarized into patterns, packages, and periods", () => {
  const sections = summarizeView({
    application: "order_entry",
    patterns: [{id: "order_entry.order_record", kind: "record"}],
    packages: [{id: "order_entry.list_to_record", version: "1", bound: true}],
    periods: [{id: "sales.period", data: "sales.order", businessDate: "sales.order.order_date", states: ["open", "closed"]}],
  });
  assert.deepEqual(sections.map((section) => section.title), ["Application", "Pattern", "Package", "期間"]);
  assert.equal(sections[2].rows[0][2], "束縛済み");
  assert.equal(sections[3].rows[0][3], "open, closed");
});

test("command run includes the plan and the reference execution", () => {
  const sections = summarizeView({
    mode: "reference",
    committed: true,
    persisted: false,
    durability: "none",
    plan: {
      capability: "sales.order.submit",
      inputs: [{name: "order", type: "sales.order", value: "ORD-1", provided: true}],
      outcomes: [{name: "submitted", payload: "sales.order"}],
    },
    outcome: {name: "submitted", payload: "sales.order"},
  });
  assert.equal(sections[0].title, "Command");
  assert.equal(sections[sections.length - 1].title, "参照実行");
  const rows = sections[sections.length - 1].rows;
  assert.equal(rows[1][0], "committed");
  assert.equal(rows[1][1], "true (unit of work)");
  assert.equal(rows[2][0], "persisted");
  assert.equal(rows[2][1], "false");
  assert.equal(rows[3][0], "durability");
  assert.equal(rows[3][1], "none");
  assert.equal(rows[9][0], "reported-effect");
  assert.equal(rows[9][1], "reported");
  assert.equal(rows[4][0], "outcome");
  assert.equal(rows[6][0], "records");
  assert.equal(rows[6][1], "0");
  assert.equal(rows[7][0], "differences");
  assert.equal(rows[7][1], "0");
});

test("package view is summarized into members and requirements", () => {
  const sections = summarizeView({
    package: "order_entry.list_to_record",
    version: "1",
    projection: "screen",
    bound: true,
    members: [{kind: "browse", pattern: "order_entry.recent_orders"}, {kind: "record", pattern: "order_entry.order_record"}],
    requires: [{kind: "semantic", id: "sales.order"}],
  });
  assert.deepEqual(sections.map((section) => section.title), ["Package", "メンバー", "前提"]);
  assert.equal(sections[0].rows[4][1], "true");
  assert.equal(sections[1].rows[1][1], "order_entry.order_record");
  assert.equal(sections[2].rows[0][1], "sales.order");
});

test("command run lists the differences the declaration did not describe", () => {
  const sections = summarizeView({
    committed: true,
    persisted: false,
    durability: "none",
    plan: {
      capability: "sales.delete_order",
      inputs: [{name: "order_id", type: "sales.order.order_id", value: "ORD-1", provided: true}],
      outcomes: [{name: "deleted", payload: "sales.order"}],
      effect: {kind: "delete", data: "sales.order"},
    },
    outcome: {name: "deleted", payload: "sales.order"},
    effect: {kind: "delete", data: "sales.order"},
    differences: [{
      code: "additional-scope",
      declared: {kind: "delete", data: "sales.order"},
      applied: {kind: "delete", data: "sales.order_line", key: "order=ORD-1"},
    }],
  });
  const last = sections[sections.length - 1];
  assert.equal(last.title, "宣言との差");
  assert.deepEqual(last.header, ["種別", "宣言", "適用"]);
  assert.equal(last.rows[0][0], "additional-scope");
  assert.equal(last.rows[0][1], "delete sales.order");
  assert.equal(last.rows[0][2], "delete sales.order_line");
});

test("validation view lists diagnostics and unanswered proposals", () => {
  const sections = summarizeView({
    diagnostics: [{diagnostic: {severity: "error", code: "INVALID_DATABASE_ACCESS", message: "datasource has invalid access mode"}}],
    proposals: [{code: "INVALID_DATABASE_ACCESS", revision: "sha256:abc", accepted: false}],
  });
  assert.deepEqual(sections.map((section) => section.title), ["診断", "修正案（未受理）"]);
  assert.equal(sections[0].rows[0][1], "INVALID_DATABASE_ACCESS");
  assert.equal(sections[1].rows[0][1], "未受理");
  assert.equal(sections[1].rows[0][2], "sha256:abc");
});
