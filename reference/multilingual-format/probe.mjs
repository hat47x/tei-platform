// E2a browser/Node comparison probe. This is not an MF2 parser.
const supported = new Set(['en-US', 'ja-JP', 'de-DE']);
export const MAX_COUNT = 2147483647;

// Inputs are already validated semantic values; no implicit locale fallback.
export function probe(locale, count) {
  if (!supported.has(locale)) {
    throw new Error(`unsupported-locale: ${String(locale)}`);
  }
  if (!Number.isSafeInteger(count) || count < 0 || count > MAX_COUNT) {
    throw new Error(`invalid-count: ${String(count)}`);
  }
  return {
    category: new Intl.PluralRules(locale, { type: 'cardinal' }).select(count),
    number: new Intl.NumberFormat(locale, {
      style: 'decimal',
      useGrouping: true,
      maximumFractionDigits: 0
    }).format(count)
  };
}
