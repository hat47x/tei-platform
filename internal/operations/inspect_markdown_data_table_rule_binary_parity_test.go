package operations

import (
	"bytes"
	"encoding/json"
	"errors"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"tei/internal/core"
	"tei/internal/semantic"
	"tei/internal/semantic/validationeval"
	"tei/internal/semantic/validationgen"
	"tei/internal/sourceindex"
	"tei/internal/validation"
	"tei/internal/workspace"
)

// tableRuleParitySource captures the logical Read even when a wrong attribute
// happens to return the same test value. It is not a database or a Provider.
type tableRuleParitySource struct {
	value   string
	missing bool
	reads   []core.CanonicalID
}

func (s *tableRuleParitySource) Number(id core.CanonicalID) (*big.Rat, error) {
	s.reads = append(s.reads, id)
	switch id {
	case "organization.transfer_request.allocation",
		"organization.transfer_request.person_id":
	default:
		return nil, errors.New("unexpected Attribute read")
	}
	if s.missing {
		return nil, errors.New("source was not obtained")
	}
	if s.value == "" {
		return nil, nil
	}
	n, ok := new(big.Rat).SetString(s.value)
	if !ok {
		return nil, errors.New("invalid numeric test input")
	}
	return n, nil
}

func (s *tableRuleParitySource) RelationKeys(relation, attribute core.CanonicalID) ([]string, error) {
	return nil, errors.New("unexpected Relation read")
}

// A marked Markdown Data table and its adjacent Rule reach the product
// Evaluator and an actually compiled/generated Go binary. Equality of result
// alone is not enough: the binary's observed Number target must be the same.
// This proves a bounded pure-Rule G/P comparison, not Builder R2-P, Provider
// R3, Owner authority, or a second Semantic engine.
func TestMarkedDataTableRuleRealGoAndEvaluatorReadParity(t *testing.T) {
	if testing.Short() {
		t.Skip("focused test compiles a generated Go binary")
	}
	goExe, err := exec.LookPath("go")
	if err != nil {
		t.Fatalf("Go compiler unavailable: %v", err)
	}
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
	generated, err := validationgen.GeneratePositive(rule, attribute.TypeRef, "main")
	if err != nil {
		t.Fatal(err)
	}
	const selectedCall = "source.Number(\"organization.transfer_request.allocation\")"
	const otherCall = "source.Number(\"organization.transfer_request.person_id\")"
	if bytes.Count(generated, []byte(selectedCall)) != 1 ||
		bytes.Count(generated, []byte("func EvaluateSelected(")) != 1 {
		t.Fatalf("generator changed the expected read or entry: %s", generated)
	}
	// One-factor adversarial implementation: the outcome stays identical for
	// the supplied values, but it reads another Attribute and must not qualify.
	misbound := bytes.Replace(generated, []byte(selectedCall), []byte(otherCall), 1)
	misbound = bytes.Replace(misbound, []byte("func EvaluateSelected("), []byte("func EvaluateMisbound("), 1)
	if bytes.Equal(generated, misbound) || !bytes.Contains(misbound, []byte(otherCall)) {
		t.Fatal("wrong-read negative was not produced")
	}
	// A Target symbol may change without changing the selected semantic Rule.
	// Keep the meaning and observed Read constant while renaming only the Go
	// entry. A future Test ID must not be keyed to this generated function name.
	renamed := bytes.Replace(generated, []byte("func EvaluateSelected("), []byte("func EvaluateRenamed("), 1)
	if bytes.Equal(generated, renamed) || !bytes.Contains(renamed, []byte("func EvaluateRenamed(")) ||
		!bytes.Contains(renamed, []byte(selectedCall)) {
		t.Fatal("method-rename positive was not produced without changing the Read")
	}

	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	pkgDir, err := os.MkdirTemp(repoRoot, "tei-data-rule-binary-parity-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(pkgDir) })
	// This temporary package belongs to module tei, which owns the generated
	// code's existing tei/internal import. It is not a delivered artifact.
	for name, data := range map[string][]byte{
		"generated.go": generated,
		"renamed.go":   renamed,
		"misbound.go":  misbound,
		"main.go":      []byte(markedDataParityDriver),
	} {
		if err := os.WriteFile(filepath.Join(pkgDir, name), data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	binary := filepath.Join(t.TempDir(), "tei-data-rule-parity")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	build := exec.Command(goExe, "build", "-o", binary, ".")
	build.Dir = pkgDir
	build.Env = append(os.Environ(), "GOWORK=off")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("compile actual generated Go: %v\n%s", err, output)
	}
	run := exec.Command(binary)
	var stdout, stderr bytes.Buffer
	run.Stdout, run.Stderr = &stdout, &stderr
	if err := run.Run(); err != nil || stderr.Len() != 0 {
		t.Fatalf("generated Go binary failed: %v stderr=%q stdout=%q", err, stderr.String(), stdout.String())
	}
	type actual struct {
		Mode      string
		Name      string
		Result    string
		Error     bool
		Reads     int
		Attribute string
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
	cases := []struct {
		name, value, want  string
		missing, wantError bool
	}{
		{"positive", "2.5", "satisfied", false, false},
		{"zero", "0", "not-satisfied", false, false},
		{"negative", "-1", "not-satisfied", false, false},
		{"negative-zero", "-0", "not-satisfied", false, false},
		{"null", "", "", false, true},
		{"not-obtained", "", "", true, true},
	}
	if len(records) != len(cases)*3 {
		t.Fatalf("binary returned %d records, want %d", len(records), len(cases)*3)
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// An independent fixed expected outcome is never derived from G.
			source := &tableRuleParitySource{value: tc.value, missing: tc.missing}
			direct, err := validationeval.Evaluate(rule, source)
			if string(direct) != tc.want || (err != nil) != tc.wantError ||
				len(source.reads) != 1 || source.reads[0] != core.CanonicalID(attributeID) {
				t.Fatalf("P did not match the fixed Law/Read: %q %v %v", direct, err, source.reads)
			}
			got := records["generated/"+tc.name]
			if got.Mode != "generated" || got.Name != tc.name ||
				got.Result != tc.want || got.Error != tc.wantError ||
				got.Reads != 1 || got.Attribute != attributeID {
				t.Fatalf("G outcome or Read disagrees with independent P contract: %+v", got)
			}
			// Identical semantic subject, judgment, and Read remain valid when
			// a regenerated Target uses a different Go method name.
			renamedGot := records["renamed/"+tc.name]
			if renamedGot.Mode != "renamed" || renamedGot.Name != tc.name ||
				renamedGot.Result != tc.want || renamedGot.Error != tc.wantError ||
				renamedGot.Reads != 1 || renamedGot.Attribute != attributeID {
				t.Fatalf("Target symbol rename changed the semantic Rule outcome or Read: %+v", renamedGot)
			}
			wrong := records["misbound/"+tc.name]
			if wrong.Mode != "misbound" || wrong.Name != tc.name ||
				wrong.Result != tc.want || wrong.Error != tc.wantError ||
				wrong.Reads != 1 || wrong.Attribute != otherAttribute {
				t.Fatalf("same-value wrong-Read negative did not execute: %+v", wrong)
			}
			if wrong.Attribute == got.Attribute {
				t.Fatal("an output-equal implementation with a different Read passed source equivalence")
			}
		})
	}
}

// This driver is compiled together with the actual product-generated Go.
// The second implementation intentionally performs a wrong but value-equal
// Read: outcome parity alone would accept it.
const markedDataParityDriver = `package main

import (
	"encoding/json"
	"errors"
	"math/big"
	"os"

	"tei/internal/core"
	"tei/internal/semantic/validationeval"
)

type probe struct {
	value   string
	missing bool
	reads   []string
}
func (p *probe) Number(id core.CanonicalID) (*big.Rat,error) {
	p.reads=append(p.reads,string(id))
	switch id {
	case "organization.transfer_request.allocation", "organization.transfer_request.person_id":
	default: return nil,errors.New("unexpected read")
	}
	if p.missing { return nil,errors.New("not obtained") }
	if p.value=="" { return nil,nil }
	n,ok:=new(big.Rat).SetString(p.value)
	if !ok { return nil,errors.New("bad input") }
	return n,nil
}
func (p *probe) RelationKeys(relation, attribute core.CanonicalID)([]string,error){
	return nil,errors.New("unexpected relation")
}
type result struct {
	Mode string
	Name string
	Result string
	Error bool
	Reads int
	Attribute string
}
func main(){
	cases:=[]struct{name,value string; missing bool}{
		{"positive","2.5",false},
		{"zero","0",false},
		{"negative","-1",false},
		{"negative-zero","-0",false},
		{"null","",false},
		{"not-obtained","",true},
	}
	enc:=json.NewEncoder(os.Stdout)
	for _,tc:=range cases {
		for _,mode:=range []string{"generated","renamed","misbound"}{
			src:=&probe{value:tc.value,missing:tc.missing}
			var got validationeval.Result
			var err error
			if mode=="generated" {got,err=EvaluateSelected(src)} else if mode=="renamed" {got,err=EvaluateRenamed(src)} else {got,err=EvaluateMisbound(src)}
			record:=result{Mode:mode,Name:tc.name,Result:string(got),Error:err!=nil,Reads:len(src.reads)}
			if len(src.reads)>0 {record.Attribute=src.reads[0]}
			if err:=enc.Encode(record);err!=nil{panic(err)}
		}
	}
}
`
