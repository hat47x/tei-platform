package languagereference

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/workspace"
)

// The expected stops are authored separately from the selected TEI APIs.
// The focused tests exercise BS-02/03/06 and BS-14's current-witness
// boundary. The remaining bootstrap/recovery cases are design-level only;
// no passing Go test implies formal Owner acceptance or full K0-K6 coverage.
func bootstrapExpectedStops(t *testing.T) map[string]string {
	t.Helper()
	path := filepath.Join("..", "..", "plan", "fixtures", "role-bootstrap",
		"bootstrap_order_recovery_expected_20261106.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Status string `json:"status"`
		Cases  []struct {
			ID       string `json:"id"`
			Expected struct {
				EarliestStop string `json:"earliestStop"`
			} `json:"expected"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatal(err)
	}
	if fixture.Status != "INDEPENDENT_EXPECTED_NOT_OWNER_ACCEPTED" {
		t.Fatalf("expected independent Expected fixture status, got %q", fixture.Status)
	}
	result := make(map[string]string, len(fixture.Cases))
	for _, c := range fixture.Cases {
		if c.ID == "" || c.Expected.EarliestStop == "" {
			t.Fatalf("fixture missing independently authored case ID/expected stop: %+v", c)
		}
		if _, exists := result[c.ID]; exists {
			t.Fatalf("duplicate fixture case ID: %s", c.ID)
		}
		result[c.ID] = c.Expected.EarliestStop
	}
	if len(result) != 15 {
		t.Fatalf("unexpected bootstrap fixture case count: %d", len(result))
	}
	return result
}

func TestBootstrapBS02ActualWorkspaceLoaderRejectsUnsupportedManifest(t *testing.T) {
	expected := bootstrapExpectedStops(t)
	if expected["BS-02"] != "seed-manifest-version" {
		t.Fatalf("independent BS-02 expected stop drifted: %q", expected["BS-02"])
	}
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "semantic"), 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := []byte("workspace: bs02\nformatVersion: 99\nsources:\n  - semantic\n")
	if err := os.WriteFile(filepath.Join(root, "tei.workspace.yaml"), manifest, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "semantic", "data.tei"),
		[]byte("namespace probe\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := workspace.Load(root)
	if err == nil || !strings.Contains(err.Error(), "unsupported workspace formatVersion 99") {
		t.Fatalf("BS-02 trusted manifest reader accepted unsupported version: %v", err)
	}
	// The same raw Source with an explicitly supported manifest version
	// must still Load. Otherwise an always-rejecting loader would pass BS-02.
	supported := []byte("workspace: bs02\nformatVersion: 1\nsources:\n  - semantic\n")
	if err := os.WriteFile(filepath.Join(root, "tei.workspace.yaml"), supported, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := workspace.Load(root); err != nil {
		t.Fatalf("BS-02 positive control rejected supported manifest and Source: %v", err)
	}
	// This is a Load-level test. It proves neither a production Language
	// Plugin registry nor a particular downstream parser invocation.
}

func TestBootstrapBS03ActualLanguageSelectionRejectsChangedImplementationPin(t *testing.T) {
	expected := bootstrapExpectedStops(t)
	if expected["BS-03"] != "language-exact-pin" {
		t.Fatalf("independent BS-03 expected stop drifted: %q", expected["BS-03"])
	}
	pin := Selection{
		LanguageID: "lang", ContractVersion: "v1", ContractFingerprint: "contract-A",
		ImplementationID: "impl-A", ImplementationFingerprint: "dA",
	}
	offered := pin
	offered.ImplementationFingerprint = "dB"
	parsed := false
	parse := func(src string) (string, error) {
		parsed = true
		return "candidate:" + src, nil
	}
	_, err := ParseSelected(pin, []Provider[string]{{
		Selection: offered,
		Parse:     parse,
	}}, "data source")
	if !errors.Is(err, ErrProviderMissing) || parsed {
		t.Fatalf("BS-03 parser bypassed the exact implementation pin: parsed=%v, err=%v", parsed, err)
	}
	// Keep the Source and the callback identical, but offer the exact
	// selection. An always-rejecting resolver must fail this positive control.
	parsed = false
	value, err := ParseSelected(pin, []Provider[string]{{
		Selection: pin,
		Parse:     parse,
	}}, "data source")
	if err != nil || !parsed || value != "candidate:data source" {
		t.Fatalf("BS-03 positive control did not select exact parser: parsed=%v, value=%q, err=%v", parsed, value, err)
	}
	// The fingerprint tokens above are fixture labels. This reference does
	// not hash or attest the actual implementation executable bytes.
}

func TestBootstrapBS06ActualOperatorLoweringRequiresExplicitPin(t *testing.T) {
	expected := bootstrapExpectedStops(t)
	if expected["BS-06"] != "operator-exact-contract" {
		t.Fatalf("independent BS-06 expected stop drifted: %q", expected["BS-06"])
	}
	source := "rows/order |> sum"
	syntax, err := ParsePipelineSyntax(source)
	if err != nil {
		t.Fatal(err)
	}
	contract := OperatorContract{
		ID: "sum", Version: "v2", Fingerprint: "contract-sum-v2",
		Effect: "pure", InputType: "rows/order", OutputType: "amount",
	}
	steps, err := LowerPinnedPipeline(syntax, nil, []OperatorContract{contract})
	var diagnostic *PipelineDiagnostic
	if len(steps) != 0 || !errors.As(err, &diagnostic) {
		t.Fatalf("BS-06 unpinned operator escaped candidate boundary: steps=%+v, err=%v", steps, err)
	}
	if diagnostic.Step != 0 ||
		diagnostic.Span != syntax.Operators[0].Span ||
		source[diagnostic.Span.Start:diagnostic.Span.End] != "sum" {
		t.Fatalf("BS-06 wrong source-bound operator diagnostic: %+v", diagnostic)
	}
	// Use the same parsed Source with the explicit, matching pin. An
	// always-failing lowering implementation must fail this positive control.
	steps, err = LowerPinnedPipeline(syntax, []OperatorContract{contract}, []OperatorContract{contract})
	if err != nil || len(steps) != 1 {
		t.Fatalf("BS-06 positive control rejected exact pinned operator: steps=%+v, err=%v", steps, err)
	}
	if steps[0].Contract != contract || steps[0].Span != syntax.Operators[0].Span {
		t.Fatalf("BS-06 positive control lost original contract or Source span: %+v", steps[0])
	}
	// The resulting candidate is not proof of product Operator registration,
	// accepted meaning, or active generation.
}

// This probe joins the already-tested K2 Language selection and K4 Operator
// lowering for one identical Source. It is still reference-only and does not
// bind a production Workspace Source, real parser executable bytes, Owner
// meaning, or the selected runtime artifact.
func TestBootstrapSameSourcePinnedLanguageThenOperator(t *testing.T) {
	expected := bootstrapExpectedStops(t)
	if expected["BS-03"] != "language-exact-pin" ||
		expected["BS-06"] != "operator-exact-contract" {
		t.Fatal("independent BS-03/BS-06 expected stop has changed")
	}
	source := "rows/order |> std.sum"
	language := Selection{
		LanguageID: "dsl.pipeline", ContractVersion: "1",
		ContractFingerprint: "contract-pipeline-v1",
		ImplementationID:    "reference-parser-A", ImplementationFingerprint: "impl-A",
	}
	operator := OperatorContract{
		ID: "std.sum", Version: "1", Fingerprint: "operator-contract-1",
		InputType: "rows/order", OutputType: "amount", Effect: "pure",
	}
	parseCalls := 0
	parser := Provider[PipelineSyntax]{
		Selection: language,
		Parse: func(src string) (PipelineSyntax, error) {
			parseCalls++
			return ParsePipelineSyntax(src)
		},
	}

	t.Run("positive exact Language and Operator pins", func(t *testing.T) {
		parseCalls = 0
		candidate, err := ParseSelected(language, []Provider[PipelineSyntax]{parser}, source)
		if err != nil || parseCalls != 1 {
			t.Fatalf("exact Language selection failed: parser calls=%d, err=%v", parseCalls, err)
		}
		steps, err := LowerPinnedPipeline(candidate, []OperatorContract{operator}, []OperatorContract{operator})
		if err != nil || len(steps) != 1 || steps[0].Contract != operator {
			t.Fatalf("pinned Operator lowering failed: %+v, %v", steps, err)
		}
		if steps[0].Span != candidate.Operators[0].Span ||
			source[steps[0].Span.Start:steps[0].Span.End] != operator.ID {
			t.Fatalf("Source token provenance lost across reference stages: %+v", steps[0].Span)
		}
	})
	t.Run("BS-03 mismatched implementation blocks parse", func(t *testing.T) {
		parseCalls = 0
		changed := parser
		changed.Selection.ImplementationFingerprint = "impl-B"
		_, err := ParseSelected(language, []Provider[PipelineSyntax]{changed}, source)
		if !errors.Is(err, ErrProviderMissing) || parseCalls != 0 {
			t.Fatalf("mismatched Language was used: parse calls=%d, err=%v", parseCalls, err)
		}
	})
	t.Run("BS-06 mismatched Operator pin blocks lower", func(t *testing.T) {
		parseCalls = 0
		candidate, err := ParseSelected(language, []Provider[PipelineSyntax]{parser}, source)
		if err != nil || parseCalls != 1 {
			t.Fatalf("identical Source could not be parsed: calls=%d, err=%v", parseCalls, err)
		}
		stale := operator
		stale.Fingerprint = "operator-contract-2"
		steps, err := LowerPinnedPipeline(candidate, []OperatorContract{stale}, []OperatorContract{operator})
		var diagnostic *PipelineDiagnostic
		if len(steps) != 0 || !errors.As(err, &diagnostic) ||
			!errors.Is(err, ErrOperatorMissing) {
			t.Fatalf("stale Operator contract was accepted: steps=%+v, err=%v", steps, err)
		}
		if diagnostic.Step != 0 || diagnostic.Span != candidate.Operators[0].Span {
			t.Fatalf("stale Operator diagnostic lost Source span: %+v", diagnostic)
		}
	})
	t.Run("changed catalog contract cannot silently use old pin", func(t *testing.T) {
		candidate, err := ParseSelected(language, []Provider[PipelineSyntax]{parser}, source)
		if err != nil {
			t.Fatal(err)
		}
		modified := operator
		modified.OutputType = "rows/text"
		steps, err := LowerPinnedPipeline(candidate, []OperatorContract{operator}, []OperatorContract{modified})
		var diagnostic *PipelineDiagnostic
		if len(steps) != 0 || !errors.As(err, &diagnostic) {
			t.Fatalf("changed catalog contract escaped pin: %+v, %v", steps, err)
		}
		if diagnostic.Span != candidate.Operators[0].Span ||
			!errors.Is(err, ErrOperatorInvalid) {
			t.Fatalf("wrong changed catalog diagnostic: %+v / %v", diagnostic, err)
		}
	})
}
