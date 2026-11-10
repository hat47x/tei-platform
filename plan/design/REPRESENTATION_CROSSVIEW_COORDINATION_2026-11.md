# TEI 表現型間の選択・状態連携・更新境界（2026-11-07）

> **統合Draftの位置づけ（2026-11-07）**：この資料は以前の設計Draftの内容を保存し、現行mainからの統合計画へ取り込み直したものです。最新の横断検証順と追加反例は `plan/design/COMPOSED_REPRESENTATION_MAPPING_QUALIFICATION_2026-11.md` を参照してください。本文中の旧Draft参照・NOT RUNは当時の証拠の限界として保持します。
- **状態**：他チャットのTEI/SHO/SUI/TAIの構想と現行Issueの照合から抽出した、追加の設計・Headless検証計画。**採用前／新試験 NOT RUN**。Core Schema、共通Host、Renderer、同期Runtime、権限Engine、独立Scenarioを新設しない。
- **前段**：[Draft #3192](https://github.com/hat47x/tei-platform/pull/3192)の「Data topology／Representation topology／Function-owned instance／媒体時間軸」を前提とする。ただし#3192は本書作成時Open Draft・main未統合。前段の構想を正式採用済みと主張しない。単独の縦断実証へは既存のAccepted Meaning/Function Contextが必要。
- **責務**：#2506 Function Data Context／Result Slice、#2507 Pattern/Representation Binding、#2497 Projectionの意味保存・損失、#2582 変更依存と再取得、#2500 View→Intent逆写像、#2598 Intent→Request対象、#997 Renderer/Native Kit、#2516 Consumer Disclosure、#2514/#1180 独立Expected/Scenario。SHO #19／SUIは異質ConsumerでありTEI Function Sessionの別名ではない。

## 1. 基本判断：共有するのは「意味上の対象」ではなく、資格付き対応

同一FunctionにTable/Tree/Graph/Chart/Audio等の複数表現インスタンスを配置し、同じContextを参照しても、表示要素の同一性と業務対象の同一性は異なる。同期は**共有されたScene Graph/Component ID/行番号によって暗黙に成立させず**、Source/Data occurrenceから各表現Occurrenceへの部分写像（選択可能・不可能・一対多・曖昧）と、その有効条件を検証する。

| 識別・状態 | 例 | 同一視を禁じる対象 |
| --- | --- | --- |
| Semantic Subject / Relation occurrence | OrderLine L1、Membershipの関係発生E1 | 表示行、Graph nodeの見かけのラベル、経路 |
| Context Definition / Instance | 受注機能のD1明細、ある時点のH1選択 | 別Functionの同じD1、別Tenant・別Selection epoch |
| Query Result occurrence / Grain | JOIN結果にL1が3回、集計group G1 | L1実体3件、groupがL1そのもの |
| Representation occurrence | Table row r7、Tree path A/X/B、Chart bar C | Semantic SubjectやDB PK |
| Interaction selection / focus | 表示Instance Aの現在選択とepoch | 書込みTarget・Host Grant・Business Effect |
| Source / Document span | 文書の段落/byte・文字範囲、SHO引用元 | Layout後のページ、朗読時刻 |
| Media position | Track/sample/frame/time interval | Domain clock、文章位置、業務状態 |

**型の再利用≠Functionの再利用≠Actionの存在≠現行Authority**。表現型のPalette/Previewは単独で存在してよいが、業務意味を持つRepresentation InstanceはFunction Data Context／用途・Target・Action offerに従属する。SUI Canvas/SHO Document Sessionには各製品固有のContext Ownerを尊重し、TEI Functionであると偽装しない。

## 2. Composition・Projection・Coordination・Operationの分離

1. **Composition**：Container/Panel/Tab/Split、Slot・並び・表示切替。どの表現型を機能に配置するか。Tab reorder/折畳み/Zoom/Scrollは原則Presentation stateであり、Relationの変更ではない。
2. **Projection**：Data/Result Sliceから表現要素への粒度/単位/順序/時間軸を含む写像。Page/Chart/Treeによる損失、Renderer版、取得済み範囲・開示条件は#2497/#2506が担当。
3. **Coordination**：一つの表示で起きたselect/focus/filter/seek/navigationを、**どの意味・どのTarget viewへ・何を転送するか**解釈する。相手が異なるgrainならhighlight候補や関連表示に留める。必要なら別Readを**候補**として示すが、UI通知だけでProvider実Read・Disclosure・Target Scopeを拡大しない。
4. **Operation**：ViewChangeを業務編集Intentとして解釈できるか#2500で審査し、既存Action→#2598の実Request/Target→#1174 Host Grant/Effect/Receiptで再判定する。表示の同期をSubmitと混同しない。
5. **Source authoring / Document output**：Studioのモデル定義変更やSHOのRaw Markdown保存は、業務データ更新と別の保存・Source Revision・Acceptance経路。SUIのCanvas配置/島変更も意味Relationの自動編集ではない。

**連携の結果の種類（説明用、公開Enum未採用）**：exact target／many target candidates／aggregate or related target／unknown／out of scope／redacted。見つからない要素がある場合、元Dataが不存在なのか、対象外・部分取得・非開示・Renderer非対応かを区別する。暗黙に選択対象を最初の1件へ丸めない。

## 3. 同期の適格性を判断する最小の証拠列（新Schemaではない）

- Origin: Source/Accepted Meaning/Context Definition＋Instance、selection epoch、Result Slice/read attempt（Provider、Scope、cohort、revision、粒度、partial）、元表示InstanceとOccurrence。
- Mapping: Data→View変換の種類と選択済み変換器/Plugin/Target版、forward mappingのID・Relation path/Path occurrence、情報損失、逆写像の一意性。媒体ならSource span↔playback interval/track/clock/alignment version。
- Event: selection/focus/filter/zoom/layout/seek/drag/editの**作用種別**、誰が起こしたか、どの範囲へ通知するか、同じイベントの再放送ループ抑止。
- Destination: 相手Function/Context Instance/Representation Instance、独立した取得・開示Scope、現在時点、表示できるか・選択候補にできるか・Operation候補を発見できるかを別々に評価。
- Qualification: Source/Model/Rule/Function/Query・Context epoch・Disclosure/Plugin/Rendererの失効条件、未評価の依存、観測範囲と次に必要なAction/Read。**同期成功は情報意味受理/書込みGrant/業務完了の証明でない**。

**公開/秘匿への追加注意**：Graphの隠しedge、集計された件数、非開示対象IDなどを、別Viewのselect候補数・highlight・tooltip・イベントログを通して漏らしてはならない。同期自体が情報開示になり得るため、Target ConsumerのDisclosureを既存#2516で判定する。

## 4. 局所状態・意味・保存の変更影響を分ける

| 変化 | 保持・再評価の原則 | 誤った推論 |
| --- | --- | --- |
| Panel/Tabの並び、Zoom/Focus、Canvas座標のみ | 明示的依存がなければData Meaning/Query Resultは不変。Target-specific配置は再描画可 | Relation/業務保存を自動起動 |
| Table→Chart/Treeなど表示型変更 | Source Contextの意味は共有。新Projectionのgrain・loss・Scope・Plugin適合だけ再確認 | 表現可能だから同じSelection/編集権限 |
| Contextの親H1→H2→H1、同IDへ戻る | epoch/Read cohortを束縛。古いH1応答を新H1へ復活させない | IDが戻ったので旧View/Actionを再資格化 |
| Result Slice/Filter/Query/Rule改版 | 個別ClaimとViewの依存に相対的に局所再評価。部分集合のmissing/NULL/unknownを保持 | 全面失効、または全件Completeへ昇格 |
| Graph↔TreeのPath collapse、Chart集計 | 出現/aggregateの一意な逆写像がなければread-only/候補へ停止 | 表示node/棒を任意Entity PATCH |
| 文書reflow、朗読voice/速度変更 | 原文spanの保持とPresentation time/alignmentの再資格化を分ける | ページ番号や旧時刻が永続Source ID |
| Plugin/Renderer版・Policy/Grant変更 | 映像/表の表示資格・Consumer開示・操作資格をそれぞれ再確認 | 旧Bundle/表示成功が現行Write grant |
| Commit応答喪失・旧Receipt到着 | 観測未知と旧/新試行を分け、必要時Receipt Lookup/独立Read | UIの同期された値＝対象Requestの成功 |

**失効単位は一律のglobal revisionではない**。依存が証明できる範囲のみ保持し、未知の依存はunknown。変更影響判定・再取得は#2582、Evidence/Claimは既存#1163等へ引き渡し、第二のReactive Engineを作らない。

## 5. 同一Fixtureでの差分判別（新Scenario SSOTは作らない）

既存#2586の公開合成H1/D1={L1,L2}/D2=3、D2部分1/3を主Fixtureとし、#2497/#2500のGraph DAG多重Path、SHOの合成文書を独立Consumer圧力に使う。**期待値は実装から自動生成せず、#2514/#1180の独立Expectedに接続**する。

| Case | 入力の一因子差 | 判別する独立Expected・禁止事項 | Owner |
| --- | --- | --- | --- |
| MV-01 二View同実体 | L1がJOIN表示3行、Tableから一行選択、FormはL1 | 意味上のL1を選べる場合、同一EntityのFormは対応。3行＝3業務Entity/3更新要求ではない | #2506/#2507 |
| MV-02 Tree多経路 | DAGでnode Xへ2経路、Graph Xを選択 | Treeでは2つのPath occurrenceに候補があり得る。1経路を勝手に実Relationの親と断定しない | #2497/#2507 |
| MV-03 集計棒 | D1合計Chartのbarを選択しTable行を同期 | Aggregate groupをL1/L2の単一Entityとしない。明示drill-downのみ候補、未宣言ならread-only | #2507/#2500 |
| MV-04 見えていない行 | D2が1/3部分、別Tab未取得、Table上0行 | Tab不可視/ページ外≠業務上の不存在。同期によって全件取得済みや対象のGrantを主張しない | #2506/#2582 |
| MV-05 Epoch ABA | H1→H2→H1で旧H1応答が最後に到着 | 同じH1の識別子でも旧response/selection/Action Offerを現行として採用しない。別Window H1は巻き込まない | #2582/#2598 |
| MV-06 秘匿Relation | Graphの非開示edgeが存在しTableでは見えない | Selectionの候補数・hover・ログから隠しedge/IDを露出しない。見えないことを不存在ともしない | #2516/#997 |
| MV-07 UI layout vs Source | SUI Canvasの座標/島だけ変更、SHO文書をreflow | UIだけの変更はTEI Relation/受理済みMeaningの変更でない。Source本文が変われば別Reader/Review資格 | #997/SHO #19 |
| MV-08 文章と音声選択 | 文章span Sを朗読音声区間Tに対応、voice版を切替 | 旧Tの同期資格失効。Source spanは保持可能。T不明ならseek/highlightの正確さを宣言しない | #2497/#2507 |

**誤実装Mutantの最初の3種類**：M1 row indexをEntity IDとして配布、M2 aggregate barを任意L1へ逆更新、M3同IDの古いSelection/Readを復活。正例・負例の両方で、無資格Operationが生成されないことを検査する。#2514が独立ExpectedとMutant判別力を担当する。

## 6. 計画と停止線

- **C0（Headless基準）**：#2506/#2507/#2497/#2500既存Contractに対し、MV-01〜04で各View occurrenceの由来と戻り先を説明できる比較票。GUI、Source意味受理、実Provider実行を要求しない。
- **C1（連動と無効化）**：MV-05/06のepoch/Disclosure、#2582のClaim単位依存と連結。select/filter/seekはread/operationではないという境界を検収。
- **C2（異質Consumer）**：MV-07/08をSHO/SUI/TEI CUIに分け、Render/Document/Canvas固有の操作意味・保存/Acceptanceを分離。必要なときだけ#997 Renderer/Nativeへ提供。新共有Canvas/Media Runtimeは作らない。
- **C3（実TargetとEffect）**：#2586 Function→Provider実Read後に、同一Context/Result Slice/Operation revisionをWeb/CLI/Nativeで照合。編集が必要なケースのみ#2500→#2598→#1174の同一Attempt/Receipt/独立Readで確認する。未接続TargetのPASSは主張しない。
- **判定**：各段階で「今回作成した設計票」「机上比較」「合成コード実行」「実Provider Read」「Target UI観測」「Host Effect/独立Read」「Owner意味受理」を別欄に記録。既存Gate A/B/CやProduct Statusは変更しない。

**引継ぎ先**：主線=#2507（表現型配置と同期適用条件）および#2506（実Context/Selection）、写像=#2497、再評価=#2582、編集=#2500/#2598、実Reader=#2586、UI=#997、独立Expected=#2514/#1180。前段#3192のmedia-axis計画がmainへ未統合なら本書のMedia論点は適用仮説に留める。
