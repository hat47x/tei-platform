# Plugin-first D1：Authoring Inspectorの実Provider選択・呼出しへ到達する最短Fit/Gap

- **状態**：現行Go実ソースの静的監査と実装着手票（2026-11-07）。設計候補であり新API/ABI採用ではない。**新しいGo build/test、実CLI/Wails、Current Grant/実Provider/実Target E2Eは NOT RUN**。
- **基線**：`hat47x/tei-platform` main `d6a452f79023744e7926669c3556d2699f546697`。GitHubで以下の実ファイルを個別参照した。行番号はその基線での目安であり実装時に再取得する。
- **既存正本**：[Plugin-first全体設計](../design/PLUGIN_FIRST_TOOLING_RUNTIME_DOGFOOD_2026-11.md) §4、§8〜13、[ADR-0032](../adr/0032-composition-kernel-replacement.md)、[ADR-0043](../adr/0043-target-native-artifacts-without-tei-host.md)。実装Owner：[Plugin Core移管 #1251](https://github.com/hat47x/tei-platform/issues/1251)、[Contract #951](https://github.com/hat47x/tei-platform/issues/951)、[Profile #1016](https://github.com/hat47x/tei-platform/issues/1016)、[Typed Adapter #1158](https://github.com/hat47x/tei-platform/issues/1158)、[Studio #1824](https://github.com/hat47x/tei-platform/issues/1824)、[Context #3262](https://github.com/hat47x/tei-platform/issues/3262)。
- **本票の責務**：既に計画された「CLIとStudioの同一Inspector Pluginを呼ぶ」を、現コードでどの入口を変更し、どの入口を流用してはならないかまで絞る。新Resolver、Registry、Lock、Test Runner、Authoring Host、第二意味正本を作らない。

## 1. 実ソースで新たに特定した決定的なGap

| 位置 | 観測した事実 | 実装上の拘束 |
| --- | --- | --- |
| `cmd/tei/main.go:79-87,138-143,342-350` | CLIは`isAcceptedCommand`を先に検査し、`inspect`だけを特別扱いで`inspectWorkspaceJSON`へ分岐する | 最初の移行は`inspect`一経路に限定。Plugin追加のため手書き分岐を増やすのではない |
| `cmd/tei/cli_inspect.go` | `operations`を直接使うCLI用結果投影がある | 互換Output/終了コードの保持とProvider dispatchは別に検証する |
| `cmd/tei-studio/main.go:27-45` | Wails `StudioBridge.InspectSubject`は`studiohost.ReadOnly`へ委譲する | ブラウザからProvider選択情報・Trusted Rootを受け取らない |
| `internal/studiohost/readonly.go:58-75,87-105` | `RootServices`とHost提供`ServiceFactory`があるが、現行は`Catalog`と`Validation`の注入に限定 | Host allowlist解決後にサービスを渡す**既存seam**を再利用。任意PathをServiceへ渡す新APIを作らない |
| `internal/studiohost/readonly.go:231-241` | `InspectSubject`はrootを検査後に`operations.InspectSubject`へ直行し、登録済みProviderを選択しない | `inspect`をHost所有の選択済みOperationへ委譲する必要。共通Go関数の利用だけではPlugin-firstにならない |
| `internal/operations/inspect.go:30-94` | `SubjectInspection`はWorld/SourceIndex fingerprint、sourceStatus、Canonicalを返す。World fingerprintはrevision token/acceptance grantではないと明記 | `inspect`成功＝Required Domain Closure完全・現在の開示資格・CAS済みではない。#3262と分ける |
| `internal/composition/registry.go:12-54` | ContractはID/Version/Shape、ProviderはID/Scope/Offers/Requiresを持つ。選択はRuntime権限付与ではない | 候補Providerの一意性・exact Scope・Dependency Epochを既存Compositionで解決する |
| `internal/composition/bootstrap.go:1-32`、`coordinator.go:39-64` | `NewSelectedCoordinator`は選択されたRegistryとHost guardを必要とし、`Acquire`で両者を検査 | 実Call前にHostのTrusted Rootと現在の操作可否を照合するための足場。ただしBuild Artifact/正式Conformance/現在GrantをCoordinatorがすべて証明するわけではない |
| **`internal/plugin/runtime/service_carrier.go:76-92`** | **`validateServiceSelection`が`Scope != "application-runtime"`を拒否する** | **既存`InvokeSelectedServiceCarrier`をそのまま`authoring-tool`へ使う実装は失敗する。文字列置換だけでScopeを緩めない** |
| `internal/plugin/runtime/model.go:516-588` | 旧Runtime RegistryにBuiltinRegistrationとArtifact/Engine固定の照合がある | これをそのままAuthoring用Universal ABIに昇格しない。必要な証拠のAdapterに限定する |

**結論**：論理CompositionのScopeは一般的である一方、**実行へ接続するLegacy Service Carrierはapplication-runtimeに限定**される。旧Carrierへ`authoring-tool`を通すためにScope検査を緩める変更は、Trusted Host/Contract Acceptance/Provider Conformanceの抜け道になり得る。D1はこの境界を尊重し、**Host-owned Authoring Operation Adapterを既存Compositionの選択へ結ぶ**小さな縦切りを先行させる。これも独立Runner/Authorityではない。

## 2. 最小の実装差分（既存コードを生かす）

1. **Contract・Required Contextを外部正本から引用**：当面は既存`tei.operations.inspect/v0`の正確なRequest/Resultを固定し、`Subject kind/id`・Exact Root・World/Source Index fingerprint・失敗の集合を記録。Required Closure、`expectedRevision`、Disclosure、`complete`が現行Operationに存在しない部分は**補助診断または未適格**とし、根拠なく既存v0が保証すると言わない。公開Contract版・ConformanceのOwnerは#951/#3262。
2. **共通Inspector Operation Portを最小化**：`Inspect(trustedRoot, typedSubject) -> typedInspection/error`相当のHost内部Portは既存`operations.InspectSubject`を実装Providerとして包む。正式API名・Go interfaceの配置はD0の依存を確認して決める。Portの存在自体はResolver／契約Publisherではない。
3. **Hostが選んだBinding**：`composition.Registry.Select`と`NewSelectedCoordinator`の既存Referenceを使う場合、`Contract Shape`・Provider ID・Scope・Epochを検証した後に、Trusted Hostだけが実装関数を注入する。Providerから`Root`やGrantを自己発行させない。`Registry.Select`は正式`Provider Conformance`でもBuild実体検証でもないため、D1のAdmission票に未充足を別記する。静的配布ではBuild対象の実Providerを選択済みに固定。
4. **CLI**：`inspectWorkspaceJSON`の内側で、引数とJSON Envelope/exit codeを保ちながら、選択済みOperationを呼ぶ。最初は旧直接経路とread-only shadow比較、切替後は旧経路への自動fallbackを拒否。
5. **Studio**：既存`ReadOnly.workspace(id)`のallowlist／root identity検査後、Host設定の選択済みInspectorを呼ぶ。`RootServices`の拡張・Host-only Adapterのどちらが変更最小かをD0で比較。Renderer/Wails bridgeはOperationを要求するだけでContract/Provider選択・Resource権限を持たない。
6. **実呼出証拠**：`Selected`と`Invoked`を別記録にする。`Provider ID+Contract Shape+Scope+Build member/Artifact ID+Source Revision+Invocation Attempt`を一致検証し、戻り値が偶然同じでも「Bを選択・Aを呼出」は不合格。現在のProvider/Artifact読取りが実装から取得できないときはunknownとする。
7. **移行の完了**：実二ConsumerのProvider実Call、否定試験、実選択Build、Host deny、同じSource/Meaningと同じ情報欠損診断がそろって初めてD1b完了。Studioの画面描画やCLI出力だけではない。D1b合格後にHelp/Studio Contributionへ一般化（D2）する。

**非採用**：`service_carrier.go`の`application-runtime`判定を`authoring-tool`も許すように単純変更すること、`StudioBridge`へProvider選択/生のPathを追加すること、`operations.InspectSubject`直呼出のままDescriptorだけPlugin登録すること、全Target共通の常駐Hostを作ること、テストHarnessだけで未接続のFunctionを「実Invoke済み」と言い換えること。

## 3. 独立Expectedを伴う最小Acceptance Packet

| ID | 固定入力/操作 | Expectedと観測される主体 |
| --- | --- | --- |
| **AS-01** | same source、同Scope、A selected/A actual calledをCLIとStudioで個別実行 | 両call-siteの実Provider invocationが観測され、Result/Failure/Source資格が一致。単一Mock結果再表示は不可 |
| **AS-02** | B selected、実AdapterがAを呼ぶ。A/Bの返却JSONはbyte同一 | **FAIL**。SelectionとActual Provider/Artifactが一致しない。結果の同一性で上書きしない |
| **AS-03** | 別Scope A/B、同Contract/ID、Host tool only許可 | application-runtime Providerはauthoring-toolへ暗黙流用不可。旧Service Carrierの拒否維持 |
| **AS-04** | Studio browserからroot path/Provider指定、または起動後にWorkspace実pathすり替え | allowlist/OS root identityによる拒否。戻り値のSource/パス/未公開存在の漏洩も検査 |
| **AS-05** | Providerなし、Provider同一適格候補二件、更新後のstale Epoch | missing/ambiguous/staleの理由つき拒否、CLIの旧直呼出へfallback不可 |
| **AS-06** | `operations.InspectSubject`が`sourceStatus:not-indexed`、または必要Source未読込 | 「未索引」「不完全」と「Subject不存在」「完全」を区別。#3262 Required集合が完結するまでは`complete`を主張しない |
| **AS-07** | stdout/UI/Harnessで内容を途中切断、Receiverがpage/end未受領 | 実Provider成功とConsumer受領完了を分離。8,192byteの成功応答でも完全としない |
| **AS-08** | Default Traitの実装なし/Manual Trait未実装/Advisoryのみ | 必須義務の未履行を拒否。Hintで実行資格を与えない（#3278） |
| **AS-09** | 同一Bundleに未選択Providerを追加 | 実Build/Link inputと実Binaryを照合して混入を検出。静的Goでは再Buildが必要 |
| **AS-10** | Provider出力同じ、Source/Grant/ScopeまたはArtifactのみ変更 | 必要な再資格化と不要失効を分離。Source fingerprint一致だけで現在Disclosure/Authorityを復活させない |
| **AS-11** | 同Contract Provider A1/A2×Consumer C1/C2 | 各exact tupleの実Target/Build/Call/ExpectedだけQualified。Pairwise全被覆でも未実行高次組合せは保証なし（#3267） |
| **AS-12** | CLI`view`のread-only広告と、実行Optionに依存した書込みの対照 | Command広告だけで実Effectを保証しない。Invocation variant/実副作用を観測（#3182） |

**実験順**：AS-03（既存scope隔離を壊さない）→AS-01/02（D1の中心）→AS-04/05（Authority/Scope）→AS-06/07（情報完全性）→AS-08/09（Traitと物理Build）→AS-10/11/12（横断）。既存#1180 Scenarioと#2514独立Expectedを参照し、本票の行を新しいScenario SSOTとして使わない。特にAS-02は**出力一致では見逃す偽陽性**を直接反証する。

## 4. 横断計画との依存関係・証拠の受け渡し

```text
#1251 D0: actual source/import & scope audit
         |
         v
#951/#1016: exact selection and typed Provider binding
         |
         v
#1158/#1824: CLI and Studio separate actual call-sites
         |        (AS-01/02 + host deny)
         v
#3262: required context and qualified read completeness
         |
         +--> #3278: Trait required fulfillment
         +--> #3267: Verified BOM exact tuple
         +--> #3270: Data Workbench read & schema-apply
         +--> #3274: HTTP error graph projection
         +--> #2831: actual CLI/studio self-build
```

**必要条件の方向に注意**：D1の基本的な実呼出を成立させるために、完全な#3262 Design Context Query、正式Trait汎用Schema、検証済BOM製品化、実DB DDL、Studio全GUIを先に完了させない。後段で同じPort/Contract/Evidenceを圧力試験に使う。逆に、D1だけでSource/Contextの完全性、BOMの組合せ資格、Current Grant、実DB/Target効果が証明されたとも言わない。

**次の実装変更に対して要求するEvidence**：exact GitHub commit・実Go toolchain・実ビルド対象・Source closure・Contract/Provider Selection/Scope/Epoch・CLI/Studio call-site・actual Provider entry・Result/Failure・Consumer receipt・Authority/Disclosure decision・独立Expectedを1 Attemptへ結ぶ。実装者が提供できない列は`unknown/not-run`で保全し、空欄をPASSにしない。#2831の再帰Self-build L1/L2/L3とは別資格とする。

## 5. 状態と採否

この時点では**静的コード検査のみ**。Scopeの既存拒否、Studio/CLIの直呼出、Operationのfingerprint項目、Compositionの既存APIは確認したが、AS-01〜12の新規実行はしていない。並行するDraft [#3280](https://github.com/hat47x/tei-platform/pull/3280)（Desktop固有）、[#3282](https://github.com/hat47x/tei-platform/pull/3282)（Trait）、[#3292](https://github.com/hat47x/tei-platform/pull/3292)（Master PlanのD1）と、提案内容を分担し、先行Draftの完成を推論しない。旧`service_carrier`は今のままapplication-runtime専用のまま保持し、authoring専用Contract/Providerの実接続は未実装として扱う。既存Source/Meaning/Runtime AuthorityのOwnerと現在資格を保存し、**二重Contract/Registry/Host/Runner/Lockを追加しない**。

---

## 6. 2026-11-08横断実装差分：Plugin選択・Capability選択・実Providerは別の事実

**基線**：main `6cf510d6301a1e133ebbe69ad7bf004f03361b54`。この追補はPR #3300のAS-01〜12を置き換えず、11/7〜8に実装・議論が進んだ他レーンを**Inspector D1の実際の前提**へ取り込む。Source、Observed、Selected、Invoked、Qualifiedの単語を同じ意味に使わない。

[PR #3337](https://github.com/hat47x/tei-platform/pull/3337)でmainへ入った`tools/reference-plugin-capability-coverage/`は、`World.Plugins[*].Provides`と`Environment`のBindingを使い、`Capability Kind + Capability ID`の**宣言上の選択**を調べるread-only Referenceである。確認済み`coverage.go`は`Selected/Unselected`と`ruleFingerprint`を返す。**実Providerの選択、Contract Conformance、実Link・実Invocation、Current Grantは実行しない**。これはD1の宣言の棚卸しには使えるが、選択済Inspectorが呼ばれた証拠ではない。

特に現行`internal/plugin/model.go`の`CapabilityKind`は固定列挙であり、Inspectorの新Contractを`editor-intelligence`等へ便宜的に押し込むことや、D1のために固定enumを増やすことは目標#951に反する。**InspectorはTooling ScopeのContract/Providerとして扱い、legacy Kind/ID行は必要な範囲でのみ橋渡し**する。

| 観測の段階 | 既存の確認入口 | 満たしたとき初めて言えること | まだ言えないこと |
| --- | --- | --- | --- |
| Plugin declared / installed | `World.Plugins`／Catalog | 定義がある | その能力が要求・選択された |
| Capability selected by Environment | `reference-plugin-capability-coverage`、`Environment` Binding | legacy Kind+IDが宣言に一致している（対象Bindingに限る） | Plugin全能力が選択された、実Providerが確定した |
| Contract/Provider resolved | #951/#1016のexact Contract/Scope/Version/Epoch | 一つの実装候補への決定論的選択 | Buildに収録された、適合・権限が確認された |
| Build/Load actual | #943/#1089/#1167、対象Go module・Artifact | 対象の実装・実Sourceが当該Artifactに含まれる | 実呼出された、Grantを得た |
| Invocation actual | CLIとStudioの別call-site → Provider entry、Attempt | 選択した実Providerが当該要求で呼ばれた | Required Source完全、現Host Grant・Effect・外部Consumer受領が成立した |
| Qualified use | #3262/#2514/#1174/#2516/#3267 | 対象Claim・Scope・Target・Expected相対で支持された事実 | 他Host/版/組合せへ自動継承できる |

**実測上の修正履歴に注意**：[初回Reference Evidence](PLUGIN_CAPABILITY_COVERAGE_REFERENCE_2026-11-08/README.md)は12能力のうち6をunselectedとして記録したが、そのうち`database-provider`2件は`database-instance.Provider=postgres/oracle`という**技術名**とCapability ID `database.postgres`等の文字列比較不能に由来する数え方の限界。現在の`coverage.go`では、既存Validationと同じ`Capability.SupportsDatabaseProvider`で技術名から選択を導出するよう変更されている。従って**初回の6という件数を最新ルールの現在値として再利用しない**。現在値は同じSource World・RuleFingerprint・Toolchainで再実行し、生データとともに比較する。さらに`editor-intelligence`はPluginとして選択されても能力Bindingがなく、`migration-provider`も同じPlugin内の別Capabilityだけが選択され得る。「Plugin選択済み⇒全Capability選択済み」という推論を拒否する。

**D1への適用**：Inspector Contractを選ぶとき、(a)要求したOperation/Scope、(b)Provider Offer、(c)実選択、(d)実Build、(e)実Invocationを**別欄**に記録する。Legacy Capability Coverageが`unselected`でも、汎用Tooling Contractの未提供・失敗と即断しない（Coverageの表現対象範囲が異なる）。逆にlegacyで`selected`でも実Inspectorの選択と認定しない。選択なし／曖昧な候補／別ScopeならD1bは停止。既存サービスCarrierの`application-runtime`制約は維持する。

## 7. 「Inspectorの正常JSON」だけでは、Design Contextを読めたことにならない

[PR #3296](https://github.com/hat47x/tei-platform/pull/3296)のDesign Context実CLI Referenceは9応答と16の負例を対象に限定した取得・受領の判別を追加しており、未達領域を`partial`として保持する。これをD1実ProviderのテストPASSに合算しない。最新の[Markdown Source Gap #3346](https://github.com/hat47x/tei-platform/issues/3346)では、`docs/`・`plan/`を`sources`で管理しても、Loaderが`*.tei`しか解釈せず、Markdownは`managedFiles`に載っても`readByLoader:false`、`model catalog/inspect/trace`の意味的対象ではない事実が報告されている。**ファイルが管理対象≠Loaderが実読込≠Parser/Roleが解釈≠Accepted Meaning≠Required Closureが完結**。

新しいInspectorのD1は最初に**現行Loaderが実読込でき、Source Indexへ到達する小さな`*.tei`**を正例に使う。一般Markdown/SHO文章/Raw Item、未採用`*.tei.md`の解釈方式はこの正例へ黙って加えず、[Markdown Authoring #2621](https://github.com/hat47x/tei-platform/issues/2621)のOwnerで並行検証する。対応PluginがないRaw ItemやCanvas occurrenceも保持対象とし、意味不明をデータ消滅と扱わない。

[CLI出力契約Gap #3345](https://github.com/hat47x/tei-platform/issues/3345)/[#3347](https://github.com/hat47x/tei-platform/issues/3347)の`view`専用semanticFingerprintとWorld/Source Index Fingerprintの比較範囲差、`validate`のdiagnostics二重構造・`requirements`のresult配列、standard entries混在を踏まえ、**異なるScope・Schema/ProjectionのFingerprintをbyte同一であるかのように比較しない**。D1の比較は意味契約で選んだVersion、Subject/Source Actual Root、World/Index Scope、Unknown/Partial、実Recipientが受領したpage/endの範囲を先に固定する。Source Indexに現れないRoleを「不存在」と返さず`unobserved/unsupported`として扱う。

Sourceだけを保存・表示するRaw Editorと、Role/Meaningを提示するInspectorを別資格とする。Rawの保全は未対応Pluginのデータ消去を禁止する一方、それだけで意味受理やCLIのテスト自動生成資格を与えない。これはIssue→Design Context→Testの既存#3262/#1180/#2514経路に引き渡す。

## 8. ビルドの実起動主体・ローカル書込・外部Tool受領を混同しない

[PR #3336](https://github.com/hat47x/tei-platform/pull/3336)でmainへ入った`reference-go-build-stage-driver`は、起動側のexecutable digest、子プロセス側のdigest/argv/PID/exit、子が報告したArtifact digestとディスク上の実digestを分離して記録する。局所実験では子ProcessによるGo Buildが観測されたが、`l3Claimed=false`を保持し**Stage1 TEIがStage2 Buildを主導したTEI self-build L3とは主張しない**。D1でCLI/Studio Providerを同じ物理Artifactへ収録したと主張するときにも、`selected provider`と`actual binary/actual invoker/actual invoked process`を別欄にする。root `go test`が成功しても別nested moduleのStudio実Buildまで自動合格としない（#1251 PB-10）。

[CLI実ファイル書込検証 #3333](https://github.com/hat47x/tei-platform/issues/3333)では、`view`がDiscovery上`effect:read-only`でも、`-execute`と`-operation-log`／`-lifecycle-state`によって実ファイルが生成される。mainの`writeFlags`は指定オプションの書込入口を明示するが、Effect語彙の正式再定義ではない。D1の`inspect`自体がread-onlyでも、shadow比較・テスト実行・Hostのログ／キャッシュ／Tool応答永続化／外部Harness送出は**別の作用と権限**。Payloadの中に`read-only`と書かれているだけではOS書込・外部送出禁止の証拠にならない。検証時に監視する対象（Workspace Source／Local Report／Tool Transport／外部Egress）を分離する。

## 9. 横断成熟差分から更新するD1最短Gate（新Hard Gateは増やさない）

| 順位 | 使用する既存資産とOwner | 次の実装者が実際に確認する一点 | それでも未成立なClaim |
| --- | --- | --- | --- |
| **D0-P** | #3337のCapability Coverage、#951のContract、#1016 Profile | legacy Kind+ID、Plugin単位選択、Tooling Contract選択の三つを分け、Inspector Offerが存在しないときは候補・不足を明示 | legacy `selected`だけではProvider適合・Build収録なし |
| **D1-P** | #1251/#1158/#1824、既存Studio allowlistと`operations.InspectSubject` | **Tooling Scopeでexact選択したInspector**をCLIとStudioの二つの実call-siteから同一実Providerへ呼ぶ。B selected→A actualの出力同一負例 | Source Required Closure、WebView GUI画面動作、他Targetは別 |
| **D1-S** | #3262/#3296、#3346/#3345/#3347 | 受信できたJSONと実Loader/Role、Source実path/FP scope、Required集合、Complete/Partialの一致を保ち、未読込を不在と誤読しない | Markdown文書すべてが意味的に読めるわけではない |
| **D1-A** | #1174/#2516、#3333、#2509 | Host ScopeとWorkspace許可、実Filesystem Writeと外部Deliveryの範囲を別に確認し、Pluginの自己宣言をGrantにしない | 本番Host Effect/Disclosure資格なし |
| **D1-B** | #943/#1251/#2831、#3336 | 選択Provider→実Build Source/Module→binary digest→invoker/invoked/Provider callを当該Attemptに束縛 | stage1→stage2再帰Self-build L3、BOM Qualifiedは別 |
| **D1-T** | #3278/#3267/#1180/#2514 | Required TraitのFulfillmentと独立Expected、Provider/Consumer 2×2 exact組合せを採否を分けて記録 | 個別Conformanceや候補Planを合成Verifiedにしない |

**追加の独立Expected観点**（既存Scenario SSOTに委譲し新Runner/テスト定義正本を作らない）：

| ID | 対照 | Expected |
| --- | --- | --- |
| AS-13 | Plugin選択済み、対象Capability Bindingなし | Inspector実Provider選択済みと偽らない。別のCapabilityが選択されても同一視しない |
| AS-14 | Kind違いの同ID、DB provider `postgres` vs Capability ID `database.postgres` | tech名とIDを文字列一致で照合せず、既存`SupportsDatabaseProvider`に沿う。Tooling InspectorへDB方式選択を流用しない |
| AS-15 | 初回Coverage raw=6 unselectedと、改訂RuleFPのSourceからの出力を混合 | stale/mismatchedの比較を拒否。再実行のない現在件数を主張しない |
| AS-16 | 管理対象Markdownだが`readByLoader:false`、Rule宣言は本文にある | `inspected`／`accepted`／`requiredComplete`をfalse/unknownとして保持。非存在とは診断しない |
| AS-17 | `view` semantic FPを`inspect` World FPへ代入、またはSource Index fingerprintだけが一致 | Scope違いのFingerprintを同等資格にしない。同一Subject/実Source/Role閉包を別検査 |
| AS-18 | CLI/Studioで同じProvider結果だがTool replyは途中切断、外部H3へ一部だけ配信 | Provider成功≠Consumer受領完了。外部送出現在資格も独立 |
| AS-19 | Pluginの`read-only`広告、`view -execute`でLocal Journal生成 | 宣言Effectと実OS書込を別検査。ログ生成を業務Source変化とは誤認しない |
| AS-20 | invoker digest≠invoked digestでArtifact bytes一致、またはselected ProviderがBinaryへ不収録 | 参照Build成功／byte一致のみでD1実Provider Link/Call、TEI self-build L3を認定しない |

**P0と並列の方針**：新しい全Source Reader／一括Markdown Parser／汎用Trait実装／Verified BOM製品／Recursive Self-buildの完成を、Inspector D1の先行条件にはしない。D0-P→D1-Pを最短のP0として継続し、D1-S/A/B/Tは同じ実ScenarioのClaimに必要な部分だけ連結する。Issue→DesignテストドッグフーディングではRaw保全・三視点・Function Readを各Ownerで並行させ、Source未読込を「未定義」と扱う誤PASSを共通D1で検出する。

**今回の新規観測の限界**：現行GitHub Source・PR/Issue・過去の限定Reference結果を照合した。D1の実Provider選択→CLI/Studio実Call、上記AS-13〜20の新実行、最新版Coverageの実再計測、外部H3の完全受領・実DB/実WebView・Owner受理は**NOT RUN**。#3337/#3296/#3336/#3333の限定実装・既存実験結果を、今回の追加試験のPASSとして加算しない。


## 10. D1aの実Product直結Baseline：CLIとStudioの同一Source投影（2026-11-08）

**この段階で固定するのは既存の直接Operation経路の互換性だけ**。新しいPlugin Provider経路の成功ではない。mainの実 `cmd/tei/cli_inspect.go` は `operations.InspectSubject` を直接呼ぶ。一方、現在mainの `internal/studiohost/readonly.go` はHost allowlistの検査後に同じOperationを直接呼ぶ。両経路が同じライブラリを用いることと、二つの独立したSurfaceの結果が揃うことを混同しない。

- 新製品Goテスト `cmd/tei/cli_inspect_studio_shadow_test.go` は既存 `reference/workspace` の**Data `sales.order`、Validation `sales.order_line.line_number_positive`、Relation `sales.order.lines`**を同じabsolute Workspaceに対してCLI JSON helperとStudio Host APIの**別呼出し**で実取得する。Source/Subject/Canonical/Workspace Identity/World Fingerprint/Source Index Fingerprint/Operation Effectを、当該既存呼出しの互換基準として照合する。
- 負例では存在しない `semantic-data` Subjectを二つのConsumerへ渡し、CLIはexit 1・`SUBJECT_NOT_FOUND`、Studioも同じ意味の`SUBJECT_NOT_FOUND`を返し、成功や空集合へ縮退しないことを確認する。CLIとStudioのEnvelope形・Workspace Host許可の差はそのまま保持する。
- [#3371](https://github.com/hat47x/tei-platform/pull/3371) は**Draft**としてStudio Hostへ `InspectionService` を注入する候補。Providerが未提供なら旧直接経路へ無断fallbackしない正例・負例は提案済みだが、CLI側にはまだ同じ選択されたServiceがなく、`composition.Registry`のexact Contract/ProviderとBuild/Invokedの証拠束縛もない。既存の直結Parity Testが通っても、**D1bのPlugin-first二Consumer証明は未達**。
- [#3337](https://github.com/hat47x/tei-platform/pull/3337)／[#3350](https://github.com/hat47x/tei-platform/pull/3350)／[#3363](https://github.com/hat47x/tei-platform/pull/3363) のPlugin/Capability Bindingと、[#3336](https://github.com/hat47x/tei-platform/pull/3336)／[#3360](https://github.com/hat47x/tei-platform/pull/3360) のBuild Stage Witnessは**別Fact**。本Parity Testの`reflect.DeepEqual`は実選択Provider Identity、Build member、Plugin Loadや同一Attemptの証明ではない。相互参照する際は既存#951/#1016のResolver、#1180/#2514の独立Expected、#3267のBOM Claim Ceilingを用いる。

**次の実装順（既存D1-Pを進め、新Gateを増やさない）**：

| 段階 | 観測と拒否する負例 | 資格の上限 |
| --- | --- | --- |
| D1a直接Baseline | 同じabsolute WorkspaceのData/Rule/RelationがCLI・Studioで別々に読める／欠落Subjectを両方拒否 | 直結Operationの同一Source互換 |
| D1a Host Service Shadow | #3371のHost選択Serviceが前段Baselineと同等のResult/Failureを返す。Missing Service・wrong Source/Subject・異Schema・書込みEffectは拒否 | Studio Hostの型付きService縦切り。まだPlugin選択ではない |
| D1b実Provider二Consumer | Provider A/Bの選択と**実CLI/Studioそれぞれのcall site→actual invoked Provider**を同一Source/Scope/Revision/Artifact/Runへ束縛。**B selected→A invoked（値が同一）**でも不合格。未選択Providerの物理混入も別に照合 | 選択済みInspectorの実Consumer適合。現在Grant/他Target/意味受理は別 |
| D1+独立Required Test | #3354の11実Reply/32局所Controlsと本3 Subject受領を別母数とし、Function/BusinessのRequired Closure、Not-loaded Markdown #3346、部分／未知、完全な外部Tool Replyを別に検収 | 全Functionの完全性、AI盲検T1-B、C1／G1/G2/G4合格を主張しない |

**受入証拠の現状**：この追補および新Go Testは独立Draftの候補。Go Codeを既存の製品CLI/Studio公開方法で比較するが、**新PR exact-head CI実測前はPASSを宣言しない**。いずれの発見済みOwnerのGateやSource/Accepted Meaningを変更しない。新Kernel/Host/Resolver/Runnerは追加しない。
