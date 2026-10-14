// Command reference-standard-function-candidates projects non-executable function
// proposals from an existing TEI Workspace. It does not create Source or bindings.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/semantic"
	"tei/internal/validation"
	"tei/internal/workspace"
)

const proposalSchema = "tei.standard-function-candidates/v0"

type Field struct {
	ID      core.CanonicalID `json:"id"`
	TypeRef core.CanonicalID `json:"type"`
}

type Candidate struct {
	DerivedID  string                  `json:"derivedId"`
	Kind       application.PatternKind `json:"kind"`
	Intent     string                  `json:"intent"`
	Data       core.CanonicalID        `json:"data"`
	Fields     []Field                 `json:"fields"`
	Executable bool                    `json:"executable"`
	Missing    []string                `json:"missing"`
}

type Report struct {
	SchemaVersion string           `json:"schemaVersion"`
	Application   core.CanonicalID `json:"application"`
	Authority     string           `json:"authority"`
	Candidates    []Candidate      `json:"candidates"`
}

// Project reads only Canonical data and existing Application ownership. Every
// result is a disposable projection; it is never an accepted Application Pattern.
// Query, identity keys, write authority and physical storage cannot be inferred
// from attribute names, types or a successful Workspace validation.
func Project(w ir.World, applicationID core.CanonicalID) (Report, error) {
	report := Report{SchemaVersion: proposalSchema, Application: applicationID, Authority: "proposal-only", Candidates: []Candidate{}}

	var app application.Application
	matches := 0
	for _, candidate := range w.Applications {
		if candidate.ID == applicationID {
			app = candidate
			matches++
		}
	}
	if matches != 1 {
		return Report{}, fmt.Errorf("Application %s must exist exactly once (found %d)", applicationID, matches)
	}

	namespaces := map[core.CanonicalID]semantic.Namespace{}
	duplicateNamespace := map[core.CanonicalID]bool{}
	for _, ns := range w.Semantics {
		if _, ok := namespaces[ns.ID]; ok {
			duplicateNamespace[ns.ID] = true
		}
		namespaces[ns.ID] = ns
	}
	data := map[core.CanonicalID]semantic.Data{}
	for _, use := range app.SemanticUses {
		if duplicateNamespace[use] {
			return Report{}, fmt.Errorf("Semantic namespace %s is ambiguous", use)
		}
		ns, ok := namespaces[use]
		if !ok {
			return Report{}, fmt.Errorf("Semantic namespace %s does not exist", use)
		}
		for _, d := range ns.Data {
			if d.ID == "" {
				return Report{}, fmt.Errorf("Semantic namespace %s contains an empty Data ID", use)
			}
			if _, exists := data[d.ID]; exists {
				return Report{}, fmt.Errorf("Data %s is ambiguous", d.ID)
			}
			data[d.ID] = d
		}
	}

	ordered := make([]string, 0, len(data))
	for id := range data {
		ordered = append(ordered, string(id))
	}
	sort.Strings(ordered)

	for _, id := range ordered {
		d := data[core.CanonicalID(id)]
		fields := make([]Field, 0, len(d.Attributes))
		for _, attr := range d.Attributes {
			fields = append(fields, Field{ID: attr.ID, TypeRef: attr.TypeRef})
		}
		sort.Slice(fields, func(i, j int) bool { return fields[i].ID < fields[j].ID })

		for _, intent := range []struct {
			name    string
			kind    application.PatternKind
			missing []string
		}{
			{"browse", application.PatternBrowse, []string{"query", "datasource", "provider", "authority"}},
			{"record", application.PatternRecord, []string{"read-binding", "provider", "authority"}},
			{"create", application.PatternCommand, []string{"capability", "binding", "persistence", "provider", "authority", "independent-read"}},
			{"update", application.PatternCommand, []string{"capability", "binding", "persistence", "provider", "authority", "independent-read"}},
		} {
			if hasManualPattern(app.Patterns, d.ID, intent.kind) && (intent.name == "browse" || intent.name == "record") {
				continue
			}
			secondaryID := "derived." + string(app.ID) + "." + id + "." + intent.name
			for _, existing := range app.Patterns {
				if string(existing.ID) == secondaryID {
					return Report{}, fmt.Errorf("derived ID %s collides with an explicit Pattern", secondaryID)
				}
			}
			report.Candidates = append(report.Candidates, Candidate{
				DerivedID:  secondaryID,
				Kind:       intent.kind,
				Intent:     intent.name,
				Data:       d.ID,
				Fields:     append([]Field(nil), fields...),
				Executable: false,
				Missing:    append([]string(nil), intent.missing...),
			})
		}
	}
	return report, nil
}

func hasManualPattern(patterns []application.Pattern, data core.CanonicalID, kind application.PatternKind) bool {
	for _, pattern := range patterns {
		if pattern.Kind == kind && pattern.Data == data {
			return true
		}
	}
	return false
}

func run(args []string, output io.Writer) error {
	flags := flag.NewFlagSet("reference-standard-function-candidates", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	root := flags.String("workspace", "", "existing TEI Workspace")
	appName := flags.String("application", "", "Canonical Application ID")
	patternName := flags.String("pattern", "", "explicitly declared Pattern ID (inspect references only)")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 || *root == "" || *appName == "" {
		return fmt.Errorf("usage: reference-standard-function-candidates -workspace PATH -application ID [-pattern EXPLICIT_ID]")
	}
	appID, err := core.ParseCanonicalID(*appName)
	if err != nil {
		return err
	}
	loaded, err := workspace.Load(*root)
	if err != nil {
		return err
	}
	if ds := validation.Validate(loaded.World); validation.HasErrors(ds) {
		return fmt.Errorf("Workspace validation failed: %v", ds)
	}
	encoder := json.NewEncoder(output)
	encoder.SetIndent("", "  ")
	if *patternName != "" {
		patternID, err := core.ParseCanonicalID(*patternName)
		if err != nil {
			return err
		}
		report, err := InspectExplicitPattern(loaded.World, appID, patternID)
		if err != nil {
			return err
		}
		return encoder.Encode(report)
	}
	report, err := Project(loaded.World, appID)
	if err != nil {
		return err
	}
	return encoder.Encode(report)
}

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
