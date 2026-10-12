package xeadsource

import "testing"

func TestCrossViewPublicAccountingRejectsUnsupportedDerivedPaths(t *testing.T) {
	base := testCrossViewPublicProjection(t)
	if len(base.DerivedEdges) != 1 {
		t.Fatalf("expected one representative derived path, got %d", len(base.DerivedEdges))
	}
	cases := []struct {
		name   string
		mutate func(*CrossViewProjection)
	}{
		{"missing derived path", func(p *CrossViewProjection) {
			p.DerivedEdges = nil
		}},
		{"duplicate derived path", func(p *CrossViewProjection) {
			p.DerivedEdges = append(p.DerivedEdges, p.DerivedEdges[0])
		}},
		{"unrelated but valid destination token", func(p *CrossViewProjection) {
			p.DerivedEdges[0].ToToken = crossViewDigest("unrelated-table")
			p.DerivedEdges[0].SourceTokens = normalizedTokens(append(
				p.DerivedEdges[0].SourceTokens, p.DerivedEdges[0].ToToken,
			)...)
		}},
		{"supported but ungrounded CRUD", func(p *CrossViewProjection) {
			p.DerivedEdges[0].Operations = "U"
		}},
		{"incomplete source token union", func(p *CrossViewProjection) {
			found := false
			for i, token := range p.DerivedEdges[0].SourceTokens {
				if token == p.SourceEdges[1].SourceTokens[1] &&
					token != p.DerivedEdges[0].FromToken &&
					token != p.DerivedEdges[0].ToToken {
					p.DerivedEdges[0].SourceTokens = append(
						p.DerivedEdges[0].SourceTokens[:i],
						p.DerivedEdges[0].SourceTokens[i+1:]...,
					)
					found = true
					break
				}
			}
			if !found {
				panic("fixture has no removable intermediate provenance token")
			}
		}},
		{"missing function table declaration", func(p *CrossViewProjection) {
			for i, edge := range p.SourceEdges {
				if edge.Kind == "function-table-operation" {
					p.SourceEdges = append(p.SourceEdges[:i], p.SourceEdges[i+1:]...)
					break
				}
			}
		}},
		{"source CRUD changed without rederivation", func(p *CrossViewProjection) {
			for i := range p.SourceEdges {
				if p.SourceEdges[i].Kind == "function-table-operation" {
					p.SourceEdges[i].Operations = "U"
				}
			}
		}},
		{"task action edge does not match observed task", func(p *CrossViewProjection) {
			for i := range p.SourceEdges {
				if p.SourceEdges[i].Kind == "task-action" {
					p.SourceEdges[i].ToToken = crossViewDigest("unobserved-action")
					p.SourceEdges[i].SourceTokens = normalizedTokens(
						p.SourceEdges[i].FromToken, p.SourceEdges[i].ToToken,
					)
				}
			}
		}},
		{"duplicate action observation", func(p *CrossViewProjection) {
			p.Actions = append(p.Actions, p.Actions[0])
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := testCrossViewDeepCopy(t, base)
			tc.mutate(&p)
			// The attacker can recalculate an ordinary hash. This must still fail.
			refingerprintCrossViewForTest(&p)
			if got, err := CrossViewPublicAccounting(p); err == nil {
				t.Fatalf("ungrounded path was published: %+v", got)
			}
		})
	}
}

func TestCrossViewPublicAccountingPreservesManyToManyAndUnresolvedEvidence(t *testing.T) {
	projection := buildCrossViewPublicFixture(t, `<System>
  <Table ID="T1"/>
  <Function ID="F1">
    <IOTable ID="IR" TableID="T1" OpR="+"/>
    <IOTable ID="IU" TableID="T1" OpU="+"/>
    <IOTable ID="IN" TableID="T1"/>
  </Function>
  <Task ID="TASK1">
    <TaskAction ID="A1">
      <TaskFunctionIO FunctionID="F1" IOID="0"/>
      <TaskFunctionIO FunctionID="F1" IOID="0"/>
    </TaskAction>
  </Task>
</System>`)
	got, err := CrossViewPublicAccounting(projection)
	if err != nil {
		t.Fatal(err)
	}
	if got.TaskFunctionIOCount != 2 ||
		got.DistinctActionFunctionPairCount != 1 ||
		got.PotentialActionFunctionTablePathCount != 2 ||
		got.PotentialOperationCombinationCounts["R"] != 1 ||
		got.PotentialOperationCombinationCounts["U"] != 1 ||
		got.UnresolvedCounts["iotable-operation-missing"] != 1 {
		t.Fatalf("source-supported mixed projection changed: %+v", got)
	}
}
