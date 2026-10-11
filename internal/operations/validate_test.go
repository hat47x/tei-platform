package operations

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func writeWorkspaceFixture(t *testing.T, root, name, body string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestValidateWorkspaceAsTypedReadOperation(t *testing.T) {
	root := t.TempDir()
	writeWorkspaceFixture(t, root, "tei.workspace.yaml", "workspace: typed_validation\nsources:\n  - semantic\n")
	writeWorkspaceFixture(t, root, "semantic/sales/order.tei", "namespace sales\n")

	valid, err := ValidateWorkspace(root)
	if err != nil {
		t.Fatal(err)
	}
	if valid.Direct != nil || valid.HasErrors() {
		t.Fatalf("unexpected validation failure: %+v", valid)
	}

	writeWorkspaceFixture(t, root, "semantic/sales/order.tei", "namespace sales\n\ndata Order {\n    attribute orderId type }\n}\n")
	invalid, err := ValidateWorkspace(root)
	if err != nil {
		t.Fatal(err)
	}
	if invalid.Direct == nil || invalid.Direct.Code != "PARSE_ERROR" || !invalid.HasErrors() {
		t.Fatalf("parser finding should remain direct: %+v", invalid)
	}
	if invalid.Direct.Source.ArtifactPath != "semantic/sales/order.tei" ||
		invalid.Direct.Source.Range.Start.Line != 4 {
		t.Fatalf("source coordinate lost: %+v", invalid.Direct)
	}
}

func TestValidateWorkspaceFailureIsNotBusinessFinding(t *testing.T) {
	_, err := ValidateWorkspace(filepath.Join(t.TempDir(), "missing"))
	var failure *ValidationFailure
	if !errors.As(err, &failure) || failure.Stage != StageLoad {
		t.Fatalf("expected typed load failure, got %v", err)
	}
}
