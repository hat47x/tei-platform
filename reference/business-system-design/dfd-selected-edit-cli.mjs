#!/usr/bin/env node
// The selected DFD edit is still a reference-only review proposal. No writes.
import {readFileSync} from 'node:fs';
import {createHash} from 'node:crypto';
import {previewSelectedDFDEdit} from './dfd-selected-edit-bridge.mjs';

const sha=bytes=>'sha256:'+createHash('sha256').update(bytes).digest('hex');
function args(argv) {
  const options=['-projection','-procedure','-review','-binding'];
  if(argv.length!==8)throw Error('FOUR_INPUT_FILES_REQUIRED');
  const paths={};
  for(let i=0;i<argv.length;i+=2) {
    const key=argv[i],path=argv[i+1];
    if(!options.includes(key)||Object.hasOwn(paths,key)||
        typeof path!=='string'||path.length===0||path.startsWith('-'))
      throw Error('INVALID_DFD_SELECTED_EDIT_ARGUMENTS');
    paths[key]=path;
  }
  if(Object.keys(paths).length!==4)throw Error('FOUR_INPUT_FILES_REQUIRED');
  return paths;
}
function load(path) {
  const bytes=readFileSync(path);
  return {path,hash:sha(bytes),value:JSON.parse(bytes.toString('utf8'))};
}
function run(){
  const paths=args(process.argv.slice(2));
  const projection=load(paths['-projection']),procedure=load(paths['-procedure']);
  const review=load(paths['-review']),binding=load(paths['-binding']);
  if(binding.value.projectionSHA256!==projection.hash||
     binding.value.procedureSHA256!==procedure.hash||
     binding.value.reviewSHA256!==review.hash)
     throw Error('STALE_OR_UNVERIFIED_DFD_EDIT_INPUT_SNAPSHOT');
  const candidate=previewSelectedDFDEdit(projection.value,procedure.value,
     review.value,binding.value);
  for(const f of [projection,procedure,review,binding]) {
    if(sha(readFileSync(f.path))!==f.hash)throw Error('DFD_EDIT_SOURCE_CHANGED_DURING_PREVIEW');
  }
  candidate.inputSnapshotsVerified=true;
  candidate.inputSnapshots={
    projection:projection.hash,procedure:procedure.hash,
    review:review.hash,binding:binding.hash
  };
  // Input byte equality is not Accepted World identity. It is not a
  // Shared Acceptance preview and cannot issue executable authority.
  process.stdout.write(JSON.stringify(candidate,null,2)+'\n');
}
try {run()}catch(err){process.stderr.write(String(err.message)+'\n');process.exitCode=1;}
