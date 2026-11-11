package reacandidate

import (
	"sort"
	"strings"
)

// CarrierFit is the Fit/Gap answer to "can the existing read-only
// application API carry this external occurrence without loss?".
//
// Each fact a candidate carries is either mapped to an existing read-only
// type/field or recorded as a gap. A gap is reported to the existing Owners
// (#1158/#3419); this package never invents a second host, reader, registry
// or meaning store to close one.
type CarrierFit struct {
	Fact        string `json:"fact"`
	Destination string `json:"destination,omitempty"`
	Gap         string `json:"gap,omitempty"`
	Present     int    `json:"present"`
	Total       int    `json:"total"`
}

// CarrierAudit is the complete Fit/Gap audit for one candidate set.
type CarrierAudit struct {
	SchemaVersion   string       `json:"schemaVersion"`
	Effect          string       `json:"effect"`
	Carried         []CarrierFit `json:"carried"`
	Missing         []CarrierFit `json:"missing"`
	AuthorityLimits []string     `json:"authorityLimits"`
}

// CarrierAuthorityLimits states what the audit does not establish.
func CarrierAuthorityLimits() []string {
	return []string{
		"carrier-audit-is-a-static-type-mapping",
		"a-carried-fact-is-not-an-accepted-meaning",
		"a-gap-is-reported-to-the-existing-owners-not-closed-here",
		"audit-does-not-add-a-second-host-reader-registry-or-meaning-store",
		"audit-does-not-observe-a-real-provider-invocation",
	}
}

type carrierFact struct {
	fact        string
	destination string
	gap         string
	applies     func(StructureCandidate) bool
}

// carrierFacts is the fixed fact table. Destinations name the existing
// read-only types that can carry a fact today; an empty destination is a gap.
func carrierFacts() []carrierFact {
	return []carrierFact{
		{
			fact:        "source-artifact-path",
			destination: "sourceindex.Artifact.Path / sourceindex.Reference.ArtifactPath",
			applies:     func(c StructureCandidate) bool { return c.SourceRevision.ArtifactPath != "" },
		},
		{
			fact:        "source-artifact-digest",
			destination: "sourceindex.Artifact.Fingerprint",
			applies:     func(c StructureCandidate) bool { return c.SourceRevision.Fingerprint != "" },
		},
		{
			fact:        "source-span",
			destination: "sourceindex.Range (offset/line/column)",
			gap:         "external line/column without an offset cannot fill sourceindex.Position.Offset, and no read-only type carries a provider-reported span that was never indexed",
			applies:     func(c StructureCandidate) bool { return c.SourceRevision.Span != nil },
		},
		{
			fact:        "external-node-identity",
			gap:         "operations.SubjectInspection and operations.ModelMapLink identify a Canonical subject by kind+ID; an external node identity has no field",
			applies: func(c StructureCandidate) bool {
				return c.Kind == CandidateKindDataElement && c.Facts.ExternalIdentity != ""
			},
		},
		{
			fact:        "external-observation-identity",
			gap:         "no read-only type carries an external occurrence identity",
			applies:     func(c StructureCandidate) bool { return c.Facts.ExternalOccurrenceID != "" },
		},
		{
			fact:        "external-edge-identity",
			gap:         "ModelMapLink has no edge identity; parallel edges between the same endpoints cannot be told apart",
			applies: func(c StructureCandidate) bool {
				return c.Kind == CandidateKindCrossReference && c.Facts.ExternalIdentity != ""
			},
		},
		{
			fact:        "relation-direction",
			destination: "operations.ModelMapLink.From/To ordering; threeelement.CrossReference.Directed with FromID/ToID",
			applies: func(c StructureCandidate) bool {
				return c.Kind == CandidateKindCrossReference && c.Facts.Direction != nil
			},
		},
		{
			fact:        "relation-label",
			destination: "operations.ModelMapLink.Relation; threeelement.CrossReference.Role",
			applies: func(c StructureCandidate) bool {
				return c.Kind == CandidateKindCrossReference && c.Facts.ObservedKind != ""
			},
		},
		{
			fact:        "external-node-kind",
			gap:         "an external analyzer node kind (module, class, table) has no read-only field and no three-element vocabulary of its own",
			applies: func(c StructureCandidate) bool {
				return c.Kind == CandidateKindDataElement && c.Facts.ObservedKind != ""
			},
		},
		{
			fact:        "external-endpoint-identities",
			gap:         "ModelMapLink endpoints are Canonical subjects; an external endpoint identity has no field",
			applies: func(c StructureCandidate) bool {
				return c.Kind == CandidateKindCrossReference && c.Facts.Direction != nil
			},
		},
		{
			fact:        "observed-state",
			gap:         "no read-only type carries the provider-reported observation state",
			applies:     func(c StructureCandidate) bool { return c.Facts.ObservedState != "" },
		},
		{
			fact:        "observed-authority",
			gap:         "no read-only type carries the provider-reported observation authority",
			applies:     func(c StructureCandidate) bool { return c.Facts.ObservedAuthority != "" },
		},
		{
			fact:        "evidence-coverage",
			gap:         "ModelMap carries the fixed string partial-projection, not a per-occurrence coverage statement",
			applies:     func(c StructureCandidate) bool { return c.Facts.EvidenceCoverage != "" },
		},
		{
			fact:        "suggested-display-name",
			destination: "threeelement.DataElement.Name (as a review suggestion only)",
			applies: func(c StructureCandidate) bool {
				return c.Kind == CandidateKindDataElement && c.Inference.SuggestedName != ""
			},
		},
	}
}

// attemptFacts audits the attempt-level provenance facts.
func attemptFacts() []carrierFact {
	return []carrierFact{
		{
			fact:    "attempt-evidence-id",
			gap:     "no read-only type carries the external attempt identity",
			applies: func(StructureCandidate) bool { return true },
		},
		{
			fact:    "provider-descriptor",
			gap:     "operations carries a Workspace identity, not an external provider id/version",
			applies: func(StructureCandidate) bool { return true },
		},
		{
			fact:        "subject-digest",
			destination: "sourceindex.Fingerprint over the same bytes, once the bytes are read into a workspace",
			applies:     func(StructureCandidate) bool { return true },
		},
		{
			fact:    "attempt-qualification",
			gap:     "no read-only type carries a provider qualification (complete/partial/unknown)",
			applies: func(StructureCandidate) bool { return true },
		},
		{
			fact:    "provider-unknowns",
			gap:     "no read-only type carries the provider's declared unknowns",
			applies: func(StructureCandidate) bool { return true },
		},
		{
			fact:        "canonical-subject-resolution",
			destination: "operations.InspectSubject, but only for a Canonical subject the caller already names",
			gap:         "an external occurrence cannot be resolved by the existing read-only operation without an explicit Owner mapping",
			applies:     func(StructureCandidate) bool { return true },
		},
	}
}

// AuditCarrierFit maps every fact the candidate set carries onto the existing
// read-only types and records what has no home yet.
func AuditCarrierFit(set CandidateSet) CarrierAudit {
	audit := CarrierAudit{
		SchemaVersion:   candidateCarrierSchemaVers,
		Effect:          EffectReadOnly,
		AuthorityLimits: CarrierAuthorityLimits(),
	}
	total := len(set.Candidates)
	appendFit := func(fit CarrierFit) {
		if fit.Destination != "" {
			audit.Carried = append(audit.Carried, fit)
			return
		}
		audit.Missing = append(audit.Missing, fit)
	}
	for _, fact := range carrierFacts() {
		fit := CarrierFit{Fact: fact.fact, Destination: fact.destination, Gap: fact.gap, Total: total}
		for _, candidate := range set.Candidates {
			if fact.applies(candidate) {
				fit.Present++
			}
		}
		appendFit(fit)
	}
	for _, fact := range attemptFacts() {
		fit := CarrierFit{Fact: fact.fact, Destination: fact.destination, Gap: fact.gap, Total: 1, Present: 1}
		appendFit(fit)
	}
	sort.Slice(audit.Carried, func(i, j int) bool { return audit.Carried[i].Fact < audit.Carried[j].Fact })
	sort.Slice(audit.Missing, func(i, j int) bool { return audit.Missing[i].Fact < audit.Missing[j].Fact })
	return audit
}

// MissingFacts lists the fact names with no existing read-only destination.
func (a CarrierAudit) MissingFacts() []string {
	out := make([]string, 0, len(a.Missing))
	for _, fit := range a.Missing {
		out = append(out, fit.Fact)
	}
	sort.Strings(out)
	return out
}

// Describe renders the audit as one line per gap for a review summary.
func (a CarrierAudit) Describe() []string {
	lines := make([]string, 0, len(a.Missing))
	for _, fit := range a.Missing {
		line := fit.Fact + ": " + fit.Gap
		lines = append(lines, strings.TrimSpace(line))
	}
	sort.Strings(lines)
	return lines
}
