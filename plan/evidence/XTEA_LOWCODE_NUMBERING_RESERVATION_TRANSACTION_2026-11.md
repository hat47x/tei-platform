# 採番の予約と取引の結合の確認（2026-11）

## 対象

- 実装: `internal/numbering/transaction.go`（`Transaction`・`UnitOfWork`・`IssuedReservation`・`ReserveInTransaction`）
- 試験: `internal/numbering/transaction_test.go`
- 対象版: `main` の `a8458f7fc` から分岐した`feat/numbering-reservation-unitofwork-20261108`
- 環境: Windows、Go `1.27.0`

## 実行

```powershell
go test ./internal/numbering -run 'TestReserveInTransaction' -count=1 -v
```

## 観測

- `TestReserveInTransactionCommitsBoth` 成功。`A-0001`を発行し、Commitで取引が確定、カウンタは1のまま。
- `TestReserveInTransactionRollbackReturnsTheCounter` 成功。Rollbackでカウンタが0に戻り、次の発行も`0001`（欠番なし）。
- `TestReserveInTransactionCommitFailureReturnsTheCounter` 成功。取引のCommitが失敗するとカウンタは0に戻り、呼出し側はCommit失敗だけを受け取る。
- `TestReserveInTransactionRefusesAMissingUnitOfWork` 成功。UnitOfWork不在は拒否。
- `TestReserveInTransactionRollsBackAFailedBegin` 成功。Begin失敗と、予約失敗時のロールバック（取引を開いたままにしない）を確認。
- パッケージ全体は`ok`。

## 読み取れること

- 採番の発行が、呼出し側の取引と同じ境界に入る。**両方Commit、または両方戻す**。
- 取引Commitが失敗した場合、番号は消費されない（カウンタが戻る）。
- 予約に失敗した場合、開いた取引を閉じる。

## 読み取れないこと

- `command.Invoker`の`UnitOfWork`実装との共通化（今は番号側に独立した小さな契約を持つ）。
- 実DBの取引（`CounterStore`は参照用の記憶域）。
- 複数規則を1つの表に置く場合の行識別。

## 位置付け

参照実装の試験記録であり、製品の受入や本番適格性を示しません。予約と取引の結合は参照実装で確認済み、実DBの取引とInvokerのUoWとの共通化は未実装です。
