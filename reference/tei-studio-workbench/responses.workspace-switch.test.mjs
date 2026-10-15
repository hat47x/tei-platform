import { test } from 'node:test';
import assert from 'node:assert/strict';
import { createResponseScope } from './responses.ts';

function deferred() {
  let resolve, reject;
  const promise = new Promise((yes, no) => { resolve = yes; reject = no; });
  return { promise, resolve, reject };
}

for (const lateOutcome of ['success', 'failure']) {
  test(`A delayed validation ${lateOutcome} never replaces B's view and diagnostics`, async () => {
    const scope = createResponseScope();
    const oldCatalog = deferred();
    const oldValidation = deferred();
    const view = { workspace: 'A', model: 'A.order', diagnostics: ['A previous warning'], notice: 'A pending' };
    const failures = [];

    // A: real Host calls are represented by deferred promises.
    const aCatalog = scope.run('catalog', () => oldCatalog.promise,
      v => { view.model = v; }, e => { failures.push(e.message); });
    const aValidation = scope.run('validation', () => oldValidation.promise,
      v => { view.diagnostics = v; view.notice = 'A valid'; },
      e => { failures.push(e.message); view.notice = 'A error'; });

    // Workspace selector changes A -> B before either Host reply arrives.
    scope.reset();
    view.workspace = 'B';
    view.model = 'B loading';
    view.diagnostics = [];
    await scope.run('catalog', async () => 'B.request_log',
      v => { view.model = v; }, e => { failures.push(e.message); });
    await scope.run('validation', async () => ['B warning'],
      v => { view.diagnostics = v; view.notice = 'B valid'; },
      e => { failures.push(e.message); });
    const expected = structuredClone(view);

    // A's catalog and validation complete after B is already visible.
    oldCatalog.resolve('A.order stale');
    if (lateOutcome === 'success') oldValidation.resolve(['A error', 'A warning']);
    else oldValidation.reject(new Error('A Host failed'));

    assert.deepEqual(await Promise.all([aCatalog, aValidation]), [false, false]);
    assert.deepEqual(view, expected);
    assert.deepEqual(view, { workspace: 'B', model: 'B.request_log', diagnostics: ['B warning'], notice: 'B valid' });
    assert.deepEqual(failures, []);
    scope.dispose();
  });
}
