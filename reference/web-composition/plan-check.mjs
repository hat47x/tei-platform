/** Read-only Web composition E1 candidate. No HTTP handler, provider resolver, or security enforcer. */
export function validateAuthenticationPlan(plan){
  if(!plan || !Array.isArray(plan.providers) || !Array.isArray(plan.policies) || !Array.isArray(plan.subjectLinks)) throw new Error("authentication plan is incomplete");
  const providers=new Map();
  for(const p of plan.providers){
    if(!p.id || !p.kind || !p.issuer || providers.has(p.id)) throw new Error("provider identity is missing or duplicated");
    providers.set(p.id,p);
  }
  for(const policy of plan.policies){
    if(!policy.id || !Array.isArray(policy.allowedProviders) || policy.allowedProviders.length===0) throw new Error("authentication policy has no explicit providers");
    if(policy.fallbackOnFailure !== false) throw new Error("authentication fallback must not be implicit");
    for(const id of policy.allowedProviders) if(!providers.has(id)) throw new Error("authentication policy references a missing provider: "+id);
    if(new Set(policy.allowedProviders).size!==policy.allowedProviders.length) throw new Error("authentication policy repeats a provider");
  }
  const links=new Map();
  for(const link of plan.subjectLinks){
    if(!link.providerId || !providers.has(link.providerId) || !link.issuer || !link.subject || !link.principalId) throw new Error("subject linking requires provider, issuer, subject and principal identity");
    if(providers.get(link.providerId).issuer!==link.issuer) throw new Error("subject issuer differs from provider identity");
    const key=JSON.stringify([link.issuer,link.subject]);
    if(links.has(key) && links.get(key)!==link.principalId) throw new Error("conflicting principal mapping");
    links.set(key,link.principalId);
  }
  return {providerCount:providers.size,policyCount:plan.policies.length,linkedSubjectCount:links.size};
}

export function composeRequestPlan(plan){
  if(!plan || !plan.route || !plan.hostPolicy || !Array.isArray(plan.steps) || !Array.isArray(plan.hostContracts)) throw new Error("request plan is incomplete");
  const {route,hostPolicy,steps,hostContracts}=plan;
  if(!route.id || !route.method || !route.path || !hostPolicy.id || !Array.isArray(hostPolicy.requiredGuards) || !Array.isArray(hostPolicy.terminalEvents)) throw new Error("host route policy is incomplete");
  if(route.actionId && route.exposure==="public") throw new Error("action route cannot silently become public");
  if(route.actionId && !hostPolicy.requiredGuards.includes("action-authorized")) throw new Error("action route has no mandatory authorization");
  const ids=new Map(),providers=new Map(),slots=new Set(),guards=new Set(),terminalEvents=new Set();
  for(const p of steps){
    if(!p.id || ids.has(p.id) || p.scope!=="server") throw new Error("step identity is duplicated or scope is not server");
    ids.set(p.id,p);
    for(const slot of p.exclusiveSlots||[]){
      if(slots.has(slot)) throw new Error("exclusive middleware slot collision: "+slot);
      slots.add(slot);
    }
    for(const contract of p.provides||[]){
      if(providers.has(contract)||hostContracts.includes(contract)) throw new Error("multiple providers of contract: "+contract);
      providers.set(contract,p.id);
    }
    for(const guard of p.guards||[]) guards.add(guard);
    for(const event of p.onTerminal||[]) terminalEvents.add(event);
  }
  for(const g of hostPolicy.requiredGuards) if(!guards.has(g)) throw new Error("required host protection missing: "+g);
  for(const event of hostPolicy.terminalEvents) if(!terminalEvents.has(event)) throw new Error("required terminal coverage missing: "+event);
  const after=new Map([...ids.keys()].map(id=>[id,new Set()]));
  for(const p of steps){
    for(const contract of p.requires||[]){
      if(hostContracts.includes(contract)) continue;
      const source=providers.get(contract);
      if(!source) throw new Error("unsatisfied step contract: "+contract);
      if(source===p.id) throw new Error("self-dependency for contract: "+contract);
      after.get(p.id).add(source);
    }
    for(const id of p.after||[]){
      if(!ids.has(id)) throw new Error("unknown predecessor: "+id);
      after.get(p.id).add(id);
    }
    for(const id of p.before||[]){
      if(!ids.has(id)) throw new Error("unknown successor: "+id);
      after.get(id).add(p.id);
    }
  }
  const order=[],remaining=new Set(ids.keys());
  while(remaining.size){
    const ready=[...remaining].filter(id=>[...after.get(id)].every(prev=>!remaining.has(prev))).sort();
    if(!ready.length) throw new Error("middleware ordering cycle");
    for(const id of ready){order.push(id);remaining.delete(id)}
  }
  return {routeId:route.id,order,requiredGuards:[...hostPolicy.requiredGuards],terminalEvents:[...hostPolicy.terminalEvents]};
}
