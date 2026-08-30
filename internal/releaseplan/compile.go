package releaseplan

import (
	"fmt"
	"sort"

	apprelease "tei/internal/application/release"
	"tei/internal/consumerimpact"
	"tei/internal/core"
	"tei/internal/database"
	dbrelease "tei/internal/database/release"
	"tei/internal/deployment"
)

type CompiledEdge struct {
	From     core.LocalName   `json:"from"`
	To       core.LocalName   `json:"to"`
	Relation string           `json:"relation"`
	Contract core.CanonicalID `json:"contract,omitempty"`
}

type ConsumerGate struct {
	Step                core.LocalName        `json:"step"`
	ImpactBeforePlan    consumerimpact.Impact `json:"impactBeforePlan"`
	RequiredRetirements []core.LocalName      `json:"requiredRetirements,omitempty"`
	ImpactAfterPlan     consumerimpact.Impact `json:"impactAfterPlan"`
	Resolved            bool                  `json:"resolved"`
}

type Compiled struct {
	ID                    core.CanonicalID `json:"id"`
	Version               string           `json:"version"`
	DeploymentEnvironment core.CanonicalID `json:"deploymentEnvironment,omitempty"`
	Steps                 []Step           `json:"steps"`
	Edges                 []CompiledEdge   `json:"edges"`
	ConsumerGates         []ConsumerGate   `json:"consumerGates,omitempty"`
	Executable            bool             `json:"executable"`
	Findings              []string         `json:"findings,omitempty"`
}

// Compile performs repository-only structural/semantic compilation. Runtime
// consumer withdrawal checks require an observed deployment Snapshot and are
// added by CompileWithDeployment.
func Compile(def Definition, apps []apprelease.Definition, dbs []dbrelease.Definition) Compiled {
	return compile(def, apps, dbs, nil)
}

// CompileWithDeployment augments the release graph with observed active
// consumers. Contract-reducing database releases are allowed only when every
// incompatible active Application release is retired before the database step.
func CompileWithDeployment(def Definition, apps []apprelease.Definition, dbs []dbrelease.Definition, state deployment.Snapshot) Compiled {
	state = state.Normalize()
	return compile(def, apps, dbs, &state)
}

func compile(def Definition, apps []apprelease.Definition, dbs []dbrelease.Definition, state *deployment.Snapshot) Compiled {
	def = def.Normalize()
	out := Compiled{ID: def.ID, Version: def.Version, Steps: append([]Step(nil), def.Steps...), Executable: true}
	if state != nil {
		out.DeploymentEnvironment = state.Environment
	}
	stepByID := map[core.LocalName]Step{}
	for _, s := range def.Steps {
		stepByID[s.ID] = s
	}
	for _, dep := range def.After {
		out.Edges = append(out.Edges, CompiledEdge{From: dep.After, To: dep.Step, Relation: "after"})
	}
	appByKey := map[string]apprelease.Definition{}
	for _, r := range apps {
		appByKey[string(r.Application)+"@"+r.Version] = r
	}
	dbByKey := map[string]dbrelease.Definition{}
	for _, r := range dbs {
		dbByKey[string(r.Database)+"@"+r.Version] = r
	}

	// Resolve referenced release units first, independently from their current
	// definitions. Historical releases are immutable snapshots.
	for _, s := range def.Steps {
		switch s.Kind {
		case StepApplicationRelease, StepApplicationRetire:
			if _, ok := appByKey[string(s.Application)+"@"+s.Version]; !ok {
				out.Executable = false
				out.Findings = append(out.Findings, fmt.Sprintf("application release not found: %s@%s", s.Application, s.Version))
			}
		case StepDatabaseRelease:
			if _, ok := dbByKey[string(s.Database)+"@"+s.Version]; !ok {
				out.Executable = false
				out.Findings = append(out.Findings, fmt.Sprintf("database release not found: %s@%s", s.Database, s.Version))
			}
		}
	}

	// Application rollout prerequisites are contract-based. If several DB
	// releases in the same plan provide the same compatible contract family,
	// an explicit predecessor edge can disambiguate which state enables the
	// rollout. We never select "latest" heuristically.
	for _, s := range def.Steps {
		if s.Kind != StepApplicationRelease {
			continue
		}
		ar, ok := appByKey[string(s.Application)+"@"+s.Version]
		if !ok {
			continue
		}
		for _, req := range ar.Databases {
			for _, cr := range req.RequiredContracts {
				matches := matchingPrerequisites(def.Steps, req.TargetDatabase, cr, dbByKey)
				chosen, status := choosePrerequisite(matches, s.ID, out.Edges)
				switch status {
				case "none":
					out.Executable = false
					out.Findings = append(out.Findings, fmt.Sprintf("no prerequisite step provides %s compatible major = %d for %s.%s", cr.ContractID, cr.CompatibleMajor, ar.Application, req.Alias))
				case "ambiguous":
					out.Executable = false
					out.Findings = append(out.Findings, fmt.Sprintf("ambiguous prerequisite for %s: %v", cr.ContractID, matches))
				case "chosen":
					out.Edges = append(out.Edges, CompiledEdge{From: chosen, To: s.ID, Relation: "requires-contract", Contract: cr.ContractID})
				}
			}
		}
	}
	out.Edges = dedupeEdges(out.Edges)

	if state != nil {
		// A retirement against observed state must name a release that is
		// active now, or one that this same plan rolls out before retiring.
		for _, s := range def.Steps {
			if s.Kind != StepApplicationRetire {
				continue
			}
			if state.IsActive(s.Application, s.Version) {
				continue
			}
			rollout := findStep(def.Steps, StepApplicationRelease, s.Application, s.Version)
			if rollout == "" || !pathExists(rollout, s.ID, out.Edges) {
				out.Executable = false
				out.Findings = append(out.Findings, fmt.Sprintf("application retire step %s targets %s@%s which is not active in environment %s and is not rolled out before retirement", s.ID, s.Application, s.Version, state.Environment))
			}
		}

		// First pass: determine which currently/planned active consumers would
		// be broken by each target database release and attach semantic retire
		// gates when a unique matching retirement exists.
		for _, s := range def.Steps {
			if s.Kind != StepDatabaseRelease {
				continue
			}
			target, ok := dbByKey[string(s.Database)+"@"+s.Version]
			if !ok {
				continue
			}
			effective := effectiveStateBefore(s.ID, *state, def.Steps, out.Edges, dbByKey)
			impact := consumerimpact.AnalyzeTarget(target, effective, apps)
			gate := ConsumerGate{Step: s.ID, ImpactBeforePlan: impact, Resolved: impact.Safe}

			if hasReducingContractChange(impact.ContractChanges) && s.Phase != PhaseContract {
				out.Executable = false
				out.Findings = append(out.Findings, fmt.Sprintf("database release step %s reduces its public contract surface but phase is %s, expected contract", s.ID, s.Phase))
			}

			for _, c := range impact.Consumers {
				matches := findRetireSteps(def.Steps, c.Application, c.Version)
				switch len(matches) {
				case 0:
					out.Executable = false
					out.Findings = append(out.Findings, fmt.Sprintf("database release step %s would break active consumer %s@%s (%s) with no retirement step", s.ID, c.Application, c.Version, c.Contract))
				case 1:
					out.Edges = append(out.Edges, CompiledEdge{From: matches[0], To: s.ID, Relation: "retires-incompatible-consumer", Contract: c.Contract})
					gate.RequiredRetirements = append(gate.RequiredRetirements, matches[0])
				default:
					out.Executable = false
					out.Findings = append(out.Findings, fmt.Sprintf("database release step %s has ambiguous retirement steps for active consumer %s@%s: %v", s.ID, c.Application, c.Version, matches))
				}
			}
			for _, c := range impact.UnknownConsumers {
				out.Executable = false
				out.Findings = append(out.Findings, fmt.Sprintf("database release step %s cannot prove safety for active consumer %s@%s: %s", s.ID, c.Application, c.Version, c.Reason))
			}
			sort.Slice(gate.RequiredRetirements, func(i, j int) bool { return gate.RequiredRetirements[i] < gate.RequiredRetirements[j] })
			out.ConsumerGates = append(out.ConsumerGates, gate)
			out.Edges = dedupeEdges(out.Edges)
		}

		// Second pass evaluates the graph after semantic retire edges were
		// added. This catches newly rolled out consumers too, rather than only
		// the releases active in the initial deployment snapshot.
		for i := range out.ConsumerGates {
			g := &out.ConsumerGates[i]
			s := stepByID[g.Step]
			target, ok := dbByKey[string(s.Database)+"@"+s.Version]
			if !ok {
				continue
			}
			effective := effectiveStateBefore(s.ID, *state, def.Steps, out.Edges, dbByKey)
			g.ImpactAfterPlan = consumerimpact.AnalyzeTarget(target, effective, apps)
			g.Resolved = g.ImpactAfterPlan.Safe
			if !g.Resolved {
				out.Executable = false
				out.Findings = append(out.Findings, fmt.Sprintf("database release step %s remains unsafe after planned retirements", s.ID))
			}
		}
	}

	out.Edges = dedupeEdges(out.Edges)
	if cycle := detectCycle(def.Steps, out.Edges); len(cycle) > 0 {
		out.Executable = false
		out.Findings = append(out.Findings, "release dependency cycle: "+fmt.Sprint(cycle))
	}
	sort.Slice(out.Edges, func(i, j int) bool {
		if out.Edges[i].From == out.Edges[j].From {
			if out.Edges[i].To == out.Edges[j].To {
				if out.Edges[i].Relation == out.Edges[j].Relation {
					return out.Edges[i].Contract < out.Edges[j].Contract
				}
				return out.Edges[i].Relation < out.Edges[j].Relation
			}
			return out.Edges[i].To < out.Edges[j].To
		}
		return out.Edges[i].From < out.Edges[j].From
	})
	sort.Slice(out.ConsumerGates, func(i, j int) bool { return out.ConsumerGates[i].Step < out.ConsumerGates[j].Step })
	sort.Strings(out.Findings)
	return out
}

func matchingPrerequisites(steps []Step, db core.CanonicalID, cr database.ContractRequirement, releases map[string]dbrelease.Definition) []core.LocalName {
	var out []core.LocalName
	for _, s := range steps {
		switch s.Kind {
		case StepDatabaseRelease:
			if s.Database != db {
				continue
			}
			r, ok := releases[string(s.Database)+"@"+s.Version]
			if !ok || r.DataReadiness != dbrelease.DataReady {
				continue
			}
			for _, pc := range r.Provides {
				if pc.ID == cr.ContractID && pc.Version.Major == cr.CompatibleMajor {
					out = append(out, s.ID)
					break
				}
			}
		case StepDatabaseGate:
			if s.Database == db && s.Contract == cr.ContractID && s.CompatibleMajor == cr.CompatibleMajor {
				out = append(out, s.ID)
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func choosePrerequisite(matches []core.LocalName, consumer core.LocalName, edges []CompiledEdge) (core.LocalName, string) {
	if len(matches) == 0 {
		return "", "none"
	}
	var ancestors []core.LocalName
	for _, m := range matches {
		if pathExists(m, consumer, edges) {
			ancestors = append(ancestors, m)
		}
	}
	if len(ancestors) == 1 {
		return ancestors[0], "chosen"
	}
	if len(ancestors) > 1 {
		return "", "ambiguous"
	}
	if len(matches) == 1 {
		return matches[0], "chosen"
	}
	return "", "ambiguous"
}

func findRetireSteps(steps []Step, app core.CanonicalID, version string) []core.LocalName {
	var out []core.LocalName
	for _, s := range steps {
		if s.Kind == StepApplicationRetire && s.Application == app && s.Version == version {
			out = append(out, s.ID)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func findStep(steps []Step, kind StepKind, app core.CanonicalID, version string) core.LocalName {
	var found core.LocalName
	for _, s := range steps {
		if s.Kind == kind && s.Application == app && s.Version == version {
			if found != "" {
				return ""
			}
			found = s.ID
		}
	}
	return found
}

func hasReducingContractChange(changes []consumerimpact.ContractChange) bool {
	for _, c := range changes {
		if c.Kind == consumerimpact.ChangeWithdrawn || c.Kind == consumerimpact.ChangeMajorChanged {
			return true
		}
	}
	return false
}

func effectiveStateBefore(target core.LocalName, initial deployment.Snapshot, steps []Step, edges []CompiledEdge, dbReleases map[string]dbrelease.Definition) deployment.Snapshot {
	out := initial.Normalize()
	active := map[string]deployment.ApplicationReleaseRef{}
	initiallyActive := map[string]bool{}
	for _, a := range out.ActiveApplications {
		k := string(a.Application) + "@" + a.Version
		active[k] = a
		initiallyActive[k] = true
	}

	rollouts := map[string][]core.LocalName{}
	for _, s := range steps {
		if s.Kind == StepApplicationRelease && pathExists(s.ID, target, edges) {
			k := string(s.Application) + "@" + s.Version
			active[k] = deployment.ApplicationReleaseRef{Application: s.Application, Version: s.Version}
			rollouts[k] = append(rollouts[k], s.ID)
		}
	}
	for _, s := range steps {
		if s.Kind != StepApplicationRetire || !pathExists(s.ID, target, edges) {
			continue
		}
		k := string(s.Application) + "@" + s.Version
		if initiallyActive[k] {
			delete(active, k)
			continue
		}
		// A plan-created release is considered retired only when every rollout
		// of that exact release is guaranteed to happen before this retirement.
		rs := rollouts[k]
		if len(rs) == 0 {
			continue
		}
		allBefore := true
		for _, rid := range rs {
			if !pathExists(rid, s.ID, edges) {
				allBefore = false
				break
			}
		}
		if allBefore {
			delete(active, k)
		}
	}
	out.ActiveApplications = out.ActiveApplications[:0]
	for _, a := range active {
		out.ActiveApplications = append(out.ActiveApplications, a)
	}

	// Database release steps that are guaranteed predecessors establish the
	// database state seen by later steps. For each database, choose the unique
	// maximal predecessor in the partial order. If two predecessor releases are
	// unordered, their effective state is ambiguous and TEI does not invent a
	// version ordering from filenames or semantic versions.
	candidates := map[core.CanonicalID][]Step{}
	for _, s := range steps {
		if s.Kind == StepDatabaseRelease && s.ID != target && pathExists(s.ID, target, edges) {
			candidates[s.Database] = append(candidates[s.Database], s)
		}
	}
	for db, xs := range candidates {
		var maxima []Step
		for _, c := range xs {
			isBeforeAnother := false
			for _, other := range xs {
				if c.ID != other.ID && pathExists(c.ID, other.ID, edges) {
					isBeforeAnother = true
					break
				}
			}
			if !isBeforeAnother {
				maxima = append(maxima, c)
			}
		}
		if len(maxima) != 1 {
			continue
		}
		r, ok := dbReleases[string(db)+"@"+maxima[0].Version]
		if ok {
			out.Databases = upsertDatabaseState(out.Databases, dbrelease.StateFromDefinition(r))
		}
	}
	return out.Normalize()
}

func upsertDatabaseState(states []dbrelease.StateSnapshot, state dbrelease.StateSnapshot) []dbrelease.StateSnapshot {
	for i := range states {
		if states[i].Database == state.Database {
			states[i] = state
			return states
		}
	}
	return append(states, state)
}

func dedupeEdges(xs []CompiledEdge) []CompiledEdge {
	seen := map[string]struct{}{}
	out := make([]CompiledEdge, 0, len(xs))
	for _, e := range xs {
		k := fmt.Sprintf("%s|%s|%s|%s", e.From, e.To, e.Relation, e.Contract)
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}
		out = append(out, e)
	}
	return out
}

func pathExists(from, to core.LocalName, edges []CompiledEdge) bool {
	if from == to {
		return true
	}
	adj := map[core.LocalName][]core.LocalName{}
	for _, e := range edges {
		adj[e.From] = append(adj[e.From], e.To)
	}
	seen := map[core.LocalName]bool{from: true}
	q := []core.LocalName{from}
	for len(q) > 0 {
		n := q[0]
		q = q[1:]
		for _, m := range adj[n] {
			if m == to {
				return true
			}
			if !seen[m] {
				seen[m] = true
				q = append(q, m)
			}
		}
	}
	return false
}

func detectCycle(steps []Step, edges []CompiledEdge) []core.LocalName {
	adj := map[core.LocalName][]core.LocalName{}
	for _, e := range edges {
		adj[e.From] = append(adj[e.From], e.To)
	}
	state := map[core.LocalName]int{}
	var stack []core.LocalName
	var cycle []core.LocalName
	var visit func(core.LocalName) bool
	visit = func(n core.LocalName) bool {
		state[n] = 1
		stack = append(stack, n)
		for _, m := range adj[n] {
			if state[m] == 0 {
				if visit(m) {
					return true
				}
			} else if state[m] == 1 {
				start := 0
				for i := range stack {
					if stack[i] == m {
						start = i
						break
					}
				}
				cycle = append(cycle, stack[start:]...)
				cycle = append(cycle, m)
				return true
			}
		}
		stack = stack[:len(stack)-1]
		state[n] = 2
		return false
	}
	for _, s := range steps {
		if state[s.ID] == 0 && visit(s.ID) {
			return cycle
		}
	}
	return nil
}
