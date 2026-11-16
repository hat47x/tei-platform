package xeadsource

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"
)

// The expectations below are written from the legacy source reading recorded in
// plan/evidence, not from this package's implementation. Each case states what
// the source literally declares; the assertions check that no case is promoted
// to a meaning the source does not carry.

func designationTokensOf(t *testing.T, xml string, attributeName string) FieldDesignationFact {
	t.Helper()
	doc, err := Parse(FormatXead, []byte(xml))
	if err != nil {
		t.Fatal(err)
	}
	facts := ExtractFieldDesignationFacts(doc)
	for _, field := range facts.Fields {
		if field.AttributeName == attributeName {
			return field
		}
	}
	t.Fatalf("no %s fact extracted", attributeName)
	return FieldDesignationFact{}
}

func tokenKinds(tokens []DesignationToken) []string {
	out := make([]string, 0, len(tokens))
	for _, token := range tokens {
		if token.Known {
			out = append(out, string(token.Kind))
		} else {
			out = append(out, "unknown")
		}
	}
	return out
}

func tokenByKind(t *testing.T, fact FieldDesignationFact, kind DesignationKind) DesignationToken {
	t.Helper()
	for _, token := range fact.Tokens {
		if token.Kind == kind {
			return token
		}
	}
	t.Fatalf("designation %q is missing from %#v", kind, fact.Tokens)
	return DesignationToken{}
}

func hasString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func hasDiagnosticCode(token DesignationToken, want string) bool {
	for _, diagnostic := range token.Diagnostics {
		if diagnostic.Code == want {
			return true
		}
	}
	return false
}

func TestFieldDesignationSeparatesAxesForEachRecognizedKind(t *testing.T) {
	fact := designationTokensOf(t, `<System><Table><TableField ID="F1"
		TypeOptions="KUBUN(KBFLAG1),VIRTUAL,NO_EDIT"/></Table></System>`, "TypeOptions")

	if got, want := tokenKinds(fact.Tokens), []string{"kubun", "virtual", "no-edit"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("token kinds = %v, want %v", got, want)
	}

	kubun := tokenByKind(t, fact, DesignationKubun)
	// A code table reference names the code identity but its members are rows of
	// an external table, so the value domain is not settled by the model file.
	if kubun.ResolvedAgainstSource {
		t.Fatalf("KUBUN claimed to resolve its value domain from the source: %#v", kubun)
	}
	if !hasDiagnosticCode(kubun, "code-table-content-not-in-source") {
		t.Fatalf("KUBUN did not record that its table content is external: %#v", kubun)
	}
	if hasString(kubun.Informs, string(AxisUpdatability)) {
		t.Fatalf("KUBUN must not inform updatability: %#v", kubun)
	}
	if !hasString(kubun.DoesNotInform, string(AxisUpdatability)) {
		t.Fatalf("KUBUN must state that it does not inform updatability: %#v", kubun)
	}

	virtual := tokenByKind(t, fact, DesignationVirtual)
	// VIRTUAL states non-persistence but never states the derivation rule.
	if !hasString(virtual.Informs, string(AxisDerivation)) {
		t.Fatalf("VIRTUAL should inform derivation existence: %#v", virtual)
	}
	if !hasString(virtual.DoesNotInform, string(AxisValueDomain)) {
		t.Fatalf("VIRTUAL must not be read as a value domain: %#v", virtual)
	}

	noEdit := tokenByKind(t, fact, DesignationNoEdit)
	// NO_EDIT is display formatting only. The legacy editability authority is a
	// separate NoUpdate attribute, which this reference does not read.
	if !reflect.DeepEqual(noEdit.Informs, []string{string(AxisDisplayFormat)}) {
		t.Fatalf("NO_EDIT informs = %v, want only display formatting", noEdit.Informs)
	}
	if !hasString(noEdit.DoesNotInform, string(AxisUpdatability)) {
		t.Fatalf("NO_EDIT must state that it does not decide updatability: %#v", noEdit)
	}
}

func TestFieldDesignationKeepsStoredValueDistinctFromDisplayName(t *testing.T) {
	fact := designationTokensOf(t, `<System><Table><TableField ID="F1"
		TypeOptions="VALUES(001;002),BOOLEAN(T;F)"/></Table></System>`, "TypeOptions")

	values := tokenByKind(t, fact, DesignationValues)
	if values.Payload == nil || *values.Payload != "001;002" {
		t.Fatalf("VALUES payload drifted from the declaration: %#v", values.Payload)
	}
	if values.PayloadPartCount != 2 {
		t.Fatalf("VALUES part count = %d, want 2", values.PayloadPartCount)
	}
	// A VALUES list conflates the stored value with the shown text; that is a
	// structural limitation of the designation, not a parse failure.
	if !hasDiagnosticCode(values, "values-conflate-stored-value-and-display-name") {
		t.Fatalf("VALUES did not record the value/display conflation: %#v", values)
	}
	if hasString(values.Informs, string(AxisDisplayName)) {
		t.Fatalf("VALUES must not claim a separable display name: %#v", values)
	}

	boolean := tokenByKind(t, fact, DesignationBoolean)
	if boolean.Payload == nil || *boolean.Payload != "T;F" {
		t.Fatalf("BOOLEAN payload drifted: %#v", boolean.Payload)
	}
	if boolean.PayloadPartCount != 2 {
		t.Fatalf("BOOLEAN part count = %d, want 2", boolean.PayloadPartCount)
	}
	if hasString(boolean.Informs, string(AxisUpdatability)) {
		t.Fatalf("BOOLEAN must not inform updatability: %#v", boolean)
	}
}

func TestFieldDesignationSeparatesValueDomainFromPhysicalType(t *testing.T) {
	fact := designationTokensOf(t, `<System><Table><TableField ID="F1"
		TypeOptions="VALUES(1;2;3)"/></Table></System>`, "TypeOptions")

	values := tokenByKind(t, fact, DesignationValues)
	// The same physical type can carry different units; VALUES says nothing
	// about the physical column type or its scale.
	for _, axis := range []SeparationAxis{AxisPhysicalType, AxisDerivation, AxisPersistence, AxisUpdatability} {
		if hasString(values.Informs, string(axis)) {
			t.Fatalf("VALUES wrongly informed %q: %#v", axis, values.Informs)
		}
		if !hasString(values.DoesNotInform, string(axis)) {
			t.Fatalf("VALUES must state that it does not inform %q: %#v", axis, values.DoesNotInform)
		}
	}
}

func TestFieldDesignationKeepsLeadingZeroCodesAsDeclared(t *testing.T) {
	// The legacy reader sanitises numeric fields with getStringNumber, which
	// drops every character outside [-0-9.] and re-parses PERCENT values as
	// floats. A code such as 001 therefore loses its leading zeros when read as
	// a number. This reference must carry the declared text unchanged.
	fact := designationTokensOf(t, `<System><Table><TableField ID="F1"
		TypeOptions="VALUES(001;002;010),NO_EDIT"/></Table></System>`, "TypeOptions")

	values := tokenByKind(t, fact, DesignationValues)
	if values.Payload == nil || *values.Payload != "001;002;010" {
		t.Fatalf("leading zeros were not preserved: %#v", values.Payload)
	}
	for _, token := range fact.Tokens {
		if token.Raw != "VALUES(001;002;010)" && token.Raw != "NO_EDIT" {
			t.Fatalf("token raw text was rewritten: %q", token.Raw)
		}
	}
	if fact.Raw != "VALUES(001;002;010),NO_EDIT" {
		t.Fatalf("raw designation text was rewritten: %q", fact.Raw)
	}
}

func TestFieldDesignationKeepsUndefinedValueUnresolved(t *testing.T) {
	fact := designationTokensOf(t, `<System><Table><TableField ID="F1"
		TypeOptions="SOMETHING_NEW(X),AUTO_NUMBER(4)"/></Table></System>`, "TypeOptions")

	if got, want := tokenKinds(fact.Tokens), []string{"unknown", "unknown"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("token kinds = %v, want %v", got, want)
	}
	for _, token := range fact.Tokens {
		if token.Known || token.Kind != "" || len(token.Informs) != 0 || len(token.DoesNotInform) != 0 {
			t.Fatalf("unknown designation gained meaning: %#v", token)
		}
		if token.Payload != nil {
			t.Fatalf("unknown designation gained a payload: %#v", token.Payload)
		}
	}
}

func TestFieldDesignationDoesNotSplitAmbiguousConcatenation(t *testing.T) {
	// NO_EDITFYEAR may have been written with two designations in mind. The
	// legacy reader sees one token and never matches NO_EDIT, so splitting it
	// here would change behavior on a guess.
	fact := designationTokensOf(t, `<System><Table><TableField ID="F1"
		TypeOptions="NO_EDITFYEAR"/></Table></System>`, "TypeOptions")

	if len(fact.Tokens) != 1 {
		t.Fatalf("ambiguous token was split into %d tokens", len(fact.Tokens))
	}
	token := fact.Tokens[0]
	if token.Known {
		t.Fatalf("ambiguous concatenation gained a known kind: %#v", token)
	}
	if token.Raw != "NO_EDITFYEAR" {
		t.Fatalf("ambiguous token raw text changed: %q", token.Raw)
	}
	if !hasDiagnosticCode(token, "ambiguous-concatenation-not-split") {
		t.Fatalf("ambiguous concatenation was not reported: %#v", token)
	}
	if hasDiagnosticCode(token, "unrecognized-designation-token") {
		t.Fatalf("ambiguity must be distinguished from plain non-recognition: %#v", token)
	}
}

func TestFieldDesignationRecordsDistinctMultipleSpecificationsWithoutPickingAWinner(t *testing.T) {
	// KUBUN and VALUES both claim a code identity / value domain. Different
	// legacy screens inspect them in opposite order, so neither can be called
	// authoritative. The reference records both and reports the disagreement.
	fact := designationTokensOf(t, `<System><Table><TableField ID="F1"
		TypeOptions="KUBUN(KBFLAG1),VALUES(T;F)"/></Table></System>`, "TypeOptions")

	if got, want := tokenKinds(fact.Tokens), []string{"kubun", "values"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("token kinds = %v, want %v", got, want)
	}
	if len(fact.Conflicts) == 0 {
		t.Fatal("competing designations must be reported as a conflict")
	}
	conflictAxes := make([]string, 0, len(fact.Conflicts))
	for _, conflict := range fact.Conflicts {
		conflictAxes = append(conflictAxes, conflict.Axis)
		if len(conflict.Kinds) < 2 {
			t.Fatalf("a conflict must name at least two kinds: %#v", conflict)
		}
	}
	if !hasString(conflictAxes, string(AxisValueDomain)) {
		t.Fatalf("KUBUN and VALUES both inform the value domain: %v", conflictAxes)
	}
	// The reference must not silently discard either declaration.
	if len(fact.Tokens) != 2 {
		t.Fatalf("a competing declaration was dropped: %#v", fact.Tokens)
	}
}

func TestFieldDesignationReportsUnbalancedParenthesisAfterCommaSplit(t *testing.T) {
	// The legacy outer rule splits on every comma and only protects the first
	// VALUES(...) run, then discards the removal result. A comma inside another
	// payload therefore leaves two unbalanced halves, and the legacy readers
	// disagree about what remains readable.
	fact := designationTokensOf(t, `<System><Table><TableField ID="F1"
		TypeOptions="KUBUN(a,b)"/></Table></System>`, "TypeOptions")

	if len(fact.Tokens) != 2 {
		t.Fatalf("comma split produced %d tokens: %#v", len(fact.Tokens), fact.Tokens)
	}
	for _, token := range fact.Tokens {
		if token.Known {
			t.Fatalf("an unbalanced half was accepted as a designation: %#v", token)
		}
		if !hasDiagnosticCode(token, "unbalanced-parenthesis-after-comma-split") {
			t.Fatalf("unbalanced half was not reported: %#v", token)
		}
	}
}

func TestFieldDesignationIsCaseSensitiveLikeTheLegacyReaders(t *testing.T) {
	fact := designationTokensOf(t, `<System><Table><TableField ID="F1"
		TypeOptions="virtual,no_edit,kubun(x)"/></Table></System>`, "TypeOptions")

	for _, token := range fact.Tokens {
		if token.Known {
			t.Fatalf("lowercase designation was accepted although legacy matching is case-sensitive: %#v", token)
		}
	}
}

func TestFieldDesignationDistinguishesParenthesizedFromBareForms(t *testing.T) {
	fact := designationTokensOf(t, `<System><Table><TableField ID="F1"
		TypeOptions="VIRTUAL,VIRTUAL(X),NO_EDIT,NO_EDIT(X)"/></Table></System>`, "TypeOptions")

	if got, want := tokenKinds(fact.Tokens), []string{"virtual", "unknown", "no-edit", "unknown"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("token kinds = %v, want %v", got, want)
	}
	// A parenthesized form of a bare flag is a different declaration, not the
	// flag with an argument.
	for _, token := range fact.Tokens {
		if token.Kind == DesignationVirtual && token.Raw == "VIRTUAL(X)" {
			t.Fatal("parenthesized VIRTUAL was treated as the bare flag")
		}
	}
}

func TestFieldDesignationDoesNotReadForeignNamespacedAttribute(t *testing.T) {
	doc, err := Parse(FormatXead, []byte(`<System xmlns:f="urn:foreign">
		<Table><TableField ID="F1" f:TypeOptions="VIRTUAL,NO_EDIT"/></Table>
	</System>`))
	if err != nil {
		t.Fatal(err)
	}
	facts := ExtractFieldDesignationFacts(doc)
	if len(facts.Fields) != 0 {
		t.Fatalf("foreign-namespaced attribute was read as a designation: %#v", facts.Fields)
	}
}

func TestFieldDesignationReadsBothDesignationAttributes(t *testing.T) {
	doc, err := Parse(FormatXeaf, []byte(`<System><Table><Field ID="F1"
		TypeOptions="KUBUN(KBFLAG1)" FieldOptions="WIDTH(10),PROMPT_LIST"/></Table></System>`))
	if err != nil {
		t.Fatal(err)
	}
	facts := ExtractFieldDesignationFacts(doc)

	var names []string
	for _, field := range facts.Fields {
		names = append(names, field.AttributeName)
	}
	if got, want := names, []string{"FieldOptions", "TypeOptions"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("designation attributes = %v, want %v", got, want)
	}
	// FieldOptions carries display and interaction settings. None of them is a
	// stored-value declaration, so they must stay unresolved here rather than be
	// folded into the value domain.
	for _, field := range facts.Fields {
		if field.AttributeName != "FieldOptions" {
			continue
		}
		for _, token := range field.Tokens {
			if token.Known {
				t.Fatalf("FieldOptions token was promoted to a value declaration: %#v", token)
			}
		}
	}
}

func TestFieldDesignationRecordsEmptyDeclarationDistinctFromAbsence(t *testing.T) {
	doc, err := Parse(FormatXead, []byte(`<System><Table>
		<TableField ID="F1" TypeOptions=""/>
		<TableField ID="F2"/>
	</Table></System>`))
	if err != nil {
		t.Fatal(err)
	}
	facts := ExtractFieldDesignationFacts(doc)
	if len(facts.Fields) != 1 {
		t.Fatalf("fact count = %d, want 1 (only the declared attribute)", len(facts.Fields))
	}
	field := facts.Fields[0]
	if !field.EmptyDeclared {
		t.Fatalf("an explicitly empty declaration was not distinguished from absence: %#v", field)
	}
	if len(field.Tokens) != 0 {
		t.Fatalf("an empty declaration produced tokens: %#v", field.Tokens)
	}
}

func TestFieldDesignationBooleanSinglePartCollapsesEmptyAndFalse(t *testing.T) {
	fact := designationTokensOf(t, `<System><Table><TableField ID="F1"
		TypeOptions="BOOLEAN(T)"/></Table></System>`, "TypeOptions")

	boolean := tokenByKind(t, fact, DesignationBoolean)
	if boolean.PayloadPartCount != 1 {
		t.Fatalf("part count = %d, want 1", boolean.PayloadPartCount)
	}
	if !hasDiagnosticCode(boolean, "boolean-single-part-collapses-empty-and-false") {
		t.Fatalf("single-part BOOLEAN did not record the empty/false collapse: %#v", boolean)
	}
}

func TestFieldDesignationTokenizationRevisionIsRecorded(t *testing.T) {
	doc, err := Parse(FormatXead, []byte(`<System><Table><TableField ID="F1" TypeOptions="VIRTUAL"/></Table></System>`))
	if err != nil {
		t.Fatal(err)
	}
	facts := ExtractFieldDesignationFacts(doc)
	// The literal is named here so a change to the declared policy is a
	// deliberate edit to this expectation, not a silent re-interpretation of
	// already recorded evidence.
	if facts.TokenizationPolicyRevision != "tei.xeadsource.field-designation-tokenization/v1" {
		t.Fatalf("tokenization policy revision = %q", facts.TokenizationPolicyRevision)
	}
	if facts.TokenizationPolicyRevision != FieldDesignationTokenizationPolicyRevision {
		t.Fatalf("exported revision constant disagrees with the declared literal: %q",
			FieldDesignationTokenizationPolicyRevision)
	}
	if _, ok := designationSpecs[DesignationKubun]; !ok {
		t.Fatal("KUBUN spec is missing")
	}
}

// Every separation axis is a declared word, so each one must be reachable from
// at least one recognized designation — either as an axis the designation
// informs, or as an axis it explicitly refuses to inform. An axis that neither
// appears would be a vocabulary word with no producer.
func TestEverySeparationAxisIsReachableFromSomeDesignation(t *testing.T) {
	informed := map[string]bool{}
	excluded := map[string]bool{}
	for _, kind := range DesignationKinds {
		spec, ok := designationSpecs[kind]
		if !ok {
			t.Fatalf("kind %q has no spec", kind)
		}
		for _, axis := range spec.informs {
			informed[axis] = true
		}
		for _, axis := range spec.excludes {
			excluded[axis] = true
		}
	}
	for _, axis := range separationAxes {
		if !informed[string(axis)] && !excluded[string(axis)] {
			t.Fatalf("separation axis %q is declared but no designation reaches it", axis)
		}
	}
	if !informed[string(AxisCodeIdentity)] {
		t.Fatalf("%q must be informed by KUBUN", AxisCodeIdentity)
	}
	if !informed[string(AxisInputAssist)] {
		t.Fatalf("%q must be informed by KUBUN, VALUES and BOOLEAN", AxisInputAssist)
	}
	// The central finding: no recognized designation can express a display name
	// distinct from the stored value. The axis exists only so that designations
	// can refuse it explicitly.
	if informed[string(AxisDisplayName)] {
		t.Fatalf("a designation claimed to inform %q, but none of the five can",
			AxisDisplayName)
	}
	if !excluded[string(AxisDisplayName)] {
		t.Fatalf("%q must be explicitly refused by VALUES, BOOLEAN or KUBUN",
			AxisDisplayName)
	}
}

func TestDesignationKindsCoverEverySpec(t *testing.T) {
	if len(DesignationKinds) != len(designationSpecs) {
		t.Fatalf("kind list and spec table disagree: %d vs %d",
			len(DesignationKinds), len(designationSpecs))
	}
	for _, kind := range DesignationKinds {
		if _, ok := designationSpecs[kind]; !ok {
			t.Fatalf("kind %q has no spec", kind)
		}
	}
}

// The designation projection is deliberately NOT part of Inspection. Inspection
// is the payload of the versioned source-adapter capability shape, so adding a
// projection to it would change a published contract. A check keeps the two
// separate and keeps the projection's own authority limit intact.
func TestFieldDesignationStaysOutOfTheInspectionContract(t *testing.T) {
	inspection, err := Inspect(FormatXead, []byte(`<System><Table><TableField ID="F1"
		TypeOptions="KUBUN(KBFLAG1),VIRTUAL"/></Table></System>`), nil)
	if err != nil {
		t.Fatal(err)
	}
	// Inspection must still encode exactly the shape it declares.
	encoded, err := json.Marshal(inspection)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(encoded, []byte("fieldDesignation")) {
		t.Fatalf("the versioned inspection shape gained an undeclared property: %s", encoded)
	}

	facts := ExtractFieldDesignationFacts(inspection.Document)
	if len(facts.Fields) != 1 {
		t.Fatalf("designation projection dropped a declaration: %#v", facts)
	}
	for _, field := range facts.Fields {
		if !hasString(field.AuthorityLimits,
			"designation-classification-is-source-observation-not-accepted-semantic-meaning") {
			t.Fatalf("field designation fact lost its authority limit: %#v", field)
		}
	}
}

func TestFieldDesignationReportsArgumentFormWithoutValues(t *testing.T) {
	// An undefined value is most likely written as a declared argument form with
	// nothing inside it. It must not read as a known declaration with an empty
	// value domain.
	fact := designationTokensOf(t, `<System><Table><TableField ID="F1"
		TypeOptions="VALUES(),BOOLEAN(),KUBUN()"/></Table></System>`, "TypeOptions")

	if got, want := tokenKinds(fact.Tokens), []string{"values", "boolean", "kubun"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("token kinds = %v, want %v", got, want)
	}
	for _, token := range fact.Tokens {
		if token.PayloadPartCount != 0 {
			t.Fatalf("empty payload reported parts: %#v", token)
		}
		if token.Kind == DesignationKubun {
			continue
		}
		if token.ResolvedAgainstSource {
			t.Fatalf("empty payload claimed a resolved value domain: %#v", token)
		}
		if !hasDiagnosticCode(token, "payload-declared-without-parts") {
			t.Fatalf("empty payload was not reported: %#v", token)
		}
	}
}

func TestFieldDesignationTreatsMisorderedParenthesesAsBroken(t *testing.T) {
	// ")(" has one of each parenthesis, so counting alone would call it balanced
	// and then report it as a plain unrecognized token, hiding that the token is
	// structurally broken.
	fact := designationTokensOf(t, `<System><Table><TableField ID="F1"
		TypeOptions=")(,KUBUN(a))"/></Table></System>`, "TypeOptions")

	if len(fact.Tokens) != 2 {
		t.Fatalf("token count = %d", len(fact.Tokens))
	}
	for _, token := range fact.Tokens {
		if token.Known {
			t.Fatalf("a malformed token was accepted: %#v", token)
		}
		if !hasDiagnosticCode(token, "unbalanced-parenthesis-after-comma-split") {
			t.Fatalf("misordered parentheses were not reported: %#v", token)
		}
	}
}
