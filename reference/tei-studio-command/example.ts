import { mountCommandPanel } from "./command.ts";

// TEIを知らない小さなアプリの入力例。権限/業務判断はこの表示部品の外側。
const root = document.getElementById("command-demo");
if (!root) throw new Error("command-demo element is required");

const panel = mountCommandPanel(root, {
  commands: [
    { id: "show-summary", label: "概要を表示", description: "ローカルのサンプル結果" },
    { id: "restricted", label: "制限された操作", description: "Host拒否の表示例" },
    { id: "outage", label: "利用不能な操作", description: "Host失敗の表示例" },
  ],
  diagnostics: [
    { id: "local-1", severity: "info", message: "サンプルアプリに接続しました。" },
  ],
  invoke: async (id, signal) => {
    if (signal.aborted) return { kind: "failure", message: "表示側が破棄されました。" };
    if (id === "show-summary") return { kind: "success", message: "3項目を確認しました。" };
    if (id === "restricted") return { kind: "denied", message: "この操作は許可されていません。" };
    return { kind: "failure", message: "サービスが利用できません。" };
  },
});

// 画面遷移時に panel.dispose() を呼び出す。Hostの副作用停止は別責務。
window.addEventListener("pagehide", () => panel.dispose(), { once: true });
