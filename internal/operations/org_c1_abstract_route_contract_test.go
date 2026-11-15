package operations

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"tei/internal/semantic"
	"tei/internal/workspace"
)

// This is a Source -> typed IR -> Query-projection witness only.
// A generated Go method name, an actual Provider read, the runtime call path,
// and an Owner-accepted Person link are deliberately NOT part of this test.
// It defines a stable, domain-level path for future generated/interpretive
// implementation conformance without adopting a new Org-specific runtime.
func orgC1StableUnitReadPath(rel semantic.Relation, queryID, datasource string, attributes []string) bool {
	if rel.ID != "organization.membership.unit" ||
		rel.Target != "organization.org_unit" ||
		rel.Cardinality != semantic.CardinalityOne ||
		len(rel.Keys) != 1 ||
		queryID != "org_axes_view.as_of_membership" ||
		datasource != "org_read" {
		return false
	}
	// Check the *typed key* rather than matching Relation.Symbol, generated
	// method name, SQL table alias, or display label. JSON is used here only
	// to inspect the IR's field IDs without introducing a new Key schema.
	keys, err := json.Marshal(rel.Keys)
	if err != nil ||
		!strings.Contains(string(keys), "organization.membership.org_unit_id") ||
		!strings.Contains(string(keys), "organization.org_unit.org_unit_id") {
		return false
	}
	return reflect.DeepEqual(attributes, []string{
		"organization.membership.membership_id",
		"organization.membership.org_unit_id",
		"organization.membership.allocation",
	})
}

func TestOrgC1AbstractUnitReadPathUsesStableSemanticIDs(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "org-staged-axis")
	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatalf("load actual .tei organization Workspace: %v", err)
	}

	var unit semantic.Relation
	membershipFound, relationFound, queryFound := false, false, false
	personPathCount := 0
	for _, namespace := range loaded.World.Semantics {
		for _, data := range namespace.Data {
			if data.ID != "organization.membership" {
				continue
			}
			membershipFound = true
			for _, rel := range data.Relations {
				if rel.Target == "organization.person" {
					personPathCount++
				}
				if rel.ID == "organization.membership.unit" {
					unit = rel
					relationFound = true
				}
			}
		}
	}
	if !membershipFound || !relationFound || personPathCount != 0 {
		t.Fatalf("actual Membership path changed: membership=%v unit=%v personPaths=%d", membershipFound, relationFound, personPathCount)
	}

	for _, query := range loaded.World.Queries {
		if query.ID != "org_axes_view.as_of_membership" {
			continue
		}
		queryFound = true
		var selected []string
		for _, column := range query.ResultColumns {
			selected = append(selected, string(column.Attribute))
		}
		if !orgC1StableUnitReadPath(unit, string(query.ID), string(query.Datasource), selected) {
			t.Fatalf("actual typed Relation and Query projection no longer describe one accepted unit-read path: relation=%+v query=%s sourceColumns=%v",
				unit, query.ID, selected)
		}

		// A UI/compiler target may rename a visible Relation symbol without
		// changing the qualified identity, cardinality, key, or Query contract.
		renamed := unit
		renamed.Symbol = "unitPresentationRenamed"
		if !orgC1StableUnitReadPath(renamed, string(query.ID), string(query.Datasource), selected) {
			t.Fatal("a surface-only symbol rename changed the abstract path")
		}

		wrongTarget := unit
		wrongTarget.Target = "organization.person"
		if orgC1StableUnitReadPath(wrongTarget, string(query.ID), string(query.Datasource), selected) {
			t.Fatal("swapping OrgUnit for Person must not preserve this typed path")
		}
		missingJoin := unit
		missingJoin.Keys = nil
		if orgC1StableUnitReadPath(missingJoin, string(query.ID), string(query.Datasource), selected) {
			t.Fatal("dropping the explicit Relation key must invalidate the path")
		}
		if orgC1StableUnitReadPath(unit, "org_axes_view.other_query", string(query.Datasource), selected) {
			t.Fatal("unrelated Query IDs must not inherit this path's evidence")
		}
		if orgC1StableUnitReadPath(unit, string(query.ID), "other_provider", selected) {
			t.Fatal("changing datasource alias requires a separate read qualification")
		}
		if orgC1StableUnitReadPath(unit, string(query.ID), string(query.Datasource), selected[:2]) {
			t.Fatal("dropping the allocation projection must invalidate this specific read contract")
		}
	}
	if !queryFound {
		t.Fatal("actual as-of Query is absent; no path witness can be asserted")
	}

	// A valid Membership -> OrgUnit path does not imply Membership -> Person,
	// a complete Read Cohort, a BRMS candidate, or an executable Host Grant.
}
