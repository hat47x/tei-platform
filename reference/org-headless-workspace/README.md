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
