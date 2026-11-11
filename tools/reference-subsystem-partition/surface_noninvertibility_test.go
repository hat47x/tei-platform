package partition

import (
	"reflect"
	"testing"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/database"
	"tei/internal/database/persistence"
	"tei/internal/ir"
	"tei/internal/operations"
	"tei/internal/semantic"
)

// This is a characterization of a lossy read-only projection. The two valid
// candidate Worlds differ in declared Capability effect intent but produce
// identical derived OperationSurfaces. No inverse can recover *all* Capability
// declarations (still less Accepted Function direct C/U/D or actual writes).
func TestOperationSurfaceCannotRecoverUnselectedCapabilityIntent(t *testing.T) {
	const data = core.CanonicalID("sales.order")
	declaredUpdate := func(id core.CanonicalID) semantic.Capability {
		return semantic.Capability{
			ID:     id,
			Inputs: []semantic.Parameter{{Name: "order", Type: data}},
			Effect: &semantic.DataEffect{Kind: semantic.DataEffectUpdate, Data: data},
		}
	}
	world := ir.World{
		Semantics: []semantic.Namespace{{
			ID: "sales",
			Data: []semantic.Data{{
				ID: data,
				Attributes: []semantic.Attribute{{
					ID: "sales.order.order_id", Symbol: "orderId", TypeRef: "std.identifier",
				}},
			}},
			Capabilities: []semantic.Capability{declaredUpdate("sales.order.approve")},
		}},
		Applications: []application.Application{{
			ID: "order_entry",
			Patterns: []application.Pattern{{
				ID: "order_entry.approve_order", Kind: application.PatternCommand,
				Capability: "sales.order.approve",
			}},
			Databases: []application.DatabaseRequirement{{
				Alias: "sales", TargetDatabase: "sales_core", Access: database.AccessReadWrite,
			}},
		}},
		Persistence: []persistence.Mapping{{
			ID: "sales_core.order", Database: "sales_core", SemanticData: data,
			Table: "orders", PrimaryKey: []core.CanonicalID{"sales.order.order_id"},
			Columns: []persistence.ColumnMapping{{
				Attribute: "sales.order.order_id", Column: "order_id",
				Storage: persistence.StorageInt64, Nullability: persistence.Required,
			}},
		}},
	}

	a := operations.DeriveOperationSurface(world, "order_entry")
	if len(a.Operations) == 0 {
		t.Fatal("positive control: first World must derive operations")
	}
	found := false
	for _, operation := range a.Operations {
		if operation.ID == "order_entry.update.sales.order" {
			found = true
			if operation.Binding != "sales.order.approve" {
				t.Fatalf("unexpected selected Capability: %+v", operation)
			}
		}
	}
	if !found {
		t.Fatal("positive control: update Operation not derived")
	}

	// Add another distinct, unreferenced Capability effect. Neither source
	// content nor approved function Directness is invented by the projection.
	// The Pattern and the canonical operation ID remain unchanged.
	world.Semantics[0].Capabilities = append(
		world.Semantics[0].Capabilities, declaredUpdate("sales.order.zmanual"),
	)
	if len(world.Semantics[0].Capabilities) != 2 {
		t.Fatal("positive control: there must be two distinct intent declarations")
	}
	b := operations.DeriveOperationSurface(world, "order_entry")
	if !reflect.DeepEqual(a, b) {
		t.Fatalf("characterization changed: an unselected Capability now appears in the Surface; revisit the inverse-closure claim: before=%+v after=%+v", a, b)
	}

	// DeriveOperationSurface is a pure in-memory projection; unlike the
	// Workspace entry point, it does not bind Source or World fingerprints.
	// Even identical public Surface bytes are no accepted Owner/Write witness.
	if a.WorldFingerprint != "" || b.WorldFingerprint != "" {
		t.Fatal("unexpected Source-qualified fingerprint in pure derived Surface")
	}
}
