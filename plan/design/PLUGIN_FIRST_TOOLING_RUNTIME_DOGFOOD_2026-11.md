# Core以外をPluginにする実装方式：CLI／StudioからApplication Runtimeまで同一基盤で検証する

- **状態**：方式設計候補・実装移行計画（2026-11-07）。この文書で正式な新Contract/ABI、全Plugin移行、実Target適合、Ownerの業務意味受理が完了したとは扱わない。
- **静的確認基準**：`hat47x/tei-platform` main `358b8cbf55d825de118b15dd590b892b94f5f24e`。GitHub上の現在のソースと既存設計を照合。ローカルcheckoutのGo/Node実行・実バイナリ構築・CLI/StudioのE2E実行は**NOT RUN**。
- **既存方針**：[ADR-0032](../adr/0032-composition-kernel-replacement.md)（Contract中心）、[ADR-0033](../adr/0033-core-and-plugin-subprojects.md)（サブプロジェクト）、[ADR-0043](../adr/0043-target-native-artifacts-without-tei-host.md)（共通Host非必須）、[ADR-0044](../adr/0044-authentication-not-a-core-concern.md)（認証方式はCore外）。
- **責務の既存Owner**：[#951](https://github.com/hat47x/tei-platform/issues/951) Contract/Resolver、[#1016](https://github.com/hat47x/tei-platform/issues/1016) Composition、[#1251](https://github.com/hat47x/tei-platform/issues/1251) 移管、[#1158](https://github.com/hat47x/tei-platform/issues/1158) UI Adapter、[#1824](https://github.com/hat47x/tei-platform/issues/1824) Studio、[#3262](https://github.com/hat47x/tei-platform/issues/3262) Design Context、[#1180](https://github.com/hat47x/tei-platform/issues/1180)/[#2514](https://github.com/hat47x/tei-platform/issues/2514) Scenario/Expected、[#943](https://github.com/hat47x/tei-platform/issues/943)/[#1097](https://github.com/hat47x/tei-platform/issues/1097)/[#2600](https://github.com/hat47x/tei-platform/issues/2600) Build、[#3267](https://github.com/hat47x/tei-platform/issues/3267) BOM、[#2831](https://github.com/hat47x/tei-platform/issues/2831) 全体Self-build。
- **本書の限定責務**：現行実装のFit/Gap、開発環境Pluginを実アプリPluginの試金石とする構成方法、先行縦切りの検査条件を接続する。**第二のResolver、Lock、Runner、Runtime Authority、Source/Meaning正本、Plugin Registryを作らない**。

## 1. 現行実装の静的監査：目標とはまだ異なる

| 観点 | 現行Mainで確認できる事実 | 目標へのGap／変更の入口 |
| --- | --- | --- |
| Coreの物理境界 | `core/subject`、`core/identity`、`core/effect`、`core/generation`への実ソース移管と旧Aliasが存在する。`core/README.md`はCore単独製品ビルド未成立と明記。 | 基本型や世代Routerの抽出は先行しているが、全Contract/Resolution/CanonicalがCoreへ収束したとはいえない。後続の実依存検査が必要。 |
| Legacy Plugin定義 | `internal/plugin/model.go:16-45`に固定`CapabilityKind`と`RuntimeKind = builtin / process`。`Definition.Requires`はPlugin ID中心。 | 新契約追加時のCore/旧Pluginモデル変更を招き得る。新しいService Contract/Provider主体に移し、Legacyは移行用read-only対応に限定。 |
| Legacy Resolver | `internal/plugin/resolve.go:16-36,67-123`はPackage ID＋compatible majorで候補を解決し、環境のPlugin/Bundleを走査する。 | Target/Scope/Shape/Effect/ConformanceのContract選択と同一ではない。旧Resolverと新Resolverを二重正本で残さない。 |
| 新Composite試作 | `internal/composition/registry.go`にContract/Provider/Scope、`selection.go`にSelection/Epochがある。 | 実製品のすべてのCLI/Studio操作へ接続されたとは証明されていない。Registry試作の存在から製品用汎用Runtime完了を推論しない。 |
| 移行Carrier | `internal/plugin/runtime/service_carrier.go`は契約選択を既存Capabilityに束縛し、現在のHost Acceptanceを呼出時に検査するが、将来のContract Resolver自体ではない。 | 移行時の互換経路として有用。旧Carrierを標準的なUniversal ABIに昇格しない。Shape・Publisher Authority・Conformanceを独立に確認。 |
| CLI | `cmd/tei/cli_commands.go:10-93`の静的`acceptedCommands`と`init`追記、`cmd/tei/main.go`の分岐ディスパッチを利用。`cli_contract.go:42-67`は選択されたread-onlyの手書きCatalog。 | CLI機能は現状、共通Contract/Providerから導出されたPlugin Contributionではない。最初は既存コマンドを保持しつつ、一つのread-only操作を同一Provider経由へ切替。 |
| Studio | `cmd/tei-studio/main.go:27-45`のWails `StudioBridge`はList/Catalog/Validate/Inspect/ObserveXeadを手書き公開。`internal/studiohost/readonly.go`にTyped Service/ReadOnly構成あり。`cmd/tei-studio/go.mod`は別のWails依存モジュール。 | 型付きAdapter分離は足場になるが、Window/Editor/Commandの標準Plugin Contributionと実Consumer同値は未実証。Wails Bridgeを業務契約正本にしない。 |
| Plugins配置 | `plugins/README.md`は参照例の存在と「製品用Pluginへの一括移行未実施」を明記。参照Go/JS/Process実装がある。 | 通常の標準CLI/Studio機能をPluginとして構築・選択・試験・利用する一連の成功例が必要。 |
| Plugin runtime | `internal/plugin/runtime/adapters.go`はDeploymentExecutor/Observation/Evidence等の固定能力を既存Registry.Invokeに接続。 | Provider実行・拒否・来歴の資産として活用。ただし種別追加ごとにAdapter固定追加する構成を標準にしない。 |
| 製品状態の線引き | `docs/status.md`のStudio PoCとComposition Referenceは、実測版・局所的範囲を限定している。 | PoC成功や`validate.ok`は「CLIとStudioがPlugin基盤で動く」の証明ではない。 |

**監査結論**：目標思想と基礎部品はあるが、**現在のCLIとStudioの通常機能は、原則としてCore以外をPluginにした製品形態ではない**。一括ソース移動ではなく、代表操作の実経路を新しいContract/Provider/Compositionへ一本ずつ切り替える。

## 2. 物理構成より先に固定する論理境界

1. **最小Core**：論理Subject/Ref、標準Record/Relation/Graphの最小表現とSource Location/Revision、Contractを参照・比較する汎用契約、Scope/Effectの不変条件、同じ入力から一意に選択するResolverへの最小入口、結果の失敗/未知状態と証拠参照。不変条件がない実務機能を「便宜上Core」に追加しない。現行の`core/`がこれら全てを実装済みという主張ではない。
2. **Core外の標準Plugin**：Role/DSL/Parser/Validator/Operator/Query/Data Provider/Generator/Builder、Formatter/Test/Driver Adapter、CLI command surface/Studio inspector・editor・view、MCP/Web/REST/API・Desktop surface、認証、Logger、業務Rule/Workflow、Target実行支援。どれも同一`Service Contract → Provider → Composition`で接続し、固定`CapabilityKind`列挙を新規追加しない。
3. **実行環境・信頼境界**：CLI process、Wails Studio、Web server、.NET/Javaの既存Host、生成Goバイナリは**Consumer/Target**であり、共通のTEI常駐Hostを全てに要求しない。OS/Hostの権限実装は適切な境界が所有し、Pluginによる自己許可を拒否する。
4. **Bootstrap例外の明示**：`cmd/tei`とStudioの起動点は選択されたPlugin集合をロードまたは静的結合して最初の操作を公開する小さな実行入口として残してよい。ただしCLIコマンド一覧/業務操作/画面メニューをもう一つの意味的正本として埋め込まない。起動失敗・構成診断の最小表示をBootstrapに認める。
5. **Plugin ≠ Class ≠ Package ≠ Implementation ≠ Instance ≠ Build Artifact**：Service Contract/Version、Provider/Implementation ID、Plugin Package/Bundle、Component型/Instance、構成上の`selected`、実Buildの`linked`、実呼出の`invoked`、現在の`authorized`を別のRef/Relationに置く。二つの論理Pluginを単一Goバイナリにまとめても、権限と効果まで共通化しない。

### Everything is DataとTraitの合成

- Plugin定義、Contract、Parameter、Invocation Offer、UI Contribution、Test Pattern、Scenario、Build Definitionを、標準データ構造上のRoleを持つデータとして読めるようにする。Markdown/表/Graph表示は同一意味構造へのProjection。
- **原子的Trait**は用途別の契約要求として合成する（例：`Discover` / `Read` / `Mutate` / `Validate` / `Render` / `Observe`）。これは列挙の提案ではなく粒度の例。各Traitは単一責務・前提条件・Failure/Effect・Scopeを持ち、明示された実装を必須とするか検証済みの標準Providerで充足する。単なるHintであればHintと明示し、未実装の実行能力として発見させない。
- 無関係なTraitは採用を強制しない。例：Read-only logはDDL Traitを持たず、認証が必要なSaaSだけAuthentication契約をCompositionに要求する。Traitの宣言だけではContract Publisher資格、Provider適合、Host Grantを得られない。
- 「Plugin自身でContractを公開できる」と「Contractが公的に認められた」は異なる。Open/unpromised APIへの下流Adapterを許し、変更責任は明示した依存側へ帰属させる。

## 3. 同一Pluginが開発環境とアプリケーションの両方で使われる形

```text
             Canonical / Source / Design Context (#3262)
                            |
       Composition Profile → Contract Resolver (#1016/#951)
                            |
              exact Provider/Artifact Lock
                            |
               typed Application Operation
                   /        |         \
      CLI Projection   Studio Projection    Runtime Projection
       (command)       (menu/panel/editor)   (Web/Go/.NET/Java)
                   \        |         /
                  same Operation/Diagnostic/Result
                           |
              Scenario / Test Evidence (#1180/#2514)
                           |
           real Build/Artifact/Invocation evidence
```

**共通なのは意味操作と契約**であり、UIそのものでも、各TargetのOS認可でもない。各Surfaceは同じ型付きRequest/Result、Source/Subject、Failure taxonomy、Expected Revision、必要Current Grantを渡し、Target固有の制約を追加する。CLI経由の成功でStudioのUI操作や外部Hostの権限・業務副作用を合格にしない。

- **一つの`Operation Offer`**（概念名、正式Schema未採用）はContract/Providerと公開可能な入力/出力Schema、read/mutate/execute等のEffect、Scope、参照要求、欠落・未許可理由を記述する。#951/#1016の選択と現在のHost権限が使えるOfferだけ提示する。
- **CLI Surface Plugin**はOperation Offerをコマンド・引数・stdout JSON/stderr・exit code・Help/Discoveryへ投影する。固定`acceptedCommands`の隣に新Registryを増やすのではなく、切替対象の一操作を従来分岐から共通Dispatchへ移し、互換シェルを縮退する。ユーザーの低負荷を優先し、全操作に冗長な`plugin invoke`を要求しない。
- **Studio Surface Plugin**は同じOfferと標準表現型（table/tree/graph/formなど）をSurface Contribution（メニュー、パネル、Inspector、Editor）へ投影する。最初はWails `StudioBridge`がTyped Operationを委譲する形でよく、Pluginが任意JavaScript/ネイティブ権限を暗黙取得しない。Surfaceそのものと操作対象インスタンス、ユーザー入力、保存対象を明確にする。
- **MCP Surface Plugin**は同じOperationをTools/Resourcesに必要時だけ公開する。CLI呼出しをハーネスが既に利用できるので、TEI専用MCP Serverを必須前提としない。開発者用MCP Controlと業務AppのMCP Targetは異なる。
- **生成Application**は開発用Surfaceを通常含まず、必要なOperator/Role/ProviderとTarget Adapterのみを選択・生成する。単一Goバイナリが既定、Java/.NET等では対象Hostの認証/権限/ライフサイクルへ接続する。開発用外部制御・テスト専用資産はproduction Buildから実際に除外する（#2391）。

## 4. 具体的な移行縦切り：手書きCLIとStudioから始める

| Phase | 置き換える実物／検証作業 | 成功条件 | Owner |
| --- | --- | --- | --- |
| **D0 監査・非侵襲計測** | `cmd/tei`のcommand分岐、Studio `Bridge`→`studiohost`→`operations`、`workspace.Load`、旧`plugin.ResolveEnvironment`と新`composition`の呼出し依存Graphを実Goから計測。 | Go importと実Request/Reader/Providerの対応表、Target別実Build closure、暗黙`init`/embed/旧KindとUnknownを列挙。 | #1251 #951 #1158 |
| **D1 Read-onlyの同一Service** | 既存`model catalog`または`inspect`の一方を、既存のTyped Serviceから**1 Contract + 1 Provider**として提示。CLI既存互換経路とStudioから同じ操作を呼ぶ。 | 同じSubject/Source/Required Closure/欠落/部分/Revisionが両Surfaceで一致。Pluginなし・不一致・Provider二重競合は失敗。 | #3262 #951 #1158 #1824 |
| **D2 Pluginとしての開発者機能** | Inspector/Validation/Graph Viewなどの代表Contributionを`plugins/<name>/`のサブプロジェクトへ。CLI Help/DiscoveryとStudioメニューを選択されたOfferから導出。 | 追加Pluginに`core/`、手書き`acceptedCommands`、`StudioBridge`の意味分岐変更が不要。未選択Pluginが機能一覧・実Buildに混入しない。 | #1251 #1158 #1824 #1124 |
| **D3 Test Pluginのドッグフーディング** | 同じContractを使うScenario Runner Adapter、Fixture/Data Generator、Coverage/Result ProjectionをPlugin化し、CLIから検証してStudioでもレポートを読む。 | #1180 Scenarioと#2514 Independent Expectedが唯一の正本。CLI/Studioと本体実行のTest Runner/受入基準が同じ、Actual Runner/Target固有E2Eは区別。 | #1180 #2514 #3262 |
| **D4 実データの圧力** | Data Source Workbench（SQLite等のread/structure inspect→DDL preview→sandbox apply）を同じOperationに載せる。 | #3270のread-only/DDL/Schema Scope、実Provider Grant、Effect Receiptと独立再読取を区別。CLI経由の結果とStudio経由の表示が整合。 | #3270 #2622 #1174 |
| **D5 Runtime向け再利用** | 開発ツールで使用したValidator/Query/Providerまたは同一Contractを実装する別Providerを**生成業務アプリのGo静的Target**へ組込み。後で外部HostへPressure。 | CLIと実Applicationの意味契約は一致。Link/Load/Execution/Host Grantが独立にPASS。開発Surfaceは除去。 | #1121 #1016 #943 #2390 |
| **D6 Full self-build** | 構成されたPlugin集合とBuild GraphでCLI、Studio、業務Runtimeの実Artifactを記述・再構築。 | #2831 stage0 seed→stage1→stage2/入力閉包/byte比較/false cache hit/独立Expected。1PluginのSelf-buildを全製品Self-buildと呼ばない。 | #2831 #943 #2600 |

D1は**新たな汎用Plugin Manager/Runnerを先に全面実装しない**ための最小実験。D2以降の採否はD1の「新コア変更件数、実Consumerの同値性、変更量」に基づいて決める。

**現行Goの必須Fit/Gap（2026-11-07）**：CLI/Studioの`inspect`はまだ`operations.InspectSubject`の直接呼出であり、`internal/plugin/runtime/service_carrier.go`の移行Carrierは`application-runtime` scope専用で`authoring-tool`を拒否する。D1実装でその拒否を文字列置換で緩めず、既存`composition`のexact選択とHost所有Inspector Adapterに結線する。選択B／実呼出A・別Scope・Root差替え・未取得・Tool途中切断などの独立Expected、具体的な変更seamと根拠は[Authoring Inspector D1 Fit/Gap／AS-01〜12](../evidence/PLUGIN_FIRST_D1_AUTHORING_ROUTE_FIT_GAP_2026-11.md)に記載する。これは実装経路の追加監査であって、正本Contractや第二Runnerを新設しない。
### D1の候補Contract（図解用データ。正式Schemaでも既存CLIコマンドでもない）

```yaml
operation:
  ref: tei.std.design.inspect   # 名前は候補
  request: {subject: SubjectRef, expectedRevision: RevisionRef, requiredScope: ScopeRef}
  result: {data: Record, relations: RelationSet, diagnostics: DiagnosticSet, completeness: Completeness}
  effect: read-only
  requiredTraits: [Discover, Read]
  failures: [not-found, not-loaded, permission-denied, ambiguous, stale, partial, unavailable]
provides:
  - provider: existing-operations-inspect-adapter
    realization: in-process-static
surfaces:
  - cli: inspect
  - studio: subject-inspector
tests:
  scenarioRef: existing-driver-scenario
  expectedRef: independently-maintained-expected
```

実装時には`SubjectRef/RevisionRef`の正式型とRequired Closureを既存の意味正本へ対応させる。ここに仮の正本を新設しない。Operationがread-onlyでもSource metadataの開示権限をHostに問い合わせる。

## 5. 方式選定：Static/Process/Target-nativeは同じ契約を満たす代替実現

| 方式 | 主用途・優先 | 制限と受入 |
| --- | --- | --- |
| **Static Go composite** | CLI・Studioバックエンド・単独Go実行物の既定。選択済みPluginのみBuild/Link。 | Go package/import/`init`/embed経由の未選択実装混入を実Buildで検査。選択済みProvider変更は再ビルド。 |
| **Target-native Library/Generated code** | 外部.NET/Java Host、Web/Node/Wasmなど。 | 契約同一性と形状・失敗・Effectの変換を検証し、Host固有のSecurity/Transactionを二重化しない。 |
| **Independent process** | 信頼境界、独立更新、異言語Providerが必要なTarget。 | handshake/protocol版、resource/permission、cancellation、timeout、出力欠落、部分作用、クラッシュ・復帰をテスト。Isolation≠自動的な権限隔離。 |
| **Dynamic/Generation switch** | 長寿命サーバー等、更新価値のあるTargetだけ。 | in-flight pin/drainとEffect release、Provider切替失敗時の旧世代継続。静的CLIへWatcherや常駐Hostを強制しない。 |

同じPluginの意味構造を使って、ビルド時だけ必要なBuilderと実Runtimeに必要なProviderを**別の依存閉包**に置く。Package/BOMは実装の束、選択済みexact Lockは契約解決結果、Build Artifactは物理成果物、Verified BOMは実組合せの限定的試験証拠。#3267の2×2実Target Matrixを共有し、単独Plugin PASSから複合PASSを合成しない。

## 6. 受入負例：開発ツール経路を実Runtimeの検査器にする

| ID | 反例／観測 | 独立Expected |
| --- | --- | --- |
| **EDP-01** | 新Inspector Pluginを追加 | `core/`と`cmd/tei`のコマンド分岐・Studioの業務分岐変更数0でCLIとStudioにOfferが現れる。未対応Targetに現れない。 |
| **EDP-02** | Plugin選択を外す／同Contract二Providerを宣言 | 消えた機能を旧手書き経路から実行できない。競合をロード順/latestで自動選択しない。 |
| **EDP-03** | 同一SubjectでCLI JSONとStudio Operationを比較 | Source/Meaningの採否/Null/未取得/partial/Revision/Required Closureが一致し、UI表示の省略で完全性が変化しない。 |
| **EDP-04** | Pluginデータを変更せずCLI/Studio UIだけ変更 | Canonical/Accepted Meaning/Runtime Providerは不必要に失効しない。ただしSurface Build Artifact/実入力の変更は検査する。 |
| **EDP-05** | 同Bundleに未選択Providerを追加・更新 | Candidate Catalogは変わり得るが選択済みBuild closureと実Link byteは、toolchain固定条件では不変。単なる`-plugin` flagでは除去証拠にしない。 |
| **EDP-06** | Contract ID/出力型は同じでNULL/Effect/Scopeだけ変更 | 旧ClientのPASSやBOMを使い回さず、該当ConsumerのConformance/Meaningを再検査。 |
| **EDP-07** | 実行前Grant失効・Cross-Tenant・同Bundleの別ProviderにGrantなし | Studio/CLI/生成AppのいずれもHostの実行直前Checkを迂回しない。 |
| **EDP-08** | stdoutの8,192 byte切断/ページ間Source変更/Response欠落 | Command exit0やUI描画を`complete`と誤認しない。snapshotに束縛したcontinuationとend markerが必要。 |
| **EDP-09** | write Provider実行後にCommit返信断 | success/failedに決め打ちせずunknownとし、同一Operation identityで確認・独立再観測。無条件再送不可。 |
| **EDP-10** | 同一ContractのProvider二版×Consumer二版 | 各exact Lock+Target+実Artifact+Independent Expectedに対する同一Attemptの結果のみVerifiedとし、未実施組合せを候補に留める。 |
| **EDP-11** | 開発用外部操作PluginをTest Buildだけ選択 | 本番Artifact/route/module/listener/debug assetsに専用外部制御面が残らない。通常UI/Accessibilityは維持する。 |
| **EDP-12** | Coreを独立Buildし、Pluginなしの最小構成を生成 | Registryや一般Runtimeを失っても基本型/Source/診断が扱える。特定Database/IdP/Wails/LoggerにCoreが逆依存しない。 |

**計測**：新Pluginを実装してCLI/Studioで発見・試験するまでの操作数と差分行数、Core修正件数、手書きAdapter追加件数、Provider競合の検知率、現Build closureとモデル上の宣言との差、Plugin未選択時のBinary size/startup/memory、CLIとStudioの同一契約テスト結果、未観測/過大PASS件数を記録する。生成AIハーネスが`inspect → scaffold → test → build → consume → inspect evidence`を**CLIのみ**で達成できることを先に検証する（コマンド名は実装が出現するまで候補扱い）。

## 7. 移行時の契約と停止条件

- 旧`CapabilityKind`で動く機能を一括削除しない。**Consumer単位**で旧経路・新Providerの対照テスト→実Consumer切替→旧経路削除/限定read-only変換。新旧を並列の正本にしない。
- TEI固有の常駐Hostや一つの万能IPC/ABIを、全Targetの前提にしない。StudioのWails Service Adapterと生成Web/Desktop ApplicationのTarget同値を混同しない。
- 実装を`plugins/`へ移したこと、Plugin Descriptorを作ったこと、Command Helpに表示したこと、Mock Providerが応答したことだけで**実Plugin消費・Conformance・Host Effect**を成功扱いにしない。
- D1を最小Go Targetで実証した後、D2で追加した抽象が開発負荷と安全上の効果を改善しなければ、Interface/Roleを縮小・撤回する。意味・認可・Buildの新しい第二基盤は採用しない。
- 本書に含むCLI/Studio共通Plugin実利用、Generated App再利用、実Build Closureテスト、Studio E2E、Self-build Stage試験は**NOT RUN**。独立Expectedとexact-head CI、実Target負例を実施してから段階的にQualificationを判断する。

## 7A. Desktopで最初に通す縦断：PluginのRead-only OperationをCLI／Studioへ同一契約で接続する（旧§8）

**追加背景（2026-11-07、#1824 Consumer・#3270 Workbench・#3274 HTTP Graph・#3262 Design Contextと照合）**：現行 `cmd/tei/cli_commands.go` の手書きコマンド、`cmd/tei-studio/main.go` の固定 `StudioBridge`、`reference/tei-studio-workbench/main.ts` のSource Subject Catalog・`validate/refresh/inspect` は、同じContract/Providerの実選択と呼出しを共有していない。これを「CLI/StudioはPlugin化済み」や「任意のData Source/Graphを汎用表示・操作済み」と見せない。Studioの既定 `/workbench/` とBootstrap `/`、生成アプリTargetの証拠は別に取る。

**D1の実装対象を絞る**。新しい汎用Plugin Manager、UI Registry、HTTP Graph Renderer、Data Source Backendを先行しない。最初のOperationは現存する **read-only `inspect` または `model catalog` の一方**に固定し、既存のTyped Service/Source Index/Host公開制約をProviderとして包んだうえで、#951/#1016の実Compositionが選んだexact ProviderをCLIとStudioの二Consumerへ引き渡す。既存の結果・診断を利用し、両者で同じMeaning/Read scope/Failureを観測できるか確認する。Wails Bridgeは薄いAdapterとして残せるが、選択Pluginに依存した意味分岐・固定Command追加を重複正本として増殖させない。

### 7A.1 Producer→Consumerの接続で消してはいけない情報

| 区間 | 取得すべき証拠（既存契約へ束縛する候補。新Schema／正式CLIは未採用） | そこから言えないこと |
| --- | --- | --- |
| **要求と資格** | Caller、Workspace/Subjectのstable ref、用途、expected World/Source Index/Accepted Revision、現在のHost/Disclosure scope | Subject名が一致するだけで現在の開示・実行が許可された |
| **Contract/Provider選択** | Contract/Version、実Provider identity、selected exact Lock、未選択/複数候補/Contract不一致の診断、実呼出しTarget | Plugin Packageに同梱／Capability宣言あり＝実Providerが選択・実行された |
| **実Source/Design Context** | Required/Discovered/Loaded/Parsed/Bound/Returnedの用途相対集合、Provider読取範囲・取得時点・粒度・NULL/欠損/非開示/partial | Catalog件数、validate.ok、終了コード0、切断されたJSON＝完全な対象設計情報 |
| **UI Presentation** | 同一Resultの公開可能なField/Relation/Source locationとLoss、表示Plugin・Target版、Selection/Workspace epoch、受領完結 | StudioのTable/Graph表示やARIAの選択＝Source Editorの受理／Dataの更新Target／Host Grant |
| **Result/Recovery** | Typed failure/unknown/unsupported/stale・受領完全性、exact Operation/Attempt、実証時は独立Expected | read-onlyの成功＝業務Action/HTTP Effect/Receipt/資源現在値／Owner Completion |

D1では**read-onlyなので外部Write Effectは発生させない**。ただし外部データを読むProviderではネットワーク・課金・秘密取得などの利用条件があり得るため、read-onlyという名前だけで暗黙接続・全件検索・外部送信を許さない。実際のProviderが未配線なら、Stub/Referenceの応答をProduct実Readとして表示せず `not-connected` を残す。

### 7A.2 他チャットの成果をDesktop固有のConsumer圧力へ接続

1. **汎用Data Source Workbench #3270／RQ-035**：初期はschema/Query Catalog等の読み取り専用候補だけを共通Operationで検証。実SQL Row/Field/Scopeを読み込んでいない場合、Table/Treeを描画できても「異種Data Source閲覧完了」とはしない。DDLのPlan/Applyはread-only Pluginの機能へ暗黙付与しない。DROPや型縮小は別のIntent/Grant/実Provider/独立再観測（#2892のMigration History unknownを含む）。
2. **HTTP Response Decision Graph #3274**：標準有向多重GraphのNode/Edge/Guard/Outcomeと、Client Reaction/Recoveryは用途別Projection。既存 `reference/tei-studio-workbench/presentation.ts::relatedGraph` はCatalogの単親・最大8 Entryだけ、`reference/tei-studio-graph/graph.ts`はself-edge未対応のread-only Rendererなので、実HTTP/業務の複雑な多重GraphのConformanceには使えない。初回は許可されたNode/Edge表／Source参照／未対応診断を共通Read Contractで表示し、適格なGraph Rendererができてから同一Sourceから比較。Graph表示やOpenAPI生成は実Server Guard/Client Retry安全性の証拠でない。
3. **Design Context Query #3262／AIテスト #3273**：CLI/Studioの読取りは、Required集合とSourceの実読取閉包、page/cursor/end marker、途中切断/欠落Reply、現在Disclosureを同じ実根拠から説明する。人間向けに短縮した表示からRequiredが欠けても「complete」としない。CLIの9 read-onlyコマンド/限定T0 PASSはP-1b全閉包QueryやAI新Case生成の証拠ではない。
4. **Verified Composition BOM #3267**：同一Contractを供給するProvider二版／Renderer二版の各exact Lock・実Build/OS/Scopeを別資格として扱う。StudioのPlugin一覧に見えること／参考的Matrixのpairwise coverageだけから、実Windows Wailsや生成Goアプリでの相互運用性を保証しない。選択されないPluginの `init`・import・asset混入も実Buildで確認する。
5. **Raw Authoring/SHO**：`*.tei.md`のParser未対応・Syntax errorでも、Hostに許された原byteのOpen/Raw Saveは意味WorkspaceのLoad成功へ依存させない。Editorの保護対象はSource raw byte/Revision/dirty bufferであり、D1のInspectorや意味受理/Provider接続の成否はRaw Draft保存の資格を与えたり奪ったりしない。

### 7A.3 既存EDP／Scenarioに追加する最初の一因子対照（新Scenario SSOTなし）

| ケース（EDP-01〜12と組み合わせる） | 同じ独立Expectedで区別する観測 |
| --- | --- |
| **DC-01** 選択済みread-only Inspect ProviderをCLIとStudioから同一固定Workspace/Subjectで呼ぶ | Contract/Provider実解決・Scope・Source/Revision・結果Field/診断・missing/partialが同じ。UIが簡略表示しても意味と完全性は変わらない |
| **DC-02** Plugin未選択・Providerが二つ・同ID別Version | 選択理由と停止理由を明示。旧CLI手書き経路やStudio固定Bridgeによる無言fallbackで実行しない |
| **DC-03** Sourceを1版変更／8,192-byte途中切断／Tool Reply欠落 | 旧Buffer/旧Response/旧pageを現在扱いしない。exit0・Catalog有件数・画面表示成功はread closure completeの証拠としない |
| **DC-04** Catalog Graphの多重辺／9要素／未知Role、HTTP decisionの自己ループ | 既存Graphへ押し込まず、表／Source navigate／unsupportedに留める。表示制約をDomain上の辺・Outcome欠落として扱わない |
| **DC-05** read-onlyログ/Schema探索→DDL Apply要求、またはHTTP write timeout | D1 read-only操作でWrite Offerを作らない。EffectUnknownや受信欠落を画面の `failure`へ縮退させて自動再送しない（#1158/#1174へ） |
| **DC-06** Read/Disclosure変更、Plugin/Renderer版変更、未使用Plugin追加 | 表示とOwner Meaning/実Grantの失効範囲を分離。秘密候補数をDOM/ARIA/Tool JSONへ漏らさない。未使用Pluginの実Artifact混入は別に計測 |

**実装順・停止線**：①D0で手書きCLI/Bridge・既存Product Reader/Providerを実Call Graphで確認→②D1でread-only一操作を実Compositionから解決しCLI JSONへ接続→③同じOperationをStudio `/workbench/`のInspectorへ接続（Node/DOMとWorkspace/Selection epochの後着負例）→④実Windows WebView2で表示/Host Disclosureを再検収→⑤#3270実Read/DDL、#3274 graph/HTTP、#3273 AIテストの各Ownerへ派生受入。実Go/Node/Windows/Provider E2E・本書でのOwner受理は **NOT RUN**。既存D0〜D6とEDP-01〜12のGate/SSOTを増やさない。

## 参考実装からの限定的な示唆

- [VS Code Extension API](https://code.visualstudio.com/api/)は標準機能のExtension化、[Extension Host](https://code.visualstudio.com/api/advanced-topics/extension-host)はUI/Workspace/Remoteという実行場所の分離・Activationを参考にできる。ただしTEIは同じExtension Hostを義務付けない。
- [HashiCorp go-plugin](https://github.com/hashicorp/go-plugin)はProcess/RPC・Protocol version・Handshakeの実装例。これを全Pluginの義務ABIにしない。
- [Gradle TestKit](https://docs.gradle.org/current/userguide/test_kit.html)はPluginを実Consumer（実Build）で試す機能試験の参考。TEIは既存Driver ScenarioとIndependent Expected、actual Build/Runnerに接続し、新Runnerを作らない。

---

## 8. 他チャット横断で確定する「同一プラグイン基盤」の正確な意味（2026-11-07深化）

**結論**：CLI／Studio／生成Applicationで共有するのは、Service Contract・Provider適格性・構成選択・Build/Artifact/Invocation/Evidenceの**仕組み**であり、同じ権限、同じProvider、同じDeployment Lock、同じ表示機能、同じSource受理を強制することではない。これは #951 が既に分けているSemantic／Authoring／Build／Runtimeのscope、および #1016 の単一Composition入口候補を維持する判断である。

| 軸 | Authoring（CLI/Studio） | Build/Test | Application Runtime | 分ける理由 |
| --- | --- | --- | --- | --- |
| Subject/Meaning | Source Observation・Candidate・Accepted Meaningを参照 | Accepted SourceとScenario/Expectedを参照 | 導出されたFunctionと対象Dataを利用 | 同じSourceは未受理状態を含み得る |
| Provider選択 | Authoring Source/Parser/Inspector、read-onlyを含む | Generator/Builder/Test Adapter | Query/Rule/Data/API Provider | 同Contract IDがあっても役割とscopeは別 |
| Lock/Artifact | Tooling Profileのexact選択、CLI/Studioの実成果物 | Build-only入力閉包、Runner/Toolchain | Target/Hostに適合するexact選択 | 意味版・Builder版・実Binary・現在Grantは別 |
| Authority | Workspace操作権限、Source秘匿、編集Intent | テスト用隔離・Artifact検査とEvidence | 現Host/Principal/Resource Grant、Effect | デバッグ可 ≠ 本番書込み可 |
| Presentation | Command/Inspector/Editor/Help | Matrix/Report/Trace | Form/Web/API/Native | Surface Contribution自体は意味・作用を所有しない |
| 実証上限 | 操作表示＋実Tool消費 | 実Build/実Runnerと独立Expected | 実Load/Invoke/Effect/独立再観測 | 局所PASSを横断PASSに足さない |

**四つのComposition対象**（Authoring / Build / Test / Runtime）は、**四つのResolverを意味しない**。既存 #951/#1016 の共通解決・固定原理を各Scopeへ適用する。Accepted Semantic SetはEnvironmentのRuntime Provider選択によって書き換えられない。Build-only依存、Test-only依存、Authoring Plugin、Runtime Providerは別の閉包として収集する。必要な間は一つのBundleが複数Scopeに対応してよいが、未選択側を実Artifactへ強制同梱しない。

**Core境界のブートストラップ問題**：全機能をPlugin化しても、そのPluginの記述を読む機構までPluginだけで自己発見することはできない。最小Coreには、型・Refと値/Record/Relationの基礎、限られた既知形式の構造読取、Source/Revisionと曖昧・未知・エラーの表現、契約照合・構成選択の安全な入口を残し得る。ただし特定業務DSL、全面的なMarkdown/SQL Parser、Graph Renderer、認証方式、DB接続、標準業務検証の実装はCoreに固定しない。「構造的に読める」ことと「Roleに従って意味解釈できる」ことを分離（#2620）。CoreがProvider不在でも診断・最低限のSource保全を行えることをD0で確認する。

**Pluginがなくても持続するデータ**：対応するParser/Editor/Validator/Rendererがdisable/uninstallされてもSource byte・Git履歴・明示的なユーザー修正は消さない。対応Role不明、型未解釈、Source未読込、開示不可を別表示。Source保存はMeaning受理ではなく、UI表示ができても実行許可ではない（#2044/#2620/#3262）。

## 9. Trait/RoleとPluginの合成は似ていて異なる：強制力の設計

他チャットのTrait原子性・Implementationの強制、およびEverything is Data #2619/#2620 を結ぶ。**Roleはデータに意味を与える解釈上の主張、Traitは対象が満たす小さな契約上の性質、Providerは具体的な実現**とし、互いを自動同一化しない。

| 宣言の種類 | できること | 検証と禁止事項 |
| --- | --- | --- |
| Required Trait | そのTraitを満たす実装の存在を要求する | 適合Providerが不足なら未実装診断。表示上のHintで代用しない |
| Default-backed Trait | 明示された標準実装を選択可能にする | Default選択も既存Resolverに従う。複数候補から無断先勝ちしない |
| Manual-implementation Trait | 利用者が指定Interfaceの実装・試験を提供するまで不成立 | AI生成のstubやメソッド名一致を完了扱いしない |
| Advisory Trait | Hint/補完/最適化候補として利用する | 「実行できる」「安全である」「対応済み」とDiscoveryで宣言しない |

**原子的粒度**：一つのTraitは観測可能な一責務と、必要なShape・Precondition・Effect/Failure/Scope・独立Conformanceを持つ。例としてSource `Read`、Schema `Inspect`、Data `Mutate`、Result `Render`、Execution `Observe`を**説明用**に分けられるが、固定Core enumではない。Trait合成は可能でも、`Read`を持つから`Mutate`やDDLを推論しない。互いに排他的なConstraintや同一Operationへ異なるEffect義務があれば診断。結合して初めて成立する振る舞いはBridge Role/Contractを明示し、個別Trait PASSを合成PASSにしない。

**三つの適合確認**：Structural（型・多重度・NULL/欠損の形式）、Semantic（意味・単位・精度・時点・拒否・Effect）、Operational（実Provider/Artifact/Target/現在Grant）を別観測。Traitを付けただけでDefault実装の物理収録、Semantic Acceptance、Host Authorityを得ることはない。

## 10. CLI・Studioで「同じProviderを実利用」の判定を厳密化

従来D1の`model catalog`／`inspect`を使う場合、**選択を通さずに共通`operations`のGo関数を直接呼んだだけ**では、UI Adapter共通化の実証に過ぎずPlugin dogfoodingではない。一方、手書きコマンドから毎回別Processを起動する実装も必須ではない。

**D1で実装者が提出する一つのContext表**：
1. Request主体：Subject kind+ID、Workspace/Source Revision、Required Closure、disclosure/Scope（#3262）。
2. Contract：version・fingerprint・Shape・Failure/Unknown/Effect・Trait適合・要求Context。契約の公開資格は#951。
3. Composition：どのTooling Profile/Targetが要求し、既存#951 ResolverでどのProviderを選んだか。未選択・複数候補・Scope不一致は停止。
4. Implementation/Build：Provider実装とartifact exact identity、実選択→コンパイル→実起動の閉包。宣言だけのロードを実消費と呼ばない（#943/#1251）。
5. Invocation：CLIとStudio各々の**実call-site→同じ選択済みProvider経路**、入力・結果の一致と実Invocationの来歴を保持。CLIがJSON化してStudioがそれを読んだだけでは独立Consumer二本とは呼ばない。
6. Result：stable data、diagnostic、not-loaded/partial/unknown、read/returned/effective scope、Tool受領complete。画面に表示しないフィールドも消滅扱いしない。
7. Claim：Source候補の意味と現在実行Authority、UI/Runner/外部API効果を分ける。

**操作の公開を三段階にする**：`declared contribution`（メタデータ発見）→ `selected & present in exact build/load`（利用可能な実体）→ `invoked with current authority`（実施）。VS CodeのContribution PointsはCommandやMenuの宣言と実ハンドラを分離しており、その構成上の示唆を取り込むが、VS CodeのExtension Host/ActivationをTEI全Targetへ義務付けない。発見時に未選択Pluginの任意コードを実行しない。CLIのHelp/Commands、StudioのMenu/Panel、MCP Tool schemaを**選択済みOfferのProjection**とする。コマンド名衝突、Stable Aliasの版不一致、Studio非表示条件、未許可の存在・件数を表示で漏らすケースは拒否する。

**静的配布での拡張性の実態**：Go単一バイナリの新しいProvider実装は、通常、選択されたソースをBuildへ含めて再ビルドする必要がある。CLI起動後にPluginファイルを置くだけで任意のGo実装を「自動ロード」できるとは約束しない。開発時に外部Processで利用するなら別の明示Realization/Protocol/実行権限の適格性が必要。静的実行ではCompositionで固定したPluginだけを生成されたimport/registration入口で結合し、`init`/blank import/埋込assetsによる無関係依存混入を測る。

**移行を二段に分割**：D1aは現行CLI/Studioの動作・Source/Result/Unknownを固定し、既存コード経路と新Plugin経路をshadow比較する（shadowは安全なread-onlyに限定）。D1bで双方の本番使用入口をProvider経由へ切り替え、**切替対象だけは旧直接経路へ黙ってfallbackしない**。旧互換入口を残す場合は期限・契約変換・実行不可条件を明記する。D1a合格はD1b完了ではない。

## 11. 追加の横断Pressure：単一仕組みを異種用途で壊してみる

| 圧力例 | 再利用するOwner | dogfoodingで確認したいもの |
| --- | --- | --- |
| Design Context QueryとAI Skill | #3262 #2605 #2509 #2516 | 全Required Closure、ページ/opaque cursor、8,192byte途中切断、非公開メタ情報、Skill/CLI/MCPでの同一契約と各Consumer受領 |
| Data Source Workbench | #3270 #2622 #1174 | SQL/NoSQL/Logのcapability差、read/DDL/Create/Alter/Dropの実Provider、実環境Grant・破壊効果と独立再読取。DBA機能を標準へ混入させない |
| HTTP Response Decision Graph | #3274 #1129 #2497 | API失敗/不確定結果/Client Reactionの有向多重Graphを標準Record/Relation/Graphで持ち、同じSourceからCLI表・Studio図・実Traceへ投影。HTTP statusだけでCommit結果を推定しない |
| テストと検証済BOM | #1180 #2514 #3267 #3223 | Scenario/Independent Expectedが同じでも、Provider×Consumer×Target exact tupleで実同居Link/Load/Invokeが必要。pairwise未試験3-wayを保証しない |
| Snapshot/Cache | #3271 #2600 #1641 | Cache hit／Build Artifact再利用／Evidence再利用／現在Grant／業務確定値の違い。Provider同版でもInput/Scope drift時には資格を使い回さない |
| Studio Pluginの失効・Source保持 | #2044 #2046 #2047 #2620 | disable/uninstall/対応不明でもRaw Sourceを保全。UIのUnknown保持・再接続・開示拒否がCLIでも一致 |
| SHO・三要素モデル・Graph Projection | #1124 #2497 #2620、SHO Composer | 三視点Data/Function/BusinessとCRUD/Cross-reference、Graph/表/Markdownを**同じMeaning Ref**から投影。SHOの文章化もモデルの第二正本にしない |
| 外部Target／AI Harness | #2390 #1121 #2509 #2831 | 同ContractのGo静的/外部Host Library等で意味とFailureを保持。HarnessのModel推論・Tool Egress・実Provider effectは別Evidence |

**共通Producerと独立Consumerを取り違えない**：CLI/Studio/Runtimeのどれも同じServiceを読めることは重要だが、3つが同じモック結果を参照していたら3つの実E2E合格ではない。Runtimeで同一Providerを共有できる要件と、Targetが異なるため契約準拠した別Providerを選ぶ要件は両立する。役割がDeveloper Tool専用なら生成業務アプリへ含めないこと自体が正しい。

## 12. D0〜D2の完成判定を小さく、しかし検証可能にする

| Gate候補 | 実験手順 | 合格／停止条件 |
| --- | --- | --- |
| **PD-0 実依存** | 現行`cmd/tei`・`cmd/tei-studio`・`internal/studiohost`・`internal/plugin`・`internal/composition`の実Go importと操作traceを記録。Go Toolchainの実入力・module/asset漏れも記録 | 依存Graphが宣言と一致している範囲とUnknownを明示。将来設計の命名だけで「移行済み」としない |
| **PD-1 Contract** | 同一`inspect` RequestについてContract/Provider exact Lock、Trait/Shape/Failure、実Provider選択の2候補と欠落を試す | 同一名前だけのProvider選択、Version/Scope不一致、undefined/Null/partialの欠落を拒否 |
| **PD-2 Two Consumers** | 実CLIの`inspect`とWails Studio Bridge経由で、同Source Revision、同Required Context、同選択Providerを呼ぶ。旧直接経路との同時比較はread-only | 診断・データ・Source/Scope・Unknownの意味が等しく、実2 call-site/実Provider呼出を観測。StudioをCLI subprocess経由の代理表示だけにしない |
| **PD-3 No Core Edit** | 互換Contractを実装する第2のInspector Providerを追加・選択してD1bを再試験。Pluginを外す対照 | CoreのKind固定列挙、CLI dispatch、Studio業務Bridgeの手書き変更0。旧経路fallback、曖昧先勝ちを拒否。ビルド定義・選択Profileの変更は可 |
| **PD-4 Physical Closure** | same toolchain/flags/seedで、未選択Providerの追加前後に`go list -deps`・実Build入力・`go tool nm`（利用可能なTarget）・Artifact byteを対照 | 条件付き実Build closureの非干渉。Manifest/Command表示や`-plugin`flagのみでは不十分 |
| **PD-5 Authority/Lifecycle** | Source Grant失効・Plugin disable・StudioのWorkspace境界・CLIの旧経路・再入／timeoutを再現 | unauthorized/unknownを隠さず、PluginからHost Authorityを昇格しない。Source保存と利用資格を別票にする |
| **PD-6 Cross Target** | #1180/#2514の同Scenario/Independent Expectedを固定し、Tooling Provider（または同Contract）と生成Go Runtimeで実消費 | semantic/input/output一致だけでなくTarget別Artifact/Load/Invoke/Current Grant/必要なら独立Effect Observation。外部Hostは別資格 |
| **PD-7 Build Dogfood** | 実Source/Provider/BuilderのBuild graphからCLIと選択済み標準Pluginを構築し、stage0→stage1→stage2を比較 | #2831のL1/L2/L3を区別。単なるGoで2回ビルドした結果をTEI self-buildと呼ばない |

**優先順位**：PD-0〜PD-3をP0、PD-4をP0並行、PD-5は同一縦切りの拒否試験、PD-6以降はP1へ。PD-0を完璧な全Repo棚卸しに膨張させない。必要なD1操作のClosureと明確に示された不足から開始し、実際の不足だけ補う。

**PD-1/2最小Fixture**：既存Reference Workspaceから`inspect`可能なSourceを一つ選び、同名異Scope Provider A/B、AのみEnabled、A Missing、Source revision drift、Required relation missing、NULL/未取得/secret、途中切断を各独立Expectedへ記録。表示Modeだけでなく実InvocationのTarget/Source/Provider/Artifactを固定。未選択Providerの追加でCLI/StudioのOfferや実Binaryに予期しない差が出る例を調査する。Fixtureの存在だけを正式Accepted Meaningとみなさない。

**合否観測値**：Coreおよび手書き分岐の変更行数、追加PluginのContract/Provider/Artifact参照欠落数、shadow差分、実Tool受領完了率、無関係Plugin Build byte混入数、誤PASS/誤失効、実Bundle size・startup p50/p95・memory・AI手順数、正確な否定試験の通過率を記録。意味Conformance、実Target/Host資格、Performanceは別Gate。性能閾値は#2600/Targetの事前Budgetへ委ねる。

## 13. Owner配分と実行記録の扱い（第二正本を作らない）

- **本計画・#1251**：CLI/Studio dogfoodingの移行順、実依存の証拠、PD-0〜PD-7の接合票。プラグイン全機能実装を#1251に集中させない。
- **#951/#1016**：Contract/Provider/Selectionの権限・曖昧性・exact Lock・Tooling/Runtimeのscope相違。新Resolver不要。
- **#1158/#1824/#1124**：型付きOperation、CLI/Studio Surface Contribution、Wails実画面。Providerに業務意味を再解釈させない。
- **#2620/#988**：Role/DSL/Expressionの意味、Trait検証、最低限のBootstrap読取。Role評価を単純なPlugin実行権限へ昇格しない。
- **#1180/#2514/#943/#1097/#3267/#2831**：Scenario/Oracle、actual Build/Evidence、BOM、全体Self-build。第二Runner/Artifact Storeを設けない。
- **#2044/#3270/#3271/#3274/#2390**：Source保持・Data/DDL・Cache・HTTP Graph・外部Hostは別々のPressure/受入Owner。各自のScopeをPlugin基盤へ要求する側であり独立第二Kernelを開発しない。

**外部比較からの採否**：VS Codeの`contributes.commands/menus`は「表示宣言と実ハンドラを分離する」を採用対象とする。Gradle TestKitは「Plugin単体テストでなく、隔離した実Consumerの実Build/実呼出を検証する」を採用対象とする。WIT/Wasm Component Modelは「Target native interface型を定義し得る」を選択肢として残すが、TEIの共通ABIや全PluginのWasm化を先に確約しない。これらは参照方式の比較でありTEIでの実測PASSではない。

参照：
- https://code.visualstudio.com/api/references/contribution-points
- https://docs.gradle.org/current/userguide/test_kit.html
- https://github.com/WebAssembly/component-model/blob/main/design/mvp/WIT.md

**実施状況**：2026-11-07の他チャット・現行main・既存Issueの静的突合と移行条件の設計追補。PD-0〜PD-7、Trait合成、actual two consumer、実Go Build/Studio/UI/Host、CLI Skill/MCP、実DB/HTTP副作用、Self-buildの新規試験は**NOT RUN**。正式Schema/ABI、数値Budget、Owner Meaning Acceptanceは**NOT ACCEPTED**。本追補は#1119のG0〜G4を置換せず、実証時の必要証拠を既存Gateへ渡す。

## 14. 横断収束：実行可能Offer・表示逆写像・受信側SinkをD1の契約監査へ接続（2026-11-08案）

**位置づけ**：#1251のD1/PD-1〜5に渡す最小の検証差分。#951/#1016のContract/Provider/Resolution、#3278のAtomic Trait、#1158のOperation/Surface、#1180/#2514のScenario/Independent Expected、#3267のBOM、#2516のDisclosure、#2497のProjection/逆写像が各正本。新しいCore enum、第二Resolver、権限・Evidence・Scenario SSOT、共通TEI Hostを作らない。以下のOperation Offerは**説明上の集合名**であり、正式Schema/ABIの採用ではない。

### 14.1 「広告できる」と「実行できる」の間に残る空白

一つの操作の受入で、次の独立観測を揃える。異なる層を一つの `available=true` に潰さない。

| 層 | 既存Ownerと確認対象 | 不成立時にしてはいけない推論 |
| --- | --- | --- |
| 意味・型 | #2619/#2620/#3278：Role、Trait義務、型、NULL、Effect、Failure | 名前/構文が合うだけで意味が満たされたとしない |
| 広告・入力 | #1155/#1158/#3242：CLI/MCP/Studioに投影する引数許容値、既定値、条件付きEffect | Help/Discoveryに載っただけで全入力Variantが実呼出し可としない |
| 到達可能性 | #1251/#951/#1016、Hookの既存Validation：必要なPattern/Handler/ProviderとScopeが実接続される | HookやSurface Contributionの宣言だけで実行済み・実装済みとしない |
| 実構成 | #943/#1089/#1251：選択Provider、実Build input、linked Artifact、Host入口で観測した実Provider | 選択B・実呼出A、同じResult bytesを合格にしない |
| 現在の効果 | #1174/#2516：Operation Variant、実Write/Receipt、対象・Grant、Unknown | `read-only`名義のNamed File Outputや実DB unknownを無作用扱いしない |
| 受信・公開 | #2509/#2515/#2516/#3262：CLI/stdout、MCP Tool、AI Harness Session/Log、Studio/SHOへの配送と保存/外部送出の観測 | ProviderがReadであるだけで結果の非永続・非外部送出・完全受領を保証しない |

**既存実例への接続**：
- Draft [#3306](https://github.com/hat47x/tei-platform/pull/3306) はHook `uses` がApplicationの `command` Patternへ到達しない場合を検証時に拒否する実装候補。PluginのOfferでも同種の「宣言はあるが実Handlerが無い」をD1負例にする。HookのCommand経路をそのままPlugin専用Core固定仕様へ転用しない。
- [#3307](https://github.com/hat47x/tei-platform/pull/3307) は既存CLI Discoveryの引数許容値・既定値と実受理を一致させる限定的変更。これを、全Input/Operation Variant/Provider適合が既に自動導出された証拠とはしない。
- 実CLI [#3182](https://github.com/hat47x/tei-platform/issues/3182) とDraft [#3302](https://github.com/hat47x/tei-platform/pull/3302)：`view`の`-execute` VariantはNamed Local Fileへ書ける。実出力と契約上のEffect広告を別々に検証し、実DB Commitへは昇格しない。
- [#3309](https://github.com/hat47x/tei-platform/pull/3309) が報告するDSH headlessの**合成canary限定**観測は、read-onlyで取得したTool結果が永続Session Logにも現れ得ることを示す。他のSink、TEI製品、第三者環境へ無条件に一般化しない。受信側保存先の実測は別Ownerで継続。

### 14.2 表現型Pluginと標準データ構造の境界

**データのIdentity/Relation/Topologyは、Canvas・Table・Graph・Markdown・SHOの表示上のCardやNodeとは別**。SUI/KJ/マインドマップで同じItemが複数Canvasへ現れても、Itemの同一性・Source/Accepted Meaning・依存を重複して複製しない。未分類のRaw ItemはPlugin未導入・無効化後も保持し、未解釈を「存在しない」に縮退させない（#2044/#2497）。表示Pluginは必要なShape/Axis/Unit/Source/Multiplicity/Orderを契約として要求し、非単射・損失あるProjectionは `renderable` と `editable/writeback` を区別する。逆写像の対象Identity/Revision/権限が一意に定まらない限り自動Writeを拒否する。新しい万能ItemKindやCanvas専用Core型を要求しない。

Data Workbench #3270、REST Decision Graph #3274、三要素モデルとCRUD/SHO、SUI Canvasを**異種Consumer**として使う。どれも同じContract/Providerの意味とEvidenceを参照できるが、画面で操作できることから元データへのWrite資格を推論しない。Source保存は開示や業務意味受理とは別Claim。

### 14.3 最小追加負例（既存#1180/#2514のIndependent Expectedへ委譲）

| ID | 反例 | Expected / 既存受入Owner |
| --- | --- | --- |
| PO-01 | Plugin Surface/HookがOperationを宣言、実Handler/Pattern到達経路なし | `declared`のみ、`executable`不成立と診断。#1251/#1158／Hook類似#3306 |
| PO-02 | CLI広告の許容値/既定値/条件付きEffectと実Provider受理が違う | Offerの対象Variantを拒否・診断。#1155/#1158/#3182 |
| PO-03 | Bを選択したがCLIまたはStudioがAへ実呼出、返却bytes一致 | Host由来の実Entry不一致でFAIL。#1251 D1-W01〜03 |
| PO-04 | Read-only SourceをToolで取得し、受信Harnessが結果を永続Sessionへ記録 | Provider EffectのReadと受信SinkのWriteを別観測。Grant/開示とSinkの実測範囲外はunknown。#2515/#2516 |
| PO-05 | 同一Itemを二Canvasへ表示し片方で非可逆編集を試す | 同一Identity・重複計数なし。逆写像/Revision未確定ならWrite禁止。#2497/#1124 |
| PO-06 | Renderer Plugin無効化、未知Raw Itemが存在 | Raw Source/Refは保持、対応するView/Write/Meaningだけunknown/unsupported。#2044 |
| PO-07 | NULL・単位・時点が不適合なProviderを型名だけで選択 | Structural/semantic Conformanceを別検証し不適合を拒否。#951/#3278 |
| PO-08 | 4バージョンtuple中Synthetic Witnessが1件だけ実Call未観測 | 実組合せQualifiedへ加えずmatrix-incomplete。#3267 |
| PO-09 | 同じLockでも受信先のDisclosure/GrantまたはSource Revisionが変化 | 過去のResult/Cache/BOM Evidenceを現在の開示・実行資格へ流用しない。#3262/#3271/#2516 |

### 14.4 最短の実装順と停止線

1. **D1を肥大化させない**：既存`inspect`一操作を、A/B Provider・CLI実Call-site・Studio実Call-site・Host観測で実証。PO-01/02/03/07を優先し、同一結果bytesの誤配線を検出する。
2. **独立した横断pressure**：D1で得たTyped Offerのread/unknown/closureを一つのData Workbench Readerへ接続する。別レーンでPO-04の受信Sink実験を#2515へ、PO-05/06の非可逆表示・Raw保持を#2497へ返す。これらをD1完了のHard Dependencyにしない。
3. **資格の引渡し**：実選択・実Build/Invoke・独立Expectedだけを#3267に渡す。合成witnessや独立Tool PASSのUnionからVerified BOMを作らない。外部Host、実DB Effect、SHO/SUI双方向編集、G4 Self-buildはそれぞれ実Target証拠で後続評価する。
4. **費用対効果**：新しいContract/Providerを一つ追加するためのCore差分、Consumer別特別処理、Open APIへのAdapter変更量、D1誤PASS/誤失効、Sinkの未観測割合、Build closure混入、入力/出力配送の完全性を記録する。第二正本や新Core enumが必要になれば抽象境界を見直す。

**状態**：横断議論とGitHub上の既存Issue/PRからの**設計・受入案**。PO-01〜09をTEI製品経路で実行したわけではない。Plugin実2 Consumer、実BOM資格、SUI/SHO Writeback、実MCP Disclosure/Sink、正式Owner Meaning Acceptanceは未受理・未実証。旧/進行中PRの局所PASSをこの統合全体のPASSへ拡張しない。

## 15. Function配置・Topology Binding・変更影響をPlugin-first適合に接続（2026-11-08横断案）

**今回の差分**：§14は「Plugin Operationを宣言・表示・実呼出できるか」を扱った。本節はその次の独立条件、すなわち**同じ能力をどのFunction Data Contextの、どの粒度・関係・表示インスタンスへ適用できるか**を限定する。他チャットの標準データ構造／標準表現型分離（#2495/#2497/#2507）、Functionに置く表現インスタンスとData Context（#2506/#1127）、ViewChange→Intent（#2500）、依存範囲の再資格化（#2582/#2600）、Plugin-first（#1251/#951/#1016）を接続する。正式新Schema/ABI/Core列挙を本節で採用しない。

### 15.1 機能の提供とFunction内の適用・配置は異なる

- **Representation Definition / Pluginの能力**：Table/Tree/Graph/Timeline/Audio/Video等の表現型、必要Shape・Axis・Unit・Cardinality・編集Intent・Target制約を宣言する。Pluginがinstall/discoverされたことは実装選択・Function配置・現在Grantを意味しない。
- **Data Context Definition/Instance**：Functionが参照するSemantic Subject/Relation path、Entity/Edge/Path occurrence/Aggregate等の粒度、取得Sliceの完全性/partial/unknown、Source/Query/Rule Revision・selection epochを持つ（#2506）。複数Contextを一つのJOIN出力へ無断同一化しない。
- **Representation Instance in Function**：表現型の実利用は必ずFunction内のインスタンスとして配置し、どのContextとRelation/Topologyに接続するかを明示する。表現型Pluginの登録時に全Functionへ自動配置しない。複数Canvasが一つのItemを別Occurrenceとして表示してもData Identityを複製しない。
- **Structure/Topology Binding**：Data側の同一性・向き・多重辺・順序・時間軸/座標系・単位・NULL・部分集合を、表現型側でどの範囲まで保持/変換/喪失するかを宣言する。形状適合と用途適切性（例えば集計値を個別Entityとして編集できるか）は独立に診断。自動推測で元の関係を増減しない。
- **Operation Intent／Writeback**：表示で可能なSelect/Drag/Editは、元のEntity/Edge/Path occurrenceとRevisionへ一意に戻せるときだけ型付きIntent候補を作る。Provider適合・Host現在Grant・実Commit/独立Readはその後の別資格（#2500/#1174）。非単射な表・Graph・SHO文章の逆書戻しは原則拒否し、Candidate/Reviewへ戻す。
- **Openな拡張との両立**：未知の表現Pluginや独自Traitでも、契約を満たす範囲を表示・適用候補にできる。ただしOpen/Stable/Internalの公開保証、利用可否、Owner Meaning Acceptance、現在実行権限は別。Trait Fulfillmentの正本は#3278、Consumer実証は#3279。

**解釈上の重要点**：`Plugin selected`、`Function-bound`、`renderable`、`editable`、`operation-offered`、`executable-now`、`effect-observed`は、同じBooleanで管理しない。必要な既存契約の結果をread-onlyに合成した用途相対の判断とし、第二Availability Registryを作らない。

### 15.2 変更時には「何を再Buildするか」と「どの表示・操作Claimを再確認するか」を別々に判断

| 変更 | 表示／Functionへの影響 | Build/BOM／実行資格への影響 |
| --- | --- | --- |
| 未選択の表現Plugin/Provider追加 | Catalog候補のみ変化。Functionの既存Bindingは原則維持。ただしDefault選択へ作用する場合はSelection再確定 | 実Build input不変ならArtifact維持候補。実import/init/Embed混入時は再評価（#1251/#2600） |
| 選択Pluginの実装改版、Contract意味は同じ | 対象Representation/AdapterのConformanceを再確認。受理済みData Meaningを自動変更しない | 実Provider/Artifact/Targetのexact資格を再確認。旧BOMのVersion文字列一致だけではPASSにしない |
| Function内の表示位置・スタイルだけ変更 | 表示Artifact/Occurrenceは変更。Entity/Relationと独立の意味・データClaimは維持候補 | Buildへ実消費された範囲だけ変更。UIだけの編集をRuntime/DB全体失効にしない |
| DataのRelation/多重度/軸・単位/粒度の変更 | 該当Binding、集計、操作Intentの意味・逆写像を再確認。別Contextの無影響は依存根拠がある場合のみ維持 | Data/Query/Provider/Generatorの実消費閉包を照合し局所再Build |
| D2部分取得・親Selection epoch更新・遅延旧応答 | D2依存Claimはpartial/stale。独立D1 Claimまで無根拠に失効させず、旧結果を新Contextへ混ぜない | Plugin実装が同じならBinaryを再Buildする理由とは限らない。Current disclosure/Action資格は再評価 |
| Provider/Source A→B→A、旧Lock・Grant再登場 | 見かけの表示一致だけで旧現在資格は復活させない。必要時に再取得・再解決 | #2600/#3267の旧Evidence再利用条件・Artifact来歴とCurrent Grantを別判定 |
| 表現Plugin無効化・不明Source | 未解釈Raw Item/Sourceを保持、対応Viewはunsupported/unknown。存在しない・削除済みへ変換しない | 選択済みProvider欠落時は黙ってDefault/旧直呼出へfallbackしない |

**増分更新は追加だけではない**：#2582のLEFT JOINの未一致行、NOT EXISTS、Top-N、集計のように、入力追加で既存表示が消える場合は`retract/replace`の必要性を明示する。Producerが部分取得しか保証できないとき、0件と「存在しない」の同一化を拒否。Cache hitやBOM適合履歴でCurrent Result/Disclosureを保証しない。

### 15.3 既存Scenario/Independent Expectedへ渡す追加圧力（PFB-01〜11）

| ID | 自作最小ケース | 独立Expected／担当 |
| --- | --- | --- |
| PFB-01 | Table Plugin install済み、FunctionにRepresentation Instance未配置 | CatalogでdiscoverableでもFunction-bound/renderedとしない。#2507/#1251 |
| PFB-02 | 同一ItemをCanvas A/Bに二Occurrenceとして表示 | Item Identityは1件、View Occurrenceは2件。片側の位置変更でDataを複製/意味変更しない。#2497/#1124 |
| PFB-03 | 同labelのNode 2件＋同source/targetの異なるEdge 2件をGraphへ投影 | Node/Edge ID・方向・多重性を保持。表示label一致を同一Entity/Edgeへ畳み込まない。#2495/#2497 |
| PFB-04 | 子D1=2件、D2=3件、JOIN投影6行とD1集計1行 | 6表示行≠子Entity 6件。集計行から個別書戻し不可。#2506/#2507/#2500 |
| PFB-05 | D1全件、D2先頭ページのみ。D1だけの合計とD1×D2のJOIN集計 | D1依存の限定Claimは維持候補。D2依存の全件Claimはpartial/unknown。#2582/#2586 |
| PFB-06 | Window1で親A→B、AのProvider応答が遅延到着。Window2はAを表示中 | W1のB Contextへ旧A結果を混ぜず、W2のAまで誤失効しない。#2506/#2582 |
| PFB-07 | Tree/Graphで多親のEdgeをdrag/edit、表示上はEntityカードを移動 | Edge/Path occurrenceを型付き対象にし、Entity deleteや全親削除へ変換しない。対象曖昧ならIntent拒否。#2500/#2497 |
| PFB-08 | RendererのAxis/Unit/時点/座標系不適合、形状だけは成立 | Render候補と適切性/変換資格を分離し、無音の単位変換・意味受理を禁止。#2495/#2507 |
| PFB-09 | 未選択Plugin追加／Default変更／実Buildへの`init`混入の3例 | Catalog変更・Selection変化・実Build混入を判別し、不要失効と誤Reuseの双方を拒否。#1016/#1251/#2600 |
| PFB-10 | 同Resultを返すProvider A/B、Bを選択・Aを実Invoke | 値一致でも実Callの不一致をFAIL。Function Bindingが正しくても実Implementation資格は別。#1251/#951 |
| PFB-11 | Plugin取り外し・Raw Itemのみ残存、CLI/Studio/SHOは異なるRenderer | Source/Identity保持と表現不能を区別。開示不能の件数/Pathを無断表示しない。#2044/#2516 |

### 15.4 段階・Owner・非目標

- **P0／D1既存優先**：#1251のInspector一操作、A/B Provider、CLI/Studio実call-site、Hostでの実Entry観測と独立Expectedを先に通す。PFB-10はD1-Wと同じScenarioを再利用し、別Runnerを作らない。
- **P0並行／Headless Binding**：#2506/#2507/#2497/#2500の既存FixtureからPFB-01〜05/07/08を対象に、一つのFunction、二Context、Table+Graphの二表現インスタンスを**read-only**でinspectする。合成入力だけならReferenceとして明記し、Product Parser/Studioでの実利用と区別する。
- **P1／変更影響**：PFB-06/09と#2582/#2600の依存Closureを結合し、Build Artifactの失効とFunction表示/Action Readinessの失効を別票で実測。未知依存はunknown。BOMのexact tuple資格は#3267が所有する。
- **P1以降／Writeback/Target**：#2500/#1174で非単射View→Intent候補→Host実Grant→実Effect→独立Readを順に検証。Java/.NET/Go、SUI/SHO/Studioは必要な実Consumerだけ選択し、新TEI Host前提を設けない。
- **測定**：Core変更数、Plugin独自Adapter数、誤Entity/Edge対象数、NULL/partial過大Claim、無関係Context/Artifactの誤失効、依存変更見落とし、CLI/Studioの意味差、独立Expectedの負例検出率、構成手順・生成AI指示数。新特別機構のコストが優位なら本抽象案を縮小/撤回する。

**責務**：#2495構造分類、#2506 Data Context、#2507 Function Pattern適合、#2497投影、#2500逆写像、#2582反応/局所失効、#951/#1016 Resolver、#1251 CLI/Studio Plugin移行、#3278/#3279 Trait、#2600 Build/Cache、#3267 BOM、#1180/#2514試験、#2516開示。既存正本へ参照・引継ぎし、新Function Binding Store、第二Resolver/Renderer Engine/Cache/Runnerや新Gateは増やさない。

**状態**：設計・負例・受入順の追補。新Function Bindingの正式Schema/ABI、PFBの実Go/Studio/GUI/Host/DB、製品Graph/Canvasの二Consumer、BOM資格、Ownerの意味受理は**NOT RUN/NOT ACCEPTED**。過去の合成Reference PASSをこの統合PASSとしない。

### 15.5 三つのTopologyと媒体軸をPlugin Consumerが再解釈しない（PR #3192/#3222からの限定受入）

二つの表現型を単純に並べるのでなく、**(a) Data topology**（同一性、Relation/Edge、集合/順序/量/時間・空間の意味）、**(b) Representation topology**（Panel/Tab/Tree/Graph/Chart/Media trackのSlot/軸/Occurrence）、**(c) Function-owned Binding topology**（どのContext/Relation occurrenceをどの表現Slot・軸へ接続し、何を変換・喪失するか）を分ける。Functionに属するのは表現型の**InstanceとBinding**であって、標準表現型のDefinitionそのものではない。各層は標準Record/Relation/Graphでread-onlyに説明可能な候補とし、独自Canvas Store/Binding Resolverを新設しない。

**軸は名前だけで同一にしない**。元DataのDomain/Event time、SourceのObservation/Acquisition time、MediaのPlayback time、文書のStructural position（文字/段落/ページ）を区別する。文章→TTS朗読、数値→Sonification、ページ→動画のように新しい時間軸を生成する変換は、原データの時間軸を「保持」したと宣言できない。Source↔Playback/Span対応の変換版、部分対応、丸め・量子化、Source Revisionを根拠として記録し、同期Selectionを元Sourceへの編集や許可に自動昇格させない。Decimal sliderの有限stepでは高精度値を無損失更新できるとしない。Tree上で多親DAGの第二親を不可視にした場合も元Relationを消さない。

**差分のOwner**：標準構造・軸/媒体の定義と既存RP-01〜08は [Draft PR #3192](https://github.com/hat47x/tei-platform/pull/3192)、複数View間のSelection/State/更新境界と既存MV-01〜08は [Draft PR #3222](https://github.com/hat47x/tei-platform/pull/3222)、Projection損失は#2497、Function Bindingは#2506/#2507、逆写像は#2500、Target Rendererは#997、Source書戻しは#1124を参照する。本Plugin-first計画は**そのRequirementがProvider選択・実Load/Invoke・Display/Write Offerへ正しく伝播するか**だけを検証し、RP/MV試験を重複定義しない。

**Plugin側の追加拒否条件（PFB-12〜14）**：
- **PFB-12**：再生時刻が一致しても原Event time・文書Source spanが異なる二Representation。選択同期は許容し得るがSource更新対象の一意性/原時刻一致を推定しない（#3192/#3222）。
- **PFB-13**：Decimal値を低分解能Sliderへ投影。読み取り/近似表示は可能でも同じ正確値へWritebackできる保証がなければ編集不可（#3192/#2497）。
- **PFB-14**：Function A/Bで同じ表現型Definitionを使うがContext/Disclosure/Selection epochが異なる。共有Plugin/Renderer実装は可、Functionの操作対象・現在Grant・保存/受信Sinkを共用しない（#2506/#2516）。

**受入停止線**：#3192/#3222は現時点ではDraftであり、実Media/Renderer/複数Consumerの合格を示さない。Plugin-first D1 Inspectorに媒体/Graph GUIの完成を要求せず、PFB-12〜14は後続Consumer pressureに留める。既存Test/Independent Expected #1180/#2514へ接続し、第二テストRunnerを置かない。
## Desktop固有のD1受入：共通Inspectorを実際に呼んだProviderへ束縛する（2026-11-07）
<!-- tei-desktop-inspect-d1-actual-consumer-20261107 -->

**統合位置と既存証拠**：§10〜13のD1a shadow／D1b切替・PD-0〜7を受け、[#1824](https://github.com/hat47x/tei-platform/issues/1824)の**既定`/workbench/` Consumer**だけの実装・試験停止線を定める。旧[Draft #3280](https://github.com/hat47x/tei-platform/pull/3280)を最新mainへそのまま適用せず、重複するContract/Plugin/Trait一般論を削除した置換差分。[Draft #3292](https://github.com/hat47x/tei-platform/pull/3292)はMaster Plan Gの縦断優先順であり本節の実Consumer受入正本ではない。**本節の追加自体は実コード・実UI・実ProviderのPASSではない**。

### D1が今不足している実Callと、先に使える既存実装

- 現main `internal/operations/inspect.go::InspectSubject` は、対象Subjectのread-only Canonical投影・World/Source Index Fingerprintを返す。ただしSourceコメントのとおりFingerprintは**記述的な同一性情報であってCAS用Revision TokenでもMeaning Acceptanceでもない**。用途別Required Domain Closure、実Parser消費、Hostの現在Disclosure、Provider Scopeの完全性を自動保証しない。
- 現CLI `cmd/tei/main.go` は `inspectWorkspaceJSON` へ静的dispatch、Studio `cmd/tei-studio/main.go::StudioBridge.InspectSubject` は `studiohost.ReadOnly.InspectSubject` へ委譲。Studio側は `operations.InspectSubject` へ接続済み。**低位の同じInspector関数の再利用は存在するが、#951/#1016でexact選択したPlugin Providerの二Consumer実Invokeの証拠ではない**。大規模移管を先行せず既存Read Serviceを同じContract Adapterで再利用する。
- `reference/tei-studio-workbench/main.ts` はCatalog SubjectとInspect、read-only `validate/refresh/inspect`に限定。`presentation.ts::relatedGraph` は単親Catalogの最大8 Entry表示で、多重Relationや実Function Result Sliceではない。Document Raw Editor、Function Data Context、Data Source実Row／DDL、HTTP Decision Graphの実Graph表示はD1のInspector成功をもって資格を付与しない。
- [merged #3285](https://github.com/hat47x/tei-platform/pull/3285)の原子的Trait・契約充足をD1でも判定し、宣言だけのRead Traitや`read-only`広告を実挙動と混同しない。[#3182](https://github.com/hat47x/tei-platform/issues/3182)ではCLIの広告と記録系副作用の不一致が既知。D1の`inspect`は外部Writeなしに限定し、Tooling側の記録やReadの外部送出は別のEffect/Disclosureとして観測する。

### 一改訂・一対象で固定する証拠と受入

| ケース（EDP/PDに従属、独立Scenario正本は増やさない） | 一因子差と期待 | 状態 |
| --- | --- | --- |
| **DC-01 exact selected→actual** | Provider A/Bが同Contractを公開、LockがBを選択しAを実呼出する故障を必ずFAIL。Caller/Host・Contract/Scope/Target・選択Pin・**実Invoker ID/Version/Artifact**・Result/Evidenceを同一Attemptで照合。単にCLI/StudioのJSONが一致しても合格にしない | 新D1検証NOT RUN |
| **DC-02 CLI↔Studio二Consumer** | 同一Workspace実Manifest bytes/Source Root・Subject・公開範囲の`InspectSubject`をCLI JSONと実`/workbench/`で観測。Source/Revision/Failure/Null/Unknownは対応し、UI列省略は`complete`を増やさない。UI後着/別Workspace/同Subject異epochは既存`createResponseScope`等で拒否 | 新D1検証NOT RUN |
| **DC-03 不在・曖昧・Trait欠落** | Plugin未選択、同優先順位A/B二Provider、Read Trait必須未実装、same ID別digest/Scopeで暗黙first-winner・旧手書きCLI/Wails Bridgeへの無断fallbackなし。適格Default Implementationのみ採用し、助言Traitを実行能力にしない | 新D1検証NOT RUN |
| **DC-04 Source/Tool閉包** | World/Source FP一致でもRequired Family欠損、Source Root片側、`*.tei.md` Parser未接続、8192-byte途中切断、JSON重複member、end marker欠落では`complete`/AIテスト生成資格にしない | [merged #3284](https://github.com/hat47x/tei-platform/pull/3284)/[#3296](https://github.com/hat47x/tei-platform/pull/3296)は別の限定Gate/CI証拠。**CLI/Studio実Provider D1のPASSではない** |
| **DC-05 異質データ・Graph** | [merged #3287](https://github.com/hat47x/tei-platform/pull/3287)の合成Postgres 3行・JSONL 5件はFixture契約。実Row/Scope/History/DDL Applyは#2586/#3270の別試験。HTTP多重辺/自己ループは現Catalog Graphへ押し込まずTable/Source参照＋unsupportedを返す | Fixture限定試験とProduct Readを分離 |
| **DC-06 実Build/TargetとCurrent Authority** | 未選択Pluginのimport/`init`/assets混入、選択版と実Linked版不一致、対象/Principal/Grant更新、receipt-old/unknown writeを別Claimとする。UIで表示できたProviderやBOM pairwise/独立Reader成功から実Target資格を推測しない | 実Build/Windows/Effect NOT RUN |

### Desktop D1のUI後着応答検収：Workspace同名ABAを独立に固定
<!-- tei-desktop-inspect-aba-regression-20261107 -->
- [Draft #3327](https://github.com/hat47x/tei-platform/pull/3327)は既存`reference/tei-studio-workbench/responses.workspace-switch.test.mjs`の`createResponseScope`に**A1→B→A2（同じWorkspace名・同じSubject ID）**を追加。旧A1の成功・失敗がA2のInspector表示／エラーを上書きしない2対照。mainと等価な`responses.ts`と追加ロジックのローカルNode 22で**2/2 PASS**、PRはDraftで新exact-head CI／Node24`npm test`／実WebViewは未確認。
- **この局所合格の用途**：DC-02の「UI response-scope epochをvalue ABAに依存させない」を限定的に補強する。ただし既存Host Bridge／CLIとStudioの選択Plugin Providerはまだ接続されておらず、**二Consumerの実Provider同一Invoke／Read cohort／Function Context・Grant／実EffectはPASSしていない**。`createResponseScope`は旧Host操作を停止・取り消すものではない。
- **次の不合格対照**：DC-01選択Provider B・実Invoke A、DC-04 Source/Tool一部欠落、DC-02 CLIと`/workbench/`の実二Consumer、MV-05 Function Context同ID ABAは、既存#951/#1016/#2586/#2514のOwnerへ残す。UIだけの2 PASSをそのままD1b移行許可にはしない。

**確認済みテストの上限**：[merged #3284](https://github.com/hat47x/tei-platform/pull/3284)は9件の既存read-only CLI応答を基にした限定Design Context `partial`／資格保留と7つの合成Negativeを、[#3296](https://github.com/hat47x/tei-platform/pull/3296)はManifest追加2・Tool応答byte改変7を足した計16対照をCIで実証した。ただし独立の外部Harness応答切断ではなく受領済JSONコピーへの改変であり、Rule/Function/Business Required Closure、AIによる新Case生成、C1、外部Host／Studio実適合は未達。`qualifiedTestGeneration=false`を維持し、D1のSource閉包の下限診断として**参照**するだけとする。

**着手順**：P0 現`InspectSubject`→CLI／Wailsの実Call Graphと`Read`の外部効果/Disclosure棚卸し → P1 既存Contract Resolverから**exact一Provider**をshadow実呼出し、同一Identity/Scope/Result＋実Invoker記録をReadback → P2 誤Provider選択、無断fallback、不完全Source／Tool応答の負例を先行 → P3 CLIとStudio`/workbench/`を同一Operationへ切替えて実Node/DOM・Windows WebView2で別Consumerを検査 → P4 #3270 Data Workbench/#3274 HTTP Graph/#3267 Verified BOM/`*.tei.md` SHOへ用途別展開。#3292全体計画Gとの二重Hard Gate化はしない。

### D1の実Studio二画面と、新しく実装された宣言投影の受入境界
<!-- tei-desktop-d1-dual-studio-surface-and-unknown-effect-20261107 -->
- [main統合 #3334](https://github.com/hat47x/tei-platform/pull/3334) は [main統合 #3318](https://github.com/hat47x/tei-platform/pull/3318) の標準業務Data（`tax/exchange/numbering/holiday`）のRole付き例を受け、`ApplicationView.standardBusiness`→`tei view`テキスト→`cmd/tei-studio/assets/view-format.mjs`の「標準業務」表へ**宣言したData/属性の投影**を追加した。Go局所試験・Node formatter試験の成功はPR記録にあり、実Wails RPC/HTTP・実Rowの計算/実DBは未実証。このformatterはStudioの**bootstrap `/`画面**であり、既定`/workbench/`の`reference/tei-studio-workbench/main.ts`が同じApplication投影を受領した証拠ではない。
- **二Consumerを二画面と混同しない**：DC-02の対象はまず**CLI `inspect`と実`/workbench/`**という別Call-site。bootstrap `/`の`view`成功・Workbench `inspect`成功・CLI `view`成功はそれぞれのAction/Schema/Route/Source/公開Scopeを記録する。内容が似た表示だからPlugin-selected Provider actual Invoke、同じFunction Context、Read Slice、Host/DB Readerを通過したと認定しない。次に追加するなら明示した`view`の二Consumerを**別Case**とし、`inspect`のD1に混ぜない。
- [Draft #3327](https://github.com/hat47x/tei-platform/pull/3327)のexact HEAD `dd2c782d4127405e86c04ebca71e88b7db5873be`では、[Node24 Workbench CI #42385644277](https://github.com/hat47x/tei-platform/actions/runs/42385644277)で既存+ABAの**20/20 test、typecheck、build PASS**。しかし`dist/main.js`対Wails埋込み`assets/workbench/main.js`の**byte一致はFAIL**（649340byte地点）。他のStudio #8・Wails PoC A・Repository Health成功も埋込み整合の代替ではない。A→B→A応答抑止は限定合格だが、**Sourceと実配布資産が不一致のため該当PR全体は未合格**。Bundle Ownerがmain基線で再現・更新方式/原典一致を確認後、exact head CIを再検査する。静的/合成テストの成果を捨てる必要はない。
- [Draft #3328](https://github.com/hat47x/tei-platform/pull/3328)はExecutor後のRecorder／Lifecycle記録失敗と`MayReplay`の矛盾を`EffectUnknown`として保全する**Command層の修正候補**。D1 `inspect`はread-onlyでありこのPRに書込機能や新UI結果Enumを追加しない。将来、`/workbench/`からeffectful Commandを公開する場合は、`CommandRecordFailed`等の**Executor後unknown**、事前拒否、Receipt観測、resource再読取を別に表示し、旧UIの一律`failure`を安全な再試行承認としない。未知の作用は#1174/#2996/#2564でReceipt/Fenceを照合するまで自動再送禁止。
- [Open #3306](https://github.com/hat47x/tei-platform/pull/3306)の「Hookの`uses`に対応するCommand Patternが無い」Validator拒否は**実行可能性の宣言チェックの改善候補**であり、複数Commandの曖昧さ・位相対応・実Plugin/Provider/Studio invokeは未証明。#3334のBusiness Role表示やHook宣言のScreen/Print一致を「業務操作可能」と表示しない。

**この追加の適格性**：#3334/#3318のmain実装と#3327 CIの限定結果をreadbackし、Open #3328/#3306の計画/実装候補を分離。新しい実DB／Wails GUI／Function Context／Plugin actual Invoke/Effect試験・Owner受理はNOT RUN。D1a→D1bのexit Gateや既存Scenario SSOTを変更せず、**既存受入DC-02/DC-06への証拠注記**のみとする。

**停止線**：Provider実Invokeを一つも観測しなければ`not-connected`。Required Source/Parser/Role/Disclosureが欠落すれば`partial/unknown/unqualified`。Commandの実副作用が不明なら`failure`へ潰して再送しない。Core/Resolver/Scenario/Authority/Host/Registryの第二正本と正式新Schema/CLIコマンドを本節で新設せず、実Provider／正本意味の受理は各Ownerへ委ねる。

## 16. 2026-11-08 現物証拠から組み直すPlugin-firstとBOMの最短受入

この追補は、直近の別レーン成果を**新しい抽象基盤の提案でなく、実際のSource・選択・Build・呼出しの照合条件**へ還元する。現行mainで受理された限定Referenceと、未統合Draftの資格を区別する。#3278のAtomic Trait原則、#951のContract/Resolver、#1016のApplication Composition、#3267のVersion Matrix/BOM、#3262のDesign Context、#1180/#2514のScenario/Independent Expected、#2831のSelf-buildが引き続き各正本・Ownerである。

### 16.1 実際に確認された新しい反例：宣言、選択、能力、実行は別

- **実main #3337**（merge `4f1541658e617a376051995605cfca1963c646a3`）に読み取り専用`tools/reference-plugin-capability-coverage`が追加された。合成`reference/workspace`では2 Environmentに対して12 Capabilityを宣言、**6 CapabilityはEnvironmentの対象Bindingから未選択**と観測。ただしこれは**欠陥の件数ではない**。`database-provider`の2件は技術名(`postgres`)と能力ID(`database.postgres`)を同一視できない算定上の限界がある。さらに、Plugin自体は選択されてもその中の`editor-intelligence` CapabilityのBindingがない例、同じ`migration.atlas` IDでも`deployment-executor`だけ選ばれ`migration-provider`は選ばれない例がある。**Plugin selection ≠ Capability selection**を具体的に実証した範囲に留める。その後の[#3368](https://github.com/hat47x/tei-platform/pull/3368)で技術名解決規則を修正した同じ12件の観測では未選択は**4件**となった。6→4はClassifier Rule Revisionの差であり、Plugin実利用が増えた根拠ではない。複数Providerが`selected`でも実一意解決済みとは言えない。actual Go link/Runtime invoke/第三者BOM適合は確認していない。
- **実main #3336**（merge `04f75301f408eb98b95ca90a103d6d8ad21ffdba`）に`tools/reference-go-build-stage-driver`が追加された。Buildを別プロセスとして実駆動し、**invoker実行ファイルのdigest／invoked実行ファイルのdigest・argv・PID・exit／子の申告artifact digestとディスクで読み直したdigest**を別々に観測できた。このReference単独ではStage1生成物によるStage2の真の再帰的セルフビルド(L3)は未実施で`l3Claimed=false`。**後続[#3440](https://github.com/hat47x/tei-platform/pull/3440)で`cmd/tei`自体のStage1 ArtifactをStage2 Invokerに使用した限定L3の実行証拠がmainへ統合済み**。ただしSeed Trust・一般的なPlugin組合せBuild・Runtime Authority・Source閉包資格は示さない。BOMの`Built`を「Builderが呼ばれた」「Reportにdigestがある」だけで保証しないための実証部品とする。
- **実CLIドッグフーディング #3149**（横断実測はclosed）から、#3345（`view`でWorld/Source Indexの版照合値が欠落・異なる範囲のfingerprint）、#3346（`docs/plan`内MarkdownはmanagedでもLoaderで解釈されない）、#3347（応答JSON shapeの違い）が個別課題として残った。**読み取れるデータ、管理されるファイル、意味を解釈したSource、AI Toolが完全受領した出力を同一視しない**。#3183/3182等の旧観測がclosedであっても、全CLIが同じLock形式をそのまま再入力できる、すべての`read-only`宣言は無副作用、といった上位保証への自動昇格はしない。
- **#3262/#3273系のT0.5/T1**：限定的なContext不足の検出と固定Scenario実行の進展はあるが、Required Rule/RelationをSource/Loader/Meaningから閉じてAIが未知の独立Negativeを発見し、同じ実SourceでMutantを殺した**T1-B**はまだ別資格。#3324はclosed/unmergedのDraftであり、単に最新Issue一覧へ現れたことをmain実装に数えない。
- **BOM [Draft #3297](https://github.com/hat47x/tei-platform/pull/3297)**：現在RQ-035はData Source Workbench、RQ-036は全容Model Map、BOMは新たな**RQ-037**として独立。旧Draft #3297のRQ-036表記はmain追随時に修正が必要。純Go ReferenceはFull/Pairwise/提供済exact Lock相対の計画と同一Attempt入力の整合を検査する。**歴史的なPR head `a6cddd6778613fa9d6b2ecedfcf0aaade5e51cbc`ではHealth/依存脆弱性CI成功**。その後PRのHEADは更新されているため、旧SHAの成功を現在HEADやmainの成功へ移さない。Draft [#3297](https://github.com/hat47x/tei-platform/pull/3297)は現在も未統合であり、実Provider/Runtime/Host/正式BOM発行とは分離する。実Provider/Runtime/Host/正式BOM発行はNOT RUN。

### 16.2 利用と資格の五段階を一つの「Plugin使用済み」に潰さない

| 観測対象 | 必要な独立証拠（既存Reader/Owner） | 検証できないこと |
| --- | --- | --- |
| **Declared** | Source上の`provides`、Contract/trait identity・version、Source occurrence | Pluginが選択・実装された |
| **Selected** | Environment/Function/Targetに相対的なPlugin Resolution/Lock。Kind+Capability IDが一致したBinding（#3337）。不一致・未Binding・未知を別分類 | Providerの具体Memberが選択された、実Load済み |
| **Realized/Built** | 既存#943/#1097/#2831の実Build Closure、実Link/import/init、invoker/invoked、toolchainとartifact byte照合。Go static と外部Hostは別Target | 実際にそのCapabilityが呼ばれた |
| **Invoked/Observed** | 実Consumerによる型付き`Service`のProvider identity・operation・attempt・target・入力/出力・失敗・読取範囲。単なる関数結果の値一致より**どのProviderが呼ばれたか**を優先 | Contract意味適合、テストの独立性 |
| **Conformant/Currently usable** | #1180/#2514の独立Expected/負例/Scope付きCoverage＋#3267 exact BOM、現在Host Grant/Disclosure/Owner Meaning・Effectはさらに別照合 | 別Version、別Target、別Tenant、将来時点までの無条件保証 |

この表の行はProduct新Core状態列挙でなく、同じ既存Subjectの異なるread-only投影である。公開面の設計案では`Open/unpromised`を基本、明示`Stable`を区別するが、**同Bundleにある未公開Internal/Secretを発見可能として外部に渡さない**。Open依存でのAdapter修復責任は下流が持ち、発行者によるStable保証や実現在Grantと混同しない。

### 16.3 最小縦断：Tooling Consumerの実選択を先に証明する

試験対象はまず**read-only `inspect`相当の一つの実Service/Contract**とする。新しいPlugin Kernel/Resolver/Script Runtimeを作らず、標準Provider Aと代替Provider B、CLI Consumer、次にStudio Consumerを同一契約へ接続する。既存CLIを使えることと、既存CLI自身がPluginとして実装済みであることは別である。

| Case | 改変する条件 | 独立Expectedと実観測 | Owner |
| --- | --- | --- | --- |
| **PDC-01** | 同じContractにProvider A/Bを宣言、Aだけ選択 | Bの存在を未選択＝欠陥としない。実CLI invocationはA。Source/Capability/Target/Lock identityを追跡 | #951/#1251/#3279 |
| **PDC-02** | Plugin Bを選ぶがCapability BindingはAのまま／未指定 | Plugin選択成功をCapability使用済みとしない。必要なBindingが未決ならunknown/失敗。Provider Aが呼ばれてもB合格としない | #3337/#951/#1016 |
| **PDC-03** | BへBindingしても実配線はAを呼ぶ（A/Bは同じ値を返す） | Provider identityと実Consumer invocation traceで**必ずFAIL**。Result一致やexit0だけで合格にしない | #1158/#1824/#2514 |
| **PDC-04** | 同Bundleへ未使用Plugin Cを同梱 | Go import/init/actual BuildにCが影響しなければ既存選択は不変。実混入するなら非干渉PASSを拒否 | #1251/#943/#1097 |
| **PDC-05** | invokerとinvokedの実行物を差替え、Child reportだけ同じDigestとする | 実Binary・Child argv/PID/exit・Disk Artifact digestの不一致を診断。L3への自動昇格なし | #2831/#3336 |
| **PDC-06** | Plugin/Ruleの必須Sourceを`docs/plan`に置くだけ／CLI出力を途中切断 | managed/parsed/loaded/returned/end-markerを分け、Required Closure不明ならBOM qualification不許可 | #3262/#3346/#2509 |
| **PDC-07** | A1/A2×Consumer C1/C2の4 exact tupleを実行 | #3297のMatrixは**計画のみ**。各tupleに同じTarget/Artifact/Independent Expected/Attemptと実際に呼び出したProviderを束縛。実呼出が欠ければnot-run | #3267/#1180/#2514 |
| **PDC-08** | A/B個別PASS・同一Attemptに見えるが別Source/Artifact/Expected/Scope | 旧資格の寄せ集めでQualified Unionを発行しない。失効Scopeだけ再検証 | #3267/#2600 |
| **PDC-09** | 本体ではCLI read-only正常だがStudioから異Provider経由／Local-only権限喪失 | Serviceの意味とProvider使用を独立観測。UI表示と実Runtime資格を別判定 | #1824/#1124/#1174 |
| **PDC-10** | Manual Required Trait未実装、代替Open Providerが存在する | #3278のAtomic Requirementが未充足なら勝手に代替実装へfallbackしない。Publisher Stableを強制しない | #3278/#3279 |

負例は既存#1180 Scenario/Independent Expectedに委譲し、ここへ第二Scenario/Oracle/Runnerを作らない。D0/PDC-01〜03（Go CLI実選択→実呼出）を最小P0、PDC-04〜08をParallel/次段P0、PDC-09〜10をStudio/異Targetで拡張。各Caseの対象Source/Revision/Tool/Process/Artifact/Conformance Evidenceは別欄のまま、同一のRequest／Run／Versionに束縛する。

### 16.4 開発の順序と受理可能な限界

1. **read-only P-1**：#3337 Referenceの`Declared/Selected/Unselected`を、#3262のRequired Closure（#3345のfingerprint範囲、#3346のMarkdown Loader非対応を含む）へ投影する。Kind/IDだけで部分的に把握できるものをcompleteとせず、`database-provider`など未照合はunknownとして分ける。
2. **P0-A**：`inspect`に相当する実ServiceをProvider A/Bで選択・実呼出しし、PDC-01〜03を同じBinary/独立Expectedで実測。**最短の合格は一つの実Consumer/実Providerの正しい束縛**であって、Studio全面移植ではない。
3. **P0-B**：#3336 Stage Driver等が持つ実invoker/invoked・Disk artifact照合を、選択したProviderと同じ実Build/Targetの証拠へ接続する。ただし本当のSelf-build L3は別#2831の資格。
4. **P0-C**：#3297のReferenceを最新mainへ適合させ、実Resolver→実Scenario Runner→実Artifact→限定BOM rowへのjoinを検証。Matrix計画、Result相互一致、実Provider Conformance、BOM発行を**4段階の別結果**にする。CIが過去のHEADでPASSでも新HEADの意味検証を省略しない。
5. **P1**：Studio/CLIから同じServiceへ呼出して意味・actual provider identityの一致を検証。Markdown Authoring/三視点/HTTP Graph/Cache Provider/SHOは既存#2621/#1124/#3274/#3271等へ渡し、P0のHard Gateへ積み増さない。

**成功判定の母数**も分ける。能力選択率、要求/Scenario coverage、コードC0/C1、Version pairwise、実Provider Invocation、HTTP Guard/Reaction、独立Expected/EffectのCoverageは**異なる分母**であり、最も高い値や一つの総合`verified`で代理しない。AI生成のテスト候補はOwnerが採用するまで提案に留め、同じLLMが作った期待値の自己採点を独立検証へ数えない。

**今後の停止条件**：未知のRequired Source、Loader未読込、別World fingerprintのCLI出力混在、Provider identityの実証欠落、旧Artifact/別TargetへのEvidence流用、同名Capabilityの異Kind無断統合、Tool結果の部分配送、Cache hit＝Current Qualification、HTTP success＝Business complete、現在Host Grant不明。これらを`empty/no-plugin/pass`へ読み替えない。

**位置づけ**：#3337/#3336の限定Referenceがmainに存在すること、および#3297の指定HEADでCI PASSしたことは**確認済み**。PDC-01〜10の新規実Consumer E2E、#3297のmain再基線化、正式Plugin API/ABI、Studio実呼出、Trust/Owner受理、本番BOM/Effectは**NOT RUN / NOT ACCEPTED**。この節は単一の新Gateを増設せず、#1119 G0〜G4および既存Ownerへ証拠を返すための計画である。

### 16.5 現在のStudio Host実装SeamをD1資格から切り離す

[Draft #3371](https://github.com/hat47x/tei-platform/pull/3371)は`studiohost`の`InspectionService`を`RootServices/ServiceFactory`へ通す型付き入口を実装候補として提示した。FactoryがあるのにInspectionが無い場合は旧`operations.InspectSubject`へ黙ってfallbackせず拒否し、Subject/schema/effect/Canonical JSON/World・SourceIndex Fingerprint、root allowlist/変更、Host-private errorを検査する。これは**Hostが選択したServiceを使うための不足辺S0**であって、#951のexact Provider/Contract SelectionからそのServiceを実際にBindingした証拠（S1）ではない。

**D1実証の実行順**：S0 Host Service Injection/拒否 → S1 `authoring-tool` Scopeのexact Resolver結果からFactoryを供給 → S2 CLI実`inspect`とStudio`/workbench/`の**別call-site**で、同一Contract・選択B/実Call Aの偽陽性をIndependent Expectedで拒否 → S3 実Build/linked Artifact/実Provider Entry/Targetを同一Attemptの資格へ接続。現`factory=nil`のPoC Builtin経路を、選択Pluginに従った成功としてカウントしない。旧Runtime-only CarrierのScopeを緩めるだけの実装は採用しない。#3354の11 CLI Reply/32局所判別は`partial`であってS1〜S3の代替ではない。#3336のinvoker/invoked/disk Artifact観測は実Provider Invocationを単独では示さない。PR #3371の旧HEADではRepository Health/Studio Host CIに失敗があり、既存の`TestInjectedOperationsOnlyReceiveAllowlistedResolvedRoot`がInspector経路変更を検知。型付きServiceの導入試験と旧Consumer Contractを一致させる必要がある。Root整形不一致（`gofmt`）と既存テスト衝突は**別失敗**として扱い、失敗理由を隠してD1前進としない。今回の追補はGo/CLI/Studio/実Provider E2E/本番BOMを実行しておらず、S1〜S3はNOT RUN。

**並行計画との分離**：§15のFunction-owned Representation Instance/Topologyと#3192/#3222の媒体/Selectionは、InspectorをPlugin化するP0のHard Dependencyにはしない。#3278のAtomic Trait Required Closureと#3267のMatrix/BOMはそれぞれ別Ownerの独立証拠を渡す。


### 16.6 共通API・配備Executor・セルフビルドの実物が示す「選択→実行」残余（11/08）

- **共通APIの採用とTransportの分離**：main統合済み[ADR-0048／#3439](https://github.com/hat47x/tei-platform/pull/3439)は、CLI/MCP/API/Web/Desktopで操作ID/入力/結果/失敗/Effect/前提を共通化し、argv/HTTP/JSON-RPC/WebViewをAdapterへ任せる。すべてHTTPサーバーへ集約しない。**Plugin-first D1の型付きInspectorはこのAPIの実Consumer**であり、別のPlugin専用操作正本を作らない。
- **Transport結果一致はProvider実呼出しではない**：main統合済み[#3458](https://github.com/hat47x/tei-platform/pull/3458)では既存CLIと参照MCPの`validate`結果文書を共有し、意味エラーも`isError`として一致させた。これは同じOperation Resultを利用する局所実績であり、MCP/CLIが同じ実選択Providerを経由した根拠ではない。`inspect`の後続CLI/Studio経路でもTransport parityとActual Plugin Invocationを分離する。
- **選択結果が実行へ渡らない具体的な別用途**：[#3468](https://github.com/hat47x/tei-platform/pull/3468)／[#3474](https://github.com/hat47x/tei-platform/pull/3474)のmain統合済み静的監査では、Environmentの`deployment-executor` Bindingを解決する`routing.Router`が非テスト実行経路で組み立てられておらず、CLIの`RouteCommands`結果はJSONの計画出力であった。実行する参照は固定Capabilityを別途選択している。これは設計上の必然的欠陥とは断定しないが、**Selected/BOM/Planから実Executeへの不連続**が他ドメインでも発生し得る反例となる。DeploymentのEffect/Authority/Host判断は既存Ownerへ委譲し、read-only Inspector P0へ強制接続しない。
- **Functionの能力選択検査は現在一律ではない**：main統合[#3408](https://github.com/hat47x/tei-platform/pull/3408)の実`tei validate`では、Capability無しの`editor.sql_analysis`が拒否される一方、Capability無しの`editor.database_workbench`は受理された。既存[#3413](https://github.com/hat47x/tei-platform/pull/3413)の利用者向け文書も同範囲を明示し、検査拡大の採否は保留。未検出＝実利用可能、または欠陥確定、とは言わない。
- **Build段階の資格は限定L3へ更新**：[#3440](https://github.com/hat47x/tei-platform/pull/3440)で`stage1ArtifactIsStage2Invoker=true`、`artifactsEqual=true`、`sameChild=true`を一環境・一Targetで実照合した。これは以前の「`cmd/tei` L3未実行」から進歩したが、`l3Claimed`自体はfalse、Source closureはrejected、Seed Trust/署名/実Plugin動的選択/Target別BOM資格は別である。#2831既存Receiptと#3267実資格の同一視を禁止する。
- **CLI Source/版の進展**：main[#3445](https://github.com/hat47x/tei-platform/pull/3445)のWindows単一Binary／read-only再実測では、`inspect`／`validate -json`／`model catalog`／`view -json`のWorld/SourceIndex Fingerprint一致を当該Workspaceで観測した。ただし`docs/plan` Markdown Loaderの意味対応・H2/H3 Tool完全配送・実Provider実行は残る。Source/Resultの整合は重要だが、Capability実行資格は独立。

**追加の限定負例（既存#1180/#2514のScenario/Independent Expectedに渡す／新Runnerなし）**：

| ID | 最小の圧力 | 限定Expectedと実証Owner |
| --- | --- | --- |
| **PDC-11** | CLI/MCPが同じ`validate`文書を返すが、Provider Bindingは未選択 | Transport parityは肯定、実Plugin Invocationはunverified（#1158/#3458/#1251） |
| **PDC-12** | Environment `deployment-executor`が選択済み、CLIのRouting Planは出力されるが固定Capabilityの別実行入口を使う | Plan生成とExecution/Targetは別Claim、観測されない接続をPASSにしない（#3474/#951） |
| **PDC-13** | `editor.database_workbench`だけ選択し必要Providerを除外、`validate`が0で終了 | 現Validationの未カバー範囲を`unknown/not-proven`として記録。実機能可否は実Operationが示すまで未判定（#3408/#2507） |
| **PDC-14** | `cmd/tei`同BinaryのStage1/2が一致するが、そのBinaryにProvider A/Bの必要Closureがない | 限定Self-buildのL3は肯定し得るが、実Plugin Build/Invoke/Verified BOM tupleをqualifiedにしない（#3440/#3267） |

**次の実施順**：S0（CLI/Studioの既存型付き注入口と失敗回帰を修復）→ S1（ADR-0048の共通Operation/Contractを#951/#1016のexact ProviderへBinding）→ S2（異なるCLI/Studio実ConsumerでB selected/A actualを拒否）→ S3（実Build Closure/Current Grant/Independent Expectedを同Attemptに照合し必要ならMatrix/BOMへ）。PDC-11〜14は異質なPressureであり、S0〜S2のHard Dependencyにはしない。共通Operationと既存Host/Authorityの正本を使い、新Resolver/Host/CI Runner/Plugin Managerを設けない。

**証拠範囲**：#3439は採用方針、#3458は`validate`限定実コード、#3474は静的依存経路、#3408は限定`validate`実行、#3440は限定セルフビルド、#3445はread-only CLI実測。これらを全体のPlugin-first End-to-end合格と合算しない。D1実Provider×CLI/Studio、BOM実4組合せ、実DB/外部Host、Owner正式意味受理は未実証。


### 16.7 Source-only driftでSemantic再利用・Source来歴・実Provider適格性を別票にする（#3477／Draft #3479）

**新たに成立した観測**：main統合[#3477](https://github.com/hat47x/tei-platform/pull/3477)は、同一Workspace内のRaw末尾LFだけを変更し、Canonical Worldは不変でもSource Index FPは変化するケースについて、**実TEI CLI**のContext→Scenario消費前後の再照合で旧Contextを拒否した（限定Fixture/単一Process環境）。一方[Draft #3479](https://github.com/hat47x/tei-platform/pull/3479)は、**import先ProviderのRaw Source**を同じくLFだけ変更し、ConsumerのWorld、Import Semantic Digest、Consumer-owned Source Index Artifact/Reference/Model Revisionは維持されるのに、`LocateImportedSource`が再読取したProviderのSource Fingerprintだけ変化することを、既存`externalPair`のGo負例で検査する。#3479の旧HEAD `f6934fd4`ではexact-head CIが成功したが、その後HEADが更新されており、**現HEAD `1451bf69`のCIは別途確認が必要**。2026-11-08時点でDraft/未統合。二例は同じ「Source-only」でも**再取得する所属・観測可能なSource集合が異なる**。#3477の成功を外部Importの完全取得へ拡大しない。

**Providerという呼称の衝突を明確に区別する**：#3479のProviderは**外部Import元のWorkspace/Sourceの提供者**であり、#951のPlugin Provider（Capability/Contractの**実装者**）と同一とは限らない。前者のRaw byte改訂から後者の実装改版・Linked Artifact変更・BOM tuple失効は**直接には導けない**。ある特定のPlugin/Builder/Generatorが当該外部Sourceを実際にInputとして消費する場合にのみ、その依存閉包へ影響を伝播させる。逆に実消費を検証せず「非依存」と断定もしない。Source位置の鮮度、Semantic Meaning、Pluginの実装同一性、実Artifact、現在Grantは元々別Claimである。

**区分を共有するが新Kernel Fingerprintを作らない**：現在の資格は既存#3262 Design Context/Source Index、#2743／外部Reference、#951/#1016 Plugin Resolver/Lock、#943/#2600 Build/Evidence、#3267 BOM、#1158 Operation、#1174 Current Grantの**用途別read-only照合**から導く。単一`worldFingerprint`、`sourceIndexFingerprint`、`semanticDigest`、`pluginVersion`のどれかだけを「全入力現在一致」の証明にしない。対象ProviderのRaw Bytes/Source Occurrenceと、Consumer側のCanonical meaning/managed Source Indexは独立。外部Plugin/Providerの実装Source/Contract/Generator inputにSource provenanceが必要な場合は、**実Build・Selection・Invocationと同じAttemptのProvider側の観測**を別に要求する。

| 変化の所在／現象 | 限定的に維持できるClaim | 再照合を要するClaim・拒否する誤推論 |
| --- | --- | --- |
| Consumer自己管理SourceのLFだけ変更、World不変、Source Index変更（main #3477） | 観測済みの同一Semantic Worldに基づく純粋意味計算は再利用**候補** | Source位置・Design Context/Scenario PacketのCurrent性は失効し得る。World不変→Source Indexも現在一致、は誤り |
| 外部Import ProviderのLFだけ変更、Consumer World/IndexとSemantic Digest不変（Draft #3479） | Consumerの限定Semantic Model/生成結果は実消費依存を照合して維持**候補** | Imported Source occurrenceの原Byte/位置、Provider Raw、P0b完全Load/独立Source provenanceはConsumerの2指紋からは判定不能 |
| Sourceは同じだがCapability分類規則/Contract/Selected Provider変更（#3379／#3267） | 確定済みの無関係Source読取観測は維持候補 | 過去の`selected`/Trait履行/BOM資格を新Rule/Lockへ無断流用しない |
| Provider実装/Generator/Build Closure変更、同一Resultを返す | 旧Artifactと旧AttemptのHistorical Evidenceは保存 | 実Build member/Host entry/Targetが異なればexact tuple再資格化。結果のByte一致だけでPASSにしない |
| Raw Providerからの`LocateImportedSource`をConsumerの古いResultで再呼出 | Consumerの旧Meaning revisionという歴史的主張 | **新しく読んだProviderの現在位置情報**を旧Import snapshotで取得済みと詐称しない。複数Source・A→B→Aの原子的時点は未証明 |
| Source/Artifactが同じでもDisclosure/Host Grant/受信Sinkが変化 | 過去の受理済み設計判断とByte Evidence | **現在の実行許可や開示・保存先**は別資格。過去のCurrent Permissionを復活させない |

**Plugin-first最小接続**：D1 S1（exact選択）→S2（CLI/Studio実Provider Entry）→S3（Build/Artifact同一Attempt）の通常正例の**前後**で、Consumer管理Source／外部Provider RawをそれぞれSource-only変更する。変更後に旧Resultを無条件で`fresh`として採用しない。Semantic Digestが変わらない場合、既存Semantic資格を必ず全失効させるわけでもない。まず`unknown/partial/stale`を対象Claim相対で記録し、現Source occurrenceと必要Closureを再取得・照合できた場合だけその用途のCurrent性を回復する。入力と出力の整合性は**単一Attemptだけ**の主張であり、異時点の二回Loadをatomic snapshotとは呼ばない。Provider Source取得経路が`ErrImportsNotBound`を返す場合、空集合／完全取得／BOM資格へ読み替えない。

**独立Expectedへ委譲する追加反例（既存#1180/#2514に載せる、実装していない）**：

| ID | 独立入力と既存観測 | 期待する停止線 |
| --- | --- | --- |
| **PDC-15** | Consumer Raw LFだけ変更、World同じ、Source Index違い（#3477の実CLI） | Semantic再利用候補とSource/Scenario Context拒否を分離。旧PacketはCurrent SourceとしてPASSにしない |
| **PDC-16** | 外部Provider Raw LFだけ変更、Consumer World/Index同一（#3479の限定Go反例） | Consumerの両FP一致だけでImported Provider Raw Freshと表示しない。Provider occurrenceの独立照合が必要 |
| **PDC-17** | `before`のConsumer Resultに対し改変後の`LocateImportedSource`を呼ぶ | Fresh Provider readを旧Immutable Snapshotの一部としない。Sourceの実取得Attemptを記録 |
| **PDC-18** | P0b Provider Sourceが`ErrImportsNotBound`、Provider/Contract/Result bytesは一致 | Required Source Closure未確認でPlugin/Inspector/BOMの**完全資格**を拒否。既に観測した部分Claimは保持 |
| **PDC-19** | Consumer ModelとSourceIndexが同じ・Provider実装Sourceだけ改版、CLI A/Studio Bが同じResult | 同値PayloadでもBuilder/Provider実Entry/Source/Target/Attemptが揃わなければ2 Consumer qualifiedとしない |
| **PDC-20** | A→B→AでFingerprint復帰、またはRead中にProvider Raw置換 | Byte一致した二観測から中間履歴不存在・multi-file同時性・Current Grantを推定しない |

**最短順序とOwner**：#3477の**既に実行済みの同一Workspace差分**をBaseline、#3479の**Draftの外部Provider差分**を別LaneのReferenceとし、後続#1251 D1 Inspector実ProviderでPDC-16/18/19を一Targetだけで実際に圧力試験する。P0bのSource/Import Full Input Closureは#2743／#3262／#2812、Provider Raw/Load/Provenanceは外部Reference Owner、実Builder・再利用は#2600、BOM 2×2のexact行資格は#3267、Function依存の独立Required集合は[#3482](https://github.com/hat47x/tei-platform/pull/3482)の別Consumerが担当。第2Resolver/Source Store/Fingerprint Service/BOM Engine/Runnerは置かない。

**採用境界**：#3477は限定的なmain実Source-only CLI試験。#3479はDraftのGo反証テスト候補であり、旧HEADの局所CI成功は現HEAD／mainへの統合と同一ではない。Provider Source Closure完成を意味しない。PDC-15〜20の一組としてのD1実二Consumer、Provider/Build/同一Attempt、現Host Grant、正式BOM/Owner Meaning Acceptanceは**NOT RUN／NOT QUALIFIED**。

## 17. Traitの原子性と実装義務：Extension TypeからPlugin/BOM/CLI/Studioへ（2026-11-07追補）

**位置づけ**：他レーンで具体化した「拡張型を明示すると、デフォルト実装が選ばれるか、手動実装が必須と診断される」「Traitは合成できるよう原子的にする」という要求を、上記 D1〜D6 の実証条件へ接続する。正式Trait DSL、Core enum、特定実装方式の採用ではない。既存[#2619](https://github.com/hat47x/tei-platform/issues/2619)/[#2620](https://github.com/hat47x/tei-platform/issues/2620) のEverything is Data・Semantic Role、[#951](https://github.com/hat47x/tei-platform/issues/951) のContract/Resolver、[#2507](https://github.com/hat47x/tei-platform/issues/2507) の型付きFunction Pattern適合、[#3267](https://github.com/hat47x/tei-platform/issues/3267) のVerified BOMを第二の正本なしに再利用する。

### 17.1 ContractとしてのTraitとHintを区別する

1. **一Trait＝一つの独立して判定できる義務**。実行要求・前提・入力/出力・失敗/未知・Effect・適用Scope/Target・Conformance Obligation・Contract identity/versionを追跡できる粒度で定義する。「DB全部」「完全なCache」「編集可能UI全体」という巨大なTraitを作らず、単一の意味要求から複数Traitを合成する。Trait名をCore固定enumとして追加しない。
2. **明示的なExtension Type/Roleの指定は、原則として行動を変える**。宣言したTrait義務が (a)適格な標準Provider/Generatorによる既定の実装に束縛される、または (b)対象Consumer/Targetでの手動実装をRequiredとして診断・検証される。実装が存在しないのに型ラベルだけ有効化することを標準の成功状態にしない。
3. **Hint/Advisoryは別の弱い関係**。提案・最適化ヒントだけの場合は実装義務・実行可能性を付与しないことを明記し、discover/inspect/Design Queryで実効Traitと区別する。利用者が「Traitを明示すれば機能が実装される」と期待したのに、単なる色付けで終わる状態を避ける。
4. **Default implementationはContract定義と別**。Defaultがあっても、選択Profile/Target/Scopeに適格で、実装Version/Artifact/入力Closureと独立Conformanceを確認できる場合だけ解決可能。未実装、Provider不在、競合、循環、Target非対応を任意の「空成功」「無断fallback」「最新Provider後勝ち」にしない。
5. **手動実装必須時はどこで止まるかを明示**。候補編集・探索時は未解決を診断として保持できるが、対象TargetのBuild/Published Offer/実Invocationに要求Traitが必要なら、実装不足を不適格として停止する。未必要のOptional Traitまで全世界に一律実装を求めない。
6. **合成衝突は解決順序で隠さない**。二Traitが同じOperation名・異なるEffect/Null/Precision/Unit/Scopeを要求した場合、単に最後に現れた宣言を選ばず、Conflictまたは明示的なBridge/Adapter/Bindingを要求する。多重参照した同一Traitは同じ意味IDに正規化できるが、異なる契約を同一視しない。
7. **Traitの静的成立と実資格は別**。Trait要求 → 具体Provider解決 → 実Builder/Link/Load → 実Dispatcher/Invocation → 現Host/Provider Grant → Effect/Receipt → 独立Observation/Consumer受領を分離。Trait実装と適合試験はAuthority、現在Disclosure、Business Acceptance、業務確定値ではない。

### 17.2 Everything is Dataの候補表現（正式ABIは作らない）

新しい専用ASTや独立Registryを設けず、既存の標準Data/Ref/Relation上に以下の読み取り関係を投影できるかを試す。

| Subject | 最小のRelationと証拠 | 誤解してはいけないこと |
| --- | --- | --- |
| Trait Contract | stable ref/version、単一義務、requires/obliges、Scope、Effect/Failure、Conformance ref | 宣言が実Providerや実権限を生むわけではない |
| Data/Function/Representation Extension Type | どのTraitをどの文脈で要求し、どのBindingを使うか | Target/Function contextなしの万能「編集可」「読取可」は不可 |
| Default/Manual Implementation | provides/implements、Contract fingerprint、Target/Profile、実選択根拠、actual artifact | ContractにDefaultを同梱しただけで実Byteが選ばれたことにしない |
| Composition | requires closure、excludes、Conflict、resolved Lock、選択理由とunknown | 依存未探索／ロード順後勝ちを成功扱いしない |
| Test/Proof projection | Requirement/Scenario ID、Independent Expected、Mutant、実際の実行Actor/Attempt、Coverage/Claim ceiling | 個別Traitのテスト合格から合成BOMの合格を作らない |

この構造は**編集用言語構文の決定ではない**。日常DSLにGoのstruct、func、package、型引数の露出を強制せず、ドメイン専門家が読む順序（対象→付加する意味/義務→既定実装または実装要求→テスト）を最優先する。

### 17.3 具体的なドッグフーディング：実際の効果を宣言へ照合する

最初は、デフォルト実装を持つread-onlyな**設計Inspect/Discover**と、標準実装が存在しない**変更Intent/Schema Alter等**の対比を使う。前者も「外部開示許可」を伴わず、後者は承認されたProvider/Hostの実作用がない限り実行不能にする。Cacheでは Atomic CAS、Snapshot Consistency、Read、Purgeを用途上必要な原子Trait候補に分け、単なる保存成功と整合性・確定値の意味を混ぜない（[#3271](https://github.com/hat47x/tei-platform/issues/3271)）。

**現行実装から得た反例**：[CLI #3182](https://github.com/hat47x/tei-platform/issues/3182)では、command discoveryがviewをread-onlyと広告する一方、明示execute+operation-log/lifecycle-stateが指定されるとローカルファイルへ実際に書き込む。同一Command名の静的なEffectラベルだけでは、特定引数・モードにおける「業務データ操作のEffect」と「ToolのFilesystem出力」を正確に表せない。今回Traitを導入するなら、**Operation Variant/Invocation Context相対のEffectの正確性を必須テストとする**。勝手にread-onlyというラベルを消す/書込権限を変える仕様変更は本追補では行わず、#3182のOwnerが決める。

### 17.4 最小の独立反証ケース（TRF-01〜12、今回NOT RUN）

| ID | 対照入力 | 要求される独立Expected |
| --- | --- | --- |
| TRF-01 Default binding | Extension Typeが1原子Traitを要求、対応Defaultがexact Targetで選択できる | 明示Version/実装Ref/Target、選択理由が出る。契約宣言だけのProvider存在・実InvocationをPASSにしない |
| TRF-02 Manual required | 同Traitを要求、Defaultなし／手動実装欠落 | 設計候補段階ではunresolved、必要TargetのBuild/Invoke前に実装不足を機械診断。無断no-opなし |
| TRF-03 Advisory only | Hintのみ付けたSubject | Hintが機能Offer/必須Plugin/Runtime Grantを増やさない |
| TRF-04 Atomic composition | ReadとValidateを別々に提供し、両方を要求 | 両方の義務が独立に充足、片方欠落時は部分不適格。無関係Traitの実装は強制しない |
| TRF-05 Collision | 同名Operationに異なるEffect/NULL/Unit要求、並びだけ反転 | 明示Bridge/Conflictを要求。列挙順・Import順・last-winsで結果が変化しない |
| TRF-06 Real effect | #3182のview previewとexecute+named output、CLIとStudioの同契約 | 宣言されたread-onlyと実Filesystem writeの違いを診断。静的Command EffectだけでNo Writeを保証しない |
| TRF-07 Provider change | 同Version別Artifact、古いBOMと新Default実装 | 現Lock/実Artifact/Input Closureと独立適合試験が一致するまでqualifiedにしない |
| TRF-08 Target mismatch | Go静的でDefaultあり、外部Host/Browserでは実装・Grant不足 | 代替候補またはunsupported。黙ったGo実装の移植・Host許可の偽装なし |
| TRF-09 Read/Write split | Readable Log Data SourceにSchema Dropを要求 | Unsupportedまたは明示外部Providerの実装要求。readだけでDDL Offerを生成しない |
| TRF-10 Reversible edit | 表への編集写像は一意、集約Graph/SHO文は多対一 | 一意でないTargetへの自動Writeを拒否。元Source navigate/差分Previewを維持（#2497） |
| TRF-11 Cache/Snapshot | Local CASのDefault・Remote Providerによる手動実装、TTL/Grant変更 | Byte hit、Snapshot整合性、Effect、現在開示の別資格。実CAS保証未観測なら失敗/unknown（#3271） |
| TRF-12 Verified composition | Trait実装Provider A1/A2 × Consumer B1/B2 | 2×2 exact tupleごとに実Link/Invoke+独立Expected。未実行行をVerified BOMへ含めない（#3267） |

### 17.5 既存D0〜D6へ載せる実装順と採否

- **D0/現状棚卸し**：実Core/Plugin/Composition/CLI/Studioで既に存在するTrait相当の要求/実装/Constraintを対応づける。新Core enumを入れる前に**標準DataとしてのRelation+Constraintだけで表現可能か**比較。未確認Source/Typed Request/Builder inputはunknownのまま。
- **D1の小さな実験**：TRF-01/02/04/06を先行。既存Inspect read-only Operationを一つだけ共有Contract/Providerに束縛し、無Defaultの場合は手動実装不在を検出する。CLI実argv/実stdout・Studio実Service呼出を独立Expectedと照合。現実に操作未実装なら未実装を明示し、DocumentだけでPASSにしない。
- **D2/D3**：表示可能なTrait義務・Missing implementation・Conflict・Target非対応を#3262 Design Query→#1180 Scenario/独立Expectedの候補へ、Source span/Revision付きで投影。AIは「必要な実装を作るか・組合せるか」を最少の操作で選べること。CandidateのExpectedを独立の正解としない。
- **D4/D5**：Data Source WorkbenchのReadとDDL、Cache Snapshot Provider、Rendererの逆編集を異質なConsumerとして検証。選択済みProviderが実TargetでBuild/Invokeされた証拠と、Hostが与えた現在Grantを別票にする。
- **D6/BOM/セルフビルド**：原子Trait単独の緑とTrait合成/Plugin Version Matrixの緑、実TEI自身のStage2再入を分けて評価。#3267はComposition資格、#2831は真のSelf-build、#2600はCache再利用の資格を持ち、相互に自動認定しない。

**採否判断の計測軸**：新Trait 1件追加に要するCore変更件数（理想0）、Extension記述の構文ノイズ、必須未実装の検出率、合成衝突の偽PASS/過剰拒否、Tool/Runtimeの実Effect広告の正確性、CLI/Studio/Targetの同一契約適合、Provider別手動コード量、Target/Lock差分による再試験範囲。少なくとも2種類の異質なConsumerと正負1件ずつを先行する。

**比較参考（TEI固有方式の採用ではない）**：Rust Traitは実装必須memberとDefault memberを区別する（https://doc.rust-lang.org/reference/items/traits.html）。Go interfaceはMethod Set/埋込みを合成し、同名別Signatureの合成は不正である（https://go.dev/ref/spec）。Java interfaceのDefault実装は多重継承時に衝突し得る（https://docs.oracle.com/javase/tutorial/java/IandI/multipleinheritance.html）。TEIはこれらの言語記法を日常DSLへ露出せず、原子性・実装必須・適合診断・明示Conflict解決という振る舞いだけを候補にする。

**現在の資格**：本節はIssue・現行Code・他チャット方針を照合した設計計画であり、Trait公開Schema/DSL、実Plugin移行、TRF-01〜12、新しいGo/CLI/Studio/Provider試験、Owner正式採用は **NOT RUN／未採用**。本DocのD1試験を通したときだけ、実装方式の判断を更新する。


**製品要求の正式な番号**：BOMは[RQ-037](../../requirements/TEI_PRODUCT_REQUIREMENTS.md)であり、[RQ-036](../../requirements/TEI_PRODUCT_REQUIREMENTS.md)は全容モデルマップ。旧Draftの設計番号を製品要件の正本と混同しない。現状のPlugin-first限定Go Consumerと実Wails/通常CLI Consumerの資格差も維持する。

## 18. Verified BOMの証拠鮮度：Source変更と選択規則変更を独立に追跡する（2026-11-08横断・受入案）

### 18.1 実mainとDraftが示した三つの非同値

- **Consumer内Source-only変更**：main統合[#3477](https://github.com/hat47x/tei-platform/pull/3477)の限定実CLI負例では、Source末尾だけ変更してもCanonical Worldが同じまま、Source Index FingerprintとScenario利用資格が変わり得る。旧Context→旧Scenarioの再利用には現在のSource Index照合が必要。これは実H3完全受領や原子的Workspace snapshotを証明しない。
- **Consumer外部のImported Provider Source変更**：[#3479](https://github.com/hat47x/tei-platform/pull/3479)は現在**Draft／未統合**で、Provider raw末尾の差分をConsumerのWorld、Semantic Digest、Source Indexが捉えられない反例を実Go Testで提案している。外部Reference/Providerの**実Source occurrence/raw fingerprint／Binding/取得時点**はConsumerのSource Indexと別の依存辺。#3479のTest/CIが確定するまで「実証済み」と呼ばず、現行#2812のNested Load snapshot受理も推測しない。
- **Capability選択Rule変更**：main統合[#3337](https://github.com/hat47x/tei-platform/pull/3337)の初回raw JSONは`unselected=6/12`、その後のmain #3350/#3363でDB Providerの選択規則が変わり、過去の`ruleFingerprint`と現在のものが異なる。見かけの`schema=v0`／`ruleVersion=v0`が一致しても履歴選択結果を再利用できない。[旧Draft #3379](https://github.com/hat47x/tei-platform/pull/3379)の歴史vs現在Regressionは、[後継Draft #3490](https://github.com/hat47x/tei-platform/pull/3490)として新main基線に移植した（実CIは別途確認）。旧rawは上書きせず出典を維持する。

これらは**同じ「リビジョン変更」という一個の真偽値へ圧縮できない**。Source Index、外部Reference/実消費、Rule implementation、Build closure、Test run、現在Grantが別々のOwnerを持つためである。

### 18.2 Requalificationの最小依存票（新Schema/新Engineではない）

| 変更するもの | 何を再確認するか | 自動的に失効させないもの |
| --- | --- | --- |
| Consumer managed Sourceだけ変更 | #3262のSource Index/Required Rule/関係/Source位置と、#1180のScenario Context pin | 既に不変と確認した無関係Target Artifact・Plugin選択 |
| 外部Provider raw/取得位置・Authority変更、Consumer World/Index不変 | 外部Reference契約（Source identity/revision/raw fingerprint／Provider実消費）と、そのProviderに依存した選択/Build/Test/BOM行 | 実消費しなかった別Provider/PluginのBOM行 |
| Capability適用Ruleの内容・実装だけ変更 | Rule Fingerprint、適用されたEnvironment/Scope/Kind+ID・現在のBindingとTrait Required Closure | 無関係ProviderのArtifact byte・業務Meaning |
| Builder/Generator/Toolchain変更 | 旧実消費closureと新Artifact実byte、Target/Hostの呼出し入口、必要独立Expected | 旧Artifact履歴／現在Grantとは別の意味受理 |
| Host Grant・Disclosureのみ失効 | 今このConsumerに公開/呼出/Effect可能かを現在のHost側へ再照合 | 過去の受理済み意味・歴史的Conformance記録 |
| 参照A→B→A／途中受領／取得不可 | 履歴の完全性・Source occurrence・欠落/非開示/partialの再確認 | Aへのbyte復帰を「途中で変わらなかった」証拠と扱わない |

**判定するのは要求された用途に相対的な依存閉包だけ**。必要なSource/Rule/Provider/Host/Build入力が未取得なら`unknown/not-loaded/stale`を保持し、`not-required/compatible/qualified`へ偽装しない。キャッシュ（#3271/#2600）のhitやVerified Compositionの過去の行は、新しいSource/Rule適格性または現在Grantを発行しない。

### 18.3 既存Scenarioへ渡す負例（BFR-01〜07）

| Case | 一要因だけ変える | 独立Expected／Claim上限 |
| --- | --- | --- |
| **BFR-01** | 消費元Worldを保持、Consumer managed Sourceの改行だけ変更 | World同一でSource Index changed。必要Context再取得まで旧Scenarioの利用資格保留（main #3477の限定既存反例） |
| **BFR-02** | Consumer World/Source Indexを保持、Imported Provider原文だけ変更 | 必要なExternal Ref/Provider occurrenceの鮮度を未確認として停止。#3479はDraftで追加実証待ち |
| **BFR-03** | 同じScope/Source/Plugin表示VersionのままCapability Rule内容だけ変更 | 旧`selected/unselected`は現在の選択資格ではない。Rule Fingerprint違いを拒否 |
| **BFR-04** | Provider Package選択は同じ、必要Kind+ID Bindingのみ削除 | selected packageとselected capability／Trait fulfillmentを分離。実Invoke不可 |
| **BFR-05** | Bを選択、実CallはA（戻り値同一） | 実Host側Provider identity/Artifact Call traceとSelectionが不一致。2×2の対象行はqualified不可 |
| **BFR-06** | 未選択・未消費のPlugin資産だけ更新 | 実依存closureで非干渉が示された行のEvidenceを機械的に全失効させない。過剰失効を検出 |
| **BFR-07** | 同じ構成で実Target/Build/Snapshotを変える、あるいは同一Attemptだが欠落Check | 同じ源のTest証拠が無ければ旧PASSを合算しない。別Target/OS/Scope、現Grantも独立 |

**所有権と順序**：#3262 Source/Required Closureと#3479 External Source、#951 Selection、#3278/#3279 Trait/Consumer、#1251実Inspector→実Call、#943/#2600 Build/Artifact、#1180/#2514独立Expected/Attempt、#3267 RQ-037 BOM。**BFRをここで独自Runnerや第二BOM Registryとして実装しない**。CLI/Studioという2 Consumerで実Providerの同一性を示すD1は依然P0。#3478の実HTTP Host Branch/監査、#3440の限定Self-buildは別Target・別Claimであり、このBOM行を自動で資格化しない。

**受理状況**：#3477/#3337/#3350/#3363はmain履歴、#3479と今回の履歴Policy比較Go RegressionはDraft、BFR-01〜07を横断する製品Inspector/Version Matrix実行、現在Host Grant、真正なTrusted Witness・Owner受理は**NOT RUN**。完成済みE2E/BOMではなく、既存計画に必要な再資格化判別を増やした段階である。

### 18.4 Consumer実行の断絶を閉じるには、呼出し元を確認する（Host/Plugin D1）

[main統合 #3446](https://github.com/hat47x/tei-platform/pull/3446)の静的走査では`internal/servicehost.NewSelectedServiceCarrierInvoker`の**非試験Callerが0**、`BindHostApplicationEnvironmentRoute`も非試験Callerが0（ただし`composition.NewSelectedCoordinator`等の部品を非試験で使う経路は別にある）。つまり「Provider選択部品が実装済み」「テスト内でService CarrierがInvokeできる」「一部のCLI/Studioが同じtyped Resultを返す」だけでは、現実のTooling Consumerから**選択済みProviderへ到達した事実をまだ得られない**。既存の`servicehost`経由を全Hostの義務にするのではなく、どの入口で実Routeを組み立てるかを選択した構成ごとに確定する。

[Draft #3371 Studio](https://github.com/hat47x/tei-platform/pull/3371)／[Draft #3381 CLI](https://github.com/hat47x/tei-platform/pull/3381)は型付きInspectorのfail-closed seamとして利用できる候補だが、当時のCI Health FAILを実Plugin適合のPASSへ転記しない。**D1の次のコード変更で必要な最小証拠**は、(a) exact Contract+SelectionをRunner/Host側で読み、(b) 実CLI/Studioの別call-siteがそのImplementation identityを呼び、(c) B選択時にAへ誤接続した同一結果の負例を拒否し、(d) 実Buildに含まれるSource/Target/ArtifactをProvider証拠へ束縛すること。Shadow比較で既存Builtinを呼んだだけの場合は`reference-parity`、実経路の切替確認後だけ`consumer-invocation-observed`とする。どちらも単独でBOMのqualifiedではない。

**BFR-08**：同一`Inspector`入力で旧Builtinと新typed Serviceの結果が等しくても、実Plugin Contract Resolution/Host Route/Provider Callを通らなければ不合格。既存実行入口が0なのにMock内部でだけService Carrierが呼ばれるケースを必須負例にする（#1251/#1158/#3279）。この検査は#1180/#2514へ渡す候補であり、製品CLI/Studioの実Bindingが成立したことはまだ**NOT RUN**。

### 18.5 Provider Manifest-only変更と、Selection→Invoke間の同一性（#3479・#3381・#3482の非重複差分）

前節の`Source/Rule`鮮度に加え、**ProviderのManifest原byte変更と、Selection完了後に実Host Entryが変わる場合**を独立に検査する。[Draft #3479](https://github.com/hat47x/tei-platform/pull/3479)は、外部ProviderのRaw Source改訂だけでなく、**Provider Manifestの末尾改行のみ改訂してもConsumer World/SourceIndexとProvider Source Anchorが不変**となる別のGo負例を追加した。ImportしたProvider側のManifest Revisionは、Consumer Context二指紋やRaw Source Anchorから導出できない。#3479の追加後HEADは独立したCI検収が必要であり、別Provider Loadの照合結果を「同一原子Snapshot」とは宣言しない。

**D1で必要な観測**：Consumer Subject/World/IndexとRequired Closure、#951のexact Provider Selectionに用いた**Provider raw Source occurrence／Manifest identity／Classifier rule fingerprint**、#1016によりHost登録した実Implementation/Target/Artifact member、実`Inspect`のEntry/AttemptとOperation結果を**別欄で結ぶ**。内部ファイルの改訂を発見しても、そのProviderを実際に使用するFunction/Operation/Build Closure以外のArtifactを全失効させない。逆に同じ結果Canonical、同じConsumer World/SourceIndex、同じPlugin表示Versionを根拠に旧Provider Source引用・Build/Invoke資格を使い回さない。

特に、**時点を一個に潰してはいけない**。`select(B)`後`invoke()`前にBのManifestが差し替わった、またはHostのEntryがAへ置き換わった場合、BとAが同じJSONを返しても実選択資格の成立ではない。`Selected`と`Actually Invoked`の整合はHost側の実呼出し観測で照合し、単なるProvider自己申告値やSynthetic Testの結果一致で保証しない。Source/Manifestを選択時に観測し、Invoke時の実消費と同時版であることが証明できなければ**unverified**とする。検査の再実行が原子的Snapshotにならない場合、未証明をPASSへ変換しない。

| Case | 一要因の対照 | 必要な独立Expected／既存Owner |
| --- | --- | --- |
| **BFR-09** | 外部Provider Manifest末尾のみ変更、Provider Raw Source/Consumer World・Indexは同じ | Manifest identity/必要Selection/実消費閉包を再照合。Provider Source FP一致だけで現在のBOM資格を維持しない。#3479/#951 |
| **BFR-10** | BのManifestを選択後に変更／Hostの実EntryをAへ交換（Result byteはBと同じ） | 実Host Entry・Source/Manifest版・Selection/Target不一致で対象Attemptをqualifiedにしない。#1251/#3381/#3371/#3267 |
| **BFR-11** | main [#3482](https://github.com/hat47x/tei-platform/pull/3482)の`sales.order.submit`宣言Requiredは13 CLI Reply／42限定Controlで一致するが、実Query/Provider Entryは未取得 | `boundedFunctionSlice.declarationStatus=bounded-present`のみ。Function実Invocation・Trait Fulfillment・Host Effect・BOM行を証明済みにしない。#3262/#2586 |
| **BFR-12** | `ErrImportsNotBound`またはProvider Manifest/Sourceが取得不能だがConsumer Contextの既知部分だけが正常 | Required Closure不足を`absent`/完全/無変更としない。必要な用途の資格はunknown/unverified。#2743/#3262/#2514 |

**実行順を変更しない**：まず#3378で旧Builtin二入口のBaselineを独立固定し、#3381が候補提示した既存Composition`→`Host登録の`inspectoradapter`局所実Go呼出・負例、#3371のStudio Factory、通常CLI/Studio入口への実Bindingを別の合否にする。次に同じAttemptでSource/Manifest/Build Artifact/現在Grant/実二Consumerの独立Expectedを照合し、#3267の2×2実BOMへ渡す。今回の文書追補・#3490の歴史Rule比較と#3479の外部Sourceテストを結合しただけでは、実Plugin二Consumer、原子的Import Snapshot、正式Trusted WitnessまたはOwner受理は**NOT RUN／NOT PROVEN**。新Source Store／Resolver／Contract Schema／Host／Test Runnerは設けない。

## 19. 実呼出しの証拠をLifecycle・HTTP・性能観測へ接続する（2026-11-08横断受入案）

### 19.1 現mainで成立したことと、未成立のProduct経路

[ADR-0048](../adr/0048-shared-application-api-and-transport-adapters.md)に従い、CLI/MCP/HTTP/Web/Desktopの共通部分は**Operationの意味、入力・結果・失敗・Effect・前提**であって、同じHostプロセス・同じTransport・同じ実Plugin Binaryを要求しない。

- [main #3371](https://github.com/hat47x/tei-platform/pull/3371)：Studio Hostの型付きInspector Serviceと、明示Serviceがない場合のfail-closed、Provider由来の非信頼Error Code/Messageの秘匿処理が成立。生成Hostへ暫定Builtin Inspectorを明示Bindingしており、Plugin B実選択とは別。
- [main #3378](https://github.com/hat47x/tei-platform/pull/3378)：6 SubjectのCLI/Studio **従来direct Operation**のread-only同値基線。Pluginの選択・実呼出しの証拠ではない。
- [Draft #3381](https://github.com/hat47x/tei-platform/pull/3381)：既存Composition→Host登録Go関数を使うB選択/B実Call・A誤登録拒否、CLI・Studioの**二つの局所Go Consumer**から同一Bを呼ぶ試験コードを統合候補に集約。**通常CLI`main.go`／Wails Bridgeの製品起動経路は依然未接続**であり、PR最新HEADのHealth・main統合とは別判定。旧重複Draft #3517はclosed/unmerged。
- [main #3516](https://github.com/hat47x/tei-platform/pull/3516)：実Workspace・実Go HTTP Host・Canonical認可・Driver・**合成Adapterの実Callback**を7正負例で検査。限定HTTP Characterization Workflowは成功、全Repo Healthは別にfailure。HTTP Adapterへの到達は確認できても、任意の本番Plugin Artifactが選択・実Linkされたことも実DB Commitも、`503`後の無副作用も証明しない。
- [main #3520](https://github.com/hat47x/tei-platform/pull/3520)：AI性能チューニング用の合成Design/Call/Statistics/Delivery/Disclosure Joinを19独立Expectedで局所検査。専用CIは成功するが、`input-consistent`は**Consumerが受け取った自己申告JSONの整合性**でありHostによるB実Callの独立証言ではない。実Function/DB/Telemetry・改善幅は未実証。
- [main #3510](https://github.com/hat47x/tei-platform/pull/3510)／[Draft #3507](https://github.com/hat47x/tei-platform/pull/3507)：Trait原子義務とLifecycle Graph/Hook Patchの**全経路上の不変条件**を別Ownerで扱う。Graph/Diagramの描画と実Targetで通ったPathを同一視しない。

### 19.2 共通に利用する「実経路の証拠」：第二Ledgerではなく読み取り専用Join

既存Source Index／Composition Selection／Build Evidence／Host Call-site／Effect・Run Journal／Tool Deliveryを**同一Attempt・同一用途Scope**へ関連付けて表示する。以下は要求する照合項目であり、新Product Schema・永続Store・Trusted Attesterの採用決定ではない。

| 区別する観測 | 証拠の所在と判定 | それだけでは証明できないこと |
| --- | --- | --- |
| **要求されたOperation** | ID/Variant、Subject・Workspace実Root、Required Source/Role、入力Revision、Consumer/Target、必要Traitと独立Expected | 原文の存在だけで意味受理や実行資格があること |
| **構成上の選択** | 選択されたContract/Provider/Scope/epoch、Capability Kind/ID、選択規則FingerprintとHost側候補解決の不在・曖昧さ | 宣言・選択だけで実BinaryへLink済み／実Invoke済み |
| **実行可能な成果物** | Target、実Build Source/Plugin memberとArtifact digest、Hostが実Load/RegistrationしたEntryとの一致 | Programに含まれるだけでそのEntryが呼ばれたこと |
| **実際の呼出し** | Host側Call-site、実Entry/Handler/Provider、入力・Attempt、開始／終了と失敗段階、SelectionからInvocationまでの変更・撤回 | Provider自己申告の`invoked`、Callbackが返っただけで業務Effectが確定したこと |
| **受領とEffect** | Consumer別のResult/Failure投影、partial/end・Tool Replyの完結、Local Journal／外部Sink・実Effect／独立Observation・現在Disclosure | 同じJSON＝同じActor／同じ結果＝同じ業務Commit／`read-only`＝出力ゼロ |
| **図表・統計への投影** | 上記Factへqualified Refで到達するEffective Lifecycle Graph・Runtime Trace・計測Window／欠測／Sampling Context | GraphにEdgeが描かれているだけで当該Pathが通ったこと／同一Query名だけで計測値が比較可能であること |

必要な一列が`unknown/partial/stale/withheld`なら、その列を他のFingerprint一致やChart表示の成功で補完しない。外部Providerのraw Source/Manifest、Consumer Source Index、選択RuleのFingerprintは[main #3479](https://github.com/hat47x/tei-platform/pull/3479)・[main #3490](https://github.com/hat47x/tei-platform/pull/3490)により**独立の失効契機**と分かっている。Semantic計算の再利用、Source引用の鮮度、実Build/Callの資格、現在Grantは必要な範囲だけ再判定する。

### 19.3 Lifecycleの不変部分・Pluginによる可変部分・実効Graph

[#3503](https://github.com/hat47x/tei-platform/issues/3503)と[Draft #3507](https://github.com/hat47x/tei-platform/pull/3507)のLifecycle Graph案を利用し、Authoritative Source→Typed Graph→選択Plugin Patch→**Effective Graph**→Target生成→**Observed Path**を別段階として扱う。標準Data/Relation/Graph・三視点/Navigation（[main #3505](https://github.com/hat47x/tei-platform/pull/3505)）からの投影であり、描画座標はSource/業務コードを変更しない。

**不変部分**は固定Node列ではなく、Operation/Scope/Variantごとに要求されたValidation、現在Host Grant、Effect/Unknown、情報開示、Required Trait等の**すべての実行可能Path上の義務**である。可変部分はHook Slot/Guard/Step順序/Target Adapter/外部イベント等のPlugin Patch。各PatchはSource位置・契約・選択Scope・版・適用順序・拒否理由を持つ候補とし、未選択Patchの混入／順序だけでの不変条件迂回／選択BからAへの実配線を拒否する。RuleやPatchを削除してValidation Pathが消えた場合、Diffが空だから無変更とは扱わない。

音声・Web ChatのStreaming delta、Tool Call、承認前入力、Resume、B2C ExperimentのExposure等は**イベント候補・観測**として扱い、Operation確定・Business Effect・Grantへ自動昇格しない。各TargetがNative/Headless/Static/Hybrid/動的実装を選んでも、共通Operationと必要な経路義務を維持する。新中央Workflow Host・Graph Compiler・VUI固有認証基盤を本節から要求しない。

### 19.4 既存Scenarioへ委譲する小さな反証（WIT-01〜09、実行前）

| ID | 最初に試す一因子変更 | 独立Expected／帰属 |
| --- | --- | --- |
| **WIT-01** | `inspect` Provider B選択、通常CLI dispatchと実Studio BridgeでそれぞれBのHost Entryを呼ぶ | 二つの実Call-siteと同版のSelection/Result。単なる局所Go二Consumerやdirect shadowを製品S2へ加算しない。#1251/#1824 |
| **WIT-02** | B選択のまま実Host登録をAへ切替、AとBのReturnを同一byteにする | 選択/実Call不一致でFAIL。値一致・統計一致・Trace自己申告で救済しない。#951/#1251 |
| **WIT-03** | Selection直後にProvider Manifest/SourceまたはRuleFingerprintを変更し、Consumer Worldを保持 | 用途相対Source/Selection再資格化まで旧Evidenceをfreshにしない。無関係Sourceだけの変更で全面再Buildもしない。#3262/#2600/#3267 |
| **WIT-04** | Host Grant失効とProvider Error内の秘密Path/Tokenを同時に与える | 実Invoke前に拒否、公開応答は許可Code/固定文、Internal Causeは外部に出ない。失敗後Effect未知を勝手に無作用にしない。#1174/#2516 |
| **WIT-05** | 呼出し成功だがTool Reply一部欠損／Session Logだけが別Sinkに保存 | Provider実CallとConsumer完全受領・Disclosureを別々に表示。Part/end/書込先不明なら後者はunverified。#2509/#2515 |
| **WIT-06** | HTTP Hostで認可拒否／CSRF／Adapter障害／Invoke後503を切替 | 主体・Driver・Callback順とHost Auditを照合。合成Adapter試験PASSを実Plugin/DB Commit/無副作用に広げない。#3274/#3516 |
| **WIT-07** | Hook Graph PatchでAuthorize/Validateを迂回する別Pathを追加 | 選択済みGraphが全Path Obligationに不適合ならTarget生成・実行資格を拒否。図として表示できることは合格でない。#3503/#3278 |
| **WIT-08** | 性能統計を同一Query名の旧Provider A／別WindowからBの最新Attemptへ付け替える | Attempt・Provider・Scope・Window・Sample欠損を比較し、AI最適化の根拠として不適格。19合成Testは実Provider測定ではない。#3520/#3271 |
| **WIT-09** | Provider B1/B2×Consumer C1/C2の4候補のうち1件だけ未実行 | 3件の結果同値でも4件目をVerified BOMにしない。実Artifact/Entry/独立Expected/同一Attemptを各行で固定。#3267 |

**順序・停止条件**：P0は[#1251](https://github.com/hat47x/tei-platform/issues/1251)の**通常CLI／Studioから選択済みBを実呼出しする一Operation**。そのProduct経路が成立したらread-only Hook一Slotを選び、Effective Graphが実Targetで通ったPathと一致する限定試験をP1とする。HTTP・音声/Chat・B2C・性能統計・Verified BOMはP0を妨げず、P1以降の**異質Consumerまたは反証用Pressure**として並行させる。WITのExpected／採否記録は#1180/#2514がOwnerであり、ここに第二Test Runnerや正式Gateを設けない。

**今回の上限**：既存コード/限定CI/他チャットの設計判断を横断した**仕様・実証順序の提案**。WIT-01〜09の同一実行、実Plugin二Consumer、Lifecycle Graph Target生成、実AI最適化、本番HTTP Plugin/実DB、外部Harness完全受領、Owner Meaning Acceptanceは **NOT RUN／NOT ACCEPTED**。部分的なGo/Node/Python Reference成功やmainへの文書統合は、この未完了資格を埋めない。
## 18. Source時点と外部Providerの実入力を、D1選択・実呼出し・BOM資格へ束縛する（2026-11-08実証接続）

**追補の理由**：§16は`Declared → Selected → Built → Invoked`を分けたが、選択した**Provider原文・Manifestの現在のbyte**がConsumerの`World/SourceIndex`で必ず把握できる、とは限らない。これはPlugin-firstを実行可能にする際の独立した欠落である。#3262のRequired Context、#951/#1016のSelection、#2600の再利用、#3267のBOM、#1180/#2514の独立Expectedを**新しい証拠正本や中央Registryなしに接続**する。既存§15のFunction局所失効は維持し、すべてのSource変更を全Artifact再Buildにはしない。

### 18.1 観測済みの二つの異なるSource-only変更と、未完成の外部受入

| 最小差分 | 現在までに実証できた範囲 | D1/BOMへ転用してはいけない推論 |
| --- | --- | --- |
| **同一Workspace内**：意味Worldは不変、Source Byte/Indexだけ変更 | [main #3477](https://github.com/hat47x/tei-platform/pull/3477)の実TEI CLI/Design Context→Scenarioは、WorldとSource Indexを別々に照合し、古いContextを限定拒否。中間A→B→Aの履歴はCurrent fingerprintのみでは復元不可 | World/Canonicalが同じなら旧Source Anchor/Scenario/Plugin Contextが現在も有効だとしない |
| **外部からimportしたProvider原文**のみ改訂 | [Draft #3479](https://github.com/hat47x/tei-platform/pull/3479)は、Consumer World/Semantic Digest/Managed Source Indexが変わらなくてもProvider raw SHA/現在`LocateImportedSource`の指紋が変わる反例を実Go Testへ追加。先行HEADのHealth successと、Manifest負例追加後HEADのCIは別判定 | ConsumerのWorld+SourceIndex二指紋一致を外部Provider SourceのFresh証明にしない |
| **外部Provider Manifest**のみ改訂（原文Source Anchor不変） | 同Draft #3479が別のManifest raw SHA改訂負例を追加。元Provider SourceとConsumer Indexでは改訂を見つけられない条件を区別する。**新HEADのCI成功・同一Nested Load Snapshotは未確認** | Plugin ID/Version・原文Source位置が同じなら実装Manifest/Selection規則も同じとしない |
| **選択Classifier Rule**だけ改訂 | main #3337→#3368の12 Capability未選択6→4はRule Revision/判定方法の変化。古いraw Evidenceは保持する | Coverageが改善した、Providerが実Loadされた、Verified BOMが更新済みとしない |
| **Functionの宣言・依存だけが部分照合** | [main #3482](https://github.com/hat47x/tei-platform/pull/3482)は`sales.order.submit`の独立Requiredを13実CLI Reply・42限定Controlsで照合しmain pushで限定PASS。Statusは`bounded-present`／全体`partial` | `application uses semantic sales`やCapability宣言だけでFunction→実Query/Provider→実Host Invocationが結線済みとしない |

**鮮度は用途相対**：同じSource表現のbyte変更でも、(a) 原文引用・再解釈・Designer/SHOの出典、(b) 構造候補・受理済意味、(c) 具体Provider実行、(d) Build Artifact、(e) Result/Disclosureへ影響する範囲が異なる。ConsumerのProject Indexのみでは外部Providerの原文/Manifest/取得時点を網羅しない。Import Referenceが`partial/unavailable/not-bound`なら存在しない・無変更と見なさない。Source修正の現在値だけでA→B→Aの経緯を証明しない。

### 18.2 一つのD1 Requestで必要な最小のInput Evidence chain

**同一`Attempt`で別欄に記録する観測（新しい固定API Schemaとはしない）**：

1. **Consumer Input**：Workspace/Function/Subject ID、Scope、直接消費したWorld/Source Index FingerprintとそのObservation時点、取得済みRequired Closureとpartial/unknown/拒否の有無。#3482の限定`bounded-present`はこの入口まで。
2. **Provider Selection Input**：Contract/Trait ID・Version/Kind、実解決結果、Selection Rule実内容Fingerprint、Environment Binding、選択に実際に使ったProvider Manifest/Source Revision/Origin、候補数と曖昧性（#951/#1016）。
3. **Provider Execution Input**：その同じ解決で選ばれた実Implementation/Build member、Host登録したProvider entry、実呼出し時のProvider Source/Manifest Revision、Process/Artifact/Target、実Call-site、入力Digest。値が同じ結果を返すA/Bでも**B選択→A実EntryはFAIL**。
4. **Result and Delivery**：型付きApplication OperationのSubject/Source Anchor、World/Index、Null/Unknown/partial、Effect/Failure、CLI/Studioの独立Consumer実受領・Disclosure/Sink。Transport文書同値、Provider実Call同一性、現在Grantは別Claim。共通操作は採用済み[ADR-0048](https://github.com/hat47x/tei-platform/blob/main/plan/adr/0048-shared-application-api-and-transport-adapters.md)を使う。
5. **Independent Qualification**：#1180/#2514の実独立Expected、Source/Provider/Attempt/Targetに束縛したScenarioとCoverage。#3267のBOMはこのうち実測済み組合せだけに資格を付与し、以前のContext/Result/Artifactを現在のHost Grantへ昇格させない。

**TOCTOU停止線**：ConsumerがSourceを観測→Providerを選択→Build/Invoke→結果を受領する間のSource/Manifest/Rule改訂は、一回の`World/Index`再取得だけで取り消せない。**選択に使ったInputと実消費したInputが同じ版と証明できる**ときに限り当該Attemptをqualified候補とする。別Provider Loadの原文SHAとConsumer LoadのFPが各々読めても、同時点の結合を保証しない場合は`unverified`。Sourceのローカルコピーや指紋の一致は、Source Authority・Import Permission・Runtime Grant・Accepted Meaningの代替ではない。

**再検証範囲**：Provider原文のみ変わったときは原文引用/取得/Provider Inputの資格を再評価し、意味非依存が実証された無関係Function/Artifactは維持候補。Manifest・Classifier Rule変更時はSelection/Required Closureと実影響を再照合する。Build Inputが変わっていなければArtifact byteまで無根拠に失効させない一方、実読み込むModule/Manifest/Plugin初期化が変われば同じ表示ResultでもBuild/Invocationの資格を再評価する。

### 18.3 既存Scenarioへ追加する最小の負例（PDC-15〜20）

| ID | 独立した改変と観測 | Expected・Owner |
| --- | --- | --- |
| **PDC-15** | 同一WorkspaceのSourceにLF改訂、World不変・SourceIndex変化 | 過去Context/Source位置はstale。意味が同じであることだけを根拠に引用やScenarioを再利用しない（#3477/#3262） |
| **PDC-16** | 外部import Provider原文のみ改訂、Consumer World/Index不変 | Provider原文SHA/Source occurrenceの現在確認なしで`fresh`やPlugin evidence reuseを返さない。#3479は限定Test候補、Nested Load一貫性は別（#951/#2600） |
| **PDC-17** | 外部Provider Manifestのみ改訂、同じCapability名・Consumer FP・Provider原文Anchor | Manifest Identity/Selection Ruleと実Resolveを再照合。単に`Source unchanged`ではqualifiedにしない（#3479/#951） |
| **PDC-18** | 旧Selection Rule Fingerprint、同名Versionの新判定規則でCandidate集合が変わる | 判定由来と母数を維持し、既存`selected`を実Providerに昇格しない。BOM Historyは別証拠（#3368/#3267） |
| **PDC-19** | 13 CLI Responseで`sales.order.submit`の宣言Requiredが一致しても、Query/Provider Entry未結線 | Declaration`bounded-present`、`invocationBinding=not-observed`。実Function実行やGrantをPASSしない（#3482/#1251） |
| **PDC-20** | Bを選択→B登録した後、Invoke前にB ManifestまたはHost EntryをAに差替え。返却Canonical byte同一 | 選択時Inputと実Entry・実Artifactを比較し、不一致をFAIL/unknownで保持。単なる結果再検証では代替不可（#1251/#3381/#3371） |

### 18.4 変更順と独立実証の境界

- **P0／既存作業を合成する**：#3378の旧Direct二Consumer shadow、#3381の既存Compositionを使う`inspectoradapter`（Host登録Bの実関数Call・誤登録拒否）、#3371のStudio Host Factoryを、正式コードに入れる前に各Draftのexact-head CIと合流時回帰を別々に通す。先に同じGo関数を2回呼んだだけの`2 Consumer PASS`を禁止。既存`operations.ValidateInspectionProjection`へGuardを収束し、CLI通常入口とStudio Production Host双方へ**実Provider Binding**をつなぐ。
- **P0並行／Source Fenceの欠落を実証**：#3477 mainのin-Workspace testを既存Fixtureとして再利用。外部Providerは#3479のSource-only/Manifest-only二負例を再利用し、ConsumerのWorld/Indexに**外部Provider同時点の資格が含まれない**という限界を明示。新Importer、Snapshot Store、Source Index別正本を作らない。
- **P1／Qualified BOMへの限定投入**：#3297/#3267のMatrixは実Provider×Consumer×Targetの**同一Attempt**と独立Expectedが揃う場合のみ合格。#3479のSource raw観測と#3482のFunction宣言検査は、それ単独ではBOMの実Invocation・Conformance欄を埋めない。現在のSource Scope/Grantの不足はunknown。
- **P1／異質なConsumerへの圧力**：REA/SHOの外部引用、KON Dataspace交換、Data WorkbenchのRead/DDL、CLI/MCP/StudioのRecipient SinkとSource Referenceを同じ既存契約で比較。ただしSHO文書生成・DB書込・外部Harness受領をInspector D1のHard Gateに積み上げない。外部Source/Import時の実Host Grantは既存Authorityが所有する。

**状態**：#3477/#3482のmain局所実証、#3384の計画統合、#3371/#3378/#3381/#3479の別Draft/未統合の実装候補を分離。**本節は実Go/CLI/Studio/Providerの新試験ではなく、PDC-15〜20の受入設計**。新Importer/Resolver/BOM issuer/Runner/Host/Authority/正式API/Owner Meaning Acceptanceは追加・取得していない。全外部Reference Current Source Closure、CLI+Studio同じ選択Provider実Call、Production BOM、Current Grantは**NOT PROVEN**。
