# E14：実Workspaceで三視点の直接参照と出典を照合する

**状態：Goの回帰テストを追加済み。実行結果は未取得。** 設計・検証用Referenceであり、完成したApplication Composition Profile、変更影響の全経路、業務的正当性、実行権限を示さない。

管理：[Issue #1016](https://github.com/hat47x/tei-platform/issues/1016)／[Draft PR #1051](https://github.com/hat47x/tei-platform/pull/1051)。関連する型付き参照・影響解析は[#1120 / PR #1139](https://github.com/hat47x/tei-platform/pull/1139)、Loaderの解析済みbytesとSource Indexの一致は[#1017 / PR #1111](https://github.com/hat47x/tei-platform/pull/1111)が担当する。

## 対象を絞った実証

[E13](E13_SAME_WORLD_DERIVED_OBSERVATION.md)では、同一の`loaded.World`からE5のApplication直接参照と`index.Build(World)`の一部を得る入口を設けた。E14では、その情報に**既存の三視点間の直接参照が本当に存在するか**を、実`reference/workspace`の`order_entry`を入力とするGoテストで確認する。

| 参照元 | 既存Derived Indexのrelation | 参照先 |
| --- | --- | --- |
| Application `order_entry` | `owns-pattern` | Pattern `order_entry.customer_reference` |
| Pattern `order_entry.customer_reference` | `relation` | Relation `sales.order.customer` |
| Relation `sales.order.customer` | `targets` | Data `customer.customer` |

この三本は、新しい構成用Graphを生成したものではない。`index.Build`が返す`DeclaredDependencies`と`SemanticDependencies`の直接宣言を**テスト側で照合**したものである。DataとFunctionを結ぶ実例として、意味情報の変更をレビュー対象へ伝えるための根拠候補になる。

同じ`workspace.Load`から組み立てたSource Indexについて、PatternとRelationの`definition`参照がそれぞれ次のPathへ一意に対応することも確認する。

- `applications/order-entry/patterns/customer-reference.tei`
- `semantic/sales/order.tei`

Source Locatorを確認できることは、Source bytesやRepository revisionが固定されたことを意味しない。出典指紋の読取り・解析時bytesの検査は既存Source Index・Loaderが担当する。

## 明示的な負例

[e14-cross-view-source_test.go](e14-cross-view-source_test.go)には、実Workspaceを使う正常系と、Source参照について次の負例を設けた。

- 別Applicationが、名前が似ているだけで`order_entry.customer_reference`を所有したことにしない。
- `sales.order.submit`の入力`sales.order`は既存Derived Indexでは`semantic-element`であり、この時点でTypeやDataのいずれかを強制決定しない。#1120側の型付き解決機構を置き換えない。
- Sourceの定義が欠落、`derived`しかない、同じ`definition`が重複、Artifactが重複、参照Pathが不一致の場合、一意の定義があると判断しない。

追加した**Goテストは4関数、うちSource負例は5サブケース**。これらはGitHubへ登録済みだが、trusted full checkout上で**未実行**。PASS数として数えない。

## E13側で合わせて直したこと

E13の`-include-direct-derived`による追加出力は、既存E5の`report.status=observed`を前提に限定される。今回、この条件を`e13PairedEvidence`へ集約し、未解決のApplicationに対して追加Derived情報を出力しないこと、明示的なオプトインがない場合に旧Envelopeの形を維持することをテスト化した。

新たに**Goテスト2関数**をE13へ追加した。これらも現時点で**未実行**。

## 実証の限界と次のGate

1. **E14は3本の直接参照の存在確認である。** 変更による逆方向のImpact探索、適用可能性、受理済み意味は導出しない。それらは#1120の`modelref.Impact`とShared Acceptanceに任せる。
2. **#1120を直接importしない。** #1120はまだDraftの並行ブランチであり、現在の#1016からその候補Packageを正本として固定しない。将来の共通checkoutでE13のWorldから`derivedbridge.FromIndex`を得て、`Validate`・`Impact`を呼ぶ結合テストを実施する。
3. **#1017も未統合。** 同じ`loaded.World`を使っても、解析時Source bytesの一貫性やファイルシステムTOCTOUまでは保証できない。
4. **SUI Stage 0は正式Applicationではない。** E14の三視点の成立をSUIへ一般化しない。Native Canvas/DocumentとActionの接続は別Consumerで検証する。
5. E5・E13・E14のGo focused testとNodeテスト、実WorkspaceのCLI、最新headでの`tools/repository-health-check.sh`全項目PASS・前後clean・main照合を得るまでは**Draftを解除・統合しない**。

実行する際は、[E13のローカル検証スクリプト](qualify-e13-local.sh)を使う。ただしスクリプトが存在することは検証成功の証拠ではない。
