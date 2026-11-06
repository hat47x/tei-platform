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
// This test bridges only BS-02/03/06; the remaining bootstrap/recovery cases
// are design-only, not implicitly qualified by a passing go test.
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
	if fixture.Status != "DESIGN_ONLY_NOT_EXECUTED" {
		t.Fatalf("expected design-only fixture status, got %q", fixture.Status)
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
	// This is a Load-level rejection. It proves neither a production
	// Language Plugin registry nor a particular downstream parser invocation.
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
	_, err := ParseSelected(pin, []Provider[string]{{
		Selection: offered,
		Parse: func(string) (string, error) {
			parsed = true
			return "candidate", nil
		},
	}}, "data source")
	if !errors.Is(err, ErrProviderMissing) || parsed {
		t.Fatalf("BS-03 parser bypassed the exact implementation pin: parsed=%v, err=%v", parsed, err)
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
	// The resulting error is a bounded reference rejection, not proof of
	// product Operator registration, accepted meaning, or active generation.
}
