// Package cswprobe is a read-only probe of the typed hand-off boundary proposed
// for Cultural Substrate Weaving (CSW). It models only the identity, provenance
// and author decision around materials and outputs. It never reads, classifies
// or summarises the prose that the author produces.
//
// This package is research scaffolding. It is not a CSW runtime, it does not
// write a Workspace, and it grants no authority over creative content.
package cswprobe

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"
)

const (
	Schema = "csw.handoff/v0"

	SeverityError = "error"

	// Attribution values are the same set the TEI Workspace declares as
	// csw.attribution. The set records a label the author chose; it is not a
	// classification computed from the body.
	AttributionAuthorStated      = "author_stated"
	AttributionGeneratedProposal = "generated_proposal"
	AttributionSourceMaterial    = "source_material"

	DecisionAdopt      = "adopt"
	DecisionReject     = "reject"
	DecisionHold       = "hold"
	DecisionUnresolved = "unresolved"
)

var sha256Pattern = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)

// Handoff is the complete record set for one hand-off file. The schema field is
// required so a later revision can be refused instead of silently re-read.
type Handoff struct {
	Schema      string       `json:"schema"`
	StudyInputs []StudyInput `json:"study_inputs"`
	Runs        []StudyRun   `json:"runs"`
	Fragments   []Fragment   `json:"fragments"`
	Decisions   []Decision   `json:"decisions"`
}

type StudyInput struct {
	InputID        string `json:"input_id"`
	SourceRevision string `json:"source_revision"`
	ContentSHA256  string `json:"content_sha256"`
	Ordinal        int    `json:"ordinal"`
}

type StudyRun struct {
	RunID              string `json:"run_id"`
	ModelRef           string `json:"model_ref"`
	HarnessRef         string `json:"harness_ref"`
	IndependentContext bool   `json:"independent_context"`
	StartedAt          string `json:"started_at"`
}

// Fragment carries no body field. The prose stays in the author's files; the
// probe holds a reference and a hash, which is all a later audit needs to check
// that the bytes it reads are the bytes that were handed off.
type Fragment struct {
	FragmentID   string `json:"fragment_id"`
	StudyInputID string `json:"study_input_id"`
	RunID        string `json:"run_id"`
	Ordinal      int    `json:"ordinal"`
	Attribution  string `json:"attribution"`
	BodySHA256   string `json:"body_sha256"`
	BodyRef      string `json:"body_ref"`
}

type Decision struct {
	DecisionID      string `json:"decision_id"`
	FragmentID      string `json:"fragment_id"`
	Decision        string `json:"decision"`
	AuthorRef       string `json:"author_ref"`
	DecidedAt       string `json:"decided_at"`
	RationaleSHA256 string `json:"rationale_sha256,omitempty"`
}

// Diagnostic names one refusal. It never proposes a meaning for a fragment.
type Diagnostic struct {
	Code     string `json:"code"`
	Severity string `json:"severity"`
	RecordID string `json:"record_id,omitempty"`
	Message  string `json:"message"`
}

// DecodeHandoff rejects any field the typed boundary does not declare. A body
// passed as a field is therefore refused rather than ignored, which keeps the
// probe from becoming a second, untyped store.
func DecodeHandoff(data []byte) (Handoff, error) {
	dec := json.NewDecoder(strings.NewReader(string(data)))
	dec.DisallowUnknownFields()
	var h Handoff
	if err := dec.Decode(&h); err != nil {
		return Handoff{}, fmt.Errorf("decode hand-off: %w", err)
	}
	if dec.More() {
		return Handoff{}, fmt.Errorf("decode hand-off: trailing data after the record set")
	}
	return h, nil
}

// Validate checks identity, referential integrity and declared labels. It does
// not rank fragments, infer topics, or promote a fragment to a decision.
func Validate(h Handoff) []Diagnostic {
	var out []Diagnostic
	add := func(code, id, format string, args ...any) {
		out = append(out, Diagnostic{Code: code, Severity: SeverityError, RecordID: id, Message: fmt.Sprintf(format, args...)})
	}

	if h.Schema != Schema {
		add("CSW_HANDOFF_SCHEMA_UNSUPPORTED", "", "schema %q is not %q", h.Schema, Schema)
	}

	inputs := map[string]StudyInput{}
	for _, in := range h.StudyInputs {
		if in.InputID == "" {
			add("CSW_HANDOFF_INPUT_ID_MISSING", "", "a study input has no input_id")
			continue
		}
		if _, dup := inputs[in.InputID]; dup {
			add("CSW_HANDOFF_INPUT_DUPLICATE", in.InputID, "study input %s is declared more than once", in.InputID)
			continue
		}
		if in.SourceRevision == "" {
			add("CSW_HANDOFF_SOURCE_REVISION_MISSING", in.InputID, "study input %s has no source_revision", in.InputID)
		}
		if !sha256Pattern.MatchString(in.ContentSHA256) {
			add("CSW_HANDOFF_CONTENT_HASH_INVALID", in.InputID, "study input %s content_sha256 %q is not sha256:<64 hex>", in.InputID, in.ContentSHA256)
		}
		if in.Ordinal <= 0 {
			add("CSW_HANDOFF_ORDINAL_NOT_POSITIVE", in.InputID, "study input %s ordinal %d is not positive", in.InputID, in.Ordinal)
		}
		inputs[in.InputID] = in
	}

	runs := map[string]StudyRun{}
	for _, run := range h.Runs {
		if run.RunID == "" {
			add("CSW_HANDOFF_RUN_ID_MISSING", "", "a study run has no run_id")
			continue
		}
		if _, dup := runs[run.RunID]; dup {
			add("CSW_HANDOFF_RUN_DUPLICATE", run.RunID, "study run %s is declared more than once", run.RunID)
			continue
		}
		if run.ModelRef == "" {
			add("CSW_HANDOFF_MODEL_REF_MISSING", run.RunID, "study run %s has no model_ref", run.RunID)
		}
		if _, err := time.Parse(time.RFC3339, run.StartedAt); err != nil {
			add("CSW_HANDOFF_STARTED_AT_INVALID", run.RunID, "study run %s started_at %q is not RFC3339", run.RunID, run.StartedAt)
		}
		runs[run.RunID] = run
	}

	fragments := map[string]Fragment{}
	ordinals := map[string]map[int]string{}
	for _, fragment := range h.Fragments {
		if fragment.FragmentID == "" {
			add("CSW_HANDOFF_FRAGMENT_ID_MISSING", "", "a fragment has no fragment_id")
			continue
		}
		if _, dup := fragments[fragment.FragmentID]; dup {
			add("CSW_HANDOFF_FRAGMENT_DUPLICATE", fragment.FragmentID, "fragment %s is declared more than once", fragment.FragmentID)
			continue
		}
		if _, ok := inputs[fragment.StudyInputID]; !ok {
			add("CSW_HANDOFF_FRAGMENT_INPUT_UNRESOLVED", fragment.FragmentID, "fragment %s references unknown study input %q", fragment.FragmentID, fragment.StudyInputID)
		}
		if _, ok := runs[fragment.RunID]; !ok {
			add("CSW_HANDOFF_FRAGMENT_RUN_UNRESOLVED", fragment.FragmentID, "fragment %s references unknown study run %q", fragment.FragmentID, fragment.RunID)
		}
		if !knownAttribution(fragment.Attribution) {
			add("CSW_HANDOFF_ATTRIBUTION_UNKNOWN", fragment.FragmentID, "fragment %s attribution %q is not a declared value", fragment.FragmentID, fragment.Attribution)
		}
		if fragment.Ordinal <= 0 {
			add("CSW_HANDOFF_ORDINAL_NOT_POSITIVE", fragment.FragmentID, "fragment %s ordinal %d is not positive", fragment.FragmentID, fragment.Ordinal)
		} else {
			if ordinals[fragment.RunID] == nil {
				ordinals[fragment.RunID] = map[int]string{}
			}
			if other, dup := ordinals[fragment.RunID][fragment.Ordinal]; dup {
				add("CSW_HANDOFF_ORDINAL_DUPLICATE", fragment.FragmentID, "fragment %s reuses ordinal %d of run %s already used by %s", fragment.FragmentID, fragment.Ordinal, fragment.RunID, other)
			} else {
				ordinals[fragment.RunID][fragment.Ordinal] = fragment.FragmentID
			}
		}
		if !sha256Pattern.MatchString(fragment.BodySHA256) {
			add("CSW_HANDOFF_BODY_HASH_INVALID", fragment.FragmentID, "fragment %s body_sha256 %q is not sha256:<64 hex>", fragment.FragmentID, fragment.BodySHA256)
		}
		if fragment.BodyRef == "" {
			add("CSW_HANDOFF_BODY_REF_MISSING", fragment.FragmentID, "fragment %s has no body_ref", fragment.FragmentID)
		}
		fragments[fragment.FragmentID] = fragment
	}

	decided := map[string]string{}
	for _, decision := range h.Decisions {
		if decision.DecisionID == "" {
			add("CSW_HANDOFF_DECISION_ID_MISSING", "", "a decision has no decision_id")
			continue
		}
		if _, ok := fragments[decision.FragmentID]; !ok {
			add("CSW_HANDOFF_DECISION_FRAGMENT_UNRESOLVED", decision.DecisionID, "decision %s references unknown fragment %q", decision.DecisionID, decision.FragmentID)
		}
		if !knownDecision(decision.Decision) {
			add("CSW_HANDOFF_DECISION_UNKNOWN", decision.DecisionID, "decision %s value %q is not a declared value", decision.DecisionID, decision.Decision)
		}
		if decision.AuthorRef == "" {
			add("CSW_HANDOFF_DECISION_AUTHOR_MISSING", decision.DecisionID, "decision %s has no author_ref; a process cannot record an author decision", decision.DecisionID)
		}
		if _, err := time.Parse(time.RFC3339, decision.DecidedAt); err != nil {
			add("CSW_HANDOFF_DECIDED_AT_INVALID", decision.DecisionID, "decision %s decided_at %q is not RFC3339", decision.DecisionID, decision.DecidedAt)
		}
		if decision.RationaleSHA256 != "" && !sha256Pattern.MatchString(decision.RationaleSHA256) {
			add("CSW_HANDOFF_RATIONALE_HASH_INVALID", decision.DecisionID, "decision %s rationale_sha256 %q is not sha256:<64 hex>", decision.DecisionID, decision.RationaleSHA256)
		}
		if other, dup := decided[decision.FragmentID]; dup {
			add("CSW_HANDOFF_DECISION_DUPLICATE", decision.DecisionID, "fragment %s already has decision %s", decision.FragmentID, other)
		} else {
			decided[decision.FragmentID] = decision.DecisionID
		}
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].RecordID != out[j].RecordID {
			return out[i].RecordID < out[j].RecordID
		}
		return out[i].Code < out[j].Code
	})
	return out
}

func knownAttribution(value string) bool {
	switch value {
	case AttributionAuthorStated, AttributionGeneratedProposal, AttributionSourceMaterial:
		return true
	default:
		return false
	}
}

func knownDecision(value string) bool {
	switch value {
	case DecisionAdopt, DecisionReject, DecisionHold, DecisionUnresolved:
		return true
	default:
		return false
	}
}
