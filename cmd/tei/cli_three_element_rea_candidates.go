package main

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"tei/internal/reacandidate"
)

const reaCandidatesEnvelopeVersion = "tei.cli.three-element.rea-candidates/v0"

// reaCandidatesResult carries the typed read-only projection and its review.
// A raw evidence digest proves only which bytes this CLI read; a candidate
// fingerprint identifies the derived projection, not Owner acceptance.
type reaCandidatesResult struct {
	RawEvidenceFingerprint string                    `json:"rawEvidenceFingerprint"`
	CandidateSet           reacandidate.CandidateSet `json:"candidateSet"`
	ReviewPacket           reacandidate.ReviewPacket `json:"reviewPacket"`
	RelationQuestion       *reacandidate.RelationQuestion `json:"relationQuestion,omitempty"`
	Summary                []string                  `json:"summary"`
	CandidateOutputWritten bool                      `json:"candidateOutputWritten"`
}

func failREACandidates(code, message string, out io.Writer) (int, error) {
	return writeReaIntakeResponse(out, reaIntakeResponse{
		SchemaVersion: reaCandidatesEnvelopeVersion,
		Error: &cliInspectionFailure{
			Code:    code,
			Message: message,
		},
	}, 2)
}

// writeNewREACandidateSet refuses to overwrite either an existing projection
// or the original evidence. It writes an artifact, never a TEI Source edit.
func writeNewREACandidateSet(destination string, set reacandidate.CandidateSet) error {
	encoded, err := json.MarshalIndent(set, "", "  ")
	if err != nil {
		return fmt.Errorf("encode candidate set: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}
	file, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("create new candidate output without overwrite: %w", err)
	}
	if _, err := file.Write(append(encoded, '\n')); err != nil {
		_ = file.Close()
		_ = os.Remove(destination)
		return fmt.Errorf("write candidate output: %w", err)
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(destination)
		return fmt.Errorf("close candidate output: %w", err)
	}
	return nil
}

// runThreeElementREACandidates uses the existing REA decoder and candidate
// generator. It does not invoke the external tool, bind source bytes, accept
// meaning, or write any TEI Source.
func runThreeElementREACandidates(args []string, out io.Writer) (int, error) {
	fs := flag.NewFlagSet("three-element rea-candidates", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	evidencePath := fs.String("evidence", "", "saved external REA JSON analysis document")
	outputPath := fs.String("output", "", "new path for typed CandidateSet/v0 JSON (never overwrites)")
	relationID := fs.String("relation-id", "", "producer edge occurrence ID for one read-only relation question (does not infer a Function)")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printHelp(out, fs, "three-element rea-candidates")
			return 0, nil
		}
		return failREACandidates("INVALID_ARGUMENT", err.Error(), out)
	}
	if fs.NArg() != 0 || strings.TrimSpace(*evidencePath) == "" {
		return failREACandidates("INVALID_ARGUMENT",
			"rea-candidates requires -evidence and no positional arguments", out)
	}
	raw, err := os.ReadFile(*evidencePath)
	if err != nil {
		return failREACandidates("EVIDENCE_READ_FAILED", err.Error(), out)
	}
	attempt, err := reacandidate.DecodeAttemptEvidence(raw)
	if err != nil {
		var evidenceError *reacandidate.EvidenceError
		if errors.As(err, &evidenceError) {
			return failREACandidates(evidenceError.Code, evidenceError.Message, out)
		}
		return failREACandidates("EVIDENCE_DECODE_FAILED", err.Error(), out)
	}
	set, err := reacandidate.GenerateCandidates(attempt)
	if err != nil {
		var evidenceError *reacandidate.EvidenceError
		if errors.As(err, &evidenceError) {
			return failREACandidates(evidenceError.Code, evidenceError.Message, out)
		}
		return failREACandidates("CANDIDATE_GENERATION_FAILED", err.Error(), out)
	}
	packet := reacandidate.BuildReviewPacket(set, nil, nil, nil)
	var question *reacandidate.RelationQuestion
	// Omitting the optional flag keeps the original full candidate review.
	// Supplying an empty ID is a selection error, not an implicit omission.
	relationRequested := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == "relation-id" {
			relationRequested = true
		}
	})
	if relationRequested {
		selected, ok := reacandidate.SelectRelationQuestion(packet, *relationID)
		if !ok {
			return failREACandidates("RELATION_SELECTION_UNAVAILABLE",
				"the selected producer relation occurrence was not uniquely observed in this attempt; no Function/Data meaning was inferred", out)
		}
		question = &selected
	}
	written := false
	if strings.TrimSpace(*outputPath) != "" {
		if err := writeNewREACandidateSet(*outputPath, set); err != nil {
			return failREACandidates("CANDIDATE_OUTPUT_WRITE_FAILED", err.Error(), out)
		}
		written = true
	}
	return writeReaIntakeResponse(out, reaIntakeResponse{
		SchemaVersion: reaCandidatesEnvelopeVersion,
		OK:            true,
		Result: reaCandidatesResult{
			RawEvidenceFingerprint: fmt.Sprintf("sha256:%x", sha256.Sum256(raw)),
			CandidateSet:           set,
			ReviewPacket:           packet,
			RelationQuestion:       question,
			Summary:                packet.Summary(),
			CandidateOutputWritten: written,
		},
	}, 0)
}
