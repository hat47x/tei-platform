import test from 'node:test';
import assert from 'node:assert/strict';
import {createActionFormStarter} from './starter.mjs';

// BBR-12: the displayed decimal string must not be rounded into a different
// typed Action target. Independent expectations are fixed here, not derived
// from the output of the renderer or from a backend implementation.
function mount(invoke) {
  const states = [];
  const React = {
    createElement: (type, props, ...children) => ({type, props: props ?? {}, children}),
    // Both the current starter and the separate in-flight guard Draft #2864.
    useRef: (initial) => ({current: initial}),
    useState: (initial) => {
      const index = states.length;
      states.push(initial);
      return [initial, (value) => { states[index] = value; }];
    },
  };
  const Component = createActionFormStarter(React, {
    fields: [{name: 'row', type: 'number', label: 'Row'}],
    invoke,
  });
  const form = Component().children.find(node => node?.type === 'form');
  assert.ok(form, 'the reference must still render a form');
  return {form, states};
}

async function submitRaw(form, raw) {
  const oldFormData = globalThis.FormData;
  globalThis.FormData = class {
    get(name) { return name === 'row' ? raw : null; }
  };
  try {
    await form.props.onSubmit({
      preventDefault() {},
      currentTarget: {elements: {namedItem: () => null}},
    });
  } finally {
    globalThis.FormData = oldFormData;
  }
}

test('BBR-12: 0 and maximum safe row index retain their exact typed value', async () => {
  for (const [input, expected] of [
    ['0', 0],
    ['9007199254740991', Number.MAX_SAFE_INTEGER],
  ]) {
    const calls = [];
    const {form, states} = mount(async values => {
      calls.push(values);
      return {kind: 'completed', stage: 'completed'};
    });
    await submitRaw(form, input);
    assert.deepEqual(calls, [{row: expected}]);
    assert.equal(states[1].kind, 'completed');
  }
});

test('BBR-12: row outside safe range never reaches an Action, even if its Number rounds', async () => {
  for (const input of [
    '9007199254740992',  // first unsafe integer
    '9007199254740993',  // rounds to 9007199254740992
    '99999999999999999999999999999999999999',
    '999999999999999999999999999999999999999999999999999999999999999',
  ]) {
    let calls = 0;
    const {form, states} = mount(async () => {
      calls++;
      return {kind: 'completed', stage: 'completed'};
    });
    await submitRaw(form, input);
    assert.equal(calls, 0, 'unsafe input must not be passed to invoke: ' + input);
    assert.deepEqual(states[1], {kind: 'input_or_transport_error', stage: 'unavailable'});
  }
});

test('BBR-12: decimal, exponent, blank and negative row are not silently coerced', async () => {
  for (const input of ['1.0', '1e3', '', '-1', '01']) {
    let calls = 0;
    const {form} = mount(async () => { calls++; return {kind: 'completed', stage: 'completed'}; });
    await submitRaw(form, input);
    assert.equal(calls, 0, 'invalid row must not be coerced: ' + input);
  }
});
