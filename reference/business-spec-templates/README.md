# Business Specification Template — Reference BS-1/BS-2

Issue: [#998](https://github.com/hat47x/tei-platform/issues/998). **Independently tracked UI starter**: [#997](https://github.com/hat47x/tei-platform/issues/997), [#1000](https://github.com/hat47x/tei-platform/issues/1000).

## A template is not an Application, renderer, or Accepted fact

`order-entry.template.json` is a **candidate index** of existing TEI business semantics and unresolved project adoption decisions. It neither copies Data definitions nor redefines their Identifier / Functional Dependency / Relation / Validation semantics. The exact-existing references are:

- Data: `reference/workspace/semantic/sales/order.tei`, `sales/validation.tei`, `customer/customer.tei`.
- Function: `reference/workspace/applications/order-entry/application.tei`, `patterns/submit-order.tei`.
- Business: `reference/business-system-design/order-entry-business-procedure.json`.

The original Business Procedure is already a **reference-authoring fixture**, not a Canonical Procedure. This candidate indexes the existing fixture and does not silently upgrade its authority.

The application identity and detailed business rules belong to the original TEI Workspace and Business Reference. The template only supplies stable source references and four project-specific unresolved questions concerning order identification, line identification, manual readiness and submission responsibility. No UI/React Component, layout or theme belongs here.

## Proposed application to a project

`template-reference.mjs` exposes:

- `inspectTemplate(candidate)`: fail closed on unsupported schema, missing three-view coverage, duplicate identity binding, UI layer leakage or malformed unresolved questions.
- `proposeAdoption(candidate, {projectId, decisions})`: produce a **proposal only**. Missing, AI-proposed or unsourced project answers remain unresolved. Even when all answers have human source references the output says `accepted:false` and `explicit_shared_acceptance_required`. This module cannot perform actual TEI Shared Acceptance.
- `inspectUpgrade(old, new, projectProposal)`: detect changes to source reference bindings and project decision questions, keeping the project proposal unchanged; force review on meaningful changes rather than overwriting accepted project meaning.

Run:

```sh
node --test reference/business-spec-templates/template-reference.test.mjs
```

Verified locally on 2026-10-08, Node.js 22.16.0: **8 passed, 0 failed**. Committed blob SHAs match the locally executed files:

- `order-entry.template.json`: `0c119ae5aaec71fb8267fa20f0b5208a888326ac`
- `template-reference.mjs`: `f849603784a88590efec596c8d1a202ee230b4da`
- `template-reference.test.mjs`: `f1a1648a9f6393c9f2aa3fce9eaf906e13006306`

## Not established

No TEI Workspace loader / Shared Acceptance / normalized World integration, no dynamic source identity resolution, no real second-project derivation of Semantic artifacts, no template distribution/Plugin Lock, no human review workflow, no measured savings. Tests check **reference-candidate structure and proposal authority only**, not that source files are semantically correct under the current Go compiler. The reference may later need to migrate to TEI's semantic DSL; do not introduce a Canonical Template kind from this artifact.

Next milestone: use a real second project to instantiate the candidate with explicit human decisions, feed any needed changes through the existing TEI Shared Acceptance boundary, and compare cross-view diagnostics against constructing the same project without a template. Binding to #1000's React starter must be via the resulting Application Action contract, not by copying business semantics into the UI template.
