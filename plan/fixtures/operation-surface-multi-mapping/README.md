# PM-M1：複数Persistence経路の標準Operation導出・実装世代の特性把握

- Parent: [#2611](https://github.com/hat47x/tei-platform/issues/2611)
- Upstream implementation candidate: [PR #2587](https://github.com/hat47x/tei-platform/pull/2587), exact starting head `dc2842d241082d6dee89343cfbd9294153d0d403`.
- Independent comparator: [PR #2627](https://github.com/hat47x/tei-platform/pull/2627), route PR-01〜12 + read slice RC-01〜20. This document is **not** an alternative implementation of that comparator.

## Why this PR is stacked

The source of `internal/operations/DeriveOperationSurface` is currently in PR #2587, not base main. This branch starts at its exact head and **adds tests and a test note only**, leaving the upstream owner branch and established operation derivation untouched. Merging upstream PR #2587 and merging this characterization are separate choices. This PR's base is the upstream work branch to show the incremental diff.

## Seven source-based candidate characterizations (product Go tests)

| Case | Fixed input | Expected CURRENT behavior, **not** target normative behavior |
| --- | --- | --- |
| **PM-M1-01** | Same `sales.order`, two writable DB requirements `archive_db`/ `sales_core`; mappings have different table and create columns | `Operation.ID` collides; `create` uses archive mapping because alias is sorted first; missing primary alternative is not represented in `Conflicts` |
| **PM-M1-02** | Rename `archive` to `z_archive`, `primary` to `a_primary`, keeping DBs and mappings fixed | `Operation.ID` unchanged, selected `Basis` changes from archive to primary, `create.Inputs` changes |
| **PM-M1-03** | Archive alias write-only, Primary alias read-only | Archive-local `list/conflict` gap and Primary-derived available `list` coexist; read permission is path-local |
| **PM-M1-04** | Modify table of unselected Primary mapping | Pure derived Surface unchanged because this mapping was discarded; workspace-level revision/digest may still change |
| **PM-M1-05** | Add a persisted status column to **selected** Archive Mapping; same ID/Table/PK | `create.Inputs` changes but `Operation.ID` and `Basis` stay the same |
| **PM-M1-06** | Change physical Storage of selected Archive Mapping order_id from Int64 to Varchar; same ID/Table/PK | Entire pure derived Surface remains equal; no claim that Workspace/Artifact ModelRevision is equal |
| **PM-M1-07** | Change Table of the **selected** Archive Mapping without altering ID/PK/Columns | Positive control: `Basis` changes, but `Operation.ID` and `create.Inputs` remain stable |

## Impact dependencies that are now independently visible

`mappingBasis` in candidate `surface.go` hashes **Mapping.ID + Table + PrimaryKey** only. It intentionally or unintentionally excludes individual ColumnMapping attributes, physical Column, Storage and Nullability. The operation surface uses `createInputs` from mapped Attributes, but changes to ColumnMapping can leave the Basis unchanged (PM-M1-05) or leave the entire pure Surface unchanged (PM-M1-06). PM-M1-07 is a positive control establishing that a Table change **does** affect the same Basis.

**Key acceptance question**: which consumers legitimately use `Operation.Basis` as a minimal route identity, and which consumers must independently track accepted Source/WorldFingerprint, physical realization, type, nullability, actual binding and target generation? A full physical change should never be inferred to be irrelevant from unchanged Basis alone. This is not a demand to put every physical field in the Operation ID, and it does not claim that existing `WorldFingerprint` or Workspace-level source revision fails to change.

For #2624's separate Function Mapping and Rule Dependency corpus, distinguish a change to an **actually selected** mapping's field and type, a change to an **unselected** mapping, and an Alias rename that silently changes the selected mapping. Explain the observed dependency set in the final Artifact/SHO/Target proof rather than inventing a global regeneration rule.

These tests intentionally characterize a **known candidate limitation**. When accepted behavior is implemented, update/remove these characterizations in the same change, replacing them with affirmative tests that preserve routes or explicitly explain ambiguity.

## Proposed acceptance of the future repair

1. Every valid input mapping candidate can be traced with its provider/database/alias/scope/meaning revision and is not silently lost because another alias sorted first. The Owner decides if multiple mappings are valid in one World; if not, the existing Validator must reject them explicitly. A second Operation ID registry is unnecessary.
2. Application-level `list/read/create/update` availability does not hide a route-local gap and does not convert a route-local failure to a global failure.
3. Binding eligibility is separate from semantic business Action selection: `effect.kind + data` is insufficient to distinguish `submit` vs `cancel` (see #2485 and #2598).
4. A consistent operation surface/digest is a **derived artifact qualification**, not Provider selection authority, a current Host grant, or evidence of a commit. Read equivalence requires the extra cohort/bag/field-completeness conditions from #2627.
5. For multi-provider sources, [#2622](https://github.com/hat47x/tei-platform/issues/2622) handles API/RDB capability and read semantics; [#2506](https://github.com/hat47x/tei-platform/issues/2506) handles Function Data Context; [#2598](https://github.com/hat47x/tei-platform/issues/2598) owns actual Request binding. [#1119](https://github.com/hat47x/tei-platform/issues/1119) retains P0 J5/PG and Owner acceptance.

## Evidence boundaries

- Before CI, all seven actual implementation Go tests are **NOT RUN**, even if the independent #2627 Go tests passed. The code has been written against the exact upstream candidate. Report exact head and workflow status when confirmed.
- These tests construct an `ir.World` directly. They do **not** prove that the full Workspace Loader/Validator accepts or rejects the fixture. A follow-up must make that acceptance explicit before calling this a supported multiple-mapping configuration.
- Real database/replica/API Query, Field-level NULL, Oracle/Function Context, Provider mapping and actual Effect/Receipt, external Host/Studio/SHO, and meaning acceptance are out of scope.
