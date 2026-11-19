package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"tei/internal/syntax"
)

// set-attribute-required changes one existing Data attribute's required cell.
// This is a typed change candidate evaluated by Shared Acceptance, NOT an
// unrestricted Studio file-write endpoint. Its implementation consumes the
// same tei.authoring/v1 parser and byte coordinates as the Workspace Loader.
const opSetAttributeRequired = "set-attribute-required"

func normalizeSetAttributeRequired(op Operation) (Operation, error) {
	if err := onlyFields(op, "id", "required"); err != nil {
		return op, err
	}
	if op.Placement != "" {
		return op, fmt.Errorf("%s does not accept placement; the Source Index selects the declaration", op.Op)
	}
	id, err := canonical("id", op.ID)
	if err != nil {
		return op, err
	}
	if op.Required == nil {
		return op, fmt.Errorf("%s requires an explicit true or false", op.Op)
	}
	required := *op.Required
	return Operation{Op: opSetAttributeRequired, ID: string(id), Required: &required}, nil
}

// requiredMarkdownCell resolves an exact attribute in one marked Data table.
// It does not grep or replace matching strings in prose, examples, or another
// Data. A missing or duplicate declaration is rejected before Source editing.
func requiredMarkdownCell(content []byte, dataID, attributeID string) (syntax.MarkdownCell, error) {
	document, err := syntax.ScanTEIMarkdownDocument(string(content))
	if err != nil {
		return syntax.MarkdownCell{}, err
	}
	var selected syntax.MarkdownCell
	matches := 0
	for _, table := range document.Tables {
		if len(table.Words) == 0 || table.Words[0].Text != "data" {
			continue
		}
		parsed, err := syntax.ParseDataTable(table)
		if err != nil {
			return syntax.MarkdownCell{}, err
		}
		if string(parsed.Value.ID) != dataID {
			continue
		}
		column := -1
		for i, header := range table.Header {
			if header.Text == "required" {
				column = i
			}
		}
		if column < 0 {
			return syntax.MarkdownCell{}, fmt.Errorf("Data %s has no required column", dataID)
		}
		for i, attribute := range parsed.Attributes {
			if string(attribute.Value.ID) != attributeID {
				continue
			}
			if i >= len(table.Rows) || column >= len(table.Rows[i]) {
				return syntax.MarkdownCell{}, fmt.Errorf("required cell for %s is absent", attributeID)
			}
			selected = table.Rows[i][column]
			matches++
		}
	}
	if matches != 1 {
		return syntax.MarkdownCell{}, fmt.Errorf("attribute %s has %d exact required cells", attributeID, matches)
	}
	start, end := selected.Anchor.Start.Offset, selected.Anchor.End.Offset
	if start < 0 || end < start || end > len(content) || string(content[start:end]) != selected.Text {
		return syntax.MarkdownCell{}, fmt.Errorf("required cell %s has a mismatched Source byte range", attributeID)
	}
	return selected, nil
}

func resolveSetAttributeRequired(s snapshot, op Operation) []Diagnostic {
	key := operationKey(op)
	fail := func(code, format string, args ...any) []Diagnostic {
		return []Diagnostic{{Code: code, Message: fmt.Sprintf(format, args...), Subject: key}}
	}
	attr, owner, ok := attributeOf(s, op.ID)
	if !ok {
		return fail("ATTRIBUTE_NOT_FOUND", "attribute %s is absent from the current Canonical World", op.ID)
	}
	if op.Required == nil {
		return fail("REQUIRED_CHOICE_ABSENT", "attribute %s requires an explicit boolean", op.ID)
	}
	if attr.Required == *op.Required {
		return fail("ATTRIBUTE_REQUIRED_UNCHANGED", "attribute %s already has required=%t", op.ID, attr.Required)
	}
	location, ok := s.loaded.Sources["semantic-data:"+owner]
	if !ok {
		return fail("ATTRIBUTE_SOURCE_NOT_FOUND", "Data %s has no exact Source Index declaration", owner)
	}
	// Native .tei syntax has different source coordinates. It must not be
	// modified by treating its text as a Markdown table.
	if !strings.HasSuffix(location.File, ".tei.md") {
		return fail("ATTRIBUTE_SOURCE_FORMAT_UNSUPPORTED", "attribute %s must be authored as a tei.authoring/v1 Data table", op.ID)
	}
	original, err := os.ReadFile(filepath.Join(s.root, filepath.FromSlash(location.File)))
	if err != nil {
		return fail("ATTRIBUTE_SOURCE_UNREADABLE", "read %s: %v", location.File, err)
	}
	cell, err := requiredMarkdownCell(original, owner, op.ID)
	if err != nil {
		return fail("ATTRIBUTE_REQUIRED_ANCHOR_UNRESOLVED", "%v", err)
	}
	if (cell.Text == "yes") != attr.Required {
		return fail("ATTRIBUTE_REQUIRED_SOURCE_MISMATCH", "attribute %s Source and Canonical required value differ", op.ID)
	}
	return nil
}

// replaceAttributeRequiredCell modifies only the parsed cell's exact source
// byte interval. All whitespace, prose, tables, newlines and sugar survive.
func replaceAttributeRequiredCell(content []byte, owner string, op Operation) ([]byte, string, error) {
	if op.Required == nil {
		return nil, "", fmt.Errorf("required choice is absent")
	}
	cell, err := requiredMarkdownCell(content, owner, op.ID)
	if err != nil {
		return nil, "", err
	}
	newValue := "no"
	if *op.Required {
		newValue = "yes"
	}
	if cell.Text == newValue {
		return nil, "", fmt.Errorf("attribute %s already declares %s", op.ID, newValue)
	}
	start, end := cell.Anchor.Start.Offset, cell.Anchor.End.Offset
	after := make([]byte, 0, len(content)-end+start+len(newValue))
	after = append(after, content[:start]...)
	after = append(after, newValue...)
	after = append(after, content[end:]...)
	return after, cell.Text, nil
}
