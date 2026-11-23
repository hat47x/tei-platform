package operations

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"tei/internal/codegen/goemit"
	"tei/internal/codegen/goemit/goemittest"
	"tei/internal/core"
	"tei/internal/semantic"
	"tei/internal/semantic/validationeval"
	"tei/internal/sourceindex"
	"tei/internal/validation"
	"tei/internal/workspace"
)

// tableRuleParitySource captures the logical Read even when a wrong attribute
// happens to return the same test value. It is not a database or a Provider.
type tableRuleParitySource struct {
	value     string
	missing   bool
	reads     []core.CanonicalID
	readState string
}

func (s *tableRuleParitySource) ReadNumber(id core.CanonicalID) (validationeval.NumberReading, error) {
	s.reads = append(s.reads, id)
	switch id {
	case "organization.transfer_request.allocation",
		"organization.transfer_request.person_id":
	default:
		return validationeval.NumberReading{}, errors.New("unexpected Attribute read")
	}
	if s.missing {
		s.readState = "not-fetched"
		return validationeval.NumberReading{State: validationeval.ValueNotFetched}, nil
	}
	if s.value == "" {
		s.readState = "null"
		return validationeval.NumberReading{State: validationeval.ValueNull}, nil
	}
	s.readState = "present"
	n, err := semantic.ParseOperatorCaseValue(s.value)
	if err != nil {
		return validationeval.NumberReading{}, err
	}
	return validationeval.NumberReading{State: validationeval.ValuePresent, Value: n}, nil
}

func (s *tableRuleParitySource) ReadRelationKeys(relation, attribute core.CanonicalID) (validationeval.RelationReading, error) {
	return validationeval.RelationReading{}, errors.New("unexpected Relation read")
}

// A marked Markdown Data table and its adjacent Rule reach the product
// Evaluator and an actually compiled/generated Go binary. Equality of result
// alone is not enough: the attribute the binary reads must be the same.
//
// The generated Go reads nothing itself. Its check function receives the
// value, and the Trace that goemit.Emit returns names the attribute that value
// comes from. The driver takes the generated mode's binding from that Trace,
// never from this test or from comment prose. The two adversarial modes are
// written by this test, so their bindings are too, and say so.
// This proves a bounded pure-Rule G/P comparison, not Builder R2-P, Provider
// R3, Owner authority, or a second Semantic engine.
func TestMarkedDataTableRuleRealGoAndEvaluatorReadParity(t *testing.T) {
	if testing.Short() {
		t.Skip("focused test compiles a generated Go binary")
	}
	goemittest.GoExecutable(t)
	fixture := filepath.Join("..", "workspace", "testdata", "tei-md-data-table")
	const sourcePath = "semantic/organization/transfer-request.tei.md"
	const ruleID = "organization.transfer_request.allocation_positive"
	const attributeID = "organization.transfer_request.allocation"
	const otherAttribute = "organization.transfer_request.person_id"
	loaded, err := workspace.Load(fixture)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.AuthoringSources) != 1 ||
		loaded.AuthoringSources[0].Language.LanguageID != "tei.authoring" ||
		loaded.AuthoringSources[0].Language.ContractVersion != "v1" ||
		loaded.AuthoringSources[0].Path != sourcePath {
		t.Fatalf("unexpected product Reader selection: %+v", loaded.AuthoringSources)
	}
	if diagnostics := validation.Validate(loaded.World); len(diagnostics) != 0 {
		t.Fatalf("selected Data and Rule are not valid: %+v", diagnostics)
	}
	idx, err := workspace.BuildSourceIndexFromLoad(fixture, loaded)
	if err != nil {
		t.Fatal(err)
	}
	original, err := os.ReadFile(filepath.Join(fixture, filepath.FromSlash(sourcePath)))
	if err != nil {
		t.Fatal(err)
	}
	if err := idx.Validate(func(path string) ([]byte, error) {
		return os.ReadFile(filepath.Join(fixture, filepath.FromSlash(path)))
	}); err != nil {
		t.Fatalf("selected Source is stale: %v", err)
	}

	read := func(kind, id string) json.RawMessage {
		t.Helper()
		subject := sourceindex.Subject{Kind: kind, ID: id}
		inspected, err := InspectionFromLoad(loaded, idx, subject)
		if err != nil {
			t.Fatal(err)
		}
		if err := ValidateInspectionProjection(subject, inspected); err != nil {
			t.Fatal(err)
		}
		if inspected.Effect != "read-only" ||
			len(inspected.Sources.References) != 1 ||
			len(inspected.Sources.Artifacts) != 1 ||
			inspected.Sources.References[0].ArtifactPath != sourcePath ||
			inspected.Sources.References[0].Role != sourceindex.RoleDefinition ||
			inspected.Sources.Artifacts[0].Fingerprint != sourceindex.Fingerprint(original) {
			t.Fatalf("selected subject lost its original definition: %+v", inspected)
		}
		if err := inspected.Sources.References[0].Range.Validate(original); err != nil {
			t.Fatal(err)
		}
		return inspected.Canonical
	}
	var attribute semantic.Attribute
	if err := json.Unmarshal(read("semantic-attribute", attributeID), &attribute); err != nil {
		t.Fatal(err)
	}
	var rule semantic.Validation
	if err := json.Unmarshal(read("semantic-validation", ruleID), &rule); err != nil {
		t.Fatal(err)
	}
	if attribute.ID != attributeID || attribute.TypeRef != "std.decimal" ||
		rule.ID != ruleID || rule.Kind != semantic.ValidationPositive ||
		rule.Attribute != attribute.ID || rule.Relation != "" {
		t.Fatalf("Data table and selected Rule no longer share a valid typed target: %+v / %+v", attribute, rule)
	}
	out := goemit.Emit(goemit.Input{PackageName: "main", Rules: []goemit.Rule{{Validation: rule, AttributeType: attribute.TypeRef}}})
	if err := out.Err(); err != nil {
		t.Fatal(err)
	}
	bindings, err := goemittest.Bindings(out)
	if err != nil || len(bindings) != 1 ||
		bindings[0] != (goemittest.Binding{Check: "CheckAllocationIsPositive", Parameter: "allocation", Attribute: attributeID}) {
		t.Fatalf("generator changed the expected binding or entry: %+v %v", bindings, err)
	}
	generated := out.File(out.Trace[0].File)
	// One-factor adversarial implementation: the decision and the outcome
	// stay identical for the supplied values, but the copy is bound to
	// another Attribute and must not qualify. The copy is renamed only so it
	// can share the package with the generated file.
	misbound := bytes.ReplaceAll(generated, []byte("AllocationIsPositive"), []byte("MisboundAllocationIsPositive"))
	// A Target symbol may change without changing the selected semantic Rule.
	// Keep the meaning and the binding constant while renaming only the Go
	// entry. A future Test ID must not be keyed to this generated function name.
	renamed := bytes.ReplaceAll(generated, []byte("AllocationIsPositive"), []byte("RenamedAllocationIsPositive"))
	if bytes.Equal(generated, misbound) || bytes.Equal(generated, renamed) ||
		!bytes.Contains(renamed, []byte("func CheckRenamedAllocationIsPositive(")) ||
		!bytes.Contains(misbound, []byte("func CheckMisboundAllocationIsPositive(")) {
		t.Fatal("rename or wrong-read copy was not produced")
	}

	// The generated mode is bound by the Trace; the two copies are bound by
	// this test: renamed to the same attribute, misbound to another one.
	var entries strings.Builder
	for _, mode := range []struct {
		name      string
		attribute string
		check     string
	}{
		{"generated", bindings[0].Attribute, bindings[0].Check},
		{"renamed", bindings[0].Attribute, "CheckRenamedAllocationIsPositive"},
		{"misbound", otherAttribute, "CheckMisboundAllocationIsPositive"},
	} {
		fmt.Fprintf(&entries, "\t\t{%q, %q, %s},\n", mode.name, mode.attribute, mode.check)
	}
	files := goemittest.Files(out)
	files["renamed.go"] = renamed
	files["misbound.go"] = misbound
	files["main.go"] = []byte(fmt.Sprintf(markedDataParityDriver, entries.String()))
	if imports, err := goemittest.TEIImports(files); err != nil || len(imports) != 0 {
		t.Fatalf("parity program depends on TEI: %v %v", imports, err)
	}
	// A module of its own, outside the repository: nothing in it imports TEI.
	pkgDir := goemittest.WriteModule(t, files)
	binary := filepath.Join(t.TempDir(), "tei-data-rule-parity")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	goemittest.RunGo(t, pkgDir, "build", "-o", binary, ".")
	run := exec.Command(binary)
	var stdout, stderr bytes.Buffer
	run.Stdout, run.Stderr = &stdout, &stderr
	if err := run.Run(); err != nil || stderr.Len() != 0 {
		t.Fatalf("generated Go binary failed: %v stderr=%q stdout=%q", err, stderr.String(), stdout.String())
	}
	type actual struct {
		CaseID    string
		Mode      string
		Name      string
		Result    string
		Error     bool
		Reads     int
		Attribute string
		ReadState string
	}
	records := map[string]actual{}
	for _, line := range bytes.Split(bytes.TrimSpace(stdout.Bytes()), []byte("\n")) {
		var got actual
		if err := json.Unmarshal(line, &got); err != nil {
			t.Fatalf("non-JSON binary evidence: %v %q", err, line)
		}
		key := got.Mode + "/" + got.Name
		if _, exists := records[key]; exists {
			t.Fatalf("duplicate Consumer result %s", key)
		}
		records[key] = got
	}
	// A null value and a value that was not obtained are both undecided:
	// Decide reports the reason, and the generated check function, which
	// receives nil for both, answers Undecided. Neither is an error any more.
	cases := []struct {
		name, value, want, wantReadState string
		missing, wantError               bool
	}{
		{"positive", "2.5", "satisfied", "present", false, false},
		{"zero", "0", "not-satisfied", "present", false, false},
		{"negative", "-1", "not-satisfied", "present", false, false},
		{"negative-zero", "-0", "not-satisfied", "present", false, false},
		{"null", "", "undecided", "null", false, false},
		{"not-obtained", "", "undecided", "not-fetched", true, false},
	}
	if len(records) != len(cases)*3 {
		t.Fatalf("binary returned %d records, want %d", len(records), len(cases)*3)
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// An independent fixed expected outcome is never derived from G.
			source := &tableRuleParitySource{value: tc.value, missing: tc.missing}
			decision, err := validationeval.Decide(rule, source, validationeval.Operators{})
			direct := string(decision.Result)
			if decision.Undecided != nil {
				direct = "undecided"
			}
			if direct != tc.want || (err != nil) != tc.wantError ||
				len(source.reads) != 1 || source.reads[0] != core.CanonicalID(attributeID) ||
				source.readState != tc.wantReadState {
				t.Fatalf("P did not match the fixed Law/Read: %q %v %v", direct, err, source.reads)
			}
			// Test identity belongs to the chosen Rule and case, not the
			// generated Go entry symbol. The observed reading state must survive
			// the P/G boundary even when both missing inputs are undecided.
			caseID := ruleID + "/" + tc.name
			got := records["generated/"+tc.name]
			if got.CaseID != caseID || got.Mode != "generated" || got.Name != tc.name ||
				got.Result != tc.want || got.Error != tc.wantError ||
				got.Reads != 1 || got.Attribute != attributeID ||
				got.ReadState != tc.wantReadState {
				t.Fatalf("G outcome or Read disagrees with independent P contract: %+v", got)
			}
			// Identical semantic subject, judgment, and Read remain valid when
			// a regenerated Target uses a different Go method name.
			renamedGot := records["renamed/"+tc.name]
			if renamedGot.CaseID != caseID || renamedGot.Mode != "renamed" || renamedGot.Name != tc.name ||
				renamedGot.Result != tc.want || renamedGot.Error != tc.wantError ||
				renamedGot.Reads != 1 || renamedGot.Attribute != attributeID ||
				renamedGot.ReadState != tc.wantReadState {
				t.Fatalf("Target symbol rename changed the semantic Rule outcome or Read: %+v", renamedGot)
			}
			wrong := records["misbound/"+tc.name]
			if wrong.CaseID != caseID || wrong.Mode != "misbound" || wrong.Name != tc.name ||
				wrong.Result != tc.want || wrong.Error != tc.wantError ||
				wrong.Reads != 1 || wrong.Attribute != otherAttribute ||
				wrong.ReadState != tc.wantReadState {
				t.Fatalf("same-value wrong-Read negative did not execute: %+v", wrong)
			}
			if wrong.Attribute == got.Attribute {
				t.Fatal("an output-equal implementation with a different Read passed source equivalence")
			}
		})
	}
}

// markedDataParityDriver is compiled together with the actual
// product-generated Go, in a module of its own. Each mode reads the attribute
// it is bound to; the misbound copy is bound to another
// attribute that happens to hold the same value, so outcome parity alone
// would accept it. The test fills in the modes.
const markedDataParityDriver = `package main

import (
	"encoding/json"
	"errors"
	"math/big"
	"os"
)

type probe struct {
	value     string
	missing   bool
	reads     []string
	readState string
}

func (p *probe) number(id string) (*big.Rat, error) {
	p.reads = append(p.reads, id)
	switch id {
	case "organization.transfer_request.allocation", "organization.transfer_request.person_id":
	default:
		return nil, errors.New("unexpected read")
	}
	if p.missing {
		p.readState = "not-fetched"
		return nil, nil
	}
	if p.value == "" {
		p.readState = "null"
		return nil, nil
	}
	p.readState = "present"
	return ParseDecimal(p.value)
}

type result struct {
	CaseID    string
	Mode      string
	Name      string
	Result    string
	Error     bool
	Reads     int
	Attribute string
	ReadState string
}

func main() {
	cases := []struct {
		name, value string
		missing     bool
	}{
		{"positive", "2.5", false},
		{"zero", "0", false},
		{"negative", "-1", false},
		{"negative-zero", "-0", false},
		{"null", "", false},
		{"not-obtained", "", true},
	}
	modes := []struct {
		name, attribute string
		check           func(*big.Rat) Result
	}{
%s	}
	enc := json.NewEncoder(os.Stdout)
	for _, tc := range cases {
		for _, mode := range modes {
			src := &probe{value: tc.value, missing: tc.missing}
			value, err := src.number(mode.attribute)
			got := ""
			if err == nil {
				got = string(mode.check(value))
			}
			// This ID is attached to the Rule/case, never to the Go method.
			record := result{CaseID: "organization.transfer_request.allocation_positive/" + tc.name, Mode: mode.name, Name: tc.name, Result: got, Error: err != nil, Reads: len(src.reads), ReadState: src.readState}
			if len(src.reads) > 0 {
				record.Attribute = src.reads[0]
			}
			if err := enc.Encode(record); err != nil {
				panic(err)
			}
		}
	}
}
`
