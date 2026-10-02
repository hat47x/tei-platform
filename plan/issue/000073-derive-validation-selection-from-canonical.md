# ISSUE-000073: Applicationが評価するValidationの選択を、Target側の一覧ではなくCanonicalな宣言から導く

- 状態: Proposed — 評価待ち。現状の読解と選択肢の整理までを記録した段階で、実装は始めていない
- 提案日: 2026-09-30
- 性格: SSOTの欠落の解消 / Reference変更を伴う実装
- 主担当要求: RQ-003（主）/ RQ-013 / RQ-017 / RQ-021
- 関連Issue: [ISSUE-000072](000072-rule-change-hand-edit-count.md)（この欠落を計測する）、[ISSUE-000027](000027-ai-proposal-acceptance-boundary.md)（`APPLICATION_VALIDATION_SELECTION_NOT_CANONICAL`を最初に記録）、[ISSUE-000038](000038-business-lifecycle-semantics.md) / [ADR-0025](../adr/0025-transition-evaluation-and-conflict-result.md)（遷移ごとの評価条件。評価待ち）、[ISSUE-000014](000014-business-validation-reference.md)
- 価値との対応: [Product Value](../../product/value/TEI_PRODUCT_VALUE.md)「組織 1. 変更の単位コストを下げる」、具体例「組織 — 規則を書き写さない」の但し書き
- Gate impact: Gate B / Gate Cの判定は変えない（各exact revisionに固定されたhistorical Evidence）。ただし、Gate Cのarchitecture conformance tool（`tools/reference-first-web-target-architecture-evidence`）は、`reference/workspace`と`reference/world.json`がHandover Baselineの`R = e5231839…`からTarget build `T = 0ba87a4e…`の間で不変であることを要求するため、この変更を含む`main`では旧`R`に対して再実行できない

## この文書の役割

受注確定の前にどの業務Validationを評価するかが、現在はTarget側のコードに手で書かれています。このIssueは、その選択をCanonicalな宣言から導くようにし、規則の追加が正本の変更だけで画面・APIからの実行へ届く状態を作ります。

## 背景

`order_entry`では、Validationの**内容**は一つの正本（`reference/workspace/semantic/sales/validation.tei`）にあり、Application DriverとFirst Web Targetが同じ定義を読んで評価します。ここはSSOTが保たれています。

一方、**どのValidationを評価するか**は次のようになっています（revision `ee5b5b14`時点の読解）。

- TEIの構文には、Applicationが使うValidationを宣言する`uses validation <id>`があります。parserはこれを`Application.Validations`へ読み込み（`internal/syntax/parser.go`）、共有Validationは参照先の存在を確かめ（`internal/validation/business_validation_reference.go`）、Shared AcceptanceのPreviewは差分とimpactに含めます（`tools/internal/changeanalysis`）。
- しかし`order_entry`はこの宣言を持っていません。
- DriverとFirst Web Targetは、この宣言を読んでいません。First Web Targetは`tools/reference-first-web-target/command_scenario.go`の2か所に、評価するCanonical IDの一覧（`sales.order_line.quantity_positive`、`sales.order.lines.line_number_unique`）を直接書いています。なお、`tools/reference-first-web-target/projection.go`は`app.Validations`をProjectionのJSONへ写し、HTML画面にも「Applicationの宣言なし」と表示しますが、表示するだけで、評価には使っていません。`order_entry`が宣言を持つと、この表示が変わり、Web TargetのEvidence artifactと、その期待値を持つtestも変わり得ます。

つまり「どの規則を適用するか」という意味の一部が、正本の外（Target側のコード）に二つ目の置き場を持っています。原則P3の「一つの関心事に、暗黙の複数正本を作らない」に反する状態です。

## 防ぎたい失敗

- **追加した規則が効かない。** 正本へ新しいValidationを足し、Shared Acceptanceで受理し、Validationもgreenなのに、画面から確定すると新しい規則が評価されない。誰もそれに気づかない。
- **Targetごとに選択が食い違う。** 第二TargetやAPI、AIエージェントの経路が、それぞれ別の一覧を持ち、同じ受注に異なる規則が適用される。
- **一覧を黙って推測する。** 宣言が無いときに「namespace内の全Validation」を評価するなどの暗黙の規則をTarget側で作り、意図しない規則まで適用する。

## 守る原則

- 評価対象の選択はCanonicalな意味であり、Target・Driverの呼び出し側は選択を持たない（原則P1、P3、P9）。
- 宣言が無い、または解決できない場合は、推測で補わずfail closedにするか、unresolvedとして明示する（原則P6、P11）。
- 新しいCore primitiveを先に作らない。既存の`uses validation`で足りるかを先に確かめる（原則P16）。

## 選択肢

| 案 | 内容 | 利点 | 懸念 |
| --- | --- | --- | --- |
| A | Applicationの`uses validation`を正本とし、Driver / Targetはそこから評価対象を得る | 構文・parser・存在確認・差分解析が既にある。Core変更が小さい | 「受注確定のときだけ」等、操作ごとの違いを表せない |
| B | Capability（`sales.order.submit`）またはBusiness Actionに、実行前に満たすValidationを宣言する | 操作ごとの違いを表せる | 新しい宣言が必要。ISSUE-000038 / ADR-0025（評価待ち）と重なる |
| C | 使っているsemantic namespaceの全Validationを評価する | 宣言が不要 | 暗黙の規則。意図しない規則まで適用し、失敗「一覧を黙って推測する」に当たる |

**推奨: 初回はA。** 現在の`order_entry`では、受注確定が唯一のCommandであり、Aで意味を失いません。操作ごとの違いが必要になった時点で、ISSUE-000038 / ADR-0025の判断に合わせてBへ進みます。Cは採りません。

## 必要能力

1. `order_entry`のApplication sourceへ`uses validation`を宣言する（Shared Acceptance Boundaryを通すには、既存のapplication.teiへ`uses validation`を加えるoperationが要る。現在のoperationは`add-application`による新規Applicationの追加だけで、既存Applicationの編集を持たない）。
2. Driverの確定経路とFirst Web Targetが、`Application.Validations`から評価対象を得る。Target側のCanonical ID一覧を削除する。
3. 宣言が無いApplicationのCommandでは、評価対象を推測せず、fail closedまたは`APPLICATION_VALIDATION_SELECTION_NOT_DECLARED`等のunresolvedとして返す（どちらにするかは実装時にADRで決める）。
4. Shared AcceptanceのPreviewの`APPLICATION_VALIDATION_SELECTION_NOT_CANONICAL`を見直す。条件（そのValidationを宣言していないApplicationだけに返す）は既にその通りで、`analysis_test.go`が固定しています。変えるのは、実行時の選択が宣言から導かれることを前提にしたcode名とmessageです（ADR-0029）。

## 具体例

変更前:

```text
正本            validation sales.order_line.unit_price_positive を追加
Shared Acceptance  accept（Validation green）
Web Target      command_scenario.go の一覧に無いので評価しない
結果            単価0の明細を含む受注が確定する
```

変更後:

```text
正本            validation を追加し、order_entry に uses validation を一行追加
Web Target      Application.Validations から評価対象を得る
結果            単価0の明細はValidation not-satisfiedになる。Target側のコードは変えない
```

## 完了条件

- [ ] 案A / B / Cの判断をADRとして提案し、owner reviewで採用する（ADR案: [ADR-0029](../adr/0029-application-validation-selection-from-canonical-declaration.md)、2026-10-01に提案中。owner reviewは未実施）
- [ ] `order_entry`へ`uses validation`を宣言する変更を、Shared Acceptance Boundary経由で受理する（explicit human Acceptance decisionが必要）
- [ ] DriverとFirst Web Targetが宣言から評価対象を得るようにし、Target側のCanonical ID一覧を削除する
- [ ] 宣言が無い場合の振る舞い（fail closed / unresolved）をtestで固定する
- [ ] 既存のCommand scenario（success / Authorization denied / Validation not-satisfied / Execution Error / Observation）が変わらないことを確かめる
- [ ] 既存Applicationへ`uses validation`を加えるShared Acceptance operationを追加する（現在は`add-application`による新規追加だけ）
- [ ] Previewのunresolved codeの改名に合わせ、`boundary_test.go`、`analysis_test.go`、`tools/reference-shared-acceptance/README.md`を更新する
- [ ] 同じ`uses validation`の重複宣言を診断にするか決め、testで固定する。宣言順序の入れ替えが差分に現れるtestも足す
- [ ] Gate Cのarchitecture conformance toolが`R`から`T`のworkspace不変を要求することに対し、後継Handoverを立てるか判断を記録する
- [ ] Business ProcedureのBP-05が保持するValidation referenceとApplicationの宣言の関係（検査するか、独立した参照とするか）を決める
- [ ] `tools/repository-health-check.sh`のfull PASSを、exact headで記録してから統合する
- [ ] ISSUE-000072の最初の計測ケースを再計測し、C3（正本の外にある意味の手直し）が減ったことを記録する
- [ ] Product Valueの具体例「組織 — 規則を書き写さない」の但し書きを、結果に合わせて更新する

## 完了後に判断できること

- 第二Target・API・AIエージェントの経路で、同じ評価対象の選択を再利用できるか
- 操作ごとの違い（案B）が必要になる具体的なReferenceがあるか

## 非主張範囲

- 業務Lifecycle（ISSUE-000038）の遷移条件の設計を、このIssueでは決めません。
- Gate B / Gate Cの判定は、それぞれのexact revisionのEvidenceとして変えません。このIssueの変更後のworkspaceを、旧baselineと同じものとして読み替えません。
- 宣言からの導出は、Validationを「どの順で」「どのtransaction境界で」評価するかまでは決めません。
