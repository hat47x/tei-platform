# RQ-C3の確認方法の観測（2026-11）

## 対象

- 要求: [X-TEA知見の受容要件](../../requirements/XTEA_LOWCODE_RECEPTION_REQUIREMENTS_2026-11.md)のRQ-C3「確認方法」
- 試験: `internal/standardbusiness/rq_c3_verification_test.go`
- 対象版: `main` の `f04c51049` から分岐した`test/rq-c3-verification-20261110`
- 環境: Windows、Go `1.27.0`

## 要求文の「確認方法」と、この試験の対応

RQ-C3は次の3つを確認するよう求めています。この試験は、要求の言葉をそのまま試験名へ写しています。

| 要求文 | 試験 |
| --- | --- |
| 適用期間の境界（改正日、年度替わり）を含む入力で、同じ評価時点から同じ結果を返す | `TestRQC3TaxRateChangesOnTheBoundary`・`TestRQC3CurrencyRateFollowsTheDeclaredPeriod` |
| 案件固有の暦・税率を、Coreの変更なしに差し替えられる | `TestRQC3TaxRateIsSwappedWithoutChangingCore`・`TestRQC3CalendarSwapsWithoutChangingCore`・`TestRQC3HolidaysChangeTheBusinessDayCount` |
| 採番の一意性・連続性・欠番の扱いを、契約として試験する | `TestRQC3NumberingIsUniqueAndContiguous`・`TestRQC3NumberingContractRefusesOverflow` |

## 観測

- `TestRQC3TaxRateChangesOnTheBoundary` 成功。改正日の前日`0.08`・当日`0.10`。同じ評価時点を2回問い合わせても同じ値。
- `TestRQC3TaxRateIsSwappedWithoutChangingCore` 成功。行の率を`0.12`へ変えるだけで、同じ宣言が新しい値を読む（Coreの変更なし）。
- `TestRQC3CalendarSwapsWithoutChangingCore` 成功。年度開始4月と1月の暦が、`2026-03-31`で**異なる年度**を返す。境界日`2026-04-01`では両者が一致する（どちらも新しい年度の初日）。
- `TestRQC3HolidaysChangeTheBusinessDayCount` 成功。休日を1日足すと営業日数が変わる。
- `TestRQC3NumberingIsUniqueAndContiguous` 成功。3回の発行が重複せず、`Release`した番号は次の呼出しで**同じ番号**になる（欠番なし）。
- `TestRQC3NumberingContractRefusesOverflow` 成功。桁の上限を超える発行は拒否される。
- `TestRQC3CurrencyRateFollowsTheDeclaredPeriod` 成功。月ごとに異なる率を読み、行の無い月は0ではなく報告になる。

## 試験を書く過程で分かったこと

自分が最初に書いた期待のうち、**3つが誤り**でした。いずれも実装ではなく、私の理解の誤りです。

1. **年度開始月の差** — `2026-03-31`で4月開始と1月開始が同じ年度を返すと書きましたが、実際は`2026`と`2027`で異なります。境界日`2026-04-01`の方が一致するので、比較する日を取り違えていました。
2. **休日の位置** — `DaysBetween`は開始日から**終了日を含めず**に数えるため、終了日を休日にすると数が変わりません。休日を区間の内側へ移しました。
3. **桁あふれの既定** — `Contract.Overflow`に既定値が入っていると書きましたが、既定は発行時に適用されます。構造体は空のままで、試験は**振る舞い**（上限超過の拒否）を見るよう直しました。

いずれも「実装が要求を満たしているか」ではなく「私の期待が実装と合っているか」の誤りでした。要求の確認方法が実際に観測できることは、7件の試験で示せています。

## 読み取れないこと

- 要求文の「同じ評価時点」を時刻まで含めて固定すること（今は日単位）。
- 税・為替の端数処理の全組合せ（各パッケージの試験が持つ）。
- 実DB・旧Runtimeでの観測（#940）。
- 要求の最終採否（現在は要求候補・仮承認）。

## 位置付け

参照実装の試験記録であり、製品の受入や本番適格性を示しません。RQ-C3の「確認方法」が参照実装で観測できることを示すもので、要求の採用判断ではありません。

