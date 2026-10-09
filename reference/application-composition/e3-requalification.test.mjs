import test from 'node:test';
import assert from 'node:assert/strict';
import {assessScopedChanges, assessCandidateApplicability} from './e3-requalification.mjs';
import {proposeUIHeading, proposeDevelopmentDatabaseBinding} from './e2-change-probe.mjs';

const ui='reference/ui-customization/web-react/project-extension.mjs';
const env='reference/workspace/environments/development/environment.tei';
const app='reference/workspace/applications/order-entry/application.tei';
const old=[{path:ui,gitBlobSha:'a'.repeat(40)},{path:env,gitBlobSha:'b'.repeat(40)},{path:app,gitBlobSha:'c'.repeat(40)}];
const selected=[{id:'ui-presentation',dependencies:[ui]},{id:'database-wiring',dependencies:[env,app]},
  {id:'application-action-ui',dependencies:[ui,app]}];
const report=current=>assessScopedChanges({baseline:old,current,consumers:selected});
const status=x=>Object.fromEntries(x.map(c=>[c.id,c]));

test('unchanged inputs do not cause blanket requalification',()=>{
  const result=report(old);
  assert.deepEqual(result.map(r=>r.status),['unchanged','unchanged','unchanged']);
});
test('UI project overlay drift is scoped; DB wiring remains unchanged',()=>{
  const changed=old.map(o=>o.path===ui?{...o,gitBlobSha:'d'.repeat(40)}:o);
  const p=status(report(changed));
  assert.equal(p['ui-presentation'].reason,'dependency_changed');
  assert.equal(p['database-wiring'].status,'unchanged');
  assert.equal(p['application-action-ui'].reason,'dependency_changed');
});
test('Application meaning drift requalifies only actual consumers',()=>{
  const changed=old.map(o=>o.path===app?{...o,gitBlobSha:'d'.repeat(40)}:o);
  const p=status(report(changed));
  assert.equal(p['ui-presentation'].status,'unchanged');
  assert.deepEqual(p['database-wiring'].changed,[app]);
  assert.equal(p['application-action-ui'].reason,'dependency_changed');
});
test('unrelated source changes are not included in the consumer closure',()=>{
  const changed=[...old,{path:'reference/sui-adoption/workspace/tei.workspace.yaml',gitBlobSha:'0'.repeat(40)}];
  assert.deepEqual(report(changed).map(x=>x.status),['unchanged','unchanged','unchanged']);
});
test('missing source identities fail closed, rather than treating absence as no change',()=>{
  const incomplete=old.filter(o=>o.path!==env);
  const r=status(report(incomplete));
  assert.equal(r['ui-presentation'].status,'unchanged');
  assert.equal(r['database-wiring'].reason,'missing_source_identity');
  assert.deepEqual(r['database-wiring'].missing,[env]);
});
test('duplicate and malformed identities cannot become silent winners',()=>{
  assert.throws(()=>report([...old,old[0]]),/invalid_or_duplicate_source/);
  assert.throws(()=>report(old.map((o,i)=>i?o:{path:'../bad',gitBlobSha:o.gitBlobSha})),/invalid_or_duplicate_source/);
  assert.throws(()=>assessScopedChanges({baseline:old,current:old,
    consumers:[{id:'same',dependencies:[ui]},{id:'same',dependencies:[env]}]}),/invalid_consumer/);
});
test('locally applicable UI edit is not automatic acceptance',()=>{
  const source="React.createElement('h2',null,'Details')";
  const proposal=proposeUIHeading(source);
  assert.deepEqual(assessCandidateApplicability({candidate:proposal,scope:report(old)[0]}),
    {status:'applicable',qualification:'local-edit-only'});
});
test('declared DB target is still unqualified even if source identities did not drift',()=>{
  const source=`database-instance sales_core.dev_v2 {\n definition sales_core\n provider postgres\n }\n bind order_entry.sales -> sales_core.dev`;
  const proposal=proposeDevelopmentDatabaseBinding(source,'sales_core.dev_v2');
  assert.equal(proposal.status,'proposed-unqualified');
  assert.equal(assessCandidateApplicability({candidate:proposal,scope:status(report(old))['database-wiring']}).status,'unresolved');
});
test('changed dependency defeats local applicability without guessing compatibility',()=>{
  const changed=old.map(o=>o.path===ui?{...o,gitBlobSha:'f'.repeat(40)}:o);
  const result=assessCandidateApplicability({candidate:{status:'proposed'},scope:report(changed)[0]});
  assert.equal(result.status,'unresolved');
  assert.equal(result.reason,'dependency_changed');
});
test('UI drift and DB drift remain different scopes',()=>{
  const current=old.map(o=>o.path===env?{...o,gitBlobSha:'4'.repeat(40)}:o);
  const p=status(report(current));
  assert.equal(p['ui-presentation'].status,'unchanged');
  assert.deepEqual(p['database-wiring'].changed,[env]);
});
test('observed E1-to-current change flags Application dependents without invalidating UI-only overlay',()=>{
  // These exact blobs were inspected from the TEI GitHub repository on 2026-10-08/09.
  // This only compares identities; it does not determine which semantic field changed.
  const baseline=[
    {path:ui,gitBlobSha:'c454f31d575be7a99b23fda6dc4955f4f8ada090'},
    {path:env,gitBlobSha:'4ffe7d719dbe8cbbd48b4d713a2c6f57d203c16c'},
    {path:app,gitBlobSha:'b559df7338d590ed173ae83b6a4653fa38770377'},
  ];
  const current=baseline.map(s=>s.path===app?{...s,gitBlobSha:'00c29ac0e557cb50fdcd13f2477d0b538a528c80'}:s);
  const outcome=status(assessScopedChanges({baseline,current,consumers:selected}));
  assert.equal(outcome['ui-presentation'].status,'unchanged');
  assert.deepEqual(outcome['database-wiring'].changed,[app]);
  assert.deepEqual(outcome['application-action-ui'].changed,[app]);
});
