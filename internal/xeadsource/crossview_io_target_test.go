package xeadsource

import "testing"

func TestCrossViewPublicAccountingPreservesKnownAndUnresolvedIOTargets(t *testing.T) {
	p := buildCrossViewPublicFixture(t, `<System>
  <Function ID="F1">
    <IOPanel ID="P1"/>
    <IOSpool ID="S1"/>
  </Function>
  <Task ID="T1">
    <TaskAction ID="A1">
      <TaskFunctionIO FunctionID="F1" IOID="P1"/>
      <TaskFunctionIO FunctionID="F1" IOID="S1"/>
      <TaskFunctionIO FunctionID="F1" IOID="absent"/>
      <TaskFunctionIO FunctionID="F1" IOID=""/>
      <TaskFunctionIO FunctionID="F1" IOID=" 0 "/>
      <TaskFunctionIO FunctionID="unknown" IOID="0"/>
    </TaskAction>
  </Task>
</System>`)
	got, err := CrossViewPublicAccounting(p)
	if err != nil {
		t.Fatal(err)
	}
	if got.TaskFunctionIOCount != 6 ||
		got.TaskFunctionIONoTargetCount != 2 ||
		got.TaskFunctionIOTargetKindCounts["IOPanel"] != 1 ||
		got.TaskFunctionIOTargetKindCounts["IOSpool"] != 1 ||
		got.UnresolvedCounts["task-function-io-target-not-found"] != 1 ||
		got.UnresolvedCounts["task-function-io-target-id-missing"] != 1 ||
		got.UnresolvedCounts["task-function-io-function-not-found"] != 1 {
		t.Fatalf("mixed source I/O target classification changed: %+v", got)
	}
}

func TestCrossViewWhitespaceZeroIsNoTargetAndNoMissingRelation(t *testing.T) {
	for _, value := range []string{"0", " 0 ", "\t0\t"} {
		t.Run(value, func(t *testing.T) {
			p := buildCrossViewPublicFixture(t, `<System>
  <Function ID="F1"><IOPanel ID="P1"/></Function>
  <Task ID="T1"><TaskAction ID="A1">
    <TaskFunctionIO FunctionID="F1" IOID="`+value+`"/>
  </TaskAction></Task>
</System>`)
			got, err := CrossViewPublicAccounting(p)
			if err != nil {
				t.Fatal(err)
			}
			if got.TaskFunctionIONoTargetCount != 1 ||
				got.SourceEdgeCounts["task-function-io-target"] != 0 ||
				len(p.Unresolved) != 0 {
				t.Fatalf("zero sentinel was treated as an interaction: %+v", got)
			}
		})
	}
}

func TestCrossViewPublicAccountingRejectsUngroundedIOTargetOutcomes(t *testing.T) {
	base := buildCrossViewPublicFixture(t, `<System>
  <Function ID="F1"><IOPanel ID="P1"/></Function>
  <Task ID="T1"><TaskAction ID="A1">
    <TaskFunctionIO FunctionID="F1" IOID="P1"/>
  </TaskAction></Task>
</System>`)
	if len(base.SourceEdges) == 0 {
		t.Fatal("source fixture generated no relations")
	}
	cases := []struct {
		name   string
		mutate func(*CrossViewProjection)
	}{
		{"resolved IO target edge removed", func(p *CrossViewProjection) {
			for i, edge := range p.SourceEdges {
				if edge.Kind == "task-function-io-target" {
					p.SourceEdges = append(p.SourceEdges[:i], p.SourceEdges[i+1:]...)
					return
				}
			}
		}},
		{"IOID zero count falsely increased", func(p *CrossViewProjection) {
			p.TaskFunctionIONoTargetCount = 1
		}},
		{"target edge duplicates the same source IO", func(p *CrossViewProjection) {
			for _, edge := range p.SourceEdges {
				if edge.Kind == "task-function-io-target" {
					p.SourceEdges = append(p.SourceEdges, edge)
					return
				}
			}
		}},
		{"resolved target has wrong function token", func(p *CrossViewProjection) {
			for i, edge := range p.SourceEdges {
				if edge.Kind == "task-function-io-target" {
					fake := crossViewDigest("another-function")
					p.SourceEdges[i].SourceTokens = normalizedTokens(edge.FromToken, fake, edge.ToToken)
					return
				}
			}
		}},
		{"resolved target without function use", func(p *CrossViewProjection) {
			for i, edge := range p.SourceEdges {
				if edge.Kind == "action-function-use" {
					p.SourceEdges = append(p.SourceEdges[:i], p.SourceEdges[i+1:]...)
					p.TaskFunctionIOCount = 0
					p.Actions[0].FunctionIOCount = 0
					p.Actions[0].ManualOrExternalCandidate = true
					return
				}
			}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := testCrossViewDeepCopy(t, base)
			tc.mutate(&p)
			refingerprintCrossViewForTest(&p)
			if got, err := CrossViewPublicAccounting(p); err == nil {
				t.Fatalf("unbacked interaction published: %+v", got)
			}
		})
	}
}

func TestCrossViewPublicAccountingRejectsBadUnresolvedTargetProvenance(t *testing.T) {
	base := buildCrossViewPublicFixture(t, `<System>
  <Function ID="F1"><IOPanel ID="P1"/></Function>
  <Task ID="T1"><TaskAction ID="A1">
    <TaskFunctionIO FunctionID="F1" IOID="unknown"/>
  </TaskAction></Task>
</System>`)
	if len(base.Unresolved) != 1 ||
		base.Unresolved[0].Code != "task-function-io-target-not-found" {
		t.Fatalf("unresolved target fixture is invalid: %+v", base.Unresolved)
	}
	cases := []struct {
		name   string
		mutate func(*CrossViewProjection)
	}{
		{"unresolved target evidence removed", func(p *CrossViewProjection) {
			p.Unresolved = nil
		}},
		{"unresolved target duplicates its source outcome", func(p *CrossViewProjection) {
			p.Unresolved = append(p.Unresolved, p.Unresolved[0])
		}},
		{"unresolved target has wrong function token", func(p *CrossViewProjection) {
			ioToken := ""
			for _, edge := range p.SourceEdges {
				if edge.Kind == "action-function-use" {
					for _, token := range edge.SourceTokens {
						if token != edge.FromToken && token != edge.ToToken {
							ioToken = token
						}
					}
				}
			}
			p.Unresolved[0].SourceTokens = normalizedTokens(ioToken, crossViewDigest("wrong-function"))
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := testCrossViewDeepCopy(t, base)
			tc.mutate(&p)
			refingerprintCrossViewForTest(&p)
			if got, err := CrossViewPublicAccounting(p); err == nil {
				t.Fatalf("unbacked unresolved target published: %+v", got)
			}
		})
	}
}
