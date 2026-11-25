package operations

import (
	"math/big"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/semantic"
	"tei/internal/semantic/validationeval"
	"tei/internal/sourceindex"
	"tei/internal/subject"
	"tei/internal/workspace"
)

// A manifest's implementation selection is a statement of the Workspace, and
// validate must refuse one this running tei cannot honour even when no example
// or rule is read in this session.
//
// Before this, such a selection was reported only while checking an example, so
// a Workspace with no example tables passed validate and failed at the first
// read:
//
//	$ tei validate                    -> ok
//	$ tei view -environment ...       -> TEI_OPERATOR_IMPLEMENTATION_UNAVAILABLE
//
// These cases need no document and no database: the selection, the declared
// operators and the available implementations are built here.

// selectionCheckWorld declares one operator per case, each with the input type
// the case needs.
func selectionCheckWorld(operators ...semantic.Operator) ir.World {
	return ir.World{Semantics: []semantic.Namespace{{ID: "stock", Operators: operators}}}
}

func selectionCheckLoaded(world ir.World, selections []workspace.OperatorImplementationSelection) workspace.Result {
	return workspace.Result{World: world, OperatorImplementations: selections}
}

func selectionCheckCodes(loaded workspace.Result) []string {
	var codes []string
	for _, finding := range CheckOperatorSelection(loaded, sourceindex.Index{}, selectionCheckAvailable()) {
		codes = append(codes, finding.Diagnostic.Code)
	}
	return codes
}

// selectionCheckAvailable is one number implementation and one presence one,
// so a case can select the wrong kind instead of only a missing name.
func selectionCheckAvailable() []validationeval.NamedImplementation {
	return []validationeval.NamedImplementation{
		{ID: "test.number", Input: semantic.OperatorInputNumber,
			Number: func(*big.Rat) (bool, error) { return true, nil }},
		{ID: "test.presence", Input: semantic.OperatorInputPresence,
			Number: func(*big.Rat) (bool, error) { return true, nil }},
	}
}

func TestCheckOperatorSelectionRefusesAnImplementationThisTeiDoesNotCarry(t *testing.T) {
	loaded := selectionCheckLoaded(
		selectionCheckWorld(semantic.Operator{ID: "stock.number_op", Symbol: "number_op", Input: semantic.OperatorInputNumber}),
		[]workspace.OperatorImplementationSelection{{Operator: "stock.number_op", Implementation: "test.does-not-exist"}},
	)
	codes := selectionCheckCodes(loaded)
	if len(codes) != 1 || codes[0] != OperatorSelectionUnresolvedCode {
		t.Fatalf("codes=%v, want exactly the selection finding", codes)
	}
	finding := CheckOperatorSelection(loaded, sourceindex.Index{}, selectionCheckAvailable())[0]
	if !strings.Contains(finding.Diagnostic.Message, "stock.number_op") ||
		!strings.Contains(finding.Diagnostic.Message, "test.does-not-exist") {
		t.Fatalf("the finding does not name the operator and the implementation: %q", finding.Diagnostic.Message)
	}
	// The subject is the operator, so a reader can go to its declaration.
	if finding.Diagnostic.Subject == nil || finding.Diagnostic.Subject.Kind != "semantic-operator" ||
		finding.Diagnostic.Subject.ID != "stock.number_op" {
		t.Fatalf("subject=%+v, want the operator", finding.Diagnostic.Subject)
	}
}

// A selection that resolves by name can still read a different kind of input
// than the operator declares. The evaluator refuses that when it is asked, so
// validate must refuse it without waiting for a read.
func TestCheckOperatorSelectionRefusesAnImplementationOfTheWrongInput(t *testing.T) {
	loaded := selectionCheckLoaded(
		selectionCheckWorld(semantic.Operator{ID: "stock.presence_op", Symbol: "presence_op", Input: semantic.OperatorInputPresence}),
		[]workspace.OperatorImplementationSelection{{Operator: "stock.presence_op", Implementation: "test.number"}},
	)
	codes := selectionCheckCodes(loaded)
	if len(codes) != 1 || codes[0] != OperatorSelectionUnresolvedCode {
		t.Fatalf("codes=%v, want exactly the selection finding", codes)
	}
	finding := CheckOperatorSelection(loaded, sourceindex.Index{}, selectionCheckAvailable())[0]
	if !strings.Contains(finding.Diagnostic.Message, "presence") ||
		!strings.Contains(finding.Diagnostic.Message, "number") {
		t.Fatalf("the finding does not name both inputs: %q", finding.Diagnostic.Message)
	}
}

// Two entries for one operator are ambiguous, and the message names the
// operator rather than an implementation name the reader could look up.
// The finding must still be produced and still point at that operator.
func TestCheckOperatorSelectionRefusesTwoEntriesForOneOperator(t *testing.T) {
	loaded := selectionCheckLoaded(
		selectionCheckWorld(semantic.Operator{ID: "stock.number_op", Symbol: "number_op", Input: semantic.OperatorInputNumber}),
		[]workspace.OperatorImplementationSelection{
			{Operator: "stock.number_op", Implementation: "test.number"},
			{Operator: "stock.number_op", Implementation: "test.presence"},
		},
	)
	codes := selectionCheckCodes(loaded)
	if len(codes) != 1 || codes[0] != OperatorSelectionUnresolvedCode {
		t.Fatalf("codes=%v, want exactly the selection finding", codes)
	}
	finding := CheckOperatorSelection(loaded, sourceindex.Index{}, selectionCheckAvailable())[0]
	if !strings.Contains(finding.Diagnostic.Message, "TEI_OPERATOR_IMPLEMENTATION_AMBIGUOUS") {
		t.Fatalf("the finding does not carry the ambiguity code: %q", finding.Diagnostic.Message)
	}
	// The message names the operator, not an implementation, so the finding
	// cannot be attributed by implementation name. It must still land on the
	// operator the two entries share.
	if finding.Diagnostic.Subject == nil || finding.Diagnostic.Subject.ID != "stock.number_op" {
		t.Fatalf("subject=%+v, want stock.number_op", finding.Diagnostic.Subject)
	}
}

// A name one available implementation carries twice is ambiguous too. Here the
// message does name the implementation, so the finding must be attributed to
// the operator that selected it, not to whichever entry happens to be first.
func TestCheckOperatorSelectionAttributesATwoEntryNameToTheOperatorThatSelectedIt(t *testing.T) {
	loaded := selectionCheckLoaded(
		selectionCheckWorld(
			semantic.Operator{ID: "stock.first_op", Symbol: "first_op", Input: semantic.OperatorInputNumber},
			semantic.Operator{ID: "stock.second_op", Symbol: "second_op", Input: semantic.OperatorInputNumber},
		),
		[]workspace.OperatorImplementationSelection{
			// This one resolves, so it must not be the one reported.
			{Operator: "stock.first_op", Implementation: "test.number"},
			{Operator: "stock.second_op", Implementation: "test.duplicated"},
		},
	)
	available := append(selectionCheckAvailable(),
		[]validationeval.NamedImplementation{
			{ID: "test.duplicated", Input: semantic.OperatorInputNumber,
				Number: func(*big.Rat) (bool, error) { return true, nil }},
			{ID: "test.duplicated", Input: semantic.OperatorInputNumber,
				Number: func(*big.Rat) (bool, error) { return true, nil }},
		}...)
	findings := CheckOperatorSelection(loaded, sourceindex.Index{}, available)
	if len(findings) != 1 {
		t.Fatalf("findings=%d, want one", len(findings))
	}
	if !strings.Contains(findings[0].Diagnostic.Message, "TEI_OPERATOR_IMPLEMENTATION_AMBIGUOUS") ||
		!strings.Contains(findings[0].Diagnostic.Message, "test.duplicated") {
		t.Fatalf("the finding does not carry the ambiguity and the name: %q", findings[0].Diagnostic.Message)
	}
	if findings[0].Diagnostic.Subject == nil || findings[0].Diagnostic.Subject.ID != "stock.second_op" {
		t.Fatalf("subject=%+v, want stock.second_op (the operator that selected the name)",
			findings[0].Diagnostic.Subject)
	}
}

// A selection that this binary can honour, and a Workspace that selected
// nothing, both produce no finding. Without this the refusal above could be a
// blanket refusal of every selection.
func TestCheckOperatorSelectionAcceptsWhatItCanHonour(t *testing.T) {
	accepted := selectionCheckLoaded(
		selectionCheckWorld(semantic.Operator{ID: "stock.number_op", Symbol: "number_op", Input: semantic.OperatorInputNumber}),
		[]workspace.OperatorImplementationSelection{{Operator: "stock.number_op", Implementation: "test.number"}},
	)
	if codes := selectionCheckCodes(accepted); len(codes) != 0 {
		t.Fatalf("a selection this tei carries was refused: %v", codes)
	}
	// The presence operator with the presence implementation, which is the case
	// a blanket "number only" check would wrongly refuse.
	presence := selectionCheckLoaded(
		selectionCheckWorld(semantic.Operator{ID: "stock.presence_op", Symbol: "presence_op", Input: semantic.OperatorInputPresence}),
		[]workspace.OperatorImplementationSelection{{Operator: "stock.presence_op", Implementation: "test.presence"}},
	)
	if codes := selectionCheckCodes(presence); len(codes) != 0 {
		t.Fatalf("a matching presence selection was refused: %v", codes)
	}
	// No selection at all is not a failure: the World's operators are simply
	// not asked about real data.
	if codes := selectionCheckCodes(selectionCheckLoaded(selectionCheckWorld(), nil)); len(codes) != 0 {
		t.Fatalf("an unselected Workspace was refused: %v", codes)
	}
}

// The finding must point at the operator's declaration when the Workspace
// records one, so a reader lands on the line to change.
func TestCheckOperatorSelectionPointsAtTheDeclaration(t *testing.T) {
	opSubject := subject.Subject{Kind: "semantic-operator", ID: "stock.number_op"}
	index := sourceindex.Index{References: []sourceindex.Reference{{
		Subject:      opSubject,
		ArtifactPath: "semantic/stock/operators.tei.md",
		Range:        sourceindex.Range{Start: sourceindex.Position{Offset: 10, Line: 3, Column: 1}},
		Role:         sourceindex.RoleDefinition,
	}}}
	loaded := selectionCheckLoaded(
		selectionCheckWorld(semantic.Operator{ID: "stock.number_op", Symbol: "number_op", Input: semantic.OperatorInputNumber}),
		[]workspace.OperatorImplementationSelection{{Operator: "stock.number_op", Implementation: "test.does-not-exist"}},
	)
	findings := CheckOperatorSelection(loaded, index, selectionCheckAvailable())
	if len(findings) != 1 {
		t.Fatalf("findings=%d, want one", len(findings))
	}
	if len(findings[0].Sources) != 1 || findings[0].Sources[0].ArtifactPath != "semantic/stock/operators.tei.md" {
		t.Fatalf("the finding does not carry the declaration source: %+v", findings[0].Sources)
	}
	// The operator id is the one the reference names, not a nearby one.
	if findings[0].Sources[0].Subject.ID != string(core.MustCanonicalID("stock.number_op")) {
		t.Fatalf("the source points at %q", findings[0].Sources[0].Subject.ID)
	}
}
