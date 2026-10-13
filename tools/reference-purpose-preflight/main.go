// Command reference-purpose-preflight demonstrates read-only business guard
// preflight with synthetic fixtures or source inspections. It cannot grant execution permission.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"tei/internal/core"
	"tei/internal/workspace"
)

type Fixture struct {
	Cases []Case `json:"cases"`
}

type Report struct {
	SchemaVersion string   `json:"schemaVersion"`
	Authority     string   `json:"authority"`
	Results       []Result `json:"results"`
}

func run(reader io.Reader, writer io.Writer) error {
	decoder := json.NewDecoder(reader)
	decoder.DisallowUnknownFields()
	var fixture Fixture
	if err := decoder.Decode(&fixture); err != nil {
		return fmt.Errorf("decode fixture: %w", err)
	}
	if len(fixture.Cases) == 0 {
		return fmt.Errorf("fixture must contain at least one case")
	}
	var remaining any
	if err := decoder.Decode(&remaining); err != io.EOF {
		return fmt.Errorf("fixture must contain exactly one JSON object")
	}
	report := Report{SchemaVersion: "tei.reference.purpose-preflight/v0", Authority: "fixture-only-no-execution", Results: make([]Result, 0, len(fixture.Cases))}
	for _, c := range fixture.Cases {
		report.Results = append(report.Results, Evaluate(c))
	}
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}

func runCLI(args []string, output io.Writer) error {
	flags := flag.NewFlagSet("reference-purpose-preflight", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	input := flags.String("input", "", "synthetic JSON fixture file (read-only)")
	root := flags.String("workspace", "", "real TEI Workspace (read-only source inspection)")
	application := flags.String("application", "", "Application ID that owns the Pattern")
	kind := flags.String("subject-kind", "", "typed subject kind for Workspace inspection")
	id := flags.String("subject-id", "", "typed subject ID for Workspace inspection")
	decisionIndex := flags.String("decision-index", "", "read-only JSON report from reference-shared-acceptance decisions")
	purpose := flags.String("purpose", "", "development intent: describe|simulate|execute-preflight (read-only)")
	environment := flags.String("environment", "", "Environment ID for exact binding inspection (execute-preflight only)")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("positional arguments are not supported")
	}
	if *environment != "" {
		if *input != "" || *purpose != "execute-preflight" || *kind != "application-pattern" {
			return fmt.Errorf("-environment requires Workspace application-pattern and -purpose execute-preflight")
		}
		if _, err := core.ParseCanonicalID(*environment); err != nil {
			return fmt.Errorf("invalid Environment ID: %w", err)
		}
	}
	if *input != "" && *root == "" && *application == "" && *kind == "" && *id == "" && *decisionIndex == "" && *purpose == "" {
		file, err := os.Open(*input)
		if err != nil {
			return err
		}
		defer file.Close()
		return run(file, output)
	}
	if *input == "" && *root != "" && *application != "" && *kind != "" && *id != "" {
		var assessment WorkspaceAssessment
		var err error
		switch *kind {
		case "application-pattern":
			assessment, err = InspectWorkspace(*root, *application, *kind, *id)
		case "semantic-validation":
			assessment, err = InspectValidation(*root, *application, *id)
		default:
			return fmt.Errorf("unsupported subject kind %q", *kind)
		}
		if err != nil {
			return err
		}
		if *decisionIndex != "" {
			if *kind != "semantic-validation" {
				return fmt.Errorf("-decision-index is supported only for semantic-validation")
			}
			file, err := os.Open(*decisionIndex)
			if err != nil {
				return fmt.Errorf("open Decision Index report: %w", err)
			}
			defer file.Close()
			if err := AttachDecisionHistory(&assessment, file); err != nil {
				return err
			}
		}
		encoder := json.NewEncoder(output)
		encoder.SetIndent("", "  ")
		if *purpose != "" {
			if *environment != "" {
				loaded, loadErr := workspace.Load(*root)
				if loadErr != nil {
					return fmt.Errorf("load scoped Environment: %w", loadErr)
				}
				binding := ObserveEnvironmentBinding(loaded.World, *environment, *application, assessment.ContractQueryUse)
				assessment.EnvironmentBinding = &binding
			}
			projection, err := ProjectForPurpose(assessment, *purpose)
			if err != nil {
				return err
			}
			return encoder.Encode(struct {
				Assessment WorkspaceAssessment `json:"assessment"`
				Readiness  PurposeProjection   `json:"readiness"`
			}{Assessment: assessment, Readiness: projection})
		}
		// Preserve the previous flat Workspace JSON when no purpose is requested.
		// The extra operation evidence is an opt-in development projection only.
		assessment.OperationWriteIntent = nil
		assessment.ContractQueryUse = nil
		assessment.EnvironmentBinding = nil
		return encoder.Encode(assessment)
	}
	return fmt.Errorf("usage: reference-purpose-preflight -input <fixture.json> OR -workspace <path> -application <id> -subject-kind application-pattern|semantic-validation -subject-id <id> [-decision-index <json>] [-purpose describe|simulate|execute-preflight] [-environment <id>]")
}

func main() {
	if err := runCLI(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
