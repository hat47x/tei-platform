// Read-only, narrow Reference observation of Environment routing toward a Contract Query.
// This is NOT executor resolution, Provider execution or authority to access any datasource.
import {proposeReferenceQueryAdapter} from './compare-two-application-references.mjs';
import {qualifyReferenceQuerySignature} from './qualify-query-signature.mjs';

export class QueryProviderRoutingError extends Error {
  constructor(code){super(code);this.code=code;}
}
const fail=code=>{throw new QueryProviderRoutingError(code);};
const obj=x=>x!==null&&typeof x==='object'&&!Array.isArray(x);
const nonempty=x=>typeof x==='string'&&x.trim().length>0;
const shape=(x,fields)=>obj(x)&&Object.keys(x).every(k=>fields.includes(k));
const providers=new Set(['generic','postgres','oracle','mysql','mariadb',
  'sqlserver','sqlite','db2','clickhouse']);
function selectOne(items,pred,code) {
  if(!Array.isArray(items))fail(code);
  const selected=items.filter(pred);
  if(selected.length>1)fail(code);
  return selected[0]||null;
}
function sameTarget(x,target) {
  return obj(x)&&x.database===target.database&&
    x.contract===target.contract&&x.query===target.query;
}

/** Fixed authorized Reference case only, using Go M0 World and 4 cli.inspect reports.
 * Missing optional runtime selection is observed as such, not made mandatory.
 * Never exposes this in-memory structure as a generic customer-Workspace JSON export.
 */
export function observeReferenceProviderRouting(world,reports,{environmentId='development'}={}) {
  if(!obj(world)||!nonempty(environmentId)||!Array.isArray(world.environments))
    fail('INVALID_ENVIRONMENT_OBSERVATION');
  // Checks actual Go Query Contract signature and overlapping App/Pattern Canonical.
  const typed=qualifyReferenceQuerySignature(world,reports);
  if(typed.status!=='declared_signature_matched')
    return {schemaVersion:'tei.research.query-provider-route/v0',
      status:'query_signature_not_qualified',routes:[],accepted:false,
      executionPermitted:false,authority:'derived_observation_only'};
  const proposal=proposeReferenceQueryAdapter(reports);
  const target={database:proposal.commonInvocation.databaseTarget,
    contract:typed.contract.id,query:typed.contract.query};
  const env=selectOne(world.environments,e=>e?.id===environmentId,'ENVIRONMENT_NOT_UNIQUE');
  if(!env)fail('ENVIRONMENT_NOT_FOUND');
  // These are typed Environment fields, and further unrelated metadata is not read.
  if(!Array.isArray(env.bindings)||!Array.isArray(env.instances))
    fail('INCOMPLETE_ENVIRONMENT_ROUTING');
  const bindings=world.contractQueryExecutorBindings??[];
  if(!Array.isArray(bindings))fail('INVALID_EXECUTOR_BINDINGS');
  const routes=typed.applicationContext.map(ctx=>{
    const route=selectOne(env.bindings,b=>b?.application===ctx.application&&
      b?.alias===ctx.datasourceAlias,'AMBIGUOUS_APPLICATION_INSTANCE_BINDING');
    if(!route)return {application:ctx.application,alias:ctx.datasourceAlias,
      status:'application_binding_not_declared',instance:null,provider:null,
      executor:'not_observed'};
    if(!shape(route,['application','alias','instance'])||!nonempty(route.instance))
      fail('INVALID_APPLICATION_INSTANCE_BINDING');
    const instance=selectOne(env.instances,i=>i?.id===route.instance,
      'AMBIGUOUS_DATABASE_INSTANCE');
    if(!instance||!shape(instance,['id','definition','provider'])||
      instance.definition!==target.database||!providers.has(instance.provider))
      fail('INVALID_DATABASE_INSTANCE_ROUTE');
    const bound=selectOne(bindings,b=>b?.environment===environmentId&&
      b?.instance===instance.id&&sameTarget(b.target,target),
      'AMBIGUOUS_EXECUTOR_BINDING');
    if(bound&&(!shape(bound,['environment','instance','target','executor'])||
      !shape(bound.target,['database','contract','query'])||!nonempty(bound.executor)))
      fail('INVALID_EXECUTOR_BINDING');
    // No implicit Capability selection: it must be verified with Go's ResolvedSet.
    return {application:ctx.application,alias:ctx.datasourceAlias,
      status:'instance_route_declared',instance:instance.id,provider:instance.provider,
      executor:bound?'explicit_binding_declared_unverified':'not_observed'};
  });
  const fullyRouted=routes.every(r=>r.status==='instance_route_declared');
  const sameInstance=fullyRouted&&routes[0].instance===routes[1].instance;
  return {
    schemaVersion:'tei.research.query-provider-route/v0',
    status:fullyRouted?'application_routes_declared':'application_routes_incomplete',
    environment:environmentId,
    target,routes,sameInstanceDeclared:sameInstance,
    // Same provider/instance routing does not certify live connectivity or access.
    verified:{typedContractDeclaration:true,applicationRouting:fullyRouted,
      pluginExecutorResolution:false,providerHealth:false,
      callerAuthorization:false,providerResultConformance:false},
    executionPermitted:false,implementationSelected:false,
    accepted:false,acceptance:'not_asserted',
    authority:'derived_observation_only'
  };
}
