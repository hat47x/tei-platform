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
