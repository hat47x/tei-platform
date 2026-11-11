package derivationcache

import (
	"testing"
)

func TestReuseDependencyRejectsTransitiveEpochChange(t *testing.T) {
	report := ReuseDependency(DependencyInput{
		CacheRequested: true,
		CachedGraph: DependencyGraph{Nodes: map[string]DependencyNode{
			"root": {ProviderID: "root", Epoch: 1, DependencyIDs: []string{"dep"}},
			"dep":  {ProviderID: "dep", Epoch: 1},
		}},
		CurrentGraph: DependencyGraph{Nodes: map[string]DependencyNode{
			"root": {ProviderID: "root", Epoch: 1, DependencyIDs: []string{"dep"}},
			"dep":  {ProviderID: "dep", Epoch: 2},
		}},
		RootProviderID: "root",
	})
	if report.Accepted || report.Code != CodeStaleInput || !report.ValueNull {
		t.Fatalf("report=%+v want STALE_INPUT with unknown value", report)
	}
	if len(report.Changed) != 1 || report.Changed[0] != "dep" {
		t.Fatalf("unexpected changed list: %v", report.Changed)
	}
}

func TestReuseDependencyRejectsRemovedTransitiveDep(t *testing.T) {
	report := ReuseDependency(DependencyInput{
		CacheRequested: true,
		CachedGraph: DependencyGraph{Nodes: map[string]DependencyNode{
			"root": {ProviderID: "root", Epoch: 1, DependencyIDs: []string{"dep"}},
			"dep":  {ProviderID: "dep", Epoch: 1},
		}},
		CurrentGraph: DependencyGraph{Nodes: map[string]DependencyNode{
			"root": {ProviderID: "root", Epoch: 1},
		}},
		RootProviderID: "root",
	})
	if report.Accepted || report.Code != CodeStaleInput || !report.ValueNull {
		t.Fatalf("report=%+v want STALE_INPUT with unknown value", report)
	}
}

func TestReuseDependencyAllowsUnchangedGraph(t *testing.T) {
	report := ReuseDependency(DependencyInput{
		CacheRequested: true,
		CachedGraph: DependencyGraph{Nodes: map[string]DependencyNode{
			"root": {ProviderID: "root", Epoch: 1, DependencyIDs: []string{"dep"}},
			"dep":  {ProviderID: "dep", Epoch: 1},
		}},
		CurrentGraph: DependencyGraph{Nodes: map[string]DependencyNode{
			"root": {ProviderID: "root", Epoch: 1, DependencyIDs: []string{"dep"}},
			"dep":  {ProviderID: "dep", Epoch: 1},
		}},
		RootProviderID: "root",
	})
	if !report.Accepted || report.Code != CodeCacheReused || report.ValueNull {
		t.Fatalf("report=%+v want CACHE_REUSED", report)
	}
}

func TestReuseDependencyRejectsMissingRoot(t *testing.T) {
	report := ReuseDependency(DependencyInput{
		CacheRequested: true,
		CachedGraph:    DependencyGraph{Nodes: map[string]DependencyNode{}},
		CurrentGraph:   DependencyGraph{Nodes: map[string]DependencyNode{}},
	})
	if report.Accepted || report.Code != CodeStaleRule || !report.ValueNull {
		t.Fatalf("report=%+v want STALE_RULE with unknown value", report)
	}
}
