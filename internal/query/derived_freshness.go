package query

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"tei/internal/core"
	"tei/internal/semantic/derivationref"
)

// CompleteDerivedSource supplies the D2 input universe. Its implementation
// must use the existing provider/driver's bounded complete result + explicit
// raw-to-Semantic mapping. A partial page is never a cacheable source.
type CompleteDerivedSource func(context.Context, int) ([]DerivedRow, bool, error)

type derivedMemoEntry struct {
	result      DerivedSelectResult
	evaluatedAt time.Time
}

// DerivedMemo is an optional, in-process D3 reference for the D2 Go plan.
// This is not a distributed or persistent cache and has no authority of its
// own. D2 field-read authorization is always required. Cache/current witness
// verification applies only when caching or freshness/alignment is requested.
const maxDerivedMemoEntries = 64

type DerivedMemo struct {
	mu      sync.RWMutex
	entries map[string]derivedMemoEntry
	order   []string
}

// Resolve returns a qualified cached result or evaluates the complete source.
// Cached rows are copied on ingress/egress. No confirmed fact is ever written.
func (m *DerivedMemo) Resolve(
	ctx context.Context,
	q EmbeddedSQL,
	plan DerivedSelect,
	sqlEvidence CompleteSourceEvidence,
	witness derivationref.CacheWitness,
	policy derivationref.CachePolicy,
	now time.Time,
	verifier derivationref.CurrentWitnessVerifier,
	canRead func(core.CanonicalID) bool,
	load CompleteDerivedSource,
) (DerivedSelectResult, bool, error) {
	if now.IsZero() {
		return DerivedSelectResult{}, false, fmt.Errorf("%w: missing trusted evaluation time", derivationref.ErrCacheEvidence)
	}
	if err := ValidateDerivedSelect(q, plan, sqlEvidence, canRead); err != nil {
		return DerivedSelectResult{}, false, err
	}
	// A purely computed result does not opt into caching, source-age
	// promises or cross-source alignment. It needs no cache-specific
	// witness/epoch/verifier, but still requires D2 SQL preflight, bounded
	// complete retrieval and current field-read authorization before AND
	// after source evaluation. The Host must supply a trusted canRead.
	if policy.MaxAge == 0 && policy.MaxSourceAge == 0 && !policy.RequireAligned {
		if load == nil {
			return DerivedSelectResult{}, false, ErrDerivedIncomplete
		}
		rows, complete, err := load(ctx, plan.MaxRows)
		if err != nil {
			return DerivedSelectResult{}, false, err
		}
		result, err := EvaluateDerivedSelect(plan, rows, complete)
		if err != nil {
			return DerivedSelectResult{}, false, err
		}
		if err := ValidateDerivedSelect(q, plan, sqlEvidence, canRead); err != nil {
			return DerivedSelectResult{}, false, err
		}
		return result, false, nil
	}
	if !witnessMatchesPlan(q, plan, witness) {
		return DerivedSelectResult{}, false, fmt.Errorf("%w: witness does not match D2 plan", derivationref.ErrCacheEvidence)
	}
	// Memo TTL only measures time since evaluation. A consumer may
	// independently require the underlying SourceAsOf to be recent.
	// Fail before cache hits AND before loading another stale full source.
	if err := derivationref.CheckSourceFreshAt(witness, policy, now); err != nil {
		return DerivedSelectResult{}, false, err
	}
	witnessKey, err := derivationref.QualifyCache(ctx, witness, policy, verifier)
	if err != nil {
		return DerivedSelectResult{}, false, err
	}
	// Key includes the full D2 plan and exact native SQL digest. Changing
	// predicate, order, offset/limit or source prevents cross-plan reuse.
	b, err := json.Marshal(struct {
		Query string
		SQL string
		Plan DerivedSelect
		Witness string
	}{Query: string(q.ID), SQL: DigestNativeSQL(q.Source), Plan: plan, Witness: witnessKey})
	if err != nil {
		return DerivedSelectResult{}, false, fmt.Errorf("%w: plan key encoding", derivationref.ErrCacheEvidence)
	}
	sum := sha256.Sum256(b)
	key := hex.EncodeToString(sum[:])

	if policy.MaxAge > 0 && m != nil {
		m.mu.RLock()
		entry, found := m.entries[key]
		m.mu.RUnlock()
		if found && derivationref.ReusableAt(entry.evaluatedAt, now, policy) {
			return copyDerivedResult(entry.result), true, nil
		}
	}
	if load == nil {
		return DerivedSelectResult{}, false, ErrDerivedIncomplete
	}
	rows, complete, err := load(ctx, plan.MaxRows)
	if err != nil {
		return DerivedSelectResult{}, false, err
	}
	result, err := EvaluateDerivedSelect(plan, rows, complete)
	if err != nil {
		return DerivedSelectResult{}, false, err
	}
	// Recheck current Authority and revisions after source computation to
	// avoid storing stale evidence after an observed epoch change.
	if _, err := derivationref.QualifyCache(ctx, witness, policy, verifier); err != nil {
		return DerivedSelectResult{}, false, err
	}
	if err := ValidateDerivedSelect(q, plan, sqlEvidence, canRead); err != nil {
		return DerivedSelectResult{}, false, err
	}
	if policy.MaxAge > 0 && m != nil {
		m.mu.Lock()
		if m.entries == nil {
			m.entries = map[string]derivedMemoEntry{}
		}
		if _, exists := m.entries[key]; !exists {
			if len(m.entries) >= maxDerivedMemoEntries {
				delete(m.entries, m.order[0])
				m.order = m.order[1:]
			}
			m.order = append(m.order, key)
		}
		m.entries[key] = derivedMemoEntry{result: copyDerivedResult(result), evaluatedAt: now}
		m.mu.Unlock()
	}
	return result, false, nil
}

func witnessMatchesPlan(q EmbeddedSQL, p DerivedSelect, witness derivationref.CacheWitness) bool {
	if witness.RuleID != p.Rule.ID || witness.TargetID != p.Binding.TargetID ||
		p.Binding.RuleID != p.Rule.ID || q.ID == "" {
		return false
	}
	attributes := map[core.CanonicalID]bool{}
	for _, input := range p.Rule.Inputs {
		id := p.Inputs[input]
		if id == "" || attributes[id] {
			return false
		}
		attributes[id] = true
	}
	if len(attributes) != len(witness.Inputs) {
		return false
	}
	for _, dep := range witness.Inputs {
		if !attributes[core.CanonicalID(dep.ID)] {
			return false
		}
	}
	return true
}

func copyDerivedResult(in DerivedSelectResult) DerivedSelectResult {
	out := DerivedSelectResult{Count: in.Count, Rows: make([]DerivedRow, len(in.Rows))}
	for i, row := range in.Rows {
		values := make(map[core.CanonicalID]derivationref.Input, len(row.Values))
		for k, v := range row.Values {
			values[k] = v
		}
		out.Rows[i] = DerivedRow{Values: values}
	}
	return out
}
