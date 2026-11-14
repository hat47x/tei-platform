# TEI Authoring Source を .tei.md へ統一する段階移行計画

> 2026-11-06 方針：ユーザーによる明示的な製品方向決定。標準のTEI記述Sourceは *.tei.md とし、旧 *.tei の新規作成を原則停止・段階廃止する。
> この文書は設計・移行・独立受入計画。既存Parser/Loader移行、ソース変換、CI、SHO/Studio動作、Owner意味受理は実装・実証前。実データの無断移行を認めない。

## 本文書と既存計画の所有境界（2026-11-07 横断レビュー）

- **移行実装・検収の主担当**：本書（[Draft #3030](https://github.com/hat47x/tei-platform/pull/3030)）の M0〜M6／TM-01〜14。既存 [Draft #2667](https://github.com/hat47x/tei-platform/pull/2667) は横断研究の履歴・Ownerへの受渡し（M0〜M5、TMD-01〜12）を残す参照資料であり、**独立した第二の移行仕様や試験Gateにはしない**。#2667の現在の冒頭にも#3030を主担当とする旨が明記されている。以前の相互参照の循環は本節の責務分離で解消する。内容差は #2621 の担当判断まで未決として示す。
- **Authoring設計の実証対象**：利用者が実際に触れる `*.tei.md` の Markdown適合性とTEI認識範囲、旧DSLからのSource span対応、SHO/Obsidianでの原文保全、AIによる部分定式化。本書のTM/AU条件は #2667のTMD/他Ownerの既存試験との**対応表を持ち、同じ固定Fixture／独立Expected**で検証する。検収IDの正本は本書側に集約し、第二のParser、Source Registry、Acceptance、Semantic SSOTは作らない。
- **意思決定の確度**：拡張子の方針はユーザー決定。Markdown内のDSL構文、Role宣言、移行方法、互換期限、Source Writer、製品E2Eは未決・未実行。プロジェクトの `.tei/decisions/` は既存Decisionの保存先であり廃止対象ではない。

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


## 8. 記述体験と機械的資格を分離する Authoring Profiles（追加の成熟差分）

### 8.1 三つのAuthoring候補を同一入力で比較

| 候補（未採用） | 著者の主な入力 | 長所 | 反例・必要な独立観測 |
| --- | --- | --- | --- |
| **A 説明＋Markdown表＋Role** | Markdownの見出し・表・式に必要部分のみRole/対象を宣言 | ドメイン専門家の自然な認知順、SHO/外部エディタの書き味 | 同形の表が説明・データ・テストのいずれか判別できない場合、**実行候補の特定を保留**。見出し名から意味を推測しない |
| **B コードFence内に既存TEI DSL** | Markdown本文＋指定された`tei`コード領域 | 旧DSLの字句を再利用しやすく限定PoCに適する | 突き抜けたDSL記号や未閉じFence、引用中の例示と実定義を区別。**Fenceを見つけただけでBuildへ登録しない** |
| **C Markdown内に独立して配置した宣言書式** | 説明文の近くに短いTEI専用構文（Markdown上も合法） | コード塊による認知ノイズを抑えられる可能性 | 新Grammar/Renderer依存・外部Readerの未知構文・曖昧パースを同じ基準で比較 |

**選定基準**は「専門家にとって自然な認知順・修正容易性・ノイズの少なさ」→「出典位置と部分保存」→「解釈の決定性/意味忠実度/Plugin/Grammar固定」→「外部Markdown互換」の順。Go ASTの実装都合、Markdownとして綺麗にレンダリングされること、AI生成時のToken数を先頭基準にしない。旧DSL全文をFenceへ包む方式を新標準の正式Grammarと先取りしない。A/B/Cに共通する標準データ構造/Role/Meaningと同じ独立Expectedを使う。PoCは先に最小の実動作から始め、比較結果をOwnerが採用するまで投影同値を主張しない。

### 8.2 書きかけの文書と、意味解釈対象の選択を独立させる

`*.tei.md` は**ファイル全体のTEI向けAuthoring意図**を識別するだけ。見出し・本文・表・サンプルコード・引用・検証対象は同一ファイル内に共存可能。定義として解釈する対象は、既存WorkspaceのSource Root/選択Profile、文書内の明示 Role/Occurrence、固定されたLanguage Plugin/Grammarと、実Loaderが読んだ範囲の積として判定する。未選択の通常Markdown、説明用の`tei`コード例、引用中の宣言、OKFの`tags: [tei]`は自動Semantic Sourceではない。

SHOの**Raw Draft Save**は、構文が壊れている途中・未閉じFence・不完全表・未知Plugin、未解決Referenceでも原文を保存できる。これらの診断はTEI解釈段階に保持し、旧Parser候補を保存成功で意味受理へ昇格しない。UIのLint/補完は世代番号・buffer版を持つread-only候補であり、AI/非同期結果の後着は古いSourceへの自動Patchをしない。

### 8.3 Source spanと入れ子構文の差

Markdown Frontmatterの長さ、BOM、CRLF、見出し改行、Fence開始・終了文字列/長さ（バッククォート・チルダ）、インデントと最終改行を踏まえ、`embedded TEI offset` → **元ファイルのbyte-offset範囲**の単一対応を観測する。TEI Parserが返す行列位置を本文全体の位置へ単純に加算するだけではBOM/CRLF・引用/複数Fenceで誤る。必ず抽出前の実bytes/SourceRevisionと対応表を固定し、該当領域への診断・局所Patchだけを許す。**抽出Snippetだけのhashと原文全体のhashを混同しない**。

無変換保存ではBOM/CRLF/空白/未知Frontmatter/元のScript糖衣などのbyte列を維持する。旧 `*.tei`を新Markdownへ変換した場合、Markdownの包み込みにより**新しいファイル全体のbyte列が旧Sourceと一致するとは主張しない**。旧元byteと新領域に対応する字句、コメント、識別子の保存・意味変化・失われた付随情報を別々に採点する。

### 8.4 文章→モデル→図表を自動的な正本変更にしない

SHO/SUIで作成したMarkdownの任意範囲を選択し、AIなしの手動表/図展開または任意のAI Skillから**TEI構造候補**を得る。元文書、提案した表/Graph、Owner受理済みMeaning、正式生成Artifactはそれぞれ別のIdentity/Revision。AIが推論して補ったActor・PK/Relation・NULL/欠損・時間条件・単位は人間未受理として明示。Markdown全体を機械的に定式化する必要はない。

SHO Editorだけで書き始め、SUIによる発想の展開を必要に応じて経由し、`*.tei.md`に切り出しても、SUIのTree/Island/空間関係の原版をMarkdown見出しへ一対一に押し込まない。文章化された仮説はMeaning受理と異なる。外部Vault/OKFに保存できることもTEI実行資格の根拠ではない。

### 8.5 Authoring側の独立Expected（#2667のTMD履歴ケースと対応付け・全件NOT RUN）

| ID | 同一固定Fixtureに追加するMutant | 独立Expected／検出する誤実装 | 所有 |
| --- | --- | --- | --- |
| **AU-01** | `*.tei.md`内の説明用FenceをBuilderが定義として処理 | 明示Selection/Roleが無ければ**実定義0**、全ファイル受理不可。空結果は「必要定義が無い」の証明にならない | #988/#2621 |
| **AU-02** | 2個のTEI領域と未閉じFence・BOM+CRLF+日本語見出し | 元byte span/診断/段落境界が正確。解釈できない部分はDraftのまま | #2621/SHO #19 |
| **AU-03** | Markdown表が説明/Case Expected/実Dataの同形 | Role未宣言なら候補保留。全表の型付きData自動登録0 | #2620/#2621 |
| **AU-04** | 並列Editorが本文改版、Source候補を旧snippet位置へApply | bufferとdiskの実Revisionを比較。採用前差分で競合を止める | SHO #4/#1124 |
| **AU-05** | 旧`*.tei`原文をFenceへ包むがコメント/空白を正規化し隠蔽 | 旧元byte/新領域/Meaning候補/診断を別観測し、正本移行成功とは表示しない | #988/#2505 |
| **AU-06** | SUIの異なるTree枝が同じIslandに集中したB型文章をTEIのHierarchyに自動転写 | 文章化の仮説は未受理、SUIの別構造と出典を維持。Ownerの明示Meaning判断まで保留 | SUI #3391/SHO #19 |
| **AU-07** | Markdownはlint PASS、Role Provider未ロード、`validate ok`、Buildが古い`*.tei`を消費 | Raw/Grammar/Loaded/Meaning/Build actual consumedを別列。必要なSourceの未読込をPASSにしない | #2823/#1163 |
| **AU-08** | `foo.tei.md`がOKFでConcept ID `foo.tei`を得て旧`foo.tei`と文字一致 | OKF Concept ID/旧Source Path/TEI Meaning IDを別。自動relocation/Owner受理0 | SHO #18/#2505 |

**実施順**：本書TM-01〜04/13/14（#2667のTMD-01〜04/07と対応）の固定1 Subjectの読込実測を最優先→同じSourceでAU-01〜05/07を最小Authoring PoC→AU-06はSUI/SHO Producer/Writerの資格が成立した後→AU-08はOKF Pluginを選択した場合のみ。Expectedは既存#2514の独立Oracleで保持し、変換実装/AI自身がExpectedを生成して合格させない。既存G0〜G4や他Issueに新Gateを増やさない。

<!-- tei-md-mixed-acceptance-freshness-20261106 -->
## 9. 同一 `*.tei.md` 内の部分受理とClaim単位の鮮度を移行受入へ結ぶ（2026-11-06、設計／NOT RUN）

### 9.1 ファイルのAuthoring意図と、受理対象Occurrenceの境界
[#2621](https://github.com/hat47x/tei-platform/issues/2621)の「`*.tei.md` はTEI向けAuthoring意図を示す」と、[#267](https://github.com/hat47x/tei-platform/issues/267)のShared Acceptanceにおける**実際のOwner Decision対象**は同じ粒度でない。たとえば一文書に、(a) Owner受理済みの`sales.order.total`宣言、(b) 未受理の`sales.order.discount`候補、(c) 説明表、(d) 独立Expected、(e) 未閉じFenceが共存できる。**ファイル全体が受理済みでも、全体が未受理でもない**。意味受理は、元Source occurrence／対象Role・Subject／Decision対象のVersion・Scopeに対する既存の判断だけを参照する。ファイル名・Frontmatter・OKFタグ・Syntax PASSを受理の印にしない。

一個のAuthoring Sourceを保存・Renameしても、**Raw Draft保存、実Readerが消費した部分、Structural Candidate、受理済みMeaningの現在適用、Claimごとの必要Dependency、現在Consumerへの開示、実Action資格**を段階別に判定する。新しいMeaning RegistryやFile-level `accepted` フラグを作らない。元ファイルのraw revisionと抽出Occurrence/span、Role/Operator Pin、Source Indexに残った宣言と実Loaded Sourceの対応を、既存機構の比較列として確認する。

### 9.2 同一文書・固定二宣言で一因子ずつ変える（TM/AU/FC/RF既存Fixtureへの差分）

**共通Fixture（公開合成、未実行）**：`order.tei.md`に受理済みRole `total`、未受理Role候補 `discount`、説明表、2つの独立Expectedを置く。候補の実数値・対象集合を実装から逆生成せず、[独立Oracle #2514](https://github.com/hat47x/tei-platform/issues/2514)で先に固定する。受理の有無は**既存Shared Acceptanceに実在するDecisionが示すときだけ**適用し、説明用Fixtureの`accepted`文字列を本物のOwner判断と呼ばない。以下はTM-03/05/06/08/12、AU-01/02/04/07、RF-02/05/11、FC-01/06の補助判別列であり、**新しいCase ID群を増やさない**。

| 単一変更 | Source／Candidateへの期待 | 受理・Claimへの期待 | Consumer・失敗時 |
| --- | --- | --- | --- |
| **説明文だけ編集、`total`本体の意味入力・依存不変** | 文書raw revisionとSpan/SourceIndexは再定位、変更領域は観測。読込済みOccurrenceとPlugin/Operator Pinを確認 | `total`の旧受理は歴史として維持。必要な意味・依存が同一と照合できるなら**現在適用の維持候補**。文書全体の一律再受理を要請しない | 引用や行番号は再確認。未知の非出力依存があれば`unknown`、無根拠なBuild skipは不可 |
| **未受理`discount`候補だけ改稿** | 当該Candidateを再導出。 `total`と重なるRole/依存・重複Subjectなら曖昧性を記録 | 依存が観測上無関係なら`total`は維持候補。新`discount`を自動受理しない | 全ファイル再Build/再Reviewも、同形候補の黙った昇格も禁止 |
| **`total`が参照するOperator/Role版だけ変更** | `total`のSource byte不変でもParser/Operator由来の候補適合を再照合 | 旧Decisionの最新版への適用を保留/Owner確認。 `discount`のDraft保存は阻害しない | 実計算・Artifactの現在利用を再資格化。既存内容Cacheの履歴は維持 |
| **Ownerが`total`の受理を撤回** | Source raw byte、構造Candidate、実Read履歴は同じ | 実在する撤回Decisionが確認できるなら`total`の**現在の意味付きClaimは不適格**。旧Decisionの履歴は維持 | 現在Meaningに依存するSHO/AI表示・Action候補・Build適用を再判定。過去Effectを消去・再送しない |
| **`discount`のみ未閉じFence／部分Parse、`total`は取得可能** | `parse partial`と実取得Scope／抽出失敗を区別。`total`の完全な出典範囲・Dependency Closureが検証できるか確認 | 完全性が独立に示せれば`total`だけ維持候補。不明なら`unknown`、**文書0定義／全定義成功の両方を主張しない** | Raw Draft Saveは可能性を保持、Semantic Promote／実Buildは未資格範囲を除いて用途別判定 |
| **同byte移動、旧`.tei`→`*.tei.md`、A→B→A** | 旧→新のSource occurrence/Parser selection/相対Link・Anchorと各revisionを区別 | 構造同値が観測されてもDecisionの現在適用を無条件継承・復活させない | 独立Read・Link再解決と現在開示を必要条件へ。未解決なら旧引用/旧Actionは保留 |

**特に禁止する二つの誤判定**：`false reuse`（文書byteや旧Decisionが一致したため、変更されたOperator／撤回後Meaning／未取得Dependencyを現在採用）、`false invalidation`（説明文の段落位置・未受理の別候補変更だけで、独立に照合済みの`total` Claimまで強制無効化）。#2506のContext Definition/Instance/Result Slice、[#2582](https://github.com/hat47x/tei-platform/issues/2582)のRead Dependency/Change Impact、[#2505](https://github.com/hat47x/tei-platform/issues/2505)の版間候補、#267のOwner Decisionにそれぞれ引き渡す。**「依存無し」の十分な観測がない場合は維持判定を保留**する。

### 9.3 追加の正本を作らない受入手順
1. **同一raw Sourceと独立Expected**を固定し、`*.tei.md` Reader/Loaderが実消費したbyte・領域・Source location/spanと、抽出結果のCoverageを区別する（M0/M2/M3、#2823）。
2. **明示RoleとOperator PinでStructural Candidateを生成**し、完全解析領域とpartial/unknown、引用・説明用表の非実行を別に報告する（#988/#2620、AU-01/02）。
3. **既存Owner Decisionだけを参照**し、意味適用に必要なOccurrence/Role/Revision/Scope/依存Closureを照合。Acceptance historyの撤回API/判断イベントが実装済みかは別に確認し、無いものを作り話にしない（#267）。
4. **Claimごとの必要ReadとConsumer資格を比較**する。同時Read Cohort/Rowset/Field completeness、現在Selection・Policy・Disclosureを確認し、局所維持・再導出・再取得・Owner再Reviewを分ける（#2506/#2582/#2516）。
5. **誤再利用と過剰失効を両方向で採点**し、実Target/Studio/SHO/外部HostのPASSは、その経路で実入力・実出力が観測されるまで保留（#2514/#2600/#1174）。実Go Reader／Markdown Writer／Owner撤回E2Eは**NOT RUN**。

**実装の優先順**は現行Loaderの`*.tei.md`実読込・Span mappingを先に確認し、それができたSourceで局所Claimの適用判定へ進む。新規Plugin/Host、全文書Acceptance状態、第二Resolver、常駐Reactive Engine、意味正本Registryは追加しない。


## 10. 現在の fail-closed 防御を、実Readerへ置換する条件（2026-11-07 横断更新）

### 10.1 既存実装と確認された証拠の上限

[#3031](https://github.com/hat47x/tei-platform/pull/3031) に由来する現行mainの実装は `internal/workspace/loader.go` が管理対象として列挙した `*.tei.md` を、現在のTEI Parserへ束縛できない状態で **`TEI_MD_PARSER_UNBOUND`** と明示して拒否する防御。対象には semantic/query/persistence/plugin/bundle/release が含まれ、大文字の末尾 `.MD` も拒否する。旧 `*.tei` と通常 `README.md` は引き続きLoaderで扱い、宣言されていないSource Root外の `*.tei.md` はWorkspaceへ入らない。

**実CI証拠**：#3031 exact PR HEAD `63953ccb5c12ead26ec1268e7c0afdba4eff66ae` の [parser-boundary probe](https://github.com/hat47x/tei-platform/actions/runs/42217379093) と [Repository Health](https://github.com/hat47x/tei-platform/actions/runs/42217378867) は completed/success（同HEADで他の関連Workflowもsuccess）。旧HEADのgofmt失敗と区別。現在mainのGuard実Sourceでも同じ拒否経路を確認したが、この照合自体は新HEADのGo試験ではない。この証拠が示すのは**明示拒否・旧形式/通常Markdownの限定回帰**まで。Guardは現在mainに存在する。一方、実 `*.tei.md` Markdown AST、Role解釈、Parser選択、Source span、Build/Host/Consumer、Owner Meaning Acceptanceは**未実装・未実証**。

### 10.2 fail-closed のまま書き続けられる体験を確保する

現Guardは **宣言済みSource Root内の管理対象 `*.tei.md` が1件でもあればWorkspace Load全体を停止**する。これによりモデルが消えても `validate ok` と見える誤成功を防げる一方、SHOで執筆途中の `order.tei.md` をSource Rootへ保存しただけでも旧 `*.tei` を使った既存CLI作業を妨げ得る。これはGuardの**意図された一時的な厳格動作**であり、一般的なTEI Markdown執筆が使えるという保証ではない。

暫定運用では、(A) **SHOの単独Writer／TEI Source Root外の草稿領域**は現Loaderに依存せず自由に保存・再開可能、(B) Source Rootに置いた `*.tei.md` は未対応として明示拒否、(C) 既存の旧形式Sourceを使うテスト・業務PoCを影響を受けない別Workspace/Checkoutで継続、と区別する。未対応ファイルを `*.md` に自動改名する、Guardを削除する、Root全体の警告だけにする、選択されないものとして無音無視することはしない。

本来の完成形では、**選択されたTEI Sourceと、普通のMarkdown／一時Draft／非管理文書を明示的なSelection/Root/Content Kind/版で区別**する。選択済みSourceのParser欠落は依然 fail-closed、宣言されないDraftはBuild Required Sourceの欠落を隠さない、解釈できた一部のOccurrenceだけから必要全集合の充足を推測しない。File kindを拡張子だけに頼らず、現Manifest/Source Root/Language Pinと実Reader Witnessで検査する。これは将来契約の候補であり現Guardの動作ではない。

### 10.3 正負の対照と実Loaderへの置換順（本書TM/AUケースに組み込む）

| 観点 | 入力と独立Expected | 現在の状態 |
| --- | --- | --- |
| **拒否の負例** | 新形式だけが宣言Rootに存在し、Parserが未束縛 → `TEI_MD_PARSER_UNBOUND`、成功/0件扱い不可 | #3031限定Go/CI対照が成立し、Guardは現mainにも存在する（正式Readerは未実証） |
| **旧形式の正例** | 旧 `model.tei`＋普通の `README.md` → 従来Semanticが実読込、例外なし | #3031 限定 Go/CI 対照が成立（Draft） |
| **Root境界の正例** | 宣言外の草稿 `drafts/order.tei.md` があっても、宣言Rootの旧Sourceを読む | #3031 限定 Go/CI 対照が成立（Draft） |
| **作業中の競合** | Source Root内の新形式草稿で現在はLoadを停止。将来はRaw Draft保存と必須Source検証を明示分離 | 現状拒否のみ確認／UIと選択契約はNOT RUN |
| **新形式の正例** | 選択した `order.tei.md` → 実Parser実消費・Span/Role/Source Index・独立Meaning→Build Input／Expectedの一連を観測 | **未実装／NOT RUN** |
| **旧新衝突** | 同一Subjectの旧新Source → 曖昧/競合診断、勝手な優先選択や旧Decision復権なし | 現状は新形式未束縛による拒否のみ。意味ID重複解決ではない |

**次の最小作業**は新しいFixtureを無制限に作ることではなく、既存のTM-02/03/04/14・AU-01/02/07に同じ1 Source/独立Expectedを束縛し、(1) Readerの実バインド、(2) 元Markdown byte→TEI部分のByte Span、(3) **実Loaded Occurrenceと未選択Draft**、(4)旧新の同一Subject競合、(5)実Builderが消費した入力と非出力依存を別観測すること。最初からSHO GUI、全Provider、全Plugin移植、実DB完了を必要条件にしない。#3031の正常/拒否試験は Guard Owner が保全し、#3030が新規Reader/Parser移行の主担当、#2823がRead Coverage、#988がGrammar/Pin、#2514が独立Expectedを所有する。

### 10.4 同一DocumentでもClaim/Decisionと実実行は別（§9との接続）

§9の受理済み`total`と未受理`discount`という一文書例を、**Source Root内でGuardに拒否される段階**と**Readerが実対応して部分Occurrenceを観測する将来段階**に分ける。Guardで拒否した場合は、過去のlegacy解析結果やSource Indexのキャッシュを「新形式を読めた」ことの代用品にしない。将来、一部の`total`だけ構文上読めても、必要なRole/Operator/Function依存とOwner Decisionの現在適用を別照合し、`discount`の未取得・保留・壊れたFenceを0件または承認済みへ縮約しない。実行・Buildが必要なClaimはそのClaimのRequired closureが満たされるまで保留する。

この区別により、**過剰な一括失効（執筆不能）**と、**不完全Readerの誤成功（業務・意味誤認）**を同時に避ける。製品実装・移行完了・Owner受理の主張ではない。


<!-- tei-md-reader-20261114-product-admission-seam-v1 -->
### 10.5 次の実装は「管理ファイルの検出」から「元Sourceを実際に解釈する」までを分割する

2026-11-14のmain実Goを照合すると、workspace.LoadはまずscanManagedFilesで宣言Root内を走査し、その直後にrejectUnboundTEIMarkdownSources(managedFiles)を呼んでいる。この検査は、管理Rootに一つでもTEI MarkdownファイルがあればDomain Loaderへ進む前に明示拒否する。一方、現在のloadSemanticsは従来形式のSourceを読み、syntax.NewParser(...).ParseSemanticFile()へ渡す。ManagedFile.ReadByLoaderは「実Loaderが読み込んだか」を示す独立項目であり、「Managedの一覧に現れた」「Meaningとして解釈できた」と同じではない。

最初の修正でGuardを単純に削除すると、読まれていない新形式をvalidate成功として隠す危険が戻る。そこで、正式なファイル形式移行を一度に行わず、**明示選択した一つの宣言領域**を既存Language/Parserへ結ぶ最小の読取経路から始める。

| 段階 | 既存コード・責務 | 新たに確認する事実と停止条件 |
| --- | --- | --- |
| **MR-0 Guardの維持** | workspaceのManagedFiles走査、tei_md_source_guard.go、既存Guard試験 | Reader未束縛の管理Root内Sourceは拒否、Root外Draftと旧形式の実Readは引続き正常。削除・無音無視に変更しない |
| **MR-1 Markdown容器** | #2621/#988のAuthoring/Language選択、現行Source Root | 元ファイルのraw bytesから**選択済みの宣言領域だけ**を抽出する。本文、説明、例示、引用は自動宣言にしない。選択方式・文法が未決なら候補のまま |
| **MR-2 実Parseと元位置** | internal/syntaxのParser/Position、Workspace Source SnapshotとSource Index | 抽出した同じバイト列を既存Semantic Parserへ渡し、結果と診断を**元Markdownのbyte span**へ対応付ける。BOM/CRLF/複数Fence/引用・インデントで対応が不明なら拒否し、別時点のDisk再読取で補わない |
| **MR-3 LoaderとBuildの実消費** | workspace.Load、ReadByLoader、Source Contribution、#2600 Build Source | 明示選択された1件の実読込と元Source位置を固定し、必要に応じてSource Index/Build Inputで実消費を別観測。全Rootの新形式を勝手に解釈せず、同一Subjectの旧新二重定義を自動解決しない |

MR-1/2は、まず**読取り専用で対象をSemantic一例に絞ったGo回帰**として切り出してよい。この時点で正式な全Domain Workspace Loaderへ入れたと主張しない。TEI用のMarkdown Fenceは第一候補だが、実在する標準Grammarだと決めつけない。選択するLanguage/Version/Roleの契約を#2621/#2620/#988で検討し、専用の第二Syntax Engineを設けない。

### 10.6 既存TM/AUケースと一つの独立Expectedを共有する

別のScenario正本は作らず、既存TM-02/03/04/05/08/13/14およびAU-01/02/07と、#1180/#2514の独立Expectedに次の実装時assertを対応付ける。

| 対照 | 読取り・意味・実行に期待する区別 |
| --- | --- |
| **MR-X1** Parser未束縛の管理Root内Source | TEI_MD_PARSER_UNBOUND。0定義／正常なvalidateとして扱わない |
| **MR-X2** Root外の途中Draft＋管理Root内の旧Source | Raw Draftは保存可能、旧Sourceの実Readは正常。DraftをMeaning/Buildに追加しない |
| **MR-X3** 選択した宣言領域＋説明・引用・例示 | 選択部分のみParserに渡る。表示やMarkdownとしての正しさを意味受理にしない |
| **MR-X4** BOM/CRLF、日本語、複数Fence、引用・インデント | Parserの診断とSource Indexは元Markdownの正確なbyte spanへ戻る。不明な対応を行番号の単純加算で推定しない |
| **MR-X5** 旧形式と新形式が同じ意味IDを宣言 | 先勝ち/後勝ちと暗黙mergeは禁止。選択・移行・受理Decisionの境界を別に記録 |
| **MR-X6** 説明文・図の配置のみ変更 | Raw Revisionとspanの再定位は必要。意味の同一性とBuild/Reviewerの現在性は依存ごとに判定 |
| **MR-X7** 意味領域／Operator版変更、A→B→A | 関係するMeaning/Artifact/Decisionを再照合。同byte復帰で過去の現在資格は自動復活しない |
| **MR-X8** 1宣言が読めたが別のRequired Sourceは不明 | managed、read、parsed、interpreted、actually consumedを別表示。局所PASSから全Dependency完備やOwner受理を推測しない |

第一PRではMR-1/2の**元Byte→選択宣言→既存Parser→元Source位置**だけを新規に検証する。MR-3の正式Workspace選択、GraphからのGo処理生成、Plugin実Call、Stage1からStage2への再構築、StudioのSource編集は次の独立PRへ分ける。Layoutだけの変更とGraphの辺方向変更による実Go生成差は、dogfooding計画§15.38/§15.44とPlugin側の固定Fixtureを再利用する。

### 10.7 Owner・製品計画との合流と現在の上限

#3030/#2621はMarkdown原文と抽出領域、#988/#1018はLanguage/Parser/版、#2620はRole、#2823は実Loader Coverage、#2514/#1180は独立Expected、#2600/#2831は実Builder入力とArtifactを担当する。**最初に作るべき実装PRはReader＋byte-spanの最小Go修正**であり、拡張子の変更、汎用Graph Parser、第二Registry、Hostを作ることではない。

Plugin生成の最初の利用先はdogfooding計画にある「同じ表示名の別Nodeと、同端点の別IDを持つ有向多重Edgeから、出辺IDと件数の実Go処理を生成する」一例とする。生成したGoを手書き処理のWrapperとして使うだけでは自己実装率へ計上しない。同じ原文/意味からのPlugin説明のRead-only参照は#4679/現Plugin文書計画へ渡し、Reader完成の必須条件にはしない。正式Studio Stage1→Stage2と通常TEI CLIからの実Provider読取は引き続き並行P0である。

**証拠の上限**：本追補はmainの実Go Sourceを静的に照合し、既存のGuard限定試験・他計画を実装順に統合したdocs-onlyの設計更新である。新たなReader、Graph Generator、実CLI/Windows/PG試験、Owner意味受理は今回NOT RUN。Guardの現在main上の存在と、TEI Markdownが正式Readerで実利用可能なことは別の主張である。

<!-- tei-md-native-markdown-graph-fixture-20261114-v1 -->
### 10.8 Markdown本来の記述力を優先し、既存DSLの埋込みを唯一の完成形にしない

TEIの新しいAuthoring Sourceは、専門家が普段使うMarkdownの文章・表・インデントリスト・図を保ったまま、明示された部分だけを標準データ構造に対応付けることを目指す。§10.5の「既存DSLを含む一領域を取り出す」MR-1/2は**Source spanと既存Parserの接続を最初に確認する実験**であり、将来の正式構文をコードFence中心に固定する決定ではない。Goのstruct/funcや型引数を新DSLへ露出させず、ドメインの認知順に沿う記述を評価する。

| Authoring表現 | 対応を検討する標準意味 | 実Readerに求める最低条件・注意点 |
| --- | --- | --- |
| **通常のMarkdown表** | 型付きデータ表、Node/Edgeの関係、Rule／テストパターン表 | 列の意味・型・ID・NULL/空集合・重複の解釈を明示Role/契約で決める。表を見つけただけで全行を業務定義として登録しない |
| **インデントリスト** | Tree/Outline、親子・所属候補 | インデントと表示順を保持。Treeの所属とGraphの任意Edgeを混同しない。再帰的な意味付けが曖昧なら候補で停止 |
| **MermaidのER図・シーケンス図** | 型付き関係・プロセス／時系列の意味候補 | **直接記述したSource**として利用可能な経路を検討する。Edge ID、多重性、向き、参加者、時刻・Roleが表現できない場合はlossy/部分候補と明記し、図から無断確定しない |
| **CSV等のリンクされた別ファイル** | 大量表データ・外部データセット | Markdownの参照箇所と実リンク先byte/Revision、取得Scope、粒度、改版を束縛する。未取得行を0件や全件充足に変換しない |
| **既存DSLのMarkdown内Fence** | 現行Lexer/Parserへ直接渡せる明示TEI領域 | 初期MR-1/2の**互換ブリッジ**。Parserが実際に解釈した領域と元Byte Spanを検証し、他のMarkdown表現が既に読めると誤報しない |
| **Graphviz、文章、関数チェーン等** | 同じ標準意味の表示・糖衣、または明示宣言候補 | 原記法を保存する。意味データ、図の配置、コード生成、Writerの適格性を分け、導出Viewを黙って第二Source正本にしない |

渡辺式ER図などの業務上の慣習は、型付き関係を保ったうえで表示・自動配置の規則に反映する。親を左、子を右へ置くといった表示上の約束から、Cardinality、キー継承、Owner判断を無条件に推論しない。Mermaidは**Sourceとして直接書く場合**と、同じMeaningから**再生成した図**の場合を区別し、両方あるときはどちらを編集正本とするかSelectionで決める。非可逆の逆変換はDiff/候補提示に留める（#2497/#4198/#2620）。

### 10.9 同じGraph Fixtureで、Markdown表現から実生成へ進める

初回はmain採用済み[Graph固定Fixture](../../testdata/tei-md-graph/README.md)（#4709）を入力正本候補として再利用する。**基準は表示名が同じ別IDの2 Nodeと、同じ向き・同じ端点を持つ別ID Edge 2本（e-1/e-2）**であり、n-1の出辺は2件、n-2は0件である。別ファイルの`edge-reversed.tei.md`ではe-2を反転し、両Nodeの出辺が各1件になる。以前の設計文にあった「同方向2本＋逆方向1本」の**3辺構成はこの固定Fixtureに存在しない**ため、初回の期待結果へ混入させない。追加の第3辺が必要なら別の明示Caseと独立Expectedの採用判断を要する。現在の[独立Expected JSON](../../testdata/tei-md-graph/independent-expected.candidate.json)は`proposal-only`であり、正式Owner採用済みの検収Oracleとは呼ばない。「出辺ID集合と件数を求める」pure Operationの実Go生成へ進む際、新しいExpected正本やGraph Compilerを作らない。

最初に比較するのは、(A) 明示Roleを持つMarkdown表、(B) 同じMeaningを表すインデントリストまたはMermaid、(C) 既存DSLを含むFenceである。**Aを専門家向けの優先候補、Cを現行Parserへの最低限の互換経路**として同じ一機能で評価する。Bのうち同一端点の多重Edge IDをLossなく表現できないものは、Viewerまたは部分候補に留め、実Go生成用Sourceとしての無条件採用をしない。表の列名、Roleの書式、Mermaidからの同一性付与は未採用の構文候補であり、この節では正式Grammarにしない。

| 比較する変更 | Source→Meaningへの期待 | Generator／Consumerで独立して検査すること |
| --- | --- | --- |
| Markdownの文章・コメント・図の配置だけ変更 | File Revision/Spanは更新するが、必要Meaningと依存が同一なら意味差なしの候補 | 旧Build skipが可能かは閉包で判定。UI再描画だけを再生成成功と呼ばない |
| Markdown表のEdge ID/向き／Rule変更 | Sourceの選択範囲、Relationの向き・多重性、必要生成入力が変わる | 生成Goの**実処理**が変わり、CLIでの実Call結果も独立Expectedに追従する |
| Mermaidを直接編集、または別SourceからMermaidを再生成 | 編集対象がAuthoritativeな場合のみMeaning更新候補。導出図の見た目だけなら意味不変 | GraphのID/向き/多重性欠落を保持できないときはlossy診断。元のSourceや元意味を黙って上書きしない |
| CSVのリンク先変更・部分取得・A→B→A | 外部Source Revision/取得時点/Scope/読み取った範囲を別記録 | 未取得を空集合とみなさない。Build/Evidenceの再利用とConsumerの現在資格を別判定 |

一機能の実装完了の条件は、「Markdownが見えた」「Source Indexにある」「Goを生成した」ではなく、**明示Source→Parser/Role→Meaning→実Go処理→選択Plugin→実Provider Call→通常CLI受領**の同一対象/Revision/Attempt照合である。後続のStage1 TEIによるStage2再構築とStudio WebView2使用は別の上位資格。Graph Pluginの詳細とPlugin文書は#4689/#4691/#4679へ渡し、Reader側でPlugin Runtimeを再実装しない。

### 10.10 Plugin文書・CLI／Studioへの引渡しは、Sourceと契約の現在版で結ぶ

Graph Pluginの出辺Operationが利用可能になったら、同じSource/Roleと**実選択されたPlugin Contract・Provider・Artifact**を起点として、人間とAIが仕様・利用例・依存先・制約を辿れるか確認する。既存の`plugin-lock`／`plugin-resolve`／`plugin-inventory`など、現行CLIで確認できる入口を最初に再利用する。**未実装の`plugin docs`コマンドを存在するAPIとしてSkillや手順書へ書かない。** 詳細マニュアルは契約に結び付けて参照し、短いProduct Skillは「対象選択→Inspect→説明参照→独立Expected→実行資格」という導線だけを担当する。上位Pluginの文書に辿れることと、内部SourceやHost権限を得ることは別である。

CLIの最初の実行基線は、ADR-0048のWails非依存・ローカルin-process Application Operationを保持する。Studioを子Backend＋IPCへ分ける設計目標、CLIの任意の子Process/Session/Remoteは#4690/#4697/#4702/#4705の別Ownerで比較し、**Markdown Readerや初回Graph Plugin生成に常駐Serverを要求しない**。後続で配置を変えても、同じSource/Contract/選択Provider/Actual Call/Consumer結果の意味を保ち、現在のGrant・Plugin Lock・Server世代・Effect未確定を混同しない。

**更新の資格**：§10.8〜10.10は、他チャットの自然なMarkdown/図表を重視する方針と、main採用済みのCLI-first／Graph/Role/Source/Plugin各計画を既存のReader移行設計へ接続したもの。今回はGo Reader/Generator・Plugin文書取得API・CLI/Studio子Server・Owner受理の新実装やE2E試験を行っていない。各工程の合格は現在の実Source・独立Expected・実Consumerで別途証明する。


<!-- tei-md-graph-fixture-guard-alignment-20261114 -->
### 10.11 実Fixtureと未束縛Readerの固定負例を一致させる（2026-11-14）

初回Graphの記述を、現在mainに存在する[固定Fixture](../../testdata/tei-md-graph/README.md)と[独立Expected候補](../../testdata/tei-md-graph/independent-expected.candidate.json)に合わせる。5ファイルは`baseline`、`edge-reversed`、`layout-only`、`duplicate-edge-id`、`diagram-stale`であり、**表が正本候補、Mermaidが導出表示**という採用前の区分を保持する。MermaidのLR/TB変更は表示差であって意味の出辺差ではなく、図が古いときも定義表の向きを図から上書きしない。Nodeの表示名は両方「対象」だがIDはn-1/n-2であり、同端点EdgeはID別に保持する。正式Readerでは、表の役割と出典位置を選択できること、重複IDを一辺に黙って縮約しないことを検証する。

本計画の最初の**実Go回帰**は、[PR #4713](https://github.com/hat47x/tei-platform/pull/4713)が追加した固定5ファイルを現行Workspace Loaderへ入力するテストである。管理Rootに配置した`*.tei.md`は引続き`TEI_MD_PARSER_UNBOUND`でSource位置付き拒否、管理Root外のMarkdown草稿は旧`*.tei`実読込みを妨げない。これは正式Readerが表・Mermaidを理解した証拠ではなく、**未対応が無音成功へ退化しない**という限定的な負例／正例である。PRのGo/CI結果とmain統合資格は別途判定し、未完了の時点でPASSや製品意味受理に昇格させない。

次にMR-1/2で、まずこの`baseline.tei.md`の定義表の**選択されたByte範囲／セル位置とRole**を読み取り、固定Node/Edge IDの出典と独立Expected候補へ対応付ける。Parser／Roleが正式未採用なら`unsupported`を返し、必要に応じて既存DSL FenceによるSource Span試験を技術的先行条件として切り分ける。原文表のReaderと既存DSLを埋め込む互換Readerを**混同しない**。その後にGraph Meaning→Go実生成→Plugin実Build/Actual Call→通常CLI Consumerを順に検証し、Stage1からStage2への再入およびStudio Windowsは各自の上位資格を保つ。

既存の#4709／#1180／#2514がFixture／Expected採用の担当であり、#3030／#2621／#988／#2620がLanguage・Reader／Roleの担当である。将来Readerが正式導入された際には、#4713の固定「未束縛拒否」テストを、**選択Parserがある場合の正例と無い場合の拒否**へ改訂する。テストを黙って削除して0件成功としない。

<!-- tei-md-reference-to-product-reader-seam-20261115 -->
### 10.12 限定Graph Readerがmainへ入った後、製品Readerへ渡す最初の実装境界

2026-11-14、[Reference Graph Reader #4722](https://github.com/hat47x/tei-platform/pull/4722) が**mainへ統合済み**となった。実Goの`reference/tei-md-graph-reader/graph.go`は、既存の[固定Markdown Graph 5例](../../testdata/tei-md-graph/README.md)について、**見出しを固定した二つのMarkdown表**からNode/Edge候補を取り出し、原ファイルbyteの半開区間`[start,end)`と行位置を保持する。Node名の重複と同端点の有向多重Edgeを区別し、Edge ID重複、未定義Node、未対応の表形式を拒否する。`CheckDerivedView`は明示された表示Aliasを使って限定Mermaidを照合するが、表示図をMeaningの正本へ昇格させない。これらは**限定Reference実装・固定Fixtureに対するGo試験の成立**である。#4722のexact HEAD `59ee9c2412f030c927b02488f70e0426681b4cc0`は取得時3/3 Workflow success。

一方、現mainの`internal/workspace/loader.go`は管理対象の`*.tei.md`を物理的に見つけた直後に`rejectUnboundTEIMarkdownSources`を呼び、**未選択ParserのSourceを`TEI_MD_PARSER_UNBOUND`で拒否する**。Referenceをmainへ置いただけで`Load`に接続されたのではない。PR #4713はこの実Fixtureを管理Rootに置いて拒否と非管理Draftの隔離を試す**別Draft**で、取得時のexact HEAD `7bbaaa636a83af84ecbbec6d55384cf09685b1aa`は8/8 Workflow successだが、正式Readerの正例ではない。

したがって次の一実装は、Referenceの表解析をそのまま第二の正式Parserへ格上げせず、**既存Workspace／Language Plugin／Role／Source Indexのどこへ、選択された候補と元byte位置を渡すか**を一つの実Loader経路で示すことに絞る。具体的には次の受入を段階ごとに分離する。

| 区間 | 既存の足場と実装すること | 実際の合格条件・負例 |
| --- | --- | --- |
| **GMR-0 Inputを固定** | `tei.workspace.yaml`の実Source Root/Workspace Pin、同一Loadの`sourceSnapshot`・ManagedFileを使う。宣言外Draftや旧`*.tei`は従来どおり | 管理対象に発見しただけで`ReadByLoader=true`にしない。別時点のDisk Readや原文を文字列化した後の推測byte位置を実消費Sourceとしない |
| **GMR-1 限定候補を取得** | 実選択されたSource/Profileに限りReference相当のNode/Edge候補・byte spanと限定診断を返す。Reader自体を第二の正本にしない | Node名「対象」の二件をIDで区別。同端点の`e-1/e-2`を保持し、Edge ID重複/未知端点/未対応表は拒否。Mermaid不一致は**表示差**として報告し、表を上書きしない |
| **GMR-2 Role/Meaningへの対応** | #988/#1018/#2620の選択Language/Version/Roleに従い、候補を**受理済みの既存Semantic/Relation表現**へ対応付けられる範囲だけ進める | `Candidate`型が読めるだけでは正式Meaningではない。既存表現でEdge ID・向き・多重性を保持できない場合は`lossy/unsupported`を明示し、図表示の成功で穴埋めしない |
| **GMR-3 Source Indexと実消費** | 正規`workspace.Load`から元Source occurrence/byte spanへ辿り、`ManagedFile.ReadByLoader`、選択Subject、Dependency/Build入力を用途別に観測 | Source Index内に参照があることと、Builderがそのbytesを使ったこと、OwnerがMeaningを受理したことは別。Parser未選択なら従来Guardを維持する |
| **GMR-4 次段へ引き渡す** | 第44節／製品SB-1〜4の固定Graphから、`outgoing(nodeID)`のRuleを**実Go判断ロジック**として生成、選択Plugin/実CLI Consumerへ送る | Referenceの`Candidate.Outgoing()`をそのまま既存Goから呼び出しただけ、あるいは固定JSONを返しただけなら**TEI-native Code Generationの合格ではない**。Stage1→Stage2はさらに別の検収 |

最初の実Go変更では**GMR-0/1と、GMR-2への対応可否（Fit/Gap）**を対象とし、GMR-3/4を一つのPRに詰め込まない。選択構文の正式採用前にReference Readerを全`*.tei.md`へ無条件適用すること、`TEI_MD_PARSER_UNBOUND`の無音削除、汎用Markdown AST/Graph Compilerの先行構築は避ける。GMR-2で受理可能なGraph/Roleの既存契約が不明なら、その不足を#988/#2620へ返す。

**Independent Expectedの出所は変えない。** `independent-expected.candidate.json`は提案であり、Referenceの出力値をもとに業務Expectedへ格上げしない。五つの原本（基準・反転・layout-only・ID重複・古いMermaid）と、BOM/CRLF/日本語・引用文・未閉Fenceの追加負例を既存#1180/#2514へ引き渡し、同Source Revision・Parser選択・Role・Consumer Attemptを固定して比較する。表のRelationshipを出典へ逆引きできない場合は編集不可とし、別のWriterやSource Registryを新設しない。

**Owner／現在の資格**：#3030/#2621はAuthoring/Reader、#988/#1018/#2620はLanguage/Role、#2823はRead Coverage、#2497はGraph identity・逆編集、#2595/#2600は実Build消費と再利用、#4544/#951は実Generator/Plugin、#1180/#2514は独立Expectedを担当。第10.12節は**Reference実装のmain統合を受けた製品Readerの実装引渡し**であり、本節自身が新しいGo製品Reader、生成Plugin、通常CLI実Call、正式Studio Stage1→Stage2、Owner受理を実行・完成させたものではない。P0は引続き#4583/#2831と#2586/#1251の二経路を優先する。

<!-- tei-md-ref-fence-hardening-20261115 -->
**表示中の定義を誤読しないための追加回帰**：[Draft #4725](https://github.com/hat47x/tei-platform/pull/4725)では、main統合済み限定ReferenceのMarkdown Fence認識を補強し、チルダFence、4文字以上の開始記号、短い偽の閉じ記号、インデント付きコード、未閉じFenceに含まれる「ノードの定義」等を**意味候補に取り込まない**ようにするGo変更と試験を追加した。既存固定5例やBOM/byte-span試験に加え、例示の内容が正式Source扱いにならないことを確かめるもので、正式CommonMark/GFM Parserの実装ではない。このPRのCIとReviewは別HEADの資格として確認し、**mainへ統合されるまでReferenceに採用済みの保証として加算しない**。#4719の写真・会議・表など多経路のAI候補も、元Source位置／選択Role／Owner受理が揃うまでは同じくDraftのまま扱う。
