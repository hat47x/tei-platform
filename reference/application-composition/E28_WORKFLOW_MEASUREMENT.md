# E28：構成選択の作業・質問・更新費用を、検証範囲と分けて記録する

- **管理**：[Issue #1016](https://github.com/hat47x/tei-platform/issues/1016)。
- **基点**：2026-10-13、TEI `a2d92df59a802901274ec00302d09b81575a3422`。新たな構成Profile、依存解決器、証拠台帳、権限を設けず、[E27の読取り監査](e27_residual_selection_audit.mjs)を使う。
- **実装**：[測定用の読み取り専用集計器](e28_workflow_measurement.mjs)／[Node回帰試験](e28_workflow_measurement.test.mjs)。

## 観測できる事実と、外部からの申告を混同しない

[E26](E26_COMPOSITION_COST_BASELINE.md)は受注明細の**既存5ファイル・67非空行**と、SUIの**隔離された候補3ファイル・9／15非空行**を測定した。[E27](E27_RESIDUAL_SELECTION_AUDIT.md)は現在のTEI checkoutで既知Sourceを観測し、10監査項目を既存4・Stage 0の1・選択未解決3・実行時の証拠不足2へ分類した。

これだけでは人間の手動作業、生成AIへの追加質問、改修・検証にかかった時間は**観測できない**。本E28は、その値をゼロや推測値で補わない。入力なしの実ソース読取りでは作業計測を`null`とし、外部から作業記録が提出されても、**提出された件数**と**独立して確認された件数**を分離する。

| 区分 | E28で数えるもの | 数えないもの |
| --- | --- | --- |
| `humanDecisions` | 提出された人間による判断イベントの件数 | E27の未解決項目数を判断実施回数へ換算しない |
| `aiQuestions` | 追加質問の提出イベントの件数 | チャット全体のトークン数、推測した質問数 |
| `manualConnections` | 手動接続の提出イベントの件数 | 自動生成された差分や未実行の構成候補 |
| `changedFiles` | 内容指紋の異なる変更元・変更先を申告した**相異なるファイルパス数** | 同一ファイルの複数イベントを別ファイルと数えない |
| `validationRuns` | 提出された検証実行イベントの件数（成功／失敗／未解決を別記） | 提出URLの存在だけをテスト成功と推認しない |
| `requalificationRuns` | 更新影響の再確認イベントの件数 | [E25](E25_CROSS_REPOSITORY_REQUALIFICATION.md)の未検証契約を自動合格にしない |

`submittedCounts`は**申告されたイベントの数値**である。`verifiedCounts`はE28では常に`null`。提出されたGitHubのURLは照合の手掛かりであって、リンクがあるだけで本人の判断、実際の手動操作、CI成功、秘密情報アクセス、権限取得は証明されない。実際の検証は既存のGitHub、Build/Release/Runtime、Source Indexなどの正本で行う必要がある。

## 使い方と境界

```sh
# 実ソースを読み取ってE27分類を再利用する。作業回数はnullのまま。
node reference/application-composition/e28_workflow_measurement.mjs .

# 外部提出イベントの合成試験（実際の作業履歴を生成しない）。
node --test reference/application-composition/e28_workflow_measurement.test.mjs
```

JavaScriptの`observeE28(root, submission)`に提出イベントを渡すこともできる。提出構造は`tei-e28-workflow-submission-v0`、案件ID／利用先／比較方式／固定した開始コミット／Target、作業の完了申告、イベント配列からなる。各イベントは重複しないID、E27の監査項目ID、操作分類、出典のGitHubリンクを持つ。変更ファイルについては安全な相対パスと前後の40桁Git blob SHA、検証イベントは結果の`passed`／`failed`／`unresolved`を申告する。

**申告は実施証拠を自動検証したことにはならない。** 不正なURL、別Consumerの監査項目、出典が失われた監査項目、同じイベントIDの二重計上、移動・更新を伴わないファイル指紋、変更ファイルへのパストラバーサルを拒否する。AIへの入力全文、顧客情報、秘密情報、Runtime Grantは収集しない。

`complete: true`は提出者による完了**申告**にすぎず、実Provider、採用済み業務意味、正式なTarget、SUIの認証済み保存を受理しない。

## 「既存の機構を使う」と「追加Profileを使う」の比較

`compareE28(existing, candidate)`は、案件ID・利用先・開始Gitコミット・Targetが同一で、片方が`existing-source`、他方が`profile-candidate`のときだけ、**申告イベント数の差**を返す。条件が異なれば`not-comparable`とする。**E29の追加条件**：双方の作業記録が提出者により`complete: true`と明示された場合だけ、申告件数の差分を返す。片方でも未完了（`partial-not-verified`）なら、案件・Target・開始コミットが同じでも`not-comparable`とする。未完了のイベント提出数を「案件全体の作業回数」として比較しない。

同条件で双方が完了申告済みでも結果は`reported-only-not-verified`、確認済みの生産性改善値は`null`である。`complete: true`はあくまで提出者による完了申告であって、E28がCI成否・正式な実行権限・導入の成立を検証した証拠ではない。

対照実験に必要な情報を、今回作った仮のProfile定義で埋め合わせない。どちらの方式でも、[E27の既存正本・残余判断](E27_RESIDUAL_SELECTION_AUDIT.md)と同じSource、同じAuthorityの検査、同じWeb/Native実行先を使う。

## AI-first案件Plugin生成との関係

`order_entry`の小さな案件固有機能を、(a)既存実装、(b)案件内改修、(c)新規Plugin生成の順に比較する方針が[#1016の追加コメント](https://github.com/hat47x/tei-platform/issues/1016)に記録された。別の並行[PR #1567](https://github.com/hat47x/tei-platform/pull/1567)は、Goによる補助判定とLitの案件UI部品を独立に実装中（確認時点ではDraft）。E28はその実装を変更せず、案件固有の生成・選択・手動接続・再検証の申告と、実際の実行証拠を区別する。

実際のGitHub PRには差分とCI結果があるが、**人間が何回判断し、AIへ何回追加質問したかはPR差分だけでは分からない**。今回は本当の二つの方式を同条件で実施していないため、改善率を示さない。

## 採否への影響

ここまでのE21〜E28で追加Profileが必要とする独立の正本は立証されていない。他方、Profilerによる削減効果も実測されていない。したがって[E10の採用保留](E10_COMPOSITION_ADOPTION_DECISION.md)を継続する。最初に実案件で取得すべきものは、出発改訂、案件要求、採用した既存Source、実際に変更したファイル、手動操作の実記録、AIへの追加質問の実記録、局所ビルド／実Target／現在の権限の各検証結果である。

SUIのCard／Document／認証・CASは引き続き[#993](https://github.com/hat47x/tei-platform/issues/993)が所有する。本実装の提出記録からそれらの成功やTEI第二Applicationの完成を推定しない。


## E30：未追跡の種類をゼロとせず、変更ファイルの参照元を区別する

E28の申告形式はそのまま利用し、**必要になった場合に限って**、提出物に任意の`coverage`（種類ごとの記録完了・未追跡）を付けられるようにした。これは開発者に詳細な設定を初回から強要する製品仕様ではなく、比較結果を誤解しないための参照試験である。

- `coverage`のキーは既存の`humanDecisions`・`aiQuestions`・`manualConnections`・`changedFiles`・`validationRuns`・`requalificationRuns`に限定し、値は`complete`または`untracked`。記載がない種類は未追跡とする。
- `submittedCounts`は、従来どおり**受け取った申告イベントの数**。新しい`claimedCoveredCounts`は、セッション全体の`complete: true`と、その種類の`coverage: complete`が明示された場合だけ数値を持つ。それ以外は`null`。単に提出件数がゼロだからといって、作業回数ゼロの観測に昇格しない。
- 両方式が同じ基点・対象で完了申告済み、かつ双方が`coverage`を提示した場合は、**共通に記録が完了した種類の差分だけ**返す。それ以外の種類の差は`null`。共通に測定した種類が一つもない場合や、片方にしか`coverage`がない場合は`not-comparable`。従来の提出形式に`coverage`がなければ、E29の完了申告条件の下で従来の「提出件数の比較」だけを維持する。
- `changed-source`イベントの`repository`には、任意で`owner/name`形式の出典を付けられる。これがある場合、**`repository`と相対`path`の組**で変更ファイルを計上する。TEIとSUIの同じ相対ファイル名を一つにまとめない。指定がない古い記録は引き続き相対パスで数えるが、出典が分かったことにはならない。リポジトリ名は申告値であり、GitHub APIで真正性を照合したことを意味しない。

例えば、`complete: true`、`coverage: {manualConnections: 'complete', aiQuestions: 'untracked'}`とし、提出イベントに手動接続が1件・AIへの質問が0件あれば、`claimedCoveredCounts.manualConnections`は申告値の`1`、`claimedCoveredCounts.aiQuestions`は`null`となる。これは「AI質問は0件だった」という認定ではない。

変更種別・出典・改訂を厳しく照合する必要がある場合は、既存の[E25リポジトリ間変更影響](E25_CROSS_REPOSITORY_REQUALIFICATION.md)とGitHubの正本へ戻る。E30も正本Source・Provider契約・実権限・SUI Native保存（#993）を複製しない。

```sh
node --test reference/application-composition/e28_workflow_measurement.test.mjs
```

これらは**提出記録の評価方法**だけの回帰試験であって、導入工数や生成AIとの実会話回数が実測できたという主張ではない。確認済みの件数は依然として`verifiedCounts: null`、本番への採用可否は別に判定する。
