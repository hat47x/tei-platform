# 期間が名指す能力の効果の検査の確認（2026-11）

## 対象

- 実装: `internal/validation/period_capability_effect.go`、`internal/validation/validation.go`の呼出し
- 試験: `internal/validation/period_capability_effect_test.go`
- 対象版: `main` の `be814895e` から分岐した`feat/period-capability-effect-20261111`
- 環境: Windows、Go `1.27.0`

## 見つけた不足

期間は`close-requires <capability>`と`correction via <capability>`を宣言でき、`internal/validation`は**その能力が存在するか**だけを検査していました。能力が**別のDataを変更する**ものでも通っていました。

その状態では、訂正規則（`COMMAND_PERIOD_CORRECTION_REQUIRED`）が逆に働きます。期間のDataを変更できる能力を通さず、期間を変更できない別Data向けの能力を「訂正」として受け入れます。**宣言の誤りが、実行時の判断を反転させます。**

## 変更

`PERIOD_CAPABILITY_EFFECTS_OTHER_DATA`を追加しました。期間が名指す能力の効果Dataが、期間のDataと違えば報告します。

| 場合 | 扱い |
| --- | --- |
| 能力が別のDataを変更する | **報告**（本検査） |
| 能力が同じDataを変更する | 通す |
| 能力が効果を宣言していない | **何も言わない** |
| 能力が存在しない | 既存の`PERIOD_UNKNOWN_CAPABILITY`が報告 |

## 設計上の判断

効果を宣言していない能力を**拒否しない**ことにしました。当初は「効果なし＝変更できないので訂正能力として不適」として拒否する実装にしましたが、既存の期間フィクスチャ（`close`・`correct`が効果を宣言していない）が妥当な宣言として通っていたためです。

効果を宣言していない能力について、モデルは「どのDataを変えるか」を何も述べていません。そこから「別Dataを変える」と推論するのは、**モデルが述べていない規則を発明する**ことになります。矛盾がある場合（別Dataを明示している）だけを報告します。

## 観測

- `TestPeriodCapabilityEffectsData` 成功。`sales.other`を変更する`sales.other.close`を`close-requires`に名指すと`EFFECTS_OTHER_DATA`が報告される。
- `TestPeriodCapabilityEffectMatchingDataIsAccepted` 成功。`sales.order`を変更する能力は報告されない。
- 既存の期間試験（効果を宣言しない`close`・`correct`）はすべて通る。
- `internal/validation`全体は`ok`。例と`reference/workspace`の`validate`は`ok`（既存警告のみ）。
- 診断コードの網羅ラチェット試験（新しいコードは試験必須）も通る。

## 読み取れること

- 期間が名指す能力が**別のDataを変更する**場合、宣言の誤りとして報告される。
- 訂正規則が逆転する余地（別Data向けの能力を訂正として受け入れる）が塞がれた。
- 効果を宣言していない能力は、モデルが何も述べていない以上、判定しない。

## 読み取れないこと

- **効果を宣言していない能力が訂正能力として妥当か**。モデルが効果を持たない能力を認めている以上、別の判断（効果の宣言を必須にする等）が要ります。
- 効果の**種類**（`update`・`create`・`delete`）と役割の対応（`close-requires`が`create`でもよいか）。
- 実Provider・実DB・旧Runtimeでの観測。

## 位置付け

参照実装の試験記録であり、製品の受入や本番適格性を示しません。期間が名指す能力の効果Dataの照合は参照実装で確認済み、効果の種類と役割の対応、効果宣言の必須化は未実装です。
