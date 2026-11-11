package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"os"
	"strings"

	"tei/internal/documentdraft"
	"tei/internal/operations"
	"tei/internal/sourceindex"
)

// documentDraftCLIResponse is the envelope for `model document-draft`. The
// result is a derived read-only projection: it carries no acceptance, no write
// authority and no model ingestion claim.
type documentDraftCLIResponse struct {
	SchemaVersion string                `json:"schemaVersion"`
	OK            bool                  `json:"ok"`
	Result        *documentdraft.Draft  `json:"result,omitempty"`
	Error         *cliInspectionFailure `json:"error,omitempty"`
}

const (
	documentDraftSchemaVersion     = "tei.cli.model-document-draft/v0"
	documentReconcileSchemaVersion = "tei.cli.model-document-reconcile/v0"
)

func writeDocumentDraftResponse(out io.Writer, response documentDraftCLIResponse, exit int) (int, error) {
	encoder := json.NewEncoder(out)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(response); err != nil {
		return 2, err
	}
	return exit, nil
}

func failDocumentDraft(out io.Writer, code, message string, exit int) (int, error) {
	return writeDocumentDraftResponse(out, documentDraftCLIResponse{
		SchemaVersion: documentDraftSchemaVersion,
		Error:         &cliInspectionFailure{Code: code, Message: message},
	}, exit)
}

// documentDraftFailureCode maps a projection failure onto the stable code the
// envelope reports. An argument problem stays INVALID_ARGUMENT with exit 2; a
// subject that is not there is a domain answer with exit 1.
func documentDraftFailureCode(err error) (string, int) {
	var draftErr *documentdraft.DraftError
	if errors.As(err, &draftErr) {
		if draftErr.Code == documentdraft.CodeInvalidArgument {
			return draftErr.Code, 2
		}
		return draftErr.Code, 1
	}
	var failure *operations.ValidationFailure
	if errors.As(err, &failure) {
		switch failure.Stage {
		case operations.StageLoad:
			return "WORKSPACE_LOAD_FAILED", 2
		case operations.StageSourceIndex:
			return "SOURCE_INDEX_FAILED", 2
		}
	}
	return "DOCUMENT_DRAFT_FAILED", 1
}

func runModelDocumentDraftCommand(args []string, out io.Writer) (int, error) {
	fs := flag.NewFlagSet("model document-draft", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	root := fs.String("workspace", "", "TEI workspace directory")
	kind := fs.String("subject-kind", "", "exact subject kind: "+strings.Join(documentdraft.SupportedElementKinds(), "|"))
	id := fs.String("subject-id", "", "exact subject ID")
	relationID := fs.String("relation-id", "", "optional exact semantic-relation ID declared by the subject")
	title := fs.String("title", "", "optional document title")
	documentKind := fs.String("document-kind", documentdraft.DocumentKindElementExplanation,
		"document kind: "+strings.Join(documentdraft.SupportedDocumentKinds(), "|"))
	application := fs.String("application", "", "exact application ID; required by "+documentdraft.DocumentKindCRUDTable)
	frozen := fs.Bool("frozen", false,
		"pin the draft to the model revision it is read at (a fixed delivery)")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printHelp(out, fs, "model document-draft")
			return 0, nil
		}
		return failDocumentDraft(out, "INVALID_ARGUMENT", err.Error(), 2)
	}
	if err := validateRepeatableArguments("model document-draft", args); err != nil {
		return failDocumentDraft(out, "INVALID_ARGUMENT", err.Error(), 2)
	}
	if err := invalidEmptyStringArgument(fs, "model document-draft"); err != nil {
		return failDocumentDraft(out, "INVALID_ARGUMENT", err.Error(), 2)
	}
	if fs.NArg() != 0 || strings.TrimSpace(*root) == "" || *root != strings.TrimSpace(*root) ||
		strings.TrimSpace(*documentKind) == "" || *documentKind != strings.TrimSpace(*documentKind) {
		return failDocumentDraft(out, "INVALID_ARGUMENT",
			"model document-draft requires an exact -workspace and -document-kind, without positional arguments", 2)
	}
	draft, err := documentdraft.Build(documentdraft.Request{
		Root:         *root,
		Subject:      sourceindex.Subject{Kind: *kind, ID: *id},
		RelationID:   *relationID,
		DocumentKind: *documentKind,
		Title:        *title,
		Application:  *application,
		Frozen:       *frozen,
	})
	if err != nil {
		code, exit := documentDraftFailureCode(err)
		return failDocumentDraft(out, code, err.Error(), exit)
	}
	return writeDocumentDraftResponse(out, documentDraftCLIResponse{
		SchemaVersion: documentDraftSchemaVersion,
		OK:            true,
		Result:        &draft,
	}, 0)
}

// documentReconcileCLIResponse is the envelope for `model document-reconcile`.
type documentReconcileCLIResponse struct {
	SchemaVersion string                         `json:"schemaVersion"`
	OK            bool                           `json:"ok"`
	Result        *documentdraft.ReconcileReport `json:"result,omitempty"`
	Error         *cliInspectionFailure          `json:"error,omitempty"`
}

func writeDocumentReconcileResponse(out io.Writer, response documentReconcileCLIResponse, exit int) (int, error) {
	encoder := json.NewEncoder(out)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(response); err != nil {
		return 2, err
	}
	return exit, nil
}

// draftDocumentEnvelope is the shape `model document-draft` writes: an envelope
// whose `result` carries the draft. A stored file is usually exactly this
// output, so the reader accepts both the envelope and the bare draft.
type draftDocumentEnvelope struct {
	SchemaVersion string               `json:"schemaVersion"`
	Result        *documentdraft.Draft `json:"result"`
}

// decodeDraftDocument reads one stored draft. When the file is the CLI envelope,
// the inner `result` is used; when it is the bare draft, it is used as it is.
// Anything else is a decode failure, not a partially completed draft.
func decodeDraftDocument(document []byte) (documentdraft.Draft, error) {
	var envelope draftDocumentEnvelope
	if err := json.Unmarshal(document, &envelope); err != nil {
		return documentdraft.Draft{}, err
	}
	if envelope.Result != nil &&
		envelope.Result.SchemaVersion == documentdraft.SchemaVersion {
		return *envelope.Result, nil
	}
	var draft documentdraft.Draft
	if err := json.Unmarshal(document, &draft); err != nil {
		return documentdraft.Draft{}, err
	}
	return draft, nil
}

func runModelDocumentReconcileCommand(args []string, out io.Writer) (int, error) {
	fs := flag.NewFlagSet("model document-reconcile", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	root := fs.String("workspace", "", "TEI workspace directory")
	draftPath := fs.String("draft", "", "path to the stored draft JSON")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printHelp(out, fs, "model document-reconcile")
			return 0, nil
		}
		return writeDocumentReconcileResponse(out, documentReconcileCLIResponse{
			SchemaVersion: documentReconcileSchemaVersion,
			Error:         &cliInspectionFailure{Code: "INVALID_ARGUMENT", Message: err.Error()},
		}, 2)
	}
	if err := validateRepeatableArguments("model document-reconcile", args); err != nil {
		return writeDocumentReconcileResponse(out, documentReconcileCLIResponse{
			SchemaVersion: documentReconcileSchemaVersion,
			Error:         &cliInspectionFailure{Code: "INVALID_ARGUMENT", Message: err.Error()},
		}, 2)
	}
	if err := invalidEmptyStringArgument(fs, "model document-reconcile"); err != nil {
		return writeDocumentReconcileResponse(out, documentReconcileCLIResponse{
			SchemaVersion: documentReconcileSchemaVersion,
			Error:         &cliInspectionFailure{Code: "INVALID_ARGUMENT", Message: err.Error()},
		}, 2)
	}
	if fs.NArg() != 0 || strings.TrimSpace(*root) == "" || *root != strings.TrimSpace(*root) ||
		strings.TrimSpace(*draftPath) == "" {
		return writeDocumentReconcileResponse(out, documentReconcileCLIResponse{
			SchemaVersion: documentReconcileSchemaVersion,
			Error: &cliInspectionFailure{Code: "INVALID_ARGUMENT",
				Message: "model document-reconcile requires an exact -workspace and a -draft path, without positional arguments"},
		}, 2)
	}
	raw, err := os.ReadFile(*draftPath)
	if err != nil {
		return writeDocumentReconcileResponse(out, documentReconcileCLIResponse{
			SchemaVersion: documentReconcileSchemaVersion,
			Error:         &cliInspectionFailure{Code: "DRAFT_READ_FAILED", Message: err.Error()},
		}, 2)
	}
	// A leading UTF-8 BOM is a byte-order mark, not JSON text. Several Windows
	// editors and shells add one, and a decoder that refuses it would report a
	// broken draft for an encoding detail. The BOM is dropped here and recorded
	// nowhere else: it carries no meaning.
	document := bytes.TrimPrefix(raw, []byte{0xEF, 0xBB, 0xBF})
	draft, err := decodeDraftDocument(document)
	if err != nil {
		return writeDocumentReconcileResponse(out, documentReconcileCLIResponse{
			SchemaVersion: documentReconcileSchemaVersion,
			Error:         &cliInspectionFailure{Code: "DRAFT_DECODE_FAILED", Message: err.Error()},
		}, 2)
	}
	report, err := documentdraft.Reconcile(draft, *root)
	if err != nil {
		code, exit := documentDraftFailureCode(err)
		return writeDocumentReconcileResponse(out, documentReconcileCLIResponse{
			SchemaVersion: documentReconcileSchemaVersion,
			Error:         &cliInspectionFailure{Code: code, Message: err.Error()},
		}, exit)
	}
	return writeDocumentReconcileResponse(out, documentReconcileCLIResponse{
		SchemaVersion: documentReconcileSchemaVersion,
		OK:            true,
		Result:        &report,
	}, 0)
}
