import { inspectEligibility, readInspection } from "./presentation.ts";
import type { Catalog } from "./presentation.ts";
import type { PinRequest } from "./pin-state.ts";

/** Conservative identity for a pinned Catalog snapshot.
 * Matching fingerprints do not grant access or prove Source/Root authority.
 */
export function catalogIdentity(catalog: Catalog | null): string | null {
  if (!catalog) return null;
  const revisions = [
    catalog.modelRevision, catalog.worldFingerprint, catalog.sourceIndexFingerprint,
  ];
  // A missing revision is not evidence that two Catalogs identify the same source.
  if (!revisions.every((value) => typeof value === "string" && value.trim().length > 0))
    return null;
  return JSON.stringify(revisions);
}

/** The same read-only Product operation used by the selected Inspector. */
type Reply =
  | { ok: true; value: unknown }
  | { ok: false; failure: { code: string } };

export type PinnedProductReader = {
  inspectSubject(workspaceId: string, kind: string, id: string): Promise<Reply>;
};

/**
 * Verify the requested Product Subject against its Catalog snapshot and the
 * actual Host reply. Only a version is retained; this is NOT a Disclosure Grant.
 * No DOM or selected-view references belong in this function.
 */
export async function readPinnedSubject(
  client: PinnedProductReader,
  request: PinRequest,
  catalog: Catalog,
): Promise<{ version: string }> {
  // The ordinary Catalog presenter accepts empty strings for display, but a
  // Pin must not treat unknown/blank revisions as evidence of continuity.
  if (catalogIdentity(catalog) === null) throw Error("CATALOG_IDENTITY_UNAVAILABLE");
  const eligibility = inspectEligibility(catalog, request.subject);
  if (!eligibility.ok) throw Error(eligibility.reason);
  const reply = await client.inspectSubject(
    request.workspaceId, request.subject.kind, request.subject.id,
  );
  if (!reply.ok) throw Error(reply.failure.code);
  const view = readInspection(reply.value, request.subject, catalog);
  return { version: view.version };
}
