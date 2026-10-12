package main

import (
	"fmt"
	"os"
	"path/filepath"

	"tei/internal/index"
	"tei/internal/modelref"
	"tei/internal/modelref/derivedbridge"
	"tei/internal/modelref/sourcebridge"
	"tei/internal/sourceindex"
	"tei/internal/subject"
	"tei/tools/internal/rmtlink"
)

// RMTIdentityLineage is a read-only, scope-bound audit of a source-derived
// token, Review-bound proposal, recorded TEI Decision and actual typed Source
// definition. It is NOT a declaration that an old-system construct and the
// accepted TEI meaning are equivalent or that a human approver is qualified.
type RMTIdentityLineage struct {
	SourceToken	string       `json:"sourceToken"`
	SourceDigest	string       `json:"sourceDigest"`
	CandidateFingerprint	string     `json:"candidateFingerprint"`
	ReviewFingerprint	string       `json:"reviewFingerprint"`
	BeforeRevision	string       `json:"beforeRevision"`
	AfterRevision	string       `json:"afterRevision"`
	Target	modelref.Key `json:"target"`
	DefinitionPath	string       `json:"definitionPath"`
	MeaningStatus	string       `json:"meaningStatus"`
	AuthorityLimits	[]string     `json:"authorityLimits"`
}

// traceRMTIdentityAcrossAcceptance insists on the current verified Decision
// ledger. Neither a caller-constructed Decision nor a successful Preview is
// sufficient. Source bytes, Source Index, Graph and Decision must agree; the
// caller still owns the before snapshot and must not relabel its scope.
func traceRMTIdentityAcrossAcceptance(
	link rmtlink.Link, context rmtlink.Context,
	before, after snapshot, scope string,
) (RMTIdentityLineage, error) {
	fail := func(reason string) (RMTIdentityLineage, error) {
		return RMTIdentityLineage{}, fmt.Errorf("RMT typed identity lineage: %s", reason)
	}
	if scope == "" || before.root == "" || before.root != after.root ||
		before.revision == "" || after.revision == "" || before.revision == after.revision {
		return fail("before and after must be distinct revisions of the same scoped Workspace")
	}
	if err := rmtlink.Verify(link, context); err != nil {
		return fail("source / proposal / Review link is invalid: " + err.Error())
	}
	if link.Status != rmtlink.StatusProposed || link.MeaningStatus != rmtlink.MeaningUnresolved ||
		context.Target.Kind != "validation" || context.Target.Change != "added" {
		return fail("link cannot promote an unreviewed semantic correlation")
	}
	if context.BaseModelRevision != before.revision {
		return fail("source proposal was not evaluated against this before revision")
	}
	ledger, err := loadDecisionRecords(after.root)
	if err != nil {
		return fail("Decision ledger cannot be verified: " + err.Error())
	}
	if ledger.ModelRevision != after.revision {
		return fail("Decision ledger and current after Workspace revision disagree")
	}
	matches := 0
	var record DecisionRecord
	for _, entry := range ledger.Records {
		d := entry.Decision
		if d.CandidateFingerprint == context.CandidateFingerprint &&
			d.ReviewBinding != nil && d.ReviewBinding.Fingerprint == context.ReviewFingerprint {
			matches++
			record = entry
		}
	}
	if matches != 1 {
		return fail("exact candidate and Review must have one recorded Decision")
	}
	d := record.Decision
	if record.Status != "current" || record.SourceEdits != "intact" ||
		record.ReviewBindingStatus != "verified" ||
		d.Schema != decisionSchema || d.Decision != "accepted" || !d.SourceChanged ||
		d.Stale != nil || d.Origin == nil || d.Origin.Kind != "ai" ||
		d.BaseModelRevision != before.revision ||
		d.ModelRevisionBefore != before.revision ||
		d.ModelRevisionAfter != after.revision {
		return fail("recorded AI Decision is stale, rejected or not bound to exact Workspace revisions")
	}
	if d.EvidenceFingerprint == "" || d.EvidenceFingerprint != evidenceFingerprintOf(d.Evidence) {
		return fail("accepted Decision evidence fingerprint does not match its contents")
	}
	sourceMatches := 0
	for _, item := range d.Evidence {
		e, ok := rmtTranscriptionFromEvidence(item)
		if !ok {
			continue
		}
		if e.EvidenceFingerprint == context.Source.EvidenceFingerprint &&
			e.SourceArtifactDigest == context.Source.SourceArtifactDigest &&
			e.SourceToken == context.Source.SourceToken &&
			e.MappingContractFingerprint == context.Source.MappingContractFingerprint {
			sourceMatches++
		}
	}
	if sourceMatches != 1 {
		return fail("accepted Decision is not bound to exactly one source observation")
	}
	diffMatches := 0
	for _, change := range d.SemanticDiff {
		if change.Kind == context.Target.Kind && change.ID == context.Target.ID &&
			change.Change == context.Target.Change {
			diffMatches++
		}
	}
	if diffMatches != 1 {
		return fail("accepted semantic delta does not contain exact reviewed subject")
	}

	key := modelref.Key{Scope: scope, Subject: subject.Subject{
		Kind: "semantic-validation", ID: context.Target.ID,
	}}
	beforeGraph := derivedbridge.FromIndex(index.Build(before.loaded.World), scope)
	afterGraph := derivedbridge.FromIndex(index.Build(after.loaded.World), scope)
	review := modelref.ReviewImpactDiff(beforeGraph.Graph, afterGraph.Graph, afterGraph.Rules)
	if !review.GraphValid {
		return fail("before/after typed reference graphs are unresolved")
	}
	created := 0
	for _, change := range review.Changes {
		if change.Kind == "added" && change.Key == key {
			created++
		}
	}
	if created != 1 {
		return fail("target is not uniquely added in the typed graph")
	}
	for _, ref := range before.index.ReferencesFor(key.Subject) {
		if ref.Role == sourceindex.RoleDefinition {
			return fail("target already had an authoritative definition before Acceptance")
		}
	}
	read := func(path string) ([]byte, error) {
		return os.ReadFile(filepath.Join(after.root, filepath.FromSlash(path)))
	}
	resolver, err := sourcebridge.NewResolver(afterGraph.Graph, after.index, read)
	if err != nil {
		return fail("accepted Source Index cannot be checked: " + err.Error())
	}
	location := resolver.Locate(key)
	if location.Status != "located" {
		return fail("accepted target has no unique Source locator: " + location.Reason)
	}
	definitions, definitionPath := 0, ""
	for _, ref := range location.References {
		if ref.Role == sourceindex.RoleDefinition {
			definitions++
			definitionPath = ref.ArtifactPath
		}
	}
	if definitions != 1 {
		return fail("accepted target has no unique definition anchor")
	}
	return RMTIdentityLineage{
		SourceToken:	context.Source.SourceToken,
		SourceDigest:	context.Source.SourceArtifactDigest,
		CandidateFingerprint:	context.CandidateFingerprint,
		ReviewFingerprint:	context.ReviewFingerprint,
		BeforeRevision:	before.revision,
		AfterRevision:	after.revision,
		Target:	key,
		DefinitionPath:	definitionPath,
		MeaningStatus:	rmtlink.MeaningUnresolved,
		AuthorityLimits:	[]string{
			"synthetic-source-token-is-not-original-system-identity-proof",
			"accepted-tei-meaning-is-not-legacy-semantic-equivalence",
			"origin-ai-is-not-human-approval",
			"recorded-decision-is-not-verified-decision-authority",
			"read-only-audit-does-not-grant-write-or-runtime-authority",
		},
	}, nil
}
