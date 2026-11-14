# Plugin A/B と公開文書参照の検証入力（PDC-0）

[Issue #4701](https://github.com/hat47x/tei-platform/issues/4701)で指定された最小A/Bと[後続の実CLI読取り #4714](https://github.com/hat47x/tei-platform/issues/4714)を具体化する**合成・採用前Fixture**です。

このディレクトリのJSONは、Plugin/Doc/Disclosure/Contractの正式なTEI ManifestやProduct APIレスポンスを定義しません。ファイルを置くだけで実WorkspaceがPluginを選択したり、公開文書のSource-backed読取りが成功するわけではありません。正式なGrammarとOwnerの受理は#4679/#951/#2621に残します。

## 何を使うか

- `public-manual.md`：Plugin Aが公開可能とする一文書の候補。本体のbyteと実Sourceを照合する対象
- `internal-manual.md`：公開禁止の一文書の候補。**拒否した際に本文・非公開ID・件数・Digestまで漏らさないこと**を確認
- `independent-expected.candidate.json`：合成されたA/B契約、直接依存、文書帰属と負例に対する**採用前の独立Expected**
- [既存のGraph/Rule Fixture](../tei-md-graph/README.md)：意味Graphの表と導出Mermaidを供給。新たなGraph正本を作らない。元の意味IDは `n-1/n-2/e-1/e-2`、同表示名・同端点の判別を引き継ぐ

JSON内の `occurrenceRole` はPDC-02の「関係の異なる意味Role」を試すための**合成注記**です。既存Graph表そのものにはRole列がないため、注記だけで正式な意味Roleが受理済みと扱いません。

## 製品実装への引渡し順

1. 現行製品の `plugin-inventory` でPlugin選択・Version・Requires・Lockを確認する。ただしFixtureのA/Bが実Worldの選択済みPluginであると推定しない。
2. **Doc 1件だけ**、Plugin Aの明示的なDocument参照から管理Sourceのbyte位置・文書版・対象Contract・現在のDisclosureへ接続する。現行の `management.Row` にDoc SourceRefが無いことをGapとして維持する。
3. 採用前ExpectedのPDC-00〜08をSource/対象/権限/版の独立比較として使う。特にPDC-03は拒否したときの**非公開情報の応答漏出を0**にする。
4. 実CLIのSource-backed Read成功、正式`*.tei.md` Reader、Graphの生成Go、実Providerと独立Consumerの結果を別票にする。Referenceの生成Go成功をDoc取得成功へ流用しない。

## 検証の状態

候補データが固定された段階であって、Goテスト・実CLI読取り・Source/Permission/Lockの同一Snapshot、Owner意味受理は**NOT RUN/NOT QUALIFIED**。JSONの比較基準を製品出力から再計算してはいけません。
