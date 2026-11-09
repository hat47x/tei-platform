package main

import (
	"testing"
)

func TestExistingProcedureDirectValidationRefsStayOnFunctionFreeStep(t *testing.T) {
	p, err := run(
		"../../reference/business-system-design/order-entry-business-procedure.json",
		"../../reference/business-system-design/order-entry-dfd-review-proposal.json",
		"../../reference/business-system-design/order-entry-procedure-reference-set.json",
		"../../reference/workspace",
	)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, node := range p.Steps {
		if node.ID != "BP-05" {
			continue
		}
		found = true
		if !node.NoApplicationFunction || len(node.FunctionRefs) != 0 {
			t.Fatal("Business-only Validation step acquired an Application Function")
		}
		if len(node.ExplicitMeaning) != 2 {
			t.Fatalf("Procedure-owned Validation links disappeared: %+v", node.ExplicitMeaning)
		}
		for _, ref := range node.ExplicitMeaning {
			if ref.Kind != "semantic-validation" || ref.Status != "resolved-id-not-accepted" ||
				ref.Authority != "procedure-reference-only" || ref.Source.File == "" {
				t.Fatalf("invalid direct semantic reference authority: %+v", ref)
			}
		}
		nav, err := navigate(p, "step:BP-05")
		if err != nil {
			t.Fatal(err)
		}
		validationLinks := 0
		for _, link := range nav.Links {
			if link.Relation == "explicit-business-semantic-reference" && link.Subject.Kind == "semantic-validation" {
				validationLinks++
				if link.Subject.Scope != "unverified-current-world" ||
					link.Subject.Authority != "identity-match-only" {
					t.Fatalf("direct ref promoted to accepted Source authority: %+v", link)
				}
			}
		}
		if validationLinks != 2 {
			t.Fatalf("navigation lost direct refs: %+v", nav.Links)
		}
	}
	if !found {
		t.Fatal("Reference Validation Step BP-05 missing")
	}
}

func TestUnknownDirectSemanticRefIsSourceAwareError(t *testing.T) {
	f, r, s, w := sample()
	f.Steps[1].Data = []string{"unknown.business.data"}
	p := build(f, r, s, w, "procedure.json", "review.json", "set.json")
	seen := false
	for _, d := range p.Diagnostics {
		if d.Code == "DFD_DIRECT_SEMANTIC_REF_NOT_EXACT" &&
			d.Severity == "error" && d.Source.File == "procedure.json" &&
			d.Source.Pointer == "/steps/1/data/0" {
			seen = true
		}
	}
	if !seen {
		t.Fatalf("missing direct semantic diagnostic: %+v", p.Diagnostics)
	}
	nav, err := navigate(p, "step:BP-04")
	if err != nil {
		t.Fatal(err)
	}
	if nav.Status != "unresolved" || nav.Source != nil {
		t.Fatal("unknown direct Data ID silently became a navigable authority")
	}
}
