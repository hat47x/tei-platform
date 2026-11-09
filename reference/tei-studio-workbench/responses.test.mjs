import test from 'node:test';
import assert from 'node:assert/strict';
import { createResponseScope } from './responses.ts';
const deferred = () => { let resolve, reject; const promise = new Promise((yes,no) => { resolve=yes; reject=no; }); return {promise,resolve,reject}; };
test('old workspace success and failure never overwrite the new view', async () => {
  const scope=createResponseScope(), old=deferred(), failed=deferred(), shown=[];
  const first=scope.run('catalog',()=>old.promise,v=>shown.push(v),e=>shown.push(e.message));
  const second=scope.run('inspection',()=>failed.promise,v=>shown.push(v),e=>shown.push(e.message));
  scope.reset();
  await scope.run('catalog',async()=> 'new workspace',v=>shown.push(v),e=>shown.push(e.message));
  old.resolve('old catalog'); failed.reject(Error('old failure'));
  assert.deepEqual(await Promise.all([first,second]),[false,false]);
  assert.deepEqual(shown,['new workspace']);
});
test('latest validation wins in the same workspace, including a late error', async () => {
  const scope=createResponseScope(), old=deferred(), shown=[];
  const first=scope.run('validation',()=>old.promise,v=>shown.push(v),e=>shown.push(e.message));
  await scope.run('validation',async()=> ['current warning'],v=>shown.push(v),e=>shown.push(e.message));
  old.reject(Error('old validation failed'));
  assert.equal(await first,false); assert.deepEqual(shown,[['current warning']]);
});
test('independent lanes publish, current failures remain visible, disposal drops results', async () => {
  const scope=createResponseScope(), pending=deferred(), shown=[];
  await scope.run('inspection',async()=> 'model',v=>shown.push(v),e=>shown.push(e.message));
  await scope.run('validation',async()=> { throw Error('current failure'); },v=>shown.push(v),e=>shown.push(e.message));
  const run=scope.run('inspection',()=>pending.promise,v=>shown.push(v),e=>shown.push(e.message));
  scope.dispose(); pending.resolve('disposed'); assert.equal(await run,false);
  assert.deepEqual(shown,['model','current failure']);
});
