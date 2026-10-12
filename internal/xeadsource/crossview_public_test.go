package xeadsource

import (
	"encoding/json"
	"strings"
	"testing"
)

func testCrossViewPublicProjection(t *testing.T) CrossViewProjection {
	t.Helper()
	return buildCrossViewPublicFixture(t, `<System>
  <Table ID="T1"/>
  <Function ID="F1"><IOTable ID="IO1" TableID="T1" OpR="+"/></Function>
  <Task ID="TASK1"><TaskAction ID="A1"><TaskFunctionIO FunctionID="F1" IOID="0"/></TaskAction></Task>
</System>`)
}

func buildCrossViewPublicFixture(t *testing.T, source string) CrossViewProjection {
	t.Helper()
	p, err := buildCrossViewProjection(crossViewInspection(t, source, crossViewMappingFingerprint()))
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func testCrossViewDeepCopy(t *testing.T, p CrossViewProjection) CrossViewProjection {
	t.Helper()
	encoded, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	var copyProjection CrossViewProjection
	if err := json.Unmarshal(encoded, &copyProjection); err != nil {
		t.Fatal(err)
	}
	return copyProjection
}

func refingerprintCrossViewForTest(p *CrossViewProjection) {
	p.ProjectionFingerprint = ""
	p.ProjectionFingerprint = crossViewFingerprint(*p)
}

func TestCrossViewPublicAccountingValidatesKnownVocabulary(t *testing.T) {
	p := testCrossViewPublicProjection(t)
	got, err := CrossViewPublicAccounting(p)
	if err != nil {
		t.Fatal(err)
	}
	if got.SourceEdgeCounts["function-table-operation"] != 1 ||
		got.PotentialOperationCombinationCounts["R"] != 1 {
		t.Fatalf("legitimate aggregate lost: %+v", got)
	}
}

func TestCrossViewPublicAccountingRejectsTamperedProjection(t *testing.T) {
	base := testCrossViewPublicProjection(t)
	p := testCrossViewDeepCopy(t, base)
	p.SourceEdges[0].Kind = "private-internal-table-name"
	if _, err := CrossViewPublicAccounting(p); err == nil ||
		!strings.Contains(err.Error(), "fingerprint") {
		t.Fatalf("modified projection was published: %v", err)
	}
}

func TestCrossViewPublicAccountingRejectsForgedPublicationKeys(t *testing.T) {
	base := testCrossViewPublicProjection(t)
	cases := []struct {
		name   string
		mutate func(*CrossViewProjection)
	}{
		{"private edge kind", func(p *CrossViewProjection) {
			p.SourceEdges[0].Kind = "private-internal-table-name"
		}},
		{"private target kind", func(p *CrossViewProjection) {
			p.SourceEdges[0].Kind = "task-function-io-target"
			p.SourceEdges[0].TargetKind = "private-internal-screen"
		}},
		{"private operation", func(p *CrossViewProjection) {
			p.DerivedEdges[0].Operations = "private-business-operation"
		}},
		{"private unresolved reason", func(p *CrossViewProjection) {
			p.Unresolved = append(p.Unresolved, CrossViewUnresolved{
				Code:         "private-internal-exception",
				RelationKind: "function-table-operation",
				SourceTokens: []string{p.Actions[0].ActionToken},
			})
		}},
		{"private token", func(p *CrossViewProjection) {
			p.SourceEdges[0].FromToken = "private-record-id"
		}},
		{"unexpected derivation rule", func(p *CrossViewProjection) {
			p.DerivedEdges[0].DerivationRuleFingerprint = "private-rule"
		}},
		{"fake source digest", func(p *CrossViewProjection) {
			p.SourceArtifactDigest = "private-document"
		}},
		{"old derivation revision", func(p *CrossViewProjection) {
			p.DerivationPolicyRevision = "tei.xeadsource.cross-view/v1"
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := testCrossViewDeepCopy(t, base)
			tc.mutate(&p)
			refingerprintCrossViewForTest(&p)
			if got, err := CrossViewPublicAccounting(p); err == nil {
				t.Fatalf("forged input was published: %+v", got)
			}
		})
	}
}
