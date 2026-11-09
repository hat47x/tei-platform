package productivitybenchmark

import (
	"os"
	"path/filepath"
	"testing"
)

func loadTestPlan(t *testing.T) Plan {
	t.Helper()
	plan, err := LoadPlan(filepath.Join("testdata", "tasks.json"))
	if err != nil {
		t.Fatal(err)
	}
	return plan
}

func TestLoadPlanHasFourTasks(t *testing.T) {
	plan := loadTestPlan(t)
	if len(plan.Tasks) != 4 {
		t.Fatalf("tasks=%d", len(plan.Tasks))
	}
	for _, task := range plan.Tasks {
		if task.Start == "" || task.Completion == "" {
			t.Fatalf("incomplete task: %+v", task)
		}
	}
}

func TestValidatePlanRejectsIncompletePlans(t *testing.T) {
	valid := loadTestPlan(t)
	cases := []struct {
		name string
		edit func(*Plan)
	}{
		{"count", func(p *Plan) { p.Tasks = p.Tasks[:3] }},
		{"id", func(p *Plan) { p.Tasks[0].ID = "" }},
		{"duplicate", func(p *Plan) { p.Tasks[1].ID = p.Tasks[0].ID }},
		{"completion", func(p *Plan) { p.Tasks[0].Completion = "" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			plan := valid
			plan.Tasks = append([]Task(nil), valid.Tasks...)
			tc.edit(&plan)
			if err := ValidatePlan(plan); err == nil {
				t.Fatalf("invalid plan was accepted: %+v", plan)
			}
		})
	}
}

func TestCoverageRequiresAllTaskMethodRuns(t *testing.T) {
	plan := loadTestPlan(t)
	dir := t.TempDir()
	for _, task := range plan.Tasks {
		for _, method := range []string{"conventional", "lowcode", "tei"} {
			for index := 1; index <= 3; index++ {
				run := validRun(method, index, 60)
				run.TaskID = task.ID
				writeRun(t, dir, run)
			}
		}
	}
	coverage, err := CoverageOf(plan, dir)
	if err != nil {
		t.Fatal(err)
	}
	if !coverage.Complete {
		t.Fatalf("coverage is incomplete: %+v", coverage)
	}
}

func TestCoverageReportsMissingRuns(t *testing.T) {
	plan := loadTestPlan(t)
	dir := t.TempDir()
	task := plan.Tasks[0]
	for _, method := range []string{"conventional", "lowcode", "tei"} {
		for index := 1; index <= 3; index++ {
			run := validRun(method, index, 60)
			run.TaskID = task.ID
			writeRun(t, dir, run)
		}
	}
	// Remove one file to leave a gap.
	if err := os.Remove(filepath.Join(dir, task.ID+"-tei-2.json")); err != nil {
		t.Fatal(err)
	}
	coverage, err := CoverageOf(plan, dir)
	if err != nil {
		t.Fatal(err)
	}
	if coverage.Complete {
		t.Fatal("coverage with a missing run was complete")
	}
	found := false
	for _, taskCoverage := range coverage.Tasks {
		if taskCoverage.TaskID != task.ID {
			continue
		}
		for _, method := range taskCoverage.Methods {
			if method.Method == "tei" && len(method.Missing) == 1 && method.Missing[0] == 2 {
				found = true
			}
		}
	}
	if !found {
		t.Fatalf("missing run was not reported: %+v", coverage)
	}
}

func TestCoverageRejectsUnknownTask(t *testing.T) {
	plan := loadTestPlan(t)
	dir := t.TempDir()
	run := validRun("tei", 1, 60)
	run.TaskID = "unknown-task"
	writeRun(t, dir, run)
	if _, err := CoverageOf(plan, dir); err == nil {
		t.Fatal("unknown task run was accepted")
	}
}
