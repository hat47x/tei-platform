import test from 'node:test';
import assert from 'node:assert/strict';
import {createOrderEntryRead} from './order-entry-context.mjs';

function response(body, status=200, type='application/json') {
  return {status, headers:{get:()=>type}, json:async()=>body};
}
const good={schema:'tei.reference.order-entry-context/v0',
  applicationId:'order_entry',scenarioId:'read-reference',selectedRowIndex:1,
  targetBuildRevision:'synthetic-T'};

test('read-only adapter does not send credentials or commands', async () => {
  const calls=[];
  const read=createOrderEntryRead(async (path,opts)=>{
    calls.push({path,opts});
    return response(good);
  });
  assert.deepEqual(await read(),{applicationId:'order_entry',
    scenarioId:'read-reference',selectedRowIndex:1});
  assert.equal(calls.length,1);
  assert.equal(calls[0].path,'/project/order-entry-context');
  assert.equal(calls[0].opts.method,'GET');
  assert.equal(calls[0].opts.credentials,'omit');
  assert.equal(calls[0].opts.body,undefined);
});

test('never treats an unknown origin, schema or application as read success', async()=>{
  for(const candidate of [
    response({...good,applicationId:'repair_intake'}),
    response({...good,scenarioId:''}),
    response({...good,targetBuildRevision:''}),
    response({...good,selectedRowIndex:-1}),
    response({...good,schema:'other'}),
    response(good,200,'text/html'),response(good,403),
  ]) {
    await assert.rejects(createOrderEntryRead(async()=>candidate)());
  }
  await assert.rejects(createOrderEntryRead(async()=>{throw Error('offline')})(),
    /unknown_read/);
});
