import test from 'node:test';
import assert from 'node:assert/strict';
import {readFileSync} from 'node:fs';
import {ObservationError,gitBlobSha,observeTeiPattern} from './observe-tei-pattern.mjs';

// These paths contain existing TEI Reference Source, not invented second Applications.
// Exact blob SHAs were fetched from the current repository's GitHub source API.
const fixtures = [
  ['submit-order.tei','edbb31af1a2018efe1f7e830cac4d06d2f59b188','command','capability','sales.order.submit'],
  ['order-lines.tei','e50f87ffe8dfa5601df4adcefe19dd798d47cb20','header-detail','relation','sales.order.lines'],
  ['customer-reference.tei','252f6396cbc2394d1f26e748b01d54f49a2e1ecf','reference',null,null],
];
const sourceDir = '../workspace/applications/order-entry/patterns/';
function observe(name, text, sha) {
  return observeTeiPattern({application:'order_entry',sourcePath:'reference/workspace/applications/order-entry/patterns/' + name,sourceText:text,expectedBlobSha:sha});
}
for (const [name,sha,kind,bindingKind,reference] of fixtures) {
  test('pinned actual TEI source: ' + name, () => {
    const sourceText = readFileSync(new URL(sourceDir + name, import.meta.url),'utf8');
    assert.equal(gitBlobSha(sourceText),sha);
    const r = observe(name,sourceText,sha);
    assert.equal(r.pattern.kind,kind);
    assert.equal(r.contract,null);
    assert.equal(r.acceptance,'unknown');
    if (bindingKind) {
      assert.equal(r.status,'observed');
      assert.deepEqual(r.pattern.binding,{kind:bindingKind,reference});
    } else {
      assert.equal(r.status,'partial');
      assert.equal(r.pattern.binding,null);
      assert.equal(r.diagnostics[0],'unsupported_or_additional_body');
    }
  });
}
test('stale or tampered source bytes cannot reuse pinned Git blob ID', () => {
  const [name,sha] = fixtures[0];
  const sourceText = readFileSync(new URL(sourceDir + name, import.meta.url),'utf8');
  assert.throws(() => observe(name,sourceText + '\n',sha),e => e instanceof ObservationError && e.code === 'blob_mismatch');
});
test('extra behavior is unresolved rather than silently discarded', () => {
  const name = fixtures[0][0];
  const original = readFileSync(new URL(sourceDir + name, import.meta.url),'utf8');
  const extra = original.replace('}', '    unknown-effect do_something\n}');
  const result = observe(name,extra,gitBlobSha(extra));
  assert.equal(result.status,'partial');
  assert.equal(result.pattern.binding,null);
  assert.equal(result.contract,null);
});
test('unrecognized Pattern form is partial and does not invent meaning', () => {
  const text='not-a-pattern';
  const result=observe('unsupported.tei',text,gitBlobSha(text));
  assert.equal(result.status,'partial');
  assert.equal(result.pattern,null);
});
test('invalid blob ID is rejected even for a syntactically valid source', () => {
  const name=fixtures[0][0];
  const sourceText=readFileSync(new URL(sourceDir + name, import.meta.url),'utf8');
  assert.throws(()=>observe(name,sourceText,'fake'),e=>e instanceof ObservationError && e.code==='invalid_blob_sha');
});
