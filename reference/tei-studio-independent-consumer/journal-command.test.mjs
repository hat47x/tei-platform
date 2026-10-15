import test from 'node:test';
import assert from 'node:assert/strict';
import { journalCommandOptions } from './journal-command.mjs';

test('Reading Journal provides minimal commands without TEI knowledge', async () => {
  let current = 'tea';
  const options = journalCommandOptions(() => current);
  assert.deepEqual(options.commands.map(x => x.id), ['count-links', 'delete-entry']);
  assert.deepEqual(options.diagnostics.map(x => x.severity), ['info']);
  assert.deepEqual(await options.invoke('count-links'), {
    kind: 'success', message: '2 related entries',
  });
  current = 'sketch';
  assert.deepEqual(await options.invoke('count-links'), {
    kind: 'success', message: '0 related entries',
  });
});

test('consumer refuses deletes regardless of displayed availability', async () => {
  const options = journalCommandOptions(() => 'map');
  assert.equal(options.commands.find(x => x.id === 'delete-entry').enabled, true);
  assert.deepEqual(await options.invoke('delete-entry'), {
    kind: 'denied', message: 'The reading journal does not permit deletion.',
  });
  assert.equal((await options.invoke('unknown')).kind, 'denied');
  assert.throws(() => journalCommandOptions(null), /note selector is required/);
});
