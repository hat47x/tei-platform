package operations

import (
	"reflect"
	"testing"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/database"
	"tei/internal/database/persistence"
	"tei/internal/ir"
)

// This file characterizes the exact PR #2587 candidate behavior. The cases
// are not normative acceptance for silently dropping alternative providers.
// They are a read-only pressure test for #2611 and #2622, pending Owner review.
func multipleMappingWorld() ir.World {
	w := surfaceTestWorld(database.AccessReadWrite, true)
	w.Applications[0].Databases = []application.DatabaseRequirement{
		{Alias: "archive", TargetDatabase: "archive_db", Access: database.AccessReadWrite},
		{Alias: "primary", TargetDatabase: "sales_core", Access: database.AccessReadWrite},
	}
	w.Persistence = append(w.Persistence, persistence.Mapping{
		ID:           "archive_db.order",
		Database:     "archive_db",
		SemanticData: "sales.order",
		Table:        "legacy_orders",
		PrimaryKey:   []core.CanonicalID{"sales.order.order_id"},
		// Archive exposes fewer columns than Primary.
		Columns: []persistence.ColumnMapping{{
			Attribute:   "sales.order.order_id",
			Column:      "order_id",
			Storage:     persistence.StorageInt64,
			Nullability: persistence.Required,
		}},
	})
	return w
}

func TestCharacterizeTwoMappingsSameSemanticOperationID(t *testing.T) {
	w := multipleMappingWorld()
	s := DeriveOperationSurface(w, "order_entry")
	op, ok := findOperation(s, "order_entry.create.sales.order")
	if !ok {
		t.Fatal("create operation not derived")
	}
	if s.Summary.Operations != 4 {
		t.Fatalf("candidate surface has unexpected operation count: %+v", s.Summary)
	}
	// Current implementation retains the lexically first alias. This proves
	// what the candidate does, not that archive should own create.
	if op.Basis != mappingBasis(w.Persistence[1]) || len(op.Inputs) != 1 {
		t.Fatalf("candidate's first-alias binding changed; review expected route: %+v", op)
	}
	if op.Basis == mappingBasis(w.Persistence[0]) {
		t.Fatal("distinct archive/primary mappings unexpectedly share basis")
	}
	if len(s.Conflicts) != 0 {
		t.Fatalf("multiple-mapping conflicts are now emitted; update characterization: %+v", s.Conflicts)
	}
	t.Log("CHARACTERIZATION: same Operation ID selects archive mapping; primary alternative has no corresponding conflict")
}

func TestCharacterizeAliasRenameChangesChosenMappingWithoutOperationIDChange(t *testing.T) {
	w := multipleMappingWorld()
	a := DeriveOperationSurface(w, "order_entry")
	before, ok := findOperation(a, "order_entry.create.sales.order")
	if !ok {
		t.Fatal("before create missing")
	}

	// Change only the alias names: the DB, Mapping, Semantic Data and access
	// rights are unchanged, yet alias sorting changes the first candidate.
	w.Applications[0].Databases[0].Alias = "z_archive"
	w.Applications[0].Databases[1].Alias = "a_primary"
	b := DeriveOperationSurface(w, "order_entry")
	after, ok := findOperation(b, "order_entry.create.sales.order")
	if !ok {
		t.Fatal("after create missing")
	}
	if before.ID != after.ID || before.Basis == after.Basis {
		t.Fatalf("expected stable ID but different selected basis: before=%+v after=%+v", before, after)
	}
	if len(before.Inputs) != 1 || len(after.Inputs) != 2 {
		t.Fatalf("alias change should expose differing input lists: before=%+v after=%+v", before.Inputs, after.Inputs)
	}
	t.Log("CHARACTERIZATION: lexical alias rename changes chosen mapping and create inputs without changing operation ID")
}

func TestCharacterizeAliasLocalGapAndAvailableReadCoexist(t *testing.T) {
	w := multipleMappingWorld()
	w.Applications[0].Databases[0].Access = database.AccessWrite
	w.Applications[0].Databases[1].Access = database.AccessRead
	s := DeriveOperationSurface(w, "order_entry")
	op, ok := findOperation(s, "order_entry.list.sales.order")
	if !ok || op.Basis != mappingBasis(w.Persistence[0]) {
		t.Fatalf("primary read route not present: %+v", s.Operations)
	}
	gap, ok := findGap(s, "sales.order", "list")
	if !ok || gap.Reason != "conflict" {
		t.Fatalf("archive write-only gap not present: %+v", s.Gaps)
	}
	if _, ok := findOperation(s, "order_entry.create.sales.order"); !ok {
		t.Fatal("archive write remains a create candidate")
	}
	t.Log("CHARACTERIZATION: alias-local read restriction and another alias's list candidate coexist")
}

func TestCharacterizeUnselectedMappingChangesDoNotAppearInDerivedSurface(t *testing.T) {
	w := multipleMappingWorld()
	a := DeriveOperationSurface(w, "order_entry")
	// The primary mapping is not selected for these ID-colliding operations.
	// A change to it is semantically relevant for a caller that expected the
	// primary route, even if it does not affect the selected surface content.
	w.Persistence[0].Table = "orders_new"
	b := DeriveOperationSurface(w, "order_entry")
	if !reflect.DeepEqual(a, b) {
		t.Fatalf("unselected mapping now affects derived Surface; update characterization: a=%+v b=%+v", a, b)
	}
	t.Log("CHARACTERIZATION: derived Surface is insensitive to a change in a discarded mapping (Workspace ModelRevision may still change)")
}

func TestCharacterizeSelectedMappingColumnChangesInputsButNotBasis(t *testing.T) {
	w := multipleMappingWorld()
	before := DeriveOperationSurface(w, "order_entry")
	a, ok := findOperation(before, "order_entry.create.sales.order")
	if !ok {
		t.Fatal("before create not derived")
	}

	// The archive mapping is currently the selected first alias. Add one
	// persisted field without changing the mapping ID, table or primary key.
	w.Persistence[1].Columns = append(w.Persistence[1].Columns, persistence.ColumnMapping{
		Attribute:   "sales.order.status",
		Column:      "status",
		Storage:     persistence.StorageVarchar,
		Nullability: persistence.Required,
	})
	after := DeriveOperationSurface(w, "order_entry")
	b, ok := findOperation(after, "order_entry.create.sales.order")
	if !ok {
		t.Fatal("after create not derived")
	}
	if a.ID != b.ID || a.Basis != b.Basis {
		t.Fatalf("current mapping basis unexpectedly changed: before=%+v after=%+v", a, b)
	}
	if len(a.Inputs) != 1 || len(b.Inputs) != 2 {
		t.Fatalf("a source mapping field must change derived inputs: before=%+v after=%+v", a.Inputs, b.Inputs)
	}
	t.Log("CHARACTERIZATION: create inputs differ while operation ID and mapping Basis remain identical")
}

func TestCharacterizeStorageTypeChangeInvisibleToPureDerivedSurface(t *testing.T) {
	w := multipleMappingWorld()
	a := DeriveOperationSurface(w, "order_entry")
	// Change the physical type of an already selected archive column.
	// The currently projected operation inputs remain attribute IDs.
	w.Persistence[1].Columns[0].Storage = persistence.StorageVarchar
	b := DeriveOperationSurface(w, "order_entry")
	if !reflect.DeepEqual(a, b) {
		t.Fatalf("storage-only change now affects derived surface: before=%+v after=%+v", a, b)
	}
	t.Log("CHARACTERIZATION: physical storage change invisible to pure Operation Surface (workspace fingerprint not checked)")
}

func TestCharacterizeSelectedMappingTableChangeAffectsBasis(t *testing.T) {
	w := multipleMappingWorld()
	a := DeriveOperationSurface(w, "order_entry")
	before, ok := findOperation(a, "order_entry.create.sales.order")
	if !ok {
		t.Fatal("before create not derived")
	}
	// Positive control: Table belongs to the declared Basis preimage.
	w.Persistence[1].Table = "migrated_archive_orders"
	b := DeriveOperationSurface(w, "order_entry")
	after, ok := findOperation(b, "order_entry.create.sales.order")
	if !ok {
		t.Fatal("after create not derived")
	}
	if before.ID != after.ID || before.Basis == after.Basis || !reflect.DeepEqual(before.Inputs, after.Inputs) {
		t.Fatalf("expected stable operation ID and inputs but changed Basis: before=%+v after=%+v", before, after)
	}
	t.Log("CHARACTERIZATION: selected table changes the mapping Basis, unlike selected Storage/Column changes")
}
