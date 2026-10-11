import test from 'node:test';
import assert from 'node:assert/strict';
import {proposeUIHeading, proposeDevelopmentDatabaseBinding, uiPath, environmentPath} from './e2-change-probe.mjs';

const ui = "export function panel(React) { return React.createElement('h2',null,'Details'); }";
const env = `environment development {
  database-instance sales_core.dev {
    definition sales_core
    provider postgres
  }
  database-instance master_data.dev {
    definition master_data
    provider oracle
  }
  bind order_entry.sales -> sales_core.dev
}`;

test('UI change is a single project-owned heading replacement', () => {
  const got=proposeUIHeading(ui);
  assert.equal(got.status,'proposed');
  assert.equal(got.path,uiPath);
  assert.equal(got.owner,'project-ui');
  assert.equal(got.after,ui.replace("'Details'","'案件の詳細'"));
  assert.ok(got.checks.includes('ui-extension-test'));
});
test('UI heading drift fails closed', () => {
  assert.equal(proposeUIHeading(ui.replace('Details','Details v2')).reason,'ui-anchor-drift');
  assert.equal(proposeUIHeading(ui+ui).reason,'ui-anchor-drift');
});
test('unsafe label cannot inject UI code or newlines', () => {
  assert.equal(proposeUIHeading(ui,"x');alert(1);//").reason,'unsafe-ui-label');
  assert.equal(proposeUIHeading(ui,'line\nbreak').reason,'unsafe-ui-label');
});
test('unmodified binding is not a new change', () => {
  assert.equal(proposeDevelopmentDatabaseBinding(env,'sales_core.dev').status,'unchanged');
});
test('missing database target stays unresolved without automatic creation', () => {
  const got=proposeDevelopmentDatabaseBinding(env,'sales_core.dev_v2');
  assert.equal(got.reason,'instance-not-unique-or-missing');
});
test('cannot silently redirect sales database to master-data instance', () => {
  const got=proposeDevelopmentDatabaseBinding(env,'master_data.dev');
  assert.equal(got.reason,'target-definition-mismatch');
});
test('declared same-definition instance is proposed but never qualified by textual matching', () => {
  const defined=env.replace('  bind order_entry.sales',`  database-instance sales_core.dev_v2 {\n    definition sales_core\n    provider postgres\n  }\n  bind order_entry.sales`);
  const got=proposeDevelopmentDatabaseBinding(defined,'sales_core.dev_v2');
  assert.equal(got.status,'proposed-unqualified');
  assert.equal(got.path,environmentPath);
  assert.match(got.after,/bind order_entry.sales -> sales_core.dev_v2/);
  assert.ok(got.checks.includes('actual-runtime-evidence'));
});
test('duplicate instance and changed binding anchor fail closed', () => {
  const block='database-instance sales_core.dev_v2 {\n definition sales_core\n provider postgres\n }';
  assert.equal(proposeDevelopmentDatabaseBinding(env+block+block,'sales_core.dev_v2').reason,'instance-not-unique-or-missing');
  assert.equal(proposeDevelopmentDatabaseBinding(env.replace('sales_core.dev\n}','elsewhere\n}'),'sales_core.dev_v2').reason,'binding-anchor-drift');
});
test('invalid target identifiers do not enter expressions', () => {
  assert.equal(proposeDevelopmentDatabaseBinding(env,'foo} evil {').reason,'invalid-instance-id');
});
