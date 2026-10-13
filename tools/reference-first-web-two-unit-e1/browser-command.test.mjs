import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { test } from "node:test";
import { runInNewContext } from "node:vm";

const source = readFileSync(new URL("./browser-command.js", import.meta.url), "utf8");

function harness(fields, response) {
  const output = { hidden: true, textContent: "", dataset: {} };
  const button = { disabled: false };
  const credential = { value: fields.credential };
  let onSubmit;
  let sent;
  const form = {
    querySelector(selector) {
      if (selector === 'button[type="submit"]') return button;
      if (selector === '[name="credential"]') return credential;
      throw Error("unexpected selector");
    },
    addEventListener(name, callback) {
      assert.equal(name, "submit");
      onSubmit = callback;
    },
  };
  const context = {
    document: {
      querySelector(selector) {
        if (selector === "#json-command") return form;
        if (selector === "#command-feedback") return output;
        throw Error("unexpected selector");
      },
    },
    FormData: class {
      constructor(value) { assert.equal(value, form); }
      get(name) { return fields[name]; }
    },
    fetch: async (url, options) => {
      sent = { url, options };
      if (response instanceof Error) throw response;
      return {
        status: response.status,
        headers: { get: () => response.contentType ?? "application/json; charset=utf-8" },
        json: async () => response.payload,
      };
    },
  };
  runInNewContext(source, context, { filename: "browser-command.js" });
  return {
    output, button, credential, get sent() { return sent; },
    async submit() {
      await onSubmit({ preventDefault() {} });
    },
  };
}
const input = {
  loginId: "fixture-editor", credential: "synthetic-password",
  from: "2026-09-18T08:30:00Z", row: "0",
};

test("JSON submit succeeds only with distinct Command and Observation", async () => {
  const h = harness(input, {
    status: 200,
    payload: { result: {
      applicationId: "order_entry", stage: "completed",
      command: { mode: "output" }, observation: { nature: "synthetic" },
    } },
  });
  await h.submit();
  assert.equal(h.sent.url, "/scenario/submit");
  assert.equal(h.sent.options.method, "POST");
  assert.equal(h.sent.options.credentials, "omit");
  assert.equal(h.sent.options.redirect, "error");
  assert.deepEqual(JSON.parse(h.sent.options.body), {
    loginId: input.loginId, credential: input.credential,
    from: input.from, row: 0,
  });
  assert.equal(h.credential.value, "");
  assert.equal(h.button.disabled, false);
  assert.equal(h.output.dataset.state, "completed");
  assert.ok(!h.output.textContent.includes(input.credential));
});

for (const stage of ["authentication-denied", "authorization-denied", "validation-not-satisfied"]) {
  test(stage + " does not invent Command success", async () => {
    const h = harness(input, {
      status: 200, payload: { result: { applicationId: "order_entry", stage } },
    });
    await h.submit();
    assert.equal(h.output.dataset.state, stage);
    assert.ok(!h.output.textContent.includes(input.credential));
    assert.equal(h.credential.value, "");
  });
}

test("execution failure displays unknown business outcome", async () => {
  const h = harness(input, {
    status: 422,
    payload: {
      result: { applicationId: "order_entry", stage: "command" },
      executionError: "synthetic server failure",
    },
  });
  await h.submit();
  assert.equal(h.output.dataset.state, "execution-error");
  assert.ok(!h.output.textContent.includes("synthetic server failure"));
});

for (const scenario of [
  new Error("disconnected after submit"),
  { status: 200, payload: { result: { applicationId: "order_entry", stage: "completed" } } },
  { status: 200, payload: { result: { applicationId: "repair_intake", stage: "completed" } } },
  { status: 502, contentType: "text/plain", payload: {} },
]) {
  test("ambiguous result fails closed without revealing credential", async () => {
    const h = harness(input, scenario);
    await h.submit();
    assert.equal(h.output.dataset.state, "unknown");
    assert.equal(h.credential.value, "");
  });
}

test("invalid row never sends a Command", async () => {
  const h = harness({ ...input, row: "-1" }, new Error("not called"));
  await h.submit();
  assert.equal(h.sent, undefined);
  assert.equal(h.output.dataset.state, "invalid");
  assert.equal(h.credential.value, "");
});
