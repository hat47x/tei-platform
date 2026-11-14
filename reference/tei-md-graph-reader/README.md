# Graph表のread-only Reader実証

ここには、[固定済みの6つの`*.tei.md`入力](../../testdata/tei-md-graph/README.md)を使い、Markdown表からNode／Edgeの**候補**を読み取る限定的なGo参照実装を置く。

## 現時点で確認すること

- `ParseCandidate`は`## ノードの定義`と`## 関係の定義`の二つの表だけを解釈する。関係の定義元は表であり、Mermaidを意味の正本にしない。
- 表のNode ID／Edge ID／起点／終点を、元のUTF-8ファイルに対する**byte offset [start, end)** と行番号で返す。BOMやCRLFを含んでも元byteを切り出せる。
- 同じ表示名のNodeと、同じ端点を持つ複数Edgeを区別する。重複IDや未知の端点は位置付き診断で拒否する。
- 引用文やコードFenceの中の見出し・表を自動解釈しない。表に未対応のエスケープや文法があれば、黙って読み替えず`UNSUPPORTED_TABLE`で止める。
- `CheckDerivedView`は、**呼出し側が明示した表示ID→Node ID対応**を使って、固定例の狭いMermaid矢印構文だけを比較する。表示名からIDを推定しない。表と図が不一致でも、表由来のGraphを上書きしない。

## 規則の限定Go生成

`ParseRuleCandidate`は、固定された`## 算定規則の定義`の一行だけを追加解釈する。規則のID・対象・出力・順序が明示的に一致した場合に限り、条件「起点 = 対象ノード」または「終点 = 対象ノード」を読み取り、元のbyte位置を保持する。文章中のルールらしい記述やMermaidのラベルから意味を推測しない。 表の日本語条件はこの限定Fixtureの読みやすい表層であり、正式な日本語DSL文法を採用するものではない。内部では`from`／`to`という限定Role候補に解決する。未対応の条件や複数規則は明示的に拒否する。

`GenerateGo`は候補GraphのEdge集合と条件から、`e.from == nodeID`または`e.to == nodeID`を含む**実行可能なGo関数**を生成する。テストはそのソースを別の`go run`プロセスでコンパイル・実行し、生成器とは独立に記録した期待結果と比較する。Edgeの反転と規則の切替では生成コードが変わり、MermaidのLR/TBだけを変えた場合は生成コードが一致する。古い派生図が意味の定義元を上書きすることはない。

これはあくまで**参照実装に限定した処理生成の実証**である。正式なTEIのRule/Operator受理、製品のBuilder/Provider、既存CLIの実Consumer、自己再構築は接続していない。コード生成が可能になったことを、`*.tei.md`の正式Reader完成やTEI製品のセルフビルド成功へ昇格させない。

## 同じ生成Binaryへの実行時Query

生成Goの`main`は、引数を付けない場合に従来どおり全NodeとEdge IDの一覧をJSONで返す。`-node n-1`または`-node n-2`を与えると、そのNodeの結果だけを返す。**存在しないNode ID、明示した空のNode ID、余分な位置引数は、空配列や全件への暗黙Fallbackとせず、非ゼロ終了で拒否する。** Node指定を省略した全件取得と、`-node ""`／`-node=`の明示空文字列は区別する。

`rule_test.go`では、固定済みの有効な各Source候補から生成したGoを**一度だけ`go build`**し、その同一Binaryに7通り（全件・Node二件・未知Node・明示空ID二方式・余分な引数）の要求を送る。Node別出力と全件出力を既存の独立Expected候補と照合し、呼出し前後の**実BinaryのSHA-256が不変**であることを確認する。これはCallerの入力で振る舞いが変わる最小の証拠であり、Source変更による再生成や固定JSONの書出しとは分けて扱う。

ここでの`-node`は**生成したReferenceプログラムの検証用引数**であって、製品`tei` CLIの公開Command名、Function Entry、Host選択済みProvider、Application APIの正式Contractではない。同じBinaryへの複数回の実行と、異なる製品Consumerが同じProviderを呼び出したことは別の資格である。次段では、受理済みMeaning/Source Revision、契約と選択Provider、実Builder/Artifact、実CLI/独立Consumerの呼出しを既存#4495/#1251/#951/#2600に従って一貫して照合する。

## 候補の構造指紋と、原文・Artifactの別資格

`CandidateShapeFingerprint`は、このReference Generatorが利用する**Node ID集合、Edge ID／起点／終点、Rule ID／起点・終点条件**を正規化したJSONからSHA-256を計算する。NodeやEdgeの表中の並び、表示名、Markdown装飾、Mermaidの配置・表示、Sourceのbyte位置は含めない。値には`tei-md-graph-reference/shape-v1`という固定の区分を加え、後日の生成規則変更で同じ識別子を流用しない。

生成されたReference Binaryは`-candidate-shape`でこの候補指紋を一行出力する。既存`-node`の検索結果や引数なしの全件JSONは変えない。`-candidate-shape`と`-node`の同時指定は不正な要求として拒否する。固定したFixtureでは、基準とレイアウトのみ変更した例は同じ指紋、Edge反転とRule切替は異なる指紋、表とMermaidの不一致例は表の構造に従った指紋となる。

**三つの指紋は用途が違う**。Source IndexのSHA-256は*元Markdownの実byte*、この構造指紋は*Referenceが解釈した候補の構造*、実Binary SHA-256は*実際にBuildしたArtifact*の識別子である。候補指紋が一致してもSourceの同一性・Readerの正当性・Ownerによる意味受理・選択Provider・現在の権限・実Consumer受領は証明できない。逆に説明やレイアウトだけの編集はSource SHA-256を変えるが、同じ処理を再生成すべきと自動判断する根拠ではない。A→B→Aで同じ内容に戻っても、過去の受理・実行資格は自動復活しない。

この指紋と照会引数は**限定Referenceにだけ存在する**。正式なTEI Providerの公開コマンド、Bundle証拠、Semantic RevisionやArtifact Revisionの新しい正本として採用しない。今後は既存#2600/#951/#4495の実消費・選択・実呼出しと照合し、必要な情報だけを製品契約へ取り込む。

## 実行

ルートモジュールから`go test ./reference/tei-md-graph-reader`を実行する。テストは、生成実装から独立に保存した`testdata/tei-md-graph/independent-expected.candidate.json`に照合する。元byte位置、レイアウトのみの変更、引用・Fence、未知端点、重複Section、未対応表、表示IDの未解決を負例に含む。

このテストが成功しても、**`*.tei.md`の製品Readerが使えることにはならない**。現行`internal/workspace/tei_md_source_guard.go`の`TEI_MD_PARSER_UNBOUND`は維持する。ここではWorkspace ManifestのSource選択、Language Plugin/Roleの承認、Source Index、実Builder入力、**製品Generator**、実Plugin/CLI Consumerを接続していない。

## 既存Source Indexとのbyte照合

`sourceindex_bridge_test.go`では、限定Readerが返したNode・Edge・Rule候補の元byte位置を、製品側で既に使用している`internal/sourceindex.RangeFromOffsets`で照合する。`NewArtifact`の原文SHA-256を固定し、元Sourceの説明を追記した場合には、古いArtifactの検証が拒否されることを確認する。実Sourceのbyte、候補の位置、将来のMeaningを同じものとして扱わないための境界試験である。

**この試験はSource Indexの意味Referenceを登録しない。** `*.tei.md`は現行`NewArtifact`では一般の`md`形式に分類され、Language PluginやRoleは拡張子から選択できない。Workspaceでの実読み込み、Source受理、Buildの実消費、選択Providerの実Callは引き続き未確認。正式な接合は#2621/#988/#3030と既存Source IndexのOwnerが検収する。

## 次の製品実装へ引き渡すこと

1. 現行Workspaceが実読込みしたSource byteと、Readerに渡すSource/Revision/Role/Selectionを既存Ownerで結び付ける。単にファイル名が`*.tei.md`であるだけでは選択しない。
2. 曖昧なMarkdown表・Mermaidの一般構文・部分取得は新しい汎用Parserへ拡張せず、既存[#2621](https://github.com/hat47x/tei-platform/issues/2621)／[#988](https://github.com/hat47x/tei-platform/issues/988)と、進行中の[PR #4704](https://github.com/hat47x/tei-platform/pull/4704)のReader検収へ引き渡す。
3. 正式な意味Roleが受理された後、今回の局所Go生成を**製品Generator→選択済み実Provider→通常CLIと独立Consumer**へ接続できるかを、[製品完成計画SB-1〜4](../../plan/workstreams/PRODUCT_COMPLETION_PLAN.md)と[#1180](https://github.com/hat47x/tei-platform/issues/1180)／[#2514](https://github.com/hat47x/tei-platform/issues/2514)で別途確認する。

**資格**：限定的なread-only Reader＋規則条件からのGo生成Reference。生成Goの別Process実行は回帰試験で実証。正式なMarkdown Language Plugin、Parser/Loader接合、Mermaid全構文、Writer/編集、製品Go生成・実Provider、Self-build、Owner意味受理は未実装・未検証。第二のParser/Resolver/Runtime/Scenario正本は導入しない。
