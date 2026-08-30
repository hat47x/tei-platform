package validation

import (
	"sort"

	apprelease "tei/internal/application/release"
	"tei/internal/core"
	"tei/internal/database"
	dbrelease "tei/internal/database/release"
	"tei/internal/deployment"
	"tei/internal/environment"
	"tei/internal/ir"
)

// ValidateDeployment validates an observed environment snapshot independently
// from canonical workspace validation. Deployment state is intentionally not
// stored in environment.tei because it changes on an operational lifecycle.
func ValidateDeployment(w ir.World, state deployment.Snapshot) []Diagnostic {
	state = state.Normalize()
	var out []Diagnostic

	var env environment.Environment
	envFound := false
	for _, e := range w.Environments {
		if e.ID == state.Environment {
			env, envFound = e, true
			break
		}
	}
	if !envFound {
		out = append(out, errd("UNKNOWN_DEPLOYMENT_ENVIRONMENT", "deployment snapshot refers to unknown environment %s", state.Environment))
	}

	appReleases := map[string]apprelease.Definition{}
	for _, r := range w.ApplicationReleases {
		appReleases[string(r.Application)+"@"+r.Version] = r
	}
	dbDefs := map[core.CanonicalID]database.Database{}
	for _, d := range w.Databases {
		dbDefs[d.ID] = d
	}
	dbReleases := map[string]dbrelease.Definition{}
	for _, r := range w.DatabaseReleases {
		dbReleases[string(r.Database)+"@"+r.Version] = r
	}

	seenApps := map[string]struct{}{}
	for _, a := range state.ActiveApplications {
		key := string(a.Application) + "@" + a.Version
		if _, dup := seenApps[key]; dup {
			out = append(out, errd("DUPLICATE_ACTIVE_APPLICATION_RELEASE", "deployment %s lists active release %s more than once", state.Environment, key))
		}
		seenApps[key] = struct{}{}
		ar, ok := appReleases[key]
		if !ok {
			out = append(out, errd("UNKNOWN_ACTIVE_APPLICATION_RELEASE", "deployment %s lists unknown active application release %s", state.Environment, key))
			continue
		}
		if !envFound {
			continue
		}
		for _, req := range ar.Databases {
			bound := false
			for _, b := range env.Bindings {
				if b.Application != ar.Application || b.Alias != req.Alias {
					continue
				}
				inst, ok := env.InstanceByID(b.Instance)
				if !ok {
					continue
				}
				bound = true
				if inst.Definition != req.TargetDatabase {
					out = append(out, errd("ACTIVE_RELEASE_BINDING_TARGET_MISMATCH", "deployment %s active %s datasource %s binds to %s, release requires %s", state.Environment, key, req.Alias, inst.Definition, req.TargetDatabase))
				}
			}
			if !bound {
				out = append(out, errd("ACTIVE_RELEASE_DATASOURCE_UNBOUND", "deployment %s active %s datasource %s is not bound", state.Environment, key, req.Alias))
			}
		}
	}

	activeSet := map[string]struct{}{}
	for _, a := range state.ActiveApplications {
		activeSet[string(a.Application)+"@"+a.Version] = struct{}{}
	}
	trafficByApp := map[core.CanonicalID]int{}
	seenRuntime := map[string]struct{}{}
	for _, rt := range state.ApplicationRuntime {
		key := string(rt.Application) + "@" + rt.Version
		if _, dup := seenRuntime[key]; dup {
			out = append(out, errd("DUPLICATE_APPLICATION_RUNTIME_STATE", "deployment %s lists runtime state %s more than once", state.Environment, key))
		}
		seenRuntime[key] = struct{}{}
		if _, ok := activeSet[key]; !ok {
			out = append(out, errd("RUNTIME_STATE_FOR_INACTIVE_RELEASE", "deployment %s has runtime state for inactive release %s", state.Environment, key))
		}
		if !rt.Lifecycle.Valid() {
			out = append(out, errd("INVALID_APPLICATION_RUNTIME_LIFECYCLE", "deployment %s runtime state %s has invalid lifecycle %q", state.Environment, key, rt.Lifecycle))
		}
		if rt.TrafficBasisPoints < 0 || rt.TrafficBasisPoints > 10000 {
			out = append(out, errd("INVALID_APPLICATION_TRAFFIC_ALLOCATION", "deployment %s runtime state %s has traffic %d outside 0..10000", state.Environment, key, rt.TrafficBasisPoints))
		}
		trafficByApp[rt.Application] += rt.TrafficBasisPoints
	}
	for app, total := range trafficByApp {
		if total > 10000 {
			out = append(out, errd("APPLICATION_TRAFFIC_OVERALLOCATED", "deployment %s application %s allocates %d traffic basis points across active releases", state.Environment, app, total))
		}
	}

	seenDB := map[core.CanonicalID]struct{}{}
	for _, st := range state.Databases {
		if _, dup := seenDB[st.Database]; dup {
			out = append(out, errd("DUPLICATE_DEPLOYMENT_DATABASE_STATE", "deployment %s lists database state %s more than once", state.Environment, st.Database))
		}
		seenDB[st.Database] = struct{}{}
		d, exists := dbDefs[st.Database]
		if !exists {
			out = append(out, errd("UNKNOWN_DEPLOYMENT_DATABASE", "deployment %s contains unknown database %s", state.Environment, st.Database))
			continue
		}
		if !st.DataReadiness.Valid() {
			out = append(out, errd("INVALID_DEPLOYMENT_DATA_READINESS", "deployment %s database %s has invalid readiness %q", state.Environment, st.Database, st.DataReadiness))
		}
		if st.Source == dbrelease.StateFromRelease {
			if st.Release == "" {
				out = append(out, errd("DEPLOYMENT_RELEASE_ID_REQUIRED", "deployment %s database %s is release-backed but has no release id", state.Environment, st.Database))
				continue
			}
			r, ok := dbReleases[string(st.Database)+"@"+st.Release]
			if !ok {
				out = append(out, errd("UNKNOWN_ACTIVE_DATABASE_RELEASE", "deployment %s database %s refers to unknown release %s", state.Environment, st.Database, st.Release))
				continue
			}
			if d.Ownership != database.OwnershipManagedHere {
				out = append(out, errd("EXTERNAL_DATABASE_CANNOT_USE_MANAGED_RELEASE_STATE", "deployment %s external database %s claims managed release %s", state.Environment, st.Database, st.Release))
			}
			if !sameProvidedContracts(st.Provides, r.Provides) || st.DataReadiness != r.DataReadiness {
				out = append(out, errd("DEPLOYMENT_DATABASE_RELEASE_STATE_DIVERGED", "deployment %s database %s state does not match immutable release %s", state.Environment, st.Database, st.Release))
			}
		} else if st.Source == dbrelease.StateFromObserved {
			if d.Ownership == database.OwnershipManagedHere && st.Release != "" {
				out = append(out, warnd("MANAGED_DATABASE_OBSERVED_WITHOUT_RELEASE_AUTHORITY", "deployment %s managed database %s is represented as observed state instead of release-backed state", state.Environment, st.Database))
			}
		} else {
			out = append(out, errd("INVALID_DEPLOYMENT_DATABASE_STATE_SOURCE", "deployment %s database %s has invalid state source %q", state.Environment, st.Database, st.Source))
		}
	}
	return out
}

func sameProvidedContracts(a, b []dbrelease.ProvidedContract) bool {
	aa := append([]dbrelease.ProvidedContract(nil), a...)
	bb := append([]dbrelease.ProvidedContract(nil), b...)
	sort.Slice(aa, func(i, j int) bool { return aa[i].ID < aa[j].ID })
	sort.Slice(bb, func(i, j int) bool { return bb[i].ID < bb[j].ID })
	if len(aa) != len(bb) {
		return false
	}
	for i := range aa {
		if aa[i].ID != bb[i].ID || aa[i].Version != bb[i].Version {
			return false
		}
	}
	return true
}
