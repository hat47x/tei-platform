# X-TEA Design Evidence Register — 2026-09

- Status: Lane C research evidence register
- Purpose: preserve reusable evidence from Kozo Watanabe / Three-Element Analysis / X-TEA source / Reference Models so TEI design decisions can return to concrete provenance instead of relying on recollection
- Authority: research evidence only; this file does not override TEI Product Requirements, ADRs, or accepted Canonical meaning
- Related:
  - `UPSTREAM_XTEA_DESIGN_FALLBACK_PROTOCOL.md`
  - `UPSTREAM_BUSINESS_SYSTEM_DESIGN_VITALITY_AUDIT_2026-09.md`
  - `ORDER_ENTRY_BUSINESS_PROCEDURE_REFERENCE_EXPERIMENT.md`

## 1. Evidence notation

```text
W1  explicit statement by Kozo Watanabe
M1  documented X-TEA / XEAD model or manual structure
S1  public XeadModeler / XeadDriver source-code behavior
R1  one Reference Model example
R2  repeated observation across multiple Reference Models
T1  TEI-local Reference experiment
C1  accepted TEI Conformance / deterministic diagnostic
```

The register intentionally separates:

```text
historical intent
historical implementation
real content pressure
TEI-local decision
```

A historical concept is not automatically a TEI concept.

---

## E-001 — Business Model is a business manual, not merely software flow

**Grade:** W1

**Source:**
- Kozo Watanabe, 「データ、UI、業務手順の『3要素』をとらえる」, 2006, part 3/3
- https://www.itmedia.co.jp/im/articles/0605/09/news085_3.html

**Observed claim:**

The third element, Business Model, is explicitly described as a business manual. A basic design without the business manual is criticized as a design for a tool whose use is unknown. The business system is described as a kind of social system.

**TEI pressure:**

- Application / Driver meaning alone is not necessarily the full business procedure.
- Manual work, actors, triggers, and system use belong to design review.
- Business procedure must not be reduced to Runtime orchestration.

**Current TEI decision:**

Business Procedure **viewpoint is required** for Lane C. Whether `Procedure` needs durable Canonical identity remains under experiment.

---

## E-002 — Business definition has structured identity, event, responsible person, and function-use procedure

**Grade:** W1

**Source:**
- Kozo Watanabe, 「蔓延する『ブルシットエンジニアリング』」, 2024-07-05
- https://dbconcept.hatenablog.com/entry/2024/07/05/143207

**Observed structure:**

Watanabe gives a concrete structured form for a business definition / business manual:

```text
- ID / name / summary
- occasion / event in which the business is performed
- responsible person
- function definitions used and their usage procedure
```

He also states that business-to-table CRUD can be derived through the function definitions rather than manually maintained as a separate artifact.

**TEI pressure:**

If TEI adopts a durable lightweight Procedure later, the historically grounded minimum is closer to:

```text
identity
purpose / summary
trigger / event
actor / responsible label or reference
usage of existing Application / Function meaning
procedure / instruction
```

than to a Workflow Engine node graph.

**Important negative evidence:**

The article strengthens the **structured Business Model viewpoint**, but does not prove that current TEI needs a top-level `Procedure` Canonical object.

---

## E-003 — CRUD matrix is a derived projection from structured design information

**Grade:** W1 + S1

**Sources:**
- Watanabe 2024 article above
- `xead/XeadModeler/src/xeadModeler/DialogDocuments.java`
- `xead/XeadModeler/src/xeadModeler/DialogMatrixList.java`

**Observed implementation:**

XeadModeler contains code to generate/display CRUD-oriented documents and matrices from model information.

**TEI pressure:**

- Do not make CRUD matrices an authored second source of truth.
- Generate Function × Data / Business × Data views from explicit references and accepted operations.
- A matrix is valuable precisely because it exposes inconsistency without owning the underlying meaning.

**Current TEI direction:**

The `order_entry` Procedure experiment derives Business Action × Application Pattern and later Business Action × Data / Relation matrices.

---

## E-004 — Function definition is structured around UI, Data I/O, parameters, and function calls

**Grade:** W1

**Source:**
- Watanabe 2024 article above

**Observed structure:**

The article lists Function definition elements including:

```text
- ID / name / summary
- input / output parameters
- input / output with UI and tables
- functions launched from the function
```

**TEI pressure:**

The historical Function Model is not equivalent to a backend API catalog.

A TEI Function viewpoint may need to be reconstructed from:

```text
Application Pattern / View
Action / Capability
Query
Data / Relation references
CRUD / call projection
```

without introducing one mandatory symmetric `Function` Canonical noun.

---

## E-005 — Task Action and Application Function are explicitly separate in XeadModeler

**Grade:** S1

**Public source:**
- repository: `xead/XeadModeler`
- `src/xeadModeler/DialogImportXEAD.java`
- `src/xeadModeler/DialogDocuments.java`

**Observed source structure:**

- `TaskAction` is an explicit model element.
- `TaskFunctionIO` is nested / associated under task-action handling.
- `TaskFunctionIO` stores `FunctionID` and `IOID` references.

This is direct implementation evidence that a business action and a Function were not simply the same object under different labels.

**TEI pressure:**

```text
Business Action != Application Pattern != Capability
```

A business action may reference an Application Function realization; it need not be one.

**Current TEI experiment rule:**

`ORDER_ENTRY_BUSINESS_PROCEDURE_REFERENCE_EXPERIMENT.md` requires at least one manual / unresolved step with **no fake Application Pattern or Capability**.

---

## E-006 — Real XEA[D] content includes business actions with no Application Function realization

**Grade:** R2

**Source:**
- repository-independent research checkpoint retained in the project Library: `xtea_research_checkpoint_v0_2.md`
- underlying corpus: 12 XEA[D] models

**Observed count:**

```text
TaskAction total: 1606
TaskAction without Application Function realization: 202
```

**TEI pressure:**

The Task/Function distinction is not only a modeling-theory possibility. It occurs materially in actual model content.

**Design consequence:**

Do not require completeness of a Business Procedure by forcing every action to bind to executable software.

---

## E-007 — Subsystem is a design / cohesion boundary; external-table CUD is a smell

**Grade:** S1 + W/M lineage

**Public source:**
- `xead/XeadModeler/src/xeadModeler/Res_ko.java`
- `DialogMatrixList.java`

**Observed source text / behavior:**

XeadModeler guidance warns that Create / Update / Delete operations against external tables should preferably be avoided because subsystem independence and maintainability improve when they are not performed.

Modeler matrices also explicitly handle `Subsystem`.

**TEI pressure:**

- Subsystem is not merely deployment packaging.
- Cross-boundary C/U/D is a strong review signal.
- Cross-boundary read may be legitimate.

**Guardrail:**

Do not mechanically convert X-TEA Subsystem into a microservice, repository, deployment unit, or Database.

TEI should first try to derive:

```text
ownership
read dependencies
direct/effective CUD
exposed Actions / contracts
```

as a design projection.

---

## E-008 — SubjectArea and Subsystem are distinct historical model concepts

**Grade:** S1

**Public source:**
- `xead/XeadModeler/src/xeadModeler/DialogMatrixList.java`

**Observed source structure:**

The Modeler obtains both:

```text
Subsystem
SubjectArea
```

as separate element collections.

**TEI pressure:**

Do not assume that current TEI namespace, Application, Database, ownership boundary, and business subject area are all the same concept.

**Current status:** unresolved.

Lane C should test whether business subject-area views can remain derived before adding a durable `SubjectArea` concept.

---

## E-009 — Instance presentation order must not be confused with business time

**Grade:** S1 + R2

**Sources:**
- `xead/XeadModeler/src/xeadModeler/DialogEditInstanceOnDatamodel.java`
- `xtea_research_checkpoint_v0_2.md`

**Observed behavior:**

Historical instances retain independent `Text` and `SortKey` fields.

Reference content shows that slide/sort progression can mean different things:

- one example represents business-state progression
- another represents explanatory presentation order

**TEI pressure:**

Split at least conceptually:

```text
Dataset
Snapshot
Scenario
Expectation
Presentation Order
```

Do not infer temporal semantics from UI sort order.

---

## E-010 — State is often derived from underlying facts, not represented by one status column

**Grade:** R2

**Source:**
- `xtea_research_checkpoint_v0_2.md`
- `xtea_state_model_evidence_v0_1.json`

**Observed corpus facts:**

Across nine XEAF models / 902 Table Scripts, the research pass found repeated state / transition behavior, including lifecycle changes around completion, cancellation, confirmation, approval, inspection, shipping and receiving.

The retained research explicitly distinguishes:

```text
State predicate
Transition
Transition Action
Consequence Rule
```

and recommends deriving state from base facts where appropriate.

**TEI pressure:**

- do not assume one mutable `status` field is the state model
- do not collapse action, transition, and consequence into one callback
- state should be testable from representative Dataset / Scenario evidence

**Current Lane C note:**

This is important for later realistic Reference pressure but is not required to add new State concepts during the first `order_entry` Business Procedure experiment.

---

## E-011 — Table Script is not one semantic category

**Grade:** R2 + S/M lineage

**Source:**
- `xtea_research_checkpoint_v0_2.md`

**Observed analysis:**

Historical Table Scripts cover at least different semantic roles:

```text
Design Function
Consequence Rule
Operational Rule
```

The research found 902 Table Scripts overall but only a much smaller set corresponding to basic-design-level reusable functions.

**TEI pressure:**

Do not expose raw lifecycle hook names as public Capability contracts.

Classify the meaning first; place provider- or lifecycle-local mechanisms below the semantic boundary.

---

## E-012 — Reference Models are content that improves both method/tool and the model itself

**Grade:** W1 + R lineage

**Sources:**
- Watanabe's public writing on executable specification / Reference Models
- prior project audit `XTEA_REFERENCE_MODEL_VITALITY_AUDIT_2026-08-21.md`
- `REFERENCE_MODEL_GATE_PLAN_v1.md`

**Observed principle:**

The practical value of a Reference Model is not only demonstration. Building/running realistic content exposes platform limitations and model defects.

**TEI pressure:**

Use both:

```text
small deterministic fixtures
and
realistic multi-subsystem Reference content
```

Do not claim business-design maturity from toy fixtures alone.

---

## E-013 — XeadModeler / XeadDriver public source is reusable research material, Reference content licensing is separate

**Grade:** S1

**Sources:**
- `xead/XeadModeler/LICENSE`
- `xead/XeadDriver/LICENSE`

**Observed license:**

Both public repositories contain BSD-3-Clause-style redistribution / modification terms with attribution and no-endorsement conditions.

**TEI research rule:**

- source code may be inspected and adapted under its license
- do not assume downloadable Reference Models, Derby databases, manuals, or third-party reproductions have the same license
- preserve provenance and independently recreate test content until an artifact's exact reuse terms are confirmed

---

## E-014 — X-TEA Business Flow is a separate data-flow model, not Procedure handoff shorthand

**Grade:** W1 + M1 + S1

**Sources:**
- X-TEA HOME / Three-Element Analysis portal: X-TEA Modeler is described as supporting business flows, ER diagrams, business manuals, and other design drawings
- Kozo Watanabe, 「データ、UI、業務手順の『3要素』をとらえる」, 2006, part 3/3
- public repository `xead/XeadModeler`
  - `src/xeadModeler/DialogDataflowNode.java`
  - `src/xeadModeler/DialogDataflowLine.java`
  - `src/xeadModeler/DialogMatrixList.java`

**Observed source structure:**

XeadModeler treats Dataflow nodes / lines as their own model structure.

A Dataflow node whose `Type` is `Process` is tied to a `TaskID`. The source also distinguishes multiple non-Process node types and role-linked nodes rather than modeling every node as a software function.

`DataflowLine` separately connects Dataflow nodes and carries its own line-level meaning / presentation attributes.

Conceptually:

```text
Task / Process node
External / role / storage-like node
        ↕
DataflowLine
```

This is different from a lightweight Procedure handoff such as:

```text
Procedure/Step
  -> Procedure/Step
  + condition
```

**TEI pressure:**

- Keep `Business Flow != Procedure Handoff`.
- Do not promote current H1/R3 handoff pressure into a complete Business Flow / DFD model.
- A future Business Flow model must be able to express information / document / business-data flow and non-Procedure participants when Reference Evidence requires them.
- Do not introduce a generic Core `Edge` merely to make handoff and Business Flow share one storage shape.

**What this evidence does not prove:**

- It does not prove TEI needs a production `BusinessFlow` Canonical kind now.
- It does not require cloning X-TEA DataflowNode / DataflowLine XML or icon vocabulary.
- It does not prove the current X-TEA node taxonomy is the right TEI taxonomy.
- It does not make every historical flow line a durable semantic identity.

**Current TEI decision:**

Current H1/R3 remains **handoff-only**. Full Business Flow / DFD remains a separate unresolved design pressure and will not be smuggled into the Business Procedure contract.

---

## E-015 — Task/Function separation does not prove a generic protected-target role taxonomy

**Grade:** S1 + T1

**Sources:**
- public repository `xead/XeadModeler`
  - TaskAction handling
  - TaskFunctionIO handling with `FunctionID` / `IOID`
- `BUSINESS_PROCEDURE_O1_AUTHORITY_DIRECTION_PRESSURE_2026-09.md`
- `authorization_target_authority_direction_pressure_test.go`

**Observed X-TEA source structure:**

Historical XeadModeler distinguishes Business-side `TaskAction` from associated `TaskFunctionIO`. The latter points to `FunctionID` / `IOID` rather than collapsing the business action and software function into one identity.

This supports:

```text
Business Action != Application Function
Business Action may use / refer to Function or IO
```

**What this evidence does not prove:**

It does **not** prove a generic TEI role taxonomy such as:

```text
realizes
protects
supports
concerns
checks
...
```

In particular, the historical Task/Function usage relation is not evidence that every TEI cross-view concern should become `ExplicitSemanticRef { Kind, Role, Target }`.

**TEI-local pressure:**

Current `order_entry` BP-06 exposes a real local distinction:

```text
supporting authority   = security.authorize
protected target       = order_entry.submit_order
direct Function        = none
```

Legacy Authorization Reference also contains an `execute / order_entry.submit_order / allow` rule, but this is security-side authority only.

Therefore:

```text
Authorization policy may corroborate Business protected target
Authorization policy must not author/backfill Business protected target
```

Policy drift is exposed as a test-only cross-view diagnostic rather than silently rewriting Business meaning.

**Second-corpus result:**

The provenance-backed manufacturing / inventory outline strongly repeats `Command != physical actuality`, but does not yet provide a second exact Application Pattern case where target / concern differs from direct realization.

**Current TEI decision:**

- keep O1 narrow and test-only
- do not infer O1 target from Authorization policy
- do not add production `applicationPatternTargets` yet
- do not generalize a ref-role enum from one exact case
- require further realistic Evidence before role taxonomy promotion

---

# 2. Current conclusions supported by the register

## C-A — Preserve the Business Procedure viewpoint unconditionally

Supported by E-001, E-002, E-005, E-006.

What remains unproven is not the viewpoint but whether it must be a durable Canonical object in current TEI.

## C-B — Keep Business Action separate from software realization

Supported strongly by E-005 and E-006.

Invalid shortcut:

```text
Business step -> invent Capability so the graph is executable
```

Preferred:

```text
Business step
  ├─ manual
  ├─ external
  ├─ unresolved realization
  └─ references existing Application / Capability when actually realized there
```

## C-C — Prefer derived matrices / views over duplicate authored documents

Supported by E-002, E-003, E-004, E-007.

Candidate TEI projections:

```text
Business Action × Application Pattern
Business Action × Data / Relation
Function / Action × Data CRUD
Subsystem / ownership × effective CUD
Function call graph
```

These projections should reveal conflicts, not own the meaning they display.

## C-D — Keep historical model nouns decomposable

Supported by E-007 through E-011.

Do not prematurely make these one-to-one Canonical imports:

```text
X-TEA Task       -> TEI Workflow node
X-TEA Subsystem  -> microservice
X-TEA SubjectArea-> namespace
X-TEA Instance   -> Scenario
X-TEA Script     -> Capability
```

Each historical noun carried more than one pressure or was used in more than one way.

## C-E — Keep Procedure handoff narrower than Business Flow

Supported by E-001 and E-014.

Current thin handoff pressure solves a specific accepted-meaning problem:

```text
Procedure/Step -> Procedure/Step + condition
```

It must not silently become the general Business Flow / DFD abstraction.

A future flow representation needs its own Reference pressure for activities, external parties, information / document / data movement, and any stable identities required by review or impact analysis.

## C-F — Keep ref-role generalization evidence-driven

Supported by E-005, E-006, and E-015.

`Business Action != Function` is strongly supported. A generic `protected/supports/concerns/...` role taxonomy is not.

Current O1 is a real TEI-local cross-view pressure, but remains one exact protected-target case. The manufacturing / inventory outline does not yet convert it into a repeated cross-domain role pattern.

# 3. Open evidence gaps

The following still need TEI-local proof:

```text
T-GAP-01
Can a derived Procedure preserve accepted business meaning and review history without durable Procedure identity?

T-GAP-02
Can Procedure / UI cross-view review detect the current read-only `master` boundary violation without duplicating datasource meaning?

T-GAP-03
Can a Business Action × Data matrix be derived far enough from existing Pattern/Query/Capability references to be useful while keeping unknown cells genuinely unknown?

T-GAP-04
Can current namespaces / ownership / Application boundaries produce a useful SubjectArea / Subsystem view without new Canonical nouns?

T-GAP-05
At realistic scale, do effective CUD and call dependencies remain understandable without an explicit Subsystem semantic contract?

T-GAP-06
What is the minimum TEI Business Flow / DFD representation, if any, once a realistic Reference requires information flow across activities, external parties, and stores without collapsing it into Procedure handoff or Runtime Workflow?

T-GAP-07
Does protected-target / concern-vs-realization meaning recur across realistic domains strongly enough to justify any generic ref-role taxonomy, or should O1 remain a field-specific/local relation?
```

`ORDER_ENTRY_BUSINESS_PROCEDURE_REFERENCE_EXPERIMENT.md` addresses T-GAP-01 through T-GAP-03 first. T-GAP-06 stays separate from the current handoff contract pressure. T-GAP-07 remains open after the manufacturing / inventory outline audit because that outline does not yet contain exact Application identities sufficient to prove a second protected-target case.

# 4. Update rule

When new X-TEA / Watanabe evidence is used in a material TEI decision:

1. add or refine an evidence entry
2. mark evidence grade and provenance
3. state what the evidence supports
4. state what it **does not** prove
5. connect it to a TEI-local Reference / test before promoting a new semantic concept

This keeps “return to X-TEA” a reproducible engineering practice rather than an appeal to tradition.