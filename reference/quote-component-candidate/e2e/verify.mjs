// Independent Host verification against the existing unaccepted candidate
// corpus. Do not conflate a typed projection failure with a Guest rejection.
import { createServer } from 'node:http';
import { readFile, stat } from 'node:fs/promises';
import path from 'node:path';
import { pathToFileURL } from 'node:url';
import assert from 'node:assert/strict';
import { createRequire } from 'node:module';

const [mode, moduleFile, corpusFile, playwrightPath] = process.argv.slice(2);
if (!['node', 'browser'].includes(mode) || !moduleFile || !corpusFile) {
  throw Error('usage: node verify.mjs node|browser transpiled.js vectors.json [playwright-package]');
}

function parseExact(raw) {
  let pos = 0;
  const reject = () => { throw Error('bad raw JSON'); };
  const ws = () => { while (/[ \t\n\r]/.test(raw[pos] ?? 'X')) pos++; };
  function string() {
    if (raw[pos] !== '"') reject();
    const start = pos++;
    let escaped = false;
    while (pos < raw.length) {
      const ch = raw[pos++];
      if (ch === '"' && !escaped) {
        try { return JSON.parse(raw.slice(start, pos)); } catch { reject(); }
      }
      if (ch === '\\' && !escaped) escaped = true;
      else escaped = false;
    }
    reject();
  }
  function value(depth) {
    if (depth > 64) reject();
    ws();
    const ch = raw[pos];
    if (ch === '"') return string();
    if (ch === '{') {
      pos++;
      const obj = Object.create(null);
      ws();
      if (raw[pos] === '}') { pos++; return obj; }
      for (;;) {
        ws();
        const k = string();
        if (Object.hasOwn(obj, k)) reject();
        ws();
        if (raw[pos++] !== ':') reject();
        obj[k] = value(depth + 1);
        ws();
        if (raw[pos] === '}') { pos++; return obj; }
        if (raw[pos++] !== ',') reject();
      }
    }
    if (ch === '[') {
      pos++;
      const a = [];
      ws();
      if (raw[pos] === ']') { pos++; return a; }
      for (;;) {
        a.push(value(depth + 1));
        ws();
        if (raw[pos] === ']') { pos++; return a; }
        if (raw[pos++] !== ',') reject();
      }
    }
    for (const [literal, v] of [['true', true], ['false', false], ['null', null]]) {
      if (raw.startsWith(literal, pos)) { pos += literal.length; return v; }
    }
    const match = raw.slice(pos).match(/^-?(?:0|[1-9][0-9]*)(?:\.[0-9]+)?(?:[eE][+-]?[0-9]+)?/);
    if (!match) reject();
    pos += match[0].length;
    return /^-?(?:0|[1-9][0-9]*)$/.test(match[0]) ? BigInt(match[0]) : Number(match[0]);
  }
  const result = value(0);
  ws();
  if (pos !== raw.length) reject();
  return result;
}
function exactFields(obj, keys) {
  if (obj === null || typeof obj !== 'object' || Array.isArray(obj)) throw Error('bad object');
  const present = Object.keys(obj);
  if (present.length !== keys.length || present.some(k => !keys.includes(k))) throw Error('bad keys');
}
function typedProjection(raw) {
  const request = parseExact(raw);
  exactFields(request, ['currency', 'lines']);
  if (typeof request.currency !== 'string' || !Array.isArray(request.lines)) throw Error('bad shape');
  const lines = request.lines.map(line => {
    exactFields(line, ['sku', 'quantity', 'unitPriceMinor']);
    if (typeof line.sku !== 'string' ||
        typeof line.quantity !== 'bigint' || typeof line.unitPriceMinor !== 'bigint' ||
        line.quantity < 0n || line.quantity > 18446744073709551615n ||
        line.unitPriceMinor < 0n || line.unitPriceMinor > 18446744073709551615n) {
      throw Error('unrepresentable in WIT');
    }
    return { sku: line.sku, quantity: line.quantity, unitPriceMinor: line.unitPriceMinor };
  });
  return { currency: request.currency, lines };
}
function isBusinessRejection(err) {
  return err === 'rejected' || err?.payload === 'rejected';
}
function normalizeResult(value) {
  return {
    currency: value.currency,
    lineCount: value.lineCount,
    subtotalMinor: String(value.subtotalMinor),
  };
}

const corpus = JSON.parse(await readFile(corpusFile, 'utf8'));
assert.equal(corpus.schema, 'tei.reference.quote-semantic-vectors/candidate-v0');
assert.equal(corpus.status, 'unaccepted');
assert.equal(corpus.contract, 'reference.order.quote-lines/v1');
assert.equal(corpus.vectors.length, 23);

const cases = corpus.vectors.map(v => {
  assert.equal(typeof v.name, 'string');
  assert.equal(v.reject === true, !v.expect);
  try { return { vector: v, request: typedProjection(v.request) }; }
  catch { return { vector: v, ingressRejected: true }; }
});
const totals = { accepted: 0, ingressRejected: 0, componentRejected: 0 };
let browser = null;
let server = null;
let run;

try {
  if (mode === 'node') {
    const { quotePreview } = await import(pathToFileURL(path.resolve(moduleFile)).href);
    assert.equal(typeof quotePreview?.preview, 'function', 'missing transpiled Component export');
    run = async req => {
      try { return { kind: 'ok', value: normalizeResult(quotePreview.preview(req)) }; }
      catch (err) {
        if (isBusinessRejection(err)) return { kind: 'business-reject' };
        throw err; // Host failure is never a business rejection.
      }
    };
  } else {
    if (!playwrightPath || !process.env.TEI_BROWSER_EXECUTABLE) {
      throw Error('real Chromium and playwright-core are mandatory for browser evidence');
    }
    const require = createRequire(import.meta.url);
    const { chromium } = require(path.resolve(playwrightPath));
    const site = path.resolve(path.dirname(moduleFile));
    const entry = path.basename(moduleFile);
    server = createServer(async (req, res) => {
      try {
        const pathname = decodeURIComponent(new URL(req.url, 'http://localhost').pathname);
        const file = path.resolve(site, '.' + pathname);
        if (!(file === site || file.startsWith(site + path.sep))) throw Error('bad path');
        const fileStat = await stat(file);
        if (!fileStat.isFile()) throw Error('not a file');
        res.setHeader('Content-Type', file.endsWith('.wasm') ? 'application/wasm' : file.endsWith('.js') ? 'text/javascript' : 'application/octet-stream');
        res.end(await readFile(file));
      } catch {
        res.writeHead(404);
        res.end('not found');
      }
    });
    await new Promise(resolve => server.listen(0, '127.0.0.1', resolve));
    browser = await chromium.launch({ executablePath: process.env.TEI_BROWSER_EXECUTABLE, headless: true, args: ['--no-sandbox'] });
    const page = await browser.newPage();
    page.on('pageerror', err => { throw err; });
    await page.goto('http://127.0.0.1:' + server.address().port + '/');
    run = async req => page.evaluate(async ({ entry, request }) => {
      const { quotePreview } = await import('/' + entry);
      if (typeof quotePreview?.preview !== 'function') throw Error('missing browser Component export');
      const typed = {
        currency: request.currency,
        lines: request.lines.map(l => ({ sku: l.sku, quantity: BigInt(l.quantity), unitPriceMinor: BigInt(l.unitPriceMinor) })),
      };
      try {
        const v = quotePreview.preview(typed);
        return { kind: 'ok', value: { currency: v.currency, lineCount: v.lineCount, subtotalMinor: String(v.subtotalMinor) } };
      } catch (err) {
        if (err === 'rejected' || err?.payload === 'rejected') return { kind: 'business-reject' };
        throw err;
      }
    }, { entry, request: { currency: req.currency, lines: req.lines.map(l => ({
      sku: l.sku, quantity: String(l.quantity), unitPriceMinor: String(l.unitPriceMinor),
    })) } });
  }

  for (const { vector, request, ingressRejected } of cases) {
    if (ingressRejected) {
      assert.equal(vector.reject, true, 'valid business input was rejected at WIT projection: ' + vector.name);
      totals.ingressRejected++;
      continue;
    }
    const outcome = await run(request);
    if (vector.reject) {
      assert.equal(outcome.kind, 'business-reject', mode + ': Guest accepted bad ' + vector.name);
      totals.componentRejected++;
    } else {
      assert.equal(outcome.kind, 'ok', mode + ': Guest rejected good ' + vector.name);
      assert.deepEqual(outcome.value, {
        currency: vector.expect.currency,
        lineCount: vector.expect.lineCount,
        subtotalMinor: String(vector.expect.subtotalMinor),
      }, mode + ': incorrect result for ' + vector.name);
      totals.accepted++;
    }
  }
  assert.equal(totals.accepted, 8);
  assert.equal(totals.ingressRejected + totals.componentRejected, 15);
  assert.ok(totals.componentRejected >= 2, 'must exercise real Guest business rejection');
  assert.ok(totals.ingressRejected >= 2, 'must preserve lexical ingress failures');
  console.log(JSON.stringify({ host: mode, ...totals, status: 'PASS', componentExecuted: true }));
} finally {
  if (browser) await browser.close();
  if (server) await new Promise(resolve => server.close(resolve));
}
