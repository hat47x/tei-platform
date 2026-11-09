package formatversion_test

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"testing"

	"tei/internal/formatversion"
)

// textCodec is a small stand-in for a plugin codec. It keeps the same central
// representation but can rename or drop attributes to simulate a version step.
type textCodec struct {
	version      formatversion.Version
	rename       map[string]string
	drop         map[string]bool
	dropSilently bool
	noop         bool
}

func (c textCodec) Version() formatversion.Version { return c.version }

func (c textCodec) Decode(source []byte) (formatversion.Document, []formatversion.Diagnostic, error) {
	document := formatversion.Document{}
	var current *formatversion.Element
	for _, line := range strings.Split(strings.TrimSpace(string(source)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		switch fields[0] {
		case "formatVersion":
			version, err := strconv.Atoi(fields[1])
			if err != nil {
				return document, nil, err
			}
			document.Declared = formatversion.Version(version)
		case "element":
			document.Elements = append(document.Elements, formatversion.Element{ID: fields[1], Kind: fields[2]})
			current = &document.Elements[len(document.Elements)-1]
		case "attribute":
			if current == nil {
				return document, nil, fmt.Errorf("attribute before element")
			}
			if len(fields) < 4 {
				return document, nil, fmt.Errorf("malformed attribute line %q", line)
			}
			state := formatversion.Absent
			switch fields[2] {
			case "absent":
				state = formatversion.Absent
			case "empty":
				state = formatversion.Empty
			case "value":
				state = formatversion.Value
			default:
				return document, nil, fmt.Errorf("unknown state %q", fields[2])
			}
			value := ""
			if len(fields) > 3 && fields[3] != "-" {
				value = fields[3]
			}
			current.Attributes = append(current.Attributes, formatversion.Attribute{Name: fields[1], State: state, Value: value})
		}
	}
	return document, nil, nil
}

func (c textCodec) Encode(document formatversion.Document) ([]byte, []formatversion.Diagnostic, error) {
	if c.noop {
		c.rename = nil
		c.drop = nil
	}
	var builder strings.Builder
	fmt.Fprintf(&builder, "formatVersion %d\n", c.version)
	var diagnostics []formatversion.Diagnostic
	elements := append([]formatversion.Element(nil), document.Elements...)
	sort.Slice(elements, func(i, j int) bool { return elements[i].ID < elements[j].ID })
	for _, element := range elements {
		fmt.Fprintf(&builder, "element %s %s\n", element.ID, element.Kind)
		attributes := append([]formatversion.Attribute(nil), element.Attributes...)
		sort.Slice(attributes, func(i, j int) bool { return attributes[i].Name < attributes[j].Name })
		for _, attribute := range attributes {
			name := attribute.Name
			if replacement, ok := c.rename[attribute.Name]; ok {
				name = replacement
				diagnostics = append(diagnostics, formatversion.Diagnostic{
					Severity:  formatversion.SeverityWarning,
					Code:      "ATTRIBUTE_RENAMED",
					ElementID: element.ID,
					Attribute: attribute.Name,
					Message:   fmt.Sprintf("attribute %q was renamed to %q", attribute.Name, name),
				})
			}
			if c.drop[attribute.Name] {
				if !c.dropSilently {
					diagnostics = append(diagnostics, formatversion.Diagnostic{
						Severity:  formatversion.SeverityWarning,
						Code:      "ATTRIBUTE_DEPRECATED",
						ElementID: element.ID,
						Attribute: attribute.Name,
						Message:   fmt.Sprintf("attribute %q is deprecated in version %d", attribute.Name, c.version),
					})
				}
				continue
			}
			value := attribute.Value
			if value == "" {
				value = "-"
			}
			fmt.Fprintf(&builder, "attribute %s %s %s\n", name, attribute.State, value)
		}
	}
	return []byte(builder.String()), diagnostics, nil
}

func newRegistry(t *testing.T, codecs ...formatversion.Codec) *formatversion.Registry {
	t.Helper()
	registry := formatversion.NewRegistry()
	for _, codec := range codecs {
		if err := registry.Register(codec); err != nil {
			t.Fatal(err)
		}
	}
	return registry
}

const v1Source = `formatVersion 1
element sales.order Order
attribute quantity value 2
attribute note empty -
attribute status absent -
`

func hasCode(t *testing.T, result formatversion.Result, code string) bool {
	t.Helper()
	for _, diagnostic := range result.Diagnostics {
		if diagnostic.Code == code {
			return true
		}
	}
	return false
}

func TestSameVersionIsIdentity(t *testing.T) {
	registry := newRegistry(t, textCodec{version: 1})
	result, err := registry.Convert([]byte(v1Source), 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != formatversion.Success || !result.OK() {
		t.Fatalf("status=%v", result.Status)
	}
	if string(result.Output) != v1Source {
		t.Fatalf("identity conversion rewrote the source:\n%s", result.Output)
	}
	// Idempotency: a second conversion yields the same bytes.
	again, err := registry.Convert(result.Output, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	if string(again.Output) != string(result.Output) {
		t.Fatalf("second identity conversion changed the bytes")
	}
}

func TestConversionPreservesTheThreeAttributeStates(t *testing.T) {
	// A real version step renames one attribute, so the conversion is not a
	// version-only bump. The other attributes keep their three-valued state.
	registry := newRegistry(t, textCodec{version: 1}, textCodec{version: 2, rename: map[string]string{"status": "state"}})
	result, err := registry.Convert([]byte(v1Source), 1, 2)
	if err != nil {
		t.Fatal(err)
	}
	if !result.OK() {
		t.Fatalf("conversion failed: %+v", result)
	}
	attributes := map[string]formatversion.Attribute{}
	for _, element := range result.Document.Elements {
		for _, attribute := range element.Attributes {
			attributes[attribute.Name] = attribute
		}
	}
	if got := attributes["quantity"]; got.State != formatversion.Value || got.Value != "2" {
		t.Fatalf("quantity=%+v", got)
	}
	if got := attributes["note"]; got.State != formatversion.Empty || got.Value != "" {
		t.Fatalf("note=%+v", got)
	}
	if got := attributes["state"]; got.State != formatversion.Absent {
		t.Fatalf("state=%+v", got)
	}
}

func TestWarningMakesTheResultSuccessWithWarnings(t *testing.T) {
	registry := newRegistry(t, textCodec{version: 1}, textCodec{version: 2, rename: map[string]string{"quantity": "qty"}})
	result, err := registry.Convert([]byte(v1Source), 1, 2)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != formatversion.SuccessWithWarnings {
		t.Fatalf("status=%v, want SuccessWithWarnings", result.Status)
	}
	if !hasCode(t, result, "ATTRIBUTE_RENAMED") {
		t.Fatalf("rename warning missing: %+v", result.Diagnostics)
	}
	for _, diagnostic := range result.Diagnostics {
		if diagnostic.ElementID == "" {
			t.Fatalf("diagnostic without element id: %+v", diagnostic)
		}
	}
}

func TestSilentAttributeLossFails(t *testing.T) {
	registry := newRegistry(t, textCodec{version: 1}, textCodec{version: 2, drop: map[string]bool{"note": true}, dropSilently: true})
	result, err := registry.Convert([]byte(v1Source), 1, 2)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != formatversion.Failure || !hasCode(t, result, "ATTRIBUTE_LOST") {
		t.Fatalf("silent loss was not detected: %+v", result)
	}
	for _, diagnostic := range result.Diagnostics {
		if diagnostic.Code == "ATTRIBUTE_LOST" {
			if diagnostic.ElementID != "sales.order" || diagnostic.Attribute != "note" {
				t.Fatalf("loss diagnostic is not element-scoped: %+v", diagnostic)
			}
		}
	}
}

func TestReportedAttributeRemovalIsAWarning(t *testing.T) {
	registry := newRegistry(t, textCodec{version: 1}, textCodec{version: 2, drop: map[string]bool{"note": true}})
	result, err := registry.Convert([]byte(v1Source), 1, 2)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != formatversion.SuccessWithWarnings || !hasCode(t, result, "ATTRIBUTE_DEPRECATED") {
		t.Fatalf("reported removal was not a warning: %+v", result)
	}
}

func TestVersionOnlyBumpFails(t *testing.T) {
	registry := newRegistry(t, textCodec{version: 1}, textCodec{version: 2, noop: true})
	result, err := registry.Convert([]byte(v1Source), 1, 2)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != formatversion.Failure || result.Diagnostics[0].Code != "VERSION_ONLY_BUMP" {
		t.Fatalf("version-only bump was accepted: %+v", result)
	}
}

func TestUnknownVersionNamesTheSupportedVersions(t *testing.T) {
	registry := newRegistry(t, textCodec{version: 1})
	result, err := registry.Convert([]byte(v1Source), 3, 1)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != formatversion.Failure || result.Diagnostics[0].Code != "UNKNOWN_SOURCE_VERSION" {
		t.Fatalf("unknown source version: %+v", result)
	}
	if !strings.Contains(result.Reason, "1") {
		t.Fatalf("supported versions are not named: %q", result.Reason)
	}
}

func TestDeclaredVersionMismatchFails(t *testing.T) {
	registry := newRegistry(t, textCodec{version: 1}, textCodec{version: 2})
	source := strings.Replace(v1Source, "formatVersion 1", "formatVersion 2", 1)
	result, err := registry.Convert([]byte(source), 1, 2)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != formatversion.Failure || result.Diagnostics[0].Code != "VERSION_MISMATCH" {
		t.Fatalf("declared/requested mismatch was accepted: %+v", result)
	}
}

func TestCodecDiagnosticWithoutElementIDIsRejected(t *testing.T) {
	registry := formatversion.NewRegistry()
	if err := registry.Register(badDiagnosticCodec{}); err != nil {
		t.Fatal(err)
	}
	if _, err := registry.Convert([]byte("formatVersion 1\n"), 1, 1); err == nil {
		t.Fatal("codec diagnostic without an element id was accepted")
	}
}

type badDiagnosticCodec struct{}

func (badDiagnosticCodec) Version() formatversion.Version { return 1 }
func (badDiagnosticCodec) Decode([]byte) (formatversion.Document, []formatversion.Diagnostic, error) {
	return formatversion.Document{Declared: 1}, []formatversion.Diagnostic{{Severity: formatversion.SeverityWarning, Code: "NO_ELEMENT"}}, nil
}
func (badDiagnosticCodec) Encode(document formatversion.Document) ([]byte, []formatversion.Diagnostic, error) {
	return []byte("formatVersion 1\n"), nil, nil
}

func TestDuplicateCodecIsRejected(t *testing.T) {
	registry := formatversion.NewRegistry()
	if err := registry.Register(textCodec{version: 1}); err != nil {
		t.Fatal(err)
	}
	if err := registry.Register(textCodec{version: 1}); err == nil {
		t.Fatal("duplicate codec version was accepted")
	}
}
