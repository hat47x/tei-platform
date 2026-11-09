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

The initial 10 public-slice checks demonstrate: T06 output equality for W0/W1 with identical *separately supplied*
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

The initial 12 focused Trait tests (T10–T13) call the **actual existing public-slice
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


### Repeatable exact-head Node qualification

[Focused CI](../../.github/workflows/tei-tea-three-view-reference-node.yml)
checks out the exact PR head, pins Node 22.16.0, records the actual Git blobs
of both modules and tests, runs **native** `node --test` on the 10 + 12 synthetic
cases and asserts a clean checkout afterwards. The workflow is not a new
Product/Gate B/C or policy qualification; it is a repeatable focused test.
V8 test-shim results and workflow creation do not establish GitHub Actions
PASS until that workflow's **exact head** reports its completed result.


### Explicit Workspace-observation witness (T12 negative control, 2026-11-09)

[Actual CLI direct Trace observations in #3321](https://github.com/hat47x/tei-platform/pull/3321)
found distinct Workspaces with **equal** World, Source Index and Snapshot
fingerprints. Snapshot equality cannot be taken as Workspace equality.

This synthetic fixture now requires a **caller-qualified, disclosure-safe opaque**
`workspaceObservation` in the public slice, returns it on the projection,
and checks equality across `publicProjection`, the Trait envelope and
every Claim. Two otherwise equal public projections with the same
`snapshot` and View IDs but *different* Workspace-observation strings
cannot share Trait claims. Missing Workspace-observation witnesses fail
closed. A private Manifest path or a sensitive Root identifier is **not**
appropriate for this public field; an upstream approved Producer must
choose a disclosure-safe value. No policy-derived value is generated here.

`workspaceObservation` is **only a caller-supplied local equality witness**.
A caller that mistakenly reuses the same string for two physical Workspace
Roots defeats this check; byte-identical Workspace copies can still be
indistinguishable without an independent Reader/Root/Attempt identity.
This is the precise remaining limitation demonstrated by #3321, not a
claim that a new TEI Workspace ID, registry, authorization token or
cryptographic proof has been implemented.

The focused synthetic suite is now **12 public-slice + 15 Trait tests**
(27 cases total). The prior 10+12 counts above are historical. The
exact-head native-Node workflow is the actual qualification source;
changes only validated by V8 test-compatible shims are *not* native CI
PASS until the new head's workflow is completed.


### Strict ordinary-array boundary (2026-11-09)

The prequalified public presentation contract now rejects sparse arrays,
element accessors, own `Symbol.iterator` overrides, unexpected metadata
properties and modified array prototypes **before iterating** the input
Subject/Relation/Claim/View arrays. The earlier record-only getter check
covered object fields but not hostile arrays. A private element accessor
must not run as a side effect of projection.

The 4 new negative tests bring the synthetic cases to **14 public-slice
and 17 Trait bind (31 total)**. Native-Node CI on the updated exact PR head
is the source of repeatable qualification. The previous 12+15 native
CI and its pass status are **historical**, not a pass for the new 31 cases.

**Limit**: ordinary-array descriptor checks are not a hardened sandbox for
arbitrary Proxy traps, globally monkeypatched Array prototypes, adversarial
memory/time exhaustion or timing-channel noninterference. The trusted
Source/Role/Policy Owner must still construct a safe, already-disclosed
public input. This Reference never reads a private model, computes
Disclosure, proves complete Source coverage, or supplies current
authority. The correct production solution is a trusted ingestion and
qualified Consumer boundary, not adding a second Policy Engine here.


### T18: successful Catalog read and unsound Workspace remain separate

[ModelCatalog validation status #3728](https://github.com/hat47x/tei-platform/pull/3728)
has merged in main. Its real `CatalogWorkspace` can successfully return a
catalog whose `validation.ok=false`; the verdict also contains
`errorCount` and `warningCount`. Diagnostic details remain in
`validate -json`. The ModelCatalog result and the transport `ok` have
different meanings.

The new `catalog-validation.mjs` is **not a duplicate validator**. It
binds only a **caller-prequalified, already disclosed** Catalog-read-success
statement and the existing `validation` summary to the exact
`publicProjection.scope / snapshot / workspaceObservation`. The output
preserves `catalogReadSucceeded:true` separately from `validation.ok`,
with no per-Subject validity, edit qualification, complete Writer inventory
or Owner Accepted meaning synthesized. Counts are passed through, not
recomputed; a failed/absent read fails closed rather than becoming
`validation.ok=false`. The `catalogReadSucceeded` fixture field is
caller-supplied and is **not** a verified CLI envelope or Reader Attempt.

The 6 synthetic T18 tests check read success alongside Validation failure,
warnings without errors, missing/failed reads, cross-Workspace/old Snapshot,
malformed counts and unqualified metadata/getters. They invoke the existing
public-slice projector, but **not** real Go CLI, a real Workspace, the
current Policy, Studio, browser DOM, or the official Validator. The raw
Catalog summary's global diagnostic counts can disclose non-public facts;
the producer must make an independent current Disclosure decision
**before** passing this subset to a lower-privilege TEA Consumer.

```sh
node --test \
  reference/tei-three-element-method/public-slice.test.mjs \
  reference/tei-three-element-method/trait-attachments.test.mjs \
  reference/tei-three-element-method/catalog-validation.test.mjs
```

This branch's suite is now **14 + 17 + 6 = 37 synthetic tests**.
Earlier counts above describe prior versions. Only the exact-head native
Node CI completed successfully for this 37-case version can establish
native Node test passage; successful V8-shim assertions alone cannot.
