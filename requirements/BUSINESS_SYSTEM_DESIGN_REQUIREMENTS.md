# TEI Business-System Design Requirement Interpretation

Status: current requirement interpretation baseline

## 1. Position of this document

This document formalizes the business-system design interpretation developed in the "渡辺幸三の業務システム設計" discussion series and connects it to the existing `TEI_PRODUCT_REQUIREMENTS.md`.

It does **not** create new Product Requirement IDs. The authoritative Product Requirement identifiers remain `RQ-001` through `RQ-024` in `TEI_PRODUCT_REQUIREMENTS.md`.

This document fixes how the existing Requirements are to be read when TEI is used for business-system design, especially where Data, Function / Interaction, Business Procedure, Editor, Driver, Reference Model, and cross-view validation interact.

When a design choice is ambiguous, use the following fallback order:

1. the design intent expressed by Kozo Watanabe around business-system design and the three-element analysis method,
2. public X-TEA Modeler / Driver manuals and source code,
3. X-TEA / CONCEPTWARE-style reference-model evidence,
4. TEI-specific Product Requirements, Reference scenarios, and observed Evidence.

The fallback is a design-origin check, not an instruction to reproduce X-TEA implementation details. TEI naming, authority boundaries, Target-neutrality, Provider isolation, and current Evidence remain controlling constraints.

## 2. Three viewpoints are required, but must not become three sources of truth

The same business system must be reviewable from at least the following three viewpoints.

```text
Data viewpoint
Function / Interaction viewpoint
Business viewpoint
```

These correspond to the historical data model, function model, and business model viewpoints.

They are not three independent semantic authorities. TEI must preserve one accepted meaning graph and allow the three viewpoints to constrain and criticize each other through explicit references and derived projections.

This requirement interprets primarily:

- `RQ-001` implementation-independent business meaning,
- `RQ-003` no implicit multiple authorities,
- `RQ-010` high-density Editor,
- `RQ-011` multiple Projections,
- `RQ-017` minimize human double-entry,
- `RQ-021` Reference / Conformance.

A matrix, business manual, UI, diagram, generated document, or AI summary must not silently become a second source of truth when the same facts can be derived from accepted identities and relations.

## 3. Data viewpoint

The Data viewpoint represents stable business meaning around data, relationships, types, constraints, and lifecycle independently of physical storage.

It must support cross-checking such questions as:

- What business facts exist?
- Which facts identify or relate business subjects?
- Which relationships are one-to-one, one-to-many, or otherwise constrained?
- Which constraints are semantic and which are physical realization details?
- Which Application / Function / Procedure depends on the meaning?

Physical table names, foreign keys, indexes, provider catalogs, and migration mechanics remain realization concerns unless explicitly accepted as business meaning.

The historical preference for data-first design is retained, but **data-first must not become data-only**. A data model that cannot explain the work and interactions around it is incomplete as a business-system design.

## 4. Function / Interaction viewpoint

The Function viewpoint includes the human- and machine-visible interaction needed to understand system behavior. It is not limited to a list of backend procedures.

Depending on the system, it may include:

- screen / form interaction,
- report or document output,
- input / output definitions,
- browse / record / reference / header-detail / command Application intent,
- query and capability use,
- batch or API interaction,
- function-to-function invocation relationships,
- target-specific UI projections.

This viewpoint interprets primarily `RQ-012`, `RQ-013`, and `RQ-014`.

UI is especially important as a design-review Projection because a user can criticize a visible interaction more easily than an abstract model. However, UI layout, widget choice, Web framework, Wails API, or another concrete Target mechanism must not become the Canonical authority for business meaning.

A Function or Application Pattern may realize part of a business action, but the two are not identical.

```text
Business action != Application Function
```

A valid business action may have no direct Application Pattern or Capability realization.

TEI must not invent fake executable Functions merely to make every business step look automated.

## 5. Business viewpoint

The Business viewpoint describes how work is undertaken in the surrounding social system, including work that remains manual, conversational, physical, external, or unresolved.

At minimum, a sufficient business-work description may need to distinguish:

- purpose,
- trigger / event,
- actor or job responsibility,
- business action / step,
- order or presentation of steps where order matters,
- decision or judgment,
- manual instruction,
- use of Application Function / View / Pattern,
- handoff,
- exception,
- completion or continuation condition,
- unresolved information.

These fields are conceptual requirements for review. This document does **not** require each field to become a dedicated Canonical primitive or `.tei` syntax.

The following separation is normative:

```text
business actor / job label != authentication principal != authorization role / policy
```

Describing a person as a "sales operator" or equivalent business actor must never grant execution authority by itself.

Likewise:

```text
business judgment != deterministic Validation
```

A set of machine-checkable constraints passing must not be promoted to a whole-business approval unless an accepted semantic contract explicitly says so.

## 6. Business flow and Procedure are related but different

TEI must not reduce the Business viewpoint to a simple sequence of screen transitions or to a runtime Workflow Engine.

The historical business-flow view is closer to a DFD-like representation of business processes / jobs, information or data sources, and flows between them. Business work is initiated by events / circumstances and should be capable of loose coupling through information and events rather than requiring one monolithic end-to-end execution graph.

A Procedure is the closer-grained description of how a particular business activity is carried out, including manual work and Function / UI use.

Therefore:

```text
Business Flow != Procedure != Runtime Workflow Orchestration
```

TEI may eventually provide Projections for all three, but it must not collapse them prematurely.

Retry, timer, queue, distributed saga, process lease, and other runtime orchestration mechanics are not implied merely because a business Procedure or Business Flow exists.

## 7. Time must be modeled where the business needs it, not imposed globally

Time is important for many business questions: event order, validity, history, lead time, bottleneck discovery, and temporal resource use.

However, ordinary business systems must not be forced into event sourcing, bitemporal storage, eventually consistent distributed workflows, or append-only ledgers merely because TEI can express time.

Use time-related semantics when the domain and Reference evidence require them. Preserve conventional transactional database semantics where they are sufficient.

This keeps the business model richer without turning a specialized temporal architecture into a universal Core assumption.

## 8. Cross-view criticism is part of requirement satisfaction

The three viewpoints must be able to expose defects in each other.

Representative checks include:

```text
Business -> Function
  a required business action has no supported interaction or intentional manual path

Business -> Data
  a step requires information not represented in accepted Data meaning

Function -> Data
  a visible editable concept has no accepted write meaning

Data -> Function
  important Data or Relation meaning is never exposed where users must inspect or maintain it

Function / Business -> Authorization
  a convenient UI or actor label would exceed accepted authority
```

The correct response to a discovered contradiction is to return to the authoritative source and Shared Acceptance Boundary, not to patch a derived Projection until the contradiction disappears visually.

## 9. Matrix and manual outputs are derived review instruments by default

The historical X-TEA practice of matrices such as Business × Function, Business × Table, and Table × Function is retained as an important review technique.

TEI should derive such matrices from explicit references wherever possible.

Examples include:

- Business Action × Application Pattern,
- Business Action × Data / Relation,
- Procedure Step × View / Action,
- Application Pattern × Data / Query / Capability,
- Data / Relation × affected Procedure / View / Function.

Unknown relationships must remain unknown rather than being inferred from names for presentation convenience.

A manually enriched explanation may exist, but generated facts and human residual explanation must remain distinguishable.

## 10. Editor interpretation

`RQ-010` must be read as a requirement for a dense business-system design environment, not a wizard that serializes one small form at a time.

The Editor should allow a developer to inspect locally connected meaning across multiple levels, including Data, Relation, Rule, View, Application intent, Procedure / business context, source, and Evidence.

At least three levels of relationship must be inspectable without forcing every relationship into separate isolated screens.

The Editor remains a Projection / authoring client. It must not become a second Canonical authority.

A useful Editor should make the three-viewpoint tensions visible rather than hiding them behind automatic generation.

## 11. Driver interpretation

`RQ-012` must be read in the lineage of an executable specification environment.

The Driver should make accepted Application meaning inspectable and executable early enough to test model coherence before a final production Target is generated or hand-coded.

This does not mean all business work is executable. The Driver validates the system-realizable portion while preserving manual / external business actions as business meaning outside direct execution.

Driver evidence must therefore distinguish at least:

- semantic meaning,
- deterministic validation,
- authorization,
- command request and Business Result,
- execution error,
- separately observed external state.

## 12. Application Pattern interpretation

`RQ-014` Application Patterns represent reusable target-independent Application intent such as `browse`, `record`, `reference`, `header-detail`, and `command`.

They are part of the Function / Interaction viewpoint. They do not by themselves define the surrounding business Procedure.

A Procedure may reference multiple Application Patterns; multiple Procedures may reuse one Pattern; a business action may reference none.

Pattern ordering must not silently be treated as a business procedure unless an accepted business model explicitly establishes that ordering.

## 13. Reference Models are executable design content, not only fixtures

`RQ-021` Reference / Conformance must eventually cover realistic business-system design pressure, not only technical fixtures.

A Reference Model should be able to serve simultaneously as:

- a reproducible Conformance input,
- an example from which a developer can learn business-system modeling,
- a pressure test for Editor / Driver / Target boundaries,
- a way to improve both the platform and the model through cross-view criticism.

CONCEPTWARE-style reference models are historical design evidence. Their structure and pressure should be studied, but their content must not be copied into TEI until licensing and provenance permit it.

Before claiming broad business-system design maturity, TEI should be tested with a reference larger than a toy form: multiple subject areas / subsystems and enough Data, business actions, Functions, and cross-relations to expose locality and traceability problems.

The exact gate and size remain planning decisions, not fixed numerical Product Requirements in this document.

## 14. AI assistance follows the same design authority

AI may help explore existing semantic knowledge, propose a Procedure, produce a matrix, draft a UI, or point out three-viewpoint inconsistencies.

It must not create a separate semantic authority.

AI proposals must cite or structurally reference the accepted identities they use, distinguish known facts from assumptions / unresolved points, and pass the same Shared Acceptance Boundary as human or Editor changes.

This interprets `RQ-015`, `RQ-016`, `RQ-017`, and `RQ-021`.

## 15. Historical X-TEA naming is provenance, not current TEI product naming

X-TEA, XEAD, XeadModeler, XeadDriver, and related names are used as historical and architectural provenance.

New TEI implementation symbols, file formats, namespaces, generated artifacts, and public product naming continue to use TEI naming.

The goal is to preserve and advance the design knowledge, not to recreate the historical product surface.

## 16. Explicitly unresolved architecture questions

The following questions remain open and must be decided from Reference evidence rather than terminology preference:

1. whether Business Procedure requires a durable Canonical identity or can remain a derived / traceable design artifact,
2. which parts of actor, event, handoff, decision, and manual instruction require structured semantics,
3. how a DFD-like Business Flow should relate to Procedure, Data, Application Pattern, and Event without becoming a runtime Workflow Engine,
4. which Subject Area / Subsystem concepts deserve durable identity in the TEI model,
5. how much target-neutral View / Interaction structure is needed before Target adapters,
6. how realistic Reference Models should be packaged, versioned, and licensed,
7. where temporal semantics become part of accepted domain meaning versus optional analysis / realization.

Until evidence resolves these points, documentation and experiments may name them, but Core schema and Product Requirement IDs must not be expanded merely to mirror X-TEA vocabulary.
