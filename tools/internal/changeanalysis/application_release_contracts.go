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

// ApplicationReleaseContractCandidates lists Contract requirements declared by
// Application Releases: for each Application release major line, database alias
// and Contract, it observes the latest Release that declares the requirement,
// and reports it only when its compatible major differs from the current
// definition's major. A release line normally repeats the same requirement
// across patch and minor releases, so listing every historical Release reports
// one state many times. A newer Release that declares the same requirement
// replaces the older one, while a requirement dropped from newer Releases is
// still reported from the latest Release that declares it. Requirements with no
// current definition are already reported by
// APPLICATION_RELEASE_CONTRACT_NOT_CURRENT, so they are not repeated here.
// Releases whose version is not numeric major.minor.patch cannot be ordered and
// are ignored. Dynamic consumers, runtime deployment state and provider
// behavior are not evaluated.
func ApplicationReleaseContractCandidates(w ir.World, snapshot string) []ApplicationReleaseContractMismatch {
	contracts := currentContractVersions(w)
	type lineKey struct {
		application core.CanonicalID
		major       int
		database    core.CanonicalID
		alias       core.LocalName
		contract    core.CanonicalID
	}
	type declaration struct {
		version         string
		compatibleMajor int
	}
	latest := make(map[lineKey]declaration)
	for _, r := range w.ApplicationReleases {
		v, ok := parseReleaseVersion(r.Version)
		if !ok {
			continue
		}
		for _, db := range r.Databases {
			for _, req := range db.RequiredContracts {
				key := lineKey{r.Application, v[0], db.TargetDatabase, db.Alias, req.ContractID}
				current, seen := latest[key]
				if seen && compareReleaseVersion(r.Version, current.version) <= 0 {
					continue
				}
				latest[key] = declaration{version: r.Version, compatibleMajor: req.CompatibleMajor}
			}
		}
	}
	var out []ApplicationReleaseContractMismatch
	for key, d := range latest {
		c, ok := contracts[key.contract]
		if !ok || c.Major == d.compatibleMajor {
			continue
		}
		out = append(out, ApplicationReleaseContractMismatch{
			Snapshot: snapshot, Application: key.application, Version: d.version,
			Alias: key.alias, Database: key.database, Contract: key.contract,
			CompatibleMajor: d.compatibleMajor,
			CurrentVersion:  fmt.Sprintf("%d.%d", c.Major, c.Minor),
		})
	}
	sort.Slice(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.Application != b.Application {
			return a.Application < b.Application
		}
		if a.Version != b.Version {
			if c := compareReleaseVersion(a.Version, b.Version); c != 0 {
				return c < 0
			}
		}
		if a.Alias != b.Alias {
			return a.Alias < b.Alias
		}
		return a.Contract < b.Contract
	})
	return out
}
