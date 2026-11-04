package artifact

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/application"
	"tei/internal/database"
	"tei/internal/database/persistence"
	"tei/internal/ir"
	"tei/internal/operations"
	"tei/internal/semantic"
)

func TestRenderWorkspaceRendersTheNegativeFixture(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "standard-crud", "workspace")

	set, err := RenderWorkspace(root, "ledger")
	if err != nil {
		t.Fatalf("render workspace: %v", err)
	}
	if set.InputRevision == "" {
		t.Fatal("the input revision must come from the workspace")
	}
	if len(set.Artifacts) != 5 {
		t.Fatalf("expected three table definitions, one function definition and one operation surface: %d", len(set.Artifacts))
	}
	kinds := map[string]int{}
	for _, artifact := range set.Artifacts {
		kinds[artifact.Kind]++
		if artifact.InputRevision != set.InputRevision {
			t.Fatalf("artifact %s is bound to another revision", artifact.Kind)
		}
	}
	if kinds[string(KindTableDefinition)] != 3 || kinds[string(KindFunctionDefinition)] != 1 || kinds[string(KindOperationSurface)] != 1 {
		t.Fatalf("unexpected artifact kinds: %+v", kinds)
	}

	// sales.note has no key, so its table definition stays unresolved and the
	// set must not claim to be complete.
	if set.Manifest.Complete {
		t.Fatalf("the negative fixture must not be a complete set: %+v", set.Manifest)
	}
	var unresolved int
	for _, artifact := range set.Artifacts {
		unresolved += len(artifact.Unresolved)
		if artifact.Kind != string(KindOperationSurface) {
			continue
		}
		if err := artifact.Verify(); err != nil {
			t.Fatalf("the surface artifact must verify: %v", err)
		}
		if !strings.Contains(artifact.Content, operations.OperationSurfaceSchemaVersion) {
			t.Fatalf("the surface artifact must carry the surface schema:\n%s", artifact.Content)
		}
		// The fixture has four gaps: note read/create, order list conflict and
		// order retire. Each gap becomes one unresolved entry.
		if len(artifact.Unresolved) != 4 {
			t.Fatalf("every gap must be carried as unresolved: %+v", artifact.Unresolved)
		}
	}
	if unresolved == 0 {
		t.Fatal("the negative fixture must carry unresolved entries")
	}
	// Generation runs CheckSet itself, so an inconsistent set can not be
	// handed to a caller as if it were consistent.
	if len(set.Findings) != 0 {
		t.Fatalf("a consistently rendered set must carry no findings: %+v", set.Findings)
	}
}

func TestRenderWorkspaceRefusesAnUnknownApplication(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "standard-crud", "workspace")
	if _, err := RenderWorkspace(root, "missing"); err == nil {
		t.Fatal("an application that the workspace does not declare must be refused")
	}
}
func TestRenderWorkspaceListsOnlyUsedNamespaces(t *testing.T) {
	root := t.TempDir()
	write := func(path, content string) {
		full := filepath.Join(root, path)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("tei.workspace.yaml", "workspace: scoped\nformatVersion: 1\nsources:\n  - semantic\n  - applications\n")
	write("semantic/used/data.tei", "namespace used\n\ndata Item {\n    attribute id type std.identifier\n}\n\ncapability used.item.touch {\n    input item used.item\n    outcome touched used.item\n}\n")
	write("semantic/unused/data.tei", "namespace unused\n\ndata Other {\n    attribute id type std.identifier\n}\n\ncapability unused.other.touch {\n    input other unused.other\n    outcome touched unused.other\n}\n")
	write("applications/app/application.tei", "application app {\n    uses semantic used\n}\n")

	set, err := RenderWorkspace(root, "app")
	if err != nil {
		t.Fatalf("render workspace: %v", err)
	}
	var content string
	for _, artifact := range set.Artifacts {
		if artifact.Kind == string(KindFunctionDefinition) {
			content = artifact.Content
		}
	}
	if !strings.Contains(content, "capability used.item.touch") {
		t.Fatalf("the used namespace must appear:\n%s", content)
	}
	if strings.Contains(content, "capability unused.other.touch") {
		t.Fatalf("a namespace the application does not use must not appear:\n%s", content)
	}
}
func TestCheckSurfaceTargetsReportsAnUnreachableData(t *testing.T) {
	payload, err := json.MarshalIndent(operationSurfacePayload{
		SchemaVersion: "tei.operations.operation-surface/v0",
		Operations: []operations.Operation{
			{ID: "app.list.sales.order", Kind: "list", Target: "sales.order", Autonomy: "unattended", Basis: "persist"},
			{ID: "app.list.ghost.data", Kind: "list", Target: "ghost.data", Autonomy: "unattended", Basis: "persist"},
		},
	}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := New(KindOperationSurface, Rule{ID: OperationSurfaceRuleID, Version: OperationSurfaceRuleVersion, Fingerprint: "sha256:rule"}, OperationSurfaceGenerator, "sha256:input", string(payload)+"\n", nil)
	if err != nil {
		t.Fatal(err)
	}
	world := ir.World{
		Applications: []application.Application{{
			ID:        "app",
			Databases: []application.DatabaseRequirement{{Alias: "sales", TargetDatabase: "sales_core", Access: database.AccessReadWrite}},
		}},
		Persistence: []persistence.Mapping{{
			ID: "sales_core.order", Database: "sales_core", SemanticData: "sales.order", Table: "orders",
		}},
	}

	findings := checkSurfaceTargets(world, world.Applications[0], []Artifact{artifact})
	if len(findings) != 1 || findings[0].Code != FindingSurfaceTargetUnreachable || findings[0].Subject != "app.list.ghost.data" {
		t.Fatalf("unexpected findings: %+v", findings)
	}
}
func TestRenderWorkspaceKeepsTheReferenceWorkspaceConsistent(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "workspace")
	set, err := RenderWorkspace(root, "order_entry")
	if err != nil {
		t.Fatalf("render the reference workspace: %v", err)
	}
	if set.InputRevision == "" {
		t.Fatal("the reference workspace must yield an input revision")
	}
	if len(set.Findings) != 0 {
		t.Fatalf("the reference workspace must stay consistent: %+v", set.Findings)
	}
	if !set.Manifest.Complete {
		var unresolved int
		for _, artifact := range set.Artifacts {
			unresolved += len(artifact.Unresolved)
		}
		if unresolved == 0 {
			t.Fatalf("an incomplete manifest must name at least one unresolved entry: %+v", set.Manifest)
		}
	}
	var surface bool
	for _, artifact := range set.Artifacts {
		if artifact.Kind == string(KindOperationSurface) {
			surface = true
			if !strings.Contains(artifact.Content, "order_entry.update.sales.order") {
				t.Fatalf("the reference surface must derive the standard update:\n%s", artifact.Content)
			}
		}
	}
	if !surface {
		t.Fatal("the reference set must include the operation surface")
	}
}
func TestCheckPeriodPreconditionsReportsAMismatch(t *testing.T) {
	world := ir.World{Semantics: []semantic.Namespace{{
		ID: "s",
		Periods: []semantic.Period{{
			ID:   "s.month",
			Data: "s.order",
		}},
	}}}
	payload, err := json.MarshalIndent(operationSurfacePayload{
		SchemaVersion: operations.OperationSurfaceSchemaVersion,
		Operations: []operations.Operation{
			{ID: "ok.create", Kind: "create", Target: "s.order", Preconditions: []string{"period:s.month:open"}, Basis: "x"},
			{ID: "wrong.create", Kind: "create", Target: "s.other", Preconditions: []string{"period:s.month:open"}, Basis: "x"},
			{ID: "unknown.create", Kind: "create", Target: "s.order", Preconditions: []string{"period:s.nope:open"}, Basis: "x"},
		},
	}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := New(KindOperationSurface, Rule{ID: OperationSurfaceRuleID, Version: OperationSurfaceRuleVersion, Fingerprint: "sha256:rule"}, OperationSurfaceGenerator, "sha256:input", string(payload)+"\n", nil)
	if err != nil {
		t.Fatal(err)
	}

	findings := checkPeriodPreconditions(world, []Artifact{artifact})
	if len(findings) != 2 {
		t.Fatalf("a mismatched and an unknown period must both be reported: %+v", findings)
	}
	codes := map[string]bool{}
	for _, finding := range findings {
		codes[finding.Code] = true
	}
	if !codes["PERIOD_PRECONDITION_UNKNOWN"] || !codes["PERIOD_PRECONDITION_MISMATCH"] {
		t.Fatalf("unexpected codes: %+v", codes)
	}
}
