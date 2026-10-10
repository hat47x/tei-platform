import test from "node:test";
import assert from "node:assert/strict";
import {inspectCapabilityInputTypes} from "./e8-typed-closure.mjs";
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
