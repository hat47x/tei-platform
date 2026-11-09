/** Adapter-only coordinates. The port always uses offsets in the ORIGINAL Source text. */
export type CoordinateRange = { from: number; to: number };

/**
 * CodeMirror's Text stores every line boundary as one UTF-16 unit, whereas a
 * Source CRLF occupies two. This map only folds line separators for rendering;
 * it never modifies the Source passed to the port or exposed in its callbacks.
 * Standalone CR is rejected: the port counts LF-delimited lines and cannot
 * consistently describe CodeMirror's interpretation of CR-only line breaks.
 */
export class CodeMirrorCoordinates {
  readonly text: string;
  private readonly sourceStarts: number[] = [];
  private readonly viewStarts: number[] = [];
  private readonly contentLengths: number[] = [];
  private readonly sourceLength: number;
  readonly viewLength: number;

  constructor(source: string) {
    if (typeof source !== "string") throw new Error("INVALID_EDITOR_SOURCE_TEXT");
    const pieces = source.split("\n");
    const normalized: string[] = [];
    let sourceStart = 0;
    let viewStart = 0;
    for (let i = 0; i < pieces.length; i++) {
      const hasNextLine = i < pieces.length - 1;
      const originalLine = pieces[i];
      const line = hasNextLine && originalLine.endsWith("\r")
        ? originalLine.slice(0, -1) : originalLine;
      if (line.includes("\r")) throw new Error("UNSUPPORTED_EDITOR_CR_ONLY_LINE_BREAK");
      this.sourceStarts.push(sourceStart);
      this.viewStarts.push(viewStart);
      this.contentLengths.push(line.length);
      normalized.push(line);
      sourceStart += originalLine.length + (hasNextLine ? 1 : 0);
      viewStart += line.length + (hasNextLine ? 1 : 0);
    }
    this.text = normalized.join("\n");
    this.sourceLength = source.length;
    this.viewLength = this.text.length;
    if (sourceStart !== source.length || viewStart !== this.text.length) {
      throw new Error("EDITOR_COORDINATE_INTERNAL_MISMATCH");
    }
  }

  private atOrBefore(starts: readonly number[], offset: number): number {
    let lo = 0;
    let hi = starts.length;
    while (lo + 1 < hi) {
      const mid = (lo + hi) >>> 1;
      if (starts[mid] <= offset) lo = mid;
      else hi = mid;
    }
    return lo;
  }

  /** Original Source UTF-16 absolute offset -> CodeMirror's normalized UTF-16. */
  toView(sourceOffset: number): number {
    if (!Number.isSafeInteger(sourceOffset) || sourceOffset < 0 ||
        sourceOffset > this.sourceLength) throw new Error("INVALID_SOURCE_OFFSET");
    const line = this.atOrBefore(this.sourceStarts, sourceOffset);
    const col = sourceOffset - this.sourceStarts[line];
    // Between the CR and LF in a CRLF is not a representable Source point.
    if (col > this.contentLengths[line]) throw new Error("SOURCE_OFFSET_INSIDE_CRLF");
    return this.viewStarts[line] + col;
  }

  /** CodeMirror's normalized UTF-16 absolute offset -> original Source UTF-16. */
  toSource(viewOffset: number): number {
    if (!Number.isSafeInteger(viewOffset) || viewOffset < 0 ||
        viewOffset > this.viewLength) throw new Error("INVALID_EDITOR_VIEW_OFFSET");
    const line = this.atOrBefore(this.viewStarts, viewOffset);
    const col = viewOffset - this.viewStarts[line];
    if (col > this.contentLengths[line]) throw new Error("EDITOR_OFFSET_INSIDE_LINE_BREAK");
    return this.sourceStarts[line] + col;
  }

  rangeToView(range: CoordinateRange): CoordinateRange {
    const from = this.toView(range.from);
    const to = this.toView(range.to);
    if (to < from) throw new Error("EDITOR_RANGE_REVERSED");
    return { from, to };
  }

  rangeToSource(range: CoordinateRange): CoordinateRange {
    const from = this.toSource(range.from);
    const to = this.toSource(range.to);
    if (to < from) throw new Error("EDITOR_RANGE_REVERSED");
    return { from, to };
  }
}
