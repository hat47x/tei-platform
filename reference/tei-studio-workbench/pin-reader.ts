import { inspectEligibility, readInspection } from "./presentation.ts";
import type { Catalog } from "./presentation.ts";
import type { PinRequest } from "./pin-state.ts";

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
  const eligibility = inspectEligibility(catalog, request.subject);
  if (!eligibility.ok) throw Error(eligibility.reason);
  const reply = await client.inspectSubject(
    request.workspaceId, request.subject.kind, request.subject.id,
  );
  if (!reply.ok) throw Error(reply.failure.code);
  const view = readInspection(reply.value, request.subject, catalog);
  return { version: view.version };
}
