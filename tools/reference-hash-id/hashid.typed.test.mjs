import test from 'node:test';
import assert from 'node:assert/strict';
import { createHash } from 'node:crypto';
import { readFileSync } from 'node:fs';
import { pathToFileURL } from 'node:url';

const tsModule = process.env.TEI_HASHID_TS_MODULE;
if (!tsModule) throw Error('TEI_HASHID_TS_MODULE must point to compiled hashid.mjs');
const { derive, frame, validate, serializeSchemeJSON, parseSchemeJSON } = await import(pathToFileURL(tsModule).href);
const fixture = JSON.parse(readFileSync(new URL('./vectors.json', import.meta.url), 'utf8'));
const base = {
  schemeId: fixture.scheme.schemeId,
  purpose: fixture.scheme.purpose,
  scope: fixture.scheme.scope,
  inputKind: fixture.scheme.inputKind,
  canonicalization: fixture.scheme.canonicalization,
};
const adapter = {
  digest(algorithm, input, byteLength) {
    if (algorithm === 'sha256') {
      const digest = createHash('sha256').update(input).digest();
      assert.equal(byteLength, digest.length);
      return digest;
    }
    if (algorithm === 'shake256') {
      return createHash('shake256', { outputLength: byteLength }).update(input).digest();
    }
    throw Error('cSHAKE256 provider is not part of this Node reference test');
  }
};

const source = new TextEncoder().encode(fixture.inputUtf8);

test('typed TS SHA-256 and SHAKE256 encodedDigest agree with shared Go vectors', () => {
  for (const [algorithm, outputBits, want] of [
    ['sha256', 256, fixture.cases.sha256hex],
    ['shake256', 160, fixture.cases.shake160hex],
  ]) {
    const scheme = {...base, algorithm, mode:'encodedDigest', outputBits, encoding:'base16lower', alphabet:'', length:0};
    assert.equal(derive(scheme, source, adapter).text, want);
  }
});

test('typed TS exactCharacters agrees with Go for base32 and decimal alphabets', () => {
  for (const [alphabet, length, want] of [
    ['0123456789abcdefghjkmnpqrstvwxyz', 32, fixture.cases.shakeExact32],
    ['0123456789', 30, fixture.cases.shakeExactDecimal30],
  ]) {
    const scheme = {...base, algorithm:'shake256', mode:'exactCharacters', outputBits:0, encoding:'', alphabet, length};
    const value = derive(scheme, source, adapter);
    assert.equal(value.text, want);
    assert.equal(value.digest, null);
  }
});

test('typed TS keeps the same digest under canonical display changes', () => {
  const scheme = {...base, algorithm:'sha256', mode:'encodedDigest', outputBits:256, encoding:'base16lower', alphabet:'', length:0};
  const first = derive(scheme, source, adapter);
  for (const [encoding, alphabet] of [
    ['base32lower', ''],
    ['base64url', ''],
    ['radix', '0123456789'],
    ['radix', '0123456789abcdefghijklmnopqrstuvwxyz'],
    ['radix', '0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz'],
  ]) {
    const changed = derive({...scheme, encoding, alphabet}, source, adapter);
    assert.deepEqual(changed.digest, first.digest);
    assert.deepEqual(frame({...scheme, encoding, alphabet}, source), frame(scheme, source));
  }
});

test('typed TS rejects unsupported canonicalization, duplicate and Unicode alphabets', () => {
  const scheme = {...base, algorithm:'shake256', mode:'exactCharacters', outputBits:0, encoding:'', alphabet:'0123456789', length:24};
  validate(scheme);
  assert.throws(() => validate({...scheme, alphabet:'001234'}));
  assert.throws(() => validate({...scheme, alphabet:'0β1'}));
  assert.throws(() => validate({...scheme, canonicalization:'json/v1'}));
  assert.throws(() => validate({...scheme, mode:'encodedDigest', outputBits:160, encoding:'base16lower', length:24}));
});

const wireFixture = JSON.parse(readFileSync(new URL('./scheme_wire_vectors.json', import.meta.url), 'utf8'));

test('typed TS scheme JSON has exactly the canonical Go wire bytes', () => {
  assert.equal(wireFixture.schemaVersion, 'tei.hashid.wire-v1.fixture');
  for (const [name, canonical] of Object.entries(wireFixture.schemes)) {
    const scheme = parseSchemeJSON(canonical);
    const remarshal = serializeSchemeJSON(scheme);
    assert.equal(remarshal, canonical, name);
    assert.deepEqual(parseSchemeJSON(remarshal), scheme);
  }
});

test('typed TS rejects noncanonical/ambiguous JSON scheme declarations', () => {
  const exact = wireFixture.schemes.exact32;
  const special = wireFixture.schemes.specialAscii;
  for (const invalid of [
    ' ' + exact,
    exact + '\n',
    exact + '{}',
    exact.replace('"format":', '"newKey":1,"format":'),
    exact.replace('"schemeId":', '"schemeId":"bad","schemeId":'),
    exact.replace('"outputBits":0,', ''),
    exact.replace('"alphabet":"0123456789abcdefghjkmnpqrstvwxyz"', '"alphabet":null'),
    exact.replace('"length":32', '"length":32.0'),
    exact.replace('tei.hashid.scheme/v1', 'tei.hashid.scheme/v2'),
    special.replace('<', String.fromCharCode(92) + 'u003c'),
    '[]',
  ]) {
    assert.throws(() => parseSchemeJSON(invalid), invalid);
  }
});

test('typed TS serialization changes for presentation but not for binary digest frame', () => {
  const a = {...base, algorithm:'shake256', mode:'encodedDigest',
    outputBits:160, encoding:'base16lower', alphabet:'', length:0};
  const b = {...a, encoding:'base32lower'};
  assert.notEqual(serializeSchemeJSON(a), serializeSchemeJSON(b));
  assert.deepEqual(frame(a, source), frame(b, source));
  assert.deepEqual(derive(a, source, adapter).digest, derive(b, source, adapter).digest);
});
