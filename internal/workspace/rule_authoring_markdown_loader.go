package workspace

import (
	"fmt"
	"path/filepath"
	"strings"

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
	var rowCaseTables []syntax.OperatorRowCasesAuthoring
	rows := 0
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
			// Two forms share the mark. The one-record form has no answer in
			// the mark; the row form states it there (section 8.1).
			//
			// The word count tells them apart: the row form names the answer,
			// so its mark has more words than the one-record form's three.
			if len(table.Words) >= 4 {
				parsed, err := syntax.ParseOperatorRowCasesTable(table)
				if err != nil {
					return 0, 0, r.collectSourceError(filename, err)
				}
				rowCaseTables = append(rowCaseTables, parsed)
				rows++
				break
			}
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
	// A row example also waits until every source is read, because it names
	// an operator the document may not declare.
	for _, table := range rowCaseTables {
		r.pendingOperatorRowCases = append(r.pendingOperatorRowCases, pendingOperatorRowCases{
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
	return regionCount + len(caseTables) + rows + len(exampleTables), definitions + len(caseTables) + rows, nil
}

// pendingOperatorCases is a case table read during Load and not yet attached
// to the operator it names.
type pendingOperatorCases struct {
	filename string
	relative string
	table    syntax.OperatorCasesAuthoring
}

// pendingOperatorRowCases is a row-example table read during Load and not yet
// attached to the operator it names.
type pendingOperatorRowCases struct {
	filename string
	relative string
	table    syntax.OperatorRowCasesAuthoring
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
				operator := &r.World.Semantics[i].Operators[j]
				if operator.ID != table.Operator {
					continue
				}
				// An example carries one value. An operator that declares more
				// than one input cannot be stated by such a table: the other
				// inputs would go unstated, and the example would say something
				// the author never wrote. The reader refuses that shape when it
				// decides a rule, but a Workspace nobody reads would then pass
				// validate and fail only at read time. Refuse it here, at the
				// table, so one statement has one diagnosis.
				// The operator is declared here, so "no source declares it" is
				// not a second finding: one statement gets one diagnosis.
				found = true
				// An example carries one value. An operator that declares more
				// than one input cannot be stated by such a table: the other
				// inputs would go unstated, and the example would say something
				// the author never wrote. The reader refuses that shape when it
				// decides a rule, but a Workspace nobody reads would then pass
				// validate and fail only at read time. Refuse it here, at the
				// table, so one statement has one diagnosis (#5894).
				multiInputTable := len(operator.Inputs) > 1
				for _, example := range table.Cases {
					if multiInputTable && example.Value.Inputs == nil {
						names := make([]string, 0, len(operator.Inputs))
						for _, input := range operator.Inputs {
							names = append(names, input.Name)
						}
						return r.collectSourceError(pending.filename, &syntax.ParseError{
							Pos: table.OperatorSpan.Start, End: table.OperatorSpan.End,
							Msg: fmt.Sprintf(
								"TEI_OPERATOR_EXAMPLE_SHAPE_MISMATCH: operator %s declares %d inputs (%s), but its examples carry one value; an example can state a single-input operator only",
								operator.ID, len(operator.Inputs), strings.Join(names, ", ")),
						})
					}
				}
				// A column that names no declared input would make an example
				// nobody can check. The declaration is in hand here, so the names
				// are checked now rather than only when a rule is evaluated.
				declared := operatorInputNames(*operator)
				for _, example := range table.Cases {
					for name := range example.Value.Inputs {
						if !declared[name] {
							span, ok := table.InputHeaderSpans[name]
							if !ok {
								span = example.InputSpans[name]
							}
							return r.collectSourceError(pending.filename, &syntax.ParseError{
								Pos: span.Start, End: span.End,
								Msg: fmt.Sprintf("operator %s declares no input named %q; an example column is one of its input names", operator.ID, name),
							})
						}
					}
					// A declared input the example does not state leaves that
					// input unread, so the example is not one the implementation
					// can be asked about. The reader refuses that shape when it
					// decides a rule; refusing it here means a Workspace nobody
					// reads cannot pass validate with an example that would then
					// fail only at read time.
					for _, input := range operatorInputNamesInOrder(*operator) {
						if _, stated := example.Value.Inputs[input]; stated {
							continue
						}
						if input == "value" && len(operator.Inputs) <= 1 {
							// The unnamed single-input form states its value in
							// the value cell, not in Inputs.
							continue
						}
						return r.collectSourceError(pending.filename, &syntax.ParseError{
							Pos: example.ExpectSpan.Start, End: example.ExpectSpan.End,
							Msg: fmt.Sprintf("operator %s declares input %q, but this example states no value for it", operator.ID, input),
						})
					}
					operator.Cases = append(operator.Cases, example.Value)
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
	if err := attachOperatorRowCases(r); err != nil {
		return err
	}
	return nil
}

// attachOperatorRowCases adds every row-example table to the operator it names.
// A table for an operator no source declares stops the Load at the table's
// position. Section 8.1 fixes the rest: one operator may have several row
// examples, but not two that state the same rows, and not a row example beside
// the one-record form.
func attachOperatorRowCases(r *Result) error {
	seen := map[core.CanonicalID][]pendingOperatorRowCases{}
	for _, pending := range r.pendingOperatorRowCases {
		table := pending.table
		fail := func(span syntax.Anchor, message string) error {
			return r.collectSourceError(pending.filename, &syntax.ParseError{
				Pos: span.Start, End: span.End, Msg: message})
		}
		found := false
		for i := range r.World.Semantics {
			for j := range r.World.Semantics[i].Operators {
				operator := &r.World.Semantics[i].Operators[j]
				if operator.ID != table.Operator {
					continue
				}
				found = true
				if !declaresRowScopedInput(*operator) {
					if err := fail(table.OperatorSpan, fmt.Sprintf(
						"operator %s declares no input read over all rows, so its example is marked <!-- tei: cases of %s --> without expecting",
						operator.ID, operator.ID)); err != nil {
						return err
					}
					continue
				}
				if len(operator.Cases) > 0 {
					// The two forms state different things about one operator;
					// section 8.1 refuses the mix rather than reading one of
					// them as the other.
					if err := fail(table.OperatorSpan, fmt.Sprintf(
						"operator %s already has a one-record example; a row example and a one-record example cannot state the same operator",
						operator.ID)); err != nil {
						return err
					}
					continue
				}
				for _, prior := range seen[operator.ID] {
					if !semantic.SameOperatorRowCaseStates(prior.table.Case, table.Case) {
						continue
					}
					if err := fail(table.OperatorSpan, fmt.Sprintf(
						"operator %s already has this row example at %s:%d:%d; one example is stated once",
						operator.ID, filepath.ToSlash(prior.relative),
						prior.table.OperatorSpan.Start.Line, prior.table.OperatorSpan.Start.Column)); err != nil {
						return err
					}
				}
				if err := requireRowCaseNamesMatchSignature(*operator, table, fail); err != nil {
					return err
				}
				operator.RowCases = append(operator.RowCases, table.Case)
			}
		}
		if !found {
			if err := fail(table.OperatorSpan, fmt.Sprintf(
				"the example is for operator %s, but no source declares that operator", table.Operator)); err != nil {
				return err
			}
			continue
		}
		seen[table.Operator] = append(seen[table.Operator], pending)
	}
	r.pendingOperatorRowCases = nil
	return nil
}

// declaresRowScopedInput reports whether some declared input reads every row.
func declaresRowScopedInput(operator semantic.Operator) bool {
	for _, input := range operator.Inputs {
		if input.OfEachRow {
			return true
		}
	}
	return false
}

// requireRowCaseNamesMatchSignature matches the table's columns to the inputs
// the operator declares: a column naming an input it does not take, and a
// declared input the table does not state, are both refused at the table.
func requireRowCaseNamesMatchSignature(
	operator semantic.Operator,
	table syntax.OperatorRowCasesAuthoring,
	fail func(syntax.Anchor, string) error,
) error {
	declared := make(map[string]bool, len(operator.Inputs))
	for _, input := range operator.Inputs {
		if input.Name == "" {
			declared["value"] = true
			continue
		}
		declared[input.Name] = true
	}
	for name, span := range table.InputHeaderSpans {
		if declared[name] {
			continue
		}
		if err := fail(span, fmt.Sprintf(
			"operator %s declares no input named %q; a row example column is one of its input names",
			operator.ID, name)); err != nil {
			return err
		}
	}
	for _, input := range operator.Inputs {
		name := input.Name
		if name == "" {
			name = "value"
		}
		if _, stated := table.InputHeaderSpans[name]; !stated {
			if err := fail(table.OperatorSpan, fmt.Sprintf(
				"operator %s declares input %q, but this row example has no column for it",
				operator.ID, name)); err != nil {
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

// operatorInputNames is the set of names an operator's declared inputs answer to.
// The unnamed single-input form answers to "value".
func operatorInputNames(operator semantic.Operator) map[string]bool {
	names := map[string]bool{}
	if len(operator.Inputs) == 0 {
		names["value"] = true
		return names
	}
	for _, input := range operator.Inputs {
		name := input.Name
		if name == "" {
			name = "value"
		}
		names[name] = true
	}
	return names
}

// operatorInputNamesInOrder is the declared input names in signature order.
// The unnamed single-input form answers to "value".
func operatorInputNamesInOrder(operator semantic.Operator) []string {
	if len(operator.Inputs) == 0 {
		return []string{"value"}
	}
	names := make([]string, 0, len(operator.Inputs))
	for _, input := range operator.Inputs {
		name := input.Name
		if name == "" {
			name = "value"
		}
		names = append(names, name)
	}
	return names
}
