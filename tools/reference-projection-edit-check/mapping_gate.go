package projectionedit

import facetmappingcheck "tei/tools/reference-facet-mapping-check"

// EditableThroughMapping reports whether an edit may be written back through the
// mapping the existing checker judged. It is the whole-mapping gate: only a
// preserved mapping whose edit-inverse the checker found eligible passes. It does
// not look inside a lossy mapping for a safe sub-operation; that needs a mapping
// row scoped to the slot being edited (see README, "操作相対").
func EditableThroughMapping(res facetmappingcheck.Result) bool {
	return res.Verdict == facetmappingcheck.VerdictPreserved && res.Inverse == facetmappingcheck.InverseEligible
}
