# reference-binding-to-execution

環境の **deployment-executor Binding** を解決し、**実際に実行**まで繋ぐ、読み取り専用の参照です。既存の `routing.RouteCommand`・`routing.Router.Dispatch`・`routing.ValidateResult` を順に呼ぶだけで、新しい解決器も選択の仕組みも作りません。

~~~text
go run ./tools/reference-binding-to-execution/cmd/tei-binding-to-execution \
  -workspace reference/workspace -environment development \
  -action-id db/apply -kind database-release -database sales_core
~~~

## 何をするか

| 段 | 使う既存の関数 |
| --- | --- |
| 選択 | `routing.RouteCommand`（環境の Binding から executor を決める） |
| 実行 | `routing.Router.Dispatch`（登録済み executor へ渡す） |
| 確認 | `routing.ValidateResult`（結果が Binding の executor から来たか照合） |

**Executor の registry は呼出し側（Host）が渡します。** この参照は Executor を自分で組み立てません。付属の CLI は記録用の Executor を1つ登録します（外部効果なし）。

## 拒否する場合

| 理由 | 扱い |
| --- | --- |
| Binding が無い／曖昧 | 実行しない。理由を `error` に残す |
| Binding が未登録の executor を指す | 実行しない。`EXECUTOR_NOT_REGISTERED` |
| 環境が存在しない | 0件で終わる（失敗ではない） |
| 結果が別の executor から来た | `ValidateResult` が拒否 |

**拒否は値として返し、エラーとして落ちません**（JSON を出して終了コード0）。拒否の内容は `error` 欄に残ります。

## この参照が示さないこと

- 製品 CLI へ繋ぐべきか（[判断材料](../../plan/workstreams/COMPOSITION_CONNECT_POINT_BRIEF_2026-11.md)の C-1〜C-4 は未決）。
- 実配備の成立（記録用 Executor は実DB・実クラウドに触れません）。
- 認証・権限・永続的な receipt・外部状態の確認。
- 並行実行（1コマンドずつ順に実行します）。

実行記録は [`plan/evidence/BINDING_TO_EXECUTION_REFERENCE_2026-11-09`](../../plan/evidence/BINDING_TO_EXECUTION_REFERENCE_2026-11-09/README.md) にあります。
