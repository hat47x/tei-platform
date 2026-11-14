package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"

	"tei/internal/externalanalysis"
	"tei/internal/operations"
)

// externalCandidatesCLIResponse is the local CLI envelope. Its result is the
// existing typed Operation view, not a second REA parser or accepted TEI model.
type externalCandidatesCLIResponse struct {
	SchemaVersion string                            `json:"schemaVersion"`
	OK            bool                              `json:"ok"`
	Result        *operations.ExternalCandidateView `json:"result,omitempty"`
	Error         *cliInspectionFailure             `json:"error,omitempty"`
}

type externalCandidatesCLIArgs struct {
	evidencePath           string
	toolVersion            string
	observationID          string
	relationID             string
	expectedEvidenceID     string
	expectedReceivedDigest string
	expectedSourceRevision string
}

// Keep the help grammar and actual parser on the same FlagSet.
func externalCandidatesCLIFlags(args *externalCandidatesCLIArgs) *flag.FlagSet {
	fs := flag.NewFlagSet("external-candidates", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&args.evidencePath, "evidence-path", "", "saved local REA Evidence output; not a remote file selector")
	fs.StringVar(&args.toolVersion, "tool-version", "", "declared producer version, e.g. 6.8.0")
	fs.StringVar(&args.observationID, "observation-id", "", "exact external observation/node ID")
	fs.StringVar(&args.relationID, "relation-id", "", "exact external directed relation ID")
	fs.StringVar(&args.expectedEvidenceID, "expected-evidence-id", "", "optional previously inspected Evidence ID")
	fs.StringVar(&args.expectedReceivedDigest, "expected-received-digest", "", "optional raw received-byte digest pin")
	fs.StringVar(&args.expectedSourceRevision, "expected-source-revision", "", "optional producer-declared Source revision pin")
	return fs
}

func writeExternalCandidatesCLIResponse(out io.Writer, response externalCandidatesCLIResponse, exit int) (int, error) {
	encoder := json.NewEncoder(out)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(response); err != nil {
		return 2, err
	}
	return exit, nil
}

func runExternalCandidatesCommand(argv []string, out io.Writer) (int, error) {
	response := externalCandidatesCLIResponse{SchemaVersion: "tei.cli.external-candidates/v0"}
	fail := func(code, message string, exit int) (int, error) {
		response.Error = &cliInspectionFailure{Code: code, Message: message}
		return writeExternalCandidatesCLIResponse(out, response, exit)
	}

	var args externalCandidatesCLIArgs
	fs := externalCandidatesCLIFlags(&args)
	if err := fs.Parse(argv); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printHelp(out, fs, "external-candidates")
			return 0, nil
		}
		return fail("INVALID_ARGUMENT", err.Error(), 2)
	}
	if fs.NArg() != 0 {
		return fail("INVALID_ARGUMENT", "external-candidates does not accept positional arguments", 2)
	}
	if err := validateRepeatableArguments("external-candidates", argv); err != nil {
		return fail("INVALID_ARGUMENT", err.Error(), 2)
	}
	if err := validateNoEmptyStringArguments(fs, "external-candidates"); err != nil {
		return fail("INVALID_ARGUMENT", err.Error(), 2)
	}
	for _, required := range []struct{ name, value string }{
		{"evidence-path", args.evidencePath},
		{"tool-version", args.toolVersion},
		{"observation-id", args.observationID},
		{"relation-id", args.relationID},
	} {
		if strings.TrimSpace(required.value) == "" || required.value != strings.TrimSpace(required.value) {
			return fail("INVALID_ARGUMENT", fmt.Sprintf("external-candidates requires exact nonempty -%s", required.name), 2)
		}
	}
	for _, optional := range []string{args.expectedEvidenceID, args.expectedReceivedDigest, args.expectedSourceRevision} {
		if optional != strings.TrimSpace(optional) {
			return fail("INVALID_ARGUMENT", "external-candidates pin values must not have surrounding whitespace", 2)
		}
	}

	// Only inspect the local environment; do not execute the external tool.
	// The saved Evidence is usable even when "rea" is not installed here.
	path, lookupErr := exec.LookPath("rea")
	tool := externalanalysis.Tool{Name: "rea", Present: lookupErr == nil}
	if lookupErr != nil {
		tool.AbsentReason = lookupErr.Error()
	} else {
		tool.ResolvedPath = path
	}
	view, err := operations.ExternalCandidates(operations.ExternalCandidateRequest{
		EvidencePath:           args.evidencePath,
		ToolVersion:            args.toolVersion,
		ObservationID:          args.observationID,
		RelationID:             args.relationID,
		ExpectedEvidenceID:     args.expectedEvidenceID,
		ExpectedReceivedDigest: args.expectedReceivedDigest,
		ExpectedSourceRevision: args.expectedSourceRevision,
		Tool:                   tool,
	}, time.Now().UTC())
	if err != nil {
		// A typed Operation failure remains distinct from an argument failure.
		// When the Operation has already received an Evidence attempt, preserve
		// its non-authoritative coverage and attempt details for local diagnosis.
		if view.Attempt.EvidenceID != "" {
			response.Result = &view
		}
		var candidateErr *operations.ExternalCandidateError
		if errors.As(err, &candidateErr) {
			exit := 1
			if candidateErr.Code == operations.ExternalCandidateInvalidRequest ||
				candidateErr.Code == operations.ExternalCandidateReadFailed {
				exit = 2
			}
			return fail(candidateErr.Code, candidateErr.Message, exit)
		}
		return fail("EXTERNAL_CANDIDATE_FAILED", err.Error(), 1)
	}
	response.OK = true
	response.Result = &view
	return writeExternalCandidatesCLIResponse(out, response, 0)
}
