// Package teicodec is the real .tei codec for the formatversion conversion
// boundary (#1920). It decodes the supported semantic subset into the central
// representation and encodes it back, so a version step is exercised against
// actual .tei syntax rather than a synthetic stand-in.
//
// The subset is deliberately explicit: a semantic file is supported when it
// contains only a namespace, data declarations, and attribute declarations.
// Any other construct is refused with an element-scoped diagnostic instead of
// being dropped silently.
package teicodec

import (
	"fmt"
	"sort"
	"strings"

	"tei/internal/formatversion"
	"tei/internal/semantic"
	"tei/internal/syntax"
)

// Codec decodes and encodes .tei semantic sources. RequireHeader is true for
// versions that mandate an explicit formatVersion declaration.
type Codec struct {
	version       formatversion.Version
	requireHeader bool
}

// New returns a codec for one version. Version 1 allows the header to be
// inherited from the Workspace; version 2 requires an explicit header.
func New(version formatversion.Version, requireHeader bool) *Codec {
	return &Codec{version: version, requireHeader: requireHeader}
}

// Version implements formatversion.Codec.
func (c *Codec) Version() formatversion.Version { return c.version }

const (
	// Format is the stable format identifier used when the .tei codec is
	// registered in the core formatversion.Formats entry.
	Format = "tei.semantic"
	// CodeUnsupportedConstruct marks a declaration the codec cannot round-trip.
	CodeUnsupportedConstruct = "UNSUPPORTED_CONSTRUCT"
	// CodeMaterializedDefault marks an inherited value written explicitly.
	CodeMaterializedDefault = "IMPLICIT_DEFAULT_MATERIALIZED"
)

// Decode parses a .tei semantic source into the central representation.
func (c *Codec) Decode(source []byte) (formatversion.Document, []formatversion.Diagnostic, error) {
	namespace, err := syntax.NewParser(string(source)).ParseSemanticFile()
	if err != nil {
		return formatversion.Document{}, nil, err
	}
	document := formatversion.Document{Declared: formatversion.Version(namespace.FormatVersion)}
	if len(namespace.Types) > 0 {
		return document, []formatversion.Diagnostic{unsupported(string(namespace.Types[0].ID), "type")}, nil
	}
	if len(namespace.Capabilities) > 0 {
		return document, []formatversion.Diagnostic{unsupported(string(namespace.Capabilities[0].ID), "capability")}, nil
	}
	if len(namespace.Validations) > 0 {
		return document, []formatversion.Diagnostic{unsupported(string(namespace.Validations[0].ID), "validation")}, nil
	}
	if len(namespace.Periods) > 0 {
		return document, []formatversion.Diagnostic{unsupported(string(namespace.Periods[0].ID), "period")}, nil
	}
	document.Elements = append(document.Elements, formatversion.Element{ID: string(namespace.ID), Kind: "namespace"})
	for _, data := range namespace.Data {
		switch {
		case data.Description != "":
			return document, []formatversion.Diagnostic{unsupported(string(data.ID), "data description")}, nil
		case data.Display != "":
			return document, []formatversion.Diagnostic{unsupported(string(data.ID), "data display")}, nil
		case data.Script != "":
			return document, []formatversion.Diagnostic{unsupported(string(data.ID), "data script")}, nil
		case len(data.Relations) > 0:
			return document, []formatversion.Diagnostic{unsupported(string(data.ID), "relation")}, nil
		case len(data.CandidateKeys) > 0:
			return document, []formatversion.Diagnostic{unsupported(string(data.ID), "candidate-key")}, nil
		case len(data.FunctionalDependencies) > 0:
			return document, []formatversion.Diagnostic{unsupported(string(data.ID), "functional-dependency")}, nil
		}
		document.Elements = append(document.Elements, formatversion.Element{ID: string(data.ID), Kind: "data"})
		for _, attribute := range data.Attributes {
			document.Elements = append(document.Elements, formatversion.Element{
				ID:         string(attribute.ID),
				Kind:       "attribute",
				Attributes: attributeToCentral(attribute),
			})
		}
	}
	return document, nil, nil
}

func unsupported(elementID, construct string) formatversion.Diagnostic {
	if elementID == "" {
		elementID = "semantic"
	}
	return formatversion.Diagnostic{
		Severity:  formatversion.SeverityFailure,
		Code:      CodeUnsupportedConstruct,
		ElementID: elementID,
		Message:   fmt.Sprintf("the .tei codec cannot round-trip %s; refusing instead of dropping it", construct),
	}
}

func attributeToCentral(attribute semantic.Attribute) []formatversion.Attribute {
	out := []formatversion.Attribute{{Name: "type", State: formatversion.Value, Value: string(attribute.TypeRef)}}
	if attribute.Required {
		out = append(out, formatversion.Attribute{Name: "required", State: formatversion.Value, Value: "true"})
	}
	if attribute.Immutable {
		out = append(out, formatversion.Attribute{Name: "immutable", State: formatversion.Value, Value: "true"})
	}
	if attribute.Role != "" {
		out = append(out, formatversion.Attribute{Name: "role", State: formatversion.Value, Value: attribute.Role})
	}
	if attribute.Description != "" {
		out = append(out, formatversion.Attribute{Name: "description", State: formatversion.Value, Value: attribute.Description})
	}
	if attribute.Display != "" {
		out = append(out, formatversion.Attribute{Name: "display", State: formatversion.Value, Value: attribute.Display})
	}
	if attribute.Value != nil {
		state := formatversion.Value
		if *attribute.Value == "" {
			state = formatversion.Empty
		}
		out = append(out, formatversion.Attribute{Name: "value", State: state, Value: *attribute.Value})
	}
	return out
}

// Encode writes the central representation back to .tei syntax. The output is
// canonical: elements and attributes are sorted, so the same document always
// produces the same bytes.
func (c *Codec) Encode(document formatversion.Document) ([]byte, []formatversion.Diagnostic, error) {
	namespaceID := ""
	for _, element := range document.Elements {
		if element.Kind == "namespace" {
			if namespaceID != "" {
				return nil, nil, fmt.Errorf("teicodec: more than one namespace element")
			}
			namespaceID = element.ID
		}
	}
	if namespaceID == "" {
		return nil, nil, fmt.Errorf("teicodec: document has no namespace element")
	}

	var diagnostics []formatversion.Diagnostic
	var builder strings.Builder
	writeHeader := c.requireHeader || document.Declared != 0
	if writeHeader {
		if c.requireHeader && document.Declared == 0 {
			// The source inherited its version from the Workspace. Writing the
			// target version makes that implicit default explicit.
			diagnostics = append(diagnostics, formatversion.Diagnostic{
				Severity:  formatversion.SeverityWarning,
				Code:      CodeMaterializedDefault,
				ElementID: namespaceID,
				Attribute: "formatVersion",
				Message:   fmt.Sprintf("inherited format version was materialized as %d", c.version),
			})
		}
		fmt.Fprintf(&builder, "formatVersion %d\n", c.version)
	}
	fmt.Fprintf(&builder, "namespace %s\n", namespaceID)

	elements := append([]formatversion.Element(nil), document.Elements...)
	sort.Slice(elements, func(i, j int) bool { return elements[i].ID < elements[j].ID })
	for _, element := range elements {
		switch element.Kind {
		case "namespace":
			continue
		case "data":
			fmt.Fprintf(&builder, "\ndata %s {\n", symbol(element.ID))
			for _, child := range elements {
				if child.Kind != "attribute" || !strings.HasPrefix(child.ID, element.ID+".") {
					continue
				}
				line, err := encodeAttributeLine(child)
				if err != nil {
					return nil, nil, err
				}
				builder.WriteString("    " + line + "\n")
			}
			builder.WriteString("}\n")
		case "attribute":
			// Attributes are emitted with their data element.
			continue
		default:
			return nil, nil, fmt.Errorf("teicodec: unknown element kind %q", element.Kind)
		}
	}
	return []byte(builder.String()), diagnostics, nil
}

func encodeAttributeLine(element formatversion.Element) (string, error) {
	values := map[string]formatversion.Attribute{}
	for _, attribute := range element.Attributes {
		values[attribute.Name] = attribute
	}
	typeAttribute, ok := values["type"]
	if !ok || typeAttribute.State == formatversion.Absent {
		return "", fmt.Errorf("teicodec: attribute %q has no type", element.ID)
	}
	line := fmt.Sprintf("attribute %s type %s", symbol(element.ID), typeAttribute.Value)
	if values["required"].State != formatversion.Absent {
		line += " required"
	}
	if values["immutable"].State != formatversion.Absent {
		line += " immutable"
	}
	if role := values["role"]; role.State == formatversion.Value {
		line += " role " + role.Value
	}
	if description := values["description"]; description.State == formatversion.Value {
		line += fmt.Sprintf(" description %q", description.Value)
	}
	if display := values["display"]; display.State == formatversion.Value {
		line += fmt.Sprintf(" display %q", display.Value)
	}
	if value, ok := values["value"]; ok {
		switch value.State {
		case formatversion.Empty:
			line += ` value ""`
		case formatversion.Value:
			line += fmt.Sprintf(" value %q", value.Value)
		case formatversion.Absent:
			return "", fmt.Errorf("teicodec: attribute %q has an absent value entry", element.ID)
		}
	}
	return line, nil
}

func symbol(id string) string {
	if index := strings.LastIndex(id, "."); index >= 0 {
		return id[index+1:]
	}
	return id
}
