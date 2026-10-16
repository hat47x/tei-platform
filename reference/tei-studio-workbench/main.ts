import { createReadOnlyTextEditor } from "../tei-studio-editor/editor.ts";
import { codeMirrorRenderer } from "../tei-studio-editor/codemirror.ts";
import { mountReadonlyGraph } from "../tei-studio-graph/graph.ts";
import { mountCommandPanel } from "../tei-studio-command/command.ts";
import { createReadOnlyClient } from "../../cmd/tei-studio/assets/client.mjs";
import { createResponseScope } from "./responses.ts";
import {
  readCatalog,
  readValidation,
  readInspection,
  relatedGraph,
  subjectKey,
  summarizeFindings,
  readXeadObservation,
} from "./presentation.ts";
import type { Entry, Subject, Finding, Catalog } from "./presentation.ts";

type Reply =
  | { ok: true; value: unknown }
  | { ok: false; failure: { code: string } };
type Client = {
  listWorkspaces(): Promise<Reply>;
  catalogWorkspace(id: string, namespace?: string): Promise<Reply>;
  validateWorkspace(id: string): Promise<Reply>;
  inspectSubject(id: string, kind: string, subject: string): Promise<Reply>;
  observeXead(id: string, relativePath: string): Promise<Reply>;
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
  selected: Subject | null = null,
  catalogSnapshot: Catalog | null = null;
let workspaceEpoch = 0,
  inspectionEpoch = 0,
  disposed = false;
const responses = createResponseScope();
let commandContext = 0;
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
function renderXead(raw: unknown) {
  const view = readXeadObservation(raw), result = element<HTMLDivElement>("xead-result");
  const sections = view.tables.map((table) => {
    const section = document.createElement("section");
    section.className = "xead-table";
    const heading = document.createElement("h3");
    heading.textContent = `${table.name || "名称なし"} · ${table.source.kind}:${table.source.id}`;
    const fields = document.createElement("ul");
    for (const field of table.fields) {
      const item = document.createElement("li");
      const type = field.dataType ? ` · 型 ${field.dataType.kind}:${field.dataType.id}` : "";
      item.textContent = `${field.name || "名称なし"} · ${field.source.kind}:${field.source.id}${type}`;
      fields.append(item);
    }
    section.append(heading, fields);
    return section;
  });
  if (view.diagnostics.length) {
    const details = document.createElement("ul");
    details.className = "xead-diagnostics";
    for (const diagnostic of view.diagnostics) {
      const item = document.createElement("li");
      item.textContent = `${diagnostic.code} · ${diagnostic.element}${diagnostic.attribute ? ` · ${diagnostic.attribute}` : ""} · ${diagnostic.occurrences}件`;
      details.append(item);
    }
    const section = document.createElement("section");
    section.className = "xead-table";
    const heading = document.createElement("h3");
    heading.textContent = "観察診断";
    section.append(heading, details);
    sections.push(section);
  }
  if (!sections.length) {
    const empty = document.createElement("p");
    empty.textContent = "表示できるテーブルまたは診断はありません。";
    sections.push(empty);
  }
  const displayedFields = view.tables.reduce((count, table) => count + table.fields.length, 0);
  const omittedTables = view.totalTables - view.tables.length;
  const omittedFields = view.totalFields - displayedFields;
  const omittedDiagnostics = view.totalDiagnostics - view.diagnostics.length;
  if (omittedTables || omittedFields || omittedDiagnostics) {
    const clipped = document.createElement("p");
    clipped.className = "hint";
    clipped.textContent = `描画件数を制限しています。未表示：テーブル ${omittedTables}件、項目 ${omittedFields}件、診断 ${omittedDiagnostics}件。`;
    sections.unshift(clipped);
  }
  result.replaceChildren(...sections);
  element("xead-status").textContent = `観察完了：テーブル ${view.totalTables}件、項目 ${view.totalFields}件、診断 ${view.totalDiagnostics}件`;
}
const xeadFailureCodes = new Set([
  "WORKSPACE_NOT_ALLOWED", "WORKSPACE_ROOT_CHANGED", "WORKSPACE_LOAD_FAILED",
  "SOURCE_INDEX_FAILED", "INVALID_SUBJECT", "UNSUPPORTED_SUBJECT_KIND",
  "SUBJECT_NOT_FOUND", "SUBJECT_AMBIGUOUS", "OPERATION_FAILED",
  "XEAD_SOURCE_READ_FAILED", "XEAD_SOURCE_TOO_LARGE", "XEAD_UNSUPPORTED_FORMAT",
  "XEAD_PARSE_FAILED", "XEAD_PROJECTION_FAILED", "HOST_TRANSPORT_FAILURE",
  "INVALID_HOST_RESPONSE", "INVALID_XEAD_RESPONSE",
]);
function xeadFailureMessage(error: unknown): string {
  const code = error instanceof Error && xeadFailureCodes.has(error.message)
    ? error.message
    : "OPERATION_FAILED";
  return `観察できませんでした：${code}`;
}
async function observeXead() {
  if (!client || !workspace.value || disposed) return;
  const id = workspace.value, path = element<HTMLInputElement>("xead-path").value.trim();
  const result = element<HTMLDivElement>("xead-result");
  responses.invalidate("xead");
  result.replaceChildren();
  if (!path) {
    element("xead-status").textContent = "作業領域相対の .xead パスを入力してください。";
    return;
  }
  element("xead-status").textContent = "観察中…";
  await responses.run("xead", () => client!.observeXead(id, path), (reply) => {
    renderXead(unwrap(reply));
  }, (error) => {
    result.replaceChildren();
    element("xead-status").textContent = xeadFailureMessage(error);
  });
}
function setFindings(findings: Finding[] | null) {
  if (!panel.setDiagnosticsForContext(commandContext, findings ?? []))
    throw Error("SELECTION_CHANGED");
  const summary = summarizeFindings(findings);
  element("diagnostic-summary").textContent = summary.text;
  return summary;
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
  return responses.run("inspection", () => client!.inspectSubject(id, subject.kind, subject.id), (reply) => {
    const view = readInspection(unwrap(reply), subject, catalogSnapshot!);
    editor.setDocument({
      id: `${id}/${subjectKey(subject)}`,
      version: view.version,
      text: view.text,
    });
    element("source-status").textContent = `参照情報：${view.sourceStatus}`;
    notice.textContent = "選択したモデルの投影を表示しています。";
    notice.dataset.state = "ready";
  }, (error) => {
    const stale = error instanceof Error && error.message === "STALE_MODEL_PROJECTION";
    editor.setDocument({
      id: "unavailable",
      version: `${context}:${request}`,
      text: stale
        ? "CatalogとInspectの版が一致しません。一覧を再読み込みしてください。"
        : "この要素の投影は取得できませんでした。",
    });
    if (stale) {
      graph.update({ width: 680, height: 240, nodes: [], edges: [] });
      element("graph-note").textContent =
        "参照中にモデルの更新を検出しました。一覧を再読み込みしてください。";
    }
    element("source-status").textContent = stale ? "版不一致" : "未取得";
    throw error;
  });
}
async function validate(signal?: AbortSignal) {
  if (!client || !workspace.value) throw Error("HOST_NOT_CONNECTED");
  const context = workspaceEpoch,
    id = workspace.value;
  let outcome: { kind: "success" | "failure"; message: string } | undefined;
  await responses.run("validation", () => client!.validateWorkspace(id), (reply) => {
  if (signal?.aborted || context !== workspaceEpoch) return;
  const { errors, warnings } = setFindings(readValidation(unwrap(reply)));
  notice.textContent = errors
    ? `検証で${errors}件のエラーが見つかりました。`
    : `検証結果：エラーなし、警告${warnings}件。`;
  notice.dataset.state = errors ? "error" : "ready";
  outcome = errors
    ? {
        kind: "failure" as const,
        message: `検証エラー ${errors}件。診断を確認してください。`,
      }
    : { kind: "success" as const, message: `検証完了。警告 ${warnings}件。` };
  }, (error) => { throw error; });
  return outcome ?? { kind: "denied" as const, message: "表示対象が変わったため結果を破棄しました。" };
}
async function loadWorkspace() {
  if (!client || !workspace.value || disposed) return;
  const context = ++workspaceEpoch,
    id = workspace.value;
  responses.reset();
  commandContext = panel.beginContext();
  ++inspectionEpoch;
  selected = null;
  entries = [];
  catalogSnapshot = null;
  renderList();
  setFindings(null);
  title.textContent = "モデルを読み込み中";
  element("source-status").textContent = "未取得";
  element("revision").textContent = "未取得";
  editor.setDocument({
    id: "empty",
    version: String(context),
    text: "作業領域を読み込んでいます。",
  });
  graph.update({ width: 680, height: 240, nodes: [], edges: [] });
  element("graph-note").textContent = "作業領域を読み込み中";
  notice.textContent = "一覧を取得しています。";
  let reply: Reply | undefined;
  await responses.run("catalog", () => client!.catalogWorkspace(id, ""), (value) => { reply = value; }, (error) => { throw error; });
  if (!reply) return;
  if (disposed || context !== workspaceEpoch) return;
  const catalog = readCatalog(unwrap(reply));
  catalogSnapshot = catalog;
  entries = catalog.entries;
  element("revision").textContent = catalog.modelRevision;
  renderList();
  const initial =
    entries.find(
      (e) =>
        e.subject.kind === "semantic-data" && e.subject.id === "sales.order",
    ) ?? entries[0];
  if (initial) {
    // Failure to Inspect must not prevent independent workspace validation.
    try {
      await inspect(initial.subject);
    } catch (error) {
      if (!disposed && context === workspaceEpoch) showFailure(error);
    }
  } else {
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
    const context = workspaceEpoch;
    const selection = inspectionEpoch;
    try {
      if (!client || !workspace.value)
        return { kind: "denied", message: "Hostに接続していません。" };
      if (id === "validate") return await validate(signal);
      if (id === "refresh") {
        await loadWorkspace();
        return { kind: "success", message: "一覧を再取得しました。" };
      }
      if (id === "inspect" && selected) {
        const accepted = await inspect(selected);
        if (!accepted || signal.aborted || selection + 1 !== inspectionEpoch)
          return { kind: "denied", message: "選択が変わったため結果を破棄しました。" };
        return { kind: "success", message: "投影を再取得しました。" };
      }
      return { kind: "denied", message: "参照する要素を選択してください。" };
    } catch (error) {
      if (!signal.aborted && context === workspaceEpoch) showFailure(error);
      return {
        kind: "failure",
        message: error instanceof Error ? error.message : "OPERATION_FAILED",
      };
    }
  },
});
workspace.addEventListener("change", () => {
  search.value = "";
  element("xead-result").replaceChildren();
  element("xead-status").textContent = "作業領域を切り替えました。未取得";
  void loadWorkspace().catch(showFailure);
});
element("xead-path").addEventListener("input", () => {
  responses.invalidate("xead");
  element("xead-result").replaceChildren();
  element("xead-status").textContent = "パスが変更されました。未取得";
});
element("observe-xead").addEventListener("click", () => void observeXead());
search.addEventListener("input", renderList);
element("validate").addEventListener(
  "click",
  () => void panel.execute("validate"),
);
window.addEventListener(
  "pagehide",
  () => {
    disposed = true;
    responses.dispose();
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
