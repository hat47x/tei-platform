# 画面モック

設計文書を補助する画面モックを置きます。単体で開けるHTMLとして保持し、実装物でもCanonicalの正本でもありません。

- 機能単位の設計本文: [`../`](../README.md)
- 作業定義と完了条件: GitHub Issues
- 採用した設計判断: [`../../adr/`](../../adr/)

## 置くときの約束

- 状態を各モックの冒頭に書き、そのモック自身をcurrent stateの正とします。採用前のものは「評価待ち」とし、実装やReference変更の根拠にしません。この索引には可変な状態を重複記載しません。
- Canonical ID、Pattern、Validation、Contract Queryの名称はReference Workspaceの実定義に一致させます。画面に出す業務データはサンプルであることを明示します。
- 実装済みの機能と設計段階を同じ表現で描きません。未取得のEvidenceを取得済みのように見せません。
- モックで見つかった決定事項は、このディレクトリではなくADRまたは対応Issueへ移します。

## 一覧

| ファイル | 内容 |
| --- | --- |
| [`tei-projection-design-sketch.html`](tei-projection-design-sketch.html) | 第一Web Target、Command / Observation、Editor、Studio、既存認証、Database Workbench、Gate Evidence、Runtime観測、CLI出力体裁、同時編集、キーボード操作、移行者向けの対照、1280幅の折り畳み、Coreとプラグインの拡張境界。前提と決定すべき論点もモック内に含みます |
| [`scope-placement-editor-sketch.html`](scope-placement-editor-sketch.html) | 業務要件とインフラを分け、設定項目（Scope Placement Binding）でつなぐエディタ画面。評価待ち。[設計文書](../SUI_MIGRATION_EXTENSION_AND_SPLIT_DESIGN_2026-10.md)の5.5に対応。区分名・判定結果はサンプル |
| [`annotation-evidence-list-sketch.html`](annotation-evidence-list-sketch.html) | #1808/#958の**未採用・読取り専用**表示スケッチ。注記・知識評価・Shared Acceptance Decision履歴を独立した一覧にし、Owner受理を判定しない。ローカル提供時のみ使用し、後述の起動方法を参照 |

## 注釈一覧の限定的な表示試作（#1808）

このスケッチだけは試験可能な投影ロジックを同じディレクトリのES moduleに分けているため、HTMLファイルの直接起動ではなく、**127.0.0.1に限定した一時的なHTTP配信**が必要です。外部サービスへデータを送信する処理はありません。権限のない原典資料や未公開モデルのJSONを公開サーバーへ配置しないでください。

```sh
python3 -m http.server 4173 --bind 127.0.0.1 --directory plan/design/mockups
# http://127.0.0.1:4173/annotation-evidence-list-sketch.html をローカルブラウザで開く
node --test plan/design/mockups/annotation-evidence-list-projection.test.mjs
```

入力欄には次の**既存の読取り専用ReferenceのJSON出力**をそれぞれ貼り付けます。

- `go run ./tools/reference-authoring-auxiliary/cmd/tei-authoring-auxiliary -input <request.json>`
- `go run ./tools/reference-epistemic-knowledge -corpus <corpus.json> -context <context.json>`

表示は既存の`schema`/`contract`を確認します。異なる契約や未対応の表現は推測して受理せず、表示を拒否します。注記`open/done`、知識の`confirmed`、`targetBinding`を並列表示しますが、両ReferenceのIDや`subject`を推測で結び付けません。原典の事実確認、TEI対応候補、source/model Owner受理、Canonical Acceptance、実行観測を**これら二つのReferenceから導出することはできません**。対応受理は「判定不可」と表示します。

効果仮説: 既存の注記の「対応済み」を、業務モデルの「受理済み」と混同する誤表示を、同じ画面上で避けられるか確かめます。既存投影の合成テストは9件です。今回のDecision履歴独立表示には別の合成テスト6件を追加しています。実GUI受入・原典/TEI対応の受理・実データでの有効性は未確認です。異なる投影を並べる実効性が無い場合は、本スケッチは撤回して既存Consumerの表示に寄せます。

### 評価の詳細表示（追加PoC、採用前）

情報評価の各行に「注意情報」と折り畳み式の「評価の詳細」を追加しました。既存`epistemic-assessment/v0`に含まれる`reviewBinding`、`metadataState`、`contentOrigin`、`ingestedBy`、`contextCompatibility`、`freshness`、`lifecycleState`、`conflict`、`inferredContextUsed`、`reasons`を**独立した表示項目**として扱います。未提供の値は「Referenceに値なし」、未知の値は「未対応の値」と表示し、正常・確認済み・受理済みを補完しません。

注意情報には参照切れ、古い確認履歴、メタデータの欠落、文脈・鮮度不足、競合、置換などの**別々の理由**を併記します。詳細中の値や理由も`textContent`で挿入し、HTMLとして解釈しません。表示に必要なAssessment項目のみを使い、on-demand detailの取得や保存、本人確認・受理、別ReferenceとのID自動対応付けを行いません。

追加の合成試験は、confirmed＋candidate-only、stale review＋partial metadata、AI由来＋user ingestion、文脈不適合・stale freshness・conflict、未提供／未知値、評価理由の非HTML化を対象とします。**一覧はproject-wide Healthやconsumer-specific freshnessの合格判定ではありません**。Nodeテストの実行、ブラウザーの正式module配信、Studioへの組込み、Windows GUI検証はそれぞれ異なる証拠として記録してください。

### Decision Indexの独立表示（追加PoC、採用前）

`reference-shared-acceptance`の既存`decisions`は、現在Workspaceで再照合できる**保存済み補助Decision記録**だけを`tei.reference.decision-index/v0`として返します。注記や知識評価に対応付ける共有IDの契約は存在しません。このためモックには独立した3つ目の一覧を置き、**ID・subject・表示名の文字列一致でjoinしません**。

```sh
go run ./tools/reference-shared-acceptance decisions -workspace /path/to/authorized/scratch-workspace
node --test plan/design/mockups/annotation-decision-index-projection.test.mjs
```

`decisions`のJSON結果は入力欄へ**任意**に貼り付けます。入力が空なら「未入力」、正しい空配列なら「Decision履歴0件」と区別します。いずれも「過去のAcceptがなかった」ことを意味しません。未知schemaや不正なレコードは表示しません。記録がある場合も以下を独立表示します。

- `decision.decision`のaccepted/rejectedは**Shared Acceptance Referenceの処理結果**であり、Ownerのmapping受理や人的承認資格ではない。
- `status=current/superseded`は当該Decisionに記録したModelRevisionと現在Workspace全体のRevisionとの照合。Source編集の個別の現在性`sourceEdits=intact/diverged/none`とは別（例：`superseded + intact`は矛盾しない）。
- `reviewBindingStatus=verified`も、Review Projectionの指紋整合であって、判断者の身元・承認権限の証明ではない。`decisionActorStatus`/`decisionAuthorityStatus`は独立表示し、当該Referenceの`unresolved`を補完しない。
- `artifactEvidenceStatus=recorded`は**外部原典の現在性を確認したことではない**。Evidence・Reviewの不整合を表示しても、注記へ自動伝播しない。
- `recordPath`や警告コードは展開時のみ文字列表示する。Evidence本文や原典のprivate locatorを収集・公開しない。

**操作直後**に`DECISION_RECORD_FAILED`の警告が返り、Source受理は成立してもDecisionが未保存の場合、後日`decisions`を読むだけではその操作の有無や警告は復元できません。rollbackが不完全な`refused + sourceChanged=true + ROLLBACK_INCOMPLETE`も、保存済みDecision Indexの一覧だけでは再現できません。別のHost Current Accepted Source witness、正式な人的承認資格、実Provider/DBの副作用、X-TEA対応受理、Windows GUI試験はこの画面の範囲外です。

新たなSource/Accepted変更、API、保存schema、Core enum、Authority、accept/rejectボタンは設けません。この第三の一覧で確認対象を取り違えたり認知負荷が増えるなら、表示は取り下げて既存Shared Acceptance Consumerへ寄せます。合成Node試験と製品Windows GUI受入は別のEvidenceとして扱います。
