package teicodec_test

import (
	"strings"
	"testing"

	"tei/internal/formatversion"
	"tei/internal/formatversion/teicodec"
)

const semanticSource = `namespace sales

data Order {
    attribute quantity type std.decimal value "2"
    attribute note type std.text value ""
    attribute status type std.code
}
`

func registry(t *testing.T) *formatversion.Registry {
	t.Helper()
	registry := formatversion.NewRegistry()
	if err := registry.Register(teicodec.New(1, false)); err != nil {
		t.Fatal(err)
	}
	if err := registry.Register(teicodec.New(2, true)); err != nil {
		t.Fatal(err)
	}
	return registry
}

func hasCode(result formatversion.Result, code string) bool {
	for _, diagnostic := range result.Diagnostics {
		if diagnostic.Code == code {
			return true
		}
	}
	return false
}

func centralAttribute(t *testing.T, document formatversion.Document, elementID, attributeName string) (formatversion.Attribute, bool) {
	t.Helper()
	for _, element := range document.Elements {
		if element.ID != elementID {
			continue
		}
		for _, attribute := range element.Attributes {
			if attribute.Name == attributeName {
				return attribute, true
			}
		}
	}
	return formatversion.Attribute{}, false
}

func TestV1ToV2MaterializesTheInheritedVersion(t *testing.T) {
	result, err := registry(t).Convert([]byte(semanticSource), 1, 2)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != formatversion.SuccessWithWarnings {
		t.Fatalf("status=%v diagnostics=%+v", result.Status, result.Diagnostics)
	}
	if !hasCode(result, teicodec.CodeMaterializedDefault) {
		t.Fatalf("materialized-default warning missing: %+v", result.Diagnostics)
	}
	text := string(result.Output)
	if !strings.HasPrefix(text, "formatVersion 2\n") {
		t.Fatalf("v2 output does not declare its version:\n%s", text)
	}
	for _, diagnostic := range result.Diagnostics {
		if diagnostic.ElementID == "" {
			t.Fatalf("diagnostic without element id: %+v", diagnostic)
		}
	}
}

func TestV1ToV2PreservesTheThreeAttributeStates(t *testing.T) {
	result, err := registry(t).Convert([]byte(semanticSource), 1, 2)
	if err != nil {
		t.Fatal(err)
	}
	if !result.OK() {
		t.Fatalf("conversion failed: %+v", result)
	}
	if got, ok := centralAttribute(t, result.Document, "sales.order.quantity", "value"); !ok || got.State != formatversion.Value || got.Value != "2" {
		t.Fatalf("quantity value=%+v ok=%v", got, ok)
	}
	if got, ok := centralAttribute(t, result.Document, "sales.order.note", "value"); !ok || got.State != formatversion.Empty || got.Value != "" {
		t.Fatalf("note value=%+v ok=%v", got, ok)
	}
	if _, ok := centralAttribute(t, result.Document, "sales.order.status", "value"); ok {
		t.Fatalf("absent value clause became an explicit value")
	}
}

func TestSameVersionConversionIsByteIdentity(t *testing.T) {
	result, err := registry(t).Convert([]byte(semanticSource), 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	if string(result.Output) != semanticSource {
		t.Fatalf("same-version conversion rewrote the source:\n%s", result.Output)
	}
}

func TestExplicitV1ToV2WithoutMaterializationIsVersionOnlyBump(t *testing.T) {
	source := "formatVersion 1\n" + semanticSource
	result, err := registry(t).Convert([]byte(source), 1, 2)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != formatversion.Failure || !hasCode(result, "VERSION_ONLY_BUMP") {
		t.Fatalf("version-only bump was accepted: %+v", result)
	}
}

func TestUnsupportedConstructIsRefusedWithAnElementID(t *testing.T) {
	source := "namespace sales\n\ntype OrderId from std.identifier\n"
	result, err := registry(t).Convert([]byte(source), 1, 2)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != formatversion.Failure || !hasCode(result, teicodec.CodeUnsupportedConstruct) {
		t.Fatalf("unsupported construct was accepted: %+v", result)
	}
	found := false
	for _, diagnostic := range result.Diagnostics {
		if diagnostic.Code == teicodec.CodeUnsupportedConstruct {
			if diagnostic.ElementID != "sales.order_id" {
				t.Fatalf("unsupported diagnostic element=%q", diagnostic.ElementID)
			}
			found = true
		}
	}
	if !found {
		t.Fatalf("unsupported diagnostic missing: %+v", result.Diagnostics)
	}
}

func TestDeclaredVersionMismatchIsRefused(t *testing.T) {
	source := strings.Replace(semanticSource, "namespace sales", "formatVersion 2\nnamespace sales", 1)
	result, err := registry(t).Convert([]byte(source), 1, 2)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != formatversion.Failure || !hasCode(result, "VERSION_MISMATCH") {
		t.Fatalf("declared/requested mismatch was accepted: %+v", result)
	}
}

func TestV1ToV2PreservesAttributeModifiers(t *testing.T) {
	source := `namespace sales

data Order {
    attribute quantity type std.decimal required immutable role key description "数量" value "2"
}
`
	result, err := registry(t).Convert([]byte(source), 1, 2)
	if err != nil {
		t.Fatal(err)
	}
	if !result.OK() {
		t.Fatalf("conversion failed: %+v", result)
	}
	elementID := "sales.order.quantity"
	for _, name := range []string{"type", "required", "immutable", "role", "description", "value"} {
		if _, ok := centralAttribute(t, result.Document, elementID, name); !ok {
			t.Fatalf("attribute %q was lost: %+v", name, result.Document.Elements)
		}
	}
}

func TestV2OutputIsCanonicalAndIdempotent(t *testing.T) {
	first, err := registry(t).Convert([]byte(semanticSource), 1, 2)
	if err != nil || !first.OK() {
		t.Fatalf("conversion failed: %+v err=%v", first, err)
	}
	second, err := registry(t).Convert(first.Output, 2, 2)
	if err != nil {
		t.Fatal(err)
	}
	if second.Status != formatversion.Success {
		t.Fatalf("v2 identity status=%v diagnostics=%+v", second.Status, second.Diagnostics)
	}
	if string(second.Output) != string(first.Output) {
		t.Fatalf("v2 identity changed the bytes:\nfirst:\n%s\nsecond:\n%s", first.Output, second.Output)
	}
}

// The real .tei codec is registered through the plugin entry, without the core
// knowing about .tei.
func TestTeiCodecRegistersThroughTheFormatsEntry(t *testing.T) {
	formats := formatversion.NewFormats()
	if err := formats.Register(teicodec.Format, teicodec.New(1, false)); err != nil {
		t.Fatal(err)
	}
	if err := formats.Register(teicodec.Format, teicodec.New(2, true)); err != nil {
		t.Fatal(err)
	}
	result, err := formats.Convert(teicodec.Format, []byte(semanticSource), 1, 2)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != formatversion.SuccessWithWarnings || !hasCode(result, teicodec.CodeMaterializedDefault) {
		t.Fatalf("format dispatch did not run the .tei codec: %+v", result)
	}
	if !strings.HasPrefix(string(result.Output), "formatVersion 2\n") {
		t.Fatalf("v2 output does not declare its version:\n%s", result.Output)
	}
}
