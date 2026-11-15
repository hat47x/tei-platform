// E1-C reference only: an independent Node process obtains the signed
// test-only release envelope over loopback HTTP, validates the exact E0
// signature/subject/profile/role bindings, then separately GETs each member.
// This is neither a product Registry client nor a current Trust authority.
import http from 'node:http';
import { createHash } from 'node:crypto';
import { parseFixture } from './wire.mjs';
import { defaultPolicy, validate } from './verify.mjs';

const MAX_BYTES = 1 << 20;
const sha256 = bytes => 'sha256:' + createHash('sha256').update(bytes).digest('hex');
const PINNED_SUBJECT = 'sha256:b486c89cb4663762e1ac0d4d2d4391f9fd74b88d716526c8d10c760a5687e4c7';

function result(status, cause = '') {
  process.stdout.write('E1-SIGNED-C status=' + status + '\n');
  if (cause) process.stderr.write(status + ': ' + cause + '\n');
  process.exitCode = status === 'complete' ? 0 : 1;
}

async function acquire(origin, pathname) {
  const uri = new URL(pathname, origin);
  if (uri.origin !== origin.origin || uri.username || uri.password) {
    throw new Error('origin-mismatch');
  }
  const response = await new Promise((resolve, reject) => {
    const req = http.get(uri, { agent: false, timeout: 5000 }, resolve);
    req.on('timeout', () => req.destroy(new Error('unavailable')));
    req.on('error', reject);
  });
  if (response.statusCode >= 300 && response.statusCode < 400) {
    response.destroy();
    throw new Error('redirect-blocked');
  }
  if (response.statusCode === 404) {
    response.destroy();
    throw new Error('missing');
  }
  if (response.statusCode !== 200) {
    response.destroy();
    throw new Error('unavailable');
  }
  const chunks = [];
  let count = 0;
  for await (const chunk of response) {
    count += chunk.length;
    if (count > MAX_BYTES) {
      response.destroy();
      throw new Error('too-large');
    }
    chunks.push(chunk);
  }
  return Buffer.concat(chunks);
}

async function main() {
  const input = process.argv[2];
  let origin;
  try {
    origin = new URL(input);
  } catch {
    throw new Error('invalid-origin');
  }
  // Bound this fixture to a literal local address; no DNS, redirect, proxy,
  // credentials, query, other scheme, or external network.
  if (origin.protocol !== 'http:' || origin.hostname !== '127.0.0.1' ||
      !origin.port || origin.pathname !== '/' || origin.search ||
      origin.hash || origin.username || origin.password) {
    throw new Error('invalid-origin');
  }

  const envelope = await acquire(origin, '/reference/signed-release.json');
  let fixture;
  try {
    fixture = parseFixture(envelope);
  } catch {
    throw new Error('invalid-wire');
  }
  try {
    validate(fixture, defaultPolicy());
  } catch {
    throw new Error('invalid-signed-subject');
  }
  if (sha256(Buffer.from(fixture.releaseText, 'utf8')) !== PINNED_SUBJECT) {
    throw new Error('unrecognized-subject');
  }

  const lines = fixture.releaseText.split('\n');
  const mandatory = new Map([
    ['core', lines[4]?.slice('artifact.core='.length)],
    ['ui', lines[5]?.slice('artifact.ui='.length)],
  ]);
  if (lines.length !== 7 || !lines[4]?.startsWith('artifact.core=') ||
      !lines[5]?.startsWith('artifact.ui=') ||
      ![...mandatory.values()].every(x => /^sha256:[a-f0-9]{64}$/.test(x))) {
    throw new Error('invalid-member-binding');
  }
  for (const [role, digest] of mandatory) {
    // Each member is acquired by a separate HTTP request after successful
    // verification of the untrusted envelope, never from envelope.inline bytes.
    const bytes = await acquire(origin, '/sha256/' + digest.slice(7));
    if (sha256(bytes) !== digest) {
      throw new Error('digest-mismatch');
    }
    // Require exactly the signed member Role/Digest; no re-binding by name,
    // HTTP ETag, cache headers, or another release's manifest.
    if (!fixture.artifacts.some(a => a.role === role &&
        sha256(Buffer.from(a.bytesBase64, 'base64')) === digest)) {
      throw new Error('role-mismatch');
    }
  }
  process.stdout.write('E1-SIGNED-C pid=' + process.pid + ' subject=' + PINNED_SUBJECT +
                       ' count=' + mandatory.size + '\n');
  result('complete');
}

main().catch(error => result(error.message || 'unavailable', String(error)));
