package businessprocedure

import (
	"os"
	"regexp"
	"sort"
	"testing"
)

// A diagnostic code is a contract: a caller branches on it. A code that no rule
// can emit reads like a contract while being unverified, so this test names the
// closed set and requires the package's own rules to reach every entry.
func TestEveryDeclaredDiagnosticCodeIsReached(t *testing.T) {
	reached := reachedCodes(t)
	declared := declaredCodes(t)

	var unreached []string
	for _, code := range declared {
		if !reached[code] {
			unreached = append(unreached, code)
		}
	}
	if len(unreached) != 0 {
		t.Fatalf("%d diagnostic codes are declared but no test reaches them: %v", len(unreached), unreached)
	}
}

// TestDeclaredDiagnosticCodesAreTheClosedSet pins the count, so adding a code
// without reaching it is a deliberate edit rather than an oversight.
func TestDeclaredDiagnosticCodesAreTheClosedSet(t *testing.T) {
	declared := declaredCodes(t)
	const expected = 18
	if len(declared) != expected {
		t.Fatalf("declared %d diagnostic codes, want %d; update this count when a code is added: %v",
			len(declared), expected, declared)
	}
}

// declaredCodePattern matches the `Code... = "BUSINESS_..."` constant shape.
var declaredCodePattern = regexp.MustCompile(`(?m)^\s*Code[A-Za-z0-9_]*\s*=\s*"([A-Z][A-Z0-9_]+)"`)

// declaredCodes reads the code constants from the package's own source, so a
// code added to a constant block is counted without editing this test.
func declaredCodes(t *testing.T) []string {
	t.Helper()
	source, err := os.ReadFile("project.go")
	if err != nil {
		t.Fatalf("read project.go: %v", err)
	}
	var codes []string
	for _, match := range declaredCodePattern.FindAllStringSubmatch(string(source), -1) {
		codes = append(codes, match[1])
	}
	sort.Strings(codes)
	if len(codes) == 0 {
		t.Fatal("no diagnostic code constant was found; the declaration shape changed")
	}
	return codes
}

// reachedCodes collects the codes the projections in this test observe. It runs
// the same projections the behavioural tests run, so a code that stops being
// emitted fails here too.
func reachedCodes(t *testing.T) map[string]bool {
	t.Helper()
	reached := map[string]bool{}
	record := func(projection Projection) {
		for _, diagnostic := range projection.Diagnostics {
			reached[diagnostic.Code] = true
		}
	}

	// The healthy projection reaches no error code; each injected change below
	// reaches one.
	record(projectFixture(t, readFixture(t)))

	noID := readFixture(t)
	noID.Procedure.ID = ""
	record(projectFixture(t, noID))

	duplicateStep := readFixture(t)
	duplicateStep.Steps = append(duplicateStep.Steps, duplicateStep.Steps[0])
	record(projectFixture(t, duplicateStep))

	noActor := readFixture(t)
	noActor.Procedure.Actor.Label = ""
	record(projectFixture(t, noActor))

	manualWithFunction := readFixture(t)
	manualWithFunction.Steps[1].Function = &Reference{Kind: ReferencePattern, ID: "order_entry.order_record"}
	record(projectFixture(t, manualWithFunction))

	manualAuto := readFixture(t)
	manualAuto.Steps[1].AutoExecute = true
	record(projectFixture(t, manualAuto))

	unknownRealization := readFixture(t)
	unknownRealization.Steps[0].Realization = "workflow"
	record(projectFixture(t, unknownRealization))

	unknownBranchStep := readFixture(t)
	unknownBranchStep.Branches[0].OnStep = "BP-A99"
	record(projectFixture(t, unknownBranchStep))

	duplicateBranch := readFixture(t)
	duplicateBranch.Branches = append(duplicateBranch.Branches, duplicateBranch.Branches[0])
	record(projectFixture(t, duplicateBranch))

	oneBranch := readFixture(t)
	oneBranch.Branches = oneBranch.Branches[:1]
	record(projectFixture(t, oneBranch))

	// The reverse direction of the decision rule: a rejection with no approval.
	onlyRejection := readFixture(t)
	onlyRejection.Branches = onlyRejection.Branches[1:]
	record(projectFixture(t, onlyRejection))

	unknownOutcome := readFixture(t)
	unknownOutcome.Branches[1].Outcome = "Rejected"
	record(projectFixture(t, unknownOutcome))

	// A branch bound to no step makes the one-sided check vacuous.
	branchStepMissing := readFixture(t)
	branchStepMissing.Branches[0].OnStep = ""
	record(projectFixture(t, branchStepMissing))

	// A step with no id cannot be referred to or addressed.
	stepIDMissing := readFixture(t)
	stepIDMissing.Steps[0].ID = ""
	record(projectFixture(t, stepIDMissing))

	// A fixture that asserts its own values are accepted.
	statusNotCandidate := readFixture(t)
	statusNotCandidate.Procedure.Actor.Status = "accepted"
	record(projectFixture(t, statusNotCandidate))

	// autoExecute on a step the reader would expect to run by itself.
	autoExecuteSystem := readFixture(t)
	autoExecuteSystem.Steps[0].AutoExecute = true
	record(projectFixture(t, autoExecuteSystem))

	conflated := readFixture(t)
	conflated.Procedure.Actor.Label = "ORDER_EDITOR"
	record(projectFixture(t, conflated))

	unresolved := readFixture(t)
	unresolved.Steps[5].Function = &Reference{Kind: ReferencePattern, ID: "order_entry.submit_order_v2"}
	record(projectFixture(t, unresolved))

	ambiguousTargets := append(declaredTargetsWithDigests(), Declared{
		Kind: ReferencePattern, ID: "order_entry.submit_order",
		Subject:  "application-pattern:order_entry.submit_order",
		Artifact: "applications/order-entry/patterns/submit-order-copy.tei",
		Line:     1, Column: 9, Role: "definition", Fingerprint: "sha256:copy",
	})
	ambiguous := readFixture(t)
	record(Project(ambiguous, Fingerprint(ambiguous), NewResolver(ambiguousTargets)))

	canonical := readFixture(t)
	canonical.Canonical = true
	record(projectFixture(t, canonical))

	return reached
}
