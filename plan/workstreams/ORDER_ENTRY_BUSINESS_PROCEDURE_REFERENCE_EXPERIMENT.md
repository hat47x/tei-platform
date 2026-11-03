# `order_entry` Business Procedure Reference Experiment

- Status: Lane C experiment design / not Canonical
- Purpose: restore the X-TEA Business Model viewpoint around the existing v1 `order_entry` Reference without introducing a Workflow Engine or duplicating Application meaning
- Design fallback: `UPSTREAM_XTEA_DESIGN_FALLBACK_PROTOCOL.md`

## 1. Question under test

Current `order_entry` is a strong Application / Driver Reference, but the business procedure that surrounds the software is not explicitly represented.

This experiment asks:

> Can the Business Model viewpoint be expressed as a derived, traceable design projection over existing TEI identities, or is a durable lightweight `Procedure` semantic identity required?

The experiment must not answer the question by adding `Procedure` first.

## 2. X-TEA pressure being reproduced

The relevant historical distinction is:

```text
Task Action = business action
Application Function = possible system realization of part of that action
```

X-TEA source/model analysis found TaskAction and Function linked explicitly rather than collapsed, and corpus analysis found 202 of 1606 TaskActions with no Application Function realization.

Therefore this experiment requires at least one valid business step that has **no Application Pattern / Capability**.

This is the pressure test. If TEI cannot represent that step without inventing a fake executable object, the Business Model viewpoint is incomplete.

## 3. Facts already authoritative in current TEI

The experiment starts only from identities already present in the Reference Workspace.

### Application

```text
Application: order_entry
Semantic namespaces: sales / customer / security
Datasource sales:
  target = sales_core
  access = read-write
Datasource master:
  target = master_data
  access = read
```

### Existing Application Patterns

```text
order_entry.recent_orders
  kind = browse
  data = sales.order
  query = order_entry.recent_orders

order_entry.order_record
  kind = record
  data = sales.order

order_entry.customer_reference
  kind = reference
  relation = sales.order.customer
  datasource = master
  contract = master_data.customer
  contract query = find_by_id

order_entry.order_lines
  kind = header-detail
  relation = sales.order.lines

order_entry.submit_order
  kind = command
  capability = sales.order.submit
```

### Existing Semantic meaning

```text
sales.order
  order_id
  customer_id
  ordered_at
  status
  total
  external_id
  relation customer -> customer.customer one
  relation lines -> sales.order_line many

sales.order_line
  order_line_id
  line_number
  item_code
  quantity
  unit_price

sales.order.submit
  input sales.order
  output sales.order
```

### Existing Validation

```text
sales.order_line.quantity_positive
sales.order.lines.line_number_unique
```

No Business Procedure fact below may silently overwrite these meanings.

## 4. Candidate business context — explicitly non-authoritative

The current Reference does **not** itself prove a real-world business trigger, actor title, or post-submit manual process.

For the experiment, use deliberately provisional labels:

```text
candidate trigger:
  an order-handling need enters the sales work context

candidate actor label:
  sales operator

candidate completion context:
  the operator has obtained the Application Business Result and, where required,
  separately checked externally observed state before continuing non-application work
```

These are experiment scaffolding, not accepted business facts.

Do not change them into `sales.order` attributes, Authorization Roles, or Capability inputs.

If later evidence from a domain Reference, stakeholder, or accepted requirement gives better terms, replace the scaffolding rather than preserving it for compatibility.

## 5. Derived Procedure projection v0

The initial form is intentionally a planning / projection artifact, not `.tei` authoring.

```text
Procedure candidate: order-handling
Purpose:
  inspect and prepare an order, submit it when acceptable,
  distinguish Application result from separately observed external state

Trigger:
  [candidate / unresolved] order-handling need enters sales context

Actor:
  [candidate label only] sales operator
```

### Step BP-00 — orient to the work

```text
business action:
  determine which order requires attention

realization:
  application-use

references:
  Application Pattern order_entry.recent_orders

business meaning owned here:
  why the operator is consulting the list

meaning NOT owned here:
  browse query semantics
  authorization
  UI layout
```

### Step BP-01 — open the order context

```text
business action:
  inspect or prepare the selected order

realization:
  application-use

references:
  Application Pattern order_entry.order_record
  Semantic Data sales.order
```

### Step BP-02 — resolve the customer reference

```text
business action:
  confirm the customer associated with the order

realization:
  application-use

references:
  Application Pattern order_entry.customer_reference
  Semantic Relation sales.order.customer

boundary:
  master_data remains read-only from order_entry
```

The Procedure does not repeat the Contract Query or input binding. It links to the existing Pattern identity.

### Step BP-03 — inspect order lines

```text
business action:
  review the items / quantities represented by the order lines

realization:
  application-use

references:
  Application Pattern order_entry.order_lines
  Semantic Relation sales.order.lines
  Validation sales.order_line.quantity_positive
  Validation sales.order.lines.line_number_unique
```

### Step BP-04 — business review not represented as an Application Function

```text
business action:
  decide whether the order is ready to submit in the surrounding business context

realization:
  manual / unresolved

Application Pattern:
  none

Capability:
  none
```

This step is mandatory for the experiment.

Its purpose is not to assert a particular approval rule. It verifies that TEI can preserve the fact that a business action may exist without pretending that a software Function implements it.

If the only way to keep BP-04 is to invent `sales.order.review` Capability with no real semantic contract, the experiment fails.

### Step BP-05 — evaluate accepted machine-checkable constraints

```text
business action:
  establish whether accepted deterministic order constraints are satisfied

realization:
  TEI validation / design-execution support

references:
  existing Canonical Validation identities

boundary:
  do not equate deterministic validation with the whole BP-04 business judgment
```

BP-04 and BP-05 are intentionally separate.

### Step BP-06 — authorization at the Application boundary

```text
business action:
  determine whether the authenticated subject may execute the submit Application action

realization:
  security Capability / Driver boundary

references:
  existing security.authorize meaning
  resource/action = existing Application meaning

boundary:
  Procedure actor label is not Authorization Role
```

A person described as the procedure's “sales operator” does not gain authority merely from that label.

### Step BP-07 — submit the order

```text
business action:
  request the accepted Application command

realization:
  application-use

references:
  Application Pattern order_entry.submit_order
  Capability sales.order.submit

result:
  Business Result or Execution Error
```

### Step BP-08 — separately observe external state when needed

```text
business action:
  verify the relevant external state when business continuation depends on actual state

realization:
  explicit read / Observation boundary

references:
  existing Driver Command / Observation separation

boundary:
  Command acknowledgement != observed business reality
```

This step does not say that every submit must always perform a particular read. It preserves the semantic distinction required when external state matters.

### Step BP-09 — continue non-application work

```text
business action:
  continue the surrounding business work after the Application interaction

realization:
  manual / external / unresolved

Application Pattern:
  none

Capability:
  none
```

BP-09 intentionally keeps the Procedure open to work outside TEI.

## 6. What this Procedure does not own

The Procedure projection must not become a second source of truth for:

- Data attributes
- Relations
- Query definitions
- Datasource access mode
- Database Contract details
- Validation expressions
- Capability signatures
- Authentication / Authorization rules
- Business Result schema
- Observation implementation
- Target UI layout
- workflow retry / timer / orchestration behavior

It may **reference** those meanings and explain why / when they are used.

## 7. First cross-view checks

### C1 — explicit reference existence

For every Procedure reference to an Application Pattern, Semantic Data / Relation, Validation, or Capability:

```text
referenced identity must resolve exactly once
```

Initial severity: semantic error in the Procedure artifact.

### C2 — no fake Function requirement

A business action with `manual / external / unresolved` realization may validly have no Application Pattern or Capability.

Initial severity if tooling demands one: architecture defect in the Procedure representation.

### C3 — actor / authorization separation

A Procedure actor label cannot satisfy or synthesize Authorization Role / subject / policy.

Initial severity: architecture policy violation if authority is inferred.

### C4 — read-only master boundary remains external to Procedure

BP-02 references `order_entry.customer_reference`. The Procedure must not restate or weaken `master` datasource access.

If a Procedure editor allows “edit customer” here merely because Customer is visible, cross-view criticism should flag that the current Application datasource is read-only.

Initial severity: architecture / impact diagnostic; semantic error only if an accepted Application mutation contradicts the datasource contract.

### C5 — validation is not the whole business judgment

Machine-checkable validation does not erase BP-04.

If tooling derives:

```text
all Validation PASS => business-ready
```

without an accepted semantic contract saying so, flag overclaim.

Initial severity: design / policy diagnostic.

### C6 — command result is not Observation

BP-07 Business Result cannot automatically satisfy BP-08 external-observation need.

Initial severity: policy violation where actual-state proof is required.

### C7 — Data / Relation change impact

If `sales.order.customer`, `sales.order.lines`, or referenced Validation identity changes, the Procedure projection should be discoverable as impacted.

Initial severity: impact information; stale explicit reference becomes semantic error.

## 8. X-TEA-style matrices to derive, not author twice

The experiment should be able to derive small matrices from references.

### Business Action × Application Pattern

| Business Action | recent_orders | order_record | customer_reference | order_lines | submit_order | no function |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| BP-00 orient | X |  |  |  |  |  |
| BP-01 open order |  | X |  |  |  |  |
| BP-02 confirm customer |  |  | X |  |  |  |
| BP-03 inspect lines |  |  |  | X |  |  |
| BP-04 business review |  |  |  |  |  | X |
| BP-05 deterministic constraints |  |  |  |  |  | X |
| BP-06 authorization |  |  |  |  |  | X |
| BP-07 submit |  |  |  |  | X |  |
| BP-08 observe |  |  |  |  |  | X |
| BP-09 continue work |  |  |  |  |  | X |

“no function” means “no direct Application Pattern realization in this Procedure matrix,” not “nothing happens technically.” Validation, security, Observation, human work, or external systems may realize the business concern through another boundary.

### Business Action × Data / Relation

This matrix should be **derived where traceable** and leave unknown cells unknown rather than infer reads / writes from names.

The goal is to expose questions such as:

- Does a business step depend on information absent from the Data Model?
- Does a visible editable concept have no accepted write meaning?
- Does a step appear to modify data outside the current Application ownership / datasource boundary?

## 9. Procedure identity experiment

Test two variants without changing Product Requirements first.

### Variant D — derived Procedure artifact

The Procedure has no Canonical identity. It is generated / maintained as a design projection that references stable existing identities.

Evaluate:

- can Source / Evidence trace it reliably?
- can impact analysis find it after referenced meaning changes?
- can multiple Procedures refer to the same Pattern without ambiguity?
- can review comments / accepted business intent survive regeneration?
- can manual steps be durable without becoming Canonical objects?

### Variant C — durable lightweight Procedure identity

Only if Variant D loses important accepted business meaning, test a minimal durable contract approximately containing:

```text
procedure identity
purpose
trigger text / reference
ordered or presented business actions
actor label / reference
manual instruction
references to existing Application / Semantic meaning
```

Still exclude by default:

```text
workflow runtime semantics
retry
timer
queue
parallel orchestration
credential / authority
provider execution
```

Do not choose Variant C because X-TEA had `Task`; choose it only if TEI-local Reference evidence requires durable identity.

## 10. Acceptance evidence to collect

The experiment should produce evidence for these claims:

```text
E1  existing Application / Semantic identities are reused without duplication
E2  at least one business action survives with no Application Function
E3  actor label does not confer authorization
E4  read-only datasource meaning cannot be overridden by Procedure/UI convenience
E5  Validation PASS is not automatically promoted to whole-business approval
E6  Command Business Result remains separate from Observation
E7  an intentional Data / Relation change produces Procedure impact information
E8  a user-understandable Procedure / interaction projection can reveal a model deficiency
E9  the deficiency can be corrected through authoritative source + shared Acceptance, not by patching the projection only
E10 derived-vs-durable Procedure identity can be decided from observed traceability loss, not aesthetic preference
```

## 11. Deliberate defect injection for the first review loop

After the baseline projection is readable, inject one temporary design inconsistency in an experiment-only branch / fixture.

Preferred first defect:

```text
Procedure / UI candidate offers “edit customer” during BP-02
while order_entry.master is read-only.
```

Expected behavior:

1. Procedure / UI review makes the business desire visible.
2. Cross-view check finds the read-only datasource conflict.
3. The designer must choose explicitly:
   - this application only references customer -> remove edit affordance
   - a separate accepted customer-maintenance Application / Capability exists -> reference it
   - requirements genuinely changed -> change authoritative Application / Contract design through normal Acceptance
4. The projection itself must not silently add write access.

This reproduces the Three-Element feedback loop:

```text
Function / Business view exposes tension
        ↓
Data / Application contract is re-examined
        ↓
accepted source changes only if business meaning actually changed
        ↓
Projection regenerates / revalidates
```

## 12. Second defect candidate

Only after the first loop works, test:

```text
Procedure says “submit” after quantity check only,
but line-number uniqueness Validation is omitted from the review projection.
```

The check should reveal that Procedure / View completeness cannot be inferred from one convenient validation.

Do not invent a generic rule engine from this example.

## 13. Decision criteria

### Keep Procedure derived if

- accepted business meaning can be preserved in existing authoritative sources or a clearly non-authoritative reviewed artifact
- references survive reload / rename rules with reliable impact analysis
- manual steps and business explanations do not need independent durable identity
- no second source of truth appears

### Promote a lightweight durable Procedure concept only if

- real accepted business meaning is otherwise lost
- cross-view traceability requires stable Procedure / step identity
- the need recurs beyond one `order_entry` example
- X-TEA / Reference evidence and TEI-local Reference evidence both support the distinction
- the concept remains independent from Workflow Engine semantics

## 14. Initial expected result

The current hypothesis is that **the Business Procedure viewpoint is necessary, but its canonicality is still unproven**.

This is deliberately narrower than the earlier prototype generation that immediately restored `Procedure` as a Surface object.

Lane C will keep the viewpoint unconditionally while re-testing whether the object identity itself belongs in current TEI.

## 15. BP-05 Validation scope and qualified evidence (2026-11-02 plan addendum)

Status: **Lane C read-only design/qualification candidate, not adopted Business meaning; no new runtime, gate, schema, or ADR**. This section refines E5/E7 and the defect candidate in section 12; it does not amend the baseline BP-05 fixture or expand Product Gate G0-G4.

### 15.1 Evidence already present vs claims still unproved

- Existing `tools/reference-business-procedure-check -review-validation-scope` compares the Application's three declared Validation identities with BP-05's two explicit references, returning one `applicationOnly` and `review-needed`. This is **a scope question**, not an automatic missing-rule diagnostic or source mutation. See [#1125 scope review](https://github.com/hat47x/tei-platform/issues/1125#issuecomment-6400716990).
- `internal/driver/final_gate_b_scenario_test.go` and `gate_b_unified_scenario_test.go` already contain **test-local synthetic positive paths** through three Validations, Authorization, test-only Command and synthetic Observation. The earlier "three-rule path not found" claim was corrected in [#1125](https://github.com/hat47x/tei-platform/issues/1125#issuecomment-6405752403), then rechecked at main `21f4e986581d483b494fe5bdf363a0b4309cf8f8`. Code inspection is not a fresh Go PASS receipt.
- Third-rule-only rejection **preventing actual Authorization/Command invocation**, full submitted-order line coverage, genuine Provider acquisition, accepted BP-05 responsibility and physical effect remain distinct claims. A `t.Fatalf` in a positive test does not prove the product's negative-side gate.
- The current test source supplies scalar `lineNumber` and `quantity` from the first row. `lineNumbers` comes from the collection in `final_gate_b`, but is a fixed fixture in `gate_b_unified`. A count of three evaluated rules does not itself prove coverage of every line.

### 15.2 Minimal discriminating inputs (design only; NOT RUN)

Keep stable row identities and one fixed editing target `A`. Use two lines with positive quantities and unique line numbers: `A(id=5001, line=1)`, `B(id=5002, line=0)`. Change **collection order only**, not identities or values:

| Input collection | First-row hypothesis | Fixed edit-target A hypothesis | All-lines-positive hypothesis |
| --- | --- | --- | --- |
| P1 = A(1), B(0) | satisfied | satisfied | not-satisfied |
| P2 = B(0), A(1) | not-satisfied | satisfied | not-satisfied |

These are predicted **third-rule** outcomes, not executed results and not Command gate verdicts. A legitimate edited-row interpretation must bind to a stable row ID, **never** `Rows[0]` by convenience. A third-rule-negative case also needs separate Authorization/Command invocation counts (zero expected if that particular operation's accepted rule makes it a blocking precondition).

**Important SQL qualification:** the Reference `order-line-find-by-order.sql` specifies `ORDER BY line_number, order_line_id`; `line_number` is persisted as required `int64`. Therefore for the two rows above **P2** is consistent with the specified SQL ordering, whereas **P1 is an intentional out-of-order synthetic executor input**, not a claim about actual PostgreSQL output. Under a qualified numeric ascending order, a **complete and relevant nonempty** result set, and preserved row order from the actual Provider to evaluation, checking its minimum line number positive can logically entail positivity of that result set. It does **not** entail that the result set includes every line required by the Business rule, that a read snapshot remains valid at the action boundary, or that all **quantities** are positive (the SQL does not order by quantity). Empty sets require a separate Owner rule; do not equate vacuous truth with an order being ready to submit.

### 15.3 One claim-oriented review record, not a new Canonical type

Retain the existing read-only comparison JSON and Source Index evidence. Attach, as an issue/review evidence packet rather than a new Core schema:

`{rule + rule version, application + step + operation, human scope classification (missing / out-of-scope / alternate / undecided), Business target and quantifier (edited row / selected row / submit cohort / undecided), selection identity, declared logical footprint, actual acquisition route + principal + result slice + completeness + ordering, source/world/fixture/lock revision, positive/negative/test-only/actual evidence, evaluation result, Authorization/Command gating evidence, current rule/target freshness, current disclosure eligibility, Owner answer + authority + acceptance status, residual gaps}`.

A finding of `out-of-scope` or `alternate route` must include its reason and exact operation/scope. Neither finding changes the checker's `review-needed` status unless the actual declared sets change. A change in accepted Business meaning is **proposed through existing Shared Acceptance**, never made by the comparator, the AI reviewer, or a presentation projection.

Cross-lane handoff **without shared implementation**: [#1125](https://github.com/hat47x/tei-platform/issues/1125) owns BP-05 responsibility/quantifier and BP-04 manual readiness; [#267](https://github.com/hat47x/tei-platform/issues/267) owns Owner acceptance; [#1708](https://github.com/hat47x/tei-platform/issues/1708) is a **closed reference** for read-set/snapshot semantics; [#2506](https://github.com/hat47x/tei-platform/issues/2506) and [#2622](https://github.com/hat47x/tei-platform/issues/2622) own the ongoing context/Provider completeness pressure; [#2628](https://github.com/hat47x/tei-platform/pull/2628) concerns actual acquisition evidence; [#1174](https://github.com/hat47x/tei-platform/issues/1174) owns actual Effect and operation authority; [#2516](https://github.com/hat47x/tei-platform/issues/2516) and SHO own audience-specific disclosure/explanation. Do not infer an absence from a failed search (#2196). The newer [#1125 S3 evidence qualification](https://github.com/hat47x/tei-platform/issues/1125#issuecomment-6578813642) is supporting context, **not** proof that the BP-05 condition is accepted.

### 15.4 Order of qualification, owners, and stop conditions

1. **S0 / read-only, Lane C**: preserve existing three-rule synthetic positive code, the two-rule Gate negative, and the `review-validation-scope` row as **separate pieces**. Inspect the current inputs and SQL order, and record P1/P2 hypothesis results as NOT RUN. Do not rebuild a Gate.
2. **S1 / narrow synthetic negative, Driver owner only if authorized**: reuse existing test seams to establish the actual third-rule failure and check whether Authorization and Command are invoked. Separately test row-order swap against a stable edited-row ID and a complete submitted cohort. Results from independent tests cannot be concatenated into one provenance claim.
3. **S2 / actual Provider boundary, #2506 / #2622 / #2628 (reuse closed #1708)**: only when product needs the stronger claim, check real SQL order, result mapping/order preservation, visibility/RLS/tenant scope, pagination/completeness, snapshot/read-set freshness and input-to-Command continuity for **one pinned operation**. Provider test PASS is not Owner acceptance or a real Business Effect.
4. **S3 / Business acceptance and consumer explanation, #1125 / #267 / #2516**: Owner answers **one question**: "For BP-05, which row set must line-number and quantity positivity cover, and is the condition BP-05's own responsibility or an identified alternate route?" Preserve `undecided` until accepted; explain the exact supported claim and hidden evidence only to qualified audiences.

Acceptance for this *design note* means the evidence/unknown distinctions and handoffs are recorded, not that product G1/G2 or the Business Procedure has passed. A synthetic pass, SQL text, content digest, absence of a retrieved counterexample, or a separate effect receipt is **insufficient** to promote an unqualified global or production Claim.

### 15.5 The qualified implication is a claim-specific proof, not a sorting optimization license

For a pinned submit operation, distinguish (a) *the concrete value evaluated*, (b) *the declared Business target cohort*, and (c) *the conclusion asserted to the Consumer*. An acceptable proof of `forall x in L: line_number(x) > 0` may use either explicit per-entity evaluation or a justified minimum-row reduction, **only if** all of these obligations are discharged together:

| Obligation | Minimum-row method: what must be checked | Existing owner / evidence limit |
| --- | --- | --- |
| **Q1 — Cohort** | `L` is defined by the still-unaccepted Owner rule (fixed edited row vs all submit lines); the query's parent/tenant/relationship predicate selects precisely that cohort for the claimed time. | #1125/#267 decision; #2506 Function Context. A known result length is not cohort completeness. |
| **Q2 — Read coverage** | All members required by `L` are present in `R`, with no page/window cap, authorization-hidden rows, missing entity/relationship edge, join-expanded duplicate grain, or failed partial read mistaken for complete. A result of 0 rows also needs its own submit-business rule. | #2506/#2622 and Draft #2627; `requested all` is not `observed complete`. |
| **Q3 — Ordered-value equivalence** | Physical `ORDER BY line_number, order_line_id` is effective for the executed provider/alias; mapped field remains comparable numeric `int64` (not lexicographic string sort); Driver preserves order until `Rows[0]`; no post-query filter/reorder changes the proof basis. | Existing SQL and test code support a **candidate** inference, not a qualified actual Provider trace. #2387/#2634 own hidden Query sort/predicate dependencies. |
| **Q4 — Rule and attribute grain** | The selected `positive` Rule/typed attribute version is the one evaluated. Its first-row result supports only the proven *line-number* claim; `quantity_positive` on the first row needs independent all-line coverage because the SQL is not ordered by quantity. `relation-unique` needs the relation keys for the same cohort/grain, not an unrelated fixed array. | Existing `validationeval` is reused; #988/#2619 cover conditional operator laws, not an auto-accepted rewrite. |
| **Q5 — Time, route and permitted use** | Source/World/Query/Operator/Mapping/Provider/Policy/Result-Slice revision and current scope match the actual operation; read freshness and authorization at commit are separately verified, and the Consumer is allowed to receive the supported explanation. | Closed #1708 remains a reusable snapshot contract; #1174 owns effect-time fence, #2516 disclosure, #2600 selective requalification. |

This is a **conjunction of conditions**, not an arithmetic invariant that can be inferred from the presence of `ORDER BY`. The inference applies to a **nonempty complete numeric ordered cohort**. It does not establish that the accepted business process has approved submission or that `sales.order.submit` caused a durable effect.

### 15.6 Three minimal counter-worlds and selective requalification (NOT RUN)

Use the already-proposed P1/P2 test pair; do not add a new Gate or second comparator. The following are **separate claim-specific witnesses** to explain what changing one dependency invalidates, and do not inherit PASS from unrelated synthetic suites:

| Case | Two worlds / one changed fact | What becomes unproved (other claims remain independently assessed) |
| --- | --- | --- |
| **W1 — Missing member** | `L={A(1),B(0)}` in both worlds. Full provider result is `[B(0),A(1)]`, but a partial/RLS-limited result is `[A(1)]`. Both visible evaluations can yield `satisfied` for the latter input. | `minimum(R)>0` no longer implies `forall L>0` until completeness and disclosure-safe scope are established; do not disclose hidden B or infer its existence from an error. |
| **W2 — Hidden Query dependency** | Same Rule, Mapping and returned fixture for the normal case, but the selected Query changes from `ORDER BY line_number` to `ORDER BY order_line_id` (or loses a membership predicate). Under A(id5001,line1),B(id5002,line0), first-row positivity can be satisfied even though B fails. | Existing minimum-row proof must be selectively **requalified**, despite equal output under a previous passing fixture; Output/Query ID and unchanged model digest are insufficient. |
| **W3 — Temporal cohort drift** | At read `R0`, complete `L={A(1)}` passes. Before the actual submit action a new B(line0) joins the relevant cohort, or an existing member is renumbered, without an admissible same-operation read/recheck. | Prior `satisfied` remains a historical evaluation of `R0`; it cannot be promoted to the current submitted cohort or commit-time authority. No assumption of shared cross-provider snapshot. |

**Selective invalidation proposal, not a new cache engine:** a change in the Rule/quantifier, relevant attribute grain, SQL predicate/order, mapping, Provider route, RLS/pagination/coverage, membership, read revision, or current Grant invalidates the **dependent proof** for the relevant Consumer/operation. A style-only document change need not invalidate a pinned numeric result, though the delivered SHO/Artifact bytes and current disclosure may still require separate verification. Re-run only the necessary checks through existing #2582/#2600/#2506/#2622/#1174 lanes; successful Build reuse is not Runtime freshness or Business Acceptance.

**Owner packet (one question, unchanged)**: "For BP-05, is line-number/quantity positivity required of the edited row or the entire submitted line cohort, and does BP-05 own that responsibility or an explicitly evidenced other path?" Until answered, the human category stays `undecided`, the existing checker stays `review-needed`, and all Q1-Q5 production claims are unqualified. There is no new acceptance outcome, Owner decision, provider-run receipt, or formal operator law in this addendum. Cross-lane source: [#2619 conditional operator laws](https://github.com/hat47x/tei-platform/issues/2619#issuecomment-6577311485), [#1125 acquired-vs-disclosed qualified Witness](https://github.com/hat47x/tei-platform/issues/1125#issuecomment-6578813642), and [#2619 hidden Query dependencies](https://github.com/hat47x/tei-platform/issues/2619#issuecomment-6578784628).

### 15.7 Quantified claim input domain: empty rows and absent fields (plan; NOT RUN)

The `Q1-Q5` qualifications above also require an **observable evaluation domain**, not just a complete-looking result count. The following counterexamples are handoff questions for existing Context/Provider owners, **not a new Validation kind, result enum, or Gate**:

| Synthetic case | Observed Input / claim distinction | Required handling |
| --- | --- | --- |
| **E0: truly empty cohort** | A complete `R=[]` has no `Rows[0]` and no minimum. Mathematical `forall x in empty: positive(x)` is vacuously true, but `order ready to submit` is a different, Owner-defined Business predicate. | Do not call the current first-row source or synthesize a positive sample. Record **no minimum / no evaluated row** separately from a domain `not-satisfied` result. Owner decides whether submitting an empty order is admissible. |
| **E1: partial or undisclosed cohort** | A provider returns `R=[]` or `[A(1)]` with partial page, tenant, RLS, joined or relationship scope. Visible zero rows / a positive first row do **not** establish that the declared `L` is empty / fully positive. | Require the **cohort-relative rowset coverage** witness; otherwise claim is `not established`, without leaking hidden member counts in a Consumer explanation. |
| **E2: field absent despite complete rows** | `R=[A]` can have complete row identities but `line_number` absent in the Consumer's mapped `Attributes`. `absent` is neither observed SQL `NULL` nor observed numeric `0`; the Reference storage's `int64 required` declaration alone does not prove the mapped field was actually observed. | Preserve **field observation coverage** separately from rowset coverage. Under current `validationeval.Evaluate`, an unavailable/nil numeric value is an **evaluation error**, while an *observed* numeric `0` is `ResultNotSatisfied`. Do not collapse input inability into Business denial. |
| **E3: cross-route equivalence** | Same logical Subject and a provider-side `Complete=true` marker on two routes do not show that both expose the same cohort, observed fields, grain, ordering, or snapshot. | Do not join independent successes: an exact route/Scope/Field/Time witness is needed before transferring this proof to another Provider/Target. |

The merged [#2627 synthetic read-cohort Reference](https://github.com/hat47x/tei-platform/pull/2627) distinguishes bag/rowset completeness from field observation completeness (including `NULL` versus `absent`); its finite tests do **not** certify an actual DB. The merged [#2640 Query-result design](https://github.com/hat47x/tei-platform/pull/2640) also distinguishes a legitimately empty result, unavailable/partial retrieval, and a zero-result-looking projection. These are different Claim classes, not interchangeable `pass`/empty/default values.

### 15.8 Query operation order and non-output dependencies are proof inputs (plan; NOT RUN)

The authored Reference SQL is still the Source of Truth: `WHERE order_id = :order_id ORDER BY line_number, order_line_id`. A read-only DerivedIR/Explain is not a second meaning authority and must not replace the author's original SQL/DSL sugar merely to normalize a diagram or Markdown document.

For the minimum-row proof, depend on the **actual executed** predicate, sort, typed comparison, data-grain mapping and their order with respect to `FILTER`/`LIMIT`. Hypothetical operator rewrites illustrate why independently correct-looking outputs are insufficient:

- Consider a *separate illustrative* relation with A(line=0, eligible=false) and B(line=1, eligible=true); `eligible` is **not** a field claimed by the current order-line Reference contract. `filter eligible -> sort line ascending -> take 1` yields B, while `sort line ascending -> take 1 -> filter eligible` yields **empty**. These are not equivalent implementations of the same selected-cohort query. **No such Rewrite is asserted to exist in the current BP-05 execution.**
- A query can keep the same Result field IDs and positive fixture values while changing a **non-output** `WHERE`, `ORDER BY`, collating/comparison semantics, or `LIMIT`. A derived IR that contains a Sort node or self-reported `Lowering=full` is **not** independent proof of original SQL analysis coverage, intended operation order, or actual Provider enforcement. The merged [#2634 Go characterization](https://github.com/hat47x/tei-platform/pull/2634) demonstrates that coverage flags can be accepted even when deliberately incomplete analyzer references are supplied; it does not prove the actual SQLGlot worker is deficient.
- Consequently, the proof's **non-output dependency** (filter/sort/limit/order/attribute grain/Provider policy, Source and execution version) must be retained in the same existing review evidence packet for **selective requalification**. A display-only projection or artifact byte match does not revalidate the query's hidden dependencies or current authority.

**Owner decision and rollout unchanged:** classify the one `review-validation-scope` difference by the existing four human outcomes; remain `undecided` until the BP-05 target cohort and responsibility are accepted. Reuse #2387 (Query Explain), #988/#2619 (conditional Operator laws), #2506/#2622 (Context and actual Provider), #1174 (effect-time authority), #2582/#2600 (runtime versus build requalification), #2516 (disclosure), and #267 (meaning acceptance). An Issue/SHO representation may explain **which conditions would support the Claim**, but may not auto-connect Business Procedure to a Query, infer an unknown missing field as zero, declare a real Command blocked, or promote a synthetic success to a production guarantee. New tests, actual Provider, third-rule rejection, whole-cohort guarantee and Owner acceptance are **NOT RUN/NOT ACCEPTED** by this document change.

### 15.9 空集合の形式的成立と業務上の完了条件をOwner判断へ引き渡す（2026-11-03、計画・未受理）

この節は[空集合の意味に関する調査ノート（統合済みPR #2722）](../research/EMPTY_SET_COMPLETION_SEMANTICS_2026-11.md)の**実験計画への受渡し**であり、調査ノートの複製、新Core `EmptyPolicy`型、Lean導入決定ではない。§15.7のE0〜E3と§15.5のQ1〜Q5を再利用する。

BP-05の対象を人手の受理票では `(TargetSet L, EmptyPolicy E, Predicate P)` と**三つの判断欄**で扱う（候補表現でありCanonical DSL/Schemaではない）。

| Ownerが判断する欄 | 判断材料 | 技術が提供できる証拠／まだ代替できないもの |
| --- | --- | --- |
| `L`（対象集合と時点） | 編集対象の一明細か、提出対象の明細集合か。固定ID集合か、提出時の親子Relation述語で再解決するか | #2506/#2622のScope/Grain/Relation・取得範囲、#1708の既存Snapshot契約。SQLの`WHERE order_id`やFKだけから業務上の集合意味を採用しない |
| `E`（空集合時の扱い） | 空を許す・提出不可・別の業務確認を要求する等の**候補**から案件の扱いを判断。不在確認や集計から一律の既定を推測しない | 観測された`R=[]`、真に`L=∅`である独立の完全性根拠、空集合でのBusiness Action判断を別列で扱う。形式的`∀x∈∅ P(x)`の真だけではSubmit可としない |
| `P`（対象行の条件） | 明細番号正値、数量正値、明細番号一意性のどれをBP-05自身が負い、どれを別経路に委ねるか | 3規則の合成正常系は現存するが、`quantity`先頭行・`lineNumber`先頭行・`lineNumbers`集合の実際の入力被覆は別。第3規則不成立時の後続停止や本番業務保証は未証明 |

**受理順序**：構造観測（SQLの型・列・Sort、FK等）→曖昧な業務意味候補→Ownerが対象Scope・空集合扱い・規則責務を受理→該当Operationの実Provider/Hostで条件を実証、を区別する。これは[#1126のER/物理表現](https://github.com/hat47x/tei-platform/issues/1126)、[#267の既存Shared Acceptance](https://github.com/hat47x/tei-platform/issues/267)、[#1125の責務四分類](https://github.com/hat47x/tei-platform/issues/1125#issuecomment-6400716990)の分担を利用する。`line_number int64 required`、SQLの`ORDER BY`、物理PKやFK、型/ID一致は、明細番号正値を**業務必須**と認定する証拠ではない。過去の意味受理や実行資格は、Source byte一致・A→B→A・プラグイン/Operator再選択から自動復活しない。

**Ownerへ渡す一問（拡張するが増やさない）**：「BP-05が対象とする明細集合`L`と空集合時の扱い`E`は何か。その`L`の明細番号・数量に求める条件`P`を、BP-05自身が負うか、特定できる別経路が負うか」。回答は意思決定の候補であり、本文を直ちにBusiness Sourceへ反映しない。別経路を採用するならOperation、対象集合、規則、版と否定時停止の証拠を要求する。

### 15.10 意味受理、形式的証明、現在利用資格を混ぜない最小検収（NOT RUN）

`Predicate P`の形式的証明と、`R=L`の実観測、`E`の業務上の採否、`P`を評価したOperationの実行資格は相互代替不能。Leanによる形式系への写像は[調査ノート](../research/EMPTY_SET_COMPLETION_SEMANTICS_2026-11.md)で未採用・特定版未検証であり、BP-05の本計画では形式証明Engineや新Registryを実装しない。

| 既存例を再利用する独立期待（全件計画段階） | 必要な結論・停止条件 |
| --- | --- |
| `L=∅, R=∅, complete=true` と `L≠∅, R=∅, partial/hidden` を対照 | 同じ画面上の0件でも、前者はOwnerの`E`判断待ち、後者は全対象Claim根拠不足。双方を`satisfied`や`not-satisfied`へ自動縮約しない |
| 同じ公開SQL・データ型・固定行でも、Ownerが`line_number`を表示順序と解釈する世界と提出必須制約とする世界 | 静的構造・同一ID/型だけでは意味同値でもBP-05責務でもない。受理記録なしなら`undecided` |
| 受理済み`L/E/P`が同一でも、Query非出力Predicate、Sort、実Providerの完全性、Role/Operator版、対象Relation membership、現在Grantのいずれかが変化 | 当該Claimの依存閉包に沿って選択的再資格化（#2582/#2600）。過去Evidenceの歴史的保持と今回の`satisfied`／Consumer開示／Commit資格は分ける |
| 第3規則に0を与える合成負例から、現状の三規則正常系の`t.Fatalf`を見比べる | 評価`ResultNotSatisfied`と**実Operationの認可/Commandが呼ばれない**ことは別命題。後者は同一経路の呼出し証拠を得るまで未確認 |

**検収出力**は既存のread-only #1125レビュー票に`L/E/P + Owner decision/revision + evidence scope (static/synthetic/actual) + Provider route/slice/completeness/as-of + rule/operator/query versions + negative-stop proof + consumer use/authority + unresolved`を関連付けるだけとする。新しいStatus/Acceptance正本、形式証明器、Host、Gate、Query/Validation DSL実装は作らない。業務確認の履歴・正式Commit/Receipt・独立ObservationのClaimは[別S3設計](https://github.com/hat47x/tei-platform/issues/1125#issuecomment-6585850700)および#1174を参照し、BP-05の評価PASSに合算しない。

**段階**：S0（本節の静的・read-only整理）→既存S1の第3規則負例→S2の実Provider/cohort証拠→S3でOwnerの`L/E/P`受理と説明。実Go/DB/Provider/Lean/Studio/SHOの追加試験、正式Owner受理、#1119製品Gateの合格はこの文書では**NOT RUN/NOT ACCEPTED**。現時点の比較器`review-needed`、人手分類`undecided`を維持する。

### 15.11 全件成立と不成立の証拠非対称性（2026-11-03、read-only計画・NOT RUN）

§15.5のQ1–Q5と§15.9の`(L,E,P)`を前提に、全行正値の評価主張を**成立証拠**と**反例証拠**に分ける。これは新しい`Result`型やValidation Engineを追加する仕様ではなく、既存`review-validation-scope`とContext/Provider証拠に付ける判定理由である。

- `∀x∈L P(x)` の**成立**には、対象`L`の意味受理、必要な`R`の集合・Field完全性、同版・同時点、適切なRule適用の証拠が必要。部分取得`R⊂L`で見えている行がすべて正値でも、全行成立とは主張できない。
- 一方、`b∈L`と`P(b)=false`を同一Scope・有効時点・正しい型で**確実に観測した**なら、他の未取得行が存在しても当該時点の全称命題は不成立と証明できる。これは**限定的な反例証人**であり、対象`b`のmembershipが未解決、既知でない`NULL/absent`、無関係tenant、古いSnapshot、非開示情報の無権限取得からは導かない。Owner未受理のBP-05責務や正式Submit拒否への昇格も禁止する。
- `line_number_unique`については、`a,b∈L`、異なる明細identity、同一番号を正しい粒度・時点で観測した場合、**重複の存在**は全件取得せず反証可能。一方、部分取得で重複を見つけられなかったことから**全件一意**とは主張できない。`quantity_positive`も同じ全称／反例の区別を持つが、既存の`ORDER BY line_number`だけで数量の全件成立は証明できない。
- 真に`L=∅`の場合の形式的な全称命題は§15.9の`E`に従って別判定する。観測`R=[]`かつcoverage unknownを`L=∅`へ昇格させない。「明示的な0は規則不成立」と「未取得・部分取得によって規則の評価ができない」を混同しない。

| 既存の二明細を使う限定比較（いずれも新規実行NOT RUN） | Predicateに関して導けること／導けないこと |
| --- | --- |
| **U1** `R=[A(1)]`、提出`L`が全件だが取得`partial`、違反行は未観測 | **全行成立は未確立**。存在するはずのBの個数や値を一般Consumerへ漏らさない |
| **U2** `R=[B(0)]`、Bが受理済み`L`へ所属すること、数値0、時点を資格確認済み、他行未取得 | **当該時点の全行正値は反証可能**。ただし正式Operationを止める根拠には別途受理規則・現在性・認可/Commandへの接続が必要 |
| **U3** 同じ`B(0)`だが、BのRelation所属／tenant／行Versionが不明、またはFieldが`absent` | **違反の主張も未確立**。`NULL/absent`を0へ変換しない |
| **U4** 異なる明細A/Bの同一行番号を同じ`L`で観測／逆に部分集合で衝突なし | 前者は当該時点の**一意性違反の証人**。後者は**全件一意性を保証しない** |

既存#2506/#2622が行・Fieldの取得とContext完全性、#2582が同一Claimの変更・Membership、#2387がQuery Predicate・Source版、#2516が表示・非開示を担当する。**一件の有効な反例を検知できることと、全件を完全に取得できることは別の能力**であり、いずれも合成Fixtureの局所PASSを本番保証へ昇格しない。

### 15.12 同一Ruleのオンライン検査と後追いData Quality監査を分離（計画・NOT RUN）

[#2622のData Quality構想](https://github.com/hat47x/tei-platform/issues/2622#issuecomment-6598969026)と[増分監査の制約](https://github.com/hat47x/tei-platform/issues/2622#issuecomment-6599457368)を既存BP-05へ限定接続する。Ownerに受理された**同じ**`L/E/P`、Rule/Operator/Relation版から、(1)提出時の必要条件の評価と、(2)特定Snapshot・対象母集団の読取専用監査を派生できる可能性はある。ただし二つは**同じ実行資格ではない**。新たなRule正本、共通Scheduler、実行Host、全面スキャンEngineを提案しない。

- **Online**：そのOperationが必須とする入力・時点・Rule版・現在権限と、規則不成立時に実際に認可/Commandを止める証拠を扱う。実ProviderのCoverage/時点が不足する場合、業務`satisfied`へ推測しない。
- **Offline**：`asOf`、対象集合の粒度・Scope、Rule/Query/Provider版、Row/Field completeness、Snapshot、RLS/取得権限を固定した**過去時点相対のQuality Finding**を返す。監査で後から行番号0を発見しても、過去のCommitを「実行されなかった」ことへ書き換えず、現在のSubmit可否・再試行権限やOwnerの業務完了を自動判定しない。
- **増分再監査**：前回Checkpointと今回の間に**関連する全変更**（Value、追加・削除、Relation membership、Query Predicate/Sort、Rule/Operator/Schema、Provider/Scope）が被覆される証拠がある場合だけ、無関係変更を理由に全面再Scanを省略する候補とする。見えた変更が0件、Source byte一致、旧合格だけでは無変更を証明しない。[#1125の変更窓の反例](https://github.com/hat47x/tei-platform/issues/1125#issuecomment-6579870075)と#2600を再利用する。
- **同一Attemptの縦断**：最小の実証に進むときは、#2586の既存Scenarioで`Owner候補L/E/P→実Function Mapping/Query→Provider Result Slice→Rule評価→Authorization/Command呼出し→Consumer表示`の**同一Attempt**と時点を照合。#1174のEffect/Receipt、#2516の現在開示資格は独立列とし、別試験の成功を継ぎ足さない。既存の第3規則不成立時停止の負例（S1）は未実施のまま残す。

**実施順は既存S0–S3を変更しない**。追加したU1–U4とOnline/Offlineの記録はread-only検収票の比較欄であり、製品の新GateでもOwner決定でもない。BP-05の`canonical:false`、checker=`review-needed`、人手=`undecided`を維持する。Go/SQL/DB/Provider/SHO/Studio/Owner意味受理と同一Attemptの新規実測は本節では**NOT RUN**。

### 15.13 Ruleの正本・適用対象・実行Phaseを混同しない（2026-11-03、計画・NOT RUN）

[Data Model配下のData Script/Rule共有設計 #1129](https://github.com/hat47x/tei-platform/issues/1129#issuecomment-6587145160)と[旧仕様Parity／TEI適合を一Attemptで分ける #2783](https://github.com/hat47x/tei-platform/pull/2783)の**BP-05固有の受渡し**。基本的なField ValidationはData Modelの属性拡張で表す方向、独自条件・更新時の導出／複合条件はData-attached Scriptを利用する方向とする。ただし現行のCanonical `sales.order_line.quantity_positive` / `sales.order_line.line_number_positive` / `sales.order.lines.line_number_unique` を、事実確認なしに新Sourceへ移設・二重定義・自動受理しない。名称・構文・Rule定義格納形式の正式採否は#1129/#988/#267の既存Ownerが扱う。

| 共有Ruleの意味側（候補） | BP-05へBindingするとき確定が必要なもの |
| --- | --- |
| Field値の正値（`line_number_positive`, `quantity_positive`） | **一行のRule定義**と、**どの行を何行評価するか**という適用量化/Contextは別。FieldへのAttachmentだけでSubmitの全行検査実施済みとは扱わない |
| Relation集合の一意性（`line_number_unique`） | 業務上の関係Role/集合L/明細identity・同版をBinding。局所`lineNumbers`固定配列や部分集合の重複未発見は全件一意の証拠ではない |
| Data Script／Derivation／更新時Hook | Script原文と型付きRule候補、Attachment、依存Read Set、`before-write / in-Tx / post-commit`等のPhaseとEffectを区別。IFで書けることと実Operationで呼ばれたことは別。派生プレビューや通知発火を正式Commitへ昇格させない |
| 標準Function／API／バッチの利用 | 同じ受理済みRule ID・版を参照し、入口別にRuleのコピーや実行Phaseの差がないか確認。画面表示/Manual BP-04を新しい強制Ruleへ自動変換しない |

**Parity（P）**は旧Source/Scriptの忠実観測と実発火の独立期待、**Adaptation（A）**はTEI Context→Rule適用→同一Request/Grant/Owner-write/Receiptという安全適合、**Extension（E）**は追加機能とする。#2783のE0–E6受入票を**既存担当に渡すだけ**で、BP-05へ旧Runtime同義・実Provider・Owner受理の実績を持ち込まない。特に本計画のS1「第3規則不成立時の後続停止」は、Data Scriptの`IF`/Rule宣言やStudio表示だけで解消しない。

### 15.14 検査の配置・実行期限・免除と現在資格（設計・NOT RUN）

[Everything is Data横断計画 #2667 の§27.6（品質保証の範囲・期限・例外）](https://github.com/hat47x/tei-platform/pull/2667)を、§15.12のOnline/Offline分離へ**最小追加**する。Rule別の検査配置は案件Ownerの判断事項であり、特定のOnline必須条件をデータ品質Jobへ移すだけでWriteの許可根拠にしてはならない。

| 配置候補（未採用の検討欄） | BP-05/Claimに対する意味上の停止条件 |
| --- | --- |
| **Online mandatory** | Submitの必要条件なら、その**同一Operation**で対象L/版/Rule入力を評価し、`not-satisfied`なら定められた後続を止める。`input unavailable / partial / error`は業務不成立と同義ではないが、**成功資格も与えない**。実停止は既存S1で独立に証明 |
| **Deferred with deadline** | 期限・母集団・Snapshot・Provider/Role・完了証拠を保持。未実施・遅延・検査不能・局所PASSと、対象全体の適格なPASSを区別。期限内Job完了があってもRuleの全件合格・現在のSubmit許可とは別 |
| **Advisory / formal-only** | 顧客向けの説明・候補・静的形式検査であり、Ownerが必須としたWrite/Business Guardを代替しない。Leanの形式命題証明は取得被覆`R=L`や実Provider強制の証明ではない |
| **例外・免除（期間／範囲限定）** | Owner/Authority、対象・適用期間・理由・Rule版・失効条件を別に記録する候補。例外は既知の違反や過去Quality Findingを消さず、無期限の全行正常主張にもならない。例外によるWrite許可はHostの現在Authority/Effect Fenceとは別に検証 |

**独立した最小反例（既存Q1–Q5/E0–E3/U1–U4/DQX/DQAを再利用、全NOT RUN）**：同じ`L={A(1),B(0)}`について (a) RLS部分取得Aのみ＋Offline 0異常、(b) B(0)の有効な反例をOfflineで後から検出、(c) Quality Jobが期限超過しているのに実行Jobだけ`completed`、(d) 期限付き例外が失効、(e) 同じRule版でも異Snapshotの結果を合算、を各々**Scope付きClaim**として比較。検査結果には「実行されたか」「どの範囲の正常・違反を証明したか」「現在の利用資格」「何が未検証か」を別列で保持。長期Quality実績やJobメトリクスを、BP-05の実Command・実Effectの証拠としない。

**実証の優先順位は変更しない**。既存S1の第三規則のみ不成立→認可/Command停止、S2の同一Provider/Context取得、S3の`L/E/P` Owner受理を先に進める。Data Script Parityと共通Rule実装は#1129/#2783、品質の監査/増分/Jobは#2622/#2667、業務Actionと独立Effectは#1174、開示は#2516へ委譲する。新Script正本・Quality Scheduler・汎用Host・Rule Engine・Gate・ADRは追加しない。Go/PG/旧Driver/EKI Job/Studio/SHO同一Attemptは本追補で**NOT RUN**。

### 15.15 Validation成立対象と実Write対象の意味接合（2026-11-03、計画・未実証）

§15.5の対象集合`L`と、§15.13の`Rule/Attachment/Phase`を、実Commandの**目的・対象・提案変更後状態**と接合する。これは[main統合済みCommand→EffectCommit Adapter #2773](https://github.com/hat47x/tei-platform/pull/2773)の現物および[別担当Draft #2780](https://github.com/hat47x/tei-platform/pull/2780)の**特性確認**から判明したBP-05固有の不足であり、Adapter/Runner/Hostを本計画で改変しない。

- 既存`internal/driver/final_gate_b_scenario_test.go`は、同じBrowse `orderID`を合成Command入力の`orderId`へ渡し、test-only Adapterがその一致を検査する**正常系**を持つ。これは**実製品の`commandadapter.Executor`への実Request拘束、実DB Target、事前Validation→Commitの同一Attemptを証明しない**。現在mainの`internal/security/effectcommit/commandadapter/executor.go`は明示`Payload`を受けると`request.Inputs`からの組立てより優先し、両者のTarget意味照合をAdapter単独では行わない。
- [Draft #2780 MC-B01](https://github.com/hat47x/tei-platform/pull/2780)は、Command入力`ORD-1`と明示Payload`ORD-2`の乖離がFake Providerの`WriteResource`まで到達する**現行構成の特性を限定Go試験で観測**したもの（当該Draftのexact-head focused Go 4/4 PASS報告）。**安全に拒否できたこと、BP-05経路を実走したこと、実PostgreSQLに誤更新したことの証拠ではない**。
- 同一のOrder Identityでも、評価した保存済み`R0`と、Editing Sessionの未保存Patch/Commandの**提案後状態**が異なる場合がある。提出前必須条件が提案後状態へ適用されるなら、保存済み`R0`の正値は未保存`B(0)`を保証しない。逆に、無条件にDB再読取りだけすればよいわけでもなく、評価対象の「現在保存状態／書込み予定状態」はOwnerが定めるPhaseに従う。親Orderの`ExpectedVersion`だけで、子明細の追加・削除やRelation変更までFenceされたとは推測しない。
- 対象Bindingは単なる`ID equality`規則ではない。SubmitがOrder statusを更新し別Resourceへ正当な予約Effectを生む場合、異なるTarget ID/Relation Roleが正しいこともある。**Accepted Action→Validation対象集合→typed Intent→実Payload/Provider Target→必要Effect対象**を、Ownerが受理したRole/Relation/Scope/Versionへ対応付ける。暗黙の同一視と、根拠のない異対象許容の双方を禁止する。

| 実証票（既存Ownerへ引渡し、BP-05同一Attemptでは全てNOT RUN） | 分離すべき結果 |
| --- | --- |
| **VB-01 Target drift**：`L(O1)`に対する三規則PASS、Command選択`O1`、明示Payloadは`O2` | Rule PASSと`O2`のWrite許可を混ぜない。対象の対応根拠なしなら`unqualified`、実行前拒否が必要な契約なら**拒否を試験** |
| **VB-02 proposed post-image**：`L(O1)`の保存済み行は全て正値、未保存Patchで明細Bを`0`に変更し、その値を提出 | 旧Readの`satisfied`と**提出予定の状態**の判定を分離。未保存Patch/選択epoch/Field観測/Rule Phaseが必要 |
| **VB-03 membership/version drift**：Header CAS版は同じだが、Validation後にChild B(0)が対象へ加入 | Parent版一致のみで`L`の現在完全性を証明しない。既存Context/Provider/Hostが保証する適切な再評価・対象集合Fenceを実測する。新分散Txを要求しない |
| **VB-04 legitimate multi-target**：受理済みSubmitが別Roleの予約Targetへ書く | 単純な`O1==effect.Target`強制は誤拒否になり得る。受理済みAction purposeとTarget RoleのMappingを必要とし、Targetの一致「だけ」で意味受理しない |

### 15.16 Command結果・Effect観測・業務完了Claimの階層（2026-11-03、計画・未実証）

[Draft #2780 MC-C/D](https://github.com/hat47x/tei-platform/pull/2780)、[実装済み`SQLEffectObserver` #2771](https://github.com/hat47x/tei-platform/pull/2771)、[別担当のPostCommit Claim Draft #2671](https://github.com/hat47x/tei-platform/pull/2671)を、**BP-05がClaimへ付けてよい資格の上限**として引き継ぐ。現mainのAdapterは`effectcommit.StatusCommitted`と`StatusReplayed`をともに空`command.Result{}`へ写すため、**Command成功だけから「今回の新規Write」を認定しない**。また、一般`COMMAND_EXECUTION_FAILED`の内部に型付き`CommitUncertainError`が残り得るので、Commit未確定を「書込みなし」や「新IDで自動再送可」へ縮約しない。

| 証拠の継ぎ目（既存試験に受渡す） | 間違ったBusiness Claimを防ぐ条件 |
| --- | --- |
| **VR-01 committed vs replayed** | 同一OperationIDの初回・Replayは`command.Result{}`だけでは区別不能。Runnerのqualified Receipt/LookupとRequestの同一性・Target版を照合し、`effect applied`と`this attempt newly committed`の主張を分離 |
| **VR-02 uncertain/outer failure** | `CommitUncertainError`、外側`UnitOfWork.Commit`失敗と内側Runner確定（MC-C01はFakeの特性）は、効果なしの証拠ではない。同じOperationIDのLookup、同一物理Tx/再試行適合と独立Readは#1174が実証する |
| **VR-03 unrelated positive read** | 現`SQLEffectObserver`は設定SQLのcount>0で`Observed=true`になり得る。Query/Args/対象Identity/Operation/Reader Cutに拘束されない`SELECT count(*)`は、無関係な過去行でも正数となるため、**今回のO1のEffect**を示す証拠ではない |
| **VR-04 business completion** | 実Receiptと対象へ束縛した独立Readが一致しても、Ownerの`L/E/P`、BP-04手作業・外部Handoff、現在Consumer開示資格は別。#2671/#1125のS3へ委譲する |

**次の実証を小さく保つ**：先にS1の「第3規則のみ不成立→認可/Command呼出0」を既存test-local継ぎ目で再現し、同じfixtureの選択Identity／Payload/適用対象を観測できる受渡し欄を残す。S2以降、対象`O1`と提案後状態・子集合版・実Target/Provider Scope・OperationID・RequestHash・Receipt・独立Readを**同一Attempt**に揃える。異なる既存PRのGo PASS・実PG局所PASSを足して本件End-to-End資格としない。UI/SHOへは「規則PASS」「Write確定」「過去Replay」「確定不明」「Owner完了」の違いを、現行開示権限で説明する。これは既存#2597 Editing Session、#2598 Intent→Request、#1174 Effect、#2586縦断、#2516開示への**受渡し**であり、新しいAction/Result Schema・Host・Recovery Engine・Gate・ADRは追加しない。BP-05のchecker=`review-needed`、人手=`undecided`、正式実Provider/Owner受理/三規則本番経路は**未実証**。
