// Package derivationsnapshot is a D11 read-only reference boundary.
// It checks whether the inputs of one evaluation belong to the read snapshot
// or version set that the case requires, and refuses to present a mixture that
// exists in no single revision as a confirmed value.
//
// It does not define the D1 rule contract, D3 freshness/caches/confirmed
// storage, D12 clocks, D7 explanations, D5 authorization, any distributed
// transaction, or a common snapshot API for every provider.
package derivationsnapshot

import (
	"errors"
	"fmt"
	"strings"

	"tei/internal/semantic/derivationref"
)

var (
	ErrMixedSnapshot      = errors.New("read set mixes snapshots or versions")
	ErrInputUnavailable   = errors.New("a required input is unavailable")
	ErrInvalidRequirement = errors.New("consistency requirement is invalid")
	ErrStaleAuthorization = errors.New("authorization changed before re-read")
	ErrPartialReadSet     = errors.New("re-read does not cover the whole input set")
)

// ErrSnapshotLost is wrapped by a read when the provider invalidated the read
// snapshot mid-evaluation. It must never be read as a NULL or an empty set.
var ErrSnapshotLost = errors.New("provider lost the read snapshot")

// Consistency is the minimum agreement one evaluation asks of its inputs.
// These are candidate terms; no permanent core enum is fixed here.
type Consistency string

const (
	SameRecord              Consistency = "same-record"
	SameTransactionSnapshot Consistency = "same-transaction-snapshot"
	VersionAligned          Consistency = "version-aligned"
	BestEffort              Consistency = "best-effort"
)

// Verdict is the outcome of checking a read set against the requirement.
type Verdict string

const (
	Guaranteed  Verdict = "guaranteed"
	Unsupported Verdict = "unsupported"
	NonAtomic   Verdict = "non-atomic"
)

// Read is one input read and its provenance. Snapshot, Version and Record are
// empty when the provider cannot prove them: an absent proof is not a match.
type Read struct {
	InputID  string
	Provider string
	Snapshot string
	Version  string
	Record   string
	Value    derivationref.Input
	Err      error
}

// ReadSetResult carries the checked reads. Confirmed is true only when the
// requirement was proven for the whole set; a non-atomic or unsupported set is
// never labelled as a confirmed current value.
type ReadSetResult struct {
	Verdict     Verdict     `json:"verdict"`
	Consistency Consistency `json:"consistency"`
	Confirmed   bool        `json:"confirmed"`
	Reads       []Read      `json:"reads"`
}

// Evaluate checks one evaluation's reads against the case requirement.
//   - A proven set is Guaranteed and Confirmed.
//   - A requirement that cannot be proven because the proof is absent (for
//     example a cross-provider read with no shared snapshot) is Unsupported.
//   - Two non-empty proofs that disagree (different snapshot tokens within one
//     provider) are rejected as ErrMixedSnapshot unless the case explicitly
//     accepts a non-atomic read, in which case the result is NonAtomic and not
//     Confirmed.
//   - BestEffort never claims more than NonAtomic.
//
// A read error is reported as ErrInputUnavailable, never converted to NULL.
func Evaluate(requirement Consistency, acceptsNonAtomic bool, reads []Read) (ReadSetResult, error) {
	if !known(requirement) {
		return ReadSetResult{}, fmt.Errorf("%w: %q", ErrInvalidRequirement, requirement)
	}
	if len(reads) == 0 {
		return ReadSetResult{}, fmt.Errorf("%w: no input read", ErrInputUnavailable)
	}
	seen := map[string]bool{}
	for _, read := range reads {
		if read.InputID == "" || read.InputID != strings.TrimSpace(read.InputID) || read.Provider == "" {
			return ReadSetResult{}, fmt.Errorf("%w: input identity", ErrInvalidRequirement)
		}
		if seen[read.InputID] {
			return ReadSetResult{}, fmt.Errorf("%w: duplicate input %s", ErrInvalidRequirement, read.InputID)
		}
		seen[read.InputID] = true
		if read.Err != nil {
			return ReadSetResult{}, fmt.Errorf("%w: %s: %w", ErrInputUnavailable, read.InputID, read.Err)
		}
	}
	if requirement == BestEffort {
		return ReadSetResult{Verdict: NonAtomic, Consistency: requirement, Reads: reads}, nil
	}
	if proven(requirement, reads) {
		return ReadSetResult{Verdict: Guaranteed, Consistency: requirement, Confirmed: true, Reads: reads}, nil
	}
	if acceptsNonAtomic {
		return ReadSetResult{Verdict: NonAtomic, Consistency: requirement, Reads: reads}, nil
	}
	if requirement == SameTransactionSnapshot && spansProviders(reads) {
		// The platform cannot atomically read across providers; the capability
		// is absent rather than an observed conflict.
		return ReadSetResult{Verdict: Unsupported, Consistency: requirement, Reads: reads}, nil
	}
	if contradicted(requirement, reads) {
		return ReadSetResult{}, fmt.Errorf("%w: %s", ErrMixedSnapshot, requirement)
	}
	return ReadSetResult{Verdict: Unsupported, Consistency: requirement, Reads: reads}, nil
}

// ReReadAfterConflict re-reads the whole input set after a conflict. A previous
// result must not be extended with one refreshed input, and a changed
// authorization version must be re-checked before the set is used again. It
// performs no distributed transaction and introduces no partial merge.
func ReReadAfterConflict(previous ReadSetResult, previousAuthVersion, currentAuthVersion string, requirement Consistency, acceptsNonAtomic bool, reads []Read) (ReadSetResult, error) {
	if previousAuthVersion == "" || currentAuthVersion == "" {
		return ReadSetResult{}, fmt.Errorf("%w: authorization version", ErrInvalidRequirement)
	}
	if previousAuthVersion != currentAuthVersion {
		return ReadSetResult{}, fmt.Errorf("%w: %s -> %s", ErrStaleAuthorization, previousAuthVersion, currentAuthVersion)
	}
	want := map[string]bool{}
	for _, read := range previous.Reads {
		want[read.InputID] = true
	}
	got := map[string]bool{}
	for _, read := range reads {
		got[read.InputID] = true
	}
	for id := range want {
		if !got[id] {
			return ReadSetResult{}, fmt.Errorf("%w: missing %s", ErrPartialReadSet, id)
		}
	}
	return Evaluate(requirement, acceptsNonAtomic, reads)
}

func proven(requirement Consistency, reads []Read) bool {
	switch requirement {
	case SameTransactionSnapshot:
		token := ""
		for _, read := range reads {
			if read.Snapshot == "" {
				return false
			}
			if token == "" {
				token = read.Snapshot
			} else if read.Snapshot != token {
				return false
			}
		}
		return true
	case VersionAligned:
		for _, read := range reads {
			if read.Version == "" {
				return false
			}
		}
		return true
	case SameRecord:
		provider, record := "", ""
		for _, read := range reads {
			if read.Record == "" {
				return false
			}
			if provider == "" {
				provider, record = read.Provider, read.Record
				continue
			}
			if read.Provider != provider || read.Record != record {
				return false
			}
		}
		return true
	default:
		return false
	}
}

func contradicted(requirement Consistency, reads []Read) bool {
	switch requirement {
	case SameTransactionSnapshot:
		token := ""
		for _, read := range reads {
			if read.Snapshot == "" {
				continue
			}
			if token == "" {
				token = read.Snapshot
			} else if read.Snapshot != token {
				return true
			}
		}
	case VersionAligned:
		versions := map[string]string{}
		for _, read := range reads {
			if read.Version == "" {
				continue
			}
			if prev, ok := versions[read.InputID]; ok && prev != read.Version {
				return true
			}
			versions[read.InputID] = read.Version
		}
	case SameRecord:
		provider, record := "", ""
		for _, read := range reads {
			if read.Record == "" {
				continue
			}
			if provider == "" {
				provider, record = read.Provider, read.Record
				continue
			}
			if read.Provider != provider || read.Record != record {
				return true
			}
		}
	}
	return false
}

func spansProviders(reads []Read) bool {
	provider := ""
	for _, read := range reads {
		if provider == "" {
			provider = read.Provider
			continue
		}
		if read.Provider != provider {
			return true
		}
	}
	return false
}

func known(requirement Consistency) bool {
	switch requirement {
	case SameRecord, SameTransactionSnapshot, VersionAligned, BestEffort:
		return true
	default:
		return false
	}
}
