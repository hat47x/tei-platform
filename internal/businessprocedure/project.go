package businessprocedure

import (
	"encoding/json"
	"fmt"
	"path"
	"path/filepath"
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
	CodeBranchOutcomeUnknown         = "BUSINESS_BRANCH_OUTCOME_UNKNOWN"
	CodeBranchStepMissing            = "BUSINESS_BRANCH_STEP_MISSING"
	CodeStepIDMissing                = "BUSINESS_STEP_ID_MISSING"
	CodeBranchIDMissing              = "BUSINESS_BRANCH_ID_MISSING"
	CodeStatusNotCandidate           = "BUSINESS_STATUS_NOT_CANDIDATE"
	CodeAutoExecuteOnSystemStep      = "BUSINESS_AUTO_EXECUTE_ON_SYSTEM_STEP"
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
	ID      string  `json:"id"`
	Label   string  `json:"label"`
	OnStep  string  `json:"onStep"`
	Outcome Outcome `json:"outcome"`
}

// ActorView is the business actor and, separately, the authorization concern.
// They are separate fields because they are separate meanings.
type ActorView struct {
	BusinessActorLabel string `json:"businessActorLabel"`
	BusinessActorState Status `json:"businessActorState"`
	// AuthorizationRoleCode is the authorization system's label. It is not the
	// business actor and does not authorize the business step.
	AuthorizationRoleCode string `json:"authorizationRoleCode,omitempty"`
	AuthorizationState    Status `json:"authorizationState,omitempty"`
	// AuthorizationCapability is the resolved authorization capability, or nil
	// when the clause names none or it does not resolve.
	AuthorizationCapability *ResolvedReference `json:"authorizationCapability,omitempty"`
}

// TriggerView is the business trigger. Event is business wording; Invocation is
// always false because the projection never turns a trigger into a run.
type TriggerView struct {
	State      Status `json:"state"`
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
//
// A target that does not look like a Source declaration is dropped rather than
// offered as a resolution. The projection claims a reference reaches a Source,
// and only a target carrying a role, a positive position and an artifact digest
// can support that claim. Dropping it makes the reference unresolved, which is
// the honest result; the alternative is a "resolved" reference that names no
// Source.
func NewResolver(targets []Declared) *Resolver {
	r := &Resolver{declared: map[ReferenceKind]map[string][]Declared{}}
	for _, target := range targets {
		if !isSourceDeclaration(target) {
			continue
		}
		byID, ok := r.declared[target.Kind]
		if !ok {
			byID = map[string][]Declared{}
			r.declared[target.Kind] = byID
		}
		byID[target.ID] = append(byID[target.ID], target)
	}
	return r
}

// isSourceDeclaration reports whether a target can be shown to name a Source
// declaration: it must carry the declaring artifact, a positive position in that
// artifact, and the digest of the bytes read there.
func isSourceDeclaration(target Declared) bool {
	if strings.TrimSpace(target.ID) == "" || strings.TrimSpace(target.Artifact) == "" {
		return false
	}
	if target.Line < 1 || target.Column < 1 {
		return false
	}
	if strings.TrimSpace(target.Fingerprint) == "" {
		return false
	}
	if target.Role != "definition" {
		return false
	}
	// A declaration lives inside the Workspace, so a path that leaves it is not
	// a Source this projection can name.
	cleaned := path.Clean(filepath.ToSlash(target.Artifact))
	if cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, "../") ||
		path.IsAbs(cleaned) || filepath.IsAbs(target.Artifact) ||
		strings.Contains(cleaned, ":") {
		return false
	}
	return true
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

// Project builds the read-only projection from a fixture and a resolver. The
// resolver must not be nil; a caller with no declared targets passes
// NewResolver(nil), which resolves nothing rather than crashing.
// realizationNames lists the accepted realization words in report order, so a
// refusal can quote the same list Valid() checks.
func realizationNames() string {
	names := make([]string, 0, len(realizations))
	for _, accepted := range realizations {
		names = append(names, string(accepted))
	}
	return strings.Join(names, ", ")
}

func Project(fixture Fixture, procedureFingerprint string, resolver *Resolver) Projection {
	if resolver == nil {
		resolver = NewResolver(nil)
	}
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
	// A reference fixture proposes meaning; it cannot declare that its own
	// candidate values are accepted. Echoing such a status would let a fixture
	// assert acceptance the projection never received.
	//
	// The clauses are a fixed slice, not a map: three findings that share a
	// code, a severity and a subject are ordered only by their message, so a
	// map's iteration order would leak into the JSON.
	for _, clause := range []struct {
		name   string
		status Status
	}{
		{"trigger", fixture.Procedure.Trigger.Status},
		{"actor", fixture.Procedure.Actor.Status},
		{"authorization", fixture.Procedure.Authorization.Status},
	} {
		// An absent status is not a claim, but it is also not a value the
		// projection can echo as one. Both an unknown and an absent status are
		// reported, so the same defect does not depend on whether the author
		// wrote a wrong word or no word.
		if clause.status.Valid() {
			continue
		}
		if clause.status == "" {
			add(CodeStatusNotCandidate, SeverityProcedure, projection.ProcedureID,
				fmt.Sprintf("%s declares no status; a reference fixture must state that its value is a candidate", clause.name))
			continue
		}
		add(CodeStatusNotCandidate, SeverityProcedure, projection.ProcedureID,
			fmt.Sprintf("%s status %q is not an accepted reference status; a fixture may only propose", clause.name, clause.status))
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
			// Name the accepted words. "unknown realization" alone leaves an
			// author who guessed "application-use" to guess again; the three
			// values are a closed set this package already owns.
			add(CodeRealizationUnknown, SeverityProcedure, id,
				fmt.Sprintf("step %q declares unknown realization %q; valid values are %s",
					id, step.Realization, realizationNames()))
		}
		// A step with no id cannot be referred to by a branch, is not
		// distinguishable in manualSteps, and cannot be addressed in a finding.
		// It is reported rather than admitted as a nameless entry.
		if id == "" {
			add(CodeStepIDMissing, SeverityProcedure, fmt.Sprintf("position-%d", i+1),
				fmt.Sprintf("step at position %d declares no id", i+1))
		}
		// A claim to run without a person is refused wherever it appears. A
		// manual step gets its own finding below, so this one covers the
		// realizations a reader would otherwise expect to run by themselves.
		if step.AutoExecute && step.Realization != RealizationManual {
			add(CodeAutoExecuteOnSystemStep, SeverityProcedure, id,
				fmt.Sprintf("step %q declares autoExecute; the projection never auto-executes a business step", id))
		}

		// A step is treated as manual work unless it declares a valid
		// non-manual realization. An unknown realization cannot be shown to be
		// application or system work, so the projection must not hand it a
		// Function: the harm of showing application work on a phone call does not
		// depend on the author spelling "manual" exactly. The unknown value is
		// reported separately, so this stays a visible decision.
		manual := step.Realization == RealizationManual || !step.Realization.Valid()
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
		// A step whose actor is the authorization role code makes the two
		// meanings indistinguishable at the point a reader sees them, even when
		// the procedure-level labels differ.
		if roleCode != "" && strings.EqualFold(view.ActorLabel, roleCode) {
			add(CodeActorConflatesAuthorization, SeverityProcedure, id,
				fmt.Sprintf("step %q takes %q as its business actor, which is the authorization role code", id, view.ActorLabel))
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
			// A manual step is not application work, so the projection does not
			// hold out its Function as one. The reference is kept as unresolved
			// instead of being shown as the step's action: a reader who looks at
			// steps[].function without reading the diagnostics must not see a
			// Function on the phone call.
			if manual {
				unresolved := *step.Function
				view.UnresolvedFunction = &unresolved
			} else if resolution.Resolved != nil {
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
	// branchesOnStep counts the declarations of each outcome, so a step that
	// declares one approval and no rejection is distinguishable from one that
	// declares nothing meaningful.
	branchesOnStep := map[string]map[Outcome]int{}
	for i, branch := range fixture.Branches {
		id := strings.TrimSpace(branch.ID)
		if id != "" {
			if first, exists := branchIDs[id]; exists {
				add(CodeBranchDuplicateID, SeverityProcedure, id,
					fmt.Sprintf("branch id %q is duplicated at positions %d and %d", id, first, i+1))
			} else {
				branchIDs[id] = i + 1
			}
		} else {
			// A branch with no identity cannot be told from another anonymous
			// branch, and the duplicate check cannot see either of them.
			add(CodeBranchIDMissing, SeverityProcedure, fmt.Sprintf("position-%d", i+1),
				fmt.Sprintf("branch at position %d declares no id", i+1))
		}
		// An outcome outside the vocabulary is not an approval or a rejection,
		// so it must not be read as either.
		if !branch.Outcome.Valid() {
			add(CodeBranchOutcomeUnknown, SeverityProcedure, id,
				fmt.Sprintf("branch %q declares unknown outcome %q", id, branch.Outcome))
			projection.Branches = append(projection.Branches, BranchView{
				ID:      id,
				Label:   branch.Label,
				OnStep:  strings.TrimSpace(branch.OnStep),
				Outcome: branch.Outcome,
			})
			continue
		}
		onStep := strings.TrimSpace(branch.OnStep)
		// A branch bound to no step makes the one-sided-decision check vacuous:
		// the decision it belongs to is unknown, so it cannot be shown to have
		// two sides. Report it instead of silently skipping the check.
		if onStep == "" {
			add(CodeBranchStepMissing, SeverityProcedure, id,
				fmt.Sprintf("branch %q names no step, so its decision cannot be checked for two sides", id))
			projection.Branches = append(projection.Branches, BranchView{
				ID:      id,
				Label:   branch.Label,
				OnStep:  "",
				Outcome: branch.Outcome,
			})
			continue
		}
		if _, exists := stepIDs[onStep]; !exists {
			add(CodeBranchStepUnknown, SeverityProcedure, id,
				fmt.Sprintf("branch %q is on step %q, which the procedure does not declare", id, onStep))
		}
		if branchesOnStep[onStep] == nil {
			branchesOnStep[onStep] = map[Outcome]int{}
		}
		branchesOnStep[onStep][branch.Outcome]++
		projection.Branches = append(projection.Branches, BranchView{
			ID:      id,
			Label:   branch.Label,
			OnStep:  onStep,
			Outcome: branch.Outcome,
		})
	}
	// A decision that opens only one side is not a decision. The check names the
	// step rather than inventing the missing branch.
	for stepID, counts := range branchesOnStep {
		switch {
		case counts[OutcomeApproval] > 0 && counts[OutcomeRejection] == 0:
			add(CodeBranchMissingRejection, SeverityProcedure, stepID,
				fmt.Sprintf("decision step %q opens an approval branch but no rejection branch", stepID))
		case counts[OutcomeRejection] > 0 && counts[OutcomeApproval] == 0:
			add(CodeBranchMissingRejection, SeverityProcedure, stepID,
				fmt.Sprintf("decision step %q opens a rejection branch but no approval branch", stepID))
		}
	}

	// The comparison is a total order, ending in the message. Two findings can
	// share a severity, a code and a subject and still differ in wording; without
	// the last term their relative order would follow whichever order the rules
	// happened to be evaluated in, and the JSON would differ between runs of the
	// same input.
	sort.SliceStable(diagnostics, func(i, j int) bool {
		if diagnostics[i].Severity != diagnostics[j].Severity {
			return severityRank(diagnostics[i].Severity) < severityRank(diagnostics[j].Severity)
		}
		if diagnostics[i].Code != diagnostics[j].Code {
			return diagnostics[i].Code < diagnostics[j].Code
		}
		left, right := strings.Join(diagnostics[i].SubjectIDs, ","), strings.Join(diagnostics[j].SubjectIDs, ",")
		if left != right {
			return left < right
		}
		return diagnostics[i].Message < diagnostics[j].Message
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
