# 標準表現型：合成写像の適格性・非単射・否定依存・開示（2026-11-07）

- **状態**：追加の横断設計／検収計画。新Core Schema/Plugin Contract、Engine、独立Test SSOT、Host、Owner業務意味の正式採用は**未**。CX-01〜10の新規実行は**NOT RUN**。
- **由来**：[旧Draft #3192](https://github.com/hat47x/tei-platform/pull/3192)（Data/Representation二つのTopology、Function-owned instance、媒体・生成軸）と[旧Draft #3222](https://github.com/hat47x/tei-platform/pull/3222)（複数Viewの選択・状態・同期）を**現main起点の一本の比較・検証計画へ統合**する。両Draftがmergeできる状態との主張ではない。
- **重複回避**：#2495=Data Facet、#2497=構造保存写像/情報損失、#2506=Function Data Context/Result Slice、#2507=Function Pattern/表示・適合、#2582=変更影響/再取得、#2500=View→Intent、#2598=実Request対象、#1174=実Effect/Receipt、#2516=用途相対Disclosure、#997=UI/Target Adapter、#2586=実Function→Provider Read、#2514/#1180=独立ExpectedとScenario、#2621/#1124=Source Authoring/Meaning Admission。異質ConsumerはSHO #19、SUI #3391/#3392。これらの担当の第二正本を作らない。

## 1. 合成可能な表現は「局所的に表示可能」と同義でない

SourceのData StructureをD、Function-owned Representation InstanceをV、そこから生成する文書/音声/外部Consumerの表示をWとする。

- **第一の対応** P: D → V：Entity/Relation/Path occurrence、grain、NULL/欠損/未取得、複数集合、単位、順序、時点、元SourceやResult SliceのScopeを条件づけた *部分写像*。
- **第二の対応** Q: V → W：Page/Outline、Chart、Tree→Markdown、Audio/video playback、SUI Canvas→SHO Draft、CUI等への写像。媒体・Target・ConsumerのDisclosureを追加条件とする。
- **連結** Q ∘ P: D → W：PとQの個別成功だけでは、Wについての業務Subject同一性・情報完全性・逆編集可能性・現在の表示資格は主張できない。**途中で失われた識別・関係・順序・時点・Scopeは、明示した追加出典と再資格化なしには下流で回復しない。**
- **複数の枝** D → V1, D → V2：両者の「表示成功」は相互同期・同一Read Snapshot/Owner/Target・同じAction資格を意味しない。Context/Read cohort/Disclosureの一致や、同一性の部分写像を別に照合する。

「構造を保つ」は**Claimと用途に相対的**。たとえばTree→Outlineは「読める文章」には適しても、「元Graphの全edgeを復元できる」には不適合。集計棒は合計を示せても「行単位でPATCHできる」には不適合。新たに全構造を一列の情報量順序へ並べない。

**Compositionの検収用論理列**（新API/Enumでない）：
1. 入出力：Source Dの実版/Subject、P/Qの実選択Implementation・Versionと入力版、Function/Result Slice/Target Consumer、use-site。
2. 各段の必要条件：Identity/Occurrence/Grain/Multiplicity、必要属性/否定依存、Relation path、Cardinality、全件閉包/部分集合、時間/単位/次元、Presence/Disclosure、生成軸とalignment。
3. 各段の保証：preserved／lossy／created／unknown を**各意味Claimごとに**記録（固定共通enumでなく説明票）。Dの同一性やread completenessなど、Pが保証しないものをQの「入力保証」として補わない。
4. 逆方向：Wのselect/focus/drag/text edit→元Dへの**一意な更新対象**を証明できる範囲と、明示Action/Owner適用が必要な範囲。一般的な可逆Lensを全Sourceに仮定しない。
5. 最新資格：Source・Query・Function Context epoch・Plugin/Renderer・policy/grant・Consumer開示・生成器/voice/clock・対象Scopeが変化した時点を区別。A→B→Aやbyte一致で旧資格を自動復活しない。
6. Evidenceと非主張：Static/pure mapping／限定合成test／実Provider read／実Consumer受領／実Target操作／Effect Receipt／独立再Read／Owner Meaning Acceptanceを混ぜない。

## 2. 非単射写像は「同じ表示から同じ世界を推論しない」

[SUI #3391](https://github.com/hat47x/sui-sensemaking/issues/3391)、SUI #3392、TEI [#2497のSUI→OKF引継ぎ](https://github.com/hat47x/tei-platform/issues/2497)を圧力事例とする。

- World A：Card Xに異なるIsland所属と2本のPath occurrenceがある。
- World B：同じ文言のCard Xが別のIsland構成・非階層Edgeを持つ。
- A/Bの文章化先は**同じMarkdown見出しと本文**を生み得る。よって `render(A) == render(B)` でも `A == B` は言えない。
- 下流Markdownが見出しを移動しても元のどのIsland/Tree Path/Graph Edgeへ変更を戻すか一意とは限らない。Source RewriteやBusiness Mutationを自動構成しない。
- たとえOKF文書側が構造検証PASSでも、出典のどの部分が表現・非開示・損失・未観測なのかと、Attested Computation/実Executorの成功は別資格。文書の`generated`/`verified`表記を実Receiptとしない。

**合成を許す条件**：必要なClaimを保持したこと、必要な元Subject/Occurrence・Context/Sourceの参照を**資格付きに**保持できること。Lossで正しい逆写像が一意に決まらない場合はread-only、または元Consumerへ戻す候補・明示Actionに留める。

## 3. Empty/Hidden/Partialには「出力に現れない依存」が残る

結果が `[]`、空Chart、非表示Tab、Graphで見えないEdge、0 Link候補、非表示のHidden Componentであっても、次の違いを保持する。

- 本当に母集団に対象が存在しないと**閉じたScope・完全なRead**で確認した場合。
- $top/ページ/Window/Filter/Query predicate/Rank/Temporal/Join/Relation Traversalで対象が表示されなかった場合。
- 未実行・未取得・Source未解析・Parser/Grammar未対応、Provider unavailable、非開示Policy、古いCache/epochだった場合。

Outputに現れない列・表現には、**否定依存**（述語列、空集合成立に必要なRelation path、Query/Provider route、Permission Scope、時点、外部Source/Parser Selection、集合の閉包）が残る。QueryのProjection列だけで「関係無し」「対象無し」「変更影響無し」と主張できない。

- **実例**：D1を完全取得してChart合計を出し、D2は1/3しか取得できない。D1-only合計は保ち得るが、D2が関係しないことを前提とする`NOT EXISTS`表示・実行Offerは不適格。
- **実例**：SQL結果が0件でも、同じSQL IDに別Tenant、RLS、search_path、別Datasource binding、read replica、許可されないfieldがあると、全世界での不存在Claimにはならない。
- **反例**：入力3件で見えた3行が、JOINによるL1の3出現か、別の3 Entityか未確定の場合、表現型の行数をEntity Count/更新対象へ昇格しない。

**対応方針**：#2387/#2506/#2582の既存Query/Context/Claim/Evidence参照を使い、第二のNegative Cache・Proof Engine・Completeness Registryを実装しない。

## 4. 選択・操作モードはPresentation state、しかし実行への影響を持ち得る

同じ表現型が選択可能でも、`select`、`focus`、`hover`、`seek`、`navigate`、`edit`、`run`は異なる意図。表示モード（閲覧/編集/実行）、Research/Full Auto等の操作モード、前回の未確定Action Offerを、**単なるUI装飾やGrantの永続表現として扱わない**。

- 同じActionカードがResearch表示では提案だけ、Full Autoでは実操作を起動し得る。モードの残留・遷移後には**現在のUser/Host Grant、対象/Request、Effect範囲を発行地点で再確認**。表示上の`enabled`やMode名を認可としない。
- Multi-Viewのselection A→B→Aは同じIDでも別epoch。旧Response/Tool Offer/Receiptを現行対象として継承しない。
- 表現型間の状態同期自体はProviderの実Readや業務保存を生じさせない。明示Operationを要求する場合だけ既存#2500→#2598→#1174経路を使用する。

## 5. 開示は「値」だけでなく全出力面に相対的

Private World W0/W1が**当該Consumerへ開示可能な情報としては同じ**であり、違いが非公開Edge/Card/Role/Provider候補/件数だけなら、Observerが見える出力は、公開資格の範囲で同一であるべきという**非干渉の検証仮説**を置く。

対象はHTML/DOM、a11y tree、Chart/Graphの配置・空白・件数、tooltip/hit map/selection候補/diagnostics、Clipboard/SHO Markdown/OKF export、Tool/Agent message、Session Log、network/effect request、receipt preview、再生メディアのメタ情報等。Timing・資源使用差は別の計測要件として未実証と明示する。**表示BodyをRedactしただけで全送出が安全とは宣言しない**。

全Worldを投影器へ渡す必要がない場合、先に資格化したPublic Sliceから投影する。発見できたSource・読んだSource・Consumerに配送したSourceの区別も保持する。診断に秘密の存在や個数を吐く設計は不適格。これは新Policy Engineでなく既存#2516/#2515と各Target/Hostの検証範囲。

## 6. 横断受入：同じ反証例を二重に作らない（CX-01〜10、NOT RUN）

| ID | 対照する一因子 | 独立Expectedで判別すべきこと | 担当 |
| --- | --- | --- | --- |
| CX-01 非単射Outline | Tree/Islandの異なる2世界→同じMarkdown | 同じbyte≠元Topology同一。書戻しはunqualified、元出典とloss保持 | #2497、SUI #3391 |
| CX-02 集計逆編集 | L1+L2=200 / L1'+L2'=200→同じChart点 | 同じ点から行ごとの更新は導けない。明示ActionなしにPATCHしない | #2500/#2507 |
| CX-03 空/否定依存 | `NOT EXISTS D2` をD2 1/3または未取得で表示 | partial/unknownを真の不存在にしない。必要なPredicate/Scopeを証拠に保持 | #2582/#2506 |
| CX-04 非開示側路 | 2つのPrivate World、Public Sliceは同じ、hidden edgeだけ違う | 件数、tooltip、selection、diagnostics/Logで秘密差が漏れない範囲を独立観測 | #2516/#997 |
| CX-05 操作モード残留 | 同じActionカードをResearch→Full Autoで使用 | 表示可能・提案・Dispatch可能・現行Grant・実Effectを別判定 | #2507/#2598 |
| CX-06 Context ABA | H1→H2→H1、最初のH1 responseが遅れて到着 | 旧Epoch/Read cohortを新H1へ復権しない。別Windowまで失効しない | #2582 |
| CX-07 文書→TTS再生成 | Source span同一でvoiceとtimebaseだけ変更 | 原文継承と新Playback alignmentを分離。旧Seek位置は無資格 | #2497、SHO #19 |
| CX-08 部分合成 | D1 2/2、D2 1/3、D1×D2→Table→Chart | D1-only Claimの保持とD2依存Claimの停止を区別。見える行数≠Coverage | #2506/#2586 |
| CX-09 Source対編集 | Canvas配置/SHO Plain Markdown/TEI `*.tei.md`同じ表記 | UI配置、Raw保存、Semantic Promote、業務Actionを別操作・版として扱う | #997/#2621/#1124 |
| CX-10 旧新実装資格 | P/Q Renderer/Plugin/Policy R0→R1→R0、見かけの結果同一 | 出力同一で旧Evidence、Owner/Grant、実Consumer/Targetを自動再資格化しない | #2497/#2514 |

独立Expectedの原理・判別力とholdoutは#2514、Scenario SSOTは#1180。固定合成World/Source rev・変換器版・用途/Consumer/Disclosure・検査するoutput surface・書戻し候補の不許可を**既存Fixtureの差分**として渡す。CX-01〜10はテストケース名の整理にとどまり、新たなテスト正本を作らない。

**まず採点を分ける**：(a) shape適合、(b) Semantic/Roleと必要依存の正しさ、(c) Loss/Completeness/Unknown診断、(d) 出力面の開示非干渉、(e) 候補Operationの適格、(f) 同版Source→実Read→実Consumer、(g) 実Request/Effect/Receipt/独立read、(h) Owner受理。局所合成PASSを他の列へ足し込まない。

## 7. 段階と引継ぎ

- **P0（read-only）**：旧PR #3192 RPと旧PR #3222 MVを現main上の一本のComparisonへ統合。最初にCX-01/02/03/08の非単射・否定依存・grainを同一公開合成fixtureで説明する。#2497/#2506のOwnerと衝突しない。
- **P1（証拠付き表示候補）**：CX-04/05/06/10のDisclosure/Mode/Epoch/VersionをCUI/Headlessへ接続。開示未確認はunverifiedへ停止。表現型の条件とAction Permissionを結び過ぎない。
- **P2（SHO/SUI/Media）**：CX-07/09とCX-01の別Consumer比較。SHO Raw DraftはTEI Semantic Admission不成立でも保存できる。SUI HybridのTree+Islandは非階層Lossを出典付きで残す。
- **P3（実Provider/Operation）**：#2586既存Function→Query→Provider実Readから、Result Slice/Context→複数Renderer/Consumerへ同一Attemptで辿る。編集を伴う場合のみ#2500→#2598→#1174の独立Action/Receipt/readと結合。未完成な`*.tei.md` Loader/Parser、未確定Datasource、Source Revisionや現行Grantの不足をMock/形式的一致で補わない。
- **終端条件**：実Go/CI、Product Provider/DB、SHO/SUI受領、実Media生成・再生、実Policy/Disclosure、Owner意味受理は**個別に証拠が揃うまでNOT RUN/UNVERIFIED**。旧DraftのPASSや新本文の公開によって現在合格に変えない。新Gate/第二Registry/新共有Hostは増設しない。

### 文献（設計の参考であり実装採用ではない）

- Horn, Perera, Cheney (2018) *Incremental Relational Lenses* — View updateのsource deltaへ戻す条件を、一般のUI編集の自動許可とは区別する。
- *A mathematical foundation to support bidirectional mappings between digital models* (2022) — Delta lens合成に法則・前提が必要な例。TEI全RepresentationにLens法則を強制しない。


## 8. 最初の製品実証を一つに絞る：実Read→Context→二表現（2026-11-07追補）

前段までのRP/MV/CX群は**要求探索の広さ**を維持する。一方、初めて採用資格を与えるための縦断実証は「1 Function・1 Query・1 Provider Read・1 Context・2 Read-only Representation」の独立した一試行を最優先とする。これは検証を狭くするためではなく、Source/Provider/Contextの**最初に欠ける接続辺**を一つ特定してから、同じ証拠を異質なConsumerへ拡張するためである。

### 8.1 現mainで確認された既存機構と、依然として欠けるもの

- [main統合 #3195](https://github.com/hat47x/tei-platform/pull/3195) の `tools/reference-operation-pattern-crosswalk` は既存 `operations.DeriveOperationSurface` を再利用し、PatternとOperationをData/Capability/Relationの条件で突き合わせる。実Go参照出力は8 Operation／5 Pattern、4 link。これは**候補の照合**であり、Product Function/Action選択やRuntime Readの証拠ではない。
- [main統合 #3204](https://github.com/hat47x/tei-platform/pull/3204) は同Dataに対する複数browseの競合、Effectless command、Persistence入力不足等を識別する。候補が複数なら**first-winner選択**して成功扱いにしない。
- [Draft #3238](https://github.com/hat47x/tei-platform/pull/3238) は既存Web API参照の限定的なPostgreSQL行取得を試験候補として追加。**実Product Functionからの選択Query→Provider→Result Slice E2aとは別**。Draft新HEADのPG/Go CIや実行結果を再確認せずPASSへ計上しない。
- [main統合 #3155](https://github.com/hat47x/tei-platform/pull/3155) では `receipt-observed` と `resource-observed` を分離した限定実装が存在する。従って「独立Resource ReadがTEI全体で未実装」は誤りだが、**受注FunctionのSQL Rowset/Provider Scope→Context→画面、保存後業務完了**は未実証。参照PG Effect/Resource観測を本CaseのPASSへ転記しない。

### 8.2 Product R1：最初のActual Read Witness

主Ownerは既存[#2586 HR-01/02](https://github.com/hat47x/tei-platform/issues/2586)。対象を `order_entry.recent_orders` の**注文Header**に固定し、まずread-onlyで次を同一Attemptに接続する。

| 順 | Witness/受渡し | 明示すべき停止線 |
| --- | --- | --- |
| R1-0 | 実際のWorkspace Root／Source raw revision・Loader実消費・Parser結果、選択されたProduct Function/Pattern IDと必須Role | 宣言・Crosswalk link・`tei operations`列挙は実Product Bindingではない |
| R1-1 | Function→Query Selection、選択したSQLのSource byte/args、DB別Query選択、DataSource alias・実Physical Provider instance | `UNBOUND_DATASOURCE`、ambiguous alias、source不一致はunresolved。SQL文字列やDatasource名だけで実Providerを推測しない |
| R1-2 | 実Readerが受領したHeader行とField presence、Target occurrence/grain、Limit/Sort/Filter/Scope/Tenant/Principal・read epoch・Rowset version | Page成功を母集団完全性にしない。別Provider/replicaの同時点性を推測しない |
| R1-3 | 受注Headerに相対的な独立Expected／取得集合・必要Predicate・否定依存・Disclosureを同じContext Instance/Read Attemptへ結合 | `status`未投影は自動で「未評価」とも「評価済み」とも断定しない。Provider実評価のWitnessがあるときだけClaim候補 |
| R1-4 | 同じContext→**FormとTableのread-only二表現**へのProjection。Subject/Field/NULL/missing/order/scope/epochを照合 | 表示二箇所の一致をBusiness Acceptance・Write可・全OrderLine読取と呼ばない |

**粒度を越える最初の失敗を明示する**：Headerの有限集合の取得成功から、OrderLine全件、受注Submit、Ruleの全称、保存後業務完了を主張しない。Pattern/Operationの候補対応が曖昧な場合はR1-0で停止し、参照ツールを本番Resolverに昇格させない。実Product Readが未実装なら、R1-0/1の最初に欠けるEdgeをSource位置・既存Ownerで記録する。

### 8.3 Read-only二表現の初回合格と不合格（独立Expectedへ接続）

- **正常**：同一Header Subject/ContextからTable行とFormの対象を一致させ、Display列の欠落、Row order、Nullable値、Display-only状態を正しく説明できる。Table↔Formを切り替えてもRead cohort・業務意味の正本が増えない。
- **負例1**：Headerを2件取得したがLIMIT 1のTableを見て全件2件が読めたと報告する実装を不合格。必要な閉包はClaim相対。
- **負例2**：JOINで同一Headerが3行表示され、その3行を3業務Entity／3更新候補と扱う実装を不合格。
- **負例3**：未取得/非開示/status述語の依存をfalse・0件と表示する実装を不合格。非出力Predicateは実Provider/Operatorで評価済みの証拠があれば、秘密値をConsumerへ開示せずにClaim資格を得る余地がある。
- **負例4**：A→B→A選択の旧response・旧Action Offerが現在のContextに再入する実装を不合格。Function/Data/Query/Policy/Pluginの版も使途ごとに資格化。
- **負例5**：Tableの2表示行が同じでも、Consumer別のDisclosure/Scope/hidden edgeが異なるのに「二View完全同値」とみなす実装を不合格。

初回R1-4は**Graph/Tree/Audioの実Renderer完成を要求しない**。既存UI Kit/Projection Contractを使ったHeadless出力2種類で十分かを確認する。実Renderer・SUI/SHOの媒体表現は次段に並列圧力試験として維持し、要求探索の上限は置かない。

### 8.4 資格の階段と先行しないもの

1. **A: 構造・意味適用候補** — Declaration/Pattern/Shapeが存在し、必要情報が分かる（合成/CUI上の読み取りに限定）。
2. **B: 実入力・Provider Read証拠** — 実Source/Query/Provider/Scope/Rowsetを同一Attemptに束縛。ここで初めて実ReadのClaimを限定的に評価。
3. **C: 複数表現の局所意味保存** — 同じContextから2種類のViewを作り、対象/粒度/欠損/Disclosureのlossを診断。Bを満たす前の合成PASSはCのProduct資格でない。
4. **D: ViewChange→typed Intent→現在Request/Grant** — 読取資格から書込資格を導かず、#2500/#2598/#1174の既存境界を再利用。
5. **E: 実Effect/独立Resource Read/業務Outcome** — Receipt・Resource観測・業務完了の主張対象と証拠は別。#3155の局所成果とProduct R1/Order Entry全体を合算しない。
6. **F: 異質Targetと生成媒体** — SHO/SUI、Graph/Tree、文書↔朗読などを、Consumer別Source/Alignment/開示/実Targetの資格で再検証。

各段の比較欄は**Required / Discovered / Loaded / Parsed / Bound / Evaluated / Enforced / Executed / Observed / Qualified**を区別する。これは[#2514](https://github.com/hat47x/tei-platform/issues/2514)の独立検証と既存Evidenceをつなぐ**read-only照合票**であり、新Core Enum、第二Proof Store、追加Gateではない。ある段がmissing/unknownでも、下流の表示候補の探索自体を禁止するわけではない。ただし上位の完了Claimや自動Writeへは昇格させない。

**既存Ownerでの検収順**：#2586 R1-0〜3 → #2506/#2507 R1-4 → #2497/#2582 CX写像・非単調性 → #2500/#2598/#1174 現行Operation → #2516/#997/SHO/SUI Target。独立Expected/Scenarioは#2514/#1180へ差分として渡す。今回の追補は設計のみ、R1-0〜4を現在の実Product/DBで走らせた証拠ではない。正式Meaning acceptance、実Target/Host、Owner判断は未。


## 9. 実読取証拠の帰属・環境効果・図表Consumer（2026-11-07追補）

§8の「Product Function→実Query/Provider Read→同一Context→二つのread-only表現」を維持し、他レーンの実測から**対象の所在・実操作の効果・投影図の意味**という3つの独立した合格条件を追加する。RP/MV/CXや既存#2514/#1180に第二の検証基準正本を増やさない。

### 9.1 最新の限定実PG成果を実証の踏み台にする

[Draft #3238](https://github.com/hat47x/tei-platform/pull/3238)の固定HEAD `469cdbadb1fcf23d65526ec3c5899f4d94e8390b` では、[#2586の記録](https://github.com/hat47x/tei-platform/issues/2586)によれば独立PG Workflow [#42346176936](https://github.com/hat47x/tei-platform/actions/runs/42346176936) が completed/success、隔離PostgreSQL 16.15上でTenant Aの固定Fixture `{order3,order9}`の物理Readを実行し、`top=2`と`top=1`のResult/Completion差を観測した。**本設計作成時に「#3238のPG CI未確認」と記した旧記述は、この固定HEADに限って更新する**。

ただしこれはReference ReaderのSQL→実PG Rowset→局所Completionの限定成功。**Product Function/Pattern→Query/SQL選択→physical Provider→Function Data Context→Form/Tableの一試行接続は未検証**。実件数一致とClaim Target Set/否定依存の完備は異なる。Q07の非出力Predicateが評価済みか、Current Host Grant/Owner業務受理があるかも別の証拠とする。#3238がmainにmergeされたか・新HEADでPASSしたかは再度確認が必要。

### 9.2 WorkspaceとFunction Instanceの境界をSelectionの同一性へ接続する

[#3149のCLI Dogfooding](https://github.com/hat47x/tei-platform/issues/3149)と[#3181](https://github.com/hat47x/tei-platform/issues/3181)では、`reference/workspace` と `examples/xtea-lowcode` に同じ `sales.order` IDがありながら、CLI応答にはWorkspace名/manifest照合値がなく、内容指紋をWorkspaceの物理的所在・権限とみなせないことが実測で指摘されている。

既存のFunction-owned Representation Instance・同一Context/Selection epochの検収に次を加える：

- 同じCanonical Subject/Function名・表示値・Query IDでも、Workspace Manifest実ID/Source Root/Source byte revision、Function Context Instance、Tenant/Principal/Provider binding、Read Attempt・Scopeが異なれば、**別の観測と別の操作対象**。Cross-Workspace selectionをexactへ自動昇格しない。
- CLIがまだ出力しないmanifest identityを「応答で検証した」と記述しない。呼出し側の実Workspace/Host bindingへ戻れる場合も、Source実読取・変換・対象Revisionの根拠を提示して初めて限定的に資格化する。
- 同じIDを別Workspaceで開き直すA→B→Aは、前のAのread/selection/Action Offerを現行Aへ復権させない。UnknownはUnknownとして提示。Workspace ID/Core enum/第二Source Registryをここで新設しない。CLI契約の改善は#3181/#3242/#2824が担当する。

### 9.3 表示のread-onlyと、ファイル・通信・実業務の副作用は別

[#3182](https://github.com/hat47x/tei-platform/issues/3182)では `tei commands` の `view: effect=read-only` の広告に対し、実際の `view -execute -operation-log` や `-lifecycle-state` が指定されたログ/DBファイルを書いたことを確認している。これは指定オプションでの局所的な作用であり、全view操作が無制限に書込むという意味ではない。

- **業務上のread-only**：業務Entity/Relation/Sourceへ更新Intentを生成しない。
- **環境側の作用**：ログ・ローカル一時ファイル・Clipboard・Export/Session Log/通信など、表示可能な出力面とは別に発生し得る効果。
- **現在の実行適格性**：実コマンド＋オプション、Host Permission/Grant、SourceとTarget、実Provider／Tool作用を実行時に評価する。CLI Discoveryの抽象 `effect` や画面のenabled表示を万能な証拠としない。

二つのread-only Form/Tableを表示するだけのHeadless検証と、`-execute`／Export／外部AIツール呼出しを**同じ試行やEffectなし**と扱わない。CLI分類の語彙と実修正は#3182/#2824、Form/TableのAffordanceは#2507/#997、実Effectは#2598/#1174を正本とする。

### 9.4 新たな異質Consumer：Subsystem ER図とCRUD Matrix

[#3250](https://github.com/hat47x/tei-platform/issues/3250)の論理Subsystem分割候補とER/CRUD派生図は、TEIの標準構造と標準表現型を圧力試験する適切なConsumer。**分割Optimizer・新Core・業務Ownerの採用判断は本設計に取り込まない**。

- Entity/RelationとFunction–DataのR/C/U/D、Procedure/Owner、宣言された直接CRUD・派生可能なEffect・Provider実観測Effectを、図表の別要素・別根拠として保持する。Missing/Partial/非開示/未観測Effectを「越境更新0件」や「外部参照無し」としない。
- ER図のノード移動・色/並べ替え/折畳みは**Presentation state**であり、Subsystem移管やRelation削除ではない。図上でOwner/所属を変える編集は、元Source/ProposalとInverse Mapping資格を要する。Read-only図が出せることは自動Source Writeの資格でない。
- 複数Subsystem図への投影では外部Refや越境Read/CUDを黙って省略せず、必要ならLossとDisclosureを明示。論理Subsystem≠DB物理配置≠Host/Tx/Commit。#3250/#1132（分割評価・診断）、#2497（Loss）、#2507（Binding）、#2500（逆編集）を再利用する。

### 9.5 最小判別票（独立Expectedに一因子だけ追加、すべて新規試験NOT RUN）

| Case | 変更する一因子 | 必須Expectedと拒否・停止線 | 既存Owner |
| --- | --- | --- | --- |
| WX-01 | 同じ `sales.order` を2 Workspaceからinspect・Contextに配置 | ID一致やworld hashだけで二Viewの現行Subject/Scope/Grantを統合しない。source/manifest帰属不足はunresolved | #3181/#2506 |
| WX-02 | Workspace A→B→A、旧AのResult Slice/Action Offerが遅延到着 | 同名・同値でも旧epochを現在のFunction Instanceへ採用しない | #2582/#2598 |
| WX-03 | Headless Form/Table previewと、`view -execute`＋明示ログ出力 | ビジネスread-onlyと環境file effectを別に観測。Discoveryのeffect一語でゼロ副作用と主張しない | #3182/#997 |
| WX-04 | 同じER Graph/CRUD Matrixでレイアウトだけ変更 | Source/Owner/Relation/CRUD責任は不変。導出図の見かけの移動を元データPatchにしない | #3250/#2497 |
| WX-05 | 越境CUD/外部Relationがpartialまたはhidden | Diagramが0 edgeでも、完全閉包がなければ越境CUD 0/孤立Entityを確定しない。開示漏えいも別評価 | #3250/#2516 |
| WX-06 | 実PG Reference全2件 vs Product Function実Read未接続 | 前者の局所PASSを保持し、後者のR1-0〜4は未実証と表示。2件がFull CohortかはClaim・Read Scope相対 | #2586/#2514 |

実際の一Attempt証拠としては `Workspace/Source → Function/Query selected → Provider physical Read/Scope → Result Slice → Context → Form/Table Projection → Consumer` の最初に未接続の辺を**一箇所だけ**記録し、同一試行に含まれない物理Read・Effect/Receipt・独立期待値を合算しない。実Product R1/二表現/WorkspaceまたぎGUI/ER Editorの検証と正式採用は今回NOT RUN。

