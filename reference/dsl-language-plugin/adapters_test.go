package languagereference

import (
	"encoding/json"
	"reflect"
	"testing"

	"tei/internal/semantic"
	"tei/internal/syntax"
)

// This is a test-only alternate language; it does not register JSON as an accepted DSL.
func TestLegacyParserAndJSONAdapter(t *testing.T) {
	teiPin := testPin()
	jsonPin := Selection{LanguageID: "reference.dsl.json", ContractVersion: "0.1.0", ContractFingerprint: "json-v0", ImplementationID: "reference.json", ImplementationFingerprint: "parser-v0"}
	providers := []Provider[semantic.Namespace]{
		{Selection: teiPin, Parse: func(s string) (semantic.Namespace, error) { return syntax.NewParser(s).ParseSemanticFile() }},
		{Selection: jsonPin, Parse: func(s string) (semantic.Namespace, error) {
			var v semantic.Namespace
			e := json.Unmarshal([]byte(s), &v)
			return v, e
		}},
	}
	source := "namespace sales\nvalidation sales.order_line.quantity_positive positive { attribute sales.order_line.quantity }"
	alternate := "{\"id\":\"sales\",\"validations\":[{\"id\":\"sales.order_line.quantity_positive\",\"symbol\":\"quantity_positive\",\"kind\":\"positive\",\"attribute\":\"sales.order_line.quantity\"}]}"
	a, err := ParseSelected(teiPin, providers, source)
	if err != nil {
		t.Fatal(err)
	}
	b, err := ParseSelected(jsonPin, providers, alternate)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(a, b) {
		t.Fatalf("different semantic candidates: %#v vs %#v", a, b)
	}
}

func TestSourcePositionPreserved(t *testing.T) {
	pin := testPin()
	providers := []Provider[semantic.Namespace]{{Selection: pin, Parse: func(s string) (semantic.Namespace, error) { return syntax.NewParser(s).ParseSemanticFile() }}}
	_, err := ParseSelected(pin, providers, "namespace sales\nvalidation sales.invalid unsupported {}")
	if err == nil {
		t.Fatal("expected parsing failure")
	}
	start, _, ok := syntax.ErrorRange(err)
	if !ok || start.Line != 2 {
		t.Fatalf("location lost: %+v %v", start, err)
	}
}
