package semanticdistribution

import (
	"encoding/json"
	"fmt"
	"strings"

	"tei/internal/modelref"
	"tei/internal/sourceindex"
	"tei/internal/subject"
	"tei/internal/workspace"
)

// ImportedTypeImpactRule is a reference-only typed graph relation, not a new
// core Kind. It connects an authored local Data to an imported Namespace when
// a local Attribute type or Relation target explicitly references that Namespace.
var ImportedTypeImpactRule = modelref.Rule{
	Relation: "uses-imported-namespace",
	FromKind: "semantic-data",
	ToKind:   "semantic-namespace",
}

// ProjectImportTypeReferences derives a conservative, read-only graph from
// actual parsed Workspace Semantics. It covers Data Attribute.TypeRef and
// Relation.Target only; other reference kinds must not be treated as scanned.
// The provider is identified logically, not by its path or package identity.
func ProjectImportTypeReferences(view workspace.Result, namespaceID string) (modelref.Graph, error) {
	var digest string
	count := 0
	for _, imp := range view.Imports {
		if imp.Namespace == namespaceID {
			digest = imp.SemanticDigest
			count++
		}
	}
	if count != 1 || digest == "" {
		return modelref.Graph{}, fmt.Errorf("imported namespace %q is not uniquely resolved", namespaceID)
	}
	if _, locallyOwned := view.Sources["semantic:"+namespaceID]; locallyOwned {
		return modelref.Graph{}, fmt.Errorf("imported namespace %q is also locally owned", namespaceID)
	}
	var providerBytes []byte
	found := 0
	for _, ns := range view.World.Semantics {
		if string(ns.ID) == namespaceID {
			var err error
			providerBytes, err = json.Marshal(ns)
			if err != nil {
				return modelref.Graph{}, fmt.Errorf("imported namespace %q: %w", namespaceID, err)
			}
			found++
		}
	}
	if found != 1 || sourceindex.Fingerprint(providerBytes) != digest {
		return modelref.Graph{}, fmt.Errorf("imported namespace %q has stale or ambiguous semantic evidence", namespaceID)
	}
	root := modelref.Key{
		Scope:   "namespace:" + namespaceID,
		Subject: subject.Subject{Kind: "semantic-namespace", ID: namespaceID},
	}
	g := modelref.Graph{
		Schema: "tei.modelref.v0",
		Nodes: []modelref.Node{{
			Key: root, Evidence: []modelref.Evidence{{
				Authority: "proposal", Context: "normalized-namespace-meaning",
				ArtifactFingerprint: digest,
			}},
		}},
	}
	prefix := namespaceID + "."
	for _, ns := range view.World.Semantics {
		if _, owned := view.Sources["semantic:"+string(ns.ID)]; !owned {
			continue // Do not pretend to own imported/stdlib data.
		}
		for _, data := range ns.Data {
			used := false
			for _, attribute := range data.Attributes {
				if strings.HasPrefix(string(attribute.TypeRef), prefix) {
					used = true
					break
				}
			}
			if !used {
				for _, relation := range data.Relations {
					if strings.HasPrefix(string(relation.Target), prefix) {
						used = true
						break
					}
				}
			}
			if !used {
				continue
			}
			key := modelref.Key{
				Scope: "namespace:" + string(ns.ID),
				Subject: subject.Subject{Kind: "semantic-data", ID: string(data.ID)},
			}
			g.Nodes = append(g.Nodes, modelref.Node{Key: key})
			g.Edges = append(g.Edges, modelref.Edge{
				From: key,
				To: modelref.Ref{
					Scope: root.Scope,
					Subject: root.Subject,
				},
				Relation: ImportedTypeImpactRule.Relation,
			})
		}
	}
	return g, nil
}

// ReviewImportedTypeImpact uses the existing modelref two-snapshot review
// rather than creating a competing impact graph or reinterpreting package
// identifiers as business Canonical IDs. This is a candidate and cannot prove
// a complete set of consumers, Accepted meaning, or Runtime impact.
func ReviewImportedTypeImpact(before, after workspace.Result, namespaceID string) (modelref.ImpactReview, error) {
	left, err := ProjectImportTypeReferences(before, namespaceID)
	if err != nil {
		return modelref.ImpactReview{}, fmt.Errorf("before: %w", err)
	}
	right, err := ProjectImportTypeReferences(after, namespaceID)
	if err != nil {
		return modelref.ImpactReview{}, fmt.Errorf("after: %w", err)
	}
	out := modelref.ReviewImpactDiff(left, right, []modelref.Rule{ImportedTypeImpactRule})
	out.AuthorityLimits = append(out.AuthorityLimits,
		"only-local-data-type-and-relation-target-references-inspected",
		"unscanned-consumers-cannot-be-certified-unaffected",
		"semantic-provenance-is-not-runtime-or-acceptance-authority",
	)
	return out, nil
}
