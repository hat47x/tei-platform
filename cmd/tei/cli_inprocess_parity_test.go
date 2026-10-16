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
	"tei/internal/studiohost"
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

func TestStudioCatalogMatchesCLIResultSemantics(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "workspace")
	for _, tc := range []struct {
		name      string
		namespace string
		wantApps  bool
	}{
		{name: "sales namespace", namespace: "sales"},
		{name: "all namespaces", wantApps: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			host, err := studiohost.NewReadOnly([]string{root})
			if err != nil {
				t.Fatal(err)
			}
			studio := host.CatalogWorkspace("workspace-1", tc.namespace)
			if !studio.OK || studio.Value == nil || studio.Failure != nil {
				t.Fatalf("Studio catalog failed: %+v", studio)
			}

			args := []string{"catalog", "-workspace", root}
			if tc.namespace != "" {
				args = append(args, "-namespace", tc.namespace)
			}
			var out bytes.Buffer
			exit, err := runModelCommand(args, &out)
			if err != nil || exit != 0 {
				t.Fatalf("CLI catalog failed: exit=%d err=%v body=%s", exit, err, out.String())
			}
			var cli modelCLIResponse
			if err := json.Unmarshal(out.Bytes(), &cli); err != nil {
				t.Fatal(err)
			}
			if !cli.OK || cli.Result == nil || cli.Error != nil {
				t.Fatalf("CLI catalog returned an invalid success envelope: %+v", cli)
			}

			// Compare typed operation fields after decoding the CLI transport envelope.
			// Entry and authority-limit order are normalized, while duplicate entries
			// remain present. This is not a JSON-byte or envelope equality check.
			if got, want := normalizeCatalogOrder(*cli.Result), normalizeCatalogOrder(*studio.Value); !reflect.DeepEqual(got, want) {
				t.Fatalf("CLI and Studio catalog meanings differ:\nCLI=%+v\nStudio=%+v", got, want)
			}
			if cli.Result.Effect != "read-only" || cli.Result.NamespaceFilter != tc.namespace {
				t.Fatalf("unexpected catalog semantics: %+v", cli.Result)
			}
			foundIndexedOrder := false
			foundApplication := false
			foundCustomerRelation := false
			var customerRelationEntries []operations.ModelCatalogEntry
			for _, entry := range cli.Result.Entries {
				if entry.Subject.Kind == "semantic-data" && entry.Subject.ID == "sales.order" {
					foundIndexedOrder = entry.SourceStatus == "indexed"
				}
				if entry.Subject.Kind == "semantic-relation" && entry.Subject.ID == "sales.order.customer" {
					customerRelationEntries = append(customerRelationEntries, entry)
					parent := sourceindex.Subject{Kind: "semantic-data", ID: "sales.order"}
					foundCustomerRelation = entry.Parent != nil && *entry.Parent == parent &&
						entry.Detail == "customer.customer" && entry.SourceStatus == "indexed" && !entry.Ambiguous
				}
				if entry.Subject.Kind == "application" && entry.Subject.ID == "order_entry" {
					foundApplication = true
				}
				if !tc.wantApps && (entry.Subject.Kind == "application" || entry.Subject.Kind == "application-pattern") {
					t.Fatalf("namespace-filtered catalog leaked workspace-level entry: %+v", entry)
				}
			}
			if !foundIndexedOrder {
				t.Fatal("sales.order must remain present with indexed source status")
			}
			if !foundCustomerRelation {
				t.Fatalf("sales.order.customer must retain its parent, target detail, indexed status, and unambiguous identity: %+v", customerRelationEntries)
			}
			if foundApplication != tc.wantApps {
				t.Fatalf("application inclusion = %t, want %t", foundApplication, tc.wantApps)
			}
		})
	}
}

func TestStudioCatalogMatchesCLIDuplicateIdentitySemantics(t *testing.T) {
	root := t.TempDir()
	writeWorkspaceFile(t, root, "tei.workspace.yaml", "workspace: catalog_parity\nsources:\n  - semantic\n")
	for _, path := range []string{"semantic/sales/first.tei", "semantic/sales/second.tei"} {
		writeWorkspaceFile(t, root, path, "namespace sales\n\ndata Order {\n}\n")
	}

	host, err := studiohost.NewReadOnly([]string{root})
	if err != nil {
		t.Fatal(err)
	}
	studio := host.CatalogWorkspace("workspace-1", "sales")
	if !studio.OK || studio.Value == nil || studio.Failure != nil {
		t.Fatalf("Studio catalog failed: %+v", studio)
	}

	var out bytes.Buffer
	exit, err := runModelCommand([]string{"catalog", "-workspace", root, "-namespace", "sales"}, &out)
	if err != nil || exit != 0 {
		t.Fatalf("CLI catalog failed: exit=%d err=%v body=%s", exit, err, out.String())
	}
	var cli modelCLIResponse
	if err := json.Unmarshal(out.Bytes(), &cli); err != nil {
		t.Fatal(err)
	}
	if !cli.OK || cli.Result == nil || cli.Error != nil {
		t.Fatalf("CLI catalog returned an invalid success envelope: %+v", cli)
	}
	if got, want := normalizeCatalogOrder(*cli.Result), normalizeCatalogOrder(*studio.Value); !reflect.DeepEqual(got, want) {
		t.Fatalf("CLI and Studio duplicate catalog meanings differ:\nCLI=%+v\nStudio=%+v", got, want)
	}

	duplicateCount := 0
	for _, entry := range cli.Result.Entries {
		if entry.Subject == (sourceindex.Subject{Kind: "semantic-data", ID: "sales.order"}) {
			duplicateCount++
			if !entry.Ambiguous {
				t.Fatalf("duplicate identity must be marked ambiguous: %+v", entry)
			}
		}
	}
	if duplicateCount != 2 {
		t.Fatalf("duplicate identity entries = %d, want both declarations", duplicateCount)
	}
}

func normalizeCatalogOrder(value operations.ModelCatalog) operations.ModelCatalog {
	value.Entries = append([]operations.ModelCatalogEntry(nil), value.Entries...)
	value.AuthorityLimits = append([]string(nil), value.AuthorityLimits...)
	sort.Strings(value.AuthorityLimits)
	sort.Slice(value.Entries, func(i, j int) bool {
		a, b := value.Entries[i], value.Entries[j]
		if a.Subject.Kind != b.Subject.Kind {
			return a.Subject.Kind < b.Subject.Kind
		}
		if a.Subject.ID != b.Subject.ID {
			return a.Subject.ID < b.Subject.ID
		}
		if (a.Parent == nil) != (b.Parent == nil) {
			return a.Parent == nil
		}
		if a.Parent != nil && b.Parent != nil {
			if a.Parent.Kind != b.Parent.Kind {
				return a.Parent.Kind < b.Parent.Kind
			}
			if a.Parent.ID != b.Parent.ID {
				return a.Parent.ID < b.Parent.ID
			}
		}
		if a.Symbol != b.Symbol {
			return a.Symbol < b.Symbol
		}
		if a.Detail != b.Detail {
			return a.Detail < b.Detail
		}
		if a.SourceStatus != b.SourceStatus {
			return a.SourceStatus < b.SourceStatus
		}
		return !a.Ambiguous && b.Ambiguous
	})
	return value
}

func TestStudioCatalogMatchesCLILoadFailureClassification(t *testing.T) {
	// An existing but empty directory is accepted into the Host allowlist; both
	// adapters then encounter the same Workspace load failure on catalog input.
	root := t.TempDir()
	host, err := studiohost.NewReadOnly([]string{root})
	if err != nil {
		t.Fatal(err)
	}
	studio := host.CatalogWorkspace("workspace-1", "")
	if studio.OK || studio.Value != nil || studio.Failure == nil || studio.Failure.Code != "WORKSPACE_LOAD_FAILED" || studio.Failure.Stage != string(operations.StageLoad) {
		t.Fatalf("expected Studio Workspace-load failure: %+v", studio)
	}

	var out bytes.Buffer
	exit, err := runModelCommand([]string{"catalog", "-workspace", root}, &out)
	if err != nil || exit != 2 {
		t.Fatalf("expected CLI Workspace-load exit 2: exit=%d err=%v body=%s", exit, err, out.String())
	}
	var cli modelCLIResponse
	if err := json.Unmarshal(out.Bytes(), &cli); err != nil {
		t.Fatal(err)
	}
	if cli.OK || cli.Result != nil || cli.Error == nil {
		t.Fatalf("expected CLI error envelope: %+v", cli)
	}
	if cli.Error.Code != studio.Failure.Code || cli.Error.Code != "WORKSPACE_LOAD_FAILED" {
		t.Fatalf("failure classes differ: CLI=%q Studio=%q", cli.Error.Code, studio.Failure.Code)
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
