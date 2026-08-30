package index

import (
	"tei/internal/core"
	"tei/internal/query"
)

// AnalyzedQueryEntry is a presentation/index envelope around the derived Query
// IR. Keeping the IR nested prevents query-analysis details from leaking into
// canonical ir.World.
type AnalyzedQueryEntry struct {
	Application core.CanonicalID `json:"application"`
	IR          query.DerivedIR  `json:"ir"`
}
