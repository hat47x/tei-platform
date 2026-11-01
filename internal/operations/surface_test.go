package operations

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/database"
	"tei/internal/database/persistence"
	"tei/internal/ir"
	"tei/internal/semantic"
)

func surfaceTestWorld(access database.Access, candidateKey bool) ir.World {
	element := semantic.Data{
		ID: "sales.order",
		Attributes: []semantic.Attribute{
			{ID: "sales.order.order_id", Symbol: "orderId", TypeRef: "std.identifier"},
			{ID: "sales.order.status", Symbol: "status", TypeRef: "std.code"},
		},
	}
	if candidateKey {
		element.CandidateKeys = []semantic.CandidateKey{{
			ID:         "sales.order.business_key",
			Symbol:     "businessKey",
			Attributes: []core.CanonicalID{"sales.order.order_id"},
		}}
	}
	return ir.World{
		Semantics: []semantic.Namespace{{
			ID:   "sales",
			Data: []semantic.Data{element},
		}},
		Applications: []application.Application{{
			ID: "order_entry",
			Databases: []application.DatabaseRequirement{{
				Alias:          "sales",
				TargetDatabase: "sales_core",
				Access:         access,
			}},
		}},
		Persistence: []persistence.Mapping{{
			ID:           "sales_core.order",
			Database:     "sales_core",
			SemanticData: "sales.order",
			Table:        "orders",
			PrimaryKey:   []core.CanonicalID{"sales.order.order_id"},
			Columns: []persistence.ColumnMapping{{
				Attribute:   "sales.order.order_id",
				Column:      "order_id",
				Storage:     persistence.StorageInt64,
				Nullability: persistence.Required,
			}},
		}},
	}
}

func findOperation(surface OperationSurface, id string) (Operation, bool) {
	for _, operation := range surface.Operations {
		if operation.ID == id {
			return operation, true
		}
	}
	return Operation{}, false
}

func TestDeriveOperationSurfaceListAndRead(t *testing.T) {
	surface := DeriveOperationSurface(surfaceTestWorld(database.AccessReadWrite, true), "")

	if surface.SchemaVersion != OperationSurfaceSchemaVersion || surface.Effect != "read-only" {
		t.Fatalf("unexpected envelope: %+v", surface)
	}
	if surface.Rule.ID != operationSurfaceRuleID || surface.Rule.Version != operationSurfaceRuleVersion {
		t.Fatalf("unexpected rule: %+v", surface.Rule)
	}
	if surface.Rule.Fingerprint == "" || surface.WorldFingerprint != "" {
		t.Fatalf("rule fingerprint must be set and world fingerprint left to the workspace entry: %+v", surface)
	}

	list, ok := findOperation(surface, "order_entry.list.sales.order")
	if !ok || list.Kind != "list" || list.Origin != "derived" || list.Autonomy != "unattended" {
		t.Fatalf("list operation missing or wrong: %+v", surface.Operations)
	}
	if len(list.Inputs) != 0 || len(list.Effects) != 0 {
		t.Fatalf("list must not have inputs or effects: %+v", list)
	}

	read, ok := findOperation(surface, "order_entry.read.sales.order")
	if !ok || read.Kind != "read" {
		t.Fatalf("read operation missing: %+v", surface.Operations)
	}
	if len(read.Inputs) != 1 || read.Inputs[0] != "sales.order.order_id" {
		t.Fatalf("read must require the business key: %+v", read.Inputs)
	}
	if read.Basis == "" || list.Basis == "" {
		t.Fatalf("derived operations must carry a basis: %+v", surface.Operations)
	}

	if len(surface.Gaps) != 0 {
		t.Fatalf("unexpected gaps: %+v", surface.Gaps)
	}
	if surface.Summary.Operations != 2 || surface.Summary.Derived != 2 ||
		surface.Summary.Unattended != 2 || surface.Summary.Gaps != 0 {
		t.Fatalf("unexpected summary: %+v", surface.Summary)
	}
	if len(surface.AuthorityLimits) == 0 {
		t.Fatal("authority limits must be present so consumers do not read the surface as authority")
	}
}

func TestDeriveOperationSurfaceFallsBackToPrimaryKey(t *testing.T) {
	surface := DeriveOperationSurface(surfaceTestWorld(database.AccessReadWrite, false), "")

	read, ok := findOperation(surface, "order_entry.read.sales.order")
	if !ok {
		t.Fatalf("read operation missing: %+v", surface.Operations)
	}
	if len(read.Inputs) != 1 || read.Inputs[0] != "sales.order.order_id" {
		t.Fatalf("read must fall back to the persistence primary key: %+v", read.Inputs)
	}
}

func TestDeriveOperationSurfaceMissingKeyIsAGap(t *testing.T) {
	world := surfaceTestWorld(database.AccessReadWrite, false)
	world.Persistence[0].PrimaryKey = nil

	surface := DeriveOperationSurface(world, "")

	if _, ok := findOperation(surface, "order_entry.read.sales.order"); ok {
		t.Fatal("read must not be derived without an identifying key")
	}
	if _, ok := findOperation(surface, "order_entry.list.sales.order"); !ok {
		t.Fatal("list must still be derived")
	}
	if len(surface.Gaps) != 1 {
		t.Fatalf("expected one gap: %+v", surface.Gaps)
	}
	gap := surface.Gaps[0]
	if gap.Target != "sales.order" || gap.Kind != "read" || gap.Reason != "missing-semantics" || gap.Detail == "" {
		t.Fatalf("unexpected gap: %+v", gap)
	}
}

func TestDeriveOperationSurfaceWriteOnlyAlias(t *testing.T) {
	surface := DeriveOperationSurface(surfaceTestWorld(database.AccessWrite, true), "")

	if len(surface.Operations) != 0 {
		t.Fatalf("write-only alias must not derive read operations: %+v", surface.Operations)
	}
	if len(surface.Gaps) != 1 || surface.Gaps[0].Reason != "conflict" || surface.Gaps[0].Kind != "list" {
		t.Fatalf("expected a list conflict gap: %+v", surface.Gaps)
	}
}

func TestDeriveOperationSurfaceReadOnlyAlias(t *testing.T) {
	surface := DeriveOperationSurface(surfaceTestWorld(database.AccessRead, true), "")

	if surface.Summary.Operations != 2 {
		t.Fatalf("read-only alias must still derive list and read: %+v", surface.Summary)
	}
}

func TestDeriveOperationSurfaceIsDeterministic(t *testing.T) {
	world := surfaceTestWorld(database.AccessReadWrite, true)
	world.Semantics[0].Data = append(world.Semantics[0].Data,
		semantic.Data{ID: "sales.customer"},
		semantic.Data{ID: "sales.invoice"},
	)
	world.Persistence = append(world.Persistence,
		persistence.Mapping{
			ID: "sales_core.customer", Database: "sales_core", SemanticData: "sales.customer",
			Table: "customers", PrimaryKey: []core.CanonicalID{"sales.customer.customer_id"},
		},
		persistence.Mapping{
			ID: "sales_core.invoice", Database: "sales_core", SemanticData: "sales.invoice",
			Table: "invoices", PrimaryKey: []core.CanonicalID{"sales.invoice.invoice_id"},
		},
	)

	first, err := json.Marshal(DeriveOperationSurface(world, ""))
	if err != nil {
		t.Fatalf("marshal surface: %v", err)
	}

	// Reverse every input order. Sorting inside the derivation must absorb it.
	for i, j := 0, len(world.Semantics[0].Data)-1; i < j; i, j = i+1, j-1 {
		world.Semantics[0].Data[i], world.Semantics[0].Data[j] = world.Semantics[0].Data[j], world.Semantics[0].Data[i]
	}
	for i, j := 0, len(world.Persistence)-1; i < j; i, j = i+1, j-1 {
		world.Persistence[i], world.Persistence[j] = world.Persistence[j], world.Persistence[i]
	}
	second, err := json.Marshal(DeriveOperationSurface(world, ""))
	if err != nil {
		t.Fatalf("marshal reversed surface: %v", err)
	}
	if string(first) != string(second) {
		t.Fatalf("operation surface depends on input order:\n%s\n%s", first, second)
	}
}

func TestDeriveOperationSurfaceApplicationFilter(t *testing.T) {
	world := surfaceTestWorld(database.AccessReadWrite, true)
	world.Applications = append(world.Applications, application.Application{
		ID: "repair_intake",
		Databases: []application.DatabaseRequirement{{
			Alias: "sales", TargetDatabase: "sales_core", Access: database.AccessReadWrite,
		}},
	})

	all := DeriveOperationSurface(world, "")
	if all.Summary.Operations != 4 {
		t.Fatalf("expected four operations for two applications: %+v", all.Summary)
	}
	filtered := DeriveOperationSurface(world, "repair_intake")
	if filtered.Summary.Operations != 2 || filtered.Application != "repair_intake" {
		t.Fatalf("filter must keep only the requested application: %+v", filtered)
	}
	if _, ok := findOperation(filtered, "repair_intake.list.sales.order"); !ok {
		t.Fatalf("filtered surface lost its operations: %+v", filtered.Operations)
	}
}

// TestSurfaceWorkspaceReference derives the surface of the reference workspace.
// It is skipped only when that workspace is not present in this checkout.
func TestSurfaceWorkspaceReference(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "workspace")
	if _, err := os.Stat(root); err != nil {
		t.Skipf("reference workspace is not available: %v", err)
	}

	surface, err := SurfaceWorkspace(root, "order_entry")
	if err != nil {
		t.Fatalf("derive reference surface: %v", err)
	}
	if surface.WorldFingerprint == "" || surface.ModelRevision == "" {
		t.Fatalf("workspace surface must bind the world and model revision: %+v", surface)
	}
	if _, ok := findOperation(surface, "order_entry.list.sales.order"); !ok {
		t.Fatalf("reference workspace lost the derived list: %+v", surface.Operations)
	}
	read, ok := findOperation(surface, "order_entry.read.sales.order")
	if !ok {
		t.Fatalf("reference workspace lost the derived read: %+v", surface.Operations)
	}
	if len(read.Inputs) == 0 {
		t.Fatalf("derived read must carry identifying inputs: %+v", read)
	}
}
