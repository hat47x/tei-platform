package okf

import (
	"sort"
	"strings"
)

// CandidateKind names what a projection proposes. Every kind is a candidate: the
// projection never asserts an accepted TEI meaning, an owner decision, or a host
// grant.
type CandidateKind string

const (
	// CandidateSubject proposes a TEI semantic subject for a concept. The
	// proposed ID is derived from the concept ID and is explicitly not a TEI
	// Meaning ID.
	CandidateSubject CandidateKind = "tei-semantic-subject-candidate"
	// CandidateSource proposes a source reference for one bundle entry. It
	// records the declared source, not a resolved retrieval.
	CandidateSourceKind CandidateKind = "tei-source-reference-candidate"
	// CandidateRelation proposes a relation between two concepts, taken from a
	// body link. OKF does not type links, so the relation kind stays unspecified.
	CandidateRelation CandidateKind = "tei-relation-candidate"
	// CandidateComputation proposes a computation contract reference for an
	// Attested Computation declaration. Nothing is executed or verified.
	CandidateComputation CandidateKind = "tei-computation-contract-candidate"
)

// Candidate is one proposed TEI structure. It carries the exact bundle evidence it
// came from so a reviewer can re-derive it without trusting this output.
type Candidate struct {
	Kind CandidateKind `json:"candidateKind"`
	// ProposedID is a stable, human-readable proposal. It is not a TEI Meaning ID,
	// a SHO Document ID, or an accepted identifier.
	ProposedID string `json:"proposedId"`
	// Source identifies the bytes the candidate was derived from.
	Source CandidateEvidence `json:"source"`
	// Basis states, in one sentence, what the document said that produced the
	// candidate.
	Basis string `json:"basis"`
	// Admission is always "candidate". A reader that finds "accepted" here is
	// reading a bug.
	Admission string `json:"admission"`
	// RequiresReview lists the qualifications this candidate lacks. It is never
	// empty: a format read cannot grant any of them.
	RequiresReview []string `json:"requiresReview"`
}

// CandidateSource is the evidence pointer for one candidate.
type CandidateEvidence struct {
	ConceptID string `json:"conceptId"`
	Path      string `json:"path"`
	Digest    string `json:"artifactDigest"`
	// Field is the frontmatter key or body construct the candidate came from.
	Field string `json:"field"`
	// Line is the 1-based line when the reader can locate it.
	Line int `json:"line,omitempty"`
	// Target is the raw value or link destination, when the candidate names one.
	Target string `json:"target,omitempty"`
}

// Loss is one piece of the source that the projection knowingly does not carry.
// OKF and TEI are not isomorphic: a loss manifest is what lets a consumer tell
// "this projection is incomplete" from "the source said nothing".
type Loss struct {
	ConceptID string `json:"conceptId,omitempty"`
	// Information names what is not carried.
	Information string `json:"information"`
	// Reason names why the projection cannot carry it.
	Reason string `json:"reason"`
	// ReturnTo names where the information still lives, so a reviewer can go back.
	ReturnTo string `json:"returnTo"`
}

// Projection is the read-only result of deriving TEI structure candidates from an
// OKF bundle. It never writes to the bundle and never claims an acceptance.
type Projection struct {
	Root       string      `json:"root"`
	Revision   Revision    `json:"bundleRevision"`
	OkfVersion string      `json:"okfVersion"`
	Profile    string      `json:"profile"`
	Candidates []Candidate `json:"candidates"`
	Losses     []Loss      `json:"losses"`
	// NotProposed names the TEI structures the projection deliberately does not
	// propose, so a consumer does not read silence as absence.
	NotProposed []string `json:"notProposed"`
	Admission   string   `json:"admission"`
	FormatNote  string   `json:"formatNote"`
}

// projectionProfileReadOnly is the only profile this reader implements. A second
// profile would need its own evidence; naming one here that does not exist would
// be a false promise.
const projectionProfileReadOnly = "read-only-candidates"

// notProposedItems lists the TEI structures a projection does not create.
var notProposedItems = []string{
	"TEI Accepted Meaning",
	"TEI Owner Decision",
	"TEI Role or Trait adoption",
	"TEI Operator or Language selection",
	"TEI Required Source closure",
	"TEI Host Grant or runtime qualification",
	"TEI Validation result (F2)",
	"SHO Document or Document Revision",
	"SUI Card, Island, Tree, or Graph occurrence",
	"source retrieval or content verification",
	"computation execution, receipt, or attester verdict",
	"document write, relocation, or repair",
}

// Project derives candidates from an already-read bundle. Passing the bundle rather
// than the root keeps the projection a pure function of the bytes the caller read.
func Project(bundle *Bundle) Projection {
	projection := Projection{
		Root:        bundle.Root,
		Revision:    bundle.Revision,
		OkfVersion:  bundle.OkfVersion,
		Profile:     projectionProfileReadOnly,
		Candidates:  []Candidate{},
		Losses:      []Loss{},
		NotProposed: append([]string{}, notProposedItems...),
		Admission:   "candidate",
		FormatNote: "Every candidate is a proposal derived from the document's own text. " +
			"Format conformance (F0) is not acceptance, and this projection is not a TEI meaning.",
	}

	conceptByPath := map[string]*Concept{}
	for i := range bundle.Concepts {
		conceptByPath[bundle.Concepts[i].Path] = &bundle.Concepts[i]
	}

	for i := range bundle.Concepts {
		concept := &bundle.Concepts[i]
		projection.Candidates = append(projection.Candidates, subjectCandidate(concept))
		projection.Candidates = append(projection.Candidates, sourceCandidates(concept)...)
		if concept.Computation != nil || concept.Executor != nil || concept.Attester != nil || concept.Runtime != "" {
			projection.Candidates = append(projection.Candidates, computationCandidate(concept))
		}
		projection.Candidates = append(projection.Candidates, relationCandidates(concept, conceptByPath)...)
		projection.Losses = append(projection.Losses, conceptLosses(concept)...)
	}

	sort.SliceStable(projection.Candidates, func(i, j int) bool {
		if projection.Candidates[i].Source.ConceptID != projection.Candidates[j].Source.ConceptID {
			return projection.Candidates[i].Source.ConceptID < projection.Candidates[j].Source.ConceptID
		}
		if projection.Candidates[i].Kind != projection.Candidates[j].Kind {
			return projection.Candidates[i].Kind < projection.Candidates[j].Kind
		}
		return projection.Candidates[i].ProposedID < projection.Candidates[j].ProposedID
	})
	sort.SliceStable(projection.Losses, func(i, j int) bool {
		if projection.Losses[i].ConceptID != projection.Losses[j].ConceptID {
			return projection.Losses[i].ConceptID < projection.Losses[j].ConceptID
		}
		return projection.Losses[i].Information < projection.Losses[j].Information
	})

	// A bundle-wide loss list: the things no candidate can carry.
	projection.Losses = append(projection.Losses, bundleLosses(bundle)...)
	sort.SliceStable(projection.Losses, func(i, j int) bool {
		if projection.Losses[i].ConceptID != projection.Losses[j].ConceptID {
			return projection.Losses[i].ConceptID < projection.Losses[j].ConceptID
		}
		return projection.Losses[i].Information < projection.Losses[j].Information
	})
	return projection
}

// subjectCandidate proposes a subject for one concept. The proposed ID is derived
// from the concept ID, with a marker that makes it visibly a proposal.
func subjectCandidate(concept *Concept) Candidate {
	base := strings.TrimSuffix(concept.ConceptID, ".md")
	proposed := base
	if strings.Contains(proposed, "/") {
		proposed = strings.ReplaceAll(proposed, "/", ".")
	}
	return Candidate{
		Kind:       CandidateSubject,
		ProposedID: "candidate:" + proposed,
		Source: CandidateEvidence{
			ConceptID: concept.ConceptID,
			Path:      concept.Path,
			Digest:    concept.Digest,
			Field:     "type",
		},
		Basis:     "the concept declares type " + quoteKey(concept.Type) + " with title " + quoteKey(concept.Title),
		Admission: "candidate",
		RequiresReview: []string{
			"TEI Role or Trait selection for this kind of subject",
			"TEI semantic namespace and subject ID assignment",
			"TEI Value Domain binding, when the concept carries data fields",
			"owner acceptance",
		},
	}
}

// sourceCandidates proposes one source reference per declared source entry.
func sourceCandidates(concept *Concept) []Candidate {
	var out []Candidate
	for index, source := range concept.Sources {
		proposed := concept.ConceptID + "#source-" + itoa(index+1)
		if source.ID != "" {
			proposed = concept.ConceptID + "#" + source.ID
		}
		out = append(out, Candidate{
			Kind:       CandidateSourceKind,
			ProposedID: "candidate:" + proposed,
			Source: CandidateEvidence{
				ConceptID: concept.ConceptID,
				Path:      concept.Path,
				Digest:    concept.Digest,
				Field:     "sources[].resource",
				Target:    source.Resource,
				Line:      sourceLine(concept, source.ID),
			},
			Basis:     "the concept declares a source entry with resource " + quoteKey(source.Resource) + " (" + string(source.Resolution) + ")",
			Admission: "candidate",
			RequiresReview: []string{
				"actual retrieval or byte comparison of the resource",
				"source identity and revision binding (MMI or the owning reader)",
				"source acceptance for any consumer purpose",
			},
		})
	}
	return out
}

// sourceLine returns the line of the sources entry when the reader recorded one.
// The reader keeps one line per document field today, so this reports that line.
func sourceLine(concept *Concept, _ string) int {
	_ = concept
	return 0
}

// computationCandidate proposes the declaration as a contract reference. It
// explicitly does not propose an execution.
func computationCandidate(concept *Concept) Candidate {
	proposed := concept.ConceptID + "#computation-contract"
	return Candidate{
		Kind:       CandidateComputation,
		ProposedID: "candidate:" + proposed,
		Source: CandidateEvidence{
			ConceptID: concept.ConceptID,
			Path:      concept.Path,
			Digest:    concept.Digest,
			Field:     "runtime",
		},
		Basis:     "the concept declares an Attested Computation contract with runtime " + quoteKey(concept.Runtime),
		Admission: "candidate",
		RequiresReview: []string{
			"a selected executor and a real run",
			"a receipt shaped by executor.receipt",
			"a deterministic attester verdict over that receipt",
			"an independent re-read of the authoritative result",
		},
	}
}

// relationCandidates proposes one relation per resolvable body link that points at
// another concept. OKF does not type links, so the relation kind is left to review.
func relationCandidates(concept *Concept, concepts map[string]*Concept) []Candidate {
	var out []Candidate
	for _, link := range concept.Links {
		ref := resolveTarget(concept.Path, link.Target, "body-link", link.Line, func(candidate string) bool {
			_, ok := concepts[candidate]
			return ok
		})
		if ref.Resolution != ResolutionDocumentRelative && ref.Resolution != ResolutionBundleRoot {
			continue
		}
		if len(ref.Resolved) == 0 {
			continue
		}
		target := concepts[ref.Resolved[0]]
		if target == nil {
			continue
		}
		proposed := concept.ConceptID + "->" + target.ConceptID
		out = append(out, Candidate{
			Kind:       CandidateRelation,
			ProposedID: "candidate:" + proposed,
			Source: CandidateEvidence{
				ConceptID: concept.ConceptID,
				Path:      concept.Path,
				Digest:    concept.Digest,
				Field:     "body-link",
				Line:      link.Line,
				Target:    link.Target,
			},
			Basis:     "the body links " + quoteKey(link.Text) + " to " + quoteKey(target.ConceptID),
			Admission: "candidate",
			RequiresReview: []string{
				"the relation kind (parent, references, depends-on, ...) from the surrounding prose",
				"the relation cardinality and required inputs",
				"TEI relation identity and endpoint subjects",
			},
		})
	}
	return out
}

// conceptLosses lists what one concept's projection cannot carry.
func conceptLosses(concept *Concept) []Loss {
	var losses []Loss
	basis := "the concept's own bytes remain the source"
	losses = append(losses, Loss{
		ConceptID:   concept.ConceptID,
		Information: "the body prose, tables, and fenced code",
		Reason:      "the projection proposes subjects, sources, relations and contracts; it does not translate prose into TEI statements",
		ReturnTo:    concept.Path + " (" + concept.Digest + ")",
	})
	if len(concept.Tags) > 0 {
		losses = append(losses, Loss{
			ConceptID:   concept.ConceptID,
			Information: "tags " + strings.Join(concept.Tags, ", "),
			Reason:      "OKF tags are cross-cutting labels, not TEI roles; no mapping is asserted",
			ReturnTo:    basis,
		})
	}
	if len(concept.UnknownKeys) > 0 {
		losses = append(losses, Loss{
			ConceptID:   concept.ConceptID,
			Information: "unknown frontmatter keys " + strings.Join(concept.UnknownKeys, ", "),
			Reason:      "a producer extension has no TEI interpretation; carrying it into a TEI structure would invent one",
			ReturnTo:    basis,
		})
	}
	if concept.TrustTierLabel != "unverified" {
		losses = append(losses, Loss{
			ConceptID:   concept.ConceptID,
			Information: "OKF trust tier " + concept.TrustTierLabel,
			Reason:      "the tier is an advisory document label derived from `verified`; it is not a review record, an identity check, or an acceptance",
			ReturnTo:    basis,
		})
	}
	if concept.Stale != "absent" {
		losses = append(losses, Loss{
			ConceptID:   concept.ConceptID,
			Information: "staleness " + concept.Stale + " from stale_after " + concept.StaleAfter,
			Reason:      "staleness is a comparison against the clock for this read; it is not a re-qualification of any earlier decision",
			ReturnTo:    basis,
		})
	}
	if concept.Status != "stable" {
		losses = append(losses, Loss{
			ConceptID:   concept.ConceptID,
			Information: "lifecycle status " + concept.Status,
			Reason:      "OKF status is the exchange document's lifecycle; it is not a SHO delivery state or a TEI acceptance state",
			ReturnTo:    basis,
		})
	}
	if concept.Generated != nil {
		losses = append(losses, Loss{
			ConceptID:   concept.ConceptID,
			Information: "generation record by " + concept.Generated.By,
			Reason:      "who wrote the document is not who verified it, and neither is an authenticated actor",
			ReturnTo:    basis,
		})
	}
	if len(concept.Verified) > 0 {
		losses = append(losses, Loss{
			ConceptID:   concept.ConceptID,
			Information: "verification events (" + itoa(len(concept.Verified)) + ")",
			Reason:      "a `human:` label is not an authenticated identity, and a verification event is not an owner decision",
			ReturnTo:    basis,
		})
	}
	return losses
}

// bundleLosses lists what no concept candidate can carry.
func bundleLosses(bundle *Bundle) []Loss {
	losses := []Loss{{
		Information: "bundle-level index and log structure",
		Reason:      "index.md and log.md describe the exchange bundle; they are not TEI declarations and are not projected as structures",
		ReturnTo:    "the bundle root (" + bundle.Revision.Value + ")",
	}}
	if len(bundle.UnresolvedReferences) > 0 {
		losses = append(losses, Loss{
			Information: "unresolved references (" + itoa(len(bundle.UnresolvedReferences)) + ")",
			Reason:      "a reference with no target is reported as unresolved; the projection does not create the missing structure",
			ReturnTo:    "the bundle root (" + bundle.Revision.Value + ")",
		})
	}
	if len(bundle.Logs) > 0 {
		losses = append(losses, Loss{
			Information: "log.md chronology",
			Reason:      "OKF log entries are prose; the projection does not turn them into TEI change history or review history",
			ReturnTo:    "the bundle root (" + bundle.Revision.Value + ")",
		})
	}
	return losses
}
