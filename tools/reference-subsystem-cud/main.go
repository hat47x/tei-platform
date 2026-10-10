// Command reference-subsystem-cud checks explicitly declared cross-view intent.
// It never promotes input declarations to runtime-effect authority.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
)

const factsSchema = "subsystem-cud-candidate-v1"
const authorityLimit = "candidate declarations only; not accepted ownership, actual CRUD effects, transaction behavior, or authorization"

// World is a deliberately read-only projection of tei world JSON. Unknown
// fields in the compiler's World are preserved by the source, not this tool.
// The adapter does not make any authority claim about a caller-supplied file.
type World struct {
	Semantics []struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	} `json:"semantics"`
	Applications []struct {
		Patterns []struct {
			ID string `json:"id"`
		} `json:"patterns"`
	} `json:"applications"`
}

type Owner struct {
	Data      string `json:"data"`
	Subsystem string `json:"subsystem"`
	Source    string `json:"source"`
}

type FunctionIO struct {
	Pattern   string `json:"pattern"`
	Subsystem string `json:"subsystem"`
	Data      string `json:"data"`
	Operation string `json:"operation"` // C, R, U, D: declared intent only
	Source    string `json:"source"`
}

type ProcedureStep struct {
	ID          string `json:"id"`
	Subsystem   string `json:"subsystem"`
	Realization string `json:"realization"` // system, manual, external
	Pattern     string `json:"pattern,omitempty"`
	Source      string `json:"source"`
}

type Facts struct {
	SchemaVersion  string          `json:"schemaVersion"`
	Owners         []Owner         `json:"owners"`
	FunctionIO     []FunctionIO    `json:"functionIO"`
	ProcedureSteps []ProcedureStep `json:"procedureSteps"`
}

type Diagnostic struct {
	Code        string `json:"code"`
	Severity    string `json:"severity"`
	Subject     string `json:"subject"`
	Source      string `json:"source,omitempty"`
	Explanation string `json:"explanation"`
}

type Coverage struct {
	WorldData       int `json:"worldData"`
	WorldPatterns   int `json:"worldPatterns"`
	OwnerFacts      int `json:"ownerFacts"`
	FunctionIOFacts int `json:"functionIOFacts"`
	ProcedureSteps  int `json:"procedureSteps"`
}

type Report struct {
	SchemaVersion  string       `json:"schemaVersion"`
	AuthorityLimit string       `json:"authorityLimit"`
	Status         string       `json:"status"`
	Coverage       Coverage     `json:"coverage"`
	Diagnostics    []Diagnostic `json:"diagnostics"`
}

func Inspect(w World, f Facts) Report {
	r := Report{SchemaVersion: factsSchema, AuthorityLimit: authorityLimit, Diagnostics: []Diagnostic{}}
	add := func(code, severity, subject, source, explanation string) {
		r.Diagnostics = append(r.Diagnostics, Diagnostic{code, severity, subject, source, explanation})
	}
	if f.SchemaVersion != factsSchema {
		add("UNSUPPORTED_FACTS_SCHEMA", "unresolved", "schema", "", "facts schema must be "+factsSchema)
		r.Status = "unresolved"
		return r
	}

	dataCount, patternCount := map[string]int{}, map[string]int{}
	for _, ns := range w.Semantics {
		for _, d := range ns.Data {
			dataCount[d.ID]++
		}
	}
	for _, a := range w.Applications {
		for _, p := range a.Patterns {
			patternCount[p.ID]++
		}
	}
	r.Coverage = Coverage{WorldData: len(dataCount), WorldPatterns: len(patternCount), OwnerFacts: len(f.Owners), FunctionIOFacts: len(f.FunctionIO), ProcedureSteps: len(f.ProcedureSteps)}
	if len(f.Owners) == 0 && len(f.FunctionIO) == 0 && len(f.ProcedureSteps) == 0 {
		add("EMPTY_CANDIDATE_SCOPE", "unresolved", "facts", "", "at least one explicit declaration is required before a clean report is possible")
	}
	if len(w.Semantics) == 0 || len(dataCount) == 0 {
		add("WORLD_DATA_UNAVAILABLE", "unresolved", "world", "", "semantic Data must be read from a specified World; do not infer it from fact declarations")
	}
	for id, count := range dataCount {
		if id == "" || count != 1 {
			add("AMBIGUOUS_CANONICAL_DATA", "unresolved", id, "", "world Data identity is empty or occurs multiple times")
		}
	}
	for id, count := range patternCount {
		if id == "" || count != 1 {
			add("AMBIGUOUS_APPLICATION_PATTERN", "unresolved", id, "", "world Pattern identity is empty or occurs multiple times")
		}
	}

	owners := map[string][]Owner{}
	for _, o := range f.Owners {
		if o.Data == "" || o.Subsystem == "" || o.Source == "" {
			add("INCOMPLETE_OWNER_DECLARATION", "unresolved", o.Data, o.Source, "owner fact requires Data, Subsystem, and source evidence")
			continue
		}
		if dataCount[o.Data] != 1 {
			add("OWNER_DATA_UNRESOLVED", "unresolved", o.Data, o.Source, "owner refers to missing or ambiguous World Data")
			continue
		}
		owners[o.Data] = append(owners[o.Data], o)
	}
	for id, os := range owners {
		if len(os) != 1 {
			add("AMBIGUOUS_DATA_OWNER", "unresolved", id, "", "multiple owner declarations are not silently collapsed, even if they agree")
		}
	}
	for _, op := range f.FunctionIO {
		subject := op.Pattern + " -> " + op.Data
		if op.Source == "" || op.Subsystem == "" || op.Pattern == "" || op.Data == "" {
			add("INCOMPLETE_IO_DECLARATION", "unresolved", subject, op.Source, "declared I/O requires source, subsystem, Pattern and Data")
			continue
		}
		if op.Operation != "C" && op.Operation != "R" && op.Operation != "U" && op.Operation != "D" {
			add("UNKNOWN_OPERATION", "unresolved", subject, op.Source, "supported declared operations are C, R, U and D")
			continue
		}
		if patternCount[op.Pattern] != 1 {
			add("IO_PATTERN_UNRESOLVED", "unresolved", subject, op.Source, "Pattern must resolve to exactly one World Pattern")
		}
		if dataCount[op.Data] != 1 {
			add("IO_DATA_UNRESOLVED", "unresolved", subject, op.Source, "Data must resolve to exactly one World Data")
			continue
		}
		matches := owners[op.Data]
		if len(matches) != 1 {
			add("IO_OWNER_UNRESOLVED", "unresolved", subject, op.Source, "cannot check ownership without exactly one explicit owner")
			continue
		}
		if op.Operation != "R" && matches[0].Subsystem != op.Subsystem {
			add("CROSS_SUBSYSTEM_WRITE_INTENT", "review-needed", subject, op.Source, "declared write crosses Data owner boundary; use an explicit owner-side function or review an exception")
		}
	}
	stepIDs := map[string]int{}
	for _, step := range f.ProcedureSteps {
		if step.Source == "" || step.Subsystem == "" || step.ID == "" {
			add("INCOMPLETE_PROCEDURE_STEP", "unresolved", step.ID, step.Source, "step requires ID, subsystem and source evidence")
			continue
		}
		stepIDs[step.ID]++
		switch step.Realization {
		case "manual", "external":
			if step.Pattern != "" {
				add("MANUAL_PATTERN_CONFLICT", "unresolved", step.ID, step.Source, "manual/external step may not claim a system Pattern without explicit realization")
			}
		case "system":
			if step.Pattern == "" || patternCount[step.Pattern] != 1 {
				add("STEP_PATTERN_UNRESOLVED", "unresolved", step.ID, step.Source, "system step requires exactly one World Pattern")
			}
		default:
			add("UNKNOWN_REALIZATION", "unresolved", step.ID, step.Source, "realization must be manual, external or system")
		}
	}
	for id, count := range stepIDs {
		if count > 1 {
			add("AMBIGUOUS_PROCEDURE_STEP", "unresolved", id, "", "step ID is not unique; do not infer an order-based durable identity")
		}
	}

	sort.Slice(r.Diagnostics, func(i, j int) bool {
		a, b := r.Diagnostics[i], r.Diagnostics[j]
		for k, v := range []string{a.Code, a.Subject, a.Source, a.Explanation} {
			y := []string{b.Code, b.Subject, b.Source, b.Explanation}[k]
			if v != y {
				return v < y
			}
		}
		return false
	})
	r.Status = "no-findings-in-declared-scope"
	for _, d := range r.Diagnostics {
		if d.Severity == "unresolved" {
			r.Status = "unresolved"
			break
		}
		if d.Severity == "review-needed" {
			r.Status = "review-needed"
		}
	}
	return r
}

func main() {
	worldPath := flag.String("world", "", "path to tei world JSON (read-only)")
	factsPath := flag.String("facts", "", "path to candidate declared cross-view facts JSON")
	flag.Parse()
	if strings.TrimSpace(*worldPath) == "" || strings.TrimSpace(*factsPath) == "" {
		fmt.Fprintln(os.Stderr, "both -world and -facts are required")
		os.Exit(2)
	}
	worldBytes, err := os.ReadFile(*worldPath)
	if err != nil {
		fail(err)
	}
	factsBytes, err := os.ReadFile(*factsPath)
	if err != nil {
		fail(err)
	}
	var w World
	if err := json.Unmarshal(worldBytes, &w); err != nil {
		fail(err)
	}
	var f Facts
	if err := json.Unmarshal(factsBytes, &f); err != nil {
		fail(err)
	}
	out := Inspect(w, f)
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		fail(err)
	}
	// A nonempty unresolved set is not a valid acceptance result. Findings
	// needing review are also non-PASS, but the CLI emits their full JSON.
	if out.Status != "no-findings-in-declared-scope" {
		os.Exit(1)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(2)
}
