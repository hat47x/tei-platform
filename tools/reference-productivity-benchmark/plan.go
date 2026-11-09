package productivitybenchmark

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const PlanSchema = "tei.reference.benchmark-plan/v0"

type Task struct {
	ID         string   `json:"id"`
	Title      string   `json:"title"`
	Start      string   `json:"start"`
	Completion string   `json:"completion"`
	Forbidden  []string `json:"forbidden,omitempty"`
}

type Plan struct {
	Schema string `json:"schema"`
	Tasks  []Task `json:"tasks"`
}

type TaskCoverage struct {
	TaskID  string           `json:"taskId"`
	Methods []MethodCoverage `json:"methods"`
}

type MethodCoverage struct {
	Method  string `json:"method"`
	Runs    []int  `json:"runs"`
	Missing []int  `json:"missing"`
}

type Coverage struct {
	Schema   string         `json:"schema"`
	Tasks    []TaskCoverage `json:"tasks"`
	Complete bool           `json:"complete"`
}

func LoadPlan(path string) (Plan, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Plan{}, err
	}
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.DisallowUnknownFields()
	var plan Plan
	if err := decoder.Decode(&plan); err != nil {
		return Plan{}, fmt.Errorf("%s: %w", path, err)
	}
	if err := ValidatePlan(plan); err != nil {
		return Plan{}, err
	}
	return plan, nil
}

func ValidatePlan(plan Plan) error {
	if plan.Schema != PlanSchema {
		return fmt.Errorf("plan schema %q is not %q", plan.Schema, PlanSchema)
	}
	if len(plan.Tasks) != 4 {
		return fmt.Errorf("the #2018 plan requires exactly 4 tasks, found %d", len(plan.Tasks))
	}
	seen := map[string]bool{}
	for _, task := range plan.Tasks {
		if strings.TrimSpace(task.ID) == "" || strings.TrimSpace(task.Title) == "" ||
			strings.TrimSpace(task.Start) == "" || strings.TrimSpace(task.Completion) == "" {
			return fmt.Errorf("task id, title, start, and completion are required")
		}
		if seen[task.ID] {
			return fmt.Errorf("duplicate task id %q", task.ID)
		}
		seen[task.ID] = true
	}
	return nil
}

func CoverageOf(plan Plan, runsDir string) (Coverage, error) {
	entries, err := os.ReadDir(runsDir)
	if err != nil {
		return Coverage{}, err
	}
	known := map[string]bool{}
	for _, task := range plan.Tasks {
		known[task.ID] = true
	}
	counts := map[string]map[string]map[int]bool{}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		run, err := readRun(filepath.Join(runsDir, entry.Name()))
		if err != nil {
			return Coverage{}, err
		}
		if !known[run.TaskID] {
			return Coverage{}, fmt.Errorf("run %s targets unknown task %q", entry.Name(), run.TaskID)
		}
		if counts[run.TaskID] == nil {
			counts[run.TaskID] = map[string]map[int]bool{}
		}
		if counts[run.TaskID][run.Method] == nil {
			counts[run.TaskID][run.Method] = map[int]bool{}
		}
		counts[run.TaskID][run.Method][run.RunIndex] = true
	}
	coverage := Coverage{Schema: "tei.reference.benchmark-coverage/v0", Complete: true}
	for _, task := range plan.Tasks {
		taskCoverage := TaskCoverage{TaskID: task.ID}
		for _, method := range []string{"conventional", "lowcode", "tei"} {
			entry := MethodCoverage{Method: method, Runs: []int{}, Missing: []int{}}
			for index := 1; index <= 3; index++ {
				if counts[task.ID][method][index] {
					entry.Runs = append(entry.Runs, index)
				} else {
					entry.Missing = append(entry.Missing, index)
				}
			}
			if len(entry.Missing) > 0 {
				coverage.Complete = false
			}
			taskCoverage.Methods = append(taskCoverage.Methods, entry)
		}
		coverage.Tasks = append(coverage.Tasks, taskCoverage)
	}
	sort.Slice(coverage.Tasks, func(i, j int) bool { return coverage.Tasks[i].TaskID < coverage.Tasks[j].TaskID })
	return coverage, nil
}
