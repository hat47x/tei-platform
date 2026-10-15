# D12 評価文脈の明示入力・時計固定・決定的再評価 Reference（#1720）

**状態: 合成Fixtureの読取り専用Reference。Invocation Context、認可、暦Provider、キャッシュ版ではない。**

このパッケージは、属性列以外の評価文脈（基準時刻・タイムゾーン・Locale）に依存する導出を、隠れた `time.Now()` やマシン既定Localeへ依存させず、**外部から供給した明示値の純粋評価**として再現可能にする最小契約を固定する。trusted Invocation Context の仕組みは #1118、演算子は #988、Lowering は #1121、キャッシュ版は #1641、複数実装の適合は #1694 が所有する。

## 入力の三区分

1. **業務入力**: `quantity`・`unit_price` など規則の意味に属する値。
2. **評価引数**: `as-of` 日時、営業日暦、Locale など、結果へ影響する必要最小限の文脈値。
3. **信頼済みContext**: principal・tenant・権限など、Hostだけが供給する値。

ユーザーが渡した `tenant` や `clock` 文字列を認可や評価の根拠にしない。`Requirement` は受理済み規則から読み、影響しない環境値まで全規則へ宣言を強制しない。

## 時計の固定

- `Clock` は1リクエスト分の基準時刻をHostが供給する。`FixedClock` は一つの瞬間に固定し、`UnfixedClock` は供給が無いことを表して**OS時刻へfallbackせず** fail-closed する。
- `ElapsedDays` はHost時計を**一度だけ**読み、リクエスト基準時刻を固定する。日付境界を跨ぐ瞬間でも同じリクエストの全行が同じ評価日を使う。
- `CalendarDate` は明示の `*time.Location` で暦日を求め、nil のタイムゾーンを拒否する。`time.Now` やProcess Environmentの既定値へ落ちない。

## 合成Fixture（elapsed-days）

起算日 2026-10-12、基準時刻 2026-10-13 00:30 JST（= 2026-10-12 15:30 UTC）。

| 入力 | 期待 | 試験 |
| --- | --- | --- |
| 評価日 2026-10-13 JST | 1 | `TestD12ElapsedDaysIsFixedByRequestClock` |
| 評価日 2026-10-14 JST | 2 | 同上 |
| 同一起算日・同じ固定時計で2回 | 同じ結果 | 同上 |
| 同じ瞬間をUTCで評価 | 0（JSTと異なる） | `TestD12TimezoneChangesTheDateBoundary` |
| 時刻未供給 / Timezone未指定 | fail-closed | `TestD12MissingContextFailsClosed` |
| 一覧の全行 | 同一のRequest基準時刻 | `TestD12OneRequestInstantForEveryRow` |
| `quantity=2, price=300` の純粋規則 | 600（時計・Locale宣言不要） | `TestD12PureRuleNeedsNoContext` |
| 宣言したLocale未供給 | 拒否、不要な規則へは強制しない | `TestD12LocaleOnlyRequiredWhenDeclared` |
| 起算日より前の評価日 | 拒否 | `TestD12EvaluationDateBeforeStartIsRejected` |

## 焦点試験

```sh
gofmt -l reference/derivation-context/*.go
go test -count=1 ./reference/derivation-context
go vet ./reference/derivation-context
```

## 未実装・依存

- `Clock` の供給元、principal/tenant/権限の信頼境界は #1118 が所有する。このReferenceはContextの値を受け取るだけで、Host identity や認可を検証も発行もしない。
- タイムゾーン/営業日暦/Locale/照合順序の正式な型・版・出所、ネットワーク由来の暦・レート・最新価格の取得は未成立。表示時の言語切替は #1453/#1551 が所有し、ここで複製しない。
- SQL/Go/Webの暗黙Clock/Locale差、Provider固有の実装差は未検証であり、#1694 の適合未証明として扱う。AI生成プラグインが暗黙の `time.Now` や外部HTTPを使う場合、その実装を純粋かつ再現可能と無条件認定しない。
- 合成Fixtureのみ。実ProviderやX-TEA #940の実動作との同等性は未検証。

実行の有無、exact SHA、成功／失敗／未検証を Issue #1720 と対象PRへ記録する。
