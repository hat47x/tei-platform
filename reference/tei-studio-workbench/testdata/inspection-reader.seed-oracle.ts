// Test-only oracle: the hand-written Inspector reader as it stood before the
// part was generated (Stage0 Seed). It is never imported by the Workbench and
// never bundled. The generated part is compared with it, not derived from it.
type Subject = { kind: string; id: string };
type Catalog = { worldFingerprint: string; sourceIndexFingerprint: string };
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
const subjectKey = (s: Subject) => JSON.stringify([s.kind, s.id]);
export function seedReadInspection(raw: unknown, expected: Subject, catalog: Catalog) {
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
