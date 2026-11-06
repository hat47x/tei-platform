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
			Columns: []persistence.ColumnMapping{
				{Attribute: "sales.order.order_id", Column: "order_id", Storage: persistence.StorageInt64, Nullability: persistence.Required},
				{Attribute: "sales.order.status", Column: "status", Storage: persistence.StorageVarchar, Nullability: persistence.Required},
			},
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

func findGap(surface OperationSurface, target, kind string) (SurfaceGap, bool) {
	for _, gap := range surface.Gaps {
		if gap.Target == target && gap.Kind == kind {
			return gap, true
		}
	}
	return SurfaceGap{}, false
}

func TestDeriveOperationSurfaceListReadAndCreate(t *testing.T) {
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

	create, ok := findOperation(surface, "order_entry.create.sales.order")
	if !ok {
		t.Fatalf("create operation missing: %+v", surface.Operations)
	}
	if create.Autonomy != "attended" {
		t.Fatalf("create must be attended: %+v", create)
	}
	if len(create.Inputs) != 2 || create.Inputs[0] != "sales.order.order_id" || create.Inputs[1] != "sales.order.status" {
		t.Fatalf("create inputs must follow the persisted columns: %+v", create.Inputs)
	}
	if len(create.Effects) != 1 || create.Effects[0] != "create:sales.order" {
		t.Fatalf("create must declare its effect: %+v", create.Effects)
	}
	if create.Basis == "" || list.Basis == "" {
		t.Fatalf("derived operations must carry a basis: %+v", surface.Operations)
	}

	// update is derived with expected-state comparison (PoC hypothesis H1).
	update, ok := findOperation(surface, "order_entry.update.sales.order")
	if !ok {
		t.Fatalf("update operation missing: %+v", surface.Operations)
	}
	if update.Concurrency != "expected-state" || update.Autonomy != "attended" {
		t.Fatalf("update must declare expected-state comparison and stay attended: %+v", update)
	}
	if len(update.Inputs) != 1 || update.Inputs[0] != "sales.order.order_id" {
		t.Fatalf("update must carry the identifying key: %+v", update.Inputs)
	}
	if len(update.Effects) != 1 || update.Effects[0] != "update:sales.order" {
		t.Fatalf("update must declare its effect: %+v", update.Effects)
	}

	if len(surface.Gaps) != 0 {
		t.Fatalf("unexpected gaps: %+v", surface.Gaps)
	}

	if surface.Summary.Operations != 4 || surface.Summary.Derived != 4 ||
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
	if _, ok := findOperation(surface, "order_entry.create.sales.order"); ok {
		t.Fatal("create must not be derived without an identifying key")
	}
	if _, ok := findOperation(surface, "order_entry.list.sales.order"); !ok {
		t.Fatal("list must still be derived")
	}
	readGap, ok := findGap(surface, "sales.order", "read")
	if !ok || readGap.Reason != "missing-semantics" {
		t.Fatalf("expected a read gap: %+v", surface.Gaps)
	}
	createGap, ok := findGap(surface, "sales.order", "create")
	if !ok || createGap.Reason != "missing-semantics" {
		t.Fatalf("expected a create gap: %+v", surface.Gaps)
	}
	if len(surface.Gaps) != 2 {
		t.Fatalf("expected two gaps, one per operation: %+v", surface.Gaps)
	}
}

func TestDeriveOperationSurfaceWriteOnlyAlias(t *testing.T) {
	surface := DeriveOperationSurface(surfaceTestWorld(database.AccessWrite, true), "")

	if _, ok := findOperation(surface, "order_entry.list.sales.order"); ok {
		t.Fatalf("write-only alias must not derive read operations: %+v", surface.Operations)
	}
	if _, ok := findOperation(surface, "order_entry.read.sales.order"); ok {
		t.Fatalf("write-only alias must not derive read operations: %+v", surface.Operations)
	}
	gap, ok := findGap(surface, "sales.order", "list")
	if !ok || gap.Reason != "conflict" {
		t.Fatalf("expected a list conflict gap: %+v", surface.Gaps)
	}
	// A write-only alias can still create, because only the write side is used.
	create, ok := findOperation(surface, "order_entry.create.sales.order")
	if !ok || create.Autonomy != "attended" {
		t.Fatalf("write-only alias must derive create: %+v", surface.Operations)
	}
}

func TestDeriveOperationSurfaceReadOnlyAlias(t *testing.T) {
	surface := DeriveOperationSurface(surfaceTestWorld(database.AccessRead, true), "")

	if surface.Summary.Operations != 2 {
		t.Fatalf("read-only alias must derive only list and read: %+v", surface.Summary)
	}
	if _, ok := findOperation(surface, "order_entry.create.sales.order"); ok {
		t.Fatalf("read-only alias must not derive create: %+v", surface.Operations)
	}
	if len(surface.Gaps) != 0 {
		t.Fatalf("read-only alias must not report write-side gaps: %+v", surface.Gaps)
	}
}

func TestDeriveOperationSurfaceCreatePreconditions(t *testing.T) {
	world := surfaceTestWorld(database.AccessReadWrite, true)
	world.Semantics[0].Validations = []semantic.Validation{
		{ID: "sales.order.order_id_positive", Symbol: "orderIdPositive", Kind: "positive", Attribute: "sales.order.order_id"},
		{ID: "sales.customer.name_positive", Symbol: "namePositive", Kind: "positive", Attribute: "customer.customer.name"},
	}
	world.Applications[0].Validations = []core.CanonicalID{
		"sales.order.order_id_positive",
		"sales.customer.name_positive",
	}

	surface := DeriveOperationSurface(world, "")

	create, ok := findOperation(surface, "order_entry.create.sales.order")
	if !ok {
		t.Fatalf("create operation missing: %+v", surface.Operations)
	}
	if len(create.Preconditions) != 1 || create.Preconditions[0] != "sales.order.order_id_positive" {
		t.Fatalf("create must carry only the validations declared for this data: %+v", create.Preconditions)
	}
}

func TestDeriveOperationSurfaceRetireGapRequiresUpdateIntent(t *testing.T) {
	world := surfaceTestWorld(database.AccessReadWrite, true)
	if _, ok := findGap(DeriveOperationSurface(world, ""), "sales.order", "retire"); ok {
		t.Fatal("retire must not be reported without an update intent on the data")
	}

	world.Semantics[0].Capabilities = []semantic.Capability{{
		ID:     "sales.order.submit",
		Symbol: "submit",
		Inputs: []semantic.Parameter{{Name: "order", Type: "sales.order"}},
		Effect: &semantic.DataEffect{Kind: semantic.DataEffectUpdate, Data: "sales.order"},
	}}

	surface := DeriveOperationSurface(world, "")
	gap, ok := findGap(surface, "sales.order", "retire")
	if !ok || gap.Reason != "missing-input" || gap.Detail == "" {
		t.Fatalf("expected a retire gap explaining the missing lifecycle: %+v", surface.Gaps)
	}
	if _, ok := findOperation(surface, "order_entry.retire.sales.order"); ok {
		t.Fatalf("retire must not be derived without a lifecycle declaration: %+v", surface.Operations)
	}
}

func TestDeriveOperationSurfaceDoesNotDuplicateAcrossAliases(t *testing.T) {
	world := surfaceTestWorld(database.AccessReadWrite, true)
	world.Applications[0].Databases = append(world.Applications[0].Databases, application.DatabaseRequirement{
		Alias:          "sales_read",
		TargetDatabase: "sales_core",
		Access:         database.AccessRead,
	})

	surface := DeriveOperationSurface(world, "")

	counts := map[string]int{}
	for _, gap := range surface.Gaps {
		counts[gap.Kind]++
	}
	if counts["update"] != 0 {
		t.Fatalf("update is derived, so it must not be reported as a gap: %+v", surface.Gaps)
	}
	updates := 0
	creates := 0
	for _, operation := range surface.Operations {
		switch operation.Kind {
		case "create":
			creates++
		case "update":
			updates++
		}
	}
	if creates != 1 || updates != 1 {
		t.Fatalf("create and update must be derived once across aliases: %+v", surface.Operations)
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
	if all.Summary.Operations != 8 {
		t.Fatalf("expected eight operations for two applications: %+v", all.Summary)
	}
	filtered := DeriveOperationSurface(world, "repair_intake")
	if filtered.Summary.Operations != 4 || filtered.Application != "repair_intake" {
		t.Fatalf("filter must keep only the requested application: %+v", filtered)
	}
	if _, ok := findOperation(filtered, "repair_intake.create.sales.order"); !ok {
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
	create, ok := findOperation(surface, "order_entry.create.sales.order")
	if !ok {
		t.Fatalf("reference workspace lost the derived create: %+v", surface.Operations)
	}
	if len(create.Inputs) == 0 || create.Autonomy != "attended" {
		t.Fatalf("derived create must carry inputs and be attended: %+v", create)
	}
	update, ok := findOperation(surface, "order_entry.update.sales.order")
	if !ok || update.Concurrency != "expected-state" {
		t.Fatalf("reference workspace must derive update with expected-state comparison: %+v", surface.Operations)
	}
	// The reference workspace declares no capability with a write effect on
	// sales.order, so the write operations are derived but have no execution
	// binding yet. The surface must say so instead of implying that they run.
	if update.Binding != "" {
		t.Fatalf("reference workspace must not invent an execution binding: %+v", update)
	}
	if create, ok := findOperation(surface, "order_entry.create.sales.order"); !ok || create.Binding != "" {
		t.Fatalf("create must be derived without a binding: %+v", create)
	}
}

// TestSurfaceWorkspaceStandardCRDFixture derives the negative fixture. It fixes
// the shape of the gaps a reader sees when a workspace declares persistence but
// omits a key, a concurrency method, a lifecycle or a readable path.
func TestSurfaceWorkspaceStandardCRDFixture(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "standard-crud", "workspace")
	if _, err := os.Stat(root); err != nil {
		t.Skipf("standard-crud fixture is not available: %v", err)
	}

	surface, err := SurfaceWorkspace(root, "ledger")
	if err != nil {
		t.Fatalf("derive fixture surface: %v", err)
	}

	wantOperations := []string{
		"ledger.list.sales.note",
		"ledger.list.sales.order",
		"ledger.read.sales.order",
		"ledger.create.sales.order",
		"ledger.update.sales.order",
	}
	var gotOperations []string
	for _, operation := range surface.Operations {
		gotOperations = append(gotOperations, operation.ID)
	}
	if len(gotOperations) != len(wantOperations) {
		t.Fatalf("ops=%v want=%v", gotOperations, wantOperations)
	}
	for i, want := range wantOperations {
		if gotOperations[i] != want {
			t.Fatalf("ops=%v want=%v", gotOperations, wantOperations)
		}
	}

	wantGaps := []string{
		"sales.note/create/missing-semantics",
		"sales.note/read/missing-semantics",
		"sales.order/list/conflict",
		"sales.order/retire/missing-input",
	}
	var gotGaps []string
	for _, gap := range surface.Gaps {
		gotGaps = append(gotGaps, gap.Target+"/"+gap.Kind+"/"+gap.Reason)
	}
	if len(gotGaps) != len(wantGaps) {
		t.Fatalf("gaps=%v want=%v", gotGaps, wantGaps)
	}
	for i, want := range wantGaps {
		if gotGaps[i] != want {
			t.Fatalf("gaps=%v want=%v", gotGaps, wantGaps)
		}
	}
	for _, gap := range surface.Gaps {
		if gap.Detail == "" {
			t.Fatalf("gap must explain itself: %+v", gap)
		}
	}

	update, ok := findOperation(surface, "ledger.update.sales.order")
	if !ok || update.Binding != "sales.order.cancel" {
		t.Fatalf("update must bind to the capability that declares the effect: %+v", update)
	}
	create, ok := findOperation(surface, "ledger.create.sales.order")
	if !ok || create.Binding != "sales.order.register" {
		t.Fatalf("create must bind to the capability that declares the effect: %+v", create)
	}
}

func TestDeriveOperationSurfaceBindingRequiresTheDataAsInput(t *testing.T) {
	world := surfaceTestWorld(database.AccessReadWrite, true)
	world.Semantics[0].Capabilities = []semantic.Capability{
		{
			ID:     "sales.order.submit",
			Symbol: "submit",
			Inputs: []semantic.Parameter{{Name: "order", Type: "sales.order"}},
			Effect: &semantic.DataEffect{Kind: semantic.DataEffectUpdate, Data: "sales.order"},
		},
		{
			ID:     "sales.customer.touch",
			Symbol: "touch",
			Inputs: []semantic.Parameter{{Name: "customer", Type: "customer.customer"}},
			Effect: &semantic.DataEffect{Kind: semantic.DataEffectUpdate, Data: "sales.order"},
		},
	}

	surface := DeriveOperationSurface(world, "")

	update, ok := findOperation(surface, "order_entry.update.sales.order")
	if !ok {
		t.Fatalf("update operation missing: %+v", surface.Operations)
	}
	if update.Binding != "sales.order.submit" {
		t.Fatalf("only the capability that accepts the written data may be the binding: %q", update.Binding)
	}
}
func TestDeriveOperationSurfaceCarriesThePeriod(t *testing.T) {
	world := surfaceTestWorld(database.AccessReadWrite, true)
	world.Semantics[0].Periods = []semantic.Period{{
		ID:            "sales.month",
		Data:          "sales.order",
		BusinessDate:  "sales.order.order_id",
		States:        []string{"open", "closing", "closed"},
		ClosedRejects: []string{"create", "update"},
	}}

	surface := DeriveOperationSurface(world, "")

	create, ok := findOperation(surface, "order_entry.create.sales.order")
	if !ok {
		t.Fatalf("create operation missing: %+v", surface.Operations)
	}
	if create.Period != "sales.month" {
		t.Fatalf("create must carry the period: %+v", create)
	}
	if len(create.PeriodRejects) != 2 || create.PeriodRejects[0] != "create" {
		t.Fatalf("create must carry the operations the closed period rejects: %+v", create.PeriodRejects)
	}
	update, ok := findOperation(surface, "order_entry.update.sales.order")
	if !ok || update.Period != "sales.month" {
		t.Fatalf("update must carry the period: %+v", update)
	}
	list, _ := findOperation(surface, "order_entry.list.sales.order")
	if list.Period != "sales.month" {
		t.Fatalf("the period governs the data, so list carries it too: %+v", list)
	}
	// A write the closed period rejects must carry the proof it needs.
	if !hasPrecondition(create, "period:sales.month:open") || !hasPrecondition(update, "period:sales.month:open") {
		t.Fatalf("writes must require an open period: create=%+v update=%+v", create.Preconditions, update.Preconditions)
	}
	if hasPrecondition(list, "period:sales.month:open") {
		t.Fatalf("a read must not require an open period: %+v", list.Preconditions)
	}
}
func hasPrecondition(operation Operation, want string) bool {
	for _, precondition := range operation.Preconditions {
		if precondition == want {
			return true
		}
	}
	return false
}
func TestDeriveOperationSurfaceCarriesTheEffectRules(t *testing.T) {
	world := surfaceTestWorld(database.AccessReadWrite, true)
	world.Semantics[0].Capabilities = []semantic.Capability{{
		ID:     "sales.order.register",
		Inputs: []semantic.Parameter{{Name: "order", Type: "sales.order"}},
		Effect: &semantic.DataEffect{Kind: "create", Data: "sales.order"},
	}}
	world.Semantics[0].EffectRules = []semantic.EffectRule{{
		ID:            "sales.order.stock",
		OnCapability:  "sales.order.register",
		Stage:         "after-commit",
		EffectKind:    "update",
		EffectData:    "sales.order",
		ViaCapability: "sales.order.register",
	}}

	surface := DeriveOperationSurface(world, "")
	create, ok := findOperation(surface, "order_entry.create.sales.order")
	if !ok {
		t.Fatalf("create operation missing: %+v", surface.Operations)
	}
	if len(create.EffectRules) != 1 || create.EffectRules[0] != "sales.order.stock" {
		t.Fatalf("the bound operation must carry the effect rules: %+v", create)
	}
	list, _ := findOperation(surface, "order_entry.list.sales.order")
	if len(list.EffectRules) != 0 {
		t.Fatalf("an operation without the binding must not carry the rules: %+v", list.EffectRules)
	}
}
