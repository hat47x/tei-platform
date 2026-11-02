/**
 * Read-only synthetic reference loader, not a TEI canonical meaning format/API.
 * Semantic/source assumptions live in source-fixture.json.
 * Selection, read observation and evaluation cut live in scenario-fixture.json.
 */
import { readFileSync } from 'node:fs';
export const fixture = Object.freeze(JSON.parse(
  readFileSync(new URL('./source-fixture.json', import.meta.url), 'utf8')
));
export const scenario = Object.freeze(JSON.parse(
  readFileSync(new URL('./scenario-fixture.json', import.meta.url), 'utf8')
));
if (scenario.sourceRevisionRef !== fixture.sourceRevision) {
  throw new Error('synthetic scenario references a different source revision');
}
export function membershipVersionsAt(data, validAt, knownAt) {
  const inWindow = (date, start, end) => start <= date && (end === null || date < end);
  return data.membershipVersions.filter(m =>
    inWindow(validAt, m.validFrom, m.validTo) &&
    inWindow(knownAt, m.knownFrom, m.knownTo));
}
export function baselineMemberships(data = fixture, run = scenario) {
  if (data.sourceRevision !== run.sourceRevisionRef) {
    throw new Error('synthetic scenario/source revision mismatch');
  }
  return membershipVersionsAt(data, run.baseline.validAt, run.baseline.knownAt);
}
