package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"

	"tei/internal/ir"
	"tei/internal/operations"
	"tei/internal/workspace"
)

// modelDDLCLIResponse is the envelope for `model ddl`. It keeps its own schema
// name because its result is generated DDL, not a catalog, a location or a
// table of contents.
type modelDDLCLIResponse struct {
	SchemaVersion string                     `json:"schemaVersion"`
	OK            bool                       `json:"ok"`
	Result        *operations.ModelDDLResult `json:"result,omitempty"`
	Error         *cliInspectionFailure      `json:"error,omitempty"`
}

const modelDDLSchemaVersion = "tei.cli.model-ddl/v0"

// runModelDDLCommand emits the PostgreSQL DDL the model's persistence mappings
// declare, one CREATE TABLE (and its indexes) per table. It is read-only: it
// reads the declared World, renders text and never touches a database. It shares
// the operations layer with the other model commands, so a Workspace and a
// `tei world` file answer through the same projection (RQ path item 4: SQL a
// first-time user can run to create the tables a definition implies).
func runModelDDLCommand(args []string, out io.Writer) (int, error) {
	response := modelDDLCLIResponse{SchemaVersion: modelDDLSchemaVersion}
	fail := func(code string, message string, exit int) (int, error) {
		response.Error = &cliInspectionFailure{Code: code, Message: message}
		return writeModelDDLResponse(out, response, exit)
	}
	fs := flag.NewFlagSet("model ddl", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	root := fs.String("workspace", "", "TEI workspace directory")
	worldPath := fs.String("world", "", "normalized M0 IR JSON (a `tei world` output)")
	format := fs.String("format", operations.ModelDDLFormatSQL, "output emphasis: "+operations.ModelDDLFormatSQL+"|"+operations.ModelDDLFormatJSON)
	subjectKind := fs.String("subject-kind", "", "optional exact subject kind: "+operations.ModelDDLSubjectKind)
	subjectID := fs.String("subject-id", "", "optional exact semantic-data element ID; with -subject-kind emits only that table")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printHelp(out, fs, "model ddl")
			return 0, nil
		}
		return fail("INVALID_ARGUMENT", err.Error(), 2)
	}
	if err := validateRepeatableArguments("model ddl", args); err != nil {
		return fail("INVALID_ARGUMENT", err.Error(), 2)
	}
	if name := emptyStringFlag(fs, emptyStringFlagsNotAllowed["model ddl"]); name != "" {
		return fail("INVALID_ARGUMENT", fmt.Sprintf("model ddl: -%s must not be an empty string", name), 2)
	}
	if fs.NArg() != 0 {
		return fail("INVALID_ARGUMENT", "model ddl does not take positional arguments", 2)
	}
	if (*root == "") == (*worldPath == "") {
		return fail("INVALID_ARGUMENT", "model ddl requires exactly one of -workspace or -world", 2)
	}
	if *root != "" && (*root != strings.TrimSpace(*root)) {
		return fail("INVALID_ARGUMENT", "model ddl requires an exact -workspace without surrounding whitespace", 2)
	}
	if *worldPath != "" && (*worldPath != strings.TrimSpace(*worldPath)) {
		return fail("INVALID_ARGUMENT", "model ddl requires an exact -world without surrounding whitespace", 2)
	}
	switch *format {
	case operations.ModelDDLFormatSQL, operations.ModelDDLFormatJSON:
	default:
		return fail("INVALID_ARGUMENT", fmt.Sprintf("model ddl: -format must be %s or %s", operations.ModelDDLFormatSQL, operations.ModelDDLFormatJSON), 2)
	}
	if (*subjectKind == "") != (*subjectID == "") {
		return fail("INVALID_ARGUMENT", "model ddl: -subject-kind and -subject-id must be given together", 2)
	}
	if *subjectKind != "" && *subjectKind != operations.ModelDDLSubjectKind {
		return fail("INVALID_ARGUMENT", fmt.Sprintf("model ddl: -subject-kind must be %s", operations.ModelDDLSubjectKind), 2)
	}

	options := operations.ModelDDLOptions{
		Format:      *format,
		SubjectKind: *subjectKind,
		SubjectID:   *subjectID,
	}
	var world ir.World
	if *root != "" {
		loaded, err := workspace.Load(*root)
		if err != nil {
			return fail("WORKSPACE_LOAD_FAILED", err.Error(), 2)
		}
		idx, err := workspace.BuildSourceIndex(*root, loaded)
		if err != nil {
			return fail("SOURCE_INDEX_FAILED", err.Error(), 2)
		}
		_, sourceFP, err := operations.WorkspaceProjectionFingerprints(loaded.World, idx)
		if err != nil {
			return fail("SOURCE_INDEX_FAILED", err.Error(), 2)
		}
		options.SourceIndexFingerprint = sourceFP
		world = loaded.World
	} else {
		loaded, err := loadWorld(*worldPath, "")
		if err != nil {
			return fail("WORLD_LOAD_FAILED", err.Error(), 2)
		}
		world = loaded
	}

	result, err := operations.ModelDDL(world, options)
	if err != nil {
		// A named subject that no table backs is reported with its own code. It
		// is an argument the caller got wrong (a subject that is not a persisted
		// Data element), so it keeps exit 2 with a machine-readable code rather
		// than a generic failure.
		var notFound *operations.ModelDDLSubjectNotFound
		if errors.As(err, &notFound) {
			return fail("MODEL_DDL_SUBJECT_NOT_FOUND", notFound.Error(), 2)
		}
		return fail("MODEL_DDL_FAILED", err.Error(), 2)
	}
	response.OK = true
	response.Result = &result
	return writeModelDDLResponse(out, response, 0)
}

func writeModelDDLResponse(out io.Writer, response modelDDLCLIResponse, exit int) (int, error) {
	encoder := json.NewEncoder(out)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(response); err != nil {
		return 2, err
	}
	return exit, nil
}
