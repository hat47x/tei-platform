package main

// Decision history is consumed as a read-only projection from the existing
// Shared Acceptance "decisions" command. This package neither verifies an
// actor's identity nor recreates the authority of Shared Acceptance.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"sort"
)

const decisionIndexSchema = "tei.reference.decision-index/v0"
const decisionSchema = "tei.reference.change-decision/v0"

type decisionIndexProjection struct {
	Schema        string                  `json:"schema"`
	ModelRevision string                  `json:"modelRevision"`
	Records       []decisionRecordExcerpt `json:"records"`
}

type decisionRecordExcerpt struct {
	Path                    string `json:"path"`
	Status                  string `json:"status"`
	SourceEdits             string `json:"sourceEdits"`
	ReviewBindingStatus     string `json:"reviewBindingStatus"`
	DecisionActorStatus     string `json:"decisionActorStatus"`
	DecisionAuthorityStatus string `json:"decisionAuthorityStatus"`
	Decision                struct {
		Schema             string `json:"schema"`
		Decision           string `json:"decision"`
		SourceChanged      bool   `json:"sourceChanged"`
		ModelRevisionAfter string `json:"modelRevisionAfter"`
		SemanticDiff       []struct {
			Kind   string `json:"kind"`
			ID     string `json:"id"`
			Change string `json:"change"`
		} `json:"semanticDiff"`
		SourceEdits []struct {
			Path             string `json:"path"`
			AfterFingerprint string `json:"afterFingerprint"`
		} `json:"sourceEdits"`
	} `json:"decision"`
}

// DecisionHistoryLink does not assert human approval. It points to a record
// reported by the existing Decision Index and binds the record's changed
// source bytes to an independently observed current Source Index witness.
type DecisionHistoryLink struct {
	RecordPath              string `json:"recordPath"`
	RecordStatus            string `json:"recordStatus"`
	ModelRevisionAfter      string `json:"modelRevisionAfter"`
	ReviewBindingStatus     string `json:"reviewBindingStatus"`
	DecisionActorStatus     string `json:"decisionActorStatus"`
	DecisionAuthorityStatus string `json:"decisionAuthorityStatus"`
}

func AttachDecisionHistory(report *WorkspaceAssessment, reader io.Reader) error {
	if report == nil || report.SubjectKind != "semantic-validation" {
		return fmt.Errorf("decision history requires a semantic-validation inspection")
	}
	// This output is informational even if the consumer supplies a fabricated
	// index. It cannot change RuleMeaning or ExecutionAllowed.
	report.DecisionHistoryStatus = "not-matched"
	report.DecisionHistoryLinks = []DecisionHistoryLink{}
	decoder := json.NewDecoder(io.LimitReader(reader, 16<<20+1))
	var index decisionIndexProjection
	if err := decoder.Decode(&index); err != nil {
		return fmt.Errorf("decode Shared Acceptance Decision Index: %w", err)
	}
	if index.Schema != decisionIndexSchema || index.ModelRevision == "" || len(index.Records) > 10000 {
		return fmt.Errorf("unsupported or invalid Shared Acceptance Decision Index")
	}
	var remaining any
	if err := decoder.Decode(&remaining); err != io.EOF {
		return fmt.Errorf("Decision Index input must contain exactly one JSON object")
	}
	report.DecisionIndexRevision = index.ModelRevision
	if report.SourceStatus != "traceable-source-declaration" || report.RuleMeaning != "canonical-validation-declared" {
		report.DecisionHistoryStatus = "subject-source-unresolved"
		return nil
	}

	// The existing Decision Index is responsible for content hash, review
	// binding, evidence, and sourceEdits status. Check the exact semantic ID
	// and observed current source fingerprint again to prevent an unrelated
	// accepted decision from qualifying this Validation.
	foundDecision := false
	for _, record := range index.Records {
		if record.Decision.Schema != decisionSchema ||
			record.Decision.Decision != "accepted" ||
			!record.Decision.SourceChanged {
			continue
		}
		exactSubject := false
		for _, change := range record.Decision.SemanticDiff {
			if change.Kind == "validation" && change.ID == report.SubjectID &&
				(change.Change == "added" || change.Change == "changed") {
				exactSubject = true
			}
		}
		if !exactSubject {
			continue
		}
		foundDecision = true
		sourceMatched := false
		for _, edit := range record.Decision.SourceEdits {
			for _, witness := range report.Witnesses {
				if edit.Path == witness.Path && edit.AfterFingerprint != "" &&
					edit.AfterFingerprint == witness.Fingerprint {
					sourceMatched = true
				}
			}
		}
		if !sourceMatched || record.SourceEdits != "intact" {
			continue
		}
		report.DecisionHistoryLinks = append(report.DecisionHistoryLinks, DecisionHistoryLink{
			RecordPath:              record.Path,
			RecordStatus:            record.Status,
			ModelRevisionAfter:      record.Decision.ModelRevisionAfter,
			ReviewBindingStatus:     record.ReviewBindingStatus,
			DecisionActorStatus:     record.DecisionActorStatus,
			DecisionAuthorityStatus: record.DecisionAuthorityStatus,
		})
	}
	sort.Slice(report.DecisionHistoryLinks, func(i, j int) bool {
		return report.DecisionHistoryLinks[i].RecordPath < report.DecisionHistoryLinks[j].RecordPath
	})
	switch {
	case len(report.DecisionHistoryLinks) > 0:
		report.DecisionHistoryStatus = "reported-source-matched"
	case foundDecision:
		report.DecisionHistoryStatus = "recorded-subject-source-not-current"
	default:
		report.DecisionHistoryStatus = "not-matched"
	}
	// Deliberately retain human-acceptance-not-verified; even a verified review
	// projection fingerprint does not attest actor identity or runtime authority.
	return nil
}

// Compile-time assertion that a real JSON input can include nested record
// payloads. Deliberately do not use DisallowUnknownFields: the Decision Index
// producer owns all remaining fields, including diagnostic and review details.
var _ = bytes.MinRead
