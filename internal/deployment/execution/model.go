package execution

import (
	"fmt"
	"sort"

	"tei/internal/core"
	dbrelease "tei/internal/database/release"
	"tei/internal/deployment"
	"tei/internal/releaseplan"
)

type EventKind string

const (
	EventDatabaseReleaseApplied EventKind = "database-release-applied"
	EventDatabaseGateSatisfied  EventKind = "database-gate-satisfied"
	EventApplicationStarted     EventKind = "application-started"
	EventTrafficShifted         EventKind = "traffic-shifted"
	EventRolloutCheckpoint      EventKind = "rollout-checkpoint"
	EventDrainStarted           EventKind = "drain-started"
	EventDrainCompleted         EventKind = "drain-completed"
	EventApplicationRetired     EventKind = "application-retired"
)

type Event struct {
	Sequence           int              `json:"sequence"`
	Step               core.LocalName   `json:"step"`
	Kind               EventKind        `json:"kind"`
	Application        core.CanonicalID `json:"application,omitempty"`
	Version            string           `json:"version,omitempty"`
	Database           core.CanonicalID `json:"database,omitempty"`
	TrafficBasisPoints int              `json:"trafficBasisPoints,omitempty"`
	Message            string           `json:"message,omitempty"`
}

type RunStatus string

const (
	RunSucceeded RunStatus = "succeeded"
	RunBlocked   RunStatus = "blocked"
)

// Run is a deterministic deployment simulation/event trace. It never calls a
// cloud/runtime deployment API; real execution must supply evidence for gates
// and checkpoints instead of treating this simulation as authority.
type Run struct {
	SolutionID      core.CanonicalID                     `json:"solutionId"`
	SolutionVersion string                               `json:"solutionVersion"`
	Environment     core.CanonicalID                     `json:"environment"`
	Status          RunStatus                            `json:"status"`
	Initial         deployment.Snapshot                  `json:"initial"`
	Final           deployment.Snapshot                  `json:"final"`
	Applications    []deployment.ApplicationRuntimeState `json:"applications,omitempty"`
	Events          []Event                              `json:"events,omitempty"`
	Findings        []string                             `json:"findings,omitempty"`
}

// Simulate executes a compiled Solution Release deterministically against an
// observed Deployment Snapshot. It models blue/green/canary routing and drain
// boundaries without claiming that operational checkpoints actually passed.
func Simulate(def releaseplan.Definition, compiled releaseplan.Compiled, initial deployment.Snapshot, dbReleases []dbrelease.Definition) Run {
	initial = initial.Normalize()
	r := Run{SolutionID: def.ID, SolutionVersion: def.Version, Environment: initial.Environment, Status: RunSucceeded, Initial: initial}
	if !compiled.Executable {
		r.Status = RunBlocked
		r.Findings = append(r.Findings, compiled.Findings...)
		r.Final = initial
		return r
	}

	appStates, err := initializeRuntime(initial)
	if err != nil {
		r.Status = RunBlocked
		r.Findings = append(r.Findings, err.Error())
		r.Final = initial
		return r
	}
	dbByKey := map[string]dbrelease.Definition{}
	for _, d := range dbReleases {
		dbByKey[string(d.Database)+"@"+d.Version] = d
	}
	state := initial
	order, err := topologicalOrder(def.Steps, compiled.Edges)
	if err != nil {
		r.Status = RunBlocked
		r.Findings = append(r.Findings, err.Error())
		r.Final = initial
		return r
	}
	seq := 0
	emit := func(e Event) {
		seq++
		e.Sequence = seq
		r.Events = append(r.Events, e)
	}

	for _, step := range order {
		switch step.Kind {
		case releaseplan.StepDatabaseRelease:
			rel, ok := dbByKey[string(step.Database)+"@"+step.Version]
			if !ok {
				r.Status = RunBlocked
				r.Findings = append(r.Findings, fmt.Sprintf("database release not found during deployment execution: %s@%s", step.Database, step.Version))
				goto done
			}
			state.Databases = upsertDBState(state.Databases, dbrelease.StateFromDefinition(rel))
			emit(Event{Step: step.ID, Kind: EventDatabaseReleaseApplied, Database: step.Database, Version: step.Version})

		case releaseplan.StepDatabaseGate:
			emit(Event{Step: step.ID, Kind: EventDatabaseGateSatisfied, Database: step.Database, Message: string(step.Contract)})

		case releaseplan.StepApplicationRelease:
			key := appKey(step.Application, step.Version)
			if _, exists := appStates[key]; !exists {
				appStates[key] = deployment.ApplicationRuntimeState{Application: step.Application, Version: step.Version, Lifecycle: deployment.LifecycleStarting}
				state.ActiveApplications = appendActive(state.ActiveApplications, deployment.ApplicationReleaseRef{Application: step.Application, Version: step.Version})
			}
			s := appStates[key]
			s.Lifecycle = deployment.LifecycleServing
			s.AcceptingWork = true
			appStates[key] = s
			emit(Event{Step: step.ID, Kind: EventApplicationStarted, Application: step.Application, Version: step.Version})

			if step.RolloutStrategy == releaseplan.RolloutBlueGreen {
				emit(Event{Step: step.ID, Kind: EventRolloutCheckpoint, Application: step.Application, Version: step.Version, Message: "simulation checkpoint before blue-green cutover; real execution requires health evidence"})
			}
			trafficSteps := step.EffectiveTrafficSteps()
			for i, bps := range trafficSteps {
				if err := shiftApplicationTraffic(appStates, step.Application, step.Version, bps); err != nil {
					r.Status = RunBlocked
					r.Findings = append(r.Findings, fmt.Sprintf("step %s: %v", step.ID, err))
					goto done
				}
				emit(Event{Step: step.ID, Kind: EventTrafficShifted, Application: step.Application, Version: step.Version, TrafficBasisPoints: bps})
				if i < len(trafficSteps)-1 {
					emit(Event{Step: step.ID, Kind: EventRolloutCheckpoint, Application: step.Application, Version: step.Version, TrafficBasisPoints: bps, Message: "simulation checkpoint; real execution must attach rollout health evidence before continuing"})
				}
			}

		case releaseplan.StepApplicationRetire:
			key := appKey(step.Application, step.Version)
			s, ok := appStates[key]
			if !ok {
				r.Status = RunBlocked
				r.Findings = append(r.Findings, fmt.Sprintf("cannot retire inactive application release %s@%s", step.Application, step.Version))
				goto done
			}
			if s.TrafficBasisPoints != 0 {
				r.Status = RunBlocked
				r.Findings = append(r.Findings, fmt.Sprintf("cannot drain %s@%s while traffic is %d basis points", step.Application, step.Version, s.TrafficBasisPoints))
				goto done
			}
			s.AcceptingWork = false
			s.Lifecycle = deployment.LifecycleDraining
			appStates[key] = s
			emit(Event{Step: step.ID, Kind: EventDrainStarted, Application: step.Application, Version: step.Version})

			s.DrainComplete = true
			s.Lifecycle = deployment.LifecycleQuiescent
			appStates[key] = s
			emit(Event{Step: step.ID, Kind: EventDrainCompleted, Application: step.Application, Version: step.Version})

			if !s.SafeToRetire() {
				r.Status = RunBlocked
				r.Findings = append(r.Findings, fmt.Sprintf("application release %s@%s is not quiescent", step.Application, step.Version))
				goto done
			}
			state.ActiveApplications = removeActive(state.ActiveApplications, step.Application, step.Version)
			delete(appStates, key)
			emit(Event{Step: step.ID, Kind: EventApplicationRetired, Application: step.Application, Version: step.Version})
		}
	}

done:
	state.ApplicationRuntime = sortedRuntimeStates(appStates)
	state = state.Normalize()
	r.Final = state
	r.Applications = append([]deployment.ApplicationRuntimeState(nil), state.ApplicationRuntime...)
	return r
}

func initializeRuntime(initial deployment.Snapshot) (map[string]deployment.ApplicationRuntimeState, error) {
	out := map[string]deployment.ApplicationRuntimeState{}
	if len(initial.ApplicationRuntime) > 0 {
		for _, s := range initial.ApplicationRuntime {
			out[appKey(s.Application, s.Version)] = s
		}
		return out, nil
	}
	byApp := map[core.CanonicalID][]deployment.ApplicationReleaseRef{}
	for _, a := range initial.ActiveApplications {
		byApp[a.Application] = append(byApp[a.Application], a)
	}
	for app, xs := range byApp {
		if len(xs) != 1 {
			return nil, fmt.Errorf("initial traffic allocation for application %s is ambiguous across %d active releases; provide applicationRuntime in deployment snapshot", app, len(xs))
		}
		a := xs[0]
		out[appKey(a.Application, a.Version)] = deployment.ApplicationRuntimeState{Application: a.Application, Version: a.Version, Lifecycle: deployment.LifecycleServing, TrafficBasisPoints: 10000, AcceptingWork: true}
	}
	return out, nil
}

func shiftApplicationTraffic(states map[string]deployment.ApplicationRuntimeState, app core.CanonicalID, targetVersion string, targetBps int) error {
	if targetBps < 0 || targetBps > 10000 {
		return fmt.Errorf("invalid traffic basis points %d", targetBps)
	}
	targetKey := appKey(app, targetVersion)
	target, ok := states[targetKey]
	if !ok {
		return fmt.Errorf("target release %s@%s is not started", app, targetVersion)
	}
	var others []string
	for k, s := range states {
		if s.Application == app && k != targetKey {
			others = append(others, k)
		}
	}
	if len(others) > 1 {
		return fmt.Errorf("traffic redistribution for %s is ambiguous across %d prior releases", app, len(others))
	}
	target.TrafficBasisPoints = targetBps
	states[targetKey] = target
	if len(others) == 1 {
		o := states[others[0]]
		o.TrafficBasisPoints = 10000 - targetBps
		states[others[0]] = o
	} else if targetBps != 10000 {
		return fmt.Errorf("cannot allocate %d basis points to sole release %s@%s", targetBps, app, targetVersion)
	}
	return nil
}

func topologicalOrder(steps []releaseplan.Step, edges []releaseplan.CompiledEdge) ([]releaseplan.Step, error) {
	byID := map[core.LocalName]releaseplan.Step{}
	indeg := map[core.LocalName]int{}
	next := map[core.LocalName][]core.LocalName{}
	for _, s := range steps {
		byID[s.ID] = s
		indeg[s.ID] = 0
	}
	seen := map[string]struct{}{}
	for _, e := range edges {
		key := string(e.From) + "|" + string(e.To)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		if _, ok := byID[e.From]; !ok {
			continue
		}
		if _, ok := byID[e.To]; !ok {
			continue
		}
		indeg[e.To]++
		next[e.From] = append(next[e.From], e.To)
	}
	var ready []core.LocalName
	for id, n := range indeg {
		if n == 0 {
			ready = append(ready, id)
		}
	}
	sort.Slice(ready, func(i, j int) bool { return ready[i] < ready[j] })
	var out []releaseplan.Step
	for len(ready) > 0 {
		id := ready[0]
		ready = ready[1:]
		out = append(out, byID[id])
		sort.Slice(next[id], func(i, j int) bool { return next[id][i] < next[id][j] })
		for _, to := range next[id] {
			indeg[to]--
			if indeg[to] == 0 {
				ready = append(ready, to)
				sort.Slice(ready, func(i, j int) bool { return ready[i] < ready[j] })
			}
		}
	}
	if len(out) != len(steps) {
		return nil, fmt.Errorf("deployment execution graph contains a cycle")
	}
	return out, nil
}

func appendActive(xs []deployment.ApplicationReleaseRef, a deployment.ApplicationReleaseRef) []deployment.ApplicationReleaseRef {
	for _, x := range xs {
		if x == a {
			return xs
		}
	}
	return append(xs, a)
}

func removeActive(xs []deployment.ApplicationReleaseRef, app core.CanonicalID, version string) []deployment.ApplicationReleaseRef {
	out := xs[:0]
	for _, x := range xs {
		if x.Application == app && x.Version == version {
			continue
		}
		out = append(out, x)
	}
	return out
}

func upsertDBState(xs []dbrelease.StateSnapshot, s dbrelease.StateSnapshot) []dbrelease.StateSnapshot {
	for i := range xs {
		if xs[i].Database == s.Database {
			xs[i] = s
			return xs
		}
	}
	return append(xs, s)
}

func appKey(app core.CanonicalID, version string) string { return string(app) + "@" + version }

func sortedRuntimeStates(m map[string]deployment.ApplicationRuntimeState) []deployment.ApplicationRuntimeState {
	out := make([]deployment.ApplicationRuntimeState, 0, len(m))
	for _, s := range m {
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Application == out[j].Application {
			return out[i].Version < out[j].Version
		}
		return out[i].Application < out[j].Application
	})
	return out
}
