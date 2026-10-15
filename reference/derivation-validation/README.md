# D15 導出値を用いる業務Validationの評価順・失敗境界 Reference（#1729）

**状態: 合成Fixtureの読取り専用Reference。汎用Validation DSL、Commit、Rollback、副作用を実装しない。**

このパッケージは、受理済みの純粋な導出値を既存の業務Validation（例: 受注金額≦与信上限）へ供給するとき、**導出の成否**と**業務制約の成否**を別の段階として保つ。導出不能を `not satisfied` に縮約すると、利用者が正常な不成立と誤認する。

## 評価の順序

`入力取得 → 型付き導出 → 業務Validation → Action可否` の順を明示する。既存のValidation評価器を導出エンジンに置き換えない。副作用を含む更新のphase/commitは #1129 / #1174 が所有する。

`Check` は二つの状態を返す。

- `Derivation`: `value` / `null` / `input-unavailable` / `evaluation-failure` / `authorization-denied`。
- `Validation`: `satisfied` / `not-satisfied` / `not-evaluated`。

`Validation` は `Derivation=value` のときだけ確定する。未取得・評価失敗・宣言されたNULL・認可拒否は `not-evaluated` のままにし、`not-satisfied` へ潰さない。認可判断は #1693 が所有し、`denied` は入力として受け取るだけでここでは判断しない。与信上限のようなValidation入力が未取得なら fail-closed する。

## 適用先の同一性

`DerivedRef` は受理済み Rule ID と型付き Output ID だけで導出値を識別する。`Name` は表示用のラベルであり、解決に使わない。同名の別Semantic属性を名前で取り違えない。適用先解決そのものは #1692 が所有する。

## 一時Previewと保存時

`Decision` は結果・入力版・phase（`preview` / `commit`）を持つ。`RecheckAtCommit` は保存直前に入力版を照合し、`preview` の版が変わっていれば `ErrStalePreview` として拒否する。同じ版なら保存時点の結果で再評価し、previewの成立を保存許可の根拠にしない。読取りSnapshot/競合は #1708、Commit/rollback/通知は #1174 / #1129 が所有する。純粋な規則はCommit後に外部更新や通知を起動しない。

## 合成Fixture（受注O1）

数量2、単価300、導出金額600。与信上限は受理済みの型付き入力。

| 条件 | 期待 | 試験 |
| --- | --- | --- |
| 上限500・金額600 | value / not-satisfied | `TestD15SatisfiedAndNotSatisfied` |
| 上限700・金額600 | value / satisfied | 同上 |
| 単価未取得 | input-unavailable / not-evaluated（not-satisfiedではない） | `TestD15UnavailableIsNotNotSatisfied` |
| 演算子不整合 | evaluation-failure / not-evaluated | `TestD15ExecutionFailureDistinct` |
| 宣言されたNULL | null / not-evaluated | `TestD15DeclaredNullIsNotNotSatisfied` |
| 認可拒否 | authorization-denied / not-evaluated | `TestD15DeniedIsDistinct` |
| 上限未取得 | value / not-evaluated、fail-closed | `TestD15LimitUnavailableFailsClosed` |
| Preview上限700→保存直前500 | 古いPreviewを根拠にしない | `TestD15PreviewRecheckedAtCommit` |
| 同名`amount`の別属性 | 明示ID以外を拒否 | `TestD15OperandIdentityIgnoresName` |
| Validationが同じ純粋規則への循環を作る | 実行前に依存循環を拒否 | `TestD15ValidationCycleIsRejectedBeforeExecution` |

## 焦点試験

```sh
gofmt -l reference/derivation-validation/*.go
go test -count=1 ./reference/derivation-validation
go vet ./reference/derivation-validation
```

## 未実装・依存

- Validationの式言語、ルールの第二正本、独立した依存グラフを作らない。循環検出は既存 D1 `ValidateGraph` を再利用する。
- 適用先解決は #1692、認可は #1693、入力Snapshot/競合は #1708、集合集計は #1706、一括評価不能と部分結果は #1724、OperatorとValidationの非同等性は #988 が所有する。
- Commit・rollback・外部副作用・通知は #1129 / #1174 が所有する。ここでは読取り評価の順序と状態区分だけを扱う。
- 一時Previewの成立を保存時点の確定判定に流用しない。M3の実CRUD/Action接続は未着手。
- 合成Fixtureのみ。実Provider、実DB保存Evidence、CLI/標準保守画面（#1127）、X-TEA #940の実動作との同等性は未検証。

実行の有無、exact SHA、成功／失敗／未検証を Issue #1729 と対象PRへ記録する。
