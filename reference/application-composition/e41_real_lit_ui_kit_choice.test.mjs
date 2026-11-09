// E41: #1016 adopts the actual project Lit source only in a design-time
// #997 UI Kit what-if. This is NOT a shipped Lit UI Kit or a Host grant.
import test from 'node:test';
import assert from 'node:assert/strict';
import {readFile} from 'node:fs/promises';
import {createHash} from 'node:crypto';
import {UIKitError,resolveUICompositionKit} from '../ui-customization/composition/resolve-ui-kit.mjs';

const kitVersion='0.1.0';
const target='web';
const lit={id:'lit',major:3}; // illustrative toolkit classification, not an Accepted TEI Kit
const react={id:'react',major:18};
const schemaVersion='tei-ui-composition-reference-v0';
const kind='ui-composition-kit';
const litRoot={
  schemaVersion,kind,id:'reference.web.lit.root',version:kitVersion,target,framework:lit,
  slots:['advice','status'],
  changes:[],
};
const litProject={
  schemaVersion,kind,id:'project.customer-advice.lit',version:'1.0.0',target,framework:lit,
  extends:{id:litRoot.id,version:litRoot.version},
  changes:[{op:'provide',slot:'advice',implementation:'project.lit.advice'}],
};
const reactRoot={
  schemaVersion,kind,id:'reference.web.react.root',version:kitVersion,target,framework:react,
  slots:['advice','status'],changes:[],
};
const projectRender=()=>'<tei-project-customer-advice></tei-project-customer-advice>';
const implementations={'project.lit.advice':projectRender};
const clone=x=>structuredClone(x);
const issueCode=code=>e=>e instanceof UIKitError&&e.code===code;
const resolve=(catalog=[litRoot,litProject],selected={id:litProject.id,version:litProject.version},
  framework=lit,impl=implementations)=>resolveUICompositionKit({
    catalog,implementations:impl,selected,target,framework,
  });
const gitBlob=bytes=>createHash('sha1').update(
  Buffer.concat([Buffer.from('blob '+String(bytes.length)+'\0','utf8'),bytes])
).digest('hex');

test('E41 real main-integrated Lit sources are exact bytes, not inferred from a Kit name',async()=>{
  const files=[
    ['../../tools/reference-project-advice-lit/web/advice-element.mjs',
      '0f415f06f9e325b3d2f544cbbe7836f0c3f3e65f'],
    ['../../tools/reference-project-advice-lit/web/lit-core.min.js',
      '30071d620d392394d56721f05fc4808e02e038e3'],
    ['../../tools/reference-project-advice-lit/web/advice-action.mjs',
      '609e78a87134287380edbb801efe641c6b541d66'],
  ];
  for(const [path,expected] of files) {
    const data=await readFile(new URL(path,import.meta.url));
    assert.equal(gitBlob(data),expected,path+' changed: requalify E41');
  }
  const litSource=await readFile(new URL(files[0][0],import.meta.url),'utf8');
  const transport=await readFile(new URL(files[2][0],import.meta.url),'utf8');
  assert.match(litSource,/customElements\.define\('tei-project-customer-advice'/);
  assert.match(litSource,/from '\.\/lit-core\.min\.js'/);
  assert.match(transport,/fetchImpl\('\/project\/advice'/);
  assert.doesNotMatch(transport,/\/scenario\/submit/);
});

test('E41 explicitly selected project Lit component occupies only the declared UI slot',()=>{
  const before=clone(litRoot);
  const result=resolve();
  assert.deepEqual(result.chain.map(x=>x.id),[litRoot.id,litProject.id]);
  assert.equal(result.selection.target,'web');
  assert.deepEqual(result.selection.framework,lit);
  assert.equal(result.slots.advice({}),projectRender());
  assert.equal(result.slots.status,null);
  assert.deepEqual(result.trace.advice,[{
    kit:litProject.id,version:litProject.version,operation:'provide',
    implementation:'project.lit.advice',
  }]);
  assert.deepEqual(litRoot,before); // no silent mutation or accepted order semantics
  assert.equal(Object.hasOwn(result.slots,'authorization'),false);
  assert.equal(Object.hasOwn(result.slots,'action'),false);
});

test('E41 common Web Target does NOT license inheriting Lit UI from React toolkit',()=>{
  const mixed=clone(litProject);
  mixed.extends={id:reactRoot.id,version:reactRoot.version};
  assert.throws(()=>resolve([reactRoot,mixed]),issueCode('framework_mismatch'));
  assert.throws(()=>resolve([litRoot,litProject],
    {id:litProject.id,version:litProject.version},react),issueCode('framework_mismatch'));
});

test('E41 exact Kit parent revision must be explicitly reselected, never auto-upgraded',()=>{
  const newer=clone(litRoot);
  newer.version='0.2.0';
  assert.throws(()=>resolve([newer,litProject]),issueCode('missing_exact_parent'));
  assert.throws(()=>resolve([litRoot,litProject],
    {id:litProject.id,version:'2.0.0'}),issueCode('kit_not_found'));
});

test('E41 replacing an upstream UI implementation requires its exact old identity',()=>{
  const base=clone(litRoot);
  base.changes=[{op:'provide',slot:'advice',implementation:'kit.default.advice'}];
  const project=clone(litProject);
  project.changes=[{op:'replace',slot:'advice',
    expect:'kit.default.advice',implementation:'project.lit.advice'}];
  const functions={...implementations,'kit.default.advice':()=>'<default />'};
  assert.equal(resolve([base,project],undefined,lit,functions).slots.advice({}),
    projectRender());
  const changedBase=clone(base);
  changedBase.changes[0].implementation='kit.updated.advice';
  const newerFns={...functions,'kit.updated.advice':()=>'<updated />'};
  assert.throws(()=>resolve([changedBase,project],undefined,lit,newerFns),
    issueCode('override_conflict'));
});

test('E41 competing project UI layers never acquire an implicit winner',()=>{
  const another=clone(litProject);
  another.id='another-project.customer-advice.lit';
  another.changes[0].implementation='another-project.lit.advice';
  const f={...implementations,'another-project.lit.advice':()=>'<other />'};
  const selected=resolve([litRoot,litProject,another],{id:litProject.id,version:litProject.version},lit,f);
  assert.equal(selected.slots.advice({}),projectRender());
  const alternate=resolve([litRoot,litProject,another],{id:another.id,version:another.version},lit,f);
  assert.equal(alternate.slots.advice({}),'<other />');
  assert.throws(()=>resolve([litRoot,litProject,clone(litProject)]),issueCode('ambiguous_kit'));
});

test('E41 unknown UI implementation and authorization-like slots fail closed',()=>{
  assert.throws(()=>resolve(undefined,undefined,lit,{}),issueCode('implementation_missing'));
  const rogue=clone(litProject);
  rogue.changes=[{op:'provide',slot:'authorization',implementation:'project.lit.advice'}];
  assert.throws(()=>resolve([litRoot,rogue]),issueCode('invalid_change'));
  const action=clone(litProject);
  action.changes=[{op:'provide',slot:'action',implementation:'project.lit.advice'}];
  assert.throws(()=>resolve([litRoot,action]),issueCode('invalid_change'));
  const undeclared=clone(litProject);
  undeclared.changes=[{op:'provide',slot:'toolbar',implementation:'project.lit.advice'}];
  assert.throws(()=>resolve([litRoot,undeclared]),issueCode('unknown_extension_point'));
});
