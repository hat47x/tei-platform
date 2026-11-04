import test from 'node:test';
import assert from 'node:assert/strict';
import { createHash } from 'node:crypto';
import { readFileSync } from 'node:fs';

const vectors = JSON.parse(readFileSync(new URL('./vectors.json', import.meta.url), 'utf8'));

function appendField(parts, bytes) {
  const size = Buffer.alloc(8);
  size.writeBigUInt64BE(BigInt(bytes.length));
  parts.push(size, bytes);
}

function frame(fields, input) {
  const parts = [];
  for (const field of fields) appendField(parts, Buffer.from(field, 'utf8'));
  appendField(parts, Buffer.from(input, 'utf8'));
  return Buffer.concat(parts);
}

function frameInput({ algorithm, mode, bits = 0, alphabet = '', length = 0 }) {
  const s = vectors.scheme;
  const fields = [
    'tei.hashid.frame.v1', s.schemeId, s.purpose, s.scope,
    s.inputKind, s.canonicalization, algorithm, mode, String(bits),
  ];
  if (mode === 'exactCharacters') fields.push(alphabet, String(length));
  return frame(fields, vectors.inputUtf8);
}

function shakeBytes(input, length) {
  return createHash('shake256', { outputLength: length }).update(input).digest();
}

function exactCharacters(input, alphabet, length) {
  const threshold = Math.floor(256 / alphabet.length) * alphabet.length;
  const source = shakeBytes(input, length * 16 + 1024);
  let text = '';
  for (const candidate of source) {
    if (candidate >= threshold) continue;
    text += alphabet[candidate % alphabet.length];
    if (text.length === length) break;
  }
  assert.equal(text.length, length, 'XOF sample budget exceeded');
  return text;
}

test('raw standard SHAKE256 empty-message vector is independent of TEI frame', () => {
  const actual = shakeBytes(Buffer.alloc(0), 32).toString('hex');
  assert.equal(actual, '46b9dd2b0ba88d13233b3feb743eeb243fcd52ea62b81b82b50c27646ed5762f');
});

test('Go SHA256 encodedDigest golden matches independent Node crypto', () => {
  const actual = createHash('sha256').update(frameInput({algorithm:'sha256', mode:'encodedDigest', bits:256})).digest('hex');
  assert.equal(actual, vectors.cases.sha256hex);
});

test('Go SHAKE256 160-bit encodedDigest golden matches independent Node crypto', () => {
  const actual = shakeBytes(frameInput({algorithm:'shake256', mode:'encodedDigest', bits:160}), 20).toString('hex');
  assert.equal(actual, vectors.cases.shake160hex);
});

test('Go SHAKE256 custom-alphabet exact-32 golden matches independent Node crypto', () => {
  const alphabet = '0123456789abcdefghjkmnpqrstvwxyz';
  const actual = exactCharacters(frameInput({algorithm:'shake256', mode:'exactCharacters', alphabet, length:32}), alphabet, 32);
  assert.equal(actual, vectors.cases.shakeExact32);
});

test('Go SHAKE256 decimal exact-30 with rejection matches independent Node crypto', () => {
  const alphabet = '0123456789';
  const actual = exactCharacters(frameInput({algorithm:'shake256', mode:'exactCharacters', alphabet, length:30}), alphabet, 30);
  assert.equal(actual, vectors.cases.shakeExactDecimal30);
});

test('scope, length and alphabet order are bound to different frames', () => {
  const basis = frameInput({algorithm:'shake256', mode:'exactCharacters', alphabet:'0123456789', length:30});
  assert.notDeepEqual(basis, frameInput({algorithm:'shake256', mode:'exactCharacters', alphabet:'0123456789', length:29}));
  assert.notDeepEqual(basis, frameInput({algorithm:'shake256', mode:'exactCharacters', alphabet:'9876543210', length:30}));
  assert.notDeepEqual(basis, frameInput({algorithm:'sha256', mode:'exactCharacters', alphabet:'0123456789', length:30}));
});
