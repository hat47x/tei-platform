package threeelement

import (
	"strings"
	"testing"
)

// These expectations are written from the contract, not from DecodeModel's
// implementation: the document below is a well-formed v0 model by hand, and the
// variants differ by exactly one property each.

const canonicalModelDocument = `{
  "schemaVersion": "tei.three-element.model/v0",
  "effect": "read-only",
  "modelId": "admission",
  "name": "Admission probe",
  "dataElements": [{"id": "order", "name": "Order"}],
  "businessElements": [],
  "functionElements": [{"id": "submit", "name": "Submit", "operations": [{"dataId": "order", "operations": "R"}]}],
  "crudDefinitions": [{"id": "c1", "functionId": "submit", "dataId": "order", "read": true, "source": "declared"}]
}`

func TestDecodeModelAcceptsTheVersionThisPackageDeclares(t *testing.T) {
	model, err := DecodeModel([]byte(canonicalModelDocument))
	if err != nil {
		t.Fatalf("canonical v0 document refused: %v", err)
	}
	if model.SchemaVersion != SchemaVersion() || model.ModelID != "admission" {
		t.Fatalf("decoded the wrong model: %+v", model)
	}
	if len(model.DataElements) != 1 || len(model.FunctionElements) != 1 || len(model.CRUDDefinitions) != 1 {
		t.Fatalf("decoded model lost an element: %+v", model)
	}
}

// Before this rule existed, a document with no version was read as v0 by
// json.Unmarshal. "No version stated" and "v0 stated" are different documents,
// and the reader can only interpret the second.
func TestDecodeModelRefusesAMissingVersion(t *testing.T) {
	document := strings.Replace(canonicalModelDocument, `"schemaVersion": "tei.three-element.model/v0",`, "", 1)
	if _, err := DecodeModel([]byte(document)); err == nil {
		t.Fatal("a document with no schemaVersion was admitted")
	} else if !strings.Contains(err.Error(), "schemaVersion") {
		t.Fatalf("the refusal does not name the version: %v", err)
	}
}

// An unknown version may define meanings this package would misread. Reading it
// as v0 is the behaviour #6113 recorded: ok=true with the unknown version echoed
// back in the response.
func TestDecodeModelRefusesAnUnknownVersion(t *testing.T) {
	for _, version := range []string{
		"tei.three-element/v0",       // the literal the draft tests used to write
		"tei.three-element.model/v1", // a later version this reader cannot know
		"totally-wrong-schema/v9",    // no TEI version at all
		"",                           // stated empty
	} {
		document := strings.Replace(canonicalModelDocument, "tei.three-element.model/v0", version, 1)
		if _, err := DecodeModel([]byte(document)); err == nil {
			t.Fatalf("version %q was admitted", version)
		}
	}
}

// A field outside the contract has no defined meaning under v0. Dropping it
// silently is what made a document carrying presence/sourceSpan produce the
// same ModelFingerprint as one that never carried them (#6113).
func TestDecodeModelRefusesAFieldTheContractDoesNotDefine(t *testing.T) {
	for name, document := range map[string]string{
		"top-level field": strings.Replace(canonicalModelDocument,
			`"effect": "read-only",`, `"effect": "read-only", "presence": "not-fetched",`, 1),
		"nested field": strings.Replace(canonicalModelDocument,
			`"dataElements": [{"id": "order", "name": "Order"}]`,
			`"dataElements": [{"id": "order", "name": "Order", "sourceSpan": {"path": "x"}}]`, 1),
	} {
		if _, err := DecodeModel([]byte(document)); err == nil {
			t.Fatalf("%s was admitted; the contract does not define it", name)
		}
	}
}

// A byte-order mark is an encoding detail, not JSON text. The CLI keeps this
// rule for models a Windows editor wrote, so it must hold after the admission
// checks as well.
func TestDecodeModelAcceptsALeadingByteOrderMark(t *testing.T) {
	document := append([]byte{0xEF, 0xBB, 0xBF}, []byte(canonicalModelDocument)...)
	model, err := DecodeModel(document)
	if err != nil {
		t.Fatalf("a BOM-only difference was refused: %v", err)
	}
	if model.ModelID != "admission" {
		t.Fatalf("decoded the wrong model: %+v", model)
	}
}

// Two documents JSON cannot tell apart by meaning must not both be accepted:
// one that says two things is refused rather than read as the first one.
func TestDecodeModelRefusesTrailingContent(t *testing.T) {
	document := []byte(canonicalModelDocument + "\n{\"schemaVersion\": \"tei.three-element.model/v0\"}")
	if _, err := DecodeModel(document); err == nil {
		t.Fatal("trailing document was ignored")
	}
}

// A document that is refused must not become a ModelFingerprint that a
// same-meaning admitted document also has. The pair below differs only by an
// undefined field, which the old reader dropped.
func TestDecodeModelRefusalKeepsUndefinedMeaningOutOfTheFingerprint(t *testing.T) {
	withExtra := []byte(strings.Replace(canonicalModelDocument,
		`"effect": "read-only",`, `"effect": "read-only", "sourceRevision": "abc",`, 1))

	plain, err := DecodeModel([]byte(canonicalModelDocument))
	if err != nil {
		t.Fatal(err)
	}
	// The extra document is refused, so there is no second fingerprint to
	// compare: the point is that the reader never produces one.
	if _, err := DecodeModel(withExtra); err == nil {
		t.Fatal("a document with an undefined field produced a model")
	}
	if len(plain.DataElements) != 1 {
		t.Fatalf("the admitted document changed shape: %+v", plain)
	}
}
