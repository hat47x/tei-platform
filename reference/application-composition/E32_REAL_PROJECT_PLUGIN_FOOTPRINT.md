# E32：案件固有Go＋Lit試作の実PR差分を、構成採用費用と分けて観測する

- **管理**：[Issue #1016](https://github.com/hat47x/tei-platform/issues/1016)。
- **参照**：案件固有拡張の[PR #1567](https://github.com/hat47x/tei-platform/pull/1567)。**固定HEAD** `be193266f1556e090b59cf2c237a23c0ca60236e`、比較基点 `aa63f482b35e9d169f150aaaac801d56db288d72`。2026-10-13にGitHubのPR本体と`pulls/1567/files`の変更一覧を照合。確認時のPRは**Draft・未統合**。
- **再現**：[E32の観測メタデータ](e32_project_plugin_pr1567_snapshot.json)／[集計器](e32_plugin_pr_footprint.mjs)／[Node回帰試験](e32_plugin_pr_footprint.test.mjs)。
- **既存機構**：[E26の既存Source記述量](E26_COMPOSITION_COST_BASELINE.md)、[E27の残余選択](E27_RESIDUAL_SELECTION_AUDIT.md)、[E28の申告と未検証の区別](E28_WORKFLOW_MEASUREMENT.md)、[E31のTEI・SUI改訂比較](E31_PAIRED_SOURCE_BASELINE.md)を使う。新しい構成Profileや第二のResolver、Lockは実装しない。

## 固定PRの変更内容を計測した結果

GitHubが返した**変更ファイル20件はすべて追加**であり、追加行1,267、削除行0だった。パス、Git blob SHA、増減行数、位置付けを個別に保存した。下の分類はレビュー用に人間が付与したもので、GitHubが正規の責務階層を決定したわけではない。

| 参照用分類 | 変更ファイル | 追加行 | 内容 |
| --- | ---: | ---: | --- |
| 案件固有のGo補助処理 | 2 | 84 | `plugins/reference-project-customer-advice/`の実装と単体試験 |
| 診断用Go Host | 4 | 423 | HTTP Hostと受注明細読取りの実装・単体試験 |
| Litの案件UI | 6 | 251 | Custom Element、通信アダプター、HTMLとNode.js試験 |
| 配布物・ライセンス | 2 | 52 | Lit 3.3.3の固定バンドルとBSD-3-Clauseライセンス |
| 検証・自動化 | 4 | 396 | Go/Nodeの専用CI、実Chrome実行、二つのGoプロセスの結合確認 |
| 説明文 | 2 | 61 | 案件Go補助と診断WebのREADME |
| **合計** | **20** | **1,267** | **GitHubの当該PR差分** |

数字は**改訂間の差分行数**であって、人間が手で1,267行書いたことでも、生成AIが1,267行生成したことでも、正式なTEI Applicationに1,267行が必要なことでもない。特にLitの同梱配布物はminify済みであり、52行を人間の実装工数と比較してはいけない。試験・CI・説明文は案件機能本体と分けて示した。

この20ファイルはすべて新規追加だった。したがって、**当該PRの差分には既存の`reference/workspace/`、標準Action、Application/Environment等の改修が含まれない**。一方、既存Applicationがまったく影響を受けないか、今後追加変更が不要かを証明するものではない。

## 実際に通っている接続と、まだ成立していない選択

固定HEADのGo、Lit、検証手順を読んで次を区別した。

| 項目 | 観測された動作・実装 | 製品構成としての限界 |
| --- | --- | --- |
| 案件固有Go処理 | `customeradvice.Evaluate`を診断Hostが直接importして実行する | 既存TEIの正規Service Contract・Provider選択から呼び出していない |
| 案件UI | Lit Custom Elementが同一Originの`POST /project/advice`を呼び、提案を表示する | UI Kitの正式な選択・Project Extensionとしての取込みは未実証 |
| 受注明細の実参照 | 診断Hostが任意の明示ループバックAPIへ`GET /scenario/read`を呼び、識別メタデータだけ渡す | **読取り専用の別プロセス接続**であり、受注ActionのHookや業務契約の拡張ではない |
| 確認試験 | Go/Node局所結合と実Chromeを、PRの専用CIで実行した。最新固定HEADに対する専用CIは成功 | 本番の本人認証、現在の権限、Provider登録、受注可否、実配備にはならない |
| リリース | Lit配布バンドルの既知Git blob SHAを明示、ライセンスを同梱 | TEI正式のArtifact/Build/Runtime受理と同一の証拠ではない |

当該固定HEADの[Go＋Node CI #38321641566](https://github.com/hat47x/tei-platform/actions/runs/38321641566)、[実Chrome CI #38321641568](https://github.com/hat47x/tei-platform/actions/runs/38321641568)、[PR全体CI #38321641563](https://github.com/hat47x/tei-platform/actions/runs/38321641563)はGitHubでいずれも**success**だった。ただし「実Chromeで補助提案と受注明細参照を表示できた」という成功を、「TEI Runtimeの正式なProviderとActionが接続された」という採用認定へ読み替えない。

## 次の採用判断に必要な実際の差分

現在の試作を製品機構へ採用するには、少なくとも次の確認が残る。

1. **利用先を選ぶ**：受注明細Applicationのどの既存Sourceを`Extend`するか、案件内の独立Pluginとして`Compose`するか。直接importでも十分な案件について新しい共通Profileを義務化しない。
2. **契約を確認する**：`customeradvice.Evaluate`を正式なService Contractへ割り当てるなら、#951の既存Provider選択・Scope・権限・未解決時の拒否に乗せる。現在の直接importを正式なProvider登録と誤認しない。
3. **UIとTargetを選ぶ**：LitをWeb UI Kitとして採用するか、Project Extensionとして使うか。#997のKit選択を重ねて書かず、実Targetでの受入と更新時の競合を確認する。
4. **現在のAuthorityと業務意味を守る**：助言のみの案件ロジックを受注可否の正式なValidationへ昇格しない。必要になった場合だけAccepted meaning側で変更を受理し、Runtime Authority・書込み・本人委譲を別に検証する。
5. **労務を実際に測る**：作業者の手動接続、生成AIへの質問、再検証の回数は、Gitの差分や27コミットというPRメタデータからは復元できない。[E28](E28_WORKFLOW_MEASUREMENT.md)の`claimedCoveredCounts`を使う際も、未計測の種類は`null`とし、実出典・作業ログがない値を補間しない。

このE32は**完成した二方式の生産性比較ではない**。既存Sourceによる方式と追加Profile方式の対照実験も行っていないため、改善率・追加質問削減率は算出しない。

## 検証可能性と制限

```sh
node --test reference/application-composition/e32_plugin_pr_footprint.test.mjs
node reference/application-composition/e32_plugin_pr_footprint.mjs
```

試験は、固定PRの**観測メタデータが自分自身と矛盾しないこと**、重複・不正パス・不正Git blob・責務分類の食い違いを拒否すること、人数や業務受理の値を勝手に生成しないことを確認する。CIはGitHubのPRを毎回読み直すものではなく、当時の申告メタデータの真正性・最新性を独立認証しない。新しいCommitでPRが更新されたら、その版を固定して**別の観測**として再評価する必要がある。

GitHubの実PR出典を読み取ったことと、E32のCIがその出典を認証したことは別である。PRの後続改訂、実マージ、配布、Accepted意味、Runtime認可についての推論には利用しない。
