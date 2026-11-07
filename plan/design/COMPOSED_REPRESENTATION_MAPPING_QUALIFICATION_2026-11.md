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
