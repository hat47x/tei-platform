# TEA prequalified public-slice Reference (T06/T07)

This is a deliberately small, read-only *synthetic* characterization of the
three-view public presentation boundary in the [TEA method design](../../plan/design/THREE_ELEMENT_METHOD_DIAGRAM_CONTRACT_2026-10.md).

`projectPublicTeaSlice` accepts only a **prequalified, already disclosed** bounded
public input from a separate trusted Source/Role/Policy owner. The fixture
boundary fails closed for extra keys on the top-level, subjects, relations,
relation endpoints, and Role state maps; raw/private world data are **not**
passed to this projection. It does **not**
load a Workspace, authenticate, evaluate Policy/Grant, resolve Role Providers,
verify Source freshness, read an actual Source, or check Authoring Acceptance.
`accepted` is an input role-resolution state, not an authority issued by this
module. The input shape is a fixture, **not** a proposed production schema.

From this fixture it deterministically derives read-only Data/Function/Business
node occurrences, qualified subject refs, a relation table, small stable layout,
public-qualified unresolved Role diagnostics, and `not-proven` global coverage.
`viewId` is never promoted to Canonical Source identity. It only permits
single-scope fixture inputs; a second scope must be projected independently.

Run with Node 22+:

```sh
node --test reference/tei-three-element-method/public-slice.test.mjs
```

The 10 public-slice checks demonstrate: T06 output equality for W0/W1 with identical *separately supplied*
prequalified public inputs, an intentionally leaky whole-graph-layout negative control, rejection
of a relation whose endpoint was not public-qualified, and T07 isolation of
an unknown/ambiguous Business or Data Role without turning unrelated accepted
Data into `unknown`. They also check same-name cross-scope rejection,
order-invariant output, and **rejection** of unqualified raw Source path/secret fields,
unknown record fields, symbol properties, malformed prototypes and accessor getters.

**Evidence limits**: Local pure Node assertions are not proof of actual Policy
sanitization, side-channel resistance (errors, event timing, caches, network),
real DOM/ARIA/Graph rendering, CLI→Workbench same-attempt binding, actual
private-source noninterference, Source/grant change notification, Provider
conformance or Owner acceptance. The harness does not take a full private
world as input: it cannot certify that a production caller supplies a correctly
qualified public slice. No user/customer/legacy X-TEA data are used.

The real Owner contracts remain #2516 (Disclosure), #1124 (Studio Consumer),
#2620 (selected Role interpretation), #951 (Provider) and #2604 (Artifact
publication). In particular, generic `unqualified_public_slice` failures must
be mapped to a constant, disclosure-safe Consumer state; this Reference does
not establish that a real UI does so. Do not promote its success to M1/M3 or
Production.

**Conformance precondition**: Even a strict shape is not proof that an
upstream Policy chose the correct Subject/Relation set, interpreted a Role
correctly, or observed all Required Source and exact Rule/Operator pins.
`accepted` in the synthetic Role map is deliberately not proof of current
Owner acceptance, and a parsed map cannot recover duplicate `role` options
that an upstream DSL parser might have silently overwritten. See
[#2620](https://github.com/hat47x/tei-platform/issues/2620) and the separate
[parser rejection candidate #2898](https://github.com/hat47x/tei-platform/pull/2898).
Do not construct this shape directly from an untrusted model or use it as a
Policy filter: the upstream qualified public read is an independent gate.
Shape checking is not hardened for arbitrary JS Proxies, malicious input array
iterators or side-channel effects, and cannot establish noninterference itself.


## T10–T13: read-only prequalified Trait attachments

The additional `trait-attachments.mjs` is a **presentation-only join** of
already qualified, already disclosed Trait Claim references into existing
Data / Function / Business occurrences. It uses `projectPublicTeaSlice`
output as its source of view occurrences. It does not create or infer Trait
Definitions, Owner Intent, actual Source Observations, Assessments, business
MECE, key/strong attribute semantics, or currently authorized eligibility.

`bindPublicTraitClaims(publicProjection, {scope, snapshot, claims})`
requires one exact caller-supplied snapshot string across the **actual public
projection**, the envelope and each claim, plus matching scope/subject/Role,
no duplicate claim IDs, and a matching published view occurrence. Obtain a
stamp by calling `projectPublicTeaSlice(qualifiedSlice, snapshot)`. Its
one-argument mode remains read-only but **cannot accept Trait attachments**.
A different or missing projection stamp rejects even when the view IDs match. The returned `intent`, `observation`, and `assessment`
lanes and local states are **fixture vocabulary only**, not a TEI canonical
schema, Evidence qualification, or a new status/authority enum. The supplied
`snapshot` is a three-way, **caller-assigned equality witness only**: it
**does not verify** the current Source, Role/Operator implementation bytes,
Policy, as-of or Artifact. A reused/ABA or forged stamp cannot be detected
without the real Producer, current policy and independent readback.
Inputs containing unqualified metadata fail closed with a generic error.

```sh
node --test reference/tei-three-element-method/public-slice.test.mjs \
            reference/tei-three-element-method/trait-attachments.test.mjs
```

The 12 focused Trait tests (T10–T13) call the **actual existing public-slice
projector**, then bind the prequalified Claim references. They prove
synthetic three-view reference correspondence; that a Java Enum/DDL
observation does not synthesize an Assessment; that no "strong attribute"
or business Identity claim is inferred; that old snapshot, other Scope,
absent/private subject, duplicate claim, unknown Role or raw private
metadata are rejected. In particular, an old/unstamped *projection* cannot
receive current claims solely because its display IDs match. One test checks that an input accessor is rejected
without being invoked. This is not a cryptographic proof of Source
freshness, nor a validation of arbitrary untrusted JS Proxies or array
accessors. The cross-view input is synthetic, not an actual Java/SQL/TEI
Source Reader or a real Workbench DOM.

**Next required integration evidence**: one exact public Semantic Subject
with independently read Script/Role/Operator pins, Java Enum or equivalent
declaration, database DDL constraint, and currently eligible Consumer
Public Slice; preserve absence of a complete Writer inventory as
`not-proven`. Then compare actual Inspector/Graph/Table/ARIA/diagnostic
output in the same Consumer and attempt. Existing Owners: #3027 (Trait
assessment, merged design), #1126 (Data identity/FD), #2620 (Role
interpretation), #2516 (Disclosure), #1124 (Studio), #267 (Acceptance).
This Reference is **not** an implementation of any of those Owners.
