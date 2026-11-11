package partition

import (
	"testing"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/database"
	"tei/internal/database/persistence"
	"tei/internal/ir"
	"tei/internal/operations"
	"tei/internal/semantic"
)

// This is a characterization of an existing read-only derived surface, not a
// product requirement or an Accepted Function direct-write producer. Its
// Binding is one selected Capability, not an enumeration of business writers.
func TestCharacterizeOperationSurfaceBindingIsNotFunctionWriterClosure(t *testing.T) {
	world := ir.World{
		Semantics: []semantic.Namespace{{
			ID: "sales",
			Data: []semantic.Data{{
				ID: "sales.order",
				Attributes: []semantic.Attribute{{
					ID: "sales.order.order_id", Symbol: "orderId", TypeRef: "std.identifier",
				}},
			}},
			Capabilities: []semantic.Capability{
				{
					ID:     "sales.order.submit",
					Inputs: []semantic.Parameter{{Name: "order", Type: "sales.order"}},
					Effect: &semantic.DataEffect{Kind: semantic.DataEffectUpdate, Data: "sales.order"},
				},
				{
					ID:     "sales.order.approve",
					Inputs: []semantic.Parameter{{Name: "order", Type: "sales.order"}},
					Effect: &semantic.DataEffect{Kind: semantic.DataEffectUpdate, Data: "sales.order"},
				},
			},
		}},
		Applications: []application.Application{{
			ID: "order_entry",
			Patterns: []application.Pattern{{
				ID:         "order_entry.submit_order",
				Kind:       application.PatternCommand,
				Capability: "sales.order.submit",
			}},
			Databases: []application.DatabaseRequirement{{
				Alias: "sales", TargetDatabase: "sales_core", Access: database.AccessReadWrite,
			}},
		}},
		Persistence: []persistence.Mapping{{
			ID: "sales_core.order", Database: "sales_core", SemanticData: "sales.order",
			Table: "orders", PrimaryKey: []core.CanonicalID{"sales.order.order_id"},
			Columns: []persistence.ColumnMapping{{
				Attribute: "sales.order.order_id", Column: "order_id",
				Storage: persistence.StorageInt64, Nullability: persistence.Required,
			}},
		}},
	}

	// The two independent declared business-effect intentions survive in World.
	if len(world.Semantics[0].Capabilities) != 2 ||
		world.Semantics[0].Capabilities[0].Effect == nil ||
		world.Semantics[0].Capabilities[1].Effect == nil {
		t.Fatal("fixture must contain two distinct Capability effect declarations")
	}
	pattern := world.Applications[0].Patterns[0]
	if pattern.Capability != "sales.order.submit" {
		t.Fatal("fixed Pattern must refer to submit, not approve")
	}

	surface := operations.DeriveOperationSurface(world, "order_entry")
	var found []operations.Operation
	for _, op := range surface.Operations {
		if op.ID == "order_entry.update.sales.order" {
			found = append(found, op)
		}
	}
	if len(found) != 1 || found[0].Binding != "sales.order.approve" ||
		found[0].Binding == string(pattern.Capability) {
		t.Fatalf("current derived surface selects a representative, not the Pattern's Function I/O: %+v", found)
	}
	if len(surface.Conflicts) != 0 {
		t.Fatalf("surface now diagnoses ambiguous bindings; revisit characterization: %+v", surface.Conflicts)
	}

	// Input declaration order does not matter; the projection selects the
	// lexicographically first Capability with that (effect-kind, Data) pair.
	world.Semantics[0].Capabilities[0], world.Semantics[0].Capabilities[1] =
		world.Semantics[0].Capabilities[1], world.Semantics[0].Capabilities[0]
	reordered := operations.DeriveOperationSurface(world, "order_entry")
	for _, op := range reordered.Operations {
		if op.ID == found[0].ID && op.Binding != found[0].Binding {
			t.Fatalf("selected binding changed with declaration order: before=%+v after=%+v", found[0], op)
		}
	}

	// Removing the unreferenced approve Capability changes the one selected
	// binding while the derived Operation ID and command Pattern remain fixed.
	world.Semantics[0].Capabilities = []semantic.Capability{world.Semantics[0].Capabilities[1]}
	removed := operations.DeriveOperationSurface(world, "order_entry")
	var after *operations.Operation
	for _, op := range removed.Operations {
		if op.ID == found[0].ID {
			o := op
			after = &o
			break
		}
	}
	if after == nil || after.Binding != "sales.order.submit" || after.ID != found[0].ID {
		t.Fatalf("a non-Pattern Capability can affect the selected binding without changing the Operation ID: %+v", after)
	}

	// No direct C/U/D Write, subsystem Owner, or partition.Problem is inferred
	// from either the synthetic World or the derived OperationSurface here.
}
