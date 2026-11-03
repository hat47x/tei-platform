# TEI → EKI P0 external contract: real OpenAPI shape / lossy-projection guard

**Status:** isolated, test-only compatibility pressure. Not a TEI Job producer, EKI API client,
runtime adapter, authorization/commit layer, or a proof of real Provider/Worker integration.

## Why

TEI Job/Flow designs may contain independent `notBefore`, `preferredCompletionAt`,
`latestUsefulAt`, an exact Executor digest, WorkOrder parameters, resource Group/data-residency
constraints, and non-idempotent Business Effects. The present EKI P0 OpenAPI exposes a
different, smaller contract. Submitting an apparently accepted Task must not silently
discard a required semantic or turn `effect_mode=write` into `none`.

This snapshot comes from **actual EKI repository Source**, not a fabricated API:

- Source: `hat47x/eki-fabric:spec/external/openapi.json`
- Git blob SHA: `388f2e3ad32ba663b4777fcfb4304b0cf6f48f3f`
- P0 OpenAPI 3.2.1, API info version 0.1.0.
- The retained JSON contains only selected schemas and path/method/response declarations;
  it is a **versioned external test fixture**, not an alternative EKI API definition.
- EKI Owner: [#112](https://github.com/hat47x/eki-fabric/issues/112),
  [#266](https://github.com/hat47x/eki-fabric/issues/266),
  [#67](https://github.com/hat47x/eki-fabric/issues/67),
  [#115](https://github.com/hat47x/eki-fabric/issues/115).
- TEI Owner: [#1130](https://github.com/hat47x/tei-platform/issues/1130),
  [#2499](https://github.com/hat47x/tei-platform/issues/2499),
  [#2514](https://github.com/hat47x/tei-platform/issues/2514).
  Crosslane planning: [Draft #2667 §§35–36](https://github.com/hat47x/tei-platform/pull/2667).

## Repeatable check

```sh
node --test tools/reference-tei-eki-contract/probe.test.cjs
```

The tests read only the selected EKI schema fixture and exercise 19 assertions/negative
cases WP-01..19. They classify TEI-like Job requirements into:

- `not-representable`: a required field/semantic cannot be expressed in the current v1 request.
- `wire-shape-only`: no known structural blocker, **not** an EKI Admission or execution success.
- `unresolved`: encoded fields still need semantics, real Artifact bytes, Worker delivery,
  Provider admission, Host effect, or current authority verification.
- `runtimeQualification=NOT_RUN`: all tests are source-static / synthetic.
  
The fixture was selected from the current OpenAPI source rather than invented, but
**test-code classifications are proposals**, not canonical EKI behavior. EKI may have
runtime behavior that is not described in this subset; change the fixture only after
re-fetching the actual EKI OpenAPI and recording its new Git blob. Newer schema success
is not inherited from the old exact-head test result.

## Observed incompatibilities / not yet proved

1. EKI `TaskRequest.parameters` exists, but its source description says parameters are
   **not passed to workers** (WorkOrder has no parameters field).
2. EKI `ReliabilityRequest.effect_mode` admits only `none` in P0. TEI business
   mutation, Script side effects or Host Commit cannot be disguised as pure compute.
3. `ExecutorRequirement.version_constraint` is recorded but not evaluated by
   admission; exact executor byte binding is not represented by a `digest` field there.
4. EKI provides one `deadline` while TEI may require distinct earliest/soft/last-useful
   times, including result-body delivery and downstream verification. Mapping requires
   explicit scope acceptance; renaming `latestUsefulAt` to `deadline` is not automatically safe.
5. EKI result/status fields and artifact retrieval path cannot establish a verified
   TEI/SHO Consumer body read or a business DB Commit/Receipt. The OpenAPI currently
   declares HTTP 501 among artifact GET responses, which is **not** a claim that all
   deployments necessarily return 501.
6. EKI does not accept an arbitrary Resource Group name / data-residency assertion
   as an authorized admission condition through these `TaskRequest` fields.

## Next real evidence

Run an actual TEI Job producer and an actual EKI Control Plane against the same accepted
Source/Executor/input bytes, recording EKI Task acceptance, fingerprint, Worker/WorkOrder
delivery, input Artifact ingest/body, output body actual read, Host/Provider effect,
deadline semantics, current Authority, and failure/unknown. Only then upgrade
`wire-shape-only` to an executable claim **for that exact revision and scope**.
The API is EKI-owned; TEI must not create a second Scheduler, Grant issuer,
Business State Store, or EKI OpenAPI implementation.

No source model changes, formal DSL adoption, new Core type, live network traffic,
credentials, real EKI/TEI execution or Owner semantic acceptance are performed here.
