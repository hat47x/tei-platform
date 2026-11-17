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
	if len(args) > 0 && args[0] == "search" {
		return runModelSearchCommand(args[1:], out)
	}
	if len(args) > 0 && args[0] == "document-draft" {
		return runModelDocumentDraftCommand(args[1:], out)
	}
	if len(args) > 0 && args[0] == "document-reconcile" {
		return runModelDocumentReconcileCommand(args[1:], out)
	}
	if len(args) > 0 && args[0] == "ddl" {
		return runModelDDLCommand(args[1:], out)
	}
	if len(args) == 0 || args[0] != "catalog" {
		return fail("INVALID_ARGUMENT", "model requires the subcommand catalog, locate, overview, search, document-draft, document-reconcile or ddl", 2)
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
	expectedFingerprint := fs.String("expected-fingerprint", "", "expected snapshot fingerprint; fails when the World or Source Index changed")
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
	result, err := operations.LocateWorkspaceWithOptions(*root,
		sourceindex.Subject{Kind: *kind, ID: *id}, *limit,
		operations.LocateOptions{ExpectedFingerprint: *expectedFingerprint})
	if err != nil {
		var located *operations.CatalogError
		if errors.As(err, &located) {
			return fail(located.Code, located.Message, 1)
		}
		// A stale snapshot fingerprint is this caller's precondition not being
		// met, which is the same class of answer as a missing or conflicting
		// flag, so it keeps exit 2 and its own code instead of the generic
		// failure above.
		var stale *operations.InspectionError
		if errors.As(err, &stale) && stale.Code == operations.InspectionStaleSnapshot {
			return fail(stale.Code, stale.Message, 2)
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
	expectedFingerprint := fs.String("expected-fingerprint", "", "expected snapshot fingerprint; fails when the World or Source Index changed")
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
	result, err := operations.OverviewWorkspaceWithOptions(*root, *limit,
		operations.OverviewOptions{ExpectedFingerprint: *expectedFingerprint})
	if err != nil {
		// A stale snapshot fingerprint is this caller's precondition not being
		// met, which is the same class of answer as a missing or conflicting
		// flag, so it keeps exit 2 and its own code instead of the generic
		// failure below.
		var stale *operations.InspectionError
		if errors.As(err, &stale) && stale.Code == operations.InspectionStaleSnapshot {
			return fail(stale.Code, stale.Message, 2)
		}
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

// modelSearchCLIResponse is the envelope for `model search`. It keeps its own
// schema name because its result is a set of matches, not a catalog, a location
// or a table of contents.
type modelSearchCLIResponse struct {
	SchemaVersion string                  `json:"schemaVersion"`
	OK            bool                    `json:"ok"`
	Result        *operations.ModelSearch `json:"result,omitempty"`
	Error         *cliInspectionFailure   `json:"error,omitempty"`
}

const modelSearchSchemaVersion = "tei.cli.model-search/v0"

// modelSearchFlags is the whole grammar `model search` accepts. It is built by
// one function so the parser, the `-h` listing and the advertisement test all
// read the same FlagSet: a flag added to one of them cannot be missing from the
// others.
func modelSearchFlags(fs *flag.FlagSet) (root, term, kind *string, limit, offset *int) {
	fs.SetOutput(io.Discard)
	root = fs.String("workspace", "", "TEI workspace directory")
	term = fs.String("term", "", "literal text to find; matched case-insensitively inside a Source line; required unless -kind is given")
	kind = fs.String("kind", "", "restrict the search to one subject kind: "+strings.Join(operations.InspectionSubjectKinds, "|"))
	limit = fs.Int("limit", operations.SearchDefaultLimit, "maximum hits listed; total and kindCounts still cover every hit, subjectCount every distinct subject")
	offset = fs.Int("offset", 0, "hits to skip; a value past the last hit is clamped and reported as requestedOffset")
	return root, term, kind, limit, offset
}

// runModelSearchCommand answers where a text appears in the Workspace's
// Sources and which element declares it. The catalog lists every element but
// searches no text, so a caller who knew a name fragment had to page through
// all entries, guess the subject kind and call `inspect` per candidate -- and
// still learn nothing about a mention inside a declaration body. Every hit
// carries the subject, the Source, the line and the line as written, so the
// match can be opened rather than trusted (RQ-036, design §19.8). Read-only; it
// shares the catalog's load, so a hit and `model catalog` describe one model
// version.
//
// The answer keeps the two units of counting apart, because one subject can
// account for several lines: `result.total` and `result.kindCounts` count HITS
// (kindCounts is total broken down by kind), while `result.subjectCount` counts
// the DISTINCT SUBJECTS among all the matches and `result.subjects` names them
// once each with how many of their lines matched. `result.distinctSubjects-
// Omitted` counts the distinct subjects among the matches the page does not
// return. None of the counts is cut by -limit except the hit list itself.
func runModelSearchCommand(args []string, out io.Writer) (int, error) {
	response := modelSearchCLIResponse{SchemaVersion: modelSearchSchemaVersion}
	fail := func(code string, message string, exit int) (int, error) {
		response.Error = &cliInspectionFailure{Code: code, Message: message}
		return writeModelSearchResponse(out, response, exit)
	}
	fs := flag.NewFlagSet("model search", flag.ContinueOnError)
	root, term, kind, limit, offset := modelSearchFlags(fs)
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printHelp(out, fs, "model search")
			return 0, nil
		}
		return fail("INVALID_ARGUMENT", err.Error(), 2)
	}
	if err := validateRepeatableArguments("model search", args); err != nil {
		return fail("INVALID_ARGUMENT", err.Error(), 2)
	}
	if name := emptyStringFlag(fs, emptyStringFlagsNotAllowed["model search"]); name != "" {
		return fail("INVALID_ARGUMENT", fmt.Sprintf("model search: -%s must not be an empty string", name), 2)
	}
	if fs.NArg() != 0 || strings.TrimSpace(*root) == "" || *root != strings.TrimSpace(*root) {
		return fail("INVALID_ARGUMENT", "model search requires an exact -workspace, without positional arguments", 2)
	}
	if strings.TrimSpace(*term) == "" && strings.TrimSpace(*kind) == "" {
		return fail("INVALID_ARGUMENT", "model search requires -term, or -kind to list one kind's sources", 2)
	}
	if *term != "" && *term != strings.TrimSpace(*term) {
		return fail("INVALID_ARGUMENT", "model search: -term must not have surrounding whitespace", 2)
	}
	if *kind != "" && !operations.SupportedInspectionKind(*kind) {
		return fail("INVALID_ARGUMENT", fmt.Sprintf("model search: -kind must be one of %s", strings.Join(operations.InspectionSubjectKinds, ", ")), 2)
	}
	if *limit < 1 || *limit > operations.SearchMaxLimit {
		return fail("INVALID_ARGUMENT", fmt.Sprintf("model search: -limit must be between 1 and %d", operations.SearchMaxLimit), 2)
	}
	if *offset < 0 {
		return fail("INVALID_ARGUMENT", "model search: -offset must not be negative", 2)
	}
	result, err := operations.SearchWorkspace(*root,
		operations.SearchQuery{Term: *term, Kind: *kind}, *limit, *offset)
	if err != nil {
		// A Source that cannot be read, or whose bytes no longer match the
		// fingerprint this read indexed, is an execution failure: the hit's
		// location is evidence about those exact bytes, so answering from
		// different ones is refused.
		var catalogErr *operations.CatalogError
		if errors.As(err, &catalogErr) {
			return fail(catalogErr.Code, catalogErr.Message, 2)
		}
		code := "MODEL_SEARCH_FAILED"
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
	return writeModelSearchResponse(out, response, 0)
}
func writeModelSearchResponse(out io.Writer, response modelSearchCLIResponse, exit int) (int, error) {
	encoder := json.NewEncoder(out)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(response); err != nil {
		return 2, err
	}
	return exit, nil
}
