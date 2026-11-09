# 採番の契約の確認（2026-11-07）

## 対象

- 実装: `internal/numbering/numbering.go`
- 試験: `internal/numbering/numbering_test.go`
- 対象版: `main` の `ec2f7d754` から分岐した`feat/numbering-contract-20261107`
- 環境: Windows、Go `1.27.0`

## 実行

```powershell
go test ./internal/numbering -count=1 -v
```

## 観測

- `TestNextIssuesInOrder` 成功。`ORD-00001`・`ORD-00002`と1ずつ進み、状態が返る。
- `TestOverflowStopsByDefault` 成功。桁上限では既定で停止し、1へ戻らない。
- `TestOverflowWrapIsChosen` 成功。`wrap`を宣言した場合だけ1へ戻る。
- `TestCheckDigitIsOneCharacter` 成功。`07`の検査数字は`9`（`079`）、`00`は`000`。
- `TestFormatRefusesAValueWiderThanTheDigits` 成功。桁を超える値は拒否。
- `TestContractWithoutAUsableWidthIsRefused` 成功。桁0・10、未知の検査数字・未知の桁あふれ規則は拒否。
- `TestNegativeCounterIsRefused` 成功。負のカウンタは拒否。
- `TestNextValueReservesWithoutFormatting` 成功。書式なしで値を予約でき、停止規則も同じ。
- パッケージ全体は`ok`。

## 読み取れること

- 桁あふれの既定は**停止**である。X-TEAは常に1へ戻し（`Session.java` L2061–2100）、使用済みの番号を再発行し得た。
- 検査数字は常に1文字である。X-TEAは`10 - sum % 10`をそのまま連結したため（同 L2053）、余りが0のとき`10`の2文字になり、桁数が宣言より1文字増えていた。
- 一意性・連続性・桁あふれ・欠番の扱いは、契約として宣言される。

## 読み取れないこと

- 採番表（Data）との結合と、取引内での発行・取消。参照実装は状態を入力に取り、状態を返すのみ。
- 同時実行時の競合（X-TEAの`UPDCOUNTER`の楽観的更新と、その結果を検査しない点）。
- 実DB・旧Runtimeでの採番の実測。

## 位置付け

参照実装の試験記録であり、製品の受入や本番適格性を示しません。`RQ-C3`の採番の契約は参照実装で確認済み、Data・取引との結合は未実装です。
