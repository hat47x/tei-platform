package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"tei/internal/reacandidate"
	"tei/internal/sourceindex"
)

// reaIntakeArgs is the grammar of `tei three-element rea-intake`. The FlagSet
// is built from this struct exactly once, so the -h listing and the parsed
// arguments cannot drift into two hand-written copies.
type reaIntakeArgs struct {
	Evidence  string
	Workspace string
	Model     string
	Mapping   string
	Before    string
	Output    string
}

// reaIntakeFlagSet registers every flag the subcommand reads.
func reaIntakeFlagSet(args *reaIntakeArgs) *flag.FlagSet {
	fs := flag.NewFlagSet("three-element rea-intake", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&args.Evidence, "evidence", "", "path to one REA attempt JSON document (required)")
	fs.StringVar(&args.Workspace, "workspace", "", "workspace whose current bytes the candidates are compared against")
	fs.StringVar(&args.Model, "model", "", "three-element model JSON to preview the adoption against (never written)")
	fs.StringVar(&args.Mapping, "mapping", "", "JSON object mapping candidateId to {kind,id} canonical subjects; requires -workspace")
	fs.StringVar(&args.Before, "before", "", "previous candidate set JSON to compare against")
	fs.StringVar(&args.Output, "output", "", "path to write the review packet JSON")
	return fs
}

// reaIntakeResponse is the CLI envelope of the candidate-only intake. It is
// deliberately the same shape as the other three-element subcommands so a
// caller parses one envelope family.
type reaIntakeResponse struct {
	SchemaVersion string                `json:"schemaVersion"`
	OK            bool                  `json:"ok"`
	Result        interface{}           `json:"result,omitempty"`
	Error         *cliInspectionFailure `json:"error,omitempty"`
}

// runThreeElementReaIntake is the review entry point of the candidate-only
// intake. It reads one external REA attempt, projects structure candidates,
// optionally binds them to one workspace, optionally previews an adoption
// against a three-element model, optionally compares with a previous candidate
// set, and writes the review packet.
//
// It has no accept path: acceptedMeaning stays false and ownerDecision stays
// pending regardless of the flags given.
func runThreeElementReaIntake(args []string, out io.Writer) (int, error) {
	var parsed reaIntakeArgs
	fs := reaIntakeFlagSet(&parsed)
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printHelp(out, fs, "three-element rea-intake")
			return 0, nil
		}
		return failReaIntake("INVALID_ARGUMENT", err.Error(), 2, out)
	}
	if fs.NArg() != 0 {
		return failReaIntake("INVALID_ARGUMENT",
			fmt.Sprintf("three-element rea-intake does not take positional arguments: %s", strings.Join(fs.Args(), " ")), 2, out)
	}
	if strings.TrimSpace(parsed.Evidence) == "" {
		return failReaIntake("INVALID_ARGUMENT", "three-element rea-intake requires -evidence", 2, out)
	}

	evidence, err := readReaEvidence(parsed.Evidence)
	if err != nil {
		var evidenceErr *reacandidate.EvidenceError
		if errors.As(err, &evidenceErr) {
			return failReaIntake(evidenceErr.Code, evidenceErr.Message, 2, out)
		}
		return failReaIntake("EVIDENCE_READ_FAILED", err.Error(), 2, out)
	}
	set, err := reacandidate.GenerateCandidates(evidence)
	if err != nil {
		return failReaIntake("CANDIDATE_GENERATION_FAILED", err.Error(), 2, out)
	}

	var binding *reacandidate.BindingResult
	if parsed.Workspace != "" {
		port, err := reacandidate.NewWorkspaceReadOnlyModel(parsed.Workspace)
		if err != nil {
			return failReaIntake("INVALID_ARGUMENT", err.Error(), 2, out)
		}
		mappings, err := readReaMapping(parsed.Mapping)
		if err != nil {
			return failReaIntake("MAPPING_READ_FAILED", err.Error(), 2, out)
		}
		result, err := reacandidate.BindCandidates(port, set, mappings)
		if err != nil {
			return failReaIntake("BINDING_FAILED", err.Error(), 2, out)
		}
		binding = &result
	} else if parsed.Mapping != "" {
		// A mapping without a workspace would silently resolve nothing while
		// looking like it was applied. Refuse instead of pretending.
		return failReaIntake("INVALID_ARGUMENT", "-mapping requires -workspace; a mapping alone resolves nothing", 2, out)
	}

	var adoption *reacandidate.AdoptionPreview
	if parsed.Model != "" {
		model, err := loadThreeElementModel(parsed.Model)
		if err != nil {
			return failReaIntake("MODEL_LOAD_FAILED", err.Error(), 2, out)
		}
		preview, err := reacandidate.PreviewAdoption(set, model)
		if err != nil {
			return failReaIntake("ADOPTION_PREVIEW_FAILED", err.Error(), 2, out)
		}
		adoption = &preview
	}

	var comparison *reacandidate.CandidateSetComparison
	if parsed.Before != "" {
		previous, err := readReaCandidateSet(parsed.Before)
		if err != nil {
			return failReaIntake("BEFORE_SET_READ_FAILED", err.Error(), 2, out)
		}
		cmp := reacandidate.CompareCandidateSets(previous, set)
		comparison = &cmp
	}

	packet := reacandidate.BuildReviewPacket(set, binding, adoption, comparison)
	raw, err := json.MarshalIndent(packet, "", "  ")
	if err != nil {
		return failReaIntake("PACKET_ENCODE_FAILED", err.Error(), 2, out)
	}
	if parsed.Output != "" {
		if err := os.WriteFile(parsed.Output, append(raw, '\n'), 0o644); err != nil {
			return failReaIntake("OUTPUT_WRITE_FAILED", err.Error(), 2, out)
		}
	}
	response := reaIntakeResponse{
		SchemaVersion: "tei.cli.three-element.rea-intake/v0",
		OK:            true,
		Result:        packet,
	}
	return writeReaIntakeResponse(out, response, 0)
}

func failReaIntake(code, message string, exit int, out io.Writer) (int, error) {
	return writeReaIntakeResponse(out, reaIntakeResponse{
		SchemaVersion: "tei.cli.three-element.rea-intake/v0",
		Error:         &cliInspectionFailure{Code: code, Message: message},
	}, exit)
}

func writeReaIntakeResponse(out io.Writer, response reaIntakeResponse, exit int) (int, error) {
	encoder := json.NewEncoder(out)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(response); err != nil {
		return 2, err
	}
	return exit, nil
}

func readReaEvidence(path string) (reacandidate.AttemptEvidence, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return reacandidate.AttemptEvidence{}, err
	}
	return reacandidate.DecodeAttemptEvidence(raw)
}

// readReaMapping decodes the optional canonical mapping. An empty path means
// no mapping, which is a valid state: every candidate stays unresolved.
func readReaMapping(path string) (map[string]sourceindex.Subject, error) {
	if strings.TrimSpace(path) == "" {
		return nil, nil
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var mappings map[string]sourceindex.Subject
	if err := json.Unmarshal(raw, &mappings); err != nil {
		return nil, fmt.Errorf("mapping must be a JSON object of candidateId to {kind,id}: %w", err)
	}
	for candidateID, subject := range mappings {
		if strings.TrimSpace(candidateID) == "" || strings.TrimSpace(subject.Kind) == "" || strings.TrimSpace(subject.ID) == "" {
			return nil, fmt.Errorf("mapping for %q must name a candidate, a subject kind and a subject id", candidateID)
		}
	}
	return mappings, nil
}

// readReaCandidateSet decodes a previous packet or candidate set. A packet
// carries the candidates under `candidates`; a bare set does too, so one
// decoder serves both shapes, and a packet's extra fields are ignored.
func readReaCandidateSet(path string) (reacandidate.CandidateSet, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return reacandidate.CandidateSet{}, err
	}
	var set reacandidate.CandidateSet
	if err := json.Unmarshal(raw, &set); err != nil {
		return reacandidate.CandidateSet{}, err
	}
	if len(set.Candidates) == 0 && set.SchemaVersion == "" {
		return reacandidate.CandidateSet{}, fmt.Errorf("file has no candidate set: %s", path)
	}
	return set, nil
}
