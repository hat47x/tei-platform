package changeanalysis

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"tei/internal/core"
	dbrelease "tei/internal/database/release"
	"tei/internal/ir"
)

// DatabaseReleaseContractCandidate is a read-only observation about the latest
// Database Release of a database. It covers only the direction that
// DATABASE_RELEASE_CONTRACT_AHEAD_OF_CURRENT_DEFINITION does not warn about:
// the latest release provides an older major than the current definition.
// Older releases are historical and are not repeated; this is not an
// acceptance finding.
type DatabaseReleaseContractCandidate struct {
	Snapshot        string           `json:"snapshot"`
	Database        core.CanonicalID `json:"database"`
	Version         string           `json:"version"`
	Contract        core.CanonicalID `json:"contract"`
	ProvidedVersion string           `json:"providedContractVersion"`
	CurrentVersion  string           `json:"currentContractVersion"`
}

// DatabaseReleaseContractCandidates lists provided Contracts of the latest
// Database Release per database whose major is lower than the current
// definition's major. Each older release would report the same historical
// state, so only the latest release is observed. Releases that provide a newer
// major are already covered by
// DATABASE_RELEASE_CONTRACT_AHEAD_OF_CURRENT_DEFINITION, and Contracts with no
// current definition are covered by DATABASE_RELEASE_CONTRACT_NOT_CURRENT, so
// both are excluded here.
func DatabaseReleaseContractCandidates(w ir.World, snapshot string) []DatabaseReleaseContractCandidate {
	contracts := currentContractVersions(w)
	var out []DatabaseReleaseContractCandidate
	for _, r := range latestDatabaseReleases(w) {
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

// latestDatabaseReleases selects the highest numeric release version per
// database. Invalid versions cannot be ordered and are ignored; a database with
// no valid release is not observed.
func latestDatabaseReleases(w ir.World) map[core.CanonicalID]dbrelease.Definition {
	latest := make(map[core.CanonicalID]dbrelease.Definition)
	for _, r := range w.DatabaseReleases {
		if _, ok := parseReleaseVersion(r.Version); !ok {
			continue
		}
		current, ok := latest[r.Database]
		if !ok || compareReleaseVersion(r.Version, current.Version) > 0 {
			latest[r.Database] = r
		}
	}
	return latest
}

func compareReleaseVersion(a, b string) int {
	av, _ := parseReleaseVersion(a)
	bv, _ := parseReleaseVersion(b)
	for i := 0; i < 3; i++ {
		if av[i] < bv[i] {
			return -1
		}
		if av[i] > bv[i] {
			return 1
		}
	}
	return 0
}

func parseReleaseVersion(v string) ([3]int, bool) {
	var out [3]int
	parts := strings.Split(v, ".")
	if len(parts) != 3 {
		return out, false
	}
	for i, part := range parts {
		n, err := strconv.Atoi(part)
		if err != nil || n < 0 {
			return out, false
		}
		out[i] = n
	}
	return out, true
}
