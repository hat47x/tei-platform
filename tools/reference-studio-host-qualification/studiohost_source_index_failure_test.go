//go:build studiohost_parity

package hostqualification

import (
	"errors"
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
	root := sourceIndexDuplicateIndexFixture(t)

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

// Missing file at an indexed source input and a missing parsed source are NOT
// the same fixture. This is the existing broken-link boundary fixture run
// through real Load/CLI/Go/ReadOnly Host with one unchanged filesystem state.
func TestMissingSourceIndexInputCLIHostErrorLocation(t *testing.T) {
	root, missingPath := missingSourceIndexInputFixture(t)
	binary := testCLIBinary(t)
	cli := observeCLI(t, binary, 2, "validate", "-workspace", root, "-json")
	_, opErr := operations.ValidateWorkspace(root)
	host, err := studiohost.NewReadOnly([]string{root})
	if err != nil {
		t.Fatal(err)
	}
	response := host.ValidateWorkspace("workspace-1")
	envelopeObject(t, response, false, "WORKSPACE_LOAD_FAILED")

	var stage *operations.ValidationFailure
	if !errors.As(opErr, &stage) || stage.Stage != operations.StageLoad {
		t.Fatalf("Go operation did not stop at the Load inventory: %v", opErr)
	}
	if cli.SchemaVersion != "tei.cli.validate/v0" || cli.OK || cli.Error == nil ||
		cli.Error.Code != "WORKSPACE_LOAD_FAILED" ||
		len(cli.Diagnostics) != 0 || response.OK || response.Value != nil ||
		response.Failure == nil || response.Failure.Stage != "load" {
		t.Fatalf("CLI/Host disagree on first error stage: CLI=%+v Go=%v Host=%+v", cli, opErr, response)
	}

	// Neither transport has line/column diagnostics for a file that could not
	// be loaded. The CLI retains the originating source path in its error text;
	// the Host deliberately redacts it, so message/location equality is NOT a
	// valid interoperability requirement for this boundary.
	if !strings.Contains(filepath.ToSlash(cli.Error.Message), missingPath) ||
		cli.Error.Message != opErr.Error() {
		t.Fatalf("CLI did not retain original Load error location: cli=%q go=%v", cli.Error.Message, opErr)
	}
	if response.Failure.Message != "workspace loading failed" ||
		cli.Error.Message == response.Failure.Message ||
		strings.Contains(response.Failure.Message, missingPath) {
		t.Fatalf("Host must use a redacted Load failure, not CLI source path: cli=%q host=%+v",
			cli.Error.Message, response.Failure)
	}
}
