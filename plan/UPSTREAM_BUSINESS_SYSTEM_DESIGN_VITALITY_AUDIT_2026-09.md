# TEI Upstream Business-System Design Vitality Audit — 2026-09

- Status: planning audit snapshot / Lane C
- Scope: business-system upstream modeling, three-view cross-checking, Procedure boundary, Reference Model pressure
- Non-goal: Gate B Evidence acquisition, Gate C Target implementation, new Workflow Engine, immediate Canonical-schema expansion
- Related baseline:
  - `UPSTREAM_ACCEPTANCE_SCENARIO_TRACEABILITY.md`
  - `UPSTREAM_TRACE_AUDIT_2026-09.md`
  - `V1_REFERENCE_APPLICATION_PLAN.md`
  - `PRODUCT_REQUIREMENT_GAP_AUDIT.md`

## 1. Purpose

This audit asks a different question from the existing upstream trace audit.

The existing trace audit checks whether Product Requirements / PAS / Plan / Reference / Evidence / Gate decisions form a coherent chain. This Lane C audit instead checks whether the **business-system design vitality that must flow through that chain is sufficiently represented**.

The primary external design lineage used as a pressure source is Kozo Watanabe's body of work on business-system upstream design, especially:

- 『業務システムのための上流工程入門』
- 「三要素分析法」series: Data Model / Function Model / Business Model
- “executable specification” / XEAD-XEAF-X-TEA lineage
- Reference Model library practice
- later data-first design arguments

The audit does not treat this lineage as a specification to copy. It extracts design pressures and tests whether current TEI preserves the underlying design value.

No new Product Requirement, PAS, Gate, Canonical object, or implementation backlog is created merely because an external method contains a named concept.

## 2. Lane C boundary

Lane C owns the following question:

> Can TEI help a designer form, challenge, explain, simulate, and revise a business-system design before implementation details become the source of truth?

Lane C therefore focuses on:

1. upstream triage and modeling sequence
2. business / function-interaction / data views
3. cross-view criticism and consistency checks
4. manual work, actors, triggers, and social-system operability
5. data-first but not data-only feedback loops
6. subsystem / subject-area design units
7. executable design verification without production side effects
8. realistic Reference Models as both platform pressure and reusable design knowledge
9. AI as exploration / critique / proposal support, not semantic authority

Lane C does **not** add blockers to the current Gate B three-Evidence convergence. It also does not redefine Gate C Semantic / Functional / Visual Evidence.

## 3. External design pressures retained from Watanabe lineage

### 3.1 Upstream design is not a linear transcription exercise

The upstream process includes requirements triage, rough framing, divergent basic design, iterative modeling, review, and acceptance. Basic design is not simply a deterministic conversion from a requirements list.

TEI implication:

- do not force `Requirement -> generated application` as the canonical upstream path
- preserve alternatives, unresolved items, and reviewable design candidates
- use deterministic validation for accepted semantics without pretending that design discovery itself is deterministic

### 3.2 A business system must be seen from multiple independent views

The three-view pressure is:

- **Data Model** — stable information structure, identity, relations, constraints, state
- **Function Model** — user-understandable interaction plus operations / calls / CRUD effects
- **Business Model** — business procedure / manual, actor, trigger, manual work, use of system functions

The value is not the three labels themselves. The value is that the views are sufficiently independent to criticize each other.

TEI should therefore not require three symmetric top-level Canonical nouns. It should ensure that the three views can be projected and cross-checked.

Current candidate mapping from the previous X-TEA vitality audit remains useful:

| Design view | TEI projection candidate |
| --- | --- |
| Data | Data + Relation + Validation / lifecycle |
| Function / interaction | View + Action / Capability + CRUD/call projection |
| Business procedure | lightweight Procedure + actor/manual/trigger + View/Action references |
| Subsystem pressure | owner + effective CUD + exposed Action boundary + dependency view |

### 3.3 Function design needs a user-comprehensible surface

A Function Model is not only a list of backend operations. UI / interaction is an important design object because it allows users to understand and criticize the proposed system.

TEI implication:

- Projection / mockup should be usable during design, not only after semantic design is considered complete
- UI remains a Projection, not Canonical authority
- feedback from UI / interaction may reveal deficiencies in Data / Relation / Procedure and must have an explicit route back to authoritative design

### 3.4 Business Model means social-system operability

A business system is not complete when software operations are defined. It must explain how people, manual work, triggers, external organizations, and system interactions combine into a socially operable procedure.

TEI implication:

- `Application Pattern` and Driver scenario alone are not sufficient evidence of the business procedure
- `Actor` in a business procedure must not automatically mean an Authorization Role
- a manual step must not be forced into an Application Function or Capability merely so the model is executable
- a business task may be realized by a human, AI agent, SaaS, external organization, physical operation, or TEI application

### 3.5 Data-first is not data-only

Starting with Data can be a strong discipline because stable information structures constrain arbitrary screen / process design. But the model must be challenged by real data, UI, and business procedures and revised when those views expose defects.

TEI implication:

```text
rough Data / Relation
      ↓
Interaction / View prototype
      ↓
Procedure / actor / manual-step review
      ↓
Reference data / Scenario execution
      ↓
conflict / omission / mismatch
      └──────────────→ revise authoritative design
```

A one-way `Data -> generated CRUD` path would retain only the surface of data-first design and lose its feedback loop.

### 3.6 Subsystem is a comprehension and coupling boundary

Subsystem decomposition is not merely deployment packaging. It is a unit for understanding cohesion, ownership, and coupling. The classic pressure to avoid cross-subsystem CUD while allowing controlled reads is especially useful as a design smell rather than a universal hard rule.

TEI implication:

- distinguish business/design ownership from physical deployment units
- derive dependency / CUD projections from accepted meaning where possible
- flag cross-boundary mutation as a review point before turning it into a universal prohibition

### 3.7 Executable specifications shorten the design-feedback loop

The important value of executable specification is not “everything is code.” It is that a specification can be run or simulated early enough to expose defects while it is still design.

TEI already strongly aligns with this through `RQ-012` and Product Principle P15.

Lane C adds one pressure: executable verification must include the **business-procedure view**, not only an Application Driver sequence.

### 3.8 Reference Models should pressure both platform and design content

A realistic Reference Model can expose platform limitations and also improve the model itself. A Reference should therefore be more than a toy fixture or a conformance sample.

TEI implication:

- keep small fixtures for deterministic conformance
- add at least one realistic multi-subsystem business Reference before claiming business-design maturity
- treat Reference content as reusable design knowledge and learning material, while preserving licensing / provenance boundaries

## 4. Initial gap audit

### 4.1 Strongly aligned areas

| Pressure | Current TEI alignment | Initial judgment |
| --- | --- | --- |
| Meaning before realization | P1, RQ-001, explicit Binding / Provider boundaries | Strong |
| Canonical != Projection | P2, RQ-010/011 | Strong |
| Native assets / legacy reality | P5/P14, RQ-004/009 | Strong |
| Explicit realization instead of runtime guess | P6, current Driver / Contract work | Strong |
| Executable design verification | RQ-012, P15, Driver / Reference | Strong |
| AI proposal != authority | P12, RQ-015/016, ADR-0022 | Strong |
| Reference / Conformance discipline | P17, RQ-021 | Strong |

### 4.2 Partial areas

#### A. Three-view presence exists, but three-view **cross-criticism** is not yet a first-class Acceptance concern

Current TEI can represent Data, Relation, View, Action / Capability, Application Pattern, Validation, and multiple Projection forms.

What is still weakly specified is the rule that these views must actively challenge one another during upstream design.

Candidate checks to investigate:

- Procedure step references a View / Action that does not exist
- View / Action requires Data not reachable from the intended business context
- Action mutates Data outside the intended owner / subsystem boundary
- Data / Relation change invalidates a View or business procedure
- user-visible View implies an edit that has no accepted Action / Validation semantics
- business procedure expects a result / state transition that no Action / Capability can establish or observe
- Application Pattern exists but no business Procedure explains when / why an actor uses it

These are not all candidates for hard compiler errors. Lane C must classify each as one of:

```text
semantic error
architecture / policy violation
review diagnostic
design question
informational impact
```

#### B. Business Procedure is underrepresented in the current Product Requirement surface

`Procedure` appears in Editor-oriented requirements and prior semantic-surface work, but the current v1 reference scenario is primarily an Application / Driver chain:

```text
Authentication
 -> browse
 -> record
 -> reference
 -> header-detail
 -> Validation
 -> Authorization
 -> Command
 -> Business Result
 -> explicit read
 -> Observation
```

This is valuable Application evidence, but it does not answer:

- what business event triggered the work
- which human / organization is responsible for each step
- which steps are manual or external
- when the application is not used
- what information is handed off between actors
- what constitutes business completion outside software acknowledgement

Lane C should first prove this gap with a small Reference before proposing a new Product Requirement.

#### C. `Procedure` boundary needs revalidation, not expansion into Workflow Engine

The previous X-TEA vitality audit proposed a deliberately thin Procedure:

- purpose / trigger
- actor label
- manual instruction
- View reference
- Action reference

and explicitly excluded branch / timer / retry / orchestration semantics from the initial boundary.

Lane C keeps that bias.

The question is not “how do we build a BPM engine?” It is:

> What is the smallest non-executable business-procedure representation that restores the social-system view and enables cross-checking?

#### D. Reference Application is technically rich but business-domain pressure is still narrow

`order_entry` already pressures multiple Databases, external-observed master data, Relation traversal, header-detail, Validation, Authentication / Authorization, Business Result, Execution Error, and Observation.

It is therefore a strong technical Reference.

For business-design maturity it still needs either:

1. an explicit business Procedure around the current scenario, including manual / actor / trigger context; or
2. a second realistic Reference with multiple subject areas / subsystems and non-software steps.

Option 1 is the smaller next experiment and should be tried first.

#### E. Subject-area / subsystem modeling unit is not yet explicit upstream

Current namespaces, Application, Database, Provider, and deployment boundaries must not be silently equated with a business modeling unit.

Lane C should investigate whether an existing identity / ownership construct can support subject-area projection before adding a new Canonical `Subsystem` concept.

## 5. Proposed Lane C upstream design loop

This is a working method, not a mandatory lifecycle schema.

```text
1. Triage
   - goal / target situation / constraints
   - what must be designed now vs deferred
   - unknown / conflict

2. Rough business framing
   - subject areas / major business events
   - actors / external parties
   - high-level information flow

3. Data-first rough model
   - Data / identity / Relation / lifecycle / constraint
   - representative instances

4. Parallel divergent modeling
   - Procedure / actor / manual steps / triggers
   - View / interaction / Action / Capability candidates
   - Data refinements

5. Cross-view criticism
   - Data ↔ View/Action
   - Data ↔ Procedure
   - Procedure ↔ View/Action
   - ownership / CUD / boundary pressure

6. Executable design probe
   - Driver / simulator / mockup
   - no production credential or uncontrolled side effect

7. Review
   - technical consistency
   - user / domain comprehension
   - unresolved items remain explicit

8. Acceptance into authoritative source
   - shared Acceptance Boundary
   - reload / Conformance

9. Reference capture
   - reusable pattern / example / Evidence where justified
```

AI may assist every exploratory / review stage, but it must not bypass the shared Acceptance Boundary.

## 6. First Reference experiment for Lane C

Use the existing `order_entry` meaning rather than creating a new idealized sample.

Add **no new runtime orchestration** initially.

Model one business procedure around the existing scenario, conceptually:

```text
trigger: customer order is received
actor: sales clerk

manual / external context
    ↓
open recent orders / create-or-open order record
    ↓
reference customer
    ↓
review / edit order lines
    ↓
validate order
    ↓
submit order
    ↓
Business Result
    ↓
separate Observation where external state matters
    ↓
manual / external follow-up
```

The experiment should answer:

1. Can existing Semantic / Application identities be referenced without duplication?
2. Can manual work be represented without inventing a Capability?
3. Can actor remain distinct from Authorization Role?
4. Can trigger be represented without introducing workflow runtime semantics?
5. Can missing View / Action / Data / Validation references be diagnosed deterministically?
6. Can Procedure remain a Projection / design object while Driver execution stays separately owned?
7. Can UI / mockup feedback reveal a Data / Relation defect and route the change back through authoritative source?

If this experiment can be expressed entirely as a derived design artifact, do not add Canonical `Procedure` merely for symmetry.

If durable semantic identity is required for cross-view traceability and cannot be reconstructed safely, then reopen the lightweight `Procedure` Canonical candidate with Reference Evidence.

## 7. Candidate conformance matrix

The first matrix should distinguish hard semantic checks from design-review diagnostics.

| From | To | Example question | Initial severity hypothesis |
| --- | --- | --- | --- |
| Procedure | View | Does the referenced interaction exist? | semantic error if explicit ref |
| Procedure | Action / Capability | Does the referenced operation exist? | semantic error if explicit ref |
| Procedure | Data | Is required business information represented? | review diagnostic |
| View | Data / Relation | Can displayed / edited information be traced? | architecture / semantic depending on binding |
| Action | Data | Are intended state changes explicit and within ownership boundary? | architecture / policy |
| Action | Validation | Is a required business constraint enforced before mutation? | review / policy initially |
| Procedure | Authorization | Is actor incorrectly being treated as permission identity? | architecture warning |
| Data change | View / Procedure | What design artifacts become impacted? | impact information |
| Procedure result | Observation | Is external completion being inferred from command acknowledgement? | policy violation where applicable |

No severity should become universal until at least one realistic Reference demonstrates the defect class.

## 8. Product Requirement / PAS impact — not yet adopted

Initial judgment: do **not** add a new Product Requirement yet.

Potential future outcomes after Reference evidence:

### Outcome A — existing requirements are sufficient

If Procedure / business-flow design can be represented as a derived Projection while RQ-001 / 010 / 011 / 012 / 014 / 017 / 021 provide enough authority and acceptance semantics, keep Product Requirements unchanged.

### Outcome B — one existing requirement needs clarification

If the missing point is only that Driver / design verification must include manual / actor / trigger context, clarify RQ-012 or RQ-011 rather than creating a new requirement.

### Outcome C — a genuinely missing product value is proven

Only if the Reference shows that TEI cannot preserve the social-system business procedure without a durable semantic contract should a new Requirement or explicit Procedure contract be proposed.

The burden of proof is on adding the concept.

## 9. Relationship to current PAS

The likely mapping is cross-cutting rather than a new PAS:

- **PAS-01**: accept / reload business-design changes through shared Acceptance
- **PAS-02**: connect accepted Application meaning to executable Driver verification
- **PAS-04**: use Target Projection / mockup as a user-comprehensible design view without making it authority
- **PAS-06**: explore Reference patterns / existing semantic knowledge before inventing new design

A separate PAS should be considered only if no current PAS can express an observable user value such as “a developer and domain user can jointly validate a business procedure including manual and system steps.”

## 10. Reference Model Gate pressure

Before claiming mature business-system upstream support, TEI should be pressured by a Reference larger than isolated fixtures.

Candidate progression:

```text
Level 0  deterministic unit / fixture
Level 1  current order_entry Application Reference
Level 2  order_entry + explicit business Procedure / actor / manual context
Level 3  multiple subject areas / subsystems with controlled cross-boundary reads and writes
Level 4  realistic domain Reference containing dozens of Data / Action-scale elements
```

Levels are pressure stages, not product maturity scores.

Licensing matters. Public structural patterns may be reconstructed as an audit mapping, but proprietary Reference content must not be copied without confirmed permission.

## 11. AI role in Lane C

Preferred AI behavior:

1. read existing Canonical / Source / Evidence first
2. identify missing cross-view links and contradictions
3. generate multiple modeling candidates when design is genuinely divergent
4. explain why a candidate conflicts with existing meaning
5. show impact before acceptance
6. preserve `unknown` where evidence is insufficient
7. submit candidate changes through the same Acceptance Boundary as human / Studio / CLI authoring

Avoid:

- generating a complete application from prose and treating it as canonical
- filling missing manual / actor information by guess
- turning similarity / ranking into semantic evidence
- creating a parallel AI-owned business ontology
- equating autonomous execution with authority

## 12. Immediate Lane C work order

1. Keep Gate B / Gate C execution scope unchanged.
2. Reconstruct the smallest `order_entry` business-procedure view using existing identities.
3. Test whether the view can remain derived or requires durable Procedure identity.
4. Implement / prototype only the minimum cross-view checks needed to expose real defects.
5. Use a UI / interaction Projection or mockup as a design-review surface and verify the return path to authoritative Data / Relation / Application meaning.
6. Record which checks are deterministic semantic errors and which remain review diagnostics.
7. Reassess RQ / PAS only after the Reference experiment.
8. Expand to a multi-subsystem Reference only after the small procedure experiment exposes concrete pressure.

## 13. Sources used for the pressure audit

Primary / near-primary public sources:

- 渡辺幸三『業務システムのための上流工程入門』, 日本実業出版社, 2003.
- 渡辺幸三「データ、UI、業務手順の『3要素』をとらえる」, 2006.
  - https://www.itmedia.co.jp/im/articles/0605/09/news085.html
  - https://www.itmedia.co.jp/im/articles/0605/09/news085_2.html
  - https://www.itmedia.co.jp/im/articles/0605/09/news085_3.html
- 渡辺幸三「“実行可能な仕様書のライブラリ”を知的社会資本に」, 2011.
  - https://www.itmedia.co.jp/im/articles/1109/28/news134_3.html
- X-TEA historical site / public material as referenced by prior repository audit.

Repository-local prior audit:

- `XTEA_REFERENCE_MODEL_VITALITY_AUDIT_2026-08-21.md` (research-library artifact used during earlier architecture review)

## 14. Initial decision

Lane C does **not** currently recommend a new Workflow Engine, new Gate, or immediate Product Requirement expansion.

The first design hypothesis is:

> TEI already preserves much of the Data / Function / executable-specification vitality, but its current v1 Reference underrepresents the Business Model as a social procedure. The smallest useful next proof is to restore that view around `order_entry`, then use cross-view criticism to decide whether a durable `Procedure` contract is actually necessary.

This hypothesis remains provisional until the Reference experiment produces repository-local Evidence.
