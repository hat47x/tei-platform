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
export type Catalog = { entries: Entry[]; modelRevision: string };
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
export const subjectKey = (s: Subject) => JSON.stringify([s.kind, s.id]);
export function readCatalog(raw: unknown): Catalog {
  const v = record(raw);
  if (
    v.effect !== "read-only" ||
    !Array.isArray(v.entries) ||
    typeof v.modelRevision !== "string"
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
  return { entries, modelRevision: v.modelRevision };
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
export function readInspection(raw: unknown, expected: Subject) {
  const v = record(raw);
  if (
    v.effect !== "read-only" ||
    subjectKey(subject(v.subject)) !== subjectKey(expected) ||
    !Object.hasOwn(v, "canonical") ||
    typeof v.worldFingerprint !== "string" ||
    typeof v.sourceIndexFingerprint !== "string"
  )
    throw Error("INVALID_INSPECTION_RESPONSE");
  return {
    text: JSON.stringify(v.canonical, null, 2),
    version: `${v.worldFingerprint}:${v.sourceIndexFingerprint}`,
    sourceStatus: String(v.sourceStatus ?? "unknown"),
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
