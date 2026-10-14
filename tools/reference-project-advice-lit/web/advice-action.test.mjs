import test from 'node:test';
import assert from 'node:assert/strict';
import {createAdviceAction} from './advice-action.mjs';

const success = {schema:'tei.reference.project-customer-advice/v0',
  kind:'review_suggested',message:'補助確認',advisory:true};
function makeResponse(body=success, status=200, type='application/json; charset=utf-8') {
  return {status,headers:{get:() => type},json:async () => body};
}

test('advice only sends local project-owned inputs', async () => {
  const calls=[];
  const action=createAdviceAction(async (path,options) => {
    calls.push({path,options});
    return makeResponse();
  });
  assert.deepEqual(await action({customerCode:'VIP-5',note:''}),
    {kind:'review_suggested',message:'補助確認'});
  assert.equal(calls.length,1);
  assert.equal(calls[0].path,'/project/advice');
  assert.deepEqual(JSON.parse(calls[0].options.body),
    {customerCode:'VIP-5',note:''});
  assert.equal(calls[0].options.method,'POST');
  assert.equal(calls[0].options.credentials,'same-origin');
});

test('rejects additional authority or credential fields and invalid shapes', async () => {
  let count=0;
  const action=createAdviceAction(async () => {count++;return makeResponse();});
  for (const input of [{customerCode:'<script>',note:''}, {customerCode:'',note:''},
    {customerCode:'A'}, {customerCode:'A',note:3}]) {
    await assert.rejects(action(input));
  }
  assert.equal(count,0);
});

test('unknown responses and transport failure cannot become advisory success', async () => {
  for (const response of [
    makeResponse(success,403), makeResponse({...success,advisory:false}),
    makeResponse({...success,kind:'completed'}), makeResponse(success,200,'text/html'),
  ]) {
    await assert.rejects(createAdviceAction(async () => response)({customerCode:'A',note:''}));
  }
  await assert.rejects(createAdviceAction(async () => {
    throw Error('connection lost');
  })({customerCode:'A',note:''}), /unknown_result/);
});
