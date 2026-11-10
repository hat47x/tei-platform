// Bounded, test-only cross-reference: E1 acceptance is not Go Host binding.
// These two representations are deliberately separate in the fixture. This
// comparator is NOT a production resolver or proof that Host callbacks ran.
import assert from 'node:assert/strict';
import {readFileSync} from 'node:fs';
import test from 'node:test';
import {fileURLToPath} from 'node:url';
import {composeRequestPlan} from './plan-check.mjs';

const fixture = JSON.parse(readFileSync(
  fileURLToPath(new URL('./e1-go-route-pair.fixture.json', import.meta.url)), 'utf8'
));
const clone = value => structuredClone(value);

// An intentional test-only comparison of exact, scoped route identity.
function sameDeclaredRoute(e1, host) {
  return e1.route.method === host.method &&
    e1.route.path === host.path &&
    e1.route.actionId === host.actionId;
}

test('HDG-B01: E1 accepts its own guarded plan and separately declared route matches', () => {
  assert.equal(composeRequestPlan(fixture.e1).routeId, 'sales.submit');
  assert.equal(fixture.e1.route.path, '/scenario/submit'); // independent expected
  assert.equal(fixture.e1.route.actionId, 'order_entry.submit_order');
  assert.equal(sameDeclaredRoute(fixture.e1, fixture.goHost), true);
});

test('HDG-B02: a different valid E1 route is accepted locally, but not paired to Go Host', () => {
  const e1 = clone(fixture.e1);
  e1.route.path = '/scenario/alternate-submit';
  assert.equal(composeRequestPlan(e1).routeId, 'sales.submit');
  assert.equal(sameDeclaredRoute(e1, fixture.goHost), false);
});

test('HDG-B03: a different E1 Action is valid locally, but not paired to Go Host', () => {
  const e1 = clone(fixture.e1);
  e1.route.actionId = 'order_entry.another_action';
  assert.equal(composeRequestPlan(e1).routeId, 'sales.submit');
  assert.equal(sameDeclaredRoute(e1, fixture.goHost), false);
});

test('HDG-B04: changed Go Host policy route cannot be repaired by unchanged E1 validity', () => {
  const host = clone(fixture.goHost);
  host.path = '/scenario/other';
  assert.equal(composeRequestPlan(fixture.e1).routeId, 'sales.submit');
  assert.equal(sameDeclaredRoute(fixture.e1, host), false);
});

test('HDG-B05: exact route pairing does not qualify the Provider/Grant or terminal audit', () => {
  const host = clone(fixture.goHost);
  host.providers.corporate = 'https://different-issuer.example.invalid';
  assert.equal(sameDeclaredRoute(fixture.e1, host), true);
  // A route-identity match is deliberately NOT treated as Provider equality,
  // Authorization conformance, actual effect, or terminal-path audit proof.
  assert.notDeepEqual(host.providers, fixture.goHost.providers);
});
