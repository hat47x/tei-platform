import { createHash, createPublicKey, verify as verifySignature } from 'node:crypto';
import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import path from 'node:path';
import { parseFixture } from './wire.mjs';

// Reference-only public keys, NOT production trusted identities.
const publisherPubHex = '03a107bff3ce10be1d70dd18e74bc09967e4d6309ba50d5f1ddc8664125531b8';
const verifierPubHex = '29acbae141bccaf0b22e1a94d34d0bc7361e526d0bfe12c89794bc9322966dd7';
const sha = (bytes) => 'sha256:' + createHash('sha256').update(bytes).digest('hex');
const strictBase64 = (s) => {
  if (typeof s !== 'string' || !/^(?:[A-Za-z0-9+/]{4})*(?:[A-Za-z0-9+/]{2}==|[A-Za-z0-9+/]{3}=)?$/.test(s)) throw new Error('invalid base64');
  const b = Buffer.from(s,'base64');
  if (b.toString('base64') !== s) throw new Error('noncanonical base64');
  return b;
};
const key = (hex) => createPublicKey({key:Buffer.concat([Buffer.from('302a300506032b6570032100','hex'),Buffer.from(hex,'hex')]),format:'der',type:'spki'});
const validateSig = (keyHex, signature, bytes) => {
  if (!verifySignature(null,bytes,key(keyHex),strictBase64(signature))) throw new Error('invalid signature');
};

export function defaultPolicy() {
  return {plugin:'tei:plugin:fixture-alpha',version:'1.0.0',ownerEpoch:3,
    profileDigest:'sha256:ee8702820103a27f205b511d8dfb25c339bddfd55c63326e0fafc2f9effa12ea',target:'linux-amd64',
    fresh:true,publisherTrusted:true,verifierTrusted:true};
}
export function validate(f,p) {
  if (!p.fresh || !p.publisherTrusted || !p.verifierTrusted) throw new Error('trust policy not currently satisfied');
  const lines=f.releaseText.split('\n');
  if (lines.length!==7 || lines[0]!=='TEI-REF-RELEASE-V0' || lines[6]!=='') throw new Error('invalid framing');
  const prefixes=['plugin=','version=','epoch=','artifact.core=','artifact.ui='];
  const vals=prefixes.map((prefix,i) => {
    if (!lines[i+1].startsWith(prefix) || lines[i+1].length<=prefix.length) throw new Error('invalid field');
    return lines[i+1].slice(prefix.length);
  });
  if (vals[0]!==p.plugin || vals[1]!==p.version || vals[2]!==String(p.ownerEpoch)) throw new Error('identity/epoch mismatch');
  const required=new Map([['core',vals[3]],['ui',vals[4]]]);
  if (f.artifacts.length!==required.size) throw new Error('required artifact count mismatch');
  const seen=new Set();
  for (const art of f.artifacts) {
    if (!required.has(art.role) || seen.has(art.role)) throw new Error('unknown or duplicate artifact');
    seen.add(art.role);
    const data=strictBase64(art.bytesBase64);
    if (sha(data)!==required.get(art.role)) throw new Error('artifact digest mismatch');
  }
  const publishBytes=Buffer.concat([Buffer.from('TEI-REF-PUBLISH-V0\0','utf8'),Buffer.from(f.releaseText,'utf8')]);
  validateSig(publisherPubHex,f.publisherSignatureBase64,publishBytes);
  const subject=sha(Buffer.from(f.releaseText,'utf8'));
  const att=f.attestation;
  if (att.subject!==subject) throw new Error('attestation subject mismatch');
  if (sha(Buffer.from(f.profileText,'utf8'))!==att.profileDigest || att.profileDigest!==p.profileDigest) throw new Error('profile mismatch');
  if (att.target!==p.target || att.result!=='passed') throw new Error('scope mismatch');
  const body=['TEI-REF-ATTEST-V0',att.subject,att.profileDigest,att.target,att.result,''].join('\n');
  validateSig(verifierPubHex,att.signatureBase64,Buffer.from(body,'utf8'));
  return true;
}
export function loadFixture(location=new URL('./fixture.json',import.meta.url)) {return parseFixture(readFileSync(location));}
if (process.argv[1] && path.resolve(process.argv[1])===fileURLToPath(import.meta.url)) {
  const f=loadFixture();
  validate(f,defaultPolicy());
  process.stdout.write('PASS reference fixture (not production qualification)\n');
}
