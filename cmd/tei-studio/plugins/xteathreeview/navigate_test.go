package xteathreeview

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"tei/internal/operations"
)

const fixtureSource = "design/order.xead"

func fixtureRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("testdata", "workspace"))
	if err != nil {
		t.Fatal(err)
	}
	return root
}

// scratchWorkspace writes one synthetic source into a new Workspace.
func scratchWorkspace(t *testing.T, source string) string {
	t.Helper()
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "design"), 0755); err != nil {
		t.Fatal(err)
	}
	manifest := "workspace: scratch\nformatVersion: 1\nsources:\n  - design\n"
	if err := os.WriteFile(filepath.Join(root, "tei.workspace.yaml"), []byte(manifest), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "design", "order.xead"), []byte(source), 0644); err != nil {
		t.Fatal(err)
	}
	return root
}

func navigate(t *testing.T, root string, pin *operations.ThreeViewPin) operations.ThreeViewNavigation {
	t.Helper()
	request := operations.ThreeViewRequest{SourcePath: fixtureSource, Pin: pin}
	value, err := Navigate(root, request)
	if err != nil {
		t.Fatalf("Navigate failed: %v", err)
	}
	if err := operations.ValidateThreeViewNavigation(request, value); err != nil {
		t.Fatalf("the plugin's answer does not satisfy the Operation contract: %v", err)
	}
	return value
}

func subjectByLocator(t *testing.T, value operations.ThreeViewNavigation, locator string) operations.ThreeViewSubject {
	t.Helper()
	for _, subject := range value.Subjects {
		if subject.SourceFact.Locator == locator {
			return subject
		}
	}
	t.Fatalf("no subject at %s", locator)
	return operations.ThreeViewSubject{}
}

// The locators below are written by hand from the fixture's element positions.
const (
	orderLocator    = "/System[1]/Table[1]"
	functionLocator = "/System[1]/Function[2]"
	taskLocator     = "/System[1]/Task[3]"
	actionLocator   = "/System[1]/Task[3]/TaskAction[1]"
	orderNoLocator  = "/System[1]/Table[1]/TableField[1]"
)

// One Data, one Function and one Business subject: from the Business view to
// the function-level CRUD row and from there to the source position, always
// through the same occurrences.
func TestThreeViewsToCRUDToSource(t *testing.T) {
	value := navigate(t, fixtureRoot(t), nil)
	if value.SchemaVersion != "tei.operations.three-view-navigation/v0" || value.Effect != "read-only" {
		t.Fatalf("unexpected schema or effect: %s %s", value.SchemaVersion, value.Effect)
	}
	data := subjectByLocator(t, value, orderLocator)
	function := subjectByLocator(t, value, functionLocator)
	task := subjectByLocator(t, value, taskLocator)
	action := subjectByLocator(t, value, actionLocator)
	if data.View != operations.ThreeViewData || function.View != operations.ThreeViewFunction ||
		task.View != operations.ThreeViewBusiness || action.View != operations.ThreeViewBusiness {
		t.Fatalf("subjects are in the wrong views: %+v", value.Subjects)
	}
	if len(value.Subjects) != 4 || action.ParentOccurrence != task.Occurrence {
		t.Fatalf("want 4 subjects with the action under its task, got %+v", value.Subjects)
	}

	find := func(kind, from, to string) operations.ThreeViewRelation {
		t.Helper()
		var found []operations.ThreeViewRelation
		for _, relation := range value.Relations {
			if relation.Kind == kind && relation.From == from && relation.To == to {
				found = append(found, relation)
			}
		}
		if len(found) != 1 {
			t.Fatalf("want exactly one %s relation, got %d", kind, len(found))
		}
		return found[0]
	}
	// Business view: task -> action -> function, as the source states it.
	contains := find("business-contains-action", task.Occurrence, action.Occurrence)
	uses := find("action-uses-function", action.Occurrence, function.Occurrence)
	// Function view: the function's own declaration over the data subject.
	declared := find("function-data-operation", function.Occurrence, data.Occurrence)
	for _, relation := range []operations.ThreeViewRelation{contains, uses, declared} {
		if relation.Nature != operations.ThreeViewNatureSourceFact {
			t.Fatalf("%s must be a source fact: %+v", relation.Kind, relation)
		}
	}
	if declared.Operations != "CR" {
		t.Fatalf("declared operations = %q, want CR", declared.Operations)
	}

	// CRUD: the same function and data occurrences, resting on that relation.
	if len(value.CRUD) != 1 {
		t.Fatalf("want one CRUD row, got %+v", value.CRUD)
	}
	row := value.CRUD[0]
	want := operations.ThreeViewCRUDRow{
		Function: function.Occurrence, FunctionName: "Order entry",
		Data: data.Occurrence, DataName: "Order",
		Create: true, Read: true,
		Scope:     operations.ThreeViewCRUDScopeFunctionDeclared,
		Relations: []string{declared.Occurrence},
	}
	if !reflect.DeepEqual(row, want) {
		t.Fatalf("CRUD row\n got %+v\nwant %+v", row, want)
	}

	// Source: every occurrence the relation rests on is one the projection
	// lists or one of its two ends, and each end has a source position.
	if data.SourceFact.Kind != "Table" || data.SourceFact.SourceID != "T-ORDER" ||
		function.SourceFact.Kind != "Function" || function.SourceFact.SourceID != "FN-ORDER-ENTRY" {
		t.Fatalf("source facts do not lead back to the source: %+v %+v", data.SourceFact, function.SourceFact)
	}
	if !contains3(declared.SourceOccurrences, function.Occurrence, data.Occurrence) {
		t.Fatalf("the CRUD relation does not rest on its own ends: %+v", declared.SourceOccurrences)
	}
	if value.Source.Path != fixtureSource || !strings.HasPrefix(value.Source.ArtifactDigest, "sha256:") {
		t.Fatalf("source binding is missing: %+v", value.Source)
	}
	// The reading rules are identified by this plugin's revision together with
	// the reader's, so a change of either makes an earlier projection stale.
	if Revision != "tei.plugin.xtea-three-view/v0" ||
		!strings.HasPrefix(value.Source.DerivationRevision, Revision+"+") ||
		value.Source.MappingFingerprint == "" {
		t.Fatalf("the projection does not name its reading rules: %+v", value.Source)
	}
	if value.Source.Coverage.Status != operations.ThreeViewCoverageComplete {
		t.Fatalf("the fixture is fully placed, coverage = %+v", value.Source.Coverage)
	}
}

func contains3(all []string, want ...string) bool {
	seen := map[string]bool{}
	for _, item := range all {
		seen[item] = true
	}
	for _, item := range want {
		if !seen[item] {
			return false
		}
	}
	return true
}

// The three layers stay apart: the source states the element, the plugin
// places it in a view as a candidate, and no meaning is accepted.
func TestSourceFactCandidateAndMeaningAreSeparate(t *testing.T) {
	value := navigate(t, fixtureRoot(t), nil)
	for _, subject := range value.Subjects {
		if subject.Candidate.Nature != operations.ThreeViewNatureStructuralCandidate ||
			len(subject.Candidate.AuthorityLimits) == 0 {
			t.Fatalf("view placement is not a limited candidate: %+v", subject)
		}
		if subject.Meaning.Status != operations.ThreeViewMeaningNotBound {
			t.Fatalf("a meaning was reported for %s", subject.SourceFact.Locator)
		}
	}
	// A provider that labels a subject accepted is refused by the contract.
	forged := value
	forged.Subjects = append([]operations.ThreeViewSubject(nil), value.Subjects...)
	forged.Subjects[0].Meaning.Status = "accepted"
	if err := operations.ValidateThreeViewNavigation(operations.ThreeViewRequest{SourcePath: fixtureSource}, forged); err == nil {
		t.Fatal("an accepted meaning passed the v0 contract")
	}
}

// The path action -> function -> data is derived. It is shown as a candidate
// with its rule and is not turned into a CRUD row of the business action.
func TestActionToDataIsADerivedCandidateNotActionCRUD(t *testing.T) {
	value := navigate(t, fixtureRoot(t), nil)
	action := subjectByLocator(t, value, actionLocator)
	data := subjectByLocator(t, value, orderLocator)
	var derived []operations.ThreeViewRelation
	for _, relation := range value.Relations {
		if relation.From == action.Occurrence && relation.To == data.Occurrence {
			derived = append(derived, relation)
		}
	}
	if len(derived) != 1 || derived[0].Kind != "action-data-potential-operation" ||
		derived[0].Nature != operations.ThreeViewNatureStructuralCandidate ||
		derived[0].DerivationRuleFingerprint == "" {
		t.Fatalf("want one derived candidate path, got %+v", derived)
	}
	for _, row := range value.CRUD {
		if row.Function == action.Occurrence {
			t.Fatalf("a CRUD row was attributed to the business action: %+v", row)
		}
	}
}

// The action has no source ID. It stays addressable as an occurrence and its
// ID is not made up from its position or its name.
func TestActionWithoutSourceIDIsNotGivenOne(t *testing.T) {
	value := navigate(t, fixtureRoot(t), nil)
	action := subjectByLocator(t, value, actionLocator)
	if action.SourceFact.SourceIDStatus != operations.ThreeViewSourceIDMissing || action.SourceFact.SourceID != "" {
		t.Fatalf("an ID was supplied for the action: %+v", action.SourceFact)
	}
	if action.Occurrence == "" {
		t.Fatal("the action is not addressable")
	}
}

// Item-level reference: the key member refers to exactly one field.
func TestKeyMemberRefersToItsField(t *testing.T) {
	value := navigate(t, fixtureRoot(t), nil)
	data := subjectByLocator(t, value, orderLocator)
	if len(value.Fields) != 2 || len(value.FieldReferences) != 1 {
		t.Fatalf("want 2 fields and 1 reference, got %+v %+v", value.Fields, value.FieldReferences)
	}
	reference := value.FieldReferences[0]
	var orderNo operations.ThreeViewField
	for _, field := range value.Fields {
		if field.SourceFact.Locator == orderNoLocator {
			orderNo = field
		}
	}
	if reference.Owner != data.Occurrence || reference.Field != orderNo.Occurrence ||
		reference.Resolution != "resolved" || reference.ReferencedID != "FLD-ORDER-NO" {
		t.Fatalf("the key member does not lead to OrderNo: %+v", reference)
	}
}

const sameNameSource = `<System FormatVersion="1.2">
  <Table ID="T1" Name="Order"/>
  <Function ID="F1" Name="Order entry"><IOTable ID="IO1" TableID="T1" OpR="+"/></Function>
  <Function ID="F2" Name="Order entry"><IOTable ID="IO1" TableID="T1" OpU="+"/></Function>
</System>`

// Two functions with one name and different IDs are two occurrences. Each
// keeps its own CRUD row; the name joins nothing.
func TestSameNameDifferentIDStaysTwoSubjects(t *testing.T) {
	value := navigate(t, scratchWorkspace(t, sameNameSource), nil)
	first := subjectByLocator(t, value, "/System[1]/Function[2]")
	second := subjectByLocator(t, value, "/System[1]/Function[3]")
	if first.Occurrence == second.Occurrence ||
		!reflect.DeepEqual(first.SameNameOccurrences, []string{second.Occurrence}) ||
		!reflect.DeepEqual(second.SameNameOccurrences, []string{first.Occurrence}) {
		t.Fatalf("same-name functions were not kept apart: %+v %+v", first, second)
	}
	rows := map[string]operations.ThreeViewCRUDRow{}
	for _, row := range value.CRUD {
		rows[row.Function] = row
	}
	if len(rows) != 2 || !rows[first.Occurrence].Read || rows[first.Occurrence].Update ||
		!rows[second.Occurrence].Update || rows[second.Occurrence].Read {
		t.Fatalf("CRUD rows were merged by name: %+v", value.CRUD)
	}
}

const duplicatedIDSource = `<System FormatVersion="1.2">
  <Table ID="T1" Name="Order"/>
  <Table ID="T1" Name="Order archive"/>
  <Function ID="F1" Name="Order entry"><IOTable ID="IO1" TableID="T1" OpU="+"/></Function>
</System>`

// Two tables carry the ID the function refers to. The relation is reported
// unresolved and no CRUD row is drawn to either table.
func TestDuplicatedSourceIDIsNotResolvedToEither(t *testing.T) {
	value := navigate(t, scratchWorkspace(t, duplicatedIDSource), nil)
	if len(value.CRUD) != 0 {
		t.Fatalf("a CRUD row was drawn to an ambiguous table: %+v", value.CRUD)
	}
	if len(value.Unresolved) != 1 || value.Unresolved[0].Code != "iotable-table-ambiguous" {
		t.Fatalf("want one ambiguous relation, got %+v", value.Unresolved)
	}
	for _, locator := range []string{"/System[1]/Table[1]", "/System[1]/Table[2]"} {
		if subjectByLocator(t, value, locator).SourceFact.SourceIDStatus != operations.ThreeViewSourceIDDuplicated {
			t.Fatalf("%s is not reported as a duplicated ID", locator)
		}
	}
	if value.Source.Coverage.Status != operations.ThreeViewCoveragePartial ||
		value.Source.Coverage.UnresolvedRelations != 1 {
		t.Fatalf("an unresolved relation left the coverage complete: %+v", value.Source.Coverage)
	}
}

const unplacedSource = `<System FormatVersion="1.2">
  <Table ID="T1" Name="Order"/>
  <Subsystem ID="S1" Name="Sales"/>
  <Private ID="X1"/>
</System>`

// A construct the rules do not place and a construct the reader does not know
// are counted, so a short list is not read as a complete one.
func TestUnplacedAndUnknownConstructsMakeCoveragePartial(t *testing.T) {
	value := navigate(t, scratchWorkspace(t, unplacedSource), nil)
	coverage := value.Source.Coverage
	if coverage.Status != operations.ThreeViewCoveragePartial ||
		coverage.UnplacedConstructs != 1 || coverage.UnknownConstructs != 1 {
		t.Fatalf("coverage = %+v", coverage)
	}
}

// A pin is the caller's. It comes back unchanged and held while the source is
// the same, whatever else is looked at; after the source changes it is stale
// and is not moved to the element that now has the same ID and name.
func TestPinIsHeldUntilTheSourceChangesAndIsNeverRebound(t *testing.T) {
	root := scratchWorkspace(t, sameNameSource)
	before := navigate(t, root, nil)
	pinned := subjectByLocator(t, before, "/System[1]/Function[2]")
	pin := &operations.ThreeViewPin{
		Occurrence: pinned.Occurrence, SourceArtifactDigest: before.Source.ArtifactDigest,
	}
	held := navigate(t, root, pin)
	if held.Pin == nil || held.Pin.ThreeViewPin != *pin || held.Pin.Status != operations.ThreeViewPinHeld {
		t.Fatalf("the pin was not held: %+v", held.Pin)
	}

	changed := strings.Replace(sameNameSource, `Name="Order"/>`, `Name="Order"/>
  <Table ID="T9" Name="Inserted"/>`, 1)
	if err := os.WriteFile(filepath.Join(root, "design", "order.xead"), []byte(changed), 0644); err != nil {
		t.Fatal(err)
	}
	after := navigate(t, root, pin)
	if after.Pin == nil || after.Pin.ThreeViewPin != *pin ||
		after.Pin.Status != operations.ThreeViewPinStaleSourceChanged {
		t.Fatalf("a pin on changed bytes was not reported stale: %+v", after.Pin)
	}
	// The function with the same ID and name still exists, as a new occurrence.
	moved := subjectByLocator(t, after, "/System[1]/Function[3]")
	if moved.SourceFact.SourceID != "F1" || moved.Occurrence == pin.Occurrence {
		t.Fatalf("the fixture no longer exercises rebinding: %+v", moved)
	}

	// A pin that names no subject of the same bytes is reported, not replaced.
	missing := &operations.ThreeViewPin{Occurrence: "sha256:0", SourceArtifactDigest: after.Source.ArtifactDigest}
	if got := navigate(t, root, missing); got.Pin.Status != operations.ThreeViewPinOccurrenceNotFound {
		t.Fatalf("an unknown pin was reported as %s", got.Pin.Status)
	}
}

func TestRefusedSourcesReturnTypedCodes(t *testing.T) {
	root := scratchWorkspace(t, `<System FormatVersion="9.9"/>`)
	if err := os.WriteFile(filepath.Join(root, "design", "note.txt"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	large := scratchWorkspace(t, `<System FormatVersion="1.2">`+strings.Repeat(" ", int(SourceLimitBytes))+`</System>`)
	cases := []struct {
		name, root, path, want string
	}{
		{"unsupported version", root, fixtureSource, "THREE_VIEW_SOURCE_REJECTED"},
		{"not an X-TEA design source", root, "design/note.txt", "THREE_VIEW_UNSUPPORTED_SOURCE"},
		{"missing", root, "design/absent.xead", "THREE_VIEW_SOURCE_READ_FAILED"},
		{"escaping path", root, "../order.xead", "THREE_VIEW_SOURCE_READ_FAILED"},
		{"too large", large, fixtureSource, "THREE_VIEW_SOURCE_TOO_LARGE"},
	}
	for _, c := range cases {
		_, err := Navigate(c.root, operations.ThreeViewRequest{SourcePath: c.path})
		var typed *operations.ThreeViewError
		if !errors.As(err, &typed) || typed.Code != c.want {
			t.Errorf("%s: got %v, want %s", c.name, err, c.want)
		}
	}
	// THREE_VIEW_PROJECTION_FAILED is the code for a projection the reader
	// could not build from an accepted source; no fixture here produces it.
	if (&operations.ThreeViewError{Code: operations.ThreeViewProjectionFailed}).Error() != "THREE_VIEW_PROJECTION_FAILED" {
		t.Fatal("projection failure code changed")
	}
}

func TestNavigateIsDeterministic(t *testing.T) {
	first := navigate(t, fixtureRoot(t), nil)
	second := navigate(t, fixtureRoot(t), nil)
	if !reflect.DeepEqual(first, second) {
		t.Fatal("two reads of the same bytes differ")
	}
}
