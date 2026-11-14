# Graph表のread-only Reader実証

ここには、[固定済みの6つの`*.tei.md`入力](../../testdata/tei-md-graph/README.md)を使い、Markdown表からNode／Edgeの**候補**を読み取る限定的なGo参照実装を置く。

## 現時点で確認すること

- `ParseCandidate`は`## ノードの定義`と`## 関係の定義`の二つの表だけを解釈する。関係の定義元は表であり、Mermaidを意味の正本にしない。
- 表のNode ID／Edge ID／起点／終点を、元のUTF-8ファイルに対する**byte offset [start, end)** と行番号で返す。BOMやCRLFを含んでも元byteを切り出せる。
- 同じ表示名のNodeと、同じ端点を持つ複数Edgeを区別する。重複IDや未知の端点は位置付き診断で拒否する。
- 引用文やコードFenceの中の見出し・表を自動解釈しない。表に未対応のエスケープや文法があれば、黙って読み替えず`UNSUPPORTED_TABLE`で止める。
- `CheckDerivedView`は、**呼出し側が明示した表示ID→Node ID対応**を使って、固定例の狭いMermaid矢印構文だけを比較する。表示名からIDを推定しない。表と図が不一致でも、表由来のGraphを上書きしない。

## 規則の限定Go生成

`ParseRuleCandidate`は、固定された`## 算定規則の定義`の一行だけを追加解釈する。規則のID・対象・出力・順序が明示的に一致した場合に限り、条件「起点 = 対象ノード」または「終点 = 対象ノード」を読み取り、元のbyte位置を保持する。文章中のルールらしい記述やMermaidのラベルから意味を推測しない。未対応の条件や複数規則は明示的に拒否する。

`GenerateGo`は候補GraphのEdge集合と条件から、`e.from == nodeID`または`e.to == nodeID`を含む**実行可能なGo関数**を生成する。テストはそのソースを別の`go run`プロセスでコンパイル・実行し、生成器とは独立に記録した期待結果と比較する。Edgeの反転と規則の切替では生成コードが変わり、MermaidのLR/TBだけを変えた場合は生成コードが一致する。古い派生図が意味の定義元を上書きすることはない。

これはあくまで**参照実装に限定した処理生成の実証**である。正式なTEIのRule/Operator受理、製品のBuilder/Provider、既存CLIの実Consumer、自己再構築は接続していない。コード生成が可能になったことを、`*.tei.md`の正式Reader完成やTEI製品のセルフビルド成功へ昇格させない。

## 実行

ルートモジュールから`go test ./reference/tei-md-graph-reader`を実行する。テストは、生成実装から独立に保存した`testdata/tei-md-graph/independent-expected.candidate.json`に照合する。元byte位置、レイアウトのみの変更、引用・Fence、未知端点、重複Section、未対応表、表示IDの未解決を負例に含む。

このテストが成功しても、**`*.tei.md`の製品Readerが使えることにはならない**。現行`internal/workspace/tei_md_source_guard.go`の`TEI_MD_PARSER_UNBOUND`は維持する。ここではWorkspace ManifestのSource選択、Language Plugin/Roleの承認、Source Index、実Builder入力、**製品Generator**、実Plugin/CLI Consumerを接続していない。

## 次の製品実装へ引き渡すこと

1. 現行Workspaceが実読込みしたSource byteと、Readerに渡すSource/Revision/Role/Selectionを既存Ownerで結び付ける。単にファイル名が`*.tei.md`であるだけでは選択しない。
2. 曖昧なMarkdown表・Mermaidの一般構文・部分取得は新しい汎用Parserへ拡張せず、既存[#2621](https://github.com/hat47x/tei-platform/issues/2621)／[#988](https://github.com/hat47x/tei-platform/issues/988)と、進行中の[PR #4704](https://github.com/hat47x/tei-platform/pull/4704)のReader検収へ引き渡す。
3. 正式な意味Roleが受理された後、今回の局所Go生成を**製品Generator→選択済み実Provider→通常CLIと独立Consumer**へ接続できるかを、[製品完成計画SB-1〜4](../../plan/workstreams/PRODUCT_COMPLETION_PLAN.md)と[#1180](https://github.com/hat47x/tei-platform/issues/1180)／[#2514](https://github.com/hat47x/tei-platform/issues/2514)で別途確認する。

**資格**：限定的なread-only Reader＋規則条件からのGo生成Reference。生成Goの別Process実行は回帰試験で実証。正式なMarkdown Language Plugin、Parser/Loader接合、Mermaid全構文、Writer/編集、製品Go生成・実Provider、Self-build、Owner意味受理は未実装・未検証。第二のParser/Resolver/Runtime/Scenario正本は導入しない。
