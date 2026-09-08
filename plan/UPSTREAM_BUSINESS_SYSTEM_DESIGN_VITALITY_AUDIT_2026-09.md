# TEI Upstream Business-System Design Audit — 2026-09

- Status: Lane C decision record
- Scope: business-system upstream modeling, multi-view cross-validation, Business Procedure boundary, and Reference pressure
- Non-goal: Gate B Evidence acquisition, Gate C Target implementation, new Workflow Engine, or immediate Canonical-schema expansion
- Formalized result:
  - `docs/business-system-design.md`
  - `docs/architecture.md`
  - `requirements/TEI_PRODUCT_REQUIREMENTS.md` RQ-025 / RQ-026
- Experiment:
  - `ORDER_ENTRY_BUSINESS_PROCEDURE_REFERENCE_EXPERIMENT.md`
- Research references:
  - `UPSTREAM_XTEA_DESIGN_FALLBACK_PROTOCOL.md`
  - `XTEA_DESIGN_EVIDENCE_REGISTER_2026-09.md`

## 1. Audit question

> Can TEI help a designer form, challenge, explain, simulate, and revise a business-system design before implementation details become the source of truth?

The existing upstream trace audit checks whether Requirement / PAS / Plan / Reference / Evidence / Gate form a coherent chain. This audit checks whether the business-system design carried through that chain is itself sufficient.

## 2. External references used as design pressure

The audit used relevant public material on business-system design, including Kozo Watanabe's published work, Three-Element Analysis, and public X-TEA / XEAD material. These are independent design references rather than a schema authority or statement of TEI lineage.

When a TEI boundary is unclear, the useful question is not “how did a historical tool model this noun?” but “what design failure or missing viewpoint was the concept intended to expose?” That question is then tested against current TEI Requirements, Architecture, Reference, and Evidence.

## 3. Findings retained by TEI

### 3.1 A business system benefits from multiple review views

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

Their value is cross-checking, not the creation of three symmetric Canonical nouns.

### 3.2 Business Action is not Application Function

Public model structure and repeated Reference observations support keeping business actions separate from software realization.

Therefore TEI must not force every Business Action into an Application Function or Capability. Human work, AI-assisted work, external organizations, SaaS, physical operations, and existing-system work remain valid business-design information even when TEI does not execute them.

### 3.3 Business Procedure covers social and organizational context

Business Procedure needs to express why work occurs, what triggers it, who or what is responsible, what remains manual or external, which Application Functions are used, and what constitutes business completion outside software acknowledgement.

A Business actor is not automatically an Authorization Role.

### 3.4 Data-first is not Data-only

A useful review loop is:

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

A one-way `Data -> generated CRUD` path is insufficient for the intended product value.

### 3.5 Function design needs a user-comprehensible surface

UI / interaction is part of Function review because users and designers need to understand and challenge the proposed system. UI remains a Projection rather than Canonical authority.

If a UI implies an operation contradicted by Data / Contract / Authorization boundaries, the correction returns to authoritative meaning rather than being hidden in Target-specific code.

### 3.6 Cohesion and ownership need reviewable boundaries

Subsystem-like and Subject-Area-like views can help expose cohesion, ownership, responsibility, cross-boundary Read/CUD, and exposed Action boundaries. Current namespace / owner / Application / Database / Capability structures should be tested before adding new Canonical nouns.

### 3.7 Executable design and realistic References remain important

Driver / Simulator should expose defects while an artifact is still design. References can serve both as platform pressure tests and as reusable examples of business-system structure.

TEI should use realistic content to improve both platform assumptions and the Reference itself, while keeping third-party content licensing separate from TEI-owned Reference material.

## 4. Initial gap and Reference result

The audit found that TEI already represented Meaning before Realization, Canonical vs Projection, native / legacy assets, explicit Binding / Provider boundaries, executable verification, AI Proposal vs authority, and Reference / Conformance discipline.

The comparatively underrepresented areas were:

1. Business Action / Procedure independent from Application Function;
2. explicit Data / Function-Interaction / Business Procedure cross-validation.

The `order_entry` Reference experiment supplied a small local pressure test without requiring runtime orchestration. It showed that manual / external context can coexist with Application use, actor can remain separate from Authorization Role, and a read-only datasource can reveal a concrete contradiction if a Procedure or UI implies editing.

This is enough Product-level evidence to require preservation of Business Procedure and cross-view criticism while leaving the final Canonical representation open.

## 5. Product Requirement decision

Two Product Requirements are adopted:

### RQ-025 — Business Action / Procedure independent from Application Function

TEI must preserve business actions, actors, triggers, manual/external work, and use of Application Functions without forcing every action into Capability or runtime orchestration.

### RQ-026 — Multi-view cross-validation

TEI must allow Data, Function / Interaction, and Business Procedure views to expose contradictions and route accepted corrections back to authoritative source.

These Requirements do not adopt a top-level Canonical `Procedure` object, Workflow Engine, or new Gate. The requirement is the product value; representation remains subject to Reference pressure.

## 6. Architecture decision

The following boundaries are formalized in `docs/business-system-design.md` and `docs/architecture.md`:

```text
Authoring DSL != Canonical Semantic IR != Runtime

Business Action != Application Function != Capability != Realization

Business Procedure actor != Authorization Role

Capability existence != invocation authority

Data / Function-Interaction / Business Procedure
    = review views over shared authoritative meaning
```

Go remains a platform implementation foundation for Parser / Resolver / Validator / Analyzer / Compiler / Runtime / Plugin Host. Individual business meaning is not fixed to Go or framework APIs.

Studio / Editor remains an Authoring Client / Projection and does not become a second source of truth.

## 7. Diagnostic classification

Cross-view checks are not all compiler errors. Distinguish at least:

```text
semantic error
architecture / policy violation
review diagnostic
design question
informational impact
```

Candidate examples include missing explicit references, writes through read-only datasources, absent business information, accidental actor-to-permission mapping, impact notification, and command acknowledgement being mistaken for external completion.

Severity becomes universal only after realistic TEI Reference Evidence reproduces the defect class.

## 8. What remains open

- whether `Procedure` needs durable Canonical identity
- whether `Subsystem` / `SubjectArea` needs a new Canonical kind
- the minimum portable Authorization fact contract
- which Business Procedure checks become deterministic Conformance
- the final State / Transition first-class boundary

These are resolved through TEI-local Reference and Evidence, not by importing a historical model shape.

## 9. Lane C next work

1. turn the `order_entry` Business Procedure experiment into a concrete derived Reference artifact;
2. generate Business Action × Application Function and Business Action × Data/Relation views;
3. reproduce the read-only customer-edit contradiction as a cross-view diagnostic;
4. determine whether stable Procedure identity is required for traceability / acceptance;
5. keep hard-error versus review-diagnostic severity provisional until reproduced;
6. expand to a multi-area Reference after the small experiment produces concrete pressure.

## 10. Decision summary

TEI will preserve a Business Procedure viewpoint and active cross-checking among Data, Function / Interaction, and Business Procedure. External references helped expose this design pressure, but the resulting requirements and architecture decisions are TEI's own and remain grounded in TEI-local Reference and Evidence.
