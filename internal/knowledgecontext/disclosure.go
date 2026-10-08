package knowledgecontext

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"tei/internal/index"
	"tei/internal/sourceindex"
	"tei/internal/subject"
)

const knowledgeDisclosureProjectionRevision = "tei.knowledge.disclosure/v1"

var selectableDisclosureSections = []string{
	"data-design",
	"definition",
	"dependencies",
	"source",
}

type DisclosureProjection struct {
	ProjectionRevision       string                                `json:"projectionRevision"`
	Fingerprint              string                                `json:"fingerprint"`
	Status                   string                                `json:"status"`
	Subject                  subject.Subject                       `json:"subject"`
	AvailableSections        []string                              `json:"availableSections"`
	SelectedSections         []string                              `json:"selectedSections"`
	DisclosedSections        []string                              `json:"disclosedSections"`
	UnavailableSections      []string                              `json:"unavailableSections,omitempty"`
	OmittedAvailableSections []string                              `json:"omittedAvailableSections,omitempty"`
	Artifacts                []sourceindex.Artifact                `json:"artifacts,omitempty"`
	References               []sourceindex.Reference               `json:"references,omitempty"`
	SemanticElement          *index.SemanticElementEntry           `json:"semanticElement,omitempty"`
	ApplicationPattern       *index.ApplicationPatternIndexEntry   `json:"applicationPattern,omitempty"`
	DataDesign               *DataDesignContext                    `json:"dataDesign,omitempty"`
	Outgoing                 []Edge                                `json:"outgoing,omitempty"`
	Incoming                 []Edge                                `json:"incoming,omitempty"`
	Unresolved               []string                              `json:"unresolved,omitempty"`
	AuthorityLimits          []string                              `json:"authorityLimits"`
}

func BuildDisclosure(ctx Context, selected []string) (DisclosureProjection, error) {
	normalized, err := normalizeDisclosureSections(selected)
	if err != nil {
		return DisclosureProjection{}, err
	}

	available := availableDisclosureSections(ctx)
	availableSet := make(map[string]struct{}, len(available))
	for _, section := range available {
		availableSet[section] = struct{}{}
	}
	selectedSet := make(map[string]struct{}, len(normalized))
	for _, section := range normalized {
		selectedSet[section] = struct{}{}
	}

	result := DisclosureProjection{
		ProjectionRevision: knowledgeDisclosureProjectionRevision,
		Status:             "complete",
		Subject:            ctx.Subject,
		AvailableSections:  append([]string(nil), available...),
		SelectedSections:   append([]string(nil), normalized...),
		Unresolved:         append([]string(nil), ctx.Unresolved...),
		AuthorityLimits: append(
			append([]string(nil), ctx.AuthorityLimits...),
			"selective-disclosure-is-derived-not-authority",
			"selective-disclosure-is-not-access-control-or-redaction",
			"disclosure-fingerprint-is-cache-identity-not-semantic-proof",
			"selection-does-not-change-qualification-or-canonical-meaning",
			"omitted-knowledge-is-not-negative-evidence",
			"unavailable-requested-section-is-not-empty-meaning",
		),
	}

	for _, section := range normalized {
		if _, ok := availableSet[section]; !ok {
			result.UnavailableSections = append(result.UnavailableSections, section)
			continue
		}
		result.DisclosedSections = append(result.DisclosedSections, section)
		switch section {
		case "source":
			result.Artifacts = append([]sourceindex.Artifact(nil), ctx.Artifacts...)
			result.References = append([]sourceindex.Reference(nil), ctx.References...)
		case "definition":
			if ctx.SemanticElement != nil {
				copy, err := cloneDisclosureValue(ctx.SemanticElement)
				if err != nil {
					return DisclosureProjection{}, err
				}
				result.SemanticElement = copy
			}
			if ctx.ApplicationPattern != nil {
				copy, err := cloneDisclosureValue(ctx.ApplicationPattern)
				if err != nil {
					return DisclosureProjection{}, err
				}
				result.ApplicationPattern = copy
			}
		case "data-design":
			copy, err := cloneDisclosureValue(ctx.DataDesign)
			if err != nil {
				return DisclosureProjection{}, err
			}
			result.DataDesign = copy
		case "dependencies":
			result.Outgoing = append([]Edge(nil), ctx.Outgoing...)
			result.Incoming = append([]Edge(nil), ctx.Incoming...)
		}
	}
	for _, section := range available {
		if _, ok := selectedSet[section]; !ok {
			result.OmittedAvailableSections = append(result.OmittedAvailableSections, section)
		}
	}
	if len(result.UnavailableSections) != 0 {
		result.Status = "partial"
	}

	sort.Strings(result.DisclosedSections)
	sort.Strings(result.UnavailableSections)
	sort.Strings(result.OmittedAvailableSections)
	sort.Strings(result.Unresolved)
	sort.Strings(result.AuthorityLimits)
	result.Fingerprint = disclosureFingerprint(result)
	return result, nil
}

func cloneDisclosureValue[T any](value *T) (*T, error) {
	if value == nil {
		return nil, nil
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("knowledge disclosure snapshot encode failed: %w", err)
	}
	var copy T
	if err := json.Unmarshal(encoded, &copy); err != nil {
		return nil, fmt.Errorf("knowledge disclosure snapshot decode failed: %w", err)
	}
	return &copy, nil
}

func normalizeDisclosureSections(values []string) ([]string, error) {
	known := make(map[string]struct{}, len(selectableDisclosureSections))
	for _, section := range selectableDisclosureSections {
		known[section] = struct{}{}
	}
	selected := map[string]struct{}{}
	for _, raw := range values {
		for _, part := range strings.Split(raw, ",") {
			section := strings.TrimSpace(part)
			if section == "" {
				continue
			}
			if _, ok := known[section]; !ok {
				return nil, fmt.Errorf(
					"unknown knowledge disclosure section %q; supported sections: %s",
					section,
					strings.Join(selectableDisclosureSections, ", "),
				)
			}
			selected[section] = struct{}{}
		}
	}
	if len(selected) == 0 {
		return nil, fmt.Errorf("knowledge disclosure requires at least one selected section")
	}
	result := make([]string, 0, len(selected))
	for section := range selected {
		result = append(result, section)
	}
	sort.Strings(result)
	return result, nil
}

func availableDisclosureSections(ctx Context) []string {
	var result []string
	if ctx.DataDesign != nil {
		result = append(result, "data-design")
	}
	if ctx.SemanticElement != nil || ctx.ApplicationPattern != nil {
		result = append(result, "definition")
	}
	if len(ctx.Outgoing) != 0 || len(ctx.Incoming) != 0 {
		result = append(result, "dependencies")
	}
	if len(ctx.Artifacts) != 0 || len(ctx.References) != 0 {
		result = append(result, "source")
	}
	sort.Strings(result)
	return result
}

func disclosureFingerprint(value DisclosureProjection) string {
	value.Fingerprint = ""
	encoded, err := json.Marshal(value)
	if err != nil {
		panic("Knowledge Disclosure Projection contains non-JSON state: " + err.Error())
	}
	return sourceindex.Fingerprint(encoded)
}
