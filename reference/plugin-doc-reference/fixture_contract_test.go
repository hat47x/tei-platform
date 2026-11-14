// Package plugindocreference audits candidate fixture input, not a TEI document
// reader, plugin resolver, disclosure engine, or owner-accepted oracle.
package plugindocreference

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	graphreader "tei/reference/tei-md-graph-reader"
)

type requirement struct {
	Plugin          string
	CompatibleMajor int
}
type candidatePlugin struct {
	ID, Version, Contract, PublicExtensionCandidate                    string
	Requires                                                           requirement
	SelectedBaseCandidate, ReferenceMode, Reexport, BaseManualReexport string
}
type candidateDocument struct {
	ID, Owner, Contract, Path, Visibility, SourceRefStatus string
}
type candidateRelation struct {
	ID, From, To, OccurrenceRole, RoleAdoption string
}
type candidateCase struct {
	ID, Variant string
	Expect      map[string]any
}
type candidateFixture struct {
	Schema, Adoption, Authority string
	ProductsNotClaimed          []string
	Graph                       struct {
		Source                   string
		SemanticIDs              struct{ Nodes, DirectedRelations []string }
		IdenticalNodeLabels      bool
		ParallelRelations        []candidateRelation
		DiagramIDsNotSemanticIDs []string
	}
	Plugins   struct{ A, B, C candidatePlugin }
	Documents struct{ Public, Private candidateDocument }
	Cases     []candidateCase
}

func readFixture(t *testing.T) candidateFixture {
	t.Helper()
	p := filepath.Join("..", "..", "testdata", "plugin-doc-reference", "independent-expected.candidate.json")
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	var f candidateFixture
	if err := json.Unmarshal(b, &f); err != nil {
		t.Fatal(err)
	}
	return f
}

func safeSourcePath(s string) bool {
	return s != "" && !strings.ContainsAny(s, "\\:") &&
		path.Clean(s) == s && strings.HasPrefix(s, "testdata/") &&
		!strings.Contains(s, "../")
}

// auditFixture only checks consistency of *candidate* input and expectations.
// In particular, a "denied" expected value does NOT exercise Product AuthZ.
func auditFixture(f candidateFixture) []string {
	var problems []string
	add := func(format string, args ...any) { problems = append(problems, fmt.Sprintf(format, args...)) }
	if f.Schema != "tei.plugin-public-doc-independent-expected-candidate/v0" || f.Adoption != "proposal-only" {
		add("fixture must remain an unadopted candidate")
	}
	for _, claim := range []string{"selected product plugin", "source-backed plugin-doc read", "formal tei.md meaning reader", "runtime provider invoke", "publication grant"} {
		found := false
		for _, v := range f.ProductsNotClaimed {
			if v == claim {
				found = true
				break
			}
		}
		if !found {
			add("forbidden product qualification: %s", claim)
		}
	}
	a, b, c := f.Plugins.A, f.Plugins.B, f.Plugins.C
	if a.ID == "" || a.Version == "" || a.Contract == "" || a.PublicExtensionCandidate == "" {
		add("base plugin identity/contract/extension incomplete")
	}
	major, err := strconv.Atoi(strings.SplitN(a.Version, ".", 2)[0])
	if err != nil || major < 1 {
		add("unreadable A major version")
	}
	if b.ID == "" || b.ID == a.ID || b.Requires.Plugin != a.ID || b.Requires.CompatibleMajor != major {
		add("B does not declare the one compatible A dependency")
	}
	if b.SelectedBaseCandidate != a.ID+"@"+a.Version || b.ReferenceMode != "link-only; no doc or graph copy" || b.Reexport != "not-authorized" {
		add("B has no exact candidate selection or safe reference boundary")
	}
	if c.ID == "" || c.ID == a.ID || c.ID == b.ID || c.Requires.Plugin != b.ID || c.Requires.CompatibleMajor != 1 || c.BaseManualReexport != "not-authorized" {
		add("C must not inherit A's republication right")
	}
	pub, priv := f.Documents.Public, f.Documents.Private
	if pub.ID == "" || priv.ID == "" || pub.ID == priv.ID || pub.Owner != a.ID+"@"+a.Version || priv.Owner != pub.Owner || pub.Contract != a.Contract {
		add("document owner, public contract or stable identity mismatch")
	}
	if pub.Visibility != "allowed-in-candidate-for-B" || priv.Visibility != "denied-to-B-and-C" {
		add("candidate visibility was expanded or removed")
	}
	if pub.SourceRefStatus != "candidate-unbound" || priv.SourceRefStatus != "candidate-unbound" {
		add("product source-binding falsely claimed")
	}
	if !safeSourcePath(pub.Path) || !safeSourcePath(priv.Path) || !safeSourcePath(f.Graph.Source) || pub.Path == priv.Path {
		add("invalid or aliased candidate source paths")
	}
	if !f.Graph.IdenticalNodeLabels || len(f.Graph.SemanticIDs.Nodes) != 2 || len(f.Graph.SemanticIDs.DirectedRelations) != 2 || len(f.Graph.ParallelRelations) != 2 {
		add("two-node two-relation pressure fixture missing")
	}
	roles, edges := map[string]bool{}, map[string]bool{}
	for _, e := range f.Graph.ParallelRelations {
		if e.ID == "" || edges[e.ID] || e.OccurrenceRole == "" || roles[e.OccurrenceRole] || e.RoleAdoption != "candidate-only" {
			add("distinct candidate relation roles/identities lost")
		}
		edges[e.ID], roles[e.OccurrenceRole] = true, true
	}
	for _, id := range f.Graph.SemanticIDs.DirectedRelations {
		if !edges[id] {
			add("semantic relation %s has no candidate occurrence", id)
		}
	}
	expectedCases := []string{"PDC-00", "PDC-01", "PDC-02", "PDC-03", "PDC-04", "PDC-05", "PDC-06", "PDC-07", "PDC-08", "PHD-02", "PHD-03", "PHD-08", "LOCK-UNKNOWN"}
	seen := map[string]bool{}
	byID := map[string]candidateCase{}
	for _, tc := range f.Cases {
		if tc.ID == "" || seen[tc.ID] {
			add("empty or duplicate case id %q", tc.ID)
		}
		seen[tc.ID] = true
		byID[tc.ID] = tc
	}
	if len(seen) != len(expectedCases) {
		add("candidate case count mismatch")
	}
	for _, id := range expectedCases {
		if !seen[id] {
			add("missing case %s", id)
		}
	}
	for id, result := range map[string]string{
		"PDC-03": "permission-denied", "PDC-04": "version-mismatch", "PDC-06": "downstream-requalification-required",
		"PHD-02": "unsupported-command", "PHD-03": "reexport-not-authorized", "PHD-08": "stale-response", "LOCK-UNKNOWN": "exact-version-unknown",
	} {
		if byID[id].Expect["result"] != result {
			add("unexpected refusal/qualification in %s", id)
		}
	}
	if byID["PDC-03"].Expect["returnedPrivateBody"] != false || byID["PDC-03"].Expect["returnedPrivateMetadata"] != false {
		add("private body or metadata might be disclosed")
	}
	if byID["PDC-05"].Expect["sourceRevisionRecheck"] != true {
		add("source recheck lost for documentation/layout mutation")
	}
	return problems
}

func TestCandidateFixtureLinksToRealGraphAndManualBytes(t *testing.T) {
	f := readFixture(t)
	if problems := auditFixture(f); len(problems) > 0 {
		t.Fatalf("invalid candidate fixture: %v", problems)
	}
	root := filepath.Join("..", "..")
	source, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(f.Graph.Source)))
	if err != nil {
		t.Fatal(err)
	}
	got, err := graphreader.ParseCandidate(source)
	if err != nil {
		t.Fatalf("existing reference reader could not read graph: %v", err)
	}
	if len(got.Nodes) != 2 || len(got.Edges) != 2 {
		t.Fatalf("graph identity count changed: %d/%d", len(got.Nodes), len(got.Edges))
	}
	if got.Nodes[0].ID != f.Graph.SemanticIDs.Nodes[0] || got.Nodes[1].ID != f.Graph.SemanticIDs.Nodes[1] ||
		got.Nodes[0].Label != got.Nodes[1].Label || got.Nodes[0].ID == got.Nodes[1].ID {
		t.Fatal("graph's same-label distinct semantic nodes were lost")
	}
	for i, e := range got.Edges {
		candidate := f.Graph.ParallelRelations[i]
		if e.ID != f.Graph.SemanticIDs.DirectedRelations[i] || e.ID != candidate.ID || e.From != candidate.From || e.To != candidate.To {
			t.Fatalf("relation %d differs from authored graph: %+v / %+v", i, e, candidate)
		}
		for _, p := range []struct {
			start, end int
			text       string
		}{
			{e.IDSource.Start, e.IDSource.End, e.ID}, {e.FromSource.Start, e.FromSource.End, e.From}, {e.ToSource.Start, e.ToSource.End, e.To},
		} {
			if p.start < 0 || p.end > len(source) || p.start > p.end || string(source[p.start:p.end]) != p.text {
				t.Fatal("source byte span no longer identifies authored relation")
			}
		}
	}
	pub, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(f.Documents.Public.Path)))
	if err != nil {
		t.Fatal(err)
	}
	priv, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(f.Documents.Private.Path)))
	if err != nil {
		t.Fatal(err)
	}
	if len(pub) == 0 || len(priv) == 0 || bytes.Equal(pub, priv) ||
		!bytes.Contains(pub, []byte("tei commands")) || !bytes.Contains(pub, []byte("baseline.tei.md")) ||
		bytes.Contains(pub, priv) {
		t.Fatal("public manual source is missing or improperly duplicates private content")
	}
	// The public and private bytes exist in the test tree; this does not
	// mean any TEI Product Consumer is authorized to retrieve them.
}

func TestCandidateFixtureMutationsAreRejected(t *testing.T) {
	original := readFixture(t)
	cases := []struct {
		name   string
		mutate func(*candidateFixture)
	}{
		{"base-version-drift", func(f *candidateFixture) { f.Plugins.A.Version = "2.0.0" }},
		{"wrong-parent", func(f *candidateFixture) { f.Plugins.B.Requires.Plugin = "fixture.other" }},
		{"unknown-exact-lock", func(f *candidateFixture) { f.Plugins.B.SelectedBaseCandidate = "" }},
		{"republication-allowed", func(f *candidateFixture) { f.Plugins.B.Reexport = "authorized" }},
		{"private-made-public", func(f *candidateFixture) { f.Documents.Private.Visibility = "allowed-in-candidate-for-B" }},
		{"false-source-acceptance", func(f *candidateFixture) { f.Documents.Public.SourceRefStatus = "accepted" }},
		{"traversal", func(f *candidateFixture) { f.Documents.Public.Path = "../private.md" }},
		{"conflated-relations", func(f *candidateFixture) { f.Graph.ParallelRelations[1].ID = "e-1" }},
		{"candidate-role-promoted", func(f *candidateFixture) { f.Graph.ParallelRelations[0].RoleAdoption = "accepted" }},
		{"lost-private-refusal", func(f *candidateFixture) {
			for i := range f.Cases {
				if f.Cases[i].ID == "PDC-03" {
					f.Cases[i].Expect["returnedPrivateMetadata"] = true
				}
			}
		}},
		{"missing-independent-case", func(f *candidateFixture) { f.Cases = f.Cases[:len(f.Cases)-1] }},
		{"product-read-claimed", func(f *candidateFixture) { f.ProductsNotClaimed = f.ProductsNotClaimed[1:] }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			raw, err := json.Marshal(original)
			if err != nil {
				t.Fatal(err)
			}
			var copy candidateFixture
			if err := json.Unmarshal(raw, &copy); err != nil {
				t.Fatal(err)
			}
			tc.mutate(&copy)
			if problems := auditFixture(copy); len(problems) == 0 {
				t.Fatal("mutation escaped fixture audit")
			}
		})
	}
}
