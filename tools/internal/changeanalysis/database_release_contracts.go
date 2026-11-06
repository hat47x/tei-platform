package changeanalysis

import (
	"fmt"
	"sort"

	"tei/internal/core"
	dbrelease "tei/internal/database/release"
	"tei/internal/ir"
)

// DatabaseReleaseContractCandidate is a read-only observation about the latest
// Database Release in a release line. It covers only the direction that
// DATABASE_RELEASE_CONTRACT_AHEAD_OF_CURRENT_DEFINITION does not warn about:
// the latest release in a major line provides an older major than the current
// definition. Older releases in the same line are historical and are not
// repeated; this is not an acceptance finding.
type DatabaseReleaseContractCandidate struct {
	Snapshot        string           `json:"snapshot"`
	Database        core.CanonicalID `json:"database"`
	Version         string           `json:"version"`
	Contract        core.CanonicalID `json:"contract"`
	ProvidedVersion string           `json:"providedContractVersion"`
	CurrentVersion  string           `json:"currentContractVersion"`
}

// DatabaseReleaseContractCandidates lists provided Contracts of the latest
// Database Release in each major release line whose major is lower than the
// current definition's major. A database can maintain several major lines in
// parallel, so one global latest release would hide a stale line (or repeat
// every historical release in a line). Each older release in the same line
// would report the same state, so only the latest release per line is observed.
// Releases that provide a newer major are already covered by
// DATABASE_RELEASE_CONTRACT_AHEAD_OF_CURRENT_DEFINITION, and Contracts with no
// current definition are covered by DATABASE_RELEASE_CONTRACT_NOT_CURRENT, so
// both are excluded here.
func DatabaseReleaseContractCandidates(w ir.World, snapshot string) []DatabaseReleaseContractCandidate {
	contracts := currentContractVersions(w)
	var out []DatabaseReleaseContractCandidate
	for _, r := range latestDatabaseReleaseLines(w) {
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

// latestDatabaseReleaseLines selects the highest numeric release version in
// each major line (database plus release major). Invalid versions cannot be
// ordered and are ignored; a line with no valid release is not observed.
func latestDatabaseReleaseLines(w ir.World) []dbrelease.Definition {
	type lineKey struct {
		database core.CanonicalID
		major    int
	}
	latest := make(map[lineKey]dbrelease.Definition)
	for _, r := range w.DatabaseReleases {
		v, ok := parseReleaseVersion(r.Version)
		if !ok {
			continue
		}
		key := lineKey{database: r.Database, major: v[0]}
		current, ok := latest[key]
		if !ok || compareReleaseVersion(r.Version, current.Version) > 0 {
			latest[key] = r
		}
	}
	out := make([]dbrelease.Definition, 0, len(latest))
	for _, r := range latest {
		out = append(out, r)
	}
	return out
}
