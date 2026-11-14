# Graph表のread-only Reader実証

ここには、[固定済みの5つの`*.tei.md`入力](../../testdata/tei-md-graph/README.md)を使い、Markdown表からNode／Edgeの**候補**を読み取る限定的なGo参照実装を置く。

## 現時点で確認すること

- `## ノードの定義`と`## 関係の定義`の二つの表だけを解釈する。関係の定義元は表であり、Mermaidを意味の正本にしない。
- 表のNode ID／Edge ID／起点／終点を、元のUTF-8ファイルに対する**byte offset [start, end)** と行番号で返す。BOMやCRLFを含んでも元byteを切り出せる。
- 同じ表示名のNodeと、同じ端点を持つ複数Edgeを区別する。重複IDや未知の端点は位置付き診断で拒否する。
- 引用文やコードFenceの中の見出し・表を自動解釈しない。表に未対応のエスケープや文法があれば、黙って読み替えず`UNSUPPORTED_TABLE`で止める。
- `CheckDerivedView`は、**呼出し側が明示した表示ID→Node ID対応**を使って、固定例の狭いMermaid矢印構文だけを比較する。表示名からIDを推定しない。表と図が不一致でも、表由来のGraphを上書きしない。

## 実行

ルートモジュールから`go test ./reference/tei-md-graph-reader`を実行する。テストは、生成実装から独立に保存した`testdata/tei-md-graph/independent-expected.candidate.json`に照合する。元byte位置、レイアウトのみの変更、引用・Fence、未知端点、重複Section、未対応表、表示IDの未解決を負例に含む。

このテストが成功しても、**`*.tei.md`の製品Readerが使えることにはならない**。現行`internal/workspace/tei_md_source_guard.go`の`TEI_MD_PARSER_UNBOUND`は維持する。ここではWorkspace ManifestのSource選択、Language Plugin/Roleの承認、Source Index、実Builder入力、正式TEIが生成したGo、実Plugin/CLI Consumerを接続していない。限定ReferenceのGo生成・独立Go Consumerは上記の別資格で扱う。

## 参照Graphから生成したGoを実行する限定試験

`GenerateOutgoingGo`は`ParseCandidate`で読んだNode/Edgeを入力に、出辺IDを返す`Outgoing`と件数を返す`OutgoingCount`を含む**実行可能なGoソース**を生成する。出辺の判定は生成関数の`switch`内にあり、手書きGoのGraph評価関数へ委譲しない。並び順は固定Fixtureの独立Expectedと同じEdge ID昇順とする。

`generate_test.go`では、生成ソースと独立Consumerを一時Go moduleへ書き出し、**実際の`go run .`で呼び出した結果**を`testdata/tei-md-graph/independent-expected.candidate.json`と照合する。基準・辺反転・配置のみ変更・古い表示図の各有効入力を比較し、重複Edge IDはReaderで拒否する。配置だけの変更では生成Goのbyte列が不変、辺反転では変化し、表とMermaidが不一致なら表の定義を優先する。生成処理は元のEdge ID／起点／終点Cellのbyte位置を参照用`Origins`として別返却し、描画位置変更を生成ロジックに混入させない。

実行にはGoが必要で、外部Networkや特定のHostを要求しない。`GOWORK=off`と`GOTOOLCHAIN=local`を使った一時moduleで検査し、生成物が偶然Readerの`Outgoing()`を呼んで同じ結果を返すだけの実装にはしない。

これは**限定Reference Graph→Go→独立Consumer**の実証である。正式`*.tei.md` Reader、採用済みRole/Business Meaning、Target Generator、BuildIntent／選択Providerの実Artifact、通常製品CLI、PluginによるSelf-buildは未接続である。新しいCompiler Core、第二Resolver、Runtime Hostや正式Expected正本を追加しない。

## 次の製品実装へ引き渡すこと

1. 現行Workspaceが実読込みしたSource byteと、Readerに渡すSource/Revision/Role/Selectionを既存Ownerで結び付ける。単にファイル名が`*.tei.md`であるだけでは選択しない。
2. 曖昧なMarkdown表・Mermaidの一般構文・部分取得は新しい汎用Parserへ拡張せず、既存[#2621](https://github.com/hat47x/tei-platform/issues/2621)／[#988](https://github.com/hat47x/tei-platform/issues/988)と、進行中の[PR #4704](https://github.com/hat47x/tei-platform/pull/4704)のReader検収へ引き渡す。
3. 正式な意味Roleが受理された後、表のEdge反転が**生成Goの実処理→実Provider→独立Consumer**に伝わるかを、[製品完成計画SB-1〜4](../../plan/workstreams/PRODUCT_COMPLETION_PLAN.md)と[#1180](https://github.com/hat47x/tei-platform/issues/1180)／[#2514](https://github.com/hat47x/tei-platform/issues/2514)で別途確認する。

**資格**：限定的なread-only Go Reference。正式なMarkdown Language Plugin、Parser/Loader接合、Mermaid全構文、Writer/編集、Self-build、Owner意味受理は未実装・未検証。第二のParser/Resolver/Runtime/Scenario正本は導入しない。
