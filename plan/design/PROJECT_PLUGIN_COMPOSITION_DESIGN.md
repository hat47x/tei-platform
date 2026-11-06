# Project Plugin Composition — 案件のアプリケーション構成を再利用するための設計候補

- **状態**: 評価待ち（2026-10-08）。設計整理と実証計画であり、production schema / Resolver / 新たなProduct Requirementの採用ではない。
- **管理Issue**: [#1016 Application Composition Profile](https://github.com/hat47x/tei-platform/issues/1016)
- **役割**: 複数の責務を持つPlugin群を、Project / Applicationごとに選択・接続・更新するときの境界を整理する。実装構造全体の正本は[Architecture](../../docs/architecture.md)、採用済み判断の正本は[ADR](../adr/README.md)であり、本書はどちらの代替でもない。

## なぜ新しい整理が必要か

インフラ接続Plugin、Web/スマートフォン向けUI資産、業務仕様テンプレートが揃っても、「どの契約にどの実装を接続するか」「環境ごとに何を変えるか」「標準構成の更新が案件の独自部分へどう影響するか」はProjectごとの作業として残る。

この不足を埋める候補が**Application Composition Profile（アプリケーション構成プロファイル）**である。各機能を再実装する新しいFrameworkではなく、既存の資産と決定権限をつなぐ設計時の構成表現として検証する。これによってWeb業務アプリと、Native Canvasを持つ非業務アプリの双方で、設計判断と接続作業を減らせるかを調べる。

## 責務の地図は固定の依存階層ではない

次の分類は「何を再利用するか」を探すための責務区分であり、L0〜L10の実行順序、上下の所有関係、必須Pluginの列挙ではない。特に最小Kernel自身を「標準Plugin」と呼んで循環依存を作らない。

| 責務群 | 提供する資産の候補 | 既存境界と注意点 |
| --- | --- | --- |
| 最小Kernel | 解決・ロード・Authority境界を支える最小Host | Plugin以外の不変部分。固定enumの増殖は避ける |
| 言語・意味 | DSL、Operator、Semantic Extension | [#988](https://github.com/hat47x/tei-platform/issues/988)、ISSUE-000077。表層のFrameworkに意味を依存させない |
| 実行・処理 | Application Action、同期・非同期処理、イベント、Scheduler | 既存Driver・Runtime・Capability Outcomeを先に再利用 |
| インフラ・データ接続 | Database、Object/File Storage、Queue、Cache等のProvider | physical realization / Environment bindingを業務意味の正本にしない |
| 外部サービス連携 | REST/GraphQL、外部IdP、外部システムAdapter | 接続先の認証や権限を自動取得しない |
| 汎用アプリケーション機能 | 状態遷移、通知、検索、集計、ジョブ、履歴管理 | 機構と業務固有の手順・規則を分離する |
| アプリケーション構造 | 標準業務Web、オフライン同期、Document Editor、共同編集、イベント駆動 | **Architecture Profile候補**。Target/Frameworkとは別の選択軸 |
| ドメイン仕様 | 受注・承認等の業務素材、知識整理・ゲーム・シミュレーションの領域固有素材 | [#998](https://github.com/hat47x/tei-platform/issues/998)は業務仕様Template。非業務DomainもCore enumへ固定しない |
| UI・操作体系 | Form / List / Canvas、Navigation、Theme、Design System、a11y | [#997](https://github.com/hat47x/tei-platform/issues/997)の**UI構成キット**とその部品。Kitそのものを単一Runtime Pluginにしない |
| Target・Framework適合 | Browser / Mobile / DesktopとReact / Vue / Flutter / Wails等のBridge | Target群・Framework・実装言語・artifactを別属性とする |
| 案件の構成・差分 | Application Composition Profile、Project Template、Project Extension | Projectごとの正本、構成の選択、Upstreamとの相違を扱う |

運用・セキュリティ・品質・開発支援・配布は、上の表の最上段や最下段に押し込まない横断領域である。認証・認可・監査、Logging / Observability、Build / Deployment、Test / Conformance、Editor・AI Tooling、多言語化・Accessibility等には独立Plugin候補がある。ただし、すでにある機構を新たなPlugin名で重複実装しない。

### 分類は少なくとも四つの直交軸を持つ

- **責務**: Semantic、Runtime、Infrastructure、Application、Domain、UI、Toolingなど。
- **資産の種類**: 契約、実装、Plugin package、構成Profile、Starter / Template、Project-owned Source、生成物。
- **利用段階**: Authoring、Build、実行、運用（複数にまたがってもよい）。
- **適用範囲**: TEI共通、Target/Framework共通、業界・業務共通、組織共通、個別案件。

これらの分類軸を単一の「Plugin階層」やCapabilityKind固定列挙へ変換しない。配布単位と意味上の所有者も一致するとは限らない。

## 異なる再利用資産を混同しない

| 資産 | 何を決めるか | 決めないこと |
| --- | --- | --- |
| **Plugin / Provider** | 特定Capability / Contractの実装、物理的実現 | 案件の業務規則、他Pluginの選択全体 |
| **Application Architecture Profile** | offline / document-editing / event-driven等の実現構造、必要なPort群 | WebというTargetだけから導かれる唯一の設計 |
| **Application Composition Profile** | あるProject / Applicationが採用する資産、参照、接続、適用制約 | Canonical Acceptance、実行権限、独立した第二Resolver |
| **UI構成キット** | Target/FrameworkのUI Host、操作部品、Starter、拡張点、UI Layer | Business meaning、Server側のAuthority |
| **Business / Domain Template** | 再利用可能な意味・制約・手順の候補 | 採用済みの案件定義、Renderer |
| **Project Template** | 初期ソース・構成を立ち上げる素材 | 生成先の継続的な正本・無条件のUpstream追従 |
| **Project Extension / Overlay** | 案件固有の設定、実装、UI差分 | 共有資産へ逆流する無断の意味変更 |

同じパッケージから複数種類の資産を配布することは許容できるが、配布形式が同じという理由で同一のAuthorityやLockへ畳まない。

たとえば「承認」の対象・承認者・状態遷移条件はBusiness meaningであり、汎用の状態遷移実行器や通知配信器はApplication機能である。「Undoボタン」はUIだが、永続的な取り消し・競合解決の意味はApplication／Domainに属する。

## 案件ごとの構成は何を宣言するか

初版では新しい独立manifest grammarを決めず、**既存のSource / Plugin Lock / Semantic Lock / Environment bindingから導出できない情報だけ**を洗い出す。次は必要情報のチェック表であり、実装済みschemaではない。

| 候補項目 | 説明 | 現時点の扱い |
| --- | --- | --- |
| Project / Application identity | どの案件・Applicationの選択か | 既存の正本を参照。file pathだけでidentityを決めない |
| Required Contract / Semantic | 必要な機能・意味・作用域 | #951 / ISSUE-000077の契約・照合を再利用 |
| Selected Provider / implementation | どの実装を選んだか | scope、適合性、Authority、exact version / fingerprintを検証 |
| Target / Framework / Architecture | Web等の実行先、React等のFramework、offline等の構造 | 別々の選択軸。別Targetへの暗黙fallbackをしない |
| UI Kit / Domain Template | 表示構成の候補、意味の候補 | UI KitのResolver、TemplateのAcceptance境界を尊重 |
| Project-owned delta | 個別案件が変更した領域 | 更新時に保持し、衝突を検知する |
| Environment binding reference | 実DB、外部API、秘密情報・権限等の参照先 | Environmentに所有させる。Secret値をProfileに格納しない |
| 接続の局所名・項目の対応付け・パラメータ | 利用側の要求をどの提供元へ割り当て、どの値をどの項目へ渡すか | 業務の意味を持つ対応付けは受理済みの意味の側に置き、Profileは割当てと技術的な調整に限る。詳細は[プラグイン間の依存と入出力の対応付け](PLUGIN_DEPENDENCY_IO_MAPPING_DESIGN_2026-10.md) |
| Provenance / version / fingerprint | どの元資産・変更・互換性判断を採用したか | 既存Lockへ戻して検証。版番号だけでは同一性を保証しない |

### 既存の解決とAuthorityに委譲する

1. Projectのauthoritative Sourceと現在の採用済み意味、明示的な構成候補を読み取る。
2. Semantic契約・Capability契約・Provider適合性を**それぞれ既存の解決境界**で照合する。複数候補の暗黙の勝者選択はしない。
3. UI Kit、Application機能、Environment bindingを各所有者が宣言したPort / Extension Pointへ接続する。UI Kitの`extends / provide / replace / decorate`（#997）を全Pluginの上書き演算へ拡張しない。
4. Application / Target / Framework / runtime environmentの整合を検査する。未解決なら、根拠・所有者・不足条件を出して停止する。
5. 採用した論理契約・実装・artifactと実際の依存を、可能な範囲で既存Lockと追跡可能なReferenceに結びつける。
6. Template更新・Plugin更新・案件差分が重なれば、変更の影響を`unchanged / applicable / conflict / unresolved`として**検証候補**に分類する。自動採用したことにはしない。

これは設計時の構成判断であり、Runtimeの最新Permission / Resource / Leaseの確認を代替しない。採用済み業務意味をUIやProfileが変更することもない。`Profile selected != meaning accepted != execution authorized`を維持する。

## 継承・組合せ・交換の境界

初期の基本は**明示的な合成（Compose）**と、契約を満たす実装の**交換（Replace）**、公開Extension Pointへの**追加（Extend）**、許可された設定の**変更（Configure）**である。

Profile間の派生は候補として考えられるが、Profileの`extends`、UI Kitの`extends`、Plugin dependency、Business Templateの採用・差分は異なる関係である。これらを一つの多重継承グラフや「後から読んだものが優先」という規則にしない。UI Kitの単一親chainは[#997のReference](../../reference/ui-customization/composition/README.md)の範囲にとどめる。

同一slotを二つのAddonが交換する場合、同名別artifact、別Framework、期待する旧実装の不一致、契約の意味的互換性の不明などは、明示選択・再検査が必要な衝突である。ScopeをまたぐProvider fallback、AIが作った未確認の業務規則の自動Acceptance、UI側の認可変更は認めない。

## Web構成キットでの要求経路と認証方式

Webアプリケーションでは、UI構成キットだけでなく、HTTP要求・応答、経路別の保護、認証方式の選択・共存をつなぐ構成規則が必要となる。この部分は[#1043](https://github.com/hat47x/tei-platform/issues/1043)と[Webアプリケーション構成キットの設計候補](WEB_APPLICATION_COMPOSITION_DESIGN.md)で評価する。構成プロファイルはWeb構成を**選択**するだけであり、Plugin Providerの解決やRuntime Authorityを重複管理しない。複数認証方式の共存を、業務権限の自動統合と混同しない。

## 二つの実Consumerで試す

**A. 受注明細を持つWeb業務Application** — 既存`order_entry`を用い、Semantic / Data / Action / Validationは既存のAccepted Source、UIはWeb/React Kit、DBと外部接続はEnvironment Bindingを使用する。案件固有の右側編集PanelはProject UI Layerに保持する。UI Kit更新で業務上の識別子やValidationの意味が変わっていないことを確認する。

**B. Native Canvasを持つ知識整理Application** — [SUI #993](https://github.com/hat47x/tei-platform/issues/993)のDocument / Card / Native React Canvasを維持する。Drag中の状態はNative UIに置き、Drop時だけApplication-owned Actionへ接続する。SUIのDocumentをTEI Canonical objectへ強制変換せず、Framework共通Form構造も要求しない。

両例で計測するのは、Project構成に必要な記述・編集箇所、重複した意味、手動接続、AIへの追加質問、Upstream更新で起きる再作業、実行・テストの必要条件である。**第二Consumerがあるというだけで共通構成機構の十分性が立証されたとはしない**。

## 実証の順序と非目標

**E0**: 本書による責務分類と既存構造へのMapping。AI向け要件整理の候補は[構成要件整理プロンプト](../../reference/application-composition/requirements-prompt.md)。この段階では設計・文書上の整理のみ。

**E1**: 既存の`order_entry`とSUIのSource / Plugin / UI / Environmentへの参照をread-onlyに列挙し、何が構成として宣言済みか、不足する記述は何かを明示する。

**E2**: 追加schemaが必要だと立証された最小部分に限り、差分・不適合の診断と二つのConsumerでの検証を行う。synthetic test、actual runtime、production adoptionを別Evidenceにする。

- 全候補のPlugin化、全Targetへの実装、巨大なUI/Workflow基盤、新しいCapability enum、新しいResolver、独自Lockの並行開発は非目標。
- Domain固有の意味（SUI・ゲーム等）をTEI Coreに移すことも非目標。
- 既存機構で足りると分かった対象は新Issueへ分割しない。必要な機構だけを#1016で評価し、隣接する#997 / #998 / #951 / #943 / #988 / #993 / GitHub Issues000040の既存担当へ引き渡す。

本書の評価を経て、Application Composition Profileが実際に独立したSource表現を要するかを決める。採用するならADRを別途作成し、Project-owned Source / Plugin Lock / Semantic Lock / Target realizationを第二正本化しない最小境界に限る。

<!-- tei-project-extension-choice-handoff-20261106 -->
## 2026-11-06 追補：案件の要求からPlugin要否へ進む最小判断（設計・NOT RUN）

拡張方式そのものの比較正本は[Runtime/Target Binary Composition設計 §14](RUNTIME_TARGET_BINARY_COMPOSITION_DESIGN.md)に委譲する。本節は**Application Composition Profileが要求を取り込む時点**に必要な三段階選択の入口だけを定め、Plugin/Host/Contract/Buildの第二Resolverを作らない。

### 既存の構成操作を先に試す

| 案件からの要求 | 最初に評価する手段 | 独立したPlugin/Implementation選択を要求する条件 |
| --- | --- | --- |
| フィルタ・集計・導出計算を追加 | 既存Operator、純粋Library/明示Data Operations | 独立版・意味・効果を持つOperator Contractを複数Consumerが必要とする |
| 保存前に業務規則を追加 | Project OwnerのAccepted Business Rule/Procedure | 別Owner/Targetに独立配布する共通能力が実例で成立した場合のみ |
| DB接続先・実SQL方言を変更 | 意味上のFunction/Relationを保ち、Environment/Provider/SQL Assetを明示Binding | 実Data Provider能力・実方言・Scope・認可/効果が既存実装で満たせない |
| 画面の独自部品を追加 | #997のKit公開点・Project UI Layer | 型付きRenderer/Target Host Bridgeに必要な実Consumer契約がある |
| 生成された業務コードへ案件の処理を追加 | #2090で宣言されたExtension Point/Project-owned Source、またはAccepted Rule/Function | 別Consumer向けに独立配布・独立更新する実装契約が必要な場合 |
| Go/Desktop・外部Targetへ展開 | 既存Target Generator/Build Compositionを参照 | 固有Toolchain・ABI・Host/Target能力・Artifactの独立実現が必要 |
| 別Processで外部製品を呼ぶ | 既存のService/Transport Contractと外部Hostの能力を確認 | 独立更新・障害分離・異言語実行が不可欠で、Single-binary等の制約と整合する |

通常利用者が選ぶのはData/Relation/Function/Action等の**業務要求と必要な結果**。DB方言、Plugin package、ローダー種別はまず既存Profile/Provider/Host側で解決し、不足・競合・未確認だけを用途に応じた診断で提示する。選択を自動化しても、業務意味の受理・現在権限・Provider実効果は自動成立しない。

### Project選択と更新時の非干渉

構成時の記録は、既存Source/Lock/Builderで保持できる参照を使い、最低限「必要Contract/Target」「選択済みProvider/Generator」「exact Sourceと実装Fingerprint」「Build closure/実Artifact」「実行時のHost capability」を区別する。候補Catalog/Bundleが更新されても、**選択・実消費・配布Artifactが不変なら無関係なProjectを過剰失効しない**。ただしDefault Provider選択、Go import/init、Generatorの読込入力、Native SQL Asset等が変われば、名目上未使用でも実依存として再資格化する（#1251/#2600）。

**案件Ownerへの最小質問**：何を実現したいか、既存機能が不足する具体例は何か、業務意味の正本/変更権限は誰か、必要Targetと厳密単一バイナリ/独立更新の要否、Read/Writeと途中失敗で残る効果、許容停止と試験可能な独立Expected。この情報が既存Source/Projectから既知なら質問を繰り返さず、未決のみ照会する。

**判定の停止線**：Providerが曖昧／Scope・Capability不整合／実Target非対応／実Source消費未特定／Sourceの意味改変を要するがOwner未受理／部分CommitやACK-lossをRecoveryできない場合、黙示の別Provider/別Process/fallbackへ切替えない。既存Core fixed kind、万能Plugin ABI、Runtime Loader、外部Hostの一般義務化を先取りしない。

**既存Issueへの受渡し**：このProfile側は要求→候補→選択の境界（#1016）。Service Contract/現在Authorityは#951、Plugin Package/Bundleは#1251、DSLは#988、Data Providerは#2622、実生成・Buildは#1121/#259/#943、更新費用は#2600/#2018、Effectは#1174。UI Kitの`extends/provide/replace/decorate`を汎用Compositionの優先順位へコピーしない。

検証は上記§14の既存 EX-01〜08／#1251 PB/PBN／#1016 PCF に対応する最小Fixtureを**共有**し、第二の期待値正本を作らない。今回の追補は設計文書のみ。実Go/DB/外部Hostの新E2E、正式Plugin粒度・ABI/Schema、Owner受理はNOT RUN／未採用。

<!-- tei-project-extension-qualification-handoff-20261106-v2 -->
### 案件Profileの選定結果を「実行資格」と混同しない（2026-11-06追補）

[Runtime/Target Binary Composition §15](RUNTIME_TARGET_BINARY_COMPOSITION_DESIGN.md) の証拠受渡しを、案件の構成表示へ**参照**する。Projectの入力として新しいReceipt/Lock Schemaを設けず、既存Source/Composition/Build/Hostが所有する参照を結ぶread-only投影とする。

| 案件画面・CUIで説明すべき状態 | 表示できる根拠 | 次へ進めない場合 |
| --- | --- | --- |
| 要求を満たす方法 | Configure/標準Rule・Operator/Plugin/Target候補と、必要な追加契約 | Plugin化の実需がないなら既存機能を優先 |
| 構成の選択 | Required Contract・Provider/Generator・Targetの明示Selectionと選択根拠 | 曖昧Provider・欠落Capability・禁止fallbackなら候補保留 |
| 生成物への収録 | 実消費Source/依存の閉包、Target別artifact/member・actual loaded bytes | 未使用物混入と必要module依存欠落を**別の失敗**として診断 |
| 実行の資格 | 実Caller/Scope/Host Grantと対象・Source/Model/Operation版 | Build収録だけでは実行可能と表示しない |
| 実業務の結果 | Request/Attempt、Commit/Unknown、Receipt/独立Read | ACK・Exit0・画面上の成功だけでは業務完了としない |

**案件Ownerへ追加で尋ねるのは結果を変える質問だけ**。単一バイナリが厳密な配布制約なのか、別Processが必要な実障害・更新周期の理由は何か、途中確定の許容とOwnerの回復責務は何か、のうちSource/要件から確定していない事項を選ぶ。細かなDB方言・Go module・Plugin packaging方式を通常利用者の必須入力としない。

**移行時の判定**：選定済みProviderが変わったときは同名FunctionでもTarget Capability、独立Expected、実Read/Write/Receiptを再評価。未選択BundleのCatalog版だけが変わっても、実Selection・実消費・Artifactが不変なら無関係Functionを過剰staleにしない。ただしSource/SQL Asset A→B→A、Provider Default順位、Go init/選択module go.sumの変化を「同じ名だから不変」と扱わない。

**担当への返却**：Project選択は#1016、Contract/Grantは#951/#1174、依存不足・過剰混入は#1251 PB-10/#1167/#2600、UIは#997、DB/SQLは#2622、Targetは#259。判断に必要な不足条件を表示して保留し、第二Resolver・万能Plugin Runtime・独自Host・新Gateを追加しない。新しい実Consumerの横断試験・Owner受理はNOT RUN。

<!-- tei-project-generated-extension-reconciliation-20261106 -->
### 案件の生成拡張点を汎用Pluginと同一視しない（#2888重複統合）

[Runtime/Target Binary Composition §14.7](RUNTIME_TARGET_BINARY_COMPOSITION_DESIGN.md) に#2888の固有差分を集約。案件が追加した保存前検査・業務規則は、まずProjectのAccepted Source/Rule/Functionとして扱う。生成物へ案件処理を接合する必要がある場合も、#2090の宣言済みHookに案件Sourceを置く方法と、独立Version/Conformanceを持つPluginを配布する方法を**別に比較**する。Hook名/型が同じでも、生成器更新で呼出順・粒度・Owner Meaningが変われば同等とは判定しない。

**案件の更新追従**は旧Generator/Kit＋Project-owned delta＋新Generator/Kitの三者比較で診断する。再生成で案件Sourceを失う・生成コードへ直接書き込む・未公開Slot/Hookへ依存する場合は保留。UIの通常表示/Action affordanceの成功を、実Host GrantやCommit資格へ昇格しない。#2090の限定Closed/チェック済み項目は全Targetの実証済みではなく、ここでは再試験を増殖させず既存Caseへ参照接続する。

**差戻し線**：独立配布のConsumer・更新責任が無いなら、新Plugin宣言/Loaderを必須化しない。業務意味の変更が必要ならOwner受理へ戻す。生成拡張点の互換性不足なら契約を修正し、案件Sourceを保護したまま生成物を再検証する。PR #2888のEXD-07は同設計§14.7と#2090へ吸収し、第二Fixture正本を設けない。docs-only／NOT RUN。


<!-- tei-project-plugin-update-impact-projection-20261106 -->
### 案件画面におけるPlugin/Bundle更新影響のread-only投影（2026-11-06、設計・未採用）

[Runtime/Target Binary Composition §17](RUNTIME_TARGET_BINARY_COMPOSITION_DESIGN.md)に更新イベント別の影響判定を集約する。Project Compositionは**実利用状況と判断に必要な不足条件を表示**するだけとし、別のUpdate Resolver/Registry/Acceptance/Grantを持たない。

| 案件側の見え方 | 提示する根拠 | 禁止する誤案内 |
| --- | --- | --- |
| 候補Bundleが更新されたが当該Application未選択 | Catalog/Selection/実消費入力の差分、必要な場合だけ候補再解決 | 全Functionを「更新必須」とする |
| 現在使用する生成器・Providerに変更がある | Selected Contract/Implementationの実Byte、必要Target/Artifact・試験不足 | 同名同version/同じ表示結果で「互換済み」とする |
| Contract/Scope/Effectが改訂された | 既存Accepted Meaning・Conformanceと現行Grantの差異 | Plugin導入やCI PASSで業務Ownerの再受理を省略 |
| Providerが利用不能・撤回された | 選択済み対象、現在のConformance/Grant、影響するAction/Function | 最初の候補へ無断fallback、旧Artifactの存在を実行許可扱い |
| Update後もArtifact内容が同じ | 実Build/Inputと各member、独立Test・Current Eligibilityの差分 | Artifact byte一致でEvidence/Grantも同一とする |
| Source/Query/Ruleが別版となった | Source occurrenceと依存、Rule時点、Read ScopeとConsumer開示 | A→B→Aで旧Source受理や旧実DB Read資格を自動復活 |

通常利用者には「対象Function」「何が変わるか」「保留の理由と必要な確認」を優先し、Bundle内部のmodule/ToolchainやDB Dialect詳細は必要な権限を持つ開発者向け表示に留める。比較の正本は既存#951/#1016/#1251/#2600/#1174の結果を参照し、古いPreviewや過去Evidenceを現在の実行許可に昇格させない。

**検収**：#1016 PCF-06と#1251 PB/PBNを流用し、無関係Plugin更新は過剰staleにしない一方、選択済み実装・SQL Binding・Source/Rule・Current Grantが変化する場合は不足診断を返せること。実Studio/CUI実装・実Build・権限・Owner受理はNOT RUN。


<!-- tei-composition-independent-gates-source-bootstrap-d1-scope-20261107 -->
## 2026-11-07 横断受入：Plugin不要・静的Provider・*.tei.md Bootstrap・業務Effectの独立ゲート（設計／NOT RUN）

本節は**案件から見た最短の実証順と、相互に前提としない条件**のみを扱う。方式選定は[Runtime/Target §14〜22](RUNTIME_TARGET_BINARY_COMPOSITION_DESIGN.md)、新Authoring移行は[Draft #3030](https://github.com/hat47x/tei-platform/pull/3030)、初回Language/Operator Bootstrapは[Draft #3028 §13](https://github.com/hat47x/tei-platform/pull/3028)、Scope相対のEvidence接合は[Draft #2903 §24](https://github.com/hat47x/tei-platform/pull/2903)を正本とし、第二Selection/Lock/Proof Store/Registry/Hostを作らない。

### 四つの経路の前提と停止条件

| 経路／次の最短検証 | 必ずそろえる証拠 | 別経路を前提にしない条件・資格上限 |
| --- | --- | --- |
| **A Plugin不要D0／Process D2**：既存見積もり23件の直接Go計算と子Process対照 | 独立Oracleのraw入力/Expected、exact HEAD/Toolchain、D0実GoとD2 Build/Invoke/SKIPを区別した結果 | 新*.tei.md Reader/動的Registry/実DB Host Grantを純計算の基線に要求しない。[Draft #3021](https://github.com/hat47x/tei-platform/pull/3021) HEAD dfc133e2の3 CI成功は限定証拠で、D1や正式Owner受理の実証ではない |
| **B D1 静的Provider**：同一Go TargetのContract選択→実Link→Invoke | Aと同じOracle/Target、[#951](https://github.com/hat47x/tei-platform/issues/951)の明示Contract/Provider選択、実Build closure/linked member/選択実装と呼出実装の一致、未選択実装の非混入 | *.tei.md移行完了をHard依存にしない。現行Go Source/JSON Oracleで比較可能。ResolvedSet/BuildPlan、D0・D2成功はD1 PASSではない（#1251/#1167/#2600） |
| **C 新*.tei.md由来Plugin/Role**：Raw MarkdownからCandidateを解決 | #3030 TM-02/03/12/14（実File発見/Read、原byte＋block span、必要Source充足、旧新重複拒否）、#3028 BS-02/03/06/15（既知Seedで読めるFormat/Language＋実Parser Pin、自己循環拒否、Operatorと有効化の分離） | C未完成でもA/Bの既存Source Reference試験を止めない。ただし新形式はRaw ReadだけでSelected/Build-qualified/Activeとは表示しない（#2621/#2823/#988/#2620） |
| **D Effectを持つ案件・多Tenant**：Host/Provider実行→Consumer完了 | 対象Action/Resource/Tenant/時点に束縛したMeaning・Current Grant、Required/Supplied/Evaluated Guard、実Read範囲・Receipt・独立Consumer Body/Disclosure（#1174/#1163/#2516、#2903 §24） | A/Bの純計算ConformanceはDのGrant/Commit資格ではない。main統合#2926 EvaluateCompletionのCoverageCompleteはCaller供給値。実Providerの母集団/epoch/Scopeで裏付くまでは業務完了を表示しない |

### Projectの最小負例と診断

- **Source存在≠実Loader消費**：managed inventoryやvalidate成功だけで利用可能とせず、必要Plugin Descriptorの実Read/ParserとRequired Sourceの被覆を確認（#2823/#3030 TM-14）。
- **初回Seed自己循環**：*.tei.mdやfrontmatterに未知Languageの名前があるだけでそのLanguageを初起動しない。Seedで確定不能ならRaw Draft保持・Interpretation/Activation保留（#3028 BS-15）。
- **同名旧新Source／Provider競合**：旧.teiと新.tei.mdの同Subject、同Contractへの複数実装、同版異byteを後勝ち/最新候補で無音選択しない。否定依存と実探索範囲が不明なら存在しないと断定しない（#951/#1016/#3030 TM-03）。
- **D0/D2→D1の資格誤転用**：同じ23件Expectedで合格してもD1実Linked Providerの呼出証拠がなければ未実施。D2 offline moduleで意図的にSKIPされたD0をrootの23件PASSへ加算しない（#3021/#2514）。
- **Effect/Scopeの証拠不一致**：Tenant A/B、無権限と業務0行、Partialと完全Read、Preview時GrantとCommit時Grant、旧Receiptと現Attemptを同じ完了としない（#2903 JOIN-01〜06）。

**優先順位**：まずAの固定HEAD・限定CIを保存し、**BのD1実静的Link＋Invokeを次の最小実装作業**として既存#951/#1251へ渡す。Cは別レーンでPlugin Descriptor1件のRaw Reader→Seed/Parser Pin→Parse＋Span検証。Dは既存Host/DBで同一Attemptの資格がそろったときだけ実E2Eを評価。無関係なGateを相互のHard dependencyへしない。Go単一Binary既定、外部Hostの現在認可、通常.mdの非自動実行、Owner意味受理の独立性は維持。

**状態**：既存Issue/PR/Go Codeの観測を案件計画の依存順に整理したdocs-onlyの追補。新しいD1実Go Link/Invoke、*.tei.md Loader、Multi-Tenant/DB同一Attempt E2E、Owner正式受理の試験はNOT RUN。新Runner/Schema/ADR/Authorityを導入しない。
