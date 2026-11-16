package externalanalysis

import (
	"tei/internal/sourceindex"
)

// receiptSchemaVersion identifies the shape of one Receipt. It versions the
// receipt, not the Evidence: the producer's own identity travels inside the
// receipt, because a receipt that does not say which producer and version
// wrote the bytes cannot be re-read later (#3394).
const receiptSchemaVersion = "tei.externalanalysis.receipt/v0"

// State classifies what TEI actually holds about one external analysis.
type State string

const (
	StateComplete   State = "complete"
	StatePartial    State = "partial"
	StateFailed     State = "failed"
	StateMismatch   State = "mismatch"
	StateMissing    State = "missing"
	StateToolAbsent State = "tool-absent"
)

func severity(state State) int {
	switch state {
	case StateComplete:
		return 0
	case StateToolAbsent:
		return 1
	case StatePartial:
		return 2
	case StateFailed:
		return 3
	case StateMismatch:
		return 4
	case StateMissing:
		return 5
	}
	return 6
}

// Receipt is what TEI keeps about one received external analysis.
type Receipt struct {
	SchemaVersion string    `json:"schemaVersion"`
	State         State     `json:"state"`
	Reason        string    `json:"reason,omitempty"`
	Evidence      *Evidence `json:"evidence,omitempty"`
	Source        *Source   `json:"source"`
	Raw           Raw       `json:"raw"`
	Envelope      *Envelope `json:"envelope,omitempty"`
	Derived       *Derived  `json:"derived,omitempty"`
	View          *View     `json:"view,omitempty"`
	Failure       *Failure  `json:"failure,omitempty"`
	Tool          Tool      `json:"tool"`
	AcquiredAt    string    `json:"acquiredAt"`
	Promotion     Promotion `json:"promotion"`
	body          []byte
}

// Evidence is the producer's identity for one analysis result.
type Evidence struct {
	ID         string   `json:"id"`
	Operation  string   `json:"operation"`
	Predicate  string   `json:"predicate,omitempty"`
	Confidence string   `json:"confidence,omitempty"`
	Authority  string   `json:"authority,omitempty"`
	Provider   Provider `json:"provider"`
	Envelope   bool     `json:"envelope"`
	Links      []string `json:"links,omitempty"`
}

// Provider is the analyzer that produced the Evidence.
type Provider struct {
	ID      string `json:"id"`
	Name    string `json:"name,omitempty"`
	Version string `json:"version,omitempty"`
}

// Source is the analysis target.
type Source struct {
	Subject   string     `json:"subject"`
	Path      string     `json:"path"`
	Format    string     `json:"format"`
	Revision  string     `json:"revision"`
	Positions []Position `json:"positions"`
}

// Position is one declared location inside the analyzed target.
type Position struct {
	Kind  string `json:"kind"`
	Path  string `json:"path"`
	Range *Range `json:"range"`
}

// Range is a byte span inside the target.
type Range = sourceindex.Range

// Raw measures the received bytes before anything is interpreted.
type Raw struct {
	Bytes       int    `json:"bytes"`
	Encoding    string `json:"encoding"`
	LineEnding  string `json:"lineEnding,omitempty"`
	Fingerprint string `json:"fingerprint"`
}

// Envelope is the producer's own record of the run.
type Envelope struct {
	Command  string `json:"command,omitempty"`
	Duration string `json:"duration,omitempty"`
}

// Derived is the request TEI hands back to the tool for one Evidence.
type Derived struct {
	SourceKind    string   `json:"sourceKind"`
	Bytes         int      `json:"bytes"`
	Fingerprint   string   `json:"fingerprint"`
	AcceptedKinds []string `json:"acceptedKinds,omitempty"`
	ByReference   string   `json:"byReference,omitempty"`
}

// View classifies one saved view delivery.
type View struct {
	Encoding         string    `json:"encoding"`
	Framing          string    `json:"framing"`
	Records          int       `json:"records"`
	Control          string    `json:"control,omitempty"`
	Emitted          int       `json:"emitted"`
	SourceCount      int       `json:"sourceCount,omitempty"`
	Collection       []string  `json:"collection"`
	EvidenceID       string    `json:"evidenceId"`
	ParentEvidenceID string    `json:"parentEvidenceId,omitempty"`
	ParentLinked     bool      `json:"parentLinked"`
	Coverage         *Coverage `json:"coverage,omitempty"`
	Limitations      []string  `json:"limitations,omitempty"`
	Unknowns         []string  `json:"unknowns,omitempty"`
	Failure          *Failure  `json:"failure,omitempty"`
	Duplicates       []string  `json:"duplicates,omitempty"`
}

// Coverage is the producer's own statement about how much of the selected
// collection the delivery covers.
type Coverage struct {
	Status     string `json:"status,omitempty"`
	Examined   int    `json:"examined,omitempty"`
	Total      *int   `json:"total,omitempty"`
	NextOffset *int   `json:"nextOffset,omitempty"`
	Exhausted  *bool  `json:"exhausted,omitempty"`
}

// Failure is the producer's own failure record.
type Failure struct {
	Code     string `json:"code"`
	Category string `json:"category"`
	Message  string `json:"message"`
}

// Tool records whether the external tool is present in this environment.
type Tool struct {
	Name         string `json:"name"`
	Version      string `json:"version"`
	Present      bool   `json:"present"`
	ResolvedPath string `json:"resolvedPath,omitempty"`
	AbsentReason string `json:"absentReason,omitempty"`
}

// Promotion is the boundary this package refuses to cross.
type Promotion struct {
	AcceptedMeaning     bool `json:"acceptedMeaning"`
	CanonicalWorld      bool `json:"canonicalWorld"`
	DatabaseObservation bool `json:"databaseObservation"`
}

// merge keeps the worse of two states, so combining an evidence receipt with a
// view receipt cannot launder a partial delivery into a complete one. The
// reason travels with the state that won.
func merge(current State, currentReason string, incoming State, incomingReason string) (State, string) {
	if severity(incoming) > severity(current) {
		return incoming, incomingReason
	}
	return current, currentReason
}

// fingerprint is the shared sourceindex digest, so "sha256:..." means the same
// thing in a receipt as it does in a Source Index.
func fingerprint(raw []byte) string {
	return sourceindex.Fingerprint(raw)
}

// EvidenceBody returns the producer's exact bytes of the Evidence record.
func (r Receipt) EvidenceBody() []byte {
	return r.body
}
