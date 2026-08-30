package deployment

// ActionKind identifies a provider-neutral deployment intent. Concrete
// executors are bound explicitly per environment; TEI never infers an executor
// from an environment technology or resource name.
type ActionKind string

const (
	ActionDatabaseRelease   ActionKind = "database-release"
	ActionDatabaseGate      ActionKind = "database-gate"
	ActionApplicationStart  ActionKind = "application-start"
	ActionTrafficShift      ActionKind = "traffic-shift"
	ActionRolloutHealthGate ActionKind = "rollout-health-gate"
	ActionStopAcceptingWork ActionKind = "stop-accepting-work"
	ActionDrainCompleteGate ActionKind = "drain-complete-gate"
	ActionApplicationRetire ActionKind = "application-retire"
)

func (k ActionKind) Valid() bool {
	switch k {
	case ActionDatabaseRelease, ActionDatabaseGate, ActionApplicationStart, ActionTrafficShift,
		ActionRolloutHealthGate, ActionStopAcceptingWork, ActionDrainCompleteGate, ActionApplicationRetire:
		return true
	default:
		return false
	}
}

type ObservationKind string

const (
	ObservationApplicationRuntime ObservationKind = "application-runtime"
	ObservationDatabaseState      ObservationKind = "database-state"
)

func (k ObservationKind) Valid() bool {
	return k == ObservationApplicationRuntime || k == ObservationDatabaseState
}

type EvidenceKind string

const EvidenceRolloutHealth EvidenceKind = "rollout-health"

func (k EvidenceKind) Valid() bool { return k == EvidenceRolloutHealth }

// ObservationForAction returns the authoritative observation surface used to
// confirm an action's external effect. Evidence gates are intentionally
// excluded because they are collected through EvidenceProvider bindings.
func ObservationForAction(k ActionKind) (ObservationKind, bool) {
	switch k {
	case ActionDatabaseRelease, ActionDatabaseGate:
		return ObservationDatabaseState, true
	case ActionApplicationStart, ActionTrafficShift, ActionStopAcceptingWork, ActionDrainCompleteGate, ActionApplicationRetire:
		return ObservationApplicationRuntime, true
	default:
		return "", false
	}
}
