package changeanalysis

import (
	"fmt"
	"sort"

	"tei/internal/core"
	"tei/internal/ir"
)

// ApplicationReleaseContractMismatch is a read-only observation about an
// Application Release's declared Contract requirement. It does not prove that
// the release is wrong: a historical release may intentionally require an
// older major than the current definition.
type ApplicationReleaseContractMismatch struct {
	Snapshot        string           `json:"snapshot"`
	Application     core.CanonicalID `json:"application"`
	Version         string           `json:"version"`
	Alias           core.LocalName   `json:"alias"`
	Database        core.CanonicalID `json:"database"`
	Contract        core.CanonicalID `json:"contract"`
	CompatibleMajor int              `json:"compatibleMajor"`
	CurrentVersion  string           `json:"currentContractVersion"`
}

// ApplicationReleaseContractCandidates lists Application Release Contract
// requirements whose compatible major differs from the current definition's
// major. Requirements with no current definition are already reported by
// APPLICATION_RELEASE_CONTRACT_NOT_CURRENT, so they are not repeated here; this
// keeps the candidate view limited to the direction validation does not
// diagnose. Dynamic consumers, runtime deployment state and provider behavior
// are not evaluated.
func ApplicationReleaseContractCandidates(w ir.World, snapshot string) []ApplicationReleaseContractMismatch {
	contracts := currentContractVersions(w)
	var out []ApplicationReleaseContractMismatch
	for _, r := range w.ApplicationReleases {
		for _, db := range r.Databases {
			for _, req := range db.RequiredContracts {
				c, ok := contracts[req.ContractID]
				if !ok || c.Major == req.CompatibleMajor {
					continue
				}
				out = append(out, ApplicationReleaseContractMismatch{
					Snapshot: snapshot, Application: r.Application, Version: r.Version,
					Alias: db.Alias, Database: db.TargetDatabase, Contract: req.ContractID,
					CompatibleMajor: req.CompatibleMajor,
					CurrentVersion:  fmt.Sprintf("%d.%d", c.Major, c.Minor),
				})
			}
		}
	}
	sort.Slice(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.Application != b.Application {
			return a.Application < b.Application
		}
		if a.Version != b.Version {
			return a.Version < b.Version
		}
		if a.Alias != b.Alias {
			return a.Alias < b.Alias
		}
		return a.Contract < b.Contract
	})
	return out
}
