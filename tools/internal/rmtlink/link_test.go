package rmtlink

import (
	"strings"
	"testing"

	"tei/tools/internal/rmtevidence"
)

func sourceFixture(t *testing.T) rmtevidence.Evidence {
	t.Helper()
	v := rmtevidence.Evidence{
		Schema: rmtevidence.Schema,
		SourceFormat: "xead",
		SourceFormatVersion: "1.2",
		SourceArtifactDigest: "sha256:" + strings.Repeat("1", 64),
		SourceToken: "sha256:" + strings.Repeat("2", 64),
		SourceConstructKnown: true,
		SourceConstruct: "Table",
		MappingContractFingerprint: "sha256:" + strings.Repeat("3", 64),
		MappingStatus: rmtevidence.MappingMapped,
		Classification: []string{"canonical-candidate", "unresolved"},
		Extension: "reference.xead.relational-semantics",
		AuthorityLimits: append([]string(nil), rmtevidence.RequiredAuthorityLimits...),
	}
	prepared, err := rmtevidence.Prepare(v)
	if err != nil {
		t.Fatal(err)
	}
	return prepared
}

func linkContext(t *testing.T) Context {
	t.Helper()
	return Context{
		Source: sourceFixture(t),
		CandidateFingerprint: "sha256:" + strings.Repeat("4", 64),
		BaseModelRevision: "sha256:" + strings.Repeat("5", 64),
		ReviewFingerprint: "sha256:" + strings.Repeat("6", 64),
		Target: SemanticTarget{
			Kind: "validation",
			ID: "sales.order_line.unit_price_positive",
			Change: "added",
		},
	}
}

func TestLinkIsOnlyAnUnresolvedProposal(t *testing.T) {
	ctx := linkContext(t)
	first, err := Prepare(ctx)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Prepare(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if first.LinkFingerprint != second.LinkFingerprint {
		t.Fatalf("non-deterministic proposal identity: %+v %+v", first, second)
	}
	if first.Status != StatusProposed || first.MeaningStatus != MeaningUnresolved {
		t.Fatalf("candidate illegally claimed acceptance: %+v", first)
	}
	if first.SourceToken != ctx.Source.SourceToken ||
		first.SourceEvidenceFingerprint != ctx.Source.EvidenceFingerprint ||
		first.CandidateFingerprint != ctx.CandidateFingerprint ||
		first.ReviewFingerprint != ctx.ReviewFingerprint {
		t.Fatalf("missing exact source/review binding: %+v", first)
	}
	if len(first.AuthorityLimits) == 0 {
		t.Fatal("candidate lost required authority limits")
	}
	if err := Verify(first, ctx); err != nil {
		t.Fatal(err)
	}
}

func TestLinkRejectsRetargetingAndStaleContext(t *testing.T) {
	ctx := linkContext(t)
	link, err := Prepare(ctx)
	if err != nil {
		t.Fatal(err)
	}
	changes := []struct {
		name string
		edit func(*Context)
	}{
		{"source snapshot changed", func(v *Context) {
			v.Source.SourceArtifactDigest = "sha256:" + strings.Repeat("a", 64)
			rebound, err := rmtevidence.Prepare(v.Source)
			if err != nil {
				t.Fatal(err)
			}
			v.Source = rebound
		}},
		{"source node changed", func(v *Context) {
			v.Source.SourceToken = "sha256:" + strings.Repeat("b", 64)
			rebound, err := rmtevidence.Prepare(v.Source)
			if err != nil { t.Fatal(err) }
			v.Source = rebound
		}},
		{"source mapping contract changed", func(v *Context) {
			v.Source.MappingContractFingerprint = "sha256:" + strings.Repeat("c", 64)
			rebound, err := rmtevidence.Prepare(v.Source)
			if err != nil { t.Fatal(err) }
			v.Source = rebound
		}},
		{"candidate changed", func(v *Context) {
			v.CandidateFingerprint = "sha256:" + strings.Repeat("d", 64)
		}},
		{"review changed", func(v *Context) {
			v.ReviewFingerprint = "sha256:" + strings.Repeat("e", 64)
		}},
		{"base revision changed", func(v *Context) {
			v.BaseModelRevision = "sha256:" + strings.Repeat("f", 64)
		}},
		{"target changed", func(v *Context) {
			v.Target.ID = "sales.order_line.quantity_positive"
		}},
	}
	for _, tc := range changes {
		t.Run(tc.name, func(t *testing.T) {
			altered := ctx
			tc.edit(&altered)
			if err := Verify(link, altered); err == nil {
				t.Fatalf("link was reusable against changed %s", tc.name)
			}
		})
	}

	link.Status = "accepted"
	if err := Verify(link, ctx); err == nil {
		t.Fatal("untrusted link claimed human Acceptance")
	}
}

func TestLinkRejectsForgedEvidenceAndUnqualifiedTarget(t *testing.T) {
	ctx := linkContext(t)
	ctx.Source.SourceToken = "sha256:" + strings.Repeat("8", 64)
	if _, err := Prepare(ctx); err == nil {
		t.Fatal("source Evidence was silently accepted despite stale fingerprint")
	}
	ctx = linkContext(t)
	ctx.Target.Kind = "runtime-effect"
	if _, err := Prepare(ctx); err == nil {
		t.Fatal("source observation got Runtime authority")
	}
	ctx = linkContext(t)
	ctx.Target.ID = "../private/source.xml"
	if _, err := Prepare(ctx); err == nil {
		t.Fatal("path became a semantic subject")
	}
	ctx = linkContext(t)
	ctx.ReviewFingerprint = ""
	if _, err := Prepare(ctx); err == nil {
		t.Fatal("link without actual Review was accepted")
	}
}
