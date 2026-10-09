package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
)

const reviewSchema = "tei.reference.spec-review/v0"

type finding struct {
	Class    string `json:"class"`
	Source   string `json:"source"`
	Code     string `json:"code"`
	Severity string `json:"severity,omitempty"`
	Subject  string `json:"subject,omitempty"`
	Message  string `json:"message"`
}

type coverage struct {
	KnowledgeContext bool `json:"knowledgeContext"`
	CrossView        bool `json:"crossView"`
	SharedAcceptance bool `json:"sharedAcceptance"`
}

type reviewPacket struct {
	Schema          string    `json:"schema"`
	CaseID          string    `json:"caseId,omitempty"`
	Status          string    `json:"status"`
	Coverage        coverage  `json:"coverage"`
	Findings        []finding `json:"findings,omitempty"`
	MissingInputs   []string  `json:"missingInputs,omitempty"`
	AuthorityLimits []string  `json:"authorityLimits"`
}

type knowledgeContextInput struct {
	Subject struct {
		Kind string `json:"kind"`
		ID   string `json:"id"`
	} `json:"subject"`
	Unresolved []string `json:"unresolved"`
}

type crossViewDiagnostic struct {
	Code     string `json:"code"`
	Severity string `json:"severity"`
	Step     string `json:"step,omitempty"`
	Message  string `json:"message"`
}

type validationScopeRow struct {
	Step            string   `json:"step"`
	Status          string   `json:"status"`
	Reason          string   `json:"reason"`
	ApplicationOnly []string `json:"applicationOnly,omitempty"`
	ProcedureOnly   []string `json:"procedureOnly,omitempty"`
}

type crossViewInput struct {
	Procedure             string                `json:"procedure"`
	Diagnostics           []crossViewDiagnostic `json:"diagnostics"`
	ValidationScopeReview []validationScopeRow  `json:"validationScopeReview,omitempty"`
}

type staleInput struct {
	BaseModelRevision    string   `json:"baseModelRevision"`
	CurrentModelRevision string   `json:"currentModelRevision"`
	Options              []string `json:"options,omitempty"`
}

type acceptanceInput struct {
	Decision            string      `json:"decision,omitempty"`
	BaseModelRevision   string      `json:"baseModelRevision,omitempty"`
	ModelRevisionBefore string      `json:"modelRevisionBefore,omitempty"`
	Stale               *staleInput `json:"stale,omitempty"`
	Refusals            []struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Subject string `json:"subject,omitempty"`
	} `json:"refusals,omitempty"`
}

func main() {
	caseID := flag.String("case-id", "", "review case identifier")
	knowledgePath := flag.String("knowledge-context", "", "tei knowledge-context JSON")
	crossViewPath := flag.String("cross-view", "", "Business Procedure checker JSON")
	acceptancePath := flag.String("shared-acceptance", "", "Shared Acceptance Preview / Decision JSON")
	flag.Parse()

	packet, err := buildReviewPacket(*caseID, *knowledgePath, *crossViewPath, *acceptancePath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(packet); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func buildReviewPacket(caseID, knowledgePath, crossViewPath, acceptancePath string) (reviewPacket, error) {
	packet := reviewPacket{
		Schema: reviewSchema,
		CaseID: strings.TrimSpace(caseID),
		AuthorityLimits: []string{
			"review-packet-is-derived-not-acceptance",
			"no-finding-is-not-proof-of-specification-completeness",
			"missing-input-is-not-negative-evidence",
			"source-diagnostic-semantics-remain-owned-by-source-tool",
		},
	}
	if strings.TrimSpace(knowledgePath) == "" &&
		strings.TrimSpace(crossViewPath) == "" &&
		strings.TrimSpace(acceptancePath) == "" {
		return reviewPacket{}, errors.New("at least one review input is required")
	}

	if strings.TrimSpace(knowledgePath) == "" {
		packet.MissingInputs = append(packet.MissingInputs, "knowledge-context")
	} else {
		var input knowledgeContextInput
		if err := readJSONFile(knowledgePath, &input); err != nil {
			return reviewPacket{}, fmt.Errorf("knowledge contextを読めません: %w", err)
		}
		packet.Coverage.KnowledgeContext = true
		subjectID := input.Subject.ID
		subject := strings.TrimSpace(input.Subject.Kind + ":" + subjectID)
		for _, unresolved := range input.Unresolved {
			reason := strings.TrimSpace(unresolved)
			if reason == "" {
				continue
			}
			packet.Findings = append(packet.Findings, finding{
				Class:   "ambiguity",
				Source:  "knowledge-context",
				Code:    "KNOWLEDGE_UNRESOLVED",
				Subject: subject,
				Message: reason,
			})
		}
	}

	if strings.TrimSpace(crossViewPath) == "" {
		packet.MissingInputs = append(packet.MissingInputs, "cross-view")
	} else {
		var input crossViewInput
		if err := readJSONFile(crossViewPath, &input); err != nil {
			return reviewPacket{}, fmt.Errorf("cross-view reportを読めません: %w", err)
		}
		packet.Coverage.CrossView = true
		for _, diagnostic := range input.Diagnostics {
			class := "source-diagnostic"
			if diagnostic.Code == "BUSINESS_FUNCTION_DATASOURCE_ACCESS_CONFLICT" {
				class = "inconsistency"
			}
			packet.Findings = append(packet.Findings, finding{
				Class:    class,
				Source:   "business-procedure-check",
				Code:     diagnostic.Code,
				Severity: diagnostic.Severity,
				Subject:  diagnostic.Step,
				Message:  diagnostic.Message,
			})
		}
		for _, row := range input.ValidationScopeReview {
			switch row.Status {
			case "review-needed":
				packet.Findings = append(packet.Findings, finding{
					Class:   "cross-view-difference",
					Source:  "business-procedure-check",
					Code:    "VALIDATION_SCOPE_REVIEW_NEEDED",
					Subject: row.Step,
					Message: row.Reason,
				})
			case "unresolved-context", "unresolved-reference":
				packet.Findings = append(packet.Findings, finding{
					Class:   "ambiguity",
					Source:  "business-procedure-check",
					Code:    "VALIDATION_SCOPE_" + strings.ToUpper(strings.ReplaceAll(row.Status, "-", "_")),
					Subject: row.Step,
					Message: row.Reason,
				})
			}
		}
	}

	if strings.TrimSpace(acceptancePath) == "" {
		packet.MissingInputs = append(packet.MissingInputs, "shared-acceptance")
	} else {
		var input acceptanceInput
		if err := readJSONFile(acceptancePath, &input); err != nil {
			return reviewPacket{}, fmt.Errorf("Shared Acceptance出力を読めません: %w", err)
		}
		packet.Coverage.SharedAcceptance = true
		if input.Stale != nil {
			packet.Findings = append(packet.Findings, finding{
				Class:   "stale-premise",
				Source:  "shared-acceptance",
				Code:    "STALE_BASE_MODEL_REVISION",
				Subject: input.Decision,
				Message: fmt.Sprintf(
					"candidateの基準revision %s はcurrent revision %s と一致しません",
					input.Stale.BaseModelRevision,
					input.Stale.CurrentModelRevision,
				),
			})
		}
	}

	sort.Slice(packet.Findings, func(i, j int) bool {
		if packet.Findings[i].Class != packet.Findings[j].Class {
			return packet.Findings[i].Class < packet.Findings[j].Class
		}
		if packet.Findings[i].Source != packet.Findings[j].Source {
			return packet.Findings[i].Source < packet.Findings[j].Source
		}
		if packet.Findings[i].Code != packet.Findings[j].Code {
			return packet.Findings[i].Code < packet.Findings[j].Code
		}
		if packet.Findings[i].Subject != packet.Findings[j].Subject {
			return packet.Findings[i].Subject < packet.Findings[j].Subject
		}
		return packet.Findings[i].Message < packet.Findings[j].Message
	})
	sort.Strings(packet.MissingInputs)

	switch {
	case len(packet.Findings) > 0:
		packet.Status = "review-needed"
	case len(packet.MissingInputs) > 0:
		packet.Status = "partial-no-finding"
	default:
		packet.Status = "no-finding-in-provided-evidence"
	}
	return packet, nil
}

func readJSONFile(path string, out any) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	decoder := json.NewDecoder(f)
	if err := decoder.Decode(out); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err == io.EOF {
		return nil
	} else if err != nil {
		return err
	}
	return errors.New("JSON documentの後ろに追加valueがあります")
}
