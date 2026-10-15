package xeadsource

import (
	"errors"
	"sort"
)

var (
	ErrStudioObservationRequiresXead       = errors.New("studio-observation-requires-xead")
	ErrStudioObservationRequiresSystemRoot = errors.New("studio-observation-requires-system-root")
)

// StudioObservation is a bounded projection of an X-TEA source document. It
// is not a Canonical model and does not grant source or execution authority.
type StudioObservation struct {
	Tables      []StudioTable      `json:"tables,omitempty"`
	Diagnostics []StudioDiagnostic `json:"diagnostics,omitempty"`
}

type StudioSourceRef struct {
	Kind string `json:"sourceKind"`
	ID   string `json:"sourceID"`
}

type StudioTable struct {
	Source StudioSourceRef    `json:"source"`
	Name   string             `json:"name,omitempty"`
	Fields []StudioTableField `json:"fields,omitempty"`
}

type StudioTableField struct {
	Source   StudioSourceRef  `json:"source"`
	Name     string           `json:"name,omitempty"`
	DataType *StudioSourceRef `json:"dataType,omitempty"`
}

// StudioDiagnostic reports only construct and attribute names and counts.
// It never copies source values, text, or namespace URIs.
type StudioDiagnostic struct {
	Code        string `json:"code"`
	Element     string `json:"element"`
	Attribute   string `json:"attribute,omitempty"`
	Occurrences int    `json:"occurrences"`
}

type studioDiagnosticKey struct {
	code, element, attribute string
}

// ProjectStudioObservation projects an already parsed X-TEA document into the
// small Studio DTO. Parse retains the full source tree in its Document; this
// function limits only the returned projection.
func ProjectStudioObservation(doc Document) (StudioObservation, error) {
	if doc.Format != FormatXead {
		return StudioObservation{}, ErrStudioObservationRequiresXead
	}
	if doc.Root.Name != "System" {
		return StudioObservation{}, ErrStudioObservationRequiresSystemRoot
	}

	out := StudioObservation{Tables: []StudioTable{}}
	counts := make(map[studioDiagnosticKey]int)
	recordUnsupportedAttributes(counts, doc.Root, nil)
	for _, child := range doc.Root.Children {
		if child.Name != "Table" {
			recordUnsupportedSubtree(counts, child)
			continue
		}

		table := StudioTable{
			Source: StudioSourceRef{Kind: "Table", ID: studioAttribute(child, "ID")},
			Name:   studioAttribute(child, "Name"),
			Fields: []StudioTableField{},
		}
		if table.Source.ID == "" {
			counts[studioDiagnosticKey{code: "missing-source-id", element: "Table", attribute: "ID"}]++
		}
		recordUnsupportedAttributes(counts, child, set("ID", "Name"))
		for _, fieldNode := range child.Children {
			if fieldNode.Name != "TableField" {
				recordUnsupportedSubtree(counts, fieldNode)
				continue
			}

			field := StudioTableField{
				Source: StudioSourceRef{Kind: "TableField", ID: studioAttribute(fieldNode, "ID")},
				Name:   studioAttribute(fieldNode, "Name"),
			}
			if field.Source.ID == "" {
				counts[studioDiagnosticKey{code: "missing-source-id", element: "TableField", attribute: "ID"}]++
			}
			if typeID := studioAttribute(fieldNode, "DataTypeID"); typeID != "" {
				field.DataType = &StudioSourceRef{Kind: "DataType", ID: typeID}
			} else {
				counts[studioDiagnosticKey{code: "missing-source-type-id", element: "TableField", attribute: "DataTypeID"}]++
			}
			recordUnsupportedAttributes(counts, fieldNode, set("ID", "Name", "DataTypeID"))
			for _, nested := range fieldNode.Children {
				recordUnsupportedSubtree(counts, nested)
			}
			table.Fields = append(table.Fields, field)
		}
		out.Tables = append(out.Tables, table)
	}
	if len(counts) > 0 {
		out.Diagnostics = sortedStudioDiagnostics(counts)
	}
	return out, nil
}

func studioAttribute(node Node, name string) string {
	for _, attribute := range node.Attributes {
		if attribute.Name == name {
			return attribute.Value
		}
	}
	return ""
}

func recordUnsupportedAttributes(counts map[studioDiagnosticKey]int, node Node, allowed map[string]struct{}) {
	for _, attribute := range node.Attributes {
		if _, ok := allowed[attribute.Name]; ok {
			continue
		}
		counts[studioDiagnosticKey{
			code:      "unsupported-attribute",
			element:   localName(node.Name),
			attribute: localName(attribute.Name),
		}]++
	}
}

func recordUnsupportedSubtree(counts map[studioDiagnosticKey]int, node Node) {
	counts[studioDiagnosticKey{
		code:    "unsupported-element",
		element: localName(node.Name),
	}]++
	recordUnsupportedAttributes(counts, node, nil)
	for _, child := range node.Children {
		recordUnsupportedSubtree(counts, child)
	}
}

func sortedStudioDiagnostics(counts map[studioDiagnosticKey]int) []StudioDiagnostic {
	keys := make([]studioDiagnosticKey, 0, len(counts))
	for key := range counts {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].code != keys[j].code {
			return keys[i].code < keys[j].code
		}
		if keys[i].element != keys[j].element {
			return keys[i].element < keys[j].element
		}
		return keys[i].attribute < keys[j].attribute
	})
	result := make([]StudioDiagnostic, 0, len(keys))
	for _, key := range keys {
		result = append(result, StudioDiagnostic{
			Code: key.code, Element: key.element, Attribute: key.attribute,
			Occurrences: counts[key],
		})
	}
	return result
}
