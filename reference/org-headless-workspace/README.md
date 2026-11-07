# TEI native authoring reference: organization relations

This is a **native .tei workspace**, not the JSON-only synthetic organization oracle. It uses the existing TEI parser, World loader, standard Data/Relation structure, persistence mapping, and derived read-only operation surface.

- Separate Person, OrgUnit, Membership (an independently identified relation occurrence), Position and Occupancy Data.
- Additional **CaseRecord** Data with a typed `CaseRecord → OrgUnit` handling-unit Relation. The actual TEI loader/test verifies that this existing structural link does **not** imply record visibility, recursive policy application, an authorization grant, or even a derived Case read operation without persistence/Application binding.
- Explicit Membership → Person / OrgUnit relations. Separate Occupancy → Position / Person relations. **Occupancy never implies Membership without an accepted bridging rule.**
- `validFrom` and `knownFrom` are ordinary authored attributes. Their presence alone does **not** implement bitemporal meaning, history migration, as-of evaluation or source acceptance.
- The only persistence mapping is Membership; `org_view` is read-only. It should expose `org_view.list.organization.membership` and `org_view.read.organization.membership`. The latter must key on `organization.membership.membership_id`, not on Person ID or display row order.
- No default create/update/unlink/approval, tenant grant, successor bridge or delegated execution is implied.

Check with the real CLI:

```sh
go run ./cmd/tei validate -workspace reference/org-headless-workspace
go run ./cmd/tei operations -workspace reference/org-headless-workspace -application org_view
go run ./cmd/tei artifacts -workspace reference/org-headless-workspace -application org_view
go test ./internal/operations -run TestOrgHeadless -count=1
```

**Scope:** This tests real Parser/Loader → Semantic World/Relation → Persistence mapping → OperationSurface, with artifact output as an additional consumer. `TestOrgHeadlessCaseRelationDoesNotAuthorizeReadOrWrite` is a native Go *negative characterization* of the typed relationship/policy boundary. It proves the absence of automatically derived Case operations in this read-only fixture, not that a default scoped-access policy could be enforced by a real Provider. It does NOT insert Persons/A1–A4 into a real DB, execute SQLGlot, observe Provider reads, calculate FTE/temporal correction, render Studio or SHO, authorize a Host action, or establish Owner business acceptance. Synthetic ORG-OC/RC/VF/FO remains a separate oracle in `reference/org-semantic-views`.

Handoff: #2386, #2586, #1127, #2506, #2497, #2628, #2598, #1174; design Draft #2389. Reuse standard language/persistence/operations rather than introducing another organization Core or specialized Host.

## Source, meaning and derived operations are different revision scopes

These native Go regression tests work with **copies of the actual .tei workspace** and the production Parser/Loader/Source Index/Operation Surface, not a separate synthetic Source model.

| Characterization | Source revision | Normalized World | Operation set | What it cannot establish |
| --- | --- | --- | --- | --- |
| Comment-only source edit | changed | unchanged | unchanged | Current source disclosure permission or real Provider/Host qualification |
| Move `semantic/organization/model.tei` to another filename under same managed root | changed, despite identical file bytes | unchanged | unchanged | Validity of old source URL/span/citation, import snapshot or consumer selection |
| A→B→A (add comment, then restore original bytes) | A ≠ B; A = C | unchanged | unchanged | That no intermediate revision or revocation ever happened |
| CaseRecord `handlingUnit` cardinality `one → many` | changed | changed | unchanged | That same Operation JSON implies the same accepted meaning, Rule or Disclosure |

The independent expected relationships in the table are asserted by `TestOrgHeadlessSourceByteEditPreservesWorldButChangesModelRevision`, `TestOrgHeadlessSourcePathMoveChangesModelRevision`, `TestOrgHeadlessSourceABAReturnsToOriginalStateButNotHistory`, and `TestOrgHeadlessSemanticRelationChangeIsInvisibleToCurrentOperations`.

**A→B→A caveat:** The source identity `ModelRevision` is a fingerprint of the *currently observed state*, not an immutable event sequence, monotonic epoch, Provider Read receipt or authorization Grant. An external Consumer that missed B cannot infer B from A and C even when they share the same Source fingerprint. Observe the lifecycle and current permission separately.

**Same Operation caveat:** A Function using the CaseRecord handling-unit relationship can depend on its declared cardinality even though the read-only Membership Operation Surface does not mention CaseRecord. Caching or skipping work solely because operation IDs or output JSON match is unsound for such a Consumer.

The test runner's fixed-head status and product limitations must be checked separately. These checks do not establish real Provider execution, currently authorized policy enforcement, atomic multi-file Source capture or Owner's meaning acceptance.

## Unselected plugin Source is not an included build component

The real Go regression `TestOrgHeadlessUndeclaredPluginRootDoesNotChangeSemanticOrOperations` copies this Workspace, creates `plugins/unused-org-profile/plugin.tei` containing intentionally invalid authoring, and executes the **actual Loader/Source Index/OperationSurface** without adding `plugins` to the manifest's declared Source roots. Its independent expected result is:

- The unselected `plugins/` file is absent from `Loaded.ManagedFiles`; the Loader does **not** parse it or promote it to available capabilities.
- `WorldFingerprint`, `ModelRevision`, and the read-only Operation set are unchanged.
- The deliberately invalid source does not cause a Parser error, because it was not requested.

This **does not prove** that a valid plugin present in a *declared* Source root but not selected by the Environment is absent from the global Source Index; that a selected Bundle's unused module is excluded from the Build; that a Go compiled binary has removed unused code; or that any compiled-in action is exposed, currently granted or actually executed. `plugin.Catalog.ResolveEnvironment` currently expands **every plugin requirement of a selected Bundle**; Function/Target use, Build inclusion and runtime permission must be separately measured.

For those still-unverified cases, see [Organization design Draft #2389 §64](https://github.com/hat47x/tei-platform/pull/2389) and [Everything is Data Draft #2667 §51](https://github.com/hat47x/tei-platform/pull/2667). The Org E1 CI's `artifacts/manifest` reader reports `findings=0 complete=False` and cannot be taken as proof of distributed Artifact completeness.