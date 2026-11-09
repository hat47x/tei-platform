/** Read-only, host-neutral text projection. No Source/Acceptance authority. */
export type PositionUnit = "utf8-byte" | "utf16-unit";
export type Severity = "error" | "warning" | "info" | "hint";
export type TextPoint = { line: number; column: number };
export type TextRange = { start: TextPoint; end: TextPoint; unit: PositionUnit };
export type EditorDiagnostic = {
  id: string; message: string; severity: Severity; range: TextRange;
};
export type EditorDocument = {
  id: string; version: string; text: string;
  diagnostics?: readonly EditorDiagnostic[];
};
export type OffsetRange = { from: number; to: number };
export type DisplayDiagnostic = OffsetRange & {
  id: string; message: string; severity: Severity;
};
export type EditorSnapshot = {
  documentId: string; version: string; text: string;
  diagnostics: readonly DisplayDiagnostic[]; selection: OffsetRange;
};
export type SelectionEvent = {
  documentId: string; version: string; selection: OffsetRange;
};
export interface EditorRenderer {
  render(snapshot: EditorSnapshot): void;
  select(range: OffsetRange): void;
  focus(): void;
  dispose(): void;
}
export interface RendererFactory<Host> {
  mount(host: Host, snapshot: EditorSnapshot,
    onSelection: (range: OffsetRange) => void): EditorRenderer;
}
export interface ReadOnlyTextEditor {
  snapshot(): EditorSnapshot;
  setDocument(document: EditorDocument): void;
  setDiagnostics(documentId: string, version: string,
    diagnostics: readonly EditorDiagnostic[]): void;
  select(range: TextRange): void;
  focus(): void;
  destroy(): void;
}
function fail(code: string): never { throw new Error(code); }
function exact(value: unknown): value is string {
  return typeof value === "string" && value.length > 0 && value.trim() === value;
}
function integer(value: unknown): value is number {
  return Number.isSafeInteger(value) && (value as number) >= 1;
}
function surrogateBoundary(text: string, index: number): boolean {
  if (index <= 0 || index >= text.length) return true;
  const left = text.charCodeAt(index - 1);
  const right = text.charCodeAt(index);
  return !(left >= 0xd800 && left <= 0xdbff && right >= 0xdc00 && right <= 0xdfff);
}
/** Convert explicit 1-based Source columns to CodeMirror's UTF-16 offsets. */
export function pointToOffset(text: string, point: TextPoint, unit: PositionUnit): number {
  if (typeof text !== "string" || !point || !integer(point.line) ||
      !integer(point.column) || (unit !== "utf8-byte" && unit !== "utf16-unit")) {
    return fail("INVALID_TEXT_POINT");
  }
  const lines = text.split("\n");
  if (point.line > lines.length) return fail("TEXT_POINT_OUT_OF_RANGE");
  const before = lines.slice(0, point.line - 1).reduce((n, l) => n + l.length + 1, 0);
  const line = lines[point.line - 1].replace(/\r$/, "");
  if (unit === "utf16-unit") {
    const columnOffset = point.column - 1;
    if (columnOffset > line.length) return fail("TEXT_POINT_OUT_OF_RANGE");
    if (!surrogateBoundary(line, columnOffset)) return fail("TEXT_POINT_SPLITS_CODEPOINT");
    return before + columnOffset;
  }
  const targetBytes = point.column - 1;
  let bytes = 0;
  let offset = 0;
  for (const scalar of line) {
    if (bytes === targetBytes) return before + offset;
    bytes += new TextEncoder().encode(scalar).length;
    offset += scalar.length;
    if (bytes > targetBytes) return fail("TEXT_POINT_SPLITS_CODEPOINT");
  }
  if (bytes === targetBytes) return before + offset;
  return fail("TEXT_POINT_OUT_OF_RANGE");
}
export function rangeToOffsets(text: string, range: TextRange): OffsetRange {
  if (!range || !range.start || !range.end) return fail("INVALID_TEXT_RANGE");
  const from = pointToOffset(text, range.start, range.unit);
  const to = pointToOffset(text, range.end, range.unit);
  if (to < from) return fail("TEXT_RANGE_REVERSED");
  return { from, to };
}
function validOffsets(text: string, range: OffsetRange): boolean {
  return !!range && Number.isSafeInteger(range.from) && Number.isSafeInteger(range.to) &&
    range.from >= 0 && range.to >= range.from && range.to <= text.length &&
    surrogateBoundary(text, range.from) && surrogateBoundary(text, range.to);
}
function normalizeDiagnostics(text: string, input: readonly EditorDiagnostic[]): DisplayDiagnostic[] {
  if (!Array.isArray(input)) return fail("INVALID_DIAGNOSTICS");
  const seen = new Set<string>();
  return input.map(d => {
    if (!d || !exact(d.id) || seen.has(d.id) || !exact(d.message) ||
        !(["error", "warning", "info", "hint"] as unknown[]).includes(d.severity)) {
      return fail("INVALID_DIAGNOSTIC");
    }
    seen.add(d.id);
    return { id: d.id, message: d.message, severity: d.severity,
      ...rangeToOffsets(text, d.range) };
  });
}
function normalizeDocument(doc: EditorDocument): EditorSnapshot {
  if (!doc || !exact(doc.id) || !exact(doc.version) || typeof doc.text !== "string") {
    return fail("INVALID_EDITOR_DOCUMENT");
  }
  return { documentId: doc.id, version: doc.version, text: doc.text,
    diagnostics: normalizeDiagnostics(doc.text, doc.diagnostics ?? []),
    selection: { from: 0, to: 0 } };
}
function copy(s: EditorSnapshot): EditorSnapshot {
  return { ...s, selection: { ...s.selection },
    diagnostics: s.diagnostics.map(d => ({ ...d })) };
}
/** Selection notification is navigation-only; it never commits edits. */
export function createReadOnlyTextEditor<Host>(
  host: Host, factory: RendererFactory<Host>, initial: EditorDocument,
  onSelection?: (event: SelectionEvent) => void
): ReadOnlyTextEditor {
  let state = normalizeDocument(initial);
  let disposed = false;
  let suppressEvents = false;
  let handle: EditorRenderer;
  const alive = () => { if (disposed) fail("EDITOR_DISPOSED"); };
  const notify = (selection: OffsetRange) => {
    if (disposed || suppressEvents || !validOffsets(state.text, selection)) return;
    state = { ...state, selection: { ...selection } };
    onSelection?.({ documentId: state.documentId, version: state.version,
      selection: { ...selection } });
  };
  handle = factory.mount(host, copy(state), notify);
  return {
    snapshot() { alive(); return copy(state); },
    setDocument(doc) {
      alive();
      const next = normalizeDocument(doc);
      if (next.documentId === state.documentId && next.version === state.version &&
          next.text !== state.text) fail("EDITOR_VERSION_REUSED_FOR_DIFFERENT_TEXT");
      suppressEvents = true;
      try { handle.render(copy(next)); } finally { suppressEvents = false; }
      state = next;
    },
    setDiagnostics(id, version, diagnostics) {
      alive();
      if (id !== state.documentId || version !== state.version) fail("STALE_EDITOR_DIAGNOSTICS");
      const next = { ...state, diagnostics: normalizeDiagnostics(state.text, diagnostics) };
      suppressEvents = true;
      try { handle.render(copy(next)); } finally { suppressEvents = false; }
      state = next;
    },
    select(range) {
      alive();
      const position = rangeToOffsets(state.text, range);
      suppressEvents = true;
      try { handle.select({ ...position }); } finally { suppressEvents = false; }
      state = { ...state, selection: { ...position } };
    },
    focus() { alive(); handle.focus(); },
    destroy() { if (disposed) return; disposed = true; handle.dispose(); }
  };
}
