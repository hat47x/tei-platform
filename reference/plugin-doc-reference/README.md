# Plugin公開文書の候補Fixture監査（Reference限定）

このディレクトリは、[PDC-0の合成Fixture](../../testdata/plugin-doc-reference/README.md)を、**既存のGraph Readerと元ファイルのbyteを使って監査するGoテスト**を置く場所です。

実行候補：

```sh
go test ./reference/plugin-doc-reference -count=1
```

`TestCandidateFixtureLinksToRealGraphAndManualBytes` は、採用前JSONのA/B/C直接依存・文書所有・版・開示候補・期待結果ケースIDを検査します。さらに、既存 `reference/tei-md-graph-reader.ParseCandidate` に`baseline.tei.md`の実byteを渡し、同表示名の二Node、同端点の二Relationと元byte位置、および現行Sourceに追加された明示Rule表のID・条件が候補と一致するか確認します。Graph／公開・非公開ManualのSHA-256を候補JSONの固定値と実byteで照合し、公開／非公開Manualの実ファイルが分かれ、公開Manualに非公開本文が複製されていないことも確認します。

`TestCandidateFixtureMutationsAreRejected` は、Aの選択版、Bの直接依存、Lock候補、再公開、非公開指定、Source digest欠落、Path traversal、関係ID、意味Roleの採用表示、拒否Expectedの漏洩、ケース欠落、製品の受理済み主張を個別に改変し、**14種類の候補Fixture改変に対する整合性検査**が不整合を検出できるか調べます。

`TestPDC0ReferenceGraphShapeDoesNotReplaceSourceOrDisclosure` は、実際のGraph／Rule候補から構造指紋とGo生成ソースを照合します。図の配置変更と説明の追記は候補構造を変えず、辺の向きやRule条件の変更は生成ソースを変える一方、原文や公開Manualを編集すれば各Sourceのbyte指紋は更新が必要です。これは**参照Fixtureに対する監査**であり、製品PluginのDoc取得・Disclosure・現在Grant・実Provider Callではありません。

これらの試験は、独立ExpectedのOwner採用、正式なPlugin/Doc/Disclosure Manifest、Productの公開文書Source Reader、`plugin-inventory`の実Plugin A/B選択、Provider実Call、AI/Hostの情報漏洩を実証しません。JSONに「permission-denied」とあることの照合は、実Hostの拒否を呼んだ証拠ではありません。

既存のGraph→Go生成・独立実行は`go test ./reference/tei-md-graph-reader`で別途評価します。正式製品側の縦切りは[#4714](https://github.com/hat47x/tei-platform/issues/4714)、Fixture/ExpectedのOwner判断は[#4701](https://github.com/hat47x/tei-platform/issues/4701)と[#1180](https://github.com/hat47x/tei-platform/issues/1180)/[#2514](https://github.com/hat47x/tei-platform/issues/2514)の担当です。新しいパーサー、Registry、AuthZ Engine、Runnerは追加していません。

**検証資格はコミット・Go Toolchain・OS・テストResultを記録した実Attemptでのみ成立**します。テストファイルをコミットしただけの段階では実行済みとはしません。

**基準Sourceの版**：現在の`baseline.tei.md`はGraph表に加えて明示的なRule表を含み、候補JSONのSHA-256はその元byte全体を固定する。Graphの同ラベルNode・多重Relationは本監査、Rule条件からの生成Goは既存のReferenceテストが判別する。
