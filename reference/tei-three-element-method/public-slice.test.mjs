import test from "node:test";
import assert from "node:assert/strict";
import {projectPublicTeaSlice} from "./public-slice.mjs";
const ref = (id, role, scope = "sales") => ({scope, kind: "subject", id, role});
const row = (id, roleStates) => ({scope: "sales", kind: "subject", id, roleStates});
const publicInput = () => ({scope: "sales", workspaceObservation: "load:sales-A",
  subjects: [
  row("order", {data: "accepted", function: "accepted"}),
  row("review", {business: "accepted"}),
], relations: [{id: "uses", kind: "uses", from: ref("review", "business"), to: ref("order", "data")}]});
const copy = x => structuredClone(x);

test("T06: fixed public slice has identical three-view graph, relation table, geometry and counts", () => {
  // Private world state stays OUTSIDE the projector's public input.
  const w0 = {publicSlice: publicInput(), privateSubjects: []};
  const w1 = {publicSlice: publicInput(), privateSubjects: [row("secret", {data: "accepted"})],
    privateEdges: [{id: "secret-edge", from: "secret", to: "order"}]};
  assert.notDeepStrictEqual(w0, w1);
  assert.deepStrictEqual(projectPublicTeaSlice(w0.publicSlice), projectPublicTeaSlice(w1.publicSlice));
  assert.equal(projectPublicTeaSlice(w0.publicSlice).coverage.allSourceCompleteness, "not-proven");
  assert.equal(projectPublicTeaSlice(w1.publicSlice).views.data.length, 1);
  assert.throws(() => projectPublicTeaSlice({...w1.publicSlice, privateSubjects: w1.privateSubjects}),
    /unqualified_public_slice/);
});

test("T06 negative control: using full graph size for layout would leak private existence", () => {
  const w0 = {publicSlice: publicInput(), privateSubjects: []};
  const w1 = {publicSlice: publicInput(), privateSubjects: [row("secret", {data: "accepted"})]};
  // Negative control only: global layout can reveal undisclosed counts.
  const leakyY = world => 40 + 72 * (world.publicSlice.subjects.length + world.privateSubjects.length);
  assert.notEqual(leakyY(w0), leakyY(w1));
  assert.deepStrictEqual(projectPublicTeaSlice(w0.publicSlice).views,
    projectPublicTeaSlice(w1.publicSlice).views);
});

test("T06: an accidental relation to an unqualified endpoint fails without identifying it", () => {
  const w = publicInput();
  w.relations.push({id: "private", kind: "uses", from: ref("review", "business"), to: ref("hidden", "data")});
  assert.throws(() => projectPublicTeaSlice(w), err => err.message === "unqualified_public_slice" && !err.message.includes("hidden"));
});

test("T07: unresolved Business Role does not remove accepted Data Role", () => {
  const w = {scope: "sales", workspaceObservation: "load:sales-A",
    subjects: [row("order", {data: "accepted", business: "unknown"})], relations: []};
  const out = projectPublicTeaSlice(w);
  assert.equal(out.views.data.length, 1);
  assert.equal(out.views.business.length, 0);
  assert.deepStrictEqual(out.publicDiagnostics.map(x => [x.role, x.status]), [["business", "unknown"]]);
});

test("T07: same-name other-scope Function cannot be selected by fallback", () => {
  const w = {scope: "sales", workspaceObservation: "load:sales-A",
    subjects: [row("order", {data: "accepted"}),
    {scope: "billing", kind: "subject", id: "order", roleStates: {function: "accepted"}}], relations: []};
  assert.throws(() => projectPublicTeaSlice(w), /unqualified_public_slice/);
});

test("T07: duplicate ids and unsupported role reject, ambiguity is retained", () => {
  for (const bad of [
    {...publicInput(), subjects: [row("order", {data: "accepted"}), row("order", {function: "accepted"})]},
    {scope: "sales", workspaceObservation: "load:sales-A",
      subjects: [row("order", {hidden: "accepted"})], relations: []},
  ]) assert.throws(() => projectPublicTeaSlice(bad), /unqualified_public_slice/);
  const ambiguous = projectPublicTeaSlice({scope: "sales", workspaceObservation: "load:sales-A",
    subjects: [row("order", {data: "ambiguous"})], relations: []});
  assert.deepStrictEqual(ambiguous.publicDiagnostics.map(x => x.status), ["ambiguous"]);
});

test("public presentation remains invariant to order of authorized subjects and relations", () => {
  const a = publicInput(), b = copy(a);
  b.subjects.reverse();
  assert.deepStrictEqual(projectPublicTeaSlice(a), projectPublicTeaSlice(b));
});

test("public input boundary rejects unqualified metadata instead of silently dropping it", () => {
  const invalidInputs = [];
  const rawPath = publicInput();
  rawPath.subjects[0].sourceLocator = "/private/path/orders.tei";
  invalidInputs.push(rawPath);
  const secret = publicInput();
  secret.subjects[0].secretValue = "secret-value";
  invalidInputs.push(secret);
  const privateRelation = publicInput();
  privateRelation.relations[0].from.privateLocator = "/private/from";
  invalidInputs.push(privateRelation);
  const privateTopLevel = {...publicInput(), privateEdges: [{id: "secret-edge"}]};
  invalidInputs.push(privateTopLevel);
  for (const input of invalidInputs) {
    assert.throws(() => projectPublicTeaSlice(input),
      error => error.message === "unqualified_public_slice" &&
        !error.message.includes("private") && !error.message.includes("secret"));
  }
});

test("extra relation fields, symbols and getters are rejected at the public boundary", () => {
  const extra = publicInput();
  extra.relations[0].privateCount = 1;
  assert.throws(() => projectPublicTeaSlice(extra), /unqualified_public_slice/);
  const symbol = publicInput();
  symbol[Symbol("private")] = 1;
  assert.throws(() => projectPublicTeaSlice(symbol), /unqualified_public_slice/);
  const strange = publicInput();
  strange.subjects[0].roleStates = Object.create({data: "accepted"});
  assert.throws(() => projectPublicTeaSlice(strange), /unqualified_public_slice/);
  const accessor = publicInput();
  let getterCalls = 0;
  Object.defineProperty(accessor.subjects[0], "id", {enumerable: true,
    get() { getterCalls++; return "order"; }});
  assert.throws(() => projectPublicTeaSlice(accessor), /unqualified_public_slice/);
  assert.equal(getterCalls, 0);
});

test("public projection carries explicit caller epoch only when supplied", () => {
  const unstamped = projectPublicTeaSlice(publicInput());
  assert.equal(Object.hasOwn(unstamped, "snapshot"), false);
  const stamped = projectPublicTeaSlice(publicInput(), "source:R1|role:P1|policy:G1");
  assert.equal(stamped.snapshot, "source:R1|role:P1|policy:G1");
  assert.deepStrictEqual(stamped.views, unstamped.views);
  assert.throws(() => projectPublicTeaSlice(publicInput(), ""), /unqualified_public_slice/);
});

test("T12: Workspace observation is distinct from a potentially equal snapshot", () => {
  const a = publicInput(), b = publicInput();
  b.workspaceObservation = "load:sales-B";
  const x = projectPublicTeaSlice(a, "same-world-source-snapshot");
  const y = projectPublicTeaSlice(b, "same-world-source-snapshot");
  assert.equal(x.snapshot, y.snapshot);
  assert.notEqual(x.workspaceObservation, y.workspaceObservation);
  assert.deepStrictEqual(x.views, y.views);
});

test("T12: Workspace observation cannot be missing or empty in public input", () => {
  const missing = publicInput();
  delete missing.workspaceObservation;
  assert.throws(() => projectPublicTeaSlice(missing, "R1"), /unqualified_public_slice/);
  const empty = publicInput();
  empty.workspaceObservation = "";
  assert.throws(() => projectPublicTeaSlice(empty, "R1"), /unqualified_public_slice/);
});


test("T06: reject exotic public arrays before invoking their values", () => {
  const getterWorld = publicInput();
  let reads = 0;
  Object.defineProperty(getterWorld.subjects, "0", {
    configurable: true, enumerable: true,
    get() { reads++; return row("secret", {data: "accepted"}); },
  });
  assert.throws(() => projectPublicTeaSlice(getterWorld), /unqualified_public_slice/);
  assert.equal(reads, 0);

  const sparseWorld = publicInput();
  delete sparseWorld.subjects[0];
  assert.throws(() => projectPublicTeaSlice(sparseWorld), /unqualified_public_slice/);
});

test("T06: relation arrays reject own iterator, symbol and extra metadata", () => {
  const ownIterator = publicInput();
  let iteratorCalls = 0;
  ownIterator.relations[Symbol.iterator] = function* () { iteratorCalls++; yield* []; };
  assert.throws(() => projectPublicTeaSlice(ownIterator), /unqualified_public_slice/);
  assert.equal(iteratorCalls, 0);

  const extra = publicInput();
  extra.relations.hiddenCount = 1;
  assert.throws(() => projectPublicTeaSlice(extra), /unqualified_public_slice/);

  const otherPrototype = publicInput();
  Object.setPrototypeOf(otherPrototype.relations, []);
  assert.throws(() => projectPublicTeaSlice(otherPrototype), /unqualified_public_slice/);
});
