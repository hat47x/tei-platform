package validation

import (
	"strings"
	"testing"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/database"
	"tei/internal/database/persistence"
	"tei/internal/ir"
)

// diagnosticMessage returns the message of the first diagnostic with the code.
func diagnosticMessage(t *testing.T, world ir.World, code string) string {
	t.Helper()
	for _, d := range Validate(world) {
		if string(d.Code) == code {
			return d.Message
		}
	}
	t.Fatalf("missing diagnostic %s", code)
	return ""
}

// An author who writes an unknown storage type, nullability or access mode is
// told only "invalid" unless the message names the alternatives. Each closed
// vocabulary now lists itself, and this pins that the list in the message is the
// list the validator accepts (#2094 round 54).
func TestClosedVocabularyDiagnosticsNameTheirValues(t *testing.T) {
	cases := []struct {
		name   string
		world  ir.World
		code   string
		values []string
	}{
		{
			name: "access",
			world: ir.World{
				Applications: []application.Application{{
					ID: "order_entry",
					Databases: []application.DatabaseRequirement{{
						Alias:  core.LocalName("sales"),
						Access: database.Access("nothing"),
					}},
				}},
			},
			code:   "INVALID_DATABASE_ACCESS",
			values: []string{"read", "write", "read-write"},
		},
		{
			name: "storage type",
			world: ir.World{
				Databases: []database.Database{{ID: "sales_core", Kind: "relational"}},
				Persistence: []persistence.Mapping{{
					ID:           "sales_core.order",
					Database:     "sales_core",
					SemanticData: "sales.order",
					Columns: []persistence.ColumnMapping{{
						Attribute: "sales.order.order_id", Column: "order_id",
						Storage: persistence.StorageType("NUMBERISH"), Nullability: persistence.Required,
					}},
				}},
			},
			code:   "INVALID_PERSISTENCE_STORAGE_TYPE",
			values: []string{"int64", "varchar", "text", "datetime-tz", "decimal", "json"},
		},
		{
			name: "nullability",
			world: ir.World{
				Databases: []database.Database{{ID: "sales_core", Kind: "relational"}},
				Persistence: []persistence.Mapping{{
					ID:           "sales_core.order",
					Database:     "sales_core",
					SemanticData: "sales.order",
					Columns: []persistence.ColumnMapping{{
						Attribute: "sales.order.order_id", Column: "order_id",
						Storage: persistence.StorageInt64, Nullability: persistence.Nullability("maybe"),
					}},
				}},
			},
			code:   "INVALID_PERSISTENCE_NULLABILITY",
			values: []string{"required", "optional"},
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
