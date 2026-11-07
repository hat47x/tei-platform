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

## 8. Desktopで最初に通す縦断：PluginのRead-only OperationをCLI／Studioへ同一契約で接続する

**追加背景（2026-11-07、#1824 Consumer・#3270 Workbench・#3274 HTTP Graph・#3262 Design Contextと照合）**：現行 `cmd/tei/cli_commands.go` の手書きコマンド、`cmd/tei-studio/main.go` の固定 `StudioBridge`、`reference/tei-studio-workbench/main.ts` のSource Subject Catalog・`validate/refresh/inspect` は、同じContract/Providerの実選択と呼出しを共有していない。これを「CLI/StudioはPlugin化済み」や「任意のData Source/Graphを汎用表示・操作済み」と見せない。Studioの既定 `/workbench/` とBootstrap `/`、生成アプリTargetの証拠は別に取る。

**D1の実装対象を絞る**。新しい汎用Plugin Manager、UI Registry、HTTP Graph Renderer、Data Source Backendを先行しない。最初のOperationは現存する **read-only `inspect` または `model catalog` の一方**に固定し、既存のTyped Service/Source Index/Host公開制約をProviderとして包んだうえで、#951/#1016の実Compositionが選んだexact ProviderをCLIとStudioの二Consumerへ引き渡す。既存の結果・診断を利用し、両者で同じMeaning/Read scope/Failureを観測できるか確認する。Wails Bridgeは薄いAdapterとして残せるが、選択Pluginに依存した意味分岐・固定Command追加を重複正本として増殖させない。

### 8.1 Producer→Consumerの接続で消してはいけない情報

| 区間 | 取得すべき証拠（既存契約へ束縛する候補。新Schema／正式CLIは未採用） | そこから言えないこと |
| --- | --- | --- |
| **要求と資格** | Caller、Workspace/Subjectのstable ref、用途、expected World/Source Index/Accepted Revision、現在のHost/Disclosure scope | Subject名が一致するだけで現在の開示・実行が許可された |
| **Contract/Provider選択** | Contract/Version、実Provider identity、selected exact Lock、未選択/複数候補/Contract不一致の診断、実呼出しTarget | Plugin Packageに同梱／Capability宣言あり＝実Providerが選択・実行された |
| **実Source/Design Context** | Required/Discovered/Loaded/Parsed/Bound/Returnedの用途相対集合、Provider読取範囲・取得時点・粒度・NULL/欠損/非開示/partial | Catalog件数、validate.ok、終了コード0、切断されたJSON＝完全な対象設計情報 |
| **UI Presentation** | 同一Resultの公開可能なField/Relation/Source locationとLoss、表示Plugin・Target版、Selection/Workspace epoch、受領完結 | StudioのTable/Graph表示やARIAの選択＝Source Editorの受理／Dataの更新Target／Host Grant |
| **Result/Recovery** | Typed failure/unknown/unsupported/stale・受領完全性、exact Operation/Attempt、実証時は独立Expected | read-onlyの成功＝業務Action/HTTP Effect/Receipt/資源現在値／Owner Completion |

D1では**read-onlyなので外部Write Effectは発生させない**。ただし外部データを読むProviderではネットワーク・課金・秘密取得などの利用条件があり得るため、read-onlyという名前だけで暗黙接続・全件検索・外部送信を許さない。実際のProviderが未配線なら、Stub/Referenceの応答をProduct実Readとして表示せず `not-connected` を残す。

### 8.2 他チャットの成果をDesktop固有のConsumer圧力へ接続

1. **汎用Data Source Workbench #3270／RQ-035**：初期はschema/Query Catalog等の読み取り専用候補だけを共通Operationで検証。実SQL Row/Field/Scopeを読み込んでいない場合、Table/Treeを描画できても「異種Data Source閲覧完了」とはしない。DDLのPlan/Applyはread-only Pluginの機能へ暗黙付与しない。DROPや型縮小は別のIntent/Grant/実Provider/独立再観測（#2892のMigration History unknownを含む）。
2. **HTTP Response Decision Graph #3274**：標準有向多重GraphのNode/Edge/Guard/Outcomeと、Client Reaction/Recoveryは用途別Projection。既存 `reference/tei-studio-workbench/presentation.ts::relatedGraph` はCatalogの単親・最大8 Entryだけ、`reference/tei-studio-graph/graph.ts`はself-edge未対応のread-only Rendererなので、実HTTP/業務の複雑な多重GraphのConformanceには使えない。初回は許可されたNode/Edge表／Source参照／未対応診断を共通Read Contractで表示し、適格なGraph Rendererができてから同一Sourceから比較。Graph表示やOpenAPI生成は実Server Guard/Client Retry安全性の証拠でない。
3. **Design Context Query #3262／AIテスト #3273**：CLI/Studioの読取りは、Required集合とSourceの実読取閉包、page/cursor/end marker、途中切断/欠落Reply、現在Disclosureを同じ実根拠から説明する。人間向けに短縮した表示からRequiredが欠けても「complete」としない。CLIの9 read-onlyコマンド/限定T0 PASSはP-1b全閉包QueryやAI新Case生成の証拠ではない。
4. **Verified Composition BOM #3267**：同一Contractを供給するProvider二版／Renderer二版の各exact Lock・実Build/OS/Scopeを別資格として扱う。StudioのPlugin一覧に見えること／参考的Matrixのpairwise coverageだけから、実Windows Wailsや生成Goアプリでの相互運用性を保証しない。選択されないPluginの `init`・import・asset混入も実Buildで確認する。
5. **Raw Authoring/SHO**：`*.tei.md`のParser未対応・Syntax errorでも、Hostに許された原byteのOpen/Raw Saveは意味WorkspaceのLoad成功へ依存させない。Editorの保護対象はSource raw byte/Revision/dirty bufferであり、D1のInspectorや意味受理/Provider接続の成否はRaw Draft保存の資格を与えたり奪ったりしない。

### 8.3 既存EDP／Scenarioに追加する最初の一因子対照（新Scenario SSOTなし）

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
