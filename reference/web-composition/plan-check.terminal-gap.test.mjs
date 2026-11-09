// HDG-21 characterisation of current E1 static planner, NOT a runtime-audit test.
// Independent expected: configurationAccepted=true, pathAuditProof=false, runtimeAudit=unknown.
// Does not certify (or disprove) actual HTTP Host terminal auditing.
import assert from 'node:assert/strict';
import test from 'node:test';
import { composeRequestPlan } from './plan-check.mjs';

// Deliberately keep the plan representative of the existing protected Action fixture.
function fixture() {
  return {
    route: {
      id: 'sales.submit',
      method: 'POST',
      path: '/scenario/submit',
      exposure: 'protected',
      actionId: 'order_entry.submit_order',
    },
    hostPolicy: {
      id: 'interactive-write',
      requiredGuards: ['authenticated', 'csrf-protected', 'action-authorized'],
      terminalEvents: ['completed', 'rejected', 'error'],
    },
    hostContracts: ['action.identity'],
    steps: [
      { id: 'context', scope: 'server', provides: ['request.context'] },
      { id: 'identity', scope: 'server', requires: ['request.context'], provides: ['identity.principal'], guards: ['authenticated'] },
      { id: 'csrf', scope: 'server', requires: ['request.context'], provides: ['csrf.checked'], guards: ['csrf-protected'] },
      { id: 'authorize', scope: 'server', requires: ['identity.principal', 'action.identity'], provides: ['action.authorized'], guards: ['action-authorized'] },
      { id: 'invoke', scope: 'server', requires: ['action.authorized', 'csrf.checked'] },
      // Contains every required terminal label but has NO path/causal audit witness.
      { id: 'orphan-audit', scope: 'server', requires: ['request.context'], onTerminal: ['completed', 'rejected', 'error'] },
    ],
  };
}

// Independently walk "requires" contracts, not the planner's topological order.
function requiresAncestor(plan, nodeId, ancestorId) {
  const byId = new Map(plan.steps.map(step => [step.id, step]));
  const provider = new Map(plan.steps.flatMap(step => (step.provides || []).map(key => [key, step.id])));
  const seen = new Set();
  const visit = id => {
    if (seen.has(id)) return false;
    seen.add(id);
    for (const contract of byId.get(id)?.requires || []) {
      const from = provider.get(contract);
      if (from === ancestorId || (from && visit(from))) return true;
    }
    return false;
  };
  return visit(nodeId);
}

test('HDG-21: current planner accepts declared terminal labels without proving per-path audit', () => {
  const plan = fixture();
  const projection = composeRequestPlan(plan);
  assert.deepEqual(projection.terminalEvents, ['completed', 'rejected', 'error']);
  assert.ok(projection.order.includes('orphan-audit'));
  // This expectation is deliberately independent of E1 ordering:
  assert.equal(requiresAncestor(plan, 'invoke', 'orphan-audit'), false);
  // The claim this test qualifies is only: E1 declared coverage; actual runtime audit is UNKNOWN.
});

test('HDG-21 variant: "after" ordering hint is not an audit/effect causal witness', () => {
  const plan = fixture();
  plan.steps.find(step => step.id === 'orphan-audit').after = ['invoke'];
  const projection = composeRequestPlan(plan);
  assert.ok(projection.order.indexOf('invoke') < projection.order.indexOf('orphan-audit'));
  assert.equal(requiresAncestor(plan, 'invoke', 'orphan-audit'), false);
});
