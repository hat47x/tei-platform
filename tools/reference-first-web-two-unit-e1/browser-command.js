// Browser-only transport adapter for the existing order_entry JSON API.
// This file contains no authentication rules, role decisions, or business
// success assumptions. It uses only textContent for untrusted API data.
(() => {
  "use strict";
  const form = document.querySelector("#json-command");
  const output = document.querySelector("#command-feedback");
  if (!form || !output) {
    throw new Error("missing generated form contract");
  }
  const button = form.querySelector('button[type="submit"]');
  const credential = form.querySelector('[name="credential"]');
  if (!button || !credential) {
    throw new Error("incomplete generated form");
  }

  let submitting = false;
  const display = (message, state) => {
    output.hidden = false;
    output.dataset.state = state;
    output.textContent = message;
  };
  form.addEventListener("submit", async event => {
    event.preventDefault();
    if (submitting) return;
    submitting = true;
    button.disabled = true;
    output.hidden = true;
    output.textContent = "";
    try {
      const fields = new FormData(form);
      const row = String(fields.get("row") ?? "");
      const from = String(fields.get("from") ?? "");
      const loginId = String(fields.get("loginId") ?? "");
      const secret = String(fields.get("credential") ?? "");
      if (!/^\d{1,8}$/.test(row) || !loginId || !secret || !from) {
        display("入力が不正です。実行していません。", "invalid");
        return;
      }
      const body = JSON.stringify({
        loginId, credential: secret, from, row: Number(row),
      });
      // Never retain the password in the form after building the request.
      credential.value = "";
      const response = await fetch("/scenario/submit", {
        method: "POST",
        mode: "same-origin",
        credentials: "omit",
        cache: "no-store",
        redirect: "error",
        headers: { "Content-Type": "application/json" },
        body,
      });
      const contentType = response.headers.get("content-type") ?? "";
      if (![200, 422].includes(response.status) ||
          !/^application\/json(?:\s*;|$)/i.test(contentType)) {
        throw new Error("unrecognized API transport response");
      }
      const payload = await response.json();
      if (!payload || typeof payload !== "object" ||
          !payload.result || payload.result.applicationId !== "order_entry") {
        throw new Error("unrecognized business result");
      }
      const result = payload.result;
      if (response.status === 422) {
        if (typeof payload.executionError !== "string" || !payload.executionError) {
          throw new Error("missing execution failure");
        }
        display("実行エラーです。業務結果は未確定です。状態を確認してください。", "execution-error");
        return;
      }
      if (payload.executionError) {
        throw new Error("ambiguous business/transport outcome");
      }
      if (result.stage === "completed") {
        if (!result.command || !result.observation) {
          throw new Error("completion has no command and observation");
        }
        display("処理完了。Commandの結果と独立したObservationを確認しました。", "completed");
      } else if (["authentication-denied", "authorization-denied", "validation-not-satisfied"].includes(result.stage)) {
        if (result.command || result.observation) {
          throw new Error("denied outcome contains a command or observation");
        }
        const labels = {
          "authentication-denied": "認証拒否",
          "authorization-denied": "認可拒否",
          "validation-not-satisfied": "検証条件未充足",
        };
        display(labels[result.stage] + "。処理は実行されていません。", result.stage);
      } else {
        throw new Error("unrecognized business stage");
      }
    } catch (_) {
      // A dropped connection after a Command may have had an effect. Never
      // claim it was rejected or retry it automatically.
      display("結果を確認できません。重複実行を避け、状態を確認してください。", "unknown");
    } finally {
      credential.value = "";
      submitting = false;
      button.disabled = false;
    }
  });
})();
