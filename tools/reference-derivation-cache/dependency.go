package derivationcache

import (
	"fmt"
	"sort"
)

// DependencyNode is one provider in a dependency graph.
type DependencyNode struct {
	ProviderID    string
	Epoch         uint64
	DependencyIDs []string
}

// DependencyGraph is the current or cached dependency graph.
type DependencyGraph struct {
	Nodes map[string]DependencyNode
}

// DependencyInput is the cache invalidation input for dependency changes.
type DependencyInput struct {
	// CacheRequested is false for a simple pure computation; no cache contract
	// is then required.
	CacheRequested bool
	// CachedGraph is the dependency graph captured at cache time.
	CachedGraph DependencyGraph
	// CurrentGraph is the current dependency graph.
	CurrentGraph DependencyGraph
	// RootProviderID is the provider whose cache is under test.
	RootProviderID string
}

// DependencyReport is the read-only result for dependency invalidation.
type DependencyReport struct {
	Code      string
	Accepted  bool
	ValueNull bool
	Value     string
	Message   string
	Changed   []string
}

func rejectDependency(code, message string) DependencyReport {
	return DependencyReport{Code: code, ValueNull: true, Message: message}
}

// transitiveDeps returns the transitive closure of dependencies for root.
func transitiveDeps(graph DependencyGraph, root string) map[string]uint64 {
	seen := map[string]uint64{root: 0}
	var walk func(string)
	walk = func(id string) {
		node, ok := graph.Nodes[id]
		if !ok {
			return
		}
		for _, dep := range node.DependencyIDs {
			if _, exists := seen[dep]; exists {
				continue
			}
			depNode, ok := graph.Nodes[dep]
			if !ok {
				continue
			}
			seen[dep] = depNode.Epoch
			walk(dep)
		}
	}
	walk(root)
	delete(seen, root)
	return seen
}

// sortedChanged returns sorted provider IDs whose epoch changed.
func sortedChanged(changed map[string]uint64) []string {
	ids := make([]string, 0, len(changed))
	for id := range changed {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// ReuseDependency checks whether a cached value is still valid given a changed
// dependency graph. It returns stale if any transitive dependency epoch changed.
func ReuseDependency(in DependencyInput) DependencyReport {
	if !in.CacheRequested {
		return DependencyReport{Code: CodeDirectEvaluation, Accepted: true, Value: "",
			Message: "単純な純粋計算に依存関係の鮮度を必須にしない。"}
	}
	if in.CachedGraph.Nodes == nil || in.CurrentGraph.Nodes == nil {
		return rejectDependency(CodeInvalidRequest, "依存関係のキャッシュ判定には cached/current 双方のグラフが必要。")
	}
	if _, ok := in.CurrentGraph.Nodes[in.RootProviderID]; !ok {
		return rejectDependency(CodeStaleRule, fmt.Sprintf("ルート %s が現在のグラフに存在しない。", in.RootProviderID))
	}

	cachedClosure := transitiveDeps(in.CachedGraph, in.RootProviderID)
	currentClosure := transitiveDeps(in.CurrentGraph, in.RootProviderID)

	changed := map[string]uint64{}
	for id, epoch := range currentClosure {
		if cachedEpoch, ok := cachedClosure[id]; !ok || cachedEpoch != epoch {
			changed[id] = epoch
		}
	}
	// Also detect removed transitive dependencies.
	for id := range cachedClosure {
		if _, ok := currentClosure[id]; !ok {
			changed[id] = 0
		}
	}

	if len(changed) > 0 {
		return DependencyReport{
			Code:      CodeStaleInput,
			Accepted:  false,
			ValueNull: true,
			Message:   fmt.Sprintf("推移的依存関係が %d 件変化した。旧値を現在値として返さない。", len(changed)),
			Changed:   sortedChanged(changed),
		}
	}
	return DependencyReport{Code: CodeCacheReused, Accepted: true, Value: "",
		Message: "推移的依存関係の全epochが一致した再利用。"}
}
