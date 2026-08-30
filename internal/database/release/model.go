package release

import (
	"sort"

	"tei/internal/core"
	"tei/internal/database"
)

type DataReadiness string

const (
	DataReady     DataReadiness = "ready"
	DataPreparing DataReadiness = "preparing"
	DataNotReady  DataReadiness = "not-ready"
	DataUnknown   DataReadiness = "unknown"
)

func (r DataReadiness) Valid() bool {
	return r == DataReady || r == DataPreparing || r == DataNotReady || r == DataUnknown
}

type ProvidedContract struct {
	ID      core.CanonicalID         `json:"id"`
	Version database.ContractVersion `json:"version"`
}

// Definition is a delivery/version boundary for a Database Unit. It is not an
// Application release and does not imply that any consumer must deploy at the
// same time.
type Definition struct {
	Database          core.CanonicalID   `json:"database"`
	Version           string             `json:"version"`
	SchemaFingerprint string             `json:"schemaFingerprint,omitempty"`
	Migrations        []string           `json:"migrations,omitempty"`
	Provides          []ProvidedContract `json:"provides,omitempty"`
	DataReadiness     DataReadiness      `json:"dataReadiness"`
}

func (r Definition) Normalize() Definition {
	out := r
	out.Migrations = append([]string(nil), r.Migrations...)
	sort.Strings(out.Migrations)
	out.Provides = append([]ProvidedContract(nil), r.Provides...)
	sort.Slice(out.Provides, func(i, j int) bool { return out.Provides[i].ID < out.Provides[j].ID })
	return out
}

type StateSource string

const (
	StateFromRelease  StateSource = "release"
	StateFromObserved StateSource = "observed"
)

// StateSnapshot is the compatibility-facing view of a database. Managed
// databases usually derive it from a Database Release; external-observed
// databases can provide the same contract surface without TEI owning their
// migration lifecycle.
type StateSnapshot struct {
	Database      core.CanonicalID   `json:"database"`
	Source        StateSource        `json:"source"`
	Release       string             `json:"release,omitempty"`
	Provides      []ProvidedContract `json:"provides,omitempty"`
	DataReadiness DataReadiness      `json:"dataReadiness"`
	ObservedAt    string             `json:"observedAt,omitempty"`
}

func StateFromDefinition(r Definition) StateSnapshot {
	return StateSnapshot{Database: r.Database, Source: StateFromRelease, Release: r.Version, Provides: append([]ProvidedContract(nil), r.Provides...), DataReadiness: r.DataReadiness}
}

func ValidVersion(v string) bool { return core.ValidNumericVersion(v) }
