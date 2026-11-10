# Subsystem partition: fixed-owner / must-link exact reference

Status: experimental, **read-only, caller-supplied candidate-only**. Issue [#3250](https://github.com/hat47x/tei-platform/issues/3250); design [Draft #3255](https://github.com/hat47x/tei-platform/pull/3255). This is not an Accepted Source adapter or a product optimizer.

## Purpose and exact result

The Go package exposes only `Evaluate(Problem)`. It neither reads external files nor edits Source or calls a Provider. Its input must explicitly supply:

- Unique Function IDs, each with a nonempty finite set of allowed Subsystem memberships.
- Distinct direct-write **relation occurrence** IDs and candidate Data IDs; each occurrence states the Function, fixed candidate Data owner, exact operation C/U/D, source locator, and **positive integer** weight. Reads, delegated effects, procedures, SQL/Trigger execution, workload frequency and migration semantics are **not** inferred.
- Optional undirected must-link relations between Functions. Every linked component must share one Subsystem.

Under **only these constraints**, form connected components of must-links. For each component, intersect the Functions' allowed Subsystem sets, then choose the allowed Subsystem retaining the most direct-write weight inside the component. Sum all remaining cross-owner write weights. Since components do not constrain one another, this yields an **exact optimum for the supplied graph**. It is not merely the independent-Function lower bound.

If a component has no common allowed Subsystem, the result is `infeasible` with **no numeric optimum**. Incomplete or invalid inputs return an error rather than silently substituting an empty graph or treating an unresolved owner as a zero-cost edge. The result lists deterministic placements, component scores and crossed relation IDs; ties select the lexicographically first Subsystem.

This proof does **not** apply directly to cannot-link, capacity limits, time-dependent owners, split Functions, cross-component business/transaction constraints, changed CRUD operations, negative weights, or unknown edges. Callers must not omit such constraints and then present the result as globally optimal. Known-only optimum zero never establishes system-wide zero unless the separate Source/Claim closure is qualified.

## Scope and authority boundaries

The **candidate input is not a qualified Source**. A caller-supplied Owner/Function I/O relation or Source fingerprint is not a business owner acceptance. If a Data ID denotes two different scopes, the caller must first disambiguate it by the existing TEI typed identity contract. This reference intentionally does not implement another Source Index, parser, Graph, Owner policy, acceptance store, optimizer DSL, CLI, transport or runtime.

Use [tools/reference-subsystem-cud](../reference-subsystem-cud/README.md) for the existing read-only cross-view candidate diagnostic, not this evaluator. Use the accepted Source/Identity/Acceptance contracts (#1120, #267) and the latest three-view read-admission work ([Draft #3492](https://github.com/hat47x/tei-platform/pull/3492)) before binding real TEI ownership/Function I/O to this input. The Draft's synthetic or limited Inspector read is not proof of Accepted Subsystem Owner or whole-program direct-write closure.

Any result returned to Studio, CLI, MCP, AI or another consumer must first use a **currently qualified, disclosure-safe Public Slice**. This pure package has no authorization or confidentiality enforcement: it can reveal candidate membership, scores and hidden edge identifiers. Filtering nodes from a private-world optimizer result is not sufficient; see #2516/#1124.

No output represents actual CRUD effects, a Tx receipt, a Source-changing proposal, a migration cutover or formal Owner approval. Do not use it to issue writes. Owners for application/operation/Source, Studio UI, independent expected, disclosure, and runtime execution remain separate.

## Tests and qualification

Run at the repository root:

    go test ./tools/reference-subsystem-partition
    go vet ./tools/reference-subsystem-partition

Tests include the design cases SP-37/38/39/41, a must-link counterexample in which independent lower bound 0 is not feasible, deterministic tie-breaking, input rejection, and an **independent exhaustive assignment oracle** over 1,372 small allow-set/must-link combinations. The exhaustive test does not call the evaluator's component-contraction procedure.

Further needed (not implemented by this package): Source closure and actual Accepted owner/Function I/O producer (SP-42), Current Grant and public noninterference (SP-31/43), Boundary Repair/Effector/Tx/Recovery (SP-44), and trusted full repository-health-check on the exact merged revision. Local/CI test PASS is not evidence for those obligations.

**No main integration or Owner Meaning Acceptance is implied by creating this reference.**
