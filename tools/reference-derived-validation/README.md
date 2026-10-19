# Data Derivation D15：導出値を用いる業務Validationの評価順と失敗境界 Reference（#1729）

本ディレクトリは **合成入力を読取り専用で検査する小さな参照** であり、TEIの正式なValidation評価器、式言語、ライフサイクル実行器、Transaction管理ではありません。実装主体は `derivedvalidation.go`、受入は `derivedvalidation_test.go`、起動点は `cmd/tei-derived-validation/` です。`正本`を書き換える経路はありません。

## 既存責務との接続

- **D1 #1637**：純粋計算から渡された導出値と状態を受け取ります。導出の入力未取得・実行失敗・NULL方針未定義を、`Validation not satisfied` へ縮約しません。
- **D4 #1692 / #988 / #1121**：受理済みの規則・サービスと型付き出力だけを参照します。Rule本文の第二正本、数式の複製、同名属性参照、独立Plugin登録は作りません。
- **#1708 / #1174 / #1129**：読取りSnapshot・競合判定・Commit・副作用はこれらのOwnerへ委譲します。本参照は保存Actionを実行しません。
- **D9 #1706 / D13 #1724 / D5 #1693**：集計の意味、評価不能と部分結果、認可はこれらのIssueが所有します。

## 評価順（採用前）

入力取得 → 型付き導出 → 業務Validation → Action可否、の順に、次の状態を相互に区別します。

| 条件 | 期待 | 結果コード |
| --- | --- | --- |
| 認可されていない | 業務制約を評価しない | `AUTHORIZATION_DENIED` |
| 純粋規則への循環依存 | 実行前に拒否 | `DEPENDENCY_CYCLE` |
| 同名属性で参照（明示IDでない） | 参照を拒否 | `AMBIGUOUS_REFERENCE` |
| 導出の入力未取得 | not satisfied と区別 | `DERIVATION_INPUT_UNAVAILABLE` |
| 導出の実行失敗 | not satisfied と区別 | `DERIVATION_EXECUTION_FAILURE` |
| 導出がNULLで方針未定義 | 0やnot satisfiedへ変換しない | `DERIVATION_NULL_UNSPECIFIED` |
| 上限など他入力が未取得 | 評価しない | `LIMIT_UNAVAILABLE` |
| 導出成功・金額>上限 | 正常な不成立 | `VALIDATION_NOT_SATISFIED` |
| 導出成功・金額≤上限（保存時相当の合成入力） | Validation成立のみ（`saveGuaranteed=false`）。実保存は未観測 | `VALIDATION_SATISFIED` |
| 導出成功・金額≤上限（Preview） | 成立だが保存時点の確定ではない（`saveGuaranteed=false`） | `VALIDATION_SATISFIED` |

## 合成Fixture

受注O1：数量2、単価300、導出受注金額600。与信上限は受理済み型付き入力。

| 条件 | 本参照の結果 |
| --- | --- |
| 上限500・金額600 | `VALIDATION_NOT_SATISFIED` |
| 上限700・金額600（保存時相当の合成入力） | `VALIDATION_SATISFIED`（`saveGuaranteed=false`、保存未実施） |
| 単価NULLで方針未定義 | `DERIVATION_NULL_UNSPECIFIED`（0やnot satisfiedにしない） |
| 単価未取得 | `DERIVATION_INPUT_UNAVAILABLE` |
| 一時編集で数量1→金額300のPreview | `VALIDATION_SATISFIED`（`saveGuaranteed=false`。保存許可やCommit成立を意味しない） |
| Preview時の上限700、保存直前に500へ変更 | 古いPreview結果を根拠に保存しない。保存Actionが現行版で再評価する（本参照は`saveGuaranteed=false`で示す） |
| `amount`という同名の別Semantic属性 | `AMBIGUOUS_REFERENCE`（明示ID参照以外は拒否） |
| Validationが導出値を参照し循環依存 | `DEPENDENCY_CYCLE` |

## 実行方法

```sh
go run ./tools/reference-derived-validation/cmd/tei-derived-validation -input request.json
```

`-input -`（既定）は標準入力から読みます。入力例：

```json
{
  "referenceByID": true,
  "authorization": "allowed",
  "derivation": { "state": "present", "amount": "600" },
  "limit": { "state": "present", "value": "500" },
  "phase": "save"
}
```

`derivation.state` は `present`、`null`、`input-unavailable`、`execution-failure`、`phase` は `preview`、`save` です。`phase=save` は**合成入力側の評価局面の指定**であり、HostやOwner Actionが保存・Commitを実行した証拠ではありません。`saveGuaranteed` は互換用の出力項目として保持しますが、本参照は実保存を観測できないため常に `false` です。

## 限界

- 合成入力に対する静的な状態判定です。実Provider、実DB、実Actionの保存・Commitの証拠ではありません。
- `-race` は cgo を伴わない可搬Goでは実行していません。
- M2（既存Validation APIへの最小Adapter）とM3（実CRUDでのPreviewと保存時再判定）は本PRの範囲外です。
