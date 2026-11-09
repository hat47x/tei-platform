# 採番の発行・取消の取引結合の確認（2026-11）

## 対象

- 実装: `internal/numbering/reservation.go`（`CounterStore`・`MemoryCounterStore`・`Reserve`・`Reservation`）
- 試験: `internal/numbering/reservation_test.go`
- 対象版: `main` の `2f22a8703` から分岐した`feat/numbering-transaction-state-20261108`
- 環境: Windows、Go `1.27.0`

## 実行

```powershell
go test ./internal/numbering -run 'TestReserve|TestRelease' -count=1 -v
```

## 観測

- `TestReserveIssuesAndKeepsTheCounter` 成功。`ORD-00001`を発行し、Commit後にカウンタが1、次の発行が`ORD-00002`。
- `TestReleaseReturnsTheCounter` 成功。`001`を発行してReleaseするとカウンタが戻り、次の発行も`001`。欠番を作らない。
- `TestReserveReportsAConcurrentChange` 成功。予約の間に別の呼出しが発行すると、先行のReleaseは`ErrNumberingConflict`で拒否され、後続の発行（`002`）を上書きしない。
- `TestReleaseAfterCommitIsRefused` 成功。Commit後のRelease・二重Commitは拒否される。
- `TestReserveRefusesAMissingStoreOrBadContract` 成功。Store不在・不正な契約は拒否される。
- パッケージ全体は`ok`。`internal/validation`・`cmd/tei`も`ok`。

## 読み取れること

- 採番の発行は、**予約（Reserve）→ Commit または Release**として取引に参加できる。
- 取り消すとカウンタが戻るので、失敗で欠番を残さない。X-TEAは例外を握りつぶして空文字を返し（`Session.java` L2004–2007）、カウンタが動いた後の失敗を説明できなかった。
- 予約の間に別の呼出しが発行した場合は、**競合として報告**し、後続の発行を壊さない。X-TEAの楽観的更新は結果を検査していなかった（同 L1995–2002）。

## 読み取れないこと

- `command.Invoker`の`UnitOfWork`との実際の結合（今は`CounterStore`を介した抽象で、取引実装は呼出し側）。
- 複数の採番規則を1つの表に置く場合の行識別（`ruleID`は呼出し側が決める文字列）。
- 実DB・旧Runtimeの実測。

## 位置付け

参照実装の試験記録であり、製品の受入や本番適格性を示しません。採番の発行・取消・競合検出は参照実装で確認済み、取引実装との結合と実DBは未確認です。
