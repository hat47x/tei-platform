// Standalone TypeScript reference: no Node.js, browser, Studio, or Host dependency.
// An adapter supplies SHA-256/SHAKE256/cSHAKE256, and is responsible for the
// fixed cSHAKE256 N="" and S="tei.hashid.cshake256.v1" customization.
export type Algorithm = 'sha256' | 'shake256' | 'cshake256';
export type Mode = 'encodedDigest' | 'exactCharacters';
export type Encoding = 'base16lower' | 'base32lower' | 'base64url' | 'radix';

export interface Scheme {
  schemeId: string;
  purpose: string;
  scope: string;
  inputKind: 'bytes';
  canonicalization: 'raw-bytes/v1';
  algorithm: Algorithm;
  mode: Mode;
  outputBits: number;
  encoding: Encoding | '';
  alphabet: string;
  length: number;
}

export interface CryptoAdapter {
  // A real adapter must supply cryptographic hashes, never a mock or PRNG.
  // The XOF output must be prefix-consistent for all requested byte lengths.
  digest(algorithm: Algorithm, input: Uint8Array, byteLength: number): Uint8Array;
}

export interface DerivedID {
  scheme: Scheme;
  text: string;
  digest: Uint8Array | null;
}

function token(s: string): boolean {
  return s.length > 0 && s.length <= 256 && /^[a-zA-Z0-9._:/-]+$/.test(s);
}

function validAlphabet(a: string): boolean {
  if (a.length < 2 || a.length > 94) return false;
  const used = new Set<string>();
  for (const c of a) {
    if (c.charCodeAt(0) < 33 || c.charCodeAt(0) > 126 || used.has(c)) return false;
    used.add(c);
  }
  return true;
}

export function validate(s: Scheme): void {
  if (![s.schemeId, s.purpose, s.scope].every(token)) throw Error('invalid scheme/purpose/scope');
  if (s.inputKind !== 'bytes' || s.canonicalization !== 'raw-bytes/v1') throw Error('unsupported input canonicalization');
  if (!['sha256', 'shake256', 'cshake256'].includes(s.algorithm)) throw Error('unsupported hash algorithm');
  if (s.mode === 'encodedDigest') {
    if (s.length !== 0 || !Number.isInteger(s.outputBits) ||
        s.outputBits < 8 || s.outputBits > 4096 || s.outputBits % 8 !== 0) throw Error('invalid digest output bits');
    if (s.algorithm === 'sha256' && s.outputBits !== 256) throw Error('SHA256 length fixed');
    if (s.encoding === 'radix') {
      if (!validAlphabet(s.alphabet)) throw Error('invalid radix alphabet');
    } else if (!['base16lower', 'base32lower', 'base64url'].includes(s.encoding) ||
               s.alphabet !== '') throw Error('invalid encoding');
  } else if (s.mode === 'exactCharacters') {
    if (s.algorithm === 'sha256' || s.outputBits !== 0 || s.encoding !== '' ||
        !Number.isInteger(s.length) || s.length < 1 || s.length > 1024 ||
        !validAlphabet(s.alphabet)) throw Error('invalid exact character profile');
  } else throw Error('unsupported hash mode');
}

const encoder = new TextEncoder();

function field(buf: Uint8Array): Uint8Array {
  const out = new Uint8Array(8 + buf.length);
  new DataView(out.buffer).setBigUint64(0, BigInt(buf.length), false);
  out.set(buf, 8);
  return out;
}

export function frame(s: Scheme, source: Uint8Array): Uint8Array {
  validate(s);
  const fields = [
    'tei.hashid.frame.v1', s.schemeId, s.purpose, s.scope,
    s.inputKind, s.canonicalization, s.algorithm, s.mode, String(s.outputBits),
  ];
  if (s.mode === 'exactCharacters') fields.push(s.alphabet, String(s.length));
  const parts = fields.map(value => field(encoder.encode(value)));
  parts.push(field(source));
  const length = parts.reduce((n, part) => n + part.length, 0);
  const out = new Uint8Array(length);
  let offset = 0;
  for (const part of parts) { out.set(part, offset); offset += part.length; }
  return out;
}

function encodeRadix(digest: Uint8Array, alphabet: string): string {
  const radix = BigInt(alphabet.length);
  let upper = (1n << BigInt(8 * digest.length)) - 1n;
  let width = 0;
  while (upper > 0n) { upper /= radix; width++; }
  let value = digest.reduce((acc, octet) => acc * 256n + BigInt(octet), 0n);
  const out = Array<string>(width).fill(alphabet[0]);
  for (let i = width - 1; i >= 0 && value > 0n; i--) {
    out[i] = alphabet[Number(value % radix)];
    value /= radix;
  }
  return out.join('');
}

function encodeDigest(digest: Uint8Array, s: Scheme): string {
  if (s.encoding === 'radix') return encodeRadix(digest, s.alphabet);
  if (s.encoding === 'base16lower') {
    return Array.from(digest, b => b.toString(16).padStart(2, '0')).join('');
  }
  if (s.encoding === 'base32lower') {
    const alphabet = 'abcdefghijklmnopqrstuvwxyz234567';
    let bits = 0, remaining = 0, out = '';
    for (const octet of digest) {
      remaining = (remaining << 8) | octet;
      bits += 8;
      while (bits >= 5) {
        bits -= 5;
        out += alphabet[(remaining >>> bits) & 31];
        remaining &= (1 << bits) - 1;
      }
    }
    if (bits) out += alphabet[(remaining << (5 - bits)) & 31];
    return out;
  }
  if (s.encoding === 'base64url') {
    const alphabet = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_';
    let out = '';
    for (let i = 0; i < digest.length; i += 3) {
      const a = digest[i], b = digest[i + 1] ?? 0, c = digest[i + 2] ?? 0;
      out += alphabet[a >>> 2] + alphabet[((a & 3) << 4) | (b >>> 4)];
      if (i + 1 < digest.length) out += alphabet[((b & 15) << 2) | (c >>> 6)];
      if (i + 2 < digest.length) out += alphabet[c & 63];
    }
    return out;
  }
  throw Error('unsupported encoding');
}

export function derive(s: Scheme, source: Uint8Array, adapter: CryptoAdapter): DerivedID {
  const message = frame(s, source);
  if (s.mode === 'encodedDigest') {
    const digest = adapter.digest(s.algorithm, message, s.outputBits / 8);
    if (digest.length !== s.outputBits / 8) throw Error('provider returned incorrect digest length');
    return { scheme: s, text: encodeDigest(digest, s), digest };
  }
  const cap = 16 * s.length + 1024;
  const samples = adapter.digest(s.algorithm, message, cap);
  if (samples.length !== cap) throw Error('provider returned incorrect XOF length');
  const size = s.alphabet.length;
  const threshold = Math.floor(256 / size) * size;
  let text = '';
  for (const candidate of samples) {
    if (candidate >= threshold) continue;
    text += s.alphabet[candidate % size];
    if (text.length === s.length) break;
  }
  if (text.length !== s.length) throw Error('rejection sampling budget exceeded');
  return { scheme: s, text, digest: null };
}

// v1 canonical JSON includes every field in this fixed order, including
// zero/empty values. It is intentionally NOT an authority/identity proof.
// An encoding-only change modifies this wire, not encodedDigest hash bytes.
export const SCHEME_WIRE_FORMAT = 'tei.hashid.scheme/v1';

export function serializeSchemeJSON(s: Scheme): string {
  validate(s);
  return JSON.stringify({
    format: SCHEME_WIRE_FORMAT,
    scheme: {
      schemeId: s.schemeId,
      purpose: s.purpose,
      scope: s.scope,
      inputKind: s.inputKind,
      canonicalization: s.canonicalization,
      algorithm: s.algorithm,
      mode: s.mode,
      outputBits: s.outputBits,
      encoding: s.encoding,
      alphabet: s.alphabet,
      length: s.length,
    },
  });
}

// Canonical equality rejects duplicate/unknown fields, missing zeros, nulls,
// different key orders, whitespace and noncanonical JSON escapes. Parse must
// never silently reinterpret a saved Scheme from a different language/version.
export function parseSchemeJSON(raw: string): Scheme {
  if (typeof raw !== 'string' || raw.length === 0 || raw.length > 4096) {
    throw Error('invalid scheme JSON length');
  }
  const input: unknown = JSON.parse(raw);
  if (input === null || typeof input !== 'object' || Array.isArray(input)) {
    throw Error('scheme envelope must be an object');
  }
  const envelope = input as { format?: unknown; scheme?: unknown };
  if (envelope.format !== SCHEME_WIRE_FORMAT || envelope.scheme === null ||
      typeof envelope.scheme !== 'object' || Array.isArray(envelope.scheme)) {
    throw Error('unsupported scheme envelope');
  }
  const s = envelope.scheme as Record<string, unknown>;
  for (const key of ['schemeId', 'purpose', 'scope', 'inputKind',
    'canonicalization', 'algorithm', 'mode', 'encoding', 'alphabet']) {
    if (typeof s[key] !== 'string') throw Error('scheme string field is missing');
  }
  for (const key of ['outputBits', 'length']) {
    if (typeof s[key] !== 'number') throw Error('scheme integer field is missing');
  }
  const checked = s as unknown as Scheme;
  const canonical = serializeSchemeJSON(checked);
  if (canonical !== raw) throw Error('noncanonical scheme JSON');
  return checked;
}
