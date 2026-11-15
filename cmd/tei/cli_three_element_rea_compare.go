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
)

const reaCompareEnvelopeVersion = "tei.cli.three-element.rea-compare/v0"

// reaCompareResult reuses the existing candidate review packet and its
// human-readable summary; it does not define a second comparison engine.
type reaCompareResult struct {
	ReviewPacket reacandidate.ReviewPacket `json:"reviewPacket"`
	Summary      []string                  `json:"summary"`
}

func failREACompare(code, message string, out io.Writer) (int, error) {
	return writeReaIntakeResponse(out, reaIntakeResponse{
		SchemaVersion: reaCompareEnvelopeVersion,
		Error: &cliInspectionFailure{
			Code:    code,
			Message: message,
		},
	}, 2)
}

// readREAComparisonCandidateSet checks the typed projection contract and
// content fingerprint before this process consumes the comparison input.
// These are consistency checks on untrusted bytes, NOT independent evidence
// provenance checks, Owner acceptance, or Current Grant checks.
func readREAComparisonCandidateSet(path string) (reacandidate.CandidateSet, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return reacandidate.CandidateSet{}, fmt.Errorf("read candidate set: %w", err)
	}
	var set reacandidate.CandidateSet
	if err := json.Unmarshal(raw, &set); err != nil {
		return reacandidate.CandidateSet{}, fmt.Errorf("decode candidate set: %w", err)
	}
	if set.SchemaVersion != "tei.rea.candidate-set/v0" ||
		set.Effect != reacandidate.EffectReadOnly {
		return reacandidate.CandidateSet{}, errors.New("unsupported candidate set schema or effect")
	}
	if set.Attempt.EvidenceID == "" || set.Attempt.Provider.ID == "" ||
		set.Attempt.Provider.Version == "" {
		return reacandidate.CandidateSet{}, errors.New("candidate set attempt provenance is incomplete")
	}
	if set.Fingerprint == "" || set.Fingerprint != reacandidate.SetFingerprint(set) {
		return reacandidate.CandidateSet{}, errors.New("candidate set fingerprint does not match its content")
	}
	limits := map[string]bool{}
	for _, limit := range set.AuthorityLimits {
		limits[limit] = true
	}
	for _, required := range reacandidate.CandidateAuthorityLimits() {
		if !limits[required] {
			return reacandidate.CandidateSet{}, errors.New("candidate set is missing a read-only authority limit")
		}
	}
	seen := map[string]bool{}
	for _, candidate := range set.Candidates {
		if candidate.CandidateID == "" || seen[candidate.CandidateID] {
			return reacandidate.CandidateSet{}, errors.New("candidate set contains an empty or duplicate candidate identity")
		}
		seen[candidate.CandidateID] = true
		if candidate.ReviewStatus != reacandidate.ReviewStatusRequiresOwner {
			return reacandidate.CandidateSet{}, errors.New("candidate set includes an unsupported candidate review status")
		}
		switch candidate.Kind {
		case reacandidate.CandidateKindDataElement, reacandidate.CandidateKindCrossReference:
		default:
			return reacandidate.CandidateSet{}, errors.New("candidate set includes an unsupported candidate kind")
		}
	}
	return set, nil
}

// runThreeElementREACompare is a product CLI consumer of the existing
// reacandidate comparison. Input is two already-generated CandidateSet/v0 JSON
// projections, not raw REA evidence and not the old rea-intake ReviewPacket.
// No source files, models, or grant decisions are modified.
func runThreeElementREACompare(args []string, out io.Writer) (int, error) {
	fs := flag.NewFlagSet("three-element rea-compare", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	beforePath := fs.String("before", "", "previous typed REA CandidateSet/v0 JSON")
	afterPath := fs.String("after", "", "current typed REA CandidateSet/v0 JSON")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printHelp(out, fs, "three-element rea-compare")
			return 0, nil
		}
		return failREACompare("INVALID_ARGUMENT", err.Error(), out)
	}
	if fs.NArg() != 0 || strings.TrimSpace(*beforePath) == "" ||
		strings.TrimSpace(*afterPath) == "" {
		return failREACompare("INVALID_ARGUMENT",
			"rea-compare requires -before and -after CandidateSet JSON paths and no positional arguments", out)
	}
	before, err := readREAComparisonCandidateSet(*beforePath)
	if err != nil {
		return failREACompare("BEFORE_CANDIDATE_SET_INVALID", err.Error(), out)
	}
	after, err := readREAComparisonCandidateSet(*afterPath)
	if err != nil {
		return failREACompare("AFTER_CANDIDATE_SET_INVALID", err.Error(), out)
	}
	comparison := reacandidate.CompareCandidateSets(before, after)
	packet := reacandidate.BuildReviewPacket(after, nil, nil, &comparison)
	return writeReaIntakeResponse(out, reaIntakeResponse{
		SchemaVersion: reaCompareEnvelopeVersion,
		OK:            true,
		Result: reaCompareResult{
			ReviewPacket: packet,
			Summary:      packet.Summary(),
		},
	}, 0)
}
