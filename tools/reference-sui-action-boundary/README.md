# SUI Native Action Boundary — reference-only

This probe tests a narrow interaction contract for specialized, non-business applications. It is not a production UI host, SUI migration, or implementation of TEI runtime authorization.

## Ownership

| Concern | Owner |
| --- | --- |
| Pointer, selection, drag preview, cancellation | Native UI component |
| Application/action binding | Reference Host |
| Authentication and authorization | Runtime integration **not yet connected** |
| Document revision check, validation, atomic persistence | Application-owned Store |
| SUI DocumentV1 schema, Card/Island semantics, SafeMode | SUI native code; **not implemented here** |
| Semantic definitions and extension evaluation | TEI semantic layer; **not connected here** |

`Host.Commit` accepts `ApplicationID`, `DocumentID`, `ActionID`, `ExpectedRevision`, and an opaque payload. It rejects missing identities, ambiguous bindings, unbound actions, and absent/denied authorization. The Store must atomically compare revision and persist; the Host alone **does not** prevent concurrent writes. A Store that ignores the revision violates the contract.

## Run

```sh
go test ./tools/reference-sui-action-boundary/actionboundary
```

No test execution or CI result is asserted by this document.

## Existing coverage and gaps

Unit tests cover a successful commit, stale revision, authorization denial, missing authorizer, cross-application binding rejection, and reuse with a second mock application. Preview/cancel is a no-call architectural convention, not a tested browser event integration.

**Unimplemented evidence:** actual SUI React Canvas adapter, SUI DocumentV1 mutation/roundtrip and field preservation, actual TEI `driver.Resolver.InvokeCapability` authorization, user/session binding, current runtime permissions, concurrent revision race, unknown Card rejection, semantic extension validation, SafeMode/export protection, browser packaging, performance and change-amplification measurements.

The next implementation should first bind the existing runtime authorization and action resolver. Do not replace them with `permit(true)` outside tests. Only then connect SUI native DocumentV1 operations and add negative integration fixtures. Do not add SUI-specific Core kinds or treat a mock second consumer as evidence of a reusable production plugin.

Tracked by GitHub issue #993 and repository-local `plan/issue/000052-sui-reference-adoption-and-plugin-surface-evaluation.md`. GitHub issue #52 is unrelated.
