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
- 既存機構で足りると分かった対象は新Issueへ分割しない。必要な機構だけを#1016で評価し、隣接する#997 / #998 / #951 / #943 / #988 / #993 / plan/issue/000040の既存担当へ引き渡す。

本書の評価を経て、Application Composition Profileが実際に独立したSource表現を要するかを決める。採用するならADRを別途作成し、Project-owned Source / Plugin Lock / Semantic Lock / Target realizationを第二正本化しない最小境界に限る。
