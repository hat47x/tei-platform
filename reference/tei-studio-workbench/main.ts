import { createReadOnlyTextEditor } from "../tei-studio-editor/editor.ts";
import { codeMirrorRenderer } from "../tei-studio-editor/codemirror.ts";
import { mountReadonlyGraph } from "../tei-studio-graph/graph.ts";
import { mountCommandPanel } from "../tei-studio-command/command.ts";
import { createReadOnlyClient } from "../../cmd/tei-studio/assets/client.mjs";
import {
  readCatalog,
  readValidation,
  readInspection,
  relatedGraph,
  subjectKey,
} from "./presentation.ts";
import type { Entry, Subject, Finding } from "./presentation.ts";

type Reply =
  | { ok: true; value: unknown }
  | { ok: false; failure: { code: string } };
type Client = {
  listWorkspaces(): Promise<Reply>;
  catalogWorkspace(id: string, namespace?: string): Promise<Reply>;
  validateWorkspace(id: string): Promise<Reply>;
  inspectSubject(id: string, kind: string, subject: string): Promise<Reply>;
};
const element = <T extends HTMLElement>(id: string) =>
  document.getElementById(id) as T;
const workspace = element<HTMLSelectElement>("workspace");
const search = element<HTMLInputElement>("search");
const list = element<HTMLDivElement>("subjects");
const notice = element<HTMLParagraphElement>("notice");
const title = element<HTMLHeadingElement>("subject-title");
let client: Client | null = null,
  entries: Entry[] = [],
  selected: Subject | null = null;
let workspaceEpoch = 0,
  inspectionEpoch = 0,
  disposed = false;
const editor = createReadOnlyTextEditor(element("editor"), codeMirrorRenderer, {
  id: "empty",
  version: "0",
  text: "作業領域への接続を待っています。",
});
const graph = mountReadonlyGraph(
  element("graph"),
  { width: 680, height: 240, nodes: [], edges: [] },
  {
    onSelect(key) {
      const entry = entries.find((e) => subjectKey(e.subject) === key);
      if (entry) void inspect(entry.subject).catch(showFailure);
    },
  },
);
function unwrap(reply: Reply): unknown {
  if (!reply.ok) throw Error(reply.failure.code);
  return reply.value;
}
function showFailure(error: unknown) {
  if (disposed) return;
  const code = error instanceof Error ? error.message : "OPERATION_FAILED";
  notice.textContent = `操作を完了できませんでした：${code}`;
  notice.dataset.state = "error";
}
function setFindings(findings: Finding[]) {
  panel.setDiagnostics(findings);
  const errors = findings.filter((d) => d.severity === "error").length;
  const warnings = findings.filter((d) => d.severity === "warning").length;
  element("diagnostic-summary").textContent =
    `エラー ${errors}・警告 ${warnings}`;
  return { errors, warnings };
}
function renderList() {
  const query = search.value.trim().toLocaleLowerCase();
  const matches = entries.filter((e) =>
    `${e.subject.id} ${e.subject.kind} ${e.symbol ?? ""}`
      .toLocaleLowerCase()
      .includes(query),
  );
  element("count").textContent = `${matches.length} 件`;
  const nodes = matches.map((entry) => {
    const button = document.createElement("button");
    button.type = "button";
    button.className = "subject";
    button.setAttribute(
      "aria-pressed",
      String(
        selected !== null && subjectKey(selected) === subjectKey(entry.subject),
      ),
    );
    const name = document.createElement("span");
    name.textContent = entry.symbol || entry.subject.id;
    const detail = document.createElement("small");
    detail.textContent = `${entry.subject.kind} · ${entry.subject.id}`;
    button.append(name, detail);
    button.addEventListener(
      "click",
      () => void inspect(entry.subject).catch(showFailure),
    );
    return button;
  });
  list.replaceChildren(...nodes);
  if (!nodes.length) {
    const empty = document.createElement("p");
    empty.textContent = "該当する要素はありません。";
    list.append(empty);
  }
}
async function inspect(subject: Subject) {
  if (!client || !workspace.value || disposed) return;
  const context = workspaceEpoch,
    request = ++inspectionEpoch,
    id = workspace.value;
  selected = subject;
  title.textContent = subject.id;
  renderList();
  element("source-status").textContent = "モデル投影を取得中";
  editor.setDocument({
    id: "pending",
    version: `${context}:${request}`,
    text: "選択した要素を取得しています。",
  });
  const related = relatedGraph(entries, subject);
  graph.update(related.input);
  graph.setSelectedNode(subjectKey(subject));
  element("graph-note").textContent = related.hidden
    ? `親子関係のうち8件を表示（ほか${related.hidden}件）`
    : "選択要素と、一覧に含まれる直接の親子関係";
  const reply = await client.inspectSubject(id, subject.kind, subject.id);
  if (disposed || context !== workspaceEpoch || request !== inspectionEpoch)
    return;
  try {
    const view = readInspection(unwrap(reply), subject);
    editor.setDocument({
      id: `${id}/${subjectKey(subject)}`,
      version: view.version,
      text: view.text,
    });
    element("source-status").textContent = `参照情報：${view.sourceStatus}`;
    notice.textContent = "選択したモデルの投影を表示しています。";
    notice.dataset.state = "ready";
  } catch (error) {
    editor.setDocument({
      id: "unavailable",
      version: `${context}:${request}`,
      text: "この要素の投影は取得できませんでした。",
    });
    element("source-status").textContent = "未取得";
    throw error;
  }
}
async function validate(signal?: AbortSignal) {
  if (!client || !workspace.value) throw Error("HOST_NOT_CONNECTED");
  const context = workspaceEpoch,
    id = workspace.value;
  const reply = await client.validateWorkspace(id);
  if (disposed || signal?.aborted || context !== workspaceEpoch)
    throw Error("SELECTION_CHANGED");
  const { errors, warnings } = setFindings(readValidation(unwrap(reply)));
  notice.textContent = errors
    ? `検証で${errors}件のエラーが見つかりました。`
    : `検証結果：エラーなし、警告${warnings}件。`;
  notice.dataset.state = errors ? "error" : "ready";
  return errors
    ? {
        kind: "failure" as const,
        message: `検証エラー ${errors}件。診断を確認してください。`,
      }
    : { kind: "success" as const, message: `検証完了。警告 ${warnings}件。` };
}
async function loadWorkspace() {
  if (!client || !workspace.value || disposed) return;
  const context = ++workspaceEpoch,
    id = workspace.value;
  ++inspectionEpoch;
  selected = null;
  entries = [];
  renderList();
  setFindings([]);
  title.textContent = "モデルを読み込み中";
  element("source-status").textContent = "未取得";
  element("revision").textContent = "未取得";
  editor.setDocument({
    id: "empty",
    version: String(context),
    text: "作業領域を読み込んでいます。",
  });
  graph.update({ width: 680, height: 240, nodes: [], edges: [] });
  notice.textContent = "一覧を取得しています。";
  const reply = await client.catalogWorkspace(id, "");
  if (disposed || context !== workspaceEpoch) return;
  const catalog = readCatalog(unwrap(reply));
  entries = catalog.entries;
  element("revision").textContent = catalog.modelRevision;
  renderList();
  const initial =
    entries.find(
      (e) =>
        e.subject.kind === "semantic-data" && e.subject.id === "sales.order",
    ) ?? entries[0];
  if (initial) await inspect(initial.subject);
  else {
    title.textContent = "モデルはありません";
    editor.setDocument({
      id: "empty",
      version: `${context}:loaded`,
      text: "一覧は空です。",
    });
  }
  if (context === workspaceEpoch) await validate();
}
const panel = mountCommandPanel(element("commands"), {
  commands: [
    {
      id: "validate",
      label: "作業領域を検証",
      description: "共通Go操作で診断を取得",
    },
    {
      id: "refresh",
      label: "一覧を再読み込み",
      description: "現在のモデルを再取得",
    },
    {
      id: "inspect",
      label: "選択要素を再取得",
      description: "正規化モデルの投影を更新",
    },
  ],
  async invoke(id, signal) {
    try {
      if (!client || !workspace.value)
        return { kind: "denied", message: "Hostに接続していません。" };
      if (id === "validate") return await validate(signal);
      if (id === "refresh") {
        await loadWorkspace();
        return { kind: "success", message: "一覧を再取得しました。" };
      }
      if (id === "inspect" && selected) {
        await inspect(selected);
        return { kind: "success", message: "投影を再取得しました。" };
      }
      return { kind: "denied", message: "参照する要素を選択してください。" };
    } catch (error) {
      showFailure(error);
      return {
        kind: "failure",
        message: error instanceof Error ? error.message : "OPERATION_FAILED",
      };
    }
  },
});
workspace.addEventListener("change", () => {
  search.value = "";
  void loadWorkspace().catch(showFailure);
});
search.addEventListener("input", renderList);
element("validate").addEventListener(
  "click",
  () => void panel.execute("validate"),
);
window.addEventListener(
  "pagehide",
  () => {
    disposed = true;
    ++workspaceEpoch;
    ++inspectionEpoch;
    panel.dispose();
    graph.dispose();
    editor.destroy();
  },
  { once: true },
);

async function connect() {
  // The runtime URL is served by the actual Wails Host. No local mock fallback.
  const runtimeURL = "/wails/runtime.js";
  const runtime = await import(runtimeURL);
  if (typeof runtime.Call?.ByName !== "function")
    throw Error("WAILS_RUNTIME_UNAVAILABLE");
  client = createReadOnlyClient((name: string, ...args: unknown[]) =>
    runtime.Call.ByName(name, ...args),
  ) as Client;
  const refs = unwrap(await client.listWorkspaces());
  if (
    !Array.isArray(refs) ||
    !refs.every(
      (r) => r && typeof r.id === "string" && typeof r.label === "string",
    )
  )
    throw Error("INVALID_WORKSPACE_LIST");
  workspace.replaceChildren(
    ...refs.map((ref) => {
      const option = document.createElement("option");
      option.value = ref.id;
      option.textContent = `${ref.label} (${ref.id})`;
      return option;
    }),
  );
  workspace.disabled = false;
  element("connection").textContent = "実Hostに接続";
  element("connection").dataset.state = "ready";
  if (!refs.length) {
    notice.textContent = "Hostが許可した作業領域はありません。";
    return;
  }
  await loadWorkspace();
}
void connect().catch((error) => {
  element("connection").textContent = "接続できません";
  showFailure(error);
});
