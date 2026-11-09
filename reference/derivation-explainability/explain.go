// Package derivationexplainability is a D7 read-only reference boundary.
// It projects one evaluated derivation value into a small machine-readable
// explanation and redacts it per disclosure decision.
//
// It does not define derivation rules (D1 #1637), query plans (D2 #1640),
// freshness or confirmed facts (D3 #1641), authorization (D5 #1693), the
// Canonical trace or Source Index (#1165/#1178) or the Scenario SSOT (#1180).
package derivationexplainability

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"tei/internal/semantic/derivationref"
)

var (
	ErrInvalidEvidence   = errors.New("derivation explanation evidence is invalid")
	ErrNotDisclosable    = errors.New("derivation explanation is not disclosable")
	ErrDisclosureFailure = errors.New("derivation explanation disclosure unavailable")
)

// Outcome is the coarse, low-sensitivity class of a resulting value. It never
// distinguishes why no value was produced; that is Reason.
type Outcome string

const (
	OutcomeValue   Outcome = "value"
	OutcomeNull    Outcome = "null"
	OutcomeNoValue Outcome = "no-value"
)

// Reason is the fine-grained cause of an Outcome. It can disclose the
// existence of inputs or related rows, so a projection may withhold it.
type Reason string

const (
	ReasonNone              Reason = ""
	ReasonInputMissing      Reason = "input-missing"
	ReasonAbsentRelation    Reason = "absent-relation"
	ReasonMultipleRelations Reason = "multiple-relations"
	ReasonDenied            Reason = "authorization-denied"
	ReasonNotEvaluable      Reason = "not-evaluable"
	ReasonUnknown           Reason = "unknown"
)

// Classify maps a D1 evaluation result or error to a coarse Outcome and a
// fine Reason. It never folds a non-NULL cause into NULL or 0.
//
// D1 wraps every input failure in ErrInput, so a relation-cardinality reason
// cannot be recovered by parsing the error text. A caller that already knows
// the failing input state must use ReasonForState instead.
func Classify(value derivationref.Input, err error) (Outcome, Reason) {
	switch {
	case err == nil && value.State == derivationref.Present:
		return OutcomeValue, ReasonNone
	case err == nil && value.State == derivationref.Null:
		return OutcomeNull, ReasonNone
	case errors.Is(err, derivationref.ErrInput):
		return OutcomeNoValue, ReasonInputMissing
	case errors.Is(err, derivationref.ErrOperator),
		errors.Is(err, derivationref.ErrCycle),
		errors.Is(err, derivationref.ErrImpure):
		return OutcomeNoValue, ReasonNotEvaluable
	case err != nil:
		return OutcomeNoValue, ReasonNotEvaluable
	default:
		return OutcomeNoValue, ReasonUnknown
	}
}

// ReasonForState names the fine reason for a D1 input state that prevented
// evaluation, without parsing an error string.
func ReasonForState(state derivationref.InputState) Reason {
	switch state {
	case derivationref.Missing:
		return ReasonInputMissing
	case derivationref.AbsentRelation:
		return ReasonAbsentRelation
	case derivationref.MultipleRelations:
		return ReasonMultipleRelations
	default:
		return ReasonUnknown
	}
}

// RuleOrigin is design provenance. RuleID and Revision are the accepted
// semantic identity and version; SourcePath is an informational Source Index
// locator that must never substitute for that identity.
type RuleOrigin struct {
	RuleID     string
	Revision   string
	SourcePath string
}

// Realization identifies the implementation that produced the value.
// A Provider that cannot supply the exact version sets VersionKnown=false and
// leaves Version empty; a projection must then report Unknown, not version 0.
type Realization struct {
	Identity     string
	Version      string
	VersionKnown bool
}

// InputEvidence is one input reference actually used by the evaluation.
type InputEvidence struct {
	InputID      string
	Version      string
	VersionKnown bool
	State        derivationref.InputState
	Value        string
}

// EvaluationContext is the class of context under which the value was
// evaluated. AsOf is evidence, not part of the explanation identity.
type EvaluationContext struct {
	Class string
	AsOf  time.Time
}

// Evidence is the machine-readable evaluation evidence for one value.
type Evidence struct {
	Rule        RuleOrigin
	TargetID    string
	Realization Realization
	Inputs      []InputEvidence
	Context     EvaluationContext
	Result      derivationref.Input
	Outcome     Outcome
	Reason      Reason
}

// Validate checks the shape of one evidence record. It rejects locators that
// would confuse a file path with a typed semantic identity, non-canonically
// ordered inputs, version contradictions and outcomes that contradict the
// result or the reason.
func Validate(ev Evidence) error {
	if !semanticID(ev.Rule.RuleID) || strings.TrimSpace(ev.Rule.Revision) == "" {
		return fmt.Errorf("%w: rule identity or revision", ErrInvalidEvidence)
	}
	if ev.Rule.SourcePath != "" && strings.TrimSpace(ev.Rule.SourcePath) != ev.Rule.SourcePath {
		return fmt.Errorf("%w: rule source path", ErrInvalidEvidence)
	}
	if !semanticID(ev.TargetID) {
		return fmt.Errorf("%w: target identity", ErrInvalidEvidence)
	}
	if !semanticID(ev.Realization.Identity) {
		return fmt.Errorf("%w: realization identity", ErrInvalidEvidence)
	}
	if err := checkVersion(ev.Realization.Version, ev.Realization.VersionKnown); err != nil {
		return fmt.Errorf("%w: realization version", err)
	}
	if strings.TrimSpace(ev.Context.Class) == "" {
		return fmt.Errorf("%w: evaluation context class", ErrInvalidEvidence)
	}
	if len(ev.Inputs) == 0 {
		return fmt.Errorf("%w: no input evidence", ErrInvalidEvidence)
	}
	seen := make(map[string]bool, len(ev.Inputs))
	for i, in := range ev.Inputs {
		if !semanticID(in.InputID) || seen[in.InputID] {
			return fmt.Errorf("%w: input identity", ErrInvalidEvidence)
		}
		seen[in.InputID] = true
		if i > 0 && ev.Inputs[i-1].InputID >= in.InputID {
			return fmt.Errorf("%w: inputs are not canonical ordered", ErrInvalidEvidence)
		}
		if err := checkVersion(in.Version, in.VersionKnown); err != nil {
			return fmt.Errorf("%w: input %s version", err, in.InputID)
		}
		switch in.State {
		case derivationref.Present:
			if _, ok := new(big.Rat).SetString(in.Value); !ok {
				return fmt.Errorf("%w: input %s value is not a decimal", ErrInvalidEvidence, in.InputID)
			}
		case derivationref.Null, derivationref.Missing,
			derivationref.AbsentRelation, derivationref.MultipleRelations:
			if in.Value != "" {
				return fmt.Errorf("%w: unavailable input %s carries a value", ErrInvalidEvidence, in.InputID)
			}
		default:
			return fmt.Errorf("%w: unknown input state", ErrInvalidEvidence)
		}
	}
	switch ev.Outcome {
	case OutcomeValue:
		if ev.Result.State != derivationref.Present {
			return fmt.Errorf("%w: value outcome requires a present result", ErrInvalidEvidence)
		}
		if _, ok := new(big.Rat).SetString(ev.Result.Decimal); !ok {
			return fmt.Errorf("%w: result is not a decimal", ErrInvalidEvidence)
		}
		if ev.Reason != ReasonNone {
			return fmt.Errorf("%w: value outcome carries a reason", ErrInvalidEvidence)
		}
	case OutcomeNull:
		if ev.Result.State != derivationref.Null || ev.Reason != ReasonNone {
			return fmt.Errorf("%w: null outcome requires a null result", ErrInvalidEvidence)
		}
	case OutcomeNoValue:
		if ev.Result != (derivationref.Input{}) {
			return fmt.Errorf("%w: no-value outcome carries a result", ErrInvalidEvidence)
		}
		if !knownReason(ev.Reason) {
			return fmt.Errorf("%w: no-value outcome requires a known reason", ErrInvalidEvidence)
		}
	default:
		return fmt.Errorf("%w: unknown outcome", ErrInvalidEvidence)
	}
	return nil
}

func checkVersion(version string, known bool) error {
	if known && strings.TrimSpace(version) == "" {
		return fmt.Errorf("%w: version marked known but empty", ErrInvalidEvidence)
	}
	if !known && version != "" {
		return fmt.Errorf("%w: unknown version must be empty", ErrInvalidEvidence)
	}
	return nil
}

// semanticID rejects locators that look like file paths. It does not define
// the Canonical ID grammar; it only prevents a path from standing in for a
// typed semantic identity. An identity such as "go/reference" is accepted,
// while "src/xf/XF100.java", "/abs/x" and "C:/x" are not.
func semanticID(id string) bool {
	if id == "" || id != strings.TrimSpace(id) || strings.ContainsRune(id, '\\') {
		return false
	}
	if strings.HasPrefix(id, "/") || strings.HasPrefix(id, "./") ||
		strings.HasPrefix(id, "../") || (len(id) >= 2 && id[1] == ':') {
		return false
	}
	for _, suffix := range []string{".go", ".java", ".md", ".json", ".html", ".xml"} {
		if strings.HasSuffix(strings.ToLower(id), suffix) {
			return false
		}
	}
	return true
}

func knownReason(reason Reason) bool {
	switch reason {
	case ReasonInputMissing, ReasonAbsentRelation, ReasonMultipleRelations,
		ReasonDenied, ReasonNotEvaluable, ReasonUnknown:
		return true
	default:
		return false
	}
}

// Identity is the stable explanation identity: the accepted rule, target,
// realization, input versions and evaluation context class that determine the
// meaning of the value. It excludes the rule's source-file path, the result
// value and AsOf, so moving a rule file or recomputing a changed value does
// not silently change what the explanation is about.
func (ev Evidence) Identity() (string, error) {
	if err := Validate(ev); err != nil {
		return "", err
	}
	type inputID struct {
		ID           string `json:"id"`
		Version      string `json:"version"`
		VersionKnown bool   `json:"versionKnown"`
	}
	inputs := make([]inputID, 0, len(ev.Inputs))
	for _, in := range ev.Inputs {
		inputs = append(inputs, inputID{ID: in.InputID, Version: in.Version, VersionKnown: in.VersionKnown})
	}
	payload, err := json.Marshal(struct {
		RuleID          string    `json:"ruleId"`
		RuleRevision    string    `json:"ruleRevision"`
		TargetID        string    `json:"targetId"`
		Realization     string    `json:"realization"`
		ProviderVersion string    `json:"realizationVersion"`
		ProviderKnown   bool      `json:"realizationVersionKnown"`
		Inputs          []inputID `json:"inputs"`
		ContextClass    string    `json:"contextClass"`
	}{
		RuleID:          ev.Rule.RuleID,
		RuleRevision:    ev.Rule.Revision,
		TargetID:        ev.TargetID,
		Realization:     ev.Realization.Identity,
		ProviderVersion: ev.Realization.Version,
		ProviderKnown:   ev.Realization.VersionKnown,
		Inputs:          inputs,
		ContextClass:    ev.Context.Class,
	})
	if err != nil {
		return "", fmt.Errorf("%w: identity encoding: %v", ErrInvalidEvidence, err)
	}
	digest := sha256.Sum256(payload)
	return hex.EncodeToString(digest[:]), nil
}

// Field names the addressable parts of an explanation whose disclosure is
// judged independently.
type Field string

const (
	FieldResult        Field = "result"
	FieldRule          Field = "rule"
	FieldRuleSource    Field = "rule-source"
	FieldTarget        Field = "target"
	FieldRealization   Field = "realization"
	FieldInputIdentity Field = "input-identity"
	FieldInputValue    Field = "input-value"
	FieldContext       Field = "context"
	FieldReason        Field = "reason"
	FieldConfirmed     Field = "confirmed"
)

// Fields lists every independent disclosure field.
var Fields = []Field{
	FieldResult, FieldRule, FieldRuleSource, FieldTarget, FieldRealization,
	FieldInputIdentity, FieldInputValue, FieldContext, FieldReason, FieldConfirmed,
}

// DisclosureAuthority is the Host/#1693 decision point. D7 loads no roles,
// grants or credentials. Any non-nil error fails closed for that field, and
// its text is never surfaced. Element is the rule/input/target/realization ID
// the field is about, or "" for a whole-explanation field.
type DisclosureAuthority interface {
	CanDisclose(ctx context.Context, subject string, field Field, element string) error
}

type RuleView struct {
	RuleID     string `json:"ruleId"`
	Revision   string `json:"revision"`
	SourcePath string `json:"sourcePath,omitempty"`
}

type RealizationView struct {
	Identity     string `json:"identity"`
	Version      string `json:"version,omitempty"`
	VersionKnown bool   `json:"versionKnown"`
}

type InputView struct {
	InputID      string `json:"inputId"`
	Version      string `json:"version,omitempty"`
	VersionKnown bool   `json:"versionKnown"`
	State        string `json:"state,omitempty"`
	Value        string `json:"value,omitempty"`
}

type ContextView struct {
	Class string    `json:"class"`
	AsOf  time.Time `json:"asOf,omitempty"`
}

type ConfirmedView struct {
	FactID      string    `json:"factId"`
	Value       string    `json:"value,omitempty"`
	RuleID      string    `json:"ruleId"`
	Revision    string    `json:"ruleRevision"`
	ConfirmedAt time.Time `json:"confirmedAt"`
}

// View is the machine-readable projection of one explanation. Withheld fields
// are absent rather than blanked, so the projection never asserts the
// existence of evidence the viewer may not see.
type View struct {
	Identity    string           `json:"identity"`
	Outcome     Outcome          `json:"outcome"`
	Result      string           `json:"result,omitempty"`
	Rule        *RuleView        `json:"rule,omitempty"`
	Target      string           `json:"target,omitempty"`
	Realization *RealizationView `json:"realization,omitempty"`
	Inputs      []InputView      `json:"inputs,omitempty"`
	Context     *ContextView     `json:"context,omitempty"`
	Reason      Reason           `json:"reason,omitempty"`
	Confirmed   *ConfirmedView   `json:"confirmed,omitempty"`
}

// Explanation bundles the current evaluation evidence with an optional
// confirmed business fact. The confirmed fact is read-side only; a new
// evaluation never overwrites it and it is never derived from the evidence.
type Explanation struct {
	Evidence  Evidence
	Confirmed *derivationref.ConfirmedFact
}

// Project produces the disclosure view for one explanation. A viewer who may
// not see the result gets no partial explanation at all. Withheld inputs are
// omitted, so neither their values nor their existence is disclosed.
func Project(ctx context.Context, subject string, authority DisclosureAuthority, ex Explanation) (View, error) {
	if err := Validate(ex.Evidence); err != nil {
		return View{}, err
	}
	if err := validateConfirmed(ex.Confirmed); err != nil {
		return View{}, err
	}
	if ctx == nil || authority == nil || strings.TrimSpace(subject) == "" || subject != strings.TrimSpace(subject) {
		return View{}, ErrNotDisclosable
	}
	if ctx.Err() != nil {
		return View{}, ErrDisclosureFailure
	}
	identity, err := ex.Evidence.Identity()
	if err != nil {
		return View{}, err
	}
	allowed := func(field Field, element string) bool {
		return authority.CanDisclose(ctx, subject, field, element) == nil
	}
	if !allowed(FieldResult, ex.Evidence.TargetID) {
		return View{}, ErrNotDisclosable
	}
	if ctx.Err() != nil {
		return View{}, ErrDisclosureFailure
	}

	view := View{Identity: identity, Outcome: ex.Evidence.Outcome}
	if ex.Evidence.Outcome == OutcomeValue {
		view.Result = ex.Evidence.Result.Decimal
	}
	if allowed(FieldRule, ex.Evidence.Rule.RuleID) {
		rule := &RuleView{RuleID: ex.Evidence.Rule.RuleID, Revision: ex.Evidence.Rule.Revision}
		if allowed(FieldRuleSource, ex.Evidence.Rule.RuleID) {
			rule.SourcePath = ex.Evidence.Rule.SourcePath
		}
		view.Rule = rule
	}
	if allowed(FieldTarget, ex.Evidence.TargetID) {
		view.Target = ex.Evidence.TargetID
	}
	if allowed(FieldRealization, ex.Evidence.Realization.Identity) {
		view.Realization = &RealizationView{
			Identity:     ex.Evidence.Realization.Identity,
			Version:      ex.Evidence.Realization.Version,
			VersionKnown: ex.Evidence.Realization.VersionKnown,
		}
	}
	if allowed(FieldContext, ex.Evidence.TargetID) {
		view.Context = &ContextView{Class: ex.Evidence.Context.Class, AsOf: ex.Evidence.Context.AsOf}
	}
	if ex.Evidence.Reason != ReasonNone && allowed(FieldReason, ex.Evidence.TargetID) {
		view.Reason = ex.Evidence.Reason
	}
	for _, in := range ex.Evidence.Inputs {
		if !allowed(FieldInputIdentity, in.InputID) {
			continue
		}
		input := InputView{InputID: in.InputID, Version: in.Version, VersionKnown: in.VersionKnown}
		if allowed(FieldInputValue, in.InputID) {
			input.State = string(in.State)
			input.Value = in.Value
		}
		view.Inputs = append(view.Inputs, input)
	}
	if ex.Confirmed != nil && allowed(FieldConfirmed, ex.Confirmed.TargetID) {
		view.Confirmed = &ConfirmedView{
			FactID:      ex.Confirmed.FactID,
			Value:       ex.Confirmed.Value.Decimal,
			RuleID:      ex.Confirmed.SourceRuleID,
			Revision:    ex.Confirmed.SourceRuleRevision,
			ConfirmedAt: ex.Confirmed.ConfirmedAt,
		}
	}
	if ctx.Err() != nil {
		return View{}, ErrDisclosureFailure
	}
	return view, nil
}

func validateConfirmed(fact *derivationref.ConfirmedFact) error {
	if fact == nil {
		return nil
	}
	if fact.FactID == "" || fact.TargetID == "" || fact.SourceRuleID == "" ||
		fact.SourceRuleRevision == "" || fact.ConfirmedAt.IsZero() {
		return fmt.Errorf("%w: confirmed fact identity", ErrInvalidEvidence)
	}
	switch fact.Value.State {
	case derivationref.Present:
		if _, ok := new(big.Rat).SetString(fact.Value.Decimal); !ok {
			return fmt.Errorf("%w: confirmed value is not a decimal", ErrInvalidEvidence)
		}
	case derivationref.Null:
		if fact.Value.Decimal != "" {
			return fmt.Errorf("%w: confirmed null carries a value", ErrInvalidEvidence)
		}
	default:
		return fmt.Errorf("%w: confirmed fact is not a settled value", ErrInvalidEvidence)
	}
	return nil
}
