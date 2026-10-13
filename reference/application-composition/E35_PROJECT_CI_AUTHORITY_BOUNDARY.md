# E35：案件Go＋Lit実CIの成功と製品Authorityの未成立を分ける

- **管理**：[Issue #1016](https://github.com/hat47x/tei-platform/issues/1016)。[E34](E34_PROJECT_BRIDGE_REVISION_REQUALIFICATION.md)でSourceとして実在を確認したGo Bridgeについて、実Go APIとの結合とLit実Chromeを**同一PR HEAD**で再確認した。
- **実例**：[案件Go＋LitのDraft PR #1567](https://github.com/hat47x/tei-platform/pull/1567)。固定HEAD `5fde80b0385b471f1c15d701f3c9f61761405c27`。2026-10-13にGitHub PR、commit workflow runs、各run jobs、SourceファイルのBlobを読み取った。
- **出典**：[固定スナップショット](e35_project_reference_ci_evidence.json)／[参照レビュー関数](e35_reference_ci_review.mjs)／[Node.js回帰試験](e35_reference_ci_review.test.mjs)。これらは参照専用で、別の成果物台帳・構成Profile・Lock・Resolver・実行Guardを作らない。

## E34の失敗から変わった実際の結果

前E34で確認したHead `596c522489dde0ad5184eba6e0220cd758f16636` の案件Go/Node専用CIは、Go単体試験まで成功しても二Goプロセスを接続する`smoke_order_entry.sh`で`reference process exited`となり失敗した。

本E35固定Headでは、診断APIビルド時のTarget Revisionを既存のビルド・実行両方へ整合させる変更、および起動失敗時の診断ログ提示が含まれた。これは**失敗から成功へ変わったGitHub検証事実**であり、今回のレビューPRが相手のDraftソースを変更した結果ではない。

| GitHub実行・正確なHEAD | 結論 | GitHub jobで個別にsuccessを確認した項目 |
| --- | --- | --- |
| [Go/Node局所CI #38378147159](https://github.com/hat47x/tei-platform/actions/runs/38378147159) | **success** | Go形式・単体試験、実`order_entry` APIへの**読取り専用Smoke**、UI通信とJavaScript試験 |
| [実Chrome CI #38378147239](https://github.com/hat47x/tei-platform/actions/runs/38378147239) | **success** | 正確な対象版とChrome、Go API＋案件Go Host＋Litの実ブラウザ操作 |
| [PR全体CI #38378147208](https://github.com/hat47x/tei-platform/actions/runs/38378147208) | **success** | 正規の全体健全性確認、PR候補HEADそのもののE1検証 |

すべてのrunの`head_sha`が同じ固定Headであることと、対象job/critical stepが`success`だったことを確認した。CIの「成功」はそれぞれのWorkflowの検査範囲の成功であり、全ての製品受入項目の成功ではない。

このHEADについてGo Bridgeと単体試験、Smoke、Host、Lit Custom Element、業務参照Consumerの**実Blob SHAを6件**固定した。レビュー器はパス・Blobの重複やずれ、workflow種別・Run/Job IDの重複、対象版の不一致、critical stepの失敗・未完了を拒否する。

## 成立したものと、成立していないもの

| 項目 | E35での証拠 | 現在の評価 |
| --- | --- | --- |
| 案件スコープ付きProvider構成候補 | BridgeのSourceとGo/Node CI | **参照プログラムで動作** |
| Coordinatorを経由する案件Go助言 | BridgeのSource、Go単体試験、Go Host Smoke | **診断用Binding/Guardで動作** |
| 受注明細の参照情報 | 実2プロセスのGo結合試験 | **読み取り専用の診断** |
| Lit画面 | 実ChromeでのUI操作 | **参照UIとして動作** |
| 正規の#951 Service Contract／Conformance | 製品の証明・受理文書は取得していない | **未確認** |
| 成果物の実Digest・署名・版固定 | Bridgeは`sha256:`＋64個の0という合成Digest | **未成立** |
| 現在のHost本人・権限・fencing・lease | BridgeのGuardは構成Bindingの一致のみ | **未成立** |
| 正式な受注Actionの意味・判定 | 専用Go補助は助言であり`POST /scenario/submit`を呼ばない | **未変更・未受理** |
| 人的作業時間・AI追加質問 | それらの実ログはない | **未計測** |

注意：**E35の固定GitHub出典ファイルはCI中にGitHubから再取得しない。** 各runの真正性は人間がGitHub上で観測したもので、参照試験は記録済みメタデータの内部整合だけを検証する。従ってレビューの出力は`pinned-reference-ci-reported-success`であり、`independentEvidenceAuthentication=false`。実Provider受理やArtifact署名をTrueにする機能は持たない。

## 同一版の条件

比較に用いる新しいHEADが異なる場合は、たとえ過去の3 CIが成功していても`not-supported-at-requested-head`とする。例えばE35対象の後続PR改訂にE34やE35の成功結果を自動流用しない。失敗、未完了、スキップ、他のPRのrunを混ぜても、参照CIの成功を宣言しない。

```sh
node --test reference/application-composition/e35_reference_ci_review.test.mjs
node reference/application-composition/e35_reference_ci_review.mjs
```

正例1件に加えて、古いrun・異なるHEAD・Go Smoke失敗・Chrome成功のみ・Blob差替え・偽Workflow・重複Run/Job等の負例を含める。成功でも、製品の業務Accepted意味やHost Authority、生成AIへの質問数を推定しない。

## E35の判断と次の取り組み

既存Compositionを使った案件Go＋Litの**診断用縦断は成立**した。E34で未確定だった2プロセス連携の成功も、固定HEADのCIでは確認できる。この進展を受け、機構ギャップの主眼は「案件用のProvider候補を作れるか」から、**正式な#951契約・検証済みArtifact・現時点のHost権限に、既存の所有者が正しく接続できるか**へ移る。

ただし今は別担当のPR #1567がDraftであり、当該PRには変更を加えない。#951や#997の正本を複製せず、プロジェクト固有の採用が必要なときに実際の手動選択・生成AI往復・差分を計測する。**独立したComposition Profileは、二方式の同条件実測なしに採用しない**。
