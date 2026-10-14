/**
 * PoC C: Host-neutral command and diagnostics UI.
 * A displayed command is not authority. Host invoke() must recheck authorization.
 */

export interface CommandDefinition {
  readonly id: string;
  readonly label: string;
  readonly description?: string;
  /** Visual availability only; never an authorization grant. */
  readonly enabled?: boolean;
}

export interface Diagnostic {
  readonly id: string;
  readonly severity: "info" | "warning" | "error";
  readonly message: string;
}

export type CommandOutcome =
  | { readonly kind: "success"; readonly message?: string }
  | { readonly kind: "denied"; readonly message: string }
  | { readonly kind: "failure"; readonly message: string };

export type CommandStatus =
  | { readonly phase: "idle" }
  | { readonly phase: "running"; readonly commandId: string }
  | { readonly phase: "success" | "denied" | "failure"; readonly commandId: string; readonly message: string };

export interface CommandSnapshot {
  readonly commands: readonly CommandDefinition[];
  readonly diagnostics: readonly Diagnostic[];
  readonly status: CommandStatus;
}

export interface CommandOptions {
  readonly commands: readonly CommandDefinition[];
  readonly diagnostics?: readonly Diagnostic[];
  /**
   * Authority and execution stay with the Host. AbortSignal signals loss of
   * interest; aborting does NOT prove cancellation of Host-side effects.
   */
  readonly invoke: (commandId: string, signal: AbortSignal) => Promise<CommandOutcome>;
}

export interface CommandPort {
  setCommands(commands: readonly CommandDefinition[]): void;
  setDiagnostics(diagnostics: readonly Diagnostic[]): void;
  snapshot(): CommandSnapshot;
  /** true = a dispatch was accepted, NOT a claim of successful execution. */
  execute(commandId: string): Promise<boolean>;
  subscribe(listener: (snapshot: CommandSnapshot) => void): () => void;
  dispose(): void;
}

function validString(value: unknown): value is string {
  return typeof value === "string" && value.trim().length > 0;
}

function normalizeCommands(value: readonly CommandDefinition[]): CommandDefinition[] {
  if (!Array.isArray(value)) throw new TypeError("commands must be an array");
  const ids = new Set<string>();
  return value.map((item) => {
    if (!item || !validString(item.id) || !validString(item.label) ||
        (item.description !== undefined && typeof item.description !== "string") ||
        (item.enabled !== undefined && typeof item.enabled !== "boolean") ||
        ids.has(item.id)) {
      throw new TypeError("invalid or duplicate command");
    }
    ids.add(item.id);
    return { id: item.id, label: item.label, description: item.description, enabled: item.enabled };
  });
}

function normalizeDiagnostics(value: readonly Diagnostic[]): Diagnostic[] {
  if (!Array.isArray(value)) throw new TypeError("diagnostics must be an array");
  const ids = new Set<string>();
  return value.map((item) => {
    if (!item || !validString(item.id) || !validString(item.message) ||
        !["info", "warning", "error"].includes(item.severity) || ids.has(item.id)) {
      throw new TypeError("invalid or duplicate diagnostic");
    }
    ids.add(item.id);
    return { id: item.id, severity: item.severity, message: item.message };
  });
}

function normalizeOutcome(value: unknown): CommandOutcome {
  if (!value || typeof value !== "object") {
    return { kind: "failure", message: "実行結果の形式が不正です。" };
  }
  const result = value as Partial<CommandOutcome>;
  if ((result.kind !== "success" && result.kind !== "denied" && result.kind !== "failure") ||
      (result.message !== undefined && typeof result.message !== "string") ||
      (result.kind !== "success" && !validString(result.message))) {
    return { kind: "failure", message: "実行結果の形式が不正です。" };
  }
  if (result.kind === "success") return { kind: "success", message: result.message };
  return { kind: result.kind, message: result.message as string };
}

export function createCommandPort(options: CommandOptions): CommandPort {
  if (!options || typeof options.invoke !== "function") throw new TypeError("invoke is required");
  let commands = normalizeCommands(options.commands);
  let diagnostics = normalizeDiagnostics(options.diagnostics ?? []);
  let status: CommandStatus = { phase: "idle" };
  let disposed = false;
  let active: AbortController | undefined;
  const listeners = new Set<(snapshot: CommandSnapshot) => void>();

  const snapshot = (): CommandSnapshot => ({
    commands: commands.map((c) => ({ ...c })),
    diagnostics: diagnostics.map((d) => ({ ...d })),
    status: { ...status },
  });
  const notify = (): void => {
    for (const listener of [...listeners]) {
      // A presentation listener cannot turn a Host outcome into a failure.
      try { listener(snapshot()); } catch { /* caller owns listener errors */ }
    }
  };
  const ensureActive = (): void => {
    if (disposed) throw new Error("command port disposed");
  };

  return {
    setCommands(value) {
      ensureActive();
      commands = normalizeCommands(value);
      notify();
    },
    setDiagnostics(value) {
      ensureActive();
      diagnostics = normalizeDiagnostics(value);
      notify();
    },
    snapshot,
    async execute(commandId) {
      if (disposed || status.phase === "running" ||
          !commands.some((c) => c.id === commandId && c.enabled !== false)) {
        return false;
      }
      const controller = new AbortController();
      active = controller;
      status = { phase: "running", commandId };
      notify();
      let outcome: CommandOutcome;
      try {
        outcome = normalizeOutcome(await options.invoke(commandId, controller.signal));
      } catch {
        // Do not expose exceptions, paths or secrets from the Host.
        outcome = { kind: "failure", message: "実行に失敗しました。" };
      }
      if (disposed || active !== controller) return false;
      active = undefined;
      status = { phase: outcome.kind, commandId, message: outcome.message ?? "完了しました。" };
      notify();
      return true;
    },
    subscribe(listener) {
      ensureActive();
      if (typeof listener !== "function") throw new TypeError("listener is required");
      listeners.add(listener);
      listener(snapshot());
      return () => { listeners.delete(listener); };
    },
    dispose() {
      if (disposed) return;
      disposed = true;
      active?.abort();
      active = undefined;
      listeners.clear();
    },
  };
}

export interface CommandPanel extends CommandPort {
  readonly element: HTMLElement;
}

/**
 * DOM adapter. No CSS, shell commands, TEI imports, Host APIs or HTML parsing.
 * Caller chooses the document, target element and styling.
 */
export function mountCommandPanel(target: HTMLElement, options: CommandOptions): CommandPanel {
  if (!target || typeof target.append !== "function" || !target.ownerDocument) {
    throw new TypeError("target must be an HTMLElement");
  }
  const model = createCommandPort(options);
  const doc = target.ownerDocument;
  const root = doc.createElement("section");
  root.setAttribute("aria-label", "コマンドと診断");
  root.className = "tei-command-panel";

  const search = doc.createElement("input");
  search.type = "search";
  search.setAttribute("aria-label", "コマンドを検索");
  const commandsList = doc.createElement("ul");
  commandsList.setAttribute("aria-label", "コマンド一覧");
  const statusNode = doc.createElement("p");
  statusNode.setAttribute("role", "status");
  statusNode.setAttribute("aria-live", "polite");
  const diagnosticsList = doc.createElement("ul");
  diagnosticsList.setAttribute("aria-label", "診断一覧");
  root.append(search, commandsList, statusNode, diagnosticsList);
  target.append(root);

  const renderCommands = (state: CommandSnapshot): void => {
    const query = search.value.trim().toLocaleLowerCase();
    const focused = (doc.activeElement as HTMLElement | null)?.dataset?.commandId;
    const nodes: HTMLElement[] = [];
    for (const command of state.commands) {
      if (!((command.label + " " + (command.description ?? "")).toLocaleLowerCase().includes(query))) continue;
      const item = doc.createElement("li");
      const button = doc.createElement("button");
      button.type = "button";
      button.dataset.commandId = command.id;
      button.textContent = command.label;
      button.disabled = command.enabled === false || state.status.phase === "running";
      button.addEventListener("click", () => { void model.execute(command.id); });
      item.append(button);
      if (command.description) {
        const description = doc.createElement("span");
        description.textContent = command.description;
        item.append(description);
      }
      nodes.push(item);
    }
    commandsList.replaceChildren(...nodes);
    if (focused && state.status.phase !== "running") {
      for (const button of Array.from(commandsList.querySelectorAll("button"))) {
        if (button.dataset.commandId === focused && !button.disabled) {
          button.focus();
          break;
        }
      }
    }
  };
  const render = (state: CommandSnapshot): void => {
    renderCommands(state);
    root.setAttribute("aria-busy", state.status.phase === "running" ? "true" : "false");
    const phase = state.status.phase;
    statusNode.textContent = phase === "idle" ? "" :
      phase === "running" ? "実行中…" :
      phase === "success" ? "成功: " + state.status.message :
      phase === "denied" ? "拒否: " + state.status.message :
      "失敗: " + state.status.message;
    diagnosticsList.replaceChildren(...state.diagnostics.map((d) => {
      const item = doc.createElement("li");
      item.dataset.severity = d.severity;
      item.textContent = d.severity + ": " + d.message;
      return item;
    }));
  };
  const onInput = (): void => { renderCommands(model.snapshot()); };
  const onListKeydown = (event: KeyboardEvent): void => {
    if (!["ArrowDown", "ArrowUp", "Home", "End"].includes(event.key)) return;
    const buttons = Array.from(commandsList.querySelectorAll("button")).filter((b) => !b.disabled);
    if (!buttons.length) return;
    const current = buttons.indexOf(doc.activeElement as HTMLButtonElement);
    const index = event.key === "Home" ? 0 :
      event.key === "End" ? buttons.length - 1 :
      event.key === "ArrowDown" ? (current + 1) % buttons.length :
      (current < 0 ? buttons.length - 1 : (current - 1 + buttons.length) % buttons.length);
    buttons[index].focus();
    event.preventDefault();
  };
  search.addEventListener("input", onInput);
  commandsList.addEventListener("keydown", onListKeydown);
  const unsubscribe = model.subscribe(render);
  let disposed = false;
  return {
    element: root,
    setCommands: model.setCommands,
    setDiagnostics: model.setDiagnostics,
    snapshot: model.snapshot,
    execute: model.execute,
    subscribe: model.subscribe,
    dispose() {
      if (disposed) return;
      disposed = true;
      unsubscribe();
      search.removeEventListener("input", onInput);
      commandsList.removeEventListener("keydown", onListKeydown);
      model.dispose();
      root.remove();
    },
  };
}
