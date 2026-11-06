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

The 9 checks demonstrate: T06 output equality for W0/W1 with identical *separately supplied*
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
