# スクリプト・利用者マクロの製品化と権限委任の実証計画（2026年11月）

- **状態**：開発計画案・未採用。個々のIssueと既存製品計画の判断境界を維持する。実装・試験・Ownerの受理をこの文書から宣言しない。
- **設計本文**：[マクロの権限委任と操作対象の接合](../design/SCRIPT_MACRO_AUTHORITY_AND_EFFECT_TARGET_BINDING_2026-11.md)（T-1〜T-18）。本文に仕組みを重複記述せず、ここでは**担当・順序・実証の受渡し**を管理する。
- **親計画**：[横断 Critical Path #1119](https://github.com/hat47x/tei-platform/issues/1119)、[大量AI並列開発 #2493](https://github.com/hat47x/tei-platform/issues/2493)、[開発全体計画](../MASTER_PLAN.md)。既存Gate A/B/C、P0-a/P0-bの資格・順序に介入しない。
- **製品Owner**：[Script/User Macro #2502](https://github.com/hat47x/tei-platform/issues/2502)、[限定委任 #2499](https://github.com/hat47x/tei-platform/issues/2499)、[実行Policy #2504](https://github.com/hat47x/tei-platform/issues/2504)。

## 目的と範囲

TEIを利用する開発者・組織・一般利用者・外部AIが、**意味上のData/Function/ActionをScript/DSLで組み合わせ**、同じ実行能力をCUI・標準UI・外部Hostから利用できるようにする。その際、開発者の広いScript自由度と一般利用者の限定権限、組織固有の承認手順、業務上の操作対象と実効果、非公開情報の開示制限を両立する。

従来のX-TEA由来SmartDB/汎用機能は、業務専用機能タイプに閉じず、**Semantic Context→Action適用条件→必要な実行契約→Consumerごとの表示・起動手段**として一般化する標準Pluginの候補とする。標準UIは一般のData/Script構造を表示し、編集時はSourceへ正当に戻せる範囲に限定する。専用UIを製品拡張の前提にしない。

初期の小さな試験は**製品スコープ上限ではなく、保証の境界を見つけるための単位**である。安価なトークンの利用増加を前提に、異質な要求や反例の探索を並列化する。一方、実権限・実DB作用・正式業務受理の根拠は担当の試験と人間判断に帰属する。

## 担当と権限

| レーン | 担当Issue | 具体的な入出力 | 当レーンが判断しないこと |
| --- | --- | --- | --- |
| **A：Script/DSL標準構造** | #988、#1124、#1824 | 原Source→構文/依存/Source span→一般Inspect→標準UIの表示/編集提案 | 実行Grant、実Effect、専用構文の正式採用 |
| **B：Context/汎用Action** | #2506、#2507、#1127、#2500、#2598 | Subject/Relation/Result Slice→Action候補→typed Intent→実Request対象の接合 | 新しい業務意味の受理、表示された候補の実行許可 |
| **C：Macro製品体験** | #2502 | 開発者Script・個人Macro・組織共有・Trigger・改版・CUI/標準UI表示の体験 | 業務承認手順の固定化、別のScript Runtime/Source正本 |
| **D：業務判断と委任** | #2499、#2386、#951/#1118 | 任意Workflow Decision→現在Issuer/Grant→実行時に限定するAction/対象/期間 | Business DecisionをHostの許可とみなすこと、共通Issuerの導入 |
| **E：実Effectと回復** | #1174、#1129、#1130、#1131/#1180 | 実Payload/Owner対象→Tx/Receipt→別Read→不明/部分成功/再開 | Receiptだけで全業務完了・元Result忠実再生を断定すること |
| **F：実環境の能力・秘密** | #2504、#2515、#2020、TAI #232 | Host/OSにおける実到達API、Raw/Secret、egress、結果開示、Target別unsupported | Tool説明やUI非表示をOS強制と呼ぶこと |
| **G：外部成果物・複数Target** | #943/#2390/#1370、#2077、EKI #465/#409 | Script承認版↔実Load閉包、外部Host、委任Taskと実Body受領 | TEI専用Host/Go依存の外部生成物、Task完了から業務完了の推定 |

この表のOwnerはIssueで定義済みの責任境界への**引継ぎ先**であり、新しい正本を意味しない。共通データ構造やScript構文を選定する場合は、それぞれの既存OwnerでConsumer反例を検証してから採用判断へ進む。

## 実証の順序と並列実装

### 既存P0：優先維持

[#1174](https://github.com/hat47x/tei-platform/issues/1174)の実PostgreSQL確定後Result/Receipt/独立Readと、[#1129](https://github.com/hat47x/tei-platform/issues/1129)の同じData Ruleの二Functionへの適用を優先する。Macro側の検証結果でこれらの通過を代替しない。部分成功、Rollback/ACK不明、歴史的結果と現在値の区別を保持する。

### S0：read-onlyなAction/Script発見（独立着手可）

同じ受注明細L1と、独立したContext D1/D2を合成Sourceとして用い、まずCUIで以下の結果を比較する。

1. **Contextの事実**：粒度・ID・Relation Role・選択epoch・取得範囲・部分/全件・Source版を示す（#2506）。
2. **Actionの適用候補**：必要なData Shape・Operation・業務対象、現在利用できる実装と未解決の理由を示す（#2507）。
3. **Script構造**：Text/Source span/型とContext参照/依存・Effectが既知か未知か、実Load対象版を検査可能な範囲で表示する（#988）。
4. **UI表示との独立**：標準表UIまたはRead-only Studio Inspectへ同じ意味と診断を表示する。標準UIから実権限を発行したとは扱わない（#1124/#997）。

**成功の範囲**：同じAccepted/合成Sourceに対する複数Consumerのread-only適用診断が比較可能。実Runner/Owner Effect/承認は未実証として残す。対象はT-14/16のread-only部分。

### S1：限定更新のIntent→Target一致（P1、既存試験を再利用）

[#2500](https://github.com/hat47x/tei-platform/issues/2500)の部分View、[#2598](https://github.com/hat47x/tei-platform/issues/2598)の古い選択Context・逆写像、[#1174 AT-1/P1-Payload-2](https://github.com/hat47x/tei-platform/issues/1174#issuecomment-6572078615)の実Request対象負例を、一つの対象束縛票で照合する。

- **正例候補**：公開されているL1.labelだけをPATCH。非公開Costは開示せず、Host/Ownerが現在の必要条件を検査できる場合だけ実行できる。
- **負例候補**：部分Viewを全体Replace、古いContextからの別L1への誤配賦、OperationIDが1001でも実Payloadが1002の対象すり替え。
- **過剰拒否の負例**：D1のL1限定更新はD2の部分取得に無関係なはずなのに、全Context一律失効で許可候補が消えてしまう。
- **別の意味で合法な正例候補**：受注1001から関連明細A/Bと在庫へ作用する受理済み業務Action。`実対象=意図ID`の文字列一致を単純な全操作規則にしない。

実DB試験はOwner #1174が担当し、Macro固有のGrant試験はS2へ分ける。S1で契約が足りなければ`unsupported/unknown`とし、正式業務Actionを仮定しない。

### S2：任意Workflow→限定Grant→Action実行（P1）

**異なる組織Workflow**（本人利用、複数人合議、期限付き代理）を同じScript/Service Contractへ接続する。承認したMacro版、Issuer権限、委任対象のFROZEN/LIVE、子Serviceの具体対象、現在Grant・時点・失効の証拠を独立に確認する。

- 合法なOrder A委任で特権Serviceを利用可能。
- Order Bへの子引数すり替えは、その対象へのGrantがない限り実効果を拒否。
- `approved=true`の自己申告、旧Script Load、異Tenant・期限切れ・委任の上限超過を拒否。
- 合議は組織Policyに従ってAction×対象×期間を合成。固定の管理者承認手順は組み込まない。
- 高い権限を持つサービスが合法に代理実行する選択肢を残し、利用者とサービスの単純な権限積集合を万能規則にしない。

S1の実対象試験を再利用し、第二の認可エンジンや独立Issuerを新設しない。Issue #2499/#1174で本番に近いProviderの資格が得られるまでE2E合格を主張しない。

### S3：標準UIとMacroライフサイクル（S0〜S2と部分並行）

**標準構造からの投影**を先に検証し、必要に応じてUI Adapterを追加する。Script Sourceを編集した際のParse/Inspect更新と、標準UI側の一意な変更を元Sourceへ戻す処理を対照する。派生/曖昧なScriptノードはRead-onlyと元SourceへのNavigateでよい。

同じMacroを個人保存、組織公開、定時実行へ移した場合に、公開のDecisionと実Grant、Load版・Runtime実能力、開始/完了、初回Result/現在値/独立Observation、開示資格を別欄で説明する。旧Script版・Rule変更時の再承認の実手順は組織の業務モデルが決める。標準のScript編集以外に専用承認画面や新ノードUIを必須としない。

S3におけるStudio/GUIの動作をもってS1/S2の実Provider許可・Effectを合格扱いにしない。

### S4：外部Host・Agent・配布（対象領域を拡張）

Java/.NET/Wasm/Process等の外部Hostへ同一Script/Action/Contractを接続できるか評価し、実ロード依存閉包・現在Host認可・Targetごとの実行隔離を区別する。AI Harnessへ送るScript/Context/Evidenceは保持Productが用途限定で投影し、Raw/Secretへの実到達と外向き経路は#2515/TAI #232の実OS試験で確認する。EKI Task受理・Claim/ACK・Body保存・元依頼元の受領は別段階とする。

**対象拡張の例**：業務受注/在庫、組織代理、科学/数理Data加工、ドキュメントの構造編集、バッチ帳票、IoTイベント、別製品による作業代行。すべての領域で同じUI/Script言語/Host保証を要求せず、意味Actionと必要なAdapterの共通性を測る。

## Everything is DataとMacroの接合（2026-11-02追加）

[横断Issue #2619](https://github.com/hat47x/tei-platform/issues/2619)で提案された「標準データ構造に意味役割を与える」構成を、利用者マクロの独立Consumerとして検証する。詳細は[設計文書10章](../design/SCRIPT_MACRO_AUTHORITY_AND_EFFECT_TARGET_BINDING_2026-11.md)。特に、単純なScript表現だけを増やすのでなく、**Source/構造・意味Role、Function Data Contextの明示Mapping、汎用Action、現在Authority、Owner Effect、結果開示**の対応を残す。

| 計画段階 | 追加する成果と試験 | 既存主担当 |
| --- | --- | --- |
| **S0：read-only基盤** | T-19〜21/26。Markdown表・式・DSL→標準構造とRoleを比較、意味参照/Source位置・Operator版・未解決を示す。明示Contextの範囲外を暗黙取得しない。テスト表のExpectedと説明文を区別する | [#2620](https://github.com/hat47x/tei-platform/issues/2620)、[#2621](https://github.com/hat47x/tei-platform/issues/2621)、[#988](https://github.com/hat47x/tei-platform/issues/988)、[#2506](https://github.com/hat47x/tei-platform/issues/2506) |
| **S1：実対象** | T-22/24。3層→2層Derived DatasetのLoad/Save、無変更更新0、ID付き限定PATCHと非一意な集計編集を分ける。作成者Context/RowIndexを実行者へ暗黙継承しない | [#2622](https://github.com/hat47x/tei-platform/issues/2622)、[#2500](https://github.com/hat47x/tei-platform/issues/2500)、[#2598](https://github.com/hat47x/tei-platform/issues/2598)、[#2597](https://github.com/hat47x/tei-platform/issues/2597) |
| **S2：委任・作用** | T-23。Read scope、Authority scope、Effect footprint、Disclosure scopeの非一致を正例/拒否例の両方で検証。合法な関連更新を不必要に拒否せず、未委任子作用とRaw開示を拒否する | [#2499](https://github.com/hat47x/tei-platform/issues/2499)、[#1174](https://github.com/hat47x/tei-platform/issues/1174)、[#2504](https://github.com/hat47x/tei-platform/issues/2504) |
| **S3：Studio標準投影** | 同一Role/SourceのCUI・標準表/Tree/Diff・SHO設計説明の二Consumer以上を比較。Sourceへの一意な逆写像、曖昧な箇所のread-onlyを分け、説明・編集・受理を同じ状態にしない | [#2620](https://github.com/hat47x/tei-platform/issues/2620)、[#1124](https://github.com/hat47x/tei-platform/issues/1124)、[#2617](https://github.com/hat47x/tei-platform/issues/2617) |
| **S4：Provider/Domain差** | T-25。RDB系とAPI系の同意味Intentの差、native SQLの明示Binding。受注/在庫に加え文書または科学データ処理等の異なるDomainで標準OperatorとMacro役割の再利用・特別実装量を比較 | [#2622](https://github.com/hat47x/tei-platform/issues/2622)、[#951](https://github.com/hat47x/tei-platform/issues/951)、[#2493](https://github.com/hat47x/tei-platform/issues/2493) |

**実証方法の原則**：既存の#2617合成14件や#2613合成20件などを、Macro製品のE2E合格へ自動昇格させない。試験済み実装のSource・対象版・Consumerと、将来Macro側が使う実データを分けて確認する。標準構造を採用する価値は、固定Core kindや専用GUIを増やさず、意味契約の再利用、明示Mapping、独立Expected、診断の一貫性を確かめられるかで評価する。

**現状**：T-19〜34、実Script/Context/Provider結合、Studio・SHO・実Hostでの製品受入はNOT RUN。#2619〜#2622はいずれも横断設計案であり、名称（Data Operations等）、新Core形式、DSL構文、Owner意味受理は未採用。既存#1119 P0-a/P0-bやGateの基準を変更しない。

## 実経路と取得証拠の受渡し（2026-11-02追補）

[設計第11章](../design/SCRIPT_MACRO_AUTHORITY_AND_EFFECT_TARGET_BINDING_2026-11.md)のT-27〜34は、[#2611](https://github.com/hat47x/tei-platform/issues/2611)、[#2622](https://github.com/hat47x/tei-platform/issues/2622)、[Draft #2627](https://github.com/hat47x/tei-platform/pull/2627)、[Draft #2628](https://github.com/hat47x/tei-platform/pull/2628)の既存結果をScript/MacroというConsumerへ接続する。新しい共通実行機構の作成は要件に含めない。

| レーン | 追加する判別対象 | 既存Owner |
| --- | --- | --- |
| S0 | T-27/29：宣言した取得範囲、実Request、観測できた実読取り、返却値を別に説明する | #2506/#2622、Draft #2628 |
| S1 | T-30/31：同じ意味データの複数Alias、ReplicaとPrimaryの時点・対象を区別 | #2611/#2598/#1174、Draft #2627 |
| S2 | T-28/32：正当な内部検証用読取りと不要な取得、経路切替時の現在委任資格・確定状態を分離 | #2499/#2504/#1174/#1130 |
| S3 | T-34：利用者へ見せる値、内部読取り、実際の出力先を区別 | #2515/#2516、TAI #232 |
| S4 | T-33：標準Data OperationsとNative SQLの演算順序・NULL・bag・集約の適合を比較 | #2387/#2622/#2514 |

**横断受渡しの列**：Semantic SubjectとAction版／Function Data Contextの取得宣言／Provider・Endpoint・Mapping・Purpose候補／選択した実経路・要求／実アクセスを観測した範囲／Result Sliceの時点・完全性／Operation Intentと実更新対象／現在のGrant・Commit・Receipt／開示の範囲。候補があること、実際に選んだこと、呼び出せる権限があること、業務作用を確定したことを別の判定として残す。

**初期作業**：同じ合成Actionに対し、狭い取得、同一結果を返す余分な取得、古いReplicaの読取り、許可されない別経路への切替を別試行として比較する。既存の局所参照試験は再利用するが、それぞれの合格をMacro製品の統合合格へ移さない。Provider内部アクセスの観測が不足する場合は、当該保証を不明として記録する。

既存#1119 P0-a/P0-bとGateを維持し、各レーンは並行調査できる。**T-27〜34、実Macro・実Provider接続、実OS/開示・Owner受理はNOT RUN**。実装や正式契約の採用は別判断。

## テスト証拠の受渡し：1ケースにつき別の保証を保持

| 試験証拠 | 最初の取得先 | 後段へ渡せる範囲 |
| --- | --- | --- |
| 元Source/Context/Action候補 | #2506/#2507/#988/#2598 | この入力/版で説明できる候補。現在の実行許可とは別 |
| Script承認版と実Load集合 | #2502/#943/#2390 | 当該実行物の識別。業務意味受理・Service作用の正しさは別 |
| 委任Decisionと現在Grant | #2386/#2499/#951 | 許可された操作・対象・期間・主体。画面表示とPayloadを別確認 |
| 実Payload/Tx/Owner Effect/Receipt | #1174/#1129/#1130 | 実際に確定した対象/作用と不確定部分。全業務完了を推定しない |
| 独立Read/履歴Result/現在表示 | #1131/#1174/#2516 | 特定時点・Scopeの観測。昔の回答の再現性と現在の開示を分離 |
| 実Host/OS隔離・外部送出 | #2504/#2515/TAI #232 | 試験したTarget/資格と送出先の範囲。別OS・別Harnessへ一般化しない |

結果の報告は`固定revision・合成／実Provider・試験したInput/Target/Host/Oracle・Observed/Expected・未実施／不明・Ownerの判断`を明記する。AIによる候補多数決・CIのPASS・Issue数だけで正式な受理や本番効果を主張しない。

## 進行・衝突調整

- 要件探索、異質な利用例、Read-only合成Fixture、独立Oracle、標準UI候補は並列に展開する。
- 共有Semantic型、Service Contract、Grant/Receipt形式、Source/Plugin Lockの変更は、該当Ownerと同じ版に照合した後にDraftで扱う。新しい第二正本やHostを作って競合を解消しない。
- 共通ファイルを変更する際は、先に進行中Draft PRの変更範囲と現行mainを確認し、別レーンの変更を上書きしない。
- 既存の作業票・検証・A0〜A4による人間判断境界に従う。**組織の権限発行・業務意味の採否・本番配備は設計やテストの自動結果から昇格させない**。
- 本計画は新しいGateを追加せず、既存#1119のP0と#2493の広域探索を両立するための**参照と作業配分**として運用する。

**現時点の達成**：IssueとSource/既存Referenceの照合、設計の更新、開発計画・反例・責務の文章化。**未達**：S0〜S4の本計画に沿ったTEI製品E2E、実Script Loader/Issuer、正式PostgreSQL委任試験、StudioとCUIの二Consumer実証、OS/Network隔離、外部EKI納品、Owner正式受理。既存レーンの限定Synthetic試験は各Issueの記録にとどめ、横断PASSとして合算しない。
