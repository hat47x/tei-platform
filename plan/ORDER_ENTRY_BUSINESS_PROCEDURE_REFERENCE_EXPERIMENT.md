# `order_entry` Business Procedure Reference Experiment

- Status: Lane C experiment design / not Canonical
- Purpose: restore the X-TEA Business Model viewpoint around the existing v1 `order_entry` Reference without introducing a Workflow Engine or duplicating Application meaning
- Parent audit: `docs/audits/UPSTREAM_BUSINESS_SYSTEM_DESIGN_VITALITY_AUDIT_2026-09.md`
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
