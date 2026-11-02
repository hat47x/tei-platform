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
