package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/documentdraft"
)

// documentDraftWorkspace is a synthetic Workspace with no customer data. It is
// written under a temporary directory so the projection can be exercised end to
// end through the CLI.
func documentDraftWorkspace(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	write := func(relative, body string) {
		path := filepath.Join(root, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("tei.workspace.yaml", "workspace: document_draft_cli_test\nformatVersion: 1\nsources:\n  - semantic\n")
	write("semantic/sales/order.tei", `namespace sales

data Order {
    attribute orderId type std.text
    attribute customerId type std.text
    relation customer to customer.customer one
}
`)
	write("semantic/customer/customer.tei", `namespace customer

data Customer {
    attribute customerId type std.text
}
`)
	return root
}

type draftEnvelope struct {
	SchemaVersion string                `json:"schemaVersion"`
	OK            bool                  `json:"ok"`
	Result        *documentdraft.Draft  `json:"result"`
	Error         *cliInspectionFailure `json:"error"`
}

type reconcileEnvelope struct {
	SchemaVersion string                         `json:"schemaVersion"`
	OK            bool                           `json:"ok"`
	Result        *documentdraft.ReconcileReport `json:"result"`
	Error         *cliInspectionFailure          `json:"error"`
}

// TestModelDocumentDraftProjectsOneElementAndRelation is the first-pass proof at
// the CLI boundary: the command answers the draft contract with the model
// reference, the relation and the occurrences.
func TestModelDocumentDraftProjectsOneElementAndRelation(t *testing.T) {
	root := documentDraftWorkspace(t)
	var out bytes.Buffer
	code, err := runModelCommand([]string{
		"document-draft", "-workspace", root,
		"-subject-kind", "semantic-data", "-subject-id", "sales.order",
		"-relation-id", "sales.order.customer",
	}, &out)
	if err != nil || code != 0 {
		t.Fatalf("exit=%d err=%v body=%s", code, err, out.String())
	}
	var envelope draftEnvelope
	if err := json.Unmarshal(out.Bytes(), &envelope); err != nil {
		t.Fatalf("not JSON: %v: %s", err, out.String())
	}
	if envelope.SchemaVersion != documentDraftSchemaVersion || !envelope.OK {
		t.Fatalf("envelope=%+v", envelope)
	}
	if envelope.Result == nil {
		t.Fatalf("no result: %s", out.String())
	}
	draft := envelope.Result
	if draft.SchemaVersion != documentdraft.SchemaVersion ||
		draft.Effect != documentdraft.EffectReadOnlyProjection {
		t.Fatalf("draft header=%+v", draft)
	}
	if len(draft.Relations) != 1 || draft.Relations[0].Resolution != documentdraft.ResolutionResolved {
		t.Fatalf("relations=%+v", draft.Relations)
	}
	if draft.Model.ModelRevision == "" || draft.Model.WorldFingerprint == "" {
		t.Fatalf("model reference=%+v", draft.Model)
	}
	if len(draft.Occurrences) == 0 {
		t.Fatal("draft has no occurrence")
	}
	for _, limit := range draft.AuthorityLimits {
		if limit == "consumer-save-is-not-tei-model-ingestion" {
			return
		}
	}
	t.Fatalf("authorityLimits=%v", draft.AuthorityLimits)
}

// TestModelDocumentDraftReportsArgumentAndDomainFailures pins the exit codes and
// the codes the envelope reports, so a caller can tell a typo from a subject
// that is not there.
func TestModelDocumentDraftReportsArgumentAndDomainFailures(t *testing.T) {
	root := documentDraftWorkspace(t)
	cases := []struct {
		name string
		args []string
		code string
		exit int
	}{
		{
			name: "missing workspace",
			args: []string{"document-draft", "-subject-kind", "semantic-data", "-subject-id", "sales.order"},
			code: "INVALID_ARGUMENT", exit: 2,
		},
		{
			name: "unknown subject",
			args: []string{"document-draft", "-workspace", root, "-subject-kind", "semantic-data", "-subject-id", "sales.nothing"},
			code: documentdraft.CodeSubjectNotFound, exit: 1,
		},
		{
			name: "unsupported kind",
			args: []string{"document-draft", "-workspace", root, "-subject-kind", "application", "-subject-id", "order_entry"},
			code: documentdraft.CodeUnsupportedKind, exit: 1,
		},
		{
			name: "relation of another element",
			args: []string{"document-draft", "-workspace", root, "-subject-kind", "semantic-data", "-subject-id", "customer.customer", "-relation-id", "sales.order.customer"},
			code: documentdraft.CodeRelationNotFound, exit: 1,
		},
		{
			name: "empty relation id",
			args: []string{"document-draft", "-workspace", root, "-subject-kind", "semantic-data", "-subject-id", "sales.order", "-relation-id="},
			code: "INVALID_ARGUMENT", exit: 2,
		},
		{
			name: "repeated workspace",
			args: []string{"document-draft", "-workspace", root, "-workspace", root, "-subject-kind", "semantic-data", "-subject-id", "sales.order"},
			code: "INVALID_ARGUMENT", exit: 2,
		},
		{
			name: "positional argument",
			args: []string{"document-draft", "-workspace", root, "-subject-kind", "semantic-data", "-subject-id", "sales.order", "extra"},
			code: "INVALID_ARGUMENT", exit: 2,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer
			code, err := runModelCommand(tc.args, &out)
			if err != nil {
				t.Fatalf("err=%v", err)
			}
			if code != tc.exit {
				t.Fatalf("exit=%d want %d body=%s", code, tc.exit, out.String())
			}
			var envelope draftEnvelope
			if err := json.Unmarshal(out.Bytes(), &envelope); err != nil {
				t.Fatalf("not JSON: %v: %s", err, out.String())
			}
			if envelope.OK || envelope.Error == nil || envelope.Error.Code != tc.code {
				t.Fatalf("envelope=%+v", envelope)
			}
			if envelope.Result != nil {
				t.Fatalf("failure carried a result: %+v", envelope.Result)
			}
		})
	}
}

// TestModelDocumentCommandsAssertTheirAdvertisedSchemaVersions keeps the
// discovery record and the response in one place. The literals appear here on
// purpose: schema_version_coverage_test.go requires a test that asserts each
// advertised schemaVersion.
func TestModelDocumentCommandsAssertTheirAdvertisedSchemaVersions(t *testing.T) {
	if documentDraftSchemaVersion != "tei.cli.model-document-draft/v0" {
		t.Fatalf("draft schemaVersion=%q", documentDraftSchemaVersion)
	}
	if documentReconcileSchemaVersion != "tei.cli.model-document-reconcile/v0" {
		t.Fatalf("reconcile schemaVersion=%q", documentReconcileSchemaVersion)
	}
	advertised := map[string]string{}
	for _, record := range commandCatalog().Commands {
		advertised[record.Name] = record.SchemaVersion
	}
	if advertised["model document-draft"] != documentDraftSchemaVersion {
		t.Fatalf("discovery advertises %q for model document-draft", advertised["model document-draft"])
	}
	if advertised["model document-reconcile"] != documentReconcileSchemaVersion {
		t.Fatalf("discovery advertises %q for model document-reconcile", advertised["model document-reconcile"])
	}
}

// TestModelDocumentReconcileReportsAChangedModel runs the two commands together:
// the draft is produced, the Source changes, and the reconciliation names what a
// person must reconfirm. The stored draft is not rewritten.
func TestModelDocumentReconcileReportsAChangedModel(t *testing.T) {
	root := documentDraftWorkspace(t)
	var draftOut bytes.Buffer
	if code, err := runModelCommand([]string{
		"document-draft", "-workspace", root,
		"-subject-kind", "semantic-data", "-subject-id", "sales.order",
		"-relation-id", "sales.order.customer",
	}, &draftOut); err != nil || code != 0 {
		t.Fatalf("draft exit=%d err=%v body=%s", code, err, draftOut.String())
	}
	var envelope draftEnvelope
	if err := json.Unmarshal(draftOut.Bytes(), &envelope); err != nil {
		t.Fatalf("not JSON: %v", err)
	}
	draftPath := filepath.Join(t.TempDir(), "draft.json")
	stored, err := json.Marshal(envelope.Result)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(draftPath, stored, 0o644); err != nil {
		t.Fatal(err)
	}

	var currentOut bytes.Buffer
	if code, err := runModelCommand([]string{
		"document-reconcile", "-workspace", root, "-draft", draftPath,
	}, &currentOut); err != nil || code != 0 {
		t.Fatalf("reconcile exit=%d err=%v body=%s", code, err, currentOut.String())
	}
	var current reconcileEnvelope
	if err := json.Unmarshal(currentOut.Bytes(), &current); err != nil {
		t.Fatalf("not JSON: %v: %s", err, currentOut.String())
	}
	if current.SchemaVersion != documentReconcileSchemaVersion || !current.OK {
		t.Fatalf("envelope=%+v", current)
	}
	if current.Result.ModelReference.Status != documentdraft.ModelReferenceCurrent {
		t.Fatalf("model reference=%+v", current.Result.ModelReference)
	}
	if len(current.Result.NeedsReconfirmation) != 0 {
		t.Fatalf("needsReconfirmation=%v", current.Result.NeedsReconfirmation)
	}

	orderPath := filepath.Join(root, "semantic", "sales", "order.tei")
	if err := os.WriteFile(orderPath, []byte(`namespace sales

data Order {
    attribute orderId type std.text
    attribute customerId type std.text
    attribute placedAt type std.datetime
    relation customer to customer.customer one
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	var changedOut bytes.Buffer
	if code, err := runModelCommand([]string{
		"document-reconcile", "-workspace", root, "-draft", draftPath,
	}, &changedOut); err != nil || code != 0 {
		t.Fatalf("reconcile exit=%d err=%v body=%s", code, err, changedOut.String())
	}
	var changed reconcileEnvelope
	if err := json.Unmarshal(changedOut.Bytes(), &changed); err != nil {
		t.Fatalf("not JSON: %v: %s", err, changedOut.String())
	}
	if changed.Result.ModelReference.Status != documentdraft.ModelReferenceStale {
		t.Fatalf("model reference=%+v", changed.Result.ModelReference)
	}
	if len(changed.Result.NeedsReconfirmation) == 0 {
		t.Fatal("a changed model produced nothing to reconfirm")
	}
	// The stored draft file is untouched by the reconciliation.
	again, err := os.ReadFile(draftPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(again) != string(stored) {
		t.Fatal("reconcile rewrote the stored draft")
	}
}

// TestModelDocumentReconcileRefusesADraftThatIsNotThisContract keeps a foreign
// or hand-edited file out of the comparison instead of comparing part of it.
func TestModelDocumentReconcileRefusesADraftThatIsNotThisContract(t *testing.T) {
	root := documentDraftWorkspace(t)
	draftPath := filepath.Join(t.TempDir(), "foreign.json")
	if err := os.WriteFile(draftPath, []byte(`{"schemaVersion":"some.other.contract/v1","effect":"read-only-projection"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	code, err := runModelCommand([]string{
		"document-reconcile", "-workspace", root, "-draft", draftPath,
	}, &out)
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if code != 1 {
		t.Fatalf("exit=%d want 1 body=%s", code, out.String())
	}
	var envelope reconcileEnvelope
	if err := json.Unmarshal(out.Bytes(), &envelope); err != nil {
		t.Fatalf("not JSON: %v: %s", err, out.String())
	}
	if envelope.OK || envelope.Error == nil || envelope.Error.Code != documentdraft.CodeInvalidDraft {
		t.Fatalf("envelope=%+v", envelope)
	}
}

// TestModelDocumentReconcileReportsAnUnreadableDraft keeps a missing or broken
// file distinct from a draft that does not match the contract.
func TestModelDocumentReconcileReportsAnUnreadableDraft(t *testing.T) {
	root := documentDraftWorkspace(t)
	missing := filepath.Join(t.TempDir(), "missing.json")
	var out bytes.Buffer
	code, err := runModelCommand([]string{
		"document-reconcile", "-workspace", root, "-draft", missing,
	}, &out)
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if code != 2 {
		t.Fatalf("exit=%d want 2 body=%s", code, out.String())
	}
	var envelope reconcileEnvelope
	if err := json.Unmarshal(out.Bytes(), &envelope); err != nil {
		t.Fatalf("not JSON: %v: %s", err, out.String())
	}
	if envelope.OK || envelope.Error == nil || envelope.Error.Code != "DRAFT_READ_FAILED" {
		t.Fatalf("envelope=%+v", envelope)
	}

	broken := filepath.Join(t.TempDir(), "broken.json")
	if err := os.WriteFile(broken, []byte("{ not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	var brokenOut bytes.Buffer
	if code, err := runModelCommand([]string{
		"document-reconcile", "-workspace", root, "-draft", broken,
	}, &brokenOut); err != nil || code != 2 {
		t.Fatalf("exit=%d err=%v body=%s", code, err, brokenOut.String())
	}
	var brokenEnvelope reconcileEnvelope
	if err := json.Unmarshal(brokenOut.Bytes(), &brokenEnvelope); err != nil {
		t.Fatalf("not JSON: %v: %s", err, brokenOut.String())
	}
	if brokenEnvelope.Error == nil || brokenEnvelope.Error.Code != "DRAFT_DECODE_FAILED" {
		t.Fatalf("envelope=%+v", brokenEnvelope)
	}
}

// TestModelDocumentCommandsRejectUnknownSubcommands keeps the dispatch honest:
// a misspelled subcommand is an argument failure, not a catalog read.
func TestModelDocumentCommandsRejectUnknownSubcommands(t *testing.T) {
	var out bytes.Buffer
	code, err := runModelCommand([]string{"document-draftt", "-workspace", "x"}, &out)
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if code != 2 {
		t.Fatalf("exit=%d want 2", code)
	}
	if !strings.Contains(out.String(), "document-draft") {
		t.Fatalf("body=%s", out.String())
	}
	var envelope draftEnvelope
	if err := json.Unmarshal(out.Bytes(), &envelope); err != nil {
		t.Fatalf("not JSON: %v: %s", err, out.String())
	}
	if envelope.Error == nil || envelope.Error.Code != "INVALID_ARGUMENT" {
		t.Fatalf("envelope=%+v", envelope)
	}
}
