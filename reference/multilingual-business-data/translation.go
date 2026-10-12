package multilingualbusinessdata

// This file is a D3 reference for preparing AI translation proposals. It adds
// no AI client, approval authority, plugin resolver, or database writer.

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	"tei/internal/modelref"
)

// TranslationTerm is a scoped, already-confirmed terminological constraint.
// A term belonging to another tenant or semantic attribute is never exposed.
type TranslationTerm struct {
	Tenant       string       `json:"tenant"`
	Field        modelref.Key `json:"field"`
	SourceLocale string       `json:"sourceLocale"`
	TargetLocale string       `json:"targetLocale"`
	Source       string       `json:"source"`
	Target       string       `json:"target"`
}

// TranslationMemory is an approved pair, not an automatically accepted value
// in the current business record. Matching must be exact and scoped.
type TranslationMemory struct {
	Tenant       string       `json:"tenant"`
	Field        modelref.Key `json:"field"`
	Edition      string       `json:"edition"`
	SourceLocale string       `json:"sourceLocale"`
	TargetLocale string       `json:"targetLocale"`
	SourceText   string       `json:"sourceText"`
	TargetText   string       `json:"targetText"`
	Confirmed    bool         `json:"confirmed"`
}

// TranslationContext is the only payload a translation adapter receives.
// In particular it does not include arbitrary repository or tenant records.
type TranslationContext struct {
	SourceText   string            `json:"sourceText"`
	SourceLocale string            `json:"sourceLocale"`
	TargetLocale string            `json:"targetLocale"`
	Terms        []TranslationTerm `json:"terms,omitempty"`
	Protected    []string          `json:"protected,omitempty"`
}

// Translator is supplied by an existing, authorized plugin binding. No
// provider discovery or external API call is made by this package.
type Translator interface {
	Name() string
	External() bool
	Translate(context.Context, TranslationContext) (string, error)
}

// SnapshotReader reloads the current authoritative snapshot. Callers own its
// access control and the transaction/acceptance boundary.
type SnapshotReader func(context.Context) (Snapshot, error)

type TranslationJob struct {
	Source                 SourceRef
	Target                 Key
	ExpectedTargetRevision string
	// The caller must have established read access before calling the probe.
	ReadAuthorized bool
	// External transmission needs an independent, explicit authorization.
	ExternalTransferAuthorized bool
	// "public" or "restricted". The D3 reference never transmits restricted text.
	Classification string
	// Exact, author-specified spans (e.g. "{orderId}", "100 mg"); not an MF2 parser.
	Protected []string
	Glossary  []TranslationTerm
	Memory    []TranslationMemory
}

type TranslationAttempt struct {
	// candidate: unaccepted text; published: an existing approved translation.
	Kind       string
	Diagnostic string
	Proposal   TranslationProposal
	Text       string
	Provider   string
	Source     SourceRef
}

func sourceText(s Snapshot, ref SourceRef) (string, string) {
	if len(Validate(s)) != 0 {
		return "", "invalid-snapshot"
	}
	current, exists := findPublication(s, ref.Key)
	if !exists || current != ref.Revision {
		return "", "source-changed"
	}
	item, exists := findRevision(s, ref)
	if !exists || item.Review != "approved" || item.Origin == "translation" {
		return "", "source-not-approved"
	}
	return item.Text, ""
}

func currentTargetKind(s Snapshot, key Key) (string, string) {
	revision, exists := findPublication(s, key)
	if !exists {
		return "", ""
	}
	item, exists := findRevision(s, SourceRef{Key: key, Revision: revision})
	if !exists {
		return "", "invalid-snapshot"
	}
	if item.Origin == "independent" || item.Origin == "original" {
		return "", "independent-target"
	}
	return revision, ""
}

func selectTerms(job TranslationJob, text string) ([]TranslationTerm, string) {
	var terms []TranslationTerm
	seen := make(map[string]string)
	for _, term := range job.Glossary {
		if term.Tenant != job.Target.Tenant || term.Field != job.Target.Field ||
			term.SourceLocale != job.Source.Key.Locale ||
			term.TargetLocale != job.Target.Locale || !strings.Contains(text, term.Source) {
			continue
		}
		if term.Source == "" || strings.TrimSpace(term.Target) == "" {
			return nil, "invalid-term"
		}
		if existing, found := seen[term.Source]; found {
			if existing != term.Target {
				return nil, "ambiguous-term"
			}
			continue
		}
		seen[term.Source] = term.Target
		terms = append(terms, term)
	}
	return terms, ""
}

func matchMemory(job TranslationJob, source string) (string, string) {
	var text string
	for _, m := range job.Memory {
		if !m.Confirmed || m.Tenant != job.Target.Tenant ||
			m.Field != job.Target.Field || m.Edition != job.Target.Edition ||
			m.SourceLocale != job.Source.Key.Locale ||
			m.TargetLocale != job.Target.Locale || m.SourceText != source {
			continue
		}
		if strings.TrimSpace(m.TargetText) == "" {
			return "", "invalid-memory"
		}
		if text != "" && text != m.TargetText {
			return "", "ambiguous-memory"
		}
		text = m.TargetText
	}
	return text, ""
}

func protect(job TranslationJob, source, candidate string, terms []TranslationTerm) string {
	if strings.TrimSpace(candidate) == "" || !utf8.ValidString(candidate) {
		return "invalid-output"
	}
	seen := map[string]bool{}
	for _, token := range job.Protected {
		if token == "" || seen[token] || !strings.Contains(source, token) {
			return "invalid-protected-token"
		}
		seen[token] = true
		if strings.Count(source, token) != strings.Count(candidate, token) {
			return "protected-token-changed"
		}
	}
	for _, term := range terms {
		if !strings.Contains(candidate, term.Target) {
			return "required-term-missing"
		}
	}
	return ""
}

// PrepareTranslation performs one reference translation attempt. It can skip a
// provider for a current publication or an exact, scoped translation-memory
// match. Generated/memorized output remains an unapproved candidate. Source and
// target revisions are rechecked against a fresh snapshot after generation.
// The real acceptance transaction must independently repeat those checks.
func PrepareTranslation(ctx context.Context, read SnapshotReader, provider Translator, job TranslationJob) TranslationAttempt {
	fail := func(code string) TranslationAttempt { return TranslationAttempt{Diagnostic: code} }
	if !job.ReadAuthorized {
		return fail("read-not-authorized")
	}
	if job.Classification != "public" && job.Classification != "restricted" {
		return fail("invalid-classification")
	}
	if read == nil {
		return fail("missing-snapshot-reader")
	}
	if provider != nil && provider.External() &&
		(!job.ExternalTransferAuthorized || job.Classification != "public") {
		return fail("external-transfer-blocked")
	}
	s, err := read(ctx)
	if err != nil {
		return fail("snapshot-unavailable")
	}
	proposal := TranslationProposal{
		Target: job.Target, ExpectedTargetRevision: job.ExpectedTargetRevision,
		Source: job.Source, Text: "validation-sentinel",
	}
	if code := CheckProposal(s, proposal); code != "candidate-current" {
		return fail(code)
	}
	src, code := sourceText(s, job.Source)
	if code != "" {
		return fail(code)
	}
	current, code := currentTargetKind(s, job.Target)
	if code != "" {
		return fail(code)
	}
	if current != "" {
		if r, ok := findRevision(s, SourceRef{Key: job.Target, Revision: current}); ok &&
			r.BasedOn != nil && *r.BasedOn == job.Source && r.Review == "approved" {
			return TranslationAttempt{
				Kind: "published", Text: r.Text, Source: job.Source,
			}
		}
	}
	terms, code := selectTerms(job, src)
	if code != "" {
		return fail(code)
	}
	for _, token := range job.Protected {
		if token == "" || !strings.Contains(src, token) {
			return fail("invalid-protected-token")
		}
	}
	var output, providerName string
	output, code = matchMemory(job, src)
	if code != "" {
		return fail(code)
	}
	if output != "" {
		providerName = "confirmed-memory"
	} else {
		if provider == nil || strings.TrimSpace(provider.Name()) == "" {
			return fail("missing-provider")
		}
		contextPayload := TranslationContext{
			SourceText: src, SourceLocale: job.Source.Key.Locale,
			TargetLocale: job.Target.Locale, Terms: terms,
			Protected: append([]string(nil), job.Protected...),
		}
		output, err = provider.Translate(ctx, contextPayload)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return fail("canceled")
			}
			return fail("translation-failed")
		}
		providerName = provider.Name()
	}
	if code = protect(job, src, output, terms); code != "" {
		return fail(code)
	}
	latest, err := read(ctx)
	if err != nil {
		return fail("snapshot-unavailable")
	}
	proposal.Text = output
	if code = CheckProposal(latest, proposal); code != "candidate-current" {
		return fail(code)
	}
	fresh, code := sourceText(latest, job.Source)
	if code != "" {
		return fail(code)
	}
	if fresh != src {
		return fail("source-content-changed")
	}
	if _, code = currentTargetKind(latest, job.Target); code != "" {
		return fail(code)
	}
	return TranslationAttempt{
		Kind: "candidate", Proposal: proposal, Text: output,
		Provider: providerName, Source: job.Source,
	}
}
