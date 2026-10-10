import test from "node:test";
import assert from "node:assert/strict";
import {inspectCapabilityInputTypes} from "./e8-typed-closure.mjs";
const world = {applications:[{id:"app",patterns:[{capability:"cap"}]}],semantics:[{capabilities:[{id:"cap",inputs:[{name:"x",type:"type-a"}]}],types:[{id:"type-a"},{id:"type-b"}]}]};
const index = {references:[{subject:{kind:"semantic-type",id:"type-a"},role:"definition",artifactPath:"a.yaml"},{subject:{kind:"semantic-type",id:"type-b"},role:"definition",artifactPath:"b.yaml"}],artifacts:[{path:"a.yaml",fingerprint:"sha256:old"},{path:"b.yaml",fingerprint:"sha256:other"}]};
test("reachable input type change is observed",()=>{const changed=structuredClone(index);changed.artifacts[0].fingerprint="sha256:new";const before=inspectCapabilityInputTypes(world,index,"app"),after=inspectCapabilityInputTypes(world,changed,"app");assert.equal(before.status,"observed-limited");assert.notDeepEqual(before.dependencies,after.dependencies);});
test("unrelated type change does not affect result",()=>{const changed=structuredClone(index);changed.artifacts[1].fingerprint="sha256:new";assert.deepEqual(inspectCapabilityInputTypes(world,index,"app"),inspectCapabilityInputTypes(world,changed,"app"));});
test("ambiguous type fails closed",()=>{const changed=structuredClone(world);changed.semantics[0].types.push({id:"type-a"});assert.equal(inspectCapabilityInputTypes(changed,index,"app").status,"unresolved");});
test("missing source evidence fails closed",()=>{assert.equal(inspectCapabilityInputTypes(world,{artifacts:[],references:[]},"app").status,"unresolved");});
