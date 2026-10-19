import { test } from 'node:test';
import { readFileSync } from 'node:fs';
import { projectAnnotationEvidence, AUXILIARY_SCHEMA, EPISTEMIC_SCHEMA, EPISTEMIC_CONTRACT } from './annotation-evidence-list-projection.mjs';
import assert from 'node:assert/strict';
import { projectDecisionIndex, DECISION_INDEX_SCHEMA } from './annotation-decision-index-projection.mjs';

const one = (patch = {}) => ({
  path: '.tei/decisions/record.json',
  status: 'current',
  sourceEdits: 'intact',
  evidenceStatus: 'current-with-recorded-artifacts',
  sourceEvidenceStatus: 'current',
  artifactEvidenceStatus: 'recorded',
  reviewBindingStatus: 'verified',
  reviewCoverageStatus: 'incomplete',
  decisionActorStatus: 'unresolved',
  decisionAuthorityStatus: 'unresolved',
  decision: { decision: 'accepted', sourceChanged: true, modelRevisionAfter: 'sha256:revision', ...patch.decision },
  ...Object.fromEntries(Object.entries(patch).filter(([key]) => key !== 'decision')),
});
const index = (records = []) => ({ schema: DECISION_INDEX_SCHEMA, modelRevision: 'sha256:revision', records });

test('受理結果・現行Source版・Review指紋整合は人的Owner承認に昇格しない', () => {
  const data = index([one()]);
  const original = JSON.stringify(data);
  const got = projectDecisionIndex(data);
  assert.equal(JSON.stringify(data), original, 'read-only');
  const row = got.records[0];
  assert.match(row.outcome, /参照実装の受理結果/);
  assert.match(row.sourceRevision, /一致/);
  assert.match(row.review, /人の承認の証明ではない/);
  assert.match(row.authority, /承認資格未確認/);
  assert.match(row.acceptedMeaning, /判定不可/);
  assert.match(row.warnings, /Review/);
});

test('rejectedかつcurrentは正式に受理された意味ではない', () => {
  const row = projectDecisionIndex(index([one({
    decision: { decision: 'rejected', sourceChanged: false },
    sourceEdits: 'none', reviewCoverageStatus: 'complete',
  })])).records[0];
  assert.match(row.outcome, /却下/);
  assert.match(row.sourceRevision, /一致/);
  assert.match(row.warnings, /Source受理ではない/);
  assert.match(row.acceptedMeaning, /判定不可/);
});

test('supersededとintactを独立して示し現在の承認資格に見せない', () => {
  const row = projectDecisionIndex(index([one({ status: 'superseded', sourceEdits: 'intact', reviewCoverageStatus: 'complete' })])).records[0];
  assert.match(row.sourceRevision, /不一致/);
  assert.match(row.sourceEdit, /現在も一致/);
  assert.match(row.warnings, /現在のWorkspace版/);
  assert.match(row.authority, /未確認/);
});

test('empty Indexは履歴なしの証明ではない。未知schemaと不正recordsを拒否', () => {
  const empty = projectDecisionIndex(index());
  assert.equal(empty.records.length, 0);
  assert.match(empty.summary, /過去の受理操作がなかった証明ではありません/);
  assert.throws(() => projectDecisionIndex({ ...index(), schema: 'unknown' }), /未対応/);
  assert.throws(() => projectDecisionIndex({ ...index(), records: null }), /未対応/);
  assert.throws(() => projectDecisionIndex(index([null])), /records\[0\]/);
  assert.throws(() => projectDecisionIndex(index([{}])), /records\[0\]/);
});

test('不整合、未知権限、警告コードと任意文字列を承認済みに変換しない', () => {
  const dangerous = '<img src=x onerror=alert(1)>';
  const row = projectDecisionIndex(index([one({
    path: dangerous, status: 'superseded', sourceEdits: 'diverged', evidenceStatus: 'diverged',
    reviewBindingStatus: 'invalid', decisionAuthorityStatus: 'verified',
    decision: { decision: 'accepted', warnings: [{ code: 'DECISION_RECORD_FAILED' }] },
  })])).records[0];
  assert.match(row.warnings, /根拠が不一致/);
  assert.match(row.warnings, /未対応の状態値/);
  assert.match(row.authority, /未対応の値/);
  assert.equal(row.details.find(x => x.label.startsWith('Decision履歴の位置')).value, dangerous);
  assert.match(row.details.find(x => x.label === '記録時の警告コード').value, /DECISION_RECORD_FAILED/);
  assert.match(row.acceptedMeaning, /判定不可/);
});


test('HTMLの三つの独立一覧を表示し未知Decision schemaで全部消去する（合成DOM）', () => {
  const html = readFileSync(new URL('./annotation-evidence-list-sketch.html', import.meta.url), 'utf8');
  const script = html.match(/<script type="module">([\s\S]*?)<\/script>/)?.[1];
  assert.ok(script);
  assert.doesNotMatch(script, /\.innerHTML\s*=/);
  class FakeElement {
    constructor(tag) { this.tag=tag; this.children=[]; this.handlers={}; this._text=''; this.value=''; }
    set textContent(v) { this._text=String(v); this.children=[]; }
    get textContent() { return this._text+this.children.map(c=>c.textContent).join(''); }
    append(...nodes) { this.children.push(...nodes); }
    replaceChildren(...nodes) { this._text=''; this.children=[...nodes]; }
    addEventListener(name,fn) { this.handlers[name]=fn; }
  }
  const ids=new Map();
  const document={
    getElementById(id){if(!ids.has(id))ids.set(id,new FakeElement('id:'+id));return ids.get(id);},
    createElement(tag){return new FakeElement(tag);},
  };
  const scriptBody=script.split('\n').filter(s=>!s.startsWith('import ')).join('\n');
  new Function('document','projectAnnotationEvidence','projectDecisionIndex',scriptBody)(
    document,projectAnnotationEvidence,projectDecisionIndex);
  const aux={schema:AUXILIARY_SCHEMA,annotations:[{id:'note-1',subject:'subject-1',kind:'note',state:'done'}],findings:[]};
  const epi={schema:EPISTEMIC_SCHEMA,contract:EPISTEMIC_CONTRACT,assessments:[]};
  document.getElementById('aux').value=JSON.stringify(aux);
  document.getElementById('epi').value=JSON.stringify(epi);
  document.getElementById('decisions').value=JSON.stringify(index([one({path:'<img onerror=alert(1)>'})]));
  const click=()=>document.getElementById('display').handlers.click();
  click();
  assert.equal(document.getElementById('notes').children.length,1);
  assert.equal(document.getElementById('decision-rows').children.length,1);
  const cells=document.getElementById('decision-rows').children[0].children;
  assert.match(cells[7].textContent,/判定不可/);
  assert.match(cells[8].textContent,/<img onerror=alert\(1\)>/);
  assert.equal(cells[8].children[0].tag,'details');
  document.getElementById('decisions').value=JSON.stringify({...index([one()]),schema:'unexpected'});
  click();
  assert.equal(document.getElementById('notes').children.length,0);
  assert.equal(document.getElementById('assessments').children.length,0);
  assert.equal(document.getElementById('decision-rows').children.length,0);
  assert.match(document.getElementById('error').textContent,/未対応/);
  document.getElementById('decisions').value='';
  click();
  assert.equal(document.getElementById('decision-rows').children.length,0);
  assert.match(document.getElementById('decision-status').textContent,/未入力/);
  assert.match(document.getElementById('decision-status').textContent,/履歴がないとは判定しません/);
});
