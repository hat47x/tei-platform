// Stage 6 JavaScript side of the sample. It imports the unchanged probe of
// reference/multilingual-format and applies the same selection rule as
// format.go: the plural category picks a variant (falling back to "other"),
// and {count} becomes the grouped integer. Reads a JSON request on stdin.
import { readFileSync } from 'node:fs';
import { probe } from '../../reference/multilingual-format/probe.mjs';

const request = JSON.parse(readFileSync(0, 'utf8'));
const results = request.cases.map(c => {
  const r = probe(c.locale, c.count);
  const variant = c.variants[r.category] ?? c.variants.other;
  return { id: c.id, category: r.category, number: r.number, text: variant.split('{count}').join(r.number) };
});
process.stdout.write(JSON.stringify({
  node: process.versions.node,
  icu: process.versions.icu ?? '',
  cldr: process.versions.cldr ?? '',
  unicode: process.versions.unicode ?? '',
  results
}));
