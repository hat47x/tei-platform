package command

import (
	"context"
	"fmt"
	"sync"

	"tei/internal/core"
)

// PeriodStateは、期間が閉じているかを問い合わせる契約である。
// 状態の保存はProviderに置く。
type PeriodState interface {
	Closed(ctx context.Context, periodID core.CanonicalID) (bool, error)
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
