package changeanalysis

import (
	"sort"

	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/releaseplan"
)

// UnreferencedUnsatisfiableGate is a read-only observation about a
// database-gate step. It is not an acceptance finding and does not prove that
// the gate is wrong: a gate may intentionally target a historical or future
// release state.
type UnreferencedUnsatisfiableGate struct {
	Snapshot        string           `json:"snapshot"`
	Solution        core.CanonicalID `json:"solution"`
	Version         string           `json:"version"`
	Step            core.LocalName   `json:"step"`
	Database        core.CanonicalID `json:"database"`
	Contract        core.CanonicalID `json:"contract"`
	CompatibleMajor int              `json:"compatibleMajor"`
}

// GateCandidates lists database-gate steps that no Application Release uses as
// a prerequisite and that neither the current Contract definition nor any
// Database Release can satisfy. Both conditions are required: a referenced gate
// is checked by release compilation, and a satisfiable gate has a provider.
// Only explicit declarations are scanned; runtime deployment state, dynamic
// consumers and provider behavior are not evaluated.
func GateCandidates(w ir.World, snapshot string) []UnreferencedUnsatisfiableGate {
	referenced := func(g releaseplan.Step) bool {
		for _, r := range w.ApplicationReleases {
			for _, db := range r.Databases {
				if db.TargetDatabase != g.Database {
					continue
				}
				for _, req := range db.RequiredContracts {
					if req.ContractID == g.Contract && req.CompatibleMajor == g.CompatibleMajor {
						return true
					}
				}
			}
		}
		return false
	}
	satisfied := func(g releaseplan.Step) bool {
		for _, db := range w.Databases {
			if db.ID != g.Database {
				continue
			}
			for _, c := range db.Contracts {
				if c.ID == g.Contract && c.Version.Major == g.CompatibleMajor {
					return true
				}
			}
		}
		for _, r := range w.DatabaseReleases {
			if r.Database != g.Database {
				continue
			}
			for _, p := range r.Provides {
				if p.ID == g.Contract && p.Version.Major == g.CompatibleMajor {
					return true
				}
			}
		}
		return false
	}
	var out []UnreferencedUnsatisfiableGate
	for _, r := range w.SolutionReleases {
		for _, s := range r.Steps {
			if s.Kind != releaseplan.StepDatabaseGate || referenced(s) || satisfied(s) {
				continue
			}
			out = append(out, UnreferencedUnsatisfiableGate{
				Snapshot: snapshot, Solution: r.ID, Version: r.Version, Step: s.ID,
				Database: s.Database, Contract: s.Contract, CompatibleMajor: s.CompatibleMajor,
			})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.Solution != b.Solution {
			return a.Solution < b.Solution
		}
		if a.Version != b.Version {
			return a.Version < b.Version
		}
		return a.Step < b.Step
	})
	return out
}
