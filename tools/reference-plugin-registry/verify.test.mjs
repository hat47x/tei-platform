import test from 'node:test';
import assert from 'node:assert/strict';
import { createHash } from 'node:crypto';
import { readFileSync } from 'node:fs';
import { parseFixture } from './wire.mjs';
import { loadFixture, defaultPolicy, validate } from './verify.mjs';
const fixture=loadFixture();
test('positive exact release and two-artifact binding',()=>{
  assert.equal(validate(fixture,defaultPolicy()),true);
});
const cases={
  'tamper artifact':(f,p)=>{f.artifacts[0].bytesBase64='dGFtcGVyZWQ=';},
  'missing member':(f,p)=>{f.artifacts.pop();},
  'malformed base64':(f,p)=>{f.artifacts[0].bytesBase64='%%%';},
  'duplicate member':(f,p)=>{f.artifacts[1].role='core';},
  'tamper release':(f,p)=>{f.releaseText+='extra\n';},
  'wrong subject':(f,p)=>{f.attestation.subject='sha256:bad';},
  'wrong target':(f,p)=>{f.attestation.target='wasm32-wasi';},
  'tamper profile':(f,p)=>{f.profileText+='altered';},
  'revoked publisher':(f,p)=>{p.publisherTrusted=false;},
  'revoked verifier':(f,p)=>{p.verifierTrusted=false;},
  'stale policy':(f,p)=>{p.fresh=false;},
  'different owner epoch':(f,p)=>{p.ownerEpoch++;},
  'forged publisher signature':(f,p)=>{f.publisherSignatureBase64='dGVzdA==';},
  'forged verifier signature':(f,p)=>{f.attestation.signatureBase64='dGVzdA==';}
};
for (const [name,change] of Object.entries(cases)) test('reject '+name,()=>{
  const f=structuredClone(fixture),p=defaultPolicy();
  change(f,p);
  assert.throws(()=>validate(f,p));
});

const rawFixture=readFileSync(new URL('./fixture.json',import.meta.url),'utf8');
const wireCases=JSON.parse(readFileSync(new URL('./wire-cases.json',import.meta.url),'utf8')).cases;
for (const c of wireCases) test('shared-wire '+c.id,()=>{
  if (c.find) assert.equal(rawFixture.split(c.find).length-1,1,'mutation anchor must occur once');
  const wire=c.find ? rawFixture.replace(c.find,c.replace) : rawFixture;
  assert.equal(createHash('sha256').update(Buffer.from(wire,'utf8')).digest('hex'),c.sha256,'same exact raw bytes');
  let accepted=false;
  try { accepted=validate(parseFixture(wire),defaultPolicy())===true; } catch { /* expected failure */ }
  assert.equal(accepted,c.accept,'both readers must agree with independent fixture expectation');
});
