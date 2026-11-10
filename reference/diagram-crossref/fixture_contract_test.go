package diagramcrossref

// Fixture-only audit: no TEI parser, projector, CLI, Studio or Provider is run.
// Passing does not qualify the AI-authored expected JSON as an independent oracle.
import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"
)

type fixture struct {
	SourceRevision string                                `json:"sourceRevision"`
	Entities       []struct{ ID, Label string }          `json:"entities"`
	Relations      []struct{ ID, From, To, Kind string } `json:"relations"`
	Functions      []struct {
		ID                   string `json:"id"`
		DeclaredCrudComplete bool   `json:"declaredCrudComplete"`
		Uses                 []struct {
			Data       string   `json:"data"`
			Operations []string `json:"operations"`
		} `json:"uses"`
	} `json:"functions"`
	BusinessActions []struct {
		ID, Kind string
		Function *string `json:"function"`
	} `json:"businessActions"`
	NegativeVariants []struct{ ID string } `json:"negativeVariants"`
	CanvasViews      []struct {
		ID        string `json:"id"`
		Instances []struct {
			OccurrenceID string `json:"occurrenceId"`
			SubjectID    string `json:"subjectId"`
		} `json:"instances"`
		RelationOccurrences []struct {
			ID         string `json:"id"`
			RelationID string `json:"relationId"`
		} `json:"relationOccurrences"`
	} `json:"canvasViews"`
}

type expectation struct {
	SourceRevision string `json:"sourceRevision"`
	ExpectedGraph  struct {
		EntityCount            int      `json:"entityCount"`
		RelationCount          int      `json:"relationCount"`
		ParallelEdgePair       []string `json:"parallelEdgePair"`
		DuplicateDisplayLabels []struct {
			Label       string
			DistinctIDs []string `json:"distinctIds"`
		} `json:"duplicateDisplayLabels"`
	} `json:"expectedGraph"`
	ExpectedFunctionDeclaredCrudMatrix struct {
		Cells map[string]map[string]string `json:"cells"`
	} `json:"expectedFunctionDeclaredCrudMatrix"`
	KnownMutantRejections []struct{ ID, Expected string } `json:"knownMutantRejections"`
	ExpectedCanvasViews   struct {
		ViewIDs             []string            `json:"viewIds"`
		OccurrenceCount     int                 `json:"occurrenceCount"`
		SharedSubjects      map[string][]string `json:"sharedSubjects"`
		RelationOccurrences map[string][]string `json:"relationOccurrences"`
	} `json:"expectedCanvasViews"`
}

func fixtureProblems(f fixture, want expectation) []string {
	var problems []string
	add := func(s string) { problems = append(problems, s) }
	if f.SourceRevision == "" || f.SourceRevision != want.SourceRevision {
		add("source revision mismatch")
	}
	if len(f.Entities) != want.ExpectedGraph.EntityCount {
		add("entity count mismatch")
	}
	if len(f.Relations) != want.ExpectedGraph.RelationCount {
		add("relation count mismatch")
	}
	entities := make(map[string]string, len(f.Entities))
	for _, e := range f.Entities {
		if e.ID == "" {
			add("empty entity identity")
		}
		if _, ok := entities[e.ID]; ok {
			add("duplicate entity identity: " + e.ID)
		}
		entities[e.ID] = e.Label
	}
	edges := make(map[string][2]string, len(f.Relations))
	for _, edge := range f.Relations {
		if edge.ID == "" {
			add("empty edge identity")
		}
		if _, ok := edges[edge.ID]; ok {
			add("duplicate edge identity: " + edge.ID)
		}
		edges[edge.ID] = [2]string{edge.From, edge.To}
		if _, ok := entities[edge.From]; !ok {
			add("unresolved source endpoint: " + edge.ID)
		}
		if _, ok := entities[edge.To]; !ok {
			add("unresolved target endpoint: " + edge.ID)
		}
	}
	pair := want.ExpectedGraph.ParallelEdgePair
	if len(pair) != 2 {
		add("parallel-edge pair missing")
	} else {
		a, aok := edges[pair[0]]
		b, bok := edges[pair[1]]
		if !aok || !bok || a != b || pair[0] == pair[1] {
			add("parallel edges lost or conflated")
		}
	}
	for _, c := range want.ExpectedGraph.DuplicateDisplayLabels {
		if len(c.DistinctIDs) < 2 {
			add("duplicate-label oracle not discriminating")
		}
		seen := map[string]bool{}
		for _, id := range c.DistinctIDs {
			if seen[id] || entities[id] != c.Label {
				add("duplicate-label subject lost: " + id)
			}
			seen[id] = true
		}
	}
	functions := map[string]bool{}
	for _, fn := range f.Functions {
		if fn.ID == "" || functions[fn.ID] {
			add("missing/duplicate function identity: " + fn.ID)
		}
		functions[fn.ID] = true
		row, ok := want.ExpectedFunctionDeclaredCrudMatrix.Cells[fn.ID]
		if !ok {
			add("missing CRUD expected row: " + fn.ID)
			continue
		}
		declared := map[string]string{}
		for _, u := range fn.Uses {
			if _, ok := entities[u.Data]; !ok {
				add("unknown Function Data: " + u.Data)
			}
			if _, ok := declared[u.Data]; ok {
				add("duplicate Function Data entry: " + fn.ID + "/" + u.Data)
			}
			operations := append([]string(nil), u.Operations...)
			sort.Strings(operations)
			for _, op := range operations {
				if op != "C" && op != "R" && op != "U" && op != "D" {
					add("unknown CRUD marker: " + op)
				}
			}
			declared[u.Data] = strings.Join(operations, "")
		}
		for id := range entities {
			// Incomplete enumeration does not erase explicitly declared operations.
			// It only makes the not-enumerated cells unknown rather than none.
			expected := "unknown-not-enumerated"
			if op, ok := declared[id]; ok {
				expected = op
			} else if fn.DeclaredCrudComplete {
				expected = "none"
			}
			if row[id] != expected {
				add(fmt.Sprintf("CRUD cell %s/%s expected %q got %q", fn.ID, id, expected, row[id]))
			}
		}
		if len(row) != len(entities) {
			add("CRUD matrix extra/missing columns: " + fn.ID)
		}
	}
	if len(functions) != len(want.ExpectedFunctionDeclaredCrudMatrix.Cells) {
		add("CRUD matrix row count mismatch")
	}
	actionIDs := map[string]bool{}
	for _, a := range f.BusinessActions {
		if actionIDs[a.ID] {
			add("duplicate Business Action identity: " + a.ID)
		}
		actionIDs[a.ID] = true
		if a.Kind == "manual" && a.Function != nil {
			add("manual Action forced onto Function")
		}
		if a.Function != nil && !functions[*a.Function] {
			add("Business Action references unknown Function")
		}
	}
	// Presentation instance identity is separate from Semantic Subject identity.
	canvasIDs := map[string]bool{}
	occurrenceIDs := map[string]bool{}
	sharedSubjects := map[string][]string{}
	relationOccurrences := map[string][]string{}
	for _, view := range f.CanvasViews {
		if view.ID == "" || canvasIDs[view.ID] {
			add("duplicate/empty canvas view ID: " + view.ID)
		}
		canvasIDs[view.ID] = true
		for _, instance := range view.Instances {
			if instance.OccurrenceID == "" || occurrenceIDs[instance.OccurrenceID] {
				add("duplicate/empty view occurrence ID: " + instance.OccurrenceID)
			}
			occurrenceIDs[instance.OccurrenceID] = true
			if _, exists := entities[instance.SubjectID]; !exists {
				add("view occurrence references unknown Semantic Subject: " + instance.SubjectID)
			}
			sharedSubjects[instance.SubjectID] = append(sharedSubjects[instance.SubjectID], instance.OccurrenceID)
		}
		for _, edge := range view.RelationOccurrences {
			if edge.ID == "" || occurrenceIDs[edge.ID] {
				add("duplicate/empty relation occurrence ID: " + edge.ID)
			}
			occurrenceIDs[edge.ID] = true
			if _, exists := edges[edge.RelationID]; !exists {
				add("view edge references unknown Relation: " + edge.RelationID)
			}
			relationOccurrences[edge.RelationID] = append(relationOccurrences[edge.RelationID], edge.ID)
		}
	}
	if len(canvasIDs) != len(want.ExpectedCanvasViews.ViewIDs) {
		add("canvas view count mismatch")
	}
	for _, id := range want.ExpectedCanvasViews.ViewIDs {
		if !canvasIDs[id] {
			add("missing required canvas view: " + id)
		}
	}
	if len(sharedSubjects) == 0 || len(occurrenceIDs) == 0 {
		add("empty canvas projection")
	}
	instanceCount := 0
	for _, view := range f.CanvasViews {
		instanceCount += len(view.Instances)
	}
	if instanceCount != want.ExpectedCanvasViews.OccurrenceCount {
		add("canvas subject occurrence count mismatch")
	}
	for id, expected := range want.ExpectedCanvasViews.SharedSubjects {
		actual := append([]string(nil), sharedSubjects[id]...)
		clone := append([]string(nil), expected...)
		sort.Strings(actual)
		sort.Strings(clone)
		if !reflect.DeepEqual(actual, clone) {
			add("shared Semantic Subject lost across canvases: " + id)
		}
	}
	for id, expected := range want.ExpectedCanvasViews.RelationOccurrences {
		actual := append([]string(nil), relationOccurrences[id]...)
		clone := append([]string(nil), expected...)
		sort.Strings(actual)
		sort.Strings(clone)
		if !reflect.DeepEqual(actual, clone) {
			add("shared Relation lost across canvases: " + id)
		}
	}
	mutants := map[string]bool{}
	for _, m := range f.NegativeVariants {
		if mutants[m.ID] {
			add("duplicate known mutant ID: " + m.ID)
		}
		mutants[m.ID] = true
	}
	expectedMutants := map[string]bool{}
	for _, m := range want.KnownMutantRejections {
		if expectedMutants[m.ID] || m.Expected != "FAIL" {
			add("invalid expected mutant: " + m.ID)
		}
		expectedMutants[m.ID] = true
	}
	if !reflect.DeepEqual(mutants, expectedMutants) {
		add("mutant input/expected IDs differ")
	}
	return problems
}

func readFixtures(t *testing.T) (fixture, expectation) {
	t.Helper()
	var f fixture
	var e expectation
	for _, item := range []struct {
		path string
		dst  any
	}{
		{"pressure-input-v0.json", &f},
		{"expected-v0.json", &e},
	} {
		b, err := os.ReadFile(item.path)
		if err != nil {
			t.Fatalf("read %s: %v", item.path, err)
		}
		if err := json.Unmarshal(b, item.dst); err != nil {
			t.Fatalf("decode %s: %v", item.path, err)
		}
	}
	return f, e
}

func TestFixtureInternalConsistencyNotProductConformance(t *testing.T) {
	f, e := readFixtures(t)
	if problems := fixtureProblems(f, e); len(problems) != 0 {
		t.Fatalf("fixture audit: %s", strings.Join(problems, "; "))
	}
}

func TestFixtureAuditRejectsFiveKnownMutations(t *testing.T) {
	f, e := readFixtures(t)
	if problems := fixtureProblems(f, e); len(problems) != 0 {
		t.Fatalf("baseline invalid: %v", problems)
	}
	cases := []struct {
		name   string
		mutate func(*fixture)
	}{
		{"drop-parallel-edge", func(x *fixture) {
			edgeID := e.ExpectedGraph.ParallelEdgePair[1]
			for i, edge := range x.Relations {
				if edge.ID == edgeID {
					x.Relations = append(x.Relations[:i], x.Relations[i+1:]...)
					return
				}
			}
		}},
		{"collapse-identities", func(x *fixture) {
			x.Entities[len(x.Entities)-1].ID = x.Entities[len(x.Entities)-2].ID
		}},
		{"mark-incomplete-CRUD-as-complete", func(x *fixture) {
			for i := range x.Functions {
				if !x.Functions[i].DeclaredCrudComplete {
					x.Functions[i].DeclaredCrudComplete = true
					return
				}
			}
		}},
		{"break-referenced-edge-target", func(x *fixture) {
			x.Relations[0].To = "no.such.subject"
		}},
		{"duplicate-view-occurrence", func(x *fixture) {
			x.CanvasViews[1].Instances[0].OccurrenceID = x.CanvasViews[0].Instances[0].OccurrenceID
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			b, err := json.Marshal(f)
			if err != nil {
				t.Fatal(err)
			}
			var x fixture
			if err := json.Unmarshal(b, &x); err != nil {
				t.Fatal(err)
			}
			tc.mutate(&x)
			if problems := fixtureProblems(x, e); len(problems) == 0 {
				t.Fatal("synthetic mutant escaped fixture audit")
			}
		})
	}
}

func TestPartialFunctionCRUDKeepsKnownCellsAndUnknownRemainder(t *testing.T) {
	f, e := readFixtures(t)
	// Use an existing declared R as the single known cell of an otherwise
	// incompletely enumerated Function; do not introduce a second fixture.
	if len(f.Functions) < 4 || f.Functions[3].ID != "fn.external.dispatch" ||
		f.Functions[3].DeclaredCrudComplete ||
		len(f.Functions[0].Uses) < 3 || f.Functions[0].Uses[2].Data != "data.customer" {
		t.Fatal("unexpected pinned fixture shape")
	}
	f.Functions[3].Uses = append(f.Functions[3].Uses, f.Functions[0].Uses[2])
	row := e.ExpectedFunctionDeclaredCrudMatrix.Cells["fn.external.dispatch"]
	row["data.customer"] = "R"
	if problems := fixtureProblems(f, e); len(problems) != 0 {
		t.Fatalf("explicit R within incomplete CRUD scope must survive: %v", problems)
	}
	// A false unknown must not overwrite the one positively known R.
	row["data.customer"] = "unknown-not-enumerated"
	if problems := fixtureProblems(f, e); len(problems) == 0 {
		t.Fatal("erasure of an explicitly known R escaped fixture audit")
	}
	row["data.customer"] = "R"
	// A cell outside the incomplete declaration is unknown, not 'none'.
	row["data.order"] = "none"
	if problems := fixtureProblems(f, e); len(problems) == 0 {
		t.Fatal("unlisted cell promoted to none escaped fixture audit")
	}
}
