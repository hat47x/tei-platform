package productivitybenchmark

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func validRun(method string, index int, ttfwc float64) Run {
	return Run{
		Schema: Schema, TaskID: "add-master", Method: method, RunIndex: index,
		StartedAt: "2026-10-19T00:00:00Z", EndedAt: "2026-10-19T00:10:00Z",
		Environment: Environment{OS: "linux", CPU: "x86_64", Memory: "16GB", Network: "offline", Model: "none", Tools: "go1.26"},
		Metrics: Metrics{
			TTFWCSeconds: ttfwc, HumanInterventions: index, RevalidationRequired: 4,
			RevalidationRun: index, AcceptedProposals: 1, RejectedProposals: 0,
			ChangedFiles: 3, GeneratedArtifacts: 1,
		},
		Completion: Completion{Met: true, Evidence: []string{"tests pass"}},
	}
}

func writeRun(t *testing.T, dir string, run Run) {
	t.Helper()
	raw, err := json.MarshalIndent(run, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	name := run.TaskID + "-" + run.Method + "-" + string(rune('0'+run.RunIndex)) + ".json"
	if err := os.WriteFile(filepath.Join(dir, name), append(raw, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestRecordWritesNormalizedRun(t *testing.T) {
	input := filepath.Join(t.TempDir(), "run.json")
	raw, err := json.MarshalIndent(validRun("tei", 1, 60), "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(input, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	out := t.TempDir()
	run, err := Record(input, out)
	if err != nil {
		t.Fatal(err)
	}
	if run.Method != "tei" {
		t.Fatalf("run=%+v", run)
	}
	if _, err := os.Stat(filepath.Join(out, "add-master-tei-1.json")); err != nil {
		t.Fatal(err)
	}
}

func TestValidateRejectsIncompleteOrImpossibleRuns(t *testing.T) {
	cases := []struct {
		name string
		edit func(*Run)
	}{
		{"method", func(r *Run) { r.Method = "magic" }},
		{"runIndex", func(r *Run) { r.RunIndex = 4 }},
		{"negative", func(r *Run) { r.Metrics.HumanInterventions = -1 }},
		{"revalidation", func(r *Run) { r.Metrics.RevalidationRun = 5 }},
		{"ttfwc", func(r *Run) { r.Metrics.TTFWCSeconds = 0 }},
		{"duration", func(r *Run) { r.Metrics.TTFWCSeconds = 700 }},
		{"evidence", func(r *Run) { r.Completion.Evidence = nil }},
		{"environment", func(r *Run) { r.Environment.Model = "" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			run := validRun("tei", 1, 60)
			tc.edit(&run)
			if err := Validate(run); err == nil {
				t.Fatalf("invalid run was accepted: %+v", run)
			}
		})
	}
}

func TestAggregateComputesMedianAndRange(t *testing.T) {
	dir := t.TempDir()
	for _, method := range []string{"conventional", "lowcode", "tei"} {
		for index, ttfwc := range []float64{60, 120, 180} {
			writeRun(t, dir, validRun(method, index+1, ttfwc))
		}
	}
	summary, err := Aggregate(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(summary.Methods) != 3 {
		t.Fatalf("methods=%+v", summary.Methods)
	}
	for _, method := range summary.Methods {
		if method.TTFWC.Median != 120 || method.TTFWC.Min != 60 || method.TTFWC.Max != 180 {
			t.Fatalf("method=%+v", method)
		}
	}
}

func TestAggregateRequiresExactlyThreeRuns(t *testing.T) {
	dir := t.TempDir()
	for _, method := range []string{"conventional", "lowcode", "tei"} {
		for index := 1; index <= 2; index++ {
			writeRun(t, dir, validRun(method, index, 60))
		}
	}
	if _, err := Aggregate(dir); err == nil {
		t.Fatal("aggregate accepted fewer than three runs")
	}
}
