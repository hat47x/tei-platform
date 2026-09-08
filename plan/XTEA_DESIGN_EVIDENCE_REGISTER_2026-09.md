# X-TEA / XEAD Design Reference Register — 2026-09

- Status: research evidence register
- Purpose: retain externally checkable design evidence that has materially informed TEI questions
- Authority: research evidence only; this file does not override TEI Product Requirements, ADRs, accepted Canonical meaning, or Conformance evidence
- Relationship: TEI does not claim to be an official successor or derivative of X-TEA / XEAD

## 1. Evidence notation

```text
P1  published explanation
M1  documented manual or model structure
S1  public source-code behavior
R1  one Reference example
R2  repeated observation across multiple References
T1  TEI-local Reference experiment
C1  accepted TEI Conformance / deterministic diagnostic
```

The register separates external description, observed implementation, content pressure, and TEI-local decision. A concept found in an external reference is not automatically a TEI concept.

## 2. Evidence entries

### E-001 — Business design includes more than software flow

**Grade:** P1

**Source:** Kozo Watanabe, 「データ、UI、業務手順の『3要素』をとらえる」, 2006.

**Observation:** The published explanation treats business procedure as part of business-system design rather than reducing the system to software functions alone.

**TEI implication:** Preserve a Business Procedure viewpoint that can include actors, triggers, manual work, and use of system functions. Whether `Procedure` needs durable Canonical identity remains a TEI-local question.

### E-002 — Business definitions can be structured and cross-referenced

**Grade:** P1

**Source:** Kozo Watanabe, 「蔓延する『ブルシットエンジニアリング』」, 2024-07-05.

**Observation:** The article presents business definitions with identity, occasion/event, responsible person, and references to function definitions and their usage.

**TEI implication:** If TEI adopts a durable Procedure representation, it should preserve the business viewpoint without assuming a Workflow Engine graph.

### E-003 — Matrices are useful as derived review projections

**Grade:** P1 + S1

**Sources:** published explanation above; public XeadModeler source.

**Observation:** CRUD-oriented documents and matrices can be produced from structured model relationships.

**TEI implication:** Prefer derived Function × Data / Business × Data views over duplicate authored matrices. A matrix should expose conflicts without owning the underlying meaning.

### E-004 — Function design spans UI, Data I/O, and calls

**Grade:** P1

**Source:** published explanation above.

**Observation:** Function definitions are described in terms broader than a backend API catalog, including input/output, UI/table use, and function calls.

**TEI implication:** Reconstruct the Function viewpoint from Application Pattern / View, Action / Capability, Query, Data references, CRUD, and call projections where possible rather than introducing a mandatory symmetric `Function` Canonical noun.

### E-005 — Business Action and Application Function are distinct in the public modeler source

**Grade:** S1

**Source:** public `xead/XeadModeler` repository.

**Observation:** Task actions and Function references are represented separately.

**TEI implication:** Keep `Business Action != Application Pattern != Capability`. A business action may have no software realization.

### E-006 — Reference content contains actions without Function realization

**Grade:** R2

**Source:** static analysis of multiple reference-model artifacts used for research; original contents are not redistributed here.

**Observation:** Repeated examples exist in which business actions do not bind to an Application Function.

**TEI implication:** Do not force every Business Procedure step to bind to executable software merely to make a graph complete.

### E-007 — Subsystem-like boundaries carry cohesion / dependency pressure

**Grade:** S1 + M1

**Source:** public XeadModeler source and documented model structure.

**Observation:** Subsystem-oriented views expose cross-boundary data operations and function dependencies.

**TEI implication:** Do not mechanically translate a historical Subsystem into a microservice, repository, deployment unit, or Database. First derive ownership, read dependencies, effective CUD, and exposed Action boundaries.

### E-008 — SubjectArea and Subsystem are separate concepts in the observed model structure

**Grade:** S1

**Observation:** Both concepts are represented separately.

**TEI implication:** Do not assume namespace, Application, Database, ownership boundary, and business subject area are one concept. Test whether Subject Area can remain a derived view.

### E-009 — Presentation order is not automatically business time

**Grade:** S1 + R2

**Observation:** Model elements can retain descriptive text and sort/presentation order independently, and reference content uses ordering for more than one purpose.

**TEI implication:** Keep Dataset, Snapshot, Scenario, Expectation, Presentation Order, and temporal semantics conceptually separable. Do not infer business time from display order.

### E-010 — State can be derived from underlying facts

**Grade:** R2

**Observation:** Static research across reference content found state/transition behavior that cannot always be reduced to one mutable `status` field.

**TEI implication:** Keep State predicate, Transition, Transition Action, and Consequence Rule separable where the business requires them. Introduce durable concepts only when TEI-local Reference evidence demonstrates the need.

### E-011 — Historical script hooks do not form one semantic category

**Grade:** R2 + S1

**Observation:** Script mechanisms are used for different roles, such as design functions, consequence rules, and operational rules.

**TEI implication:** Classify meaning before exposing any mechanism as a public Capability contract.

### E-012 — Realistic References can improve both platform and model

**Grade:** P1 + R2

**Observation:** Running or reviewing realistic content can expose both platform gaps and model defects.

**TEI implication:** Use both small deterministic fixtures and realistic multi-area Reference content. Do not infer business-design maturity from toy fixtures alone.

### E-013 — Source-code licensing and Reference-content licensing are separate questions

**Grade:** S1

**Source:** license files in the public XeadModeler / XeadDriver repositories.

**Observation:** Public source has its own license terms; manuals, downloadable models, databases, and third-party reproductions may have different terms.

**TEI rule:** Check the exact artifact's terms. Until reuse permission is clear, inspect and summarize where permitted, preserve provenance, and independently recreate TEI test content rather than copying substantial third-party material.

### E-014 — Business Flow is broader than a Procedure Handoff

**Grade:** P1 + M1 + S1

**Sources:** Kozo Watanabe's Three-Element Analysis explanation and public XeadModeler data-flow source structure.

**Observation:** Business Flow is represented as a separate flow model with process/task-linked nodes, non-process participants, and explicit flow lines. That structure is broader than a relation between two Procedure steps.

**TEI implication:** Keep the current thin Handoff meaning (`Procedure/Step -> Procedure/Step + condition`) separate from full Business Flow / DFD. Do not introduce a generic Core edge or Business Flow Canonical kind until TEI-local realistic evidence requires information/document/data flow and non-Procedure participants.

### E-015 — Task/Function separation does not imply a generic reference-role taxonomy

**Grade:** S1 + T1

**Sources:** public XeadModeler TaskAction / TaskFunctionIO structure and the TEI-local BP-06 protected-target experiment.

**Observation:** Historical Task actions and Function references are distinct. TEI's BP-06 also distinguishes supporting authority (`security.authorize`) from the protected Application target (`order_entry.submit_order`) while having no direct Function realization.

**TEI implication:** Keep `Business Action != Application Function`, but do not generalize one protected-target case into a universal `{Kind, Role, Target}` taxonomy. Authorization policy may corroborate a Business protected target but must not author or backfill it.

## 3. Current conclusions

### C-A — Preserve the Business Procedure viewpoint

Supported by E-001, E-002, E-005, and E-006. What remains open is whether it needs durable Canonical identity.

### C-B — Keep Business Action separate from software realization

A valid Business Action may be manual, external, unresolved, or supported by existing Application / Capability meaning. Do not invent software bindings merely for structural completeness.

### C-C — Prefer derived cross-check views

Business Action × Application Pattern, Business Action × Data / Relation, Function / Action × Data CRUD, ownership × effective CUD, and call graphs are candidates for derived projections rather than duplicate sources of truth.

### C-D — Keep external model nouns decomposable

Do not make one-to-one Canonical translations such as `Task -> Workflow node`, `Subsystem -> microservice`, `SubjectArea -> namespace`, `Instance -> Scenario`, or `Script -> Capability` without TEI-local proof.

### C-E — Keep Procedure Handoff narrower than Business Flow

Supported by E-001 and E-014. The current Handoff pressure preserves accepted continuation responsibility; it does not own external participants, information/document/data flow, or a full DFD.

### C-F — Keep cross-view reference roles evidence-driven

Supported by E-005, E-006, and E-015. Task/Function separation is well supported, while a generic protected/supports/concerns role taxonomy is not. Keep the current protected-target representation narrow until repeated TEI-local evidence proves a broader abstraction.

## 4. Open TEI-local evidence gaps

- Can a derived Procedure preserve accepted business meaning and review history without durable Procedure identity?
- Can Procedure / UI cross-view review detect read/write boundary violations without duplicating datasource meaning?
- Can a useful Business Action × Data matrix be derived while keeping unknown cells genuinely unknown?
- Can current namespace / ownership / Application boundaries produce useful Subject Area / Subsystem views without new Canonical nouns?
- At realistic scale, are effective CUD and call dependencies understandable without an explicit Subsystem semantic contract?
- What minimum Business Flow / DFD representation, if any, is required by a realistic TEI Reference without collapsing it into Procedure Handoff or Runtime Workflow?
- Does protected-target / concern-vs-realization meaning recur across realistic TEI domains strongly enough to justify a broader reference-role abstraction?

## 5. Update rule

When an external reference materially affects a TEI decision:

1. identify the public or independently reproducible evidence;
2. state what it supports and what it does not prove;
3. distinguish observed behavior from recommendation;
4. connect the question to TEI-local Reference / test evidence before promoting a new semantic concept;
5. avoid relying on private chat history or temporary handoff artifacts as public justification.
