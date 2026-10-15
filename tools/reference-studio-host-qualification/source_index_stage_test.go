package hostqualification

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/operations"
	"tei/internal/workspace"
)

// Build the same Load-success/SourceIndex-only failure in both the ordinary
// CLI/Operation test and the studiohost_parity-tagged real Go Host test.
// This is distinct from a missing source file: the loader must succeed first.
func sourceIndexDuplicateIndexFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	write := func(name, content string) {
		t.Helper()
		full := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("tei.workspace.yaml", "workspace: index_only_failure\nsources:\n  - databases\n")
	write("databases/sales-core/database.tei",
		"database sales_core {\n    kind relational\n    ownership managed-here\n    uses semantic sales\n}\n")
	write("databases/sales-core/persistence/order.tei",
		"persist sales_core.order {\n"+
			"    database sales_core\n"+
			"    semantic sales.order\n"+
			"    table public.orders\n"+
			"    index duplicate_idx on sales.order.id\n"+
			"    index duplicate_idx on sales.order.id\n"+
			"}\n")
	return root
}

// Run in ordinary CI without studiohost_parity: distinguish the genuine
// Source Index failure from a broken-link input that already fails Load.
func TestSourceIndexStageFailureCLIAndGoOperations(t *testing.T) {
	root := sourceIndexDuplicateIndexFixture(t)
	loaded, loadErr := workspace.Load(root)
	if loadErr != nil {
		t.Fatalf("expected successful Load before Source Index: %v", loadErr)
	}
	if len(loaded.World.Persistence) != 1 || len(loaded.World.Persistence[0].Indexes) != 2 {
		t.Fatalf("expected two parsed indices: %+v", loaded.World.Persistence)
	}
	_, indexErr := workspace.BuildSourceIndex(root, loaded)
	if indexErr == nil || !strings.Contains(indexErr.Error(), "duplicate index name") {
		t.Fatalf("Source Index did not reject duplicate names: %v", indexErr)
	}

	binary := testCLIBinary(t)
	for _, tc := range []struct {
		name string
		args []string
		schema string
		run func() error
	}{
		{
			name: "validate",
			args: []string{"validate", "-workspace", root, "-json"},
			schema: "tei.cli.validate/v0",
			run: func() error { _, err := operations.ValidateWorkspace(root); return err },
		},
		{
			name: "catalog",
			args: []string{"model", "catalog", "-workspace", root},
			schema: "tei.cli.model/v0",
			run: func() error { _, err := operations.CatalogWorkspace(root, ""); return err },
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cli := observeCLI(t, binary, 2, tc.args...)
			opErr := tc.run()
			var typed *operations.ValidationFailure
			if !errors.As(opErr, &typed) || typed.Stage != operations.StageSourceIndex ||
				cli.SchemaVersion != tc.schema || cli.OK || cli.Error == nil ||
				cli.Error.Code != "SOURCE_INDEX_FAILED" ||
				!strings.Contains(cli.Error.Message, "duplicate index name") ||
				cli.Error.Message != opErr.Error() || len(cli.Diagnostics) != 0 || len(cli.Result) != 0 {
				t.Fatalf("genuine Source Index stage differs: CLI=%+v Go=%v", cli, opErr)
			}
		})
	}
}
