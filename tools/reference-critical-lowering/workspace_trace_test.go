package main

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/driver"
	"tei/internal/environment"
	"tei/internal/plugin"
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

	// Observe the actual Environment-to-Instance selection. This checks
	// declared identity wiring only, not connectivity or current schema compatibility.
	var foundEnvironment bool
	for _, env := range loaded.World.Environments {
		if env.ID != "development" {
			continue
		}
		if foundEnvironment {
			t.Fatal("duplicate development Environment")
		}
		foundEnvironment = true
		plugins, err := (plugin.Catalog{
			Plugins: loaded.World.Plugins,
			Bundles: loaded.World.Bundles,
		}).ResolveEnvironment(env)
		if err != nil {
			t.Fatalf("existing Plugin resolver rejected development: %v", err)
		}
		if plugins.Environment != env.ID || len(plugins.Plugins) == 0 {
			t.Fatalf("development plugin resolution missing: %+v", plugins)
		}
		for _, requirement := range f.c.Requirements {
			for _, need := range requirement.DataContracts {
				var instanceID core.CanonicalID
				for _, binding := range env.Bindings {
					if binding.Application != app.ID || string(binding.Alias) != need.Datasource {
						continue
					}
					if instanceID != "" {
						t.Fatalf("ambiguous Environment binding for %s", need.Datasource)
					}
					instanceID = binding.Instance
				}
				if instanceID == "" {
					t.Fatalf("missing Environment binding for %s", need.Datasource)
				}
				instance, ok := env.InstanceByID(instanceID)
				if !ok || string(instance.Definition) != need.Database {
					t.Fatalf("Datasource %s bound to incompatible database definition: %+v", need.Datasource, instance)
				}
			}
		}
	}
	if !foundEnvironment {
		t.Fatal("reference development Environment missing")
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

func TestEnvironmentDatasourceBindingMutationsFailClosed(t *testing.T) {
	loaded, err := workspace.Load(filepath.Join("..", "..", "reference", "workspace"))
	if err != nil { t.Fatal(err) }
	var dev environment.Environment
	for _, e := range loaded.World.Environments { if e.ID == "development" { dev = e } }
	if dev.ID == "" { t.Fatal("missing development Environment") }
	check := func(env environment.Environment) bool {
		var selected core.CanonicalID
		for _, binding := range env.Bindings {
			if binding.Application != "order_entry" || binding.Alias != "sales" { continue }
			if selected != "" { return false } // duplicates are never ordered fallbacks
			selected = binding.Instance
		}
		if selected == "" { return false }
		instance, ok := env.InstanceByID(selected)
		return ok && instance.Definition == "sales_core"
	}
	if !check(dev) { t.Fatal("actual sales datasource does not bind sales_core") }
	copyEnv := func() environment.Environment {
		e := dev
		e.Bindings = append([]environment.ApplicationDatabaseBinding(nil), dev.Bindings...)
		return e
	}
	missing := copyEnv()
	kept := missing.Bindings[:0]
	for _, b := range missing.Bindings {
		if b.Application != "order_entry" || b.Alias != "sales" { kept = append(kept, b) }
	}
	missing.Bindings = kept
	if check(missing) { t.Fatal("missing binding accepted") }

	duplicate := copyEnv()
	for _, b := range duplicate.Bindings {
		if b.Application == "order_entry" && b.Alias == "sales" {
			duplicate.Bindings = append(duplicate.Bindings, b)
			break
		}
	}
	if check(duplicate) { t.Fatal("ambiguous binding accepted") }

	misbound := copyEnv()
	for i, b := range misbound.Bindings {
		if b.Application == "order_entry" && b.Alias == "sales" {
			misbound.Bindings[i].Instance = "master_data.dev"
		}
	}
	if check(misbound) { t.Fatal("wrong database instance accepted") }
}
