package businessprocedure

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// Diagnostic is one finding about the projected procedure.
type Diagnostic struct {
	Code       string   `json:"code"`
	Severity   string   `json:"severity"`
	Message    string   `json:"message"`
	SubjectIDs []string `json:"subjectIds,omitempty"`
}

// Severity values. Both block a clean projection; the difference is why.
const (
	// SeverityProcedure marks a finding about the procedure's own meaning: a
	// manual step was erased, an event was treated as an automatic run, or two
	// branches were collapsed.
	SeverityProcedure = "procedure-error"
	// SeverityReference marks a finding about a reference that does not resolve
	// to exactly one declared target.
	SeverityReference = "reference-error"
)

// Diagnostic codes. Each is emitted by at least one rule below and pinned by a
// test that reaches it.
const (
	CodeProcedureIDMissing           = "BUSINESS_PROCEDURE_ID_MISSING"
	CodeStepDuplicateID              = "BUSINESS_PROCEDURE_STEP_DUPLICATE"
	CodeStepActorUnattributed        = "BUSINESS_STEP_ACTOR_UNATTRIBUTED"
	CodeManualStepHasFunction        = "BUSINESS_MANUAL_STEP_HAS_FUNCTION"
	CodeManualStepAutoExecute        = "BUSINESS_MANUAL_STEP_AUTO_EXECUTE"
	CodeRealizationUnknown           = "BUSINESS_STEP_REALIZATION_UNKNOWN"
	CodeBranchStepUnknown            = "BUSINESS_BRANCH_STEP_UNKNOWN"
	CodeBranchDuplicateID            = "BUSINESS_BRANCH_DUPLICATE"
	CodeBranchMissingRejection       = "BUSINESS_DECISION_MISSING_REJECTION"
	CodeActorConflatesAuthorization  = "BUSINESS_ACTOR_CONFLATES_AUTHORIZATION"
	CodeReferenceUnresolved          = "BUSINESS_REFERENCE_UNRESOLVED"
	CodeReferenceAmbiguous           = "BUSINESS_REFERENCE_AMBIGUOUS"
	CodeFixtureNotAuthoringReference = "BUSINESS_FIXTURE_NOT_AUTHORING_REFERENCE"
)

// ActorLimits are the fixed authority limits of the projection. They are not a
// per-caller decision.
func ActorLimits() []string {
	return []string{
		"business-actor-is-not-an-authorization-role",
		"business-event-is-not-a-function-launch",
		"manual-step-is-not-auto-executed",
		"procedure-reference-does-not-grant-execution",
	}
}

// ProjectionLimits are the fixed limits of the whole projection.
func ProjectionLimits() []string {
	return []string{
		"projection-is-a-derived-read-only-projection",
		"reference-fixture-is-not-accepted-business-meaning",
		"unresolved-reference-is-reported-not-inferred",
		"approval-and-rejection-are-separate-branches",
	}
}

// StepView is one step after reference resolution.
type StepView struct {
	ID          string      `json:"id"`
	Action      string      `json:"action"`
	Realization Realization `json:"realization"`
	// ActorLabel is the business actor of this step: its own label, or the
	// procedure actor's. It never comes from the authorization clause.
	ActorLabel string `json:"actorLabel,omitempty"`
	Manual     bool   `json:"manual"`
	// ManualInstruction survives into the projection or is explicitly absent.
	// A manual step is never dropped for lacking one.
	ManualInstruction string `json:"manualInstruction,omitempty"`
	// AutoExecuted is always false. It is present so a reader can see the
	// projection decided not to auto-execute, rather than not mentioning it.
	AutoExecuted bool `json:"autoExecuted"`
	// NoFunction is true for a step that declares no Function. A manual step is
	// expected to have no Function; an application or system step with no
	// Function is shown the same way rather than being filled in.
	NoFunction bool `json:"noFunction"`
	// Function is the resolved Function reference, if the step has one and it
	// resolves to exactly one target.
	Function *ResolvedReference `json:"function,omitempty"`
	// UnresolvedFunction is the raw reference the projection could not resolve.
	// It is kept and named, not guessed at.
	UnresolvedFunction *Reference `json:"unresolvedFunction,omitempty"`
}

// ResolvedReference is a reference that resolved to exactly one declared
// target, with its Source location.
type ResolvedReference struct {
	Kind        ReferenceKind `json:"kind"`
	ID          string        `json:"id"`
	Subject     string        `json:"subject"`
	Artifact    string        `json:"artifact"`
	Line        int           `json:"line"`
	Column      int           `json:"column"`
	Role        string        `json:"role"`
	Fingerprint string        `json:"fingerprint,omitempty"`
}

// BranchView is one branch after projection.
type BranchView struct {
	ID      string `json:"id"`
	Label   string `json:"label"`
	OnStep  string `json:"onStep"`
	Outcome string `json:"outcome"`
}

// ActorView is the business actor and, separately, the authorization concern.
// They are separate fields because they are separate meanings.
type ActorView struct {
	BusinessActorLabel string `json:"businessActorLabel"`
	BusinessActorState string `json:"businessActorState"`
	// AuthorizationRoleCode is the authorization system's label. It is not the
	// business actor and does not authorize the business step.
	AuthorizationRoleCode string `json:"authorizationRoleCode,omitempty"`
	AuthorizationState    string `json:"authorizationState,omitempty"`
	// AuthorizationCapability is the resolved authorization capability, or nil
	// when the clause names none or it does not resolve.
	AuthorizationCapability *ResolvedReference `json:"authorizationCapability,omitempty"`
}

// TriggerView is the business trigger. Event is business wording; Invocation is
// always false because the projection never turns a trigger into a run.
type TriggerView struct {
	State      string `json:"state"`
	Text       string `json:"text"`
	Invocation bool   `json:"invocation"`
}

// Projection is the read-only result.
type Projection struct {
	SchemaVersion        string       `json:"schemaVersion"`
	Effect               string       `json:"effect"`
	ProcedureID          string       `json:"procedureId"`
	Name                 string       `json:"name,omitempty"`
	Purpose              string       `json:"purpose,omitempty"`
	Application          string       `json:"application,omitempty"`
	ProcedureFingerprint string       `json:"procedureFingerprint"`
	Trigger              TriggerView  `json:"trigger"`
	Actor                ActorView    `json:"actor"`
	Steps                []StepView   `json:"steps"`
	Branches             []BranchView `json:"branches"`
	// ManualSteps names every manual step. A projection that lost a manual step
	// would have a shorter list than the source.
	ManualSteps []string `json:"manualSteps"`
	// ManualStepsWithoutFunction names the manual steps that declare no
	// Function. This is the expected case and is listed so its absence in a
	// particular procedure is visible.
	ManualStepsWithoutFunction []string     `json:"manualStepsWithoutFunction"`
	Diagnostics                []Diagnostic `json:"diagnostics"`
	AuthorityLimits            []string     `json:"authorityLimits"`
	ActorLimits                []string     `json:"actorLimits"`
}

// Declared is one declared target the projection can resolve a reference to.
// ID must be unique within Kind for the reference to resolve.
type Declared struct {
	Kind        ReferenceKind
	ID          string
	Subject     string
	Artifact    string
	Line        int
	Column      int
	Role        string
	Fingerprint string
}

// Resolver resolves a Reference against the declared targets of the World.
// A caller supplies the targets; this package does not read the World itself,
// so the projection cannot become a second model reader.
type Resolver struct {
	declared map[ReferenceKind]map[string][]Declared
}

// NewResolver groups declared targets by kind and ID. Duplicate IDs are kept,
// not collapsed: a doubled ID is ambiguous and must not resolve to the first.
func NewResolver(targets []Declared) *Resolver {
	r := &Resolver{declared: map[ReferenceKind]map[string][]Declared{}}
	for _, target := range targets {
		byID, ok := r.declared[target.Kind]
		if !ok {
			byID = map[string][]Declared{}
			r.declared[target.Kind] = byID
		}
		byID[target.ID] = append(byID[target.ID], target)
	}
	return r
}

// Resolution is the outcome of resolving one reference.
type Resolution struct {
	Resolved  *ResolvedReference
	Ambiguous bool
	// Reason explains an unresolved or ambiguous result in words.
	Reason string
}

// Resolve resolves one reference. It resolves only an exact identity match, and
// only when exactly one declared target carries it.
func (r *Resolver) Resolve(ref Reference) Resolution {
	if strings.TrimSpace(ref.ID) == "" {
		return Resolution{Reason: "reference has no id"}
	}
	if _, ok := ref.Kind.subjectKind(); !ok {
		return Resolution{Reason: fmt.Sprintf("reference kind %q is not a resolvable kind", ref.Kind)}
	}
	byID, ok := r.declared[ref.Kind]
	if !ok {
		return Resolution{Reason: fmt.Sprintf("no %s is declared", ref.Kind)}
	}
	matches := byID[ref.ID]
	switch len(matches) {
	case 0:
		return Resolution{Reason: fmt.Sprintf("%s %q is not declared", ref.Kind, ref.ID)}
	case 1:
		match := matches[0]
		return Resolution{Resolved: &ResolvedReference{
			Kind:        match.Kind,
			ID:          match.ID,
			Subject:     match.Subject,
			Artifact:    match.Artifact,
			Line:        match.Line,
			Column:      match.Column,
			Role:        match.Role,
			Fingerprint: match.Fingerprint,
		}}
	default:
		// A name is never used to choose between two declarations.
		return Resolution{
			Ambiguous: true,
			Reason:    fmt.Sprintf("%s %q is declared %d times", ref.Kind, ref.ID, len(matches)),
		}
	}
}

// Project builds the read-only projection from a fixture and a resolver.
func Project(fixture Fixture, procedureFingerprint string, resolver *Resolver) Projection {
	projection := Projection{
		SchemaVersion:        ProjectionSchemaVersion,
		Effect:               EffectReadOnly,
		ProcedureID:          strings.TrimSpace(fixture.Procedure.ID),
		Name:                 fixture.Procedure.Name,
		Purpose:              fixture.Procedure.Purpose,
		Application:          fixture.Application,
		ProcedureFingerprint: procedureFingerprint,
		ActorLimits:          ActorLimits(),
		AuthorityLimits:      ProjectionLimits(),
		Trigger: TriggerView{
			State: fixture.Procedure.Trigger.Status,
			Text:  fixture.Procedure.Trigger.Text,
			// A business trigger never becomes a Function invocation.
			Invocation: false,
		},
		Actor: ActorView{
			BusinessActorLabel:    fixture.Procedure.Actor.Label,
			BusinessActorState:    fixture.Procedure.Actor.Status,
			AuthorizationRoleCode: fixture.Procedure.Authorization.RoleCode,
			AuthorizationState:    fixture.Procedure.Authorization.Status,
		},
		Steps:                      make([]StepView, 0, len(fixture.Steps)),
		Branches:                   make([]BranchView, 0, len(fixture.Branches)),
		ManualSteps:                []string{},
		ManualStepsWithoutFunction: []string{},
		Diagnostics:                []Diagnostic{},
	}

	var diagnostics []Diagnostic
	add := func(code, severity, subject, message string) {
		diagnostics = append(diagnostics, Diagnostic{
			Code:       code,
			Severity:   severity,
			Message:    message,
			SubjectIDs: []string{subject},
		})
	}

	if projection.ProcedureID == "" {
		add(CodeProcedureIDMissing, SeverityProcedure, "",
			"the procedure declares no id")
	}
	if fixture.Canonical {
		add(CodeFixtureNotAuthoringReference, SeverityProcedure, projection.ProcedureID,
			"a reference fixture must not claim to be Canonical business meaning")
	}

	// A business actor label that repeats the authorization role code makes the
	// two meanings indistinguishable at the point a reader reads them.
	actorLabel := strings.TrimSpace(fixture.Procedure.Actor.Label)
	roleCode := strings.TrimSpace(fixture.Procedure.Authorization.RoleCode)
	if actorLabel != "" && roleCode != "" && strings.EqualFold(actorLabel, roleCode) {
		add(CodeActorConflatesAuthorization, SeverityProcedure, projection.ProcedureID,
			fmt.Sprintf("business actor %q repeats the authorization role code; a business actor is not an authorization role", actorLabel))
	}

	// Resolve the authorization capability independently of the business actor.
	if ref := strings.TrimSpace(fixture.Procedure.Authorization.Capability); ref != "" {
		resolution := resolver.Resolve(Reference{Kind: ReferenceCapability, ID: ref})
		recordReferenceDiagnostic(add, projection.ProcedureID, "authorization capability", Reference{Kind: ReferenceCapability, ID: ref}, resolution)
		projection.Actor.AuthorizationCapability = resolution.Resolved
	}

	stepIDs := map[string]int{}
	for i, step := range fixture.Steps {
		id := strings.TrimSpace(step.ID)
		if id != "" {
			if first, exists := stepIDs[id]; exists {
				add(CodeStepDuplicateID, SeverityProcedure, id,
					fmt.Sprintf("step id %q is duplicated at positions %d and %d", id, first, i+1))
			} else {
				stepIDs[id] = i + 1
			}
		}
		if !step.Realization.Valid() {
			add(CodeRealizationUnknown, SeverityProcedure, id,
				fmt.Sprintf("step %q declares unknown realization %q", id, step.Realization))
		}

		manual := step.Realization == RealizationManual
		view := StepView{
			ID:                id,
			Action:            step.Action,
			Realization:       step.Realization,
			Manual:            manual,
			ManualInstruction: step.ManualInstruction,
			// The projection never auto-executes a step, and it never promotes a
			// business trigger or a manual step to an automatic run.
			AutoExecuted: false,
			NoFunction:   step.Function == nil,
		}
		view.ActorLabel = strings.TrimSpace(step.ActorLabel)
		if view.ActorLabel == "" {
			view.ActorLabel = actorLabel
		}
		if view.ActorLabel == "" {
			add(CodeStepActorUnattributed, SeverityProcedure, id,
				fmt.Sprintf("step %q has no business actor and the procedure declares none", id))
		}

		// A manual step that claims to run automatically is the exact confusion
		// this projection exists to prevent.
		if manual && step.AutoExecute {
			add(CodeManualStepAutoExecute, SeverityProcedure, id,
				fmt.Sprintf("manual step %q declares autoExecute; a business event does not launch a Function", id))
		}
		// A manual step with a Function reference would make the handwork look
		// like application work.
		if manual && step.Function != nil {
			add(CodeManualStepHasFunction, SeverityProcedure, id,
				fmt.Sprintf("manual step %q references a Function; a manual step is not an application action", id))
		}

		if step.Function != nil {
			resolution := resolver.Resolve(*step.Function)
			recordReferenceDiagnostic(add, id, "function", *step.Function, resolution)
			if resolution.Resolved != nil {
				view.Function = resolution.Resolved
			} else {
				// Keep the raw reference. Dropping it would make the projection
				// look as if the step never named anything.
				unresolved := *step.Function
				view.UnresolvedFunction = &unresolved
			}
		}

		if manual {
			projection.ManualSteps = append(projection.ManualSteps, id)
			if step.Function == nil {
				projection.ManualStepsWithoutFunction = append(projection.ManualStepsWithoutFunction, id)
			}
		}
		projection.Steps = append(projection.Steps, view)
	}

	branchIDs := map[string]int{}
	branchesOnStep := map[string]map[string]bool{}
	for i, branch := range fixture.Branches {
		id := strings.TrimSpace(branch.ID)
		if id != "" {
			if first, exists := branchIDs[id]; exists {
				add(CodeBranchDuplicateID, SeverityProcedure, id,
					fmt.Sprintf("branch id %q is duplicated at positions %d and %d", id, first, i+1))
			} else {
				branchIDs[id] = i + 1
			}
		}
		onStep := strings.TrimSpace(branch.OnStep)
		if onStep != "" {
			if _, exists := stepIDs[onStep]; !exists {
				add(CodeBranchStepUnknown, SeverityProcedure, id,
					fmt.Sprintf("branch %q is on step %q, which the procedure does not declare", id, onStep))
			}
			if branchesOnStep[onStep] == nil {
				branchesOnStep[onStep] = map[string]bool{}
			}
			branchesOnStep[onStep][strings.TrimSpace(branch.Outcome)] = true
		}
		projection.Branches = append(projection.Branches, BranchView{
			ID:      id,
			Label:   branch.Label,
			OnStep:  onStep,
			Outcome: branch.Outcome,
		})
	}
	// A decision that opens only one branch is not a decision. The check names
	// the step rather than inventing the missing branch.
	for stepID, outcomes := range branchesOnStep {
		if outcomes["approval"] && !outcomes["rejection"] {
			add(CodeBranchMissingRejection, SeverityProcedure, stepID,
				fmt.Sprintf("decision step %q opens an approval branch but no rejection branch", stepID))
		}
		if outcomes["rejection"] && !outcomes["approval"] {
			add(CodeBranchMissingRejection, SeverityProcedure, stepID,
				fmt.Sprintf("decision step %q opens a rejection branch but no approval branch", stepID))
		}
	}

	sort.SliceStable(diagnostics, func(i, j int) bool {
		if diagnostics[i].Severity != diagnostics[j].Severity {
			return severityRank(diagnostics[i].Severity) < severityRank(diagnostics[j].Severity)
		}
		if diagnostics[i].Code != diagnostics[j].Code {
			return diagnostics[i].Code < diagnostics[j].Code
		}
		return strings.Join(diagnostics[i].SubjectIDs, ",") < strings.Join(diagnostics[j].SubjectIDs, ",")
	})
	// Keep the field a list rather than a null: a reader that branches on
	// diagnostics must see "no finding", not "field absent".
	if diagnostics == nil {
		diagnostics = []Diagnostic{}
	}
	projection.Diagnostics = diagnostics
	return projection
}

// recordReferenceDiagnostic reports a reference that did not resolve to exactly
// one declared target. It distinguishes absent from ambiguous, because a caller
// acts differently on each.
func recordReferenceDiagnostic(add func(code, severity, subject, message string), stepID, role string, ref Reference, resolution Resolution) {
	if resolution.Resolved != nil {
		return
	}
	subject := ref.ID
	if strings.TrimSpace(subject) == "" {
		subject = stepID
	}
	if resolution.Ambiguous {
		add(CodeReferenceAmbiguous, SeverityReference, subject,
			fmt.Sprintf("%s reference %s %q is ambiguous: %s", role, ref.Kind, ref.ID, resolution.Reason))
		return
	}
	add(CodeReferenceUnresolved, SeverityReference, subject,
		fmt.Sprintf("%s reference %s %q does not resolve: %s", role, ref.Kind, ref.ID, resolution.Reason))
}

func severityRank(severity string) int {
	switch severity {
	case SeverityProcedure:
		return 0
	case SeverityReference:
		return 1
	}
	return 2
}

// Fingerprint returns a stable digest of the fixture content. Two documents
// with the same meaning have the same digest; the digest is not a Source
// revision and does not authorize a write.
func Fingerprint(fixture Fixture) string {
	normalized := fixture
	// The digest describes the authored document, not the claim to be Canonical.
	normalized.Canonical = false
	b, _ := json.Marshal(normalized)
	return fingerprintOf(b)
}
