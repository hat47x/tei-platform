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
	relative := relPath(root, filename)

	// Report every problem of the source before deciding to contribute nothing.
	rejected := false
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
		if len(region.Rules.Issues) == 0 && len(region.Rules.Value.Validations) != len(region.Rules.File.Rules) {
			if err := reject(region.Region.Open, "a rule of this region produced no validation"); err != nil {
				return 0, 0, err
			}
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
	return len(regions), definitions, nil
}
