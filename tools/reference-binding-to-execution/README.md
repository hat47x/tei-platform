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
| 結果が別の executor から来た | `ValidateResult` が不正な応答として検出。**Executor.Execute が呼ばれた事実は消さない** |

### 呼出し前の拒否と呼出し後の不確定を分離（v1）

`bindingexecution.Run`は従来、`Router.Dispatch`がエラーを返した場合や`ValidateResult`が不一致を検出した場合に、一律で`refusedCount`を増やしていました。しかし、**実際にはExecutorが呼び出された後**にエラーになる場合があり、そのときに「拒否＝未実行」と表示するとEffect Unknownを隠してしまいます。

| v1のフィールド | 数える事実 | **示さないこと** |
| --- | --- | --- |
| `ranCount`、`executorInvoked` | `Executor.Execute`へ実際に入った回数／当該Commandで入ったか | 外部処理のCommit／状態反映／現在のGrant |
| `refusedCount` | Binding不在、未登録、`Supports`不一致など、`Execute`前の停止 | 既に実行した処理の失敗数 |
| `unverifiedCount` | `Execute`到達後のエラー、または別Executor等の不正な応答 | 適用済み・未適用の確定。**`ranCount`の内数** |
| `bindingResolved` | 正しいBindingに対する妥当な応答まで受領 | 永続化、成功したビジネス効果、Clientの受領 |

不確定な場合も`error`に原因を記録し、**自動再送や暗黙のObservationは行いません**。実行前拒否と実行後不確定はどちらもレポートの値として返るため、付属CLIは現在どちらでもJSONを出して終了コード0となります。製品の無人実行・監視でこの終了コードを成功判定へ流用しないことが必要です。

`tei.reference.binding-to-execution/v1`では、この限定意味変更に合わせ`RuleVersion`と`RuleFingerprint`も更新します。過去の[実行記録](../../plan/evidence/BINDING_TO_EXECUTION_REFERENCE_2026-11-09/README.md)のv0 JSON・SHA-256を改変せず、各版に対応した証拠として保持します。

[合成Effect負例](bindingexecution_effect_unknown_test.go)は、同じ`Executor.Execute`エラーでも内部状態が適用済み／未適用となり得ること、正常なAccepted応答でも効果が無い可能性を持つことを明示します。**メモリ上のテスト用状態であり、実DB・実Host・Receiptの受理ではありません**。

## この参照が示さないこと

- 製品 CLI へ繋ぐべきか（[判断材料](../../plan/workstreams/COMPOSITION_CONNECT_POINT_BRIEF_2026-11.md)の C-1〜C-4 は未決）。
- 実配備の成立（記録用 Executor は実DB・実クラウドに触れません）。
- 認証・権限・永続的な receipt・外部状態の確認。
- 並行実行（1コマンドずつ順に実行します）。

実行記録は [`plan/evidence/BINDING_TO_EXECUTION_REFERENCE_2026-11-09`](../../plan/evidence/BINDING_TO_EXECUTION_REFERENCE_2026-11-09/README.md) にあります。
