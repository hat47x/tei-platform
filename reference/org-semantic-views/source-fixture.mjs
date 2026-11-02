/** Read-only reference helper. This is test data, not a TEI canonical model or API. */
import { readFileSync } from 'node:fs';
export const fixture = Object.freeze(JSON.parse(
  readFileSync(new URL('./source-fixture.json', import.meta.url), 'utf8')
));
export function membershipVersionsAt(data, validAt, knownAt) {
  const inWindow = (date, start, end) => start <= date && (end === null || date < end);
  return data.membershipVersions.filter(m => 
    inWindow(validAt, m.validFrom, m.validTo) &&
    inWindow(knownAt, m.knownFrom, m.knownTo));
}
export function baselineMemberships(data = fixture) {
  return membershipVersionsAt(data, data.baseline.validAt, data.baseline.knownAt);
}
