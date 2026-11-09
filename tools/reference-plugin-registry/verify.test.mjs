import test from 'node:test';
import assert from 'node:assert/strict';
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
