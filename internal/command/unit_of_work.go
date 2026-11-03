package command

import "context"

// Transactionは、Commandの実行を囲む取引である。
// Commitが成功した場合だけ、実行結果を確定する。
type Transaction interface {
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
}

// UnitOfWorkは、Commandの取引を開始する契約である。
// 実装はL4（Provider・接続）に置く。
type UnitOfWork interface {
	Begin(ctx context.Context) (Transaction, error)
}
