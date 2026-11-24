package workspace

import (
	"fmt"
	"path/filepath"

	"tei/internal/core"
	"tei/internal/semantic"
	"tei/internal/syntax"
)

// loadSourceAnchor is the exact byte range of a declaration as the Load-time
// parse recorded it. Source Index uses it instead of parsing the file a second
// time, so the value and its position cannot come from two different readings.
type loadSourceAnchor struct {
	File  string
	Start int
	End   int
}

func joinWorkspacePath(root, slashPath string) string {
	return filepath.Join(root, filepath.FromSlash(slashPath))
}

// loadRuleAuthoringMarkdown is the built-in implementation of
// tei.rule-authoring/v0 for TEI Markdown. It converts every rule of every
// definition region into the existing semantic.Validation and appends it to
// the namespace the region names. It adds no validation kind and no second
// meaning model: the existing validator and evaluator read the result as they
// read a `validation` declaration.
//
// A source is all or nothing. One unreadable region, one unresolved reference,
// or one identity that is already declared elsewhere stops the whole source,
// so a partly read document never contributes plausible-looking meaning.
func loadRuleAuthoringMarkdown(root string, r *Result, filename string, src []byte) (int, int, error) {
	regions, err := syntax.ParseRuleAuthoringMarkdown(string(src))
	if err != nil {
		return 0, 0, r.collectSourceError(filename, err)
	}
	return mergeAuthoringMarkdown(root, r, filename, regions, nil)
}

// loadAuthoringMarkdown is the built-in implementation of tei.authoring/v1
// for TEI Markdown. It reads what tei.rule-authoring/v0 reads from `tei`
// fences and, in addition, Data item tables marked with a tei comment. A
// table becomes the existing semantic.Data; no new meaning model is added.
func loadAuthoringMarkdown(root string, r *Result, filename string, src []byte) (int, int, error) {
	document, err := syntax.ScanTEIMarkdownDocument(string(src))
	if err != nil {
		return 0, 0, r.collectSourceError(filename, err)
	}
	regions, err := syntax.ParseRuleAuthoringRegions(string(src), document.Regions)
	if err != nil {
		return 0, 0, r.collectSourceError(filename, err)
	}
	tables := make([]syntax.DataTableAuthoring, 0, len(document.Tables))
	var caseTables []syntax.OperatorCasesAuthoring
	var exampleTables []syntax.RuleExampleAuthoring
	for _, table := range document.Tables {
		// The first word of the mark says what the table defines.
		switch table.Words[0].Text {
		case "data":
			parsed, err := syntax.ParseDataTable(table)
			if err != nil {
				return 0, 0, r.collectSourceError(filename, err)
			}
			tables = append(tables, parsed)
		case "cases":
			parsed, err := syntax.ParseOperatorCasesTable(table)
			if err != nil {
				return 0, 0, r.collectSourceError(filename, err)
			}
			caseTables = append(caseTables, parsed)
		case "example":
			parsed, err := syntax.ParseRuleExampleTable(table)
			if err != nil {
				return 0, 0, r.collectSourceError(filename, err)
			}
			exampleTables = append(exampleTables, parsed)
		default:
			return 0, 0, r.collectSourceError(filename, &syntax.ParseError{Pos: table.Mark.Start, End: table.Mark.End,
				Msg: "unknown tei mark; a table is marked <!-- tei: data <Name> in <namespace> -->, <!-- tei: cases of <operator> --> or <!-- tei: example of <data> -->"})
		}
	}
	regionCount, definitions, err := mergeAuthoringMarkdown(root, r, filename, regions, tables)
	if err != nil {
		return 0, 0, err
	}
	// Examples are attached after every source is read, because the operator
	// may be declared in a document that is read later.
	relative := relPath(root, filename)
	for _, table := range caseTables {
		r.pendingOperatorCases = append(r.pendingOperatorCases, pendingOperatorCases{
			filename: filename, relative: relative, table: table,
		})
	}
	// Example records define nothing. They are resolved after every source is
	// read and kept beside the World.
	for _, table := range exampleTables {
		r.pendingRuleExamples = append(r.pendingRuleExamples, pendingRuleExample{
			filename: filename, relative: relative, table: table,
		})
	}
	return regionCount + len(caseTables) + len(exampleTables), definitions + len(caseTables), nil
}

// pendingOperatorCases is a case table read during Load and not yet attached
// to the operator it names.
type pendingOperatorCases struct {
	filename string
	relative string
	table    syntax.OperatorCasesAuthoring
}

// attachOperatorCases adds every case table to the operator it names. A table
// for an operator no source declares, and a second table for one operator,
// stop the Load at the table's own position.
func attachOperatorCases(r *Result) error {
	attached := map[core.CanonicalID]pendingOperatorCases{}
	for _, pending := range r.pendingOperatorCases {
		table := pending.table
		fail := func(message string) error {
			return r.collectSourceError(pending.filename, &syntax.ParseError{
				Pos: table.OperatorSpan.Start, End: table.OperatorSpan.End, Msg: message})
		}
		if prior, duplicate := attached[table.Operator]; duplicate {
			if err := fail(fmt.Sprintf("operator %s already has a case table at %s:%d:%d; write its examples in one table",
				table.Operator, filepath.ToSlash(prior.relative), prior.table.OperatorSpan.Start.Line, prior.table.OperatorSpan.Start.Column)); err != nil {
				return err
			}
			continue
		}
		found := false
		for i := range r.World.Semantics {
			for j := range r.World.Semantics[i].Operators {
				if r.World.Semantics[i].Operators[j].ID != table.Operator {
					continue
				}
				found = true
				// The parser reads every column that is not expect/note/value
				// as an input name and leaves the signature check here, so this
				// is where a table that names an input the operator does not
				// declare, or leaves a declared input unstated, has to stop.
				// Without it the table looks valid to `tei validate` and is
				// refused only when a read decides the rule.
				if err := requireCaseColumnsMatchSignature(r, pending, r.World.Semantics[i].Operators[j]); err != nil {
					return err
				}
				for _, example := range table.Cases {
					r.World.Semantics[i].Operators[j].Cases = append(r.World.Semantics[i].Operators[j].Cases, example.Value)
				}
			}
		}
		if !found {
			if err := fail(fmt.Sprintf("the examples are for operator %s, but no source declares that operator", table.Operator)); err != nil {
				return err
			}
			continue
		}
		attached[table.Operator] = pending
	}
	r.pendingOperatorCases = nil
	return nil
}

// requireCaseColumnsMatchSignature matches each example's input names to the
// inputs the operator declares. A column that names an input the operator does
// not take, and a declared input that an example leaves unstated, are both
// refused at the table's own position: an example the implementation could not
// be asked about is not an example of that operator.
//
// The first input is the rule subject and is named `value` in the single-value
// form, so a single-value example states no input names of its own and has
// nothing to match.
func requireCaseColumnsMatchSignature(
	r *Result,
	pending pendingOperatorCases,
	operator semantic.Operator,
) error {
	declared := operator.Inputs
	if len(declared) == 0 {
		return nil
	}
	names := make(map[string]bool, len(declared))
	for _, input := range declared {
		names[input.Name] = true
	}
	reject := func(span syntax.Anchor, message string) error {
		return r.collectSourceError(pending.filename, &syntax.ParseError{
			Pos: span.Start, End: span.End, Msg: message})
	}
	for _, example := range pending.table.Cases {
		for name, span := range example.InputSpans {
			if names[name] {
				continue
			}
			if err := reject(span, fmt.Sprintf(
				"operator %s declares no input named %q; a case column names one of the operator's inputs",
				operator.ID, name)); err != nil {
				return err
			}
		}
		// An example of a multi-input operator names one column per input after
		// the first. A table that states only the `value` column leaves every
		// other input unstated, which is the same refusal: the subject alone is
		// not an example of a three-input operator.
		stated := example.Value.Inputs
		if len(stated) == 0 && len(declared) > 1 {
			span := example.ValueSpan
			if len(example.InputSpans) == 0 && span == (syntax.Anchor{}) {
				span = example.ExpectSpan
			}
			if err := reject(span, fmt.Sprintf(
				"operator %s declares %d inputs, but this example states one value only; a case names one column per input",
				operator.ID, len(declared))); err != nil {
				return err
			}
			continue
		}
		for _, input := range declared {
			if input.Name == declared[0].Name {
				// The subject's value is the `value` column; a table may name it
				// too, and it is already stated when the map holds it.
				continue
			}
			if _, ok := stated[input.Name]; ok {
				continue
			}
			// A multi-input example has no single value to quote, so the
			// message says which example by its position rather than by
			// naming a value it does not have.
			message := fmt.Sprintf(
				"operator %s declares input %q, but this example states no value for it",
				operator.ID, input.Name)
			span := example.ValueSpan
			if example.Value.Value != "" {
				message = fmt.Sprintf(
					"operator %s declares input %q, but the example for %s states no value for it",
					operator.ID, input.Name, example.Value.Value)
			} else if span == (syntax.Anchor{}) {
				// A multi-input example has no single value cell, so its
				// position comes from the expect cell that is always present.
				span = example.ExpectSpan
			}
			if err := reject(span, message); err != nil {
				return err
			}
		}
	}
	return nil
}

// mergeAuthoringMarkdown adds the definitions of one source to the World, or
// none of them.
func mergeAuthoringMarkdown(root string, r *Result, filename string, regions []syntax.RuleAuthoringMarkdownRegion, tables []syntax.DataTableAuthoring) (int, int, error) {
	relative := relPath(root, filename)

	// Report every problem of the source before deciding to contribute nothing.
	rejected := false
	declaredOperators := map[core.CanonicalID]SourceLocation{}
	reject := func(span syntax.Anchor, message string) error {
		rejected = true
		return r.collectSourceError(filename, &syntax.ParseError{Pos: span.Start, End: span.End, Msg: message})
	}
	for _, region := range regions {
		for _, issue := range region.Rules.Issues {
			if err := reject(issue.Span, issue.Error()); err != nil {
				return 0, 0, err
			}
		}
		if len(region.Rules.Issues) == 0 && (len(region.Rules.Value.Validations) != len(region.Rules.File.Rules) ||
			len(region.Rules.Value.Operators) != len(region.Rules.File.Operators)) {
			if err := reject(region.Region.Open, "a declaration of this region produced no meaning"); err != nil {
				return 0, 0, err
			}
		}
		for _, operator := range region.Rules.File.Operators {
			key := "semantic-operator:" + string(operator.ID)
			prior, declared := r.Sources[key]
			if !declared {
				prior, declared = declaredOperators[operator.ID]
			}
			if declared {
				message := fmt.Sprintf("operator %s is already declared at %s:%d:%d; one identity cannot have two declarations",
					operator.ID, filepath.ToSlash(prior.File), prior.Line, prior.Column)
				if err := reject(operator.IDSpan, message); err != nil {
					return 0, 0, err
				}
				continue
			}
			declaredOperators[operator.ID] = SourceLocation{File: relative, Line: operator.IDSpan.Start.Line, Column: operator.IDSpan.Start.Column}
		}
		for _, rule := range region.Rules.File.Rules {
			if prior, declared := r.Sources["semantic-validation:"+string(rule.ID)]; declared {
				message := fmt.Sprintf("%s is already declared at %s:%d:%d; one identity cannot have two declarations",
					rule.ID, filepath.ToSlash(prior.File), prior.Line, prior.Column)
				if err := reject(rule.IDSpan, message); err != nil {
					return 0, 0, err
				}
			}
		}
	}
	declaredData := map[core.CanonicalID]SourceLocation{}
	for _, table := range tables {
		id := table.Value.ID
		prior, declared := r.Sources["semantic-data:"+string(id)]
		if !declared {
			prior, declared = declaredData[id]
		}
		if declared {
			message := fmt.Sprintf("data %s is already declared at %s:%d:%d; one identity cannot have two declarations",
				id, filepath.ToSlash(prior.File), prior.Line, prior.Column)
			if err := reject(table.SymbolSpan, message); err != nil {
				return 0, 0, err
			}
			continue
		}
		declaredData[id] = SourceLocation{File: relative, Line: table.SymbolSpan.Start.Line, Column: table.SymbolSpan.Start.Column}
	}
	if rejected {
		return 0, 0, nil
	}

	byNamespace := map[core.CanonicalID]int{}
	for i, existing := range r.World.Semantics {
		byNamespace[existing.ID] = i
	}
	record := func(key string, span syntax.Anchor) {
		r.Sources[key] = SourceLocation{File: relative, Line: span.Start.Line, Column: span.Start.Column}
		if r.sourceAnchors == nil {
			r.sourceAnchors = map[string]loadSourceAnchor{}
		}
		r.sourceAnchors[key] = loadSourceAnchor{File: relative, Start: span.Start.Offset, End: span.End.Offset}
	}
	definitions := 0
	for _, region := range regions {
		file := region.Rules.File
		idx, known := byNamespace[file.Namespace]
		if !known {
			idx = len(r.World.Semantics)
			byNamespace[file.Namespace] = idx
			r.World.Semantics = append(r.World.Semantics, semantic.Namespace{ID: file.Namespace})
			record("semantic:"+string(file.Namespace), file.NamespaceSpan)
		}
		for i, validation := range region.Rules.Value.Validations {
			r.World.Semantics[idx].Validations = append(r.World.Semantics[idx].Validations, validation)
			record("semantic-validation:"+string(validation.ID), file.Rules[i].IDSpan)
			definitions++
		}
		for i, operator := range region.Rules.Value.Operators {
			r.World.Semantics[idx].Operators = append(r.World.Semantics[idx].Operators, operator)
			record("semantic-operator:"+string(operator.ID), file.Operators[i].IDSpan)
			definitions++
		}
		if r.semanticFilesByNamespace == nil {
			r.semanticFilesByNamespace = make(map[string][]string)
		}
		namespace := string(file.Namespace)
		listed := false
		for _, existing := range r.semanticFilesByNamespace[namespace] {
			listed = listed || existing == relative
		}
		if !listed {
			r.semanticFilesByNamespace[namespace] = append(r.semanticFilesByNamespace[namespace], relative)
		}
	}
	for _, table := range tables {
		idx, known := byNamespace[table.Namespace]
		if !known {
			idx = len(r.World.Semantics)
			byNamespace[table.Namespace] = idx
			r.World.Semantics = append(r.World.Semantics, semantic.Namespace{ID: table.Namespace})
			record("semantic:"+string(table.Namespace), table.NamespaceSpan)
		}
		r.World.Semantics[idx].Data = append(r.World.Semantics[idx].Data, table.Value)
		record("semantic-data:"+string(table.Value.ID), table.SymbolSpan)
		for _, attribute := range table.Attributes {
			record("semantic-attribute:"+string(attribute.Value.ID), attribute.SymbolSpan)
		}
		definitions++
		if r.semanticFilesByNamespace == nil {
			r.semanticFilesByNamespace = make(map[string][]string)
		}
		namespace := string(table.Namespace)
		listed := false
		for _, existing := range r.semanticFilesByNamespace[namespace] {
			listed = listed || existing == relative
		}
		if !listed {
			r.semanticFilesByNamespace[namespace] = append(r.semanticFilesByNamespace[namespace], relative)
		}
	}
	return len(regions) + len(tables), definitions, nil
}
