# REF-SPEC-01：Validation参照差の確認記録

この記録は、ApplicationとBusiness ProcedureのValidation参照差を調査した**根拠と未検証範囲**をまとめたものです。仕様と操作手順は[Business-system design Reference](../../reference/business-system-design/README.md)、研究上の評価目的は[技術リサーチ統合方針](../../plan/TECH_RESEARCH_INTEGRATION_2026-10.md)を正本とし、ここでは重複して定義しません。関連する[Draft PR #1040](https://github.com/hat47x/tei-platform/pull/1040)と[Issue #915](https://github.com/hat47x/tei-platform/issues/915)は、Validation比較の変更とbrownfield onboarding全体の研究を区別して扱います。

## 静的なSource照合（2026-10-09）

- 対象Git revision：797ce7e99851b0c5cd054bf0b93824e329653eaf
- 方法：同一revisionのApplication .tei、Semantic Validation .tei、Business Procedure JSON、Customer編集probeを個別取得し、対象宣言を読み取った。Workspace LoaderやGo checkerの実行結果ではない。
- 元Workspaceの書き換え：なし。

| 対象Source（reference/以下） | Git blob SHA |
| --- | --- |
| workspace/applications/order-entry/application.tei | 00c29ac0e557cb50fdcd13f2477d0b538a528c80 |
| workspace/semantic/sales/validation.tei | fd9e3ad6a81e3201ade8e987377137a3ba702d5f |
| business-system-design/order-entry-business-procedure.json | ac1980d90ddd702a362078e1d4dd8dd70f5bc1a2 |
| business-system-design/order-entry-customer-edit-conflict.json | 2786138549a4b3f227f06969f57bf2a6ba47a99f |

確認できた事実は、order_entryのApplication宣言にValidationが3件あり、BP-05の明示参照にはそのうち2件があることです。共通はsales.order_line.quantity_positiveとsales.order.lines.line_number_unique、Application側のみはsales.order_line.line_number_positiveです。Procedure側のみのIDはありません。**差があるだけで業務上の欠陥とは断定しません。**

Customer編集probeにはBP-02でread-writeを要求する入力がありましたが、この静的確認だけでは既存checkerが拒否診断を返したことを実証していません。

## 限定的なGo実行の経過

過去の隔離モジュールはGitHubのGo原文とblob SHAを一致させ、必要なTEI型や集計関数を**代替実装**して試験したものです。後から追加されたコード、実Workspace、現行PR全体の検証記録ではありません。次のPASSはそれぞれ**記載された当時の原文**だけに適用されます。

| 時点 | 対象と主な確認点 | 隔離実行で得た結果 |
| --- | --- | --- |
| 10/09 | 比較関数 blob c061d451：差分、一致、文脈・参照未解決 | Go 1.23.2、6テストPASS、vet PASS |
| 10/09 | 出典処理 blob a86ee7dc：一意な定義、曖昧性、指紋欠落 | 4テストPASS、vet PASS |
| 10/09 | 比較 blob 7f5c2ed7＋出典修正 blob f494a1f5：artifact path重複の反証 | 修正前FAILを再現、修正後7テストPASS、vet PASS |
| 10/10 | 出典処理 blob 8116f2bf：出典消失後に古い根拠を残さない | 修正前FAILを再現、修正後2テストPASS、vet PASS |
| 10/10 | 比較・出典の原文二ファイルを同じ隔離ビルドで使用 | 4テストPASS、vet PASS |
| 10/10 | 未解決理由の比較処理 blob cad187b3：side・ID・reason・重複排除 | 4トップレベルテスト（分類7例）PASS、vet PASS |
| 10/10 | Step IDガードを含む比較処理 blob dea1f479＋出典 blob 8116f2bf | 3トップレベルテスト（Step境界6例）PASS、vet PASS |

原文blobと操作条件の詳細は[Issue #915の2026-10-09〜10の実行コメント](https://github.com/hat47x/tei-platform/issues/915)およびこの文書のGit履歴から辿れます。検証対象を限定した結果であり、Repository全体のPASSを意味しません。

## 設計上の反証と実装への反映

| 失敗し得るケース | 反映先 | 判断 |
| --- | --- | --- |
| Validationの未定義・重複定義や片側の宣言重複 | validation_scope_review.go と回帰テスト | unresolved-referenceに理由を残し、集合差を確定しない |
| Step IDの空欄・重複、Application文脈が不明 | 同上 | unresolved-contextに保留する |
| 共通Validationの出典を表示できない | validation_scope_review.go / validation_scope_review_sources.go | sharedValidationsも追跡する |
| 同一artifact pathが何度も現れる | validation_scope_review_sources.go | 2回目以降は曖昧性を解除しない |
| Source Indexの再照合後に古い出典が残る | 同上 | sourceEvidence / unresolvedSourceを毎回再生成する |
| 未解決行が正常な比較行の根拠まで失わせる | validation_scope_review_test.go（72f4b81） | 正常な比較行は独立して追跡する |
| Validationの出典artifactが欠落している | 同テスト（0b22b91） | 有効なApplication出典だけは残す |
| 空または空白のみの出典パス・指紋 | 出典処理（7ccc6b5、96c9d6d）とテスト（67585fe、3df3445） | 不正な根拠として未解決にする |

出典の重複・空白への防御は、補助関数へ不正なIndexを渡す負例です。通常のSource Index生成経路で確認された製品障害とは記録しません。回帰テスト関数24件はPRに存在しますが、**現行Repositoryでの一括実行結果は未取得**です。

## 統合判断と未検証事項（2026-10-11時点）

**未取得の証拠：** 実Repositoryのclean checkoutにおける最新headのGoテスト、実Workspace/Source Indexを使う結合確認、既存blocking diagnosticとの動作互換、full tools/repository-health-check.shの全項目PASS、実行前後のclean状態、設計者によるA/Bレビュー工数の実測。

CIは停止中です。手動検証を省略してよいわけではありません。以前の隔離Go試験やコードの静的照合は、最新headの統合資格とみなしません。mainとPR headが動いた場合は再資格確認が必要です。**Draft PRをmainへ統合しない停止条件を維持します。**

この文書は時点付きの監査記録です。現在のPR head・main SHA・Draft状態は[PR #1040](https://github.com/hat47x/tei-platform/pull/1040)で確認してください。Gitのコミット履歴とIssueコメントに残る詳細な実行経過を、ここへ重複して追記しません。
