# E18：意味上の関係変更を、二つの解析済みソース状態から追跡する

**位置づけ：** [Issue #1016](https://github.com/hat47x/tei-platform/issues/1016) の限定的な回帰検査。構成プロファイルの採用、業務意味の受理、Providerの適合、実行権限を示さない。

[E17](E16_POSTMERGE_TYPED_IMPACT.md#e17同一解析済みbytesと二つのファイル状態)では、同じ意味定義のまま本文へ改行を追加し、ソース指紋だけが変わることを確認した。しかし、**実際の意味定義の変更とApplicationへの影響候補**は未確認だった。

[E18のGoテスト](e18_semantic_source_change_test.go)では、実際の`reference/workspace`を一時ディレクトリへ複製して変更する。元の参照用作業領域は編集しない。

## 1. 関係名と使用側の宣言を変更する

- 意味定義 `semantic/sales/order.tei` の関係を`sales.order.customer`から`sales.order.account_contact`へ変更する。
- 受注明細の`applications/order-entry/patterns/customer-reference.tei`が参照する関係名も同時に変更する。
- 変更前後それぞれの`workspace.Load`と`workspace.BuildSourceIndex`を使い、両ファイルの**実際の指紋が変わったこと**、関係とPatternの定義元が一意なこと、Applicationの直接依存が未解決でないことを確かめる。
- 変更前に解析した結果からSource Indexを再生成しても、変更後の本文へすり替わらないことを確認する。
- 両時点の既存`index.Build`→`derivedbridge.FromIndex`から、`modelref.ReviewImpactDiff`で削除された旧関係と追加された新関係を識別する。既存Graphの直接参照を使い、PatternとApplicationが変更の再確認候補に含まれることを検査する。

**新しい依存関係グラフ、Resolver、更新Lockは作らない。** 二つの時点の型付きグラフは既存の構成要素からそれぞれ生成する。

## 2. 多重度だけを変更する

別の一時コピーで、関係の`one`を`many`へ変更する。ソース指紋は更新されるが、現時点のDerived Indexは関係の多重度を型付きグラフへ投影していない。そのため、`modelref.Diff`だけを見れば変更なしと返る可能性がある。

ここでは**Source側の変更通知を優先して再確認を要求する**。型付き差分が空という理由で、変更なし・互換・適用可能・受理済みへ昇格させない。この差は今後のDerived Indexへの意味属性追加を検討する材料となるが、この試験のために本番側のモデルや規約を増やさない。

## 確認できる範囲と残る課題

- 二つの状態は一時ディレクトリにある**異なる実ファイル内容**であり、**異なるGitコミットの真正性を確認する試験ではない**。
- 型付き投影には既知の未解決参照が残る。選択した既知の参照経路を確認しても、全依存閉包を得たことにはならない。
- 関係名・多重度の編集は**試験的な候補変更**。意味の受理・データ移行・Queryの型互換・Provider接続・Runtime権限を自動的に正当化しない。
- 検証は`go test -count=1 ./reference/application-composition -run '^TestE18'`で実施できる。結果は対象のPRコミットとGitHub Actionsの記録に結び付ける。

次段階では、二つの実Git改訂を固定した出典・案件構成の完全な影響評価、SUI Nativeという第二Consumerの実接続、権限と環境ごとの採用判定が残る。
