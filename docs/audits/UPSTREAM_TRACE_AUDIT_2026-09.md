# TEI Upstream Trace Audit — 2026-09

- Status: planning audit snapshot
- Scope: forward-facing plans between Gate B handover, Gate C, Studio / AI, and the second Target
- Upstream baseline: [`UPSTREAM_ACCEPTANCE_SCENARIO_TRACEABILITY.md`](../../plan/UPSTREAM_ACCEPTANCE_SCENARIO_TRACEABILITY.md)

## Purpose

This audit checks whether the current forward plan can be traced from Product Requirements through Product Acceptance Scenarios (PAS) to plan / ADR / Reference / Evidence / Gate decisions without adding a second planning authority.

It is an audit snapshot, not a new Product Requirement set, Gate, Canonical schema, completion score, or implementation backlog.

The audit deliberately avoids blanket backfilling every historical issue. PAS labels are most useful where they clarify the next design and Evidence decisions.

## Audit principle

The primary trace is:

```text
Product Requirement
      ↓
Product Acceptance Scenario
      ↓
ADR / Plan Issue / Design
      ↓
Reference Scenario / Fixture
      ↓
Evidence
      ↓
Gate / Product claim
```

A document may participate in more than one PAS. That does not mean the PAS is complete, and passing one Reference does not automatically verify every related Product Requirement.

Evidence nature, repository revision, Provider / Target / Host conditions, and unresolved items remain explicit.

## Forward-plan mapping

| Plan / contract | Primary PAS | Secondary PAS | Main role | Audit result |
| --- | --- | --- | --- | --- |
| `GATE_B_TO_C_HANDOVER_CONTRACT.md` | PAS-02 | PAS-03 | Fix the Driver / Application meaning and Gate B Evidence to one Handover Baseline revision | Semantically aligned; wording cleanup needed because historical 4-lane text remains after current 3-lane scope |
| `issue/000030-gate-b-to-c-handover-baseline.md` | PAS-02 | PAS-03 | Instantiate the Handover Baseline after Gate B Ready | Semantically aligned; same historical 4-lane wording risk |
| `GATE_C_EVIDENCE_CONTRACT.md` | PAS-04 | PAS-02 | Evaluate first Web Target against the fixed Driver baseline | Strong alignment; already separates Semantic, Functional, Visual, and Performance evidence |
| `issue/000028-first-web-target-conformance-evidence.md` | PAS-04 | PAS-02 | Execute Gate C Evidence against Handover Baseline | Strong alignment; no new semantic concept required before Gate C |
| `issue/000026-target-capability-execution-profile-boundary.md` | PAS-04 | PAS-03 | Pressure-test Target-neutral meaning with a second Target / Host | Strong alignment; RQ-024 is already explicit and should remain target-specific rather than become a universal early threshold |
| `issue/000008-editor-and-ai-followup-note.md` | PAS-01 | PAS-06 | Hold detailed Studio / AI design until Gate C, then start from shared Canonical meaning and Evidence | Strong conceptual alignment; executable Studio authoring-loop Reference intentionally remains post-Gate-C work |
| `issue/000027-ai-proposal-acceptance-boundary.md` | PAS-01 | PAS-06 | Verify AI Proposal != Canonical authority and reuse the shared Acceptance Boundary | Strong alignment; no AI-specific semantic acceptance path should be introduced |

## PAS-by-PAS audit

### PAS-01 — meaning-preserving authoring and reload

Current forward plans already contain the important boundary:

```text
file / Studio / CLI / AI candidate
          ↓
shared Acceptance Boundary
          ↓
authoritative source
          ↓
reload / conformance
```

`ISSUE-000008` and `ISSUE-000027` already preserve these principles:

- Proposal / candidate is not Canonical authority
- AI cannot bypass Validation / Conformance
- stale base revision must not be silently applied
- accepted authoritative source must reload without AI
- Editor / CLI / AI should converge on the same acceptance principles
- Wails, if used for Studio, remains Host / Adapter technology and does not define Application meaning

### Remaining PAS-01 gap

The missing item is not another abstract requirement. It is a small executable post-Gate-C Studio Reference that demonstrates one concrete authoring loop, preferably a Relation edit or similarly bounded semantic change:

```text
select existing semantic subject
      ↓
propose change against readable baseModelRevision
      ↓
preview Semantic Diff / impact / UI effect
      ↓
accept or reject
      ↓
shared Acceptance Boundary
      ↓
authoritative source
      ↓
reload
      ↓
same expected Canonical identity / conformance
```

This should remain deferred until Gate C so the first Studio interaction does not freeze unverified Presentation or Application semantics.

Do not introduce mandatory user-authored UUIDs, Patch-only authoring, or a second Studio-owned semantic store merely to implement this loop.

## PAS-02 — execute and verify Application meaning through Driver

The current Driver / Handover line is already coherent:

```text
Authentication
  → browse
  → record
  → reference
  → header-detail
  → Validation
  → Authorization
  → Command
  → Business Result
  → separate explicit read
  → Observation
```

The important result distinction is preserved:

```text
Business Result
Execution Error
Observation
Developer Diagnostic
```

Gate B should continue to focus on the current three required source-dependent Evidence lanes converging on the same clean revision:

1. deterministic Repository Evidence (#310)
2. generated-artifact reproducibility Evidence (#255)
3. real PostgreSQL Provider Evidence (#308)

Actual Oracle Evidence remains valuable PAS-03 / external-provider evidence, but it is not a current Gate B Ready blocker.

### Documentation hygiene gap

Both `GATE_B_TO_C_HANDOVER_CONTRACT.md` and `ISSUE-000030` correctly state that the current Gate B scope is the three lanes above and that older four-lane text is historical.

However, later forward-looking sections still use phrases such as “4 Evidence lane”, “4 lane completion Evidence”, and “4 laneを新しいrevisionへ揃える”. This is explicitly overridden by the newer scope section, so it is not a semantic contradiction in the current decision, but it is a traceability hazard for the first real Handover Baseline.

Before the first Handover Baseline is instantiated, normalize forward-looking wording to the current three required Gate B lanes and move Oracle to a separate optional / unresolved external Evidence subsection. Historical rationale may remain clearly marked as history.

This cleanup must not change Gate B scope again.

## PAS-03 — staged migration / external asset replacement without losing meaning

The Handover and Target plans correctly preserve these boundaries:

- Application intent and Canonical Contract remain separate from Provider realization
- external-observed facts are not invented from synthetic or PostgreSQL evidence
- Provider-specific details remain in Provider / Adapter boundaries
- actual Oracle evidence, when available, remains separately attributable
- absence of actual Oracle evidence is carried as unresolved rather than silently filled by the Web Target

No new Gate B blocker is needed for PAS-03.

The next meaningful PAS-03 proof after Gate B is to carry unresolved external-provider facts through Handover without allowing the first Web Target to invent them.

## PAS-04 — project the same Application meaning to multiple Targets

`GATE_C_EVIDENCE_CONTRACT.md`, `ISSUE-000028`, and `ISSUE-000026` already form a coherent PAS-04 chain:

```text
Handover Baseline
      ↓
first Web Target
      ↓
Semantic Conformance
Interaction / Functional Evidence
Visual Fidelity Evidence
Performance measurement
      ↓
classified Target / Runtime / Trust differences
      ↓
second Target pressure test
```

The audit confirms several boundaries that should not be weakened:

- Visual Fidelity cannot substitute for Semantic or Functional evidence
- a VLM / screenshot score cannot define Canonical success
- Web transport / DOM / HTTP status cannot redefine Business Result, Authorization, or Execution Error
- first Target evidence must not prematurely finalize Presentation IR
- second Target should be chosen for architectural pressure, not merely convenience
- Studio using a Wails host is not evidence that generated Wails Desktop is a conformant business Target

RQ-024 should remain measurable per Target / Driver / Provider boundary. Do not invent a single performance threshold before real target measurements exist.

## PAS-05 — safe execution and restart under uncertain external state

PAS-05 is intentionally not a new Gate B or Gate C blocker in the current v1 line.

The result / Observation distinction already protects against the most immediate category error: Command acknowledgement must not be treated as proof that external state is established.

Broader recovery concerns such as fencing, durable handles, replay safety, ownership, lifecycle, and uncertain external completion belong to the runtime / deployment / plugin lifecycle work where those states can be observed with a real Reference.

Audit result: no missing upstream requirement; avoid pulling this work backward into Gate B.

## PAS-06 — explore existing semantic knowledge before new implementation

`ISSUE-000008` and `ISSUE-000027` already contain the core policy:

- search existing Application Pattern / Capability / Validation / Relation / Source first
- distinguish AI Proposal from accepted Decision
- trace proposals to Canonical IDs / Source / Evidence
- keep retrieval / ranking / trajectory / derived experience non-authoritative
- send every accepted change through the shared Acceptance Boundary

### Remaining PAS-06 gap

The repository does not yet need a large AI knowledge subsystem. The smallest post-Gate-C Reference should instead be a repository-backed, read-only semantic exploration path usable by both a human-facing Studio and AI assistance:

```text
purpose / intent
      ↓
search existing semantic assets
      ↓
inspect identity / dependency / Source / Evidence
      ↓
classify applicable / incompatible / unknown
      ↓
explicit-reference candidate
      ↓
shared Acceptance Boundary when a change is proposed
```

Search ranking or name similarity must not become semantic evidence.

Do not create a vector database, ontology mirror, or AI-owned duplicate knowledge base before this small path shows that the repository-native semantic assets are insufficient.

## Cross-cutting RQ audit

### RQ-021 — Reference / Conformance reproducibility

Present across all six PAS. Current plans already preserve revision-fixed Evidence and explicit Evidence type boundaries. The main remaining risk is wording drift around historical Gate B lane counts, not missing conformance concepts.

### RQ-022 — distinguish product from design / research verification

Current plans are generally disciplined here:

- architecture probe is not product completion
- synthetic evidence is not actual-provider evidence
- visual evaluation is not Semantic Conformance
- an experimental framework is not a platform standard
- Issue close / CI green is not a Gate by itself

Keep this explicit in future Reference evidence records.

### RQ-023 — maintainability by one maintainer

The PAS layer should reduce, not increase, document count.

Policy from this audit:

- do not backfill PAS labels into every historical issue
- add PAS trace only when a forward plan is being materially revised or when it will produce Gate / Reference Evidence
- keep Requirements, PAS, ADR, Plan Issue, and Evidence roles distinct
- prefer one shared Acceptance Boundary over authoring-channel-specific pipelines
- prefer Provider / Target / Host adapters over repository splitting unless measured pressure requires otherwise

### RQ-024 — measurable performance

First Web Target should establish measurement points. The second Target should pressure-test whether the chosen measurements and thresholds are Target-specific or genuinely common.

Do not turn the PAS audit into a performance target registry.

## Duplicate concepts found

The audit found repeated wording, but not harmful duplicate authorities, in these areas:

1. Gate C Semantic / Functional / Visual separation appears in both the Gate C contract and `ISSUE-000028`.
   - Keep the contract authoritative for Evidence meaning.
   - Keep the Issue focused on executing that contract.
2. Shared AI / Editor Acceptance Boundary appears in `ISSUE-000008`, `ISSUE-000027`, and ADR-0022.
   - ADR owns the architecture decision.
   - `ISSUE-000027` owns the post-Gate-C Reference proof.
   - `ISSUE-000008` owns only the timing / sequencing rule.
3. Target Capability / Execution / Trust classification appears in Gate C Evidence and `ISSUE-000026`.
   - Gate C records observed differences.
   - `ISSUE-000026` owns cross-target pressure-testing and the decision whether separate concepts are needed.
4. Gate B lane descriptions remain duplicated between Handover contract and `ISSUE-000030`.
   - Keep only the current three-lane rule in forward-looking sections.
   - Historical four-lane rationale may remain as clearly historical text.

## Missing trace judged intentional rather than defective

The following are intentionally not fully specified yet:

- Studio UI / navigation / visual layout details before Gate C
- final Proposal schema
- final Presentation IR
- complete Target Capability taxonomy
- universal performance thresholds
- large AI knowledge / retrieval subsystem
- second Target choice before Gate C evidence is available
- detailed PAS-05 recovery model before runtime / deployment evidence exists

Filling these now would reduce, rather than improve, upstream discipline.

## Action order

1. Finish current Gate B Evidence convergence on one clean revision; do not expand its blocker set.
2. Before first Handover Baseline, normalize forward-looking Handover wording from historical four-lane language to the current three required lanes, with Oracle separately optional / unresolved.
3. Instantiate Handover Baseline using PAS-02 as primary and PAS-03 as secondary trace.
4. Run first Web Target / Gate C as PAS-04, preserving separate Semantic / Functional / Visual / Performance evidence.
5. After Gate C, create the smallest Studio authoring Reference for PAS-01 using the shared Acceptance Boundary.
6. Reuse the same boundary for AI Proposal / CLI / file-native authoring; do not create AI-specific authority.
7. Add PAS-06 as a small repository-native read-only semantic exploration Reference before any large AI knowledge subsystem.
8. Choose the second Target from the uncertainties exposed by Gate C and pressure-test PAS-04 plus RQ-024.
9. Keep PAS-05 in runtime / deployment scope until a real recovery Reference exists.

## Decision from this audit

No new Product Requirement, Gate, Canonical concept, or GitHub Issue is required by this audit.

The current upstream model is structurally coherent. The next improvement should be trace cleanup and executable References, not another abstraction layer.
