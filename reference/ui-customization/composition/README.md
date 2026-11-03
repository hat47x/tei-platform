# UI構成キット（UI Composition Kit）— 階層化Reference

対応: [#997](https://github.com/hat47x/tei-platform/issues/997)。Web/React縦切り: [#1000](https://github.com/hat47x/tei-platform/issues/1000)。**業務仕様テンプレートは別レイヤー**: [#998](https://github.com/hat47x/tei-platform/issues/998)。

## 呼称の再検討

採用候補の第一位は **「UI構成キット」（UI Composition Kit）** とする。プロジェクトに取り込んで変更できる**構成資産の集合**を指し、個々のRuntime実装を指す「Plugin」とは明確に区別する。

| 候補 | 利点 | 問題 | 判定 |
| --- | --- | --- | --- |
| 基本UIプラグイン | 既存の呼称に近い | 単一Pluginとして読みやすく、生成元・配布・Runtime実装・案件差分を混同 | **置換** |
| UIテンプレート | 短い | TEI内部の**業務仕様テンプレート**と紛らわしい。派生元だけに見える | 不採用 |
| UIフレームワーク | 汎用的 | React/Flutter等の技術基盤そのものと混同 | 不採用 |
| UIスターターキット | SIerが改造して使う起点が明瞭 | 初期コピー後も継続するExtension/Delegate/Plugin運用を表しにくい | 内部の一資産の名称 |
| UIコンポーネントキット | 可視UI資産を説明 | Host/Action接続・Prompt・案件差分を含む範囲が狭い | 部分資産 |
| **UI構成キット** | Starter、Framework Bridge、UI Renderer、拡張点、案件Layerをまとめられる | 未定義の新Core kindと誤解されないよう説明が必要 | **採用候補** |

「UI構成キット」は**論理的な配布・構成単位**であって、TEIに新しい固定Core `UIKind` / `CapabilityKind`を追加するものではない。実際のコード・Theme・Renderer・Target Bridge・Project Overlay・プロンプトは、既存Plugin / Source / Build/Target artifactへ適切にMappingする。BundleとPlugin packageと個々のRuntime Capabilityも区別する。

標準配布物の呼び分け例:

- `UI Host Bridge`: TEI Action・Result・AuthorityとWeb/React等を接続する実装。
- `UI Starter`: 案件生成・編集の出発点となるソース。
- `UI Component / Renderer Set`: Form、List、Panel等のUI部品。
- `Project UI Layer`: 案件が所有する差分・構成・Native UI。
- `UI要件整理プロンプト`: 既存資産を検査し案件要件を最小指示から抽出する開発用素材。

## 「継承・移譲・上書き」は別の仕組み

| 操作 | 意味 | 推奨する場所 |
| --- | --- | --- |
| **参照継承（extends）** | ベースKitのUI公開点を引き継ぐ。元ソースを複製せず、宣言された差分だけを加える | **デザイン時・Build時**の明示的な単一親チェーン |
| **組合せ（composition / contribution）** | 独立したUI部品・機能を既存の公開点へ追加する | Hostの**型付きExtension Point**。必要なら子要素複数・順序明示 |
| **移譲（delegation）** | UIはUI描画を、Action/Validation/PermissionはTEI側の**正本と実行契約**を利用する。Native Canvas操作の詳細はNative側に移譲する | Runtime Port／Application Action Binding |
| **明示的な上書き（replace）** | 現在採用されている特定UI公開点のRendererを交換する。置換対象と期待する前実装Identityを宣言する | Project UI Layer / Extension |
| **装飾（decorate）** | 元のUI実装を呼び出し、外側から変更する。元処理を使わない差替えとは別 | UI renderer / wrapper |
| **ソース分岐（fork）** | 契約から外れる大幅改変を案件側のSourceとして所有する | 最後の選択肢。Upstream自動追従は約束しない |

**クラス継承は第一選択にしない。** 多重継承と暗黙の優先順位は組合せ順序依存とUpgrade時の破綻を生む。UI構成キットの初期実装では、単一親の`extends` + **名前付き公開点** + **明示replace/decorate**が第一候補。複数Kitを一つの配布物で扱う場合も、Plugin依存グラフとUI Renderer優先度を自動的に一致させない。将来の複数基底は単一親の受入Evidenceで不足が判明してから設計する。

### 層モデル

```text
TEI Accepted Business Meaning (#998 は候補素材のみ)
               |
      Application Action / Data / Authority  <-- UIからはdelegate、上書き禁止
               |
    Framework/Target Host Bridge [runtime]
               |
  TEI UI構成キット Web/React [source + logical package]
               |
   企業・業界共通UI Layer [optional UI-only]
               |
      案件固有UI Layer [explicit UI-only changes]
               +-- Native component [its own gesture, undo and domain state]
```

Web/React LayerをWeb/VueやReact Nativeへそのまま継承しない。Target/Frameworkと互換majorを照合し、互換性が違えば別のUI実装を選ぶ。一方、Framework-neutralな**Action Contract**は共有できる。

## Referenceでの仮説実装

`resolve-ui-kit.mjs`は**デザイン時**のReference（実Runtime Plugin resolverではない）。Root Kitがvisual extension pointを宣言し、子Kitは次のoperationだけを使う。

- `provide`: 空の公開点にUI実装を渡す。
- `replace`: `expect`に合う現実装を明示してUI実装を交換する。
- `decorate`: `expect`が一致するときに既存Rendererを包む。

各Layerは`id`とexact `version`を指定し、`extends`もexact pinにする。解決時に継承cycle、duplicate identity、missing parent、target/framework不一致、暗黙overwrite、期待する上書き対象の不一致、実装の欠落、未宣言のSlotを拒否する。得られた各Slotには**どのLayerが何を変更したかというtrace**を付与する。

```js
// 概念例（このReferenceは未配布のKit descriptor候補）
root.slots = ['header', 'aside', 'footer'];
company.extends = {id: root.id, version: root.version};
company.changes = [{op:'replace',slot:'header',expect:'standard.header',implementation:'company.header'}];
project.extends = {id: company.id, version: company.version};
project.changes = [
  {op:'decorate',slot:'header',expect:'company.header',implementation:'project.decorateHeader'},
  {op:'provide',slot:'aside',implementation:'project.aside'}
];
// 結果: header=company-headerをproject装飾 / aside=project / footer=未指定
```

Visual Slotの名前は特定の固定enumをCoreに設けず、信頼されたUI Host Kitが宣言する。Rootが宣言していない公開点の利用は拒否する。`action.invoke`や`authorization`をUI Slotに転用する設計は受け入れない。**名称を禁止するだけではSecurityにならない**ため、本番ではFramework Hostの信頼境界とServer側のAuthority/Validationを別途担保する。

**検証:**

```sh
node --test reference/ui-customization/composition/resolve-ui-kit.test.mjs
```

2026-10-08、Node.js v22.16.0で **13/13 PASS**。テストを実行したファイルとRepositoryへ登録したGit blob SHAが一致している:
- `resolve-ui-kit.mjs`: `a852de9751e596ac0460215aa389a1d3f404c267`
- `resolve-ui-kit.test.mjs`: `805fd52a99be9dd77bee329e17566da9bee7c366`

検証したのは明示Layer構成、非破壊、上書き・装飾、exact pin、循環、名前衝突、未宣言点、権限境界の名目分離、新しいvisual slotの拡張可能性。**本物のReactDOM UIに接続した実証、Plugin Lock/Builder、生成AIの変更、実SUI Native Component、性能・可搬性実証は未了**。Referenceの`implementations`は信頼されたProject Registryからの関数であり、未検証のmanifestにある名前から任意コードを実行するものではない。

### 仮説の制限と次のGate

1. Visual Extension Pointの表示契約（ReactElement / Fragment / mobile widgetなど）を**型付き・Framework互換**で検証する必要がある。現在のReferenceはSlot名とJS関数の存在だけを検査する。
2. exact version pinは**artifact digest / Plugin Lock / source fingerprintの真正性を保証しない**。同じversion名を付けた異なるソースや悪意あるComponentはProductionでは別途拒否する。
3. 同じParentを二つのAddonが拡張し、同じSlotをreplaceするケースは、現時点では重ねられない。暗黙の順位づけを設けず、どちらの変更を採用するか**案件が明示**するまでは未解決にする。
4. Project Overrideの上書き対象がVersionアップで変わった場合は黙って追従せず`override_conflict`で止め、AIへ**元の意図・旧実装・新実装・差分**を提示して再生成・テスト・人の採否へ進む。
5. Native UIは複雑なstateやパフォーマンスをUI Kitへ吸収しない。TEI Action Contractへの委譲が成立すればよく、SUI実連携は[#993](https://github.com/hat47x/tei-platform/issues/993)が担当する。

### 参照した先行事例

- [Backstage Frontend Extensions / Extension Overrides](https://backstage.io/docs/frontend-system/architecture/extensions/) — attach point、extension tree、同一IDの明示的な差し替え、pluginのoverride。実装と構成の分離に参考になるが、そのままTEIの安全性モデルとして採用しない。
- [Backstage Frontend Plugins](https://backstage.io/docs/frontend-system/architecture/plugins/) — application-owned override packageの扱い。
- [shadcn Registry](https://ui.shadcn.com/docs/registry/getting-started) — UI部品だけでなくプロジェクトが所有するsource/config/rules等を配る考え方。Source-ownedなKitとRuntime Pluginの区別に参考になる。

これらを踏まえても、**配布単位としての「Kit」／実行機能の「Plugin」／案件の「UI Layer」／TEI内部の「業務仕様テンプレート」**を同一物とはしない。

<!-- tei-ui-extension-upgrade-handoff-20261103 -->
## 2026-11-03 横断追補：Web/Desktop UI拡張と更新追従の設計・実証計画（Draft／NOT RUN）

**目的**：#997 の既存 Kit/Project Layer と #1824 の Wails/Studio、#259 の生成Desktop Target、#951 の Service Contract を接続する。これは #997 の UI-2/UI-3/UI-5 に対する**追加の受入観点**であり、新Core kind、新UI専用Host/Resolver、正式Extension API/ABI、追加Gateの採用ではない。下記の名前・分類は比較用の設計語彙。

### 拡張方式の選択（分離は目的でなく必要条件から決める）

| 方式 | 使いどころ | 所有境界と更新時の扱い |
| --- | --- | --- |
| 宣言的表示・操作配置 | 通常のForm/List/Tree/Action配置・表示条件 | Accepted Meaning/Actionを参照するPresentation側の宣言。業務Rule/権限をUIに複製しない |
| Kit公開点への provide/replace/decorate | 特殊入力、可視化、右Panel、既存UI部品の明示差替 | Root Kitが公開する型付きSlot・期待Implementationを照合。案件ExtensionのSourceと依存を独立所有 |
| Target内の静的コード合成 | WailsのGo Service、React/TypeScript独自部品、OS連携 | Design/Build時に選択・Lock・生成し、Go単一実行ファイル等のTarget成果物へ束縛。実行時の任意Go動的ロードは初期対象外 |
| 外部プロセス／Remote Provider | 別の更新周期、障害隔離、専門Native Library、既存外部アプリ | 既存Service/Transport契約による別配置。起動/再接続/資格/通信断/不明効果を検証。ブラウザ内で任意ローカル実行ファイルの起動を前提にしない |
| 案件Source fork | 上記公開契約で扱えない大改造 | Upstream自動追従を約束しない。変更の所有・再適用・再テストは案件が明示的に引き受ける |

**二つの合成を混ぜない**：Kit Root→企業→案件の visual contribution 解決（本Reference）と、Contract→Provider→Target Artifactの選択/実装合成（#951/#1016/#1223）は異なる操作。KitのSlotは認可・Data Provider・業務Actionをoverrideできない。標準RendererもカスタムRendererも、公開契約に接続するという点で対称だが、すべてのUIを同じDOM・gesture・ABIへ強制しない。

### 最小公開面（型の候補。正式Schemaではない）

- **Kit側のSlot宣言**：slot identity、提供可能なContribution役割、Target/Framework範囲、表示データの型・粒度・NULL/欠損/非開示/partial、配置・順序・mount/unmount/dispose、focus/keyboard/IME/a11y、theme/localization・CSS衝突範囲。Slotは信頼されたKitが宣言し、未知Slotや曖昧な複数replaceを拒否する。
- **Extension側の提供宣言**：extension ID、publisher/source provenance、exact parent Kit ID/version + content digest、対象Slot、provide/replace/decorate、旧Implementation期待ID、コード・資産・CSSのfingerprint、framework/toolchain/runtime依存、Target、必要なOS/通信Capability、公開するUIイベント、登録・解除処理。
- **Interaction側の既存契約を参照**：Model/Meaning/Action identity、Source/Model/World revision、Data Context instanceとselection epoch、grain/completeness/scope、Validation/Diagnostic、Request/Result/Failure/Unknown。UIで表示されたOfferやenabledは現在の実行Grantではない。実Request時にHost/Providerの資格、対象Key、expected stateを再確認する（#1158/#1127/#2500/#2507）。
- **境界**：描画/局所Preview・Drag・Undoの途中状態はUIが管理してよいが、Accepted Meaning変更、Source Accept、業務保存、実Provider CommitをSlotの直接操作にしない。Actionを通じても送信ACKとCommit/独立Observationを混同しない。
- **能力と安全**：同一WebView/同一Originの任意スクリプト追加は安全なsandboxではない。WebViewのBridge公開、非信頼HTML/外部Navigation、OS File/Process/Secret、CSP/asset/URL、Browser/APIのIdentityをTarget別に点検。別Process化も自動的な権限隔離や業務保証ではない。

### 更新手順：旧版の無条件追従ではなく三者比較

1. **現行固定点**：旧Root Kit/Framework/Extension契約・版・fingerprint、案件Project Layer（独自Source・Slot contribution・依存）、現在の生成/配布Artifactを記録する。名称/semver一致だけで内容・意味の同等性を推測しない。
2. **候補固定点**：新Root Kit/Renderer/Bridge/Toolchainの実物と公表契約を固定し、旧Kit→新Kitの差と、旧Kit→案件差分を別々に導出する。
3. **静的照合**：Slot欠落、Slot shape/lifecycle/意味Binding変更、旧Implementation ID不一致、依存/CSS・登録順衝突、Target/Go/Wails/Framework不適合、Capability増加、禁止された直接DOM/内部API依存を検出する。判定不能は *unverified* とし、自動fallback・silent rewriteを行わない。
4. **隔離された候補Build**：案件Extensionの原Sourceを保持し、新Kit上に再合成する。Generator/Bindingの変更と案件変更を別diffにし、対象Targetごとのbundle・binding・dependency closure・artifact identityを保存する。生成器自身のPASSを独立Evaluatorや実UIのPASSとみなさない。
5. **試験・受理**：元の同一Actionと独立Expectedで、標準Renderer／案件Renderer／Nativeを比較。a11y・IME・状態復元・focus・partial/denied/unknown/commit観測を確認し、必要に応じ実Wails/Web browser E2Eを別Evidenceとする。
6. **選択**：Compatibleと宣言された更新でも実行証拠不足なら本番への自動切替はしない。衝突/未検証時は**現在の動作確認済み固定版を維持**し、候補更新だけを保留・説明する。案件の上書きや意味変更は人のレビュー/Owner受理へ戻す。旧成果物を削除・上書きしない。

更新診断の最低区別は、(a) 物理Source/依存の変化、(b) 公開UI契約の変化、(c) Meaning/Action bindingの変化、(d) Build/Target実行可否、(e) 実画面の振舞いの未実証。契約一致・Type一致・Build成功を業務的な同等性と取り違えない。Referenceテストの 13 PASS はこれらの実証に転用できない。

### 最小差分Fixture（既存UI-2/UI-3/UI-4/UI-5へ配置）

| ID | 元の機能と意図的変更 | 独立した期待／拒否 |
| --- | --- | --- |
| UX-01 | 受注標準Formに右PanelのカスタムRendererを追加→標準Kitの内部DOM/CSS構造だけを変更 | 公開Slotが安定なら案件Source/Action意味を維持。内部DOM selectorを直接読んだ実装は契約外依存として検出/更新保留 |
| UX-02 | root Slot移動/削除、旧Implementation ID差替、2 Addonの同一Slot replace | 黙った勝者決定・二重登録・案件差分消失を拒否。変更位置/必要な選択を診断 |
| UX-03 | 同じ表示型の金額を税抜→税込へ変更、Selection epoch失効、Data Context partial | 表示の型が同じでもMeaning/単位/粒度/時点の受理を再利用しない。古いOfferの保存を拒否 |
| UX-04 | Wailsアプリへ案件Go Serviceを静的合成してBinding生成→Kit/Service更新 | Go側の契約/生成TSバインディング/Asset/Capability閉包を固定し、再ビルド。Wails起動だけでAction/OS認可合格としない |
| UX-05 | Native Canvasのpointermove→cancel、drop時だけAction | cancelでは永続変更なし。dropで対象・revision・Grantを再確認。高頻度gestureをGo RPCへ強制しない |
| UX-06 | 外部Processの応答喪失、ACK後のCommit不明、旧Process残留 | 自動再Submitしない。観測/復旧と子Process lifecycle・秘密・資源を別証拠にする |
| UX-07 | Web/ReactとWails/WebViewで同一Action・2種類のUI配置 | Business result/validation/failure/unknownを比較し、環境固有OS機能はunsupportedと表示。Studio E2Eを生成Desktop E2Eへ転用しない |

**優先順**：UX-01/02を #1000 の実Reactの第一Referenceに追加 → UX-03を既存Data Context/Action Fixtureで確認 → UX-04/07を #1824 PoC B と #259 の**異なる対象**として比較 → UX-05は #993 の実装状態を前提に再利用 → UX-06は既存Process/Effect Receipt側へ委譲。少なくとも2種類のApplication/Consumerでの再利用が示されるまで共通Extension ABIは確定しない。

**Ownerの未決判断**：Studioと生成ApplicationのKit共有範囲、最初の実Framework/Native例、どのUI公開点を安定保証するか、互換変更時の要求Reviewとサポート期間、プロセス隔離に必要な具体的脅威モデル。ここでは新Issue/ADR・正式Policy・本番採用を行わない。

関連担当：[#997](https://github.com/hat47x/tei-platform/issues/997)、[#1000](https://github.com/hat47x/tei-platform/issues/1000)、[#1824](https://github.com/hat47x/tei-platform/issues/1824)、[#259](https://github.com/hat47x/tei-platform/issues/259)、[#951](https://github.com/hat47x/tei-platform/issues/951)、[#1158](https://github.com/hat47x/tei-platform/issues/1158)、[#943](https://github.com/hat47x/tei-platform/issues/943)、[#2391](https://github.com/hat47x/tei-platform/issues/2391)。

<!-- tei-ui-extension-crosslane-20261103-b -->
## 2026-11-03 再深化：UI拡張の寿命・用途相対互換性・ビルド配置（計画追補／NOT RUN）

この節は直前のUX-01〜07を置換せず、他レーンの **#1127/#2500/#2507/#2582/#2598（Context/Offer/Intent/失効）**、**#1130/#1167/#1182（Function起動と単一/複数Build）**、**#1174（実Effect）**、**#2391（外部UI制御のビルド時除去）**をUI-3/UI-5の**受入条件**へ具体的に接続する。新しい中央Registry/Host/Resolver、万能UI ABI、専用Build Engineは作らない。

### 「拡張できる」と「更新できる」を同じ判定にしない

互換性は利用目的と実証範囲を明示する **Claim** として扱う。公開Slotが存在する／型が同じ／親Kitのsemverが同じ／React buildがPASS、という個別事実から、Actionの意味同等性、現在Grant、WebView安全性、業務Commitの同等性を推論しない。

| 変更と用途 | 静的に再利用できる候補 | 再確認が必要な境界 |
| --- | --- | --- |
| 親Kitの内部DOM・スタイル変更、公開Slot/Action契約は維持 | Slot identity/Bindingの旧比較結果（exact dependencyが不変である範囲のみ） | 実ブラウザのvisual/a11y/IME/focus/CSS、未公開DOMへの依存、描画performance |
| Slotのmount順・life cycle・data props・旧Implementation ID変更 | 変更が及ばない別Slotの限定証拠 | 対象Extensionのdispose/再購読・状態再初期化、型だけでなく意味Role・grain・scope |
| 同型データのMeaning/Rule/Context/Provider/取得範囲変更 | source不変な無関係RendererのBuild artifactなど | Offer/Intent適用条件、selection epoch、NULL/missing/withheld/partial、現在の実行先資格 |
| Wails/Go Bridge/Serviceの版、Build Unit配置、Target Toolchain変更 | 受理済みBusiness Meaning自体（意味Sourceが不変の場合） | 静的リンク・bridge公開範囲・Dependency closure・IPC/HTTP経路・Artifact実物・OSごとの機能 |
| 実行時のUser/Role/Policy/Resource/Source/対象版失効 | 変更前の履歴的な説明資料 | 現Request/Host Grant/Provider target/Expected-stateとEffectの独立Observation |

**決定単位**は「Kit全体のPASS」ではなく `(Target, Slot/Extension, Context/Action use, artifact/build, environment)` に相対的な根拠付きの判断。結果は、明示根拠のある**再利用可能／再検証必要／衝突／不足・不明／非対応**を区別し、競合/不明を成功として自動昇格しない。これは既存のIssueで扱う更新診断の観点であり、新共通status enumではない。

### 状態・非同期処理・再描画の寿命

UIの編集・起動・保存には少なくとも次の異なる寿命がある。

- **Renderer寿命**：mount→update→unmount、Event購読、Timer、Drag selection、focus/IME、破棄・再購読、複数pane/window。置換Kitは登録された購読・Resourceを解除し、旧Rendererから後着したイベントを新Instanceへ無条件適用しない。SUI Canvasの高頻度pointer/undoはNative側に保持する。
- **Editing/Context寿命**：Function Context InstanceとResult Slice/取得Scope/epoch、未送信Draft・validation結果・unsaved edits。Kit更新はData Sourceへの保存でもDraft破棄の許可でもない。保存すべきDraftの所有と復元資格は既存Editing Session/Source側へ委譲し、Renderer内部local stateと区別する。
- **Operation/Effect寿命**：Save Intent、送信済Request、応答喪失、Provider Commit、Receipt、独立Read/Observation。画面更新・再接続・二重click・timeoutで**副作用あるSubmitを再実行しない**。同期的なUI連打抑止は必要でも、Host側idempotency/現Grant/実Txの保証ではない。

具体例：受注D1を編集中に親Kitを更新、非同期Read(D1旧epoch)の返答が後着し、利用者はD2へ選択変更済み。旧返答がD2の表示を上書きしたり、古い対象D1に`submit`を実行したりしてはならない。保存状態が「通信断で不明」のままUIが再起動した場合、前回の未確定Requestと新規Submitを区別し、既存Owner/Providerへの照会に進む。新たなReact固有の第二保存Engineを作らない。

### UI配置と物理Build配置は別軸

UIが **Function/Actionの論理Identity** を参照することと、その実装がどのGo binary/Browser asset/API/外部Workerへ入るかは別。#1130の単一Build既定・必要時だけ複数Build指定を採用候補として接続し、UI Extension側に`Function -> binary`の第二正本テーブルを作らない。

```text
Accepted Business Function / Action
      |             └─ CLI / cron / Web / Desktop からの論理Entry（公開とGrantは別）
      v
Application Interaction Contract
      |
UI Kit: Web renderer / Wails renderer / custom component
      |
Target/Build Profile（単一Unit既定、必要な時のみsplit intent）
      |                     |
 Web UI assets          Desktop executable (Go+Wails)
      |                     |
 API/Provider Unit <----+   or external Worker when explicitly selected
```

- 単一Buildの時点では利用者に全Functionの配属を入力させず、既存Composition/Builderの導出表で表示する。複数Buildへ分割するときだけ、既存Function IDと必要Entry/Target/配置Intentを管理する。Functionの業務所属・Meaning・Ruleは変えない。
- Webフロントの静的資産にDesktopのGo Service、DB Provider、秘密、Editor/テスト専用MCPを混入させない。ビルドに含むことと、起動入口を公開することと、現Grantと、Effect確定は四つの別事実。
- UI Kitをv2へ上げてもAPI Unitがv1のままという**混在配備**や、ブラウザキャッシュの旧JSが新APIを呼ぶ経路を、単一のApplication model revisionで隠さない。UI asset/API/desktop buildそれぞれの実物・契約世代を照合し、不適合なら実Operationを停止し、旧配備の利用可能性は配備側で判断する。
- 調整対象は増分Buildの依存閉包（#943/#2600）と既存Function→Build/Entry計画（#1130）。UI配属変更だけでAccepted Meaningを改訂したことにしない。別Unitを横断する同一Transactionは自動成立しない。

### Trust Boundary：Kit Slotはサンドボックスではない

案件が信頼して静的合成したReact/TS/Goコードであっても、**任意JSを同じWebViewへ導入すると同じOrigin/Bridgeの権限を共有する恐れ**がある。manifestに`required capability`と記したことはOS権限強制ではない。信頼できないExtensionは単純なSlotへの挿入を許す設計にせず、必要な場合は実Origin/iframe/IPC/process isolationと公開Methodの絞込みをTarget別に実証する。WailsのBridgeや開発MCPに任意JS実行・Go直呼出しが残れば、UIからのAction delegationだけでは迂回防止にならない。外部制御Test/DevelopmentとProduction artifactの差は#2391で物理除去を独立判定。

### 実証ケース（UX-01〜07へ追加、独立Expectedは#1180を再利用）

| ID | 比較・負例 | 拒否／観測条件とOwner |
| --- | --- | --- |
| UX-08 | Kit更新/複数pane、旧epochのRead/Event/購読が後着 | 古いRendererが新Context/Draftを上書きしない。cleanup/focus/IME/a11yも検証。#1000/#1824 |
| UX-09 | 単一Go binary→画面/Jobを分離、同じFunction IDをWeb/cronから呼ぶ | Business Meaning不変、依存閉包・Artifactと論理Entry/公開Route/現在Grant別証拠。#1130/#1167/#1182 |
| UX-10 | Web asset v2 / API v1、旧JS cacheと新API、Desktop別v1、Slot contractのみ一致 | 混在配備の不適合を診断。UI配置が同じでもRequest/Result/Failure/unknownを失わない。#259/#1182 |
| UX-11 | 外部HTML/不正ExtensionがWails Bridge・試験MCPへ到達 | 実的に未公開／到達不可か検証。依存/Asset/公開Methodの監査。Production制御面の物理除去。#1824/#2391 |
| UX-12 | 二重Submit、timeout/ACK喪失中のUI再起動、旧Requestの再送 | フロント同期排他とHost冪等性/Receiptは別。未知Commitを失敗や成功へ丸めず自動再Submitしない。#1000/#1174 |
| UX-13 | Role/Provider/Source Scopeの変更、0→1→0件のABA、Contextはpartial | 旧Offerの実行資格・不存在推論を再利用しない。無関係Slotは過剰失効させない。#2507/#2582/#2598 |
| UX-14 | 親Kitは同version・違うSource digest、または拡張PluginとToolchain版だけ更新 | Lock/source fingerprint/actual loaded artifactの不一致で更新を保留。Semverや成功した旧Buildの証拠を転用しない。#943/#951/#997 |

**最小進行（既存Issueの段階を維持）**：P0（read-only差分・依存・Failure票）→ P1（#1000の実React Form＋案件Panel、UX-01/02/08/12）→ P2（同一Headless ActionのOffer/Intent/独立Expected、UX-03/13）→ P3（#1824の実Wailsと #259 の生成Desktopを**別成果物**で、UX-04/11）→ P4（#1130の単一/複数BuildとBrowser/API別配備、UX-09/10/14）。#993 Nativeは別の独立Pressureとして扱う。最初のP0/P1で新UI kind/Extension ABI/Service APIの正式化を求めない。

**測定する値**：同一Scenario/Target固定下の案件固有変更行・変更ファイル数、手動追従回数、AI指示量・追加質問数、実UI/実Actionの修正工数、増分Build対象数・時間、Artifact byte・起動負荷、Listener/Bridge公開面、未許可・不適合の拒否と誤った自動再実行の件数。数値閾値や性能改善は実測後に判断。P0机上照合、Node合成PASS、Go局所PASS、実Browser/WebView E2E、実Provider Effect/Receipt、Owner意味受理を**別Evidence**として記録する。

**判断保留**：UI拡張の長期安定公開点をどこまで約束するか、Render-state移行の製品保証、Kit更新時の利用中Sessionの扱い、Untrusted addonの許容範囲、生成DesktopとStudioの共有Kitの範囲。上流が壊れないと宣言しただけで自動更新を採用しない。
