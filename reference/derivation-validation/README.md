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

## 既存Canonical Validationへの限定接続（M2の部分証拠）

`canonical_integration_test.go` は、**既存D1の `derivationref.Multiply` が返した値を、既存 `validationeval.Evaluate` の `Source.Number` へ明示的なSemantic Attribute IDで渡す**合成試験である。業務Validationは既存の `semantic.ValidationPositive` をそのまま使い、Validation式・計算Rule・認可機構の第二正本を作らない。

- 数量2×単価300→導出600→positive成立、数量0→導出0→positive不成立。
- 入力未取得、宣言NULL、演算子評価不能は、`not-satisfied` ではなく評価エラーとして扱う。試験専用Sourceが原因を区別する。
- 表示名が同じでも異なるSemantic Attribute IDの値を転用しない。
- **証拠の限界**：ここでの `positive` は合成の検証ルールであり、元モデルの意味受理や実案件の導出Bindingではない。既存 `validationeval` は `positive` と `relation-unique` のみを扱い、受注金額≦与信上限を表す既存Canonical Validation kindは確認できていない。そのため従来の上限比較の `Check` と、新たに接続したpositive試験の成立を同一視しない。D15 M2全体の完了も主張しない。
- 実Provider / Host Authority / Snapshot / Transaction / Commit / DB行効果 / X-TEA原典再現は試験していない。これらは #1693 / #1708 / #1129 / #1174 とOwner判断に委譲する。

`go test -count=1 ./reference/derivation-validation` が、この接続試験と従来の状態区分試験を同時に実行する。

## 未実装・依存

- Validationの式言語、ルールの第二正本、独立した依存グラフを作らない。循環検出は既存 D1 `ValidateGraph` を再利用する。
- 適用先解決は #1692、認可は #1693、入力Snapshot/競合は #1708、集合集計は #1706、一括評価不能と部分結果は #1724、OperatorとValidationの非同等性は #988 が所有する。
- Commit・rollback・外部副作用・通知は #1129 / #1174 が所有する。ここでは読取り評価の順序と状態区分だけを扱う。
- 一時Previewの成立を保存時点の確定判定に流用しない。M3の実CRUD/Action接続は未着手。
- 合成Fixtureのみ。実Provider、実DB保存Evidence、CLI/標準保守画面（#1127）、X-TEA #940の実動作との同等性は未検証。

実行の有無、exact SHA、成功／失敗／未検証を Issue #1729 と対象PRへ記録する。
