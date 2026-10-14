// Package hostadmission checks the exact E1 Application unit selection against
// an independently obtained, CURRENT Host-owned acceptance snapshot.
//
// It neither creates nor signs Source/Builder/Deployment approval, and is not
// an alternative BuildArtifactLock. An unauthenticated or caller-owned Source
// makes this comparison non-authoritative.
package hostadmission

import (
	"context"
	"fmt"
	"regexp"
	"strings"
)

const (
	BrowserUnit = "browser"
	APIUnit     = "api"
)

var (
	gitRevision = regexp.MustCompile(`^[0-9a-f]{40}$`)
	unitDigest  = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)
)

// Unit is one exact physical build, distinct from the accepted Application.
// Browser and API may have independent Target build revisions T.
type Unit struct {
	ID                  string
	TargetBuildRevision string
	ArtifactDigest      string
	ArtifactBytes       int64
}

// Selection is assembled from actual observed artifact bytes and the accepted
// Application B/R. It is a *query*, not a Host-issued approval.
type Selection struct {
	ApplicationID              string
	HandoverBaselineID         string
	BaselineRepositoryRevision string
	Units                      []Unit
}

// MeasuredE1Pair is a read-only selection extracted from the EXISTING E1
// two-unit measurement after byte/build preflight. Its plain hashes are not
// Host acceptance, trusted Builder proof or a newly minted Artifact Lock.
// The present E1 builder uses one common source T for both physical units.
type MeasuredE1Pair struct {
	ApplicationID              string
	HandoverBaselineID         string
	BaselineRepositoryRevision string
	BrowserAndAPISourceRevision string
	BrowserSHA256              string
	BrowserBytes               int64
	APISHA256                  string
	APIBytes                   int64
}

// FromMeasuredE1Pair adapts the currently generated two-unit observation to
// an exact Host query. The caller must have independently verified its bytes
// and provenance; this transformation cannot authenticate a measurement.
func FromMeasuredE1Pair(m MeasuredE1Pair) (Selection, error) {
	selected := Selection{
		ApplicationID: m.ApplicationID,
		HandoverBaselineID: m.HandoverBaselineID,
		BaselineRepositoryRevision: m.BaselineRepositoryRevision,
		Units: []Unit{
			{ID: BrowserUnit, TargetBuildRevision: m.BrowserAndAPISourceRevision,
				ArtifactDigest: "sha256:" + m.BrowserSHA256, ArtifactBytes: m.BrowserBytes},
			{ID: APIUnit, TargetBuildRevision: m.BrowserAndAPISourceRevision,
				ArtifactDigest: "sha256:" + m.APISHA256, ArtifactBytes: m.APIBytes},
		},
	}
	if err := validateSelection(selected); err != nil {
		return Selection{}, err
	}
	return selected, nil
}

// BuildAcceptance is one Host-returned CURRENT Builder acceptance. The
// EvidenceRef has no authority unless the Host authenticates its source.
type BuildAcceptance struct {
	UnitID         string
	ArtifactDigest string
	Accepted       bool
	EvidenceRef    string
}

// Snapshot must originate from a Host-controlled source which authenticates
// current approvals, trusted issuers, Build evidence and revocations.
// Returning a JSON receipt or a synthetic snapshot is NOT sufficient.
type Snapshot struct {
	Selection           Selection
	SourceAccepted      bool
	SourceAcceptanceRef string
	Builds              []BuildAcceptance
	DeploymentAccepted  bool
	DeploymentRef       string
	AuthorityRevision   string
}

// Source is the REQUIRED Host-owned port. No nil-source, caller-provided
// snapshot, or cached previous approval fallback is permitted.
type Source interface {
	ReadCurrentHostAcceptance(context.Context, Selection) (Snapshot, error)
}

// Result holds the references checked for this one read. It is not a
// reusable capability token, a signed proof, or a deployment command.
type Result struct {
	AuthorityRevision   string
	SourceAcceptanceRef string
	DeploymentRef       string
	BuildEvidenceRefs   map[string]string
}

func validateSelection(s Selection) error {
	if strings.TrimSpace(s.ApplicationID) == "" ||
		s.ApplicationID != strings.TrimSpace(s.ApplicationID) ||
		strings.TrimSpace(s.HandoverBaselineID) == "" ||
		s.HandoverBaselineID != strings.TrimSpace(s.HandoverBaselineID) ||
		!gitRevision.MatchString(s.BaselineRepositoryRevision) {
		return fmt.Errorf("invalid Application Handover B/R selection")
	}
	if len(s.Units) != 2 {
		return fmt.Errorf("exactly one Browser and one API unit are required")
	}
	seen := map[string]bool{}
	for _, u := range s.Units {
		if u.ID != BrowserUnit && u.ID != APIUnit {
			return fmt.Errorf("unknown physical unit %q", u.ID)
		}
		if seen[u.ID] {
			return fmt.Errorf("duplicate physical unit %q", u.ID)
		}
		seen[u.ID] = true
		if !gitRevision.MatchString(u.TargetBuildRevision) ||
			!unitDigest.MatchString(u.ArtifactDigest) || u.ArtifactBytes <= 0 {
			return fmt.Errorf("invalid exact %s Target build T/artifact identity", u.ID)
		}
	}
	if !seen[BrowserUnit] || !seen[APIUnit] {
		return fmt.Errorf("Browser and API must both be selected")
	}
	return nil
}

func unitsByID(units []Unit) map[string]Unit {
	out := make(map[string]Unit, len(units))
	for _, u := range units {
		out[u.ID] = u
	}
	return out
}

// CheckCurrent reads the Host source on EVERY invocation. It checks independent
// Source, Builder and Deployment acceptance for EXACT observed unit identities
// and rejects absent, revoked, ambiguous, stale or drifted records.
//
// The caller MUST supply an authenticated Host-owned Source. This function
// cannot establish that Source's trust root or verify a remote signature.
func CheckCurrent(ctx context.Context, source Source, selected Selection) (Result, error) {
	if source == nil {
		return Result{}, fmt.Errorf("Host current acceptance source is missing")
	}
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	if err := validateSelection(selected); err != nil {
		return Result{}, err
	}
	// Host implementations may mutate the argument; do not allow such
	// mutations to change the request's own comparison target.
	expected := selected
	expected.Units = append([]Unit(nil), selected.Units...)
	hostQuery := selected
	hostQuery.Units = append([]Unit(nil), selected.Units...)
	snapshot, err := source.ReadCurrentHostAcceptance(ctx, hostQuery)
	if err != nil {
		return Result{}, fmt.Errorf("Host current acceptance unavailable: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	if err := validateSelection(snapshot.Selection); err != nil {
		return Result{}, fmt.Errorf("Host accepted unit selection invalid: %w", err)
	}
	if snapshot.Selection.ApplicationID != expected.ApplicationID ||
		snapshot.Selection.HandoverBaselineID != expected.HandoverBaselineID ||
		snapshot.Selection.BaselineRepositoryRevision != expected.BaselineRepositoryRevision {
		return Result{}, fmt.Errorf("Host accepted Application B/R differs from requested Handover")
	}
	observedUnits := unitsByID(snapshot.Selection.Units)
	for _, unit := range expected.Units {
		if observedUnits[unit.ID] != unit {
			return Result{}, fmt.Errorf("Host accepted %s exact T/digest/bytes differ", unit.ID)
		}
	}
	if !snapshot.SourceAccepted || strings.TrimSpace(snapshot.SourceAcceptanceRef) == "" {
		return Result{}, fmt.Errorf("Host current Source acceptance is missing or revoked")
	}
	if len(snapshot.Builds) != 2 {
		return Result{}, fmt.Errorf("Host must accept both exact unit builds")
	}
	evidence := make(map[string]string, 2)
	for _, build := range snapshot.Builds {
		want, ok := observedUnits[build.UnitID]
		if !ok || evidence[build.UnitID] != "" {
			return Result{}, fmt.Errorf("Host build acceptance has an unknown or duplicate unit")
		}
		if !build.Accepted || build.ArtifactDigest != want.ArtifactDigest ||
			strings.TrimSpace(build.EvidenceRef) == "" {
			return Result{}, fmt.Errorf("Host current %s Builder acceptance is absent, revoked or differs", build.UnitID)
		}
		evidence[build.UnitID] = build.EvidenceRef
	}
	if len(evidence) != 2 || evidence[BrowserUnit] == "" || evidence[APIUnit] == "" {
		return Result{}, fmt.Errorf("Host unit Builder acceptance is incomplete")
	}
	if !snapshot.DeploymentAccepted || strings.TrimSpace(snapshot.DeploymentRef) == "" {
		return Result{}, fmt.Errorf("Host current deployment acceptance is missing or revoked")
	}
	if strings.TrimSpace(snapshot.AuthorityRevision) == "" {
		return Result{}, fmt.Errorf("Host current authority revision is missing")
	}
	return Result{
		AuthorityRevision:   snapshot.AuthorityRevision,
		SourceAcceptanceRef: snapshot.SourceAcceptanceRef,
		DeploymentRef:       snapshot.DeploymentRef,
		BuildEvidenceRefs:   evidence,
	}, nil
}
