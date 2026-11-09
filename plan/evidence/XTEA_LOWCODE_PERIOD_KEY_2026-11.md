# 期間キーの導出の確認（2026-11-07）

## 対象

- 実装: `internal/businesscalendar/period_key.go`
- 試験: `internal/businesscalendar/period_key_test.go`
- 対象版: `feat/period-key-derivation-20261107`のhead（`main` = `0b36094a8`から分岐）
- 環境: Windows、Go `1.27.0`

## 実行

```powershell
go test ./internal/businesscalendar -count=1 -v
```

## 観測

- `TestPeriodKeyForFollowsTheDeclaredStart` 成功。4月始まりの暦で、`2026-04-15`は年度2027・月順序1、`2027-03-15`は年度2027・月順序12。キーの表記は`2027-12`。
- `TestPeriodKeyForJanuaryStart` 成功。1月始まりでは月順序が暦月と同じ。
- `TestPeriodStartInvertsTheKey` 成功。期間開始日から同じキーが戻る（3例）。
- `TestPeriodStartRefusesAnImpossibleSequence` 成功。順序0・13・-1は拒否され、別の年へ回り込まない。
- `TestPeriodKeyDoesNotShiftAcrossAClosingDay` 成功。開始月の末日は月順序1のままで、翌日が順序2になる。
- パッケージ全体は`ok`（18件）。

## 読み取れること

- 期間キーは、宣言された開始月から導く。設定値ではない。
- 期間キーから期間開始日を戻せるので、閉めの記録を範囲へ戻せる。
- X-TEAの「締め日を過ぎた日付を翌月へずらす」規則（`getMSeqOfDate` L2476–2484）は引き継がない。日付はその日付が名指す月に属する。

## 読み取れないこと

- 月内の締め（20日締めなど）を期間にした場合の扱い。独立した期間として宣言する設計だが、実装はしていない。
- 期間の状態（`open`・`closing`・`closed`）との結び付き。キーの導出は状態を見ない。
- 実DB・旧Runtimeでの締めの実測。

## 位置付け

参照実装の試験記録であり、製品の受入や本番適格性を示しません。`RQ-C3`の期間キー導出は参照実装で確認済み、月内締めの期間化と状態との結合は未実装です。

