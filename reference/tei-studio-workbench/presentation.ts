/** Studio-specific read-only projections; no Source or acceptance implementation. */
export type Subject = { kind: string; id: string };
export type Entry = {
  subject: Subject;
  parent?: Subject;
  symbol?: string;
  sourceStatus: string;
};
export type Finding = {
  id: string;
  severity: "info" | "warning" | "error";
  message: string;
};
export type Catalog = {
  entries: Entry[];
  modelRevision: string;
  worldFingerprint: string;
  sourceIndexFingerprint: string;
};
export type XeadObservation = {
  totalTables: number;
  totalFields: number;
  totalDiagnostics: number;
  tables: Array<{
    source: { kind: string; id: string };
    name: string;
    fields: Array<{
      source: { kind: string; id: string };
      name: string;
      dataType?: { kind: string; id: string };
    }>;
  }>;
  diagnostics: Array<{
    code: string;
    element: string;
    attribute: string;
    occurrences: number;
  }>;
};
const MAX_XEAD_TABLES = 200;
const MAX_XEAD_FIELDS = 2000;
const MAX_XEAD_DIAGNOSTICS = 500;
export function summarizeFindings(findings: readonly Finding[] | null) {
  if (findings === null)
    return { verified: false, errors: 0, warnings: 0, text: "未検証" };
  const errors = findings.filter((d) => d.severity === "error").length;
  const warnings = findings.filter((d) => d.severity === "warning").length;
  return { verified: true, errors, warnings, text: `エラー ${errors}・警告 ${warnings}` };
}
function record(value: unknown): Record<string, unknown> {
  if (!value || typeof value !== "object" || Array.isArray(value))
    throw Error("INVALID_MODEL_RESPONSE");
  return value as Record<string, unknown>;
}
function subject(value: unknown): Subject {
  const v = record(value);
  if (
    typeof v.kind !== "string" ||
    !v.kind ||
    typeof v.id !== "string" ||
    !v.id
  )
    throw Error("INVALID_SUBJECT_RESPONSE");
  return { kind: v.kind, id: v.id };
}
function optionalString(value: unknown): string {
  if (value === undefined) return "";
  if (typeof value !== "string") throw Error("INVALID_XEAD_RESPONSE");
  return value;
}
function sourceRef(value: unknown): { kind: string; id: string } {
  const v = record(value);
  if (typeof v.sourceKind !== "string" || typeof v.sourceID !== "string")
    throw Error("INVALID_XEAD_RESPONSE");
  return { kind: v.sourceKind, id: v.sourceID };
}
/** Keeps only source metadata approved for the read-only Workbench view. */
export function readXeadObservation(raw: unknown): XeadObservation {
  const value = record(raw), observation = record(value.observation);
  const limits = value.authorityLimits;
  const requiredLimits = [
    "xead-observation-is-not-canonical-meaning",
    "xead-source-identifiers-are-not-canonical-identities",
    "xead-observation-does-not-authorize-edits-or-execution",
    "unsupported-source-constructs-remain-unresolved",
  ];
  if (value.effect !== "read-only" || value.schemaVersion !== "tei.operations.xead-observation/v0" ||
      !Array.isArray(limits) || !limits.every((limit) => typeof limit === "string") ||
      !requiredLimits.every((limit) => limits.includes(limit)))
    throw Error("INVALID_XEAD_RESPONSE");
  const tables = observation.tables === undefined ? [] : observation.tables;
  const diagnostics = observation.diagnostics === undefined ? [] : observation.diagnostics;
  if (!Array.isArray(tables) || !Array.isArray(diagnostics))
    throw Error("INVALID_XEAD_RESPONSE");
  const visibleTables: XeadObservation["tables"] = [];
  let totalFields = 0, visibleFields = 0;
  for (const [tableIndex, rawTable] of tables.entries()) {
    const table = record(rawTable);
    const fields = table.fields === undefined ? [] : table.fields;
    if (!Array.isArray(fields)) throw Error("INVALID_XEAD_RESPONSE");
    totalFields += fields.length;
    const visibleTableFields: XeadObservation["tables"][number]["fields"] = [];
    for (const rawField of fields) {
      const field = record(rawField);
      const source = sourceRef(field.source), name = optionalString(field.name);
      const dataType = field.dataType === undefined ? undefined : sourceRef(field.dataType);
      if (tableIndex < MAX_XEAD_TABLES && visibleFields < MAX_XEAD_FIELDS) {
        visibleTableFields.push({ source, name, ...(dataType === undefined ? {} : { dataType }) });
        visibleFields++;
      }
    }
    const source = sourceRef(table.source), name = optionalString(table.name);
    if (tableIndex < MAX_XEAD_TABLES) visibleTables.push({ source, name, fields: visibleTableFields });
  }
  const visibleDiagnostics: XeadObservation["diagnostics"] = [];
  for (const [index, rawDiagnostic] of diagnostics.entries()) {
    const diagnostic = record(rawDiagnostic);
    if (typeof diagnostic.code !== "string" || typeof diagnostic.element !== "string" ||
        !Number.isSafeInteger(diagnostic.occurrences) || (diagnostic.occurrences as number) < 0 ||
        (diagnostic.attribute !== undefined && typeof diagnostic.attribute !== "string"))
      throw Error("INVALID_XEAD_RESPONSE");
    if (index < MAX_XEAD_DIAGNOSTICS) {
      visibleDiagnostics.push({
        code: diagnostic.code,
        element: diagnostic.element,
        attribute: optionalString(diagnostic.attribute),
        occurrences: diagnostic.occurrences as number,
      });
    }
  }
  return {
    totalTables: tables.length,
    totalFields,
    totalDiagnostics: diagnostics.length,
    tables: visibleTables,
    diagnostics: visibleDiagnostics,
  };
}
export const subjectKey = (s: Subject) => JSON.stringify([s.kind, s.id]);
export function readCatalog(raw: unknown): Catalog {
  const v = record(raw);
  if (
    v.effect !== "read-only" ||
    !Array.isArray(v.entries) ||
    typeof v.modelRevision !== "string" ||
    typeof v.worldFingerprint !== "string" ||
    typeof v.sourceIndexFingerprint !== "string"
  )
    throw Error("INVALID_CATALOG_RESPONSE");
  const seen = new Set<string>();
  const entries = v.entries.map((rawEntry) => {
    const e = record(rawEntry),
      s = subject(e.subject),
      key = subjectKey(s);
    if (seen.has(key) || typeof e.sourceStatus !== "string")
      throw Error("INVALID_CATALOG_ENTRY");
    seen.add(key);
    return {
      subject: s,
      sourceStatus: e.sourceStatus,
      ...(e.parent === undefined ? {} : { parent: subject(e.parent) }),
      ...(typeof e.symbol === "string" ? { symbol: e.symbol } : {}),
    };
  });
  return {
    entries,
    modelRevision: v.modelRevision,
    worldFingerprint: v.worldFingerprint,
    sourceIndexFingerprint: v.sourceIndexFingerprint,
  };
}
export function readValidation(raw: unknown): Finding[] {
  const v = record(raw);
  if (
    !Object.hasOwn(v, "Direct") ||
    !Object.hasOwn(v, "Diagnostics") ||
    (v.Diagnostics !== null && !Array.isArray(v.Diagnostics))
  )
    throw Error("INVALID_VALIDATION_RESPONSE");
  const diagnostics: unknown[] = [];
  if (v.Direct !== null) diagnostics.push(v.Direct);
  for (const item of (v.Diagnostics ?? []) as unknown[])
    diagnostics.push(record(item).diagnostic);
  return diagnostics.map((value, index) => {
    const d = record(value);
    if (
      !["info", "warning", "error"].includes(String(d.severity)) ||
      typeof d.code !== "string" ||
      typeof d.message !== "string"
    )
      throw Error("INVALID_DIAGNOSTIC_RESPONSE");
    return {
      id: `${index}:${d.code}`,
      severity: d.severity as Finding["severity"],
      message: `${d.code}: ${d.message}`,
    };
  });
}
export function readInspection(raw: unknown, expected: Subject, catalog: Catalog) {
  const v = record(raw);
  if (
    v.effect !== "read-only" ||
    subjectKey(subject(v.subject)) !== subjectKey(expected) ||
    !Object.hasOwn(v, "canonical") ||
    typeof v.worldFingerprint !== "string" ||
    typeof v.sourceIndexFingerprint !== "string"
  )
    throw Error("INVALID_INSPECTION_RESPONSE");
  if (!catalog ||
      v.worldFingerprint !== catalog.worldFingerprint ||
      v.sourceIndexFingerprint !== catalog.sourceIndexFingerprint)
    throw Error("STALE_MODEL_PROJECTION");
  return {
    text: JSON.stringify(v.canonical, null, 2),
    version: `${v.worldFingerprint}:${v.sourceIndexFingerprint}`,
    sourceStatus: String(v.sourceStatus ?? "unknown"),
    worldFingerprint: v.worldFingerprint,
    sourceIndexFingerprint: v.sourceIndexFingerprint,
  };
}
export function relatedGraph(entries: readonly Entry[], selected: Subject) {
  const key = subjectKey(selected);
  const active = entries.find((e) => subjectKey(e.subject) === key);
  const candidates = entries.filter(
    (e) =>
      subjectKey(e.subject) === key ||
      (e.parent && subjectKey(e.parent) === key) ||
      (active?.parent && subjectKey(e.subject) === subjectKey(active.parent)),
  );
  const ordered = active
    ? [active, ...candidates.filter((e) => subjectKey(e.subject) !== key)]
    : candidates;
  const visible = ordered.slice(0, 8),
    visibleKeys = new Set(visible.map((e) => subjectKey(e.subject)));
  return {
    hidden: Math.max(0, candidates.length - visible.length),
    input: {
      width: 680,
      height: 240,
      nodes: visible.map((e, i) => ({
        id: subjectKey(e.subject),
        label: e.symbol || e.subject.id,
        x: 95 + (i % 4) * 165,
        y: 55 + Math.floor(i / 4) * 115,
      })),
      edges: visible
        .filter((e) => e.parent && visibleKeys.has(subjectKey(e.parent)))
        .map((e) => ({
          id: `parent:${subjectKey(e.subject)}`,
          source: subjectKey(e.parent!),
          target: subjectKey(e.subject),
        })),
    },
  };
}
