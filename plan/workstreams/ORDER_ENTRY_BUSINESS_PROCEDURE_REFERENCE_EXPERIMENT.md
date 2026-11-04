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
