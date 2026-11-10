package command

import (
	"context"
	"fmt"
	"sync"

	"tei/internal/ir"
	"tei/internal/semantic"
)

// ReferenceExecutorは、宣言された最初の結末と効果をそのまま返す参照実装である。
// 業務ロジックも永続化も行わない。Executor契約の配線を確認するためだけに使う。
type ReferenceExecutor struct {
	World ir.World
}

func (e ReferenceExecutor) Execute(_ context.Context, request Request) (Result, error) {
	var capability *semantic.Capability
	for i := range e.World.Semantics {
		for j := range e.World.Semantics[i].Capabilities {
			if e.World.Semantics[i].Capabilities[j].ID == request.Capability {
				capability = &e.World.Semantics[i].Capabilities[j]
				break
			}
		}
		if capability != nil {
			break
		}
	}
	if capability == nil {
		return Result{}, fmt.Errorf("capability %s is not defined", request.Capability)
	}
	result := Result{}
	// A capability may declare several outcomes, and nothing declares which one
	// a given execution produces. Returning the first one reported a decision
	// the declaration never made — and after the loader sorts outcomes by name,
	// "first" means whichever name sorts earliest, so the answer changed with a
	// rename. Only a single outcome can be returned without inventing a rule.
	switch len(capability.Outcomes) {
	case 0:
	case 1:
		outcome := capability.Outcomes[0]
		result.Outcome = &outcome
	default:
		result.OutcomeUndetermined = fmt.Sprintf(
			"capability %s declares %d outcomes and no declared rule selects one",
			capability.ID, len(capability.Outcomes),
		)
	}
	if capability.Effect != nil {
		result.Effect = capability.Effect
	}
	return result, nil
}

// MemoryUnitOfWorkは、CommitとRollbackの呼出しだけを記録する参照実装である。
// 実データの取引は行わない。
type MemoryUnitOfWork struct {
	mu         sync.Mutex
	commits    int
	rollbacks  int
	FailBegin  bool
	FailCommit bool
}

func (u *MemoryUnitOfWork) Begin(context.Context) (Transaction, error) {
	u.mu.Lock()
	defer u.mu.Unlock()
	if u.FailBegin {
		return nil, fmt.Errorf("reference unit of work cannot begin")
	}
	return &memoryTransaction{unit: u}, nil
}

// Countsは、CommitとRollbackの回数を返す。
func (u *MemoryUnitOfWork) Counts() (commits, rollbacks int) {
	u.mu.Lock()
	defer u.mu.Unlock()
	return u.commits, u.rollbacks
}

type memoryTransaction struct {
	unit *MemoryUnitOfWork
}

func (t *memoryTransaction) Commit(context.Context) error {
	t.unit.mu.Lock()
	defer t.unit.mu.Unlock()
	if t.unit.FailCommit {
		return fmt.Errorf("reference unit of work cannot commit")
	}
	t.unit.commits++
	return nil
}

func (t *memoryTransaction) Rollback(context.Context) error {
	t.unit.mu.Lock()
	defer t.unit.mu.Unlock()
	t.unit.rollbacks++
	return nil
}
