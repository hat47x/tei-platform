package release

import (
	"sort"

	"tei/internal/application"
	"tei/internal/core"
)

type CompatibilityState string

const (
	Compatible CompatibilityState = "compatible"
	Blocked    CompatibilityState = "blocked"
	Unknown    CompatibilityState = "unknown"
)

type ContractCheck struct {
	Datasource    core.LocalName     `json:"datasource"`
	Database      core.CanonicalID   `json:"database"`
	Contract      core.CanonicalID   `json:"contract"`
	RequiredMajor int                `json:"requiredMajor"`
	ProvidedMajor int                `json:"providedMajor,omitempty"`
	ProvidedMinor int                `json:"providedMinor,omitempty"`
	State         CompatibilityState `json:"state"`
	Reason        string             `json:"reason,omitempty"`
}

type ApplicationCompatibility struct {
	Application core.CanonicalID   `json:"application"`
	Overall     CompatibilityState `json:"overall"`
	Checks      []ContractCheck    `json:"checks"`
}

// CheckApplication evaluates one application against a set of independently
// versioned database states. Every DatabaseRequirement is evaluated; there is
// no single-database shortcut in the model.
func CheckApplication(app application.Application, states []StateSnapshot) ApplicationCompatibility {
	byDB := map[core.CanonicalID]StateSnapshot{}
	for _, s := range states {
		byDB[s.Database] = s
	}
	out := ApplicationCompatibility{Application: app.ID, Overall: Compatible}
	for _, req := range app.Databases {
		s, exists := byDB[req.TargetDatabase]
		if !exists {
			for _, cr := range req.RequiredContracts {
				out.Checks = append(out.Checks, ContractCheck{Datasource: req.Alias, Database: req.TargetDatabase, Contract: cr.ContractID, RequiredMajor: cr.CompatibleMajor, State: Unknown, Reason: "database state unavailable"})
			}
			out.Overall = combine(out.Overall, Unknown)
			continue
		}
		provided := map[core.CanonicalID]ProvidedContract{}
		for _, p := range s.Provides {
			provided[p.ID] = p
		}
		for _, cr := range req.RequiredContracts {
			c := ContractCheck{Datasource: req.Alias, Database: req.TargetDatabase, Contract: cr.ContractID, RequiredMajor: cr.CompatibleMajor}
			p, ok := provided[cr.ContractID]
			switch {
			case !ok:
				c.State, c.Reason = Blocked, "required database contract is not provided"
			case s.DataReadiness != DataReady:
				c.ProvidedMajor, c.ProvidedMinor = p.Version.Major, p.Version.Minor
				c.State, c.Reason = Blocked, "database data is not ready for consumers"
			case p.Version.Major != cr.CompatibleMajor:
				c.ProvidedMajor, c.ProvidedMinor = p.Version.Major, p.Version.Minor
				c.State, c.Reason = Blocked, "provided contract major version is outside the compatible major family"
			default:
				c.ProvidedMajor, c.ProvidedMinor = p.Version.Major, p.Version.Minor
				c.State = Compatible
			}
			out.Checks = append(out.Checks, c)
			out.Overall = combine(out.Overall, c.State)
		}
	}
	sort.Slice(out.Checks, func(i, j int) bool {
		if out.Checks[i].Datasource == out.Checks[j].Datasource {
			return out.Checks[i].Contract < out.Checks[j].Contract
		}
		return out.Checks[i].Datasource < out.Checks[j].Datasource
	})
	return out
}

func combine(a, b CompatibilityState) CompatibilityState {
	if a == Blocked || b == Blocked {
		return Blocked
	}
	if a == Unknown || b == Unknown {
		return Unknown
	}
	return Compatible
}
