// Package semanticdomaincoverage is a read-only reference for #2095.
//
// It reports, for a small semantic model, which elements have no accepted
// meaning definition, which elements match more than one definition, and
// whether each element separates an implementation-derived "what" from a
// human-authored "why". It never writes an identifier or a definition back,
// and a proposal that is not accepted is not treated as a definition.
package semanticdomaincoverage

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
)

const (
	InputSchema  = "tei.reference.semantic-domain-coverage-input/v0"
	ReportSchema = "tei.reference.semantic-domain-coverage-report/v0"

	// StatusPassed means no element matches more than one accepted definition.
	StatusPassed = "passed"
	// StatusWarning means at least one element matches more than one accepted
	// definition. The report still lists every element.
	StatusWarning = "warning"
	// StatusRejected means the input itself is inconsistent, for example a
	// duplicate identifier or a definition that names a missing element.
	StatusRejected = "rejected"

	// ElementTyped means exactly one accepted definition names the element.
	ElementTyped = "typed"
	// ElementUntyped means no accepted definition names the element.
	ElementUntyped = "untyped"
	// ElementAmbiguous means more than one accepted definition names the element.
	ElementAmbiguous = "ambiguous"
)

// Element is one canonical item. ID is the identifier and is never changed.
// What is what the implementation shows; Why is the human-authored meaning.
type Element struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	What string `json:"what,omitempty"`
	Why  string `json:"why,omitempty"`
}

// Domain is one meaning definition. Accepted distinguishes a definition that
// a person has accepted from a proposal that is still only a candidate.
type Domain struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Elements []string `json:"elements"`
	Accepted bool     `json:"accepted"`
}

// Input is the model the check reads.
type Input struct {
	Schema   string    `json:"schema"`
	Elements []Element `json:"elements"`
	Domains  []Domain  `json:"domains"`
}

// ElementOutcome is the result for one element.
type ElementOutcome struct {
	ID      string   `json:"id"`
	Status  string   `json:"status"`
	Domains []string `json:"domains,omitempty"`
}

// WhatWhy records whether an element has an implementation description and a
// human-authored meaning. A missing Why is reported, not filled in.
type WhatWhy struct {
	Element string `json:"element"`
	What    bool   `json:"what"`
	Why     bool   `json:"why"`
}

// Report is the result. Untyped and ambiguous elements are not failures of the
// tool; they are what the next authoring pass should look at.
type Report struct {
	Schema          string           `json:"schema"`
	Status          string           `json:"status"`
	AcceptedDomains int              `json:"acceptedDomains"`
	ProposedDomains int              `json:"proposedDomains"`
	Elements        []ElementOutcome `json:"elements"`
	WhatWhy         []WhatWhy        `json:"whatWhy"`
	Problems        []string         `json:"problems,omitempty"`
	Note            string           `json:"note"`
}

// LoadInput reads an input file.
func LoadInput(path string) (Input, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Input{}, err
	}
	var input Input
	if err := json.Unmarshal(data, &input); err != nil {
		return Input{}, fmt.Errorf("decode %s: %w", path, err)
	}
	return input, nil
}

// Check builds the report. A malformed input produces StatusRejected with the
// reasons in Problems; it does not guess a definition.
func Check(input Input) Report {
	report := Report{
		Schema: ReportSchema,
		Note:   "要素の識別子は変更しません。受理されていない提案は定義として扱いません。未記入は失敗ではなく、次に書くものです。",
	}

	var problems []string
	if input.Schema != InputSchema {
		problems = append(problems, fmt.Sprintf("schema is %q, want %q", input.Schema, InputSchema))
	}

	elements := make(map[string]Element, len(input.Elements))
	for _, element := range input.Elements {
		if element.ID == "" {
			problems = append(problems, "element with empty id")
			continue
		}
		if _, exists := elements[element.ID]; exists {
			problems = append(problems, fmt.Sprintf("duplicate element id %q", element.ID))
			continue
		}
		elements[element.ID] = element
	}

	acceptedByElement := make(map[string][]string, len(elements))
	seenDomains := make(map[string]bool, len(input.Domains))
	for _, domain := range input.Domains {
		if domain.ID == "" {
			problems = append(problems, "domain with empty id")
			continue
		}
		if seenDomains[domain.ID] {
			problems = append(problems, fmt.Sprintf("duplicate domain id %q", domain.ID))
			continue
		}
		seenDomains[domain.ID] = true
		if len(domain.Elements) == 0 {
			problems = append(problems, fmt.Sprintf("domain %q has no elements", domain.ID))
		}
		if domain.Accepted {
			report.AcceptedDomains++
		} else {
			report.ProposedDomains++
		}
		for _, elementID := range domain.Elements {
			if _, exists := elements[elementID]; !exists {
				problems = append(problems, fmt.Sprintf("domain %q refers to missing element %q", domain.ID, elementID))
				continue
			}
			if domain.Accepted {
				acceptedByElement[elementID] = append(acceptedByElement[elementID], domain.ID)
			}
		}
	}

	ids := make([]string, 0, len(elements))
	for id := range elements {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	for _, id := range ids {
		domains := acceptedByElement[id]
		sort.Strings(domains)
		outcome := ElementOutcome{ID: id, Domains: domains}
		switch len(domains) {
		case 0:
			outcome.Status = ElementUntyped
		case 1:
			outcome.Status = ElementTyped
		default:
			outcome.Status = ElementAmbiguous
		}
		report.Elements = append(report.Elements, outcome)

		element := elements[id]
		report.WhatWhy = append(report.WhatWhy, WhatWhy{
			Element: id,
			What:    element.What != "",
			Why:     element.Why != "",
		})
	}

	report.Problems = problems
	switch {
	case len(problems) > 0:
		report.Status = StatusRejected
	case hasAmbiguous(report.Elements):
		report.Status = StatusWarning
	default:
		report.Status = StatusPassed
	}
	return report
}

func hasAmbiguous(outcomes []ElementOutcome) bool {
	for _, outcome := range outcomes {
		if outcome.Status == ElementAmbiguous {
			return true
		}
	}
	return false
}

// UnacceptedDomains returns the ids of domains that the input marks as not
// accepted. It is a convenience for callers that want to show them separately.
func UnacceptedDomains(input Input) []string {
	var ids []string
	for _, domain := range input.Domains {
		if !domain.Accepted {
			ids = append(ids, domain.ID)
		}
	}
	sort.Strings(ids)
	return ids
}
