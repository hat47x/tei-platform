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

<!-- action-offer-claim-support-and-invalidation-20261102 -->
## 9. Action Offerを「用途相対のClaim」として評価する：出力依存・行成立依存・再資格化（2026-11-02／設計候補）

### 9.1 他レーンから判明した不足と今回の位置づけ

[#2624](https://github.com/hat47x/tei-platform/pull/2624)のDisplay Mapping／Rule Inputの局所合成38件（20+18）、[#2387](https://github.com/hat47x/tei-platform/issues/2387)のQuery非出力依存、[#2634](https://github.com/hat47x/tei-platform/pull/2634)の合成Analyzer→実Go DerivedIR特性、[#2582](https://github.com/hat47x/tei-platform/issues/2582)の変更失効、[#2640](https://github.com/hat47x/tei-platform/pull/2640)の「0件と負の依存」、[#2496](https://github.com/hat47x/tei-platform/pull/2496)の構造保存・依存閉包を参照し、**Action Offerが同じ表示値に対して同じ利用資格を維持してよいか**を、別の観点として#2507へ引き渡す。

例：Tableに L1(id,amount=100) が同じように表示される二世界でも、見えていないWHERE/EXISTS条件、与信Rule、Sort/Top-N、Provider read cohort、利用主体・版が違えば、Submit、編集、一括操作の候補が同じとは限らない。逆に無関係D2に変更があっても、D1だけに依存するL1表示・read-only説明を無条件に失効させる必要はない。**描画値一致／同じOutput Field Lineage／Query ID一致は、Action適用条件の成立根拠の一致を意味しない。**

従来の§8は「宣言Read・実取得・公開」の三境界を示した。本節は、その内部で**各Action/Claimを成立させる依存の種類と、変更後も成立する証拠**を整理する。SQL構造抽出は#2387、Data/Rule/Providerは#2506/#2622/#1129、増分失効は#2582、実Request/Authorityは#2598/#1174、独立Oracleは#2514のOwnerに残す。第二Dependency Engine、Negative Cache Engine、Policy/Host/Registryは作らない。

### 9.2 Action Offerが依存する七つの異質な根拠（固定Core schemaではない）

| 用途別の根拠 | 何を判定するか | 典型的な見落とし／委譲先 |
| --- | --- | --- |
| **Value/Type** | 出力値・Presence、数量/金額の単位・演算法則、丸め時点、型/版 | 出力値の一致だけで丸め・入力欠落・非表示依存の一致とする。#1705/#2593/#988 |
| **Membership/Predicate** | 行が存在する条件、JOIN/EXISTS/WHERE/HAVING、Relation occurrence、対象絞り込み | SELECT列のLineageだけを見る／行が0なら依存は空だとする。#2387/#1706 |
| **Order/Window/Quantifier** | 並び順、LIMIT/Top-N、集計・Not Existsの全件条件、ページ/時間窓と負の依存 | Orderを変えても表示項目が同じだからClaimは安定／partialを全件にする。#2387/#2506/#2593 |
| **Rule/Action Precondition** | Validation、業務状態、関連データ、Owner Serviceでの限定的な条件判定 | Display MapにないAccount/Payment依存を無視してSubmit可能とする。#1129/#2624/#2507 |
| **Provider/Time/Target** | 検査と実行に使った実際のProvider/DB alias、RLS/Principal、Snapshot/Query/Operator profile、現在の時点 | 同じSQL/意味ID/alias文字列や同名Snapshotから同じ母集団と断言。#2611/#2622/#1708 |
| **Change/Evidence Validity** | 取得・選択・Rule/Source/Provider改版後、今も根拠が利用できるか／無影響を言えるか | 結果がA→B→Aと戻っただけで途中変更なしと主張、通知がないから不変とする。#2582/#1641/#2505 |
| **Authority/Disclosure/Effect** | OfferをこのConsumerへ公開してよいか、実行時の許可・対象一致・実Effectを何で確認するか | disabled=falseをHost Grantと同一視、非公開Action件数や失敗理由を漏らす。#2516/#2598/#1174 |

この七分類は**受入票の質問集合**であり、全Actionが一律に七種の全入力を必要とするわけではない。ActionごとのPurpose、要求量化（1件／全件／存在／不存在）、Read Set/Effect Footprint、指定Owner Serviceの契約から**必要なものだけ**を選ぶ。未知のQuery/Provider依存は推測で補完せず、当該用途について未確立とする。Value lineageが空でも、0件・不存在ClaimにはPredicate/Read Scope/完全集合性の「負の依存」が残る。

### 9.3 Offerの提示・再利用・実行は別の資格

- **定義上の候補**：受理済みAction/Patternが指定Semantic/Shapeに適合するか。実際に値を読めたか、現在Grantがあるかとは独立する。
- **現在の証拠付き適用候補**：必要なPredicate/Rule/Collection・Version/Snapshot/Selection条件が、実際にその時点で満たされると主張できる範囲に限定する。証拠不足は未確立、反証がある場合は不適合/失効。過度な一律全面失効と誤った再利用を両方評価する。
- **開示可能なOffer**：Consumerごとの公開許可を別に検査。拒否理由/候補数/所要時間も観測可能な情報。信頼済みreleased actionsや非公開情報境界のOwnerを参照し、新公開用Policyを作らない。
- **実行要求**：Offerをキャッシュして再表示しても、対象・作用・Version Fence・現在Grant・Owner Serviceの必要条件は実行地点で再照合する。OfferのepochやTTLはOperation/Receipt/現在Policyの代替にならない。
- **歴史説明の保存**：旧版で何を確認できたかは監査用に残せるが、「当時妥当だった」から「いま操作可能」への再利用は別Claimとする。現在表示・履歴証拠・編集準備を一個のstaleフラグで同時に制御しない。

用途相対の依存閉包を用いる際は、**値を作る依存**、**その結果が選ばれる／存在する依存**、**判断に必要な業務Rule**、**実行先の束縛・権限・観測**を区別する。前二者はSQL/Operatorの構造から参照できても、現行DerivedIRのValue Lineageだけで「すべて取得済み」と証明しない（#2634）。実Providerの読取りやBusiness Acceptanceまで構造解析から推測しない。

### 9.4 既存Fixtureに接続する新しい反例（OA-01〜12、当節としてNOT RUN）

| ケース | 入力・変化 | 判別したい適用判断／証拠 |
| --- | --- | --- |
| **OA-01 非表示WHERE** | L1表示(amount=100)は同じ、hidden credit_scoreの条件閾値または実値だけ変化 | 表示値Lineageが不変でも、行が選択された理由・Actionの適用Claimを再確認。#2387/#2634 |
| **OA-02 空集合の負依存** | t0配送=0→t1でP1追加→t2削除して再び0（ABA） | t0時点の0件、現在の0件、期間中に変更なしは別。0件だから依存空という最適化を拒否。#2582/#1641 |
| **OA-03 TOP-N・演算順** | Filter→Sort→LIMIT 1 と LIMIT 1→局所Filter、非表示rank変更 | 同じ演算名であっても部分結果/実行順でClaimが変化。Provider pushdownとローカル処理の同義性を限定比較。#2387/#2622 |
| **OA-04 Rule入力と表示** | 表示Order.amount=600同値、Account.credit_limitが700と500 | Display Footprintだけの一致からSubmit候補同値を導かない。必要Rule入力・Owner Serviceの検査と結果開示は別。#2624/#1129/#2516 |
| **OA-05 依存局所維持** | HにD1=2件（全件）、D2=3件（部分）。D2のみ更新 | D1-only表示・合計は依存独立が示せれば維持、D2/Join依存Claimは再確認。全Context一律staleを避ける。#2582 |
| **OA-06 存在／不存在の非単調性** | NOT EXISTS、LEFT JOIN未一致、追加の右側一致、遅着Event | 追加で旧結果のretract/replaceが必要。insert-onlyの表示更新を不適用と診断。#2387/#2582 |
| **OA-07 多Providerの時点** | D1はPrimary snapshot s1、D2はReplica snapshot s1という同名ラベル | 同じ名称で全体の一貫したRead Cohortを推測しない。Commit前のOwner条件に必要な一致証拠がないなら保留。#2622/#1708 |
| **OA-08 Operator意味Profile差** | 同じ表示数値でもSQL/JavaのNULL、Collation、十進丸め時点・中間精度が異なる | Operator/Targetの実意味契約が適合する範囲だけRule/Measure/Action候補を維持。#2387 QX-28/29、#988/#2077 |
| **OA-09 部分行と部分Field** | Row setはcompleteでもprivateCost fieldがabsent、別Responseではnull | 全行取得の主張から必要属性の観測完全性を導かず、未取得・NULL・欠落を区別。#2506/#2627 |
| **OA-10 旧Offerから実Requestへ** | AでOffer生成→B選択/Source版変更→古いA候補をMacro/GUIが再送 | #2598の実対象/版/Authority照合で拒否または明示再解決。新Operationで無確認再試行しない。#2597/#1174 |
| **OA-11 開示と判定差** | 非公開Payment/Action有無が違う二世界、公開すべき項目は同じ | 公開不可Consumerの候補ID・個数・診断差の漏えいを防ぐ。有限JSON比較はOS/Timing/Egress非干渉の証明でない。#2516/#2515 |
| **OA-12 独立Oracleと継続資格** | Source/Operator/Rule改版、旧Expectedと新実行で偶然同値 | 値一致だけで旧Oracleの独立性や現行適用を認定せず、Source→Expected→Actualの由来と資格を別確認。#2514/#1163 |

**受入順序**：既存#2634の実Go DerivedIRには合成Analyzer入力、#2624は独立PythonのDisplay/Rule入力がある。この二つを同じRead Contextに「つなげられるか」を最初の静的・read-only Gapとして測定する。次に#2582の依存失効Refと#2507のA/G/Action Offerを二Consumerへ接続し、最後に#2598/#1174の実Request・Effectへ渡す。新しい並列OracleのPASSを一つの結合PASSとして数えない。

**具体的な比較票**：Action purpose/受理版、Function Definition+Instance/epoch、Declared Required Read、Query operation・Predicate/Sort/anti-join/Rule依存、実Provider alias・profile/Scope/観測時点、Row/Field completeness、対象Identity/Grain、Owner前提、公開Consumer、Offerの診断、変更/失効理由、実Request target/grant/Receipt、独立Expected/Actual、必要な再確認箇所。既存#1131/#1163/各Ownerの情報を参照する**比較列**であり、共通ABI/Signature/Claim Certificateの新形式ではない。

**測定指標**（#2018）：誤ったOffer再利用件数、必要なOfferの過剰失効件数、再取得回数・時間、Consumerごとの差分、検査できた負依存の割合、同一試行でSource/Provider/Consumer/実Requestが接続した辺の数、Core特殊分岐・Adapter/専用UI量、再利用された既存Validator/Contract数。

**停止線・未決**：Accepted Actionの意味、どのClaimに完全な依存閉包が必要か、Owner側条件検査の公開レベル、Provider Update/Snapshot保証、Operatorプロファイル適合は個別受理待ち。提示の正しさ／開示／現在Grant／実行効果を同一booleanへ縮約しない。新しいReactive Engine/Negative Cache/Claim Solver/Policy/Host/Registryを増設せず、実TEI/DB/GUI/外部AI、Go縦断、Owner意味受理、OA-01〜12の試験は当節としてNOT RUN。

<!-- 20261103-derived-operation-reality-bridge -->
## 10. 実在するOperation SurfaceからSemantic Action/実Requestへ：製品コード基線の接合（2026-11-03／設計・NOT RUN）

### 10.1 この時点で静的に確認できた既存製品基線

[merged PR #2587](https://github.com/hat47x/tei-platform/pull/2587)は標準操作と成果物導出およびCLI入口をmainへ追加済み。[merged PR #2624](https://github.com/hat47x/tei-platform/pull/2624)の明示Function Mapping/Rule Footprint、[merged PR #2634](https://github.com/hat47x/tei-platform/pull/2634)のGo Query依存特性、[merged PR #2640](https://github.com/hat47x/tei-platform/pull/2640)のContract Query結果意味は、いずれも**統合された個別成果**として区別する。これらのmergeは本書§8 I0〜I5や§9 OA-01〜12を同一Source/Consumer/Hostで実証したものではない。

現行mainのRead-onlyなソース照合から、次が実在する。
- cmd/tei/main.go のdispatchに operations と artifacts が存在。cmd/tei/cli_operations_artifacts.go の operations は workspace からSurfaceWorkspaceを導出してJSON出力する。artifacts は application 指定を要し、出力先を指定しない場合は成果物集合をJSONで出力する。
- internal/operations/surface.go は tei.operations.operation-surface/v0 の derived、target-neutral、read-onlyなOperationSurfaceを提供する。worldFingerprint、modelRevision、rule ID/version/fingerprint、operations、gaps、conflicts、authorityLimitsを持つ。Operationにはkind、target、origin、inputs、preconditions、effects、concurrency、binding、autonomy、basisを持つ。ただし**Surface自身のeffect=read-onlyと、列挙されたOperationの作用（create/update等）は別**。autonomyやbindingを現在のRuntime Grantとはみなさない。
- cmd/tei/cli_contract.go の commands 一覧は selected-read-only-commands の限定列挙で、operations/artifactsが載っていない。「コマンド一覧にない」≠「実mainに実装なし」≠「Skill/Hostで実行が許可済み」。逆に、dispatch分岐やOperation IDが存在しても、対応Action/Capabilityが現在利用できることは意味しない。
- artifacts の -out は MkdirAll/WriteFile による実ファイル変更。出力ディレクトリが既にある場合の旧ファイル残存や中途失敗を、この関数だけで原子的公開済みとみなさない。Manifest/Findings・実ファイル・実Consumerがロードした版の検査は既存#2604/#2633に委譲する。
- cmd/tei/main.go には共通loadWorldの後にoperations/artifacts分岐があり、各処理内でWorkspaceを別途Loadするコード経路がある。これは**ソース上の重複Loadの観測**であって、競合・TOCTOU障害が実発生したとの断定ではない。独立CLI呼出しを束ねるときに、一つのWorld/Source版を共有したと自動推論してはならない。受渡し用には実消費Source・版・Source Index fingerprintの一致を再取得して判定する（#1163/#2629/#267）。

### 10.2 三つの異なる発見・束縛の問題

| 段階 | 既存の物証と問い | 本書で追加する判断 | 正本Owner |
| --- | --- | --- | --- |
| **D0 Derived Surface** | 受理済みWorldから標準list/read/create/updateとgap/conflictを決定的に導出できたか | Surfaceが表示できること≠Actionが現在適合・実行可能。Rule fingerprint/Source versionを固定し、利用できなかった理由も保持 | #1127/#2485、実main internal/operations |
| **D1 Context-aware Action** | 特定Function/InstanceのSemantic Subject、Role、Predicate/Rule、対象Scope、Result Slice、取得版、欠損に適合するか | Sourceへの同形Operationを別Contextへ配置しても、意味的Preconditionの充足・現在性は独立。OA-01〜12を再利用し、第二標準Operation Engineを作らない | #2507/#2506/#2582 |
| **D2 Realization/Consumer** | Capability Binding、Target/Provider、Action/Skill/UI/CUIの露出、実Request・Effectの証拠はあるか | Derivedのbinding候補やCUI tool記述 ≠ 実Invokerが受信したRequest ≠ 現在Grant ≠ Effect/独立read。成果物出力/外部公開も別資格 | #951/#2598/#1174/#2516/#2605/#2649 |

特にD0の標準create/update候補から、Submit/Cancel/Approve等の業務Action目的を**名前やデータShapeだけで補完しない**。特殊なDomain（Document/Event/Simulation/Robot）への拡張は、標準CRUDの成功ではなく#2507の要求Shape/Effect、#2622のProvider実現、#2493の別Pressureで判断する。

### 10.3 最小の実在製品縦切り（試験票、全件NOT RUN）

| ID | 同じ固定Sourceで比較する内容 | 誤成功の停止線 |
| --- | --- | --- |
| **OB-01 CLI入口と限定Discovery** | 実CLI commands の限定一覧、operations/artifacts の実dispatch・引数・JSON、CLI版を照合 | commandsの不在を実装不在と判定／dispatch存在を実行許可と判定 |
| **OB-02 Surfaceの正負と基線** | reference/standard-crud等の既存合成WorkspaceでSurface.operations/gaps/conflicts/ruleFP/modelRevisionを独立期待と比較 | gapsを成功したActionとして表示／Surfaceのeffect=read-onlyを中のwrite Operationに伝染 |
| **OB-03 対象同一性とAction前提** | Surfaceの標準Operation target/basis/bindingと、#2586 H/D1/D2のContext/Rule/選択epochを**同一Sourceに再構成できる場合だけ**照合 | 同名targetや同じBinding文字列から実Business Action/Owner受理を創作 |
| **OB-04 Source改版と二重Load** | CLI実呼出しごとにSource Index/World/Model revision、Surface rule fingerprint、Artifact inputRevisionを比較。隔離Fixtureで読取中の入力改版も圧力候補 | 二回のLoad・別CLI実行を一Snapshotと推定。改版途中の実障害は未観測 |
| **OB-05 Artifact副作用と整合** | artifacts のstdout/出力先指定を別試行とし、manifest/findings、実ディレクトリのMember/版・旧ファイル残存/失敗を記録 | stdoutのみのRead資格を -out に転用／manifestのみで原子的公開・Consumer実ロードを証明 |
| **OB-06 Skill/External Discovery** | #2649の実CLI/Skill受渡しとD0候補の両方を比較。Action候補のDisclosure、Tool存在、Invocation入力、Host Grantを別記 | Skill load/Tool宣言を意味受理・現在Authority・Operation実行済みに昇格 |
| **OB-07 Bound capability** | #2485合成Driverの実受信target/input/BusinessResultとD0/ D1候補を同一試行で突合。Field PATCHとReplace、欠損/NULL、Provider版を負例とする | Operation ID/Binding一致のみで対象・業務Effect一致を宣言 |
| **OB-08 Actual Provider/Receipt** | #1119 P0-a/P0-bと#1163で、実Host認可/Transaction/Receipt/別invocation独立Readを同一Source/対象/時点へ結ぶ | isolated driver PASSやCI mergeを実DB・Owner業務受理に変換 |

**着手順**：OB-01/02を、merged #2587の実main CLIと独立Expectedで先行させる。OB-03は必要なFunction Context/Ruleが同一Workspaceに実在するかを先に調べ、不在なら接続Gapとして報告する。OB-04/05はSource/Artifactの実消費物と副作用の比較、OB-06は#2649 Product Skill担当へ、OB-07/08は既存Driver・Host担当へ渡す。異なるDraftのテストケースをひとつの製品E2Eへ組み替えたと装わない。

**優先順位・受入停止線**：新Operation/CLI ABI、Domain Action enum、Skill実行権限、全コマンド自動発見、共通Runtime/Host/Registry/Tx、原子的Artifact Publisherをこの文書で新設しない。実mainのRead-onlyコード所見と既存PRのmerge状況は確認済みだが、OB-01〜08を実際のCLI/Go/DB/GUI/外部Harnessで実行したものではない。Ownerの意味受理、実Provider効果、#1119 G0〜G4も未取得の範囲を保持する。

<!-- 20261103-transitive-invocation-effects-and-skill-evaluation -->
## 11. Semantic Action Offerから実CLI/Skill Invocationへ：合成実行の作用と、効果測定の資格（2026-11-03）

### 11.1 操作の「意味的な作用」と、実呼出し経路の「観測される作用」を分ける

直前の§10で、実mainのOperation Surfaceはread-onlyな**導出結果**であり、中に列挙されたcreate/update等の作用と別だと確認した。さらに、[#2607 Product Skill Draft](https://github.com/hat47x/tei-platform/pull/2607)のS0cでは、Markdownに現れる最初の tei validate 等のread-only例だけを静的判別すると、後続の shell 連結や pipe/tee による作用を見落とし得ることが具体化された。既存Verifierに TS-16〜19 を追加したが、**6入力の限定分類が報告された段階であり、更新後の11テスト全件、実CLI・外部Harness・OS書込みは独立資格**である。

同じ「read-only」という語でも、少なくとも次は独立に確認する。
1. **意味的Action Effect**：Accepted Operation/Service ContractのPurpose、対象、Read/Write/Append/Compute、Rule Phase、Business Result。#1127/#2507/#1129が意味の適合を所有し、OperationSurfaceの effect=read-only は**候補を説明するコマンド自体**の属性。
2. **Tool Invocation Footprint**：実CLI binary、argv全体、環境設定、shell wrapper/chain/redirect/pipe、参照ScriptやTool側の再呼出し、対象Workspace/Path、実File/Process/Networkの効果。Tool説明・Markdownコードフェンス・静的Verifierの合格だけでは物理的副作用なしを保証しない。#2605/#2607/#2649、外部Harnessの実Tool観測はTAI #270/#271、OS情報境界は#2515。
3. **Consumer Observation/Disclosure**：CLI stdout、stderr、status、Tool候補一覧、実HTTP受信、ログ/Diagnostic、SHO/AIへ渡した値と用途別許可。最終回答が同じでも観測面は同じとは限らない。#2516/#1178/#2515。
4. **Business Completion**：実Owner Service/Providerが実際に処理したRequest、対象Identity、Commit/Receipt、別invocation独立Read、Owner意味受理。#2598/#1174/#1163。CLI成功、Artifact生成、Skillの実Loadをこの資格へ自動昇格させない。

**承認判断は実際に実行する連鎖に相対的**であり、単一のコマンド名またはOperation IDに静的な「安全」フラグを付けて終わらない。たとえば初回Read-only Product Skill Profileでは tei commands/model catalog/inspect/trace/validate -json を限定候補として扱う一方、tei artifacts -out や tee/redirect 等のファイル書込み、外部送出、複合shellはそのProfileへ黙って追加しない。tei operations はRead-onlyなSurface表示を実装するが、selected-read-only discoveryで列挙されないため、存在と**今回のTool Profileに含めること**を別の判断とする。Sandbox/Hostの実write・network権限は既存実行環境が制御し、TEIに第二Sandbox/Authority Engineを追加しない。

**開示の負例**：機密Actionの存在をButtonに出さなくても、Tool discoveryの数、Plan、diagnosticsや拒否理由が異なるだけで秘匿状態が推測できる可能性がある。ただし合成JSONの同値や静的shell文字列検査を、Timing/OS/Networkまで含めた非干渉証明とは呼ばない。

### 11.2 実証を増やすより、最初の単一Taskを固定して比較する

直近の確定基線は[merged TEI #2626](https://github.com/hat47x/tei-platform/pull/2626)の実Go CLI（[Focused CI #41462654810](https://github.com/hat47x/tei-platform/actions/runs/41462654810)）である。公開Referenceの catalog/inspect/trace/validate、二表現の再読取、同byte Source移動による WorldFP 不変と SourceIndexFP/Trace/引用位置変更は限定実行済み。**Product Skill install/load・外部Harness実受信は未実施**。同様に、[TAI Draft #270 SK-02](https://github.com/hat47x/tai-pleroma/pull/270)の4 Session H2→H3→H3→H2 の適格性Checkerは公開合成E01〜32の別の局所報告であり、Skillの正答・改善率は not-evaluated。

本PRが増やすのは新Benchmark/Runnerではなく、#2605（実Tool/Skill）、TAI #270（比較適格性）、#2018（独立品質・費用比較）、#2514（独立Expected）へ渡す**同一Taskの検収票**である。

| 接合 | 最初の対象 | 前進と呼べる条件 | 不可とする誤昇格 |
| --- | --- | --- | --- |
| **E0 実製品のread-only対照** | 合成公開Workspaceで特定Subjectを型付きIDで検出し、Trace/Source位置/Warningを回答 | 実CLI binary+argv+exit+Model/SourceIndex/Trace版、必要Sourceと診断の引用が一致 | 以前のfocused CLI PASSを現在checkout/Skill全機能へ転用 |
| **E1 H2/H3 Product Skill差** | **同じTaskとCLI**をSkill無しH2／実Load済みH3で実行 | 4独立Sessionのbinary、Task、Prompt、Tool Profile、Grant/Disclosure、Skill+依存Script bytesの条件を事前照合し、その後**独立Expectedで正答を採点** | checker eligible-to-score を正答・Skill効率改善と呼ぶ／H3未LoadやH2 Skill混入を許す |
| **E2 実Invokeの作用** | 許可したread-onlyのみ、複合Shellと artifacts -out は負例 | 実argv全体、実File/Network/OS observerの可視範囲、失敗・unsupported・not-runを分ける | 先頭 read-only コマンドで全lineの作用を決定／静的検査でOS非漏えいを証明 |
| **E3 多Consumerと再資格化** | Source同byte移動、非表示Rule/Predicate更新、Skill依存Script改版を**一軸ずつ** | 意味再計算の要否と、Source出典・Tool/Consumer公開・現在Grantの再資格化を別欄にする。依存Edge不明ならunknown | WorldFP不変から引用/Skill/Actionが現在有効と判断／旧PASS全採用 |
| **E4 意味的Actionと実Effect** | 同じMeaning/Targetに対するBound Operationと実Host/Provider | #2598の実Request target/payload/版、#1174のReceipt/独立read、業務Ownerの意味受理を同一試行で照合 | Skill/Tool成功→DB更新・Business完了と断定 |

**最小の進め方**：まずE0/E1を#2605とTAIの既存試験で成立させ、E2で実際の副作用を確認する。E3は#2600 IB-11〜14/#2516 W0cに同じSource版の証拠を渡す。E4は#1119のP0-a/P0-b/G0〜G4をOwnerが満たすまで未達。新しい大規模なScenario群を増設せず、既存の#2616 I0〜I5/OB-01〜08から必要な検査だけを参照する。

### 11.3 正しさと開発生産性の独立評価

**比較の一次結果は「当該Taskを正しく完了したか」**である。その後に、#2018の既存BRC-09〜12/BRC-10 H2/H3事前登録に沿って、実Tool回数、架空コマンド、Source/Scope/Warning/Grantの誤昇格、追加Skill/Adapter保守費、Token/Tool費、wall time、人手レビュー・手戻りを取得する。初期配布費とN件目の限界費用を別記し、分母を「採点可能だったTask/Session」と「正答完了Task」で混同しない。4 Sessionだけではp95や全Harnessへの一般化を主張せず、未測定の価格や時間はunknownとする。

**資格表示**：static-inspected、synthetic-test-passed、actual-cli-observed、harness-load-observed、tool-invocation-observed、consumer-observed、provider-effect-observed、owner-accepted を単一progress数値へ圧縮しない。これは既存Evidenceへの**説明ラベル案**であり、正式enum/新Gate/Registryを採用する提案ではない。unsupported、not_run、insufficient-evidence、fail、successは根拠と対象を明記し、成功/失敗の二値へ丸めない。

**停止線と現状**：E0の既存TEI限定Read-only実証は#2626の根拠を参照。#2607のTS-16〜19の新全体11件/実checkout/CI、TAI SK-02の実4 Session、実Tool/OS/Networkの観測、同Taskの正答・費用改善、Action→Provider Effect、Owner意味受理は**本節としてNOT RUN**。外部HarnessをTEI Target生成物の実行必須条件にしない。新Host/Policy/Skill Runner/Universal Shell Parser/Benchmark Engine/Authorityの第二正本を設けない。

<!-- 20261103-real-cli-ob01-02-test-bridge -->
## 12. OB-01/02の「実CLI」に対する最小検証をGitHub上へ接続（2026-11-03）

### 12.1 設計から試験実行導線へ移した範囲
§10でNOT RUNとしたOB-01（CLI discoveryとdispatchの差）／OB-02（OperationSurfaceの期待ID/gap/binding/版）について、既存[#1127標準CRUD負例](https://github.com/hat47x/tei-platform/blob/main/reference/standard-crud/README.md)を**実際にビルドしたGo CLI**で読む試験スクリプトと、PR exact HEADで走る専用GitHub Actionsを本Draftに追加した。

- [試験スクリプト](../../plan/fixtures/action-surface-cli/verify_real_cli.py) はGitHubチェックアウト内の公開合成 `reference/standard-crud/workspace`を実CLI `operations -workspace ... -application ledger`で読み、既存READMEに凍結された5 Operation／4 Gapの順序、missing-semantics・conflict・missing-input、Rule ID/Version/Fingerprint、World/Model Revision、authorityLimits、write Operationのbinding/Effects/expected-state、同じ固定入力での再実行同値を検証する。既存Go単体テストをそのままPASSとして流用せず、CLI subprocess/JSON解釈・Pathを含む実受渡しを試す。
- **新たに静的Sourceで確認した差**：`tei operations` はstdoutへのread-only Surface表示だけでなく `-out <file>` を受け、`os.WriteFile`で**実ファイルを書き込む**。従来§10/§11で強調した `artifacts -out`と同様、`operations -out`も「read-onlyなOperation Surface」と「CLI invocation全体のOS作用」が別である具体例。スクリプトは一時ディレクトリのみで `-out` を実行し、stdout版とのJSON同値と実ファイル生成を確認する。ファイル書込みの内容同値は原子的な出版・権限・秘密非送出の保証ではない。
- [Focused Workflow](../../.github/workflows/action-surface-real-cli.yml) はPR exact HEADを固定、チェックアウト状態/スクリプトBlobを表示、`go.mod` のGoを設定、Checkout**外**の一時パスへビルド、実CLI/公開負例Workspaceを実行し、終了後のgit cleanを確認する。変更経路以外のGo/Provider/Host/実DBを実行しない。

**現状の厳密な区別**：ここで行ったのは**GitHubへの実試験SourceとCI実行導線の保存**である。Workflow run のexact-head successful conclusionとlogを別途照合できるまでは、OB-01/02の新規実CLI PASSと主張しない。元の#2587 internal/operations Go単体・#2626の実CLI限定成功は先行別証拠。OB-03〜08は引き続きNOT RUN。仮に本試験がPASSしても、Meaning受理、Rule全入力閉包、CLI完全discovery、実Product Skill/Harness、実Provider、Artifact公開の原子性、Host Grant/Receipt/独立Read、業務Owner受理は未確認。

### 12.2 残されたドキュメントドリフト（他Ownerへの引渡し）
既存[OPERATION_SURFACE_DESIGN](OPERATION_SURFACE_DESIGN_2026-11.md)と[DERIVED_CRUD_OPERATION_DESIGN](DERIVED_CRUD_OPERATION_DESIGN_2026-11.md)は初期設計時点の `tei operation surface`／W5 CLI未着手／全コマンドdiscoveryへの追加を記述している箇所がある。一方、現mainには `tei operations` の実dispatchがあり、`tei commands` は意図的に selected-read-only-commands を返す。**既存文書は歴史的な提案と現実のCLI仕様が一部食い違っている**。本PRで他Ownerの設計文書やCLI ABIを無断変更せず、#1127の担当へ「実dispatch/引数/既存契約/現行Statusを照合して注記を修正する」差分のみ引き渡す。help usageの列挙漏れも同じ現実確認対象。CLIが存在することから正式公開・全利用資格を推論しない。

### 12.3 他チャットの業務状態・ジョブ管理への非昇格
標準Operation Surfaceに `Autonomy: unattended` と表示されても、E​​KIのジョブ起動資格、Workflowの状態遷移許可、期間状態、リソース予約（GPU/CPU/Memory）や継続実行の安全性は導出できない。状態は業務レコードの属性/履歴/対象固有契約、ジョブの実行とリソース調停は各Runtime/外部Coordinatorが担い、TEIはその意味モデルとシミュレーション/可視化・監査根拠を提供する。#1125 Business Procedureの手作業・契機は自動実行定義と混同せず、#2622 Providerが実現可能な副作用との差を残す。具体的なEKI連携はOwner側の独立Fixtureと実観測が得られるまで、新しい万能State Engine/Host/Resource RegistryをTEIへ追加しない。

<!-- 20261103-ob01-02-actual-exacthead-success -->
### 12.4 実Go CLIによる限定OB-01/02：exact-head CI成功を確認

保存後に実行された[GitHub Actions #41607482804](https://github.com/hat47x/tei-platform/actions/runs/41607482804)の **operation-surface-real-cli = success**を、ジョブstepと実行ログを再取得して確認した。対象HEADは **`326b905a195b954caad5a062b88e260d91f48236`**、試験スクリプトGit blobは **`53994f47d9851a9d850805817937efb5b2e314b3`**、Go **1.26.10**、公開負例Workspaceを実際のビルド済みGo executableに渡した。

観測値：`operationCount=5`、`gapCount=4`、`worldFingerprint=sha256:bd53dbaca0ac2c20bc911e0b69083467f2ad38273911269d525db666e681817f`、`modelRevision=sha256:c60ed68c7c85ab218883b18970df85db1ebcae7d9090336452766d7a95d2b04d`、`ruleFingerprint=sha256:92adf0fa3b20816fb6068780a6dd8b9b1fb77973e6c2cff823313346b0e59ae0`。実stdout二度一致・事前固定の5 Operation/4 GapとBinding/Effect/Concurrency/AuthorityLimits、`operations -out`の**実ファイル生成とJSON同値**、Checkoutのclean pre/postを試験した。

**成立したClaim**：現HEADの一つの公開合成Workspaceで、Read-only `operations` Surface表示と、同じcommandの`-out`によるローカルFile Writeが実Go CLIで観測でき、Gap/Bindingを黙って成功/Grantへ昇格させない判別条件を満たした。これはCLI/Artifactの**出力先差**を確認したのであり、Source変更中の二重Load Race、原子的Artifact公開、Shell連結でのOS作用、Product Skill実Load、業務Actionの入力Closure、Provider/Host実Effect・Receipt、Owner意味受理までは証明していない。既存§10のOB-03〜08と§11のE1〜E4はNOT RUN。Source/Codeの本体改変なし。最新HEADで新変更が入れば、既存Runをそのまま現在のQualificationへ転用しない。

<!-- 20261103-real-handoff-runtime-async-audit -->
## 13. Actionの対象同一性、実行時の最小フェンス、EKI非同期検証を接合する（2026-11-03）

### 13.1 なぜOB-02の成功から業務成功へ進めないか
§12の限定実Go CLIでは、標準Operation Surfaceが`ledger.update.sales.order`→`sales.order.cancel`のBindingを正しく導出できた。しかし、Binding名と`Outcome: cancelled`が正しくても、**どの注文に実際に作用したか**は別問題である。[#2598 IB-01〜08](https://github.com/hat47x/tei-platform/issues/2598)と[#1174 AT-1](https://github.com/hat47x/tei-platform/issues/1174)が対象同一性のOwner。

新しい [Go test-only characterization](../../internal/driver/action_surface_handoff_test.go) は、公開`standard-crud`の`operations.SurfaceWorkspace`から実`Operation.Binding`を取得し、既存`driver.Resolver.InvokeCapability`へ渡す。呼出者が注文`O-1`を選んでも、**故意に誤ったAdapterが`O-2`を変更し`Outcome=cancelled`を返す**公開合成負例を固定する。Driverが正しいCapabilityを解決・呼出したことと、業務対象`O-1`の確定を区別する検出用Witnessであり、**Driverに新たな対象一致ガードを実装したものではない**。実Provider/Host認可、実Transaction/DBコミット、Receiptはこの合成で検証しない。

**検査位置の結論**：Adapterが返した結果だけを後から比較しても、別対象への不可逆な副作用を防止できない。業務Operation/Requestで要求する対象と実Providerへ渡る対象について、書込み直前の既存Host/Adapterの契約における**target/scope/expected-state/current grant/副作用の許可範囲**を束縛・確認する必要がある。実際の更新対象はReceipt・post-image・別readで確認する。単一行の`orderId`一致検査を、複数行Operationや集計への汎用規則に昇格させない。対象集合・関係・時間条件は受理済みMeaningの用途別契約で定める。

### 13.2 バリデーションを実行段階ごとに配置する

他チャットのGo単一バイナリ／DSL Source Preservation、TEI Desktop/SHO、EKIのTask/Attempt/Leaseとレプリカ監査計画を、**ホットパスの最小保証**と**時間をかけられる独立監査**に分ける。静的な全モデル検証を毎回実行せず、しかし実行前に必須の安全なチェックを「性能向上」の名目で省略しない。

| 時点 | 最小検査と不確実性 | 既存Owner／処理を置く場所 | 禁止する誤判定 |
| --- | --- | --- | --- |
| 編集・Preview | DSL構文、Owner受理前Candidate、外部参照・依存、Source Span、UI表示・Read-only | #988/#2621/#1124とStudioの既存診断 | Previewの`Committed`を業務Commit／Event発行と呼ぶ |
| Load・Build | 元Source＋明示外部Import/Role/Operator/Generator版、Meaningの受理範囲、依存閉包、Contract conformance | #2620/#2639/#2667/#2600の既存Loader/Builder | WorldFP同一からSource引用・Plugin/Tool/Consumerが有効とする |
| 書込み要求の直前・Commit | request target/key/scopeと用途別Action/Capabilityの対応、Provider現在能力、Version/expected-state、current Grant/Period、冪等性・Commit不明・Receipt規約 | #2598/#1174/#1129と実Host/Provider | CLI/Tool/Jobの受理から業務更新完了を推論／不明Commitを盲目再送 |
| 完了直後 | 実対象・Effect/BusinessResult、Commit/Receipt、別read、ReadのScope/epoch/cohortと拒否/不明 | #1163/#1174の観測、#2516のConsumer | 自己応答やレプリカ旧値/0件を確定観測とする |
| 非同期・広域監査 | 全集合整合、履歴、Relation全件、期間横断、負の依存、変更後再資格化、複数Provider/Replicaを**水位と時点つき**で再評価 | TEIの検査定義を既存Batch/TaskのExecutionへ接続。EKIはジョブ/資源/Leaseの調停（別Owner）。結果のQuality/Business判定はTEI/業務Owner | EKI`Job Accepted`／`Task succeeded`だけをモデル/DB品質の合格とする |

**最小の常時負荷**：対象を限定したexpected-state/CASや認可・Effect Fenceは書込みに必要で省けない。広域集計・履歴再照合、全Relation走査、第二Providerとの総突合せ等を、要件を満たす範囲で夜間・独立Job・Replicaへ委譲する。ただしReplicaが古い場合は`as-of watermark`／Snapshot/LSN/対象Scope/更新からの遅延と完全性を記録し、単に差があるから自動ロールバック・再送しない。非同期検証が必須の業務については、業務受理条件がその完了待ちを要求する場合だけ`BusinessVerified`へ昇格する。

### 13.3 TEIとEKIの境界：ジョブ受理は品質証明ではない

Goでビルドした単一ネイティブバイナリはTEIの標準実行形態を維持し、EKIは**任意の配備・実行基盤**として、Resource/Task/Attempt/Lease/スケジュール・リトライ・Telemetryを提供できるようにする。TEI独自の常駐Hostや中央Registryを前提にしない。TEI側はモデル・純粋な検査定義・必要な契約およびObservationの意味を記述し、ジョブ実体の起動と資源隔離はEKI/外部基盤が担当する。

特に以下の状態は**別々の観測証拠**として表示する（正式な共通enumの新設ではない）。
- `Plan Prepared`／`Reference Execute`：候補・実行計画／隔離またはReference実行に限る。
- `Job Accepted`／`Attempt Running`／`Job Succeeded`：実行基盤での受付／生存／プロセス完了であり、業務効果を証明しない。
- `Provider Commit Known`／`Commit Unknown`：実Providerの確定状態。Response喪失を成功／失敗へ丸めない。
- `Business/Quality Verified`：必要な独立Read/業務期待/範囲/版を満たした場合のみ。

長時間監査・レプリカ上の検証を非同期化することは、常に即時の完全性保証と交換可能とは限らない。境界内で即時検査を必要とする制約（対象同一性、権限、CAS、二重作用防止）はCommit直前に残し、その他の検査をOwnerが用途別に分類する。

### 13.4 実証順序・引継ぎ・未完了
1. **OB-07A（本DraftのGo test-only）**：OperationSurface→Driverの実Go経路と誤対象Adapterの独立観測を同一公開Workspaceで比較。結果は**まだ業務保証ではない**。CIのexact HEAD/Go test結果は別記録する。
2. **OB-07B（#2598 Owner）**：同じ`selected target`／Action Subject/Scope/Context epoch／Source Revision／`request payload`と`actual effect subject`の受渡しを、既存Driver/Host適用境界で事前確認できるか評価。未対応なら`not proven`で止め、Action Offerから無条件Invokeへ進めない。
3. **OB-08（#1174/#1119 Owner）**：Versioned actual Provider/実PostgreSQLにおいて、Commit/ACK-loss、Receipt、独立Readとtarget食違い、CAS拒否、Runtime Grant/Periodの現行性を別の同一試行票に記録。
4. **OB-09（EKI/運用Ownerへの契約候補、未実施）**：同一Meaning・Snapshotが固定される最小広域監査Taskを一つ配備し、`Job Accepted`と`Quality Verified`を別観測。Replica watermark/lag、全件可視性、部分失敗、結果を待つ業務と待たない業務を区別し、最小CPU/IO/Latency比較を#2018/#2600に返す。EKI必須化はしない。

**Evidence境界**：実Go CLIのOB-01/02は§12で限定成功済み。OB-07Aは[Focused Actions #41611947261](https://github.com/hat47x/tei-platform/actions/runs/41611947261)の候補HEAD `7fc6cd7512f590ce6102c3165862272a95460172`・Go1.26.10で**実Goテスト1件PASS**（実CLI OB-01/02・クリーンCheckoutも同job PASS）を確認。ただし合成Adapterの誤対象作用という**負例の現挙動の特性確認**であり、製品の事前target/fence機構が安全に拒否できるという証明ではない。OB-07B/08/09、実Go単一バイナリの配布性能、EKI実Task/Lease、Replica/水位、実DB/Host、SHO/Studio/Consumerの一体受入、Owner意味受理は**NOT RUN**。共通Host/新State Table/Proof Engine/新Quality Registry・直列Gateを追加しない。

<!-- 20261103-phase-guard-and-deadline-bounded-action -->
## 14. Offer時点からCommit・非同期検証まで：期間・委任・期限の「有効性」を混ぜない（2026-11-03）

### 14.1 Source照合と、他Laneとの非重複

他チャットの業務Lifecycle/SQL状態/組織委任・EKI資源調停議論を、現mainのGo実装と照合した。[#2744 Buri/Flow/EKI設計Draft](https://github.com/hat47x/tei-platform/pull/2744)には、[TEI×EKIの期限・予算・連鎖失敗](https://github.com/hat47x/tei-platform/blob/4b36fb6713d3c78f61bc119386c8ff450c79f4da/plan/design/TEI_EKI_DEADLINE_BUDGET_RECOVERY_COORDINATION_2026-11.md)（LB-01〜17）、[Pool資源の委任Gate](https://github.com/hat47x/tei-platform/blob/4b36fb6713d3c78f61bc119386c8ff450c79f4da/plan/design/TEI_EKI_AGGREGATE_DELEGATION_UNRESOLVED_GATES_2026-11.md)（DG-01〜14）、[ジョブ実行委譲境界](https://github.com/hat47x/tei-platform/blob/4b36fb6713d3c78f61bc119386c8ff450c79f4da/plan/design/TEI_EKI_JOB_DELEGATION_BOUNDARY_2026-11.md)がすでにある。この§14はEKIの資源最適化やTEIのBusiness Lifecycleの意味Ownerを再定義せず、**#2616のUI非依存Action Offer→現実のExecute/Commit/Verify**の一段を接続する。

現mainの製品コードには`operations.CheckPeriods`、`operations.GuardCapabilityWith`、`periodstate.File`、参照実行者の`submitIfReadyGuarded`がある。期状態は受理済みMeaningにある定義と異なり、実行時Sourceが供給する。[merged #2754](https://github.com/hat47x/tei-platform/pull/2754)のFile Readerは、JSON内に対象Periodが**存在しない場合にはunguarded**、不正な状態値・ファイル読取り不能はerrorとする。これは意図的なReference方針であり、**「存在しない状態＝現在openと立証済み」という主張ではない**。また実行者の`PeriodGuard.Source=nil`は明示的にGuardなしの実装経路である。Guardが導入済みだからすべてのDriver呼出しが原子的に期間/Grant検証済みとはいわない。事前GuardとCommitの間のstate change/TOCTOUは#1174等の実Host/Provider Ownerの契約。

### 14.2 一つのActionで変わる5つの時計と、異なる権限

| 時点・時計 | 現在必要な証拠と否定しなければならない推論 | 既存Owner |
| --- | --- | --- |
| **Offer/Preview at t0** | Accepted Meaningの適用範囲、Function/Ruleのhidden read・NULL/未取得/partial、Actorの候補と実行targetの識別根拠。提示されたButton/Operation IDは現在Grantではない。 | #2507/#2506/#2516、#2499 |
| **Request at t1** | Subject/Rowset/業務Keyと物理PK、Scope/epoch、Source/Role/Operator、Expected old-state・選択したActionの対象の一致。t0からの組織異動・期限委任撤回・最新Business stateで候補資格が変わる。 | #2598/#2499/#2620 |
| **Provider execute/Commit at t2** | 現Grant・現Period・本当のTarget、CAS/同時更新、Effect fence、Idempotency/Commit identity、既存Host/Adapterが保証できる範囲。事前OK・Process成功・Driver Outcomeのいずれも原子的Commitの証明ではない。 | #1174/#1129/#2564 |
| **EKI Task admission→Attempt/Lease at t3** | Jobの`not_before/target_complete/latest_useful`、残り時間/物理Budget、Resource/Trust/Data locality、現在DLG-1/BUD-1/CAP-1/Provider Admission。予測上eligibleと実Reservation/Allocationは別。試行成功はBusiness Commitでない。 | #1130、EKI #112/#57/#58/#108/#109/#140 |
| **Independent observation/verification at t4** | 同じ業務対象・実Result Body、Source/Receipt・Read Scope/Snapshot/epoch、Replica watermark/lag、独立Expected。t4が期限超過なら、t3が成功でも期限内の業務完了とは認定しない。 | #1163/#2514/#2516、#2018/EKI |

厳密な時系列はTaskが業務Commit前の純粋Computeであるか、Commit後の副作用通知か、検証Jobかで異なる。上表のt0〜t4は**証拠境界のラベル**であり、全Jobに固定の時系列や新しいState Machine公開enumを強制するものではない。特に`Group Job`が再試行・部分成功した場合、Task/Attempt/Lease/Provider ExecutionRef/Business Operation・Event IDを取り違えない。委任先が実Commit対象を推測したり、自身のSuccessだけでBusinessVerifiedを発行しない。

### 14.3 期限と回復予算の最小計算単位

`latest_useful`を過ぎてから実結果が返ったときの業務利用可否と、`target_complete`到達率を別にし、**計算終了だけでなくOutput転送・独立検証・業務受領に要する時間**を見積りへ入れる。失敗後は`remaining_time = deadline - observed_now`を初回値からリセットせず、残存予算と再試行可能な副作用の資格を再評価する。EKI内部の予測/Shadow値は権限や予約の代わりにならない。とくに試行timeout+Provider ACK欠落では元ExecutionRef照会を優先し、期限圧力だけで二重実行を許可しない（#2744 LB-12/17）。

異種資源の選択は直接実行も正常な候補として扱い、Broker往復/転送/検証で期限を浪費する小Jobを強制的にEKIへ流さない。TEIの組織可視性/Workflow承認はEKIのAdmin Budget/Local Governorを増やさず、EKI Pool Offerや予測実績から業務レコードの閲覧Grantを増やさない。分布・内部Node数の開示もAudienceごとに別に審査する。費用はまず物理量/観測者を記録し、価格・課金・最適化PolicyをEKI Coreの新責務として押し付けない。

### 14.4 境界ごとに何を受け入れるか：AJ-01〜08（独立Expectedの反証候補）

| ID | 固定して比較する負例 | 受理できる最小観測・未証明境界 |
| --- | --- | --- |
| **AJ-01 Period TOCTOU** | t0でopen、t1でclosing、t2でclosed／Source file欠落／不正値 | 固定SourceからのOperation候補と**現在StateSourceの再読取**を分ける。GuardによるrefusalをProvider内部の原子的Fence成功に昇格しない。 |
| **AJ-02 組織・委任撤回** | t0 A部門のAction Offer、t1異動／時間制限付き代理権失効、t2外部Providerの古いGrant | 機能の表示と、t2のcurrent Principal/Scope/Grant・実Effect=0の拒否を別確認。|
| **AJ-03 対象入替え・ABA** | t0選択O-1、t2のAdapterはO-2へCommit／同じ表面値に戻った別Revision | 対象とeffectの現在一致・独立Readが必要。成功Outcomeや結果値Digest一致だけでは不十分（§13既存Go負例）。 |
| **AJ-04 期限後の検証** | 計算は締切前、Artifact Body取得／Quality検証／業務受取は締切後 | Task SuccessとOn-time Verified Business Completionを別採点。レプリカの旧Snapshotを現在の受理根拠にしない。 |
| **AJ-05 故障後の残り時間** | 50分Deadline、15分失敗＋二度目失敗、残り12分で再実行・検証16分 | 不可能な期限を成功予測へ丸めない。Ownerが承認する部分Result/Fallbackだけ提示。 |
| **AJ-06 実行権限と資源予約** | TEIのRole/Workflow承認あり、EKI BUD-1は残るがProvider Quota/Local Governor拒否 | Offer/Forecast/eligible/reservable/actually allocated/observedを別票。自動Scaleout/支払は不可。 |
| **AJ-07 ACK-lossと二重Retry** | Provider Job受理後のACK消失、EKI/Providerがともにretryしようとする | Operation/Task/Attempt/ExecutionRef/Receiptの相関・lookupを先行。Effect未知なら再送を保留。 |
| **AJ-08 組織階層・実績開示** | 低権限ConsumerへPool内部のNode数/精度高い成功確率や非表示注文の候補件数が出る | 業務データと資源の双方を別のDisclosure Boundaryで遮断。最終回答だけ一致でもTool計画/診断/ログが漏れれば不合格。 |

**試行の最短順**：①[main起点の独立Draft PR #2784のGo期間Guard試験](https://github.com/hat47x/tei-platform/pull/2784)で`operations.GuardCapabilityWith`と実JSON StateSourceの更新を固定 → ②#2598でAJ-02/03のRequest Targetとcurrent Grantを既存Adapter/Provider前の契約で検証 → ③#1130とEKI #112が**純粋Job一件**のInput Artifact digest/body→実Task→Output bytes→TEI独立Oracleを先に観測 → ④実Provider Commit/Receiptと期限・予算・回復のend-to-end計測 → ⑤SHO/AI/Studioなど実Consumerの必要開示だけを評価。EKI #112の既存Spike報告にあるArtifact ingest 422／Result API 501を実E2E成功扱いしない。新しいJob Runner/Authority Registry/万能State Schemaの追加や、現有#1119 G0〜G4の再定義は行わない。

**この§14の証拠資格**：既存main/PR #2744のコード・文書静的照合と、実Goの限定負例・参照Gateの接合設計。AJ-01の新Go試験は、GuardCapabilityWith/periodstate.Fileが既にある**新main基線の独立Draft PR #2784**へ分離した。旧#2616 branchに新main APIを無検証に引用した初版はCI失敗となったため、その試験コードを#2616から移し、旧Focused Workflowの安定構成を復元した。#2784現HEADのGo試験/CI/logの成功を確認するまではPASSとしない。AJ-02〜08の同一縦断実Consumer/EKI/Provider/OS/Replica、Deadline/Costの物理実測・Owner意味受理はNOT RUN。正式DSL/TEI/EKI ABIとBusiness Grantは追加・採用しない。

<!-- 20261103-action-invocation-effect-evidence-join -->
## 15. Function/Action Invocationの「実行要求と業務効果」を、既存Command/Effectcommitの反例へ接合する（2026-11-03）

### 15.1 横断照合で得た新しい結論

**標準操作が見つかること ≠ 業務Actionの目的が受理されていること ≠ Requestの対象が拘束されていること ≠ Commit/Receiptが分かること ≠ Consumerへ結果が正しく届くこと**。他チャットのDSL/Studio/AI/Batch共通Function Invocation論点と、現行実Goの独立した証拠を一試行の検収列に直す。

| 実証Source（同一製品E2Eには合算しない） | 現在実測した特徴と厳密な限界 |
| --- | --- |
| [#2616 OB-01/02・OB-07A](https://github.com/hat47x/tei-platform/actions/runs/41633212269)：actual Go CLI + derived Surface→Driver | 公開`ledger`はOperation 5／Gap 4、`operations -out`は実File write。合成Adapterで選択O-1に対しO-2へのEffectでも成功Outcomeが返る。Offerの正しさも対象の正しさも保証しない。 |
| [#2666実CLI A1](https://github.com/hat47x/tei-platform/actions/runs/41632245693)：Operation→Artifact/Manifest | 公開`order_entry`はOperation 8／Gap 0でもManifest.Complete=false。**Operation候補の未解決0はArtifact・Function Context・外部Consumerの完全性を保証しない**。 |
| [#2780 MC-B01/C01/D01/D02](https://github.com/hat47x/tei-platform/actions/runs/41632377548)：実Command Invoker→Adapter→Effectcommit Runner + 合成Provider | (B) Commandの選択ORD-1と明示PayloadのORD-2不一致を拒否せず転送、(C) 内側commit成功と外側UnitOfWork commit失敗が分裂、(D01)初回とreplayedをCommand空Resultが区別しない、(D02)commit uncertainの型付きCauseはerrors.As可能だが外側Error Codeは汎用。4/4 PASSは**ギャップの検出**であり正しいフェンスの成立ではない。 |
| [新main起点Draft #2784](https://github.com/hat47x/tei-platform/pull/2784)：actual`periodstate.File`→`GuardCapabilityWith` | Preview後の期間open/closing/closed/欠落/unavailable/invalid変化を検査する。既存Referenceの**キー欠落→unguarded**と**File失効→error**を区別。これはDriver/ProviderのTx内期間Fenceではない。 |

これらは別のSource/Branch、別のAdapterと異なる実行場所で得られた局所証拠であり、操作5/8件・Go試験4件・期間ガードという結果を足して**同一Business Operationが正しく実行できた**と主張しない。特に#2780のCommand Adapterと#2616のDriverは**別の入口**であり、同じFrameworkの自動実行保証に統合しない。

### 15.2 「用途の同一性」と「対象の同一性」と「副作用の同一性」

形式を先に固定せず、同一のAttempt/Operationについて次の**照合関係**を既存Model・Request・Receiptの読み取りで表示できるか検証する。新たな公開Schema/Registry/Hostは設けない。

| 証拠の結合点 | 最低限区別する情報 | 無証拠で採用してはいけない主張 |
| --- | --- | --- |
| **Purpose**：Function/Action | `update`等のOperation Kindと、`cancel`／`submit`／`approve`等の**業務目的・受理済みMeaning、適用Rule、対象Role** | 標準CRUDの`update`から取消等の固有業務意味を生成した |
| **Subject**：Offer→Request | 選択業務ID＋Occurrence/Scope/epoch、RowsetとRelation多重度、業務Key vs PK、Expected Version | Clientが送った単一`orderId`値＝Providerが実更新する対象集合 |
| **Current authority**：Dispatch→Commit | Request版・現在のPrincipal/Grant/Period/Provider Capability/実Scopeの再照合、認可範囲、同時更新 | Offer時の権限・Preflight Guard＝Commit時点の権限/期間Fence |
| **Effects and transaction** | 実Effect Target/Phase、関係するTxn boundary、Idempotency identity、内外UoW同一Txの根拠 | 片方のCommit成功/外側Rollbackで全副作用を原子的に確定/取消できる |
| **Outcome and Recovery** | New/Replay/NotCommitted/Uncertainの区別、同一OperationIDでのReceipt/Lookup・未知状態 | `command.Result{}`/一般Error/成功文字列＝実Commit確定または未実行 |
| **Observations and consumers** | 独立Readの対象・時点・Snapshot・Read completeness、Consumerの実受領byte/開示版 | Adapter自己応答、Manifest Gap=0、EKI Task Success＝BusinessVerified |

**利用者Interaction**はこれらの表示/要求/入力と中断・再開をTarget固有UI（Web/Desktop/CLI/Batch）へ投影するだけで、別の業務意味や実行認可の正本を作らない。CLI/Job等が入力不要なら同期Promptを要求せず、対話が必要な場合も「入力待ち」「未実行」「Commit不明」「結果未受領」を別表示する。AIはPlan/候補/Previewを構成できるが、Grant Issuer・ビジネスMeaning Owner・実Hostとは区別する。Source origin/Revision、Function Contextのhidden predicate/Rule依存、取得完全性が不明なら、画面のEnable状態をCommit許可と誤認しない。

### 15.3 三つの停止点を最小受入として固定

- **Q1：実行前**：選択目的/対象・Request Payload・current Scope/Principal/Grant/Period/ExpectedVersionと、Adapter/Providerの書込み対象・権限・現在Snapshotを同一Attemptで突合。合法な関連明細へのEffectも存在するので、文字列IDが異なれば一律拒否する短絡規則は採らない。業務Meaningに基づく「許された対象集合」を定義できない場合は **unsupported／不明**。
- **Q2：実確定時**：Commandの内外TransactionとEffectcommitが一つの物理Txに参加する根拠を持てない場合は、原子的Commit/一括Rollbackを主張しない。ACK-loss/Commit uncertainでは、同じOperationIDでReceipt Lookupを先に行い、別OperationIDで盲目的に再実行しない。RunのSuccessとResource/Business Effectを分ける。
- **Q3：結果提供時**：必要なReceipt・対象・独立Read・Manifest Body・Reader SnapshotとAudienceの現在Disclosure/意味契約を照合。Commit済みでもBusinessVerifiedは別。Consumerには`committed`と`replayed`、`unknown`を同じ「成功/失敗」へ丸めた可視化を行わない。EKIの`completed`は実業務Effect/期限内検証済み結果の証明ではない。

### 15.4 次の検証順序・各Owner・保留

1. **現行境界の合流表（設計／Owner #2598・#2507）**：同じ公開`sales.order`のAction Purpose、Operation/Binding、選択subject、Command.Input、明示Payload、Provider target、現在のPeriod/Grantを一行に固定する。現行の別入口間で結合できない箇所は**unverified**とし、横断の擬似Successを作らない。上の#2616/#2666/#2780/#2784既存FixtureとIndependent Expectedを再利用。
2. **実行安全化（実装・実Provider Owner #1174/#1129）**：Q1 mismatch/失効の事前拒否で**実Effect=0**、Q2 atomic boundary／Receipt／ACK-loss recovery、Q3別readで対象・結果を確認できたら初めて受入対象とする。合成Adapterの望ましくない挙動を仕様として保存しない。
3. **Consumer/Job接続（#2516/#1130、TAI/SHO/EKI各Owner）**：Function InvocationをUI・CLI・Batch等で同じ業務契約に束縛し、Runtime InteractionとResource Executionだけを適応。EKI #112の実Artifact ingest/Result retrieval未達を飛ばして実「結果あり」とは言わない。実Web/Desktop/SHO/AI/Host/EKI、実SQL/Provider、Deadline/物理費用、Owner意味受理は未実施。

この節は**限定実証の非同値性を既存計画へ接合したもの**であり、新正式DSL/Operation Runner/Effect Fence API/共通TX Coordinator・Action Schema・Role Authority正本を採択しない。#1119 G0〜G4、#1174/#1130の既存Owner順序、既存Go単体バイナリと外部Hostへの自由度を保持する。

<!-- 20261104-action-input-to-execution-evidence-qualification -->
## 16. 11/4横断再照合：Action Offerの「実際に使われた意味」を証明してから実行資格を判定する

### 16.1 新しい失敗クラスと、既存#2616の範囲

同じ日に提出された[TEI #2822（宣言0件でもvalidate ok）](https://github.com/hat47x/tei-platform/issues/2822)、[#2823（Source root誤り・読込み対象外を黙殺）](https://github.com/hat47x/tei-platform/issues/2823)、[#2824（Discoveryが各CLI JSON契約の範囲を広告できない）](https://github.com/hat47x/tei-platform/issues/2824)と、[Generator実消費Source #2595](https://github.com/hat47x/tei-platform/issues/2595)を再照合した。従来§8〜15は、正しいFunction/ContextとAction Offerが**存在する前提**で、対象・Grant・Period・Effectを区別してきた。しかし、Source宣言・Catalog・Validation成功だけでその前提が満たされるわけではない。

本節での限定的な判断は、「全てを強制的に再解析する」でも「統合した証明Registryを作る」でもない。**そのAction/Claimが依存する意味が、当該Attemptで実際に使われたことを、既存のSource/Build/Execution/Receiptの記録から追跡できる場合だけ資格を主張する**。製品の既存`validate ok`や`operations effect=read-only`の意味を、ここで勝手に変更しない。

| 独立の証拠段階 | 求める事実 | 誤った昇格と拒否条件 | 所有Issue |
| --- | --- | --- | --- |
| **D0 Declared** | Function/Rule/Relation/Period、Source occurrence・byte/revision、必要Data Context・Scope | manifest上のSource名が有るだけで存在・Reader到達を証明しない | #988/#2571/#2823 |
| **D1 Resolved / Parsed** | 許可されたSource root、実発見Member集合、read/skip/unknownと診断、必要Role/Plugin版 | 空Workspaceでもvalidate ok、標準Catalogの非0件だけでUser Meaning取得を認定しない | #2822/#2823/#2639 |
| **D2 Built / Runtime consumed** | 実際に入力されたSource/Operator/Generator/Executable byte、実Target Artifact、依存閉包 | 宣言Source D1と実Build入力D2、Script A→B→A、同Hash別toolchain/実Providerを自動同一視しない | #2595/#2600/#1109 |
| **D3 Action request at current instant** | Purpose、選択subject/Scope/epoch、現在Role/Grant/Period、Provider Binding・expected version | 旧Offerやhidden predicateが無いという主張から現在実行資格を与えない | #2506/#2507/#2598/#2499 |
| **D4 Provider effect / Receipt** | 実Write対象集合、同一Tx/CAS/Effect Fence、Idempotency・Commit Unknown/Lookup | Action表示、Command success、外側ErrorをCommit成功/無効果の証明にしない | #1174/#1129 |
| **D5 Consumer observed** | 実Artifact Body/Manifest Member、独立read Snapshot、Audience Disclosureと取得時点 | Artifact Gap=0・Job completed・計算Output同値をBusinessVerifiedへ昇格しない | #2514/#2516/#2604/#1130 |

D0〜D5は**検収票の論理列**であり、すべてのActionが6段の新Runtimeを実装するという意味ではない。読取専用Offerの表示にD4は不要だが、書込みの現在許可にはD3/4が欠かせない。古いObservation/Claimはその時点の歴史的証拠として保存し、Source/Grant/Reader変更では**現在の実行/公開資格だけ**を失効させる。空集合・否定条件・Relation path・行/Field grain・partialなど非出力依存は、対象Claimに必要な範囲で閉包に含める。

### 16.2 独立Expectedで反証すべき最小組合せ（AEQ-01〜08）

| ID | 固定したInput/Mutation | 合格となる判定・誤った成功 |
| --- | --- | --- |
| **AEQ-01 Empty semantics** | ユーザー宣言0件、標準定義はCatalogに存在、`validate ok` | 対象の業務Action/Ruleが未導入なら`unsupported/unresolved`。Catalogが非空だからMeaning有効とはしない |
| **AEQ-02 Declared but skipped** | manifest root誤字・存在しないroot／`.TEI`・`.tei.bak`・`Application.tei`／workspace-filesの列挙のみ | 宣言・探索・解析・消費を別列。未読込Ruleに依存するActionを現在有効としない |
| **AEQ-03 Contract discovery** | CLI discoveryは`world`の非安定契約やmode別schemaを表示できない | `tei commands`だけで安定JSONの一式性を推論しない。実CommandのSchema/契約範囲とReaderを照合 |
| **AEQ-04 Hidden negative dependency** | 非表示Predicate/NOT EXISTSが空／Relation D2はpartial／0行表示 | 0行≠取得完了・依存なし。対象書込み候補のRule/Target/件数/粒度は再判定 |
| **AEQ-05 Source/Build ABA** | Source D1を宣言、実BuilderがD2を読む→再びD1。出力hash一致、別Generator版 | 実消費byteとToolchainの証拠がなければBuild結果再利用可能性と現在意味資格を分ける |
| **AEQ-06 Function×Build×Invocation** | 同じBusiness Functionを既定の単一Goバイナリ→batch用2 Buildへ移動し、旧UnitからCommand受領 | Function IDは保存、Build/Artifact/公開Entry/実RuntimeとCurrent Grantは個別に検証。Compile-in≠Public/Granted |
| **AEQ-07 Period and actor changed** | t0のOffer時open/許可、t1でclosing/異動、t2でProvider dispatch。Fileは欠落/invalidの場合あり | Guard source missing≠open。旧Offer・旧Grantは無効。Provider effect=0の安全性は実Txで別実証 |
| **AEQ-08 Delivery and replay** | 同OperationIDの再送・外側Commit failure・ACK loss・Result Body欠落・古いConsumer Snapshot | Committed/Replayed/Uncertainを別に扱う。Effect Unknownなら新IDで再送しない。独立Readと現在Audience資格が揃うまでBusinessVerifiedと呼ばない |

### 16.3 再利用する試験と、増やさないもの

**最小P0**：[既存Driver Scenario #1180](https://github.com/hat47x/tei-platform/issues/1180)と[JSON追加候補のDraft #2828](https://github.com/hat47x/tei-platform/pull/2828)の固定Scenario/Independent Expectedを利用し、AEQ-01〜04を単一Workspace・同一Source Revisionで**read-only**照合。Workspace/CLI契約そのものの修正は#2822〜24のOwnerに委譲し、#2616で第二Parser・Discoveryを書かない。

**P1**：#2616 OBの現実のCLI（ledger:5 operations/4 gaps）、#2666 Operation→Artifact（order_entry:Gap0でもManifest incomplete）、#2595実Build入力を同一Attemptに束縛できるかを調べる。合流不能な部分には`not-connected`と出し、別実行の成功ログを一試行の成功に合成しない。Function Identity／Build Identity／Invocation Identityの分離と、Go単一バイナリ既定・必要時のみ複数Buildは既存#1130/#2788へ委譲する。

**P2**：#2598のTarget/Period/Grantの現在性、#1174の実Provider Commit/Receipt、#2516の独立Reader/開示を、必要な場合だけエンドツーエンドに接続。初回は**read-onlyで閉包と欠落を確認**、次に一業務書込みで`target mismatch/commit unknown`を合成→実Providerへ段階的に展開。前回の[#2784](https://github.com/hat47x/tei-platform/pull/2784)のGo期間ガード、[#2780](https://github.com/hat47x/tei-platform/pull/2780)のCommand/effectcommitは別実行・別Ownerの既存負例である。

**性能・Authorityの停止線**：Source closureのread-only再検査と、Providerの必須current Grant/Target/CAS/Period Fenceを混ぜない。再利用可否は#2600、全件再検査の費用とfalse reuseは#2018。失敗時に「Sourceを読んでいない」を「正しい拒否が実装済み」と呼ばず、現行Referenceが黙って成功したら**現状を再現した負例**として報告。新TEI Host/Universal Proof Engine/Policy/Registryや標準DSLは導入しない。AEQ-01〜08を同一実Target・同一Attemptで通した試験、実Owner Accepted Meaning、実AI/Web/EKI Providerは**NOT RUN**。

<!-- 20261105-unattended-combined-preconditions-evidence-coverage -->
## 17. 11/5横断実装再点検：無人実行の「前提チェックが通った」と「十分な前提を検査した」を区別する

### 17.1 現mainで前進したことと安全保証の限界

[main統合 #2850](https://github.com/hat47x/tei-platform/pull/2850)により、`execution.EffectPreconditions(rules, observer)`は`after-commit`／`compensation`のEffectに対してObserverが無い場合、無人`RunPlan`を操作直前で止める。現mainの`PeriodPreconditions`と`CombinePreconditions`を使い、期間・Effectの検査関数を合成できる。これは**実装済みの限定的な入場ガード**であり、前節§13〜16で要件としていた全ての証拠が揃ったことを意味しない。

本物の`internal/execution/runplan.go`の静的照合により、以下の非同値を追加する。

| 現行コードで観察できる条件 | 言えること | 言えないこと・要追加検証 |
| --- | --- | --- |
| `EffectPreconditions`にnon-nil`EffectObserver`が渡された | 同じCapabilityに対する確定後／補償Ruleの「観測手段が全く無い」場合の事前停止を解除できる | 実際の`ObserveEffect`呼出し・対象Effectごとの観測対応・応答完全性・受領/検証/失効を確認していない。現実装は**Observerの存在を確認するだけ**で、対象Effectの実観測をこの関数内で行わない |
| `PeriodPreconditions(surface, states)`が成功 | その引数`states`についてGuardがRefuseしなかった | statesの取得時刻・出所・必須Keyの存在・currentness・Commit時の期間/認可一致。Source欠落とopenは非同値（#2784） |
| `CombinePreconditions`が空理由を返す | **渡されたnon-nil検査**は拒否理由を返さなかった | 業務上**必要な検査がすべて渡された**こと。Period検査が丸ごと欠けても、Effect検査だけがnon-nilなら合成関数はnon-nilであり、条件次第で進行できる |
| `RunPlan`の`blocked` | そのStepは実行せず、Blockと理由を記録した | Plan全体に副作用が無いこと。2番目のStepでBlockされる前に、**1番目が既に実行済み**の場合がある（既存`TestUnattendedRunStopsWhenAPreconditionIsNotMet`もこの順序を示す） |
| `runlog`がattempt/fenceを記録し、`StepResult: resolved`で`completed` | Journal内の試行と操作結果を追跡できる | `Journal.Last`→`Append`が外部横断の排他CASになったこと、`StepFunc(ctx,operation)`へfenceが自動束縛されたこと、Provider Commit/Business Result/Observer確認が成立したこと |

とくに**partial precondition coverage**と**partial effect after blocked**は異なる問題である。前者は「検査が存在するから十分」という誤り、後者は「途中で止まったから何も起きていない」という誤り。RunPlan自体が汎用Authority正本になったと見なさず、**業務・Host・Providerの実行前提と結果資格**を呼出し側/既存Ownerが明示する。

### 17.2 ACU-01〜08：実証の最小反例（既存Scenarioを再利用）

| ID | 反例 | 必要な独立Expected・成果資格 |
| --- | --- | --- |
| **ACU-01 Missing component** | 期間Guard必須Operationなのに`CombinePreconditions(EffectPreconditions(...))`しか渡さない | 現行挙動を観測して`unsafe coverage gap`と報告。必要Guard集合の不足を入場前に拒否する方式は#1130のOwner判断対象 |
| **ACU-02 Observer token** | Observer型の実装はあるが、対象Effectを観測不可/古い結果/別Scope | non-nilだけで`observed/verified`へ昇格しない。実Observer応答、対象・Receipt・Source/Time・Disclosureが別資格 |
| **ACU-03 Static period** | Preflight時open、次の操作またはProvider Commit前にclosed/Grant revoked | 期間Sourceの時刻/版とProvider内current Fenceを区別。実Effect=0負例は#1174/Owner実Provider |
| **ACU-04 Block after progress** | Step A resolvedし書込み可能、Step B precondition blocked | `Block B`でもAの実効果は不明/既確定の可能性。Run-level`no effects`を主張せず、step別Receipt/補償資格を確認 |
| **ACU-05 Two contenders** | 同runKeyを2 Processが同時にJournal.Lastした後Appendする | Journalだけの順序を物理排他と推測しない。Owner/Lease/fence/CASの実外部Writeで負例を作る |
| **ACU-06 Resumed effect** | Attempt1でA実Commit後にACK-loss、Attempt2で記録済みStepをskipまたは未記録Stepを再実行 | StepStatusとProvider実Effectを同一視しない。同OperationID/Receipt Lookupと独立ReadによるReplay/Uncertainの境界を検証 |
| **ACU-07 AI scenario evidence** | #2828のJSON候補追加/Go PASS、#2830の`generated-pinned`等のResult資格 | required/discovered/loaded/executed/independent-oracle/qualifiedを分ける。新Caseファイル数=Coverage分母ではない |
| **ACU-08 Meaning and Source drift** | Role/Operatorが自己記述され、Script SourceやRule版がOffer→Dispatchで変更 | #2620の`Role/Operator`有限BootstrapとAccepted Meaningを区別。Source実消費(#2595)、失効(#958/#2829)、Owner Acceptanceと現在Grantを別判定 |

### 17.3 担当と次の接続順

1. **P0: 既存コードでのcharacterization**：#1130の`RunPlan`と`CombinePreconditions`、#2784の期間File、#2850のEffectを用い、ACU-01/02/04で「止めたStepは未実行でも前Stepの効果はあり得る」「Observerの存在は実観測ではない」「必要な検査の欠落は自動検出されない」を、既存Goテスト近傍で同じScope/RunKeyの独立Expectedに照合。これは製品安全性成功の証明ではない。
2. **P1: 既存Scenario SSOTとSource Closure**：#1180/#2811に`required/discovered/attempted/verified`とCase世代・独立Oracleを渡し、#2828/#2830の**別PRのCI PASSを一回の実Target End-to-Endに合算しない**。Source実読込#2822〜24、Build byte#2595、Role Provider#2620/#2639の未整合はunknown。
3. **P2: 本番側の原子的実行・結果確認**：#1174/#1129にcurrent Principal/Grant/Period/Target/CAS→Commit/Receipt/Observerの同一作用境界、#2146に外部排他、#2516に実Consumer受領、#1130に部分完了/回復、任意EKIにはTask/Lease/Result bodyを担当させる。Direct Go/外部Host/EKIの選択でBusiness Meaningを変えない。

**停止線**：`RunPlan`は現時点ではSource/Role実消費、認可Grant、Observer効果完了、実DBの分散排他を自動で証明しない。既存の`blocked`／`completed`を正式`BusinessVerified`へ昇格しない。ここでは新RunPlan/Policy Engine/Host/Registry/正式DSLを設計しない。ACU-01〜08の独立Oracleを伴う同一実Target横断E2E、実DB/エッジ、Owner受理は**NOT RUN**。
