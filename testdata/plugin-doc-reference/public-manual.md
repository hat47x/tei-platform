# Graph 基本機能の利用案内（合成Fixture）

この文書は、Pluginの公開説明と契約・Sourceの対応を検証するための**入力データ**です。TEIの正式な公開マニュアル契約、Owner受理済みの操作説明、製品CLIでの実行例ではありません。

## 対応するモデル

公開候補の契約：`fixture.graph-base` version `1.0.0` / `graph-read/v1`。

Graphの意味定義は [既存のMarkdown Graph入力](../tei-md-graph/baseline.tei.md) の表にあり、Node `n-1` と `n-2` は同じ表示名ですが別IDです。Edge `e-1` と `e-2` は同じ起点・終点でも別の関係です。表示用の Mermaid `N1/N2` は意味IDではありません。

公開Extension Point候補 `graph.read.after` は、この合成テストで下位Plugin `fixture.graph-extension` の参照先とします。これを製品TEIの採用済みExtension Pointとして扱ってはいけません。

## 調査手順

実際のTEIバイナリで利用できる操作を `tei commands` から調べ、Workspace/Environmentで選択されたPluginを `plugin-inventory` の一件詳細から確認します。本FixtureのPlugin IDは現行製品Workspaceに登録されているものではないため、そのまま`plugin-inventory`に指定しても実参照の成功にはなりません。

現在の契約版と文書版が一致すること、文書の元byteと取得範囲、Audience/現在の開示範囲を照合してから、公開Graphの読取りへ進みます。本文の取得を、Writer/Providerの起動許可へ昇格させません。

## 検証上の禁止事項

- この説明の記載だけで未実装の `tei plugin docs` を実在コマンドとみなさない。
- 上位文書をBが参照できても、Bから第三者Cへの再公開が許可されたとはみなさない。
- Source Indexで参照が0件であることを、対象Subjectが存在しない証拠とみなさない。
- Graphの限定Referenceテストを、正式`*.tei.md` Readerや製品Go生成の受理とみなさない。
