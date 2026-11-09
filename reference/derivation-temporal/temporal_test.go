package derivationtemporal

import (
	"errors"
	"testing"
	"time"

	"tei/internal/semantic/derivationref"
)

func date(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

func present(decimal string) derivationref.Input {
	return derivationref.Input{State: derivationref.Present, Decimal: decimal}
}

func multiply(quantity, price derivationref.Input) (derivationref.Input, error) {
	return derivationref.Multiply(quantity, price)
}

func discounted(quantity, price derivationref.Input) (derivationref.Input, error) {
	amount, err := derivationref.Multiply(quantity, price)
	if err != nil {
		return amount, err
	}
	return derivationref.Multiply(amount, present("0.9"))
}

func revisions() map[string]ComputeFunc {
	return map[string]ComputeFunc{"v1": multiply, "v2": discounted}
}

// Price P: 300 from 2026-04-01, 350 from 2026-06-01. Order O was confirmed on
// 2026-04-15 for quantity 2 at 600.
func history() []PricePoint {
	return []PricePoint{
		{ValidFrom: date(2026, time.April, 1), Price: present("300")},
		{ValidFrom: date(2026, time.June, 1), Price: present("350")},
	}
}

func confirmedFact() *ConfirmedFact {
	return &ConfirmedFact{
		Amount:     present("600"),
		ValidTime:  date(2026, time.April, 15),
		RecordedAt: date(2026, time.April, 15),
	}
}

func baseRequest(kind Kind, basis PriceBasis, revision string) Request {
	return Request{
		Kind:               kind,
		ValidTime:          date(2026, time.April, 15),
		RecordedAt:         date(2026, time.April, 15),
		EvaluatedAt:        date(2026, time.October, 16),
		Quantity:           present("2"),
		PriceBasis:         basis,
		History:            history(),
		CurrentPrice:       present("350"),
		RuleRevision:       revision,
		AvailableRevisions: revisions(),
	}
}

func TestD14HistoricalReplayUsesThenPriceAndRule(t *testing.T) {
	outcome, err := Replay(baseRequest(HistoricalReplay, BasisHistorical, "v1"))
	if err != nil {
		t.Fatal(err)
	}
	if outcome.Kind != HistoricalReplay || !outcome.Replayable || outcome.Value.Decimal != "600" {
		t.Fatalf("historical replay wrong: %+v", outcome)
	}
}

func TestD14WhatIfDoesNotChangeConfirmedFact(t *testing.T) {
	stored := confirmedFact()
	req := baseRequest(WhatIf, BasisHistorical, "v2")
	req.Confirmed = stored
	outcome, err := Replay(req)
	if err != nil {
		t.Fatal(err)
	}
	if outcome.Kind != WhatIf || outcome.Replayable || outcome.Value.Decimal != "540" {
		t.Fatalf("what-if wrong: %+v", outcome)
	}
	if outcome.Confirmed != nil {
		t.Fatal("what-if carried the confirmed fact as its own value")
	}
	if stored.Amount.Decimal != "600" {
		t.Fatalf("confirmed fact was overwritten: %+v", stored)
	}
}

func TestD14CurrentComputationUsesCurrentPrice(t *testing.T) {
	outcome, err := Replay(baseRequest(CurrentComputation, BasisCurrent, "v1"))
	if err != nil {
		t.Fatal(err)
	}
	if outcome.Kind != CurrentComputation || outcome.Replayable || outcome.Value.Decimal != "700" {
		t.Fatalf("current computation wrong: %+v", outcome)
	}
}

func TestD14MissingHistoryIsNotEstimated(t *testing.T) {
	req := baseRequest(HistoricalReplay, BasisHistorical, "v1")
	req.History = nil // provider has only the current price 350
	_, err := Replay(req)
	if !errors.Is(err, ErrHistoryUnknown) {
		t.Fatalf("missing history was not rejected: %v", err)
	}
}

func TestD14MissingRevisionIsNotSubstituted(t *testing.T) {
	req := baseRequest(HistoricalReplay, BasisHistorical, "v1")
	req.AvailableRevisions = map[string]ComputeFunc{"v2": discounted}
	_, err := Replay(req)
	if !errors.Is(err, ErrRevisionUnknown) {
		t.Fatalf("missing before-revision substituted by the current one: %v", err)
	}
}

func TestD14ConfirmedFactReadIsNotAReplay(t *testing.T) {
	req := Request{
		Kind:        ConfirmedFactRead,
		ValidTime:   date(2026, time.April, 15),
		EvaluatedAt: date(2026, time.October, 16),
		Confirmed:   confirmedFact(),
	}
	outcome, err := Replay(req)
	if err != nil {
		t.Fatal(err)
	}
	if outcome.Kind != ConfirmedFactRead || outcome.Replayable || outcome.Value.Decimal != "600" {
		t.Fatalf("confirmed fact read wrong: %+v", outcome)
	}
	if outcome.Confirmed == nil || outcome.Confirmed.RecordedAt != date(2026, time.April, 15) {
		t.Fatalf("stored fact was not carried through: %+v", outcome.Confirmed)
	}
}

func TestD14KindAndBasisAreValidated(t *testing.T) {
	if _, err := Replay(baseRequest(HistoricalReplay, BasisCurrent, "v1")); !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("historical replay with the current basis accepted: %v", err)
	}
	if _, err := Replay(baseRequest(CurrentComputation, BasisHistorical, "v1")); !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("current computation with the historical basis accepted: %v", err)
	}
	if _, err := Replay(baseRequest(Kind("nonsense"), BasisCurrent, "v1")); !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("unknown kind accepted: %v", err)
	}
	if _, err := Replay(Request{Kind: ConfirmedFactRead, ValidTime: date(2026, time.April, 15)}); !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("confirmed-fact read without a fact accepted: %v", err)
	}
}
