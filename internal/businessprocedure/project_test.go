package businessprocedure

import (
	"encoding/json"
	"os"
	"path/filepath"
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

// declaredTargets are the existing identities the reference workspace declares.
// They are written here independently of the fixture so a reference that stops
// existing is caught rather than silently resolved by a fixture-local copy.
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
			Line:     27, Column: 1, Role: "definition"},
		{Kind: ReferenceCapability, ID: "security.authorize",
			Subject:  "semantic-capability:security.authorize",
			Artifact: "semantic/security/authorization.tei",
			Line:     20, Column: 1, Role: "definition"},
		{Kind: ReferenceValidation, ID: "sales.order_line.quantity_positive",
			Subject:  "semantic-validation:sales.order_line.quantity_positive",
			Artifact: "semantic/sales/validation.tei",
			Line:     3, Column: 1, Role: "definition"},
		{Kind: ReferenceValidation, ID: "sales.order.lines.line_number_unique",
			Subject:  "semantic-validation:sales.order.lines.line_number_unique",
			Artifact: "semantic/sales/validation.tei",
			Line:     7, Column: 1, Role: "definition"},
	}
}

func projectFixture(t *testing.T, fixture Fixture) Projection {
	t.Helper()
	return Project(fixture, Fingerprint(fixture), NewResolver(declaredTargets()))
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
	byOutcome := map[string]BranchView{}
	for _, branch := range projection.Branches {
		byOutcome[branch.Outcome] = branch
	}
	approval, ok := byOutcome["approval"]
	if !ok {
		t.Fatal("the approval branch is missing")
	}
	rejection, ok := byOutcome["rejection"]
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
		if branch.Outcome == "rejection" {
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
	if submit.Function.Line < 1 || submit.Function.Column < 1 {
		t.Fatalf("function resolved without a source position: %+v", submit.Function)
	}
	if submit.Function.Subject != "application-pattern:order_entry.submit_order" {
		t.Fatalf("subject=%q is not the declared identity", submit.Function.Subject)
	}

	// The system step resolves to a Validation, a different model than a Pattern.
	check := findStep(t, projection, "BP-A04")
	if check.Function == nil || check.Function.Kind != ReferenceValidation {
		t.Fatalf("the validation step did not resolve to a validation: %+v", check)
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
	targets := append(declaredTargets(), Declared{
		Kind: ReferencePattern, ID: "order_entry.submit_order",
		Subject:  "application-pattern:order_entry.submit_order",
		Artifact: "applications/order-entry/patterns/submit-order-copy.tei",
		Line:     1, Column: 9, Role: "definition",
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

// A step that repeats the authorization role code as its business actor is the
// conflation the issue names.
func TestActorRepeatingRoleCodeIsReported(t *testing.T) {
	fixture := readFixture(t)
	fixture.Procedure.Actor.Label = "ORDER_EDITOR"
	projection := projectFixture(t, fixture)
	if codesOf(projection)[CodeActorConflatesAuthorization] != 1 {
		t.Fatalf("an actor repeating the role code was not reported: %+v", projection.Diagnostics)
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
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
