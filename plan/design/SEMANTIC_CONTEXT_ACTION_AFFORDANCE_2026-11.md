# Semantic Contextに基づく汎用Action発見・UI Affordance・実Request接合（2026-11-02）

**状態：設計提案／SA-01〜12 NOT RUN／Owner未受理。** 既存のX-TEA継承SmartDB・汎用機能を、UI画面タイプから独立したActionの構成として展開するための反証可能な計画。正式なDSL Syntax、Core Runtime schema、Action ABI、新たなHostやRegistryを採用する文書ではない。

## 1. 基本仮説：どの画面に置くかより、何を対象として何をするか

標準・専門Actionを「画面タイプ別のボタン／専用UI機能」ではなく、**受理済みSemantic Data/Relation、Function Data Context、意味的Role、適用条件、既存Service Contract**に関連付ける。Table、Form、Graph、Tree、Chart、Headless CUI、外部Hostは、そのAction候補を用途別に表現するConsumerである。

X-TEA由来SmartDBは業務データアクセス以外も対象になり得る**汎用Plugin/Pattern群**として評価し、TEI Coreの特殊なBusiness Hostや汎用DBエンジンを増やさない。単純な標準UIは標準データ構造から構成する。UIの表示／活性などは簡潔なScript/DSLを優先し、Studioの共通Text/Property/Table/Tree/Dependency Viewで編集・説明できるかを検証する（[#988 Script-first追補](https://github.com/hat47x/tei-platform/issues/988#issuecomment-6573241174)）。ノード図や専門UIの追加は必須条件ではない。

**分離したい五つの主張**：①Actionを意味的に発見可能、②Consumerが表示・提示可能、③ViewChangeから対象を正しく束縛可能、④現在のHost/Providerで実Requestを実行可能、⑤独立Effect／業務完了が確認済み。①〜③の合成PASSから④⑤は導けない。

## 2. 既存責務を使う最小分割

| 関心 | 主な検査 | 既存Owner |
| --- | --- | --- |
| Data/Meaning | Entity・Attribute・Relation・Role・Key、Subject、Unit、Presence、Owner意味受理 | #1120/#1126/#2495/#267 |
| Function Data Context | Definitionと配置Instance、Selection epoch、Scope、Grain、Result Slice completeness、Source/Rule版・as-of | #1127/#2506/#2582 |
| Function Pattern / Action | 要求Shape/Roleと対象Operation、前提、意味的Action purpose、Effect footprint、適合不足、Service Contract参照 | **#2507**、#951/#988/#1129 |
| read-only Action Discovery | Context×Patternから意味的候補・不一致・不十分な根拠・Target未対応を説明。許可や実行保証はしない | #2507/#1127 |
| Presentation Affordance | Button/Menu/Shortcut/Command/Property等に投影。visible/enabledをBusiness/Authorityへ昇格しない | #997/#1000/#2391、Studioは#1124/#1824 |
| ViewChange→Intent candidate | 編集対象となるField/Entity/Relation occurrence、不可逆集計、部分PATCHとReplaceの違い | #2497/#2500 Draft #2517 |
| Candidate→Request | current target/Action/Context/Provider Mapping/Meaning/Payloadの実束縛 | #2598/#2611/#2485 |
| Authority/Effect/Business | 現在Host Grant、Tx、Receipt、独立read、Ownerの最終完了規則 | #1174/#1129/#1125/#1163 |
| AI/文書への説明 | Claimの十分性、用途相対開示、履歴と現在性 | #2516、SHO #4/#6、TAI #168 |

同じAction定義を二つのContext Instanceに適用した際、定義が共通であることと、選択対象・Dirty State・Authority・Provider世代が同じことを混同しない（#2507 AG-03、#2597）。Actionの前提/保証の局所適合は、複合処理全体の安全性を意味しない（#2507 AG-01〜20）。

## 3. 適用判定と提示・実行の別

**意味的な適用の候補**を考えるときは、そのActionが実際に必要とする条件に限り以下を検査する。

1. Subject/Kind/Relation Role、Nodeとedge occurrence、Semantic ID、Key、Role、Grainが一致するか。画面の行番号・Graphの座標は業務IDではない。
2. Actionが単件変更・存在主張・全件更新・関係変更・集計のどれを要求し、Scope/取得完全性がその量化条件に十分か。部分取得の単件PATCHは成立し得るが、同じ部分取得で全件DELETEを認めない。
3. Fieldのmissing/null/false、Sourceの省略、Integer/Decimalの精度、集計の中間状態・JOIN bag多重度、時点・Ruleが、当該Actionの意味を維持しているか。
4. 既存Service Contractで、Action purpose、Input/Output、Read/Write footprint、失敗の意味を説明できるか。submitとcancelは同じTarget/Shapeでも別Action。
5. Actionを表示したいConsumerが必要なPresentation機能を持つか。未対応Rendererは表現不能とし、Actionの意味的不存在にはしない。
6. 実行要求の時点では、Provider/Target/Mapper、現在の許可、Resource/Scope、Revision Fence、CommitとEffectを別Ownerが検証する。発見時点のread-only説明やUI活性表示は現在の実行許可ではない。

簡潔な表示制御Scriptの例（正式DSL構文ではない）：visible(order.discount) = customer.kind == corporate、enabled(action.submit) = form.valid && !form.busy。ScriptによりButtonがhidden/disabledでも、他Consumerから当該Actionが存在しなくなるわけではない。逆にenabled=trueでもHost Grantなしなら実Effectは拒否される。UIでの禁止は業務不変条件を代替しない。

編集操作を伴わない「このContextで利用できるAction候補の説明」はread-onlyで独立に検証できる。UI編集からの逆対応は#2500、Action候補の実Request使用は#2598、Effectは#1174が担当し、責務を一つに束ねない。

## 4. 最初の合成Fixtureと独立期待

**架空の固定入力**：親H、明細D1={L1.amount=100,L2.amount=100}（2/2取得）、配送D2={P1,P2,P3}（初期1/3取得）。D1独立SUM=200。全D2をJOINした場合の6行SUM=600、SUM(DISTINCT amount)=100。組織Relationshipの別対照はPerson P2と二所属edge A2/A3。Owner受理済みの業務定義ではない。

| ID | 正負対／反証したい誤成功 | Ownerへの受渡し |
| --- | --- | --- |
| **SA-01 同一Operation・異UI** | L1 amountを100→120とする明示Field PATCH。Table・Form・CUIでは同じMeaning Target/Action Purpose候補を提示。UIコマンド名が違っても意味差分を複製しない | #2507→#2500 |
| **SA-02 集計の逆操作** | Chartの200→220からL1に20加算すると無断決定しない。明示的な配賦Service Actionがある場合だけ別候補 | #2593/#2500 |
| **SA-03 NodeとRelation edge** | P2 Nodeの選択をA3 edgeのunlinkやP2 deleteに自動変換しない | #1126/#2500 |
| **SA-04 Context部分取得** | D1 2/2のL1単件PATCHと、D2 1/3から配送全件を確定するActionを別判定。Partial→Allの局所型適合を拒否 | #2506/#2507 |
| **SA-05 UI表示制御とGrant** | enabled=falseでもGrantの不存在を断定せず、enabled=trueでもRevoked Grantなら実Effect拒否 | #988/#997/#1174 |
| **SA-06 Context複数配置** | 同じFunction定義のInstance I1/I2間でepoch・親anchor・dirty stateを共有しない。A→B→Aの再選択で旧候補を現行にしない | #2597/#2598 |
| **SA-07 PresenceとUpdate footprint** | 見えないprivateCost/optionalTitleがあってもField限定PATCH候補は可能。表示Recordからの全面Replace・暗黙clearを拒否 | #2585/#2497/#2500 |
| **SA-08 UI未実装と意味の不在** | CUIにはAction候補が存在するがGraph Rendererに操作Widgetがない場合はnot-rendered/unsupported。意味的にActionなしと判定しない | #997/#2390 |
| **SA-09 Provider経路** | 同じSemantic IDでもPrimary DBとArchive DBがある。Action候補のID/shape一致から実Commit先を確定しない | #2611/#2598 |
| **SA-10 Script-first共通Studio** | 小さなvisible/enabled ScriptからText→parse/inspect→標準Property/Tree→編集→原Source再評価が往復する条件を確認。曖昧Spanはread-only | #988 SF-01/03、#1124 |
| **SA-11 Domain Plugin拡張** | 専門Action Pluginを追加したTargetと不在のTargetで、データの共通read-only探索は維持しつつ専門Actionをunsupportedにする | #951/#2507 |
| **SA-12 同形・異義** | 同じorderへのsubmit/cancel、同じAction名でも旧Meaning/Provider版を、同一の実行意味に無断束縛しない | #2507 AG-07/08、#2598 IB-12 |

試験票の観測列候補：Source/Action/Context version、Subject/Role/Grain/Scope、Consumer、表示の有無、Action候補IDとTarget、適用に不足する前提、現在のAuthoritative Provider/Grant、実Request/Effect/Receiptの有無、unsupported/unknown。これは新たな共通Manifest/APIではない。

**区別して測定するもの**：①新専用UIの実装必要数、②意味的Action定義の重複数、③Plugin追加時のCore修正数、④誤った実行候補と過剰な全面拒否、⑤Consumer間の同一Meaningの保持、⑥実Providerへの正しい束縛・拒否・結果観測。単にButtonが描かれたことや局所Fixture PASSを業務適合に数えない。

## 5. 実行順序と受入範囲

- **M0 本Draft**：既存責務と最小反例を固定。X-TEA資料の具体Function名とSmartDB機能分類は原典・ライセンス確認後に対応付け、推測で正式Action mappingを採用しない。
- **M1 Headless read-only**：#2506のContext×#2507 Patternから、SA-01/03/04/05/06のAction候補を一つの合成Fixtureで説明し、#2500の既存Intent Referenceと差分確認。#2507 Draft #2613のAG局所20件PASSを本SAの実証へ転記しない。
- **M2 複数Consumer**：同じAction意味をCUIとTable/Graphの最低2 Consumerで比較。見えない/未実装を命令不可・意味不存在・権限不足へ自動転換しない。Script-firstの標準Studio UIを別のread-only比較。
- **M3 実Request**：#2598の候補→実Payload/Target/Provider経路と、#1174のAuthority/Tx/Receipt・#1163の独立readへ接続。M1/M2だけで実業務保存が成立したとしない。
- **M4 汎用性評価**：業務Dataだけでなく非業務の独立Data/Relationにも、追加専用画面なしでActionを発見できるか。専門UI/Adapterはその追加価値が検証されたときの任意選択。TEI Product Skill（#2605）とTAI Skill（TAI #168）は実在Toolだけを案内し、概念Actionを実コマンドと偽らない。

**未決**：SmartDB移植の具体語彙、正式Operation IDとDomain Pluginの対応、固定対象/遅延述語対象のAction契約、要求Collection completeness、適用状態の診断の公開形式、Script/DSL公開Syntax、UI共通Presenterの最小メタ情報、Ownerの業務Action意味。既存#1119 P0/G0〜G4と実DB・Host縦切りの優先順位は変更しない。実Go/SQL/GUI/Provider/外部Host/Owner受理は今回NOT RUN。

<!-- 20261102-crosslane-everything-data-action-plan -->
## 6. 11/2横断統合：Everything is DataからSemantic Actionを構成する（設計候補）

この節は#2619/#2620/#2621/#2622のデータ中心設計、#988/#2617のScript-first標準UI、#2507 Draft #2613のA/G適合、#2597 Draft #2601のEditing Session、#2598のRequest束縛、#2502 Draft #2602のMacro委任を**参照**し、Action発見に必要な接合点だけを記録する。各Draftの内容を製品の採用済み契約や実動証拠として扱わず、各Ownerのファイルを変更しない。

### 6.1 静的な定義と利用時の文脈を混合しない

1. **Authoritative Definition**：Function、Action、Rule、Provider適合要件、Presentation Bindingは、Roleと安定参照を持つ標準データ構造として記述する。Role/Operatorを解釈する既存PluginのID・版を参照する。Action目的とEffectsは、表示要素や物理SQL名から推測しない。
2. **Dynamic Context Evidence**：Function InstanceとSelection epoch、Entity/Relation/Path occurrence/Groupのgrain、Result Sliceと権限付きscope、Read revision/time、Editing SessionのPatch/dirty、QueryのLoss/Unknownは**利用時の入力**であり、Definitionの一部として正本化しない。
3. **Derived Action Offer**：上記1と2の型付き参照を用いて、候補・不適合理由・不足情報を用途別に投影する**再計算可能な結果**。定義・現在の実行許可・持続的なBusiness Request・OperationIDのいずれでもない。古いOfferにTTLだけを付けても権限の再確認を省略できない。
4. **Interaction/Editing Intent**：UI変更やMacro呼出しから生じる候補は#2500/#2597が解釈し、#2598が現在の対象・時点・根拠を実Requestへ束縛する。**Editor上の一つの保存ボタン＝一つのDB transaction**とはしない。
5. **Authority/Effect**：外部Host・Providerの現在Grantと実対象/副作用を#1174/#1129で検証。受理されたBusiness Actionの存在と現在Grantの存在は別。Action Offerを公開する権利も#2516/#1178で独立に扱う。

比較用表記：OfferCandidate = Project(Applicability(Definition, ContextEvidence), ConsumerProfile, DisclosureScope)。**API/型名の採用ではない。** 意味的な構造適合、対象解決の確実性、Provider実装可否、現在Grant、結果観測は異なる述語とし、一つのbooleanにしない。

### 6.2 複数の構文は一つの意味解釈へ向ける

Markdown表、Graphviz記述、関数式、メソッドチェーン、簡潔なUI制御式などは、#2620/#2621で検討している**標準データ構造へのsyntax projection/sugar**として比較する。特定の構文をCore実行権限や意味正本へ直接昇格させない。

- 正規化前後に必要：Stable Semantic ID、型/Role、Operator/Plugin版、参照と依存、入力・出力のgrain、null/presence/quantity/time、effects、Source location/spanと解釈の根拠。見た目が一致するだけで意味一致とはしない。
- **二つの表記が等価となる条件**は、単なる同形JSONではなく、同じ受理済みMeaning/Plugin/Source条件の下で、各Claimが必要とする区別を保持していること（#2497/#2516）。未解決Role・曖昧Binding・循環・順序依存を黙って最初の候補へ解決しない。
- TEI StudioはText、Property、Table、Tree、Graph、Diffの**標準Projection**を使用し、Role Pluginが補完・制約・診断を追加。専用Node/Function-type Editorは強制しない。往復でSource byte/spanの一意性がない表示はread-onlyにし、編集候補は既存Source Acceptanceへ接続する（#1124/#2617）。
- UI visible/enabledを短いDSLで記述可能とするが、表示評価はread-onlyのExpression、権限はHost、EffectはOperationへ分離。非表示でもAction定義が存在し得るし、enabled=trueでも実行許可にはならない。
- Data-attached Ruleは#2622 Data Operationsの適用先と#1129のphaseを介して接続。Rule Definition/Attachmentの意味をUI要素やForm用Scriptへ二重定義しない。

### 6.3 Actionは画面へ配置するだけでなく、複合・Macro・外部呼出しにも使う

同じSemantic Action/Operator参照を（A）Table/Form/Menu/Graph等のUI Affordance、（B）CUI/AI/外部Hostの明示呼出し、（C）Script/Macroが構成する高次Actionに投影する。ただし複合ActionのA/G・部分成功・Commit後取消不能・再試行安全性は#2507/#2597/#1174が検査し、UI複数配置による意味の同一性だけで**複合処理のatomicity/業務完了**は主張しない。

委任Macroの権限範囲は#2502/#2499の既存出版/委任契約と現行Hostに委ねる。対象意図Order Aと実Payload/実Effect Order Bの不一致、非公開列を省略した限定PATCHと全置換の違い、同じ定義の二Instanceにおけるdirty/epochの独立性は、既存#2598/#2597の負例を再利用する。

## 7. 追加の横断反証と最小受渡し（旧SA-01〜12の再実装をしない）

| ID | 新たな接合点 | 誤りとして検出するもの | 正本Owner / 最小観測 |
| --- | --- | --- | --- |
| XA-01 | Markdown表 / 式 / メソッドチェーン → Action Definition Role | 構文一致だけで同一意味、丸め/NULL/Effect欠落 | #2620/#2621/#988：意味参照・Source差分のread-only比較 |
| XA-02 | Script visible/enabled → 標準Studio UIとCUI候補 | enabled=true→実権限付与、非表示→Action消滅 | #2617/#997/#2507：同一Definition・別Consumerでの分離 |
| XA-03 | Action Definition共通 / Context Instance I1・I2独立 | 親Aの旧結果・dirtyを親Bへ転写 | #2597/#2506：anchor、epoch、Patch別の診断 |
| XA-04 | Data Rule + Operator + Provider別実現 | DB由来Ruleが実装先を変えると順序・Effectが変わる | #1129/#2622：phaseとCapabilityの適合差、actual Effectは別 |
| XA-05 | Macro → Action候補 → 実Request | 候補の公開と現行Grant、意図した対象と実Payload対象の取り違え | #2502/#2598/#1174：Request/Grant/Effectの別証拠 |
| XA-06 | Composite ActionとSave Intent | 同一Button・複数Patchを必ず一Txと推定 | #2507/#2597：A/G・部分/unknown・独立Receipt |
| XA-07 | 非RDB/業務外PluginとCUI | 全DomainにCRUDを自動付与、Provider非対応を意味不在と誤認 | #2493/#2622：Append/Compute等の明示Contractの比較 |
| XA-08 | Action候補公開の情報境界 | 非公開Actionの存在/件数や拒否理由を低権限へ開示 | #2516/#1178：公開許可済み投影のみ比較 |

### 同一試験の接合段階と測定対象

- **A0: Definition比較** — 現行#2620/#2621と#988のSyntax→標準Structure→Role参照で、Actionの必要条件・Source/Plugin版が残るかを観測（実正式DSL parseは未達）。
- **A1: Headless Applicability** — #2506/#2586の固定H/D1/D2と#2507 Draft #2613の既存A/G・SA局所試験を基線とし、同一Definition/Contextを消費した結果の差分を測る。局所42件PASSを実TEI縦断PASSと数えない。
- **A2: Multi-consumer** — CUI/JSONとTable/Formまたは標準Studioを同じMeaning/epoch/Scopeで比較。UI固有Action Rule数、Bindingに要した追加設定/コード量、unsupported理由と開示差を記録する。
- **A3: Request-bound Integration** — #2500/#2597/#2598のViewChange→Intent→Target bindingを既存#2485 Operation Surfaceへ渡し、Requestに使われた対象・版・Providerが元の用途と一致するかを検証。見かけのUI操作成功を保存成功としない。
- **A4: Effect/Actual Consumer** — #1174/#1163/#1119の現行P0・既存Scenarioに従い、実Provider/DB、現在Grant、Receipt、独立readを同一試行で確認。外部HostとMacro公開は別資格。新Gateを設けない。

**評価指標**（#2018へ）：定義1件からの異種Consumer数、Source/Role/Actionの意味的重複、必要なCore特殊分岐、独立Adapterコード/設定量、実Plugin互換差、誤適用/過剰拒否、秘密開示、request対象逸脱、再確認時間、AIに渡した実文脈量。試験の完了と実Target証拠を別台帳にしない（既存Evidence/Scenario参照）。

**停止条件**：業務Action identityを構文/画面名から推測、Context部分集合を全件扱い、Macroの効果を隠蔽、単一OfferにHost Grantを内包、Providerのatomicityを推測、既存正本を複製、未確認の実DB/Business受理をPASS化した場合は対象の採用判断を保留する。

この節は横断**設計・実証受渡しの計画のみ**。#2619/#2620/#2621/#2622、#2613/#2617/#2602/#2601は独立Draft/Issueであり、統合・採用・実TEI/GUI/DB試験・Owner受理は今回の成果ではない。

<!-- crosslane-three-footprints-and-endtoend-packet-20261102 -->
## 8. 11/2横断追補：Action発見を支える三つのデータ境界と同一試行の証拠（設計・NOT RUN）

[#2624](https://github.com/hat47x/tei-platform/pull/2624)のFunction別明示Mappingは局所合成20件、[#2625](https://github.com/hat47x/tei-platform/pull/2625)のSyntax Sugar正規化は局所合成23件、[#2613](https://github.com/hat47x/tei-platform/pull/2613)のFunction Pattern/Action適用は局所合成42件PASSの報告がある。各Draftは別の入力・模擬解釈・実行環境を用いており、その合計を「Sourceから実Actionを実行した」証拠とはできない。成熟した次の課題は、既存のSource/Context/Operation/Providerを**同じ固定版と実際のConsumer入力で接合すること**である。新しい評価器や第二Scenario保存機構は作らない。

### 8.1 三つの範囲を区別する

| 範囲 | 正確に確認するもの | 不適切な推論 | 主Owner |
| --- | --- | --- | --- |
| **Declared/Required Read Footprint** | Function Data Contextで明示された対象Relation path/Field/Scopeと、Actionが成立するために必要とする入力・完全性 | Data ModelにRelationがあるから全件自動取得／未取得前提を不存在扱い | #2506/#2624、#2507、#2622 |
| **Actual Acquisition Footprint** | Providerが実際に発行したQuery/API Request・読取Field/Relation・実DB Alias・Principal/RLS/search_path・版/時点・Snapshot保証 | 出力JSONに秘密Fieldがないから実Providerも読んでいないと主張 | #2622/#2387/#2611/#951 |
| **Consumer Disclosure Footprint** | CUI/Studio/SHO/外部AIへ実際に届く値・Action名/件数・診断・結果・それぞれの開示資格 | read-onlyなら外部送信可／候補数や失敗理由から秘密が漏れないと推測 | #2516/#1178/#2515、#997 |

**適用判定への含意**：Actionが要求するReadが明示Mappingで満たされなければ、当該Claimの材料不足として扱う。追加情報の取得が必要なら既存の許可済みResolver/Providerへ要求する設計候補を返せるが、Action Applicability自身が読取権限を発行することはない。ProviderのField Projection未対応で過剰な物理読取が生じた場合、後段で表示を消しても物理取得はなかったことにならない。対象のData Minimization条件に合うか既存Provider/Hostの受入へ返す。出力が同値でもStatus/Timing/Log/反復API呼出し/実LLM Egressの安全性は別実測（#2515）。

### 8.2 画面に見えないがActionが必要とする情報

L1の単一Field PATCHはL1 ID、対象属性、Version等で意味的候補化できる場合がある。一方、注文Submitが未MappingのPayment状態を必要とするなら、非表示・未取得を「未払い」や「確認済み」と解釈せず、当該必要条件は未確立とする。信頼境界内のOwner Serviceがその条件を検査する契約が受理されているなら、秘密の生値をConsumerへ出す代わりに、許可された検査結果と保証範囲を利用する候補を比較する。Action名の存在自体を公開できるかも独立（#2613 SA-21/22、#2516）。どの場合も実行時の現在Grant・対象・版はHostで再確認する。

**Provider能力の非同値性**：Field PATCHのみを求めるActionと、ProviderのFull Replaceしかない場合は同値ではない。自動read-modify-writeは非表示Field消失、余計なRead/Write権限、競合、Snapshot不整合を起こし得る。Ownerの受理、必要な完全Image、Version Fenceなどの保証を確認できなければ安全な代替能力は未確立（#2622 DO-04/05、#2598 IB-03）。同一Semantic IDに複数DB Aliasがあれば、Action OfferやUI選択だけで書込先を決定しない（#2611）。

### 8.3 単一の統合試験へ引き渡す観測表

これは新しい公開Schemaではなく、既存#1131/#1180/#1163 Scenario/Evidenceに対応する**検収時の列**である。基線は#2586のH・D1(L1,L2)・D2(3件)。#2624からPayment/secret Fieldを追加するときは「新しい合成拡張」と明記し、独立DraftのFixtureを同じAccepted Sourceと見なさない。

| 段階 | 実際に同一試行で観測するもの | 主な負例と担当 |
| --- | --- | --- |
| **I0 Source/Role** | Source bytes/occurrence・実Language/Role/Operator/Plugin exact版・Resolved Meaning/Acceptance | 説明用表の実行誤昇格・同名Ref誤束縛。#2620/#2621/#988/#1124 |
| **I1 Context** | 実Consumerが受けた明示Mapping・D1/D2それぞれのgrain/Scope/epoch・Result Sliceの実完全性 | 未MappingのPayment暗黙Read、D2部分1/3を全件に昇格。#2506/#2624/#2586 |
| **I2 Provider Read** | 実Query/APIと物理Source/alias/Principal、取得Field/Relation・実Provider読取証拠 | JSONで秘密を削除しても、Providerが秘密を過剰取得する。#2622/#2611/#2387 |
| **I3 Offer/Consumer** | 受理済Action purpose/必要条件、公開した候補IDと理由・Consumer Scope | Payment未取得でSubmit可能、非公開Actionの存在・件数漏洩。#2507/#2516/#997 |
| **I4 Request** | L1のField PATCH、Context/epoch、Action、Payload/実Target/Provider Binding | L1→L2漂移、PATCH→Replace、JOIN 3出現を3更新に変換。#2500/#2597/#2598/#2485 |
| **I5 Effect** | 実行時Grant・OperationID/Receipt、実更新対象・独立Invocation read・Business Result | ACK不明を失敗確定、2Provider部分成功を全成功とする。#1174/#1163/#1119 |

**実証順**：I0/I1 read-only → I2 actual Provider read（許可された隔離環境）→ I3 異質な二Consumer → I4 実Request束縛 → I5 Host/Provider Effect。最低限「L1 PATCH候補」「未取得Paymentに依存するSubmitの未確立」「複数Provider aliasの曖昧性拒否」「非公開Actionの開示差分なし」「partialから全件操作を不正導出しない」を独立Expectedとする。I0〜I5の証拠辺が同じ試行で結べなければ、各局所PASSがあっても横断結論は not established。

**計測**（#2018/#2493）：論理Read依存・実Provider取得Field/Request・Consumer公開Field、不要Read数、Adapter/専用UI/特別Core分岐数、再利用した既存Contract数、誤候補と過剰拒否、実対象逸脱、同版でつながったEvidence辺の数。公開Evidenceに秘密値や接続文字列を複製しない。

**状態**：今回の§8は計画の追補のみ。I0〜I5の実TEI結合、実Provider読取・外部送信・Go/DB/GUI試験とOwner受理はNOT RUN。局所Python PASSを製品資格へ昇格させない。新Canonical/Host/Registry/権限Engine/Grammar/新Gateは追加しない。
