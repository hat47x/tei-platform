// Read-only observation of a deliberately narrow TEI pattern source subset.
// Not the TEI parser. Does not infer or authorize Canonical/Accepted semantics.
import {createHash} from 'node:crypto';

export class ObservationError extends Error {
  constructor(code) { super(code); this.code = code; }
}

export function gitBlobSha(content) {
  if (typeof content !== 'string') throw new ObservationError('invalid_source');
  return createHash('sha1').update('blob ' + Buffer.byteLength(content) + '\0').update(content).digest('hex');
}

export function observeTeiPattern({application, sourcePath, sourceText, expectedBlobSha}) {
  if (![application, sourcePath, sourceText, expectedBlobSha].every(x => typeof x === 'string' && x.length > 0)) {
    throw new ObservationError('invalid_input');
  }
  if (!/^[a-f0-9]{40}$/.test(expectedBlobSha)) throw new ObservationError('invalid_blob_sha');
  if (gitBlobSha(sourceText) !== expectedBlobSha) throw new ObservationError('blob_mismatch');
  const raw = /^\s*pattern\s+([\w.-]+)\s+(command|header-detail|reference)\s*\{([\s\S]*)\}\s*$/.exec(sourceText);
  const source = {locator:sourcePath,digest:expectedBlobSha};
  const base = {application,source,authority:'source_observation_only',acceptance:'unknown',contract:null};
  if (!raw) return {...base,status:'partial',pattern:null,diagnostics:['unsupported_pattern_syntax']};
  const [,id,kind,contents] = raw;
  const bindingKind = kind === 'command' ? 'capability' : 'relation';
  const single = new RegExp('^\\s*' + bindingKind + '\\s+([\\w.-]+)\\s*$').exec(contents);
  if (!single) return {...base,status:'partial',pattern:{id,kind,binding:null},diagnostics:['unsupported_or_additional_body']};
  return {...base,status:'observed',pattern:{id,kind,binding:{kind:bindingKind,reference:single[1]}},diagnostics:[]};
}
