// Package productivitybenchmark validates and aggregates #2018 productivity
// run records. It does not execute comparisons or invent results.
package productivitybenchmark

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const Schema = "tei.reference.benchmark-run/v0"

var methods = map[string]bool{"conventional": true, "lowcode": true, "tei": true}

type Environment struct {
	OS      string `json:"os"`
	CPU     string `json:"cpu"`
	Memory  string `json:"memory"`
	Network string `json:"network"`
	Model   string `json:"model"`
	Tools   string `json:"tools"`
}

type Metrics struct {
	TTFWCSeconds         float64 `json:"ttfwcSeconds"`
	HumanInterventions   int     `json:"humanInterventions"`
	RevalidationRequired int     `json:"revalidationRequired"`
	RevalidationRun      int     `json:"revalidationRun"`
	AcceptedProposals    int     `json:"acceptedProposals"`
	RejectedProposals    int     `json:"rejectedProposals"`
	ChangedFiles         int     `json:"changedFiles"`
	GeneratedArtifacts   int     `json:"generatedArtifacts"`
}

type Completion struct {
	Met      bool     `json:"met"`
	Evidence []string `json:"evidence"`
}

type Run struct {
	Schema      string      `json:"schema"`
	TaskID      string      `json:"taskId"`
	Method      string      `json:"method"`
	RunIndex    int         `json:"runIndex"`
	StartedAt   string      `json:"startedAt"`
	EndedAt     string      `json:"endedAt"`
	Environment Environment `json:"environment"`
	Metrics     Metrics     `json:"metrics"`
	Completion  Completion  `json:"completion"`
	Unresolved  []string    `json:"unresolved,omitempty"`
}

type Range struct {
	Median float64 `json:"median"`
	Min    float64 `json:"min"`
	Max    float64 `json:"max"`
}

type MethodSummary struct {
	Method               string `json:"method"`
	Runs                 int    `json:"runs"`
	TTFWC                Range  `json:"ttfwcSeconds"`
	HumanInterventions   Range  `json:"humanInterventions"`
	RevalidationRun      Range  `json:"revalidationRun"`
	RevalidationRequired Range  `json:"revalidationRequired"`
}

type Summary struct {
	Schema  string          `json:"schema"`
	TaskID  string          `json:"taskId"`
	Methods []MethodSummary `json:"methods"`
}

func readRun(path string) (Run, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Run{}, err
	}
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.DisallowUnknownFields()
	var run Run
	if err := decoder.Decode(&run); err != nil {
		return Run{}, fmt.Errorf("%s: %w", path, err)
	}
	if err := Validate(run); err != nil {
		return Run{}, fmt.Errorf("%s: %w", path, err)
	}
	return run, nil
}

func Validate(run Run) error {
	if run.Schema != Schema {
		return fmt.Errorf("schema %q is not %q", run.Schema, Schema)
	}
	if strings.TrimSpace(run.TaskID) == "" {
		return fmt.Errorf("taskId is required")
	}
	if !methods[run.Method] {
		return fmt.Errorf("method %q is not conventional, lowcode, or tei", run.Method)
	}
	if run.RunIndex < 1 || run.RunIndex > 3 {
		return fmt.Errorf("runIndex must be 1..3")
	}
	started, err := time.Parse(time.RFC3339, run.StartedAt)
	if err != nil {
		return fmt.Errorf("startedAt must be RFC3339: %w", err)
	}
	ended, err := time.Parse(time.RFC3339, run.EndedAt)
	if err != nil {
		return fmt.Errorf("endedAt must be RFC3339: %w", err)
	}
	if ended.Before(started) {
		return fmt.Errorf("endedAt is before startedAt")
	}
	for name, value := range map[string]string{
		"environment.os": run.Environment.OS, "environment.cpu": run.Environment.CPU,
		"environment.memory": run.Environment.Memory, "environment.network": run.Environment.Network,
		"environment.model": run.Environment.Model, "environment.tools": run.Environment.Tools,
	} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s is required", name)
		}
	}
	if run.Metrics.TTFWCSeconds <= 0 {
		return fmt.Errorf("ttfwcSeconds must be positive")
	}
	if run.Metrics.TTFWCSeconds > ended.Sub(started).Seconds()+1 {
		return fmt.Errorf("ttfwcSeconds exceeds the recorded duration")
	}
	nonNegative := map[string]int{
		"humanInterventions": run.Metrics.HumanInterventions, "revalidationRequired": run.Metrics.RevalidationRequired,
		"revalidationRun": run.Metrics.RevalidationRun, "acceptedProposals": run.Metrics.AcceptedProposals,
		"rejectedProposals": run.Metrics.RejectedProposals, "changedFiles": run.Metrics.ChangedFiles,
		"generatedArtifacts": run.Metrics.GeneratedArtifacts,
	}
	for name, value := range nonNegative {
		if value < 0 {
			return fmt.Errorf("%s must not be negative", name)
		}
	}
	if run.Metrics.RevalidationRun > run.Metrics.RevalidationRequired {
		return fmt.Errorf("revalidationRun exceeds revalidationRequired")
	}
	if run.Completion.Met && len(run.Completion.Evidence) == 0 {
		return fmt.Errorf("completion met requires evidence")
	}
	return nil
}

func Record(inputPath, outDir string) (Run, error) {
	run, err := readRun(inputPath)
	if err != nil {
		return Run{}, err
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return Run{}, err
	}
	name := fmt.Sprintf("%s-%s-%d.json", safeName(run.TaskID), run.Method, run.RunIndex)
	raw, err := json.MarshalIndent(run, "", "  ")
	if err != nil {
		return Run{}, err
	}
	if err := os.WriteFile(filepath.Join(outDir, name), append(raw, '\n'), 0o644); err != nil {
		return Run{}, err
	}
	return run, nil
}

func safeName(value string) string {
	value = strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' {
			return r
		}
		return '-'
	}, value)
	return strings.Trim(value, "-")
}

func median(values []float64) float64 {
	sort.Float64s(values)
	middle := len(values) / 2
	if len(values)%2 == 0 {
		return (values[middle-1] + values[middle]) / 2
	}
	return values[middle]
}

func summarize(values []int) Range {
	floats := make([]float64, len(values))
	for i, value := range values {
		floats[i] = float64(value)
	}
	return rangeOf(floats)
}

func rangeOf(values []float64) Range {
	if len(values) == 0 {
		return Range{}
	}
	sorted := append([]float64(nil), values...)
	sort.Float64s(sorted)
	return Range{Median: median(append([]float64(nil), sorted...)), Min: sorted[0], Max: sorted[len(sorted)-1]}
}

func Aggregate(dir string) (Summary, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return Summary{}, err
	}
	type key struct{ task, method string }
	groups := map[key][]Run{}
	tasks := map[string]bool{}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		run, err := readRun(filepath.Join(dir, entry.Name()))
		if err != nil {
			return Summary{}, err
		}
		tasks[run.TaskID] = true
		groups[key{run.TaskID, run.Method}] = append(groups[key{run.TaskID, run.Method}], run)
	}
	if len(tasks) != 1 {
		return Summary{}, fmt.Errorf("aggregate expects runs for exactly one task, found %d", len(tasks))
	}
	var taskID string
	for id := range tasks {
		taskID = id
	}
	var summaries []MethodSummary
	for method := range methods {
		runs := groups[key{taskID, method}]
		if len(runs) != 3 {
			return Summary{}, fmt.Errorf("method %s has %d runs; exactly 3 are required", method, len(runs))
		}
		ttfwc := make([]float64, len(runs))
		interventions := make([]int, len(runs))
		runScope := make([]int, len(runs))
		requiredScope := make([]int, len(runs))
		for i, run := range runs {
			ttfwc[i] = run.Metrics.TTFWCSeconds
			interventions[i] = run.Metrics.HumanInterventions
			runScope[i] = run.Metrics.RevalidationRun
			requiredScope[i] = run.Metrics.RevalidationRequired
		}
		summaries = append(summaries, MethodSummary{
			Method: method, Runs: len(runs), TTFWC: rangeOf(ttfwc),
			HumanInterventions: summarize(interventions), RevalidationRun: summarize(runScope),
			RevalidationRequired: summarize(requiredScope),
		})
	}
	sort.Slice(summaries, func(i, j int) bool { return summaries[i].Method < summaries[j].Method })
	return Summary{Schema: "tei.reference.benchmark-summary/v0", TaskID: taskID, Methods: summaries}, nil
}
