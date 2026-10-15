// Reject ambiguous JSON object keys at the E1 Browser Host boundary.
// Python's existing verify_frozen_unit_pair.py rejects duplicates, but native
// JSON.parse silently accepts the final duplicate. Keep Host and builder-side
// diagnostic receipts consistent. No extra package/Policy/Authority is added.
//
// JSON.parse remains responsible for syntax/value validation; this scanner
// only detects repeated object keys, including escaped equivalents and nested
// objects, before JSON.parse can discard an earlier declaration.
export function parseStrictDiagnosticJSON(source) {
  if (typeof source !== "string") throw new TypeError("diagnostic JSON must be text");
  let offset = 0;
  const white = () => {
    while (offset < source.length && /[ \t\r\n]/.test(source[offset])) offset++;
  };
  const quoted = () => {
    const start = offset;
    if (source[offset++] !== '"') throw new SyntaxError("expected JSON string");
    while (offset < source.length) {
      const c = source[offset++];
      if (c === '"') return JSON.parse(source.slice(start, offset));
      if (c !== "\\") {
        if (c.charCodeAt(0) < 32) throw new SyntaxError("unescaped JSON control character");
        continue;
      }
      if (offset === source.length) break;
      const escape = source[offset++];
      if (escape === "u") {
        if (!/^[0-9a-fA-F]{4}$/.test(source.slice(offset, offset + 4))) {
          throw new SyntaxError("invalid JSON unicode escape");
        }
        offset += 4;
      } else if (!'"\\/bfnrt'.includes(escape)) {
        throw new SyntaxError("invalid JSON escape");
      }
    }
    throw new SyntaxError("unterminated JSON string");
  };
  const value = depth => {
    if (depth > 128) throw new SyntaxError("diagnostic JSON nesting exceeds limit");
    white();
    const c = source[offset];
    if (c === "{") {
      offset++;
      white();
      if (source[offset] === "}") { offset++; return; }
      const keys = new Set();
      while (true) {
        white();
        const key = quoted();
        if (keys.has(key)) throw new SyntaxError("duplicate diagnostic JSON key: " + key);
        keys.add(key);
        white();
        if (source[offset++] !== ":") throw new SyntaxError("expected JSON colon");
        value(depth + 1);
        white();
        const next = source[offset++];
        if (next === "}") return;
        if (next !== ",") throw new SyntaxError("expected JSON comma or brace");
      }
    }
    if (c === "[") {
      offset++;
      white();
      if (source[offset] === "]") { offset++; return; }
      while (true) {
        value(depth + 1);
        white();
        const next = source[offset++];
        if (next === "]") return;
        if (next !== ",") throw new SyntaxError("expected JSON comma or bracket");
      }
    }
    if (c === '"') { quoted(); return; }
    // JSON.parse validates the exact syntax of literals/numbers after the
    // scanner finds object boundaries, so do not implement a second grammar.
    const start = offset;
    while (offset < source.length && !/[,\]} \t\r\n]/.test(source[offset])) offset++;
    if (offset === start) throw new SyntaxError("expected diagnostic JSON value");
  };
  white();
  value(0);
  white();
  if (offset !== source.length) throw new SyntaxError("extra diagnostic JSON input");
  return JSON.parse(source);
}
