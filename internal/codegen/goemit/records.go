package goemit

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"

	"tei/internal/codegen/record"
)

// Writer is the name goemit gives itself in a Manifest.
const Writer = "goemit"

// TraceLines gives one record.TraceLine for each decision and check function
// in o.Trace, in the order of o.Trace, decision first. The line range is read
// from the emitted file itself: it runs from the first line of the function's
// doc comment, or of the function when it has none, to the line of its
// closing brace. provider names the generator in each line and may be empty.
//
// The files are parsed rather than the lines being remembered by Emit, so the
// ranges stay true for an Output a caller has changed, for example by
// rewriting its comments. A Trace whose file is missing, does not parse or
// does not declare the named function gives an error.
func (o Output) TraceLines(provider string) ([]record.TraceLine, error) {
	var lines []record.TraceLine
	for _, trace := range o.Trace {
		content := o.File(trace.File)
		if content == nil {
			return nil, fmt.Errorf("trace of %s names file %s, which was not emitted", trace.Element, trace.File)
		}
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, trace.File, content, parser.ParseComments)
		if err != nil {
			return nil, fmt.Errorf("trace of %s: %w", trace.Element, err)
		}
		for _, symbol := range []struct{ name, role string }{
			{trace.Decision, "decision"},
			{trace.Check, "check"},
		} {
			fn := findFunc(file, symbol.name)
			if fn == nil {
				return nil, fmt.Errorf("trace of %s: %s does not declare function %s", trace.Element, trace.File, symbol.name)
			}
			start := fn.Pos()
			if fn.Doc != nil {
				start = fn.Doc.Pos()
			}
			lines = append(lines, record.TraceLine{
				Schema:   record.TraceSchema,
				File:     trace.File,
				Lines:    [2]int{fset.Position(start).Line, fset.Position(fn.End()).Line},
				Symbol:   symbol.name,
				Element:  trace.Element,
				Role:     symbol.role,
				Provider: provider,
			})
		}
	}
	return lines, nil
}

func findFunc(file *ast.File, name string) *ast.FuncDecl {
	for _, decl := range file.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok && fn.Recv == nil && fn.Name.Name == name {
			return fn
		}
	}
	return nil
}

// Manifest lists the emitted files with the digest of each, written by
// writer. Input is left empty until the input record exists (#5773).
func (o Output) Manifest(writer string) record.Manifest {
	files := make([]record.FileContent, len(o.Files))
	for i, file := range o.Files {
		files[i] = record.FileContent{Path: file.Path, Content: file.Content}
	}
	return record.NewManifest("", files, writer, "")
}
