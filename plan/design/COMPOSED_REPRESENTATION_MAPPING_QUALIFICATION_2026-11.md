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



## 10. 実SQLの出典を分ける：Product Queryと限定PG Referenceの非同一性（2026-11-07追補）

§8のR1実証を実ファイルに結び直した。**同じLogical Query IDの登場**、**同じsales_coreに関連する読み取り**、**実PGから受け取ったRowset**を、同じSource/Statement/Target集合の実行と混同しない。初回Readが曖昧なままForm/Table二表現を同じContextへ束縛するなら、表示値が合ってもProduct E2資格は不成立。

### 10.1 実mainで確認した入力の相違（現時点の固定Blob）

| 観測項目 | Product Source：現main `reference/workspace/` | 隔離PG Reference：現main `reference/web-api-contract/` |
| --- | --- | --- |
| 定義元 | `applications/order-entry/queries/recent_orders.tei`（Blob `a570bceddc7d61aae62ac315a886cc9aedcac082`） | `fixtures/postgres_scoped_orders.sql`（Blob `5113f47d131da7c6b00c7f00dd67fcd7298a53da`）、`scoped_postgres_integration_test.go` |
| 実SQL出典 | `sql postgres file "recent_orders.sql"`。SQL別Blob `ecc4fb7daab99bd8f03d490f56f7af47c9ebd97a` | `ScopedPostgresOrdersReference` が隔離PG上で動作する専用実装 |
| 物理的なFROM/対象 | `FROM public.orders AS o` | `public.tei_web_scoped_orders_reference` を試験用にCREATE/INSERT |
| 選択/絞込 | `WHERE o.ordered_at >= :from`。`ORDER BY o.ordered_at DESC`。SELECTには `order_id, customer_id, ordered_at, status, total_amount` | Tenant A/B専用Scope Authorityで `tenant_id` を制約。ReadPlanの `$select`、`$orderby=orderId asc/desc`、`$top=1` 等を比較。試験では `from=2026-10-01T00:00:00Z` を受渡す |
| Target/認可 | Queryは `datasource sales`。Applicationは `database sales { target sales_core; access read-write }` を宣言 | TestのProviderも `Database: sales_core` を指定するが、Authorityは固定合成 `isolatedTenantScopeAuthority` であり実Product/Host Policyではない |
| 成功の上限 | 実Sourceは存在する。現時点でProduct Function選択→そのSQL byte/args→実Physical Providerに渡った証拠はない | 固定HEADのDraft #3238／PG Workflow #42346176936に隔離PGの限定PASSがある。実行SQL/Source/Targetの違うProduct E2 PASSへ転記不可 |

**重要**：Product SQLの目視上 `tenant_id` 条件が無いことから、実ProviderのRLS/ACLが無いとは推論しない。逆にApplicationの`database sales → sales_core`宣言、ReferenceでのTenant test Green、同じQuery IDの出現だけで、Product実Providerが当該TenantへScopedとなったとも推論しない。実際のSource/Physical Statement/Parameters/Policy/Readerの観測が要る。

### 10.2 初めて欠ける受渡し辺：Queryの選択と実行を同一Attemptで結ぶ

Product R1-0〜4の検証では、独立Referenceの `ExecuteScopedCanonicalRead` をそのまま「`order_entry.recent_orders`が実行された」と呼ばない。初回E2の実証に必要なのは以下の**実際に消費された**結合である：

`Function/Pattern instance → selected Query ID/source rev → SQL file resolved actual raw bytes → runtime parameters (:from) + datasource sales binding → selected physical Provider/actual statement + bind → observed Rowset/Field/Scope/read epoch → Function Data Context/Result Slice → Form & Table`

- `recent_orders.tei`・`recent_orders.sql` のSource両方の実取得/解析/実消費と、SQL-only変更の失効を分離。名前の一致やLoader列挙だけで実実行Inputとはしない。新`*.tei.md`移行後のReader/grammarは#3030/#3031/#2621に委譲し、旧Source正常動作から新形式PASSを推論しない。
- `:from`の値・時区間・Timezone・NULL/未供給時の扱いが未確認なら、`SQL_INPUT_UNRESOLVED`相当の**未資格**として扱う（この語彙は診断候補であって公開Error Code採用ではない）。
- `ordered_at >= :from`という窓の外側についての不存在や、先頭ページから母集団全体についての全称Claimを立証しない。SQLの `ORDER BY ordered_at DESC` とReferenceの `ORDER BY orderId asc/desc` は異なる。ページ内先頭が同じIDになる場合も、選択順序・全件被覆の同値証拠にはしない。
- `result.status`がSELECT Mappingに現れることと、業務Predicate `status=READY` が**当該Read Attempt/Scopeで実評価されたこと**は別。必要なPredicate witnessはWHERE/Operator/実比較で確認し、必要以上にstatusを低権限Consumerへ公開しない。
- Headlessで同一Context/Read SliceからForm/Tableへ投影する場合でも、両Viewの**個別受領と版**を照合する。Context IDが同じだけで別Read/Replica/時間窓の**joint snapshotが成立する**とは推論しない。両者で同じRead Resultを使うのか、各々が再Readしたのかを明示。共同Snapshotが必要なClaimだけ既存#1708/#2514の資格を求める。
- 複数候補のQuery実現/Provider Aliasがある場合は候補集合・選択根拠・不採用候補の否定依存を保持。SQLが同じでもrealization/physical DB/policyが違えば現行Effect/Disclosure資格を再利用しない。これは#2387/#2862/#2611の既存Ownerに受渡す。

### 10.3 独立Expectedに追加する最小Source差分（QS-01〜06、今回の新実行なし）

| ID | 対照条件 | 独立に期待する判定 | 所有先 |
| --- | --- | --- | --- |
| QS-01 | 同じ `order_entry.recent_orders` Query IDでProduct `public.orders`と隔離PG `public.tei_web_scoped_orders_reference` | ReferenceのPG PASSはそのSource/Providerに限定。Product function Query選択と物理実行はunverified | #2586/#2514 |
| QS-02 | `:from`を変更／欠落／誤Time zoneで実行 | Read対象の時間窓を別々に扱う。未束縛は成功とせず、当該Window外の不存在を証明しない | #2387/#2586 |
| QS-03 | `ordered_at DESC`と`orderId ASC`の同件数同ID／異順序 | Output順とPage/Topの意味を分け、同数・同IDでも順序差があるならページ同値を証明しない | #2497/#2506 |
| QS-04 | `sales_core` logical target同一、実Provider instance/Tenant/RLS異なる | 同じDB名を同じ物理Reader/Current Authorityとしない | #2611/#2516 |
| QS-05 | SQL-only byte改版、Query definition byte/IDは同じ | 実SQL選択入力とQuery SourceContributorの失効を検出。旧Model/Receipt/Output資格を保持しない | #2595/#2600/#3030 |
| QS-06 | 同一ContextからForm/Table、同じSubjectだが片方が後着/別Read cohort | 局所の値/Identityを保ち、joint snapshotと現在の共同Claimはunknown。別epochを勝手に統合しない | #2506/#2582/#2514 |

検収順を **QS-01/02：実Source→SQL/引数/Provider binding→実Rowset**、次に **QS-03/06：実Rowset→二Viewの同一性・時点**、並列に **QS-04/05：Scope・Plugin/SQL改版負例** とする。実行計画は#2586 HR-01/02を正本とし、#2514/#1180の固定Fixture/Independent Expectedを再利用。新Runner/SQL Executor/Read Registry/新Proof Storeは追加しない。

**今回の証拠限界**：GitHub上の上記Source/SQL/fixture/試験ファイルのBlobと本文を実確認した。SQLの実行結果は#3238固定HEADの公開された限定CI記録に依存する。本追補のQS-01〜06、製品Query/DB read、Function→Context→二表現、実joint snapshot/Owner意味受理は**NOT RUN／未実証**。現在mainの将来変更はBlobごと再照合する。


## 11. 実Productの非単射と操作面の版ずれを同じ資格票へ（2026-11-07追補）

### 11.1 既存の有限実測が増えた部分だけ資格を更新

| 確認された独立実証 | 実際に成立した狭い命題 | 依然として成立しない広い命題 |
| --- | --- | --- |
| [SUI Draft #3396](https://github.com/hat47x/sui-sensemaking/pull/3396)／[固定HEAD frontend CI #42345896949](https://github.com/hat47x/sui-sensemaking/actions/runs/42345896949) | SUI製品の実 `buildContextProjection` に対し3/3対照がPASS。元Cardの非公開 `meta.source` 差、Island membership／Canvas座標差では公開Projectionが同一になり得る。露出するCard本文／レビュー済みIsland題名の変更ではProjectionが変化するPositive controlもPASS | SUIの逆Patch拒否、SHO Writer／OKF Export、TEIのP/Q合成写像、元Source実Reader、Host権限の合格 |
| [TEI merged #3253](https://github.com/hat47x/tei-platform/pull/3253) | 上記SUIの非単射とTEI CLIのcaller-supplied scope（#2881の局所実測）を現mainの研究§13に帰属。表示上のScope／Keyから元Source IdentityやRelation Identityを逆推論しない | 一般的な二Consumerの非干渉、全Source Closure、逆編集資格、TEI Product Reader/Renderer |
| [TEI merged #3254](https://github.com/hat47x/tei-platform/pull/3254) | 合成Workspaceの一属性変更でOperation Surfaceの `worldFingerprint`／`modelRevision` が変わり、Operation 5件の `basis` は不変という実出力を取得。記録時点では実行時照合の受渡し入口が見つからない | `basis`自体の誤り、Operationがすべて無効化されること、実行時の適格な自動再判定実装、現在Host Grant／実DBの検証 |

**誤った合算を避ける**：CX-01の「非単射が現実に起こり得る」はSUIの**実Product局所試験で観測済み**へ更新する。一方、TEIの合成Q∘P・逆写像API・Product二Viewは実行していないため、CX-01のTEI E2E合格件数は**0のまま**。TEIが実際のSUI非公開属性を受け取った、または復元しなければならないとは要求しない。

### 11.2 表示等価・Source帰属・操作の現在資格を三つに分離

**例A（非単射・読み取り専用は成功してよい）**：Source AとBで非公開Source cue／Canvas/Island所属が異なるのに、公開Projection Wが一致する。この結果は、公開仕様がそれらを意図的に落としている限り**正しい**。ただし `same W bytes` は `same Source` や `unique inverse Patch` を導かない。編集要求時は元Source/Path occurrence/Versionへ別根拠で戻り、候補が複数なら自動書込みを拒否するか元ProductでOwnerへ候補を提示する。

**例B（表示は同じ・Operation資格は変わり得る）**：Data Modelへ属性を追加すると、Operation 5件の `basis` が同じでもWorld/Model Revisionが変化したという#3254の実測がある。現在のAction Offerは、元Source・Rule／Predicate／Operation Input／Scope/Grant／実行Targetとの**必要依存**を再照合してから再利用可否を判断する。`basis`一致のみで旧Offerを実Requestへ昇格させない。逆に、新属性が当該View/Claim/Actionの依存閉包の外にあると独立に確認できれば、**過剰失効を避けて局所表示の資格を維持する余地**がある（影響判定は#2582、実Actionは#2598/#1174）。全属性変更を一律拒否する新共通Gateは作らない。

**例C（Policyが変わっても見かけ同一）**：Public Slice/出力byteが同じでも、別Tenant/Scope/Provider/Policy revの現在のAction Permissionや実Consumer開示の資格は別。#2516で開示対象を確定したうえでProjectionを評価し、元Private Worldの差がTooltip・選択候補・ログ等のside-channelへ漏れない検査とは区別する。

### 11.3 独立検証基準で測る具体的な「誤認」

新しいCase群・生成器を追加するのでなく、既存CX-01/CX-05/CX-06/CX-10とQS-06の**独立Expectedに次の判別点を一因子追加する**。

1. 公開Projection byteを固定して非公開 `meta.source`／Island所属だけ変えたとき、**read-only Projectionは不変**でよい。しかし同じ公開値を根拠に一意な元Source IDを選び、Reverse Updateを発行した実装は不合格。
2. Sourceへの一属性追加により `worldFingerprint` が変わり `basis` が同じ場合、**旧Action Offerの現行資格を確認せず実Requestへ通す**実装は不合格。逆に影響外のDisplayを無根拠にすべて再生成する過剰反応は性能/説明の評価対象（安全違反と一律にはみなさない）。
3. 出力を変える公開Card/Island titleの一因子変更にもProjectionが不変となる実装は不合格。常時同一比較で3件の負例がすべてPASSする「空虚なVerifier」を許さない。
4. 一つのFunction Context IDだけを同じにし、Source/Scope/Read Cohortが異なるTableとFormを同期する実装は不合格。共同Snapshotの要件があるClaimに限り、版とReader/Providerの同時成立Evidenceを要求する。

**テストデータ生成との接続**：Data/Function/Businessから制約付き正負Datasetを導く新しい並行Draft [#3258](https://github.com/hat47x/tei-platform/pull/3258)・[#3259](https://github.com/hat47x/tei-platform/pull/3259)・[#3260](https://github.com/hat47x/tei-platform/pull/3260)・[#3261](https://github.com/hat47x/tei-platform/pull/3261)は全てDocs-onlyの**競合する採用前候補**として扱い、#3236へ第二Generator/Dataset/Scenario SSOTを持ち込まない。将来受理される制約付きDatasetがあれば、上記の「非公開Sourceだけ変える」「Operation/Roleだけ変える」などの対照要因を#1131の既存Dataset／#1180 Scenarioと#2514の**独立Expected**へ配分する。原典未確認の「強属性」をToggle/Slider等の適合条件に自動昇格させず、既存Facet/Value Domainの受理済み制約だけを使う。

**次工程と責務**：P0=SUIの局所非単射・TEIの操作面driftという有限な**既存の実測**を、#2497/#2507/#2582のread-only資格票へ持ち込む。P1=#2586の実Function→SQL/Provider ReadにSource/Context/Scope Witnessを加え、同じResult SliceからTable/Form二表現を局所比較。P2=#2500/#2598/#1174で一意なInverse/現在Grant/Effectを**別試験**にする。既存#2514/#1180へ候補と既知Mutantだけ渡す。新Go/Node test、実TEI Product 二View、実Provider/Reverse Write、Owner意味受理は**今回NOT RUN**。


## 12. Product Readと三視点Diagramを並列P0にする：共通Source/Identity・異なる資格（2026-11-07追補）

### 12.1 優先度を修正する理由

[三視点Diagram/CRUD製品受入 Draft #3263](https://github.com/hat47x/tei-platform/pull/3263) はX-TEA Modeler由来のData/Function/Business各図面の表示・編集、CRUD等のcross-referenceを**TEIモデラーのP0**と位置づけている。§8〜10の「Product Header実Read→Form/Table」を**唯一のP0**として図面/CRUDをその後に置くと、標準Modelerとしての中心価値を遅らせる。次の二つを**並列P0**とし、同じSource/Meaning/Identity証拠へ合流させる。

| レーン | 実証対象／最初の独立Expected | 必須Witness | 完了と呼ばないもの |
| --- | --- | --- | --- |
| **P0-M Model→Diagram/CRUD** | 受理済みの限定 Data / Function / Business Sourceから、3図面とFunction×Data CRUD表を**headless read-only**で導出し同じSource IDへnavigate。複合Key、1:Nの関係Role違い、手作業・DFDを含む | Required Design Source/Relation/Role/CRUD宣言の閉包、出典Span/Revision、ノード・辺・Occurrenceと表セルの型/根拠、未解決・非開示 | 正しいRendererの可読性、可逆Editor/Studio実接続、実Runtime DB Effect、SHO納品 |
| **P0-R Function→実Read→二表現** | §8/§10の `order_entry.recent_orders` Product Query/SQL/Provider実Readを同一Attemptで束縛して、Result Slice→Data Context→Form/Tableを読み取り専用比較 | Function/Query選択、実SQL raw+args、Physical Provider、Row/Field/Scope/Read epoch、Subject/Context | 参照PG局所PASSだけでProduct E2、全OrderLine母集団、実Edit/Commit、同Contextだけで共同Snapshot |
| **P0-C 接合の品質** | P0-Mで図上の `sales.order` とP0-RのHeader Subjectが一致するか。**同じ表示Keyではなく型付きSource/Identity/Scope**で結ぶ | 元Source revision/Locator、関係Role、対象粒度、Target Occurrence、Query Result/Provider readが支持する範囲 | CRUD表の `R`宣言からそのQuery実行完了を推論、`C/U/D`宣言から現在Grantや実Provider Effectを推論 |

P0-MはP0-Rに**依存しない**。Data/Function/Business Sourceの閉包があれば、DB未接続でも図の構造・CRUD宣言のread-only適格性を検証できる。P0-Cでは両方のWitnessが存在する部分だけjoinし、片方が欠測なら対応セルを`unverified/unresolved`として記録する。これが「Modelerのモデリング能力」と「生成Applicationの実行能力」の境界である。

### 12.2 図・表・操作を同じ構造へ合成するが、意味正本は増やさない

TEI標準構造は `Subject/Relation/Occurrence/Role/Identity/Time/Scope` を保持し、別の**表現型Instance**がDiagram（Graph）、CRUD Matrix（Table）、Form/Tableを投影する。Diagram Profileは配置・表示順・強調・属性と具体例の並置を表すが、表示座標や行列セルをData/Business/Functionの新正本へ格上げしない。汎用Graphの辺ID/方向/多重性を保ち、Relationの1:N表示だけで組成・参照・複合定義域を同一視しない。

Cross-reference各セルは用途ごとの根拠を区別する：

- **declared**：Function/Business Sourceに明記されたread/create/update/delete・手作業・外部授受。
- **derived candidate**：依存経路/Business Procedure/Ruleから可能性として導出された操作（採用済みOperationや実行済みCommandではない）。
- **observed effect**：特定Attemptにおける実Request→Provider Effect/Receipt/独立Readで観測された範囲（実対象・版・Grant次第）。

`not declared`、`not observed`、`not fetched`、`withheld`、`partial`、`proven absent` は別状態。行列の空欄は越境CUD=0や全網羅を示さず、閉包が証明されるまで断定しない。実Effectや業務Procedure InstanceとDFDのデータ授受を混同しない（#1125/#1129/#1174/#3250）。

図上編集は(a)配置だけのPresentation Change、(b)一意に逆写像できる型付きSource Proposal、(c)非単射で逆変換不可能な表示変更、(d)実Action Intentに区分。Meaning Proposalを保存する前にはSource位置/現在Revision/Owner Acceptance、実Operationの前には現在Grant/RequestとEffectをそれぞれ検証。Mermaid風SHO図解は独立した文書ソース/RendererのConsumerであり、TEI図への参照Bindingがない限り自動的にTEI Meaningを変更しない。M-01〜10の独立Expectedと具体的Editor/SHO受入は既存#3263/#1124/#2497、SHO #5/#6/#19に置く。

### 12.3 テスト生成の前段：Required Design Closureと受領完全性

[#3262](https://github.com/hat47x/tei-platform/issues/3262) のDesign Context QueryはP0-M/P0-R双方の**生成前提**である。Required Source/Meaning/Role/Relation/Requirement/Scenarioを用途相対に定め、`declared → discovered → loaded → parsed → bound → returned → received`を同一Workspace/Revision/Consumer Scopeで照合。CLI/将来MCP/Studioは**同じ閉包の別投影**とし、CLI一覧やTool呼出しの成功を全Source受領に昇格させない。

- [main統合 #3264](https://github.com/hat47x/tei-platform/pull/3264) の限定実証：単一実TEI Binaryで9種類のread-only CLI、Product Skill静的14件、Required Source `readByLoader:true`、欠落/異版/未読込の合成負例を検査。**Design Contextの全Required集合・実MCP/SHO/外部Harness受領完結は未実証**。
- [main統合 #3266](https://github.com/hat47x/tei-platform/pull/3266)/[#3268](https://github.com/hat47x/tei-platform/pull/3268)：既存Pinned Scenario 6 Step、実CLI負例4/独立義務2による**限定的な判別力の検証**とSource/Scenario変更で起動するCI Pathの拡張。Go局所statement coverageはC0相当（限定試行10.1%）、C1/全要件/NFRは未計測。**新しいModel→図/CRUDのテストデータ生成が実装済みとの主張ではない**。
- [#3273](https://github.com/hat47x/tei-platform/pull/3273) は未カバーRuleから**新しい**Scenario負例を提案・評価するT1のDocs-only Draft。P0-Mは既存#1180 Scenario/ #2514 Independent ExpectedへM-01〜10のRequiredを示し、#1131のDataset/AI生成器や第二Oracleを#3236で作らない。
- 実Toolの戻りが途中切断・欠落し、別Revision/SourceのPageが混在したら`incomplete/stale`。非開示Subjectの存在・件数・Hintまで外部Consumerへ洩らさない。Expectedは同じ生成AI/RendererのGot出力から逆算せず、Owner受理モデル/固定Fixture/既知Mutant等の**独立根拠**を持つ（#2514）。

### 12.4 既存Driver Scenarioへ手渡す最小の横断受入票（新Runner/Case SSOTなし）

| 識別 | 一因子変更 | 独立Expected／実証可能な否定 | 既存Owner |
| --- | --- | --- | --- |
| DM-01 | 1:N数値多重度同一、Relationの役割だけ変更 | 図の辺Role/出典が変化。CRUDセルや業務Compositionを同一視しない | #1126/#3263 |
| DM-02 | Business DFD矢印だけ追加、Procedure Step実行証拠なし | データ授受を説明、Workflow/実行順/Provider Effectの完了は未確認 | #1125/#3263 |
| DM-03 | あるFunctionの`R`宣言を追加、実Product Readerなし | CRUD宣言のセルは変わるが実SQL/Rowset/Provider Readはunknown | #1127/#2586 |
| DM-04 | 図レイアウト/Markdown説明だけ変更 | Model意味/CRUD宣言は不変。納品Artifactは変わり得るためRenderer/SHOを局所再確認 | #2497/#1124/SHO #19 |
| DM-05 | Diagramの表示グループが複数元Relationに対応 | グループから特定Source/Edgeを一意に選んで自動Reverse Patchしない | #2497/#2500 |
| DM-06 | Design Queryの1 Page/Tool返信が欠落してCRUD=0に見える | Required/Received閉包不成立なら`partial/incomplete`、不在の証明へ昇格しない | #3262/#2514 |
| DM-07 | 同一`sales.order`だが別Workspace/Source RevのDiagramとForm | Cross-view選択/編集/Actionを同一現行Subjectと扱わない | #3181/#2506 |
| DM-08 | Pattern宣言は同じ、BOM/Plugin/Provider版だけ変更 | 図の局所不変部分とCurrent Renderer/Adapter/Operation適格性を分離し、未検証版を実行資格へ昇格しない | #3267/#951/#2582 |

これは要求探索と**実装計画の接合表**であって、現在の製品内で自動検出されているCase群ではない。G0/G1/G2等のOwner/Gateは#1119、実Go/CLI P0-M、Product P0-R、Studio/SHO Diagram、Formal Meaning Acceptance、実MCP/Harness/Provider/Writeの実行はすべて**今回NOT RUN**。P0-CのHeadless側で初めに揃えるべき出力は「三図・CRUDの同一Source参照」「不適格セル・失効位置」「Function→Query→Provider/Outputとの差」であり、ピクセル完全一致・独自Diagram DSL・固有Hostはこの段階の前提にしない。

## 13. 三視点FixtureとDesign Context Gateを実装入口へ接続（2026-11-07追補）

### 13.1 現行コードの具体的Fit/Gapを固定する

[Draft #3263](https://github.com/hat47x/tei-platform/pull/3263) HEAD `d45fdbe8996882528f3fd6aeff99cca2152a22f3` は `reference/diagram-crossref/pressure-input-v0.json`（Blob `ddc1c8bae70a04135db60853057a811e4902e6e6`）と `expected-v0.json`（Blob `1b083589fd7644dd518bb2231e218566d60de82e`）を保存した。ただし**両者は合成・AI草案**で、前者は**TEI Parserに読める形式ではない**。内部整合15/15から独立Expectedの業務受理や現行CLI受領を推論しない。

現mainの確認済みコード：

| 実装入口 | 今ある機構 | Diagram/CRUD E2Eまでの最初の欠落 |
| --- | --- | --- |
| `internal/modelref/model.go` | `Key{Scope,Subject}`、`Ref{Scope,Subject,Version}`、型付きEdgeのRelation／出典Evidence、厳格なRef解決・検査 | ProductのData/Function/Businessを**一つの実Loader revision**で対応付けた根拠付きProjectionと、図上Occurrenceへの逆参照は別。`modelref.Edge.Relation`を表示辺の位置や一意な実更新先とみなさない |
| `tools/reference-er-table-crossview/projection.go` | 受理構造のread-only ER/物理表のProjection。Data/Relation/Key/FD/Physical Mappingの比較・Unresolvedを出す | **Function×Data CRUD**／Business Action×Function/Procedure／manual handoffの結合はこのToolの現在責務ではない。第二ERP正本を作らない |
| `tools/reference-business-dfd/diagram_context.go` | Handoff/Actor/Triggerの候補をDOTへ描く。HandoffはRuntime処理順ではない | 図の矢印を実Effect/Receiptへ昇格させず、Business Data FlowのSource/Role/Function pathと突き合わせる |
| `reference/tei-studio-graph/graph.ts` | `GraphNode.id`、`GraphEdge.id/source/target`、多重辺を保持し、endpoint不在を拒否。表示・Node selectのReference | `onSelect`は**nodeId**。Edge/Field slot選択、元Relationの型付きSource/Revisionへの復帰、逆編集の可否判定は現実装で保証されない。自己辺は`unsupported_self_edge`として診断対象 |
| `cmd/tei/cli_contract.go` | `model catalog`/`source-index`/`inspect`等の**選択された**read-only CLI入口 | 現契約に `diagram` / `crud-matrix` の製品コマンドを確認できない。仮のCLI名を既存機能としてSkillへ広告しない |
| `tools/reference-issue-test-ability/context_gate.py` | main統合[#3284](https://github.com/hat47x/tei-platform/pull/3284)/[#3296](https://github.com/hat47x/tei-platform/pull/3296)で9実CLI ReplyのSource/指紋／不完全受領を試験側で評価 | 固定Requiredは公開Referenceの`semantic-data:sales.order`が中心。**Function/Business/Role/Relationの独立Required Closure**は未資格。Gateで`partial`／`qualifiedTestGeneration=false`となる範囲を、三視点図/CRUDの「complete」へ格上げしない |

**新たな実証済み基線**：[PR #3296固定HEADの専用CI #42371310577](https://github.com/hat47x/tei-platform/actions/runs/42371310577)で、旧7＋Manifest/Root 2＋Tool Reply Byte 7、計16の限定Synthetic ControlがPASS。既存Pinned Scenario 6 Step/6判別対照もPASS。7 Byte Mutantは受領済みJSONの**一時コピーを改変**したもので外部H3 Toolの生応答断絶を観測した証拠ではない。Source/Parser/Function/Business全閉包、実MCP、C1、Graph/CRUD Projectionは未実証。この基線に対して二つ目の`packet.py`を持つ[未マージ・Closed #3293](https://github.com/hat47x/tei-platform/pull/3293)を復活させず、既存Gate/Scenarioへ追加Requiredを渡す。

### 13.2 「一枚の図・一表の受入」を実行できる最小分解

**P0-M0（Input資格の明示）**：#3263 synthetic JSONをそのままTEI Workspaceとして扱わない。まず受理済みSource/既存`reference/workspace`の意味表現に対応できる部分を照合し、**未表現のRole/Relation/Business宣言はunresolved**。不足するDSL/Readerは#1125/#1126/#2621へ戻す。Required集合にはSource locator・Model revision・Entity/Relation/Function/Business/CRUD宣言・Disclosure/Loader実消費が含まれる。#3262 Gateで該当Familyの閉包が不足なら`qualified=false`として検証可能な局所Projectionだけ続ける。

**P0-M1（read-only Projectionと元Sourceへの逆引き）**：同名別IDの`data.order.status`/`data.flow.status`、同端点の二Relation `rel.line.sku.primary`/`rel.line.sku.alt`、一つの関係が`contained-child`、もう一つが`reference`である差を入力に取り、Node ID、**Edge ID・方向・Role・元Relation qualified ref**を失わずGraphへ投影する。Source端点が欠損した場合は無音で辺を捨てずUnresolved。図上選択・Source navigationを付与する段階では、現Graph Referenceがnode-onlyであることを明示し、**Edge選択/元関係復帰は未実装ならFAIL/UNSUPPORTED**とする（フロント表示のPASSを遡及可能性のPASSとしない）。

**P0-M2（CRUDの別根拠付きCells）**：`fn.order.register`→`data.order`のC／`fn.ship.finalize`→`data.stock`のU、`fn.external.dispatch`の`declaredCrudComplete:false`を対照に、**明示宣言・候補Path・実Provider Effect観測**を別列とする。未列挙Functionのセルは`unknown-not-enumerated`であり`none`にしない。手作業`action.phone.confirm`にFunction/CRUD/Commitを捏造しない。JOINのO100/L1,L2とO100/P1,P2,P3から6行が出ても、明細6件としない。

**P0-M3（検証基準の独立化）**：#3263 expected-v0.jsonは**固定候補**として扱い、#2514のOwner/別検証基準で①Source宣言に対する期待を独立チェック、②Positive Controlで公開ID/Role変更時の変化を確認、③N01〜N12の少なくとも1件を実Product Projectionへ投入し対象Testが直接FAILすることを示してから、限定Conformanceに利用する。JSON内部の15項目一致・全Mutantを単純`FAIL`と列挙しただけでは判別力を認定しない。#1180 Scenario正本を用い、別Verifier/Case SSOTを増やさない。

**P0-M4（Consumer分離）**：CLI→Studioは同じGraph/Matrix Source・Scope・Policy・Revisionの`received`と描画・Selectに限定。TEI参照図→SHO Writer、Document/Caption変更、実ファイル再読込、HTTP Decision Graphへの横断投影、Plugin/BOMの実Conformanceは各Ownerで別証拠。**Graphのノードをクリックできること≠Relation編集の資格**。Diagram Layout保存≠Model Accepted Meaning変更。

### 13.3 最初の合否を曖昧にしない４つの対照（新Go/CLIは今回NOT RUN）

| 検査 | Positive/Negativeの独立対照 | 確定してよい結論の上限 |
| --- | --- | --- |
| DGX-01 | 同端点2辺・同名2 Subjectを保持。Mutantは辺IDを消す／labelでSubjectをマージ | Headless GraphのID・多重辺を保持できたか。Source read完備やWrite資格ではない |
| DGX-02 | `declaredCrudComplete=true`の関係外セル`none`と、`false`の未知セルを比較。Mutantは両者を空欄へ潰す | Function宣言の局所Closed Scopeについての不在だけ。実Provider Effect 0ではない |
| DGX-03 | Gate入力のRoot欠落/JSON途中切断/古いFPでblocked。正常限定入力も`partial`で留める | Design Contextの有限Consumer負例。三視点Required Closureを網羅した証明ではない |
| DGX-04 | 同じ表・DiagramでもProvider/Plugin/Source版変更。Sourceだけ同じ・Renderer版変更を別対照に | ローカルProjectionの再利用とCurrent Consumer/Action/Artifact/Grant資格を独立に診断。BOM全互換の証拠ではない |

Ownerは#3263（Fixture・三視点独立期待）、#1124/#1125/#1126（Source/Modeler）、#2497（Loss/逆写像）、#3262（Design Context Closure）、#2514/#1180（独立Expected/Scenario）、#2586（実Product Read）、#997（Target/Studio）、#3267（BOM）。**最初に不足する辺は、合成JSONの存在→TEIの実Source/Loader/Meaning→Required Closure→Projection Output→Consumer Received**である。実行可能なPoCの順をこの境界に沿わせ、Provider実Readがまだ無いことを図面Read-only実証の一般的ブロッカーにしない。

今回新たなGo/CLI/Studio/Provider/SHO/外部Harness動作、独立Expected受理、実Ownerの意味採用は**NOT RUN**。この節は検収作業の実装方式を決定する**採用前のFit/Gap**であり、正式Schema、MCP、第二Parser/Source Registry/Graph Model/Runnerを固定しない。


## 14. Fixture実試験とSource/Lock鮮度を分離した初回採否（2026-11-07追補）

この節は§12（P0-M/R/Cの並列性）と§13（実Loader→Graph/CRUD不足辺）を変更せず、**新たにGitHub上で確認できた実試験・CI失敗**を検収順へ組み込む。既存#3263/#3262/#2514/#1180/#3182/#3267の責務を置き換えない。

### 14.1 実試験として確認できる命題の更新

| 証拠／実装 | 確認済みの限定命題 | この命題からは推論しない |
| --- | --- | --- |
| [Draft #3263](https://github.com/hat47x/tei-platform/pull/3263)（HEAD `aeea395ca79c2fe84e068cb94d6f43c8982e1bdf`） | `reference/diagram-crossref/fixture_contract_test.go` に**Fixtureを監査するGoテスト**が追加された。合成9 Entity・7 Relation、同名別ID、CRUD、不正入力等の構造上の制約を検査する意図。分離StubによるGo 2 Test/4 subtestは実行済み | そのStubでGitHub上の二つの実JSONを読んでTESTしたこと、TEI Loader/Graph/CRUD投影の適格性、別AIが書いた期待を独立Oracleとして受理したこと。最新HEADの[Repository Health #42373811364](https://github.com/hat47x/tei-platform/actions/runs/42373811364)は**FAIL**であり、Go試験の完全CI PASSではない |
| [main統合 #3299](https://github.com/hat47x/tei-platform/pull/3299)（固定HEADの[専用CI #42373330734](https://github.com/hat47x/tei-platform/actions/runs/42373330734)） | 実TEI CLIの旧Pinned Scenarioが、コピーしたWorkspaceで**実Source改変**／**Plugin Lock改変**を拒否。Source/Lockの元byteへ戻すA→B→Aで再成功するContent pinの限界も観測 | 中間変更が起こらなかったこと、受理済みMeaning不変、Actual Plugin Load、Provider×Consumer Conformance、現在Grant、Graph/CRUDの実Source Reader接続 |
| [Draft #3302](https://github.com/hat47x/tei-platform/pull/3302)（HEAD `1e43e52f48432a24c74c7f8151999fe6a641f6bd`） | `tei commands`のread-only `view`広告から、明示実行`-execute`系を除き、その差を`excludedFlagNames`で機械可読にする**修正候補** | このHeadが安全であること。実[Health #42373744325](https://github.com/hat47x/tei-platform/actions/runs/42373744325)は**FAIL**で、`gofmt`不一致と`TestCommandCatalogFlagsMatchAcceptedFlags`が広告Flag集合と実受付Flag集合の違いを検出。実行Flagの効果をCLIの静的`effect:read-only`から一律推測する設計はまだ採用しない |
| [統合Draft #3303](https://github.com/hat47x/tei-platform/pull/3303)（旧HEADの[Health #42373568969](https://github.com/hat47x/tei-platform/actions/runs/42373568969)） | `gofmt`、`go vet`、`tei validate`は各々PASS。失敗は`tools/repository-plan-check`の設計索引登録不足とRQ-035のScenario/Evidence対応不足に局在 | 全Health PASS。今回**自PR追加3文書の索引**を登録したが、他レーン文書の索引とRQ-035証拠Mapは[Draft #3297](https://github.com/hat47x/tei-platform/pull/3297)が引継ぎ済み。別OwnerのPASをこのPRで推測修正しない |

### 14.2 三視点Diagram/CRUDを実装する前の必要な「三段階の固定」

**S0: Target Fixture/独立Expectedの固定** — #3263のPressure Input/Expectedは両方AI草案であり、InputとExpectedが別ファイルでも正答の独立性は証明しない。Fixture契約Testで構造不備・既知Mutantを検出し、Source・Role・Key/FD/CRUDの業務意味を#2514/#1124/#1126が別根拠で限定的に確認する。正常Fixture自身が`synthetic`と明記する場合は、その限定Claimのみで利用可能。

**S1: Current Source/Invocation の固定** — 現実のTEI Readerは`actual Source revision→Loader→Parsed/Meaning/Required Closure`を満たした分だけProjectionへ渡す。実Plugin/BOM選択とLock byte、現在のProvider Link/Load/Invokeは別。#3299のA→B→A再受理はContent pinの正常動作でも、Read Cohort、Screen Selection Epoch、Host Grant、Secret Disclosureが旧版から復活した証拠ではない。再利用の単位は**Consumerの特定Claim/必要Dependencyと現在Scope**に相対的であり、全World一括の強制Invalidationは要らない。

**S2: Output/Operation の固定** — `node/edge occurrence→元Source`、`CRUDセル→宣言Function/Business Path`の逆引きは表示正本と別資格。CLI/read-only Consumerで必要なAttribute・Graph Edge/CRUD Cell・Closureが受領できたかを確かめ、`view -execute`・Export・File Log等の実行Variantは実Call/Targetの効果を再評価する。成功したUI Previewだけで`zero local file effects`や業務Commitまで宣言しない。CLI Scope変更は#3182/#3302/#951の実装Owner。

### 14.3 最小の差分Expected（既存Caseへ付加、ここで実行していない）

| 検出すべき誤認 | 対照 | 該当Case/Owner |
| --- | --- | --- |
| Fixture自体のTest PASSをProduct Diagram/CRUD PASSへ昇格 | 実Parser受入とGo fixture-only Testを同時に提示し、一方の未達を正しく`not-run`へ残す | DGX-01/02、#3263/#2514 |
| Source A→B→Aの同内容再現で旧画面のAction Offerを復権 | Content pinは再一致しても旧Selection epoch/Current Grantは別 | WX-02/CX-10、#2582/#2598 |
| Read-only Discoveryの`view`に`-execute`を混ぜ、OS書込みがないと主張 | Previewとlog/DB fileを出す実CLI Invocation Variantを分ける | WX-03、#3182/#951 |
| 並行CIの失敗を本PRのGraph/CRUD不合格／他PRの成功として合算 | 固定HEAD・base/merge checkout・Code/Doc/Indexの失敗種別を独立に示す | DGX-03、#2514/#1119 |

**次の必要な最小実行**：#3263の正規のGitHub fixture Testで実保存JSONを読む→Source/Meaning側のAccepted部分へMappingする→同じSource revでGraph/CRUDの`got`を生成する→独立Expectedと1実Mutantで比較→CLI/Studioの受領差を検査する。#3299の既存Context Gate/Scenario/Change Detection、#3302の実行Variant、#3297のBOMは各所有契約を再利用する。作業としての第二Runner、第二Graph Model、第二Source/Permission Storeは作らない。

**今回の操作と受入の限界**：#3263/#3299/#3302/#3303のIssue・ソース・CIログを再調査し、既存資料に条件を反映した。#3303の索引3件以外には新実行用Go/CLI Source変更はない。実TEI Product Diagram/CRUD、実Loader/Target/Studio/SHO/Provider Effect、Oracle Owner受理は**NOT RUN**。


<!-- tei-crossview-t1a-tt56-operation-drift-20261108 -->
## 15. 三視点の入力閉包・Rule判別・Operation現在性を別資格で進める（2026-11-08追補）

本節は§12の**P0-M（三視点とCRUD／Read不要）・P0-R（実Product Read）・P0-C（同ScopeのCrossview）**を維持し、§13/14に増えた設計/実測のうち**検証能力の実前進だけ**を当該Ownerへ結ぶ。旧[統合Draft #3236](https://github.com/hat47x/tei-platform/pull/3236)は閉鎖済み・未マージで、本PRが引継ぎ先。PR #3179も既に閉鎖・未マージ。旧PRの `MASTER_PLAN.md` をそのまま再投入しない。

### 15.1 Contextの「必要情報がない」と「必要Ruleに達した」を別に判定

- main統合[#3284](https://github.com/hat47x/tei-platform/pull/3284)/[#3296](https://github.com/hat47x/tei-platform/pull/3296)/[#3299](https://github.com/hat47x/tei-platform/pull/3299)は、実CLI応答・Loader証拠、欠落/陳腐化/片側Root欠落、SourceとPlugin Lockの実変更を試験専用Consumerで判別できた。ただし `boundedReference=partial`、`qualifiedTestGeneration=false` と現在Consumer/Host・意味受理の別資格は維持する。Fingerprint一致・JSON受領成功から、図/CRUDに必要なRule/Relation/Businessの列挙完了を推定しない。
- main統合[#3304](https://github.com/hat47x/tei-platform/pull/3304)の[実CLI CI #42378033965](https://github.com/hat47x/tei-platform/actions/runs/42378033965) は、**既に選定された**行番号正数Ruleに `line_number=0` を与えた追加第7Stepの`not-satisfied`を、既存Pinned Scenario 6Step・独立Property・3実CLI負例と区別して確認した。**T1-A（既知境界の新Step実走）**であり、AIが未知Ruleを盲検発見したT1-B、全Required閉包、製品C1/NFRではない。実測後のPR文書HEADはCIで実走したプログラムHEADと異なるため、実証の引用は前記固定CIとその範囲へ限定する。
- 未統合[Draft #3326](https://github.com/hat47x/tei-platform/pull/3326)は**同じWorld/Index Fingerprintで必要なValidation/RuleがCLI出力から部分的に消える**TT-56を追加検査する候補。最初の実CIでは`source-index`のsubjectモードを全件Indexと誤認して失敗したため、既存`source-index -workspace`のindexモードを**10番目のread-only Reply**として取得する方式に修正。これは試験Consumerの入力不足が作ったFalse Missingであり、実ParserのRule欠落を発見したわけではない。**修正後の新HEAD CIが成功するまではPASS未確定**。別の[Draft #3324](https://github.com/hat47x/tei-platform/pull/3324)も同じTT-56をinspect-validationで解く競合候補。実FixtureのRequired・実CLI現行契約と負例判別能力を比較して既存#3262/#2514へ採否を返し、**二つのRule Registry/二つのRequired正本を同時採用しない**。

### 15.2 図・表の読み取りから実操作までのAdmissionを混同しない

main統合[Reference #3325](https://github.com/hat47x/tei-platform/pull/3325)は、`WorldFingerprint`+`ModelRevision`のExpectedを呼出側から与える**参照用drift-check**で、同Workspaceなら`current`、属性改版なら`stale`、Expected未指定なら`unchecked`を区別した。`basis`は属性変更を捕捉しないため比較対象にしない。**既定の`tei operations`・実Invoker・現Host Grantには未接続**であり、旧View/Operation IDや同じBasisを「いまDispatch可」とみなさない。表示のみの図/CRUDセルと、Command/Update/外部EgressのAdmissionを分離し、read-only投影により変わらない範囲の局所再利用は許すが、未観測の依存を「影響なし」と断定しない。

| 既存Lane／受入場所 | 新しく引き継ぐ最小の独立負例 | 証拠として使える範囲 |
| --- | --- | --- |
| **P0-M0 / #3263・#3262・#2514** | 同一FP・正常JSONのまま`validation.tei`の必須Rule/Role・Relationが1件だけ欠落、または`source-index`がsubject modeのまま | `required - observed`と実読込・Role/Sourceの突合を出し、`partial/unknown`へ停止。TEI Graph/CRUD Product PASSでない |
| **P0-M1/M2 / #1124・#3263** | 同端点の多重有向Relation、同label別Entity、未列挙Functionを元Sourceへ逆参照できない | NodeだけでなくEdge ID/Relation Role/Path occurrence、CRUDセルの`declared/derived/unknown`を別Witnessにする。Graphic描画だけで完全なDependency閉包としない |
| **P0-R / #2586・#2506・#2514** | Reference PGではRowset取得、しかしProduct `recent_orders.sql` の`public.orders`・実`:from`・選択Providerの証拠が無い | Product Function→実Query/SQL byte/args→実Provider Rowset/Field/Scope/Context/Consumerの**最初の欠落Edge**に停止。別SQL・別母集団を同じRead成功と合算しない |
| **P0-C / #2507・#2582・#2598** | 同じDiagram/CRUDとForm/Table、同じOperation IDだがSource/World改版、Selection Epoch又は現在Grantが異なる | 表示の同一性はScope/Revisionが合うClaimのみ認定。参照drift-check`current`≠実Invoker admission、`unchecked`≠pass |
| **T1-A→T1-B / #3262・#3273・#1180** | 既知ゼロ値のT1-A実走を、未知のRuleをAIが発見・生成したT1-Bと表示上で同じ「新規生成済」にする | #3304のT1-Aは実CLI候補判別だけ。未知Rule探索・独立Holdout・C1/NFR/Owner採択は次の別実験 |

### 15.3 次に実行すべき実測と停止線

**A：設計Contextの正本を一つにする（後続で成立）**。TT-56の#3324／#3326は、[main #3354](https://github.com/hat47x/tei-platform/pull/3354)により単一`context_gate.py`へ統合され、既存CLI **11 Reply**（全Source Index＋Focused Rule Inspect）と3 Ruleの独立Required・Canon/Relation/Source照合に収束した。既存9/途中10/現11 Replyの数の差を全Domain ClosureやT1-B資格の差としない。#3262/#2514が正式なRequired Scopeと独立基準を引き続き所有する。

**B：図とCRUDの第一の実Consumerを成立させる**。#3263の合成JSONを実TEI Sourceとして偽装せず、実Loader→Meaning→Graph/CRUD Projection→Edge/Cell Source navigate→CLI/Studio受領の一Attemptを、DGX-01/02・独立Expected・実Mutationで検証する。P0-Mのread-only First Sliceは、P0-Rの実DBが未接続でも限定受入可能。

**C：最初の実Product Readを完了する**。#2586でFunction/Pattern→Selected Query `recent_orders`→Source SQL byte+`:from`→physical Provider→Result Field Presence/Scope/Cohort→Function Context→Form/Tableを一件固定する。実隔離PG Reference・#3304のtest CLI合格・#3325 drift-checkの局所成功を、このP0-Rの代替証拠にしない。

**本節の実行資格**：既存main統合済み#3304/#3325・#3296/#3299の実試験記録と、#3324/#3326の未統合Draft Source/CIの照合を既存設計票へ追補。新しい製品Go/PG/Studio/Graph/Host/MCP実試行と独立ExpectedのOwner意味受理は**今回NOT RUN**。この計画の変更自体は実装・製品受理を意味しない。新Parser/Resolver/Runner/Domain Registry/Host/Gate・第二Scenario正本は導入しない。

<!-- tei-modeler-context-two-greens-selected-actual-effects-20261108 -->
## 16. 必要Rule取得の二方式・Plugin Selected/Actual・CLI副作用を同じ製品受入へ接続（2026-11-08）

本節は§12〜15の **P0-M（三視点・CRUD）／P0-R（Product実Read）／P0-C（同一Identity・Scopeの接合）**を維持し、新たにmainへ統合された実測を正確な段階へ割り当てる。Go/CLI実行の新しい正本・Plugin契約・Domain Registryを定義するものではない。

### 16.1 Design Context：二つのTT-56実装は両方Green、ただし同じ資格ではない

[Draft #3324](https://github.com/hat47x/tei-platform/pull/3324) HEAD `4f4b114aa826b1a603044c5eef34407f536f186d` は既存`inspect`をSemantic Validationの単一必須Ruleへ追加し、Required Rule／Attribute／Role／Source Index Anchorを突き合わせる**1 RuleのConsumer**。 [専用CI #42385606408](https://github.com/hat47x/tei-platform/actions/runs/42385606408) と[Health #42385606448](https://github.com/hat47x/tei-platform/actions/runs/42385606448)は固定HEADでcompleted/success。

[Draft #3326](https://github.com/hat47x/tei-platform/pull/3326) HEAD `c2d2da10cc93022697204e66e83c30248b3810a0` は既存`source-index`の**subjectモードとは別のworkspace全件indexモード**を10番目の応答として追加し、公開合成Validation 3 Ruleの独立Requiredと照合する。[専用CI #42385726717](https://github.com/hat47x/tei-platform/actions/runs/42385726717) と[Health #42385726733](https://github.com/hat47x/tei-platform/actions/runs/42385726733) は固定HEADでcompleted/success。初回の`wrong-mode`失敗は**試験Consumerの取得モード誤認**であり、製品SourceにRuleが無かった証拠ではない。両PRは既存`tools/reference-issue-test-ability/context_gate.py`と同じCI/Design文書を変更しており、**同時に機械的mergeできるかは未確認**。CI Greenの足算だけでRequired情報の真の完結を証明しない。

**統合時の独立対照（Owner #3262/#2514、2026-11-08時点で後続#3354に統合済み）**：同じSemantic Validation 1件（さらに他2件を加える拡張）に対し、(1)必須だけ欠ける整形式/同一FP、(2)subject/indexモード取り違え、(3)Index古版、(4)非Required欠落をそれぞれ両手法へ入力し、`required/discovered/loaded/parsed/returned/received` とUnknown理由、Byte/Tool応答量を比較する。一つの採用ConsumerがOwner決定した**用途別Required**を満たせば十分であり、「全件Indexがあるから全業務Ruleが必ず必要」とはしない。**本段落の歴史的な二案比較は後続main統合[#3354](https://github.com/hat47x/tei-platform/pull/3354)で決着**。#3324/#3326はClosed/Unmerged。実MCP/外部Harness完全受領・T1-B・C1・正式Owner意味受理は依然未成立。

### 16.2 宣言されたData／Plugin能力と、実行・適用・保存を切り離す

| 確認対象 | mainの成立済み局所証拠 | 次の受入で要求する追加Witness・非推論 |
| --- | --- | --- |
| **Data役割→Application/Studio表示** | [#3334](https://github.com/hat47x/tei-platform/pull/3334) は税／為替／採番／休日という標準業務Dataの**宣言済みData ID・属性Role**を`ApplicationView.standardBusiness`とStudio表示へ投影した。Go/Node限定試験とCLI例の観測あり | Roleの表示は行レベルの税計算・為替換算・番号発行・休日判定の実実行でない。Function/Rule/Providerの実Consumerと同じMeaning/Scopeで再照合 |
| **Capability宣言とEnvironmentの選択** | [#3337](https://github.com/hat47x/tei-platform/pull/3337) は参照Workspaceで`provides`12件に対し未選択を6件と出す**当時の参照実装**を統合 | 技術名`postgres`/`oracle`と能力IDの字面照合は誤判定し得る。[main統合 #3350](https://github.com/hat47x/tei-platform/pull/3350) は既存`SupportsDatabaseProvider`を使って技術名とCapabilityを正しく照合し、未選択は**4件**に変化した（旧#3337の6件は当時の参照計算の履歴、**別ルール／別Fingerprint**）。4件を実Provider不足・欠陥や実呼出し不成功と断定しない |
| **Pluginの実Selected→Actual Invoked** | main統合[#3300](https://github.com/hat47x/tei-platform/pull/3300) はCLIとStudioの直接`InspectSubject`呼び出し、旧Service Carrierの`application-runtime`専用Scopeを実Sourceで照合した | 同じProvider Contractの明示selected/invoked/Resultを**CLI・Studio 2 Consumer**で観測し、Provider B選択時にA実Invokeという負例を拒否。D1実Plugin配線は[#3301](https://github.com/hat47x/tei-platform/pull/3301)/#951/#1251の未実装領域。静的Go LinkとRuntime/Host Grantも別 |
| **CLI read-only広告と環境副作用** | main統合[#3333](https://github.com/hat47x/tei-platform/pull/3333) は`tei commands` の`view.writeFlags`として`-operation-log`と`-lifecycle-state`を列挙し、`view -execute`による実ファイル生成とno-flag時の無生成を局所Go試験した | `effect:read-only`は**業務意味の非変更**であり、OSファイル・Trace・送出・外部DBの無効果の断言ではない。実Call Variant・Host Scope・Named Outputと実観測を一緒に記録する（#3182/#2515/#1174）。`writeFlags`は完全Effect Modelではない |
| **Build側invokerとinvoked** | main統合[#3336](https://github.com/hat47x/tei-platform/pull/3336) はStage Driverが子Go Buildを起動し、実行ファイルdigest/argv/exitとディスク実Artifact digestを別観測できる参照を追加 | Stage1生成成果物自身がinvokerとなるL3のself-build、Product Plugin/Provider Bind、Owner Meaning/現在Grantは未証明。機能の`declared`・Build`included`・`consumer received`・`currently invocable`は別資格（#2831/[Draft #3283](https://github.com/hat47x/tei-platform/pull/3283)） |

### 16.3 実証の順序・停止線（新Runner／第二資格表なし）

1. **P0-Mの1 Subject／2 Canvas／複数Relation**：[#3263](https://github.com/hat47x/tei-platform/pull/3263)の合成Fixtureの実Go契約試験は[最新HEAD Health #42385303844](https://github.com/hat47x/tei-platform/actions/runs/42385303844)がfailure。Fixture Input/ExpectedのAI草案同士の内部一致やCanvas共有Occurrenceがあっても、製品TEI Loader→Graph/CRUD→Edge/Cell Source逆引き→Consumer受領のPASSにしない。実TEI Sourceから現行モデルを取り込める範囲を明示し、Sourceに無いRole/Relationを推定生成しない。
2. **P0-Rの1 Function／1 Query／1 physical Provider**：`order_entry.recent_orders`のProduct SQLと`public.tei_web_scoped_orders_reference`の隔離PG Referenceは**別Statement/母集団**。Function→実selected Query/SQL byte+`:from`→physical Provider・実Rowset/Scope→Contextの最初の不足辺を[#2586](https://github.com/hat47x/tei-platform/issues/2586) HR-01/02で一つのAttemptへ接続。Plugin能力の環境selected／DB Instance存在／自動導出Operation／Read-only CLI表示を代替Witnessにしない。
3. **P0-Cの2 Consumer**：Data構造とFunction-owned Presentation Instanceは別。Form/Table・Graph/CRUDが同じSemantic IDを表示しても、Source/Workspace manifest・Read Cohort・Role・Occurrence・Grain・現在のDisclosure/Grant一致を別に証明する。Plugin D1、Client Reaction/HTTP、Rule T1-B、SHO/SUI、書込みIntentと独立再Readは各Ownerの別のClaimにする。

**状態／非主張**：これは現行mainの有限参照/Go/CLI実測と未統合Draftの差分を製品受入へ引き継ぐ**設計計画**。本節で新たなGo/Node/PG/CLI/Studio/EKI/SHO/Browser E2Eを実行していない。実三図/CRUD、実Product Read、Plugin二Consumer実Provider Invoke、Runtime Self-Build L3、未知RuleのT1-B、現Host Grant/Business Effect、正式Owner Meaning Acceptanceは**NOT RUN／NOT ACCEPTED**。既存#3262/#2586/#3263/#1251/#951/#2514/#1180/#2831のCase/Ownerへ追加し、第二Reader/Runner/Host/Parser/Authority/Registry・第二Independent Expected正本を作らない。

<!-- tei-composed-plan-tt56-unified-handoff-confirmation-20261108 -->
## 17. 単一Context Gateの成立と「空集合／引継ぎ／実効果」の判定分離（2026-11-08）

**main統合による責務の確定**：[PR #3354](https://github.com/hat47x/tei-platform/pull/3354)は#3324のFocused Validation Inspect、#3326の全Source Index＋独立3 Rule Requiredを**単一の試験専用Consumer**へ統合した。旧二PRは**Closed/Unmerged**であり、二つのContext Gateが製品として併存するわけではない。固定HEAD `0ab2b99f028a867da6681239f8b79691b9a877b4`の[専用CI #42391707962](https://github.com/hat47x/tei-platform/actions/runs/42391707962)はcompleted/success。実CLI read-only **11 Reply**、必要3 Rule／Focused Canonical・Relation・Definition Role/Source照合、**32件の合成Consumer判別Controls**とPinned 6 Step／既知Rule T1-A第7Stepを同じCIで検証。これらはコードMutant killでも全Domain Closureでもなく、`boundedReference=partial`、`qualifiedTestGeneration=false`、C1/NFR未計測、未知Rule自律発見T1-B・外部H3配送・Owner Meaning受理未達を維持する。

**判定の三つの独立要因**：
1. **Required集合の完全性** — 11 Reply・Source Index・Focused Inspectがそろっても、#3263で必要な全Business Action／Relation／FunctionがRequiredに列挙されたとはいえない。公開3 Ruleの一致を三視点図面の「Relation 0件／未定義なし」へ昇格させない。Scope/Actor/Disclosure/Snapshotごとに`required minus observed`を明記。
2. **空集合・業務引継ぎの完了性** — [main統合PR #3353](https://github.com/hat47x/tei-platform/pull/3353)では、旧実装で`EmptyPolicyHandoff`かつ`Targets=[]`・`EmptyJustified=true`でも`HandoffConfirmed=false`なら`CompletionHolds`となり得た反例を修正し、`CompletionUnverified`へ抑制した。6件の回帰を追加し、固定PR HEADの[Repository Health #42391982442](https://github.com/hat47x/tei-platform/actions/runs/42391982442)はcompleted/success。**対象の空が独立に正当化されたこと**と、**相手方が引継ぎを受けたこと**は別Witness。Caller申告の`HandoffConfirmed`も実外部Receiptではない。空のCRUDセルや空のWorkflow Inboxに転用して業務完了と認定しない（#1125/#2386/#1444、独立Expected #2514）。
3. **参照・選択・実行の現在性** — [main統合 #3350](https://github.com/hat47x/tei-platform/pull/3350)ではDB技術名に対応するPlugin Capabilityの選択を既存`SupportsDatabaseProvider`で評価し、参照Workspaceの未選択は**6→4**へ修正された。ただし`selected`は**実Binary実装／実Providerを呼んだ／DB Resultを消費した／現在Grantがある**ことを示さない。[main #3325](https://github.com/hat47x/tei-platform/pull/3325)の`current/stale/unchecked`もCLI参照drift照合の資格のみ。図・表・Formが正常表示されていても実Application ReadやCommand/Host実効果の資格は別。

**既存受入への明示的な投入順**：P0-Mは#3263の3図/CRUDで独立RequiredとRole/Edge/Cell Source逆参照を試験し、P0-Rは#2586で`Function→選択Query/SQL byte+:from→physical Provider→Rowset/Field/Scope→Context`を一Attemptで検証する。P0-Cはその二つの**同一World/Source Revision/Subject/Scope/Read Cohort**が観測できたときだけ結ぶ。Plugin D1/Build・Design Query TT-56は前段の部分資格であり、Product Readの代替ではない。既存Test Runner/Scenario/Predicate/Host/Authority SSOTを増やさない。

**本追補の証拠上限**：main統合PRの固定HEAD専用CIと現行Draftの内容を照合したDocs-only更新。#3353の修正は空集合×Handoffの**局所Go判定**であり、`CoverageComplete`／`HandoffConfirmed`は依然Caller申告で、外部相手方の独立Receiptや実Provider結果ではない。新Go/Node/CLI/PG/Studio/SHO/実Provider/Host試験・三視点独立Expectedの正式Owner受理は**今回NOT RUN**。



## 18. 不完全CRUDの既知セル保持と多Canvasの同一Subject共有を局所受入に追加（2026-11-08）

### 18.1 実Fixtureの新しい二つの反例

[Draft #3263](https://github.com/hat47x/tei-platform/pull/3263)の現在Sourceには、二つのCanvasに同じSemantic Subjectを複数の**異なるView Occurrence**として配置するPressure Fixtureと、そのSource RelationのOccurrence ID、既知Mutantの監査が追加された。さらに`fixture_contract_test.go`には`TestPartialFunctionCRUDKeepsKnownCellsAndUnknownRemainder`が追加されている。このテストは**Fixture入力とExpectedが内的に矛盾しないかの検査**であり、TEI Product Loader→三図/CRUDの実生成・Owner独立Oracleの合格ではない。旧HEAD `2bc0d0f3aa3960751d57745209d94a8253538d72`の[Health #42385303844](https://github.com/hat47x/tei-platform/actions/runs/42385303844)は設計索引不足でFAIL。その後[Draft #3263](https://github.com/hat47x/tei-platform/pull/3263)の索引修正HEAD `6d3eb7f6cf7fc68c4401947faf9cad157ae565af`を確認したが、**その新HEADのGo/CI結果は別途判定**し、旧FAIL・旧PASSを流用しない。

**部分情報の単調性**：`fn.external.dispatch`は`declaredCrudComplete=false`でも、明示`data.customer: R`が与えられたセルは**Rのまま**保持する。同じFunctionの宣言されていない`data.order`等は`unknown-not-enumerated`であり、`none`にしない。`complete=false`を理由に**既知Rをunknownへ消す**のも、未列挙を**noneへ確定**するのも誤り。これはMatrix全体を三値一色へ塗りつぶさず**各セルの明示事実と列挙閉包**を別根拠に持つということ。宣言されたRは実Provider Readの成功や業務Effect完了ではない。

**共有Subject／多Occurrence**：Canvas A/Bに同じ`Semantic Subject ID`を置けるが、`View Occurrence ID`はCanvasごとに別。Subject Identityと位置・Size・折畳み・表示フィルタ等のPresentation Stateを混ぜない。表示要素の移動・Canvas分割・片方のOccurrence削除は、他CanvasのSubject・Relationや元Source意味の削除を含意しない。二Canvasで同一Relationを表示する場合も**元RelationのID＋方向＋Role／複数表示OccurrenceのID**をそれぞれ保持。表示辺を欠いたCanvasと、Sourceから辺が消えたWorldの二つを区別する。

### 18.2 次の最小判別を既存Expected／Scenarioへ渡す

| 判別 | 一因子差分 | 独立Expected（Graph表示/CRUDを別判定） | Owner |
| --- | --- | --- | --- |
| MC-01 | `declaredCrudComplete=false`のFunctionに明示`data.customer:R`を1件追加 | 対象セルだけR、他未列挙はunknown。既知RをUnknown化・他セルをNone化する両Mutantを落とす | #3263/#2514 |
| MC-02 | Semantic Subject同一でCanvas Bへ表示Occurrenceだけ追加 | Subject・元Relation・Meaning版は不変。Canvas数/Occurrence数は変化する。編集対象をCanvas Bの位置情報から自動決定しない | #3263/#1124/#2497 |
| MC-03 | Canvas Aのカード表示だけ削除、Canvas Bに同Subjectが残る | AのOccurrenceだけ除去。元Subject/Relation削除・共有Canvas Bの消滅・Source Rewriteを起こさない | #3263/#2500 |
| MC-04 | 空のBusiness Workflow対象集合でHandoff未確認、明示Rセルは存在 | `#3353`の局所修正に従いCompletionはUnverified。CRUDの既知R/列挙未知と外部Handoffの業務完了を相互に代用しない | #1125/#2386/#2514 |

実装順は**Fixture監査→実TEI Source/Meaning→Headless二Canvas Graph／Matrixを同一Revisionで実出力→Occurrence/CRUD CellからQualified Sourceへ戻る→独立Expectedで1 Mutant以上検出**。両Canvas間の編集同期／SHO Writer／実Host Permissionや`view -execute`のEffectは別の試行。大きいCanvas分割の操作性を先に新しいCore Canvas/Task型にしない。Cardが未整理のDocument/Itemであっても、標準Data Structureはその表示方式と独立して保持し、可視カード化を強制しない。

本追補はSourceと既存テストの**実観測→受入条件への引継ぎ**であり、MC-01〜04の新Product Graph/CRUD E2E、Go Mutation実行、複数Canvas GUI、SHO、業務Receipt、Owner意味受理は**NOT RUN**。第二Graph/Domain/Scenario SSOT、独立Context Reader、Host／Runnerを増やさない。


## 19. 三種の入力Sourceから三視点へ渡す前の実体帰属（2026-11-08追補）

### 19.1 「Sourceとして存在する」を「業務意味として読めた」にしない

他レーンの実CLI調査（[#3346](https://github.com/hat47x/tei-platform/issues/3346)、[Draft #3356](https://github.com/hat47x/tei-platform/pull/3356)）では、`docs/`・`plan/`配下のMarkdownをWorkspace `sources`に置いても、現行Loaderは`*.tei`の意味解釈を行い、設計Markdownは`managedFiles`に現れても`readByLoader:false`に留まる。これらを`Data/Function/Businessを読み込んだ`証拠へ転記してはならない。#2621のMarkdown Authoring/Sourceと、#3262の実Required/Received ClosureがOwner。別のMarkdown専用意味Parserを本PRで作らない。

**三視点モデルへの入場は異なる3種類**として扱う。

| 入力の区分 | 実際の到達点 | 三視点への受渡し上限 |
| --- | --- | --- |
| **Accepted TEI Source** | 選択Workspace／実Loaderが読み込んだSource、型付きSubject/Relation/Role、Ownerが限定的に受理した意味 | 対象Scope/Revision/Closure内でGraph/CRUDに利用。Displayと編集Intent/現在Grantは別 |
| **Managed Markdown / Imported Document** | 管理対象・内容保全／Source Locationを示せるが、`readByLoader:false`やParser/Meaning未受理 | 文書の引用・候補・Raw Itemとして提示可。宣言済みData/Relation/Function/CRUDの存在へ自動昇格しない |
| **Observed Physical DB Schema** | Catalog Readerが観測したTable/PK/FK/UNIQUE、観測時点・Fingerprint、物理Provider/Scope | Data/Relation/Keyの**Structural Candidate**。複合列や方向を保持するが、業務Relation/CRUD、Tenant Grant、実DDL Effect・Owner受理ではない |

[Draft #3372](https://github.com/hat47x/tei-platform/pull/3372)は既存`SchemaView`に`CapturedAt/Fingerprint`、`TableView`に`UniqueConstraints/ForeignKeys`をread-only追加する**未マージ製品コード候補**で、実`Postgres CatalogReader→Introspect→Derived→Workbench`の経路を**fake Catalog**から検査する。これを受理するなら、物理複合FKの実観測原型とSource RevisionをCandidateに渡せるが、**実PG Catalog・Actual Read/Consumer・Accepted Relationは未検証**。Presentational ER/CRUDは物理Constraintが無いことから業務Relation無しを推論せず、同じ列の別FK二本を1辺に畳まない。受理票は#3270/#1126/#3263がOwnerで、#3303は候補への意味受入条件だけを記録する。

### 19.2 Design Context Gateは進展したが、閉包の範囲は限定

main統合済み[PR #3354](https://github.com/hat47x/tei-platform/pull/3354) の固定HEAD[専用CI #42391707962](https://github.com/hat47x/tei-platform/actions/runs/42391707962)は**11実Read-only CLI Reply／32合成Consumer Control、既存6 pinned Scenario・T1-A候補と独立拒否**を検証した。3つの独立Required Rule、Focused ValidationのCanonical属性・Relation/Parent・Source Index Definition／Artifact・版一致まで照合する。しかし`boundedReference=partial`、`qualifiedTestGeneration=false`のまま。**Rule/Relationの限定確認を、Function/Business/全業務モデルの閉包やDiagram/CRUD出力の完全性へ昇格しない**。

三視点P0-Mへの最短接続は、まず実TEI Sourceに存在する最小のData/Relation/RuleだけをGraphに投影し、Source LocatorとRevisionを保持して`supported`とすること。未受理のFunction CRUD/Business Action/外部Docは`unknown/incomplete`と明示してよく、Productが要求する三図/CRUD「完全」の合格は別のRequired集合を独立に観測するまで保留する。

### 19.3 同一記号による誤接合を落とす差分検査

| 対照 | 一因子差分 | 独立Expected／停止条件 | Owner |
| --- | --- | --- | --- |
| SA-01 | 同じ`sales.order`文字列のManaged Markdownと実TEI Loader済みSubject | 同名だけで同じAccepted Meaning/Source Locatorとしない。`readByLoader:false`の宣言からCRUDを確定しない | #3346/#3262 |
| SA-02 | Physical FK二本が同じ端点／異なる列組・制約IDを持つ | 物理観測は2本を保持。業務1:N/組成・同一Relationという受理は未実施 | #3270/#1126/#3263 |
| SA-03 | Schema Snapshotの`Fingerprint`あり、Source/Provider ScopeやSnapshot完全性が未証明 | 指紋の存在だけで全DBの閉包・現行実Read・Key不変を認定しない | #3270/#2514 |
| SA-04 | Rule/Relationの11実CLI受領は成立、Function/Businessが未取得 | 局所RequiredだけPASSし三視点/CRUD完全性は`partial`。観測されないCUDを`none`にしない | #3262/#2514 |
| SA-05 | Source A→B→AでFingerprintが戻ったが、Graph/Canvas表示Occurrence/操作epochは旧値 | Source内容の再一致と現在のSelection/Action Offer/Grantを別検収にする | #2582/#2598 |
| SA-06 | WorkbenchのFK・UNIQUE候補だけでDiagramを出し、Query/Providerの実ReadもCRUD宣言も無い | Read-only物理ER候補なら限定表示可能、受理済み業務ER/CRUD及びRDB実読取の成功は`unverified` | #3372/#2497/#2586 |

このSA-01〜06は新しいProduct Gate/Fixture/Scenario正本ではなく、#3263の三視点Fixture／#2514の独立Expected／#1180のScenarioへ**一因子差分**で渡す。既存Graph/ModelRef/Workbench/read-only Design Contextの再利用を優先し、第二Graph・DB/Markdown Parser・Owner Registry・Host・Plugin Resolverは新設しない。

**現在の実証上限**：#3354の固定CIはSource/Rule/Relationの限定Consumer検査、#3372は未マージのfake-Catalog製品コード試験候補、#3346のMarkdownはRaw/managedのまま。本PRから新しいGraph/CRUD Product投影、実PG Catalog Reader、Studio/SHO/Host副作用、Owner意味受理、SA-01〜06のGo Mutant Testは**NOT RUN**。


## 20. 最新mainの一機能入場：三視点・実Read・共通APIの検収を別票にする（2026-11-08）

この節は§14〜19の採用前Fit/Gapを**現mainの最新状態に照らして更新する**。その節に記載した「11 CLIのみ」「MCP入口なし」「Workbench #3372が現行Draft」等の歴史的状態を2026-11-08現在の達成状態として再利用しない。

### 20.1 到達した限定実測と、なお足りない証拠

| レーン | 現main／担当PRで確認できた狭い成果 | 検収時に停止する境界 |
| --- | --- | --- |
| Design Context / Function | main統合[PR #3482](https://github.com/hat47x/tei-platform/pull/3482)：固定HEAD専用CIで**13実CLI Reply**・42件の局所Consumer Controlsを実行し、`semantic-capability:sales.order.submit`のDeclaration、`application:order_entry`依存、3 Validation/Source Indexを限定照合 | `boundedFunctionSlice.declarationStatus=bounded-present`は**Product Function→Query/physical Provider実Invoke**ではない。検証Function `sales.order.submit` の証拠を、別の `order_entry.recent_orders` の実Readへ移し替えない。Function/Business全Required Closure・T1-B・実Host権限は未資格 |
| CLI/MCP/API共通操作 | main統合[PR #3403](https://github.com/hat47x/tei-platform/pull/3403)に**MCP参照Adapter 4 read-only tool**が存在し、main統合[ADR-0048 / #3439](https://github.com/hat47x/tei-platform/pull/3439)はCLI/MCP/API/Web/Desktopの共通Application API方針を採用。[#3458](https://github.com/hat47x/tei-platform/pull/3458)で`validate`のCLIとMCP参照Adapterの正常・Load失敗・意味エラーを局所照合 | 現時点で「MCP入口なし」と言わない一方、残る3 toolの全Body一致、実Studio／Web/外部H3の完全受信・開示、ProviderのSelected=Actual Invoke、現在Grant/Effect/Receiptを保証しない。**正本のMeaning/Effect/OwnerはTransportを越えて同じ契約、応答形/到着はAdapter別** |
| Source/Import鮮度 | main統合[#3477](https://github.com/hat47x/tei-platform/pull/3477)で同一WorkspaceのSource-only改訂をWorld/SourceIndex再照合で限定拒否。未マージ[Draft #3479](https://github.com/hat47x/tei-platform/pull/3479)では**Importした別ProviderのRaw byte改版がConsumerの二指紋に表れない**反例をGo試験候補として追加 | `World FP`と`SourceIndex FP`が同じでも、必要なImported Providerの`Source Location/byte/occurrence`が同じとは判定しない。DraftのGo/CIが未確認ならこの反例は**提案された判別**であり製品保証ではない。A→B→A中間履歴やAtomic Snapshotも別 |
| 物理DB→業務三視点 | Workbench旧Draft [#3372](https://github.com/hat47x/tei-platform/pull/3372)は**closed/unmerged**、後継[Draft #3486](https://github.com/hat47x/tei-platform/pull/3486)が物理FK/UNIQUE/Index、観測時刻・Fingerprintのdeep-copyとfake CatalogのGo検証を所有 | 物理Constraint/Indexの存在・非存在は、受理済み業務Relation、Function CRUD、リアルPG Reader、現行Scope/Grant/DDL Applyの証拠ではない。#3486 exact HEAD／Healthは別途確認 |
| 三視点Graph/CRUD | [Draft #3263](https://github.com/hat47x/tei-platform/pull/3263)では合成Fixture監査、Source/Relation identity/複数Canvas・CRUD不完全セルの検証を拡張中 | 合成FixtureやSourceTokenが正しく見えても、**受理済み実TEI Source→Graph/CRUD→Edge/Cell Source Navigator→実Consumer受領**の一試行は未実証。2本の同端点RelationをKeyやlabelで統合しない |
| Product Read と実Effect | [main #3481](https://github.com/hat47x/tei-platform/pull/3481)は一機能横断受入順を全体計画へ統合。Go HostのRoute/Action2視点は[main #3485](https://github.com/hat47x/tei-platform/pull/3485)で合成Fixtureを使った別契約比較を追加 | E1 Route/Actionの局所文字列一致、`HTTP 202`、Graphの宣言`R`、MCP `ok`は、実Provider/業務Outcome/独立Resource Readの同一Attempt成功ではない |

### 20.2 実装の最初の欠落辺を「Consumer共通」と「製品実行」に分ける

新しい一括Renderer、第二API/Source Registry、第二Test Runnerを設けず、受理する限定Fixture・Source/Scope・Versionを固定して、既存コードで次の独立経路を計測する。

**M0（構造）**：実TEI Loaderで確認できる一つのData Subjectと**異なるIDを持つ二Relation**を、型付きGraphへread-only投影する。GraphのNode/Edge/Occurrenceと`Source Locator/Revision`を対応づけ、Relationが不足・hidden・import staleの場合は`unresolved/partial`。WorkbenchのFKは`structural candidate`のまま。Function CRUDの未取得セルは`unknown`、明示`R`は失わない。逆Patch/Canvas Editorは別資格。

**R0（実データ）**：#2586の単一Product Functionを実選択し、`Query ID/SQL Source byte + :from実値→datasource実Provider→Executed Statement/Rowset/Scope/Read Cohort→Data Context→Form/Table`を**同一Attempt**で観測。#3482の`sales.order.submit`の宣言成功から`order_entry.recent_orders`を実行済みとしない。未接続ならR0だけ`unverified`、M0の部分Graph検収を不当に不合格としない。

**C0（共通Consumer）**：ADR-0048の共通Operationから、既に一致が確認できた`validate`をCLIと参照MCPへ投影する。Graph/CRUDやProduct Readがまだ共通API契約として登録されていない場合、存在しないToolを広告しない。Workspace/Source/Scope/Version/実受領の必要範囲と、Tool副作用・外部Sink保存/Current Grantを別に示す。

**X0（交差）**：M0とR0の両方が揃ったときだけ`Subject Qualified Identity/Relation Role/Workspace/source occurrence/Policy/Read Epoch`で結ぶ。同じ表示Key・Logical DB名・Normalized World FPだけから実同一性・共同Snapshotを宣言しない。

### 20.3 追加する独立Expectedの判別点（既存Ownerへ委譲、NOT RUN）

| Case | 一因子の対照 | 拒否すべき誤認 | Owner |
| --- | --- | --- | --- |
| IA-01 | Consumer二FP固定で外部Imported ProviderのRaw byteだけ改版 | 結果値が同じでも現在の原典引用・Source Revisionを自動継承しない。外部Sourceは未受領ならunknown | #3479/#3262/#2514 |
| IA-02 | `sales.order.submit`宣言・必要Validationは揃うが、そのFunction/Query/Providerの実Invocationは無し | 読込済みDeclarationとProduct Function実行の資格を別欄へ置き、`Function executed`と宣言しない | #3482/#2586 |
| IA-03 | 同じNode間に別IDの物理FK二本、Business Accepted Relationは一部だけ | physical observed/structural candidate/accepted意味を別列で保存。表示一致から一意Inverse Patchしない | #3486/#3263/#2497 |
| IA-04 | CLI/MCP`validate`正常値が一致し、別Transport Consumerは途中切断または別Workspace | 局所同値を保持し、実消費Replyの完備/Grant/Source Scopeが不足すれば他ConsumerへPASSを伝播しない | #3420/#3262/#2516 |

実行FixtureとIndependent Expectedは既存#3263/#1180/#2514、Interface/Consumerは#1158/#3420、Sourceは#3262/#3479、Workbenchは#3270/#3486、Product実Readは#2586が正本。#3303は**合成写像とConsumer表示の資格**に集中し、全体優先順はmain統合#3481、詳細取込計画は[Draft #3483](https://github.com/hat47x/tei-platform/pull/3483)へ委譲する。今回のIA-01〜04のGo/CLI/DB/Studio/SHO実行、実Provider Read・逆書込・Owner Meaning Acceptanceは**NOT RUN**。
