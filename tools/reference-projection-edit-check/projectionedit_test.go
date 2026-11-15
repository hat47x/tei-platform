package projectionedit

import (
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"testing"

	"tei/internal/workspace"
	facetmappingcheck "tei/tools/reference-facet-mapping-check"
)

// Expected values in this file are written by hand from the reference Source
// text and from #2497 / TASK 7. None of them is read back from Judge or Apply.

var (
	repoRoot string
	accept   Acceptance
)

func TestMain(m *testing.M) {
	_, file, _, _ := runtime.Caller(0)
	repoRoot = filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	dir, err := os.MkdirTemp("", "projection-edit-bin-")
	if err != nil {
		panic(err)
	}
	binary := filepath.Join(dir, "acceptance")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	build := exec.Command("go", "build", "-o", binary, "./tools/reference-shared-acceptance")
	build.Dir = repoRoot
	if out, err := build.CombinedOutput(); err != nil {
		panic(string(out))
	}
	accept = Acceptance{Binary: binary}
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

func scratch(t *testing.T) string {
	t.Helper()
	dst := filepath.Join(t.TempDir(), "workspace")
	src := filepath.Join(repoRoot, "reference", "workspace")
	err := filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
	return dst
}

func digests(t *testing.T, dir string) map[string]string {
	t.Helper()
	out := map[string]string{}
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(data)
		rel, _ := filepath.Rel(dir, path)
		out[filepath.ToSlash(rel)] = hex.EncodeToString(sum[:])
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func changed(before, after map[string]string) (modified, added []string) {
	for p, d := range after {
		old, ok := before[p]
		switch {
		case !ok:
			added = append(added, p)
		case old != d:
			modified = append(modified, p)
		}
	}
	sort.Strings(modified)
	sort.Strings(added)
	return
}

// attributeTypes reads every attribute's type through the workspace loader, not
// through the projection under test.
func attributeTypes(t *testing.T, dir string) map[string]string {
	t.Helper()
	loaded, err := workspace.Load(dir)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	out := map[string]string{}
	for _, ns := range loaded.World.Semantics {
		for _, d := range ns.Data {
			for _, a := range d.Attributes {
				out[string(a.ID)] = string(a.TypeRef)
			}
		}
	}
	return out
}

func read(t *testing.T, dir, rel string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func write(t *testing.T, dir, rel, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, filepath.FromSlash(rel)), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func row(t *testing.T, dir, subject string) Row {
	t.Helper()
	rows, err := accept.Project(dir)
	if err != nil {
		t.Fatal(err)
	}
	r, ok := Find(rows, subject)
	if !ok {
		t.Fatalf("no row for %s", subject)
	}
	return r
}

const (
	orderFile = "semantic/sales/order.tei"
	lineID    = "sales.order_line.order_line_id"
)

func ctx(grant bool) Context { return Context{WriteGrant: grant} }

func setType(v string) Edit { return Edit{Op: opSetAttrType, NewValue: v} }

// A positive row edit reaches Source only through existing acceptance, changes
// exactly the type token, and survives an independent reread and reprojection.
func TestPositiveEditThroughExistingAcceptance(t *testing.T) {
	dir := scratch(t)
	before := digests(t, dir)
	typesBefore := attributeTypes(t, dir)
	original := read(t, dir, orderFile)
	r := row(t, dir, lineID)

	if r.Definitions != 1 || r.Value != "sales.order_line_id" || r.Artifact != orderFile {
		t.Fatalf("row=%+v", r)
	}
	v, err := accept.Apply(dir, r, setType("sales.order_id"), ctx(true))
	if err != nil || v.State != StateApplied {
		t.Fatalf("verdict=%+v err=%v", v, err)
	}

	wantFile := strings.Replace(original,
		"attribute orderLineId type sales.order_line_id",
		"attribute orderLineId type sales.order_id", 1)
	if wantFile == original {
		t.Fatal("test expectation did not change the text")
	}
	if got := read(t, dir, orderFile); got != wantFile {
		t.Fatalf("Source differs from the hand-written expectation:\n%s", got)
	}
	modified, added := changed(before, digests(t, dir))
	if !reflect.DeepEqual(modified, []string{orderFile}) {
		t.Fatalf("modified=%v", modified)
	}
	for _, p := range added {
		if !strings.HasPrefix(p, ".tei/decisions/") {
			t.Fatalf("unexpected new file %s", p)
		}
	}
	typesAfter := attributeTypes(t, dir)
	for id, was := range typesBefore {
		want := was
		if id == lineID {
			want = "sales.order_id"
		}
		if typesAfter[id] != want {
			t.Fatalf("%s: type=%q want %q", id, typesAfter[id], want)
		}
	}
	if len(typesAfter) != len(typesBefore) {
		t.Fatal("attribute set changed")
	}
	if got := row(t, dir, lineID); got.Value != "sales.order_id" || got.Definitions != 1 {
		t.Fatalf("reprojection=%+v", got)
	}
}

// Two attributes share the symbol subjectId. Editing one identity must not touch
// the other.
func TestSameSymbolDifferentIdentityIsNotConfused(t *testing.T) {
	dir := scratch(t)
	types := attributeTypes(t, dir)
	var same []string
	for id := range types {
		if strings.HasSuffix(id, ".subject_id") {
			same = append(same, id)
		}
	}
	sort.Strings(same)
	if len(same) < 2 {
		t.Fatalf("fixture has no same-symbol pair: %v", same)
	}
	target := same[0]
	r := row(t, dir, target)
	v, err := accept.Apply(dir, r, setType("customer.customer_id"), ctx(true))
	if err != nil || v.State != StateApplied {
		t.Fatalf("state=%s diagnostics=%v err=%v", v.State, v.Diagnostics, err)
	}
	after := attributeTypes(t, dir)
	for _, id := range same {
		want := types[id]
		if id == target {
			want = "customer.customer_id"
		}
		if after[id] != want {
			t.Fatalf("%s: type=%q want %q", id, after[id], want)
		}
	}
}

// Anything that is not APPLIED leaves every byte of the workspace untouched.
func TestNonAppliedStatesWriteNothing(t *testing.T) {
	cases := []struct {
		name   string
		want   string
		codes  []string
		mutate func(t *testing.T, dir string, r *Row)
		edit   Edit
		grant  bool
	}{
		{name: "position moved since the row was read", want: StateStale, edit: setType("sales.order_id"), grant: true,
			mutate: func(t *testing.T, dir string, r *Row) {
				write(t, dir, orderFile, "\n"+read(t, dir, orderFile))
			}},
		{name: "value changed underneath", want: StateStale, edit: setType("sales.order_id"), grant: true,
			mutate: func(t *testing.T, dir string, r *Row) {
				write(t, dir, orderFile, strings.Replace(read(t, dir, orderFile),
					"attribute orderLineId type sales.order_line_id", "attribute orderLineId type std.identifier", 1))
			}},
		{name: "no write grant", want: StateWithheld, edit: setType("sales.order_id"), grant: false},
		{name: "no-op edit", want: StateNoop, edit: setType("sales.order_line_id"), grant: true},
		{name: "unknown subject", want: StateUnsupported, edit: setType("sales.order_id"), grant: true,
			mutate: func(t *testing.T, dir string, r *Row) { r.SubjectID = "sales.order_line.nope" }},
		{name: "operation not defined for the property", want: StateUnsupported, grant: true,
			edit: Edit{Op: "delete-attribute"}},
		{name: "row did not resolve to one definition", want: StateAmbiguous, edit: setType("sales.order_id"), grant: true,
			mutate: func(t *testing.T, dir string, r *Row) { r.Definitions = 2 }},
		{name: "representation differs", want: StateRejected, edit: setType("std.text"), grant: true,
			codes: []string{"ATTRIBUTE_TYPE_REPRESENTATION_DIFFERS"}},
		{name: "a model constraint elsewhere refuses a same-representation type", want: StateRejected,
			edit: setType("sales.order_line_id"), grant: true, codes: []string{"HEADER_DETAIL_BINDING_INPUT_TYPE_MISMATCH"},
			mutate: func(t *testing.T, dir string, r *Row) { *r = row(t, dir, "sales.order.order_id") }},
		{name: "undeclared type", want: StateRejected, edit: setType("sales.missing_type"), grant: true,
			codes: []string{"ATTRIBUTE_TYPE_NOT_FOUND"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := scratch(t)
			r := row(t, dir, lineID)
			if tc.mutate != nil {
				tc.mutate(t, dir, &r)
			}
			before := digests(t, dir)
			v, err := accept.Apply(dir, r, tc.edit, ctx(tc.grant))
			if err != nil {
				t.Fatal(err)
			}
			if v.State != tc.want {
				t.Fatalf("state=%s want %s (%+v)", v.State, tc.want, v)
			}
			for _, code := range tc.codes {
				found := false
				for _, d := range v.Diagnostics {
					found = found || d == code
				}
				if !found {
					t.Fatalf("diagnostics=%v lacks %s", v.Diagnostics, code)
				}
			}
			if modified, added := changed(before, digests(t, dir)); len(modified)+len(added) != 0 {
				t.Fatalf("workspace changed: modified=%v added=%v", modified, added)
			}
		})
	}
}

// After STALE, an explicit reprojection of the same subject can be applied.
func TestStaleRecoversByReprojection(t *testing.T) {
	dir := scratch(t)
	old := row(t, dir, lineID)
	write(t, dir, orderFile, "\n"+read(t, dir, orderFile))
	if v, _ := accept.Apply(dir, old, setType("sales.order_id"), ctx(true)); v.State != StateStale {
		t.Fatalf("old row state=%s", v.State)
	}
	fresh := row(t, dir, lineID)
	if fresh.Start != old.Start+1 {
		t.Fatalf("span moved by %d, want 1", fresh.Start-old.Start)
	}
	if v, err := accept.Apply(dir, fresh, setType("sales.order_id"), ctx(true)); err != nil || v.State != StateApplied {
		t.Fatalf("verdict=%+v err=%v", v, err)
	}
}

// Known limit, asserted so it is not forgotten: a content revision cannot see
// A -> B -> A. The edit is judged on the bytes now, which are the bytes read.
func TestABAIsNotDetectedByContentRevision(t *testing.T) {
	dir := scratch(t)
	r := row(t, dir, lineID)
	original := read(t, dir, orderFile)
	write(t, dir, orderFile, strings.Replace(original, "sales.order_line_id", "std.identifier", 1))
	write(t, dir, orderFile, original)
	v, err := accept.Judge(dir, r, setType("sales.order_id"), ctx(true))
	if err != nil || v.State != StateEligible {
		t.Fatalf("verdict=%+v err=%v", v, err)
	}
}

// Every attribute row can navigate to Source, whether or not it is editable.
func TestEveryRowNavigatesToItsSourceSpan(t *testing.T) {
	dir := scratch(t)
	rows, err := accept.Project(dir)
	if err != nil {
		t.Fatal(err)
	}
	types := attributeTypes(t, dir)
	if len(rows) != len(types) {
		t.Fatalf("rows=%d attributes=%d", len(rows), len(types))
	}
	for _, r := range rows {
		text := read(t, dir, r.Artifact)
		if r.Start < 0 || r.End > len(text) || text[r.Start:r.End] != types[r.SubjectID] {
			t.Fatalf("%s: span text %q, loader says %q", r.SubjectID, text[r.Start:r.End], types[r.SubjectID])
		}
	}
}

// Edits the operation must accept are not refused for caution's sake.
func TestPossibleEditsAreNotRefused(t *testing.T) {
	possible := []struct{ subject, to string }{
		{lineID, "sales.order_id"},
		{lineID, "std.identifier"},
	}
	refused := 0
	for _, p := range possible {
		dir := scratch(t)
		v, err := accept.Apply(dir, row(t, dir, p.subject), setType(p.to), ctx(true))
		if err != nil {
			t.Fatal(err)
		}
		if v.State != StateApplied {
			refused++
			t.Errorf("%s -> %s: state=%s diagnostics=%v", p.subject, p.to, v.State, v.Diagnostics)
		}
	}
	if refused != 0 {
		t.Fatalf("%d possible edits refused", refused)
	}
}

// The mapping side: an edit-inverse is never allowed through a projection the
// existing checker finds lossy or rejected. Expected eligibility is the
// hand-written column of plan/fixtures/facet-mapping/expected.json.
func TestLossyProjectionsAreNeverEditableThroughTheChecker(t *testing.T) {
	cases, err := facetmappingcheck.LoadCases(filepath.Join(repoRoot, "plan/fixtures/facet-mapping/cases.json"))
	if err != nil {
		t.Fatal(err)
	}
	expected, err := facetmappingcheck.LoadExpected(filepath.Join(repoRoot, "plan/fixtures/facet-mapping/expected.json"))
	if err != nil {
		t.Fatal(err)
	}
	report := facetmappingcheck.Check(cases, facetmappingcheck.Options{})
	want := map[string]string{}
	for _, e := range expected.Expectations {
		want[e.ID] = e.Inverse
	}
	misAllowed := 0
	for _, res := range report.Results {
		if EditableThroughMapping(res) && want[res.ID] != facetmappingcheck.InverseEligible {
			misAllowed++
			t.Errorf("%s allowed an edit the expected file does not", res.ID)
		}
	}
	if misAllowed != 0 {
		t.Fatalf("%d lossy edits allowed", misAllowed)
	}
}

// Characterization of a Gap (TASK 7 S4), not a wished-for outcome: the existing
// checker takes its losses from the structure and representation facets, so
// re-scoping a Graph -> Tree mapping to the node-attribute slot does not make that
// slot editable. Operation-relative editability therefore cannot yet be expressed
// as a mapping row. When #4351 can scope losses to a slot, this test should change
// with it; until then the whole-mapping gate stays closed.
func TestNodeAttributeSlotOfALossyGraphTreeMappingStaysIneligible(t *testing.T) {
	cases, err := facetmappingcheck.LoadCases(filepath.Join(repoRoot, "plan/fixtures/facet-mapping/cases.json"))
	if err != nil {
		t.Fatal(err)
	}
	var c facetmappingcheck.Case
	for _, x := range cases.Cases {
		if x.ID == "XMAP-02" {
			c = x
		}
	}
	if c.ID == "" {
		t.Fatal("XMAP-02 missing")
	}
	c.ID = "XMAP-02-node-slot"
	c.Mapping.Source.Occurrence = "single"
	c.Mapping.Source.Grain = "node"
	c.Mapping.Identity = facetmappingcheck.IdentityColumn{Kind: "1:1", ClaimedPreserved: true, Evidence: []string{"node subject id carried on the row"}}
	c.Mapping.Grain = facetmappingcheck.GrainColumn{SourceGrain: "node", TargetGrain: "node", ClaimedPreserved: true, Evidence: []string{"same grain"}}
	c.Mapping.Multiset = facetmappingcheck.MultisetColumn{Fanout: 1, ClaimedPreserved: true, Evidence: []string{"one row per node"}}
	c.Mapping.Loss.Declared = nil
	c.Mapping.Operation = facetmappingcheck.OperationColumn{Mode: "edit-inverse", InverseEligible: true}
	report := facetmappingcheck.Check(facetmappingcheck.CaseFile{SchemaVersion: cases.SchemaVersion, Cases: []facetmappingcheck.Case{c}}, facetmappingcheck.Options{})
	res := report.Results[0]
	if EditableThroughMapping(res) {
		t.Fatalf("a node-slot mapping on a multigraph -> tree projection was allowed: %+v", res)
	}
	if res.Verdict != facetmappingcheck.VerdictLossy || res.Inverse != facetmappingcheck.InverseIneligible {
		t.Fatalf("verdict=%s inverse=%s losses=%v", res.Verdict, res.Inverse, res.Losses)
	}
}
