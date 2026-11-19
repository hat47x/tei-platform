package teimdreader

import (
	"bytes"
	"reflect"
	"strings"
	"testing"
)

func TestDerivedViewAuditDistinguishesEdgeLossAndEndpointChange(t *testing.T) {
	const removed = "    N1 -->|e-2| N2\n"
	const extra = "    N1 -->|e-3| N2\n"
	base := string(fixtureSource(t, "baseline.tei.md"))
	aliases := map[string]string{"N1": "n-1", "N2": "n-2"}
	cases := []struct {
		name, src string
		want      DerivedViewAudit
	}{
		{"unchanged parallel edges", base, DerivedViewAudit{Exact: true}},
		{"layout-only", string(fixtureSource(t, "layout-only.tei.md")), DerivedViewAudit{Exact: true}},
		{"endpoint change", string(fixtureSource(t, "diagram-stale.tei.md")), DerivedViewAudit{EndpointChanges: []string{"e-2"}}},
		{"missing parallel edge", strings.Replace(base, removed, "", 1), DerivedViewAudit{MissingEdgeIDs: []string{"e-2"}}},
		{"unexpected edge", strings.Replace(base, removed, removed+extra, 1), DerivedViewAudit{UnexpectedEdgeIDs: []string{"e-3"}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			source := []byte(tc.src)
			snapshot := bytes.Clone(source)
			candidate, err := ParseCandidate(source)
			if err != nil {
				t.Fatal(err)
			}
			before := candidate
			got, err := AuditDerivedView(source, candidate, aliases)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("audit = %#v, want %#v", got, tc.want)
			}
			exact, err := CheckDerivedView(source, candidate, aliases)
			if err != nil || exact != tc.want.Exact {
				t.Fatalf("legacy view check = %t, %v; want %t", exact, err, tc.want.Exact)
			}
			if !reflect.DeepEqual(candidate, before) || !bytes.Equal(source, snapshot) {
				t.Fatal("read-only audit mutated candidate or original source bytes")
			}
		})
	}
}

func TestDerivedViewAuditDoesNotClaimNodeCoverageOrSemanticAcceptance(t *testing.T) {
	base := string(fixtureSource(t, "baseline.tei.md"))
	src := strings.Replace(base, "| n-2 | 対象 |", "| n-2 | 対象 |\n| n-3 | 対象 |", 1)
	candidate, err := ParseCandidate([]byte(src))
	if err != nil {
		t.Fatal(err)
	}
	if len(candidate.Nodes) != 3 {
		t.Fatal("test setup did not retain the isolated third node")
	}
	audit, err := AuditDerivedView([]byte(src), candidate, map[string]string{"N1": "n-1", "N2": "n-2"})
	if err != nil || !audit.Exact {
		t.Fatalf("edge-only diagram should remain exact, got %#v, %v", audit, err)
	}
	// Exact means that every *edge* matches; it does not establish that all
	// nodes were drawn, nor authorize a model edit.
}

func TestDerivedViewAuditFailsClosedForAmbiguousDisplayIdentity(t *testing.T) {
	source := fixtureSource(t, "baseline.tei.md")
	candidate, err := ParseCandidate(source)
	if err != nil {
		t.Fatal(err)
	}
	_, err = AuditDerivedView(source, candidate, map[string]string{"N1": "n-1"})
	errCode(t, err, "UNBOUND_DISPLAY_ALIAS")
	duplicate := strings.Replace(string(source), "    N1 -->|e-2| N2\n", "    N1 -->|e-1| N2\n", 1)
	_, err = AuditDerivedView([]byte(duplicate), candidate, map[string]string{"N1": "n-1", "N2": "n-2"})
	errCode(t, err, "DUPLICATE_VIEW_EDGE")
}
