// Package derivationtemporal is a D14 read-only reference boundary.
// It keeps the time axes and the meanings of a business value apart: a
// recomputation of a past valid-time with the then-accepted rule, a what-if
// that deliberately uses a different rule, a current computation, and the
// stored confirmed fact are four different results, not one "asOf" number.
//
// It does not define D1 rules, D3 freshness/confirmed storage, D11 in-read
// snapshot, D12 clocks, D7 explanations, or any bitemporal database.
package derivationtemporal

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"tei/internal/semantic/derivationref"
)

var (
	ErrInvalidRequest  = errors.New("derivation temporal request is invalid")
	ErrHistoryUnknown  = errors.New("historical input is unavailable")
	ErrRevisionUnknown = errors.New("historical rule revision is unavailable")
)

// Kind is the meaning of a result on the time axis.
type Kind string

const (
	CurrentComputation Kind = "current-computation"
	HistoricalReplay   Kind = "historical-replay"
	WhatIf             Kind = "what-if"
	ConfirmedFactRead  Kind = "confirmed-fact-read"
)

// PriceBasis states explicitly which price a request uses. There is no silent
// fallback from a missing historical price to the current one.
type PriceBasis string

const (
	BasisHistorical PriceBasis = "historical"
	BasisCurrent    PriceBasis = "current"
)

// PricePoint is one historical input value with its valid-from instant.
type PricePoint struct {
	ValidFrom time.Time
	Price     derivationref.Input
}

// ConfirmedFact is a stored business fact: the value a past Action saved. A
// recomputation never overwrites it, even when the numbers differ.
type ConfirmedFact struct {
	Amount     derivationref.Input
	ValidTime  time.Time
	RecordedAt time.Time
}

// ComputeFunc is one accepted rule body for a revision. The reference passes
// the resolved quantity and price; it does not introduce a rule language.
type ComputeFunc func(quantity, price derivationref.Input) (derivationref.Input, error)

// Request is one recomputation request. ValidTime is the business effective
// instant, RecordedAt is when the provider observed the input and EvaluatedAt
// is when this recomputation runs; they are not interchangeable.
type Request struct {
	Kind         Kind
	ValidTime    time.Time
	RecordedAt   time.Time
	EvaluatedAt  time.Time
	Quantity     derivationref.Input
	PriceBasis   PriceBasis
	History      []PricePoint
	CurrentPrice derivationref.Input
	RuleRevision string
	// AvailableRevisions maps a revision to its accepted body. A revision that
	// is absent is unknown, not the current one.
	AvailableRevisions map[string]ComputeFunc
	Confirmed          *ConfirmedFact
}

// Outcome is the labelled result. Replayable is true only for a proven
// historical replay. Confirmed is carried through unchanged and only for
// ConfirmedFactRead.
type Outcome struct {
	Kind         Kind                `json:"kind"`
	Value        derivationref.Input `json:"value"`
	RuleRevision string              `json:"ruleRevision,omitempty"`
	Replayable   bool                `json:"replayable"`
	Confirmed    *ConfirmedFact      `json:"confirmed,omitempty"`
}

// Replay resolves one request. A historical replay requires both the price
// valid at ValidTime and the selected rule revision; if either is missing it
// fails instead of substituting the current price or rule. A what-if is always
// labelled what-if and never carries the confirmed fact.
func Replay(req Request) (Outcome, error) {
	if err := validate(req); err != nil {
		return Outcome{}, err
	}
	if req.Kind == ConfirmedFactRead {
		return Outcome{
			Kind:      ConfirmedFactRead,
			Value:     req.Confirmed.Amount,
			Confirmed: req.Confirmed,
		}, nil
	}
	price, err := req.resolvePrice()
	if err != nil {
		return Outcome{}, err
	}
	compute, ok := req.AvailableRevisions[req.RuleRevision]
	if !ok {
		return Outcome{}, fmt.Errorf("%w: %s", ErrRevisionUnknown, req.RuleRevision)
	}
	value, err := compute(req.Quantity, price)
	if err != nil {
		return Outcome{}, err
	}
	return Outcome{
		Kind:         req.Kind,
		Value:        value,
		RuleRevision: req.RuleRevision,
		Replayable:   req.Kind == HistoricalReplay,
	}, nil
}

func (req Request) resolvePrice() (derivationref.Input, error) {
	switch req.PriceBasis {
	case BasisCurrent:
		return req.CurrentPrice, nil
	case BasisHistorical:
		price, ok := priceAt(req.History, req.ValidTime)
		if !ok {
			return derivationref.Input{}, fmt.Errorf("%w: valid-time %s", ErrHistoryUnknown, req.ValidTime.Format(time.RFC3339))
		}
		return price, nil
	default:
		return derivationref.Input{}, fmt.Errorf("%w: price basis", ErrInvalidRequest)
	}
}

func validate(req Request) error {
	if req.ValidTime.IsZero() {
		return fmt.Errorf("%w: valid-time", ErrInvalidRequest)
	}
	switch req.Kind {
	case ConfirmedFactRead:
		if req.Confirmed == nil {
			return fmt.Errorf("%w: confirmed fact", ErrInvalidRequest)
		}
		return nil
	case CurrentComputation:
		if req.PriceBasis != BasisCurrent {
			return fmt.Errorf("%w: current computation needs the current basis", ErrInvalidRequest)
		}
	case HistoricalReplay:
		if req.PriceBasis != BasisHistorical {
			return fmt.Errorf("%w: historical replay needs the historical basis", ErrInvalidRequest)
		}
	case WhatIf:
		// either basis is allowed as a deliberate choice
	default:
		return fmt.Errorf("%w: kind %q", ErrInvalidRequest, req.Kind)
	}
	if strings.TrimSpace(req.RuleRevision) == "" {
		return fmt.Errorf("%w: rule revision", ErrInvalidRequest)
	}
	return nil
}

func priceAt(history []PricePoint, at time.Time) (derivationref.Input, bool) {
	var best *PricePoint
	for i := range history {
		if !history[i].ValidFrom.After(at) {
			if best == nil || history[i].ValidFrom.After(best.ValidFrom) {
				best = &history[i]
			}
		}
	}
	if best == nil {
		return derivationref.Input{}, false
	}
	return best.Price, true
}
