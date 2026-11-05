package changeanalysis

import (
	"fmt"
	"sort"

	"tei/internal/core"
	"tei/internal/ir"
)

// DatabaseReleaseContractCandidate is a read-only observation about a Database
// Release's provided Contract. It covers only the direction that
// DATABASE_RELEASE_CONTRACT_AHEAD_OF_CURRENT_DEFINITION does not warn about:
// the recorded release provides an older major than the current definition.
// Historical releases are not necessarily wrong; this is not an acceptance
// finding.
type DatabaseReleaseContractCandidate struct {
	Snapshot        string           `json:"snapshot"`
	Database        core.CanonicalID `json:"database"`
	Version         string           `json:"version"`
	Contract        core.CanonicalID `json:"contract"`
	ProvidedVersion string           `json:"providedContractVersion"`
	CurrentVersion  string           `json:"currentContractVersion"`
}

// DatabaseReleaseContractCandidates lists provided Contracts whose major is
// lower than the current definition's major. Releases that provide a newer
// major are already covered by DATABASE_RELEASE_CONTRACT_AHEAD_OF_CURRENT_DEFINITION,
// and Contracts with no current definition are covered by
// DATABASE_RELEASE_CONTRACT_NOT_CURRENT, so both are excluded here.
func DatabaseReleaseContractCandidates(w ir.World, snapshot string) []DatabaseReleaseContractCandidate {
	contracts := currentContractVersions(w)
	var out []DatabaseReleaseContractCandidate
	for _, r := range w.DatabaseReleases {
		for _, p := range r.Provides {
			c, ok := contracts[p.ID]
			if !ok || p.Version.Major >= c.Major {
				continue
			}
			out = append(out, DatabaseReleaseContractCandidate{
				Snapshot: snapshot, Database: r.Database, Version: r.Version,
				Contract: p.ID, ProvidedVersion: fmt.Sprintf("%d.%d", p.Version.Major, p.Version.Minor),
				CurrentVersion: fmt.Sprintf("%d.%d", c.Major, c.Minor),
			})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.Database != b.Database {
			return a.Database < b.Database
		}
		if a.Version != b.Version {
			return a.Version < b.Version
		}
		return a.Contract < b.Contract
	})
	return out
}
