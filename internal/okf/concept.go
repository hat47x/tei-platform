package okf

import (
	"strings"

	"gopkg.in/yaml.v3"
)

// knownFrontmatterKeys are the keys OKF v0.2 standardizes. Everything else is a
// producer extension: the reader keeps it and lists it, and does not reject it
// (OKF v0.2 ・ゑｽｧ4.1).
var knownFrontmatterKeys = map[string]bool{
	"type": true, "title": true, "description": true, "resource": true, "tags": true,
	"generated": true, "verified": true, "status": true, "stale_after": true,
	"sources": true, "usage_window": true,
	"runtime": true, "parameters": true, "computation": true, "executor": true, "attester": true,
}

// statusValues is OKF v0.2 ・ゑｽｧ5.4's closed set. A value outside it is an
// observation, not a format error: the format does not forbid producer values,
// but a consumer cannot read an unknown status as any of the three.
var statusValues = map[string]bool{
	"draft": true, "stable": true, "deprecated": true,
}

// buildConcept turns one parsed document into a Concept plus every path-valued
// reference it declares. The references are returned separately so the bundle can
// list the unresolved ones once.
func buildConcept(doc rawDoc, existed map[string]bool) (Concept, []ResolvedRef) {
	parsed := doc.document
	concept := Concept{
		ConceptID:             conceptIDFor(doc.path),
		Path:                  doc.path,
		Digest:                doc.digest,
		Links:                 append([]bodyLink{}, doc.body.Links...),
		Footnotes:             append([]bodyFootnote{}, doc.body.Footnotes...),
		Sources:               []Source{},
		Verified:              []Verification{},
		Diagnostics:           append([]Diagnostic{}, parsed.Diagnostics...),
		HasComputationSection: doc.body.HasComputationSection,
		FencedLanguageCount:   len(doc.body.FencedLanguages),
	}

	// A document with no frontmatter block has already been reported as
	// FRONTMATTER_MISSING; the type checks below then add their own finding, which
	// is the same order the upstream reference takes (parse, then validate).
	var refs []ResolvedRef
	concept.Type, concept.TypeClass = readType(doc, &concept.Diagnostics)

	if title, ok := fieldText(parsed.field("title")); ok {
		concept.Title = title
	}
	if description, ok := fieldText(parsed.field("description")); ok {
		concept.Description = description
	}
	concept.Tags = readTags(parsed)

	if field := parsed.field("resource"); field != nil {
		if text, ok := fieldText(field); ok {
			concept.Resource = text
			ref := resolveTarget(doc.path, text, "resource", field.Line, func(candidate string) bool { return existed[candidate] })
			concept.ResourceResolution = ref.Resolution
			concept.ResourceResolved = ref.Resolved
			refs = append(refs, ref)
			if isUnresolved(ref.Resolution) {
				concept.Diagnostics = append(concept.Diagnostics, unresolvedDiagnostic(doc.path, concept.ConceptID, "resource", text, ref))
			}
		}
	}

	concept.Status, concept.StatusAbsent = readStatus(doc, &concept.Diagnostics)
	concept.StaleAfter, concept.Stale = readStaleness(doc, &concept.Diagnostics)
	concept.Generated = readGenerated(doc, &concept.Diagnostics)
	concept.Verified, concept.TrustTierLabel = readVerified(doc, &concept.Diagnostics)

	var more []ResolvedRef
	concept.Runtime, concept.Parameters, concept.Computation, concept.Executor, concept.Attester, more =
		readComputationFamily(doc, existed, &concept.Diagnostics)
	refs = append(refs, more...)

	var sourceRefs []ResolvedRef
	concept.Sources, sourceRefs = readSources(doc, existed, &concept.Diagnostics)
	refs = append(refs, sourceRefs...)

	concept.Diagnostics = append(concept.Diagnostics, footnoteSourceJoin(doc.path, concept)...)

	// Unknown keys: preserved and listed, never rejected. The reader cannot tell
	// a producer's deliberate extension from a typo, so it reports every extra key
	// and leaves the classification to the consumer.
	for _, key := range parsed.Order {
		if !knownFrontmatterKeys[key] {
			concept.UnknownKeys = append(concept.UnknownKeys, key)
		}
	}

	// OKF v0.2 section 13.1 supersedes `timestamp` with `generated.at` and a body
	// `# Citations` list with `sources`. A v0.1 document is still readable; the
	// reader records the legacy shape and deliberately does not synthesize the
	// newer fields from it.
	if field := parsed.field("timestamp"); field != nil {
		concept.Diagnostics = append(concept.Diagnostics, Diagnostic{
			Code:      CodeLegacyTimestampPresent,
			Severity:  SeverityInfo,
			Path:      doc.path,
			ConceptID: concept.ConceptID,
			Line:      field.Line,
			Message:   "`timestamp` is the OKF v0.1 field superseded by `generated.at`; it is read as a legacy value and not as a generation record",
		})
	}
	if doc.body.HasCitationsSection {
		concept.Diagnostics = append(concept.Diagnostics, Diagnostic{
			Code:      CodeLegacyCitationsSection,
			Severity:  SeverityInfo,
			Path:      doc.path,
			ConceptID: concept.ConceptID,
			Message:   "the body has a `# Citations` section, the OKF v0.1 form superseded by `sources`; the reader does not synthesize sources from it",
		})
	}

	concept.Diagnostics = append(concept.Diagnostics, readerAbstentions(doc.path, concept)...)
	concept.FormatConformant = !hasFormatError(concept.Diagnostics)
	return concept, refs
}

// readTags returns `tags` as a list of scalars. A single scalar and a one-element
// list are both common producer forms; both are kept without inventing entries.
func readTags(parsed *parsedDocument) []string {
	field := parsed.field("tags")
	if field == nil {
		return nil
	}
	if text, ok := fieldText(field); ok {
		return []string{text}
	}
	items, ok := sequenceItems(field.Node)
	if !ok {
		return nil
	}
	var tags []string
	for _, item := range items {
		if text, ok := scalarText(item); ok {
			tags = append(tags, text)
		}
	}
	return tags
}

// readType reads `type` as OKF v0.2 ・ゑｽｧ4.1 requires: a short non-empty string. The
// upstream reference accepts any truthy value, which the specification does not;
// this reader follows the specification and records the difference, because a
// consumer that routes on a list or a mapping cannot route at all.
func readType(doc rawDoc, diags *[]Diagnostic) (string, string) {
	field := doc.document.field("type")
	if field == nil {
		*diags = append(*diags, Diagnostic{
			Code:      CodeTypeMissing,
			Severity:  SeverityError,
			Path:      doc.path,
			ConceptID: conceptIDFor(doc.path),
			Line:      1,
			Message:   "OKF v0.2 ・ゑｽｧ11 requires a non-empty `type`; the frontmatter has no type key",
		})
		return "", "missing"
	}
	n := resolveAlias(field.Node)
	if n == nil {
		*diags = append(*diags, Diagnostic{
			Code:      CodeTypeNotString,
			Severity:  SeverityError,
			Path:      doc.path,
			ConceptID: conceptIDFor(doc.path),
			Line:      field.Line,
			Message:   "`type` could not be resolved to a value",
		})
		return "", "not-string"
	}
	if n.Kind != yaml.ScalarNode {
		*diags = append(*diags, Diagnostic{
			Code:      CodeTypeNotString,
			Severity:  SeverityError,
			Path:      doc.path,
			ConceptID: conceptIDFor(doc.path),
			Line:      field.Line,
			Message:   "OKF v0.2 ・ゑｽｧ4.1 defines `type` as a short string, but the value is " + yamlKindName(n.Kind),
		})
		return "", "not-string"
	}
	if n.Tag == "!!null" {
		*diags = append(*diags, Diagnostic{
			Code:      CodeTypeEmpty,
			Severity:  SeverityError,
			Path:      doc.path,
			ConceptID: conceptIDFor(doc.path),
			Line:      field.Line,
			Message:   "`type` is null; OKF v0.2 ・ゑｽｧ11 requires a non-empty value",
		})
		return "", "empty"
	}
	if n.Tag != "!!str" {
		// A non-string scalar (an int, a bool, a timestamp) is not a string in the
		// YAML sense. The raw value is still reported so a human can see what the
		// document said.
		*diags = append(*diags, Diagnostic{
			Code:      CodeTypeNotString,
			Severity:  SeverityError,
			Path:      doc.path,
			ConceptID: conceptIDFor(doc.path),
			Line:      field.Line,
			Message:   "OKF v0.2 ・ゑｽｧ4.1 defines `type` as a string; the value carries the YAML tag " + n.Tag,
		})
		return n.Value, "not-string"
	}
	value := n.Value
	if strings.TrimSpace(value) == "" {
		*diags = append(*diags, Diagnostic{
			Code:      CodeTypeEmpty,
			Severity:  SeverityError,
			Path:      doc.path,
			ConceptID: conceptIDFor(doc.path),
			Line:      field.Line,
			Message:   "`type` is whitespace only; a consumer cannot route on it",
		})
		return value, "empty"
	}
	return value, "string"
}

// readStatus reports the lifecycle value. Absent status is `stable` per ・ゑｽｧ5.4, and
// the absence is recorded so a reader does not lose the difference.
func readStatus(doc rawDoc, diags *[]Diagnostic) (string, bool) {
	field := doc.document.field("status")
	if field == nil {
		*diags = append(*diags, Diagnostic{
			Code:      CodeStatusAbsent,
			Severity:  SeverityInfo,
			Path:      doc.path,
			ConceptID: conceptIDFor(doc.path),
			Message:   "`status` is absent; OKF v0.2 ・ゑｽｧ5.4 defines that as stable",
		})
		return "stable", true
	}
	text, ok := fieldText(field)
	if !ok {
		*diags = append(*diags, Diagnostic{
			Code:      CodeStatusUnknown,
			Severity:  SeverityWarning,
			Path:      doc.path,
			ConceptID: conceptIDFor(doc.path),
			Line:      field.Line,
			Message:   "`status` is not a scalar value and cannot be read as draft, stable, or deprecated",
		})
		return "", false
	}
	if !statusValues[text] {
		*diags = append(*diags, Diagnostic{
			Code:      CodeStatusUnknown,
			Severity:  SeverityWarning,
			Path:      doc.path,
			ConceptID: conceptIDFor(doc.path),
			Line:      field.Line,
			Message:   "`status` " + quoteKey(text) + " is outside draft|stable|deprecated; it is not readable as any of them",
		})
	}
	return text, false
}

// readStaleness reports `stale_after` and whether the concept is stale now. A
// value that is not an ISO 8601 datetime with an explicit offset yields
// "unknown": the specification forbids guessing a zone, and reporting "not stale"
// for an unreadable deadline would assert freshness the document never claimed.
func readStaleness(doc rawDoc, diags *[]Diagnostic) (string, string) {
	field := doc.document.field("stale_after")
	if field == nil {
		return "", "absent"
	}
	text, ok := fieldText(field)
	if !ok {
		*diags = append(*diags, Diagnostic{
			Code:      CodeStaleAfterInvalid,
			Severity:  SeverityWarning,
			Path:      doc.path,
			ConceptID: conceptIDFor(doc.path),
			Line:      field.Line,
			Message:   "`stale_after` is not a scalar value; staleness is unknown",
		})
		return "", "unknown"
	}
	instant, hasOffset, ok := parseISODateTime(text)
	if !ok {
		*diags = append(*diags, Diagnostic{
			Code:      CodeStaleAfterInvalid,
			Severity:  SeverityWarning,
			Path:      doc.path,
			ConceptID: conceptIDFor(doc.path),
			Line:      field.Line,
			Message:   "`stale_after` " + quoteKey(text) + " is not an ISO 8601 datetime; staleness is unknown",
		})
		return text, "unknown"
	}
	if !hasOffset {
		*diags = append(*diags, Diagnostic{
			Code:      CodeStaleAfterWithoutOffset,
			Severity:  SeverityWarning,
			Path:      doc.path,
			ConceptID: conceptIDFor(doc.path),
			Line:      field.Line,
			Message:   "OKF v0.2 ・ゑｽｧ5 requires an explicit UTC offset; " + quoteKey(text) + " names a different instant in every zone, so staleness is unknown",
		})
		return text, "unknown"
	}
	if !clockNow().Before(instant) {
		return text, "stale"
	}
	return text, "fresh"
}

// readGenerated reports the `generated` mapping. Its `by` is REQUIRED within the
// mapping (・ゑｽｧ5.2); a missing `by` is an observation because the document is still
// readable, and the reader does not invent an actor.
func readGenerated(doc rawDoc, diags *[]Diagnostic) *Generation {
	field := doc.document.field("generated")
	if field == nil {
		return nil
	}
	entries, ok := mappingEntries(field.Node)
	if !ok {
		*diags = append(*diags, Diagnostic{
			Code:      CodeGeneratedWithoutBy,
			Severity:  SeverityWarning,
			Path:      doc.path,
			ConceptID: conceptIDFor(doc.path),
			Line:      field.Line,
			Message:   "`generated` is not a mapping, so no actor or time can be read",
		})
		return nil
	}
	generation := &Generation{}
	if text, ok := scalarText(entries["by"]); ok {
		generation.By = text
	}
	if text, ok := scalarText(entries["at"]); ok {
		generation.At = text
	}
	if generation.By == "" {
		*diags = append(*diags, Diagnostic{
			Code:      CodeGeneratedWithoutBy,
			Severity:  SeverityWarning,
			Path:      doc.path,
			ConceptID: conceptIDFor(doc.path),
			Line:      field.Line,
			Message:   "OKF v0.2 ・ゑｽｧ5.2 requires `generated.by`; the mapping has no actor",
		})
	}
	if generation.At == "" {
		*diags = append(*diags, Diagnostic{
			Code:      CodeGeneratedWithoutAt,
			Severity:  SeverityWarning,
			Path:      doc.path,
			ConceptID: conceptIDFor(doc.path),
			Line:      field.Line,
			Message:   "`generated.at` is absent; the last meaningful change time is unknown",
		})
	}
	return generation
}

// readVerified normalizes `verified` to a list of events (・ゑｽｧ5.2 requires treating a
// bare mapping as a one-element list) and derives the advisory trust tier (・ゑｽｧ5.3).
// Neither the tier nor any `human:` label is an identity check, a review record, or
// an acceptance.
func readVerified(doc rawDoc, diags *[]Diagnostic) ([]Verification, string) {
	events := []Verification{}
	field := doc.document.field("verified")
	if field == nil {
		return events, "unverified"
	}
	items, ok := sequenceItems(field.Node)
	if !ok {
		*diags = append(*diags, Diagnostic{
			Code:      CodeVerifiedEntryNotMapping,
			Severity:  SeverityWarning,
			Path:      doc.path,
			ConceptID: conceptIDFor(doc.path),
			Line:      field.Line,
			Message:   "`verified` is neither a mapping nor a list, so no verification event can be read",
		})
		return events, "unverified"
	}
	for _, item := range items {
		entries, ok := mappingEntries(item)
		if !ok {
			*diags = append(*diags, Diagnostic{
				Code:      CodeVerifiedEntryNotMapping,
				Severity:  SeverityWarning,
				Path:      doc.path,
				ConceptID: conceptIDFor(doc.path),
				Line:      field.Line,
				Message:   "a `verified` entry is not a mapping; it is skipped",
			})
			continue
		}
		event := Verification{}
		if text, ok := scalarText(entries["by"]); ok {
			event.By = text
		}
		if text, ok := scalarText(entries["at"]); ok {
			event.At = text
		}
		if event.By == "" {
			*diags = append(*diags, Diagnostic{
				Code:      CodeVerifiedEntryWithoutBy,
				Severity:  SeverityWarning,
				Path:      doc.path,
				ConceptID: conceptIDFor(doc.path),
				Line:      field.Line,
				Message:   "a `verified` entry has no `by`; it records no verifier",
			})
		}
		events = append(events, event)
	}
	if len(events) == 0 {
		*diags = append(*diags, Diagnostic{
			Code:      CodeVerifiedEmptyMapping,
			Severity:  SeverityWarning,
			Path:      doc.path,
			ConceptID: conceptIDFor(doc.path),
			Line:      field.Line,
			Message:   "`verified` carries no event; the concept remains unverified",
		})
	}
	return events, trustTierLabel(events)
}

// trustTierLabel applies OKF v0.2 ・ゑｽｧ5.3. The tier is advisory (the specification
// says so explicitly) and must not be surfaced as access control or acceptance.
func trustTierLabel(events []Verification) string {
	if len(events) == 0 {
		return "unverified"
	}
	for _, event := range events {
		if strings.HasPrefix(event.By, "human:") {
			return "human-reviewed"
		}
	}
	return "machine-confirmed"
}

// readComputationFamily reads the Attested Computation contract keys. Every value
// is recorded as a declaration; nothing here is opened or followed.
func readComputationFamily(doc rawDoc, existed map[string]bool, diags *[]Diagnostic) (
	string, []Parameter, *PathRef, *ExecutorDecl, *PathRef, []ResolvedRef,
) {
	var runtime string
	var parameters []Parameter
	var computation *PathRef
	var executor *ExecutorDecl
	var attester *PathRef
	var refs []ResolvedRef

	if field := doc.document.field("runtime"); field != nil {
		if text, ok := fieldText(field); ok {
			runtime = text
		}
	}
	if field := doc.document.field("parameters"); field != nil {
		if items, ok := sequenceItems(field.Node); ok {
			for _, item := range items {
				entries, ok := mappingEntries(item)
				if !ok {
					continue
				}
				parameter := Parameter{}
				if text, ok := scalarText(entries["name"]); ok {
					parameter.Name = text
				}
				if text, ok := scalarText(entries["type"]); ok {
					parameter.Type = text
				}
				if text, ok := scalarText(entries["required"]); ok {
					parameter.Required = text
				}
				parameters = append(parameters, parameter)
			}
		}
	}

	if field := doc.document.field("computation"); field != nil {
		if text, ok := fieldText(field); ok {
			ref := resolveTarget(doc.path, text, "computation", field.Line, func(candidate string) bool { return existed[candidate] })
			computation = &PathRef{Resource: text, Resolution: ref.Resolution, Resolved: ref.Resolved}
			refs = append(refs, ref)
			*diags = append(*diags, Diagnostic{
				Code:      CodeComputationNotOpened,
				Severity:  SeverityInfo,
				Path:      doc.path,
				ConceptID: conceptIDFor(doc.path),
				Line:      field.Line,
				Message:   "the computation path is recorded and not opened; reading a bundle never loads a computation",
			})
		}
	}

	if field := doc.document.field("executor"); field != nil {
		if entries, ok := mappingEntries(field.Node); ok {
			executor = &ExecutorDecl{}
			if text, ok := scalarText(entries["resource"]); ok {
				executor.Resource = text
				ref := resolveTarget(doc.path, text, "executor.resource", field.Line, func(candidate string) bool { return existed[candidate] })
				executor.Resolution = ref.Resolution
				executor.Resolved = ref.Resolved
				refs = append(refs, ref)
				*diags = append(*diags, Diagnostic{
					Code:      CodeExecutorNotOpened,
					Severity:  SeverityInfo,
					Path:      doc.path,
					ConceptID: conceptIDFor(doc.path),
					Line:      field.Line,
					Message:   "the executor resource is recorded and not opened or executed; it is not a permission to run anything",
				})
			}
			if items, ok := sequenceItems(entries["receipt"]); ok {
				for _, item := range items {
					if text, ok := scalarText(item); ok {
						executor.Receipt = append(executor.Receipt, text)
					}
				}
			}
		}
	}

	if field := doc.document.field("attester"); field != nil {
		if entries, ok := mappingEntries(field.Node); ok {
			attester = &PathRef{}
			if text, ok := scalarText(entries["resource"]); ok {
				attester.Resource = text
				ref := resolveTarget(doc.path, text, "attester.resource", field.Line, func(candidate string) bool { return existed[candidate] })
				attester.Resolution = ref.Resolution
				attester.Resolved = ref.Resolved
				refs = append(refs, ref)
				*diags = append(*diags, Diagnostic{
					Code:      CodeAttesterNotOpened,
					Severity:  SeverityInfo,
					Path:      doc.path,
					ConceptID: conceptIDFor(doc.path),
					Line:      field.Line,
					Message:   "the attester resource is recorded and not opened or executed; an attester file is not a verdict",
				})
			}
		}
	}

	if runtime != "" || parameters != nil || computation != nil || executor != nil || attester != nil {
		*diags = append(*diags, Diagnostic{
			Code:      CodeAttestedComputationDeclared,
			Severity:  SeverityInfo,
			Path:      doc.path,
			ConceptID: conceptIDFor(doc.path),
			Message:   "the concept declares a computation contract; this is a declaration only, with no execution, receipt, or verdict",
		})
	}
	return runtime, parameters, computation, executor, attester, refs
}

// readSources reads the `sources` family and resolves each entry's resource. A
// missing `resource` is reported; a scope descriptor is recorded as unfetchable
// rather than as an error, because OKF v0.2 ・ゑｽｧ5.1 allows it explicitly.
func readSources(doc rawDoc, existed map[string]bool, diags *[]Diagnostic) ([]Source, []ResolvedRef) {
	sources := []Source{}
	var refs []ResolvedRef
	field := doc.document.field("sources")
	if field == nil {
		return sources, refs
	}
	items, ok := sequenceItems(field.Node)
	if !ok {
		*diags = append(*diags, Diagnostic{
			Code:      CodeSourceMissingResource,
			Severity:  SeverityWarning,
			Path:      doc.path,
			ConceptID: conceptIDFor(doc.path),
			Line:      field.Line,
			Message:   "`sources` is neither a mapping nor a list, so no source entry can be read",
		})
		return sources, refs
	}
	for _, item := range items {
		entries, ok := mappingEntries(item)
		if !ok {
			*diags = append(*diags, Diagnostic{
				Code:      CodeSourceMissingResource,
				Severity:  SeverityWarning,
				Path:      doc.path,
				ConceptID: conceptIDFor(doc.path),
				Line:      field.Line,
				Message:   "a `sources` entry is not a mapping; it is skipped",
			})
			continue
		}
		source := Source{}
		if text, ok := scalarText(entries["id"]); ok {
			source.ID = text
		}
		if text, ok := scalarText(entries["title"]); ok {
			source.Title = text
		}
		if text, ok := scalarText(entries["author"]); ok {
			source.Author = text
		}
		if text, ok := scalarText(entries["last_modified"]); ok {
			source.LastModified = text
		}
		if text, ok := scalarText(entries["usage_count"]); ok {
			source.UsageCount = text
		}
		resource, ok := scalarText(entries["resource"])
		if !ok || strings.TrimSpace(resource) == "" {
			source.ResourceKind = "missing"
			*diags = append(*diags, Diagnostic{
				Code:      CodeSourceMissingResource,
				Severity:  SeverityWarning,
				Path:      doc.path,
				ConceptID: conceptIDFor(doc.path),
				Line:      field.Line,
				Message:   "OKF v0.2 ・ゑｽｧ5.1 requires `resource` within a sources entry; this entry has none",
			})
			sources = append(sources, source)
			continue
		}
		source.Resource = resource
		ref := resolveTarget(doc.path, resource, "sources[].resource", field.Line, func(candidate string) bool { return existed[candidate] })
		source.Resolution = ref.Resolution
		source.Resolved = ref.Resolved
		refs = append(refs, ref)
		switch ref.Resolution {
		case ResolutionAbsoluteURL:
			source.ResourceKind = "absolute-url"
			*diags = append(*diags, Diagnostic{
				Code:      CodeURLNotFetched,
				Severity:  SeverityInfo,
				Path:      doc.path,
				ConceptID: conceptIDFor(doc.path),
				Line:      field.Line,
				Message:   "the source names an URL; the reader does not fetch it, so retrieval success is not evaluated",
			})
		case ResolutionScopeDescriptor:
			source.ResourceKind = "scope-descriptor"
			*diags = append(*diags, Diagnostic{
				Code:      CodeScopeDescriptorNotFetched,
				Severity:  SeverityInfo,
				Path:      doc.path,
				ConceptID: conceptIDFor(doc.path),
				Line:      field.Line,
				Message:   "the source names a population or scope; OKF v0.2 ・ゑｽｧ5.1 permits this and the reader cannot follow it",
			})
		default:
			source.ResourceKind = "bundle-path"
		}
		if isUnresolved(ref.Resolution) {
			*diags = append(*diags, unresolvedDiagnostic(doc.path, conceptIDFor(doc.path), "sources[].resource", resource, ref))
		}
		sources = append(sources, source)
	}
	return sources, refs
}

// footnoteSourceJoin checks that a `sources[].id` and the body footnotes agree in
// both directions. OKF v0.2 ・ゑｽｧ5.1 makes the footnote label the join key; a label
// with no entry, or an entry no claim cites, is a quality observation.
func footnoteSourceJoin(path string, concept Concept) []Diagnostic {
	if len(concept.Footnotes) == 0 && len(concept.Sources) == 0 {
		return nil
	}
	var diags []Diagnostic
	sourceIDs := map[string]bool{}
	for _, source := range concept.Sources {
		if source.ID != "" {
			sourceIDs[source.ID] = true
		}
	}
	cited := map[string]bool{}
	for _, footnote := range concept.Footnotes {
		cited[footnote.Label] = true
		if !sourceIDs[footnote.Label] {
			diags = append(diags, Diagnostic{
				Code:      CodeFootnoteWithoutSourceID,
				Severity:  SeverityWarning,
				Path:      path,
				ConceptID: concept.ConceptID,
				Line:      footnote.Line,
				Message:   "footnote " + quoteKey(footnote.Label) + " does not match any `sources[].id`; OKF v0.2 ・ゑｽｧ5.1 makes the label the join key",
			})
		}
		if !footnote.Defined {
			diags = append(diags, Diagnostic{
				Code:      CodeFootnoteWithoutSourceID,
				Severity:  SeverityWarning,
				Path:      path,
				ConceptID: concept.ConceptID,
				Line:      footnote.Line,
				Message:   "footnote " + quoteKey(footnote.Label) + " is referenced but has no definition",
			})
		}
	}
	for _, source := range concept.Sources {
		if source.ID != "" && !cited[source.ID] {
			diags = append(diags, Diagnostic{
				Code:      CodeSourceIDWithoutFootnote,
				Severity:  SeverityInfo,
				Path:      path,
				ConceptID: concept.ConceptID,
				Message:   "source id " + quoteKey(source.ID) + " is not cited by any footnote; per-claim attribution is incomplete",
			})
		}
	}
	return diags
}

// readerAbstentions records, per concept, the actions the reader declined to take.
// These are informational: they exist so a consumer of the output cannot assume a
// URL was reached or a linked file was opened.
func readerAbstentions(path string, concept Concept) []Diagnostic {
	var diags []Diagnostic
	if len(concept.Links) > 0 {
		diags = append(diags, Diagnostic{
			Code:      CodeLinksNotEnumerated,
			Severity:  SeverityInfo,
			Path:      path,
			ConceptID: concept.ConceptID,
			Message:   itoa(len(concept.Links)) + " body link(s) were scanned with a structural Markdown reader; destinations are recorded but not fetched",
		})
	}
	diags = append(diags, Diagnostic{
		Code:      CodeHistoryNotAssessed,
		Severity:  SeverityInfo,
		Path:      path,
		ConceptID: concept.ConceptID,
		Message:   "this read is one snapshot of the current bytes; A-B-A and prior review or receipt state are not assessed",
	})
	return diags
}

// unresolvedDiagnostic turns a failed path resolution into an observation. The
// severity is Warning, never Error: OKF v0.2 ・ゑｽｧ6.1 says a broken link "is not
// malformed".
func unresolvedDiagnostic(path, conceptID, kind, target string, ref ResolvedRef) Diagnostic {
	code := CodeLinkUnresolved
	message := "the " + kind + " reference " + quoteKey(target) + " matches no file in the bundle; OKF v0.2 ・ゑｽｧ6.1 tolerates this"
	switch {
	case escapesBundle(path, target):
		code = CodePathEscapesBundle
		message = "the " + kind + " reference " + quoteKey(target) + " attempts to leave the bundle root; the reader does not follow it"
	case ref.Resolution == ResolutionAmbiguous:
		code = CodePathAmbiguousBase
		message = "the " + kind + " reference " + quoteKey(target) + " resolves against both the document directory and the bundle root, to different files: " + strings.Join(ref.Resolved, ", ")
	case ref.Resolution == ResolutionInvalid:
		code = CodePathEscapesBundle
		message = "the " + kind + " reference " + quoteKey(target) + " cannot be interpreted inside the bundle"
	}
	return Diagnostic{
		Code:      code,
		Severity:  SeverityWarning,
		Path:      path,
		ConceptID: conceptID,
		Line:      ref.Line,
		Message:   message,
	}
}

// yamlKindName names a YAML node kind for a diagnostic message.
func yamlKindName(kind yaml.Kind) string {
	switch kind {
	case yaml.DocumentNode:
		return "a document"
	case yaml.SequenceNode:
		return "a list"
	case yaml.MappingNode:
		return "a mapping"
	case yaml.ScalarNode:
		return "a scalar"
	case yaml.AliasNode:
		return "an alias"
	}
	return "an unknown node"
}
