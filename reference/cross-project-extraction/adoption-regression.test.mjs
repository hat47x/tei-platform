import test from 'node:test';
import assert from 'node:assert/strict';
import { createHash } from 'node:crypto';
import { mkdtemp, readFile, rm, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { orderChecksBefore, orderChecksAfter, orderNeedsApproval } from './projects/order.mjs';
import { invoiceChecksBefore, invoiceChecksAfter, invoiceNeedsApproval } from './projects/invoice.mjs';
import { CONTRACT } from './shared/required-fields.mjs';

// This is a consumer-side read-only probe. It does NOT resolve production TEI plugins.
const ARTIFACT = Object.freeze({
  id: 'tei.reference.required-fields',
  version: 'candidate-1',
  contract: 'required-text+nonnegative-safe-integer/v1',
  gitBlob: 'd26b35b98de88f73bea3530081f41b215aa7ac74',
});

const projects = Object.freeze({
  order: Object.freeze({ before: orderChecksBefore, after: orderChecksAfter, expected: {
    format: 'tei.reference.validation-plan/v0',
    rules: [{ op: 'required-text', field: 'orderId' }, { op: 'nonnegative-safe-integer', field: 'totalYen' }],
  } }),
  invoice: Object.freeze({ before: invoiceChecksBefore, after: invoiceChecksAfter, expected: {
    format: 'tei.reference.validation-plan/v0',
    rules: [{ op: 'required-text', field: 'invoiceId' }, { op: 'nonnegative-safe-integer', field: 'amountYen' }],
  } }),
});

function gitBlobSha(bytes) {
  return createHash('sha1').update('blob ' + bytes.length + '\0').update(bytes).digest('hex');
}

async function assertPinnedArtifact(pin, source = new URL('./shared/required-fields.mjs', import.meta.url)) {
  if (!pin || pin.id !== ARTIFACT.id || pin.version !== ARTIFACT.version ||
      pin.contract !== ARTIFACT.contract || pin.gitBlob !== ARTIFACT.gitBlob) {
    throw new Error('PIN_NOT_APPROVED');
  }
  if (gitBlobSha(await readFile(source)) !== pin.gitBlob) {
    throw new Error('ARTIFACT_DRIFT');
  }
  if (CONTRACT !== pin.contract) throw new Error('CONTRACT_MISMATCH');
}

async function preview(projectId, selection, artifactSource) {
  const consumer = projects[projectId];
  if (!consumer) throw new Error('UNKNOWN_PROJECT');
  if (selection?.mode === 'local' && Object.keys(selection).length === 1) {
    return consumer.before();
  }
  if (selection?.mode === 'shared' && Object.keys(selection).length === 2) {
    await assertPinnedArtifact(selection.pin, artifactSource);
    return consumer.after();
  }
  throw new Error('UNSUPPORTED_SELECTION');
}

const local = Object.freeze({ mode: 'local' });
const shared = Object.freeze({ mode: 'shared', pin: ARTIFACT });

// The two consumers own independent selections. Nothing in preview writes files or changes adoption state.
test('the shared candidate has an immutable, checked Git blob identity', async () => {
  await assertPinnedArtifact(ARTIFACT);
  assert.equal(CONTRACT, ARTIFACT.contract);
});

test('order stays local while invoice explicitly adopts; invoice can roll back', async () => {
  const accepted = Object.freeze({ order: local, invoice: local });
  const invoiceOnly = Object.freeze({ ...accepted, invoice: shared });
  assert.deepStrictEqual(await preview('order', invoiceOnly.order), projects.order.expected);
  assert.deepStrictEqual(await preview('invoice', invoiceOnly.invoice), projects.invoice.expected);
  assert.deepStrictEqual(await preview('invoice', accepted.invoice), projects.invoice.expected);
  assert.equal(accepted.invoice.mode, 'local');
  assert.equal(invoiceOnly.order.mode, 'local');
});

test('order adopts independently while invoice stays local; order can roll back', async () => {
  const accepted = Object.freeze({ order: local, invoice: local });
  const orderOnly = Object.freeze({ ...accepted, order: shared });
  assert.deepStrictEqual(await preview('order', orderOnly.order), projects.order.expected);
  assert.deepStrictEqual(await preview('invoice', orderOnly.invoice), projects.invoice.expected);
  assert.deepStrictEqual(await preview('order', accepted.order), projects.order.expected);
  assert.equal(orderOnly.invoice.mode, 'local');
});

test('unapproved version, contract, or artifact identity must not be used', async () => {
  for (const changed of [
    { version: 'candidate-2' },
    { contract: 'nullable-amount/v1' },
    { gitBlob: '0'.repeat(40) },
    { id: 'another.provider' },
  ]) {
    await assert.rejects(() => preview('order', { mode: 'shared', pin: { ...ARTIFACT, ...changed } }), /PIN_NOT_APPROVED/);
  }
});

test('artifact drift is rejected even when the logical id and version remain unchanged', async () => {
  const data = await readFile(new URL('./shared/required-fields.mjs', import.meta.url));
  const mutated = Buffer.concat([data, Buffer.from('//unexpected modification\n')]);
  const dir = await mkdtemp(join(tmpdir(), 'tei-promotion-adoption-'));
  try {
    const alteredPath = join(dir, 'plugin.mjs');
    await writeFile(alteredPath, mutated);
    assert.notEqual(gitBlobSha(mutated), ARTIFACT.gitBlob);
    await assert.rejects(() => assertPinnedArtifact(ARTIFACT, alteredPath), /ARTIFACT_DRIFT/);
  } finally {
    await rm(dir, { recursive: true, force: true });
  }
});

test('unresolved consumer or implicit fallback never authorizes a choice', async () => {
  await assert.rejects(() => preview('unknown', shared), /UNKNOWN_PROJECT/);
  await assert.rejects(() => preview('order', { mode: 'shared' }), /UNSUPPORTED_SELECTION/);
  await assert.rejects(() => preview('order', { mode: 'newest' }), /UNSUPPORTED_SELECTION/);
  await assert.rejects(() => preview('invoice', { mode: 'local', pin: ARTIFACT }), /UNSUPPORTED_SELECTION/);
});

test('approval remains project-owned after any selection', async () => {
  assert.equal(orderNeedsApproval({totalYen: 100_000}), true);
  assert.equal(orderNeedsApproval({totalYen: 99_999}), false);
  assert.equal(invoiceNeedsApproval({overseas: true, amountYen: 50_000}), true);
  assert.equal(invoiceNeedsApproval({overseas: false, amountYen: 50_000}), false);
});
