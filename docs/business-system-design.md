# Business-System Design Architecture

Status: current architecture baseline

## 1. Purpose

This document records the business-system design architecture that TEI should preserve while its implementation evolves.

It formalizes the design line developed through the "渡辺幸三の業務システム設計" discussions and connects that line to the current TEI architecture.

The corresponding requirement interpretation is `requirements/BUSINESS_SYSTEM_DESIGN_REQUIREMENTS.md`.

This document is intentionally stricter about **meaning boundaries** than about concrete schema. It describes what must remain distinguishable before deciding which concepts become dedicated Canonical types or `.tei` syntax.

## 2. Design lineage and fallback

TEI starts from the lineage of Kozo Watanabe's business-system design work, the three-element analysis method, X-TEA Modeler / Driver, and reference-model practice.

When a design choice is ambiguous, the fallback order is:

```text
Watanabe design intent
    ↓
public X-TEA manuals / source
    ↓
reference-model evidence
    ↓
TEI Product Requirements / Reference / Evidence
```

The purpose of fallback is to recover the reason behind a design distinction. It is not a compatibility requirement with historical classes, XML, UI structures, or runtime implementations.

TEI remains free to use a smaller kernel, different authoring forms, different Targets, and different implementation technology where the original design meaning is preserved or improved.

## 3. One business system, three mutually constraining viewpoints

A business system is inspected through three main viewpoints.

```text
                  Business
                 /        \
                /          \
             Data -------- Function / Interaction
```

The three viewpoints are not a pipeline and not three independent models that are later merged heuristically.

They are different projections over accepted business meaning and should constrain one another continuously.

### Data viewpoint

Focuses on:

- business facts,
- types,
- identities,
- relations,
- cardinality,
- semantic constraints,
- lifecycle.

### Function / Interaction viewpoint

Focuses on:

- what a human or machine can ask the system to do,
- what information is shown or accepted,
- screen / form / report interaction,
- browse / record / reference / command behavior,
- batch / API behavior,
- queries and capability calls,
- function invocation relationships.

### Business viewpoint

Focuses on:

- why work starts,
- who or what is responsible for the work,
- what business action is performed,
- what is manual or external,
- what system interaction is used,
- what judgment, handoff, exception, and continuation exist.

A design is incomplete when one viewpoint cannot explain or criticize the others.

## 4. Data-first, but not Data-only

The historical data-first preference remains valuable because stable business data and relationships often outlive screens and implementation technology.

TEI therefore continues to keep Semantic Data and physical persistence separate.

```text
business Data meaning
        ↓
Persistence Mapping
        ↓
physical schema / Provider
```

However, Data-first does not mean that a sufficiently normalized data model completes business-system design.

A Data model must survive questions from the Function and Business viewpoints:

- Can a user actually inspect or maintain the required meaning?
- Does a business activity need information that the Data model does not represent?
- Does a visible edit imply a mutation that no accepted Data / Capability contract permits?
- Does the surrounding work reveal a missing relation or constraint?

The feedback loop is part of design, not a later UI implementation concern.

## 5. Function model includes UI, but UI is not authority

The Function viewpoint includes UI because UI makes a system understandable and criticizable by users and designers.

This may include:

```text
screen / form
report / document
input / output
query / browse
record interaction
reference interaction
header-detail interaction
command
batch
API
function call relationship
```

Current TEI Application Patterns such as `browse`, `record`, `reference`, `header-detail`, and `command` are target-independent pieces of this viewpoint.

A concrete Web screen, desktop window, report renderer, or other Target representation is a Projection.

```text
accepted Application meaning
        ↓
Presentation / Interaction projection
        ↓
Target adapter
        ↓
concrete UI / API / MCP / VUI
```

The Projection is allowed to reveal a design defect. It is not allowed to silently redefine the underlying meaning in order to hide the defect.

## 6. Business action and Application Function must remain distinct

One of the central architecture boundaries is:

```text
Business Action != Application Function
```

A person may perform a meaningful business action without using TEI at all.

A business action may be:

- manual,
- physical,
- conversational,
- performed in another system,
- supported by an Application Pattern,
- supported by a Capability,
- unresolved during design.

Therefore TEI must allow:

```text
Business Action -> no Application Function
```

without inventing an executable placeholder.

Likewise, one Application Function may support more than one business context, and one business action may use more than one Function.

This N:M relation is a more faithful architecture than treating the menu or screen hierarchy as the business model.

## 7. Business Flow is not merely a task sequence

The historical three-element Business Model should not be reduced to a flowchart where business tasks are connected only by control-flow arrows.

A useful Business Flow is closer to a DFD-like view in which business processes / activities and information or data-source nodes exchange flows.

Conceptually:

```text
Event / circumstance
        ↓
Business Activity A
        │
        ├── information / document / business data ──> Data source / storage / external party
        │
        └── information / event ──────────────────────> Business Activity B
```

Each business activity can have its own trigger / event and can be loosely coupled from other activities.

This matters because a business system is a social system, not simply one synchronous software transaction.

TEI must therefore keep these concepts distinguishable:

```text
Business Flow
Business Procedure
Application interaction
Runtime Workflow Orchestration
```

A future graph Projection may display them together, but that does not make them one semantic object.

## 8. Procedure describes how one area of work is carried out

A Procedure is the closer-grained business description around an activity.

A useful conceptual form is:

```text
Procedure
├─ purpose
├─ trigger / event
├─ actor / job context
├─ step / business action
│  ├─ manual instruction
│  ├─ optional View / Pattern / Capability reference
│  ├─ optional Data / Relation reference
│  └─ unresolved context
├─ decision / judgment
├─ exception
├─ handoff
└─ completion / continuation
```

This is an architecture vocabulary, not yet a fixed Canonical schema.

Current evidence has not decided whether Procedure requires a durable Canonical identity or can remain a traceable derived design artifact.

That decision must be made from Reference experiments, especially whether accepted business meaning is lost during regeneration, impact analysis, reuse, or review.

## 9. Manual work is first-class business meaning

Manual work must not be treated as an embarrassing gap to be automated away in the model.

If the business genuinely includes a judgment, phone call, physical inspection, conversation, external approval, or other non-Application action, the model should be able to say so directly.

This yields an important rule:

```text
no Function reference
```

can be a valid and intentional result for a business step.

Similarly, a deterministic Validation check does not automatically replace a human or organizational business judgment.

```text
Validation PASS != Business Approval
```

unless an accepted semantic contract explicitly defines that equivalence.

## 10. Actor and authority are separate axes

The Business viewpoint may describe an actor, job, department, or responsibility.

The security architecture separately deals with authenticated principals, memberships, scopes, policies, and authorization decisions.

```text
Business Actor / Job
        ≠
Authenticated Principal
        ≠
Authorization Role / Policy
```

A business-model label must never grant runtime authority by implication.

When a business Procedure says that a "sales operator" performs a step, authorization still flows through the normal TEI Capability / policy boundary.

This preserves compatibility with existing systems whose organization and authorization structures do not match a single universal Role table.

## 11. Function / Data / Business cross-checks

The architecture should support explicit cross-view criticism.

### Business -> Function

Question:

```text
How is this business action supported?
```

Valid answers include an Application Pattern, Capability, manual action, external system, or unresolved status.

### Business -> Data

Question:

```text
What information does this work depend on or change?
```

A missing business fact or relation should become visible.

### Function -> Data

Question:

```text
Does this visible operation have an accepted read / write meaning?
```

For example, an "edit" affordance must not manufacture write authority against a read-only datasource.

### Data -> Function

Question:

```text
Where is this important meaning inspectable or maintainable?
```

Unreachable business meaning may indicate an incomplete Function design.

### Business / Function -> Security

Question:

```text
Does the proposed interaction exceed accepted authority?
```

Convenient UI must not bypass security semantics.

## 12. Matrices are derived cross-check projections

X-TEA-style matrices remain a valuable design instrument.

TEI should prefer deriving them from explicit relations rather than authoring the same relationship twice.

Examples:

```text
Business Action × Application Pattern
Business Action × Data / Relation
Procedure Step × View / Action
Business Activity × Data source
Application Pattern × Query / Capability
Data / Relation × impacted Procedure / View
```

A matrix may have authored annotations, but the tool must distinguish:

```text
derived fact
human residual explanation
unknown
```

Unknown cells should not be automatically filled from naming similarity.

## 13. The Editor is a dense projection over connected meaning

The Editor should support the locality needed to perform the previous cross-checks.

It should not force Data, Relation, Rule, View, Application intent, Procedure, source, and Evidence into unrelated wizard pages merely because they are different types.

A useful design workspace can expose three or more levels of connected meaning while keeping the current subject in context.

Conceptually:

```text
Business Procedure
   └─ Step
       └─ Application Pattern
           ├─ Data / Relation
           ├─ Query / Capability
           ├─ Rule / Validation
           └─ Source / Evidence
```

or the reverse direction when starting from Data.

The Editor is still an authoring / projection client. The accepted file-native source and Shared Acceptance Boundary remain authoritative.

## 14. Driver as executable specification environment

The Driver exists to make design meaning executable and testable before final Target implementation.

It should be possible to exercise accepted Application meaning and distinguish:

```text
Authentication
Application interaction
Validation
Authorization
Command
Business Result
Execution Error
explicit read
Observation
```

This lineage is close to the X-TEA Driver idea that specifications should be executable without waiting for conventional generated application code.

TEI extends the idea by keeping Provider, authority, Target, and Evidence boundaries explicit.

The Driver executes the system-realizable portion. It does not require the surrounding whole business Procedure to become executable.

## 15. Target-neutral Presentation remains separate from business meaning

A Business Procedure may refer to a View or Application Pattern, but must not depend on a concrete Web component, Wails API, HTML template, or mobile widget unless that detail is itself accepted Target-specific meaning.

The same business meaning should be able to pressure at least two Targets.

```text
Business / Application meaning
        ↓
Target-neutral interaction intent
        ├─> Web Target
        ├─> Desktop Target
        ├─> Mobile Target
        ├─> API Target
        ├─> MCP Target
        └─> VUI Target
```

Target differences are handled through Target Capability and Execution / Trust Profile, not by modifying business meaning for convenience.

## 16. Time is an opt-in business dimension

Time can increase business resolution by exposing:

- event order,
- validity periods,
- state history,
- lead time,
- waiting time,
- bottlenecks,
- temporal resource usage.

But TEI must not impose a universal event-sourcing or bitemporal architecture on ordinary transactional systems.

Use temporal semantics where the domain requires them and conventional ACID transaction semantics where they are sufficient.

Likewise, distributed eventual consistency must not become the default mental model for a system that can be modeled accurately with a conventional database transaction.

## 17. Reference Model as executable learning and pressure content

A Reference Model is more than a technical fixture.

It should be useful as:

- a reproducible model,
- a learning example,
- a platform pressure test,
- a design-review target,
- a source of regression Evidence.

The desired feedback loop is:

```text
Reference Model
      ↓ exercises
TEI Platform / Editor / Driver / Target
      ↓ exposes weakness
Model or Platform correction
      ↓
Reference / Conformance re-run
```

This is close to the historical CONCEPTWARE role, but TEI must preserve provenance and licensing boundaries.

Reference-model structure may be studied before content is reused. Historical content must not be copied without confirmed permission.

A realistic future Reference should include enough subject areas / subsystems, Data, business actions, Functions, and relations to expose problems that a toy `order_entry` example cannot.

## 18. AI assistance is a critic and proposer, not authority

AI can improve business-system design by:

- searching existing semantic knowledge,
- proposing cross-references,
- drafting Procedure text,
- generating matrix / graph / UI Projections,
- detecting missing Data / Function / Business relationships,
- suggesting questions where the model is under-specified.

AI output remains a Proposal or derived artifact until accepted.

It should preserve the distinction between:

```text
known accepted fact
inference
proposal
unresolved question
```

and cite the Canonical identities / Source / Evidence it used.

The same Shared Acceptance Boundary applies to human, Studio, CLI, and AI changes.

## 19. What should not be pushed into the Core prematurely

The business-system design lineage does not justify adding every historical concept as a Micro Kernel primitive.

In particular, do not add a Core primitive only because X-TEA had a similarly named definition.

Candidate concepts such as Procedure, Event, Subject Area, Subsystem, View, Decision, or Handoff should be introduced at the smallest durable layer justified by TEI Reference evidence.

The kernel should continue to favor small reusable identity / type / relation / expression / constraint / lifecycle / provenance mechanisms where they are sufficient.

This preserves the original design pressure without recreating historical implementation structure.

## 20. Open evidence questions

The following questions remain intentionally unresolved.

### O-01 Procedure identity

Can a traceable derived Procedure preserve accepted business meaning, or does it require a durable identity?

### O-02 Business Flow representation

What is the minimum structure needed to represent DFD-like business activities, data / information sources, events, and flows without creating a runtime Workflow Engine?

### O-03 Actor structure

Which actor / job information must be structured, and which can remain descriptive while security authority remains separate?

### O-04 Decision / rationale

Which business judgments need durable Question / Decision / Rationale history, and which belong in ordinary requirements / Git history?

### O-05 Subject Area / Subsystem

Does TEI need durable subject-area or subsystem meaning beyond file organization and Application / Database boundaries?

### O-06 Target-neutral View

How much View / interaction structure must be accepted before Target adapters without recreating a universal widget DSL?

### O-07 Reference Model scale

What model size and domain diversity are sufficient to claim business-system design maturity rather than toy-model success?

### O-08 Temporal semantics

Which time concepts are business meaning, and which remain optional analysis or persistence realization?

These questions belong in Reference experiments, ADRs, and Evidence. They should not be resolved by terminology preference alone.

## 21. Relationship to current TEI documents

This architecture should be read together with:

- `requirements/TEI_PRODUCT_REQUIREMENTS.md`
- `requirements/BUSINESS_SYSTEM_DESIGN_REQUIREMENTS.md`
- `docs/architecture.md`
- `plan/UPSTREAM_ACCEPTANCE_SCENARIO_TRACEABILITY.md`
- `plan/APPLICATION_PATTERN_V1_DESIGN.md`

The existing Gate B / Gate C sequence remains unchanged.

Business-system design work must not silently turn the actual-provider Gate B work into a new gate, and must not count Studio's developer-facing Host as an Application Target.

Procedure / Business Flow experiments may continue in planning branches without changing the established Product Evidence sequence until their own Evidence justifies a formal requirement or architecture change.
