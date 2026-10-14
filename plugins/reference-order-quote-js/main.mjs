// Independent, read-only JavaScript realization of the quote reference.
// This is a comparison specimen, not a proposed universal plugin ABI.
import { createHash } from 'node:crypto';

const fp = (text) => 'sha256:' + createHash('sha256').update(text).digest('hex');
const contract = 'reference.order.quote-lines/v1';
const implementation = 'order-quote.v1';
const pluginID = 'tei.reference.order_quote';
const carrierID = 'transport.order_quote_carrier';

// JSON.parse alone silently overwrites duplicate keys and rounds integers.
// Keep the business input lexical tokens until their shape is validated.
// All JSON member names are exact and are checked at each object depth.
function parseStrictJSON(source) {
  let offset = 0;
  const fail = () => { throw new Error('invalid business JSON'); };
  const whitespace = () => {
    while (/[ \t\r\n]/.test(source[offset] ?? 'X')) offset++;
  };
  const string = () => {
    if (source[offset++] !== '"') fail();
    const begin = offset - 1;
    let escaped = false;
    while (offset < source.length) {
      const ch = source[offset++];
      if (!escaped && ch === '"') {
        try { return JSON.parse(source.slice(begin, offset)); } catch { fail(); }
      }
      if (!escaped && ch === '\\') escaped = true;
      else escaped = false;
    }
    fail();
  };
  const value = (depth) => {
    if (depth > 64) fail();
    whitespace();
    const ch = source[offset];
    if (ch === '"') return string();
    if (ch === '{') {
      offset++;
      const fields = Object.create(null);
      const seen = new Set();
      whitespace();
      if (source[offset] === '}') { offset++; return fields; }
      while (offset < source.length) {
        whitespace();
        const key = string();
        if (seen.has(key)) fail();
        seen.add(key);
        whitespace();
        if (source[offset++] !== ':') fail();
        fields[key] = value(depth + 1);
        whitespace();
        if (source[offset] === '}') { offset++; return fields; }
        if (source[offset++] !== ',') fail();
      }
      fail();
    }
    if (ch === '[') {
      offset++;
      const elements = [];
      whitespace();
      if (source[offset] === ']') { offset++; return elements; }
      while (offset < source.length) {
        elements.push(value(depth + 1));
        whitespace();
        if (source[offset] === ']') { offset++; return elements; }
        if (source[offset++] !== ',') fail();
      }
      fail();
    }
    if (source.startsWith('null', offset)) { offset += 4; return null; }
    if (source.startsWith('true', offset)) { offset += 4; return true; }
    if (source.startsWith('false', offset)) { offset += 5; return false; }
    const match = source.slice(offset).match(/^-?(?:0|[1-9][0-9]*)(?:\.[0-9]+)?(?:[eE][+-]?[0-9]+)?/);
    if (!match) fail();
    offset += match[0].length;
    return /^-?(?:0|[1-9][0-9]*)$/.test(match[0]) ? BigInt(match[0]) : Number(match[0]);
  };
  const result = value(0);
  whitespace();
  if (offset !== source.length) fail();
  return result;
}

function exactFields(value, names) {
  if (value === null || typeof value !== 'object' || Array.isArray(value)) throw new Error('invalid fields');
  const found = Object.keys(value);
  if (found.length !== names.length || found.some((key) => !names.includes(key))) throw new Error('invalid fields');
  return value;
}

function calculate(rawBusiness) {
  const request = exactFields(parseStrictJSON(rawBusiness), ['currency', 'lines']);
  if (request.currency !== 'JPY' || !Array.isArray(request.lines) ||
      request.lines.length < 1 || request.lines.length > 50) throw new Error('invalid request');
  let total = 0n;
  for (const value of request.lines) {
    const line = exactFields(value, ['sku', 'quantity', 'unitPriceMinor']);
    // Go len(string) counts UTF-8 bytes, not JS UTF-16 code units.
    if (typeof line.sku !== 'string' || line.sku.trim() === '' ||
        Buffer.byteLength(line.sku, 'utf8') > 64 ||
        typeof line.quantity !== 'bigint' || line.quantity < 1n || line.quantity > 100n ||
        typeof line.unitPriceMinor !== 'bigint' || line.unitPriceMinor < 0n ||
        line.unitPriceMinor > 1000000000n) throw new Error('invalid line');
    total += line.quantity * line.unitPriceMinor;
    if (total > 1000000000000n) throw new Error('limit exceeded');
  }
  // The accepted bound is <=1e12, exactly representable as a JS Number.
  return { currency: 'JPY', lineCount: request.lines.length, subtotalMinor: Number(total) };
}

function base64Bytes(value) {
  if (typeof value !== 'string' || !/^(?:[A-Za-z0-9+/]{4})*(?:[A-Za-z0-9+/]{2}==|[A-Za-z0-9+/]{3}=)?$/.test(value)) {
    throw new Error('bad binary encoding');
  }
  const bytes = Buffer.from(value, 'base64');
  if (bytes.toString('base64') !== value) throw new Error('noncanonical binary encoding');
  return bytes;
}

function execute(request) {
  if (request?.protocolVersion !== 1 || !request.requestId ||
      request.plugin?.id !== pluginID || request.plugin?.version !== '1.0.0' ||
      request.capabilityKind !== 'contract-query-executor' || request.capabilityId !== carrierID ||
      !request.descriptorFingerprint || request.operation !== 'generic-dispatch-v0') {
    throw new Error('invalid carrier');
  }
  const response = {
    protocolVersion: request.protocolVersion,
    requestId: request.requestId,
    plugin: request.plugin,
    capabilityId: request.capabilityId,
    descriptorFingerprint: request.descriptorFingerprint,
    status: 'error'
  };
  try {
    const dispatch = request.payload;
    if (dispatch?.schema !== 'tei.reference.capability-transport-dispatch/v0' ||
        dispatch.contractId !== contract ||
        dispatch.contractFingerprint !== fp('reference.order.quote-lines.contract/v1') ||
        dispatch.requestShapeFingerprint !== fp('reference.order.quote-lines.request/v1') ||
        dispatch.resultShapeFingerprint !== fp('reference.order.quote-lines.result/v1') ||
        dispatch.target !== 'reference/order_entry' || dispatch.payloadCodec !== 'application/json' ||
        dispatch.provider?.pluginId !== pluginID || dispatch.provider?.pluginVersion !== '1.0.0' ||
        dispatch.provider?.implementationId !== implementation ||
        typeof dispatch.provider?.artifactDigest !== 'string' ||
        !dispatch.provider.artifactDigest.startsWith('sha256:')) throw new Error('invalid dispatch');
    const quote = calculate(base64Bytes(dispatch.payload).toString('utf8'));
    const result = {
      schema: 'tei.reference.capability-transport-result/v0',
      contractId: dispatch.contractId,
      contractFingerprint: dispatch.contractFingerprint,
      target: dispatch.target,
      provider: dispatch.provider,
      requestShapeFingerprint: dispatch.requestShapeFingerprint,
      resultShapeFingerprint: dispatch.resultShapeFingerprint,
      payloadCodec: dispatch.payloadCodec,
      payload: Buffer.from(JSON.stringify(quote)).toString('base64')
    };
    response.status = 'ok';
    response.payload = result;
  } catch {
    response.error = 'invalid or unsupported read-only quote';
  }
  return response;
}

try {
  const chunks = [];
  let size = 0;
  for await (const chunk of process.stdin) {
    size += chunk.length;
    if (size > (1 << 20)) throw new Error('oversized carrier');
    chunks.push(chunk);
  }
  const request = JSON.parse(Buffer.concat(chunks).toString('utf8'));
  process.stdout.write(JSON.stringify(execute(request)) + '\n');
} catch {
  // Never echo untrusted request, principal or credentials to stderr.
  process.stderr.write('invalid read-only quote carrier\n');
  process.exitCode = 2;
}
