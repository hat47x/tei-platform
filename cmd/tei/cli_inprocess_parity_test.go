package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"tei/internal/operations"
	"tei/internal/sourceindex"
)

// These tests keep the CLI an adapter. For the same Workspace and request the
// CLI JSON result must equal the typed in-process operation result, so Web or
// Desktop adapters can call the operation without re-implementing semantics
// (#1158 acceptance 1). They run the actual CLI transport functions and the
// actual operations on a real Workspace; no HTTP or Desktop adapter is
// involved, so they do not prove those adapters.

// jsonValue normalizes a typed Go value into its JSON document so the CLI
// bytes and the in-process value are compared on the same representation.
func jsonValue(t *testing.T, v any) any {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	var out any
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func TestInspectCLIResultEqualsInProcessOperation(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "workspace")
	for _, subject := range []sourceindex.Subject{
		{Kind: "semantic-data", ID: "sales.order"},
		{Kind: "semantic-relation", ID: "sales.order.customer"},
	} {
		var out bytes.Buffer
		code, err := inspectWorkspaceJSON(&out, root, "", subject.Kind, subject.ID, 0)
		if err != nil || code != 0 {
			t.Fatalf("%v: exit=%d err=%v body=%s", subject, code, err, out.String())
		}
		var cli struct {
			OK     bool            `json:"ok"`
			Result json.RawMessage `json:"result"`
		}
		if err := json.Unmarshal(out.Bytes(), &cli); err != nil || !cli.OK {
			t.Fatalf("%v: bad CLI document: %v %s", subject, err, out.String())
		}
		var cliResult any
		if err := json.Unmarshal(cli.Result, &cliResult); err != nil {
			t.Fatal(err)
		}
		typed, err := operations.InspectSubject(root, subject)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(cliResult, jsonValue(t, typed)) {
			t.Fatalf("%v: CLI inspect result differs from the typed operation:\ncli=%s\ngo=%+v", subject, cli.Result, typed)
		}
	}
}

func TestInspectCLIFailureCodesEqualOperationErrors(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "workspace")
	for _, subject := range []sourceindex.Subject{
		{Kind: "semantic-data", ID: "sales.not_found"},
		{Kind: "no-such-kind", ID: "sales.order"},
		{Kind: "semantic-data", ID: " sales.order"},
	} {
		var out bytes.Buffer
		code, err := inspectWorkspaceJSON(&out, root, "", subject.Kind, subject.ID, 0)
		if err != nil || code == 0 {
			t.Fatalf("%v: expected nonzero exit, got %d err=%v", subject, code, err)
		}
		var cli cliInspectionReport
		if err := json.Unmarshal(out.Bytes(), &cli); err != nil || cli.OK || cli.Error == nil {
			t.Fatalf("%v: failure must be one JSON error: %v %s", subject, err, out.String())
		}
		_, opErr := operations.InspectSubject(root, subject)
		var typed *operations.InspectionError
		if !errors.As(opErr, &typed) || typed.Code != cli.Error.Code {
			t.Fatalf("%v: CLI code %q differs from operation error %v", subject, cli.Error.Code, opErr)
		}
	}
}

func TestValidateCLIDiagnosticsEqualInProcessOperation(t *testing.T) {
	cases := map[string]string{
		"semantic": `application order_entry {
    database sales {
        target sales_core
        access invalid
    }
}
`,
		"syntax": "application order_entry {\n",
	}
	for name, source := range cases {
		root := t.TempDir()
		writeWorkspaceFile(t, root, "tei.workspace.yaml", "workspace: parity\nsources:\n  - applications\n")
		writeWorkspaceFile(t, root, "applications/order-entry/application.tei", source)
		var out bytes.Buffer
		code, err := validateWorkspaceJSON(&out, root)
		if err != nil || code != 1 {
			t.Fatalf("%s: exit=%d err=%v body=%s", name, code, err, out.String())
		}
		report := readValidationReport(t, &out)
		typed, err := operations.ValidateWorkspace(root)
		if err != nil {
			t.Fatal(err)
		}
		if !typed.HasErrors() || report.OK {
			t.Fatalf("%s: CLI and operation must both report errors: cli=%+v typed=%+v", name, report, typed)
		}
		var want []string
		if typed.Direct != nil {
			want = append(want, typed.Direct.Code)
		}
		for _, projected := range typed.Diagnostics {
			want = append(want, projected.Diagnostic.Code)
		}
		var got []string
		for _, item := range report.Diagnostics {
			got = append(got, item.Diagnostic.Code)
		}
		if len(got) == 0 || !reflect.DeepEqual(sortedCopy(got), sortedCopy(want)) {
			t.Fatalf("%s: CLI diagnostics %v differ from operation diagnostics %v", name, got, want)
		}
	}
}

func sortedCopy(in []string) []string {
	out := append([]string(nil), in...)
	sort.Strings(out)
	return out
}
