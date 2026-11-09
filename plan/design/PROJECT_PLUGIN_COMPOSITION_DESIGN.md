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
| **A Plugin不要D0／Process D2**：既存見積もり23件の直接Go計算と子Process対照 | 独立検証基準のraw入力/独立期待値、exact HEAD/Toolchain、D0実GoとD2 Build/Invoke/SKIPを区別した結果 | 新*.tei.md Reader/動的Registry/実DB Host Grantを純計算の基線に要求しない。[Draft #3021](https://github.com/hat47x/tei-platform/pull/3021) HEAD dfc133e2の3 CI成功は限定証拠で、D1や正式Owner受理の実証ではない |
| **B D1 静的Provider**：同一Go TargetのContract選択→実Link→Invoke | Aと同じ独立検証基準/Target、[#951](https://github.com/hat47x/tei-platform/issues/951)の明示Contract/Provider選択、実Build closure/linked member/選択実装と呼出実装の一致、未選択実装の非混入 | *.tei.md移行完了をHard依存にしない。現行Go Source/JSON形式の期待値で比較可能。ResolvedSet/BuildPlan、D0・D2成功はD1 PASSではない（#1251/#1167/#2600） |
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

<!-- tei-project-admission-auth-witness-update-20261107 -->
## 2026-11-07 現行コードの前進を案件資格へ接続：Source Admission・Grant根拠・実Effect（設計追補）

前節のA〜D四経路を**新しい共通Gateに組み替えない**。別レーンで進んだ実Go修正により、案件の構成表示で言えること／言えないことの境界を更新する。Format/Loaderの移行正本は[Draft #2667](https://github.com/hat47x/tei-platform/pull/2667)（横断移行）、[Draft #3030](https://github.com/hat47x/tei-platform/pull/3030)（Authoring形式）であり、[Draft #3031](https://github.com/hat47x/tei-platform/pull/3031)は移行を実装したものではない。

| 実装と固定資格 | Projectに表示してよい限定事実 | 表示してはならない昇格 |
| --- | --- | --- |
| **新Markdown Source Admission**：[Draft #3031](https://github.com/hat47x/tei-platform/pull/3031) HEAD 63953ccbの局所CIは、対象外扱いで*.tei.mdを黙って通す代わりに未Bindingを診断するGuardのfocused/Workspace試験が成功 | 「TEI向けMarkdown Sourceを検出したが、Parser未Bindingのため解釈・構成には使えない」。旧*.tei処理の互換と区別 | Rawファイル発見、validateの一部成功、Guard PASS→Markdown Parser/Meaning/Plugin選択の実資格 |
| **WorkspaceのRoot診断**：main統合[#3073](https://github.com/hat47x/tei-platform/pull/3073)は未知/不存在のsources root、既存Loaderが読まない近似名ファイルへ診断を追加 | 実Loaderの既知範囲でroot存在・綴り・対象外に気付ける。sourceRootFindingsとvalidateの診断理由を示せる | root findings 0件・declarationCounts>0・validate ok→全Required Source/Import/Provider Readの完全性、*.tei.md実解釈の成功 |
| **Grantの発行・宣言境界**：mainへmerge済み[#3032](https://github.com/hat47x/tei-platform/pull/3032)のissuedBy、[#3035](https://github.com/hat47x/tei-platform/pull/3035)のaccess.Ground（Capabilityの宣言、論理Scope→実Path解像、ResolvedByの記録） | 「宣言されたCapability・Resource・Access範囲との整合を検査した」。主張したissuerと解像根拠、実Scopeを提示 | 発行者名/ResolvedByが文字列で一致→発行・解像の真正性、Host/IdPの現在権限、Tenant全Scopeの許可 |
| **Consumer側の発行照合**：main統合[#3038](https://github.com/hat47x/tei-platform/pull/3038)の任意 -grants/-declaration、[#3075](https://github.com/hat47x/tei-platform/pull/3075)の共通GroundFile。[#3047](https://github.com/hat47x/tei-platform/pull/3047)は宣言権限と発行権限の局所照合 | 実Invocationが照合を明示したとき、その範囲の宣言整合を確認済みと表示できる。参照GrantAuthorizerがCommandテストをPASSした事実も区別 | 任意照合が全Consumerで強制済み、GrantAuthorizerがCLI/Studio/本番Hostへ配線済み、issuer/ResolvedByが真正、本番Effectが認可済み |
| **Command実行時の認可・Effect**：main統合[#3034](https://github.com/hat47x/tei-platform/pull/3034)の旧規則Adapter（[#3064](https://github.com/hat47x/tei-platform/pull/3064)で依存方向を整理）、[#3042](https://github.com/hat47x/tei-platform/pull/3042)の必須認可/期間検査、[#3055](https://github.com/hat47x/tei-platform/pull/3055)/[#3068](https://github.com/hat47x/tei-platform/pull/3068)のDifferences・Applied未観測、[#3054](https://github.com/hat47x/tei-platform/pull/3054)のpersisted/durability | 参照Commandが受けたRequired Guard、宣言効果と列挙済みAppliedとの差、観測の有無、Unit-of-work完了/永続性を独立提示できる | Menu表示＝実行許可、Applied空＝差分0/副作用0、committed:true＝DB永続保存、unknown効果＝安全再送、参照Test＝実Provider/Receipt |
| **テスト判定基準の命名**：main merge済み[#3039](https://github.com/hat47x/tei-platform/pull/3039)/[#3040](https://github.com/hat47x/tei-platform/pull/3040) | 人間向け文書では「独立期待値／独立検証基準／テスト判定基準」を使う。DB製品は「Oracle Database」と表す | 既存コード識別子、履歴のGit SHA/パス、Fixtureの機械的な全面改名。過去の試験結果を新しいHeadのPASSに加算 |

**停止線を一つに見せない**。実際の必要関係は次のとおり。

- **D1純粋quoteの実験**：既存[D0/D2 PR #3021](https://github.com/hat47x/tei-platform/pull/3021)の固定23件の独立期待値を使い、選択Provider B→実Go Link→**Bの実Call**を同一Target/同一試行で照合する。*.tei.md ReaderやDB Grantは不要。選択BでもAが呼ばれ同結果となる負例で選択と実呼出しを判別（#1251/#951、#2885 §18.13）。旧quote Providerがpackage mainで、現D2がGo Sourceを一時offline moduleへコピーする事情から、単純な共通packageへの抽出でD2を壊さないことも実測条件。
- **新形式のPluginへ切替**：#3031の未Binding診断が有効であることと、#2667/#3030の実Markdown Reader/Seed/Parser Pin/Role・Source span/実消費を区別。Guardを解除する前に必要Source/重複旧新/正常例・拒否例の独立期待値を固定。旧Sourceの正常CIを新*.tei.md SourceのPASSへ転記しない。
- **Effectを持つPlugin**：現在の対象Action/Tenant/Resource/Scopeで、発行主体の真正性（未実証なら未資格）、宣言と実解像、Consumer側の実検査有無、実Host/Provider現在強制、Applied/Receipt/独立Readerを別照合する。PreflightでGrantがあってもCommit時の撤回発効点が分からなければunknownを維持。#3032/#3035/#3034の限定Go実装から、未検証の本番発行者署名・完全な権限チェーンを推論しない（#951/#1174/#2516）。
- **Reader不完全・適用操作不明**：Providerが未取得・部分的・非開示なら0件を「不存在」と解釈しない。Appliedを返さないExecutorの空値は「未報告」であり、「副作用ゼロ」ではない。同一AttemptのCoverageとEffectを別の実観測で検証する。

**優先・Owner**：D1静的Link/Invoke（#1251/#951）を最小の次実装とし、Source Readerの正式移行（#2667/#3030/#2823/#988）と、Grantの真正性・Host/Receipt（#1174と既存Access Owner）は別レーンで進める。#3038/#3075/#3047/#3042/#3054/#3055/#3068/#3073はmainへ統合済み。ただし#3031の未Binding GuardはDraft未統合であり、新Markdown Parserがmainで稼働したとの主張はしない。今回の新Go/OS/DB E2E、署名付き実Grant、Owner意味受理はNOT RUN。第二Registry/Host/Status API/検証基準正本は設けない。

<!-- tei-composition-claim-interpretation-after-reference-command-20261107 -->
### 実装が進んだ後も資格を強め過ぎないための一試行の対照（既存Independent Expectedの投影）

参照Commandで順に、(a)宣言されたCapability/Grantの照合、(b)実行時Guard、(c)宣言した効果、(d)報告されたApplied・差分、(e)CLI/Studioのcommitted/persisted/durability、(f)Operation/Receiptと独立Readを表示する場合、**全欄が埋まっていても同じTarget/Tenant/Source/Attemptを参照すること**が必要。個別PASSの寄せ集めを一つの『実行済み』にしない。

| 既存の限定対照 | Project側の判定 | 未証明のこと |
| --- | --- | --- |
| Source rootが未知／未読込だがResultは空 | Source coverage未確立として保留。既知空Workspace（宣言0件）と区別 | 対象業務にDataが存在しないという確定 |
| CallerがGrant/宣言を供給しCommand側で照合PASS | Scope適合の局所検査として表示。真のissuer/Host現在強制は不明 | Grant真正性、承認済みConsumer、別Tenant許可 |
| Declared effectあり、Appliedを列挙しないExecutorが成功 | AppliedUnobservedとして表示。差分がないのではなく比較不能 | 副作用ゼロ、書込み範囲正確、実DB効果 |
| 参照Unit-of-workでcommitted:true・persisted:false、durability:none | Unit-of-work完了の限定表記、保存は未確認 | 永続DB Commit、外部Receiverの受領 |
| Effect不明・同Owner/Generationでの再Submit要求 | COMMAND_EFFECT_UNKNOWN と再送拒否を区別し、既存Receipt/独立Readによる回復へ | 未適用と断定、別Keyで安全再送、無条件Rollback |
| Commandでread-only datasource宣言へWrite要求 | [#3070](https://github.com/hat47x/tei-platform/pull/3070)の宣言上の事前拒否を表示 | DB/Provider側での実強制、宣言外Dataの可否 |

これらは#1174/#2514/#2823/#2586の既存独立期待値・実観測Ownerに渡す追加対照であり、新しいOperation State、UI Schema、Proof Storeを定義しない。Appliedの差分がある状態で参照処理が成功しても、差分が解消済みとは表示しない。純粋quoteのD1 Link/Invoke測定には、DBの存続・Grant/Receipt/Appliedを無関係な必須条件として追加しない。**追加Product横断試験はNOT RUN**。

<!-- tei-project-tool-plugin-choice-handoff-20261107 -->
### 外部Toolの実装資格とPlugin選択の交差（Runtime/Target §23、未採用）

[merged #3121](https://github.com/hat47x/tei-platform/pull/3121)は参照Toolの能力をEnvironment Grantへ追加した際、Plugin Catalogで解決できず `PERMISSION_GRANT_CAPABILITY_NOT_RESOLVED` となることを実検証した。案件側の確認事項は「Toolを論理Pluginとして契約・実装・Build/Lockへ登録する実利益があるか」「Plugin以外のTool Principalを既存Hostが明示委譲できるか」の2案と、それぞれの現在Grant・Consumer適用先・Revocationを同Scopeに結べるかだけ。Toolが存在することからPlugin導入もGrant発行も自動提案しない。詳細比較・負例と案の未採否は[Runtime/Target §23](RUNTIME_TARGET_BINARY_COMPOSITION_DESIGN.md)へ委譲する。

[merged #3077](https://github.com/hat47x/tei-platform/pull/3077)/[merged #3108](https://github.com/hat47x/tei-platform/pull/3108)のHook宣言外呼出し拒否・Command委譲は局所Fit。Hook Runner・実Host Grantは未配線／未実証であり、案件UIの「Hookが有効」「Tool実行可能」「業務操作完了」を同じ状態へまとめない。[merged #3124](https://github.com/hat47x/tei-platform/pull/3124)/[merged #3127](https://github.com/hat47x/tei-platform/pull/3127)の `PersistenceState=observed` は現Adapterでは**Receipt Lookupの観測**であってBusiness Resource現在値の確認ではない。表示・説明は「何を読んだか」を明示する。参照Toolの選定案をD1純粋quoteの実Go Link/Invokeの前提にはしない。

**状態**：案件への説明可能性の要求候補のみ。ToolをPlugin化する採用、Tool Principalの追加、Hook→本番Host E2E、Receipt→実Resource読取、Owner受理はNOT RUN／未決。別のStatus enum/Registry/二重Authorityを増設しない。
<!-- tei-extension-three-axis-decision-20261105 -->
## 2026-11-05 横断追補：拡張点・Plugin独立性・実行方式を三段階で選ぶ（設計／NOT RUN）

**目的**：#988（Language/Operator）、#2622（Data Operations/Provider）、#951（Contract/Conformance）、#1223（Composition/Target条件付き世代切替）、#1251（Bundleと物理除外）、#943/#2600（Source→Build→Artifact/増分再資格化）を、**要求Ownerが選択に用いる一枚の判断手順**へ接続する。既存の静的／動的／Process一般比較やUI Kit #997のSlot Resolverを再実装しない。本文のApplication Composition Profileは**選択結果の参照先**であり、新しい正本・Schema/第二Resolver/Host/Capability enumではない。

### P0 最初に三つの異なる問いを分ける

1. **拡張対象／意味契約**：Source表面を読むLanguage、値/集合の意味を定めるOperator、外部Resourceを実現するProvider、Artifactを作るTarget Generator、案件固有のRule/Function、既存UI/生成コードの拡張点のどれか。
2. **変更・所有・配布責任**：設定/Plain Data/Accepted Ruleで足りるか、共通の静的Library/案件Sourceで足りるか、明示Extension Pointか、独立identity・version・Conformance・更新責任を要する論理Pluginか。配布Bundleを大きくする判断と、Plugin独立性の採否は別。**Pluginとして配布するためだけに外部Process・動的Loaderを要求しない**。
3. **物理的な実現・実行時配置**：Build時の静的合成、実行環境での動的Moduleロード、既存Hostと別Process/RPCのいずれか。生成ツール側の動的Parser/Generator導入は、生成済みアプリケーションのRuntime Loader搭載を意味しない。

第三の問いを先に固定しない。原則、**Goでビルドした単一ネイティブ実行ファイルを標準候補**とし、別Targetでは対象Host固有のlibrary/JAR/Wasm/Web asset等を許す（ADR-0043）。「単一配布パッケージ」と「追加実行物・Loaderを持たない単一Binary」は異なる要求である。

### P1 拡張種別・契約・棄却条件の決定表

| 対象／例 | 最小契約・正本 | まず選ぶ実現 | 失敗の波及範囲／棄却する条件 |
| --- | --- | --- | --- |
| **構文**：管道/表記糖衣、別Script dialect | 選択Language/Version、Source bytes/span、Grammar/parse diagnostics、Lowering→Plain Data Candidate、Source roundtripとAccepted Meaningの分離。#988/#2639 | 既存Language実装への明示切替またはbuild-timeに選ぶParser。糖衣だけなら新Pluginは不要 | Parser登録順で意味が変わる／Source位置や元糖衣を復元不能／Source A→B→Aで旧Accepted意味を勝手に復活。Coreの固定Parserに毎回branchを増やす場合は#1019の圧力観点で再評価 |
| **純粋Operator**：filter/aggregate/decimal | 演算ID、入力/出力のsemantic role/型、NULL・欠損・多重度・順序・精度/丸め、決定性、read/write Effect=none、独立Expected。#988/#2507 | 多くは**静的LibraryまたはData Operationsの標準関数**。複数独立Consumerで契約差替・独立versionが実際に必要ならOperator Contract-Provider | 同型結果を意味同等と扱う／読み取りや外部I/Oを黙って持つ／業務ルールを汎用Operatorへ埋め込む場合は棄却・責務再分類 |
| **Owner固有のRule/Function**：受注承認、税区分、保存前条件 | Project Accepted Meaning、Definition/Instance/Scope、Owner受理/Revision、実Action・Rule、検査・Effectの区別。#2622/#1129/#267 | **Plain Data/明示Rule/Data Operations/Function**として保持。技術的な独立Pluginにせず、依存する既存Provider/Operatorを選ぶ | Plugin版だけでOwner意味受理を代行／共通Library更新で案件規則が無音変更／UI/Providerが第二業務正本を持つ |
| **Data/DB Provider**：RDB/OData/File/API | Service/Capability ID、target/alias/scope、Read/Write粒度、query/NULL/partial、PATCH/replace、Tx/atomicity、expected version、current Grant、Commit/Receipt/unknown、実Datasource Binding。#2622/#951/#1174 | 既存Hostの権限境界で選ぶ独立Provider。Go同一プロセス静的合成を第一候補、異言語/隔離要件ならProcess等 | 未知Schema/SQL dialectを黙示fallback／0件=不存在／batch=atomicと推測／実write対象・権限・部分効果を観測不可。Provider追加でSemantic Dataの正本を複製 |
| **Target Generator**：Go/Wails、Java JAR、.NET DLL | Target/Profile、Input/Meaning/Source provenance、Generation/Toolchain Version、Build closure、output members/digest、link/runtime依存。#1121/#259/#943 | **生成時のみ**選択したGeneratorを静的利用。生成アプリはTarget環境に適合する最小Artifact | 生成器を使わないTargetにもRuntime Loaderや秘密/DB driverが残る／semantic behaviorの差をTarget独自仕様へ黙示変換／actual linked/loaded bytes未確認で完全を主張 |
| **生成コード・UI拡張**：案件独自計算、Wails Panel/Go Service | 宣言されたExtension Point/Slot/版/入力・出力・順序、案件が所有するSource、上流generator/Kit差分。#2090/#997/#2785 | **Project Layer/静的Source合成**が第一候補。独立配布・独立Consumerが必要ならPluginを検討 | 生成ファイルを直接編集／再生成で案件変更消失／base API/呼出順変更を無検出／Slotを認可迂回にする |
| **外部Process**：他言語Native/既存製品連携 | **新しい意味Plugin種別ではない**。上記Service Contract＋transport/lifecycle、child executable digest、引数・principal/scope伝播、timeout/restart/cleanup、Result/Unknown。#1145/#2390/#1223 | 障害隔離/独立更新・異言語Hostが必要な場合のみ既存HostのProvider実現として採用 | 厳密単一Binaryに反する／独立プロセスにしても権限を強制できない／ACK loss時の結果不明や部分効果を二値に潰す／Host側起動物を検証できない |

**旧#2090の採用済み範囲を再利用**：宣言した拡張点の契約変化と再生成中の案件Source保持を検出する限定受入がClosedとして記録済み。ただし全Target、実Wails、意味受理、最新main実E2Eまで完了と推定しない。

### P2 実装形態は三つの異なる物理的選択（相互に混同しない）

| 選択 | 成立させる条件 | 強み | 棄却／他方式への切替を検討する条件 |
| --- | --- | --- | --- |
| **静的合成** | Contract/Provider/Generatorをbuild-timeにexact選択、Toolchainと実消費Inputを固定、対象Artifactをbuild/test | Go単一Binary、link時依存除外、起動時の版揺れが少ない。論理Pluginを静的同梱してよい | 独立パッチのための再ビルド/配布が実測SLOを超える、同一プロセス障害が許容されない |
| **動的ロード** | 対象Hostが実ロード・unload/更新の資格と隔離・generation pinning、current capabilityを実装 | 独立版更新・特定Hostの拡張方式に適合する可能性 | Go単一ネイティブBinaryに実行時追加ロードが不可／Hostが安全な解放・依存切替・Trustを強制できない。**Goの任意動的Plugin hot-swapを既定にしない** |
| **外部Process／RPC** | 信頼境界、実起動Artifact、transport contract、backpressure/timeout、readiness/stop、監査、失敗時のOwner | 障害・更新単位を分離、異言語・既存製品と接続 | 複数Process/依存が配布制約に違反／呼出遅延やIPC費用が要件不適合／部分Commit・ACK lossの復旧が未定義。Process分離＝認可隔離/業務Tx成立ではない |

**注意**：書式上「Plugin」と呼んでも必ず動的ロードするわけではない。動的ロードできることと長時間稼働時に無停止交換できることも別。#1223のreadiness/cutover/drain/fencingは常駐Targetを実際に選んだ時だけ適用し、静的Go配布物へSupervisorを必須搭載しない。

### P3 要求Ownerが選定前に答える最小の七問

1. **追加価値**：どの既存Data Operation/Function/Language/Provider/Target/Extension Pointでは表現できないか。新たにPlugin境界を作ることで何を独立配布・更新・再利用できるか。
2. **意味と権限のOwner**：Source/Role/RuleとPublisher/Contractは誰が所有するか。Owner Acceptance、Contract publication/Provider Conformance、現在Host Grantを別々に誰が判断するか。
3. **実影響と失敗予算**：pure/read/write/long-lived streamのどれか。部分確定・補償・再実行/冪等性/unknownの許容条件、実行継続性要求は何か。
4. **配置**：要求Target/OS/Toolchainは何か。「単一Go実行物」が厳格か、単一インストーラでもよいか。Build時だけ利用するか実Runtimeで必要か。
5. **更新・互換**：独立更新の頻度、許容停止時間、旧版共存期間、契約・schema・行動同等性の判定範囲、Provider失効時の切戻し規則は何か。
6. **Source/Build来歴**：実消費byte、生成器、Toolchain、SelectされたProvider、Build Closure、exact linked/loaded Artifactを何で固定・観測するか。未選択Bundle更新は対象Buildへ影響するか。
7. **独立テスト**：どの固定Source/Revision/独立Expected、対象Host、故障注入を使って検証するか。synthetic fixtureと実DB/OS/Provider/Owner受理の証拠をどう分けるか。

これらは新しい入力FormやDSL Schemaではない。#1016の要求整理PromptやIssueレビューに用いる質問であり、既知の情報を再質問しない。主要な未確定点だけOwnerへ戻す。

### P4 採否を誤らない負例（共通Expectedを既存#1180/#2514へ委譲）

| ID | 最小比較 | 判定と担当 |
| --- | --- | --- |
| EXD-01：**非Plugin反証** | 単一Projectのpure operatorまたはOwner Ruleを静的Library/Data RuleとPlugin双方で実装 | 同じ意味・独立Expectedが保て、Plugin方式だけ配布/更新/試験の負担が増えるならPlugin必須化を棄却。#988/#2622/#2018 |
| EXD-02：**構文・意味分離** | 同じPlain Data候補を二つの表層から導出。Source位置・糖衣・Roleを意図的に変える | Lowering成功/同型だけでAccepted Meaning同一としない。#2639/#988 |
| EXD-03：**選択非干渉** | 高凝集Bundleへ未選択Provider/Generatorを追加し、Go import/init/embedと実link結果を確認 | Selection/Build Closureが不変なら必要成果物も不変。未選択扱いでもinit副作用が出れば不合格。#1251/#1016/#2600 |
| EXD-04：**Selection drift** | 同名・同versionで異digest、Default競合、選択DB別SQLの入替、Source A→B→A | 実選択/消費と資格の失効を区別。暗黙fallback・旧evidence復活は拒否。#951/#2622/#2600 |
| EXD-05：**Providerの意味・作用** | 同型のRDB/OData読取でpartial/null/NULL、PATCH/Replace/Batchを変更 | 行値一致だけで互換とせずScope/粒度/実Readと保存Effectを別判定。#2622/#1174 |
| EXD-06：**先行効果残留** | Step A commit後Step B block、またはProcess ACK loss→再起動 | 全体Effect=0、Exit0=Business Complete、自動再Submitの誤判定を禁止。実Receipt/独立readまでunknown。#1130/#1174 |
| EXD-07：**再生成拡張点** | base Hook引数/順序変更、案件Source保持で3回再生成 | 直接生成Source改造・無検出破損は不合格。#2090で受理済みの限定契約をreuseし、別Target実E2Eは追加で判定。#997/#2785 |
| EXD-08：**Target/Host差** | 同じAccepted FunctionをGo単一Binaryと外部Host Library/Processで利用 | Contract/Owner Meaningを再定義せず、実Build/Invocation/Host Grantと必要CapabilityをTarget別に観測。#1121/#2390/#259 |
| EXD-09：**失敗隔離の反証** | in-process panicとchild process crash、未観測実DB effectとOS Resource残留 | Process分離で障害の影響がどこまで減るか実測。隔離できない外部副作用・秘密/権限を隠さない。#1223/#2515 |

**順序**：まずEXD-01/02/03をread-only owner map＋既存Go/Node Referenceの範囲で評価 → 一つのread-only ServiceでEXD-04/05/08 → 一つの実EffectあるProviderでEXD-06/09 → 最後に必要な案件UI/再生成のEXD-07を別Targetで確認する。実装レーンは独立に並列化可能だが、同一Attempt/Source/Expectedの統合判定は#1119/#1163に戻す。既存Fixture/Oracleを複製しない。

**採用の出口**：候補ごとに「必要Contract／非Plugin代替／選択したProvider/Generator／物理配置／固定Source・Build Artifact／失敗影響／Test isolation／Owner未決」を一行で記録。**必要な実Consumer・独立Expected・故障注入がない方式を『標準採用』しない**。条件未確定はunverified/unsupportedと表示し、静的・動的・Process間で無音fallbackしない。

**既存Issueへの反映先**：決定表の参照入口=#1016、本番契約のOwner=#951、構文・演算=#988/#2639、データ操作/Provider=#2622、生成Target=#1121/#259、Bundleと未使用除外=#1251/#943/#1167、キャッシュ・Evidence再利用=#2600、実Effect・Authority=#1174、Process/外部Host=#1145/#2390/#1223、UI/生成Hook=#997/#2090。新Issue、共通実行Host、Plugin万能ABI、固定Capability enum、新Policy/Lock、業務意味の第二正本を作らない。

**状態**：今回の追補は**契約・配置方式の選定に関する設計/検収案**。EXD-01〜09、実DB／Windows Wails／他言語実Host、動的交換／無停止切替、実Provider Commit/Receipt、正式Owner意味受理はこの追補で**NOT RUN／未採用**。Closed Issueや既存Referenceの限定PASSを新しい統合E2E・Production保証へ読み替えない。


<!-- tei-plugin-standard-graph-representation-20261107 -->
## 2026-11-07 Pluginの組合せ・接続を標準データ構造と標準表現型だけで説明する（設計候補／実装NOT RUN）

### 目的・採否の境界

Plugin向けの専用Graph Core型、特製Node/Edgeデータモデル、新Resolver、新権限Engine、Canvas専用の意味正本を作らない。既存[#2495](https://github.com/hat47x/tei-platform/issues/2495)（標準構造）・[#2497](https://github.com/hat47x/tei-platform/issues/2497)（構造保存写像）・[#2507](https://github.com/hat47x/tei-platform/issues/2507)（Function Pattern/Presentation Binding）・[#1016](https://github.com/hat47x/tei-platform/issues/1016)（案件選択）へ乗せ、**Record/Collection/Relation/Graph/時点付きFacet**をPlugin DomainのRole付きデータとして解釈する。Graph、Tree、Table、Matrix、Timeline等は**標準表現型**の候補であり、型の定義はFunctionから独立、具体的な表現インスタンスは**Function内だけ**に置く（Headlessは0件可）。[Draft #3236](https://github.com/hat47x/tei-platform/pull/3236)の合成写像・情報損失、および[Draft #3222](https://github.com/hat47x/tei-platform/pull/3222)の複数View同期とは責務を重複させない。

ここで「Pluginトポロジー」は**次の相対的な構造投影**の呼び名であり、新Canonical形式名ではない。

| 観測する関係層 | 元の標準データ構造とPlugin Domain Role | 最低限保持する異同 |
| --- | --- | --- |
| 論理要求 | Function/ApplicationのRequired Capability/Contract、ProviderのProvides、typed I/O PortのConsumes/Produces | Contract ID+version、Input/Output Shape、Relation向き/多重度、NULL/Failure/Effect、必須/任意、未解決 |
| Provider選択・接続 | 案件Profile/Environment/Targetで選択したBinding/Selection **Record** がContract・Provider・ConsumerをRelationで結ぶ | Consumer/Function・Contract・Provider・Target・Tenant/Scope・Selection revisionを独立特定。多項関係を単なる「線」へ潰さない |
| Bundleと構成資産 | Bundle Contains Provider/Source/Asset、Provider Declares Dependency、Release/ArtifactのPackaging Relation | 同梱≠選択≠実Build収録。Bundle単位の更新を選択済みProvider以外へ過剰伝播しない |
| 実Build | Build Plan Requires / actual selected Member / Artifact Includes を、Source/Build fingerprintに束縛 | 計画依存≠実Go import/init/embed/Link。rejected/unknown Source closureを「依存なし」としない（#1251/#2831） |
| 実Runtime | Consumer Instance Invoked Provider Instance、Observed Read/Effect/ReceiptのAttempt相対Relation | Selected Provider≠invoked target。Host/Tenant/current Grant、Read Scope、時点、受領証拠をGraph定義へ「既定許可」として注入しない（#1174） |

**関係の最小性質**：安定した各端点のqualified identity（Plugin ID・Capability ID・Business IDを混用しない）、**Relation occurrence ID**、source/targetとrole/direction、cardinality/multiplicity、選択範囲（Application/Function/Target/Tenant）、valid time と観測時点、source provenance/revision、Relationの「宣言／解決／実消費／観測」の根拠、Required Source/世界の取得完全性を必要な粒度で保持する。未知・stale・permission-denied・partialは消えた辺/0件へ正規化しない。同labelの二Node・同一二Node間の多重有向辺を保存する。特定機能の多数端点接続は新Hypergraph型を作らず、端点・Roleを持つ**Connection Record + Relation集合**で表す。

### 標準表現型の選択と情報損失

| 標準表現型 | Plugin構成で最も有用な閲覧 | 表示だけでは成立しないこと |
| --- | --- | --- |
| **Graph**（有向多重辺） | required/provides/selected/contains/invokedの関係を種類・時点・Scope別の層に分けて表示。Portを必要なら独立NodeまたはRole付きEndpointとして表現 | 一本の矢印で依存/データ配送/呼出/許可を兼用すること。矢印の描画だけでBinding/Grantを変更しない |
| **Table**（Record/Relation表） | Node/Relation occurrence ID、端点、kind、source revision、Context、evidence、unresolved reasonの**損失の少ない検査面** | 表示行数＝業務実体/実消費件数、NULL＝不存在 |
| **Tree**（依存探索・Bundle階層） | 特定Rootからの展開、親子・Bundle内容や段階的な詳細 | 共通Providerへの複数親、DAG/循環、同一Nodeの別Path occurrenceを自動同一化。Tree化には複製PathとLossを明示 |
| **Matrix**（相互適合） | Consumer × Contract/ProviderのRequired/Provided/Selected/Qualifiedを版・Target別比較 | 値の色やチェック欄だけで選択を確定・現在権限付与 |
| **Timeline/差分Table** | Contract/Provider/Artifact変更、Selection改訂と失効、Invocation/Receiptの独立Attempt | valid time、Source revision、実Read/Execution time、媒体clockを同じ時間軸へ勝手に混ぜない |

Graphはデータの意味正本ではなく、**標準Graph→Representation Instanceへの写像**。Layout/zoom/drag/focus・表示層選択はPresentation Stateであり、Source/Selectionの書換えではない。Graph編集が必要なら、View occurrence→元Relation occurrence/Port/Bindingへの**逆写像の一意性**を先に確定し、Action Offer→明示Intent→既存#951 Resolver/Owner review→Host Grant/Effectへ渡す。情報損失・多対一・部分取得・未開示を伴う場合は表示・候補生成までで停止する（#2497/#2500/#2516）。

### 単一の標準構造を複数表示に投影する有限例

**前提（説明用であり正式DSL/Schemaではない）**：Function FはContract Qを必須とする。Provider A/Bが同じQを提供し、Bundle XはA/Bを両方同梱する。Project/Target/Tenantを条件とするSelection Record Sは**Bを指定**する。実Binary/Runtimeの観測Eでは**Aが呼ばれた**とする。A/Bの返却結果は同じ23期待値と一致する場合を想定する。

| 元データの独立Relation | Graphでの表示候補 | Tableで確認すべき正本ID・異常 |
| --- | --- | --- |
| F requires Q / A provides Q / B provides Q | 論理層の要求・提供の有向多重辺 | F/Q/A/Bのqualified ID、Contract版、二つのProviderの存在 |
| X contains A / X contains B | Bundle階層の二Edge | Bundleに2つあるだけでBの選択確定ではない |
| S selects B for (F,Q,Target,Tenant,Revision) | Scope付き選択Edge（SはRecordに保持） | 選択根拠・Profile/Lock・Tenant Scope・証拠時点 |
| E invokes A for (F,Q,Target,Attempt) | Runtime観測の別レイヤー（selectionと視覚的に差異表示） | **B selected / A invoked => 不適合**。計算結果が同一でもD1 PASS不可 |
| Eの実呼出しを取得できない | 観測層unknown/partialの表示 | 「A/Bいずれも呼ばれない」ではなく「証拠未取得」 |

**単位・軸の扱い**：構成の辺に物理単位は不要だが、Latency(ms)、Bytes、Cost(currency/時点)、Throughput(events/s)、Availability区間などを属性として載せる場合は標準Quantity/Time Facetへ束縛する。Graphのx/y座標は基本的にLayout値で、業務上の距離・依存強度・時間/性能の単位ではない。性能比較Chartや時間軸へ投影するときだけ元の単位・基準時刻・変換を検証する（#2495/#2497）。

### 検収・実施順（既存Expected SSOTを再利用）

1. **P0 Headless**：既存ContractとD0/D1の23件独立Expected、A/B・Bundle X・Selection S・実Call Eを、標準Record/Relation/Collection/Graphで読める構造候補へ写像。Graph/Table/Treeの各投影で同じSource-qualified identityとRelation occurrenceが追えることを検査。ラベル同名二Node、多重Edge、同じProviderへの多親、DAG→TreeのPath重複を追加の**判別条件**とする（新Fixture SSOTは作らない）。
2. **P1 Context別・複数表現**：一つのFunctionのData ContextからTableとGraphを投影し、Treeの非単射性/Partial Read/hidden Edge、Contract違い・同版異byte・Tenant A/B・Selection epoch ABAを診断。別Function/別Tenantへ選択状態を共有してもAuthorityへ昇格しない（#2506/#2507/#2497/#3222）。
3. **P2 実消費**：選択B→Go binaryの実Link/Invoke B、B選択・A呼出し誤接続、未選択Bundle収録、実Source closure/独立Consumer資格を既存#1251/#2390で検証。Graphは**その実Evidenceの表示**であって証拠作成者ではない。
4. **P3 操作導線**：Function内Graph Instanceに「接続候補を見る」「選択変更を提案する」「出典と未検証を確認する」Action Offerを配置し、実Source/Binding変更・再Build・Runtime Grantは既存の適切なHost/Ownerへ委譲。自動Link/Unlink、Drag-to-Grant、Graphからの直接Commitは禁止。

**担当**：Plugin構成Source・選択=#1016、Capability Contract/Binding=#951、標準形状=#2495、Projectionと情報損失=#2497、Function文脈と表示型=#2506/#2507/#997、Actual Build/Call=#1251/#2600、Evidence/独立Expected=#1163/#2514、Host/Effect=#1174。#3236/#3222の表現全般と重複するCore型・Renderer・逆写像Engineを作らず、Plugin固有の**関係Roleと受入例**だけを本節で所有する。

**証拠状態**：この追補は既存標準データ構造・標準表現型をPlugin Domainに適用する計画。新Graph Parser、実表現Renderer、Function内Graph Instance、実Go D1、Provider、Host、Owner正式受理は**NOT RUN／未採用**。Graphが描けることをContractの意味同等性・Current Grant・実Effectとして扱わない。


<!-- tei-plugin-crossref-tea-crud-typed-relations-20261107 -->
### 三要素分析法・CRUD・Subsystemとの共通Cross-reference：型付きRelationの多視点投影

[三要素Method](THREE_ELEMENT_METHOD_DIAGRAM_CONTRACT_2026-10.md)のData/Function/Business、[Subsystem分割 #3255](https://github.com/hat47x/tei-platform/pull/3255)のdirect CRUDとEffective Effect、[標準表現の合成写像 #3236](https://github.com/hat47x/tei-platform/pull/3236)のLossを、前節のPlugin Graphに接続する。**Plugin専用のGraph/CRUD Core型や独立した表編集Engineは作らない**。共通の導出単位は、標準Record/Relationのqualified Subject ID、Relation occurrence ID、endpoint/role/direction、Scope/Source Revision、証拠種別と開示資格である。Graph、Table、Matrix、Treeは、この同じ**公開可能なSource Slice**の異なる投影とする。

| 元の関係（標準データ構造） | Graphでの表現 | Matrix/Cross-referenceでの導出の禁止事項 |
| --- | --- | --- |
| Function requires Contract、Provider provides Contract | 要求/提供の有向・多重辺 | Consumer × Contract × Providerのセルは複数fact集合。available/selected/build-included/invokedを単一Booleanに潰さない |
| Profile/Target/Tenant/Revisionを持つSelection Record | Connection Recordのrole付き端点とBへの選択線 | 選択は多項関係。別Tenant/版での選択を一本の矢印で上書きしない |
| Bundle contains Provider、Build consumes Source、Runtime invokes Provider | 同梱、実収録、実Callを異なるRelation/Layerに分離 | 同梱＝選択＝実リンク＝実Callとは推論しない。未知の実消費を0件とはしない |
| Business Step → Function → Data | 業務Data授受とFunction I/Oの型付き線 | Business StepからFunctionへ到達可能でも、Step自身のC/R/U/DやBusiness Effectを自動生成しない |
| Function → Operation → Data（direct/derived/observed） | Data読取/書込、Rule/Trigger経由の推移効果、実Provider/Receipt観測を分離 | Function × DataのCRUDセルは**Direct Declared C/R/U/D、Derived Effective Impact、Observed Effect**を別に保持。空欄は未取得／不在を区別 |
| 複合Key/多親DAG/Relation occurrence | 元Relationと参加Field/Endpoint Roleを別Recordで参照 | 一つの図上Edgeを消しただけでComposite Key/Field等の一意逆Patchを主張しない（TEA T03／#2497） |
| Subsystem候補 × Function/Data | 所属候補と越境Data/Effect辺 | サブシステム境界はCRUD凝集・直接/間接効果・未取得Coverageを照合。Folderや見た目から物理DB/サービス境界を断定しない（#3255） |

図上のEdgeは**Semantic Relation、特定Ruleから導出したDerived Relation、単なるPresentation Edge**のいずれかを区別する。DerivedはRule/Source Pin、ObservedはAttempt/Provider/Scopeに束縛。Layout・Zoom・表示Matrixの列順の変更は原則Presentation Stateであり、Relation/選択Sourceの書込みではない。元データと図/表の関係は、表示Occurrenceと元Relation occurrenceを別IDで保持する。Graph→Tree時の複数Path・集計時の非単射・隠されたEdgeを、元Relationの不存在/単一性へ昇格しない。

### 共通のread-only InspectionをCLI/CUI/MCPへ投影する条件（正式コマンド／Schema未採用）

AI用CLI/MCPと対人Studioから同じ既存Source/Query/Function Contextを使うため、まずHeadlessで以下を**既存モデルから導出できるか**照合する。

| 照合するもの | 必須情報と失敗時の扱い |
| --- | --- |
| Source/Identity | qualified ID、Role付きRelation occurrence、endpoint identity、元Source位置・Revision、Function/Context |
| 問いの閉包 | Relation kind、Target/Tenant/as-of、Readerが実取得したScope、required Sourceの欠損・partial・hidden・unknown。閉包不明の空結果をabsentへしない |
| Interpretation/Evidence | Declared/Derived/Observedの層、Rule/Operator/Providerの明示Pin、実消費/実Callの独立Evidence、未観測を表示 |
| Projection | Graph Node/EdgeとMatrix cell factが**同じRelation occurrenceを逆参照**できる。変換Loss、Path重複、公開Consumer範囲を明示 |
| 編集 | 一意にsource-ownedなRelationならtyped Proposal候補。多対一/Derived/Observed/非開示/旧Sourceの場合はread-onlyと理由。DragやMatrixセル変更だけでCommitしない |

CLIに新しい名称のコマンドがあることを前提にせず、既存のreader/inspect/operation境界で足りない属性を先に測る。三要素MethodがProperty/Key/Roleに対して追加する意味規則は#1124/#1126のOwner、Plugin選択の意味は#951/#1016のOwnerに残す。Diagram ProfileとGraph Layoutは投影側であり、業務・契約・現在Grantの第二正本ではない。

### 同一Fixtureから導出する独立対照（新Test/Expected正本を作らない）

| Case | 一因子の変更 | 各Viewと判定の独立期待 |
| --- | --- | --- |
| **PX-01** | Provider A/BがQを提供、Bundleは両方含み、選択Bだが実Call A | Graph選択層と実行層が不一致。Matrix selected B / invoked A。同一23計算ExpectedでもD1不合格（#1251） |
| **PX-02** | FunctionはData DのRを宣言、D更新は選択未確定Rule経由 | CRUD表のdirect Rのみ確認済み、derived Uはunresolved。実Uや効果なしと断定しない |
| **PX-03** | Business Step→Function→DataだがOperation Source未読込 | DFD線は表示可能でもStepのCRUD・実Commit/Receiptを作らない（#1125） |
| **PX-04** | 複親DAGをTreeに変え、同じSourceを二Pathに表示 | Table/GraphではRelation occurrence数を保持、Tree表示Node数を実Source行数へ読み替えない |
| **PX-05** | 公開Slice同じままPrivate Plugin/Relationだけ追加 | 公開Graph/MatrixのEdge、列、件数、layout、ARIA、Tool JSON、検索候補が秘密差分で変わらない（TEA T06/#2516） |
| **PX-06** | Source位置だけ移動、Meaning ID不変 | Graph/Matrixの意味投影は再利用候補、旧Source Jumpはstale。新位置再読込の確認を要求（TEA T01） |
| **PX-07** | GraphのComposite Relation線をDragで変更 | Layout-onlyならSource writes 0。意味変更なら複合Proposal、Source版再照合、既存Shared Acceptanceへ（TEA T03/#267） |
| **PX-08** | Pattern/SourceにOperation記述あり、実Operationsに未導出 | Declared/Derived/ExecutableをGraph/Matrixで分離し、Action OfferやGrantを実装済み扱いしない（#2507/#2586） |

**実施順の補強**：前節P0へ「同一Qualified Relation Slice→Graph＋Table＋MatrixのHeadless再導出と、Projection間での元Relation ID/Scope一致」を最初の合否条件として追加する。TEAのData/Function/BusinessとPluginのContract/Providerを**同じ写像機構**で試すが、意味Roleや期待値は互いに流用しない。P1はFunction内複数View・公開Scope/Selection epoch、P2は実Source/Build/Provider Evidence、P3はView Edit→Proposal→Owner受理を既存#2497/#2507/#267へ渡す。**PX-01〜08は今回計画のみ。新Graph/Matrix Renderer、CLI/MCP End-to-end、実D1/DB・Owner意味受理はNOT RUN**。
