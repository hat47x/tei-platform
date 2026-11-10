package validation

import (
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/database"
	"tei/internal/ir"
	"tei/internal/semantic"
)

// The second batch of closed vocabularies (#4094). Each diagnostic must name
// the values the validator accepts, so an author can fix the declaration
// without reading the source.
func TestMoreVocabularyDiagnosticsNameTheirValues(t *testing.T) {
	cases := []struct {
		name   string
		world  ir.World
		code   string
		values []string
	}{
		{
			name: "database ownership",
			world: ir.World{
				Databases: []database.Database{{
					ID: "sales_core", Kind: "relational", Ownership: database.Ownership("managed-elsewhere"),
				}},
			},
			code:   "INVALID_DATABASE_OWNERSHIP",
			values: []string{"managed-here", "external-observed"},
		},
		{
			name: "relation cardinality",
			world: ir.World{
				Semantics: []semantic.Namespace{{
					ID: core.CanonicalID("security"),
					Data: []semantic.Data{{
						ID: core.CanonicalID("security.probe"),
						Relations: []semantic.Relation{{
							ID: core.CanonicalID("security.probe.b"), Symbol: "b",
							Target: core.CanonicalID("sales.order"), Cardinality: semantic.Cardinality("several"),
						}},
					}},
				}},
			},
			code:   "INVALID_SEMANTIC_RELATION_CARDINALITY",
			values: []string{"one", "many"},
		},
	}
	for _, c := range cases {
		message := diagnosticMessage(t, c.world, c.code)
		for _, value := range c.values {
			if !strings.Contains(message, value) {
				t.Errorf("%s: message does not name %q: %s", c.name, value, message)
			}
		}
		if !strings.Contains(message, "valid values are") {
			t.Errorf("%s: message does not introduce the list: %s", c.name, message)
		}
	}
}
