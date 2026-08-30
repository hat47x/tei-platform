package deployment

import (
	"sort"

	"tei/internal/core"
	dbrelease "tei/internal/database/release"
)

// ApplicationReleaseRef identifies one immutable Application release that is
// currently capable of consuming environment resources. Multiple versions of
// one Application may be active simultaneously during blue/green or gradual
// rollout.
type ApplicationReleaseRef struct {
	Application core.CanonicalID `json:"application"`
	Version     string           `json:"version"`
}

// ApplicationLifecycle is live operational state, separate from immutable release identity.
type ApplicationLifecycle string

const (
	LifecycleStarting  ApplicationLifecycle = "starting"
	LifecycleServing   ApplicationLifecycle = "serving"
	LifecycleDraining  ApplicationLifecycle = "draining"
	LifecycleQuiescent ApplicationLifecycle = "quiescent"
)

func (s ApplicationLifecycle) Valid() bool {
	return s == LifecycleStarting || s == LifecycleServing || s == LifecycleDraining || s == LifecycleQuiescent
}

// ApplicationRuntimeState captures routing and work-admission state. Traffic is
// basis points (0..10000). Traffic=0 alone is not enough to prove safe
// retirement because background/job work may still be accepted.
type ApplicationRuntimeState struct {
	Application        core.CanonicalID     `json:"application"`
	Version            string               `json:"version"`
	Lifecycle          ApplicationLifecycle `json:"lifecycle"`
	TrafficBasisPoints int                  `json:"trafficBasisPoints"`
	AcceptingWork      bool                 `json:"acceptingWork"`
	DrainComplete      bool                 `json:"drainComplete"`
}

func (s ApplicationRuntimeState) SafeToRetire() bool {
	return s.TrafficBasisPoints == 0 && !s.AcceptingWork && s.DrainComplete && s.Lifecycle == LifecycleQuiescent
}

// Snapshot is observed deployment state, not canonical environment
// configuration. It deliberately lives outside environment.tei because active
// releases and live database states change on a different lifecycle.
type Snapshot struct {
	Environment        core.CanonicalID          `json:"environment"`
	ObservedAt         string                    `json:"observedAt,omitempty"`
	ActiveApplications []ApplicationReleaseRef   `json:"activeApplications,omitempty"`
	ApplicationRuntime []ApplicationRuntimeState `json:"applicationRuntime,omitempty"`
	Databases          []dbrelease.StateSnapshot `json:"databases,omitempty"`
}

func (s Snapshot) Normalize() Snapshot {
	out := s
	out.ActiveApplications = append([]ApplicationReleaseRef(nil), s.ActiveApplications...)
	sort.Slice(out.ActiveApplications, func(i, j int) bool {
		if out.ActiveApplications[i].Application == out.ActiveApplications[j].Application {
			return out.ActiveApplications[i].Version < out.ActiveApplications[j].Version
		}
		return out.ActiveApplications[i].Application < out.ActiveApplications[j].Application
	})
	out.ApplicationRuntime = append([]ApplicationRuntimeState(nil), s.ApplicationRuntime...)
	sort.Slice(out.ApplicationRuntime, func(i, j int) bool {
		if out.ApplicationRuntime[i].Application == out.ApplicationRuntime[j].Application {
			return out.ApplicationRuntime[i].Version < out.ApplicationRuntime[j].Version
		}
		return out.ApplicationRuntime[i].Application < out.ApplicationRuntime[j].Application
	})
	out.Databases = append([]dbrelease.StateSnapshot(nil), s.Databases...)
	sort.Slice(out.Databases, func(i, j int) bool { return out.Databases[i].Database < out.Databases[j].Database })
	return out
}

func (s Snapshot) DatabaseState(id core.CanonicalID) (dbrelease.StateSnapshot, bool) {
	for _, db := range s.Databases {
		if db.Database == id {
			return db, true
		}
	}
	return dbrelease.StateSnapshot{}, false
}

func (s Snapshot) IsActive(app core.CanonicalID, version string) bool {
	for _, a := range s.ActiveApplications {
		if a.Application == app && a.Version == version {
			return true
		}
	}
	return false
}
