package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"
	"time"
)

const (
	corpusSchema     = "tei.reference.epistemic-corpus/v0"
	contextSchema    = "tei.reference.epistemic-context/v0"
	projectionSchema = "tei.reference.epistemic-assessment/v0"
)

type Origin string

const (
	OriginUser     Origin = "user"
	OriginAI       Origin = "ai"
	OriginSystem   Origin = "system"
	OriginImported Origin = "imported"
	OriginExternal Origin = "external"
	OriginUnknown  Origin = "unknown"
)

func (v Origin) valid() bool {
	switch v {
	case OriginUser, OriginAI, OriginSystem, OriginImported, OriginExternal, OriginUnknown:
		return true
	default:
		return false
	}
}

type IngestionActor string

const (
	IngestedByUser      IngestionActor = "user"
	IngestedByAI        IngestionActor = "ai"
	IngestedBySystem    IngestionActor = "system"
	IngestedByConnector IngestionActor = "connector"
	IngestedByImport    IngestionActor = "import"
)

func (v IngestionActor) valid() bool {
	switch v {
	case IngestedByUser, IngestedByAI, IngestedBySystem, IngestedByConnector, IngestedByImport:
		return true
	default:
		return false
	}
}

type MetadataState string

const (
	MetadataComplete MetadataState = "complete"
	MetadataPartial  MetadataState = "partial"
	MetadataAbsent   MetadataState = "absent"
)

func (v MetadataState) valid() bool {
	switch v {
	case MetadataComplete, MetadataPartial, MetadataAbsent:
		return true
	default:
		return false
	}
}

type AssertionMode string

const (
	ModeFact        AssertionMode = "fact"
	ModeObservation AssertionMode = "observation"
	ModeRequirement AssertionMode = "requirement"
	ModeDecision    AssertionMode = "decision"
	ModeConstraint  AssertionMode = "constraint"
	ModeHypothesis  AssertionMode = "hypothesis"
	ModeQuestion    AssertionMode = "question"
	ModeInstruction AssertionMode = "instruction"
	ModePreference  AssertionMode = "preference"
	ModeOther       AssertionMode = "other"
)

func (v AssertionMode) valid() bool {
	switch v {
	case ModeFact, ModeObservation, ModeRequirement, ModeDecision, ModeConstraint,
		ModeHypothesis, ModeQuestion, ModeInstruction, ModePreference, ModeOther:
		return true
	default:
		return false
	}
}

func (v AssertionMode) canBeWorkingPremise() bool {
	switch v {
	case ModeFact, ModeObservation, ModeRequirement, ModeDecision, ModeConstraint:
		return true
	default:
		return false
	}
}

type ReviewEventKind string

const (
	ReviewEventConfirm  ReviewEventKind = "confirm"
	ReviewEventReject   ReviewEventKind = "reject"
	ReviewEventWithdraw ReviewEventKind = "withdraw"
)

func (v ReviewEventKind) valid() bool {
	switch v {
	case ReviewEventConfirm, ReviewEventReject, ReviewEventWithdraw:
		return true
	default:
		return false
	}
}

type ScopeProvenance string

const (
	ScopeObserved ScopeProvenance = "observed"
	ScopeDeclared ScopeProvenance = "declared"
	ScopeInferred ScopeProvenance = "inferred"
)

func (v ScopeProvenance) valid() bool {
	return v == ScopeObserved || v == ScopeDeclared || v == ScopeInferred
}

type TextQuoteSelector struct {
	Exact  string `json:"exact"`
	Prefix string `json:"prefix,omitempty"`
	Suffix string `json:"suffix,omitempty"`
}

type TargetAnchor struct {
	SourceRef                 string             `json:"sourceRef"`
	SourceRevisionFingerprint string             `json:"sourceRevisionFingerprint"`
	StructuralPath            string             `json:"structuralPath,omitempty"`
	Quote                     *TextQuoteSelector `json:"quote,omitempty"`
}

type TargetRef struct {
	Kind   string        `json:"kind"`
	ID     string        `json:"id"`
	Anchor *TargetAnchor `json:"anchor,omitempty"`
}

type TargetSegment struct {
	StructuralPath string `json:"structuralPath"`
	Text           string `json:"text"`
}

type TargetSurface struct {
	SourceRef           string          `json:"sourceRef"`
	RevisionFingerprint string          `json:"revisionFingerprint"`
	Segments            []TargetSegment `json:"segments"`
}

type ScopeBinding struct {
	Dimension  string          `json:"dimension"`
	Value      string          `json:"value"`
	Provenance ScopeProvenance `json:"provenance"`
}

type EvidenceRef struct {
	Kind        string `json:"kind"`
	Ref         string `json:"ref"`
	Fingerprint string `json:"fingerprint,omitempty"`
}

const reviewMethodHuman = "human-review"

type ReviewEvent struct {
	Sequence           int             `json:"sequence"`
	ID                 string          `json:"id"`
	Kind               ReviewEventKind `json:"kind"`
	Reviewer           string          `json:"reviewer"`
	Method             string          `json:"method"`
	MeaningFingerprint string          `json:"meaningFingerprint"`
	OccurredAt         string          `json:"occurredAt"`
	Evidence           []EvidenceRef   `json:"evidence,omitempty"`
	Conditions         []string        `json:"conditions,omitempty"`
}

type Assertion struct {
	ID                 string         `json:"id"`
	Target             TargetRef      `json:"target"`
	Text               string         `json:"text"`
	MeaningFingerprint string         `json:"meaningFingerprint"`
	Origin             Origin         `json:"origin"`
	IngestedBy         IngestionActor `json:"ingestedBy"`
	MetadataState      MetadataState  `json:"metadataState,omitempty"`
	Mode               AssertionMode  `json:"assertionMode"`
	Context            []ScopeBinding `json:"context,omitempty"`
	ValidFrom          string         `json:"validFrom,omitempty"`
	ValidUntil         string         `json:"validUntil,omitempty"`
	CheckedAt          string         `json:"checkedAt,omitempty"`
	Evidence           []EvidenceRef  `json:"evidence,omitempty"`
	ReviewEvents       []ReviewEvent  `json:"reviewEvents,omitempty"`
	Supersedes         []string       `json:"supersedes,omitempty"`
	ConflictsWith      []string       `json:"conflictsWith,omitempty"`
	ReviewerCandidates []string       `json:"reviewerCandidates,omitempty"`
}

type Criticality string

const (
	CriticalityLow      Criticality = "low"
	CriticalityNormal   Criticality = "normal"
	CriticalityHigh     Criticality = "high"
	CriticalityCritical Criticality = "critical"
)

func (v Criticality) valid() bool {
	switch v {
	case CriticalityLow, CriticalityNormal, CriticalityHigh, CriticalityCritical:
		return true
	default:
		return false
	}
}

type Dependency struct {
	AssertionID  string      `json:"assertionId"`
	Consumer     string      `json:"consumer"`
	Criticality  Criticality `json:"criticality"`
	MaxAgeSeconds int        `json:"maxAgeSeconds,omitempty"`
}

type CoverageArea struct {
	ID                 string      `json:"id"`
	Target             TargetRef   `json:"target"`
	Criticality        Criticality `json:"criticality"`
	MappedAssertionIDs []string    `json:"mappedAssertionIds,omitempty"`
}

type Corpus struct {
	Schema        string         `json:"schema"`
	Assertions    []Assertion    `json:"assertions"`
	Dependencies  []Dependency   `json:"dependencies,omitempty"`
	CoverageAreas []CoverageArea `json:"coverageAreas,omitempty"`
}

type ContextQuery struct {
	Schema         string          `json:"schema"`
	Bindings       []ScopeBinding  `json:"bindings"`
	AsOf           string          `json:"asOf"`
	TargetSurfaces []TargetSurface `json:"targetSurfaces,omitempty"`
}

var sha256Pattern = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)

func decodeStrict[T any](data []byte) (T, error) {
	var value T
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&value); err != nil {
		return value, err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err == io.EOF {
		return value, nil
	} else if err != nil {
		return value, err
	}
	return value, fmt.Errorf("JSONの後ろに余分な内容があります")
}

func normalizeCorpus(input Corpus) (Corpus, error) {
	input.Schema = strings.TrimSpace(input.Schema)
	if input.Schema != corpusSchema {
		return input, fmt.Errorf("unsupported corpus schema %q", input.Schema)
	}
	if len(input.Assertions) == 0 {
		return input, fmt.Errorf("corpus has no assertions")
	}

	ids := map[string]struct{}{}
	for i := range input.Assertions {
		a, err := normalizeAssertion(input.Assertions[i])
		if err != nil {
			return input, fmt.Errorf("assertions[%d]: %w", i, err)
		}
		if _, exists := ids[a.ID]; exists {
			return input, fmt.Errorf("duplicate assertion id %q", a.ID)
		}
		ids[a.ID] = struct{}{}
		input.Assertions[i] = a
	}
	for i := range input.Assertions {
		for _, id := range append(append([]string{}, input.Assertions[i].Supersedes...), input.Assertions[i].ConflictsWith...) {
			if _, ok := ids[id]; !ok {
				return input, fmt.Errorf("assertion %s references unknown assertion %s", input.Assertions[i].ID, id)
			}
			if id == input.Assertions[i].ID {
				return input, fmt.Errorf("assertion %s cannot reference itself", id)
			}
		}
	}
	if err := validateSupersessionAcyclic(input.Assertions); err != nil {
		return input, err
	}

	for i := range input.Dependencies {
		d := &input.Dependencies[i]
		d.AssertionID = strings.TrimSpace(d.AssertionID)
		d.Consumer = strings.TrimSpace(d.Consumer)
		if _, ok := ids[d.AssertionID]; !ok {
			return input, fmt.Errorf("dependencies[%d] references unknown assertion %q", i, d.AssertionID)
		}
		if d.Consumer == "" || !d.Criticality.valid() || d.MaxAgeSeconds < 0 {
			return input, fmt.Errorf("dependencies[%d] has invalid consumer / criticality / maxAgeSeconds", i)
		}
	}
	coverageAreas, err := normalizeCoverageAreas(input.CoverageAreas, ids)
	if err != nil {
		return input, err
	}
	input.CoverageAreas = coverageAreas
	sort.Slice(input.Assertions, func(i, j int) bool { return input.Assertions[i].ID < input.Assertions[j].ID })
	sort.Slice(input.Dependencies, func(i, j int) bool {
		if input.Dependencies[i].AssertionID != input.Dependencies[j].AssertionID {
			return input.Dependencies[i].AssertionID < input.Dependencies[j].AssertionID
		}
		if input.Dependencies[i].Criticality != input.Dependencies[j].Criticality {
			return input.Dependencies[i].Criticality < input.Dependencies[j].Criticality
		}
		return input.Dependencies[i].Consumer < input.Dependencies[j].Consumer
	})
	return input, nil
}

func validateSupersessionAcyclic(assertions []Assertion) error {
	edges := map[string][]string{}
	for _, assertion := range assertions {
		edges[assertion.ID] = append([]string{}, assertion.Supersedes...)
	}
	state := map[string]int{}
	var visit func(string) error
	visit = func(id string) error {
		switch state[id] {
		case 1:
			return fmt.Errorf("supersession cycle detected at %q", id)
		case 2:
			return nil
		}
		state[id] = 1
		for _, target := range edges[id] {
			if err := visit(target); err != nil {
				return err
			}
		}
		state[id] = 2
		return nil
	}
	for id := range edges {
		if err := visit(id); err != nil {
			return err
		}
	}
	return nil
}

func normalizeTargetRef(target TargetRef) (TargetRef, error) {
	target.Kind = strings.TrimSpace(target.Kind)
	target.ID = strings.TrimSpace(target.ID)
	if target.Kind == "" || target.ID == "" {
		return target, fmt.Errorf("kind and id are required")
	}
	if target.Anchor == nil {
		return target, nil
	}
	anchor := target.Anchor
	anchor.SourceRef = strings.TrimSpace(anchor.SourceRef)
	anchor.SourceRevisionFingerprint = strings.TrimSpace(anchor.SourceRevisionFingerprint)
	anchor.StructuralPath = strings.TrimSpace(anchor.StructuralPath)
	if anchor.SourceRef == "" {
		return target, fmt.Errorf("anchor sourceRef is required")
	}
	if !sha256Pattern.MatchString(anchor.SourceRevisionFingerprint) {
		return target, fmt.Errorf("anchor sourceRevisionFingerprint %q is not exact sha256", anchor.SourceRevisionFingerprint)
	}
	if anchor.Quote != nil && strings.TrimSpace(anchor.Quote.Exact) == "" {
		return target, fmt.Errorf("anchor quote exact is required")
	}
	if anchor.StructuralPath == "" && anchor.Quote == nil {
		return target, fmt.Errorf("anchor requires structuralPath or quote")
	}
	return target, nil
}

func normalizeCoverageAreas(values []CoverageArea, assertionIDs map[string]struct{}) ([]CoverageArea, error) {
	out := make([]CoverageArea, 0, len(values))
	seen := map[string]struct{}{}
	for i, area := range values {
		area.ID = strings.TrimSpace(area.ID)
		if area.ID == "" || !area.Criticality.valid() {
			return nil, fmt.Errorf("coverageAreas[%d] has invalid id or criticality", i)
		}
		if _, ok := seen[area.ID]; ok {
			return nil, fmt.Errorf("duplicate coverage area id %q", area.ID)
		}
		seen[area.ID] = struct{}{}
		target, err := normalizeTargetRef(area.Target)
		if err != nil {
			return nil, fmt.Errorf("coverageAreas[%d] target: %w", i, err)
		}
		area.Target = target
		area.MappedAssertionIDs = normalizeStrings(area.MappedAssertionIDs)
		for _, id := range area.MappedAssertionIDs {
			if _, ok := assertionIDs[id]; !ok {
				return nil, fmt.Errorf("coverageAreas[%d] references unknown assertion %q", i, id)
			}
		}
		out = append(out, area)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func normalizeAssertion(a Assertion) (Assertion, error) {
	a.ID = strings.TrimSpace(a.ID)
	a.Text = strings.TrimSpace(a.Text)
	a.MeaningFingerprint = strings.TrimSpace(a.MeaningFingerprint)
	if a.ID == "" || a.Text == "" {
		return a, fmt.Errorf("id / text are required")
	}
	target, err := normalizeTargetRef(a.Target)
	if err != nil {
		return a, fmt.Errorf("target: %w", err)
	}
	a.Target = target
	if !sha256Pattern.MatchString(a.MeaningFingerprint) {
		return a, fmt.Errorf("meaningFingerprint %q is not exact sha256", a.MeaningFingerprint)
	}
	if a.MetadataState == "" {
		a.MetadataState = MetadataComplete
	}
	if !a.Origin.valid() || !a.IngestedBy.valid() || !a.MetadataState.valid() || !a.Mode.valid() {
		return a, fmt.Errorf("invalid origin / ingestedBy / metadataState / assertionMode")
	}

	context, err := normalizeBindings(a.Context)
	if err != nil {
		return a, fmt.Errorf("context: %w", err)
	}
	a.Context = context
	if err := validateOptionalTime("validFrom", a.ValidFrom); err != nil {
		return a, err
	}
	if err := validateOptionalTime("validUntil", a.ValidUntil); err != nil {
		return a, err
	}
	if err := validateOptionalTime("checkedAt", a.CheckedAt); err != nil {
		return a, err
	}
	if a.ValidFrom != "" && a.ValidUntil != "" {
		from, _ := time.Parse(time.RFC3339, a.ValidFrom)
		until, _ := time.Parse(time.RFC3339, a.ValidUntil)
		if until.Before(from) {
			return a, fmt.Errorf("validUntil precedes validFrom")
		}
	}
	evidence, err := normalizeEvidenceRefs(a.Evidence)
	if err != nil {
		return a, fmt.Errorf("evidence: %w", err)
	}
	a.Evidence = evidence
	reviewEvents, err := normalizeReviewEvents(a.ReviewEvents)
	if err != nil {
		return a, fmt.Errorf("reviewEvents: %w", err)
	}
	a.ReviewEvents = reviewEvents
	a.Supersedes = normalizeStrings(a.Supersedes)
	a.ConflictsWith = normalizeStrings(a.ConflictsWith)
	a.ReviewerCandidates = normalizeStrings(a.ReviewerCandidates)
	return a, nil
}

func normalizeReviewEvents(values []ReviewEvent) ([]ReviewEvent, error) {
	out := make([]ReviewEvent, 0, len(values))
	seenIDs := map[string]struct{}{}
	seenSequences := map[int]struct{}{}

	for i, event := range values {
		event.ID = strings.TrimSpace(event.ID)
		event.Reviewer = strings.TrimSpace(event.Reviewer)
		event.Method = strings.TrimSpace(event.Method)
		event.MeaningFingerprint = strings.TrimSpace(event.MeaningFingerprint)
		event.OccurredAt = strings.TrimSpace(event.OccurredAt)

		if event.Sequence <= 0 || event.ID == "" || event.Reviewer == "" {
			return nil, fmt.Errorf("reviewEvents[%d] requires positive sequence, id, and reviewer", i)
		}
		if !event.Kind.valid() {
			return nil, fmt.Errorf("reviewEvents[%d] has invalid kind %q", i, event.Kind)
		}
		if event.Method != reviewMethodHuman {
			return nil, fmt.Errorf("reviewEvents[%d] method %q is unsupported; v0 requires %q", i, event.Method, reviewMethodHuman)
		}
		if !sha256Pattern.MatchString(event.MeaningFingerprint) {
			return nil, fmt.Errorf("reviewEvents[%d] meaningFingerprint %q is not exact sha256", i, event.MeaningFingerprint)
		}
		if err := validateOptionalTime("reviewEvents.occurredAt", event.OccurredAt); err != nil || event.OccurredAt == "" {
			if err != nil {
				return nil, err
			}
			return nil, fmt.Errorf("reviewEvents[%d].occurredAt is required", i)
		}
		if _, ok := seenIDs[event.ID]; ok {
			return nil, fmt.Errorf("duplicate review event id %q", event.ID)
		}
		if _, ok := seenSequences[event.Sequence]; ok {
			return nil, fmt.Errorf("duplicate review event sequence %d", event.Sequence)
		}
		seenIDs[event.ID] = struct{}{}
		seenSequences[event.Sequence] = struct{}{}

		evidence, err := normalizeEvidenceRefs(event.Evidence)
		if err != nil {
			return nil, fmt.Errorf("reviewEvents[%d] evidence: %w", i, err)
		}
		event.Evidence = evidence
		event.Conditions = normalizeStrings(event.Conditions)
		out = append(out, event)
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].Sequence < out[j].Sequence
	})
	for i := 1; i < len(out); i++ {
		if out[i].Sequence <= out[i-1].Sequence {
			return nil, fmt.Errorf("review event sequence must increase")
		}
		previousAt, _ := time.Parse(time.RFC3339, out[i-1].OccurredAt)
		currentAt, _ := time.Parse(time.RFC3339, out[i].OccurredAt)
		if currentAt.Before(previousAt) {
			return nil, fmt.Errorf(
				"review event occurredAt must not move backward: sequence %d=%s sequence %d=%s",
				out[i-1].Sequence,
				out[i-1].OccurredAt,
				out[i].Sequence,
				out[i].OccurredAt,
			)
		}
	}
	return out, nil
}

func normalizeEvidenceRefs(values []EvidenceRef) ([]EvidenceRef, error) {
	out := make([]EvidenceRef, 0, len(values))
	seen := map[string]struct{}{}
	for i, value := range values {
		value.Kind = strings.TrimSpace(value.Kind)
		value.Ref = strings.TrimSpace(value.Ref)
		value.Fingerprint = strings.TrimSpace(value.Fingerprint)
		if value.Kind == "" || value.Ref == "" {
			return nil, fmt.Errorf("evidence[%d] requires kind and ref", i)
		}
		if value.Fingerprint != "" && !sha256Pattern.MatchString(value.Fingerprint) {
			return nil, fmt.Errorf("evidence[%d] fingerprint %q is not exact sha256", i, value.Fingerprint)
		}
		key := value.Kind + "|" + value.Ref + "|" + value.Fingerprint
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, value)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Kind != out[j].Kind {
			return out[i].Kind < out[j].Kind
		}
		if out[i].Ref != out[j].Ref {
			return out[i].Ref < out[j].Ref
		}
		return out[i].Fingerprint < out[j].Fingerprint
	})
	return out, nil
}

func normalizeContext(input ContextQuery) (ContextQuery, time.Time, error) {
	input.Schema = strings.TrimSpace(input.Schema)
	if input.Schema != contextSchema {
		return input, time.Time{}, fmt.Errorf("unsupported context schema %q", input.Schema)
	}
	bindings, err := normalizeBindings(input.Bindings)
	if err != nil {
		return input, time.Time{}, err
	}
	input.Bindings = bindings
	surfaces, err := normalizeTargetSurfaces(input.TargetSurfaces)
	if err != nil {
		return input, time.Time{}, err
	}
	input.TargetSurfaces = surfaces
	asOf, err := time.Parse(time.RFC3339, strings.TrimSpace(input.AsOf))
	if err != nil {
		return input, time.Time{}, fmt.Errorf("invalid asOf: %w", err)
	}
	input.AsOf = asOf.Format(time.RFC3339)
	return input, asOf, nil
}

func normalizeTargetSurfaces(values []TargetSurface) ([]TargetSurface, error) {
	out := make([]TargetSurface, 0, len(values))
	seen := map[string]struct{}{}
	for i, surface := range values {
		surface.SourceRef = strings.TrimSpace(surface.SourceRef)
		surface.RevisionFingerprint = strings.TrimSpace(surface.RevisionFingerprint)
		if surface.SourceRef == "" || !sha256Pattern.MatchString(surface.RevisionFingerprint) {
			return nil, fmt.Errorf("targetSurfaces[%d] requires sourceRef and exact revisionFingerprint", i)
		}
		if _, ok := seen[surface.SourceRef]; ok {
			return nil, fmt.Errorf("duplicate target surface %q", surface.SourceRef)
		}
		seen[surface.SourceRef] = struct{}{}
		paths := map[string]struct{}{}
		for j := range surface.Segments {
			segment := &surface.Segments[j]
			segment.StructuralPath = strings.TrimSpace(segment.StructuralPath)
			if segment.StructuralPath == "" || segment.Text == "" {
				return nil, fmt.Errorf("targetSurfaces[%d].segments[%d] requires structuralPath and text", i, j)
			}
			if _, ok := paths[segment.StructuralPath]; ok {
				return nil, fmt.Errorf("targetSurfaces[%d] has duplicate structuralPath %q", i, segment.StructuralPath)
			}
			paths[segment.StructuralPath] = struct{}{}
		}
		sort.Slice(surface.Segments, func(i, j int) bool {
			return surface.Segments[i].StructuralPath < surface.Segments[j].StructuralPath
		})
		out = append(out, surface)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].SourceRef < out[j].SourceRef })
	return out, nil
}

func normalizeBindings(values []ScopeBinding) ([]ScopeBinding, error) {
	out := make([]ScopeBinding, 0, len(values))
	seen := map[string]struct{}{}
	for i, value := range values {
		value.Dimension = strings.TrimSpace(value.Dimension)
		value.Value = strings.TrimSpace(value.Value)
		if value.Dimension == "" || value.Value == "" || !value.Provenance.valid() {
			return nil, fmt.Errorf("binding[%d] has invalid dimension / value / provenance", i)
		}
		if _, ok := seen[value.Dimension]; ok {
			return nil, fmt.Errorf("duplicate context dimension %q", value.Dimension)
		}
		seen[value.Dimension] = struct{}{}
		out = append(out, value)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Dimension < out[j].Dimension })
	return out, nil
}

func hasNonInferredScope(values []ScopeBinding) bool {
	for _, value := range values {
		if value.Provenance == ScopeObserved || value.Provenance == ScopeDeclared {
			return true
		}
	}
	return false
}

func validateOptionalTime(name, raw string) error {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	if _, err := time.Parse(time.RFC3339, raw); err != nil {
		return fmt.Errorf("invalid %s: %w", name, err)
	}
	return nil
}

func normalizeStrings(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}
