package xeadsource

import (
	"sort"
	"strings"
)

// This file classifies only a small, explicitly bounded set of legacy
// TypeOptions/FieldOptions designations. It is a read-only projection: it
// records what the source literally declares, which separation axis each
// designation can inform, and what remains unresolved.
//
// It deliberately is not a general TypeOptions parser, not a new type
// registry, and not a Driver-compatible evaluator. The legacy readers use
// four mutually inconsistent reading rules (raw substring, exact token
// equality, keyword-prefix extraction, and a VALUES-only pre-tokenization
// that discards its own result), so no single tokenization can be called
// "the" X-TEA meaning. The tokenization below is therefore declared as a
// policy revision rather than presented as legacy parity.

const FieldDesignationNature = "source-fact"

// FieldDesignationTokenizationPolicyRevision names the lexical rule used to
// split a designation string into tokens. Changing the rule must change this
// value so recorded evidence is not silently re-interpreted.
const FieldDesignationTokenizationPolicyRevision = "tei.xeadsource.field-designation-tokenization/v1"

// DesignationKind is the closed set of source designations this reference
// recognizes. Any other token stays unresolved.
type DesignationKind string

const (
	DesignationKubun   DesignationKind = "kubun"
	DesignationValues  DesignationKind = "values"
	DesignationBoolean DesignationKind = "boolean"
	DesignationVirtual DesignationKind = "virtual"
	DesignationNoEdit  DesignationKind = "no-edit"
)

// DesignationKinds lists every recognized kind in report order.
var DesignationKinds = []DesignationKind{
	DesignationKubun,
	DesignationValues,
	DesignationBoolean,
	DesignationVirtual,
	DesignationNoEdit,
}

// SeparationAxis names the distinct meanings a legacy designation can carry.
// The whole point of this reference is that a designation informs some axes
// and explicitly does not inform the others.
type SeparationAxis string

const (
	// AxisCodeIdentity is the stored code value of a field.
	AxisCodeIdentity SeparationAxis = "code-identity"
	// AxisDisplayName is the human-readable name shown for a stored code.
	AxisDisplayName SeparationAxis = "display-name"
	// AxisValueDomain is the set of values a field may take.
	AxisValueDomain SeparationAxis = "value-domain"
	// AxisInputAssist is the input component or helper used while entering.
	AxisInputAssist SeparationAxis = "input-assist"
	// AxisUpdatability is whether the field may be updated after creation.
	AxisUpdatability SeparationAxis = "updatability"
	// AxisPhysicalType is the physical database type or column presence.
	AxisPhysicalType SeparationAxis = "physical-db-type"
	// AxisDerivation is the rule that computes a value instead of storing it.
	AxisDerivation SeparationAxis = "derivation"
	// AxisPersistence is whether the field is stored at all.
	AxisPersistence SeparationAxis = "persistence"
	// AxisDisplayFormat is the rendering format of an already-stored value.
	AxisDisplayFormat SeparationAxis = "display-formatting"
)

// SeparationAxes lists every axis in report order.
var SeparationAxes = []SeparationAxis{
	AxisCodeIdentity,
	AxisDisplayName,
	AxisValueDomain,
	AxisInputAssist,
	AxisUpdatability,
	AxisPhysicalType,
	AxisDerivation,
	AxisPersistence,
	AxisDisplayFormat,
}

type DesignationDiagnostic struct {
	Code   string `json:"code"`
	Detail string `json:"detail,omitempty"`
}

// DesignationToken is one comma-separated token of a designation string.
type DesignationToken struct {
	Ordinal int `json:"ordinal"`
	// Raw is the exact source text of the token, parentheses included. It is
	// the authoritative record; nothing here replaces it with an interpretation.
	Raw   string          `json:"raw"`
	Known bool            `json:"known"`
	Kind  DesignationKind `json:"kind,omitempty"`
	// Payload preserves the declared argument text. A nil pointer means the
	// kind takes no argument or the argument could not be read as declared.
	Payload *string `json:"payload,omitempty"`
	// PayloadPartCount is how many ';' separated parts the payload declared.
	// The parts themselves are not copied: the raw token already holds them.
	PayloadPartCount int `json:"payloadPartCount,omitempty"`
	// Informs lists the axes this designation can legitimately inform.
	Informs []string `json:"informs,omitempty"`
	// DoesNotInform lists axes this designation must never be read as, even
	// when a legacy reader happens to consult it for that purpose.
	DoesNotInform []string `json:"doesNotInform,omitempty"`
	// ResolvedAgainstSource states whether the designation's meaning is fully
	// determined by the model file alone.
	ResolvedAgainstSource bool                    `json:"resolvedAgainstSource"`
	Diagnostics           []DesignationDiagnostic `json:"diagnostics,omitempty"`
}

type FieldDesignationFact struct {
	Locator         string                `json:"locator"`
	SourceName      string                `json:"sourceName"`
	AttributeName   string                `json:"attributeName"`
	Raw             string                `json:"raw"`
	EmptyDeclared   bool                  `json:"emptyDeclared,omitempty"`
	Tokens          []DesignationToken    `json:"tokens"`
	Conflicts       []DesignationConflict `json:"conflicts,omitempty"`
	Nature          string                `json:"nature"`
	AuthorityLimits []string              `json:"authorityLimits"`
}

type DesignationConflict struct {
	Axis  string   `json:"axis"`
	Kinds []string `json:"kinds"`
	Raws  []string `json:"raws,omitempty"`
}

type FieldDesignationFacts struct {
	TokenizationPolicyRevision string                 `json:"tokenizationPolicyRevision"`
	Fields                     []FieldDesignationFact `json:"fields,omitempty"`
}

var fieldDesignationAuthorityLimits = []string{
	"designation-classification-is-source-observation-not-accepted-semantic-meaning",
	"legacy-designation-does-not-by-itself-grant-updatability-physical-type-or-value-identity",
	"unrecognized-designation-must-not-be-split-or-inferred",
	"legacy-reading-rules-are-inconsistent-so-no-tokenization-is-driver-parity",
}

// designationAttributes are the only attributes this reference reads.
var designationAttributes = []string{"TypeOptions", "FieldOptions"}

type designationSpec struct {
	kind     DesignationKind
	paren    bool
	informs  []string
	excludes []string
}

// designationSpecs is the bounded semantic table. Each entry states what the
// designation can inform and, more importantly, what it must not be read as.
var designationSpecs = map[DesignationKind]designationSpec{
	// KUBUN names a code table. The code values and their display names live in
	// an external table named by the model, so the designation declares that a
	// value domain exists without settling its members. That is exactly why it
	// conflicts with VALUES rather than overriding it.
	DesignationKubun: {
		kind:     DesignationKubun,
		paren:    true,
		informs:  []string{string(AxisCodeIdentity), string(AxisValueDomain), string(AxisInputAssist)},
		excludes: []string{string(AxisUpdatability), string(AxisPhysicalType), string(AxisDerivation), string(AxisPersistence)},
	},
	// VALUES lists literal values. Because the same literal is both the stored
	// value and the shown text, it cannot express a display name distinct from
	// the code.
	DesignationValues: {
		kind:     DesignationValues,
		paren:    true,
		informs:  []string{string(AxisValueDomain), string(AxisInputAssist)},
		excludes: []string{string(AxisUpdatability), string(AxisPhysicalType), string(AxisDerivation), string(AxisPersistence), string(AxisDisplayName)},
	},
	// BOOLEAN binds a truth value to a stored literal pair. Which stored
	// literals mean true and false is declared; what the truth means is not.
	DesignationBoolean: {
		kind:     DesignationBoolean,
		paren:    true,
		informs:  []string{string(AxisValueDomain), string(AxisInputAssist)},
		excludes: []string{string(AxisUpdatability), string(AxisPhysicalType), string(AxisDerivation), string(AxisPersistence), string(AxisDisplayName)},
	},
	// VIRTUAL states that the field is not a stored column. It does not state
	// the derivation rule, when it is computed, or whether a value is cached.
	DesignationVirtual: {
		kind:     DesignationVirtual,
		paren:    false,
		informs:  []string{string(AxisDerivation), string(AxisPersistence), string(AxisPhysicalType)},
		excludes: []string{string(AxisValueDomain), string(AxisUpdatability), string(AxisDisplayFormat), string(AxisInputAssist)},
	},
	// NO_EDIT is a rendering designation: it keeps a stored value unformatted
	// so that leading zeros and separators survive. It is never read as an
	// editability rule. In the legacy reader the actual editability authority
	// is the NoUpdate attribute, which this reference does not read.
	DesignationNoEdit: {
		kind:     DesignationNoEdit,
		paren:    false,
		informs:  []string{string(AxisDisplayFormat)},
		excludes: []string{string(AxisUpdatability), string(AxisPhysicalType), string(AxisValueDomain), string(AxisDerivation), string(AxisPersistence)},
	},
}

// ExtractFieldDesignationFacts reads only the declared designation strings of
// known source nodes. It never promotes a designation to accepted meaning and
// never completes an unresolved one.
func ExtractFieldDesignationFacts(doc Document) FieldDesignationFacts {
	out := FieldDesignationFacts{
		TokenizationPolicyRevision: FieldDesignationTokenizationPolicyRevision,
	}
	var walk func(Node)
	walk = func(n Node) {
		if n.Known {
			for _, attributeName := range designationAttributes {
				raw, declared := exactAttribute(n, attributeName)
				if !declared {
					continue
				}
				out.Fields = append(out.Fields,
					classifyFieldDesignation(n, attributeName, raw))
			}
		}
		for _, child := range n.Children {
			walk(child)
		}
	}
	walk(doc.Root)
	sort.SliceStable(out.Fields, func(i, j int) bool {
		if out.Fields[i].Locator != out.Fields[j].Locator {
			return out.Fields[i].Locator < out.Fields[j].Locator
		}
		return out.Fields[i].AttributeName < out.Fields[j].AttributeName
	})
	return out
}

func classifyFieldDesignation(n Node, attributeName, raw string) FieldDesignationFact {
	fact := FieldDesignationFact{
		Locator:         n.Locator,
		SourceName:      n.Name,
		AttributeName:   attributeName,
		Raw:             raw,
		EmptyDeclared:   raw == "",
		Nature:          FieldDesignationNature,
		AuthorityLimits: cloneStrings(fieldDesignationAuthorityLimits),
	}
	for i, part := range splitDesignationTokens(raw) {
		fact.Tokens = append(fact.Tokens, classifyDesignationToken(i, part))
	}
	fact.Conflicts = designationConflicts(fact.Tokens)
	return fact
}

// splitDesignationTokens declares the lexical rule: split on every comma and
// keep each part exactly as written. Commas inside parentheses are not
// protected, matching the legacy outer rule; classifying the resulting
// unbalanced tokens as unresolved is the honest outcome, because the legacy
// readers themselves disagree about whether such a payload survives.
func splitDesignationTokens(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if strings.TrimSpace(part) == "" {
			continue
		}
		out = append(out, part)
	}
	return out
}

func classifyDesignationToken(ordinal int, raw string) DesignationToken {
	token := DesignationToken{Ordinal: ordinal, Raw: raw}
	trimmed := strings.TrimSpace(raw)

	if open, close := parenBalance(trimmed); open != close {
		token.Diagnostics = append(token.Diagnostics, DesignationDiagnostic{
			Code:   "unbalanced-parenthesis-after-comma-split",
			Detail: "the comma split left an unbalanced parenthesis; the declared argument cannot be read",
		})
		return token
	}

	kind, payload, ok := matchDesignation(trimmed)
	if !ok {
		token.Diagnostics = append(token.Diagnostics, unknownTokenDiagnostics(trimmed)...)
		return token
	}

	spec := designationSpecs[kind]
	token.Known = true
	token.Kind = kind
	token.Informs = cloneStrings(spec.informs)
	token.DoesNotInform = cloneStrings(spec.excludes)
	token.Payload = payload
	token.ResolvedAgainstSource = true

	switch kind {
	case DesignationKubun:
		// A code table ID is named here, but its members are rows of an
		// external table. The value domain is therefore not determined by the
		// model file.
		token.ResolvedAgainstSource = false
		token.Diagnostics = append(token.Diagnostics, DesignationDiagnostic{
			Code:   "code-table-content-not-in-source",
			Detail: "only the code table reference is declared; its codes and display names are external",
		})
	case DesignationValues:
		if payload != nil {
			token.PayloadPartCount = countPayloadParts(*payload)
		}
		token.Diagnostics = append(token.Diagnostics, DesignationDiagnostic{
			Code:   "values-conflate-stored-value-and-display-name",
			Detail: "each listed literal serves as both the stored value and the shown text",
		})
	case DesignationBoolean:
		if payload == nil {
			break
		}
		parts := countPayloadParts(*payload)
		token.PayloadPartCount = parts
		switch {
		case parts == 1:
			token.Diagnostics = append(token.Diagnostics, DesignationDiagnostic{
				Code:   "boolean-single-part-collapses-empty-and-false",
				Detail: "with one stored literal, an absent value and the false literal are not distinguishable",
			})
		case parts > 2:
			token.Diagnostics = append(token.Diagnostics, DesignationDiagnostic{
				Code:   "boolean-payload-not-a-true-false-pair",
				Detail: "more than two stored literals were declared for a truth value",
			})
		}
	}
	return token
}

// unknownTokenDiagnostics reports an ambiguity without resolving it. A token
// that begins with a known designation is flagged so a reviewer can look, but
// it is never decomposed into the designations it merely resembles.
func unknownTokenDiagnostics(trimmed string) []DesignationDiagnostic {
	for _, kind := range DesignationKinds {
		if strings.HasPrefix(trimmed, designationName(kind)) {
			return []DesignationDiagnostic{{
				Code:   "ambiguous-concatenation-not-split",
				Detail: "the token starts with a known designation but continues; splitting it would change behavior on an unverified guess",
			}}
		}
	}
	return []DesignationDiagnostic{{
		Code:   "unrecognized-designation-token",
		Detail: "the token is not one of the bounded, reviewed designations",
	}}
}

// designationName is the legacy spelling: the source text is uppercase and the
// readers match it case-sensitively, so a differently-cased token is not the
// same designation.
func designationName(kind DesignationKind) string {
	return strings.ToUpper(strings.ReplaceAll(string(kind), "-", "_"))
}

func matchDesignation(trimmed string) (DesignationKind, *string, bool) {
	for _, kind := range DesignationKinds {
		spec := designationSpecs[kind]
		if spec.paren {
			prefix := designationName(kind) + "("
			if !strings.HasPrefix(trimmed, prefix) {
				continue
			}
			if !strings.HasSuffix(trimmed, ")") {
				return "", nil, false
			}
			payload := trimmed[len(prefix) : len(trimmed)-1]
			return kind, &payload, true
		}
		if trimmed == designationName(kind) {
			return kind, nil, true
		}
	}
	return "", nil, false
}

func parenBalance(value string) (int, int) {
	open, close := 0, 0
	for _, r := range value {
		switch r {
		case '(':
			open++
		case ')':
			close++
		}
	}
	return open, close
}

func countPayloadParts(payload string) int {
	count := 0
	for _, part := range strings.Split(payload, ";") {
		if strings.TrimSpace(part) != "" {
			count++
		}
	}
	return count
}

// designationConflicts reports tokens that claim the same axis with different
// meanings. The legacy readers resolve such a pair silently and in different
// orders in different screens, so this reference records the disagreement and
// picks no winner.
func designationConflicts(tokens []DesignationToken) []DesignationConflict {
	byAxis := map[string]map[DesignationKind][]string{}
	for _, token := range tokens {
		if !token.Known {
			continue
		}
		for _, axis := range token.Informs {
			if byAxis[axis] == nil {
				byAxis[axis] = map[DesignationKind][]string{}
			}
			byAxis[axis][token.Kind] = append(byAxis[axis][token.Kind], token.Raw)
		}
	}
	out := make([]DesignationConflict, 0)
	for axis, kinds := range byAxis {
		if len(kinds) < 2 {
			continue
		}
		conflict := DesignationConflict{Axis: axis}
		for kind := range kinds {
			conflict.Kinds = append(conflict.Kinds, string(kind))
		}
		sort.Strings(conflict.Kinds)
		for _, raws := range kinds {
			conflict.Raws = append(conflict.Raws, raws...)
		}
		sort.Strings(conflict.Raws)
		out = append(out, conflict)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Axis < out[j].Axis })
	return out
}
