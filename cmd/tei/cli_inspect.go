package main

import (
	"encoding/json"
	"errors"
	"io"

	"tei/internal/operations"
	"tei/internal/sourceindex"
)

type cliInspectionFailure struct {
	Code          string `json:"code"`
	Message       string `json:"message"`
	Kind          string `json:"kind,omitempty"`
	Retryable     string `json:"retryable,omitempty"`
	EffectUnknown bool   `json:"effectUnknown,omitempty"`
}
type cliInspectionReport struct {
	SchemaVersion string                        `json:"schemaVersion"`
	OK            bool                          `json:"ok"`
	Result        *operations.SubjectInspection `json:"result,omitempty"`
	Error         *cliInspectionFailure         `json:"error,omitempty"`
}

// cliInspectionService is a typed Host-injected call site, not a new Plugin
// Resolver or an authorization grant. The selected implementation must be
// supplied by a trusted CLI composition host; it receives no filesystem path
// from command-line arguments.
type cliInspectionService interface {
	Inspect(sourceindex.Subject) (operations.SubjectInspection, error)
}

type builtinCLIInspectionService struct{ root string }

func (s builtinCLIInspectionService) Inspect(subject sourceindex.Subject) (operations.SubjectInspection, error) {
	return operations.InspectSubject(s.root, subject)
}

// inspectWorkspaceJSON preserves the current built-in CLI reference path.
// It is not evidence of an actual selected Plugin Provider invocation.
func inspectWorkspaceJSON(out io.Writer, root, world, kind, id string, positional int) (int, error) {
	return inspectWorkspaceJSONWithService(out, root, world, kind, id, positional, builtinCLIInspectionService{root: root})
}

// canonicalInspectionMessages reports whether this service's failures are the
// canonical Operation's own errors, unchanged. The in-process builtin always is;
// a Host-selected implementation says so through its registration.
func canonicalInspectionMessages(service cliInspectionService) bool {
	if _, builtin := service.(builtinCLIInspectionService); builtin {
		return true
	}
	if declared, ok := service.(interface{ CanonicalInspectionMessages() bool }); ok {
		return declared.CanonicalInspectionMessages()
	}
	return false
}

// inspectWorkspaceJSONWithService keeps CLI transport separate from the
// implementation selected by a trusted host. A missing selected service must
// fail closed rather than fall back to the built-in Inspector.
// A nonzero exit still emits exactly one machine-readable JSON document.
func inspectWorkspaceJSONWithService(out io.Writer, root, world, kind, id string, positional int, service cliInspectionService) (int, error) {
	report := cliInspectionReport{SchemaVersion: "tei.cli.inspect/v0"}
	if root == "" || world != "" || kind == "" || id == "" || positional != 0 {
		// A missing flag is an argument problem, not an unusable subject
		// value (#3733). INVALID_SUBJECT stays for a supplied-but-invalid
		// subject, which the operation decides.
		report.Error = &cliInspectionFailure{
			Code:    resultCodeInvalidArgument,
			Message: "inspect requires -workspace, -subject-kind and -subject-id; -world and positional arguments are not supported",
		}
		return writeInspectionReport(out, report, 2)
	}
	if service == nil {
		report.Error = &cliInspectionFailure{
			Code:    "INSPECTION_PROVIDER_MISSING",
			Message: "the selected inspection service is unavailable",
		}
		return writeInspectionReport(out, report, 2)
	}
	subject := sourceindex.Subject{Kind: kind, ID: id}
	result, err := service.Inspect(subject)
	if err != nil {
		var inspection *operations.InspectionError
		var stage *operations.ValidationFailure
		code := 2
		// Unknown Provider errors can contain host-private paths or credentials, so
		// the default message stays generic. The Load and Source Index stages below
		// replace it with this process's own error text, which keeps the CLI answer
		// identical to the MCP adapter for the same input (#3420).
		failure := &cliInspectionFailure{Code: "INSPECTION_FAILED", Message: "workspace inspection failed"}
		switch {
		case errors.As(err, &inspection):
			// A third-party implementation can construct arbitrary errors,
			// including a forged code or host-private text. Preserve the
			// legacy builtin's diagnostic, but accept only classified
			// Inspector codes from an injected implementation.
			switch inspection.Code {
			case operations.InspectionInvalidSubject,
				operations.InspectionUnsupportedKind,
				operations.InspectionNotFound,
				operations.InspectionAmbiguous:
				failure.Code = inspection.Code
				// The message is shown verbatim only when the service states that it
				// forwards the canonical Operation's own errors. An unverified
				// implementation could put host-private paths or credentials into its
				// text, so its message stays masked. When the text is canonical it must
				// match the MCP adapter for the same input (#3420).
				if canonicalInspectionMessages(service) {
					failure.Message = inspection.Message
				} else {
					failure.Message = "inspection service reported " + inspection.Code
				}
				if inspection.Code == operations.InspectionNotFound || inspection.Code == operations.InspectionAmbiguous {
					code = 1
				}
			}
		case errors.As(err, &stage):
			// These two stages fail while reading this process's own Workspace. The
			// text describes the read, not a Provider, so it is safe to report and
			// must match the MCP adapter's answer for the same input (#3420).
			failure.Message = err.Error()
			switch stage.Stage {
			case operations.StageLoad:
				failure.Code = "WORKSPACE_LOAD_FAILED"
			case operations.StageSourceIndex:
				failure.Code = "SOURCE_INDEX_FAILED"
			}
		}
		report.Error = failure
		return writeInspectionReport(out, report, code)
	}
	if err := operations.ValidateInspectionProjection(subject, result); err != nil {
		report.Error = &cliInspectionFailure{
			Code:    "INSPECTION_PROVIDER_INVALID",
			Message: "inspection service returned an inconsistent projection",
		}
		return writeInspectionReport(out, report, 2)
	}
	report.OK = true
	report.Result = &result
	return writeInspectionReport(out, report, 0)
}

func writeInspectionReport(out io.Writer, report cliInspectionReport, exitCode int) (int, error) {
	enc := json.NewEncoder(out)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(report); err != nil {
		return 2, err
	}
	return exitCode, nil
}
