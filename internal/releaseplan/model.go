package releaseplan

import (
	"sort"

	"tei/internal/core"
)

type Phase string

const (
	PhaseExpand     Phase = "expand"
	PhasePopulate   Phase = "populate"
	PhaseTransition Phase = "transition"
	PhaseRollout    Phase = "rollout"
	PhaseContract   Phase = "contract"
	PhaseVerify     Phase = "verify"
	PhaseOther      Phase = "other"
)

// Phases lists every accepted phase in the order the plan reads (#4094).
var Phases = []Phase{PhaseExpand, PhasePopulate, PhaseTransition, PhaseRollout, PhaseContract, PhaseVerify, PhaseOther}

func (p Phase) Valid() bool {
	for _, valid := range Phases {
		if p == valid {
			return true
		}
	}
	return false
}

type RolloutStrategy string

const (
	RolloutDirect    RolloutStrategy = "direct"
	RolloutCanary    RolloutStrategy = "canary"
	RolloutBlueGreen RolloutStrategy = "blue-green"
	RolloutGradual   RolloutStrategy = "gradual"
)

// RolloutStrategies lists every accepted strategy in report order. An empty
// value means "not declared" and is accepted, but it is not a strategy a
// diagnostic should offer: the message quotes this slice only (#4094).
var RolloutStrategies = []RolloutStrategy{RolloutDirect, RolloutCanary, RolloutBlueGreen, RolloutGradual}

func (r RolloutStrategy) Valid() bool {
	if r == "" {
		return true
	}
	for _, valid := range RolloutStrategies {
		if r == valid {
			return true
		}
	}
	return false
}

type StepKind string

const (
	StepDatabaseRelease    StepKind = "database-release"
	StepApplicationRelease StepKind = "application-release"
	StepApplicationRetire  StepKind = "application-retire"
	StepDatabaseGate       StepKind = "database-gate"
)

// StepKinds lists every accepted step kind in report order (#4094).
var StepKinds = []StepKind{StepDatabaseRelease, StepApplicationRelease, StepApplicationRetire, StepDatabaseGate}

func (k StepKind) Valid() bool {
	for _, valid := range StepKinds {
		if k == valid {
			return true
		}
	}
	return false
}

type Step struct {
	ID              core.LocalName   `json:"id"`
	Kind            StepKind         `json:"kind"`
	Phase           Phase            `json:"phase"`
	Database        core.CanonicalID `json:"database,omitempty"`
	Application     core.CanonicalID `json:"application,omitempty"`
	Version         string           `json:"version,omitempty"`
	Contract        core.CanonicalID `json:"contract,omitempty"`
	CompatibleMajor int              `json:"compatibleMajor,omitempty"`
	RolloutStrategy RolloutStrategy  `json:"rolloutStrategy,omitempty"`
	TrafficSteps    []int            `json:"trafficSteps,omitempty"`
}

func (s Step) EffectiveTrafficSteps() []int {
	if s.Kind != StepApplicationRelease {
		return nil
	}
	if len(s.TrafficSteps) > 0 {
		return append([]int(nil), s.TrafficSteps...)
	}
	switch s.RolloutStrategy {
	case RolloutCanary:
		return []int{1000, 5000, 10000}
	case RolloutGradual:
		return []int{2500, 5000, 7500, 10000}
	case RolloutBlueGreen, RolloutDirect, "":
		return []int{10000}
	default:
		return nil
	}
}

type Dependency struct {
	Step  core.LocalName `json:"step"`
	After core.LocalName `json:"after"`
}

// Definition is an orchestration/delivery plan only. Referenced Application
// and Database releases remain independently versioned immutable units.
type Definition struct {
	ID      core.CanonicalID `json:"id"`
	Version string           `json:"version"`
	Steps   []Step           `json:"steps"`
	After   []Dependency     `json:"after,omitempty"`
}

func (d Definition) Normalize() Definition {
	out := d
	out.Steps = append([]Step(nil), d.Steps...)
	out.After = append([]Dependency(nil), d.After...)
	sort.Slice(out.Steps, func(i, j int) bool { return out.Steps[i].ID < out.Steps[j].ID })
	sort.Slice(out.After, func(i, j int) bool {
		if out.After[i].Step == out.After[j].Step {
			return out.After[i].After < out.After[j].After
		}
		return out.After[i].Step < out.After[j].Step
	})
	return out
}
