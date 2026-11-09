import test from 'node:test';
import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import { probe, MAX_COUNT } from './probe.mjs';

const fixture = JSON.parse(await readFile(new URL('./testdata/integer-forms.json', import.meta.url), 'utf8'));

test('shared integer formatting corpus', async t => {
  assert.equal(fixture.schema, 'tei.multilingual.integer-forms.e2a');
  assert.ok(fixture.cases.length > 0);
  const seen = new Set();
  for (const item of fixture.cases) {
    await t.test(item.id, () => {
      assert.ok(item.id && !seen.has(item.id), 'case IDs must be unique');
      seen.add(item.id);
      assert.deepEqual(probe(item.locale, item.count), {
        category: item.category,
        number: item.number
      });
    });
  }
});

test('fail closed on unsupported locale and invalid count', () => {
  assert.throws(() => probe('fr-FR', 1), /unsupported-locale/);
  assert.throws(() => probe('EN_us', 1), /unsupported-locale/);
  for (const bad of [-1, 1.5, MAX_COUNT + 1, NaN, Infinity, '1', null]) {
    assert.throws(() => probe('en-US', bad), /invalid-count/);
  }
});

test('does not select a different language or modify input', () => {
  const first = probe('ja-JP', 12345);
  assert.deepEqual(first, { category: 'other', number: '12,345' });
  assert.deepEqual(probe('ja-JP', 12345), first);
});
