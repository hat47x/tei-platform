//go:build studiohost_parity

package hostqualification

import (
	"encoding/json"
	"errors"
	"path/filepath"
	"reflect"
	"testing"

	"tei/internal/operations"
	"tei/internal/sourceindex"
	"tei/internal/studiohost"
)

// These tests intentionally bind only to #1832's real, Host-neutral Go
// ReadOnly bridge. They do not connect to Wails IPC or assert authorization
// for subjects outside the preconfigured Workspace allowlist.

func envelopeObject(t *testing.T, response any, success bool, failureCode string) map[string]any {
	t.Helper()
	encoded, err := json.Marshal(response)
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err := json.Unmarshal(encoded, &document); err != nil {
		t.Fatal(err)
	}
	if document["ok"] != success {
		t.Fatalf("Host ok differs from operation completion: %s", encoded)
	}
	value, hasValue := document["value"]
	failure, hasFailure := document["failure"]
	if success {
		if !hasValue || value == nil || hasFailure {
			t.Fatalf("successful response must have value, no failure: %s", encoded)
		}
	} else {
		if hasValue || !hasFailure {
			t.Fatalf("denied/failed response must have failure, no value: %s", encoded)
		}
		decoded, ok := failure.(map[string]any)
		if !ok || decoded["code"] != failureCode || decoded["message"] == "" {
			t.Fatalf("failure response lacks classified code/message: %s", encoded)
		}
	}
	return document
}

func TestRealStudioHostWorkspaceIDBoundaryAndEnvelopes(t *testing.T) {
	reference := filepath.Join("..", "..", "reference", "workspace")
	host, err := studiohost.NewReadOnly([]string{reference})
	if err != nil {
		t.Fatal(err)
	}
	listed := host.ListWorkspaces()
	listDoc := envelopeObject(t, listed, true, "")
	if !listed.OK || len(listed.Value) != 1 || listed.Value[0].ID != "workspace-1" {
		t.Fatalf("unexpected Host allowlist: %+v (%v)", listed, listDoc)
	}

	for _, id := range []string{"", "workspace-2", "workspace-999", "../reference/workspace", reference, " workspace-1"} {
		t.Run("unregistered-"+id, func(t *testing.T) {
			v := host.ValidateWorkspace(id)
			c := host.CatalogWorkspace(id, "sales")
			i := host.InspectSubject(id, "semantic-data", "sales.order")
			for _, response := range []any{v, c, i} {
				envelopeObject(t, response, false, "WORKSPACE_NOT_ALLOWED")
			}
			if v.Failure == nil || c.Failure == nil || i.Failure == nil ||
				v.Failure.Stage != "" || c.Failure.Stage != "" || i.Failure.Stage != "" {
				t.Fatalf("allowlist rejection must precede operations: validate=%+v catalog=%+v inspect=%+v", v, c, i)
			}
		})
	}

	// There is no CLI/Operation notion of renderer-supplied Workspace ID.
	// Refuse to map an unknown ID to a guessed filesystem path for 'parity'.
}

func TestRealStudioHostWarningNotEquivalentToSemanticValidOrTransportFailure(t *testing.T) {
	reference := filepath.Join("..", "..", "reference", "workspace")
	host, err := studiohost.NewReadOnly([]string{reference})
	if err != nil {
		t.Fatal(err)
	}
	binary := testCLIBinary(t)
	cli := observeCLI(t, binary, 0, "validate", "-workspace", reference, "-json")
	goValue, goErr := operations.ValidateWorkspace(reference)
	h := host.ValidateWorkspace("workspace-1")
	doc := envelopeObject(t, h, true, "")
	if cli.SchemaVersion != "tei.cli.validate/v0" ||
		goErr != nil || h.Value == nil || h.Failure != nil ||
		!cli.OK || cli.Error != nil || goValue.HasErrors() || h.Value.HasErrors() {
		t.Fatalf("warnings must not become operation/transport failure: CLI=%+v Go=%+v err=%v Host=%+v", cli, goValue, goErr, h)
	}
	cliFind := cliFindings(cli)
	goFind := typedFindings(t, goValue)
	hostFind := typedFindings(t, *h.Value)
	if len(cliFind) != 3 ||
		!reflect.DeepEqual(cliFind, goFind) ||
		!reflect.DeepEqual(goFind, hostFind) ||
		!reflect.DeepEqual(normalizedJSON(t, goValue), normalizedJSON(t, *h.Value)) {
		t.Fatalf("fixed warning fixture must preserve all three findings: CLI=%v Go=%v Host=%v", cliFind, goFind, hostFind)
	}
	for _, finding := range cli.Diagnostics {
		if finding.Diagnostic.Severity != "warning" {
			t.Fatalf("expected warning, got %s", finding.Diagnostic.Severity)
		}
	}
	for _, finding := range h.Value.Diagnostics {
		if string(finding.Diagnostic.Severity) != "warning" {
			t.Fatalf("Host warning unexpectedly changed severity: %+v", finding)
		}
	}
	value, ok := doc["value"].(map[string]any)
	if !ok || value["Direct"] != nil {
		t.Fatalf("Host validation value must retain Direct and Diagnostics: %+v", doc)
	}
	diagnostics, ok := value["Diagnostics"].([]any)
	if !ok || len(diagnostics) != 3 {
		t.Fatalf("Host JSON must retain all three warning projections: %+v", doc)
	}
	// The status has separate dimensions: Host OK=true, semantic HasErrors=false,
	// diagnostics nonempty. Empty diagnostics is not synonymous with valid.
}

// Semantic errors are findings inside a successful Go Host invocation.
// CLI reports exit=1, while Host ok=true: these are NOT contradictory results.
func TestRealStudioHostTransportSuccessWithSemanticErrors(t *testing.T) {
	broken := fixtureWorkspace(t, "hostsemanticinvalid", "namespace hostsemanticinvalid\n\ndata Broken {\n    attribute invalid type }\n}\n")
	host, err := studiohost.NewReadOnly([]string{broken})
	if err != nil {
		t.Fatal(err)
	}
	binary := testCLIBinary(t)
	cli := observeCLI(t, binary, 1, "validate", "-workspace", broken, "-json")
	goValue, goErr := operations.ValidateWorkspace(broken)
	response := host.ValidateWorkspace("workspace-1")
	doc := envelopeObject(t, response, true, "")
	if goErr != nil || response.Value == nil || !response.OK || response.Failure != nil ||
		cli.OK || cli.Error != nil || !goValue.HasErrors() || !response.Value.HasErrors() {
		t.Fatalf("Host invocation success must not erase semantic errors: CLI=%+v Go=%+v err=%v Host=%+v",
			cli, goValue, goErr, response)
	}
	c, g, h := cliFindings(cli), typedFindings(t, goValue), typedFindings(t, *response.Value)
	if len(c) == 0 || !reflect.DeepEqual(c, g) || !reflect.DeepEqual(g, h) {
		t.Fatalf("semantic errors must be preserved by all three paths: CLI=%v Go=%v Host=%v", c, g, h)
	}
	if doc["failure"] != nil {
		t.Fatalf("semantic errors must be reported as value, not transport failure: %+v", doc)
	}
}

func TestRealStudioHostSubjectRejectionsOnIdenticalCLIAndGoInputs(t *testing.T) {
	reference := filepath.Join("..", "..", "reference", "workspace")
	host, err := studiohost.NewReadOnly([]string{reference})
	if err != nil {
		t.Fatal(err)
	}
	binary := testCLIBinary(t)
	for _, tc := range []struct {
		name, kind, id, code string
		exit                 int
	}{
		{"missing", "semantic-data", "sales.subject_that_does_not_exist", operations.InspectionNotFound, 1},
		{"unsupported-kind", "unknown-kind", "sales.order", operations.InspectionUnsupportedKind, 2},
		{"invalid-blank-kind", "", "sales.order", operations.InspectionInvalidSubject, 2},
		{"invalid-blank-id", "semantic-data", "", operations.InspectionInvalidSubject, 2},
		{"invalid-whitespace", "semantic-data", " sales.order", operations.InspectionInvalidSubject, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cli := observeCLI(t, binary, tc.exit,
				"inspect", "-workspace", reference, "-subject-kind", tc.kind, "-subject-id", tc.id)
			_, opErr := operations.InspectSubject(reference, sourceindex.Subject{Kind: tc.kind, ID: tc.id})
			h := host.InspectSubject("workspace-1", tc.kind, tc.id)
			envelopeObject(t, h, false, tc.code)
			var typed *operations.InspectionError
			if !errors.As(opErr, &typed) || typed.Code != tc.code ||
				cli.SchemaVersion != "tei.cli.inspect/v0" ||
				cli.OK || cli.Error == nil || cli.Error.Code != tc.code ||
				h.OK || h.Value != nil || h.Failure == nil || h.Failure.Code != tc.code ||
				h.Failure.Stage != "" {
				t.Fatalf("subject classification differs for fixed input %+v: CLI=%+v Go=%v Host=%+v", tc, cli, opErr, h)
			}
		})
	}
}

func TestRealStudioHostInvalidFilterRetainsOperationFailure(t *testing.T) {
	reference := filepath.Join("..", "..", "reference", "workspace")
	host, err := studiohost.NewReadOnly([]string{reference})
	if err != nil {
		t.Fatal(err)
	}
	binary := testCLIBinary(t)
	cli := observeCLI(t, binary, 2,
		"model", "catalog", "-workspace", reference, "-namespace", " sales")
	_, goErr := operations.CatalogWorkspace(reference, " sales")
	h := host.CatalogWorkspace("workspace-1", " sales")
	envelopeObject(t, h, false, "OPERATION_FAILED")
	if cli.SchemaVersion != "tei.cli.model/v0" || cli.OK || cli.Error == nil ||
		cli.Error.Code != "INVALID_ARGUMENT" || goErr == nil ||
		h.OK || h.Value != nil || h.Failure == nil || h.Failure.Stage != "" {
		t.Fatalf("invalid exact namespace incorrectly classified: CLI=%+v Go=%v Host=%+v", cli, goErr, h)
	}
	// Shared semantic rejection, but distinct transport-stage codes:
	// CLI rejects an invalid flag before calling Go operations, whereas Host
	// maps the unclassified operation error to OPERATION_FAILED.
	if h.Failure.Code != "OPERATION_FAILED" {
		t.Fatalf("expected classified Host operation error: %+v", h.Failure)
	}
}
