package cswprobe

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/semantic"
	"tei/internal/validation"
	"tei/internal/workspace"
)

func TestWorkspaceDeclaresOnlyTheHandoffBoundary(t *testing.T) {
	loaded := loadWorkspace(t)
	namespace := namespaceByID(t, loaded.World.Semantics, "csw")

	var dataIDs []string
	for _, data := range namespace.Data {
		dataIDs = append(dataIDs, string(data.ID))
	}
	sort.Strings(dataIDs)
	want := []string{"csw.author_decision", "csw.candidate_fragment", "csw.study_input", "csw.study_run"}
	if strings.Join(dataIDs, ",") != strings.Join(want, ",") {
		t.Fatalf("data declarations = %v want %v", dataIDs, want)
	}

	for _, typ := range namespace.Types {
		if typ.ID != "csw.attribution" {
			continue
		}
		got := append([]string(nil), typ.Values...)
		sort.Strings(got)
		wantValues := []string{AttributionAuthorStated, AttributionGeneratedProposal, AttributionSourceMaterial}
		sort.Strings(wantValues)
		if strings.Join(got, ",") != strings.Join(wantValues, ",") {
			t.Fatalf("csw.attribution values = %v want %v", got, wantValues)
		}
	}

	if len(namespace.Capabilities) != 4 {
		t.Fatalf("capabilities = %d want 4", len(namespace.Capabilities))
	}
	if len(namespace.Validations) != 2 {
		t.Fatalf("validations = %d want 2", len(namespace.Validations))
	}
}

// The typed boundary must not carry the prose. This test fails if a later
// revision adds a body attribute to CandidateFragment, because the probe would
// then be storing creative content as modeled data.
func TestCandidateFragmentHasNoBodyAttribute(t *testing.T) {
	loaded := loadWorkspace(t)
	namespace := namespaceByID(t, loaded.World.Semantics, "csw")
	for _, data := range namespace.Data {
		if data.ID != "csw.candidate_fragment" {
			continue
		}
		var attributes []string
		for _, attribute := range data.Attributes {
			attributes = append(attributes, attribute.Symbol)
		}
		sort.Strings(attributes)
		want := "attribution,bodyRef,bodySha256,fragmentId,ordinal,runId,studyInputId"
		if strings.Join(attributes, ",") != want {
			t.Fatalf("candidate fragment attributes = %v; the boundary keeps references and hashes, not prose", attributes)
		}
		return
	}
	t.Fatal("csw.candidate_fragment is not declared")
}

func TestDecodeFixtureAcceptsOnlyDeclaredFields(t *testing.T) {
	data := readFixture(t, "handoff_v0.json")
	handoff, err := DecodeHandoff(data)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(handoff.StudyInputs) != 2 || len(handoff.Runs) != 2 || len(handoff.Fragments) != 4 || len(handoff.Decisions) != 2 {
		t.Fatalf("record counts = %d/%d/%d/%d", len(handoff.StudyInputs), len(handoff.Runs), len(handoff.Fragments), len(handoff.Decisions))
	}

	// A body passed as a field is refused instead of quietly dropped.
	withBody := strings.Replace(string(data), `"body_ref": "work/synthetic/f-1.md"`, `"body_ref": "work/synthetic/f-1.md", "body_text": "prose must stay outside the typed boundary"`, 1)
	if _, err := DecodeHandoff([]byte(withBody)); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("a body_text field was not refused: %v", err)
	}
}

func TestValidateFixturePasses(t *testing.T) {
	handoff := decodeFixture(t, "handoff_v0.json")
	if diagnostics := Validate(handoff); len(diagnostics) != 0 {
		t.Fatalf("valid fixture produced diagnostics: %+v", diagnostics)
	}
}

func TestValidateFixtureRefusesUnknownAttributionAndProcessDecision(t *testing.T) {
	handoff := decodeFixture(t, "handoff_v0_invalid.json")
	got := map[string]bool{}
	for _, diagnostic := range Validate(handoff) {
		got[diagnostic.Code] = true
	}
	for _, want := range []string{"CSW_HANDOFF_ATTRIBUTION_UNKNOWN", "CSW_HANDOFF_DECISION_AUTHOR_MISSING"} {
		if !got[want] {
			t.Fatalf("codes = %v; %s was not reported", got, want)
		}
	}
}

func TestValidateRefusesDuplicateOrdinalWithinARun(t *testing.T) {
	handoff := decodeFixture(t, "handoff_v0.json")
	handoff.Fragments[1].Ordinal = handoff.Fragments[0].Ordinal
	got := map[string]bool{}
	for _, diagnostic := range Validate(handoff) {
		got[diagnostic.Code] = true
	}
	if !got["CSW_HANDOFF_ORDINAL_DUPLICATE"] {
		t.Fatalf("codes = %v; duplicate ordinal was not reported", got)
	}
}

func TestValidateRefusesDecisionWithoutFragment(t *testing.T) {
	handoff := decodeFixture(t, "handoff_v0.json")
	handoff.Decisions[0].FragmentID = "missing"
	if !hasCode(Validate(handoff), "CSW_HANDOFF_DECISION_FRAGMENT_UNRESOLVED") {
		t.Fatal("a decision for an unknown fragment was accepted")
	}
}

func loadWorkspace(t *testing.T) workspace.Result {
	t.Helper()
	loaded, err := workspace.Load(filepath.Join("workspace"))
	if err != nil {
		t.Fatalf("load Workspace: %v", err)
	}
	for _, diagnostic := range validation.Validate(loaded.World) {
		if diagnostic.Severity == validation.Error {
			t.Fatalf("Workspace validation error: %s %s", diagnostic.Code, diagnostic.Message)
		}
	}
	return loaded
}

func namespaceByID(t *testing.T, namespaces []semantic.Namespace, id core.CanonicalID) semantic.Namespace {
	t.Helper()
	for _, namespace := range namespaces {
		if namespace.ID == id {
			return namespace
		}
	}
	t.Fatalf("semantic namespace %s is not declared", id)
	return semantic.Namespace{}
}

func decodeFixture(t *testing.T, name string) Handoff {
	t.Helper()
	handoff, err := DecodeHandoff(readFixture(t, name))
	if err != nil {
		t.Fatalf("decode %s: %v", name, err)
	}
	return handoff
}

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("fixtures", name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return data
}

func hasCode(diagnostics []Diagnostic, code string) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == code {
			return true
		}
	}
	return false
}

func BenchmarkDecodeAndValidate(b *testing.B) {
	data, err := os.ReadFile(filepath.Join("fixtures", "handoff_v0.json"))
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		handoff, err := DecodeHandoff(data)
		if err != nil {
			b.Fatal(err)
		}
		if diagnostics := Validate(handoff); len(diagnostics) != 0 {
			b.Fatalf("unexpected diagnostics: %+v", diagnostics)
		}
	}
}

func BenchmarkWorkspaceLoadAndValidate(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		loaded, err := workspace.Load(filepath.Join("workspace"))
		if err != nil {
			b.Fatal(err)
		}
		for _, diagnostic := range validation.Validate(loaded.World) {
			if diagnostic.Severity == validation.Error {
				b.Fatalf("%s %s", diagnostic.Code, diagnostic.Message)
			}
		}
	}
}
