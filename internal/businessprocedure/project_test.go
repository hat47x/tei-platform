package businessprocedure

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fixturePath is the public synthetic example the issue asks to fix: a
// one-level business procedure from order intake through phone confirmation,
// approval or rejection, to shipping preparation.
func fixturePath(t *testing.T) string {
	t.Helper()
	return filepath.Join("..", "..", "reference", "business-system-design",
		"order-acceptance-business-procedure.json")
}

func readFixture(t *testing.T) Fixture {
	t.Helper()
	raw, err := os.ReadFile(fixturePath(t))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	var fixture Fixture
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatalf("decode fixture: %v", err)
	}
	return fixture
}

// declaredTargets are the declared identities the reference workspace carries,
// with the exact Source position of each declaration. The values are read from
// the Workspace's own Source Index, not derived from this package, and
// TestDeclaredTargetsMatchTheRealIndex fails if they drift from it.
//
// The unit tests use this table so they do not have to load a Workspace; the
// conformance test is what keeps the table honest.
func declaredTargets() []Declared {
	return []Declared{
		{Kind: ReferencePattern, ID: "order_entry.recent_orders",
			Subject:  "application-pattern:order_entry.recent_orders",
			Artifact: "applications/order-entry/patterns/recent-orders.tei",
			Line:     1, Column: 9, Role: "definition"},
		{Kind: ReferencePattern, ID: "order_entry.order_record",
			Subject:  "application-pattern:order_entry.order_record",
			Artifact: "applications/order-entry/patterns/order-record.tei",
			Line:     1, Column: 9, Role: "definition"},
		{Kind: ReferencePattern, ID: "order_entry.customer_reference",
			Subject:  "application-pattern:order_entry.customer_reference",
			Artifact: "applications/order-entry/patterns/customer-reference.tei",
			Line:     1, Column: 9, Role: "definition"},
		{Kind: ReferencePattern, ID: "order_entry.order_lines",
			Subject:  "application-pattern:order_entry.order_lines",
			Artifact: "applications/order-entry/patterns/order-lines.tei",
			Line:     1, Column: 9, Role: "definition"},
		{Kind: ReferencePattern, ID: "order_entry.submit_order",
			Subject:  "application-pattern:order_entry.submit_order",
			Artifact: "applications/order-entry/patterns/submit-order.tei",
			Line:     1, Column: 9, Role: "definition"},
		{Kind: ReferenceCapability, ID: "sales.order.submit",
			Subject:  "semantic-capability:sales.order.submit",
			Artifact: "semantic/sales/order.tei",
			Line:     27, Column: 12, Role: "definition"},
		{Kind: ReferenceCapability, ID: "security.authorize",
			Subject:  "semantic-capability:security.authorize",
			Artifact: "semantic/security/authorization.tei",
			Line:     17, Column: 12, Role: "definition"},
		{Kind: ReferenceValidation, ID: "sales.order_line.quantity_positive",
			Subject:  "semantic-validation:sales.order_line.quantity_positive",
			Artifact: "semantic/sales/validation.tei",
			Line:     3, Column: 12, Role: "definition"},
		{Kind: ReferenceValidation, ID: "sales.order.lines.line_number_unique",
			Subject:  "semantic-validation:sales.order.lines.line_number_unique",
			Artifact: "semantic/sales/validation.tei",
			Line:     7, Column: 12, Role: "definition"},
	}
}

// declaredTargetsWithDigests is declaredTargets with a per-identity digest. The
// Source Index supplies these digests in the real path; the table carries them
// here because a target without a digest is not a Source declaration the
// projection accepts.
func declaredTargetsWithDigests() []Declared {
	targets := declaredTargets()
	for i := range targets {
		targets[i].Fingerprint = "sha256:" + string(targets[i].ID)
	}
	return targets
}

func projectFixture(t *testing.T, fixture Fixture) Projection {
	t.Helper()
	return Project(fixture, Fingerprint(fixture), NewResolver(declaredTargetsWithDigests()))
}

func codesOf(projection Projection) map[string]int {
	out := map[string]int{}
	for _, diagnostic := range projection.Diagnostics {
		out[diagnostic.Code]++
	}
	return out
}

func findStep(t *testing.T, projection Projection, id string) StepView {
	t.Helper()
	for _, step := range projection.Steps {
		if step.ID == id {
			return step
		}
	}
	t.Fatalf("step %q is not in the projection", id)
	return StepView{}
}

// The example the issue asks to fix must project without a finding, so every
// negative test below is a change from a known-good baseline.
func TestSyntheticOrderProcedureProjectsCleanly(t *testing.T) {
	projection := projectFixture(t, readFixture(t))
	if len(projection.Diagnostics) != 0 {
		t.Fatalf("the public example produced %d diagnostics: %+v",
			len(projection.Diagnostics), projection.Diagnostics)
	}
	if projection.Effect != EffectReadOnly {
		t.Fatalf("effect=%q, want %q", projection.Effect, EffectReadOnly)
	}
	if projection.SchemaVersion != ProjectionSchemaVersion {
		t.Fatalf("schemaVersion=%q, want %q", projection.SchemaVersion, ProjectionSchemaVersion)
	}
}

// #1125: the phone confirmation must survive as a step. A projection that keeps
// only what a Function can explain loses exactly the work this model exists for.
func TestPhoneConfirmationManualStepSurvivesWithNoFunction(t *testing.T) {
	projection := projectFixture(t, readFixture(t))

	phone := findStep(t, projection, "BP-A02")
	if !phone.Manual {
		t.Fatal("the phone confirmation step is not marked manual")
	}
	if !phone.NoFunction || phone.Function != nil {
		t.Fatalf("the phone confirmation step was attributed a Function: %+v", phone)
	}
	if phone.ManualInstruction == "" {
		t.Fatal("the phone confirmation lost its instruction; a manual step must not be silently emptied")
	}
	if !containsString(projection.ManualSteps, "BP-A02") {
		t.Fatalf("manualSteps does not list BP-A02: %v", projection.ManualSteps)
	}
	if !containsString(projection.ManualStepsWithoutFunction, "BP-A02") {
		t.Fatalf("manualStepsWithoutFunction does not list BP-A02: %v", projection.ManualStepsWithoutFunction)
	}

	// The baseline has three manual steps; none may be dropped.
	if len(projection.ManualSteps) != 3 {
		t.Fatalf("manualSteps=%v, want three manual steps", projection.ManualSteps)
	}
	if len(projection.Steps) != 7 {
		t.Fatalf("projection has %d steps, want 7; a step was dropped", len(projection.Steps))
	}
}

// #1125: a business actor is not an authorization role. The two must be
// readable as two different values, and neither may be derived from the other.
func TestBusinessActorAndAuthorizationRoleStayDistinct(t *testing.T) {
	projection := projectFixture(t, readFixture(t))

	if projection.Actor.BusinessActorLabel != "受注担当" {
		t.Fatalf("business actor=%q, want 受注担当", projection.Actor.BusinessActorLabel)
	}
	if projection.Actor.AuthorizationRoleCode != "ORDER_EDITOR" {
		t.Fatalf("authorization role code=%q, want ORDER_EDITOR", projection.Actor.AuthorizationRoleCode)
	}
	if projection.Actor.BusinessActorLabel == projection.Actor.AuthorizationRoleCode {
		t.Fatal("business actor and authorization role code were conflated")
	}
	for _, step := range projection.Steps {
		if step.ActorLabel == "" {
			t.Fatalf("step %s lost its business actor", step.ID)
		}
		if step.ActorLabel == projection.Actor.AuthorizationRoleCode {
			t.Fatalf("step %s took its actor from the authorization role code", step.ID)
		}
	}
	if projection.Actor.AuthorizationCapability == nil {
		t.Fatal("the declared authorization capability did not resolve")
	}
	if projection.Actor.AuthorizationCapability.ID != "security.authorize" {
		t.Fatalf("authorization capability=%q, want security.authorize",
			projection.Actor.AuthorizationCapability.ID)
	}
	if !containsString(projection.ActorLimits, "business-actor-is-not-an-authorization-role") {
		t.Fatalf("actorLimits=%v, want business-actor-is-not-an-authorization-role", projection.ActorLimits)
	}
}

// #1125: an event is not an automatic run. Neither the trigger nor a manual step
// may be marked executed, and a fixture that claims otherwise is refused.
func TestTriggerAndManualStepsAreNotAutoExecuted(t *testing.T) {
	projection := projectFixture(t, readFixture(t))

	if projection.Trigger.Invocation {
		t.Fatal("the business trigger was turned into a Function invocation")
	}
	for _, step := range projection.Steps {
		if step.AutoExecuted {
			t.Fatalf("step %s was projected as auto-executed", step.ID)
		}
	}

	// Inject the confusion the projection must refuse.
	fixture := readFixture(t)
	for i := range fixture.Steps {
		if fixture.Steps[i].ID == "BP-A02" {
			fixture.Steps[i].AutoExecute = true
		}
	}
	injected := projectFixture(t, fixture)
	if codesOf(injected)[CodeManualStepAutoExecute] != 1 {
		t.Fatalf("autoExecute on a manual step was not reported: %+v", injected.Diagnostics)
	}
	if findStep(t, injected, "BP-A02").AutoExecuted {
		t.Fatal("a manual step was projected as auto-executed after the claim")
	}
}

// #1125: approval and rejection are two branches, not one branch with two
// labels. Losing either one loses the decision.
func TestApprovalAndRejectionAreSeparateBranches(t *testing.T) {
	projection := projectFixture(t, readFixture(t))

	if len(projection.Branches) != 2 {
		t.Fatalf("branches=%+v, want two separate branches", projection.Branches)
	}
	byOutcome := map[Outcome]BranchView{}
	for _, branch := range projection.Branches {
		byOutcome[branch.Outcome] = branch
	}
	approval, ok := byOutcome[OutcomeApproval]
	if !ok {
		t.Fatal("the approval branch is missing")
	}
	rejection, ok := byOutcome[OutcomeRejection]
	if !ok {
		t.Fatal("the rejection branch is missing")
	}
	if approval.ID == rejection.ID {
		t.Fatal("approval and rejection share one branch identity")
	}
	if approval.OnStep != rejection.OnStep {
		t.Fatalf("the branches are on different decisions: %q and %q", approval.OnStep, rejection.OnStep)
	}

	// Collapsing rejection must be reported rather than read as a one-way flow.
	fixture := readFixture(t)
	var kept []Branch
	for _, branch := range fixture.Branches {
		if branch.Outcome == OutcomeRejection {
			continue
		}
		kept = append(kept, branch)
	}
	fixture.Branches = kept
	collapsed := projectFixture(t, fixture)
	if codesOf(collapsed)[CodeBranchMissingRejection] != 1 {
		t.Fatalf("a decision with only an approval branch was not reported: %+v", collapsed.Diagnostics)
	}
}

// #1125: a Function reference must reach one target and that target's Source.
// A reference that resolves to nothing, or to more than one declaration, must be
// reported and must not be filled in.
func TestFunctionReferenceResolvesToUniqueTargetAndSource(t *testing.T) {
	projection := projectFixture(t, readFixture(t))

	submit := findStep(t, projection, "BP-A06")
	if submit.Function == nil {
		t.Fatal("the submit step did not resolve its Function")
	}
	if submit.Function.ID != "order_entry.submit_order" {
		t.Fatalf("function=%q, want order_entry.submit_order", submit.Function.ID)
	}
	if submit.Function.Artifact != "applications/order-entry/patterns/submit-order.tei" {
		t.Fatalf("artifact=%q, want the declaring source", submit.Function.Artifact)
	}
	// The exact position, not merely a positive one: a wrong line or column is
	// a source location that does not name the declaration.
	if submit.Function.Line != 1 || submit.Function.Column != 9 {
		t.Fatalf("source is %d:%d, want 1:9 (the pattern keyword in submit-order.tei)",
			submit.Function.Line, submit.Function.Column)
	}
	if submit.Function.Role != "definition" {
		t.Fatalf("role=%q, want definition", submit.Function.Role)
	}
	if submit.Function.Subject != "application-pattern:order_entry.submit_order" {
		t.Fatalf("subject=%q is not the declared identity", submit.Function.Subject)
	}

	// The system step resolves to a Validation, a different model than a Pattern,
	// and to its own exact declaration position.
	check := findStep(t, projection, "BP-A04")
	if check.Function == nil || check.Function.Kind != ReferenceValidation {
		t.Fatalf("the validation step did not resolve to a validation: %+v", check)
	}
	if check.Function.Artifact != "semantic/sales/validation.tei" || check.Function.Line != 3 || check.Function.Column != 12 {
		t.Fatalf("validation source is %s:%d:%d, want semantic/sales/validation.tei:3:12",
			check.Function.Artifact, check.Function.Line, check.Function.Column)
	}
	if check.Function.Artifact != "semantic/sales/validation.tei" {
		t.Fatalf("artifact=%q, want semantic/sales/validation.tei", check.Function.Artifact)
	}
}

// #1125: an unresolved reference must stay unresolved. Guessing a target from a
// similar name is exactly what makes a projection look complete when it is not.
func TestUnresolvedReferenceIsNotGuessed(t *testing.T) {
	fixture := readFixture(t)
	for i := range fixture.Steps {
		if fixture.Steps[i].ID == "BP-A06" {
			// A near-miss of the real id: same words, different identity.
			fixture.Steps[i].Function = &Reference{
				Kind: ReferencePattern,
				ID:   "order_entry.submit_order_v2",
			}
		}
	}
	projection := projectFixture(t, fixture)

	if codesOf(projection)[CodeReferenceUnresolved] != 1 {
		t.Fatalf("the unresolvable reference was not reported: %+v", projection.Diagnostics)
	}
	step := findStep(t, projection, "BP-A06")
	if step.Function != nil {
		t.Fatalf("an unresolved reference was resolved to %+v", step.Function)
	}
	if step.UnresolvedFunction == nil || step.UnresolvedFunction.ID != "order_entry.submit_order_v2" {
		t.Fatalf("the raw unresolved reference was dropped: %+v", step.UnresolvedFunction)
	}
}

// #1125: two declarations of one identity are ambiguous. The projection must not
// choose the first one.
func TestAmbiguousReferenceIsReportedNotChosen(t *testing.T) {
	targets := append(declaredTargetsWithDigests(), Declared{
		Kind: ReferencePattern, ID: "order_entry.submit_order",
		Subject:  "application-pattern:order_entry.submit_order",
		Artifact: "applications/order-entry/patterns/submit-order-copy.tei",
		Line:     1, Column: 9, Role: "definition", Fingerprint: "sha256:copy",
	})
	fixture := readFixture(t)
	projection := Project(fixture, Fingerprint(fixture), NewResolver(targets))

	if codesOf(projection)[CodeReferenceAmbiguous] != 1 {
		t.Fatalf("an ambiguous reference was not reported: %+v", projection.Diagnostics)
	}
	if step := findStep(t, projection, "BP-A06"); step.Function != nil {
		t.Fatalf("an ambiguous reference was resolved to %+v", step.Function)
	}
}

// #1125: a manual step must not be dressed up as application work.
func TestManualStepWithFunctionIsReported(t *testing.T) {
	fixture := readFixture(t)
	for i := range fixture.Steps {
		if fixture.Steps[i].ID == "BP-A02" {
			fixture.Steps[i].Function = &Reference{
				Kind: ReferencePattern,
				ID:   "order_entry.order_record",
			}
		}
	}
	projection := projectFixture(t, fixture)
	if codesOf(projection)[CodeManualStepHasFunction] != 1 {
		t.Fatalf("a manual step with a Function was not reported: %+v", projection.Diagnostics)
	}
	if !findStep(t, projection, "BP-A02").Manual {
		t.Fatal("the step stopped being manual because it named a Function")
	}
}

func TestUnknownRealizationIsReported(t *testing.T) {
	fixture := readFixture(t)
	fixture.Steps[0].Realization = "workflow"
	projection := projectFixture(t, fixture)
	if codesOf(projection)[CodeRealizationUnknown] != 1 {
		t.Fatalf("an unknown realization was not reported: %+v", projection.Diagnostics)
	}
}

func TestBranchOnUnknownStepIsReported(t *testing.T) {
	fixture := readFixture(t)
	fixture.Branches[0].OnStep = "BP-A99"
	projection := projectFixture(t, fixture)
	if codesOf(projection)[CodeBranchStepUnknown] != 1 {
		t.Fatalf("a branch on an unknown step was not reported: %+v", projection.Diagnostics)
	}
}

func TestDuplicateStepAndBranchIDsAreReported(t *testing.T) {
	fixture := readFixture(t)
	fixture.Steps = append(fixture.Steps, fixture.Steps[0])
	fixture.Branches = append(fixture.Branches, fixture.Branches[0])
	projection := projectFixture(t, fixture)
	codes := codesOf(projection)
	if codes[CodeStepDuplicateID] != 1 {
		t.Fatalf("a duplicated step id was not reported: %+v", projection.Diagnostics)
	}
	if codes[CodeBranchDuplicateID] != 1 {
		t.Fatalf("a duplicated branch id was not reported: %+v", projection.Diagnostics)
	}
}

func TestMissingProcedureIDIsReported(t *testing.T) {
	fixture := readFixture(t)
	fixture.Procedure.ID = ""
	projection := projectFixture(t, fixture)
	if codesOf(projection)[CodeProcedureIDMissing] != 1 {
		t.Fatalf("a missing procedure id was not reported: %+v", projection.Diagnostics)
	}
}

// A fixture must not claim to be accepted Canonical meaning.
func TestCanonicalClaimIsReported(t *testing.T) {
	fixture := readFixture(t)
	fixture.Canonical = true
	projection := projectFixture(t, fixture)
	if codesOf(projection)[CodeFixtureNotAuthoringReference] != 1 {
		t.Fatalf("a Canonical claim was not reported: %+v", projection.Diagnostics)
	}
}

// A procedure actor that repeats the authorization role code is the conflation
// the issue names. Every step inherits it, so each step is reported as well as
// the procedure: the reader sees the finding where the actor is shown.
func TestActorRepeatingRoleCodeIsReported(t *testing.T) {
	fixture := readFixture(t)
	fixture.Procedure.Actor.Label = "ORDER_EDITOR"
	projection := projectFixture(t, fixture)
	// One procedure finding plus one per step that inherited the actor.
	want := 1 + len(fixture.Steps)
	if got := codesOf(projection)[CodeActorConflatesAuthorization]; got != want {
		t.Fatalf("conflation findings=%d, want %d: %+v", got, want, projection.Diagnostics)
	}
}

func TestStepWithoutAnyActorIsReported(t *testing.T) {
	fixture := readFixture(t)
	fixture.Procedure.Actor.Label = ""
	projection := projectFixture(t, fixture)
	if codesOf(projection)[CodeStepActorUnattributed] == 0 {
		t.Fatalf("steps without any business actor were not reported: %+v", projection.Diagnostics)
	}
}

// The projection must not carry an authority shortcut: no grant, no permission,
// no auto-run switch.
func TestProjectionCarriesNoAuthorityShortcut(t *testing.T) {
	projection := projectFixture(t, readFixture(t))
	if projection.Effect != EffectReadOnly {
		t.Fatalf("effect=%q, want read-only", projection.Effect)
	}
	if projection.Trigger.Invocation {
		t.Fatal("the projection carries a trigger invocation flag that is true")
	}
	if !containsString(projection.AuthorityLimits, "unresolved-reference-is-reported-not-inferred") {
		t.Fatalf("authorityLimits=%v does not name the unresolved rule", projection.AuthorityLimits)
	}
	if !containsString(projection.AuthorityLimits, "approval-and-rejection-are-separate-branches") {
		t.Fatalf("authorityLimits=%v does not name the branch rule", projection.AuthorityLimits)
	}
}

// The fingerprint describes the document, not the claim to be Canonical.
func TestFingerprintIgnoresTheCanonicalClaim(t *testing.T) {
	fixture := readFixture(t)
	plain := Fingerprint(fixture)
	fixture.Canonical = true
	if Fingerprint(fixture) != plain {
		t.Fatal("the fingerprint changed with the Canonical claim; it must describe the document")
	}
	// It must depend on the content: a digest that ignores the document would be
	// constant and would compare two different procedures as equal.
	changed := readFixture(t)
	changed.Steps[0].Action = "受注を受け付ける（変更後）"
	if Fingerprint(changed) == plain {
		t.Fatal("the fingerprint did not change with the procedure content")
	}
}

// A step-level actor that repeats the authorization role code is the same
// conflation as a procedure-level one, even when the procedure-level labels
// differ. The earlier check only compared the procedure actor, so this fixture
// produced no finding while carrying the confusion.
func TestStepActorRepeatingRoleCodeIsReported(t *testing.T) {
	fixture := readFixture(t)
	for i := range fixture.Steps {
		if fixture.Steps[i].ID == "BP-A02" {
			fixture.Steps[i].ActorLabel = fixture.Procedure.Authorization.RoleCode
		}
	}
	projection := projectFixture(t, fixture)
	if codesOf(projection)[CodeActorConflatesAuthorization] != 1 {
		t.Fatalf("a step whose actor is the role code was not reported: %+v", projection.Diagnostics)
	}
	if findStep(t, projection, "BP-A02").ActorLabel != fixture.Procedure.Authorization.RoleCode {
		t.Fatal("the conflation check silently replaced the actor instead of reporting it")
	}
}

// An outcome outside the closed vocabulary must not be read as an approval or a
// rejection. Before the vocabulary, two spellings of one decision looked like
// two unrelated branches and a one-sided decision looked complete.
func TestUnknownBranchOutcomeIsReported(t *testing.T) {
	fixture := readFixture(t)
	for i := range fixture.Branches {
		if fixture.Branches[i].Outcome == OutcomeRejection {
			fixture.Branches[i].Outcome = "Rejected"
		}
	}
	projection := projectFixture(t, fixture)
	if codesOf(projection)[CodeBranchOutcomeUnknown] != 1 {
		t.Fatalf("an outcome outside the vocabulary was not reported: %+v", projection.Diagnostics)
	}
	// The misspelled rejection must not be counted as a rejection, so the
	// decision is now one-sided and that must be reported too.
	if codesOf(projection)[CodeBranchMissingRejection] != 1 {
		t.Fatalf("the misspelled rejection was counted as a rejection: %+v", projection.Diagnostics)
	}
}

// The reverse direction of the decision rule: a rejection without an approval.
func TestDecisionWithOnlyRejectionIsReported(t *testing.T) {
	fixture := readFixture(t)
	var kept []Branch
	for _, branch := range fixture.Branches {
		if branch.Outcome == OutcomeApproval {
			continue
		}
		kept = append(kept, branch)
	}
	fixture.Branches = kept
	projection := projectFixture(t, fixture)
	if codesOf(projection)[CodeBranchMissingRejection] != 1 {
		t.Fatalf("a decision with only a rejection branch was not reported: %+v", projection.Diagnostics)
	}
}

// Two approvals on one step are not a decision with two sides; the missing
// rejection must still be reported.
func TestTwoApprovalsDoNotStandInForARejection(t *testing.T) {
	fixture := readFixture(t)
	var kept []Branch
	for _, branch := range fixture.Branches {
		if branch.Outcome == OutcomeRejection {
			continue
		}
		kept = append(kept, branch)
	}
	second := kept[0]
	second.ID = "BR-A05-APPROVE-2"
	fixture.Branches = append(kept, second)
	projection := projectFixture(t, fixture)
	if codesOf(projection)[CodeBranchMissingRejection] != 1 {
		t.Fatalf("two approvals hid the missing rejection: %+v", projection.Diagnostics)
	}
}

// Severities are contract values a caller branches on. This pins the words and
// the order the projection sorts by.
func TestDiagnosticSeveritiesAreThePinnedValues(t *testing.T) {
	if SeverityProcedure != "procedure-error" || SeverityReference != "reference-error" {
		t.Fatalf("severities are %q and %q", SeverityProcedure, SeverityReference)
	}
	if severityRank(SeverityProcedure) >= severityRank(SeverityReference) {
		t.Fatal("a procedure finding must sort before a reference finding")
	}

	// A projection with both kinds sorts procedure findings first.
	fixture := readFixture(t)
	fixture.Steps[1].Function = &Reference{Kind: ReferencePattern, ID: "order_entry.order_record"}
	fixture.Steps[5].Function = &Reference{Kind: ReferencePattern, ID: "order_entry.submit_order_missing"}
	projection := projectFixture(t, fixture)
	if len(projection.Diagnostics) < 2 {
		t.Fatalf("expected both a procedure and a reference finding: %+v", projection.Diagnostics)
	}
	if projection.Diagnostics[0].Severity != SeverityProcedure {
		t.Fatalf("first finding is %q, want %q", projection.Diagnostics[0].Severity, SeverityProcedure)
	}
}

// A manual step that keeps its instruction is the expected case; a manual step
// with no instruction is kept too, because the projection must not drop the work
// just because the wording is missing.
func TestManualStepWithoutInstructionIsKeptAndNamed(t *testing.T) {
	fixture := readFixture(t)
	for i := range fixture.Steps {
		if fixture.Steps[i].ID == "BP-A02" {
			fixture.Steps[i].ManualInstruction = ""
		}
	}
	projection := projectFixture(t, fixture)
	if !containsString(projection.ManualSteps, "BP-A02") {
		t.Fatalf("a manual step with no instruction was dropped: %v", projection.ManualSteps)
	}
	if findStep(t, projection, "BP-A02").ID != "BP-A02" {
		t.Fatal("the manual step did not survive")
	}
}

// A branch bound to no step silently disabled the one-sided-decision check: the
// decision it belongs to was unknown, so it could not be shown to have two sides.
func TestBranchWithNoStepIsReported(t *testing.T) {
	fixture := readFixture(t)
	for i := range fixture.Branches {
		fixture.Branches[i].OnStep = ""
	}
	projection := projectFixture(t, fixture)
	if codesOf(projection)[CodeBranchStepMissing] != 2 {
		t.Fatalf("stepless branches were not reported: %+v", projection.Diagnostics)
	}
}

// A step with no id cannot be referred to by a branch, is indistinguishable in
// manualSteps, and cannot be addressed in a finding.
func TestStepWithNoIDIsReported(t *testing.T) {
	fixture := readFixture(t)
	fixture.Steps[1].ID = ""
	projection := projectFixture(t, fixture)
	if codesOf(projection)[CodeStepIDMissing] != 1 {
		t.Fatalf("a missing step id was not reported: %+v", projection.Diagnostics)
	}
}

// A reference fixture proposes meaning. It must not be able to assert that its
// own candidate values are accepted, and it must not leave the status unsaid:
// an absent status and a wrong one are both reported, so the defect does not
// depend on which the author wrote.
func TestNonCandidateStatusIsReported(t *testing.T) {
	fixture := readFixture(t)
	fixture.Procedure.Trigger.Status = "accepted"
	fixture.Procedure.Actor.Status = "accepted"
	fixture.Procedure.Authorization.Status = "accepted"
	projection := projectFixture(t, fixture)
	if codesOf(projection)[CodeStatusNotCandidate] != 3 {
		t.Fatalf("non-candidate statuses were not reported: %+v", projection.Diagnostics)
	}

	// An absent status is not a claim, but the projection must not echo an empty
	// state as one either. It is reported with its own wording.
	empty := readFixture(t)
	empty.Procedure.Trigger.Status = ""
	emptyProjection := projectFixture(t, empty)
	if codesOf(emptyProjection)[CodeStatusNotCandidate] != 1 {
		t.Fatalf("an absent status was not reported: %+v", emptyProjection.Diagnostics)
	}
	var message string
	for _, diagnostic := range emptyProjection.Diagnostics {
		if diagnostic.Code == CodeStatusNotCandidate {
			message = diagnostic.Message
		}
	}
	if !strings.Contains(message, "declares no status") {
		t.Fatalf("an absent status used the wrong wording: %q", message)
	}
	// The /valid baseline still reports nothing.
	if codesOf(projectFixture(t, readFixture(t)))[CodeStatusNotCandidate] != 0 {
		t.Fatal("the candidate-only baseline was reported")
	}
}

// autoExecute is refused on a step the reader would expect to run by itself,
// not only on a manual one.
func TestAutoExecuteOnNonManualStepIsReported(t *testing.T) {
	fixture := readFixture(t)
	fixture.Steps[0].AutoExecute = true // BP-A01 is an application step
	projection := projectFixture(t, fixture)
	if codesOf(projection)[CodeAutoExecuteOnSystemStep] != 1 {
		t.Fatalf("autoExecute on an application step was not reported: %+v", projection.Diagnostics)
	}
	if findStep(t, projection, "BP-A01").AutoExecuted {
		t.Fatal("an application step was projected as auto-executed")
	}
	// A manual step gets the manual-specific finding instead, not both.
	manualAuto := readFixture(t)
	manualAuto.Steps[1].AutoExecute = true
	codes := codesOf(projectFixture(t, manualAuto))
	if codes[CodeManualStepAutoExecute] != 1 || codes[CodeAutoExecuteOnSystemStep] != 0 {
		t.Fatalf("a manual autoExecute claim was reported under the wrong code: %v", codes)
	}
}

// A resolved reference must carry its artifact digest into the projection, so
// the Source it names is the bytes the index read.
func TestResolvedReferenceCarriesTheArtifactDigest(t *testing.T) {
	projection := projectFixture(t, readFixture(t))
	for _, step := range projection.Steps {
		if step.Function == nil {
			continue
		}
		want := "sha256:" + string(step.Function.ID)
		if step.Function.Fingerprint != want {
			t.Fatalf("step %s resolved %q with digest %q, want %q",
				step.ID, step.Function.ID, step.Function.Fingerprint, want)
		}
	}
	if projection.Actor.AuthorizationCapability == nil {
		t.Fatal("the authorization capability did not resolve")
	}
	if projection.Actor.AuthorizationCapability.Fingerprint == "" {
		t.Fatal("the authorization capability resolved without its artifact digest")
	}
}

// A target that does not name a real Source declaration must not resolve: the
// projection claims a reference reaches a Source, so a target without a role, a
// position or a digest cannot support that claim.
func TestTargetsWithoutASourceAreNotResolvable(t *testing.T) {
	valid := Declared{
		Kind: ReferencePattern, ID: "order_entry.submit_order",
		Subject:  "application-pattern:order_entry.submit_order",
		Artifact: "applications/order-entry/patterns/submit-order.tei",
		Line:     1, Column: 9, Role: "definition", Fingerprint: "sha256:x",
	}
	cases := map[string]func(*Declared){
		"no digest":      func(d *Declared) { d.Fingerprint = "" },
		"no position":    func(d *Declared) { d.Line, d.Column = 0, 0 },
		"no artifact":    func(d *Declared) { d.Artifact = "" },
		"authoring role": func(d *Declared) { d.Role = "authoring" },
		"absolute path":  func(d *Declared) { d.Artifact = "C:/Windows/win.ini" },
		"escape":         func(d *Declared) { d.Artifact = "../../etc/passwd" },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			target := valid
			mutate(&target)
			resolution := NewResolver([]Declared{target}).Resolve(Reference{
				Kind: ReferencePattern,
				ID:   "order_entry.submit_order",
			})
			if resolution.Resolved != nil {
				t.Fatalf("%s resolved: %+v", name, resolution.Resolved)
			}
		})
	}
}

// A reference diagnostic must carry the reference severity, so a consumer that
// branches on severity sees it as a reference problem.
func TestReferenceFindingsCarryTheReferenceSeverity(t *testing.T) {
	fixture := readFixture(t)
	fixture.Steps[5].Function = &Reference{Kind: ReferencePattern, ID: "order_entry.submit_order_missing"}
	projection := projectFixture(t, fixture)
	var found bool
	for _, diagnostic := range projection.Diagnostics {
		if diagnostic.Code == CodeReferenceUnresolved {
			found = true
			if diagnostic.Severity != SeverityReference {
				t.Fatalf("unresolved reference has severity %q, want %q",
					diagnostic.Severity, SeverityReference)
			}
		}
	}
	if !found {
		t.Fatal("the unresolved reference was not reported")
	}
}

// A branch with no identity cannot be distinguished from another anonymous
// branch, and the duplicate check cannot see either of them.
func TestBranchWithNoIDIsReported(t *testing.T) {
	fixture := readFixture(t)
	for i := range fixture.Branches {
		fixture.Branches[i].ID = ""
	}
	projection := projectFixture(t, fixture)
	if codesOf(projection)[CodeBranchIDMissing] != 2 {
		t.Fatalf("anonymous branches were not reported: %+v", projection.Diagnostics)
	}
	// Whitespace is not an identity either.
	spaced := readFixture(t)
	spaced.Branches[0].ID = "   "
	if codesOf(projectFixture(t, spaced))[CodeBranchIDMissing] != 1 {
		t.Fatal("a whitespace-only branch id was accepted as an identity")
	}
}

// A manual step must not hold out a Function as its action. A consumer that
// reads steps[].function without reading diagnostics must not see application
// work on the phone call.
func TestManualStepDoesNotCarryAResolvedFunction(t *testing.T) {
	fixture := readFixture(t)
	for i := range fixture.Steps {
		if fixture.Steps[i].ID == "BP-A02" {
			fixture.Steps[i].Function = &Reference{Kind: ReferencePattern, ID: "order_entry.order_record"}
		}
	}
	projection := projectFixture(t, fixture)
	step := findStep(t, projection, "BP-A02")
	if !step.Manual {
		t.Fatal("the step stopped being manual")
	}
	if step.Function != nil {
		t.Fatalf("a manual step was given the Function %+v", step.Function)
	}
	if step.UnresolvedFunction == nil || step.UnresolvedFunction.ID != "order_entry.order_record" {
		t.Fatalf("the manual step's Function reference was dropped entirely: %+v", step.UnresolvedFunction)
	}
	if codesOf(projection)[CodeManualStepHasFunction] != 1 {
		t.Fatalf("the manual step's Function was not reported: %+v", projection.Diagnostics)
	}
}

// The same input must produce the same bytes. Three findings that share a
// severity, a code and a subject differ only in wording, so a comparator that
// stops before the message leaves their order to whichever order the rules ran.
func TestProjectionDiagnosticsAreDeterministic(t *testing.T) {
	fixture := readFixture(t)
	fixture.Procedure.Trigger.Status = "accepted"
	fixture.Procedure.Actor.Status = "accepted"
	fixture.Procedure.Authorization.Status = "accepted"

	first, err := json.Marshal(projectFixture(t, fixture))
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 50; i++ {
		next, err := json.Marshal(projectFixture(t, fixture))
		if err != nil {
			t.Fatal(err)
		}
		if string(next) != string(first) {
			t.Fatalf("run %d differs from the first run;\nfirst: %s\nnext:  %s", i, first, next)
		}
	}

	// The three findings must be ordered by their message, not by chance.
	projection := projectFixture(t, fixture)
	var messages []string
	for _, diagnostic := range projection.Diagnostics {
		if diagnostic.Code == CodeStatusNotCandidate {
			messages = append(messages, diagnostic.Message)
		}
	}
	if len(messages) != 3 {
		t.Fatalf("status findings=%v, want three", messages)
	}
	for i := 1; i < len(messages); i++ {
		if messages[i-1] > messages[i] {
			t.Fatalf("status findings are not in message order: %v", messages)
		}
	}
}

// An unknown realization cannot be shown to be application or system work, so
// the projection must not hand it a Function. Otherwise the "a phone call is not
// application work" harm returns whenever the author misspells the realization.
func TestUnknownRealizationDoesNotGetAFunction(t *testing.T) {
	fixture := readFixture(t)
	for i := range fixture.Steps {
		if fixture.Steps[i].ID == "BP-A02" {
			fixture.Steps[i].Realization = "telephone"
			fixture.Steps[i].Function = &Reference{Kind: ReferencePattern, ID: "order_entry.order_record"}
		}
	}
	projection := projectFixture(t, fixture)
	step := findStep(t, projection, "BP-A02")
	if step.Function != nil {
		t.Fatalf("an unknown realization was given the Function %+v", step.Function)
	}
	if step.UnresolvedFunction == nil {
		t.Fatal("the reference was dropped instead of kept as unresolved")
	}
	if !step.Manual {
		t.Fatal("an unknown realization was treated as application work")
	}
	if codesOf(projection)[CodeRealizationUnknown] != 1 {
		t.Fatalf("the unknown realization was not reported: %+v", projection.Diagnostics)
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
