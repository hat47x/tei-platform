// E25: two actual SUI Git revisions and one pinned TEI revision, compared
// through the existing E3 read-only source-identity checker. These Git blob
// IDs were independently observed at the pinned revisions. They are evidence
// inputs, not a new package lock or a verified dependency closure.
import test from 'node:test';
import assert from 'node:assert/strict';
import {assessScopedChanges} from './e3-requalification.mjs';

const suiRepo = 'hat47x/sui-sensemaking';
const teiRepo = 'hat47x/tei-platform';
const sui = path => `${suiRepo}/${path}`;
const tei = path => `${teiRepo}/${path}`;
const files = {
  dispatch: sui('03_Implement/frontend/src/api/tei_card_move_action.ts'),
  client: sui('03_Implement/frontend/src/api/client.ts'),
  receiver: sui('03_Implement/backend/src/sui_sensemaking_api/routes/docs.py'),
  move: sui('03_Implement/backend/src/sui_sensemaking_api/card_move_command.py'),
  clientTest: sui('03_Implement/frontend/src/api/client.test.ts'),
  casTest: sui('03_Implement/backend/tests/test_document_action_cas.py'),
  canvasTest: sui('03_Implement/frontend/src/canvas/CardView.native_commit.test.ts'),
  ci: sui('.github/workflows/native-action-contract-ci.yml'),
  protocol: tei('tools/reference-sui-action-boundary/action-protocol-v1.schema.json'),
  app: tei('reference/workspace/applications/order-entry/application.tei'),
  pattern: tei('reference/workspace/applications/order-entry/patterns/submit-order.tei'),
  ui: tei('reference/ui-customization/web-react/project-extension.mjs'),
  typedCandidate: tei('reference/application-composition/e24_native_input_contract_boundary_test.go'),
};

const same = [
  [files.dispatch, '667e1ed411851db57b887cc77443234ffed4ce75'],
  [files.client, 'ca1851ecb4dafd4bab98716451575453bce8ec94'],
  [files.receiver, 'd30b75e4e7ef78a07e00a10f00256536c56b3352'],
  [files.move, 'cc46de3a60c97d48f4eab49e7bab5e5f461d40f7'],
  [files.protocol, '280b04486dfc65a9a213c9084ef1b56d32b157de'],
  [files.app, '00c29ac0e557cb50fdcd13f2477d0b538a528c80'],
  [files.pattern, 'edbb31af1a2018efe1f7e830cac4d06d2f59b188'],
  [files.ui, 'c454f31d575be7a99b23fda6dc4955f4f8ada090'],
  [files.typedCandidate, 'b235400f28d4b7f94a499d76b3187ef82b738ef9'],
];
const oldEvidence = [
  [files.clientTest, '1fa722f2efc4f8345c9d29b95192c90f7d43e1bc'],
  [files.casTest, '178804f3ab7896b241f7588383264ee0500f411d'],
  [files.canvasTest, '2f422e2350e28934476443389cf7e0a3c1d32b04'],
];
const newEvidence = [
  [files.clientTest, 'c3884fd3df9e10add994134836b0863efdd69854'],
  [files.casTest, '83305e742554313a45eea4b9481fd58973985c2a'],
  [files.canvasTest, '7d6c64bafba635cde910ebd59c16b5d7714c96e0'],
  [files.ci, '84fe3181ea2bdc7c256d54f38981e6cf385ab294'],
];
const entries = rows => rows.map(([path, gitBlobSha]) => ({path, gitBlobSha}));
const baseline = entries([...same, ...oldEvidence]);
const current = entries([...same, ...newEvidence]);

// These dependencies are curated from the observed source contracts. The
// checker does not discover or authenticate them. In particular, "unchanged"
// is NOT an authorization, compatibility proof or complete dependency closure.
const consumers = [
  {id: 'sui-native-dispatch', dependencies: [files.dispatch, files.client, files.protocol]},
  {id: 'sui-native-receiver', dependencies: [files.receiver, files.move, files.protocol]},
  {id: 'sui-client-evidence', dependencies: [files.clientTest]},
  {id: 'sui-storage-evidence', dependencies: [files.casTest]},
  {id: 'sui-canvas-evidence', dependencies: [files.canvasTest]},
  {id: 'sui-ci-evidence', dependencies: [files.ci]},
  {id: 'order-web-command', dependencies: [files.app, files.pattern, files.protocol]},
  {id: 'order-web-overlay', dependencies: [files.ui, files.app]},
  {id: 'tei-typed-candidate', dependencies: [files.typedCandidate, files.protocol]},
];
const assess = (old = baseline, now = current, scope = consumers) =>
  Object.fromEntries(assessScopedChanges({baseline: old, current: now, consumers: scope})
    .map(result => [result.id, result]));

test('E25 actual SUI revision changed evidence but not the four native runtime sources', () => {
  const report = assess();
  for (const id of ['sui-native-dispatch', 'sui-native-receiver',
    'order-web-command', 'order-web-overlay', 'tei-typed-candidate']) {
    assert.deepEqual(report[id], {id, status: 'unchanged', changed: []});
  }
  for (const [id, path] of [
    ['sui-client-evidence', files.clientTest],
    ['sui-storage-evidence', files.casTest],
    ['sui-canvas-evidence', files.canvasTest],
  ]) {
    assert.equal(report[id].status, 'unresolved');
    assert.equal(report[id].reason, 'dependency_changed');
    assert.deepEqual(report[id].changed, [path]);
  }
  assert.equal(report['sui-ci-evidence'].status, 'unresolved');
  assert.equal(report['sui-ci-evidence'].reason, 'missing_source_identity');
  assert.deepEqual(report['sui-ci-evidence'].missing, [files.ci]);
});

test('E25 same SHA inputs across two distinct repositories cannot be collapsed by basename', () => {
  const name = 'sample/README.md';
  const left = tei(name);
  const right = sui(name);
  const old = entries([[left, 'a'.repeat(40)], [right, 'b'.repeat(40)]]);
  const now = entries([[left, 'a'.repeat(40)], [right, 'c'.repeat(40)]]);
  const report = assess(old, now, [
    {id: 'tei-doc', dependencies: [left]},
    {id: 'sui-doc', dependencies: [right]},
  ]);
  assert.equal(report['tei-doc'].status, 'unchanged');
  assert.deepEqual(report['sui-doc'].changed, [right]);
});

test('E25 changed TEI protocol would require both applications to requalify', () => {
  const changed = current.map(source => source.path === files.protocol
    ? {...source, gitBlobSha: 'f'.repeat(40)} : source);
  const report = assess(baseline, changed);
  assert.deepEqual(report['sui-native-dispatch'].changed, [files.protocol]);
  assert.deepEqual(report['sui-native-receiver'].changed, [files.protocol]);
  assert.deepEqual(report['order-web-command'].changed, [files.protocol]);
  assert.deepEqual(report['tei-typed-candidate'].changed, [files.protocol]);
  assert.equal(report['order-web-overlay'].status, 'unchanged');
});

test('E25 missing SUI receiver identity is unresolved without affecting order web', () => {
  const withoutReceiver = current.filter(source => source.path !== files.receiver);
  const report = assess(baseline, withoutReceiver);
  assert.equal(report['sui-native-receiver'].reason, 'missing_source_identity');
  assert.deepEqual(report['sui-native-receiver'].missing, [files.receiver]);
  assert.equal(report['order-web-command'].status, 'unchanged');
});

test('E25 no second lock: duplicate, invalid or cross-repo ambiguous source is rejected', () => {
  assert.throws(() => assess(baseline, [...current, current[0]]), /invalid_or_duplicate_source/);
  assert.throws(() => assess(baseline, [...current, {path: '../bad', gitBlobSha: 'a'.repeat(40)}]),
    /invalid_or_duplicate_source/);
  assert.throws(() => assess(baseline, current, [
    {id: 'ambiguous', dependencies: [files.client, files.client]},
  ]), /invalid_consumer/);
});

test('E25 exactly three existing SUI test blobs changed, and the CI source was added', () => {
  const oldByPath = new Map(baseline.map(item => [item.path, item.gitBlobSha]));
  const modified = current.filter(item => oldByPath.has(item.path) &&
    oldByPath.get(item.path) !== item.gitBlobSha).map(item => item.path);
  const added = current.filter(item => !oldByPath.has(item.path)).map(item => item.path);
  assert.deepEqual(modified.sort(), [files.clientTest, files.casTest, files.canvasTest].sort());
  assert.deepEqual(added, [files.ci]);
});
