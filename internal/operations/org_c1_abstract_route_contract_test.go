package operations

import (
	"path/filepath"
	"reflect"
	"testing"

	"tei/internal/query"
	"tei/internal/semantic"
	"tei/internal/workspace"
)

// This tests a Source -> typed IR -> Query-interface contract, not execution.
// The business path is identified by semantic IDs, explicit join keys and
// named inputs/outputs, never by a generated Go function or UI label.
func orgC1StableUnitReadPath(rel semantic.Relation, q query.EmbeddedSQL) bool {
	if rel.ID != "organization.membership.unit" ||
		rel.Target != "organization.org_unit" ||
		rel.Cardinality != semantic.CardinalityOne ||
		rel.JoinTable != "" ||
		len(rel.Keys) != 1 ||
		rel.Keys[0].SourceAttribute != "organization.membership.org_unit_id" ||
		rel.Keys[0].TargetAttribute != "organization.org_unit.org_unit_id" ||
		q.ID != "org_axes_view.as_of_membership" ||
		q.Datasource != "org_read" {
		return false
	}
	// Parameters are named: rearranging their authoring order must not change
	// the contract, but losing a non-output dependency must invalidate it.
	actualInputs := make(map[string]string, len(q.Inputs))
	for _, in := range q.Inputs {
		name := string(in.Name)
		if _, duplicate := actualInputs[name]; duplicate {
			return false
		}
		actualInputs[name] = string(in.Type)
	}
	if !reflect.DeepEqual(actualInputs, map[string]string{
		"valid_at":       "std.datetime",
		"known_at":       "std.datetime",
		"allocation_min": "std.decimal",
	}) {
		return false
	}
	// The raw column labels and semantic attributes must both match. This
	// does NOT inspect the actual SELECT/WHERE predicates or run the Provider.
	actualResults := make(map[string]string, len(q.ResultColumns))
	for _, col := range q.ResultColumns {
		if _, duplicate := actualResults[col.Column]; duplicate {
			return false
		}
		actualResults[col.Column] = string(col.Attribute)
	}
	return reflect.DeepEqual(actualResults, map[string]string{
		"membership_id": "organization.membership.membership_id",
		"org_unit_id":   "organization.membership.org_unit_id",
		"allocation":    "organization.membership.allocation",
	})
}

func TestOrgC1AbstractUnitReadPathUsesStableSemanticIDs(t *testing.T) {
	loaded, err := workspace.Load(filepath.Join("..", "..", "reference", "org-staged-axis"))
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
		t.Fatalf("actual Membership changed: membership=%v unit=%v personPaths=%d", membershipFound, relationFound, personPathCount)
	}

	for _, q := range loaded.World.Queries {
		if q.ID != "org_axes_view.as_of_membership" {
			continue
		}
		queryFound = true
		if !orgC1StableUnitReadPath(unit, q) {
			t.Fatalf("real Source does not satisfy typed Relation and Query interface: relation=%+v query=%+v", unit, q)
		}
		t.Run("presentation rename preserves semantic route", func(t *testing.T) {
			renamed := unit
			renamed.Symbol = "unitPresentationRenamed"
			if !orgC1StableUnitReadPath(renamed, q) {
				t.Fatal("display symbol rename changed the semantic route")
			}
		})
		t.Run("input declaration order does not change named bindings", func(t *testing.T) {
			reordered := q
			reordered.Inputs = append([]query.Input(nil), q.Inputs...)
			reordered.Inputs[0], reordered.Inputs[1] = reordered.Inputs[1], reordered.Inputs[0]
			if !orgC1StableUnitReadPath(unit, reordered) {
				t.Fatal("named parameter reordering changed the route")
			}
		})
		t.Run("result declaration order does not change named mappings", func(t *testing.T) {
			reordered := q
			reordered.ResultColumns = append([]query.ResultColumnMapping(nil), q.ResultColumns...)
			reordered.ResultColumns[0], reordered.ResultColumns[1] = reordered.ResultColumns[1], reordered.ResultColumns[0]
			if !orgC1StableUnitReadPath(unit, reordered) {
				t.Fatal("named result column reordering changed the route")
			}
		})
		for _, tc := range []struct {
			name  string
			rel   semantic.Relation
			query query.EmbeddedSQL
		}{
			{
				name: "substitute Person as Relation target",
				rel: func() semantic.Relation {
					r := unit
					r.Target = "organization.person"
					return r
				}(),
				query: q,
			},
			{
				name: "reverse Relation key direction",
				rel: func() semantic.Relation {
					r := unit
					r.Keys = append([]semantic.RelationKey(nil), unit.Keys...)
					r.Keys[0].SourceAttribute, r.Keys[0].TargetAttribute =
						r.Keys[0].TargetAttribute, r.Keys[0].SourceAttribute
					return r
				}(),
				query: q,
			},
			{
				name: "remove Relation key",
				rel: func() semantic.Relation {
					r := unit
					r.Keys = nil
					return r
				}(),
				query: q,
			},
			{
				name: "change Relation to join table",
				rel: func() semantic.Relation {
					r := unit
					r.JoinTable = "unexpected_join"
					return r
				}(),
				query: q,
			},
			{
				name: "change Query identity",
				rel: unit,
				query: func() query.EmbeddedSQL {
					changed := q
					changed.ID = "org_axes_view.other_query"
					return changed
				}(),
			},
			{
				name: "change datasource",
				rel: unit,
				query: func() query.EmbeddedSQL {
					changed := q
					changed.Datasource = "other_provider"
					return changed
				}(),
			},
			{
				name: "remove non-output allocation filter parameter",
				rel: unit,
				query: func() query.EmbeddedSQL {
					changed := q
					changed.Inputs = append([]query.Input(nil), q.Inputs[:2]...)
					return changed
				}(),
			},
			{
				name: "change knownAt input type",
				rel: unit,
				query: func() query.EmbeddedSQL {
					changed := q
					changed.Inputs = append([]query.Input(nil), q.Inputs...)
					changed.Inputs[1].Type = "std.decimal"
					return changed
				}(),
			},
			{
				name: "drop allocation result",
				rel: unit,
				query: func() query.EmbeddedSQL {
					changed := q
					changed.ResultColumns = append([]query.ResultColumnMapping(nil), q.ResultColumns[:2]...)
					return changed
				}(),
			},
			{
				name: "map allocation to wrong semantic attribute",
				rel: unit,
				query: func() query.EmbeddedSQL {
					changed := q
					changed.ResultColumns = append([]query.ResultColumnMapping(nil), q.ResultColumns...)
					changed.ResultColumns[2].Attribute = "organization.membership.org_unit_id"
					return changed
				}(),
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				if orgC1StableUnitReadPath(tc.rel, tc.query) {
					t.Fatal("semantic route changed but contract remained true")
				}
			})
		}
	}
	if !queryFound {
		t.Fatal("actual as-of Query absent; no semantic route witness can be asserted")
	}
	// Even a passed Source/IR contract does not show a runtime join,
	// completed Read Cohort, Person identity, Rule decision or Host Grant.
}
