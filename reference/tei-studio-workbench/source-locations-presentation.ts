import type { SourceLocations } from "./source-locations.ts";

/**
 * Display strings derived only from the Source Index metadata that
 * readSourceLocations has already validated. These are inert text values.
 * They are not source bytes, Editor positions, open URLs, or write authority.
 */
export type SourceLocationDisplayRow = Readonly<{
  pathText: string;
  roleText: string;
  rangeText: string;
  fingerprintText: string;
}>;
export type SourceLocationsPresentation = Readonly<{
  statusText: string;
  countsText: string;
  completenessText: string;
  rows: readonly SourceLocationDisplayRow[];
}>;

/** Call only after readSourceLocations and the current response-lane check. */
export function presentSourceLocations(
  view: SourceLocations,
): SourceLocationsPresentation {
  if (view.completenessEstablished !== false) {
    throw Error("INVALID_SOURCE_LOCATIONS_PRESENTATION");
  }
  return {
    statusText: `参照情報：${view.sourceStatus}`,
    countsText: `観測した参照 ${view.observedReferences} 件／表示 ${view.shownReferences.length} 件／省略 ${view.omittedReferences} 件`,
    completenessText: "参照が全件そろっているかは確認されていません。",
    rows: view.shownReferences.map((location) => ({
      pathText: location.workspaceRelativePath,
      roleText: location.role,
      rangeText: `元Sourceの行 ${location.start.line}:${location.start.column}–${location.end.line}:${location.end.column}（バイト ${location.start.byteOffset}–${location.end.byteOffset}）`,
      fingerprintText: location.artifactFingerprint,
    })),
  };
}
