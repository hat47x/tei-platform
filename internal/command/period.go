package command

import (
	"context"
	"fmt"
	"sync"

	"tei/internal/core"
	"tei/internal/operations"
)

// PeriodStateは、期間が閉じているかを問い合わせる契約である。
// 状態の保存はProviderに置く。
type PeriodState interface {
	Closed(ctx context.Context, periodID core.CanonicalID) (bool, error)
}

// MapPeriodStateは、期間IDと状態の対応から閉塞を判定する。
type MapPeriodState map[core.CanonicalID]operations.PeriodState

// NewMapPeriodStateは、期間状態の対応を写す。
func NewMapPeriodState(states map[string]operations.PeriodState) MapPeriodState {
	mapped := MapPeriodState{}
	for id, state := range states {
		mapped[core.CanonicalID(id)] = state
	}
	return mapped
}

// Closedは、期間が閉じているかを返す。未指定の期間は閉じていない扱いにする。
func (m MapPeriodState) Closed(_ context.Context, periodID core.CanonicalID) (bool, error) {
	state, ok := m[periodID]
	if !ok {
		return false, nil
	}
	return state == operations.PeriodClosed, nil
}

// ClosedScopeは、期間の半分が閉じているかを返す。範囲の鍵は`sales.month#1`の形で、
// 対応表の鍵と同じ書き方である。未指定の範囲は閉じていない扱いにする（期間全体の
// 未指定と同じ扱い）。期間全体の鍵`sales.month`を半分の答えとして読まない。
func (m MapPeriodState) ClosedScope(_ context.Context, scope string) (bool, error) {
	state, ok := m[core.CanonicalID(scope)]
	if !ok {
		return false, nil
	}
	return state == operations.PeriodClosed, nil
}

// MemoryPeriodStateは、期間の閉塞状態だけを保持する参照実装である。
type MemoryPeriodState struct {
	mu     sync.Mutex
	closed map[core.CanonicalID]bool
}

// NewMemoryPeriodStateは、空の参照状態を作る。
func NewMemoryPeriodState() *MemoryPeriodState {
	return &MemoryPeriodState{closed: map[core.CanonicalID]bool{}}
}

// SetClosedは、期間の閉塞状態を設定する。
func (s *MemoryPeriodState) SetClosed(periodID core.CanonicalID, closed bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed[periodID] = closed
}

// Closedは、期間が閉じているかを返す。
func (s *MemoryPeriodState) Closed(_ context.Context, periodID core.CanonicalID) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	closed, ok := s.closed[periodID]
	if !ok {
		return false, fmt.Errorf("period %s has no state", periodID)
	}
	return closed, nil
}

// SetScopeClosedは、期間の半分の閉塞状態を設定する。
func (s *MemoryPeriodState) SetScopeClosed(scope string, closed bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed[core.CanonicalID(scope)] = closed
}

// ClosedScopeは、期間の半分が閉じているかを返す。未設定の範囲は閉じていない扱いに
// する。期間全体の鍵で代用せず、設定された範囲だけを答える。
func (s *MemoryPeriodState) ClosedScope(_ context.Context, scope string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.closed[core.CanonicalID(scope)], nil
}
