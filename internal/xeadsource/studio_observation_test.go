package xeadsource

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func parseStudioFixture(t *testing.T, source string) Document {
	t.Helper()
	doc, err := Parse(FormatXead, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	return doc
}

func TestProjectStudioObservationProjectsDirectTablesAndFields(t *testing.T) {
	doc := parseStudioFixture(t, `<System><Table ID="T1" Name="Order"><TableField ID="F1" Name="OrderID" DataTypeID="D1"/></Table></System>`)

	got, err := ProjectStudioObservation(doc)
	if err != nil {
		t.Fatal(err)
	}
	want := StudioObservation{Tables: []StudioTable{{
		Source: StudioSourceRef{Kind: "Table", ID: "T1"}, Name: "Order",
		Fields: []StudioTableField{{
			Source: StudioSourceRef{Kind: "TableField", ID: "F1"},
			Name:   "OrderID", DataType: &StudioSourceRef{Kind: "DataType", ID: "D1"},
		}},
	}}}
	if !reflect.DeepEqual(got, want) {
		gotJSON, _ := json.Marshal(got)
		wantJSON, _ := json.Marshal(want)
		t.Fatalf("projection = %s, want %s", gotJSON, wantJSON)
	}
}

func TestProjectStudioObservationDoesNotPromoteUnsupportedOrNamespacedAncestors(t *testing.T) {
	doc := parseStudioFixture(t, `<System xmlns:x="urn:private-uri"><x:Table ID="fake-ns-id"/><Unsupported><System><Table ID="fake-nested-id"/></System></Unsupported><Table ID="T1"><Unsupported><TableField ID="fake-field-id"/></Unsupported><x:TableField ID="fake-ns-field-id"/><TableField ID="F1" Name="Allowed" DataTypeID="D1"><Nested ID="fake-child-id"/></TableField></Table></System>`)

	got, err := ProjectStudioObservation(doc)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Tables) != 1 || got.Tables[0].Source.ID != "T1" {
		t.Fatalf("unsupported ancestor promoted a table: %#v", got.Tables)
	}
	if len(got.Tables[0].Fields) != 1 || got.Tables[0].Fields[0].Source.ID != "F1" {
		t.Fatalf("unsupported ancestor promoted a field: %#v", got.Tables[0].Fields)
	}
	if countDiagnostic(got.Diagnostics, "unsupported-element", "Table", "") != 2 ||
		countDiagnostic(got.Diagnostics, "unsupported-element", "TableField", "") != 2 {
		t.Fatalf("namespaced lookalikes were not diagnosed: %#v", got.Diagnostics)
	}
}

func TestProjectStudioObservationDropsSecretsAndUnsupportedValues(t *testing.T) {
	doc := parseStudioFixture(t, `<System DatabasePassword="synthetic-system-secret"><Table ID="T1" Name="PublicName"><TableField ID="F1" Name="Code" DataTypeID="D1" Default="synthetic-default-secret"/><Instance Text="synthetic-instance-secret"/><UnknownElement Private="synthetic-unknown-secret">synthetic-unknown-text</UnknownElement></Table></System>`)

	got, err := ProjectStudioObservation(doc)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	output := string(encoded)
	for _, forbidden := range []string{
		"synthetic-system-secret", "synthetic-default-secret",
		"synthetic-instance-secret", "synthetic-unknown-secret", "synthetic-unknown-text",
	} {
		if strings.Contains(output, forbidden) {
			t.Fatalf("projection retained source value %q: %s", forbidden, output)
		}
	}
	for _, required := range []string{
		`"element":"System","attribute":"DatabasePassword"`,
		`"element":"TableField","attribute":"Default"`,
		`"element":"Instance"`,
		`"element":"UnknownElement","attribute":"Private"`,
	} {
		if !strings.Contains(output, required) {
			t.Fatalf("name-only diagnostic %s missing: %s", required, output)
		}
	}
}

func TestProjectStudioObservationDoesNotEchoNamespaceURIOrNamespacedAttributeValue(t *testing.T) {
	doc := parseStudioFixture(t, `<System xmlns:x="urn:synthetic-private"><Table ID="T1" x:Name="synthetic-ns-name"><x:Unknown x:Secret="synthetic-ns-secret"/></Table></System>`)

	got, err := ProjectStudioObservation(doc)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	output := string(encoded)
	for _, forbidden := range []string{"urn:synthetic-private", "synthetic-ns-name", "synthetic-ns-secret"} {
		if strings.Contains(output, forbidden) {
			t.Fatalf("projection leaked namespace data %q: %s", forbidden, output)
		}
	}
	if countDiagnostic(got.Diagnostics, "unsupported-attribute", "Table", "Name") != 1 {
		t.Fatalf("namespaced Name was not treated as unsupported: %#v", got.Diagnostics)
	}
}

func TestProjectStudioObservationValidatesFormatAndRoot(t *testing.T) {
	doc := parseStudioFixture(t, `<System/>`)
	doc.Format = FormatXeaf
	if _, err := ProjectStudioObservation(doc); err != ErrStudioObservationRequiresXead {
		t.Fatalf("format error = %v", err)
	}
	doc.Format = FormatXead
	doc.Root.Name = "OtherRoot"
	if _, err := ProjectStudioObservation(doc); err != ErrStudioObservationRequiresSystemRoot {
		t.Fatalf("root error = %v", err)
	}
}

func countDiagnostic(diagnostics []StudioDiagnostic, code, element, attribute string) int {
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == code && diagnostic.Element == element && diagnostic.Attribute == attribute {
			return diagnostic.Occurrences
		}
	}
	return 0
}
