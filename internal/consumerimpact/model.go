package consumerimpact

import (
	"sort"

	apprelease "tei/internal/application/release"
	"tei/internal/core"
	dbrelease "tei/internal/database/release"
	"tei/internal/deployment"
)

type ChangeKind string

const (
	ChangeAdded        ChangeKind = "added"
	ChangeWithdrawn    ChangeKind = "withdrawn"
	ChangeMajorChanged ChangeKind = "major-changed"
	ChangeMinorChanged ChangeKind = "minor-changed"
)

type ContractChange struct {
	Contract     core.CanonicalID `json:"contract"`
	Kind         ChangeKind       `json:"kind"`
	CurrentMajor int              `json:"currentMajor,omitempty"`
	CurrentMinor int              `json:"currentMinor,omitempty"`
	TargetMajor  int              `json:"targetMajor,omitempty"`
	TargetMinor  int              `json:"targetMinor,omitempty"`
}

type Consumer struct {
	Application   core.CanonicalID `json:"application"`
	Version       string           `json:"version"`
	Datasource    core.LocalName   `json:"datasource,omitempty"`
	Contract      core.CanonicalID `json:"contract,omitempty"`
	RequiredMajor int              `json:"requiredMajor,omitempty"`
	ProvidedMajor int              `json:"providedMajor,omitempty"`
	Reason        string           `json:"reason"`
}

type Impact struct {
	Environment      core.CanonicalID `json:"environment,omitempty"`
	Database         core.CanonicalID `json:"database"`
	CurrentRelease   string           `json:"currentRelease,omitempty"`
	TargetRelease    string           `json:"targetRelease"`
	ContractChanges  []ContractChange `json:"contractChanges,omitempty"`
	Consumers        []Consumer       `json:"impactedConsumers,omitempty"`
	UnknownConsumers []Consumer       `json:"unknownConsumers,omitempty"`
	Safe             bool             `json:"safe"`
}

// AnalyzeTarget evaluates whether replacing the current state of one database
// with target would preserve every currently active Application release. A
// caret requirement (^4) is treated as the compatible major family 4, not as
// "major >= 4"; a v5 contract is therefore not silently accepted by a v4
// consumer.
func AnalyzeTarget(target dbrelease.Definition, state deployment.Snapshot, apps []apprelease.Definition) Impact {
	state = state.Normalize()
	out := Impact{Environment: state.Environment, Database: target.Database, TargetRelease: target.Version, Safe: true}
	if current, ok := state.DatabaseState(target.Database); ok {
		out.CurrentRelease = current.Release
		out.ContractChanges = contractChanges(current.Provides, target.Provides)
	}

	appByKey := map[string]apprelease.Definition{}
	for _, r := range apps {
		appByKey[string(r.Application)+"@"+r.Version] = r
	}
	targetProvided := map[core.CanonicalID]dbrelease.ProvidedContract{}
	for _, p := range target.Provides {
		targetProvided[p.ID] = p
	}

	for _, active := range state.ActiveApplications {
		ar, ok := appByKey[string(active.Application)+"@"+active.Version]
		if !ok {
			out.UnknownConsumers = append(out.UnknownConsumers, Consumer{Application: active.Application, Version: active.Version, Reason: "active application release snapshot is unavailable"})
			out.Safe = false
			continue
		}
		for _, req := range ar.Databases {
			if req.TargetDatabase != target.Database {
				continue
			}
			for _, cr := range req.RequiredContracts {
				p, exists := targetProvided[cr.ContractID]
				switch {
				case !exists:
					out.Consumers = append(out.Consumers, Consumer{Application: ar.Application, Version: ar.Version, Datasource: req.Alias, Contract: cr.ContractID, RequiredMajor: cr.CompatibleMajor, Reason: "required contract would be withdrawn"})
					out.Safe = false
				case p.Version.Major != cr.CompatibleMajor:
					out.Consumers = append(out.Consumers, Consumer{Application: ar.Application, Version: ar.Version, Datasource: req.Alias, Contract: cr.ContractID, RequiredMajor: cr.CompatibleMajor, ProvidedMajor: p.Version.Major, Reason: "target contract is outside the consumer's compatible major family"})
					out.Safe = false
				case target.DataReadiness != dbrelease.DataReady:
					out.Consumers = append(out.Consumers, Consumer{Application: ar.Application, Version: ar.Version, Datasource: req.Alias, Contract: cr.ContractID, RequiredMajor: cr.CompatibleMajor, ProvidedMajor: p.Version.Major, Reason: "target database data is not ready for consumers"})
					out.Safe = false
				}
			}
		}
	}
	sort.Slice(out.Consumers, func(i, j int) bool {
		if out.Consumers[i].Application == out.Consumers[j].Application {
			if out.Consumers[i].Version == out.Consumers[j].Version {
				return out.Consumers[i].Contract < out.Consumers[j].Contract
			}
			return out.Consumers[i].Version < out.Consumers[j].Version
		}
		return out.Consumers[i].Application < out.Consumers[j].Application
	})
	sort.Slice(out.UnknownConsumers, func(i, j int) bool {
		if out.UnknownConsumers[i].Application == out.UnknownConsumers[j].Application {
			return out.UnknownConsumers[i].Version < out.UnknownConsumers[j].Version
		}
		return out.UnknownConsumers[i].Application < out.UnknownConsumers[j].Application
	})
	return out
}

func contractChanges(current, target []dbrelease.ProvidedContract) []ContractChange {
	cm := map[core.CanonicalID]dbrelease.ProvidedContract{}
	tm := map[core.CanonicalID]dbrelease.ProvidedContract{}
	ids := map[core.CanonicalID]struct{}{}
	for _, p := range current {
		cm[p.ID] = p
		ids[p.ID] = struct{}{}
	}
	for _, p := range target {
		tm[p.ID] = p
		ids[p.ID] = struct{}{}
	}
	var out []ContractChange
	for id := range ids {
		c, cok := cm[id]
		t, tok := tm[id]
		ch := ContractChange{Contract: id}
		switch {
		case !cok && tok:
			ch.Kind = ChangeAdded
			ch.TargetMajor, ch.TargetMinor = t.Version.Major, t.Version.Minor
		case cok && !tok:
			ch.Kind = ChangeWithdrawn
			ch.CurrentMajor, ch.CurrentMinor = c.Version.Major, c.Version.Minor
		case c.Version.Major != t.Version.Major:
			ch.Kind = ChangeMajorChanged
			ch.CurrentMajor, ch.CurrentMinor = c.Version.Major, c.Version.Minor
			ch.TargetMajor, ch.TargetMinor = t.Version.Major, t.Version.Minor
		case c.Version.Minor != t.Version.Minor:
			ch.Kind = ChangeMinorChanged
			ch.CurrentMajor, ch.CurrentMinor = c.Version.Major, c.Version.Minor
			ch.TargetMajor, ch.TargetMinor = t.Version.Major, t.Version.Minor
		default:
			continue
		}
		out = append(out, ch)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Contract < out[j].Contract })
	return out
}
