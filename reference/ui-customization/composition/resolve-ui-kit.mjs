// ISSUE #997: design-time Reference for explicitly layered UI Composition Kits.
// NOT a production TEI Plugin Resolver or a runtime Permission / Action adapter.
export class UIKitError extends Error {
  constructor(code, detail = '') { super(code + (detail ? ': ' + detail : '')); this.name = 'UIKitError'; this.code = code; }
}
const fail = (code, detail) => { throw new UIKitError(code, detail); };
const token = v => typeof v === 'string' && v.trim() !== '';
const exactVersion = v => typeof v === 'string' && /^\d+\.\d+\.\d+$/.test(v);
// Kit owners declare visual extension points; the resolver never introduces a new
// Core slot enum when a framework needs a different UI layout.
const visualSlot = s => token(s) && /^[a-z][a-z0-9-]*$/.test(s) &&
  !new Set(['action','permission','authorization','validation','semantic','resource','save']).has(s);
const permittedOps = new Set(['provide', 'replace', 'decorate']);

function inspect(manifest) {
  if (!manifest || manifest.schemaVersion !== 'tei-ui-composition-reference-v0' ||
      manifest.kind !== 'ui-composition-kit' || !token(manifest.id) || !exactVersion(manifest.version) ||
      !token(manifest.target) || !manifest.framework || !token(manifest.framework.id) ||
      !Number.isInteger(manifest.framework.major) || manifest.framework.major < 1 ||
      !Array.isArray(manifest.changes)) fail('invalid_manifest');
  if (manifest.extends != null && (!token(manifest.extends.id) || !exactVersion(manifest.extends.version))) fail('invalid_parent');
  if (manifest.extends == null && (!Array.isArray(manifest.slots) || !manifest.slots.length ||
    new Set(manifest.slots).size !== manifest.slots.length ||
    !manifest.slots.every(visualSlot))) fail('invalid_root_slots');
  if (manifest.extends != null && manifest.slots != null) fail('child_cannot_redefine_contract');
  const seen = new Set();
  for (const c of manifest.changes) {
    if (!c || !permittedOps.has(c.op) || !visualSlot(c.slot) || !token(c.implementation) ||
        seen.has(c.slot)) fail('invalid_change');
    seen.add(c.slot);
    if (c.op === 'provide' ? c.expect !== undefined : !token(c.expect)) fail('invalid_expectation');
  }
}

/**
 * A single explicit parent chain is intentional. Other independent plugins
 * may contribute via separate named points, but not via implicit MRO.
 * Implementations is a registry supplied by the project (functions, not code
 * executed from an untrusted manifest). The final slots match React starter.
 */
export function resolveUICompositionKit({catalog, implementations, selected, target, framework}) {
  if (!Array.isArray(catalog) || !implementations || typeof implementations !== 'object' ||
      !selected || !token(selected.id) || !exactVersion(selected.version) || !token(target) ||
      !framework || !token(framework.id) || !Number.isInteger(framework.major)) fail('invalid_request');
  const index = new Map();
  for (const m of catalog) {
    inspect(m);
    const key = m.id + '@' + m.version;
    if (index.has(key)) fail('ambiguous_kit', key);
    index.set(key, m);
  }
  let m = index.get(selected.id + '@' + selected.version);
  if (!m) fail('kit_not_found');
  const chain = [];
  const visited = new Set();
  while (m) {
    const key = m.id + '@' + m.version;
    if (visited.has(key)) fail('inheritance_cycle', key);
    visited.add(key);
    if (m.target !== target || m.framework.id !== framework.id ||
        m.framework.major !== framework.major) fail('framework_mismatch', key);
    chain.push(m);
    if (!m.extends) break;
    const parentKey = m.extends.id + '@' + m.extends.version;
    m = index.get(parentKey);
    if (!m) fail('missing_exact_parent', parentKey);
  }
  chain.reverse();
  const slots = Object.fromEntries(chain[0].slots.map(s => [s, null]));
  const trace = Object.fromEntries(chain[0].slots.map(s => [s, []]));
  const slotImpl = Object.fromEntries(chain[0].slots.map(s => [s, null]));
  for (const kit of chain) {
    for (const c of kit.changes) {
      if (!Object.hasOwn(slots,c.slot)) fail('unknown_extension_point',c.slot);
      const existing = slotImpl[c.slot];
      if (c.op === 'provide' && existing !== null) fail('slot_already_provided', c.slot);
      if (c.op !== 'provide' && (existing === null || existing !== c.expect))
        fail('override_conflict', c.slot);
      const fn = Object.hasOwn(implementations, c.implementation) ? implementations[c.implementation] : null;
      if (typeof fn !== 'function') fail('implementation_missing', c.implementation);
      if (c.op === 'decorate') {
        const parentRender = slots[c.slot];
        slots[c.slot] = context => fn(parentRender, context);
      } else slots[c.slot] = fn;
      slotImpl[c.slot] = c.implementation;
      trace[c.slot].push({kit:kit.id, version:kit.version, operation:c.op, implementation:c.implementation});
    }
  }
  return {selection:{id:selected.id,version:selected.version,target,framework:{...framework}},
    chain:chain.map(m=>({id:m.id,version:m.version})), slots, trace};
}
