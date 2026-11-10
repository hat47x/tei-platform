# EV-R0 — Studio four-lane synthetic Evidence display

**Status:** standalone read-only Reference **candidate** for [#1124](https://github.com/hat47x/tei-platform/issues/1124) M3. No Product/Policy/Owner acceptance. No core schema, second decision index, runtime Host, authority engine, source writer, UI transport or release mechanism.

## Exact reference base

- main: `dd773c1ad8735a8c6aeb67ef51f8437ac3320e17` (2026-11-04).
- Existing Decision Index schema: `tei.reference.decision-index/v0` from `tools/reference-shared-acceptance/decision.go`; witness/status/diagnostics are independent evidence, never Owner meaning acceptance.
- `plan/design/mockups/annotation-decision-index-projection.mjs` and `annotation-evidence-list-projection.mjs` already test fail-closed display and human confirmation vs acceptance. EV-R0 does **not** edit or reimplement them.
- [#2516](https://github.com/hat47x/tei-platform/issues/2516): claim-relative dependency qualification, disclosure and actual recipient delivery are separate; no second Policy evaluator.
- [#2604](https://github.com/hat47x/tei-platform/issues/2604): physical Artifact Set publication and external Reader byte consistency are owned there; EV-R0 cannot infer them from Manifest/digest.

## Synthetic fixture contract (not a TEI ABI)

`projectEvidenceView(frame)` accepts only `tei.reference.studio-evidence-r0-fixture/v0` and a `release.state='prepared-public-fixture'` marker. It projects four separate lanes: `operation` (selected S, candidate T, revision, epoch), `declaredTransition` (Action), `observedEffect` (Command / Effect readback / Recipient Delivery independently), and `ownerAcceptedMeaning` (always `unknown` in EV-R0; source Decision, Review and witness fields remain separate).

Only **synthetic public, pre-filtered** data belongs in `frame.public`. The release marker is intentionally forgeable **and has no security or product authority**. A real Consumer must obtain its **current** policy-approved display slice from existing Authority/Disclosure owners **before invoking** this presenter; EV-R0 cannot decide what may be disclosed, sanitize arbitrary secrets, assert a trusted snapshot or revoke already displayed DOM/cache/recipient copies. For non-prepared states (`unknown`/`withheld`/revoked), it returns the same constant generic fallback without examining `frame.public`. Unrecognized public fields and values fail closed without reflecting raw inputs in error messages.

`DecisionRecord.decisionWitnessStatus` and `witnessDiagnostics[].code` are mapped here as **already allowed** synthetic display fields. Real Decision records, signer identities, raw diagnostic messages, private source paths and internal dependency graphs must never be passed through without upstream release qualification. `sourceRevision=current` in a Decision is not operation chronology; ledger absence cannot prove an operation never happened. Equal `OperationID`/Attempt strings do not prove effect causality. Reference `Committed=true` is not external DB commit. A received HTTP request remains received even if the later model/command responds with an error.

## Small acceptance set (fixture-only, not M3 product acceptance)

| ID | Independent expected observation | Excluded Owner responsibility |
| --- | --- | --- |
| EV-R0-01 | Pure projection, S != T retained; invalid schema or extra secret fields refused | ModelRef/Intent resolution (#1120/#2500) |
| EV-R0-02 | `reference/committed` retains `effect=unobserved`, `owner=unknown` | Effect/Receipt (#1174/#2771/#2773) |
| EV-R0-03 | Different Operation/Attempt is not joined; equal strings still not causal proof | Observer/Provider binding (#2586/#1174) |
| EV-R0-04 | verified witness + invalid diagnostics coexist; no Owner meaning | Shared Acceptance and #958/#267 |
| EV-R0-05 | unchanged WorldFP but moved Source Index → stale locator remains stale | Source re-read/relink (#2626/#2604/SHO) |
| EV-R0-06 | `failed` command + `received` delivery + `unobserved` effect remain distinct | Real recipient egress (#2515/#2516) |
| EV-R0-07 | Two unseen worlds produce identical public fallback, without existence/count leakage | Actual disclosure noninterference (#2516/#1178) |
| EV-R0-08 | Missing release and malformed public display data do not publish details | Current Policy/Grant enforcement (#2516) |

Run focused synthetic checks with `node --test reference/tei-studio-evidence-view/project.test.mjs`. Passing tests establish **only the local pure projection**; they do not establish actual CLI/Source byte, Decision ledger readback, Query effective read-set, Manifest→Reader generation, grant revocation and old DOM/ARIA/cache erasure, Web/Desktop/SHO/AI receipt, real Provider effect or Owner meaning.

## W1 independent review and automated verification boundary (2026-11-06)

- W1 reviewer report (provided externally): isolated checkout of the **exact original head** `eee5ac743fad5868edc0534db128ba1ffa31b712`; Node **24.21.0**; EV-R0-01..08 **8/8 PASS**, no definite defect in the synthetic candidate scope; only the original three fixture files differ from its base. The report is a **separate independent local run**, not GitHub Actions evidence for these tests.
- The pre-change Actions for that head (Repository Health, Studio PoC A and dependency vulnerability checks) finished successfully. Those workflows **did not establish execution of `project.test.mjs`**. The old PR text's `GitHub exact-HEAD CI NOT RUN` must not be read as describing the later existing Actions status.
- The dedicated `.github/workflows/tei-studio-evidence-view-r0-ci.yml` added after W1 runs `node --test reference/tei-studio-evidence-view/project.test.mjs` with pinned Node **24.21.0**, exact PR candidate head checkout, logged file blob IDs and post-test clean tree. It is triggered by changes to this fixture or the workflow; it is **not** a new product gate, an authentication test or a second Policy Engine. **Do not record PASS for the new workflow before its own run completes.**
- The implementation intentionally **does not map** real Product `EffectUnknown`/commit-response-loss to a display disposition: `unknown` is outside the present synthetic enum and will be rejected rather than silently rendered as `failed` or `unobserved`. The actual effect/receipt and Consumer mapping belong to their existing Owners. This must be revisited before an actual Command-to-UI connection.
- These tests never authenticate the forgeable `prepared-public-fixture` marker or observe actual protected reads, production DOM/ARIA/cache/logs, outgoing network bytes, current Grant/Policy, Source/Artifact/SHO consumer readback, Provider causality, or business Owner meaning acceptance. A low-privilege two-world **constant fallback** is not a proof of actual Consumer noninterference.

## Non-overlap and integration order

Do not change Workbench/Graph/Command, PR2/PR3 (`#2583/#2457`), the generic Script-first Editor (`#2631`), the existing annotation mockups, `#958` evidence rebinding (`#2829`), internal Provider raw snapshots (`#2812`), Policy tests (`#2781`), or Artifact publication tests (`#2817`). This fixture is a separate M3 **display probe**, parallel to #1124's M0→M1 actual CLI/Consumer and PR2/PR3. Later, hand only qualified and disclosed Producer output to the UI; every product-level Owner maintains their own test and acceptance.
