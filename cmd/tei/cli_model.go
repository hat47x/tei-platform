package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"

	"tei/internal/operations"
	"tei/internal/sourceindex"
)

// Modeler commands share the existing operations layer with future web and
// desktop targets. No model edit or XEA[D] acceptance occurs in this transport.
type modelCLIResponse struct {
	SchemaVersion string                   `json:"schemaVersion"`
	OK            bool                     `json:"ok"`
	Result        *operations.ModelCatalog `json:"result,omitempty"`
	Error         *cliInspectionFailure    `json:"error,omitempty"`
}

func runModelCommand(args []string, out io.Writer) (int, error) {
	response := modelCLIResponse{SchemaVersion: "tei.cli.model/v0"}
	fail := func(code string, message string, exit int) (int, error) {
		response.Error = &cliInspectionFailure{Code: code, Message: message}
		return writeModelResponse(out, response, exit)
	}
	if len(args) > 0 && args[0] == "locate" {
		return runModelLocateCommand(args[1:], out)
	}
	if len(args) > 0 && args[0] == "overview" {
		return runModelOverviewCommand(args[1:], out)
	}
	if len(args) == 0 || args[0] != "catalog" {
		return fail("INVALID_ARGUMENT", "model requires the subcommand catalog, locate or overview", 2)
	}
	fs := flag.NewFlagSet("model catalog", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	root := fs.String("workspace", "", "TEI workspace directory")
	namespace := fs.String("namespace", "", "optional exact semantic namespace ID")
	if err := fs.Parse(args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printHelp(out, fs, "model catalog")
			return 0, nil
		}
		return fail("INVALID_ARGUMENT", err.Error(), 2)
	}
	if fs.NArg() != 0 || strings.TrimSpace(*root) == "" || *root != strings.TrimSpace(*root) ||
		*namespace != strings.TrimSpace(*namespace) {
		return fail("INVALID_ARGUMENT", "model catalog requires an exact -workspace and optional exact -namespace, without positional arguments", 2)
	}
	// This command parses its own FlagSet, so the shared repeat check does not
	// see it. Both flags hold one value; a second -namespace silently replaced
	// the first (#3675).
	if err := validateRepeatableArguments("model catalog", args[1:]); err != nil {
		return fail("INVALID_ARGUMENT", err.Error(), 2)
	}
	if name := emptyStringFlag(fs, emptyStringFlagsNotAllowed["model catalog"]); name != "" {
		// An explicit -namespace "" is not "no namespace": it used to reach
		// the catalog as an empty filter and return every entry, which is a
		// third outcome beside "omitted" and "a namespace ID" (#3605).
		return fail("INVALID_ARGUMENT", fmt.Sprintf("model catalog: -%s must not be an empty string; omit the flag to list every namespace", name), 2)
	}
	result, err := operations.CatalogWorkspace(*root, *namespace)
	if err != nil {
		code := "MODEL_CATALOG_FAILED"
		var failure *operations.ValidationFailure
		if errors.As(err, &failure) {
			switch failure.Stage {
			case operations.StageLoad:
				code = "WORKSPACE_LOAD_FAILED"
			case operations.StageSourceIndex:
				code = "SOURCE_INDEX_FAILED"
			}
		}
		return fail(code, err.Error(), 2)
	}
	// A namespace that does not exist is not "a namespace with no entries".
	// The projection owns that decision so the MCP adapter answers the same
	// code and message instead of inventing its own (#3420).
	if failure := result.NamespaceFilterError(); failure != nil {
		return fail(failure.Code, failure.Message, 1)
	}
	response.OK = true
	response.Result = &result
	return writeModelResponse(out, response, 0)
}

func writeModelResponse(out io.Writer, response modelCLIResponse, exit int) (int, error) {
	encoder := json.NewEncoder(out)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(response); err != nil {
		return 2, err
	}
	return exit, nil
}

// modelLocateCLIResponse is the envelope for `model locate`. It keeps its own
// schema name because its result is a location, not a catalog.
type modelLocateCLIResponse struct {
	SchemaVersion string                    `json:"schemaVersion"`
	OK            bool                      `json:"ok"`
	Result        *operations.ModelLocation `json:"result,omitempty"`
	Error         *cliInspectionFailure     `json:"error,omitempty"`
}

const modelLocateSchemaVersion = "tei.cli.model-locate/v0"

// runModelLocateCommand answers where one exact subject sits in the whole
// model: its path from the top, children, siblings and declared references
// (RQ-036, design §19.8). It is read-only and shares the catalog's load, so a
// diagram and this text can be compared by fingerprint.
func runModelLocateCommand(args []string, out io.Writer) (int, error) {
	response := modelLocateCLIResponse{SchemaVersion: modelLocateSchemaVersion}
	fail := func(code string, message string, exit int) (int, error) {
		response.Error = &cliInspectionFailure{Code: code, Message: message}
		return writeModelLocateResponse(out, response, exit)
	}
	fs := flag.NewFlagSet("model locate", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	root := fs.String("workspace", "", "TEI workspace directory")
	kind := fs.String("subject-kind", "", "exact subject kind from model catalog entries[].subject.kind")
	id := fs.String("subject-id", "", "exact subject ID")
	limit := fs.Int("limit", operations.LocationDefaultLimit, "maximum entries per list (children, siblings)")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printHelp(out, fs, "model locate")
			return 0, nil
		}
		return fail("INVALID_ARGUMENT", err.Error(), 2)
	}
	if err := validateRepeatableArguments("model locate", args); err != nil {
		return fail("INVALID_ARGUMENT", err.Error(), 2)
	}
	if name := emptyStringFlag(fs, emptyStringFlagsNotAllowed["model locate"]); name != "" {
		return fail("INVALID_ARGUMENT", fmt.Sprintf("model locate: -%s must not be an empty string", name), 2)
	}
	if fs.NArg() != 0 || strings.TrimSpace(*root) == "" || *root != strings.TrimSpace(*root) ||
		strings.TrimSpace(*kind) == "" || *kind != strings.TrimSpace(*kind) ||
		strings.TrimSpace(*id) == "" || *id != strings.TrimSpace(*id) {
		return fail("INVALID_ARGUMENT", "model locate requires exact -workspace, -subject-kind and -subject-id, without positional arguments", 2)
	}
	if *limit < 1 || *limit > operations.LocationMaxLimit {
		return fail("INVALID_ARGUMENT", fmt.Sprintf("model locate: -limit must be between 1 and %d", operations.LocationMaxLimit), 2)
	}
	result, err := operations.LocateWorkspace(*root, sourceindex.Subject{Kind: *kind, ID: *id}, *limit)
	if err != nil {
		var located *operations.CatalogError
		if errors.As(err, &located) {
			return fail(located.Code, located.Message, 1)
		}
		code := "MODEL_LOCATE_FAILED"
		var failure *operations.ValidationFailure
		if errors.As(err, &failure) {
			switch failure.Stage {
			case operations.StageLoad:
				code = "WORKSPACE_LOAD_FAILED"
			case operations.StageSourceIndex:
				code = "SOURCE_INDEX_FAILED"
			}
		}
		return fail(code, err.Error(), 2)
	}
	response.OK = true
	response.Result = &result
	return writeModelLocateResponse(out, response, 0)
}

func writeModelLocateResponse(out io.Writer, response modelLocateCLIResponse, exit int) (int, error) {
	encoder := json.NewEncoder(out)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(response); err != nil {
		return 2, err
	}
	return exit, nil
}

// modelOverviewCLIResponse is the envelope for `model overview`.
type modelOverviewCLIResponse struct {
	SchemaVersion string                    `json:"schemaVersion"`
	OK            bool                      `json:"ok"`
	Result        *operations.ModelOverview `json:"result,omitempty"`
	Error         *cliInspectionFailure     `json:"error,omitempty"`
}

const modelOverviewSchemaVersion = "tei.cli.model-overview/v0"

// runModelOverviewCommand returns the model's table of contents: one row per
// top-level section with the shape of its subtree and the problems found
// anywhere under it (RQ-036, design §19.8 G5). Read-only; it shares the
// catalog's load, so its fingerprints match model catalog and model locate.
func runModelOverviewCommand(args []string, out io.Writer) (int, error) {
	response := modelOverviewCLIResponse{SchemaVersion: modelOverviewSchemaVersion}
	fail := func(code string, message string, exit int) (int, error) {
		response.Error = &cliInspectionFailure{Code: code, Message: message}
		return writeModelOverviewResponse(out, response, exit)
	}
	fs := flag.NewFlagSet("model overview", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	root := fs.String("workspace", "", "TEI workspace directory")
	limit := fs.Int("limit", operations.OverviewDefaultLimit, "maximum sections listed")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printHelp(out, fs, "model overview")
			return 0, nil
		}
		return fail("INVALID_ARGUMENT", err.Error(), 2)
	}
	if err := validateRepeatableArguments("model overview", args); err != nil {
		return fail("INVALID_ARGUMENT", err.Error(), 2)
	}
	if name := emptyStringFlag(fs, emptyStringFlagsNotAllowed["model overview"]); name != "" {
		return fail("INVALID_ARGUMENT", fmt.Sprintf("model overview: -%s must not be an empty string", name), 2)
	}
	if fs.NArg() != 0 || strings.TrimSpace(*root) == "" || *root != strings.TrimSpace(*root) {
		return fail("INVALID_ARGUMENT", "model overview requires an exact -workspace, without positional arguments", 2)
	}
	if *limit < 1 || *limit > operations.OverviewMaxLimit {
		return fail("INVALID_ARGUMENT", fmt.Sprintf("model overview: -limit must be between 1 and %d", operations.OverviewMaxLimit), 2)
	}
	result, err := operations.OverviewWorkspace(*root, *limit)
	if err != nil {
		code := "MODEL_OVERVIEW_FAILED"
		var failure *operations.ValidationFailure
		if errors.As(err, &failure) {
			switch failure.Stage {
			case operations.StageLoad:
				code = "WORKSPACE_LOAD_FAILED"
			case operations.StageSourceIndex:
				code = "SOURCE_INDEX_FAILED"
			}
		}
		return fail(code, err.Error(), 2)
	}
	response.OK = true
	response.Result = &result
	return writeModelOverviewResponse(out, response, 0)
}

func writeModelOverviewResponse(out io.Writer, response modelOverviewCLIResponse, exit int) (int, error) {
	encoder := json.NewEncoder(out)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(response); err != nil {
		return 2, err
	}
	return exit, nil
}
