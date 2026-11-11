package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"tei/internal/externalanalysis"
)

// reaEvidenceOuterArgs holds the flags of `rea-evidence`. They are read in one
// place so the flag set and the command body cannot drift.
type reaEvidenceOuterArgs struct {
	evidencePath        string
	viewPath            string
	requestPath         string
	providerVersion     string
	expectedEvidenceID  string
	expectedFingerprint string
	expectedSourceRev   string
	requestOut          string
	toolPath            string
}

// reaEvidenceResult is the command's result body: the receipt, plus the path the
// derived request was written to when the caller asked for one.
type reaEvidenceResult struct {
	Receipt            externalanalysis.Receipt `json:"receipt"`
	DerivedRequestPath string                   `json:"derivedRequestPath,omitempty"`
}

// reaEvidenceExit maps a receipt state to a process exit code.
//
// The mapping follows the convention this repository already uses: an input the
// caller can fix is exit 2, a receipt that says something the caller needs to
// act on is exit 1, and a receipt the command was able to produce is exit 0. A
// receipt from an environment without the external tool exits 0: the command did
// its job on the saved bytes, and the receipt says those bytes could not have
// been produced here.
func reaEvidenceExit(state externalanalysis.State) int {
	switch state {
	case externalanalysis.StateComplete, externalanalysis.StateToolAbsent:
		return 0
	case externalanalysis.StatePartial, externalanalysis.StateFailed:
		return 1
	default:
		return 2
	}
}

// reaEvidenceFailureCode names the envelope error for a state, so a caller can
// tell "the bytes are not what they claim" from "the delivery is not whole"
// without reading prose.
func reaEvidenceFailureCode(state externalanalysis.State) string {
	switch state {
	case externalanalysis.StatePartial:
		return "EVIDENCE_PARTIAL"
	case externalanalysis.StateFailed:
		return "EVIDENCE_FAILED"
	case externalanalysis.StateMismatch:
		return "EVIDENCE_MISMATCH"
	case externalanalysis.StateMissing:
		return "EVIDENCE_MISSING"
	}
	return "EVIDENCE_UNUSABLE"
}

// reaEvidenceJSON receives one saved external analysis output, inspects the
// optional saved view delivery and derived request that belong to it, and
// reports the receipt.
//
// The command never invokes the external tool. It reads bytes that were already
// produced somewhere else and measures them, which is why a machine without the
// tool installed can still run it — and why the receipt records whether the tool
// was here at all.
func reaEvidenceJSON(out io.Writer, args reaEvidenceOuterArgs, extraArgs int) (int, error) {
	const command = "rea-evidence"
	if strings.TrimSpace(args.evidencePath) == "" {
		return resultInvalidArgument(out, command, "rea-evidence requires -analysis-output")
	}
	if extraArgs != 0 {
		return resultInvalidArgument(out, command, "rea-evidence does not accept positional arguments")
	}
	if args.viewPath != "" && strings.TrimSpace(args.providerVersion) == "" {
		return resultInvalidArgument(out, command, "-view requires -provider-version: the framing a delivery is checked against belongs to a version, and an undeclared version is not guessed")
	}
	if args.requestOut != "" && strings.TrimSpace(args.providerVersion) == "" {
		return resultInvalidArgument(out, command, "-request-out requires -provider-version: the source.kind a request may use belongs to a version, and an undeclared version is not guessed")
	}

	raw, err := os.ReadFile(args.evidencePath)
	if err != nil {
		return emitResult(out, resultFailure(command, "EVIDENCE_READ_FAILED", fmt.Errorf("read the saved external analysis output: %w", err)), 2)
	}
	receipt := externalanalysis.Receive(raw, probeExternalTool(args.toolPath), time.Now())
	derivedPath := ""

	if args.requestOut != "" {
		kinds, ok := externalanalysis.AcceptedSourceKinds(args.providerVersion)
		if !ok {
			return emitResult(out, resultFailure(command, resultCodeInvalidArgument, fmt.Errorf("tool version %q is not one this command has observed; the accepted source.kind set must be declared before a request is derived", args.providerVersion)), 2)
		}
		request, err := externalanalysis.DeriveViewRequest(receipt.EvidenceBody(), reaEvidenceViewRequest(), kinds)
		if err != nil {
			return emitResult(out, resultFailure(command, resultCodeInvalidArgument, err), 2)
		}
		if err := writeNoClobber(args.requestOut, request); err != nil {
			return emitResult(out, resultFailure(command, "EVIDENCE_REQUEST_WRITE_FAILED", err), 2)
		}
		receipt = receipt.WithRequest(request, kinds)
		derivedPath = args.requestOut
	}

	if args.requestPath != "" {
		request, err := os.ReadFile(args.requestPath)
		if err != nil {
			return emitResult(out, resultFailure(command, "EVIDENCE_READ_FAILED", fmt.Errorf("read the derived request: %w", err)), 2)
		}
		var kinds []string
		if strings.TrimSpace(args.providerVersion) != "" {
			kinds, _ = externalanalysis.AcceptedSourceKinds(args.providerVersion)
		}
		receipt = receipt.WithRequest(request, kinds)
	}

	if args.viewPath != "" {
		view, err := os.ReadFile(args.viewPath)
		if err != nil {
			return emitResult(out, resultFailure(command, "EVIDENCE_READ_FAILED", fmt.Errorf("read the saved view delivery: %w", err)), 2)
		}
		framing, ok := externalanalysis.FramingFor(args.providerVersion)
		if !ok {
			return emitResult(out, resultFailure(command, resultCodeInvalidArgument, fmt.Errorf("tool version %q is not one this command has observed; the framing must be declared before a view delivery is inspected", args.providerVersion)), 2)
		}
		receipt = receipt.WithView(view, framing)
	}

	report := cliResultEnvelope{
		SchemaVersion: resultSchemaVersion(command),
		Mode:          reaEvidenceMode(args),
		Result:        reaEvidenceResult{Receipt: receipt, DerivedRequestPath: derivedPath},
	}
	code := reaEvidenceExit(receipt.State)
	if err := receipt.VerifyPins(externalanalysis.Pins{
		EvidenceID:     args.expectedEvidenceID,
		Fingerprint:    args.expectedFingerprint,
		SourceRevision: args.expectedSourceRev,
	}); err != nil {
		report.Error = &cliFailure{Code: "EVIDENCE_PIN_MISMATCH", Message: err.Error()}
		return emitResult(out, report, 1)
	}
	if code != 0 {
		report.Error = &cliFailure{Code: reaEvidenceFailureCode(receipt.State), Message: receipt.Reason}
	}
	return emitResult(out, report, code)
}

// reaEvidenceViewRequest is the view this command derives: the bounded
// javascript-facts projection the coordination loop starts from. It is a
// starting point, not a taxonomy — the tool owns its view families, and a caller
// that needs another one derives the request through the package directly.
func reaEvidenceViewRequest() externalanalysis.ViewRequest {
	return externalanalysis.ViewRequest{
		Kind:     "javascript-facts",
		Selector: []byte(`{"kind":"javascript-facts","property_keys":[],"offset":0,"limit":32}`),
	}
}

// reaEvidenceMode names the shape the response has, the way the other read-only
// commands do: which optional inputs were supplied.
func reaEvidenceMode(args reaEvidenceOuterArgs) string {
	mode := "evidence"
	if args.requestPath != "" || args.requestOut != "" {
		mode = "request"
	}
	if args.viewPath != "" {
		if mode == "request" {
			return "request-view"
		}
		return "view"
	}
	return mode
}

// probeExternalTool records whether the external tool is present in this
// environment. It looks, and nothing more: the command never runs the tool, and
// a presence check is not an invocation.
//
// An explicitly declared path is checked on disk rather than trusted, so a
// caller that names a tool that is not there gets a receipt that says so
// instead of one that claims a fresh run was possible.
func probeExternalTool(explicitPath string) externalanalysis.Tool {
	tool := externalanalysis.Tool{Name: "rea"}
	if strings.TrimSpace(explicitPath) != "" {
		if _, err := os.Stat(explicitPath); err != nil {
			tool.AbsentReason = fmt.Sprintf("宣言されたtool path %s を読めません: %v", explicitPath, err)
			return tool
		}
		tool.Present = true
		tool.ResolvedPath = explicitPath
		return tool
	}
	path, err := exec.LookPath("rea")
	if err != nil {
		tool.AbsentReason = err.Error()
		return tool
	}
	tool.Present = true
	tool.ResolvedPath = path
	return tool
}

// writeNoClobber writes a derived file without overwriting an existing one: the
// derived request is the record of what was sent, and silently replacing it
// would destroy the record of an earlier send.
func writeNoClobber(path string, content []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("prepare %s: %w", path, err)
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if errors.Is(err, os.ErrExist) {
		return fmt.Errorf("%s already exists: the derived request is a record of what was sent, so it is not overwritten", path)
	}
	if err != nil {
		return fmt.Errorf("create %s: %w", path, err)
	}
	defer file.Close()
	if _, err := file.Write(content); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}
