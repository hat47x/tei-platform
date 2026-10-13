import test from 'node:test';
import assert from 'node:assert/strict';
import {comparePair,CandidateInputError} from './compare-candidates.mjs';

function entry(app, name) {
  return {id:app+'.'+name,application:app,kind:'ui',
    source:{locator:app+'/'+name,digest:'sha256:synthetic',revision:'r1'},
    contract:{id:'std.example',fingerprint:'example-v1'},
    signature:{inputs:[{name:'record',type:'std.record',nullable:false}],outputs:[],effects:[],failures:[]},
    platform:{target:'web',framework:'lit'},labels:['受注','明細']};
}
test('Japanese business terms contribute to low-authority lexical ranking',()=>{
  const result=comparePair(entry('app-a','form'),entry('app-b','form'));
  assert.ok(result.lexicalHint>0);
  assert.equal(result.acceptance,'not_accepted');
});
test('equivalent typed fields ignore object property insertion order',()=>{
  const other=entry('app-b','form');
  other.signature.inputs=[{nullable:false,type:'std.record',name:'record'}];
  assert.equal(comparePair(entry('app-a','form'),other).state,'reuse_candidate');
});
test('duplicate field names are invalid even with matching contract identifiers',()=>{
  const other=entry('app-b','form');
  other.signature.inputs.push({name:'record',type:'std.record',nullable:false});
  assert.throws(()=>comparePair(entry('app-a','form'),other),
    e=>e instanceof CandidateInputError && e.code==='duplicate_field_name');
});
