// Experimental business implementation of quote-preview-pure. No Node imports,
// external services, filesystem, timers, randomness, or principal access.
// This is an independent WIT Guest, NOT a trusted plugin or production ABI.
const rejection = () => { throw 'rejected'; };
function validSku(sku) {
  if (typeof sku !== 'string' || sku.trim() === '') return false;
  return new TextEncoder().encode(sku).byteLength <= 64;
}
export const quotePreview = {
  preview(request) {
    if (request?.currency !== 'JPY' || !Array.isArray(request?.lines) ||
        request.lines.length === 0 || request.lines.length > 50) rejection();
    let sum = 0n;
    for (const line of request.lines) {
      if (!validSku(line.sku) ||
          typeof line.quantity !== 'bigint' || line.quantity < 1n || line.quantity > 100n ||
          typeof line.unitPriceMinor !== 'bigint' || line.unitPriceMinor < 0n ||
          line.unitPriceMinor > 1000000000n) rejection();
      sum += line.quantity * line.unitPriceMinor;
      if (sum > 1000000000000n) rejection();
    }
    return { currency: 'JPY', lineCount: request.lines.length, subtotalMinor: sum };
  },
};
