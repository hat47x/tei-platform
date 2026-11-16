package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"tei/internal/businessprocedure"
	"tei/internal/workspace"
)

// businessProcedureResponse is the envelope for `tei business-procedure`.
type businessProcedureResponse struct {
	SchemaVersion string                        `json:"schemaVersion"`
	OK            bool                          `json:"ok"`
	OKScope       string                        `json:"okScope,omitempty"`
	Result        *businessprocedure.Projection `json:"result,omitempty"`
	Error         *cliInspectionFailure         `json:"error,omitempty"`
	// Workspace names the Workspace whose Source Index resolved the references,
	// so a reader can tell which Workspace a projection came from.
	Workspace *workspace.WorkspaceIdentity `json:"workspace,omitempty"`
}

func writeBusinessProcedureResponse(out io.Writer, response businessProcedureResponse, exit int) (int, error) {
	encoder := json.NewEncoder(out)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(response); err != nil {
		return 2, err
	}
	return exit, nil
}

func failBusinessProcedure(code, message string, exit int, out io.Writer) (int, error) {
	return writeBusinessProcedureResponse(out, businessProcedureResponse{
		SchemaVersion: "tei.cli.business-procedure/v0",
		Error:         &cliInspectionFailure{Code: code, Message: message},
	}, exit)
}

// businessProcedureFlagSet is the whole grammar of the command. It parses its
// own arguments, so help and parsing read one definition.
func businessProcedureFlagSet(args *businessProcedureArgs) *flag.FlagSet {
	fs := flag.NewFlagSet("business-procedure", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&args.workspace, "workspace", "", "TEI workspace directory that declares the referenced identities")
	fs.StringVar(&args.procedure, "procedure", "", "path to a one-level business procedure reference JSON")
	return fs
}

type businessProcedureArgs struct {
	workspace string
	procedure string
}

// runBusinessProcedureCommand projects a one-level business procedure read-only.
//
// It resolves each step's Function reference against the Workspace's own Source
// Index, so a reference reaches one declared target and that target's Source, or
// is reported unresolved. It writes nothing.
func runBusinessProcedureCommand(argv []string, out io.Writer) (int, error) {
	var args businessProcedureArgs
	fs := businessProcedureFlagSet(&args)
	if err := fs.Parse(argv); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printHelp(out, fs, "business-procedure")
			return 0, nil
		}
		return failBusinessProcedure("INVALID_ARGUMENT", err.Error(), 2, out)
	}
	if fs.NArg() != 0 {
		return failBusinessProcedure("INVALID_ARGUMENT",
			"business-procedure does not accept positional arguments", 2, out)
	}
	// This command parses its own FlagSet, so the shared pass that refuses a
	// repeated single-value flag does not run for it (#3675).
	if err := validateRepeatableArguments("business-procedure", argv); err != nil {
		return failBusinessProcedure("INVALID_ARGUMENT", err.Error(), 2, out)
	}
	if strings.TrimSpace(args.procedure) == "" {
		return failBusinessProcedure("INVALID_ARGUMENT",
			"business-procedure requires -procedure", 2, out)
	}
	if strings.TrimSpace(args.workspace) == "" {
		return failBusinessProcedure("INVALID_ARGUMENT",
			"business-procedure requires -workspace", 2, out)
	}

	fixture, err := readBusinessProcedureFixture(args.procedure)
	if err != nil {
		return failBusinessProcedure("PROCEDURE_LOAD_FAILED", err.Error(), 2, out)
	}
	if fixture.SchemaVersion != businessprocedure.FixtureSchemaVersion {
		return failBusinessProcedure("PROCEDURE_SCHEMA_UNSUPPORTED",
			fmt.Sprintf("procedure schemaVersion %q is not %q",
				fixture.SchemaVersion, businessprocedure.FixtureSchemaVersion), 2, out)
	}
	if fixture.ArtifactRole != businessprocedure.FixtureArtifactRole {
		return failBusinessProcedure("PROCEDURE_ARTIFACT_ROLE_UNSUPPORTED",
			fmt.Sprintf("procedure artifactRole %q is not %q",
				fixture.ArtifactRole, businessprocedure.FixtureArtifactRole), 2, out)
	}

	loaded, err := workspace.Load(args.workspace)
	if err != nil {
		return failBusinessProcedure(resultCodeWorkspaceLoadFailed, err.Error(), 2, out)
	}
	declared, err := businessprocedure.DeclaredFromWorkspace(args.workspace, loaded)
	if err != nil {
		return failBusinessProcedure(resultCodeSourceIndexFailed, err.Error(), 2, out)
	}

	projection := businessprocedure.Project(
		fixture,
		businessprocedure.Fingerprint(fixture),
		businessprocedure.NewResolver(declared),
	)

	// The envelope's ok reports that the projection was produced, not that the
	// procedure is accepted or that its diagnostics are absent.
	response := businessProcedureResponse{
		SchemaVersion: "tei.cli.business-procedure/v0",
		OK:            true,
		OKScope:       "projection-produced-only",
		Result:        &projection,
	}
	response.Workspace = workspaceIdentityPointer(loaded)
	return writeBusinessProcedureResponse(out, response, 0)
}

// readBusinessProcedureFixture decodes the procedure document strictly. An
// unknown member is refused rather than dropped: a misspelling of
// `manualInstruction` must not read as a manual step that has no instruction.
func readBusinessProcedureFixture(path string) (businessprocedure.Fixture, error) {
	file, err := os.Open(path)
	if err != nil {
		return businessprocedure.Fixture{}, err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	var fixture businessprocedure.Fixture
	if err := decoder.Decode(&fixture); err != nil {
		return businessprocedure.Fixture{}, err
	}
	// A second JSON document after the first is a different input, not more of
	// this one.
	if _, err := decoder.Token(); err != io.EOF {
		return businessprocedure.Fixture{}, fmt.Errorf("%s holds more than one JSON document", path)
	}
	return fixture, nil
}
