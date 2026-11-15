# REA外部候補選択の合成Fixture

`rea_external_candidates_synthetic.json`は、`internal/operations`のread-only候補選択と`cmd/tei`の独立CLI検証だけに使う合成入力です。実REAが生成した原本ではありません。Evidence IDと`meta.command`に合成であることを明記しています。

Node 7件、Relation 2件、Unknown 1件と、宣言された不完全なCoverageを持たせています。選択されたRelation、同端点の重複、Producer ID欠落時のDerived Handle衝突、未選択件数、Source Revision変更、構造と意味受理の分離を検証するためのデータです。

実保存原本は`internal/externalanalysis/testdata/rea-6.8.0/analysis.json`であり、`normalized_result={}`のためこの合成Graphは存在しません。保存原本ではReceiptの正当性とGraph欠落に対する候補投影拒否を検証します。**合成候補の試験成功を、実ツールによるGraph取得成功やOwner意味受理の証拠として扱いません。** Raw Digestは各入力Byteから別々に計算します。

独立したREA 6.8.0の実View配送CIの失敗は、合成Fixtureでは解消しません。Providerごとに取得したViewの原Byteと状態を別の実行記録で評価します。
