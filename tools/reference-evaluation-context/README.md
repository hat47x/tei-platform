# Data Derivation D12：評価文脈の明示入力・時計固定 Reference（#1720）

本ディレクトリは **評価に影響する文脈依存を明示して再現可能にする最小契約を読取り専用で検査する小さな参照** であり、TEIの正式なtrusted Invocation Context、Authority、業務暦Provider、受入済みAPIではありません。実装主体は `evaluationcontext.go`、受入は `evaluationcontext_test.go`、起動点は `cmd/tei-evaluation-context/` です。`正本`を書き換える経路はありません。

設計の入手先は [EVALUATION_CONTEXT_DETERMINISM_2026-10](../../plan/design/EVALUATION_CONTEXT_DETERMINISM_2026-10.md)。本参照はそのM2相当の合成Fixtureです。

## 入力の三区分

| 区分 | 例 | 本参照での扱い |
| --- | --- | --- |
| Semantic入力 | `startDate`、`quantity`、`unitPrice` | 型付き業務入力 |
| 宣言された評価引数 | 評価日（as-of）、timezone、Locale | 必要なものだけ明示し、未宣言なら黙って決めない |
| Host が供給する信頼済み文脈 | principal、tenant、権限 | #1118 のOwnerへ委譲。文字列で渡された値を認可根拠にしない |

## 判定する意味（採用前）

| 条件 | 期待 | 結果コード |
| --- | --- | --- |
| `elapsed-days` に宣言された評価日 | 基準値から計算 | `OK_DAYS` |
| 同じ固定評価日の再評価 | 同一の意味結果 | `OK_DAYS`（同一値） |
| 業務暦・TZが影響するのに未宣言 | 一方を勝手に採用しない | `TIMEZONE_UNDECLARED` |
| 評価日未提供 | fail-closed、OS時刻へfallbackしない | `EVALUATION_DATE_MISSING` |
| 単純な数量×単価 | 時計宣言なしで評価 | `OK_AMOUNT` |
| 実現が一行ごとに時計参照 | 禁止、基準時刻から計算 | `NONDETERMINISTIC_REEVALUATION` |
| 実現が隠れた外部取得 | 純粋と認定しない | `HIDDEN_IO_IN_RULE` |
| 正体不明の規則 | 拒否 | `UNSUPPORTED_RULE` |

一行ごとの時計参照と隠れたI/Oは他の文脈整合の前提を壊すため、最初に判定します。

## 合成Fixture（受注明細）

規則 `elapsed-days(startDate, evaluationDate)`、起算日 2026-10-12。

| 操作 | 本参照の結果 |
| --- | --- |
| 評価日 2026-10-13 | `OK_DAYS`、`value=1` |
| 評価日 2026-10-14 | `OK_DAYS`、`value=2`（同じ固定評価日は2回実行しても不変） |
| 業務暦判断が必要でtimezone未指定 | `TIMEZONE_UNDECLARED`（2026-10-13 00:30 JST = 2026-10-12 15:30 UTC の境界を勝手に決めない） |
| `today` 要求なのに評価日未提供 | `EVALUATION_DATE_MISSING`、OS時刻にfallbackしない |
| `quantity=2, price=300` | `OK_AMOUNT`、`value=600`（宣言不要） |
| AI生成プラグインが暗黙 `time.Now`／外部HTTP | `NONDETERMINISTIC_REEVALUATION`／`HIDDEN_IO_IN_RULE` |

## 実行方法

```sh
go run ./tools/reference-evaluation-context/cmd/tei-evaluation-context -input request.json
```

`-input -`（既定）は標準入力から読みます。入力例：

```json
{
  "kind": "elapsed-days",
  "startDate": "2026-10-12",
  "evaluationDate": "2026-10-14",
  "evaluationDateProvided": true
}
```

## 責務と分界

- trusted Invocation Context は #1118、一般Operatorは #988、Loweringは #1121、キャッシュ版・再利用は #1641（D3）、複数実現は #1694（D6）、説明根拠は #1696（D7）、入力時点の整合は #1708（D11）が所有します。
- 汎用業務暦製品、複数時間軸の完成形、全Ruleへの時刻必須化、外部サービスの自動取得、独自認可エンジンはしません。`正本`を書き換えません。
- 生成AIのPlugin実装を、隠れた時計参照・隠れたI/Oがあるまま純粋かつ再現可能と認定しません。適合は #1694（D6）へ接続します。

## 限界

- 合成入力に対する静的な判定です。実Host・実Providerの時計/locale差を測定した証拠ではありません。Target固有の暗黙Clock／Locale差は適合未証明として #1694 へ引き渡します。
- 暦・祝日・レート・照合順序の検証は対象外です（本参照は日数計算と文脈分類のみ）。
- `-race` は cgo を伴わない可搬Goでは実行していません。
- M3（SQL/Goの同等性と #1641 の再利用条件への接続）は本PRの範囲外であり、未実証として残します。
