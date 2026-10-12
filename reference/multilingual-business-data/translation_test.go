package multilingualbusinessdata

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type testTranslator struct {
	name     string
	external bool
	reply    string
	calls    int
	input    TranslationContext
	onCall   func()
	err      error
}

func (t *testTranslator) Name() string { return t.name }
func (t *testTranslator) External() bool { return t.external }
func (t *testTranslator) Translate(_ context.Context, input TranslationContext) (string, error) {
	t.calls++
	t.input = input
	if t.onCall != nil {
		t.onCall()
	}
	return t.reply, t.err
}

func translationFixture() (Snapshot, TranslationJob) {
	s, ja, en := example()
	newOriginal := Revision{
		Key: ja, Revision: "ja-2", Text: "商品 {orderId} を 100 mg で保管してください。",
		Origin: "original", Review: "approved",
	}
	s.Revisions = append(s.Revisions, newOriginal)
	s.Publications[0].Revision = "ja-2"
	job := TranslationJob{
		Source: SourceRef{Key: ja, Revision: "ja-2"},
		Target: en, ExpectedTargetRevision: "en-1",
		ReadAuthorized: true, Classification: "restricted",
		Protected: []string{"{orderId}", "100 mg"},
		Glossary: []TranslationTerm{{
			Tenant: en.Tenant, Field: en.Field,
			SourceLocale: ja.Locale, TargetLocale: en.Locale,
			Source: "保管", Target: "storage",
		}},
	}
	return s, job
}

func readerFor(s *Snapshot) SnapshotReader {
	return func(context.Context) (Snapshot, error) { return *s, nil }
}

func TestTranslationCandidateRemainsUnpublished(t *testing.T) {
	s, job := translationFixture()
	provider := &testTranslator{
		name: "offline-test-double", reply: "Use storage for {orderId} at 100 mg.",
	}
	out := PrepareTranslation(context.Background(), readerFor(&s), provider, job)
	if out.Diagnostic != "" || out.Kind != "candidate" ||
		out.Provider != "offline-test-double" || out.Proposal.Text != provider.reply ||
		out.Proposal.Source != job.Source || out.Proposal.Target != job.Target {
		t.Fatalf("incorrect proposal: %+v", out)
	}
	if provider.calls != 1 || provider.input.SourceText != "商品 {orderId} を 100 mg で保管してください。" ||
		len(provider.input.Terms) != 1 || len(provider.input.Protected) != 2 {
		t.Fatalf("translator did not receive scoped context: %+v", provider.input)
	}
	if got := Select(s, job.Target); got.Diagnostic != "stale-translation" {
		t.Fatalf("generated text modified publication: %+v", got)
	}
	if got := CheckProposal(s, out.Proposal); got != "candidate-current" {
		t.Fatalf("candidate failed to preserve D1 contract: %s", got)
	}
}

func TestReusePublishedApprovedTranslationWithoutProvider(t *testing.T) {
	s, ja, en := example()
	job := TranslationJob{
		Source: SourceRef{Key: ja, Revision: "ja-1"},
		Target: en, ExpectedTargetRevision: "en-1",
		ReadAuthorized: true, Classification: "public",
	}
	provider := &testTranslator{name: "unused", reply: "wrong"}
	out := PrepareTranslation(context.Background(), readerFor(&s), provider, job)
	if out.Kind != "published" || out.Text != "Store safely." ||
		out.Diagnostic != "" || provider.calls != 0 {
		t.Fatalf("fresh approved publication must be reused: %+v, calls=%d", out, provider.calls)
	}
}

func TestExactScopedMemoryReusesTextButNeverApprovesIt(t *testing.T) {
	s, job := translationFixture()
	job.Glossary = nil
	job.Memory = []TranslationMemory{
		{Tenant: "another-tenant", Field: job.Target.Field, Edition: "catalog",
			SourceLocale: "ja-JP", TargetLocale: "en-US", SourceText: s.Revisions[2].Text,
			TargetText: "Other tenant translation", Confirmed: true},
		{Tenant: job.Target.Tenant, Field: job.Target.Field, Edition: job.Target.Edition,
			SourceLocale: "ja-JP", TargetLocale: "en-US", SourceText: s.Revisions[2].Text,
			TargetText: "Keep {orderId} at 100 mg.", Confirmed: true},
	}
	provider := &testTranslator{name: "unused", reply: "not needed"}
	out := PrepareTranslation(context.Background(), readerFor(&s), provider, job)
	if out.Kind != "candidate" || out.Text != "Keep {orderId} at 100 mg." ||
		out.Provider != "confirmed-memory" || provider.calls != 0 {
		t.Fatalf("confirmed translation memory must be proposed, not published: %+v", out)
	}
	if got := Select(s, job.Target); got.Diagnostic != "stale-translation" {
		t.Fatalf("memory changed published text: %+v", got)
	}
}

func TestAmbiguousScopedMemoryFailsClosed(t *testing.T) {
	s, job := translationFixture()
	job.Glossary = nil
	m := TranslationMemory{Tenant: job.Target.Tenant, Field: job.Target.Field,
		Edition: job.Target.Edition, SourceLocale: "ja-JP", TargetLocale: "en-US",
		SourceText: s.Revisions[2].Text, TargetText: "Keep {orderId} at 100 mg.", Confirmed: true}
	job.Memory = []TranslationMemory{m, m}
	job.Memory[1].TargetText = "Store {orderId} at 100 mg."
	p := &testTranslator{name: "must-not-call"}
	out := PrepareTranslation(context.Background(), readerFor(&s), p, job)
	if out.Diagnostic != "ambiguous-memory" || p.calls != 0 {
		t.Fatalf("ambiguous memory selected implicitly: %+v", out)
	}
}

func TestReadAndExternalTransferGuards(t *testing.T) {
	for _, tc := range []struct {
		name          string
		read          bool
		class         string
		transfer      bool
		external      bool
		expectedError string
	}{
		{"no read authorization", false, "public", true, true, "read-not-authorized"},
		{"private remote forbidden", true, "restricted", true, true, "external-transfer-blocked"},
		{"no remote permission", true, "public", false, true, "external-transfer-blocked"},
		{"bad classification", true, "", false, false, "invalid-classification"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, job := translationFixture()
			job.ReadAuthorized, job.Classification = tc.read, tc.class
			job.ExternalTransferAuthorized = tc.transfer
			provider := &testTranslator{name: "external-test-double", external: tc.external, reply: "Store {orderId} at 100 mg in storage."}
			reads := 0
			read := func(context.Context) (Snapshot, error) {
				reads++
				return s, nil
			}
			out := PrepareTranslation(context.Background(), read, provider, job)
			if out.Diagnostic != tc.expectedError || provider.calls != 0 || reads != 0 {
				t.Fatalf("information leaked before permission check: %+v calls=%d reads=%d", out, provider.calls, reads)
			}
		})
	}
}

func TestOnlyScopedGlossaryIsTransmitted(t *testing.T) {
	s, job := translationFixture()
	foreign := job.Glossary[0]
	foreign.Tenant = "another-tenant"
	foreign.Target = "SECRET TERM"
	job.Glossary = append(job.Glossary, foreign)
	p := &testTranslator{name: "local", reply: "Store {orderId} at 100 mg in storage."}
	out := PrepareTranslation(context.Background(), readerFor(&s), p, job)
	if out.Diagnostic != "" || len(p.input.Terms) != 1 ||
		p.input.Terms[0].Target == "SECRET TERM" {
		t.Fatalf("foreign glossary was exposed: %+v %+v", out, p.input.Terms)
	}
}

func TestTranslationOutputInvariantViolations(t *testing.T) {
	for _, tc := range []struct {
		name, text, expected string
	}{
		{"missing placeholder", "Keep at 100 mg in storage.", "protected-token-changed"},
		{"numeric constant changed", "Store {orderId} at 200 mg in storage.", "protected-token-changed"},
		{"missing mandatory term", "Keep {orderId} at 100 mg.", "required-term-missing"},
		{"empty response", "   ", "invalid-output"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, job := translationFixture()
			p := &testTranslator{name: "local", reply: tc.text}
			out := PrepareTranslation(context.Background(), readerFor(&s), p, job)
			if out.Diagnostic != tc.expected || out.Kind != "" {
				t.Fatalf("incorrect output accepted: %+v", out)
			}
		})
	}
}

func TestSourceOrTargetChangesDuringTranslation(t *testing.T) {
	for _, change := range []string{"source-revision", "source-content", "target-revision"} {
		t.Run(change, func(t *testing.T) {
			s, job := translationFixture()
			p := &testTranslator{name: "local", reply: "Store {orderId} at 100 mg in storage."}
			p.onCall = func() {
				switch change {
				case "source-revision":
					s.Revisions = append(s.Revisions, Revision{
						Key: job.Source.Key, Revision: "ja-3", Text: "New original",
						Origin: "original", Review: "approved",
					})
					s.Publications[0].Revision = "ja-3"
				case "source-content":
					s.Revisions[2].Text = "Tampered same revision"
				case "target-revision":
					s.Revisions = append(s.Revisions, Revision{
						Key: job.Target, Revision: "en-2", Text: "Reviewed separately",
						Origin: "independent", Review: "approved",
					})
					s.Publications[1].Revision = "en-2"
				}
			}
			out := PrepareTranslation(context.Background(), readerFor(&s), p, job)
			want := map[string]string{
				"source-revision": "source-changed",
				"source-content":  "source-content-changed",
				"target-revision": "target-changed",
			}[change]
			if out.Diagnostic != want {
				t.Fatalf("concurrent update not observed: %+v, want %s", out, want)
			}
		})
	}
}

func TestIndependentAuthoredTextCannotBeOverwrittenAsTranslation(t *testing.T) {
	s, job := translationFixture()
	s.Revisions[1].Origin, s.Revisions[1].BasedOn = "independent", nil
	p := &testTranslator{name: "local"}
	out := PrepareTranslation(context.Background(), readerFor(&s), p, job)
	if out.Diagnostic != "independent-target" || p.calls != 0 {
		t.Fatalf("independent official text treated as translated: %+v", out)
	}
}

func TestProviderErrorAndMissingProvider(t *testing.T) {
	s, job := translationFixture()
	if r := PrepareTranslation(context.Background(), readerFor(&s), nil, job); r.Diagnostic != "missing-provider" {
		t.Fatalf("missing provider: %+v", r)
	}
	p := &testTranslator{name: "local", err: errors.New("synthetic outage")}
	if r := PrepareTranslation(context.Background(), readerFor(&s), p, job); r.Diagnostic != "translation-failed" {
		t.Fatalf("provider error: %+v", r)
	}
	p.err = nil
	p.reply = strings.Repeat("x", 10)
	if r := PrepareTranslation(context.Background(), readerFor(&s), p, job); r.Diagnostic != "protected-token-changed" {
		t.Fatalf("protected spans: %+v", r)
	}
}
