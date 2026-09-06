# TEI Upstream Business-System Design Vitality Audit — 2026-09

- Status: Lane C decision record / supersedes the initial planning-only snapshot
- Scope: business-system upstream modeling, three-view cross-validation, Business Procedure boundary, X-TEA / Reference Model pressure
- Non-goal: Gate B Evidence acquisition, Gate C Target implementation, new Workflow Engine, immediate Canonical-schema expansion
- Formalized result:
  - `docs/business-system-design.md`
  - `docs/architecture.md`
  - `requirements/TEI_PRODUCT_REQUIREMENTS.md` RQ-025 / RQ-026
- Experiment:
  - `ORDER_ENTRY_BUSINESS_PROCEDURE_REFERENCE_EXPERIMENT.md`
- Research provenance:
  - `UPSTREAM_XTEA_DESIGN_FALLBACK_PROTOCOL.md`
  - `XTEA_DESIGN_EVIDENCE_REGISTER_2026-09.md`

## 1. Audit question

Lane C asks:

> Can TEI help a designer form, challenge, explain, simulate, and revise a business-system design before implementation details become the source of truth?

The existing upstream trace audit checks whether Requirement / PAS / Plan / Reference / Evidence / Gate form a coherent chain. This audit checks whether the **business-system design vitality flowing through that chain is itself sufficient**.

## 2. Design lineage used as pressure

The primary historical lineage is Kozo Watanabe's business-system design practice:

- 『業務システムのための上流工程入門』
- Three-Element Analysis: Data / Function / Business
- XEAD / X-TEA Modeler / Driver
- executable specifications
- realistic Reference Model practice
- later data-first arguments

This lineage is not a schema specification for TEI.

When a TEI boundary is unclear, Lane C first asks what business-design failure the historical concept prevented, then checks Watanabe's statements, X-TEA manuals/model structure, public source, Reference Model usage, and finally current TEI Requirements / Architecture / Reference.

## 3. Findings that survived the audit

### 3.1 A business system requires multiple independent views

The retained views are:

```text
Data
  identity / relation / constraint / lifecycle / state / history

Function / Interaction
  View / Application Pattern / Query / Action / Capability / CRUD / calls

Business Procedure
  purpose / trigger / actor / business action / manual / external work /
  use of Application Functions
```

The value is their ability to criticize one another, not three symmetric Canonical nouns.

### 3.2 Business Action is not Application Function

XeadModeler source keeps `TaskAction` separate from Function and links them through `TaskFunctionIO` / Function references.

Prior X-TEA corpus analysis also found:

```text
12 XEA[D] models
TaskAction total: 1606
TaskActions without Application Function realization: 202
```

Therefore TEI must not force every Business Action into an Application Function or Capability.

Human work, AI work, external organizations, SaaS, physical operations, and existing-system work remain valid business design information even when TEI does not execute them.

### 3.3 Business Model means social-system operability

Business Procedure must explain more than an Application Driver sequence:

- why the work is performed
- what triggers it
- who / what is responsible
- what is manual or external
- which Application Functions are used and how
- what counts as business completion outside software acknowledgement

An actor is not automatically an Authorization Role.

### 3.4 Data-first is not Data-only

The useful loop is:

```text
rough Data / Relation
      ↓
View / interaction prototype
      ↓
Business Procedure / actor / manual review
      ↓
Dataset / Scenario / Driver / Simulator
      ↓
conflict / omission / mismatch
      ↓
return to authoritative source
```

A one-way `Data -> generated CRUD` path is insufficient.

### 3.5 Function design needs a user-comprehensible surface

UI / interaction is part of Function design because users and designers must be able to understand and criticize the proposed system.

UI remains a Projection rather than Canonical authority. If a UI implies an operation contradicted by Data / Contract / Authorization boundaries, the design must return to authoritative meaning rather than hiding the contradiction in Target-specific code.

### 3.6 Subsystem / Subject Area is a design boundary before it is a deployment boundary

Subsystem pressure is useful for cohesion, ownership, responsibility, cross-boundary Read/CUD, and exposed Action boundaries.

Cross-subsystem C/U/D is a strong review signal; controlled Read is not universally forbidden.

Current namespace / owner / Application / Database / Capability structures must be tested before adding a new Canonical `Subsystem` or `SubjectArea` kind.

### 3.7 Executable specification and Reference Model remain central

Driver / Simulator should expose defects while the artifact is still design.

Reference Models are not only fixtures. They are both:

1. platform/model pressure tests
2. reusable business-system design knowledge

Realistic content must be allowed to improve the platform and the model itself.

## 4. Initial gap and Reference result

The initial audit found that current TEI strongly represented:

- Meaning before Realization
- Canonical != Projection
- Native / legacy assets
- explicit Binding / Provider boundaries
- executable design verification
- AI Proposal != authority
- Reference / Conformance discipline

The underrepresented areas were:

1. Business Action / Procedure independent from Application Function
2. explicit three-view cross-validation

The `order_entry` Reference experiment then showed a useful minimal pressure without requiring new runtime orchestration:

- existing `recent_orders`, `order_record`, `customer_reference`, `order_lines`, `submit_order` remain Application meaning
- at least one valid Business Action can exist without Application Pattern / Capability
- actor can remain separate from Authorization Role
- manual / external context can surround Application use
- read-only `master` datasource creates a concrete cross-view contradiction if UI / Procedure attempts customer editing

This is sufficient Product-level evidence that the **capability to preserve Business Procedure and cross-view criticism is required**, even though the final Canonical representation of Procedure remains open.

## 5. Product Requirement decision

The initial snapshot deliberately deferred Product Requirement expansion until a small Reference made the missing value concrete.

That threshold has now been crossed.

Two Product Requirements are adopted:

### RQ-025 — Business Action / Procedure independent from Application Function

TEI must preserve business actions, actors, triggers, manual/external work, and use of Application Functions without forcing every action into Capability or runtime orchestration.

### RQ-026 — Three-view cross-validation

TEI must allow Data, Function / Interaction, and Business Procedure views to expose contradictions and route accepted corrections back to authoritative source.

These Requirements do **not** adopt a top-level Canonical `Procedure` object, Workflow Engine, or new Gate.

The requirement is the product value. The representation is still subject to Reference pressure.

## 6. Architecture decision

The following boundaries are now formalized in `docs/business-system-design.md` and `docs/architecture.md`:

```text
Authoring DSL != Canonical Semantic IR != Runtime

Business Action != Application Function != Capability != Realization

Business Procedure actor != Authorization Role

Capability existence != invocation authority

Data / Function-Interaction / Business Procedure
    = independent review views over shared authoritative meaning
```

Go remains a platform implementation foundation for Parser / Resolver / Validator / Analyzer / Compiler / Runtime / Plugin Host. Individual business meaning is not fixed to Go or framework APIs.

Studio / Editor remains an Authoring Client / Projection and does not become a second source of truth.

## 7. Diagnostic classification

Cross-view checks are not all compiler errors.

Use at least:

```text
semantic error
architecture / policy violation
review diagnostic
design question
informational impact
```

Candidate examples:

| From | To / concern | Initial classification |
| --- | --- | --- |
| Business Procedure | missing explicit View / Action ref | semantic error candidate |
| Procedure / UI | write through read-only datasource | architecture / policy violation |
| Procedure | required business information absent from Data | review diagnostic |
| Business actor | silently mapped to permission Role | architecture warning |
| Data change | affected View / Procedure | informational impact |
| command result | inferred external completion | policy violation |

Severity becomes universal only after realistic Reference Evidence reproduces the defect class.

## 8. What remains open

The audit intentionally leaves these unresolved:

- whether `Procedure` needs durable Canonical identity
- whether `Subsystem` / `SubjectArea` needs a new Canonical kind
- the minimum portable Authorization fact contract
- which Business Procedure checks become deterministic Conformance
- the final State / Transition first-class boundary

These are not gaps to fill by speculation.

Use `order_entry` first, then a realistic multi-subsystem Reference, and add Core concepts only when existing identities / derived Projections cannot preserve the required meaning safely.

## 9. Lane C next work

1. turn the `order_entry` Business Procedure experiment into a concrete derived reference artifact
2. generate Business Action × Application Function and Business Action × Data/Relation views
3. reproduce the read-only customer-edit contradiction as a cross-view diagnostic
4. determine whether stable Procedure identity is actually required for traceability / acceptance
5. keep hard-error versus review-diagnostic severity provisional until reproduced
6. expand to a multi-subsystem Reference only after the small experiment produces concrete pressure

## 10. Decision summary

Lane C now concludes:

> TEI already preserves much of the Data / Function / executable-specification vitality of the X-TEA lineage. The missing Product-level value was not a Workflow Engine or another symmetric metamodel object; it was the ability to preserve the Business Model as a social procedure and to let Data, Function / Interaction, and Business Procedure actively criticize one another.

That value is now formalized as RQ-025 / RQ-026 while the smallest correct Canonical representation remains an evidence-driven architecture question.
