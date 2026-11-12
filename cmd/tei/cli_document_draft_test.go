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

// TestModelDocumentReconcileAcceptsTheEnvelopeItPrints pins the round trip a
// caller actually performs: the command's own stdout is stored and read back.
// The envelope's schemaVersion names the CLI contract, not the draft contract,
// so the reader must open the envelope instead of refusing it.
func TestModelDocumentReconcileAcceptsTheEnvelopeItPrints(t *testing.T) {
	root := documentDraftWorkspace(t)
	var draftOut bytes.Buffer
	if code, err := runModelCommand([]string{
		"document-draft", "-workspace", root,
		"-subject-kind", "semantic-data", "-subject-id", "sales.order",
	}, &draftOut); err != nil || code != 0 {
		t.Fatalf("draft exit=%d err=%v body=%s", code, err, draftOut.String())
	}
	envelopePath := filepath.Join(t.TempDir(), "stored.json")
	if err := os.WriteFile(envelopePath, draftOut.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	code, err := runModelCommand([]string{
		"document-reconcile", "-workspace", root, "-draft", envelopePath,
	}, &out)
	if err != nil || code != 0 {
		t.Fatalf("exit=%d err=%v body=%s", code, err, out.String())
	}
	var envelope reconcileEnvelope
	if err := json.Unmarshal(out.Bytes(), &envelope); err != nil {
		t.Fatalf("not JSON: %v: %s", err, out.String())
	}
	if !envelope.OK || envelope.Result == nil ||
		envelope.Result.ModelReference.Status != documentdraft.ModelReferenceCurrent {
		t.Fatalf("envelope=%+v", envelope)
	}
	// A bare draft is still accepted: the file may have been trimmed by hand.
	barePath := filepath.Join(t.TempDir(), "bare.json")
	var bare draftEnvelope
	if err := json.Unmarshal(draftOut.Bytes(), &bare); err != nil {
		t.Fatal(err)
	}
	bareBytes, err := json.Marshal(bare.Result)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(barePath, bareBytes, 0o644); err != nil {
		t.Fatal(err)
	}
	var bareOut bytes.Buffer
	if code, err := runModelCommand([]string{
		"document-reconcile", "-workspace", root, "-draft", barePath,
	}, &bareOut); err != nil || code != 0 {
		t.Fatalf("bare exit=%d err=%v body=%s", code, err, bareOut.String())
	}
}

// documentDraftCRUDWorkspace is a synthetic Workspace with an application, a
// mapped data element and a capability effect, written under a temporary
// directory so the CRUD projection runs through the real loader.
func documentDraftCRUDWorkspace(t *testing.T) string {
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
	write("tei.workspace.yaml", "workspace: crud_cli_test\nformatVersion: 1\nsources:\n  - semantic\n  - applications\n  - databases\n")
	write("semantic/sales/crud.tei", `namespace sales

capability sales.order.create {
    input order sales.order
    output sales.order
    effect create sales.order
}
`)
	// The mapped data must be declared: the operation surface skips a mapping
	// whose semantic data the Workspace does not declare, and the CLI fixture
	// must show a real matrix rather than an empty one.
	write("semantic/sales/order.tei", `namespace sales

data Order {
    attribute orderId type std.text
    attribute customerId type std.text
}
`)
	write("applications/order-entry/application.tei", `application order_entry {
    uses semantic sales

    database sales {
        target sales_core
        access read-write
    }
}
`)
	write("databases/sales-core/database.tei", "database sales_core {\n    kind relational\n    ownership managed-here\n    uses semantic sales\n}\n")
	write("databases/sales-core/persistence/order.tei", `persist sales_core.order {
    database sales_core
    semantic sales.order
    table public.orders
    column sales.order.order_id -> order_id storage varchar required
    column sales.order.customer_id -> customer_id storage varchar required

    primary-key sales.order.order_id
}
`)
	return root
}

// TestModelDocumentDraftCRUDTableProjectsTheMatrix is the first-pass proof for
// the crud-table stage at the CLI boundary.
func TestModelDocumentDraftCRUDTableProjectsTheMatrix(t *testing.T) {
	root := documentDraftCRUDWorkspace(t)
	var out bytes.Buffer
	code, err := runModelCommand([]string{
		"document-draft", "-workspace", root,
		"-document-kind", documentdraft.DocumentKindCRUDTable,
		"-application", "order_entry",
	}, &out)
	if err != nil || code != 0 {
		t.Fatalf("exit=%d err=%v body=%s", code, err, out.String())
	}
	var envelope draftEnvelope
	if err := json.Unmarshal(out.Bytes(), &envelope); err != nil {
		t.Fatalf("not JSON: %v: %s", err, out.String())
	}
	draft := envelope.Result
	if draft == nil || draft.DocumentKind != documentdraft.DocumentKindCRUDTable {
		t.Fatalf("envelope=%+v", envelope)
	}
	if draft.Scope == nil || draft.Scope.Application != "order_entry" {
		t.Fatalf("scope=%+v", draft.Scope)
	}
	table, ok := occurrenceByIDForTest(draft, "occ-table-1")
	if !ok || table.Content.Table == nil || len(table.Content.Table.Rows) == 0 {
		t.Fatalf("matrix missing: %+v", draft.Occurrences)
	}
	if draft.Model.ModelRevision == "" || draft.Model.WorldFingerprint == "" {
		t.Fatalf("model reference=%+v", draft.Model)
	}
}

// occurrenceByIDForTest finds one occurrence in a draft returned by the CLI.
func occurrenceByIDForTest(draft *documentdraft.Draft, id string) (documentdraft.Occurrence, bool) {
	for _, occurrence := range draft.Occurrences {
		if occurrence.OccurrenceID == id {
			return occurrence, true
		}
	}
	return documentdraft.Occurrence{}, false
}

// TestModelDocumentDraftCRUDTableRefusesUnknownAndMixedInputs keeps the two
// input shapes apart at the CLI boundary too.
func TestModelDocumentDraftCRUDTableRefusesUnknownAndMixedInputs(t *testing.T) {
	root := documentDraftCRUDWorkspace(t)
	cases := []struct {
		name string
		args []string
		code string
		exit int
	}{
		{
			name: "unknown application",
			args: []string{"document-draft", "-workspace", root, "-document-kind", documentdraft.DocumentKindCRUDTable, "-application", "no_such_app"},
			code: documentdraft.CodeApplicationNotFound, exit: 1,
		},
		{
			name: "missing application",
			args: []string{"document-draft", "-workspace", root, "-document-kind", documentdraft.DocumentKindCRUDTable},
			code: "INVALID_ARGUMENT", exit: 2,
		},
		{
			name: "element selector with application",
			args: []string{"document-draft", "-workspace", root, "-document-kind", documentdraft.DocumentKindCRUDTable, "-application", "order_entry", "-subject-kind", "semantic-data", "-subject-id", "sales.order"},
			code: "INVALID_ARGUMENT", exit: 2,
		},
		{
			name: "element kind with an application flag",
			args: []string{"document-draft", "-workspace", root, "-subject-kind", "semantic-data", "-subject-id", "sales.order", "-application", "order_entry"},
			code: "INVALID_ARGUMENT", exit: 2,
		},
		{
			name: "unsupported document kind",
			args: []string{"document-draft", "-workspace", root, "-document-kind", "not-a-document-kind", "-subject-kind", "semantic-data", "-subject-id", "sales.order"},
			code: documentdraft.CodeUnsupportedDocKind, exit: 1,
		},
		{
			// The diagram kind is implemented, so requesting it without a model is
			// an argument failure rather than an unsupported kind.
			name: "diagram kind without a model",
			args: []string{"document-draft", "-workspace", root, "-document-kind", documentdraft.DocumentKindThreeElementDiagram, "-subject-kind", "semantic-data", "-subject-id", "sales.order"},
			code: documentdraft.CodeInvalidArgument, exit: 2,
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
		})
	}
}

// TestModelDocumentDraftFrozenDeliveryPinsTheRevision proves the frozen input at
// the CLI boundary: a pinned draft records the revision, and a later model
// change makes the delivery expire rather than offering a refreshed projection.
func TestModelDocumentDraftFrozenDeliveryPinsTheRevision(t *testing.T) {
	root := documentDraftCRUDWorkspace(t)
	var out bytes.Buffer
	code, err := runModelCommand([]string{
		"document-draft", "-workspace", root,
		"-document-kind", documentdraft.DocumentKindCRUDTable,
		"-application", "order_entry", "-frozen",
	}, &out)
	if err != nil || code != 0 {
		t.Fatalf("exit=%d err=%v body=%s", code, err, out.String())
	}
	var envelope draftEnvelope
	if err := json.Unmarshal(out.Bytes(), &envelope); err != nil {
		t.Fatalf("not JSON: %v: %s", err, out.String())
	}
	draft := envelope.Result
	if !draft.Frozen || draft.FrozenRevision != draft.Model.ModelRevision {
		t.Fatalf("pin=%+v", draft)
	}
	draftPath := filepath.Join(t.TempDir(), "frozen.json")
	stored, err := json.Marshal(draft)
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
	if current.Result.ModelReference.Status != documentdraft.ModelReferenceCurrent {
		t.Fatalf("model reference=%+v", current.Result.ModelReference)
	}

	// A model change expires the delivery: the report names the frozen pin.
	orderPath := filepath.Join(root, "databases", "sales-core", "persistence", "order.tei")
	body, err := os.ReadFile(orderPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(orderPath, []byte(strings.Replace(
		string(body), "table public.orders", "table public.orders_v2", 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	var expiredOut bytes.Buffer
	if code, err := runModelCommand([]string{
		"document-reconcile", "-workspace", root, "-draft", draftPath,
	}, &expiredOut); err != nil || code != 0 {
		t.Fatalf("reconcile exit=%d err=%v body=%s", code, err, expiredOut.String())
	}
	var expired reconcileEnvelope
	if err := json.Unmarshal(expiredOut.Bytes(), &expired); err != nil {
		t.Fatalf("not JSON: %v: %s", err, expiredOut.String())
	}
	if expired.Result.ModelReference.Status != documentdraft.ModelReferenceStale {
		t.Fatalf("model reference=%+v", expired.Result.ModelReference)
	}
	frozenDiff := false
	for _, diff := range expired.Result.ModelReference.Diffs {
		if diff == "frozenRevision" {
			frozenDiff = true
		}
	}
	if !frozenDiff {
		t.Fatalf("diffs=%v do not name the frozen pin", expired.Result.ModelReference.Diffs)
	}
	// The stored draft is untouched by the reconciliation.
	again, err := os.ReadFile(draftPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(again) != string(stored) {
		t.Fatal("reconcile rewrote the stored draft")
	}
}

// TestModelDocumentDraftFrozenDeliveryIsNotAKind pins that the pin is not a kind:
// naming it as one is refused rather than silently treated as a pin.
func TestModelDocumentDraftFrozenDeliveryIsNotAKind(t *testing.T) {
	root := documentDraftCRUDWorkspace(t)
	var out bytes.Buffer
	code, err := runModelCommand([]string{
		"document-draft", "-workspace", root,
		"-document-kind", documentdraft.DocumentKindFrozenDelivery,
		"-subject-kind", "semantic-data", "-subject-id", "sales.order",
	}, &out)
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if code != 1 {
		t.Fatalf("exit=%d want 1 body=%s", code, out.String())
	}
	var envelope draftEnvelope
	if err := json.Unmarshal(out.Bytes(), &envelope); err != nil {
		t.Fatalf("not JSON: %v: %s", err, out.String())
	}
	if envelope.OK || envelope.Error == nil ||
		envelope.Error.Code != documentdraft.CodeUnsupportedDocKind {
		t.Fatalf("envelope=%+v", envelope)
	}
}

// TestModelDocumentDraftSpecificationBundlesTheSections proves the specification
// stage at the CLI boundary: repeatable -section flags become chapters, and an
// undeclared subject is skipped with a reason instead of producing a chapter.
func TestModelDocumentDraftSpecificationBundlesTheSections(t *testing.T) {
	root := documentDraftCRUDWorkspace(t)
	var out bytes.Buffer
	code, err := runModelCommand([]string{
		"document-draft", "-workspace", root,
		"-document-kind", documentdraft.DocumentKindSpecification,
		"-section", "semantic-data:sales.order",
		"-section", "semantic-data:sales.nothing",
	}, &out)
	if err != nil || code != 0 {
		t.Fatalf("exit=%d err=%v body=%s", code, err, out.String())
	}
	var envelope draftEnvelope
	if err := json.Unmarshal(out.Bytes(), &envelope); err != nil {
		t.Fatalf("not JSON: %v: %s", err, out.String())
	}
	draft := envelope.Result
	if draft == nil || draft.DocumentKind != documentdraft.DocumentKindSpecification {
		t.Fatalf("envelope=%+v", envelope)
	}
	if draft.Scope == nil || len(draft.Scope.SkippedSubjects) != 1 {
		t.Fatalf("scope=%+v", draft.Scope)
	}
	if draft.Scope.SkippedSubjects[0].Subject.ID != "sales.nothing" {
		t.Fatalf("skipped=%+v", draft.Scope.SkippedSubjects[0])
	}
	chapters := 0
	for _, occurrence := range draft.Occurrences {
		if occurrence.Kind == "explanation" {
			chapters++
		}
	}
	if chapters != 1 {
		t.Fatalf("chapter count=%d want 1: %+v", chapters, draft.Occurrences)
	}
}

// TestModelDocumentDraftSpecificationSectionGrammar pins the -section grammar: a
// malformed value is an argument failure, not a section the caller did not ask
// for.
func TestModelDocumentDraftSpecificationSectionGrammar(t *testing.T) {
	root := documentDraftCRUDWorkspace(t)
	for _, section := range []string{"", "no-colon", ":sales.order", "semantic-data:", "semantic-data:sales.order@"} {
		var out bytes.Buffer
		code, err := runModelCommand([]string{
			"document-draft", "-workspace", root,
			"-document-kind", documentdraft.DocumentKindSpecification,
			"-section", section,
		}, &out)
		if err != nil {
			t.Fatalf("err=%v", err)
		}
		if code != 2 {
			t.Fatalf("section %q exit=%d want 2 body=%s", section, code, out.String())
		}
	}
}

// TestModelDocumentDraftSpecificationRejectsMixedInputs keeps the three input
// shapes apart at the CLI boundary too.
func TestModelDocumentDraftSpecificationRejectsMixedInputs(t *testing.T) {
	root := documentDraftCRUDWorkspace(t)
	cases := []struct {
		name string
		args []string
	}{
		{
			name: "no sections",
			args: []string{"document-draft", "-workspace", root, "-document-kind", documentdraft.DocumentKindSpecification},
		},
		{
			name: "element selector",
			args: []string{"document-draft", "-workspace", root, "-document-kind", documentdraft.DocumentKindSpecification,
				"-subject-kind", "semantic-data", "-subject-id", "sales.order", "-section", "semantic-data:sales.order"},
		},
		{
			name: "application",
			args: []string{"document-draft", "-workspace", root, "-document-kind", documentdraft.DocumentKindSpecification,
				"-application", "order_entry", "-section", "semantic-data:sales.order"},
		},
		{
			name: "top-level relation",
			args: []string{"document-draft", "-workspace", root, "-document-kind", documentdraft.DocumentKindSpecification,
				"-relation-id", "sales.order.customer", "-section", "semantic-data:sales.order"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer
			code, err := runModelCommand(tc.args, &out)
			if err != nil {
				t.Fatalf("err=%v", err)
			}
			if code != 2 {
				t.Fatalf("exit=%d want 2 body=%s", code, out.String())
			}
			var envelope draftEnvelope
			if err := json.Unmarshal(out.Bytes(), &envelope); err != nil {
				t.Fatalf("not JSON: %v: %s", err, out.String())
			}
			if envelope.OK || envelope.Error == nil || envelope.Error.Code != "INVALID_ARGUMENT" {
				t.Fatalf("envelope=%+v", envelope)
			}
		})
	}
}

// TestModelDocumentDraftSpecificationReconcileReportsTheMovedChapters runs the
// specification through the CLI reconcile: an unchanged model reports current,
// and a changed one reports the chapters that moved.
func TestModelDocumentDraftSpecificationReconcileReportsTheMovedChapters(t *testing.T) {
	root := documentDraftCRUDWorkspace(t)
	var draftOut bytes.Buffer
	if code, err := runModelCommand([]string{
		"document-draft", "-workspace", root,
		"-document-kind", documentdraft.DocumentKindSpecification,
		"-section", "semantic-data:sales.order",
		"-section", "semantic-data:customer.customer",
	}, &draftOut); err != nil || code != 0 {
		t.Fatalf("draft exit=%d err=%v body=%s", code, err, draftOut.String())
	}
	var envelope draftEnvelope
	if err := json.Unmarshal(draftOut.Bytes(), &envelope); err != nil {
		t.Fatalf("not JSON: %v", err)
	}
	draftPath := filepath.Join(t.TempDir(), "spec.json")
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
	if !current.OK || current.Result.ModelReference.Status != documentdraft.ModelReferenceCurrent {
		t.Fatalf("envelope=%+v", current)
	}

	// A change to one element moves its chapter's content.
	orderPath := filepath.Join(root, "semantic", "sales", "order.tei")
	body, err := os.ReadFile(orderPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(orderPath, []byte(strings.Replace(
		string(body), "    attribute customerId type std.text",
		"    attribute customerId type std.text\n    attribute placedAt type std.datetime", 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	if string(body) == string(mustRead(t, orderPath)) {
		t.Fatal("fixture did not change; the test proves nothing")
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
	contentChanged := 0
	for _, occurrence := range changed.Result.Occurrences {
		if occurrence.Status == "content-changed" {
			contentChanged++
		}
	}
	if contentChanged == 0 {
		t.Fatalf("no chapter reported a content change: %+v", changed.Result.Occurrences)
	}
	// The stored draft is untouched by the reconciliation.
	again, err := os.ReadFile(draftPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(again) != string(stored) {
		t.Fatal("reconcile rewrote the stored draft")
	}
}

// mustRead reads one file or fails the test. It exists so a fixture change can
// be proven rather than assumed.
func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return body
}

// TestModelDocumentDraftThreeElementDiagramProjectsTheModel proves the diagram
// kind at the CLI boundary: a three-element analysis model becomes one diagram
// occurrence, and an unresolved edge survives it.
func TestModelDocumentDraftThreeElementDiagramProjectsTheModel(t *testing.T) {
	modelPath := writeThreeElementModelFixture(t)
	var out bytes.Buffer
	code, err := runModelCommand([]string{
		"document-draft", "-document-kind", documentdraft.DocumentKindThreeElementDiagram,
		"-model", modelPath,
	}, &out)
	if err != nil || code != 0 {
		t.Fatalf("exit=%d err=%v body=%s", code, err, out.String())
	}
	var envelope draftEnvelope
	if err := json.Unmarshal(out.Bytes(), &envelope); err != nil {
		t.Fatalf("not JSON: %v: %s", err, out.String())
	}
	draft := envelope.Result
	if draft == nil || draft.DocumentKind != documentdraft.DocumentKindThreeElementDiagram {
		t.Fatalf("envelope=%+v", envelope)
	}
	// The diagram reads no Workspace: none is recorded.
	if draft.Model.Workspace != nil || draft.Model.WorldFingerprint != "" {
		t.Fatalf("model=%+v carries a Workspace read", draft.Model)
	}
	if len(draft.Occurrences) != 1 || draft.Occurrences[0].Kind != "diagram" {
		t.Fatalf("occurrences=%+v", draft.Occurrences)
	}
	diagram := draft.Occurrences[0].Content.Diagram
	if diagram == nil {
		t.Fatal("the diagram occurrence carries no diagram content")
	}
	if len(diagram.Nodes) != 3 {
		t.Fatalf("nodes=%+v", diagram.Nodes)
	}
	if len(diagram.Edges) != 2 {
		t.Fatalf("edges=%+v", diagram.Edges)
	}
	if len(diagram.UnresolvedEdges) != 1 {
		t.Fatalf("unresolvedEdges=%v", diagram.UnresolvedEdges)
	}
	foundLimit := false
	for _, limit := range draft.AuthorityLimits {
		if limit == "diagram-element-mapping-is-model-declared-not-tei-decided" {
			foundLimit = true
		}
	}
	if !foundLimit {
		t.Fatalf("authorityLimits=%v", draft.AuthorityLimits)
	}
}

// TestModelDocumentDraftThreeElementDiagramReconcileReportsAMovedModel runs the
// diagram through the CLI reconcile: an unchanged model reports current, a
// changed one reports the diagram as moved.
func TestModelDocumentDraftThreeElementDiagramReconcileReportsAMovedModel(t *testing.T) {
	modelPath := writeThreeElementModelFixture(t)
	var draftOut bytes.Buffer
	if code, err := runModelCommand([]string{
		"document-draft", "-document-kind", documentdraft.DocumentKindThreeElementDiagram,
		"-model", modelPath,
	}, &draftOut); err != nil || code != 0 {
		t.Fatalf("draft exit=%d err=%v body=%s", code, err, draftOut.String())
	}
	var envelope draftEnvelope
	if err := json.Unmarshal(draftOut.Bytes(), &envelope); err != nil {
		t.Fatalf("not JSON: %v", err)
	}
	draftPath := filepath.Join(t.TempDir(), "diagram.json")
	stored, err := json.Marshal(envelope.Result)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(draftPath, stored, 0o644); err != nil {
		t.Fatal(err)
	}
	var currentOut bytes.Buffer
	if code, err := runModelCommand([]string{
		"document-reconcile", "-draft", draftPath,
	}, &currentOut); err != nil || code != 0 {
		t.Fatalf("reconcile exit=%d err=%v body=%s", code, err, currentOut.String())
	}
	var current reconcileEnvelope
	if err := json.Unmarshal(currentOut.Bytes(), &current); err != nil {
		t.Fatalf("not JSON: %v: %s", err, currentOut.String())
	}
	if !current.OK || current.Result.ModelReference.Status != documentdraft.ModelReferenceCurrent {
		t.Fatalf("envelope=%+v", current)
	}

	// One attribute is added to the model: the diagram moves.
	raw, err := os.ReadFile(modelPath)
	if err != nil {
		t.Fatal(err)
	}
	var model map[string]any
	if err := json.Unmarshal(raw, &model); err != nil {
		t.Fatal(err)
	}
	data := model["dataElements"].([]any)
	first := data[0].(map[string]any)
	attributes := first["attributes"].([]any)
	first["attributes"] = append(attributes, map[string]any{
		"id": "sales.order.placed_at", "name": "Placed at", "type": "std.datetime",
	})
	model["dataElements"] = data
	changed, err := json.Marshal(model)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(modelPath, changed, 0o644); err != nil {
		t.Fatal(err)
	}
	var movedOut bytes.Buffer
	if code, err := runModelCommand([]string{
		"document-reconcile", "-draft", draftPath,
	}, &movedOut); err != nil || code != 0 {
		t.Fatalf("reconcile exit=%d err=%v body=%s", code, err, movedOut.String())
	}
	var moved reconcileEnvelope
	if err := json.Unmarshal(movedOut.Bytes(), &moved); err != nil {
		t.Fatalf("not JSON: %v: %s", err, movedOut.String())
	}
	if moved.Result.ModelReference.Status != documentdraft.ModelReferenceStale {
		t.Fatalf("model reference=%+v", moved.Result.ModelReference)
	}
	if len(moved.Result.NeedsReconfirmation) != 1 {
		t.Fatalf("needsReconfirmation=%v", moved.Result.NeedsReconfirmation)
	}
	// The stored draft is untouched by the reconciliation.
	again, err := os.ReadFile(draftPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(again) != string(stored) {
		t.Fatal("reconcile rewrote the stored draft")
	}
}

// writeThreeElementModelFixture writes one synthetic three-element analysis model
// and returns its path. It is a separate artifact from a TEI Workspace, which is
// exactly the boundary this kind keeps.
func writeThreeElementModelFixture(t *testing.T) string {
	t.Helper()
	model := map[string]any{
		"schemaVersion": "tei.three-element/v0",
		"effect":        "read-only",
		"modelId":       "cli-diagram-test",
		"name":          "CLI diagram test",
		"dataElements": []map[string]any{
			{"id": "sales.order", "name": "Order", "attributes": []map[string]any{
				{"id": "sales.order.order_id", "name": "Order ID", "type": "std.identifier", "key": true},
			}},
			{"id": "customer.customer", "name": "Customer"},
		},
		"functionElements": []map[string]any{
			{"id": "sales.order.submit", "name": "Submit order", "operations": []map[string]any{
				{"dataId": "sales.order", "operations": "C"},
			}},
		},
		"crossReferences": []map[string]any{
			{"id": "xref-fn-data", "kind": "function-to-data",
				"fromId": "sales.order.submit", "fromKind": "function",
				"toId": "sales.order", "toKind": "data", "directed": true},
			{"id": "xref-data-missing", "kind": "data-to-data",
				"fromId": "sales.order", "fromKind": "data",
				"toId": "sales.nothing", "toKind": "data", "directed": true},
		},
	}
	encoded, err := json.Marshal(model)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "model.json")
	if err := os.WriteFile(path, encoded, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// TestModelDocumentDraftElementKindStillRequiresTheSubject pins that the CRUD
// input relaxation did not weaken the element path.
func TestModelDocumentDraftElementKindStillRequiresTheSubject(t *testing.T) {
	root := documentDraftCRUDWorkspace(t)
	for _, args := range [][]string{
		{"document-draft", "-workspace", root},
		{"document-draft", "-workspace", root, "-subject-id", "sales.order"},
	} {
		var out bytes.Buffer
		code, err := runModelCommand(args, &out)
		if err != nil || code != 2 {
			t.Fatalf("argv=%v exit=%d err=%v body=%s", args, code, err, out.String())
		}
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
