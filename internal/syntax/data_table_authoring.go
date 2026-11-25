package syntax

import (
	"fmt"
	"strings"

	"tei/internal/core"
	"tei/internal/semantic"
)

// This file reads a Data definition written as an item table:
//
//	<!-- tei: data TransferRequest in organization -->
//	| attribute | type | required | display | description |
//	| --- | --- | --- | --- | --- |
//	| transferRequestId | organization.transfer_request_id | yes | 異動申請ID | |
//	| effectiveFrom | std.datetime | yes | 発効日 | この日から新しい所属になる |
//
// One row is one attribute. The result is the existing semantic.Data, with
// the same identities the native `data` declaration derives, so a table and a
// native declaration of the same items mean the same thing.
//
// The table defines the shape of records. It does not hold records: who
// belongs where is data in a store, not a definition, and is not written here.
//
// An empty cell means "not stated". It is allowed only where the native
// declaration may leave the option out, and it is never read as a value such
// as an empty string, zero, or "no end".

// DataTableColumn is a column a Data item table may have.
type DataTableColumn string

const (
	DataTableColumnAttribute   DataTableColumn = "attribute"
	DataTableColumnType        DataTableColumn = "type"
	DataTableColumnRequired    DataTableColumn = "required"
	DataTableColumnDisplay     DataTableColumn = "display"
	DataTableColumnDescription DataTableColumn = "description"
)

var dataTableColumns = []DataTableColumn{
	DataTableColumnAttribute, DataTableColumnType, DataTableColumnRequired,
	DataTableColumnDisplay, DataTableColumnDescription,
}

// DataTableAttribute is one row: the attribute and the spans of the cells an
// editor or a diagnostic points at.
type DataTableAttribute struct {
	Value      semantic.Attribute
	SymbolSpan Anchor
	TypeSpan   Anchor
}

// DataTableAuthoring is one parsed Data item table. Spans are byte ranges of
// the whole document.
type DataTableAuthoring struct {
	Namespace     core.CanonicalID
	NamespaceSpan Anchor
	Value         semantic.Data
	SymbolSpan    Anchor
	Attributes    []DataTableAttribute
}

// ParseDataTable reads a marked table as a Data definition. The result is a
// candidate: it is not accepted meaning and carries no authority.
func ParseDataTable(table MarkdownTable) (DataTableAuthoring, error) {
	var out DataTableAuthoring
	words := table.Words
	if len(words) != 4 || words[0].Text != "data" || words[2].Text != "in" {
		return out, &ParseError{Pos: table.Mark.Start, End: table.Mark.End,
			Msg: "unknown tei mark; a Data item table is marked <!-- tei: data <Name> in <namespace> -->"}
	}
	symbol, namespaceWord := words[1], words[3]
	namespace, err := core.ParseCanonicalID(namespaceWord.Text)
	if err != nil || strings.Contains(namespaceWord.Text, ".") {
		return out, cellError(namespaceWord, "the namespace of a Data item table must be a top-level canonical id")
	}
	if strings.Contains(symbol.Text, ".") {
		return out, cellError(symbol, "write the Data name without a namespace; the namespace follows \"in\"")
	}
	segment, err := core.SymbolToSegment(symbol.Text)
	if err != nil {
		return out, cellError(symbol, err.Error())
	}
	dataID, err := core.ParseCanonicalID(string(namespace) + "." + segment)
	if err != nil {
		return out, cellError(symbol, err.Error())
	}
	out.Namespace, out.NamespaceSpan = namespace, namespaceWord.Anchor
	out.Value = semantic.Data{ID: dataID, Symbol: symbol.Text}
	out.SymbolSpan = symbol.Anchor

	columns := map[DataTableColumn]int{}
	for i, header := range table.Header {
		column := DataTableColumn(header.Text)
		known := false
		for _, candidate := range dataTableColumns {
			known = known || candidate == column
		}
		if !known {
			return out, cellError(header, fmt.Sprintf("unknown column %q; a Data item table has the columns attribute, type, required, display, description", header.Text))
		}
		if _, duplicate := columns[column]; duplicate {
			return out, cellError(header, fmt.Sprintf("column %q appears more than once", header.Text))
		}
		columns[column] = i
	}
	for _, required := range []DataTableColumn{DataTableColumnAttribute, DataTableColumnType} {
		if _, present := columns[required]; !present {
			return out, &ParseError{Pos: table.Mark.Start, End: table.Mark.End,
				Msg: fmt.Sprintf("a Data item table needs the column %q", required)}
		}
	}
	cell := func(row []MarkdownCell, column DataTableColumn) (MarkdownCell, bool) {
		index, present := columns[column]
		if !present {
			return MarkdownCell{}, false
		}
		return row[index], true
	}

	// A table with no body row states no attribute. The scanner reads it so a
	// row-scoped example can state an empty relation; here it would be a Data
	// item with no attributes, which is refused at the table.
	if len(table.Rows) == 0 {
		return out, &ParseError{Pos: table.Mark.Start, End: table.Mark.End,
			Msg: "a Data item table needs at least one attribute row; a table with no row states no attribute"}
	}
	seen := map[core.CanonicalID]Anchor{}
	for _, row := range table.Rows {
		attributeCell, _ := cell(row, DataTableColumnAttribute)
		typeCell, _ := cell(row, DataTableColumnType)
		if attributeCell.Text == "" {
			return out, cellError(attributeCell, "the attribute cell is empty; every row names one attribute")
		}
		if typeCell.Text == "" {
			return out, cellError(typeCell, fmt.Sprintf("attribute %s has no type; an empty cell is not read as a default type", attributeCell.Text))
		}
		attributeSegment, err := core.SymbolToSegment(attributeCell.Text)
		if err != nil {
			return out, cellError(attributeCell, err.Error())
		}
		attributeID, err := core.ParseCanonicalID(string(dataID) + "." + attributeSegment)
		if err != nil {
			return out, cellError(attributeCell, err.Error())
		}
		if prior, duplicate := seen[attributeID]; duplicate {
			return out, cellError(attributeCell, fmt.Sprintf("attribute %s is written more than once; the first row is at %d:%d",
				attributeID, prior.Start.Line, prior.Start.Column))
		}
		seen[attributeID] = attributeCell.Anchor
		if !strings.Contains(typeCell.Text, ".") {
			return out, cellError(typeCell, fmt.Sprintf("type %q is not a qualified canonical id; write it with its namespace, for example std.%s", typeCell.Text, typeCell.Text))
		}
		typeRef, err := core.ParseCanonicalID(typeCell.Text)
		if err != nil {
			return out, cellError(typeCell, err.Error())
		}
		attribute := semantic.Attribute{ID: attributeID, Symbol: attributeCell.Text, TypeRef: typeRef}

		if requiredCell, present := cell(row, DataTableColumnRequired); present {
			switch requiredCell.Text {
			case "yes":
				attribute.Required = true
			case "no", "":
			default:
				return out, cellError(requiredCell, fmt.Sprintf("required is written yes or no, not %q", requiredCell.Text))
			}
		}
		for column, target := range map[DataTableColumn]*string{
			DataTableColumnDisplay:     &attribute.Display,
			DataTableColumnDescription: &attribute.Description,
		} {
			textCell, present := cell(row, column)
			if !present {
				continue
			}
			if strings.ContainsAny(textCell.Text, "\"\\") {
				return out, cellError(textCell, fmt.Sprintf("a %s cell cannot contain a double quote or a backslash", column))
			}
			*target = textCell.Text
		}
		out.Value.Attributes = append(out.Value.Attributes, attribute)
		out.Attributes = append(out.Attributes, DataTableAttribute{
			Value: attribute, SymbolSpan: attributeCell.Anchor, TypeSpan: typeCell.Anchor,
		})
	}
	return out, nil
}

func cellError(cell MarkdownCell, message string) error {
	return &ParseError{Pos: cell.Anchor.Start, End: cell.Anchor.End, Msg: message}
}
