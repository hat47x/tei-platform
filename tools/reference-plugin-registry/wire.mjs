// Test-only fixture wire parser: reject duplicate/case-alias/unknown keys before
// verification, without declaring a production TEI Canonical JSON profile.
const expected = {
  fixture: ['releaseText','publisherSignatureBase64','profileText','artifacts','attestation'],
  artifact: ['role','bytesBase64'],
  attestation: ['subject','profileDigest','target','result','signatureBase64'],
};

function rejectDuplicateKeys(raw) {
  let offset = 0;
  const white = () => { while (/[\x20\t\r\n]/.test(raw[offset] ?? '\0')) offset++; };
  const quoted = () => {
    const start = offset++;
    let escaped = false;
    while (offset < raw.length) {
      const c = raw[offset++];
      if (escaped) { escaped = false; continue; }
      if (c === '\\') { escaped = true; continue; }
      if (c === '"') return JSON.parse(raw.slice(start,offset));
    }
    throw new Error('unterminated string');
  };
  const value = (depth) => {
    if (depth > 32) throw new Error('wire nesting too deep');
    white();
    if (raw[offset] === '{') {
      offset++; white();
      const seen = new Set();
      while (raw[offset] !== '}') {
        const key = quoted();
        if (seen.has(key)) throw new Error('duplicate wire key: ' + key);
        seen.add(key); white(); offset++; // colon; JSON.parse already checked grammar
        value(depth + 1); white();
        if (raw[offset] !== ',') break;
        offset++; white();
      }
      offset++; // }
    } else if (raw[offset] === '[') {
      offset++; white();
      while (raw[offset] !== ']') {
        value(depth + 1); white();
        if (raw[offset] !== ',') break;
        offset++; white();
      }
      offset++; // ]
    } else if (raw[offset] === '"') {
      quoted();
    } else {
      while (offset < raw.length && !/[\s,\]}]/.test(raw[offset])) offset++;
    }
  };
  value(0); white();
  if (offset !== raw.length) throw new Error('unexpected trailing wire data');
}

function exactFields(value, type) {
  if (value === null || typeof value !== 'object' || Array.isArray(value)) throw new Error('expected ' + type + ' object');
  const names = Object.keys(value);
  const fields = expected[type];
  if (names.length !== fields.length || names.some(n => !fields.includes(n))) throw new Error('missing/unknown ' + type + ' field');
  for (const key of fields) {
    if (key === 'artifacts' || key === 'attestation') continue;
    if (typeof value[key] !== 'string') throw new Error('expected string at ' + type + '.' + key);
  }
}

export function parseFixture(raw) {
  const value = typeof raw === 'string' ? raw : new TextDecoder('utf-8',{fatal:true}).decode(raw);
  if (Buffer.byteLength(value) > (1 << 20)) throw new Error('oversized fixture');
  const parsed = JSON.parse(value);
  rejectDuplicateKeys(value);
  exactFields(parsed, 'fixture');
  if (!Array.isArray(parsed.artifacts)) throw new Error('expected artifact array');
  parsed.artifacts.forEach(a => exactFields(a,'artifact'));
  exactFields(parsed.attestation, 'attestation');
  return parsed;
}
