package main

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/driver"
	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

// This is a read-only consumer of the *existing* Workspace loader, Canonical
// World, Driver and Source Index. It does not execute the Web Target or any
// Capability, and does not turn synthetic Plugin Trust into production Authority.
func TestExistingOrderEntrySourceToDriverTrace(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "workspace")
	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatalf("load actual order_entry workspace: %v", err)
	}
	resolver := driver.NewResolver(loaded.World)
	for _, tc := range []struct {
		pattern string
		verify func(application.Pattern, driver.PatternResolution) bool
	}{
		{"order_entry.submit_order", func(_ application.Pattern, r driver.PatternResolution) bool {
			return r.Capability != nil && r.Capability.ID == "sales.order.submit"
		}},
		{"order_entry.recent_orders", func(_ application.Pattern, r driver.PatternResolution) bool {
			return r.Data != nil && r.Data.ID == "sales.order"
		}},
		{"order_entry.order_lines", func(_ application.Pattern, r driver.PatternResolution) bool {
			return r.Relation != nil && r.Relation.Relation.ID == "sales.order.lines"
		}},
		{"order_entry.customer_reference", func(_ application.Pattern, r driver.PatternResolution) bool {
			return r.Datasource != nil && r.Datasource.Alias == "master"
		}},
	} {
		resolved, err := resolver.ResolvePattern(core.MustCanonicalID("order_entry"), core.MustCanonicalID(tc.pattern))
		if err != nil {
			t.Fatalf("resolve actual %s: %v", tc.pattern, err)
		}
		if !tc.verify(resolved.Pattern, resolved) {
			t.Fatalf("lost Canonical cross-view link: %+v", resolved)
		}
	}
	idx, err := workspace.BuildSourceIndex(root, loaded)
	if err != nil {
		t.Fatal(err)
	}
	refs := idx.ReferencesFor(sourceindex.Subject{
		Kind: "application-pattern", ID: "order_entry.submit_order",
	})
	if len(refs) == 0 {
		t.Fatal("submit order has no source-index trace")
	}
	found := false
	for _, r := range refs {
		if r.ArtifactPath == "applications/order-entry/patterns/submit-order.tei" {
			found = true
		}
	}
	if !found {
		t.Fatalf("wrong provenance of submit action: %+v", refs)
	}

	f := orderFixture("tei.language.dsl")
	var app *application.Application
	for i := range loaded.World.Applications {
		if loaded.World.Applications[i].ID == "order_entry" {
			app = &loaded.World.Applications[i]
		}
	}
	if app == nil {
		t.Fatal("order_entry Application missing from real Workspace")
	}
	declared := make(map[string]bool)
	for _, db := range app.Databases {
		for _, req := range db.RequiredContracts {
			key := fmt.Sprintf("%s/%s/%s/^%d", db.Alias, db.TargetDatabase, req.ContractID, req.CompatibleMajor)
			declared[key] = true
		}
	}
	for _, requirement := range f.c.Requirements {
		for _, need := range requirement.DataContracts {
			key := need.Datasource + "/" + need.Database + "/" + need.Contract + "/" + need.VersionConstraint
			if !declared[key] {
				t.Fatalf("synthetic lowering added a DB requirement not declared in real Source: %s", key)
			}
		}
	}

	// Bind illustrative Candidate origin refs to Source Index fingerprints,
	// not invented "sha256:<path>" labels. The Source Index owns file locators,
	// while Acceptance (currently synthetic) would own the accepted provenance.
	artifacts := make(map[string]string, len(idx.Artifacts))
	for _, artifact := range idx.Artifacts {
		artifacts[artifact.Path] = artifact.Fingerprint
	}
	bind := func(ref *Ref) {
		t.Helper()
		const prefix = "reference/workspace/"
		if !strings.HasPrefix(ref.ID, prefix) {
			t.Fatalf("source dependency %s is outside the expected reference workspace", ref.ID)
		}
		localPath := strings.TrimPrefix(ref.ID, prefix)
		fingerprint := artifacts[localPath]
		if fingerprint == "" {
			t.Fatalf("source dependency %s has no real Source Index artifact", localPath)
		}
		ref.Fingerprint = fingerprint
	}
	bind(&f.c.Source)
	for i := range f.c.SourceInputs {
		bind(&f.c.SourceInputs[i])
	}
	f.a.CandidateDigest = Digest(f.c)
	neutral, err := Neutralize(f.c, f.a, stubTrust{})
	if err != nil {
		t.Fatalf("source-index fingerprint bound Candidate: %v", err)
	}
	if neutral.Origin.Source.Fingerprint != f.c.Source.Fingerprint ||
		len(neutral.Origin.SourceInputs) != len(f.c.SourceInputs) {
		t.Fatalf("provenance lost across accepted boundary: %+v", neutral.Origin)
	}
	for i, source := range f.c.SourceInputs {
		if neutral.Origin.SourceInputs[i] != source {
			t.Fatalf("source fingerprint lost at index %d", i)
		}
	}
}
