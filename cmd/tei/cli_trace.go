package main

import (
	"encoding/json"
	"errors"
	"io"
	"strings"

	"tei/internal/modelref"
	"tei/internal/operations"
	"tei/internal/subject"
)

type cliTraceReport struct {
	SchemaVersion string                     `json:"schemaVersion"`
	OK            bool                       `json:"ok"`
	Result        *operations.WorkspaceTrace `json:"result,omitempty"`
	Error         *cliInspectionFailure      `json:"error,omitempty"`
}

// traceWorkspaceJSON translates CLI-specific flags and exit codes. The typed
// dependency traversal and Source navigation are owned by operations/modelref.
func traceWorkspaceJSON(
	out io.Writer,
	root, world, scope, kind, id, direction, mode, expectedFingerprint string,
	offset, limit, positional int,
) (int, error) {
	report := cliTraceReport{SchemaVersion: "tei.cli.trace/v0"}
	if strings.TrimSpace(root) == "" || world != "" ||
		strings.TrimSpace(scope) == "" || strings.TrimSpace(kind) == "" ||
		strings.TrimSpace(id) == "" || scope != strings.TrimSpace(scope) ||
		kind != strings.TrimSpace(kind) || id != strings.TrimSpace(id) ||
		positional != 0 {
		report.Error = &cliInspectionFailure{
			Code: "INVALID_ARGUMENT",
			Message: "trace requires -workspace, -scope, -subject-kind, -subject-id and no -world/positional arguments",
		}
		return writeTraceReport(out, report, 2)
	}
	result, err := operations.TraceWorkspace(root,
		modelref.Key{Scope: scope, Subject: subject.Subject{Kind: kind, ID: id}},
		modelref.TraceOptions{
			Direction: direction, Mode: mode, Offset: offset, Limit: limit,
			ExpectedFingerprint: expectedFingerprint,
		},
	)
	if err != nil {
		code := 2
		failure := &cliInspectionFailure{Code: "TRACE_FAILED", Message: err.Error()}
		var typed *modelref.TraceError
		var stage *operations.ValidationFailure
		switch {
		case errors.As(err, &typed):
			failure.Code = typed.Code
			switch typed.Code {
			case "SUBJECT_NOT_FOUND", "SUBJECT_AMBIGUOUS", "STALE_SNAPSHOT":
				code = 1
			}
		case errors.As(err, &stage):
			switch stage.Stage {
			case operations.StageLoad:
				failure.Code = "WORKSPACE_LOAD_FAILED"
			case operations.StageSourceIndex:
				failure.Code = "SOURCE_INDEX_FAILED"
			}
		}
		report.Error = failure
		return writeTraceReport(out, report, code)
	}
	report.OK = true
	report.Result = &result
	return writeTraceReport(out, report, 0)
}

func writeTraceReport(out io.Writer, report cliTraceReport, code int) (int, error) {
	enc := json.NewEncoder(out)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(report); err != nil {
		return 2, err
	}
	return code, nil
}
