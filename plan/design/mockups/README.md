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
| [`annotation-evidence-list-sketch.html`](annotation-evidence-list-sketch.html) | #1808/#958の**未採用・読取り専用**表示スケッチ。既存Reference出力の注記作業状態と人の確認・対象位置を別一覧にし、Owner受理を判定しない。ローカル提供時のみ使用し、後述の起動方法を参照 |

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

効果仮説: 既存の注記の「対応済み」を、業務モデルの「受理済み」と混同する誤表示を、同じ画面上で避けられるか確かめます。限定テストは3件（読取り専用・混在、曖昧/参照切れ、未知schema/契約）。実GUI受入・原典/TEI対応の受理・実データでの有効性は未確認です。異なる投影を並べる実効性が無い場合は、本スケッチは撤回して既存Consumerの表示に寄せます。
