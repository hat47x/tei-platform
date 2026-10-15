//go:build studiohost_parity

package hostqualification

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/operations"
	"tei/internal/studiohost"
	"tei/internal/workspace"
)

// Construct a REAL Load-success/BuildSourceIndex-failure fixture instead of
// synthesizing a ValidationFailure. The loader's persistence parser retains
// both index declarations; BuildSourceIndex rejects duplicate source identity.
func TestSourceIndexFailureAfterSuccessfulLoadAcrossCLIAndHost(t *testing.T) {
	root := t.TempDir()
	writeFixtureFile := func(path, content string) {
		t.Helper()
		full := filepath.Join(root, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	writeFixtureFile("tei.workspace.yaml", "workspace: index_only_failure\nsources:\n  - databases\n")
	writeFixtureFile("databases/sales-core/database.tei",
		"database sales_core {\n    kind relational\n    ownership managed-here\n    uses semantic sales\n}\n")
	writeFixtureFile("databases/sales-core/persistence/order.tei",
		"persist sales_core.order {\n"+
			"    database sales_core\n"+
			"    semantic sales.order\n"+
			"    table public.orders\n"+
			"    index duplicate_idx on sales.order.id\n"+
			"    index duplicate_idx on sales.order.id\n"+
			"}\n")

	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatalf("fixture must load successfully before Source Index construction: %v", err)
	}
	if len(loaded.World.Persistence) != 1 || len(loaded.World.Persistence[0].Indexes) != 2 {
		t.Fatalf("fixture did not retain two persistence indexes: %+v", loaded.World.Persistence)
	}
	_, indexErr := workspace.BuildSourceIndex(root, loaded)
	if indexErr == nil || !strings.Contains(indexErr.Error(), "duplicate index name") {
		t.Fatalf("must fail during index construction for duplicate source identity: %v", indexErr)
	}

	host, err := studiohost.NewReadOnly([]string{root})
	if err != nil {
		t.Fatal(err)
	}
	binary := testCLIBinary(t)
	t.Run("validate", func(t *testing.T) {
		cli := observeCLI(t, binary, 2, "validate", "-workspace", root, "-json")
		_, opErr := operations.ValidateWorkspace(root)
		var typed *operations.ValidationFailure
		response := host.ValidateWorkspace("workspace-1")
		envelopeObject(t, response, false, "SOURCE_INDEX_FAILED")
		if !errors.As(opErr, &typed) || typed.Stage != operations.StageSourceIndex ||
			cli.SchemaVersion != "tei.cli.validate/v0" || cli.OK ||
			cli.Error == nil || cli.Error.Code != "SOURCE_INDEX_FAILED" ||
			len(cli.Diagnostics) != 0 || response.OK || response.Value != nil ||
			response.Failure == nil || response.Failure.Stage != "source-index" {
			t.Fatalf("index-only failure incorrectly reported: CLI=%+v Go=%v Host=%+v", cli, opErr, response)
		}
	})
	t.Run("catalog", func(t *testing.T) {
		cli := observeCLI(t, binary, 2, "model", "catalog", "-workspace", root)
		_, opErr := operations.CatalogWorkspace(root, "")
		var typed *operations.ValidationFailure
		response := host.CatalogWorkspace("workspace-1", "")
		envelopeObject(t, response, false, "SOURCE_INDEX_FAILED")
		if !errors.As(opErr, &typed) || typed.Stage != operations.StageSourceIndex ||
			cli.SchemaVersion != "tei.cli.model/v0" || cli.OK ||
			cli.Error == nil || cli.Error.Code != "SOURCE_INDEX_FAILED" ||
			response.OK || response.Value != nil || response.Failure == nil ||
			response.Failure.Stage != "source-index" {
			t.Fatalf("catalog index-only failure incorrectly reported: CLI=%+v Go=%v Host=%+v", cli, opErr, response)
		}
	})
}
