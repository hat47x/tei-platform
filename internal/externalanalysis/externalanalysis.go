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
//
// The states are ordered by how much is missing, and the order is the merge
// rule: when one receipt combines an evidence receipt with a view receipt, the
// worse state wins. A caller must not read "complete" as "the analysis was
// right"; it reads as "these bytes arrived and match the framing the recorded
// producer version produces".
type State string

const (
	// StateComplete: the bytes arrived and match the declared contract.
	StateComplete State = "complete"
	// StatePartial: bytes arrived, but the delivery is not the whole thing —
	// an interrupted stream, a missing terminal record, a page that is not
	// exhausted, or a framing expectation nobody declared.
	StatePartial State = "partial"
	// StateFailed: the external tool reported a failure of its own.
	StateFailed State = "failed"
	// StateMismatch: the bytes have a shape the contract does not declare —
	// a response envelope standing in for the Evidence body, a source.kind the
	// recorded version does not accept, an encoding we refuse to guess.
	StateMismatch State = "mismatch"
	// StateMissing: the Evidence content was never obtained — empty bytes, or
	// a record whose required identity and content fields are absent.
	StateMissing State = "missing"
	// StateToolAbsent: the external tool is not available in this
	// environment, so no fresh Evidence could be produced here at all.
	StateToolAbsent State = "tool-absent"
)

// severity orders the states for the merge rule documented on State.
//
// Tool absence sits just above complete, and that placement is the point: it is
// a caveat on an otherwise whole receipt — these bytes are fine, this
// environment just could not have produced them — so any problem with the
// delivery itself outranks it. A receipt whose view is interrupted says
// "partial" even when the tool is also absent, because hiding the interrupted
// delivery behind a missing tool would be the more flattering lie.
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
//
// A receipt is an observation about bytes, not a statement about the analyzed
// system. Every field it carries is either copied from the producer's own
// record or measured from the received bytes, so a later reader can tell the
// two apart: nothing here is inferred from the analysis result.
type Receipt struct {
	SchemaVersion string `json:"schemaVersion"`
	State         State  `json:"state"`
	// Reason says, in one sentence, why the state is not complete. It names
	// what was observed, so a reader does not have to guess which of the
	// contract's rules fired.
	Reason string `json:"reason,omitempty"`

	// Evidence is the producer's own record of what it analyzed: its
	// identity, the target, the provider and the authority it claims. It is
	// absent when the bytes never carried one.
	Evidence *Evidence `json:"evidence,omitempty"`
	// Source is where the analyzed bytes were and which exact revision the
	// Evidence describes.
	Source *Source `json:"source,omitempty"`
	// Raw measures the bytes as they arrived, before any decoding.
	Raw Raw `json:"raw"`
	// Envelope is the producer's own execution record, present only when the
	// saved bytes carried the CLI response wrapper.
	Envelope *Envelope `json:"envelope,omitempty"`
	// Derived is the request TEI would hand back to the tool, with its own
	// digest. It never shares the digest with Raw: a derived request that
	// carried the Evidence's fingerprint would mean the Evidence bytes were
	// passed on unchanged, which is not what a request is.
	Derived *Derived `json:"derived,omitempty"`
	// View classifies one saved view delivery against the framing the
	// recorded producer version produces.
	View *View `json:"view,omitempty"`
	// Failure is the producer's own failure record, when it reported one.
	Failure *Failure `json:"failure,omitempty"`
	// Tool records whether the external tool could even run here. It is an
	// observation about this environment, not about the Evidence.
	Tool Tool `json:"tool"`

	// AcquiredAt is when TEI received the bytes. It is not when the analysis
	// ran; that time belongs to the producer and travels inside the Evidence
	// and the envelope.
	AcquiredAt string `json:"acquiredAt"`

	// Promotion records what this receipt is not. Every field is false and
	// stays false: receiving an analysis is not accepting its meaning, not
	// admitting it to the Canonical World, and not observing a database.
	Promotion Promotion `json:"promotion"`

	// body is the producer's exact bytes of the Evidence record. It stays
	// unexported so a receipt serialized for a reader never carries the whole
	// Evidence inline, while DeriveViewRequest can still embed the bytes as
	// received instead of a re-serialized copy.
	body []byte
}

// Evidence is the producer's identity for one analysis result.
type Evidence struct {
	// ID is the producer's Evidence identifier. It identifies the producer's
	// record, never a TEI subject: the two namespaces are not equated here.
	ID         string   `json:"id"`
	Operation  string   `json:"operation"`
	Predicate  string   `json:"predicate,omitempty"`
	Confidence string   `json:"confidence,omitempty"`
	Authority  string   `json:"authority,omitempty"`
	Provider   Provider `json:"provider"`
	// Envelope reports that the saved bytes carried the CLI response wrapper
	// around this record.
	Envelope bool     `json:"envelope"`
	Links    []string `json:"links,omitempty"`
}

// Provider is the analyzer that produced the Evidence, as the producer named
// it. The provider's own version is not the version of the CLI that framed the
// bytes; the latter travels in Tool.
type Provider struct {
	ID      string `json:"id"`
	Name    string `json:"name,omitempty"`
	Version string `json:"version,omitempty"`
}

// Source is the analysis target: which bytes were read, and which revision of
// them the Evidence describes.
type Source struct {
	// Subject is the target as the producer named it.
	Subject string `json:"subject,omitempty"`
	// Path is where the target was, as the producer recorded it. It is the
	// producer's path, not a TEI workspace path.
	Path string `json:"path,omitempty"`
	// Format is the target's form as the producer classified it.
	Format string `json:"format,omitempty"`
	// Revision is the digest of the exact target bytes this Evidence
	// describes. A later receipt for the same Evidence ID with a different
	// revision is a different observation, not the same one re-read.
	Revision string `json:"revision,omitempty"`
	// Positions are the locations the Evidence itself declares. A position
	// without a Range is a named place, not a byte span.
	Positions []Position `json:"positions,omitempty"`
}

// Position is one declared location inside the analyzed target.
type Position struct {
	Kind  string `json:"kind"`
	Path  string `json:"path,omitempty"`
	Range *Range `json:"range,omitempty"`
}

// Range is a byte span inside the target. It is the sourceindex range, reused
// rather than redefined, so a span means the same thing here as everywhere else
// in TEI.
type Range = sourceindex.Range

// Raw measures the received bytes before anything is interpreted.
type Raw struct {
	Bytes       int    `json:"bytes"`
	Encoding    string `json:"encoding"`
	LineEnding  string `json:"lineEnding,omitempty"`
	Fingerprint string `json:"fingerprint"`
}

// Envelope is the producer's own record of the run, taken from the CLI
// response wrapper. It is the closest thing to an execution record the saved
// bytes carry, and it is copied, not trusted as proof the command succeeded.
type Envelope struct {
	Command  string `json:"command,omitempty"`
	Duration string `json:"duration,omitempty"`
}

// Derived is the request TEI hands back to the tool for one Evidence.
type Derived struct {
	// SourceKind is the source.kind the request uses.
	SourceKind string `json:"sourceKind"`
	// Bytes and Fingerprint measure the derived request, never the Evidence.
	Bytes       int    `json:"bytes"`
	Fingerprint string `json:"fingerprint"`
	// AcceptedKinds are the source.kind values the recorded tool version
	// accepts, copied from the declaration the caller supplied.
	AcceptedKinds []string `json:"acceptedKinds,omitempty"`
	// ByReference names the file a request points at when its source.kind
	// carries a path instead of the Evidence. The referenced bytes are
	// outside this receipt's fingerprint, which is a limitation of the
	// receipt, not a property of the Evidence.
	ByReference string `json:"byReference,omitempty"`
}

// View classifies one saved view delivery.
type View struct {
	Encoding string `json:"encoding"`
	// Framing is the shape the delivery actually has, named the way the
	// producer's protocol names it.
	Framing string `json:"framing"`
	// Records counts the delivered data records, excluding control records.
	Records int `json:"records"`
	// Control names the control record the delivery ended with, if any.
	Control string `json:"control,omitempty"`
	// Emitted and SourceCount are the producer's own counts from the
	// terminal record. They count the delivery, not the analysis.
	Emitted     int `json:"emitted,omitempty"`
	SourceCount int `json:"sourceCount,omitempty"`
	// Collection is the path the producer says the records came from.
	Collection []string `json:"collection,omitempty"`
	// EvidenceID is the view delivery's own Evidence identifier. It is not
	// the parent's: a view is its own record about the parent.
	EvidenceID string `json:"evidenceId,omitempty"`
	// ParentEvidenceID is the producer's declared parent, and ParentLinked
	// says whether it was checked against the received Evidence and matched.
	ParentEvidenceID string `json:"parentEvidenceId,omitempty"`
	ParentLinked     bool   `json:"parentLinked"`
	// Coverage, Limitations and Unknowns are copied from the delivery. They
	// describe what the producer saw, not what exists.
	Coverage    *Coverage `json:"coverage,omitempty"`
	Limitations []string  `json:"limitations,omitempty"`
	Unknowns    []string  `json:"unknowns,omitempty"`
	// Failure is the producer's own failure record, when the delivery turned
	// out to be a failure delivery.
	Failure *Failure `json:"failure,omitempty"`
	// Duplicates names identities the delivery carries more than once. A set
	// of observations that names one identity twice is not a set.
	Duplicates []string `json:"duplicates,omitempty"`
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
	Code     string `json:"code,omitempty"`
	Category string `json:"category,omitempty"`
	Message  string `json:"message,omitempty"`
}

// Tool records whether the external tool is present in this environment.
type Tool struct {
	Name string `json:"name"`
	// Version is the caller's declaration of which tool version produced the
	// bytes. It is a declaration, not something the bytes prove.
	Version      string `json:"version,omitempty"`
	Present      bool   `json:"present"`
	ResolvedPath string `json:"resolvedPath,omitempty"`
	AbsentReason string `json:"absentReason,omitempty"`
}

// Promotion is the boundary this package refuses to cross. Every field is
// false by construction; see the package documentation.
type Promotion struct {
	AcceptedMeaning     bool `json:"acceptedMeaning"`
	CanonicalWorld      bool `json:"canonicalWorld"`
	DatabaseObservation bool `json:"databaseObservation"`
}

// merge keeps the worse of two states, so combining an Evidence receipt with a
// view receipt cannot launder a partial delivery into a complete one. The
// reason travels with the state that won.
func merge(current State, currentReason string, incoming State, incomingReason string) (State, string) {
	if severity(incoming) > severity(current) {
		return incoming, incomingReason
	}
	return current, currentReason
}

// fingerprint is the shared sourceindex digest, so "sha256:…" means the same
// thing in a receipt as it does in a Source Index.
func fingerprint(raw []byte) string {
	return sourceindex.Fingerprint(raw)
}

// EvidenceBody returns the producer's exact bytes of the Evidence record.
//
// It exists so a request can embed the Evidence as it was received instead of
// a re-serialized copy: key order belongs to the producer, and a request whose
// bytes cannot be traced back to the received ones cannot be re-read later. The
// body is not part of the receipt's serialized form; a receipt written for a
// reader carries the measurements, not the whole Evidence.
func (r Receipt) EvidenceBody() []byte {
	return r.body
}
