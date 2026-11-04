package changeanalysis

import (
	"fmt"
	"sort"

	"tei/internal/core"
	"tei/internal/ir"
)

// ApplicationReleaseContractMismatch is a read-only observation about an
// Application Release's declared Contract requirement. It does not prove that
// the release is wrong: a historical release may intentionally require a
// Contract that the current definition no longer provides.
type ApplicationReleaseContractMismatch struct {
	Snapshot        string           `json:"snapshot"`
	Application     core.CanonicalID `json:"application"`
	Version         string           `json:"version"`
	Alias           core.LocalName   `json:"alias"`
	Database        core.CanonicalID `json:"database"`
	Contract        core.CanonicalID `json:"contract"`
	CompatibleMajor int              `json:"compatibleMajor"`
	CurrentVersion  string           `json:"currentContractVersion,omitempty"`
	Reason          string           `json:"reason"`
}

const (
	ApplicationReleaseContractNotCurrent = "not-in-current-definition"
	ApplicationReleaseContractMajor      = "compatible-major-mismatch"
)

// ApplicationReleaseContractCandidates lists Application Release Contract
// requirements whose Contract has no current definition or whose compatible
// major differs from the current definition's major. Application Release
// requirements are otherwise checked only for existence and ownership by
// validation, so this read-only view makes the major direction visible.
// Dynamic consumers, runtime deployment state and provider behavior are not
// evaluated.
func ApplicationReleaseContractCandidates(w ir.World, snapshot string) []ApplicationReleaseContractMismatch {
	contracts := map[core.CanonicalID]databaseContractVersion{}
	for _, db := range w.Databases {
		for _, c := range db.Contracts {
			contracts[c.ID] = databaseContractVersion{major: c.Version.Major, minor: c.Version.Minor}
		}
	}
	var out []ApplicationReleaseContractMismatch
	for _, r := range w.ApplicationReleases {
		for _, db := range r.Databases {
			for _, req := range db.RequiredContracts {
				item := ApplicationReleaseContractMismatch{
					Snapshot: snapshot, Application: r.Application, Version: r.Version,
					Alias: db.Alias, Database: db.TargetDatabase, Contract: req.ContractID,
					CompatibleMajor: req.CompatibleMajor,
				}
				c, ok := contracts[req.ContractID]
				switch {
				case !ok:
					item.Reason = ApplicationReleaseContractNotCurrent
				case c.major != req.CompatibleMajor:
					item.Reason = ApplicationReleaseContractMajor
					item.CurrentVersion = fmt.Sprintf("%d.%d", c.major, c.minor)
				default:
					continue
				}
				out = append(out, item)
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

type databaseContractVersion struct{ major, minor int }
