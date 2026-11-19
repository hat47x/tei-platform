import { inspectEligibility, readInspection, subjectKey } from "./presentation.ts";
import type { Catalog, Subject } from "./presentation.ts";

/**
 * S-LOC-0: bounded, read-only Source Index metadata from a single qualified
 * InspectSubject reply. This is NOT a raw Source read, a source editor address,
 * a Disclosure Grant, or evidence that the Index covers every occurrence.
 *
 * No DOM, Wails service, new Host method, or filesystem operation is involved.
 */
const MAX_VISIBLE_REFERENCES = 20;
const SOURCE_LOCATIONS_ERROR = "INVALID_SOURCE_LOCATIONS_RESPONSE";

type BytePosition = { line: number; column: number; byteOffset: number };
export type SourceLocation = {
  workspaceRelativePath: string;
  role: string;
  start: BytePosition;
  end: BytePosition;
  artifactFingerprint: string;
};
export type SourceLocations = {
  sourceStatus: string;
  observedReferences: number;
  shownReferences: SourceLocation[];
  omittedReferences: number;
  /** The count refers only to the returned Source Index observation. */
  completenessEstablished: false;
};

function invalid(): never {
  throw Error(SOURCE_LOCATIONS_ERROR);
}
function object(value: unknown): Record<string, unknown> {
  if (!value || typeof value !== "object" || Array.isArray(value)) invalid();
  return value as Record<string, unknown>;
}
function exactSubject(raw: unknown, expected: Subject): void {
  const subject = object(raw);
  if (typeof subject.kind !== "string" || typeof subject.id !== "string" ||
      subjectKey({ kind: subject.kind, id: subject.id }) !== subjectKey(expected))
    invalid();
}
function relativePath(raw: unknown): string {
  if (typeof raw !== "string" || !raw || raw.startsWith("/") ||
      raw.includes("\\") || raw.includes("\0") ||
      raw.split("/").some((segment) => segment === "" || segment === "." || segment === "..") ||
      raw.split("/")[0].includes(":"))
    invalid();
  // A workspace-relative Source Index path is metadata, NEVER a file:// URL.
  return raw;
}
function bytePosition(raw: unknown): BytePosition {
  const value = object(raw);
  const { offset, line, column } = value;
  if (!Number.isSafeInteger(offset) || (offset as number) < 0 ||
      !Number.isSafeInteger(line) || (line as number) < 1 ||
      !Number.isSafeInteger(column) || (column as number) < 1)
    invalid();
  return { line: line as number, column: column as number, byteOffset: offset as number };
}
function byteRange(raw: unknown): { start: BytePosition; end: BytePosition } {
  const range = object(raw);
  const start = bytePosition(range.start), end = bytePosition(range.end);
  if (start.byteOffset > end.byteOffset) invalid();
  return { start, end };
}

/**
 * The caller supplies the Catalog for the SAME read-only Workspace selection.
 * readInspection rejects a different Subject, write response, invalid limits
 * and mismatched World/Source Index fingerprints before locations are exposed.
 */
export function readSourceLocations(
  raw: unknown, expected: Subject, catalog: Catalog,
): SourceLocations {
  const inspection = readInspection(raw, expected, catalog);
  const sources = object(object(raw).sources);
  exactSubject(sources.subject, expected);

  // Go's nil []Reference may serialize as null, not [].
  const references: unknown = sources.references === null ? [] : sources.references;
  const artifacts: unknown = sources.artifacts === null ? [] : sources.artifacts;
  if (!Array.isArray(references) || !Array.isArray(artifacts)) invalid();

  const fingerprints = new Map<string, string>();
  for (const rawArtifact of artifacts) {
    const artifact = object(rawArtifact);
    const path = relativePath(artifact.path);
    if (fingerprints.has(path) ||
        typeof artifact.fingerprint !== "string" || !artifact.fingerprint)
      invalid();
    fingerprints.set(path, artifact.fingerprint);
  }

  const visible: SourceLocation[] = [];
  for (const [i, rawReference] of references.entries()) {
    const reference = object(rawReference);
    exactSubject(reference.subject, expected);
    const path = relativePath(reference.artifactPath);
    const fingerprint = fingerprints.get(path);
    if (!fingerprint || typeof reference.role !== "string" || !reference.role)
      invalid();
    const range = byteRange(reference.range);
    if (i < MAX_VISIBLE_REFERENCES) {
      visible.push({
        workspaceRelativePath: path,
        role: reference.role,
        start: range.start,
        end: range.end,
        artifactFingerprint: fingerprint,
      });
    }
  }
  return {
    sourceStatus: inspection.sourceStatus,
    observedReferences: references.length,
    shownReferences: visible,
    omittedReferences: Math.max(0, references.length - visible.length),
    completenessEstablished: false,
  };
}

/**
 * A product-Workbench draft may reuse the Reference editor's typed envelope,
 * but Source Index locations are only observations. This function neither
 * verifies raw Source bytes nor grants a Writer, Review, or Accept capability.
 * It must never be used as an authorization decision.
 */
export type RequiredEditDraftPreview = {
  status: "unreviewed-draft";
  workspaceHint: string;
  sourceObservation: SourceLocation;
  sourceIndexCompletenessEstablished: false;
  sourceBytesVerified: false;
  sourceWriteAuthorized: false;
  edit: {
    schema: "tei.reference.studio-edit/v0";
    expectedModelRevision: string;
    id: string;
    origin: "studio";
    intent: string;
    view: "required";
    subject: string;
    operation: "set-attribute-required";
    required: boolean;
  };
};

function exactDraftText(value: unknown, label: string): string {
  if (typeof value !== "string" || !value || value !== value.trim())
    throw Error("INVALID_REQUIRED_EDIT_" + label);
  return value;
}

/**
 * Prepare ONE unreviewed Data-attribute candidate from a current, qualified
 * read-only inspection. A unique observed anchor is not evidence that the
 * Index is complete or that this Workspace is writable.
 */
export function previewRequiredEditDraft(
  inspection: unknown,
  selected: Subject,
  catalog: Catalog,
  workspaceHint: string,
  candidateID: string,
  intent: string,
  required: boolean,
): RequiredEditDraftPreview {
  if (selected.kind !== "semantic-attribute" ||
      typeof selected.id !== "string" || !selected.id)
    throw Error("REQUIRED_EDIT_NEEDS_ATTRIBUTE");
  if (typeof required !== "boolean")
    throw Error("INVALID_REQUIRED_EDIT_CHOICE");
  exactDraftText(workspaceHint, "WORKSPACE");
  exactDraftText(candidateID, "CANDIDATE_ID");
  exactDraftText(intent, "INTENT");
  if (!/^sha256:[0-9a-f]{64}$/.test(catalog.modelRevision))
    throw Error("INVALID_REQUIRED_EDIT_MODEL_REVISION");

  // The same Catalog must qualify the Subject before inspecting. A Graph glyph,
  // a table row number and a Pin observation are not authoring identities.
  const eligibility = inspectEligibility(catalog, selected);
  if (!eligibility.ok)
    throw Error("REQUIRED_EDIT_SUBJECT_" + eligibility.reason);
  const locations = readSourceLocations(inspection, selected, catalog);
  const definitions = locations.shownReferences.filter(ref => ref.role === "definition");
  if (locations.sourceStatus !== "indexed" ||
      locations.omittedReferences !== 0 ||
      definitions.length !== 1 ||
      !/^sha256:[0-9a-f]{64}$/.test(definitions[0].artifactFingerprint))
    throw Error("REQUIRED_EDIT_SOURCE_OCCURRENCE_UNRESOLVED");

  return {
    status: "unreviewed-draft",
    workspaceHint,
    sourceObservation: definitions[0],
    sourceIndexCompletenessEstablished: false,
    sourceBytesVerified: false,
    sourceWriteAuthorized: false,
    edit: {
      schema: "tei.reference.studio-edit/v0",
      expectedModelRevision: catalog.modelRevision,
      id: candidateID,
      origin: "studio",
      intent,
      view: "required",
      subject: "attribute:" + selected.id,
      operation: "set-attribute-required",
      required,
    },
  };
}
