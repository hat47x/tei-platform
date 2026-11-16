import { subjectKey } from "./presentation.ts";
import type { readInspection, Subject } from "./presentation.ts";

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
