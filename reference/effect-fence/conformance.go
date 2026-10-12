package effectfence

import (
	"context"
	"fmt"
	"sync"
)

// ConformanceFixture は、隔離された試験環境における更新・観測用の入口を表す。
// 本番の権限を変更するための公開APIではない。各試験は空の状態から開始する。
// 外部Adapterを検証する際は、使い捨ての保存先・模擬権限をこの形に接続する。
type ConformanceFixture interface {
	SetAuthority(Scope, uint64, bool) error
	AdvanceResource(Scope, string) error
	PrepareForTest(context.Context, Request) (any, Result, error)
	CommitForTest(context.Context, any) (Result, error)
	Observe(Scope) (string, uint64)
	Lookup(Scope, string) (Receipt, bool)
}

type ConformanceResult struct {
	Name   string
	Passed bool
	Detail string
}

// CheckAtomicConformance は、実際の副作用の原子性に関する最小試験を実行する。
// ここでPASSしても永続性や本番認可・別プロセス間の原子性の証拠にはならない。
// 呼出元はテストした対象、配置、版、実行環境を別途記録する必要がある。
func CheckAtomicConformance(ctx context.Context, fresh func() (ConformanceFixture, error)) []ConformanceResult {
	scope := Scope{Application: "conformance-app", Capability: "save", Subject: "editor", Action: "write", Resource: "record-1"}
	req := func(id string) Request {
		return Request{Scope: scope, OperationID: id, ExpectedResourceVersion: 0, Value: "更新後"}
	}
	type scenario struct {
		name string
		run  func(ConformanceFixture) error
	}
	prepare := func(f ConformanceFixture, r Request) (any, error) {
		p, result, err := f.PrepareForTest(ctx, r)
		if err != nil {
			return nil, err
		}
		if result.Disposition != Prepared {
			return nil, fmt.Errorf("事前検査に失敗: %s (%s)", result.Disposition, result.Reason)
		}
		return p, nil
	}
	commit := func(f ConformanceFixture, p any) (Result, error) { return f.CommitForTest(ctx, p) }
	scenarios := []scenario{
		{name: "commit-observe-replay", run: func(f ConformanceFixture) error {
			r := req("op-1")
			p, err := prepare(f, r)
			if err != nil {
				return err
			}
			result, err := commit(f, p)
			if err != nil || result.Disposition != Committed || result.Receipt.ResourceVersion != 1 {
				return fmt.Errorf("確定結果が不正: %+v: %v", result, err)
			}
			if value, version := f.Observe(scope); value != r.Value || version != 1 {
				return fmt.Errorf("確定後の状態が不正: %q / %d", value, version)
			}
			receipt, ok := f.Lookup(scope, r.OperationID)
			if !ok || receipt != result.Receipt {
				return fmt.Errorf("確定記録を再照会できません")
			}
			_, again, err := f.PrepareForTest(ctx, r)
			if err != nil || again.Disposition != Replayed || again.Receipt != receipt {
				return fmt.Errorf("再送時に元の結果を返しません: %+v: %v", again, err)
			}
			if _, version := f.Observe(scope); version != 1 {
				return fmt.Errorf("再送で二重に更新されました: %d", version)
			}
			return nil
		}},
		{name: "revoked-before-commit", run: func(f ConformanceFixture) error {
			p, err := prepare(f, req("op-2"))
			if err != nil {
				return err
			}
			if err = f.SetAuthority(scope, 2, false); err != nil {
				return err
			}
			result, err := commit(f, p)
			if err != nil || result.Disposition != Denied {
				return fmt.Errorf("失効した許可で操作が通過: %+v: %v", result, err)
			}
			if _, version := f.Observe(scope); version != 0 {
				return fmt.Errorf("失効後に保存されました: %d", version)
			}
			return nil
		}},
		{name: "epoch-changed-before-commit", run: func(f ConformanceFixture) error {
			p, err := prepare(f, req("op-3"))
			if err != nil {
				return err
			}
			if err = f.SetAuthority(scope, 2, true); err != nil {
				return err
			}
			result, err := commit(f, p)
			if err != nil || result.Disposition != Denied {
				return fmt.Errorf("古い許可版が通過: %+v: %v", result, err)
			}
			if _, version := f.Observe(scope); version != 0 {
				return fmt.Errorf("古い許可版で書込み: %d", version)
			}
			return nil
		}},
		{name: "resource-changed-before-commit", run: func(f ConformanceFixture) error {
			p, err := prepare(f, req("op-4"))
			if err != nil {
				return err
			}
			if err = f.AdvanceResource(scope, "別の書き手"); err != nil {
				return err
			}
			result, err := commit(f, p)
			if err != nil || result.Disposition != Denied {
				return fmt.Errorf("古い保存先版が通過: %+v: %v", result, err)
			}
			if value, version := f.Observe(scope); value != "別の書き手" || version != 1 {
				return fmt.Errorf("先行する書込みを破壊: %q / %d", value, version)
			}
			return nil
		}},
		{name: "parallel-same-operation", run: func(f ConformanceFixture) error {
			r := req("op-5")
			p1, err := prepare(f, r)
			if err != nil {
				return err
			}
			p2, err := prepare(f, r)
			if err != nil {
				return err
			}
			var wg sync.WaitGroup
			results := make(chan Result, 2)
			errs := make(chan error, 2)
			for _, p := range []any{p1, p2} {
				wg.Add(1)
				go func(p any) {
					defer wg.Done()
					result, err := commit(f, p)
					results <- result
					errs <- err
				}(p)
			}
			wg.Wait()
			close(results)
			close(errs)
			for err := range errs {
				if err != nil {
					return err
				}
			}
			counts := map[Disposition]int{}
			for result := range results {
				counts[result.Disposition]++
			}
			if counts[Committed] != 1 || counts[Replayed] != 1 {
				return fmt.Errorf("同一操作の実行回数が不正: %+v", counts)
			}
			if _, version := f.Observe(scope); version != 1 {
				return fmt.Errorf("二重に更新されました: %d", version)
			}
			return nil
		}},
		{name: "no-cross-scope-authority", run: func(f ConformanceFixture) error {
			r := req("op-6")
			r.Scope.Subject = "viewer"
			_, result, err := f.Prepare(ctx, r)
			if err != nil || result.Disposition != Denied {
				return fmt.Errorf("別主体への権限流用: %+v: %v", result, err)
			}
			if _, version := f.Observe(scope); version != 0 {
				return fmt.Errorf("別主体が保存しました: %d", version)
			}
			return nil
		}},
	}

	results := make([]ConformanceResult, 0, len(scenarios))
	for _, s := range scenarios {
		check := ConformanceResult{Name: s.name}
		if err := ctx.Err(); err != nil {
			check.Detail = err.Error()
		} else if fresh == nil {
			check.Detail = "試験用の作業領域生成が未指定"
		} else if f, err := fresh(); err != nil {
			check.Detail = err.Error()
		} else if f == nil {
			check.Detail = "試験用の作業領域がありません"
		} else if value, version := f.Observe(scope); value != "" || version != 0 {
			check.Detail = "隔離された空の作業領域ではありません"
		} else if err := f.SetAuthority(scope, 1, true); err != nil {
			check.Detail = err.Error()
		} else if err := s.run(f); err != nil {
			check.Detail = err.Error()
		} else {
			check.Passed = true
		}
		results = append(results, check)
	}
	return results
}
