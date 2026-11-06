package changeanalysis

import (
	"tei/internal/core"
	"tei/internal/database"
	"tei/internal/ir"
)

// currentContractVersions returns the current definition's Contract versions by
// identity. Candidate views only compare these values; they do not mutate the
// current definition.
func currentContractVersions(w ir.World) map[core.CanonicalID]database.ContractVersion {
	out := make(map[core.CanonicalID]database.ContractVersion)
	for _, db := range w.Databases {
		for _, c := range db.Contracts {
			out[c.ID] = c.Version
		}
	}
	return out
}
