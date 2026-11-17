import { inspectEligibility, readInspection, subjectKey } from "./presentation.ts";
import type { Catalog, Subject } from "./presentation.ts";

/** Pure presentation of a previously validated read-only InspectSubject reply.
 * Caller must use readInspection and a current response lane first.
 * This module has no Host, DOM, Source write, Grant, or Effect implementation.
 */
export type InspectionPresentation = Readonly<{
  document: Readonly<{ id: string; version: string; text: string }>;
  sourceStatusText: string;
  authorityLimits: readonly string[];
  noticeText: string;
}>;

export function presentInspection(
  view: ReturnType<typeof readInspection>, workspaceId: string, selected: Subject,
): InspectionPresentation {
  return {
    document: {
      id: `${workspaceId}/${subjectKey(selected)}`,
      version: view.version,
      text: view.text,
    },
    // Preserve partial/unavailable status; neither establishes complete coverage.
    sourceStatusText: `参照情報：${view.sourceStatus}`,
    authorityLimits: [...view.authorityLimits],
    noticeText: "選択したモデルの投影を表示しています。",
  };
}

/** An atomic read-only projection boundary for future Workbench consumers.
 * The Catalog must belong to the caller's current response lane. The Host reply
 * has no workspace ID, so this function cannot verify the requested workspace
 * against the Host's actual authorization scope; the Host and caller own that.
 *
 * Unlike separate readInspection + presentInspection calls, the selected
 * qualified Subject cannot be substituted between validation and rendering.
 */
export type InspectorPresentationContext = Readonly<{
  workspaceId: string;
  selected: Subject;
  catalog: Catalog;
}>;

export function readInspectorPresentation(
  raw: unknown, context: InspectorPresentationContext,
): InspectionPresentation {
  if (typeof context.workspaceId !== "string" ||
      !context.workspaceId.trim() ||
      context.workspaceId !== context.workspaceId.trim()) {
    throw Error("INVALID_INSPECTION_WORKSPACE");
  }
  const eligible = inspectEligibility(context.catalog, context.selected);
  if (!eligible.ok) throw Error(eligible.reason);
  const view = readInspection(raw, context.selected, context.catalog);
  return presentInspection(view, context.workspaceId, context.selected);
}
