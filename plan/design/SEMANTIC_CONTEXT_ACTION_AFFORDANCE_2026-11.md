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
