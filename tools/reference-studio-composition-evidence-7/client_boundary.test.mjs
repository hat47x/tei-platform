// #7 independently exercises exact #1832 client; not a Host authorization proof.
import test from 'node:test';
import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import { createHash } from 'node:crypto';
import { fileURLToPath } from 'node:url';
import { createReadOnlyClient } from '../../cmd/tei-studio/assets/client.mjs';

const hostClientPath = fileURLToPath(new URL('../../cmd/tei-studio/assets/client.mjs', import.meta.url));
function gitBlob(bytes) {
  return createHash('sha1').update(Buffer.from('blob ' + bytes.length + '\0')).update(bytes).digest('hex');
}

test('actual client matches #1832 exact revision', async () => {
  assert.equal(gitBlob(await readFile(hostClientPath)), '1afec5762a96bf34eead768a9fc7de4b7eced564');
});

test('only four read-only Host RPC methods are exposed', async () => {
  const calls = [];
  const client = createReadOnlyClient(async (...args) => {
    calls.push(args);
    return { ok: true, value: { effect: 'read-only' } };
  });
  assert.deepEqual(Object.keys(client).sort(), [
    'catalogWorkspace', 'inspectSubject', 'listWorkspaces', 'validateWorkspace',
  ]);
  assert.ok(Object.isFrozen(client));
  assert.equal(client.saveWorkspace, undefined);
  assert.equal(client.acceptDraft, undefined);
  assert.equal(client.setRoot, undefined);
  await client.listWorkspaces();
  await client.catalogWorkspace('workspace-1', 'namespace');
  await client.validateWorkspace('workspace-1');
  await client.inspectSubject('workspace-1', 'semantic-data', 'sales.order');
  assert.deepEqual(calls, [
    ['main.StudioBridge.ListWorkspaces'],
    ['main.StudioBridge.CatalogWorkspace', 'workspace-1', 'namespace'],
    ['main.StudioBridge.ValidateWorkspace', 'workspace-1'],
    ['main.StudioBridge.InspectSubject', 'workspace-1', 'semantic-data', 'sales.order'],
  ]);
});

test('Host-denied workspace is never promoted into success', async () => {
  const client = createReadOnlyClient(async (name, workspaceID) => {
    assert.ok(name.startsWith('main.StudioBridge.'));
    assert.equal(workspaceID, '../outside');
    return { ok: false, failure: { code: 'WORKSPACE_NOT_ALLOWED', message: 'not allowed' } };
  });
  for (const call of [
    () => client.catalogWorkspace('../outside'),
    () => client.validateWorkspace('../outside'),
    () => client.inspectSubject('../outside', 'semantic-data', 'sales.order'),
  ]) {
    const reply = await call();
    assert.equal(reply.ok, false);
    assert.equal(reply.failure.code, 'WORKSPACE_NOT_ALLOWED');
    assert.equal(Object.hasOwn(reply, 'value'), false);
  }
});

test('successful operation diagnostics remain data, not denial', async () => {
  const value = { Direct: null, Diagnostics: [
    { diagnostic: { severity: 'warning', code: 'X', message: 'warning' } },
  ] };
  const client = createReadOnlyClient(async () => ({ ok: true, value }));
  const result = await client.validateWorkspace('workspace-1');
  assert.equal(result.ok, true);
  assert.strictEqual(result.value, value);
  assert.equal(result.value.Diagnostics.length, 1);
});

test('invalid Host envelope fails closed', async () => {
  for (const invalid of [
    null, undefined, 0, 'string', {}, { ok: true },
    { ok: 'true', value: {} }, { ok: false }, { ok: false, failure: {} },
  ]) {
    const client = createReadOnlyClient(async () => invalid);
    for (const result of [
      await client.listWorkspaces(),
      await client.catalogWorkspace('workspace-1'),
      await client.validateWorkspace('workspace-1'),
      await client.inspectSubject('workspace-1', 'x', 'y'),
    ]) {
      assert.equal(result.ok, false);
      assert.equal(result.failure.code, 'INVALID_HOST_RESPONSE');
    }
  }
});

test('transport exception hides private message', async () => {
  const client = createReadOnlyClient(async () => { throw Error('/private/host/secret'); });
  const result = await client.validateWorkspace('workspace-1');
  assert.equal(result.ok, false);
  assert.equal(result.failure.code, 'HOST_TRANSPORT_FAILURE');
  assert.ok(!JSON.stringify(result).includes('/private/host/secret'));
});
