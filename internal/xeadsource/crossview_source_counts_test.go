package xeadsource

import "testing"

func TestCrossViewPublicAccountingRejectsInconsistentSourceTotals(t *testing.T) {
	base := testCrossViewPublicProjection(t)
	cases := []struct {
		name   string
		mutate func(*CrossViewProjection)
	}{
		{"task count below observed tasks", func(p *CrossViewProjection) {
			p.TaskCount = 0
		}},
		{"task function I/O count inflated", func(p *CrossViewProjection) {
			p.TaskFunctionIOCount++
		}},
		{"IOTable count inflated", func(p *CrossViewProjection) {
			p.IOTableCount++
		}},
		{"per-action function I/O count inflated", func(p *CrossViewProjection) {
			p.Actions[0].FunctionIOCount++
		}},
		{"manual candidate contradicts I/O", func(p *CrossViewProjection) {
			p.Actions[0].ManualOrExternalCandidate = true
		}},
		{"extra unresolved IOTable", func(p *CrossViewProjection) {
			p.Unresolved = append(p.Unresolved, CrossViewUnresolved{
				Code:         "iotable-operation-missing",
				RelationKind: "function-table-operation",
				SourceTokens: []string{p.Actions[0].ActionToken},
			})
		}},
		{"mislabeled unresolved relation", func(p *CrossViewProjection) {
			p.Unresolved = append(p.Unresolved, CrossViewUnresolved{
				Code:         "iotable-operation-missing",
				RelationKind: "action-function-use",
				SourceTokens: []string{p.Actions[0].ActionToken},
			})
		}},
		{"orphan task action edge", func(p *CrossViewProjection) {
			for i, edge := range p.SourceEdges {
				if edge.Kind == "task-action" {
					p.SourceEdges = append(p.SourceEdges[:i], p.SourceEdges[i+1:]...)
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
				t.Fatalf("source evidence mismatch was published: %+v", got)
			}
		})
	}
}

func TestCrossViewPublicAccountingKeepsIndependentUnresolvedSources(t *testing.T) {
	p := buildCrossViewPublicFixture(t, `<System>
  <TaskAction ID="orphan-action">
    <TaskFunctionIO FunctionID="unknown-function" IOID="0"/>
  </TaskAction>
  <TaskFunctionIO FunctionID="another-unknown-function" IOID="0"/>
  <IOTable ID="orphan-table" TableID="unknown-table"/>
</System>`)
	got, err := CrossViewPublicAccounting(p)
	if err != nil {
		t.Fatal(err)
	}
	if got.TaskCount != 0 ||
		got.TaskActionCount != 1 ||
		got.TaskFunctionIOCount != 2 ||
		got.IOTableCount != 1 ||
		got.UnresolvedCounts["task-action-without-task"] != 1 ||
		got.UnresolvedCounts["task-function-io-function-not-found"] != 1 ||
		got.UnresolvedCounts["task-function-io-without-task-action"] != 1 ||
		got.UnresolvedCounts["iotable-without-function"] != 1 {
		t.Fatalf("independent unresolved records were not retained: %+v", got)
	}
}

func TestCrossViewPublicAccountingRejectsMissingUnresolvedIOTable(t *testing.T) {
	p := buildCrossViewPublicFixture(t, `<System>
  <Table ID="T1"/>
  <Function ID="F1"><IOTable TableID="T1"/></Function>
</System>`)
	if _, err := CrossViewPublicAccounting(p); err != nil {
		t.Fatalf("genuine unresolved IOTable rejected: %v", err)
	}
	p.Unresolved = nil
	refingerprintCrossViewForTest(&p)
	if got, err := CrossViewPublicAccounting(p); err == nil {
		t.Fatalf("missing unresolved IOTable evidence was published: %+v", got)
	}
}
