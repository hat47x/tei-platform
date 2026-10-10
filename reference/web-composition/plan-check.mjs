/** Read-only Web composition E1 candidate. No HTTP handler, provider resolver, or security enforcer. */
export function validateAuthenticationPlan(plan){
  if(!plan || !Array.isArray(plan.providers) || !Array.isArray(plan.policies) || !Array.isArray(plan.subjectLinks)) throw new Error("authentication plan is incomplete");
  if(plan.providers.length===0 || plan.policies.length===0) throw new Error("authentication plan must have an active policy and provider");
  const providers=new Map(),policyIDs=new Set();
  for(const p of plan.providers){
    if(!p.id || !p.kind || !p.issuer || providers.has(p.id)) throw new Error("provider identity is missing or duplicated");
    providers.set(p.id,p);
  }
  for(const policy of plan.policies){
    if(!policy.id || !Array.isArray(policy.allowedProviders) || policy.allowedProviders.length===0) throw new Error("authentication policy has no explicit providers");
    if(policyIDs.has(policy.id)) throw new Error("duplicate authentication policy identity: "+policy.id);
    policyIDs.add(policy.id);
    if(policy.fallbackOnFailure !== false) throw new Error("authentication fallback must not be implicit");
    for(const id of policy.allowedProviders) if(!providers.has(id)) throw new Error("authentication policy references a missing provider: "+id);
    if(new Set(policy.allowedProviders).size!==policy.allowedProviders.length) throw new Error("authentication policy repeats a provider");
  }
  const links=new Map(),exactLinks=new Set();
  for(const link of plan.subjectLinks){
    if(!link.providerId || !providers.has(link.providerId) || !link.issuer || !link.subject || !link.principalId) throw new Error("subject linking requires provider, issuer, subject and principal identity");
    if(providers.get(link.providerId).issuer!==link.issuer) throw new Error("subject issuer differs from provider identity");
    const exactKey=JSON.stringify([link.providerId,link.issuer,link.subject]);
    if(exactLinks.has(exactKey)) throw new Error("duplicate identity link declaration");
    exactLinks.add(exactKey);
    const key=JSON.stringify([link.providerId,link.issuer,link.subject]);
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
  if(route.actionId){
    for(const guard of ['authenticated','csrf-protected','action-authorized'])
      if(!hostPolicy.requiredGuards.includes(guard)) throw new Error("action route missing mandatory protection: "+guard);
    for(const terminal of ['completed','rejected','error'])
      if(!hostPolicy.terminalEvents.includes(terminal)) throw new Error("action route missing mandatory terminal audit: "+terminal);
  }
  if(new Set(hostContracts).size!==hostContracts.length) throw new Error("duplicate host contract");
  if(new Set(hostPolicy.requiredGuards).size!==hostPolicy.requiredGuards.length) throw new Error("duplicate required guard");
  const ids=new Map(),providers=new Map(),slots=new Set(),guards=new Set(),terminalEvents=new Set(),invalidGuardEvidence=[];
  // An executable standard guard must publish the specific contract proving
  // its result. A label attached to an unrelated prerequisite is not evidence.
  // Custom/plugin guards need a separately defined typed guard contract.
  const standardGuardEvidence=new Map([
    ['authenticated','identity.principal'],
    ['csrf-protected','csrf.checked'],
    ['action-authorized','action.authorized']
  ]);
  // Plugin-defined guards are admissible only with an explicit evidence
  // contract; a newly named guard alone cannot satisfy a host policy.
  const custom=hostPolicy.guardEvidence||{};
  if(typeof custom!=='object' || Array.isArray(custom)) throw new Error("guard evidence bindings must be a map");
  for(const [guard,evidence] of Object.entries(custom)){
    if(!guard || typeof evidence!=='string' || !evidence.trim()) throw new Error("custom guard requires an evidence contract");
    if(standardGuardEvidence.has(guard) && standardGuardEvidence.get(guard)!==evidence)
      throw new Error("standard guard evidence cannot be overridden: "+guard);
    standardGuardEvidence.set(guard,evidence);
  }
  for(const guard of hostPolicy.requiredGuards){
    if(!standardGuardEvidence.has(guard)) throw new Error("required guard has no declared evidence contract: "+guard);
  }
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
    for(const guard of p.guards||[]){
      const evidence=standardGuardEvidence.get(guard);
      if(evidence && !(p.provides||[]).includes(evidence))
        invalidGuardEvidence.push({guard,evidence});
      guards.add(guard);
    }
    for(const event of p.onTerminal||[]) terminalEvents.add(event);
  }
  if(route.actionId && !steps.some(p=>p.id==="invoke" && (p.requires||[]).includes("action.authorized"))) throw new Error("action invocation must depend on authorization");
  for(const g of hostPolicy.requiredGuards) if(!guards.has(g)) throw new Error("required host protection missing: "+g);
  for(const event of hostPolicy.terminalEvents) if(!terminalEvents.has(event)) throw new Error("required terminal coverage missing: "+event);
  const after=new Map([...ids.keys()].map(id=>[id,new Set()]));
  // Only required contracts establish a causal prerequisite. Ordering hints
  // alone cannot prove a guard is executed before an Action.
  const requiredBy=new Map([...ids.keys()].map(id=>[id,new Set()]));
  for(const p of steps){
    for(const contract of p.requires||[]){
      if(hostContracts.includes(contract)) continue;
      const source=providers.get(contract);
      if(!source) throw new Error("unsatisfied step contract: "+contract);
      if(source===p.id) throw new Error("self-dependency for contract: "+contract);
      after.get(p.id).add(source);
      requiredBy.get(p.id).add(source);
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
  if(route.actionId){
    const ancestors=new Set();
    const visit=id=>{
      for(const prerequisite of requiredBy.get(id)||[]){
        if(!ancestors.has(prerequisite)){ancestors.add(prerequisite);visit(prerequisite)}
      }
    };
    visit("invoke");
    for(const guard of hostPolicy.requiredGuards){
      if(![...ancestors].some(id=>(ids.get(id).guards||[]).includes(guard)))
        throw new Error("required guard is not a prerequisite of Action: "+guard);
    }
  }
  // Report disconnected guards before invalid evidence: both are defects,
  // but a missing causal dependency has to remain visible in old regressions.
  if(invalidGuardEvidence.length){
    const {guard,evidence}=invalidGuardEvidence[0];
    throw new Error("guard declaration lacks corresponding evidence: "+guard+" -> "+evidence);
  }
  return {routeId:route.id,order,requiredGuards:[...hostPolicy.requiredGuards],terminalEvents:[...hostPolicy.terminalEvents]};
}
