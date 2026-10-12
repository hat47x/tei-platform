package changeanalysis

import (
	"tei/internal/core"
	"tei/internal/ir"
)

// logicalRuleEdges connects explicitly authored business assertions only to
// their owning Data and referenced attributes. No DB PK/FK or runtime edge is
// inferred. Referenced attributes are validated by the shared World validator.
func logicalRuleEdges(after ir.World, kind string, id core.CanonicalID) []ImpactEdge {
	var edges []ImpactEdge
	from := subjectKey(kind, id)
	for _, ns := range after.Semantics {
		for _, data := range ns.Data {
			var determinants, dependents []core.CanonicalID
			switch kind {
			case "candidate-key":
				for _, key := range data.CandidateKeys {
					if key.ID == id {
						determinants = key.Attributes
						break
					}
				}
			case "functional-dependency":
				for _, fd := range data.FunctionalDependencies {
					if fd.ID == id {
						determinants, dependents = fd.Determinants, fd.Dependents
						break
					}
				}
			}
			if determinants == nil && dependents == nil {
				continue
			}
			edges = append(edges, ImpactEdge{
				From: from, To: subjectKey("data", data.ID),
				Via: "semantic.data." + kind,
				Reason: "This explicit logical rule is declared by this Data",
			})
			for _, attr := range determinants {
				role := "determinant"
				if kind == "candidate-key" {
					role = "field"
				}
				edges = append(edges, ImpactEdge{
					From: from, To: subjectKey("attribute", attr),
					Via: "semantic." + kind + "." + role,
					Reason: "This attribute is explicitly named by the rule",
				})
			}
			for _, attr := range dependents {
				edges = append(edges, ImpactEdge{
					From: from, To: subjectKey("attribute", attr),
					Via: "semantic.functional-dependency.dependent",
					Reason: "This attribute is explicitly named as a dependent",
				})
			}
			return edges
		}
	}
	return nil
}
