// Independent Expected for the *selected-view* lane (Inspect) and the isolated
// Pin lane. The rows below are written from the requirement, not from the
// current implementation: switching Workspace or Subject must not let an older
// reply decide what the user is looking at, and an explicit Pin must survive a
// transient selection change. A lane-scope utility is exercised here; it is not
// a Host Grant, a cancellation of Host work, or a disclosure guarantee.
import test from 'node:test';
import assert from 'node:assert/strict';
import { createResponseScope } from './responses.ts';
import {
  initialPinState, pinSubject, invalidatePin, pinRequest, acceptPinObservation,
} from './pin-state.ts';

const A = { kind: 'semantic-data', id: 'sales.order' };

function deferred() {
  let resolve, reject;
  const promise = new Promise((yes, no) => { resolve = yes; reject = no; });
  return { promise, resolve, reject };
}

// A stand-in for the parts of the Workbench the user can see after Inspect.
function view() {
  return {
    workspaceId: 'A',
    subject: null,
    content: 'A loading',
    sourceStatus: '取得中',
    notice: '一覧を取得しています。',
  };
}

test('ASYNC-INSP-01: late success from Inspect in A never overwrites the visible B', async () => {
  const scope = createResponseScope();
  const oldA = deferred();
  const shown = view();

  const pendingA = scope.run('inspection', () => oldA.promise, (reply) => {
    shown.subject = reply.subject;
    shown.content = reply.text;
    shown.sourceStatus = reply.sourceStatus;
    shown.notice = '選択したモデルの投影を表示しています。';
  }, () => { shown.content = 'この要素の投影は取得できませんでした。'; });

  // Workspace selector changes A -> B while A's Host call is still pending.
  scope.reset();
  shown.workspaceId = 'B';
  shown.content = '作業領域を読み込んでいます。';
  shown.notice = '一覧を取得しています。';
  await scope.run('inspection', async () => ({
    subject: 'sales.invoice', text: 'B projection', sourceStatus: 'indexed',
  }), (reply) => {
    shown.subject = reply.subject;
    shown.content = reply.text;
    shown.sourceStatus = reply.sourceStatus;
    shown.notice = '選択したモデルの投影を表示しています。';
  }, () => { shown.content = '取得できませんでした。'; });
  const expectedForB = structuredClone(shown);

  oldA.resolve({ subject: 'sales.order', text: 'A stale projection', sourceStatus: 'indexed' });
  assert.equal(await pendingA, false);
  assert.deepEqual(shown, expectedForB);
  assert.equal(shown.subject, 'sales.invoice');
  assert.equal(shown.content, 'B projection');
  scope.dispose();
});

test('ASYNC-INSP-02: late failure from Inspect in A never damages the visible B', async () => {
  const scope = createResponseScope();
  const oldA = deferred();
  const shown = view();
  const painted = [];

  const pendingA = scope.run('inspection', () => oldA.promise, () => {
    painted.push('A success');
  }, (error) => {
    painted.push('A failure: ' + error.message);
    shown.content = 'この要素の投影は取得できませんでした。';
    shown.sourceStatus = '未取得';
    shown.notice = '操作を完了できませんでした：' + error.message;
  });

  scope.reset();
  shown.workspaceId = 'B';
  await scope.run('inspection', async () => ({
    subject: 'sales.invoice', text: 'B projection', sourceStatus: 'indexed',
  }), (reply) => {
    shown.subject = reply.subject;
    shown.content = reply.text;
    shown.sourceStatus = reply.sourceStatus;
    shown.notice = '選択したモデルの投影を表示しています。';
  }, () => { shown.content = '取得できませんでした。'; });
  const expectedForB = structuredClone(shown);

  oldA.reject(new Error('SUBJECT_NOT_FOUND'));
  assert.equal(await pendingA, false);
  assert.deepEqual(shown, expectedForB);
  assert.deepEqual(painted, []);
  assert.equal(shown.notice, '選択したモデルの投影を表示しています。');
  scope.dispose();
});

for (const lateOutcome of ['success', 'failure']) {
  test(`ASYNC-INSP-03 (${lateOutcome}): A→B→A never reuses the first A reply as the new A credential`, async () => {
    // Both visits resolve to the same Workspace id and the same Subject id.
    // Returning to the same name is not evidence that the earlier reply is current.
    const scope = createResponseScope();
    const firstA = deferred();
    const shown = view();
    const accepted = [];
    const publish = (reply) => {
      shown.subject = reply.subject;
      shown.content = reply.text;
      accepted.push(reply.attempt);
    };
    const fail = (error) => { accepted.push('error:' + error.message); };

    const a1 = scope.run('inspection', () => firstA.promise, publish, fail);

    scope.reset();
    shown.workspaceId = 'B';
    await scope.run('inspection', async () => ({ attempt: 'B', subject: 'sales.invoice', text: 'B projection' }), publish, fail);

    scope.reset();
    shown.workspaceId = 'A';
    await scope.run('inspection', async () => ({ attempt: 'A2', subject: 'sales.order', text: 'A2 projection' }), publish, fail);
    const expectedForA2 = structuredClone(shown);

    if (lateOutcome === 'success') firstA.resolve({ attempt: 'A1', subject: 'sales.order', text: 'A1 stale projection' });
    else firstA.reject(new Error('A1 failed'));

    assert.equal(await a1, false);
    assert.deepEqual(shown, expectedForA2);
    assert.deepEqual(accepted, ['B', 'A2']);
    assert.equal(shown.content, 'A2 projection');
    scope.dispose();
  });
}

test('ASYNC-INSP-04: a transient selection change neither releases the Pin nor revives it', async () => {
  // The Pin is local intent held outside the response scope. Moving the ordinary
  // selection must not release it, and a reply that belonged to the previous
  // Workspace generation must not be accepted as the current one. Both halves
  // use the real Pin API, so clearing the intent is observable.
  const scope = createResponseScope();
  let pin = pinSubject(initialPinState(), 'workspace-a', 1, A);

  const shown = view();
  shown.workspaceId = 'workspace-a';
  shown.subject = 'sales.invoice';
  shown.content = 'B projection';

  // Ordinary selection changes B -> C in the same Workspace, in its own lane.
  await scope.run('inspection', async () => ({ subject: 'sales.payment', text: 'C projection' }),
    (reply) => { shown.subject = reply.subject; shown.content = reply.text; },
    () => {});
  assert.equal(shown.subject, 'sales.payment');

  // The explicit Pin survived that transient selection change.
  const stillPinned = pinRequest(pin);
  assert.ok(stillPinned, 'a selection change must not release an explicit Pin');
  assert.deepEqual(stillPinned.subject, A);

  // A Workspace-generation change invalidates the observation but keeps intent.
  const staleRequest = stillPinned;
  scope.reset();
  pin = invalidatePin(pin);
  assert.equal(pin.observation.state, 'needs-requalification');
  assert.ok(pin.intent, 'requalification must not silently unpin');

  // The old request cannot qualify the current intent.
  assert.equal(acceptPinObservation(pin, staleRequest, { state: 'inspected' }), pin,
    'a reply from the previous generation must not be accepted');
  assert.equal(pin.observation.state, 'needs-requalification');
  assert.deepEqual(shown, {
    workspaceId: 'workspace-a', subject: 'sales.payment',
    content: 'C projection', sourceStatus: '取得中',
    notice: '一覧を取得しています。',
  });
  scope.dispose();
});

test('ASYNC-INSP-05: the current latest reply is published, including its failure', async () => {
  // Guarding against over-rejection: only superseded replies are dropped.
  const scope = createResponseScope();
  const shown = view();
  const published = [];

  const first = deferred();
  const pendingFirst = scope.run('inspection', () => first.promise, (reply) => {
    published.push(reply.text); shown.content = reply.text;
  }, () => { published.push('first failure'); });

  const second = scope.run('inspection', async () => ({ text: 'second projection' }), (reply) => {
    published.push(reply.text); shown.content = reply.text;
  }, () => { published.push('second failure'); });

  assert.equal(await second, true);
  first.resolve({ text: 'first projection' });
  assert.equal(await pendingFirst, false);
  assert.deepEqual(published, ['second projection']);
  assert.equal(shown.content, 'second projection');

  // A current failure must still be visible: it is the latest answer, not stale.
  assert.equal(await scope.run('inspection', async () => { throw Error('CURRENT_DENIED'); },
    () => { published.push('unexpected success'); },
    (error) => { published.push('failure:' + error.message); }), true);
  assert.deepEqual(published, ['second projection', 'failure:CURRENT_DENIED']);
  scope.dispose();
});

test('ASYNC-INSP-06: disposal during a pending Inspect publishes nothing', async () => {
  const scope = createResponseScope();
  const pending = deferred();
  const shown = view();
  const task = scope.run('inspection', () => pending.promise, (reply) => {
    shown.content = reply.text;
  }, () => { shown.content = 'failure painted'; });
  scope.dispose();
  pending.resolve({ text: 'too late' });
  assert.equal(await task, false);
  assert.equal(shown.content, 'A loading');
});
