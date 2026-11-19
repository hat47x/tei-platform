package main

import (
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/driver"
	"tei/internal/semantic/validationeval"
	"tei/internal/workspace"
	"tei/tools/reference-application-binary/internal/browsegen"
)

type vgCase struct {
	Name      string          `json:"name"`
	Exit      int             `json:"exit"`
	Input     json.RawMessage `json:"input"`
	Results   []string        `json:"results"`
	StoppedAt string          `json:"stoppedAt"`
}

type vgExpected struct {
	Rules []string `json:"rules"`
	Cases []vgCase `json:"cases"`
}

type vgOutput struct {
	Results []struct {
		Validation string `json:"validation"`
		Result     string `json:"result"`
	} `json:"results"`
	StoppedAt string `json:"stoppedAt"`
}

func vgExpectedFile(t *testing.T) vgExpected {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", "validation_gate_expected.json"))
	if err != nil {
		t.Fatal(err)
	}
	var e vgExpected
	if err := json.Unmarshal(raw, &e); err != nil {
		t.Fatal(err)
	}
	return e
}

func vgGenerate(t *testing.T, ws string) (*browsegen.ValidationPlan, string) {
	t.Helper()
	plan, err := browsegen.NewValidationPlan(ws, b1App)
	if err != nil {
		t.Fatalf("NewValidationPlan: %v", err)
	}
	out := t.TempDir()
	if _, err := browsegen.GenerateValidation(plan, out); err != nil {
		t.Fatalf("GenerateValidation: %v", err)
	}
	return plan, out
}

// vgRun feeds the input on stdin and returns the exit code and decoded output.
func vgRun(t *testing.T, bin string, input []byte) (int, []string, string, string) {
	t.Helper()
	cmd := exec.Command(bin, "validate", b1App)
	cmd.Stdin = strings.NewReader(string(input))
	var stdout, stderr strings.Builder
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	code := 0
	if err != nil {
		ee, ok := err.(*exec.ExitError)
		if !ok {
			t.Fatalf("run: %v", err)
		}
		code = ee.ExitCode()
	}
	var o vgOutput
	if s := strings.TrimSpace(stdout.String()); s != "" {
		if err := json.Unmarshal([]byte(s), &o); err != nil {
			t.Fatalf("output is not JSON: %v: %s", err, s)
		}
	}
	var res []string
	for _, r := range o.Results {
		res = append(res, r.Validation+"="+r.Result)
	}
	return code, res, o.StoppedAt, stderr.String()
}

// vgDiffs runs every hand-fixed case and returns the names that do not match.
func vgDiffs(t *testing.T, bin string, exp vgExpected) []string {
	t.Helper()
	var bad []string
	for _, c := range exp.Cases {
		code, res, stopped, _ := vgRun(t, bin, c.Input)
		if code != c.Exit || strings.Join(res, ",") != strings.Join(c.Results, ",") || stopped != c.StoppedAt {
			bad = append(bad, fmt.Sprintf("%s: exit %d results %v stopped %q", c.Name, code, res, stopped))
		}
	}
	return bad
}

func vgBuild(t *testing.T, out string) (*browsegen.Receipt, string) {
	return b1BuildBinary(t, out)
}

// V-T1: the generated gate matches the hand-fixed expectation on every case.
func TestVGGeneratedGateMatchesFixedExpected(t *testing.T) {
	exp := vgExpectedFile(t)
	plan, out := vgGenerate(t, b1Workspace(t))
	var ids []string
	for _, r := range plan.Rules {
		ids = append(ids, r.ID)
	}
	if strings.Join(ids, ",") != strings.Join(exp.Rules, ",") {
		t.Fatalf("rules %v, want the declared %v", ids, exp.Rules)
	}
	rc, bin := vgBuild(t, out)
	if err := browsegen.Verify(out, bin, rc); err != nil {
		t.Fatal(err)
	}
	for _, d := range vgDiffs(t, bin, exp) {
		t.Error(d)
	}
	// Input that is not JSON is a usage error, not an evaluation.
	if code, _, _, _ := vgRun(t, bin, []byte("{")); code != 2 {
		t.Errorf("non-JSON input: exit %d, want 2", code)
	}
	// V-T2 uses the same binary: the existing Evaluator must agree on every case.
	vgDifferential(t, bin, exp)
	// No effect or database is part of the artifact.
	src, _ := os.ReadFile(filepath.Join(out, "main.go"))
	for _, bad := range []string{"database/sql", "INSERT", "UPDATE", "pgx", "sqlite"} {
		if strings.Contains(string(src), bad) {
			t.Errorf("rule gate contains %q", bad)
		}
	}
}

type vgSource struct {
	Numbers map[string]string `json:"numbers"`
	RelKeys []struct {
		Relation  string   `json:"relation"`
		Attribute string   `json:"attribute"`
		Keys      []string `json:"keys"`
	} `json:"relationKeys"`
}

func (s vgSource) Number(a core.CanonicalID) (*big.Rat, error) {
	raw, ok := s.Numbers[string(a)]
	if !ok {
		return nil, fmt.Errorf("no value for %s", a)
	}
	v, ok := new(big.Rat).SetString(raw)
	if !ok {
		return nil, fmt.Errorf("%q is not a number", raw)
	}
	return v, nil
}

func (s vgSource) RelationKeys(rel, attr core.CanonicalID) ([]string, error) {
	for _, e := range s.RelKeys {
		if e.Relation == string(rel) && e.Attribute == string(attr) {
			return e.Keys, nil
		}
	}
	return nil, fmt.Errorf("no keys for %s / %s", rel, attr)
}

// V-T2: the existing Evaluator, run over the same inputs in the same order,
// must agree with the generated gate on results, stopping point and
// "could not be evaluated".
func vgDifferential(t *testing.T, bin string, exp vgExpected) {
	t.Helper()
	loaded, err := workspace.Load(b1Workspace(t))
	if err != nil {
		t.Fatal(err)
	}
	rules, err := driver.NewResolver(loaded.World).ResolveApplicationValidations(core.MustCanonicalID(b1App))
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range exp.Cases {
		var src vgSource
		if err := json.Unmarshal(c.Input, &src); err != nil {
			t.Fatal(err)
		}
		var want []string
		wantExit, wantStop := 0, ""
		for _, r := range rules {
			res, err := validationeval.Evaluate(r, src)
			if err != nil {
				wantExit, wantStop = 3, string(r.ID)
				break
			}
			want = append(want, string(r.ID)+"="+string(res))
			if res != validationeval.ResultSatisfied {
				wantExit, wantStop = 1, string(r.ID)
				break
			}
		}
		code, got, stopped, _ := vgRun(t, bin, c.Input)
		if code != wantExit || strings.Join(got, ",") != strings.Join(want, ",") || stopped != wantStop {
			t.Errorf("%s: gate exit %d %v stop %q, Evaluator exit %d %v stop %q", c.Name, code, got, stopped, wantExit, want, wantStop)
		}
	}
}

func vgReplaceInFile(t *testing.T, path, old, new string) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), old) {
		t.Fatalf("%s does not contain %q", path, old)
	}
	if err := os.WriteFile(path, []byte(strings.Replace(string(raw), old, new, 1)), 0o644); err != nil {
		t.Fatal(err)
	}
}

func vgAppTei(ws string) string {
	return filepath.Join(ws, "applications", "order-entry", "application.tei")
}

// V-T3: changing the declared order changes which rule decides.
func TestVGDeclaredOrderDecides(t *testing.T) {
	ws := b1CopyWorkspace(t)
	vgReplaceInFile(t, vgAppTei(ws),
		"    uses validation sales.order_line.quantity_positive\n    uses validation sales.order.lines.line_number_unique\n",
		"    uses validation sales.order.lines.line_number_unique\n    uses validation sales.order_line.quantity_positive\n")
	_, out := vgGenerate(t, ws)
	_, bin := vgBuild(t, out)
	for _, c := range vgExpectedFile(t).Cases {
		if !strings.HasPrefix(c.Name, "both quantity and keys bad") {
			continue
		}
		code, res, stopped, _ := vgRun(t, bin, c.Input)
		if code != 1 || stopped != "sales.order.lines.line_number_unique" || len(res) != 1 {
			t.Errorf("reordered Source: exit %d %v stopped %q, want line_number_unique to decide", code, res, stopped)
		}
	}
}

// V-T4: removing a declared Validation removes the rule, changes the Revision,
// and an artifact built before the change is refused.
func TestVGRemovedRuleChangesGateAndStalesOldBinary(t *testing.T) {
	ws := b1CopyWorkspace(t)
	planA, out := vgGenerate(t, ws)
	rc, bin := vgBuild(t, out)
	vgReplaceInFile(t, vgAppTei(ws), "    uses validation sales.order_line.line_number_positive\n", "")
	planB, err := browsegen.NewValidationPlan(ws, b1App)
	if err != nil {
		t.Fatal(err)
	}
	if len(planB.Rules) != 2 || planA.SourceRevision == planB.SourceRevision || planA.ModelDigest == planB.ModelDigest {
		t.Fatalf("rules %d, revisions %s/%s", len(planB.Rules), planA.SourceRevision, planB.SourceRevision)
	}
	if _, err := browsegen.GenerateValidation(planB, out); err != nil {
		t.Fatal(err)
	}
	if err := browsegen.Verify(out, bin, rc); err == nil || !strings.Contains(err.Error(), "STALE_BINARY") {
		t.Fatalf("old binary must be refused, got %v", err)
	}
}

// V-T6: a generator that gets the meaning slightly wrong must be caught by the
// hand-fixed cases. Each mutant is built and run; one that passes means the
// cases are too weak.
func TestVGMutantsAreCaught(t *testing.T) {
	exp := vgExpectedFile(t)
	mutants := []struct{ name, old, new string }{
		{"positive accepts zero", "v.Sign() > 0", "v.Sign() >= 0"},
		{"duplicates never detected", "seen[k] = true", "_ = k"},
		{"no short-circuit", "\t\t\tcode = 1\n\t\t\tbreak\n", "\t\t\tcode = 1\n"},
		{"unevaluable becomes business refusal", "code = 3\n", "code = 1\n"},
	}
	for _, m := range mutants {
		t.Run(m.name, func(t *testing.T) {
			_, out := vgGenerate(t, b1Workspace(t))
			vgReplaceInFile(t, filepath.Join(out, "main.go"), m.old, m.new)
			_, bin := vgBuild(t, out)
			if bad := vgDiffs(t, bin, exp); len(bad) == 0 {
				t.Fatalf("mutant %q passed every fixed case", m.name)
			}
		})
	}
}
