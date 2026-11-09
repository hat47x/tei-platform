import assert from 'node:assert/strict';
import test from 'node:test';
import {validateRouteTable} from './route-table.mjs';

const clone=value=>structuredClone(value);

function action(id='sales.submit',path='/scenario/session-submit'){
  return {
    route:{id,method:'POST',path,exposure:'protected',actionId:'order_entry.submit_order'},
    hostPolicy:{
      id:'interactive-write',
      requiredGuards:['authenticated','csrf-protected','action-authorized'],
      terminalEvents:['completed','rejected','error']
    },
    hostContracts:['action.identity'],
    steps:[
      {id:'context',scope:'server',provides:['request.context']},
      {id:'identity',scope:'server',requires:['request.context'],provides:['identity.principal'],guards:['authenticated']},
      {id:'csrf',scope:'server',requires:['request.context'],provides:['csrf.checked'],guards:['csrf-protected']},
      {id:'authorize',scope:'server',requires:['identity.principal','action.identity'],provides:['action.authorized'],guards:['action-authorized']},
      {id:'invoke',scope:'server',requires:['action.authorized','csrf.checked']},
      {id:'audit',scope:'server',requires:['request.context'],onTerminal:['completed','rejected','error']}
    ]
  };
}

function callback(id='corporate.callback',path='/login/callback/corporate',method='GET'){
  return {
    route:{id,method,path,exposure:'public'},
    hostPolicy:{id:'oidc-callback',requiredGuards:[],terminalEvents:['completed','rejected','error']},
    hostContracts:[],
    steps:[
      {id:'context',scope:'server',provides:['request.context']},
      {id:'callback-check',scope:'server',requires:['request.context'],provides:['callback.verified']},
      {id:'audit',scope:'server',requires:['request.context'],onTerminal:['completed','rejected','error']}
    ]
  };
}

test('two protected Action routes and an explicit public callback have distinct matches',()=>{
  const routes=[action(),action('sales.other','/scenario/other'),callback()];
  const result=validateRouteTable(routes);
  assert.equal(result.routeCount,3);
  assert.deepEqual(result.routes.map(r=>r.routeId),['corporate.callback','sales.other','sales.submit']);
  assert.deepEqual(result,validateRouteTable([...routes].reverse()));
  assert.equal(result.routes[1].evidence.requiredGuards.includes('action-authorized'),true);
  assert.equal(result.routes[0].actionId,null);
});

test('an identical method and path cannot be owned by different route identities',()=>{
  const a=action();
  const b=action('malicious.shadow');
  assert.throws(()=>validateRouteTable([a,b]),/duplicate or overlapping route match: POST/);
  assert.throws(()=>validateRouteTable([b,a]),/duplicate or overlapping route match: POST/);
});

test('duplicate route identities cannot name different paths',()=>{
  assert.throws(
    ()=>validateRouteTable([action(),action('sales.submit','/another')]),
    /duplicate route identity: sales.submit/
  );
});

test('GET and HEAD cannot silently share the same path under net\/http match semantics',()=>{
  const get=callback('callback.get','/login/callback');
  const head=callback('callback.head','/login/callback','HEAD');
  assert.throws(()=>validateRouteTable([get,head]),/overlapping route match: HEAD/);
  assert.throws(()=>validateRouteTable([head,get]),/overlapping route match: HEAD/);
});

test('a path on different methods is distinct when GET/HEAD overlap is absent',()=>{
  const a=action('submit','/mixed');
  const get=callback('view','/mixed');
  assert.equal(validateRouteTable([a,get]).routeCount,2);
});

test('host and other unimplemented selectors must not be silently ignored',()=>{
  for(const [key,value] of [
    ['host','admin.example.invalid'],
    ['origin','https://example.invalid'],
    ['tenant','tenant-a'],
    ['headers',{'X-Tenant':'tenant-a'}],
    ['query',{admin:'true'}],
    ['scheme','https'],
    ['forwardedHost','internal.invalid']
  ]){
    const p=action();
    p.route[key]=value;
    assert.throws(()=>validateRouteTable([p]),/unsupported route match selector: /,key);
  }
});

test('the table never relaxes the protected Action guard checks',()=>{
  const p=action();
  p.steps.find(x=>x.id==='csrf').guards=[];
  assert.throws(()=>validateRouteTable([p]),/required host protection missing|prerequisite of Action/);
  const q=action();
  q.route.method='GET';
  assert.throws(()=>validateRouteTable([q]),/protected Action reference requires POST/);
});

test('malformed table inputs fail closed',()=>{
  for(const input of [undefined,null,{},[],false]){
    assert.throws(()=>validateRouteTable(input),/at least one request plan/);
  }
  for(const bad of [null,[],{},'route']){
    assert.throws(()=>validateRouteTable([{route:bad}]),/explicit route object|nonempty string/);
  }
  const q=action();
  q.route.id=' ';
  assert.throws(()=>validateRouteTable([q]),/route identity must be a nonempty string/);
});

test('validation does not mutate plans, including their original order',()=>{
  const routes=[action(),callback()];
  const original=clone(routes);
  validateRouteTable(routes);
  assert.deepEqual(routes,original);
});
