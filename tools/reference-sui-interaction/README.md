# Native Interaction / Application Action bridge — Reference E1

This is an **executable, browser-portable Reference candidate**, not a production TEI Runtime or SUI Canvas integration. See [ISSUE-000052](../../plan/issue/000052-sui-reference-adoption-and-plugin-surface-evaluation.md) and GitHub Issue #993.

## Product question

Can a specialized non-business Canvas keep its native implementation while TEI reuses a domain-neutral interaction-to-application-action boundary, without making Card a TEI Canonical definition or introducing a Core Canvas kind?

The first experiment isolates four responsibilities:

| Layer | Owner in this Reference | Does it perform durable writes? |
| --- | --- | --- |
| Gesture begin / preview / cancel | browser-local `beginGesture` / `previewGesture` / `cancelGesture` | No |
| Intent construction | `finishGesture`: application ID, Document ID, expected revision, target, position | No |
| Command preparation | `commitIntent`: authority check, snapshot/revision check, domain adapter and validation | No (until `save`) |
| Authoritative persistence | caller-provided `save` with **atomic compare-and-swap** | Yes, only on accepted commit |

`moveSuiCard` is a **Reference-only SUI adapter**, separate from the generic gesture and command boundary. It updates just `cards[].x/y` in a cloned `DocumentV1` snapshot. The test fixture deliberately includes a held Card, its source, an unknown Edge type, an Affiliation, review metadata and an unknown document field to detect accidental data loss.

A second, fictional `diagram` consumer uses a node-specific adapter to verify that the generic boundary need not know the SUI Card schema. **This is not proof that a second real Application has adopted the interface.**

## Run

Requires Node.js >= 20 for the built-in test runner and `structuredClone` in tests; **no external npm packages** and no network are required.

```sh
node --test tools/reference-sui-interaction/bridge.test.mjs
```

Local execution on 2026-10-08 using Node.js v22.16.0: **11 tests passed, 0 failed**. Source committed to this Reference matches the locally executed files by Git blob SHA.

## The negative cases exercised

- Pointer preview and cancel cause zero calls to storage.
- App binding mismatch fails before authorization, loading or persistence.
- Authorization denial fails before reading the document.
- Stale `expectedRevision` fails preflight.
- Concurrent revision change fails at storage's conditional save; a read-time check is insufficient.
- Domain validation rejects without mutation.
- Missing or duplicate card ID rejects without mutation.
- Non-finite coordinates fail before persistence.
- Missing policy / validation port fails closed.
- Unchanged domain fields, including hold and future unknown relation values, survive accepted coordinate change.
- A domain-specific adapter can be swapped without changing the generic gesture / commit code.

## Important authority and implementation limits

1. No actual React Canvas event source is wired yet. `beginGesture`, `previewGesture`, `finishGesture` simulate the interface a native Canvas adapter would call. Rendering, hit testing, polygon geometry, pointer capture, transform conversion, keyboard operations and accessibility are **not** covered.
2. `ports.authorize` and `ports.validate` are injected test boundaries, not TEI's actual Permission / semantic validator or SUI's full DocumentV1 validator. They **must** be replaced by the authoritative runtime services before integration.
3. `ports.save` must verify both scoped authorization and `expectedRevision` **atomically** with durable persistence. The reference only demonstrates injected CAS via memory. Passing the initial permission and revision checks is not sufficient against a concurrent write or permission revocation.
4. The real SUI `PUT /docs/{id}` allows Last Write Wins if `If-Match` is absent; an actual adapter must always supply a valid `If-Match` / ETag or equivalent storage precondition. A successful Reference CAS is **not** evidence that SUI currently enforces optimistic concurrency by default.
5. SUI DocumentV1 and external safe-output rules remain native authority. An opaque metadata-preserving clone is **not** a guarantee of correct SafeMode export or AI context filtering.
6. `position.set` and its target kind are small experiment-local action terms, not adopted TEI Core / Plugin / DSL vocabulary. Browser portability of ES modules does not establish Target-specific packaging, bundle size, performance or memory compliance.
7. Only the test boundary has been run. No TEI First Web Target, full repository `go test`, installed SUI frontend integration, persistence server, production deployment or real second-consumer tests are claimed.

## Next comparison milestone

Integrate the native SUI Canvas via a thin adapter; bind the `authorize / load / validate / save` ports to actual runtime mechanisms and make permission/revision checking authoritative at commit. Replay one real drag/cancel/collision flow; compare changed files, duplicated definitions, manual synchronization and user-visible behavior against the existing native SUI implementation. Only then decide whether this interface belongs in a reusable Interaction Host plugin.
