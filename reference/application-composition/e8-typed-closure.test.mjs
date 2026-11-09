import test from "node:test";
import assert from "node:assert/strict";
import {inspectCapabilityInputTypes, compareCapabilityInputDefinitionSources} from "./e8-typed-closure.mjs";
const sha = c => 'sha256:' + c.repeat(64);
const world = {applications:[{id:"app",patterns:[{capability:"cap"}]}],semantics:[{capabilities:[{id:"cap",inputs:[{name:"x",type:"type-a"}]}],types:[{id:"type-a"},{id:"type-b"}]}]};
const index = {references:[{subject:{kind:"semantic-type",id:"type-a"},role:"definition",artifactPath:"a.yaml"},{subject:{kind:"semantic-type",id:"type-b"},role:"definition",artifactPath:"b.yaml"}],artifacts:[{path:"a.yaml",fingerprint:sha('a')},{path:"b.yaml",fingerprint:sha('b')}]};
test("reachable input type change is observed",()=>{const changed=structuredClone(index);changed.artifacts[0].fingerprint=sha('c');const before=inspectCapabilityInputTypes(world,index,"app"),after=inspectCapabilityInputTypes(world,changed,"app");assert.equal(before.status,"observed-limited");assert.notDeepEqual(before.dependencies,after.dependencies);});
test("unrelated type change does not affect result",()=>{const changed=structuredClone(index);changed.artifacts[1].fingerprint=sha('c');assert.deepEqual(inspectCapabilityInputTypes(world,index,"app"),inspectCapabilityInputTypes(world,changed,"app"));});
test("ambiguous type fails closed",()=>{const changed=structuredClone(world);changed.semantics[0].types.push({id:"type-a"});assert.equal(inspectCapabilityInputTypes(changed,index,"app").status,"unresolved");});
test("missing source evidence fails closed",()=>{assert.equal(inspectCapabilityInputTypes(world,{artifacts:[],references:[]},"app").status,"unresolved");});

test("no capability references are not mistaken for observed closure",()=>{const changed=structuredClone(world);changed.applications[0].patterns=[];assert.equal(inspectCapabilityInputTypes(changed,index,"app").status,"no-capability-references");});
test("missing input type fails closed",()=>{const changed=structuredClone(world);changed.semantics[0].capabilities[0].inputs[0].type="";assert.equal(inspectCapabilityInputTypes(changed,index,"app").status,"unresolved");});

test("malformed fingerprint cannot be treated as a valid observed type source",()=>{
 const changed=structuredClone(index);changed.artifacts[0].fingerprint='sha256:not-a-sha256';
 const result=inspectCapabilityInputTypes(world,changed,'app');
 assert.equal(result.status,'unresolved');
 assert.ok(result.issues.includes('invalid_source_index'));
});
test("malformed unrelated source artifact also fails the existing E4 index gate",()=>{
 const changed=structuredClone(index);changed.artifacts[1].fingerprint='sha256:bad';
 assert.equal(inspectCapabilityInputTypes(world,changed,'app').status,'unresolved');
});
test("escaping an indexed path cannot be accepted as type evidence",()=>{
 const changed=structuredClone(index);
 changed.references[0].artifactPath='../outside.tei';
 changed.artifacts[0].path='../outside.tei';
 assert.equal(inspectCapabilityInputTypes(world,changed,'app').status,'unresolved');
});
test("duplicate Artifact path fails closed even when fingerprints agree",()=>{
 const changed=structuredClone(index);changed.artifacts.push({...changed.artifacts[0]});
 assert.equal(inspectCapabilityInputTypes(world,changed,'app').status,'unresolved');
});
test("multiple valid capability inputs using the same Type share one dependency",()=>{
 const changed=structuredClone(world);
 changed.semantics[0].capabilities[0].inputs.push({name:'y',type:'type-a'});
 const result=inspectCapabilityInputTypes(changed,index,'app');
 assert.equal(result.status,'observed-limited');
 assert.equal(result.dependencies.length,1);
});
test("invalid or absent input shape is diagnosed without throwing",()=>{
 for(const [input,idx] of [[null,index],[{},index],[world,null],[world,{}]]) {
  assert.equal(inspectCapabilityInputTypes(input,idx,'app').status,'unresolved');
 }
});

test("capability without inputs is not evidence of any reached Type definition",()=>{
 const changed=structuredClone(world);changed.semantics[0].capabilities[0].inputs=[];
 const result=inspectCapabilityInputTypes(changed,index,'app');
 assert.equal(result.status,'no-input-types');
 assert.deepEqual(result.dependencies,[]);
});
test("malformed World and Index members return unresolved rather than crash",()=>{
 const cases=[
  [{...world,applications:[null]},index],
  [{...world,applications:[{id:'app',patterns:[null]}]},index],
  [{...world,semantics:[null]},index],
  [{...world,semantics:[{capabilities:[null],types:[]}]},index],
  [{...world,semantics:[{capabilities:[{id:'cap',inputs:[null]}],types:[]}]},index],
  [world,{...index,artifacts:[null]}],
  [world,{...index,references:[null]}],
 ];
 for (const [typed,source] of cases) {
  const result=inspectCapabilityInputTypes(typed,source,'app');
  assert.equal(result.status,'unresolved');
  assert.ok(result.issues.includes('invalid_snapshot_shape'));
 }
});

test("actual order_entry capability input is Semantic Data, not a Semantic Type",()=>{
 // Identifiers mirror the real reference/workspace semantic/sales/order.tei:
 // capability sales.order.submit { input order sales.order }
 const real={applications:[{id:'order_entry',patterns:[{
   id:'order_entry.submit_order',capability:'sales.order.submit',
 }]}],semantics:[{id:'sales',
   capabilities:[{id:'sales.order.submit',inputs:[{name:'order',type:'sales.order'}]}],
   data:[{id:'sales.order'}],types:[{id:'sales.order_id'}],
 }]};
 const artifact='semantic/sales/order.tei';
 const src={artifacts:[{path:artifact,fingerprint:sha('d')}],
   references:[{subject:{kind:'semantic-data',id:'sales.order'},
    role:'definition',artifactPath:artifact}]};
 const result=inspectCapabilityInputTypes(real,src,'order_entry');
 assert.equal(result.status,'observed-limited');
 assert.deepEqual(result.dependencies,[{
   subject:{kind:'semantic-data',id:'sales.order'},path:artifact,fingerprint:sha('d'),
 }]);
});
test("changing the reachable Data Source evidence changes the observed dependency",()=>{
 const real={applications:[{id:'app',patterns:[{capability:'submit'}]}],
   semantics:[{capabilities:[{id:'submit',inputs:[{name:'order',type:'sales.order'}]}],
     data:[{id:'sales.order'}]}]};
 const src={artifacts:[{path:'sales/order.tei',fingerprint:sha('a')}],
   references:[{subject:{kind:'semantic-data',id:'sales.order'},
     role:'definition',artifactPath:'sales/order.tei'}]};
 const changed=structuredClone(src);changed.artifacts[0].fingerprint=sha('c');
 const before=inspectCapabilityInputTypes(real,src,'app');
 const after=inspectCapabilityInputTypes(real,changed,'app');
 assert.equal(before.status,'observed-limited');
 assert.equal(after.status,'observed-limited');
 assert.notDeepEqual(before.dependencies,after.dependencies);
});
test("duplicate Data identities do not resolve by declaration order",()=>{
 const changed=structuredClone(world);
 changed.semantics[0].capabilities[0].inputs[0].type='sales.order';
 changed.semantics[0].data=[{id:'sales.order'},{id:'sales.order'}];
 assert.equal(inspectCapabilityInputTypes(changed,index,'app').status,'unresolved');
});
test("cross-category collisions cannot silently favor Type over Data",()=>{
 const changed=structuredClone(world);
 changed.semantics[0].data=[{id:'type-a'}];
 const result=inspectCapabilityInputTypes(changed,index,'app');
 assert.equal(result.status,'unresolved');
 assert.ok(result.issues.includes('input_definition_missing_or_ambiguous:type-a'));
});
test("Data Source without a unique definition remains unresolved",()=>{
 const changed=structuredClone(world);
 changed.semantics[0].capabilities[0].inputs[0].type='sales.order';
 changed.semantics[0].data=[{id:'sales.order'}];
 assert.equal(inspectCapabilityInputTypes(changed,index,'app').status,'unresolved');
});
test("malformed semantic Data declarations fail closed",()=>{
 const changed=structuredClone(world);
 changed.semantics[0].data=[null];
 const result=inspectCapabilityInputTypes(changed,index,'app');
 assert.equal(result.status,'unresolved');
 assert.ok(result.issues.includes('invalid_snapshot_shape'));
});

const compare = ({baselineWorld=world,currentWorld=world,baselineIndex=index,currentIndex=index,
  applicationId='app'}={})=>compareCapabilityInputDefinitionSources({
  baselineWorld,currentWorld,baselineIndex,currentIndex,applicationId,
});
test("directly selected input definition Source unchanged returns limited-only verdict",()=>{
 const result=compare();
 assert.equal(result.status,'limited-unchanged');
 assert.equal(result.dependencyCount,1);
 assert.equal(result.fullClosure,false);
 assert.equal(result.authority,'none');
});
test("changing only the reached input Type Source requires requalification",()=>{
 const changed=structuredClone(index);
 changed.artifacts[0].fingerprint=sha('c');
 const result=compare({currentIndex:changed});
 assert.equal(result.status,'unresolved');
 assert.equal(result.reason,'input_definition_source_changed');
 assert.equal(result.affected.length,1);
 assert.equal(result.affected[0].reason,'source_fingerprint_changed');
});
test("an unrelated valid Type Source change does not invalidate this narrow consumer",()=>{
 const changed=structuredClone(index);
 changed.artifacts[1].fingerprint=sha('c');
 assert.equal(compare({currentIndex:changed}).status,'limited-unchanged');
});
test("a Capability switching to another existing input Type changes the selected set",()=>{
 const changed=structuredClone(world);
 changed.semantics[0].capabilities[0].inputs[0].type='type-b';
 const result=compare({currentWorld:changed});
 assert.equal(result.status,'unresolved');
 assert.equal(result.reason,'input_definition_set_changed');
});
test("missing selected current Source evidence never becomes unchanged",()=>{
 const changed=structuredClone(index);
 changed.references.splice(0,1);
 const result=compare({currentIndex:changed});
 assert.equal(result.status,'unresolved');
 assert.equal(result.reason,'incomplete_input_definition_observation');
 assert.equal(result.currentStatus,'unresolved');
});
test("no input signature or no referenced Capability cannot create closure",()=>{
 const empty=structuredClone(world);
 empty.semantics[0].capabilities[0].inputs=[];
 assert.equal(compare({currentWorld:empty}).reason,'incomplete_input_definition_observation');
 const absent=structuredClone(world);
 absent.applications[0].patterns=[];
 assert.equal(compare({currentWorld:absent}).status,'unresolved');
});
test("selected Semantic Data input Source drift is detected through E4",()=>{
 const dataWorld=structuredClone(world);
 dataWorld.semantics[0].capabilities[0].inputs[0].type='sales.order';
 dataWorld.semantics[0].data=[{id:'sales.order'}];
 const dataIndex={artifacts:[{path:'semantic/sales/order.tei',fingerprint:sha('d')}],
  references:[{subject:{kind:'semantic-data',id:'sales.order'},
   artifactPath:'semantic/sales/order.tei',role:'definition'}]};
 const changed=structuredClone(dataIndex);
 changed.artifacts[0].fingerprint=sha('e');
 const result=compare({baselineWorld:dataWorld,currentWorld:dataWorld,
   baselineIndex:dataIndex,currentIndex:changed});
 assert.equal(result.status,'unresolved');
 assert.equal(result.reason,'input_definition_source_changed');
 assert.equal(result.affected[0].subject.kind,'semantic-data');
});
