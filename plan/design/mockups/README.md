# 画面モック

設計文書を補助する画面モックを置きます。単体で開けるHTMLとして保持し、実装物でもCanonicalの正本でもありません。

- 機能単位の設計本文: [`../`](../README.md)
- 作業定義と完了条件: [`../../issue/`](../../issue/README.md)
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
