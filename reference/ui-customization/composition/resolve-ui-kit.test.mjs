import test from 'node:test';
import assert from 'node:assert/strict';
import {UIKitError, resolveUICompositionKit} from './resolve-ui-kit.mjs';
const env={target:'web',framework:{id:'react',major:18}};
const root={schemaVersion:'tei-ui-composition-reference-v0',kind:'ui-composition-kit',
  id:'tei.ui-kit.web.react.starter',version:'0.1.0',...env,
  slots:['header','aside','footer'],changes:[{op:'provide',slot:'header',implementation:'standard.header'}]};
const company={schemaVersion:root.schemaVersion,kind:root.kind,id:'company.ui.web.react',version:'1.0.0',...env,
  extends:{id:root.id,version:root.version},changes:[{op:'replace',slot:'header',expect:'standard.header',implementation:'company.header'}]};
const project={schemaVersion:root.schemaVersion,kind:root.kind,id:'project.ui.web.react',version:'1.0.0',...env,
  extends:{id:company.id,version:company.version},changes:[
    {op:'decorate',slot:'header',expect:'company.header',implementation:'project.decorateHeader'},
    {op:'provide',slot:'aside',implementation:'project.aside'}]};
const implementations={
  'standard.header':()=>'<standard />',
  'company.header':()=>'<company />',
  'project.decorateHeader':(next,ctx)=>`<project>${next(ctx)}</project>`,
  'project.aside':()=>'<project-aside />',
};
const clone = v => structuredClone(v);
function resolve(options={}){return resolveUICompositionKit({catalog:[root,company,project],implementations,
  selected:{id:project.id,version:project.version},...env,...options});}
function hasCode(code){return e=>e instanceof UIKitError&&e.code===code;}

test('three-level root->company->project: explicit replace, decorate, contribute and provenance',()=>{
  const r=resolve();
  assert.deepEqual(r.chain.map(x=>x.id),[root.id,company.id,project.id]);
  assert.equal(r.slots.header({}),'<project><company /></project>');
  assert.equal(r.slots.aside({}),'<project-aside />');
  assert.equal(r.slots.footer,null);
  assert.deepEqual(r.trace.header.map(x=>x.operation),['provide','replace','decorate']);
});
test('base kit remains unchanged and ordinary delegation retains original action outside UI',()=>{
  const old=clone(root);
  const r=resolve();
  assert.deepEqual(root,old);
  assert.equal(r.selection.target,'web');
  assert.deepEqual(Object.keys(r.slots),['header','aside','footer']);
  assert.equal(Object.hasOwn(r.slots,'action.invoke'),false);
});
test('version-pinned parent must exist: no silent minor or major fallback',()=>{
  const newer=clone(root);newer.version='0.2.0';
  assert.throws(()=>resolve({catalog:[newer,company,project]}),hasCode('missing_exact_parent'));
});
test('overriding other than expected inherited implementation fails closed',()=>{
  const conflict=clone(project);conflict.changes[0].expect='standard.header';
  assert.throws(()=>resolve({catalog:[root,company,conflict]}),hasCode('override_conflict'));
});
test('incompatible Web/React and Mobile/React Native cannot share UI implementation inheritance',()=>{
  const mobile=clone(project);mobile.target='mobile';mobile.framework={id:'react-native',major:1};
  assert.throws(()=>resolve({catalog:[root,company,mobile],target:'mobile',framework:{id:'react-native',major:1}}),hasCode('framework_mismatch'));
});
test('cycle and duplicate manifest identities are rejected',()=>{
  const a=clone(root);a.extends={id:project.id,version:project.version};delete a.slots;
  assert.throws(()=>resolve({catalog:[a,company,project]}),hasCode('inheritance_cycle'));
  assert.throws(()=>resolve({catalog:[root,root,company,project]}),hasCode('ambiguous_kit'));
});
test('no unknown extension point may override privileged action, authority, domain validation',()=>{
  for(const slot of ['action.invoke','permission','validation','semantic.order']) {
    const rogue=clone(project);rogue.changes=[{op:'provide',slot,implementation:'project.aside'}];
    assert.throws(()=>resolve({catalog:[root,company,rogue]}),hasCode('invalid_change'));
  }
});
test('double claim to same slot in one project layer is rejected regardless of enumeration order',()=>{
  const dup=clone(project);dup.changes.push({op:'replace',slot:'aside',expect:'project.aside',implementation:'project.aside'});
  assert.throws(()=>resolve({catalog:[root,company,dup]}),hasCode('invalid_change'));
});
test('providing existing slot without explicit replace fails, regardless of package naming',()=>{
  const bad=clone(company);bad.changes=[{op:'provide',slot:'header',implementation:'company.header'}];
  assert.throws(()=>resolve({catalog:[root,bad,project]}),hasCode('slot_already_provided'));
});
test('unknown or missing implementation fails at assembly, not as silent fallback',()=>{
  const incomplete={...implementations};delete incomplete['project.aside'];
  assert.throws(()=>resolve({implementations:incomplete}),hasCode('implementation_missing'));
});
test('unavailable selected kit fails instead of selecting another compatible candidate',()=>{
  assert.throws(()=>resolve({selected:{id:project.id,version:'2.0.0'}}),hasCode('kit_not_found'));
});
test('new visual extension point can be declared by a trusted UI host without Core enumeration',()=>{
  const base=clone(root);base.slots.push('floating-toolbar');
  const custom=clone(project);custom.changes.push({op:'provide',slot:'floating-toolbar',implementation:'project.toolbar'});
  const impl={...implementations,'project.toolbar':()=>'<tools />'};
  const r=resolve({catalog:[base,company,custom],implementations:impl});
  assert.equal(r.slots['floating-toolbar']({}),'<tools />');
});
test('unknown extension point fails rather than creating an undeclared UI behavior',()=>{
  const p=clone(project);p.changes.push({op:'provide',slot:'unregistered',implementation:'project.aside'});
  assert.throws(()=>resolve({catalog:[root,company,p]}),hasCode('unknown_extension_point'));
});
