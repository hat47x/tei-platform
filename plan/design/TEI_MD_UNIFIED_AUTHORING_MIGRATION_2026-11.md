# TEI Authoring Source を .tei.md へ統一する段階移行計画

> 2026-11-06 方針：ユーザーによる明示的な製品方向決定。標準のTEI記述Sourceは *.tei.md とし、旧 *.tei の新規作成を原則停止・段階廃止する。
> この文書は設計・移行・独立受入計画。既存Parser/Loader移行、ソース変換、CI、SHO/Studio動作、Owner意味受理は実装・実証前。実データの無断移行を認めない。

## 1. 決定の範囲と目的

普通の文章は *.md、TEI向け定義を主目的とするMarkdownは *.tei.md。既存の *.tei は legacy/import 対象に下げ、原則として新規 Authoring Source の標準名にしない。*.tei.md がMarkdownでありTEIに関係することをファイル名だけで分かるようにする。末尾 .md を保ちSHO・Obsidian・一般Markdownツールとの相互運用を目指す。

**対象外**：ていではない一般文書 *.md、Workspace の tei.workspace.yaml、Sourceとは別のJSON/YAML/SQL、バイナリ成果物、記録ディレクトリ .tei/decisions/、その他の点で始まる管理用パスは自動改名しない。元の *.tei が旧Gate固定版にある場合も履歴を変更しない。

**絶対条件**：新拡張子だけではOwner Accepted Meaning、Host Grant、実Provider Effectは得られない。ファイル名の判定、Markdownとしての形式適合、TEI言語解釈、Role解決、Meaning Acceptance、実Build/Execution資格を独立の段階で扱う。

## 2. 現行mainの確認された障害（2026-11-06）

- internal/workspace/loader.go は plugin.tei、bundle.tei、application.tei、database.tei、environment.tei、migration.tei や *.tei の固定名/グロブを使う。semantic root は strings.HasSuffix(name, ".tei") で絞る。Workspace に新拡張子を置くだけでは同じSourceとして消費されない。これは実コードの静的確認であり、実テストではない。
- loader.go の legacy manifest は tei.workspace.tei を構文解析する。現在は tei.workspace.yaml を主経路としているため、manifest の互換廃止は通常の Source ファイル変換と切り分ける。
- internal/syntax/parser.go は TokenLexerを用いてTEI DSLを解析する。Markdown frontmatter/heading/fenced blockを読み取る入口は現実装で確認できず、*.tei を *.tei.md へ単純Renameして解析可能としない。
- internal/workspace/loader.go は ManagedFile.Extension を filepath.Ext(name) で保存する。*.tei.md はここでは .md と識別されるため、専用の複合拡張子判定と既存Index/診断/APIの扱いを要する。
- reference/workspace/tei.workspace.yaml は source root の宣言を持つ。固定Gate B/C/共有受理が関係する既存 *.tei Source は手編集・一括移行せず、既存AGENTS.mdの固定履歴とAccepted Sourceの手順を守る。

## 3. 新しいAuthoring形式の意味（候補、文法未決）

*.tei.md は **UTF-8 Markdownファイル**。説明文、Markdown表、リスト、式、Graphviz等の糖衣を保存できる。TEI DSLを含むときは fenced code block 等のMarkdownとして妥当な領域に明示し、実Parser/Language Pluginをその領域だけに適用する方法を第一のPoC候補とする。Fenced text以外の表/章にRoleを与える方式はTEI #2621/#2620の別の検証対象であり、全ての表を勝手に実行しない。

例えば概念的には以下。構文例であり、正式TEI Language/Metadata Profileではない。

````markdown
---
type: Data Model
tags: [tei]
---

# Order

受注モデルの説明。

```tei
semantic sales.order {
  // この行以下は例示用の仮構文
}
```
````

前記の入れ子フェンス表示は文書中では概念例。実装はMarkdown ParserのSource位置/byte範囲がTEI Lexer/Parserへ正しく渡せる表現を決めてから対応する。実Parserが対応しない仮DSLをfixture PASSとして扱わない。正式DSLがMarkdownと両立しない場合、内部Snippetとして保持するなど、Markdown互換で成立する記述経路を追加検証する。

OKF v0.2のConcept Fileとして利用するときは任意Pluginを使い、type等の要件を満たす。一般の *.tei.md をOKF Plugin常駐の必須依存にしない。tagsや拡張tei項目はAuthoring intent／候補の選択にとどまりMeaning受理ではない。OKFのファイル名由来Concept IDとTEIの安定Meaning IDは別。

## 4. 移行手順

| 段階 | 実施・観測 | 廃止ゲート／失敗時 |
| --- | --- | --- |
| M0 現状Inventory | 全Rootで *.tei、固定名、読み込んだ実Source、manifest、SourceIndex、Document Link、CLI/生成物とFixtureの参照を列挙。固定版・部分消費・未知Sourceを分ける | 未列挙・Loader未読込みを0件扱いしない |
| M1 新規標準とLint | *.tei.md を新規テンプレート/生成/Studio・SHO候補の既定名に。新 *.tei はlint/deprecation診断、旧参照は維持 | 既存Sourceをrenameしない |
| M2 Markdown Container Reader | 有効なMarkdownを読み、指定済みRole/DSL領域のSource span/byteを抽出し、実Language Plugin/Parserへ。BOM、CRLF、日本語/引用・重複宣言/不明Roleを診断 | 元DSL parserに直接Markdown全文を入れない |
| M3 Loader/Build選択 | 固定名とglobに *.tei.md 対応。複合拡張子を .md 一般と区別。既存source root/manifest/SourceIndex/依存閉包を検査 | *.tei と *.tei.md が同一Subjectなら無断で片方を選ばない |
| M4 明示Import | legacy *.tei を新 *.tei.md 文書へ包み込み、Original Source byte/span→new spanの写像、ID、相対リンク・generated artifact依存・Reviewの失効/再受理要否を提示。作業Draftで差分確認 | 内容・Meaning不変は独立Expected/既存Acceptanceでしか主張しない |
| M5 限定切替 | 固定業務Scenarioと異質Domainで新Source→モデル→図表/コード生成の独立対照。SHO Writer・Studio read-only/編集資格・外部Markdown Editorで再読込 | 実Build対象が同一か、Target Outcomeが同じかは別票 |
| M6 legacy縮退 | 新規 *.tei 拒否/既存は明示legacy importer・read-onlyへ。現行Sourceの移行が検証できた後だけlegacy Loader/生成入口を除去 | 固定Gate・証拠・履歴・第三者リンクを破壊しない |

## 5. 出典・同一性・編集衝突

新しいパスと旧パスは異なるSource Location。同じ意味IDを持ってもFile identity、Source Revision、Occurrence、Link、原byte、Decision/Reviewと実Loader消費は同一視しない。*.tei.md/旧*.teiの同じSubjectを二つの編集正本にせず、移行時には一意の受理対象を指定する。A→B→Aや同じdigestでも別所在を別事件として監査し、歴史の受理を自動復活しない。

SHOは未完成/不正構文でもRaw Draftを保存可能とし、TEIの正式意味採用とは区別。元文書に本文・表・Snippetと説明が混在しても、人間が選んだ部分しか厳密化対象にしない。AI Skillは部分抽出、曖昧さの列挙、図表候補・独立Expectedのレビューを助けるだけで、自動Owner受理は行わない。

## 6. 独立Expected（すべて移行計画、NOT RUN）

| ID | Fixture/故障例 | Expected |
| --- | --- | --- |
| TM-01 | plain.mdは通常の文章 | TEI Sourceの存在/意味受理/生成を推測しない |
| TM-02 | model.tei.mdが正しいMarkdown+明示TEI定義 | 新Readerが元byte/spanとTEI領域を別観測。実Loaderが読むまではParse PASSを主張しない |
| TM-03 | 同内容のmodel.tei.mdとmodel.tei | 重複Subjectを明示診断。先勝ち/後勝ちをしない |
| TM-04 | 旧model.teiをそのまま改名 | Markdown適合/TEI解析成功を推測せず、変換候補のまま |
| TM-05 | BOM/CRLF/日本語/引用中の宣言 | no-op保存でraw bytes保持。Spanの位置は実Sourceに対応 |
| TM-06 | 旧DSLの複数定義をMarkdownの複数領域へ | 元と新のIdentity/依存/重複・対象集合を照合。未対応の形式はunsupported |
| TM-07 | 固定Gate B/C旧ファイルと共有Decision | 移行が歴史的固定版・Decision pathを改変しない |
| TM-08 | 移行中の外部Editor更新、R0→R1→R0 | 現Source再検証。旧Reviewは自動復帰しない |
| TM-09 | 旧Sourceと新SourceのBuild結果が同じ | Source実消費/Toolchain/Build Artifact/実Target効果の一致は別条件 |
| TM-10 | OKF tags:tei だけの普通のmd | TEI Sourceとして自動受理しない |
| TM-11 | Obsidian・SHO・標準Markdown Editorが新Sourceを開く | 未知TEIフェンス/Frontmatter保全。編集できたことを意味受理としない |
| TM-12 | Plugin/Parser欠落、未読込Sourceや0件 | Raw Draftは利用可能、Meaning/Buildはunsupported/unknownと明示 |
| TM-13 | tei.workspace.yaml や .tei/decisions/ | 対象外を拡張子一括置換しない |
| TM-14 | exact fixed-name/ glob Loader各domainとSourceIndex | 対象Sourceが実際に読み込まれたことを独立に測定し、Catalog非空だけで成功としない |

## 7. Issue Ownerと現状

- TEI #2621＝Markdown Authoringと移行後のSource fidelity、#988＝Language/Parser、#2620＝Role、#1124＝Studio Source編集と正式Acceptance、#2823＝実Loader欠落、#2514＝独立Oracle、#2600＝Build依存、#1440＝AI自然言語→モデル候補。
- SHO #19＝軽量Writer/Editor、SHO #18＝任意OKF、SHO #20＝Obsidian/Cloud/Vault。SUI #3391の探索成果はSHOのMarkdown Sourceとして受け、TEI側の意味に自動昇格させない。
- この方針はユーザーによる方向付け。現行Core/DSL/Source形式・Legacy廃止期日・破壊的移行の実施判断は個々の実証とOwner Reviewに分解し、現mainの能力だとは表示しない。新しいSemantic Registry、Host、Acceptance経路は作らない。
