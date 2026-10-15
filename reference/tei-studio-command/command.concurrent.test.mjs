import assert from 'node:assert/strict';
import { test } from 'node:test';
import { createCommandPort } from './command.ts';

const commands = [{ id: 'validate', label: 'Validate' }, { id: 'refresh', label: 'Refresh' }];
const deferred = () => { let resolve, reject; const promise = new Promise((yes, no) => { resolve = yes; reject = no; }); return { promise, resolve, reject }; };
const finding = (id, severity = 'warning') => [{ id, severity, message: `issue:${id}` }];

test('C5-01: validation burst never double dispatches within a context', async () => {
  const wait = deferred();
  const calls = [];
  const port = createCommandPort({ commands, invoke: (id, signal) => { calls.push({ id, signal }); return wait.promise; } });
  const pending = Array.from({ length: 30 }, () => port.execute('validate'));
  assert.equal(port.snapshot().status.phase, 'running');
  assert.equal(calls.length, 1);
  wait.resolve({ kind: 'success', message: 'only once' });
  const results = await Promise.all(pending);
  assert.equal(results.filter(Boolean).length, 1);
  assert.equal(port.snapshot().status.phase, 'success');
  port.dispose();
});

test('C5-02: workspace switch invalidates a pending operation, clears status and diagnostics', async () => {
  const previous = deferred();
  const next = deferred();
  const signals = [];
  const port = createCommandPort({ commands, invoke: (_, signal) => { signals.push(signal); return signals.length === 1 ? previous.promise : next.promise; } });
  const oldContext = port.beginContext();
  port.setDiagnosticsForContext(oldContext, finding('old'));
  const oldRun = port.execute('validate');
  assert.equal(port.snapshot().status.phase, 'running');
  const newContext = port.beginContext();
  assert.notEqual(oldContext, newContext);
  assert.ok(signals[0].aborted);
  assert.equal(port.snapshot().status.phase, 'idle');
  assert.deepEqual(port.snapshot().diagnostics, []);
  const newRun = port.execute('validate');
  assert.equal(signals.length, 2, 'new workspace can validate without waiting for old request');
  previous.resolve({ kind: 'success', message: 'OLD SUCCESS' });
  assert.equal(await oldRun, false);
  assert.equal(port.snapshot().status.phase, 'running');
  next.resolve({ kind: 'success', message: 'new success' });
  assert.equal(await newRun, true);
  assert.equal(port.snapshot().status.message, 'new success');
  port.dispose();
});

test('C5-03: Host denial, rejection and subsequent retry stay distinct', async () => {
  const results = [
    { kind: 'denied', message: 'not permitted' },
    { kind: 'failure', message: 'network failed' },
    { kind: 'success', message: 'validated with warning' },
  ];
  const port = createCommandPort({ commands, invoke: async () => results.shift() });
  assert.equal(await port.execute('validate'), true, 'dispatch accepted, not semantic success');
  assert.equal(port.snapshot().status.phase, 'denied');
  assert.equal(await port.execute('validate'), true);
  assert.equal(port.snapshot().status.phase, 'failure');
  assert.equal(await port.execute('validate'), true);
  assert.equal(port.snapshot().status.phase, 'success');
  port.dispose();
});

test('C5-04: stale validation diagnostics cannot overwrite a newer workspace or reuse its token', () => {
  const port = createCommandPort({ commands, invoke: async () => ({ kind: 'success' }) });
  const old = port.beginContext();
  assert.equal(port.setDiagnosticsForContext(old, finding('old')), true);
  const current = port.beginContext();
  assert.equal(port.setDiagnosticsForContext(current, finding('new')), true);
  assert.equal(port.setDiagnosticsForContext(old, finding('late-error', 'error')), false);
  assert.deepEqual(port.snapshot().diagnostics, finding('new'));
  const refreshed = port.beginContext();
  assert.equal(port.setDiagnosticsForContext(current, finding('late-while-same-workspace')), false);
  assert.equal(port.setDiagnosticsForContext(refreshed, []), true);
  assert.deepEqual(port.snapshot().diagnostics, []);
  port.dispose();
});

test('C5-05: result envelope and semantic diagnostics must not be conflated', async () => {
  // Semantic interpretation belongs to Host adapter, not the UI panel.
  const envelopes = [
    { kind: 'failure', message: 'validation errors: 1' },
    { kind: 'success', message: 'validation completed, warnings: 1' },
  ];
  const port = createCommandPort({ commands, invoke: async () => envelopes.shift() });
  const context = port.beginContext();
  port.setDiagnosticsForContext(context, finding('syntax-error', 'error'));
  assert.equal(await port.execute('validate'), true); // accepted dispatch, semantically failed
  assert.equal(port.snapshot().status.phase, 'failure');
  assert.equal(port.snapshot().diagnostics[0].severity, 'error');
  port.setDiagnosticsForContext(context, finding('warning', 'warning'));
  await port.execute('validate');
  assert.equal(port.snapshot().status.phase, 'success');
  assert.equal(port.snapshot().diagnostics[0].severity, 'warning');
  port.dispose();
});

test('C5-06: synchronous listener context change prevents starting a stale Host call', async () => {
  let called = 0;
  const port = createCommandPort({ commands, invoke: async () => { called++; return { kind: 'success' }; } });
  port.subscribe((state) => {
    if (state.status.phase === 'running') port.beginContext();
  });
  assert.equal(await port.execute('validate'), false);
  assert.equal(called, 0);
  assert.equal(port.snapshot().status.phase, 'idle');
  port.dispose();
});

test('C5-07: old Host rejection after switching cannot show a new failure or leak the old error', async () => {
  const old = deferred();
  const newer = deferred();
  let invocations = 0;
  const seen = [];
  const port = createCommandPort({
    commands,
    invoke: () => ++invocations === 1 ? old.promise : newer.promise,
  });
  port.subscribe(state => seen.push(state.status.phase));
  const oldToken = port.beginContext();
  const oldRun = port.execute('validate');
  const nextToken = port.beginContext();
  assert.notEqual(oldToken, nextToken);
  const nextRun = port.execute('validate');
  const seenBeforeLateError = seen.length;
  old.reject(new Error('old secret path /private/tenant-a'));
  assert.equal(await oldRun, false);
  assert.equal(seen.length, seenBeforeLateError, 'stale failure has no notification');
  assert.equal(port.snapshot().status.phase, 'running');
  assert.equal(port.setDiagnosticsForContext(oldToken, finding('old-error', 'error')), false);
  newer.resolve({ kind: 'denied', message: 'new Host denial' });
  assert.equal(await nextRun, true);
  assert.deepEqual(port.snapshot().status, {
    phase: 'denied', commandId: 'validate', message: 'new Host denial',
  });
  assert.equal(JSON.stringify(port.snapshot()).includes('/private/tenant-a'), false);
  port.dispose();
});

test('C5-08: a new workspace accepts its own validation once despite old and new bursts', async () => {
  const old = deferred();
  const newer = deferred();
  let calls = 0;
  const signals = [];
  const port = createCommandPort({
    commands,
    invoke: (_id, signal) => {
      signals.push(signal);
      return ++calls === 1 ? old.promise : newer.promise;
    },
  });
  const firstToken = port.beginContext();
  const oldBurst = Array.from({ length: 20 }, () => port.execute('validate'));
  const newToken = port.beginContext();
  assert.notEqual(firstToken, newToken);
  assert.equal(signals[0].aborted, true);
  const newBurst = Array.from({ length: 20 }, () => port.execute('validate'));
  assert.equal(calls, 2, 'one invocation per context');
  newer.resolve({ kind: 'success', message: 'new valid result' });
  assert.equal((await Promise.all(newBurst)).filter(Boolean).length, 1);
  old.resolve({ kind: 'success', message: 'old result should not surface' });
  assert.equal((await Promise.all(oldBurst)).filter(Boolean).length, 0);
  assert.deepEqual(port.snapshot().status, {
    phase: 'success', commandId: 'validate', message: 'new valid result',
  });
  port.dispose();
});

test('C5-09: rejected stale diagnostics do not validate payload or send a notification', () => {
  const port = createCommandPort({ commands, invoke: async () => ({ kind: 'success' }) });
  const stale = port.beginContext();
  const fresh = port.beginContext();
  let count = 0;
  port.subscribe(() => { count++; });
  assert.equal(port.setDiagnosticsForContext(stale, [{ id: '', severity: 'bad', message: '' }]), false);
  assert.equal(count, 1);
  assert.equal(port.setDiagnosticsForContext(fresh, finding('current')), true);
  assert.equal(count, 2);
  port.dispose();
  assert.throws(() => port.beginContext(), /disposed/);
});
