/**
 * Read-only collision check for a small set of Web request plans.
 *
 * Each route is first checked by the existing single-route E1 validator.
 * This module neither resolves Providers nor installs HTTP handlers. Its
 * exact method/path keys do not claim proxy, origin or tenant isolation.
 */
import {composeRequestPlan} from './plan-check.mjs';

const supportedRouteKeys = new Set(['id','method','path','exposure','actionId']);

export function validateRouteTable(plans){
  if(!Array.isArray(plans) || plans.length===0)
    throw new Error('route table requires at least one request plan');

  const routeIDs=new Set(),matchOwners=new Map(),records=[];
  for(const plan of plans){
    const route=plan?.route;
    if(!route || typeof route!=='object' || Array.isArray(route))
      throw new Error('route table requires an explicit route object');
    for(const key of Object.keys(route)){
      if(!supportedRouteKeys.has(key))
        throw new Error('unsupported route match selector: '+key);
    }
    if(typeof route.id!=='string' || !route.id.trim())
      throw new Error('route identity must be a nonempty string');
    if(routeIDs.has(route.id))
      throw new Error('duplicate route identity: '+route.id);
    routeIDs.add(route.id);

    // Reuse the E1 guard, authorization, terminal-audit and exact-match
    // checks rather than permitting a looser multi-route policy.
    const evidence=composeRequestPlan(plan);

    // net/http GET patterns also handle HEAD. Treat the overlapping
    // request-method space as a collision; do not resolve by list order.
    const matchedMethods=route.method==='GET' ? ['GET','HEAD'] : [route.method];
    for(const method of matchedMethods){
      const key=JSON.stringify([method,route.path]);
      if(matchOwners.has(key))
        throw new Error('duplicate or overlapping route match: '+method+' '+route.path+
          ' ('+matchOwners.get(key)+' / '+route.id+')');
      matchOwners.set(key,route.id);
    }

    records.push({
      routeId:route.id,method:route.method,path:route.path,
      exposure:route.exposure,actionId:route.actionId??null,
      evidence
    });
  }

  // A route table must be reproducible regardless of declaration order;
  // sorting is for evidence only and never selects a winning handler.
  const key=x=>JSON.stringify([x.method,x.path,x.routeId]);
  records.sort((a,b)=>key(a)<key(b)?-1:key(a)>key(b)?1:0);
  return {routeCount:records.length,routes:records};
}
