package xeadsource

import (
	"reflect"
	"strings"
	"testing"
)

func crossViewReconciliationFixture(t *testing.T) ([]byte, []byte, CrossViewProjection) {
	t.Helper()
	source := []byte(`<System FormatVersion="1.2">
  <Table ID="T1"/>
  <Function ID="F1"><IOTable TableID="T1" OpR="+"/></Function>
  <Task ID="TASK1">
    <TaskAction ID="A1"><TaskFunctionIO FunctionID="F1" IOID="0"/></TaskAction>
  </Task>
  <Task ID="TASK2"/>
</System>`)
	contract := validCrossViewTranscriptionContract()
	projection, err := BuildCrossViewProjection(FormatXead, source, contract)
	if err != nil {
		t.Fatal(err)
	}
	if projection.TaskFunctionIONoTargetCount != 1 ||
		projection.TaskCount != 2 || len(projection.Actions) != 1 {
		t.Fatalf("bad initial fixture: %+v", projection)
	}
	return source, contract, projection
}

func TestCrossViewSourceReconciledAccountingChecksExactSource(t *testing.T) {
	source, contract, projection := crossViewReconciliationFixture(t)
	got, err := CrossViewSourceReconciledAccounting(FormatXead, source, contract, projection)
	if err != nil {
		t.Fatal(err)
	}
	direct, err := CrossViewPublicAccounting(projection)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, direct) {
		t.Fatalf("valid aggregate differed: reconciled=%+v direct=%+v", got, direct)
	}
	if got.TaskFunctionIONoTargetCount != 1 || got.TaskCount != 2 {
		t.Fatalf("genuine source-only counts were lost: %+v", got)
	}
}

func TestCrossViewSourceReconciledAccountingRejectsRefingerprintedSourceOnlyCountChanges(t *testing.T) {
	source, contract, baseline := crossViewReconciliationFixture(t)
	for _, tc := range []struct {
		name string
		edit func(*CrossViewProjection)
	}{
		{"IOID zero count omitted", func(p *CrossViewProjection) { p.TaskFunctionIONoTargetCount = 0 }},
		{"unused Task invented", func(p *CrossViewProjection) { p.TaskCount = 3 }},
		{"unused Task omitted", func(p *CrossViewProjection) { p.TaskCount = 1 }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := testCrossViewDeepCopy(t, baseline)
			tc.edit(&p)
			refingerprintCrossViewForTest(&p)
			// The ordinary public API intentionally tests projection
			// self-consistency, which cannot reconstruct absent source nodes.
			if _, err := CrossViewPublicAccounting(p); err != nil {
				t.Fatalf("fixture no longer exposes source-only accounting gap: %v", err)
			}
			if got, err := CrossViewSourceReconciledAccounting(FormatXead, source, contract, p); err == nil {
				t.Fatalf("refingerprinted source count was accepted: %+v", got)
			}
		})
	}
}

func TestCrossViewSourceReconciledAccountingRejectsOtherSourceAndContract(t *testing.T) {
	source, contract, baseline := crossViewReconciliationFixture(t)
	cases := []struct {
		name     string
		source   []byte
		contract []byte
	}{
		{"edited source ID", []byte(strings.Replace(string(source), `TASK2`, `TASK3`, 1)), contract},
		{"different source content", append(append([]byte(nil), source...), ' '), contract},
		{"different contract bytes", source, append(append([]byte(nil), contract...), ' ')},
		{"invalid contract", source, []byte(`{}`)},
		{"unsupported format profile", []byte(`<System FormatVersion="9.9"/>`), contract},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got, err := CrossViewSourceReconciledAccounting(FormatXead, tc.source, tc.contract, baseline); err == nil {
				t.Fatalf("source or contract drift was accepted: %+v", got)
			}
		})
	}
}

func TestCrossViewSourceReconciledAccountingDoesNotBypassInternalValidation(t *testing.T) {
	source, contract, projection := crossViewReconciliationFixture(t)
	projection.ProjectionFingerprint = "sha256:" + strings.Repeat("0", 64)
	if _, err := CrossViewSourceReconciledAccounting(FormatXead, source, contract, projection); err == nil {
		t.Fatal("invalid internal fingerprint was accepted")
	}
}
