package derivationref

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"sort"
	"time"
)

// Freshness is a reference-only cache qualification, not Canonical Authority.
// Pure D1 evaluation does not require any of these optional cache fields.
var (
	ErrCacheEvidence = errors.New("derivation cache evidence is incomplete")
	ErrCacheDenied   = errors.New("derivation cache reuse is not authorized")
	ErrCacheStale    = errors.New("derivation cache is stale")
	ErrNotCoherent   = errors.New("source snapshot alignment is unproven")
	ErrConfirmed    = errors.New("confirmed fact cannot be verified")
)

type Revision struct {
	ID       string `json:"id"`
	Revision string `json:"revision"`
}

type SourceObservation struct {
	SourceID    string    `json:"sourceId"`
	Revision    string    `json:"revision"`
	ObservedAt  time.Time `json:"observedAt"`
	SourceAsOf  time.Time `json:"sourceAsOf"`
	SnapshotRef string    `json:"snapshotRef,omitempty"`
}

// CacheScope values must originate from current Host context; a request's
// self-reported tenant, subject or grant is not authorization evidence.
type CacheScope struct {
	Environment    string `json:"environment"`
	Tenant         string `json:"tenant"`
	Subject        string `json:"subject"`
	PermissionEpoch string `json:"permissionEpoch"`
}

// CacheWitness ties one calculation to exact dependencies, accepted meaning,
// runtime authority context and source time. It carries no credentials.
type CacheWitness struct {
	RuleID             string              `json:"ruleId"`
	RuleRevision       string              `json:"ruleRevision"`
	TargetID           string              `json:"targetId"`
	SemanticRevision   string              `json:"semanticRevision"`
	AcceptanceRevision string              `json:"acceptanceRevision"`
	ProviderRevision   string              `json:"providerRevision"`
	Inputs             []Revision          `json:"inputs"`
	Relations          []Revision          `json:"relations,omitempty"`
	Sources            []SourceObservation `json:"sources"`
	Scope              CacheScope          `json:"scope"`
}

// CachePolicy applies only when a consumer opts into caching. MaxAge is
// the age of the computed result; MaxSourceAge independently limits the age
// of source data as-of the consumer's evaluation time. A zero bound disables
// that specific age check, not the current-revision/authority verification.
type CachePolicy struct {
	MaxAge         time.Duration
	MaxSourceAge   time.Duration
	RequireAligned bool
}

// CurrentWitnessVerifier is supplied by the active Host/Provider. It must
// independently check current source revisions, acceptance and permission
// epoch; a digest or declaration is not proof of these claims.
type CurrentWitnessVerifier interface {
	VerifyCurrent(context.Context, CacheWitness) error
	VerifyAligned(context.Context, CacheWitness) error
}

// QualifyCache checks shape and current evidence on *every* read/write.
// A coherent cross-provider snapshot requires external Provider verification.
// Equal strings or timestamp proximity alone never establish coherence.
func QualifyCache(ctx context.Context, witness CacheWitness, policy CachePolicy, verifier CurrentWitnessVerifier) (string, error) {
	if err := validateWitness(witness); err != nil {
		return "", err
	}
	if verifier == nil {
		return "", ErrCacheDenied
	}
	if policy.MaxAge < 0 || policy.MaxSourceAge < 0 {
		return "", fmt.Errorf("%w: invalid age bound", ErrCacheEvidence)
	}
	if err := verifier.VerifyCurrent(ctx, witness); err != nil {
		return "", fmt.Errorf("%w: %v", ErrCacheDenied, err)
	}
	if policy.RequireAligned && len(witness.Sources) > 1 {
		for _, source := range witness.Sources {
			if source.SnapshotRef == "" {
				return "", ErrNotCoherent
			}
		}
		if err := verifier.VerifyAligned(ctx, witness); err != nil {
			return "", fmt.Errorf("%w: %v", ErrNotCoherent, err)
		}
	}
	// JSON encodes map-free, canonical-ordered fields deterministically.
	// Neither current evaluation time nor consumer TTL is a data identity.
	payload, err := json.Marshal(witness)
	if err != nil {
		return "", fmt.Errorf("%w: witness encoding: %v", ErrCacheEvidence, err)
	}
	digest := sha256.Sum256(payload)
	return hex.EncodeToString(digest[:]), nil
}

func validateWitness(w CacheWitness) error {
	if w.RuleID == "" || w.RuleRevision == "" || w.TargetID == "" ||
		w.SemanticRevision == "" || w.AcceptanceRevision == "" ||
		w.ProviderRevision == "" || w.Scope.Environment == "" ||
		w.Scope.Tenant == "" || w.Scope.Subject == "" ||
		w.Scope.PermissionEpoch == "" || len(w.Inputs) == 0 ||
		len(w.Sources) == 0 {
		return fmt.Errorf("%w: missing rule, dependency, authority or source revision", ErrCacheEvidence)
	}
	for _, dependencies := range [][]Revision{w.Inputs, w.Relations} {
		seen := map[string]bool{}
		for _, dep := range dependencies {
			if dep.ID == "" || dep.Revision == "" || seen[dep.ID] {
				return fmt.Errorf("%w: invalid dependency revision", ErrCacheEvidence)
			}
			seen[dep.ID] = true
		}
		if !sort.SliceIsSorted(dependencies, func(i, j int) bool {
			return dependencies[i].ID < dependencies[j].ID
		}) {
			return fmt.Errorf("%w: dependencies are not canonical ordered", ErrCacheEvidence)
		}
	}
	seen := map[string]bool{}
	for _, source := range w.Sources {
		if source.SourceID == "" || source.Revision == "" || seen[source.SourceID] ||
			source.ObservedAt.IsZero() || source.SourceAsOf.IsZero() ||
			source.SourceAsOf.After(source.ObservedAt) {
			return fmt.Errorf("%w: invalid observation/provenance", ErrCacheEvidence)
		}
		seen[source.SourceID] = true
	}
	if !sort.SliceIsSorted(w.Sources, func(i, j int) bool {
		return w.Sources[i].SourceID < w.Sources[j].SourceID
	}) {
		return fmt.Errorf("%w: sources are not canonical ordered", ErrCacheEvidence)
	}
	return nil
}

// CheckSourceFreshAt evaluates optional consumer source staleness against
// the trusted evaluation time, not against when this result entered the memo.
// It rejects impossible/future observations even without an age bound.
// Every source must satisfy the bound; newer data from another source cannot
// conceal one stale dependency. Source timestamps still require the
// CurrentWitnessVerifier to attest their provenance independently.
func CheckSourceFreshAt(witness CacheWitness, policy CachePolicy, now time.Time) error {
	if now.IsZero() || policy.MaxSourceAge < 0 {
		return fmt.Errorf("%w: missing evaluation time or invalid source age bound", ErrCacheEvidence)
	}
	for _, source := range witness.Sources {
		if source.SourceAsOf.IsZero() || source.ObservedAt.IsZero() ||
			source.SourceAsOf.After(source.ObservedAt) || source.ObservedAt.After(now) {
			return fmt.Errorf("%w: source %s has invalid chronology", ErrCacheEvidence, source.SourceID)
		}
		if policy.MaxSourceAge > 0 && now.Sub(source.SourceAsOf) > policy.MaxSourceAge {
			return fmt.Errorf("%w: source %s is older than consumer bound", ErrCacheStale, source.SourceID)
		}
	}
	return nil
}

// ReusableAt decides TTL independently of SourceAsOf/ObservedAt.
// Even a cache younger than MaxAge is not reusable unless the caller has
// separately validated current permissions and source provenance.
func ReusableAt(evaluatedAt, now time.Time, policy CachePolicy) bool {
	if evaluatedAt.IsZero() || now.IsZero() || now.Before(evaluatedAt) ||
		policy.MaxAge <= 0 {
		return false
	}
	return now.Sub(evaluatedAt) <= policy.MaxAge
}

// ConfirmedFact is an immutable read-side projection of an external persisted
// business fact. D1/D2 evaluation and cache eviction never write this record.
type ConfirmedFact struct {
	FactID             string
	TargetID           string
	Value              Input
	SourceRuleID       string
	SourceRuleRevision string
	ConfirmedAt        time.Time
	CommandID          string
	PersistedRevision  string
}

// ConfirmedReader must be backed by an existing authoritative persistence
// adapter. Neither its in-memory fixture nor this API is a commit fence.
type ConfirmedReader interface {
	ReadConfirmed(context.Context, string) (ConfirmedFact, error)
}

// ConfirmedAuthority delegates current disclosure permission to the existing
// Host Authority; it does not confer write/approve/commit permission.
type ConfirmedAuthority interface {
	CanReadConfirmed(context.Context, ConfirmedFact) error
}

func ReadConfirmed(ctx context.Context, factID string, reader ConfirmedReader, authority ConfirmedAuthority) (ConfirmedFact, error) {
	if factID == "" || reader == nil || authority == nil {
		return ConfirmedFact{}, ErrConfirmed
	}
	fact, err := reader.ReadConfirmed(ctx, factID)
	if err != nil {
		return ConfirmedFact{}, fmt.Errorf("%w: read failed: %v", ErrConfirmed, err)
	}
	if fact.FactID != factID || fact.TargetID == "" ||
		fact.SourceRuleID == "" || fact.SourceRuleRevision == "" ||
		fact.ConfirmedAt.IsZero() || fact.CommandID == "" ||
		fact.PersistedRevision == "" ||
		(fact.Value.State != Present && fact.Value.State != Null) {
		return ConfirmedFact{}, ErrConfirmed
	}
	if fact.Value.State == Present {
		if _, ok := new(big.Rat).SetString(fact.Value.Decimal); !ok {
			return ConfirmedFact{}, ErrConfirmed
		}
	}
	if err := authority.CanReadConfirmed(ctx, fact); err != nil {
		return ConfirmedFact{}, fmt.Errorf("%w: current read denied", ErrCacheDenied)
	}
	return fact, nil
}
