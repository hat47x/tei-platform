# スクリプト・利用者マクロの製品化と権限委任の実証計画（2026年11月）

- **状態**：開発計画案・未採用。個々のIssueと既存製品計画の判断境界を維持する。実装・試験・Ownerの受理をこの文書から宣言しない。
- **設計本文**：[マクロの権限委任と操作対象の接合](../design/SCRIPT_MACRO_AUTHORITY_AND_EFFECT_TARGET_BINDING_2026-11.md)（T-1〜T-61）。本文に仕組みを重複記述せず、ここでは**担当・順序・実証の受渡し**を管理する。
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

## 変更影響とArtifact公開の選択的再資格化（2026-11-02追補）

[設計第12章](../design/SCRIPT_MACRO_AUTHORITY_AND_EFFECT_TARGET_BINDING_2026-11.md)の**T-35〜44**は、Everything is DataのData/Rule/Query、MacroのAction/Grant、生成物配布、SHO/AI等Consumerの**現在のClaim資格**を同一版で追うための計画。新しいClaim Engine・Cache・Schedulerを創設せず、[#2582](https://github.com/hat47x/tei-platform/issues/2582)、[#2600](https://github.com/hat47x/tei-platform/issues/2600)、[実行プロトコルDraft #2636](https://github.com/hat47x/tei-platform/pull/2636)、[#2630](https://github.com/hat47x/tei-platform/pull/2630)、[#2604](https://github.com/hat47x/tei-platform/issues/2604)の既存証拠をMacro Consumerへ接続する。

| 段階 | 追加試験 | 既存担当・必要な実証 |
| --- | --- | --- |
| **S0：依存の説明** | T-35/36/37/42。Output値Lineageと非出力WHERE/EXISTS/Sort/Limit依存、Rule/Scope、依存欠落・誤「full」、無関係変更の局所維持 | #2387、Draft #2634、#2582/#2514 |
| **S1：対象と実現** | T-38。同じOperation ID/BasisでもMapping列/Storage改版でConsumer Inputs・実Provider依存が異なり得る | #2611、Draft #2632、#2598/#2600 |
| **S2：変更後の実行資格** | T-41/44。過去のEffect Receipt/承認履歴を保持しつつ、ABA/Grant失効後の現在起動・Commit資格を再照合 | #2499/#1174/#2582/#2516 |
| **S3：公開されたMacro一式** | T-39/40。Consumer用途別Expected ArtifactSet、Manifest内部整合、同世代公開、実Load/実Reader byteを分けて証明 | #2482、#2604、Draft #2630/#2633、#2502/#943 |
| **S4：全体の再検証費用** | T-43。F0/F1/S1/S2のfull/選択的/負例比較とJ0/J1/J2のClaim再確認費用。正しさ成立後に時間測定 | #2600、Draft #2636、#2018/#2514 |

**最小一巡**：同一の合成受注Macroを、(F0)固定Source/Context/Provider/Artifact版で全検証し、(F1)次の単独変更後に全検証、(S1)依存に基づき必要箇所だけ再評価、(S2)誤再利用を誘う負例として実行する。次を**同じInput/expected/Host/Consumer定義に固定**し、比較できなければ`not-established`とする。

1. Hidden WHERE thresholdだけ変更し出力JSONは変えないが、行membership Claimは再評価する。
2. 非表示Sort/EXISTS/Top-Nや、未列挙依存の変更は値Lineageと別の資格として扱う。
3. Mapping Storageまたは未選択Provider変更は、実依存・World/Artifact・Target適合への波及を個別判定する。
4. Macro本体が同じでもConsumer必要Member欠落、途中世代混在、実Loader不一致を検出する。
5. SHOの無関係な説明修正では、業務Effect/Library再ビルドと文書公開のClaimを局所化する。
6. Grant失効や開示Policy変更は過去のArtifact/Receiptを削除せず、**今後の実行・表示**の適格性だけを再評価する。既確定Effectを検証目的で再Submitしない。

**Evidenceの独立比較**：原Sourceと実消費byte／構造と受理意味／値依存と行集合・順序依存／選択したProvider・Context時点／実ビルドArtifactとConsumer全Member・公開世代／過去の実行・独立Oracle／現在Grant・開示の各列を持つ。過去`PASS`を今回の再実行`PASS`へ偽装せず、保持した履歴と今回の用途への再利用適格性を別に報告する。

**判別指標**：固定反例範囲でのfalse reuse件数（許されない省略）、false invalidation件数（証拠付き無関係なのに再実行）、実選択された必要Memberの漏れ、Target/Read/Disclosureの不明件数、requalificationの呼出し・費用、同一Host/入力でのcold/warmとedit-to-usable p50/p95。**再計算の正しさを検証した後に速度を測り、未測定の高速化は報告しない**。新たなProduction要件やGateを増やすのではなく、#2600/#2018の品質観測へ渡す。

**実証状況**：この追補は設計/Workstreamの接合案。#2632/#2633/#2634の実Go候補ファイルが存在することと、その正確なHeadでのGo/CI PASSを別に確認する。T-35〜44のMacro Producer→Consumer→Host→Result/開示の同一試行、外部Reader、性能測定、Owner意味受理は**NOT RUN**。他OwnerのDraftやCritical Path P0-a/P0-b・Gate A/B/Cは変更しない。

## 実CLI起点の最短Consumer縦断 M0〜M4（2026-11-02追加）

[設計第13章](../design/SCRIPT_MACRO_AUTHORITY_AND_EFFECT_TARGET_BINDING_2026-11.md)のT-45〜53を、**既にmainへ統合された実TEI CLIから開始する最初の接続実験**とする。[PR #2587](https://github.com/hat47x/tei-platform/pull/2587)はmerge commit `8464db0bbbac216ede15d3344e72b7e8bd90cb65`で統合され、[#2643](https://github.com/hat47x/tei-platform/pull/2643)のstatus更新を含む参照mainは`d8f5114f17793354bbab0c766190699ad2b5d60b`。従来の`9c9ef4d...`に基づく未統合候補の記述は履歴・設計の根拠であり、**現在のCLI実在性を示すものではない**。一方、統合済みはMacroや実DB/外部Host/Owner受理の完了ではない。

### M0：実CLIによるHeadless read-only調査（最優先・独立実施可能）

実行対象：mainの`reference/standard-crud/workspace`／Application`ledger`。テストの前に**固定checkoutと実行物**を確かめる。以下は既存CLIの構成から導ける**実行予定のコマンド**であり、この文書追加時に新実行した記録ではない。

~~~sh
go run ./cmd/tei commands
go run ./cmd/tei validate -workspace reference/standard-crud/workspace -json
go run ./cmd/tei operations -workspace reference/standard-crud/workspace -application ledger
go run ./cmd/tei artifacts -workspace reference/standard-crud/workspace -application ledger
~~~

- `commands`の`tei.cli.discovery/v0`は限定された`selected-read-only-commands`。`operations/artifacts`のdispatchがmainにあっても現在このCatalogには無い。**カタログの非掲載・Capabilityの不在・現在実行権限**を同一視しない（#2605）。
- `validate`のexit/status/`ok`/Warning/Source fingerprint、`operations`のOperations/Gaps、`artifacts`のManifest/Findingsを**同一Source版で**記録。Fingerprintが出ない・不一致ならSnapshot一貫性を主張しない。
- `artifacts`の`-out`はローカルディレクトリへ書く。上記read-only調査とは別のM1として明示した隔離出力先で試す。現在の`-out`時標準出力はManifestのみで、書かれた`manifest.json`中のFindingsをstdoutで読めたとはしない。
- M0の成功は、**実Commandの出力と不足をread-onlyで再現可能に記録したこと**。Operationの公開・Macro実行・業務効果やSHO納品を意味しない。#2587の合成既報のPASSを新たな同一試行へ転記しない。

### M1〜M4：同じSource/用途に対する段階別Consumer試験

| Packet | 接合点・必要な成果物 | Owner／禁止する飛躍 |
| --- | --- | --- |
| **M1 Artifact Reader** | M0と同じ版の`tei artifacts`を隔離ディレクトリへ明示出力、実Member/Manifest/Findings/必要Application Scopeを読み直し、公開世代とReader bytesを検査 | #2482/#2604/#2630、#2633。`-out`の完了≠一式の読取成功 |
| **M2 TEI Product SkillとTAI** | 同じ読取り専用CLI調査をSkillなし/ありで比較し、実Tool/Command、公開候補名、Source版・Warning/Unknownの差を測る。TAI無しTEI単独条件を保つ | #2605、[TAI #270](https://github.com/hat47x/tai-pleroma/pull/270)/[#271](https://github.com/hat47x/tai-pleroma/issues/271)/[#272](https://github.com/hat47x/tai-pleroma/pull/272)。Skill load≠Host権限 |
| **M3 Macroと実作用** | `Operation Surface`候補→typed Intent→選抜Provider/現在Grant→実Request/Owner Effect/Receipt。読取途中で権限失効するD2負例は専門実装へ | #2598/#2499/#1174、#2637。候補/Preview≠実DB更新 |
| **M4 SHO/EKI成果物** | 同一Version/SourceでSHOの説明・出典/保存、EKIの同Attempt Body/digest/実Reader取得を別観測。最終回答とRenderer/a11y/Tool/Log/送出の公開面を比較 | #2516、SHO #10、EKI #465/#409、TAI #272。ACK≠Body納品 |

**共有の受渡し票**：実CLI/Skill/Host/Consumer名・Version／Source/Meaning/Role/Operator/Plugin版／Function Context InstanceとScope／Operation候補・Gap／実Provider/Request/現在Grant／Artifactの必要Member・実Loaded byte／Provider Effect/Receipt／Output/Rendered DOM/ログ・Network/外部送出・実Reader／Expected/Observed/未観測/Owner判断。

**最小の異質な反例**：同じ最終回答でも外部Tool名が異なる／同じMarkdownでも実リンク先が異なる／read開始時には許可されていたが返却前に失効／EKI ACKはあるが同AttemptのBody未取得／Quote中の`trusted-script`を正式Roleとして誤採用。内部実装と外部に公開したScopeが違う場合、情報非干渉を最終Answerだけで判定しない。合法に公開するTool名/内部Readまで一律拒否する方式にはしない。

**完了条件と状態**：M0だけを先にCLI製品Sourceへの接続結果として報告し、続くM1/M2/M3/M4は別Ownerの実証を待つ。新しい一律横断Gateは作らない。T-45〜53、今回のM0〜M4実CLI・Skill・実Provider・SHO/EKI Reader統合は**NOT RUN**。mainへ統合した#2587の既存test実績は、その試験範囲だけで保持する。P0-a実PG、P0-b二Function、Gate A/B/Cは不変。

## M0実証票：Source IndexとTrusted受理の間の停止線（2026-11-02追加）

[設計第14章](../design/SCRIPT_MACRO_AUTHORITY_AND_EFFECT_TARGET_BINDING_2026-11.md)のT-54〜61を、#2587統合後のM0を実際に評価する際の**停止条件と受渡し票**に追加する。main`b1c019fec4f05ca65e3d7491608cb849ce523d29`でSourceを静的確認した。

**存在する二つの経路の区別**：

- `operations.SurfaceWorkspace`および`artifact.RenderWorkspace`は、`workspace.Load`の後に**非strict**`BuildSourceIndex`を使う。通常のLoadから渡された解析済み/追加管理ファイルのbytesは優先されるが、範囲外には安全なRoot付きファイル再読込みのfallbackがある。**毎回再読込みしているという記述にしない**。
- `sourceacceptance.LoadVerifiedWorkspace`、main統合済み[#2642](https://github.com/hat47x/tei-platform/pull/2642)のIssuer署名側は**strict**`BuildSourceIndexFromLoad`を使用。捕捉済みbytes以外へfallbackせず、別Root・未束縛importを受理しない。これはCurrent Host Grant・ビジネス受理を自動作成するものではなく、`ManifestIdentity`の確認・import closure・複数ファイルのatomic snapshotも別。
- [#2647](https://github.com/hat47x/tei-platform/pull/2647)が改善を試みる`trace`元Source読取りは、固定Root/サイズ上限の**一読取り**の保証と、複数Requestでの同時点Source一致を分ける。Open PRの挙動をmain採用済みとしない。

### M0-1 実測するコマンドと出力の一致（予定、NOT RUN）

既に[M0の実行コマンド](#m0実cliによるheadless-read-only調査最優先独立実施可能)を定義した。**その同じ一巡で**、次をSource/受理/実行の独立Claimとして記録する。

| 観測対象 | 判別する事実／不一致時の扱い |
| --- | --- |
| 実TEIバイナリ | `vcsRevision`/`vcsModified`、実OS/Go、実Workspace path・Application、固定checkout。`go run`などで埋込みVCSが欠ける場合は未識別とする |
| `validate -json` | exit/status、`ok`、Warning/Diagnostics、`worldFingerprint`、`sourceIndexFingerprint`、失敗なら両者の欠損を保持 |
| `operations` | `schemaVersion`、`worldFingerprint`、`modelRevision`、Ruleの版/指紋、Operations/Gaps/Conflicts/AuthorityLimits |
| `artifacts` stdout | `inputRevision`、`manifest.inputRevision`、各member、`complete`、`findings`（`-out`時stdoutはManifestのみ）。必要memberと実Readerは別 |
| 比較 | `validate.worldFingerprint`↔`operations.worldFingerprint`、`operations.modelRevision`↔`artifacts.inputRevision`。両値を得た場合に限って一致/不一致を示す。`validate.sourceIndexFingerprint`を`ModelRevision`と**当然同一の値**とは扱わない |
| M0→Trusted | `LoadVerifiedWorkspace`のstrict/index/read acceptanceは**M0で未実行**。表示済み`worldFingerprint`やOperation候補だけで`accepted/authorized`へ変換しない |
| M0→Provider/Consumer | 実Payload/DB/Receipt、署名済み配布物、SHO/EKI実Read、OS Network Egressは**M0では未実施**。未知を`0件/許可済み/完了`へ写さない |

同じ値が返る場合も、独立した複数CLI Processを通じた原子的Snapshotの証明ではない。単独の管理ファイル変更・Rule/Role改版・SourceのABAを、`View=同じ`だけでskip可能にしない。まず再現可能な固定Fixtureで出力を観測し、別の変更後Sourceを使うのは**新しい試行**として分ける。

### M0-2 新しい8反例の担当と実測順序

| 試験群 | 担当・次の実行証拠 | 優先度 |
| --- | --- | --- |
| **T-54/55/58** | #2605・#2502が同じ実CLI一巡で、read-only derivation成功、非解釈管理ファイル変更、CLI間Source driftの診断を観測する。Fixtureの予測と実結果を同じ票で対照 | **M0で優先** |
| **T-56/57/59** | #1017/#2642/#2647でstrict捕捉・import拒否・traceのRoot/個別読取りを検証。実Provider/OS Source側の不足をMacroの合格にしない | **M0並列／Owner別** |
| **T-60** | #2638/#2605、TAI #270がRole/Plugin依存Pin、Skillの実参照bytes/Tool、現在Grantを保持。read-only Skillと実行マクロを分離 | **M2** |
| **T-61** | #2482/#2604/#2630/#2633のSet/Publisher/Reader資格をM1、Host Effect/ReceiptはM3で受け取る | **M1→M3** |

**M0受理の最小定義**：mainの実バイナリを固定した同一Workspaceで、成功/Warning/Gap/Findings/Ruleと各Fingerprint・Revisionを記録し、既知の`sales.note`／`sales.order`負例を**独立した期待値と比較**できること。4つのコマンドを実行してexit=0だっただけでは合格にしない。曖昧なSnapshotや同一値だけの一致は、明示した不明残余にする。高機密な元Sourceや診断を公開Issue/AIへ無断転載しない。

**改版・制約**：テストの説明は現在のSource-static観測に基づく。実際のM0試行・strictと非strictの統合照合・import/ABA/Source driftの注入・本番Host/Provider/OS/SHO/EKI・Owner受理はすべて**NOT RUN**。#1119の既存P0/Gateは変更せず、新しいCLI Contract、全局Source Index Engine、実行Host、横断承認Gateを創設しない。

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
