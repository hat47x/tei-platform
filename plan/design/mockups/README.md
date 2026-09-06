# 画面モック

設計文書を補助する画面モックを置きます。単体で開けるHTMLとして保持し、実装物でもCanonicalの正本でもありません。

- 機能単位の設計本文: [`../`](../README.md)
- 作業定義と完了条件: [`../../issue/`](../../issue/README.md)
- 採用した設計判断: [`../../adr/`](../../adr/)

## 置くときの約束

- 状態を冒頭に書きます。採用前のものは「評価待ち」とし、実装やReference変更の根拠にしません。
- Canonical ID、Pattern、Validation、Contract Queryの名称はReference Workspaceの実定義に一致させます。画面に出す業務データはサンプルであることを明示します。
- 実装済みの機能と設計段階を同じ表現で描きません。未取得のEvidenceを取得済みのように見せません。
- モックで見つかった決定事項は、このディレクトリではなくADRまたは対応Issueへ移します。

## 一覧

| ファイル | 内容 | 状態 |
| --- | --- | --- |
| [`tei-projection-design-sketch.html`](tei-projection-design-sketch.html) | 第一Web Target、Command / Observation、Editor、Studio、既存認証、Database Workbench、Gate Evidence、Runtime観測、CLI出力体裁、同時編集、キーボード操作、移行者向けの対照、1280幅の折り畳みの13フレームと、決定すべき論点22件 | 評価待ち |

`tei-projection-design-sketch.html`の前提は、業務端末（1280〜1920のデスクトップ、キーボード中心）、移行元は一般的なローコード開発ツール、Editorは複数人での同時編集を視野に入れる、静的モックとして描く、の4点です。
