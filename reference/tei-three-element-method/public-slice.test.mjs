import test from "node:test";
import assert from "node:assert/strict";
import {projectPublicTeaSlice} from "./public-slice.mjs";
const ref = (id, role, scope = "sales") => ({scope, kind: "subject", id, role});
const row = (id, roleStates) => ({scope: "sales", kind: "subject", id, roleStates});
const publicInput = () => ({scope: "sales", subjects: [
  row("order", {data: "accepted", function: "accepted"}),
  row("review", {business: "accepted"}),
], relations: [{id: "uses", kind: "uses", from: ref("review", "business"), to: ref("order", "data")}]});
const copy = x => structuredClone(x);

test("T06: fixed public slice has identical three-view graph, relation table, geometry and counts", () => {
  const w0 = publicInput();
  const w1 = {...publicInput(), privateSubjects: [row("secret", {data: "accepted"})],
    privateEdges: [{id: "secret-edge", from: "secret", to: "order"}]};
  assert.deepStrictEqual(projectPublicTeaSlice(w0), projectPublicTeaSlice(w1));
  assert.equal(projectPublicTeaSlice(w0).coverage.allSourceCompleteness, "not-proven");
  assert.equal(projectPublicTeaSlice(w1).views.data.length, 1);
});

test("T06 negative control: using full graph size for layout would leak private existence", () => {
  const w0 = publicInput(), w1 = {...publicInput(), privateSubjects: [row("secret", {data: "accepted"})]};
  // Not part of the production projector: exposes why global layout is unsafe.
  const leakyY = world => 40 + 72 * (world.subjects.length + (world.privateSubjects?.length ?? 0));
  assert.notEqual(leakyY(w0), leakyY(w1));
  assert.deepStrictEqual(projectPublicTeaSlice(w0).views, projectPublicTeaSlice(w1).views);
});

test("T06: an accidental relation to an unqualified endpoint fails without identifying it", () => {
  const w = publicInput();
  w.relations.push({id: "private", kind: "uses", from: ref("review", "business"), to: ref("hidden", "data")});
  assert.throws(() => projectPublicTeaSlice(w), err => err.message === "unqualified_public_slice" && !err.message.includes("hidden"));
});

test("T07: unresolved Business Role does not remove accepted Data Role", () => {
  const w = {scope: "sales", subjects: [row("order", {data: "accepted", business: "unknown"})], relations: []};
  const out = projectPublicTeaSlice(w);
  assert.equal(out.views.data.length, 1);
  assert.equal(out.views.business.length, 0);
  assert.deepStrictEqual(out.publicDiagnostics.map(x => [x.role, x.status]), [["business", "unknown"]]);
});

test("T07: same-name other-scope Function cannot be selected by fallback", () => {
  const w = {scope: "sales", subjects: [row("order", {data: "accepted"}),
    {scope: "billing", kind: "subject", id: "order", roleStates: {function: "accepted"}}], relations: []};
  assert.throws(() => projectPublicTeaSlice(w), /unqualified_public_slice/);
});

test("T07: duplicate ids and unsupported role reject, ambiguity is retained", () => {
  for (const bad of [
    {...publicInput(), subjects: [row("order", {data: "accepted"}), row("order", {function: "accepted"})]},
    {scope: "sales", subjects: [row("order", {hidden: "accepted"})], relations: []},
  ]) assert.throws(() => projectPublicTeaSlice(bad), /unqualified_public_slice/);
  const ambiguous = projectPublicTeaSlice({scope: "sales", subjects: [row("order", {data: "ambiguous"})], relations: []});
  assert.deepStrictEqual(ambiguous.publicDiagnostics.map(x => x.status), ["ambiguous"]);
});

test("public presentation remains invariant to order of authorized subjects and relations", () => {
  const a = publicInput(), b = copy(a);
  b.subjects.reverse();
  assert.deepStrictEqual(projectPublicTeaSlice(a), projectPublicTeaSlice(b));
});

test("public subject cannot leak raw source paths or private fields", () => {
  const w = publicInput();
  w.subjects[0].sourceLocator = "/private/path/orders.tei";
  w.subjects[0].secretValue = "secret-value";
  const out = JSON.stringify(projectPublicTeaSlice(w));
  assert.equal(out.includes("/private/path"), false);
  assert.equal(out.includes("secret-value"), false);
});
