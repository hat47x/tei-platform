import { readFileSync } from 'node:fs';
import { test } from 'node:test';
import assert from 'node:assert/strict';
import { projectAnnotationEvidence, AUXILIARY_SCHEMA, EPISTEMIC_CONTRACT, EPISTEMIC_SCHEMA } from './annotation-evidence-list-projection.mjs';

const fixture = () => ({
  auxiliary: { schema: AUXILIARY_SCHEMA, annotations: [
    { id: 'n2', subject: '', kind: 'issue', state: 'open', text: '<img src=x onerror=alert(1)>' },
    { id: 'n1', subject: 'Data.Order', kind: 'question', state: 'done', text: 'Reviewed note; not an accepted mapping' },
  ], findings: [{ code: 'annotation-without-subject', subject: 'n2' }] },
  epistemic: { contract: EPISTEMIC_CONTRACT, schema: EPISTEMIC_SCHEMA, assessments: [
    { assertionId: 'e3', statementKind: 'hypothesis', confirmationState: 'confirmed', targetBinding: 'exact', useState: 'candidate-only' },
    { assertionId: 'e2', statementKind: 'fact', confirmationState: 'unreviewed', targetBinding: 'detached', useState: 'review-required' },
    { assertionId: 'e1', statementKind: 'fact', confirmationState: 'rejected', targetBinding: 'ambiguous', useState: 'blocked' },
  ] },
});

test('注記の完了・人の確認・対応受理を混同せず、独立した一覧へ投影する', () => {
  const { auxiliary, epistemic } = fixture();
  const before = structuredClone({ auxiliary, epistemic });
  const got = projectAnnotationEvidence(auxiliary, epistemic);
  assert.deepEqual({ auxiliary, epistemic }, before, 'read-only');
  assert.deepEqual(got.notes.map(x => x.id), ['n1', 'n2']);
  assert.match(got.notes[0].workState, /対応済み/);
  assert.match(got.notes[0].acceptance, /判定不可/);
  assert.match(got.assessments.find(x => x.id === 'e3').confirmation, /人が確認/);
  assert.match(got.assessments.find(x => x.id === 'e3').useState, /候補のみ/);
  assert.match(got.assessments.find(x => x.id === 'e3').acceptance, /判定不可/);
  assert.equal(got.notes[0].subject, 'Data.Order');
  assert.equal(got.assessments.length, 3, '注記のsubjectを勝手にassertionと結合しない');
});

test('対象位置が曖昧・参照切れでも別要素へ付け替えない', () => {
  const { auxiliary, epistemic } = fixture();
  const got = projectAnnotationEvidence(auxiliary, epistemic);
  assert.deepEqual(got.assessments.filter(a => a.needsLocatorReview).map(a => a.id), ['e1', 'e2']);
  assert.match(got.assessments[0].targetBinding, /曖昧/);
  assert.match(got.assessments[1].targetBinding, /参照切れ/);
  assert.match(got.notes[1].locator, /参照なし/);
  assert.deepEqual(got.notes[1].findings, ['annotation-without-subject']);
});

test('未知のcontract/schemaはfail closed、未知の状態値は受理へ昇格しない', () => {
  const { auxiliary, epistemic } = fixture();
  assert.throws(() => projectAnnotationEvidence({ ...auxiliary, schema: 'unknown' }, epistemic), /未対応/);
  assert.throws(() => projectAnnotationEvidence(auxiliary, { ...epistemic, contract: 'unknown' }), /未対応/);
  assert.throws(() => projectAnnotationEvidence(auxiliary, { ...epistemic, schema: 'unknown' }), /未対応/);
  assert.throws(() => projectAnnotationEvidence({ ...auxiliary, findings: null }, epistemic), /未対応/);
  const unknown = projectAnnotationEvidence({ ...auxiliary, annotations: [{ id: 'unknown', subject: '', kind: '??', state: 'accepted' }] }, epistemic);
  assert.match(unknown.notes[0].workState, /未対応の値/);
  assert.doesNotMatch(unknown.notes[0].workState, /注記：対応済み/);
});


const completeAssessment = () => ({
  assertionId: 'e4', statementKind: 'hypothesis', confirmationState: 'confirmed',
  useState: 'candidate-only', targetBinding: 'exact', reviewBinding: 'current',
  metadataState: 'complete', contentOrigin: 'user', ingestedBy: 'user',
  contextCompatibility: 'compatible', freshness: 'current',
  lifecycleState: 'active', conflict: 'none', inferredContextUsed: false, reasons: [],
});

test('人が確認した仮説でも候補に留め、詳細の独立軸を保持する', () => {
  const { auxiliary, epistemic } = fixture();
  epistemic.assessments = [completeAssessment()];
  const before = structuredClone(epistemic);
  const got = projectAnnotationEvidence(auxiliary, epistemic).assessments[0];
  assert.deepEqual(epistemic, before, 'read-only');
  assert.match(got.confirmation, /人が確認/);
  assert.match(got.useState, /候補のみ/);
  assert.match(got.acceptance, /判定不可/);
  assert.equal(got.detailRows.length, 10);
  assert.ok(got.detailRows.some(d => d.label === '確認履歴の現在性' && /現在の対象に有効/.test(d.value)));
  assert.equal(got.hasAttention, false);
  assert.match(got.attentionSummary, /受理・全体健全性を意味しない/);
});

test('旧review・メタデータ欠落・AI由来と取込み主体を混同せず注意を分ける', () => {
  const { auxiliary, epistemic } = fixture();
  epistemic.assessments = [{
    ...completeAssessment(), reviewBinding: 'stale', metadataState: 'partial',
    contentOrigin: 'ai', ingestedBy: 'user',
  }];
  const got = projectAnnotationEvidence(auxiliary, epistemic).assessments[0];
  assert.match(got.confirmation, /人が確認/, 'Referenceの元の表示値を勝手に書き換えない');
  assert.match(got.useState, /候補のみ/);
  assert.ok(got.attention.some(s => s.includes('過去の内容確認')));
  assert.ok(got.attention.some(s => s.includes('メタデータが不足')));
  assert.ok(got.attention.some(s => s.includes('AI生成情報')));
  assert.ok(got.detailRows.some(d => d.label === '内容の由来' && /AI/.test(d.value)));
  assert.ok(got.detailRows.some(d => d.label === '取込み主体' && /利用者/.test(d.value)));
  assert.match(got.acceptance, /判定不可/);
});

test('文脈・鮮度・競合・置換・参照切れと評価理由を独立に表示する', () => {
  const { auxiliary, epistemic } = fixture();
  const reason = '<img src=x onerror=alert(1)>';
  epistemic.assessments = [{
    ...completeAssessment(), targetBinding: 'detached',
    contextCompatibility: 'incompatible', freshness: 'stale', conflict: 'present',
    lifecycleState: 'superseded', inferredContextUsed: true, reasons: [reason],
  }];
  const got = projectAnnotationEvidence(auxiliary, epistemic).assessments[0];
  assert.equal(got.needsLocatorReview, true);
  assert.ok(got.attention.some(s => s.includes('対象位置')));
  assert.ok(got.attention.some(s => s.includes('文脈')));
  assert.ok(got.attention.some(s => s.includes('鮮度')));
  assert.ok(got.attention.some(s => s.includes('競合')));
  assert.ok(got.attention.some(s => s.includes('置換')));
  assert.ok(got.attention.some(s => s.includes('推定')));
  assert.equal(got.detailRows.find(d => d.label.startsWith('評価理由')).value, reason,
    '表示用文字列を保持し、DOM側でtextContentにより非HTMLとして出す');
});

test('未提供・未知の詳細状態を良好・対応受理へ昇格させない', () => {
  const { auxiliary, epistemic } = fixture();
  epistemic.assessments = [{ ...completeAssessment(),
    metadataState: undefined, reviewBinding: 'unexpected', freshness: 'mystery',
    conflict: undefined, reasons: undefined, inferredContextUsed: undefined,
  }];
  const got = projectAnnotationEvidence(auxiliary, epistemic).assessments[0];
  assert.equal(got.hasAttention, true);
  assert.ok(got.attention.some(s => s.includes('未提供')));
  assert.ok(got.attention.some(s => s.includes('未対応の状態値')));
  assert.ok(got.detailRows.some(d => d.value === 'Referenceに値なし'));
  assert.ok(got.detailRows.some(d => d.value.includes('未対応の値')));
  assert.match(got.acceptance, /判定不可/);
});


test('HTML画面配線: 詳細はDOMテキスト、未知schema時は表示を消して拒否する（疑似DOM）', () => {
  const html = readFileSync(new URL('./annotation-evidence-list-sketch.html', import.meta.url), 'utf8');
  const script = html.match(/<script type="module">([\s\S]*?)<\/script>/)?.[1];
  assert.ok(script, '画面module scriptが存在する');
  assert.doesNotMatch(script, /\.innerHTML\s*=/, '入力をHTMLへ評価しない');

  class FakeElement {
    constructor(tag) { this.tag = tag; this.children = []; this.handlers = {}; this._text = ''; this.value = ''; }
    set textContent(v) { this._text = String(v); this.children = []; }
    get textContent() { return this._text + this.children.map(c => c.textContent).join(''); }
    append(...nodes) { this.children.push(...nodes); }
    replaceChildren(...nodes) { this._text = ''; this.children = [...nodes]; }
    addEventListener(name, callback) { this.handlers[name] = callback; }
  }
  const ids = new Map();
  const document = {
    getElementById(id) {
      if (!ids.has(id)) ids.set(id, new FakeElement('id:' + id));
      return ids.get(id);
    },
    createElement(tag) { return new FakeElement(tag); },
  };
  const moduleBody = script.split('\n').filter(line => !line.startsWith('import ')).join('\n');
  new Function('document', 'projectAnnotationEvidence', moduleBody)(document, projectAnnotationEvidence);
  const { auxiliary, epistemic } = fixture();
  const escaped = '<svg/onload=alert(1)>';
  epistemic.assessments = [{ ...completeAssessment(), reasons: [escaped], metadataState: 'partial' }];
  document.getElementById('aux').value = JSON.stringify(auxiliary);
  document.getElementById('epi').value = JSON.stringify(epistemic);
  document.getElementById('display').handlers.click();

  const rows = document.getElementById('assessments').children;
  assert.equal(rows.length, 1);
  const tds = rows[0].children;
  assert.match(tds[6].textContent, /メタデータが不足/);
  assert.match(tds[7].textContent, /評価理由/);
  assert.match(tds[7].textContent, /<svg\/onload=alert\(1\)>/);
  assert.equal(tds[7].children[0].tag, 'details', '元データはDOMテキストで表示');

  document.getElementById('epi').value = JSON.stringify({ ...epistemic, schema: 'unknown' });
  document.getElementById('display').handlers.click();
  assert.equal(document.getElementById('assessments').children.length, 0);
  assert.equal(document.getElementById('notes').children.length, 0);
  assert.match(document.getElementById('error').textContent, /未対応/);
});
